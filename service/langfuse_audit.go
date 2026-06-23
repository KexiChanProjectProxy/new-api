package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// langfuseMaxPayloadBytes is the hard cap (64 KiB) applied to each redacted
// request / response / error payload before emission. Anything larger is
// truncated and a marker is appended so downstream consumers know data was
// dropped.
const langfuseMaxPayloadBytes = 64 * 1024

// langfuseTruncationMarker is appended to a truncated payload so consumers
// can detect that truncation occurred. It is intentionally a human-readable
// sentinel rather than a JSON field so it survives even when the payload is
// not valid JSON (e.g. raw text or partial bytes).
const langfuseTruncationMarker = "[truncated]"

// LangfuseTruncationMarkers records original byte lengths and whether each
// payload bucket was truncated. It is embedded in LangfuseAuditSnapshot and
// surfaced in metadata so consumers can render a "truncated" badge without
// scanning for the sentinel.
type LangfuseTruncationMarkers struct {
	RequestTruncated  bool `json:"request_truncated"`
	RequestOriginal   int  `json:"request_original_length"`
	ResponseTruncated bool `json:"response_truncated"`
	ResponseOriginal  int  `json:"response_original_length"`
	ErrorTruncated    bool `json:"error_truncated"`
	ErrorOriginal     int  `json:"error_original_length"`
}

// BinaryPlaceholder replaces a raw binary response body in the audit snapshot.
// Langfuse ingests text/JSON; binary blobs (images, audio, ...) are replaced
// with this struct so trace consumers still get content addressing.
type BinaryPlaceholder struct {
	ContentType   string `json:"content_type"`
	ContentLength int64  `json:"content_length"`
	SHA256        string `json:"sha256"`
	OmittedReason string `json:"omitted_reason"`
}

// LangfuseAuditMetadata is the structured metadata attached to every Langfuse
// observation built from a relay request. All fields are derived from
// RelayInfo / gin.Context and are safe to emit (no raw secrets).
type LangfuseAuditMetadata struct {
	// Timing
	StartTime time.Time `json:"start_time"`

	// Client identity (masked)
	ResolvedClientIP string `json:"resolved_client_ip,omitempty"`
	MaskedTokenKey   string `json:"masked_token_key,omitempty"`
	Username         string `json:"username,omitempty"`
	UserId           int    `json:"user_id"`

	// Correlation IDs
	RequestId         string `json:"request_id,omitempty"`
	UpstreamRequestId string `json:"upstream_request_id,omitempty"`

	// Routing / model
	ModelName  string `json:"model_name,omitempty"`
	GroupName  string `json:"group,omitempty"`
	UsingGroup string `json:"using_group,omitempty"`

	// Channel
	ChannelId         int    `json:"channel_id"`
	ChannelType       int    `json:"channel_type"`
	UpstreamModelName string `json:"upstream_model_name,omitempty"`
	IsModelMapped     bool   `json:"is_model_mapped"`

	// Retry chain (channel IDs tried, in order)
	RetryChain  []int  `json:"retry_chain,omitempty"`
	RetryIndex  int    `json:"retry_index"`
	RelayFormat string `json:"relay_format,omitempty"`

	// Usage / billing (populated at settlement time by Task 4/5; zero-valued
	// at request-snapshot time).
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	Quota            int `json:"quota"`

	// Latency (ms). FirstResponseLatencyMs is -1 until FirstResponseTime is set.
	FirstResponseLatencyMs int64 `json:"first_response_latency_ms"`
	TotalLatencyMs         int64 `json:"total_latency_ms"`

	// Flags
	IsStream     bool `json:"is_stream"`
	IsPlayground bool `json:"is_playground"`
}

// LangfuseAuditSnapshot is the in-memory representation of one relay request
// ready to be turned into Langfuse observations. It is built in three phases:
//
//  1. BuildLangfuseRequestSnapshot — at request validation time, captures the
//     validated client-visible DTO before upstream conversion.
//  2. DeriveLangfuseMetadata — extracts masked metadata from RelayInfo.
//  3. MaskLangfuseAudit — redacts tokens / sensitive strings and enforces the
//     64 KiB size cap on each payload bucket.
//
// Response/error payloads are populated later (Task 4/5) via the SetResponse*
// and SetError helpers.
type LangfuseAuditSnapshot struct {
	RequestPayload    []byte                    `json:"request_payload"`
	ResponsePayload   []byte                    `json:"response_payload,omitempty"`
	ErrorPayload      []byte                    `json:"error_payload,omitempty"`
	BinaryResponse    *BinaryPlaceholder        `json:"binary_response,omitempty"`
	Metadata          LangfuseAuditMetadata     `json:"metadata"`
	TruncationMarkers LangfuseTruncationMarkers `json:"truncation_markers"`
}

