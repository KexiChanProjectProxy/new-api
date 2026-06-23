package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// withLangfuseConfig sets the live config vars for the duration of t and
// restores them on cleanup. Tests must not leak config state.
func withLangfuseConfig(t *testing.T, enabled bool, baseURL, pub, secret string) {
	t.Helper()
	origEnabled := common.LangfuseRequestLogEnabled
	origBase := common.LangfuseRequestLogBaseURL
	origPub := common.LangfuseRequestLogPublicKey
	origSecret := common.LangfuseRequestLogSecretKey
	common.LangfuseRequestLogEnabled = enabled
	common.LangfuseRequestLogBaseURL = baseURL
	common.LangfuseRequestLogPublicKey = pub
	common.LangfuseRequestLogSecretKey = secret
	t.Cleanup(func() {
		common.LangfuseRequestLogEnabled = origEnabled
		common.LangfuseRequestLogBaseURL = origBase
		common.LangfuseRequestLogPublicKey = origPub
		common.LangfuseRequestLogSecretKey = origSecret
	})
}

// resetExporter returns the singleton to a pristine stopped state and primes
// a fresh http client. Called from every test to avoid cross-test bleed.
func resetExporter(t *testing.T) {
	t.Helper()
	e := langfuseExporterSingleton
	e.mu.Lock()
	e.started = false
	e.cfg = langfuseConfig{}
	e.fp = ""
	e.httpClient = &http.Client{Timeout: langfuseEmitTimeout}
	e.mu.Unlock()
}

// newCaptureServer returns an httptest.Server that records every POST it
// receives (body + auth header) into the returned slice in arrival order.
// The server returns HTTP 200 by default.
func newCaptureServer(t *testing.T, status int) (*httptest.Server, *captureState) {
	t.Helper()
	st := &captureState{mu: &sync.Mutex{}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		st.mu.Lock()
		st.calls = append(st.calls, capturedCall{
			path:   r.URL.Path,
			auth:   r.Header.Get("Authorization"),
			ct:     r.Header.Get("Content-Type"),
			body:   body,
			status: status,
		})
		st.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, st
}

type captureState struct {
	mu    *sync.Mutex
	calls []capturedCall
}

type capturedCall struct {
	path   string
	auth   string
	ct     string
	body   []byte
	status int
}

func (s *captureState) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

func (s *captureState) snapshot() []capturedCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]capturedCall, len(s.calls))
	copy(out, s.calls)
	return out
}

// minimalSnapshot returns a valid, already-masked snapshot for emission tests.
func minimalSnapshot() *LangfuseAuditSnapshot {
	return &LangfuseAuditSnapshot{
		RequestPayload: []byte(`{"model":"gpt-4o","messages":[]}`),
		Metadata: LangfuseAuditMetadata{
			RequestId: "req-test-1",
			UserId:    1,
		},
	}
}

// --- Tests -----------------------------------------------------------------

// TestLangfuseHotReloadRebuildsExporter verifies that changing the live config
// (URL or credentials) causes subsequent Emit calls to hit the NEW endpoint
// with the NEW auth, without an explicit Restart call.
func TestLangfuseHotReloadRebuildsExporter(t *testing.T) {
	resetExporter(t)
	defer resetExporter(t)

	srv1, st1 := newCaptureServer(t, http.StatusOK)
	srv2, st2 := newCaptureServer(t, http.StatusOK)

	// Initial config -> srv1 with creds pub1/secret1.
	withLangfuseConfig(t, true, srv1.URL, "pk-lh-1", "sk-lh-1")
	StartLangfuseExporter()
	defer StopLangfuseExporter()

	if err := EmitLangfuseAudit(minimalSnapshot()); err != nil {
		t.Fatalf("emit #1: %v", err)
	}
	if got := st1.count(); got != 1 {
		t.Fatalf("srv1 calls after emit #1 = %d, want 1", got)
	}

	// Hot-reload: flip BaseURL + credentials to srv2. No Restart call.
	withLangfuseConfig(t, true, srv2.URL, "pk-lh-2", "sk-lh-2")
	if err := EmitLangfuseAudit(minimalSnapshot()); err != nil {
		t.Fatalf("emit #2: %v", err)
	}
	if got := st2.count(); got != 1 {
		t.Fatalf("srv2 calls after emit #2 = %d, want 1", got)
	}
	if got := st1.count(); got != 1 {
		t.Fatalf("srv1 calls after hot reload = %d, want still 1", got)
	}

	// Verify the second call carried the new auth header.
	calls2 := st2.snapshot()
	if len(calls2) != 1 {
		t.Fatalf("expected 1 srv2 call, got %d", len(calls2))
	}
	wantAuth := "Basic " + base64Std("pk-lh-2:sk-lh-2")
	if calls2[0].auth != wantAuth {
		t.Errorf("auth after reload = %q, want %q", calls2[0].auth, wantAuth)
	}
	if calls2[0].ct != langfuseContentType {
		t.Errorf("content-type = %q, want %q", calls2[0].ct, langfuseContentType)
	}
	if calls2[0].path != langfuseIngestionPath {
		t.Errorf("path = %q, want %q", calls2[0].path, langfuseIngestionPath)
	}
}

