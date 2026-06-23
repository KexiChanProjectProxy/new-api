package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/google/uuid"
)

// Langfuse OTLP/HTTP ingestion constants.
//
// Endpoint:  {BaseURL}/api/public/otel
// Auth:      HTTP Basic, username=PublicKey, password=SecretKey (base64)
// Headers:   Content-Type: application/json
//
//	x-langfuse-ingestion-version: 4
//
// This transport deliberately avoids any OpenTelemetry SDK dependency — it
// shapes a minimal JSON body and POSTs it via net/http to Langfuse's Ingestion
// API (/api/public/ingestion). Unlike the OTLP endpoint, this API accepts
// structured trace/generation events directly and renders them immediately
// in the Langfuse UI.
const (
	langfuseIngestionPath = "/api/public/ingestion"
	langfuseContentType   = "application/json"

	// langfuseEmitTimeout caps a single POST attempt. Langfuse ingest is
	// fire-and-forget from the gateway's perspective; we never want a slow
	// trace backend to stall a relay request.
	langfuseEmitTimeout = 10 * time.Second

	// langfuseDefaultFlushTimeout bounds graceful-shutdown drain.
	langfuseDefaultFlushTimeout = 5 * time.Second
)

// langfuseConfig is the effective (snapshot) config used by the exporter at
// any given moment. It is computed from the package-level
// common.LangfuseRequestLog* vars which are kept hot by model.updateOptionMap.
type langfuseConfig struct {
	Enabled   bool
	BaseURL   string
	PublicKey string
	SecretKey string
}

// fingerprint returns a stable hash of the config fields that, when changed,
// require the exporter to be rebuilt (new endpoint URL + credentials). It is
// intentionally a hex string so it compares cheaply under a read lock.
func (c langfuseConfig) fingerprint() string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%v|%s|%s|%s",
		c.Enabled, c.BaseURL, c.PublicKey, c.SecretKey)))
	return hex.EncodeToString(sum[:])
}

// endpoint joins BaseURL with the OTLP/HTTP path, trimming any trailing slash.
func (c langfuseConfig) endpoint() string {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		return ""
	}
	return base + langfuseIngestionPath
}

// authHeader returns the HTTP Basic Authorization value (with scheme prefix)
// or "" if credentials are incomplete.
func (c langfuseConfig) authHeader() string {
	if c.PublicKey == "" || c.SecretKey == "" {
		return ""
	}
	creds := c.PublicKey + ":" + c.SecretKey
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(creds))
}

// langfuseExporter is the singleton OTLP/HTTP client. It is goroutine-safe:
// all config reads take an RLock; a rebuild (on config drift) takes a Lock.
// In-flight POSTs are tracked by a WaitGroup so Flush can drain them.
type langfuseExporter struct {
	mu sync.RWMutex

	// httpClient is shared across rebuilds; it is safe for concurrent use
	// and carries sensible defaults via http.DefaultTransport.
	httpClient *http.Client

	// cfg is the currently-applied effective config.
	cfg langfuseConfig

	// fingerprint of cfg, cached so Emit can detect drift cheaply.
	fp string

	// inFlight tracks outstanding Emit POSTs for Flush.
	inFlight sync.WaitGroup

	// started guards against double-init / double-shutdown.
	started bool
}

// langfuseExporterSingleton is the process-wide instance. It is always
// non-nil (zero-value usable) so callers can safely call Emit / Flush before
// StartLangfuseExporter — they will just no-op while Enabled is false.
var langfuseExporterSingleton = &langfuseExporter{
	httpClient: &http.Client{Timeout: langfuseEmitTimeout},
}