// BuildLangfuseRequestSnapshot captures the validated client-visible request
// DTO as JSON, BEFORE any upstream conversion or param override is applied.
// It must be called immediately after GetAndValidateRequest succeeds, using
// relayInfo.Request (which holds the validated dto.Request).
//
// The returned bytes are NOT yet masked or size-capped; call MaskLangfuseAudit
// before emitting. Marshaling failures (e.g. an exotic DTO that cannot be
// serialized) are returned as errors rather than silently dropped so callers
// can decide whether to skip the trace entirely.
func BuildLangfuseRequestSnapshot(req dto.Request) ([]byte, error) {
	if req == nil {
		return nil, fmt.Errorf("langfuse: cannot snapshot nil request")
	}
	raw, err := common.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("langfuse: failed to marshal request snapshot: %w", err)
	}
	return raw, nil
}

// DeriveLangfuseMetadata extracts a fully-masked LangfuseAuditMetadata from a
// RelayInfo and its gin.Context. It is safe to call with a nil context (some
// code paths synthesize RelayInfo without an HTTP request); in that case
// context-derived fields (username, request IDs, retry chain) are left empty.
//
// Token keys are ALWAYS masked via model.MaskTokenKey — the raw key is never
// placed into metadata. Latency fields are computed from StartTime and
// FirstResponseTime; if FirstResponseTime is zero (stream never produced a
// first chunk), FirstResponseLatencyMs is set to -1.
func DeriveLangfuseMetadata(relayInfo *relaycommon.RelayInfo, c *gin.Context) LangfuseAuditMetadata {
	meta := LangfuseAuditMetadata{}

	if relayInfo != nil {
		meta.StartTime = relayInfo.StartTime
		meta.ResolvedClientIP = relayInfo.ResolvedClientIP
		meta.MaskedTokenKey = model.MaskTokenKey(relayInfo.TokenKey)
		meta.UserId = relayInfo.UserId
		// UserEmail is deliberately omitted to avoid PII leakage; Username is pulled from context below.
		meta.ModelName = relayInfo.OriginModelName
		meta.GroupName = relayInfo.TokenGroup
		meta.UsingGroup = relayInfo.UsingGroup
		meta.RequestId = relayInfo.RequestId
		meta.IsStream = relayInfo.IsStream
		meta.IsPlayground = relayInfo.IsPlayground
		meta.RelayFormat = string(relayInfo.RelayFormat)
		meta.RetryIndex = relayInfo.RetryIndex

		if relayInfo.ChannelMeta != nil {
			meta.ChannelId = relayInfo.ChannelId
			meta.ChannelType = relayInfo.ChannelType
			meta.UpstreamModelName = relayInfo.UpstreamModelName
			meta.IsModelMapped = relayInfo.IsModelMapped
		}

		// FirstResponseTime zero-value => stream not yet started / non-stream.
		if !relayInfo.FirstResponseTime.IsZero() {
			meta.FirstResponseLatencyMs = relayInfo.FirstResponseTime.UnixMilli() - relayInfo.StartTime.UnixMilli()
		} else {
			meta.FirstResponseLatencyMs = -1
		}
		// TotalLatencyMs is finalized at settlement time; we seed it with
		// elapsed-so-far so an early-emitted trace still has something useful.
		if !relayInfo.StartTime.IsZero() {
			meta.TotalLatencyMs = time.Since(relayInfo.StartTime).Milliseconds()
		}
	}

	if c != nil {
		if meta.RequestId == "" {
			meta.RequestId = c.GetString(common.RequestIdKey)
		}
		meta.UpstreamRequestId = c.GetString(common.UpstreamRequestIdKey)
		meta.Username = c.GetString("username")
		// use_channel is the ordered list of channel IDs attempted for this
		// request (including retries). It is the canonical "retry chain".
		if chain := c.GetStringSlice("use_channel"); len(chain) > 0 {
			meta.RetryChain = make([]int, 0, len(chain))
			for _, idStr := range chain {
				meta.RetryChain = append(meta.RetryChain, common.String2Int(idStr))
			}
		}
		// If metadata didn't carry a client IP (RelayInfo nil or unset), fall
		// back to the gin context.
		if meta.ResolvedClientIP == "" && c.Request != nil {
			meta.ResolvedClientIP = strings.TrimSpace(c.ClientIP())
		}
	}

	return meta
}