// TestLangfuseDisableStopsEmission verifies that flipping Enabled=false stops
// all HTTP traffic; Emit returns nil and no server records a call.
func TestLangfuseDisableStopsEmission(t *testing.T) {
	resetExporter(t)
	defer resetExporter(t)

	srv, st := newCaptureServer(t, http.StatusOK)

	// Start enabled.
	withLangfuseConfig(t, true, srv.URL, "pk-dis", "sk-dis")
	StartLangfuseExporter()
	defer StopLangfuseExporter()

	if err := EmitLangfuseAudit(minimalSnapshot()); err != nil {
		t.Fatalf("emit while enabled: %v", err)
	}
	if got := st.count(); got != 1 {
		t.Fatalf("calls while enabled = %d, want 1", got)
	}

	// Disable via hot reload.
	withLangfuseConfig(t, false, srv.URL, "pk-dis", "sk-dis")
	for i := 0; i < 5; i++ {
		if err := EmitLangfuseAudit(minimalSnapshot()); err != nil {
			t.Fatalf("emit while disabled returned err: %v", err)
		}
	}
	if got := st.count(); got != 1 {
		t.Fatalf("calls after disable = %d, want still 1 (emission must stop)", got)
	}
}

// TestLangfuseFlushOnShutdown verifies that in-flight emissions are drained
// before StopLangfuseExporter returns. We simulate a slow handler and confirm
// Flush/Stop wait for it (within the 5s budget).
func TestLangfuseFlushOnShutdown(t *testing.T) {
	resetExporter(t)
	defer resetExporter(t)

	handlerDone := make(chan struct{})
	handlerStarted := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerStarted <- struct{}{}
		// Hold the response long enough that Emit hasn't returned when we
		// call Stop, but short enough to stay inside the 5s flush budget.
		select {
		case <-time.After(400 * time.Millisecond):
		case <-handlerDone:
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	withLangfuseConfig(t, true, srv.URL, "pk-flush", "sk-flush")
	StartLangfuseExporter()

	emitErr := make(chan error, 1)
	go func() {
		emitErr <- EmitLangfuseAudit(minimalSnapshot())
	}()

	// Wait until the handler has actually received the request so we know
	// the in-flight WaitGroup counter is incremented.
	select {
	case <-handlerStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("handler never received the POST")
	}

	// Stop must block until the handler completes (400ms) — proving Flush
	// drained the in-flight request rather than abandoning it.
	stopStart := time.Now()
	StopLangfuseExporter()
	elapsed := time.Since(stopStart)
	if elapsed < 300*time.Millisecond {
		t.Errorf("Stop returned in %v, expected to wait >= 300ms for in-flight drain", elapsed)
	}
	close(handlerDone)

	select {
	case err := <-emitErr:
		if err != nil {
			t.Errorf("in-flight Emit returned err after flush: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("in-flight Emit never returned after flush")
	}
}

// TestLangfuseEmitBeforeStartIsNoop ensures Emit is safe to call before
// StartLangfuseExporter — it must return nil and never attempt a POST.
func TestLangfuseEmitBeforeStartIsNoop(t *testing.T) {
	resetExporter(t)
	defer resetExporter(t)
	srv, st := newCaptureServer(t, http.StatusOK)
	withLangfuseConfig(t, true, srv.URL, "pk", "sk")
	if err := EmitLangfuseAudit(minimalSnapshot()); err != nil {
		t.Errorf("emit before start returned err: %v", err)
	}
	if got := st.count(); got != 0 {
		t.Errorf("server received %d calls, want 0 (exporter not started)", got)
	}
}

// TestLangfuseFlushRespectsTimeout verifies FlushLangfuse honors its context
// deadline when an in-flight POST hangs beyond the budget.
func TestLangfuseFlushRespectsTimeout(t *testing.T) {
	resetExporter(t)
	defer resetExporter(t)

	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block // never resolves until test cleanup
	}))
	t.Cleanup(func() {
		close(block)
		srv.Close()
	})

	withLangfuseConfig(t, true, srv.URL, "pk-to", "sk-to")
	StartLangfuseExporter()
	defer StopLangfuseExporter()

	started := make(chan struct{})
	go func() {
		close(started)
		_ = EmitLangfuseAudit(minimalSnapshot())
	}()
	<-started
	// Give Emit a tick to increment inFlight.
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := flushLangfuse(ctx); err == nil {
		t.Error("flushLangfuse returned nil, want context deadline exceeded")
	}
}

// base64Std is a tiny helper to avoid pulling encoding/base64 into every test
// file's import block; tests use it only for expected-auth construction.
func base64Std(s string) string {
	const tbl = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	// Manual base64 encode (std, padded) to keep imports minimal.
	var out bytes.Buffer
	var buf uint32
	var bits uint
	for i := 0; i < len(s); i++ {
		buf = (buf << 8) | uint32(s[i])
		bits += 8
		for bits >= 6 {
			bits -= 6
			out.WriteByte(tbl[(buf>>bits)&0x3F])
		}
	}
	if bits > 0 {
		buf <<= 6 - bits
		out.WriteByte(tbl[buf&0x3F])
	}
	for out.Len()%4 != 0 {
		out.WriteByte('=')
	}
	return out.String()
}