// currentConfig reads the live effective config from the common package. The
// common.LangfuseRequestLog* vars are written under model.OptionMap's RWMutex
// by model.updateOptionMap, so reading them here is safe (string/bool reads
// are atomic on amd64/arm64 in practice, and any torn read only causes the
// fingerprint to mismatch on the next Emit — which triggers a rebuild, the
// correct outcome).
func currentLangfuseConfig() langfuseConfig {
	return langfuseConfig{
		Enabled:   common.LangfuseRequestLogEnabled,
		BaseURL:   common.LangfuseRequestLogBaseURL,
		PublicKey: common.LangfuseRequestLogPublicKey,
		SecretKey: common.LangfuseRequestLogSecretKey,
	}
}

// StartLangfuseExporter initializes the singleton exporter and primes it with
// the current config. Safe to call multiple times; only the first call has
// effect. It does NOT start a background loop — emission is driven by Task 5
// calling EmitLangfuseAudit per terminal request.
func StartLangfuseExporter() {
	e := langfuseExporterSingleton
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.started {
		return
	}
	e.cfg = currentLangfuseConfig()
	e.fp = e.cfg.fingerprint()
	e.started = true
	if e.cfg.Enabled {
		common.SysLog("langfuse request-log exporter started -> " + e.cfg.endpoint())
	}
}

// StopLangfuseExporter flushes in-flight emissions with the default bounded
// timeout and marks the singleton stopped. It is idempotent.
func StopLangfuseExporter() {
	StopLangfuseExporterWithContext(context.Background())
}

// StopLangfuseExporterWithContext flushes in-flight emissions, bounded by the
// supplied context, then marks the singleton stopped. Idempotent.
func StopLangfuseExporterWithContext(ctx context.Context) {
	e := langfuseExporterSingleton
	e.mu.Lock()
	if !e.started {
		e.mu.Unlock()
		return
	}
	e.mu.Unlock()
	// Drain outside the write lock so Emit's WaitGroup.Done can proceed.
	flushLangfuse(ctx)
	e.mu.Lock()
	e.started = false
	e.mu.Unlock()
	if ctx.Err() == nil {
		common.SysLog("langfuse request-log exporter stopped")
	}
}

// FlushLangfuse blocks until all in-flight Emit POSTs have completed or the
// default 5s timeout elapses. Returns the context error (nil on clean drain).
// Safe to call concurrently with Emit.
func FlushLangfuse() error {
	ctx, cancel := context.WithTimeout(context.Background(), langfuseDefaultFlushTimeout)
	defer cancel()
	return flushLangfuse(ctx)
}