// SetResponsePayload captures a non-streaming response body for later
// emission. The body is stored verbatim; masking + truncation happen in
// MaskLangfuseAudit. Pass the raw upstream response bytes (already buffered
// by the relay layer).
func (s *LangfuseAuditSnapshot) SetResponsePayload(body []byte) {
	if s == nil {
		return
	}
	s.ResponsePayload = body
}

// SetResponsePayloadFromString is a convenience wrapper for text responses.
func (s *LangfuseAuditSnapshot) SetResponsePayloadFromString(body string) {
	if s == nil {
		return
	}
	s.ResponsePayload = []byte(body)
}

// SetBinaryResponse replaces the response payload with a BinaryPlaceholder,
// computing the SHA-256 of the raw bytes so the trace is still content-addressed.
// The raw bytes are never stored on the snapshot. Pass a human-readable reason
// for omission (e.g. "image/png response").
func (s *LangfuseAuditSnapshot) SetBinaryResponse(contentType string, body []byte, omittedReason string) {
	if s == nil {
		return
	}
	sum := sha256.Sum256(body)
	s.BinaryResponse = &BinaryPlaceholder{
		ContentType:   contentType,
		ContentLength: int64(len(body)),
		SHA256:        hex.EncodeToString(sum[:]),
		OmittedReason: omittedReason,
	}
	// Drop any previously-set text response payload — the binary placeholder
	// is authoritative.
	s.ResponsePayload = nil
}

// SetError captures an error payload (typically a marshaled upstream error
// response or a synthesized NewAPIError). Like the response payload, it is
// masked + truncated in MaskLangfuseAudit.
func (s *LangfuseAuditSnapshot) SetError(errPayload []byte) {
	if s == nil {
		return
	}
	s.ErrorPayload = errPayload
}

// SetErrorFromError captures a Go error as the audit error payload. If the
// error implements json.Marshaler or is a *types.NewAPIError, it is serialized
// via common.Marshal; otherwise its Error() string is stored.
func (s *LangfuseAuditSnapshot) SetErrorFromError(err error) {
	if s == nil || err == nil {
		return
	}
	// types.NewAPIError carries structured fields we want to preserve.
	if apiErr, ok := err.(*types.NewAPIError); ok {
		if raw, mErr := common.Marshal(apiErr); mErr == nil {
			s.ErrorPayload = raw
			return
		}
	}
	// Last resort: store the plain message.
	s.ErrorPayload = []byte(err.Error())
}

// SetUsage populates the usage fields in metadata. Called at settlement time.
func (s *LangfuseAuditSnapshot) SetUsage(prompt, completion, total int) {
	if s == nil {
		return
	}
	s.Metadata.PromptTokens = prompt
	s.Metadata.CompletionTokens = completion
	s.Metadata.TotalTokens = total
}

// SetQuota records the final charged quota (in quota units, not dollars).
func (s *LangfuseAuditSnapshot) SetQuota(quota int) {
	if s == nil {
		return
	}
	s.Metadata.Quota = quota
}

// FinalizeLatency fixes TotalLatencyMs to the actual elapsed time from
// StartTime to "now" (or to the optional end argument if non-zero). Called
// at request completion.
func (s *LangfuseAuditSnapshot) FinalizeLatency(end ...time.Time) {
	if s == nil || s.Metadata.StartTime.IsZero() {
		return
	}
	var endTime time.Time
	if len(end) > 0 && !end[0].IsZero() {
		endTime = end[0]
	} else {
		endTime = time.Now()
	}
	s.Metadata.TotalLatencyMs = endTime.UnixMilli() - s.Metadata.StartTime.UnixMilli()
}

// MaskLangfuseAudit applies all redaction + masking + truncation rules to an
// already-populated snapshot in place. It is idempotent: calling it twice
// produces the same output. Specifically it:
//
//  1. Runs the request/response/error payloads through common.MaskSensitiveInfo
//     to redact URLs, IPs, domains, and api_key:... patterns at the string level.
//  2. Enforces the 64 KiB cap on each of request/response/error payloads,
//     recording the original length and a truncation flag in
//     TruncationMarkers.
//  3. Re-masks the token key in metadata (defensive — DeriveLangfuseMetadata
//     already masks, but if a caller hand-built metadata this catches it).
//
// This function does NOT emit anything; it only prepares the snapshot for
// emission by Task 4/5.
func MaskLangfuseAudit(snapshot *LangfuseAuditSnapshot) {
	if snapshot == nil {
		return
	}

	// 1. String-level sensitive-info redaction on each text payload.
	snapshot.RequestPayload = maskBytes(snapshot.RequestPayload)
	snapshot.ResponsePayload = maskBytes(snapshot.ResponsePayload)
	snapshot.ErrorPayload = maskBytes(snapshot.ErrorPayload)

	// 2. 64 KiB cap enforcement with markers.
	snapshot.RequestPayload, snapshot.TruncationMarkers.RequestTruncated, snapshot.TruncationMarkers.RequestOriginal =
		capPayload(snapshot.RequestPayload, langfuseMaxPayloadBytes)
	snapshot.ResponsePayload, snapshot.TruncationMarkers.ResponseTruncated, snapshot.TruncationMarkers.ResponseOriginal =
		capPayload(snapshot.ResponsePayload, langfuseMaxPayloadBytes)
	snapshot.ErrorPayload, snapshot.TruncationMarkers.ErrorTruncated, snapshot.TruncationMarkers.ErrorOriginal =
		capPayload(snapshot.ErrorPayload, langfuseMaxPayloadBytes)

	// 3. Defensive token-key re-mask.
	snapshot.Metadata.MaskedTokenKey = model.MaskTokenKey(snapshot.Metadata.MaskedTokenKey)
}