// flushLangfuse is the lock-free inner drain. It waits on the in-flight WG
// via a background goroutine so the context deadline is honored even when a
// POST hangs.
func flushLangfuse(ctx context.Context) error {
	e := langfuseExporterSingleton
	done := make(chan struct{})
	go func() {
		e.inFlight.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// maybeRebuildLocked is called under at least a read lock. If the live config
// fingerprint differs from the cached one, it upgrades to a write lock and
// rebuilds the effective cfg + fp. Returns the effective config and endpoint
// to use for this Emit, plus an "enabled" flag.
//
// Callers must NOT hold any lock when invoking this — it manages its own
// lock upgrades and may temporarily release the read lock.
func (e *langfuseExporter) maybeRebuild() (langfuseConfig, bool) {
	live := currentLangfuseConfig()
	liveFp := live.fingerprint()

	e.mu.RLock()
	started := e.started
	cachedFp := e.fp
	cachedCfg := e.cfg
	e.mu.RUnlock()

	if !started {
		return langfuseConfig{}, false
	}
	if cachedFp == liveFp {
		return cachedCfg, cachedCfg.Enabled
	}

	// Drift detected — rebuild under write lock. Re-check under the lock to
	// avoid clobbering a concurrent rebuild.
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.fp == liveFp {
		// Another goroutine already rebuilt to the same fingerprint.
		return e.cfg, e.cfg.Enabled
	}
	prevEnabled := e.cfg.Enabled
	e.cfg = live
	e.fp = liveFp
	switch {
	case !prevEnabled && live.Enabled:
		common.SysLog("langfuse request-log exporter enabled -> " + live.endpoint())
	case prevEnabled && !live.Enabled:
		common.SysLog("langfuse request-log exporter disabled")
	case prevEnabled && live.Enabled:
		// Hot reload of URL/credentials while staying enabled.
		common.SysLog("langfuse request-log exporter reconfigured -> " + live.endpoint())
	}
	return e.cfg, e.cfg.Enabled
}

// emit posts one OTLP/HTTP ingestion request to Langfuse. It assumes the
// caller has already verified the exporter is enabled and config is current.
// In-flight tracking (WaitGroup) is handled by the public EmitLangfuseAudit.
func (e *langfuseExporter) emit(ctx context.Context, cfg langfuseConfig, body []byte) error {
	endpoint := cfg.endpoint()
	if endpoint == "" {
		return fmt.Errorf("langfuse: BaseURL is empty")
	}
	auth := cfg.authHeader()
	if auth == "" {
		return fmt.Errorf("langfuse: PublicKey/SecretKey missing")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("langfuse: build request: %w", err)
	}
	req.Header.Set("Content-Type", langfuseContentType)
	req.Header.Set("Authorization", auth)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("langfuse: POST %s: %w", endpoint, err)
	}
	defer resp.Body.Close()
	// Drain the body (capped) to allow connection reuse. We do not parse the
	// response — Langfuse ingest errors are logged but never block emission.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4*1024))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("langfuse: ingest returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// buildLangfuseIngestionBody marshals the snapshot into a Langfuse Ingestion
// API batch body. Each relay request produces one trace-create and one
// generation-create event so the full request/response is visible in the
// Langfuse UI as a single trace with a generation observation.
//
// Body shape:
//
//	{
//	  "batch": [
//	    { "id": "...", "timestamp": "...", "type": "trace-create",
//	      "body": { "id": "...", "name": "new-api-request", ... } },
//	    { "id": "...", "timestamp": "...", "type": "generation-create",
//	      "body": { "id": "...", "traceId": "...", "model": "...",
//	                "input": ..., "output": ..., "usage": {...} } }
//	  ]
//	}
func buildLangfuseIngestionBody(snapshot *LangfuseAuditSnapshot) ([]byte, error) {
	now := time.Now().UTC()
	traceID := "newapi-" + snapshot.Metadata.RequestId
	if traceID == "newapi-" {
		traceID = fmt.Sprintf("newapi-%d", now.UnixNano())
	}
	genID := traceID + "-gen"

	// Parse request payload from JSON for structured display.
	var input any = snapshot.RequestPayload
	if len(snapshot.RequestPayload) > 0 {
		var parsed any
		if common.Unmarshal(snapshot.RequestPayload, &parsed) == nil {
			input = parsed
		}
	}

	// Build output: error first, then response, then binary placeholder.
	var output any
	level := "DEFAULT"
	var statusMessage *string
	if len(snapshot.ErrorPayload) > 0 {
		level = "ERROR"
		s := "relay request failed"
		statusMessage = &s
		var parsed any
		if common.Unmarshal(snapshot.ErrorPayload, &parsed) == nil {
			output = parsed
		} else {
			output = string(snapshot.ErrorPayload)
		}
	} else if len(snapshot.ResponsePayload) > 0 {
		var parsed any
		if common.Unmarshal(snapshot.ResponsePayload, &parsed) == nil {
			output = parsed
		} else {
			output = string(snapshot.ResponsePayload)
		}
	} else if snapshot.BinaryResponse != nil {
		output = snapshot.BinaryResponse
	}

	nowStr := now.Format("2006-01-02T15:04:05.000Z")
	startStr := snapshot.Metadata.StartTime.UTC().Format("2006-01-02T15:04:05.000Z")

	meta := map[string]any{
		"model":      snapshot.Metadata.ModelName,
		"group":      snapshot.Metadata.GroupName,
		"channel_id": snapshot.Metadata.ChannelId,
		"request_id": snapshot.Metadata.RequestId,
		"client_ip":  snapshot.Metadata.ResolvedClientIP,
		"is_stream":  snapshot.Metadata.IsStream,
		"source":     "new-api",
	}

	body := struct {
		Batch []any `json:"batch"`
	}{
		Batch: []any{
			map[string]any{
				"id":        uuid.New().String(),
				"timestamp": nowStr,
				"type":      "trace-create",
				"body": map[string]any{
					"id":       traceID,
					"name":     "new-api-relay-request",
					"userId":   fmt.Sprintf("%d", snapshot.Metadata.UserId),
					"metadata": meta,
					"input":    input,
					"output":   output,
				},
			},
			map[string]any{
				"id":        uuid.New().String(),
				"timestamp": nowStr,
				"type":      "generation-create",
				"body": map[string]any{
					"id":              genID,
					"traceId":         traceID,
					"name":            "relay-request",
					"startTime":       startStr,
					"endTime":         nowStr,
					"model":           snapshot.Metadata.ModelName,
					"modelParameters": map[string]any{
						"relay_format": snapshot.Metadata.RelayFormat,
						"is_stream":    snapshot.Metadata.IsStream,
					},
					"input":         input,
					"output":        output,
					"level":         level,
					"statusMessage": statusMessage,
					"usage": map[string]any{
						"input":  snapshot.Metadata.PromptTokens,
						"output": snapshot.Metadata.CompletionTokens,
						"unit":   "TOKENS",
					},
					"version": "1",
				},
			},
		},
	}
	return common.Marshal(&body)
}

// EmitLangfuseAudit is the public emission entry point. It wraps the snapshot
// into an Ingestion API batch body and POSTs it. It is a no-op (returns nil)
// when the exporter is disabled or not started, so Task 5 can call it
// unconditionally from the terminal relay paths without guarding.
func EmitLangfuseAudit(snapshot *LangfuseAuditSnapshot) error {
	if snapshot == nil {
		return nil
	}
	e := langfuseExporterSingleton
	cfg, enabled := e.maybeRebuild()
	if !enabled {
		return nil
	}

	body, err := buildLangfuseIngestionBody(snapshot)
	if err != nil {
		return fmt.Errorf("langfuse: marshal body: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), langfuseEmitTimeout)
	defer cancel()

	e.inFlight.Add(1)
	defer e.inFlight.Done()
	if err := e.emit(ctx, cfg, body); err != nil {
		common.SysError(err.Error())
		return err
	}
	return nil
}

// EmitLangfuseAuditFromSink finalizes and emits a Langfuse record from a
// LangfuseSnapshotSink (the interface declared on relay/common.RelayInfo to
// avoid an import cycle). The sink is type-asserted to *LangfuseAuditSnapshot
// — the only production implementation. If the sink is nil or does not match
// the concrete type, this is a no-op.
//
// usage/quota/latency are finalized on the snapshot before emission. This is
// the single emission entry point called from the terminal success paths
// (PostTextConsumeQuota / PostAudioConsumeQuota) and the terminal failure
// path in controller.Relay.
//
// errPayload, when non-nil, replaces the snapshot's error payload (used by the
// failure path). It is left untouched when nil (success path).
//
// Emission runs in a goroutine so it never blocks the relay request path; the
// exporter's in-flight WaitGroup still tracks it for graceful-shutdown Flush.
func EmitLangfuseAuditFromSink(sink relaycommon.LangfuseSnapshotSink, prompt, completion, total, quota int, errPayload []byte) {
	if sink == nil {
		return
	}
	snapshot, ok := sink.(*LangfuseAuditSnapshot)
	if !ok || snapshot == nil {
		return
	}
	snapshot.SetUsage(prompt, completion, total)
	snapshot.SetQuota(quota)
	if errPayload != nil {
		snapshot.SetError(errPayload)
	}
	snapshot.FinalizeLatency()
	MaskLangfuseAudit(snapshot)

	gopool.Go(func() {
		_ = EmitLangfuseAudit(snapshot)
	})
}