// maskBytes runs common.MaskSensitiveInfo over a byte payload. It is a no-op
// for empty payloads or payloads that are not valid UTF-8 (binary blobs that
// slipped through should have been routed via SetBinaryResponse).
func maskBytes(in []byte) []byte {
	if len(in) == 0 {
		return in
	}
	// MaskSensitiveInfo is string-oriented; bail on obviously-binary payloads
	// to avoid producing garbage. A simple heuristic: if the payload contains
	// a NUL byte in the first 1KiB, treat it as binary and leave it alone
	// (truncation will still apply).
	scanLen := len(in)
	if scanLen > 1024 {
		scanLen = 1024
	}
	for i := 0; i < scanLen; i++ {
		if in[i] == 0 {
			return in
		}
	}
	masked := common.MaskSensitiveInfo(string(in))
	return []byte(masked)
}

// capPayload enforces the maxBytes cap on a single payload bucket. If the
// payload fits, it is returned unchanged. If it exceeds the cap, it is
// truncated to (maxBytes - len(marker)) and the truncation marker is appended,
// so the final byte length is exactly maxBytes. The original length and a
// truncated=true flag are returned alongside.
//
// The marker is appended as raw bytes (not JSON-encoded) so it is visible
// regardless of payload format.
func capPayload(in []byte, maxBytes int) ([]byte, bool, int) {
	originalLen := len(in)
	if originalLen <= maxBytes {
		return in, false, originalLen
	}
	truncated := make([]byte, 0, maxBytes)
	body := maxBytes - len(langfuseTruncationMarker)
	if body < 0 {
		body = 0
	}
	if body > originalLen {
		body = originalLen
	}
	truncated = append(truncated, in[:body]...)
	truncated = append(truncated, []byte(langfuseTruncationMarker)...)
	return truncated, true, originalLen
}

// BuildLangfuseAudit is a convenience wrapper that runs the three-phase
// build (snapshot -> metadata -> mask) in one call, returning a fully-prepared
// LangfuseAuditSnapshot ready for emission. It is the typical entry point for
// callers that have a RelayInfo + context in hand at request-validation time.
//
// If request marshaling fails, the returned snapshot is nil and the error is
// returned (caller decides whether to skip tracing).
func BuildLangfuseAudit(relayInfo *relaycommon.RelayInfo, c *gin.Context) (*LangfuseAuditSnapshot, error) {
	if relayInfo == nil {
		return nil, fmt.Errorf("langfuse: relayInfo is nil")
	}
	reqRaw, err := BuildLangfuseRequestSnapshot(relayInfo.Request)
	if err != nil {
		return nil, err
	}
	snapshot := &LangfuseAuditSnapshot{
		RequestPayload: reqRaw,
		Metadata:       DeriveLangfuseMetadata(relayInfo, c),
	}
	MaskLangfuseAudit(snapshot)
	return snapshot, nil
}

// IsBinaryContentType reports whether a content-type indicates a binary
// response that should be captured via SetBinaryResponse rather than as raw
// bytes. Used by the relay layer to decide which capture path to take.
func IsBinaryContentType(contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if ct == "" {
		return false
	}
	// Strip any parameters (e.g. "; charset=utf-8").
	if i := strings.Index(ct, ";"); i != -1 {
		ct = strings.TrimSpace(ct[:i])
	}
	switch {
	case strings.HasPrefix(ct, "image/"),
		strings.HasPrefix(ct, "audio/"),
		strings.HasPrefix(ct, "video/"),
		strings.HasPrefix(ct, "application/octet-stream"),
		strings.HasPrefix(ct, "application/zip"),
		strings.HasPrefix(ct, "application/pdf"),
		strings.HasPrefix(ct, "application/x-"):
		return true
	}
	return false
}
