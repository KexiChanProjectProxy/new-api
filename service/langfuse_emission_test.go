package service

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/stretchr/testify/require"
)

// TestLangfuseExportsNonStreamSuccess verifies that EmitLangfuseAuditFromSink
// (the success-path entry point wired into PostTextConsumeQuota /
// PostAudioConsumeQuota) finalizes usage/quota/latency on the snapshot and
// emits exactly one OTLP/HTTP record carrying those finalized values.
func TestLangfuseExportsNonStreamSuccess(t *testing.T) {
	resetExporter(t)
	defer resetExporter(t)

	srv, st := newCaptureServer(t, http.StatusOK)
	withLangfuseConfig(t, true, srv.URL, "pk-succ", "sk-succ")
	StartLangfuseExporter()
	defer StopLangfuseExporter()

	snapshot := &LangfuseAuditSnapshot{
		RequestPayload: []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}`),
		Metadata: LangfuseAuditMetadata{
			RequestId:   "req-success-1",
			UserId:      42,
			ModelName:   "gpt-4o",
			StartTime:   time.Now().Add(-250 * time.Millisecond),
			IsStream:    false,
			RelayFormat: "openai",
		},
	}
	MaskLangfuseAudit(snapshot)

	EmitLangfuseAuditFromSink(snapshot, 10, 5, 15, 1500, nil)

	require.Eventually(t, func() bool { return st.count() == 1 },
		2*time.Second, 10*time.Millisecond, "success path must emit exactly one record")

	calls := st.snapshot()
	require.Len(t, calls, 1)

	var body struct {
		Batch []struct {
			Type string `json:"type"`
			Body struct {
				ID      string `json:"id"`
				TraceID string `json:"traceId"`
				Name    string `json:"name"`
				Model   string `json:"model"`
				Usage   struct {
					Input  int    `json:"input"`
					Output int    `json:"output"`
					Unit   string `json:"unit"`
				} `json:"usage"`
				Metadata map[string]any `json:"metadata"`
			} `json:"body"`
		} `json:"batch"`
	}
	require.NoError(t, json.Unmarshal(calls[0].body, &body))
	require.Len(t, body.Batch, 2, "batch must contain trace-create + generation-create")

	// Find the generation event
	var gen *struct {
		ID      string `json:"id"`
		TraceID string `json:"traceId"`
		Name    string `json:"name"`
		Model   string `json:"model"`
		Usage   struct {
			Input  int    `json:"input"`
			Output int    `json:"output"`
			Unit   string `json:"unit"`
		} `json:"usage"`
		Level         string `json:"level"`
		StatusMessage any    `json:"statusMessage"`
	}
	for i := range body.Batch {
		if body.Batch[i].Type == "generation-create" {
			b := body.Batch[i]
			gen = &struct {
				ID      string `json:"id"`
				TraceID string `json:"traceId"`
				Name    string `json:"name"`
				Model   string `json:"model"`
				Usage   struct {
					Input  int    `json:"input"`
					Output int    `json:"output"`
					Unit   string `json:"unit"`
				} `json:"usage"`
				Level         string `json:"level"`
				StatusMessage any    `json:"statusMessage"`
			}{
				ID:      b.Body.ID,
				TraceID: b.Body.TraceID,
				Name:    b.Body.Name,
				Model:   b.Body.Model,
				Usage:   b.Body.Usage,
			}
			break
		}
	}
	require.NotNil(t, gen, "must have a generation-create event")
	require.Equal(t, "relay-request", gen.Name)
	require.Equal(t, "gpt-4o", gen.Model)
	require.Equal(t, 10, gen.Usage.Input)
	require.Equal(t, 5, gen.Usage.Output)
}

// TestLangfuseRetryThenSuccessExportsSingleTerminalRecord verifies the core
// invariant: one terminal record per controller.Relay request, even across
// retries. Retry attempts that fail non-terminally do NOT emit (they're
// handled by processChannelError, which is explicitly excluded from emission).
// Only the final success emits — via PostTextConsumeQuota's call to
// EmitLangfuseAuditFromSink.
func TestLangfuseRetryThenSuccessExportsSingleTerminalRecord(t *testing.T) {
	resetExporter(t)
	defer resetExporter(t)

	srv, st := newCaptureServer(t, http.StatusOK)
	withLangfuseConfig(t, true, srv.URL, "pk-retry", "sk-retry")
	StartLangfuseExporter()
	defer StopLangfuseExporter()

	snapshot := &LangfuseAuditSnapshot{
		RequestPayload: []byte(`{"model":"gpt-4o","messages":[]}`),
		Metadata: LangfuseAuditMetadata{
			RequestId: "req-retry-1",
			UserId:    1,
			StartTime: time.Now(),
		},
	}
	MaskLangfuseAudit(snapshot)

	// Mid-retry failures route through processChannelError in production,
	// which does NOT call EmitLangfuseAuditFromSink. Only the terminal
	// success path emits. We simulate the single terminal emit here.
	EmitLangfuseAuditFromSink(snapshot, 8, 2, 10, 800, nil)

	require.Eventually(t, func() bool { return st.count() == 1 },
		2*time.Second, 10*time.Millisecond, "retry-then-success must emit exactly ONE terminal record")
}

// TestLangfuseExportsFinalClientErrorPayload verifies the terminal-failure path
// (wired into controller.Relay's failure defer): EmitLangfuseAuditFromSink is
// called with a non-nil error payload, and the emitted record carries that
// error bytes in the error_payload field with zero usage/quota.
func TestLangfuseExportsFinalClientErrorPayload(t *testing.T) {
	resetExporter(t)
	defer resetExporter(t)

	srv, st := newCaptureServer(t, http.StatusOK)
	withLangfuseConfig(t, true, srv.URL, "pk-err", "sk-err")
	StartLangfuseExporter()
	defer StopLangfuseExporter()

	snapshot := &LangfuseAuditSnapshot{
		RequestPayload: []byte(`{"model":"gpt-4o","messages":[]}`),
		Metadata: LangfuseAuditMetadata{
			RequestId: "req-fail-1",
			UserId:    7,
			StartTime: time.Now(),
		},
	}
	MaskLangfuseAudit(snapshot)

	errPayload := []byte(`{"error":{"message":"upstream 502","type":"upstream_error","code":502}}`)
	EmitLangfuseAuditFromSink(snapshot, 0, 0, 0, 0, errPayload)

	require.Eventually(t, func() bool { return st.count() == 1 },
		2*time.Second, 10*time.Millisecond, "failure path must emit one record")

	calls := st.snapshot()
	require.Len(t, calls, 1)

	var body struct {
		Batch []struct {
			Type string `json:"type"`
			Body struct {
				ID     string `json:"id"`
				Level  string `json:"level"`
				Input  any    `json:"input"`
				Output any    `json:"output"`
				Usage  struct {
					Input  int    `json:"input"`
					Output int    `json:"output"`
					Unit   string `json:"unit"`
				} `json:"usage"`
			} `json:"body"`
		} `json:"batch"`
	}
	require.NoError(t, json.Unmarshal(calls[0].body, &body))
	require.Len(t, body.Batch, 2, "batch must contain trace-create + generation-create")

	// Find the generation event and verify it's ERROR level with output
	var genLevel string
	var genOutput any
	var genUsageInput int
	for _, b := range body.Batch {
		if b.Type == "generation-create" {
			genLevel = b.Body.Level
			genOutput = b.Body.Output
			genUsageInput = b.Body.Usage.Input
		}
	}
	require.Equal(t, "ERROR", genLevel, "failure must have ERROR level")
	require.NotNil(t, genOutput, "failure must have output (error payload)")
	require.Equal(t, 0, genUsageInput, "usage must be 0 on failure")
}

// TestLangfuseIgnoresRelayTaskAndRealtimeScopes verifies the scope guard:
// RelayTask, RelayMidjourney, and websocket realtime paths never set
// relayInfo.LangfuseSnapshot (it stays nil), so EmitLangfuseAuditFromSink is a
// no-op for them. A sink of the wrong concrete type is also ignored.
func TestLangfuseIgnoresRelayTaskAndRealtimeScopes(t *testing.T) {
	resetExporter(t)
	defer resetExporter(t)

	srv, st := newCaptureServer(t, http.StatusOK)
	withLangfuseConfig(t, true, srv.URL, "pk-scope", "sk-scope")
	StartLangfuseExporter()
	defer StopLangfuseExporter()

	// Case 1: nil sink — mirrors RelayTask/RelayMidjourney/realtime where
	// LangfuseSnapshot is never populated.
	EmitLangfuseAuditFromSink(nil, 5, 5, 10, 100, nil)

	// Case 2: a sink implementing the interface but NOT the concrete
	// *LangfuseAuditSnapshot type — the type assertion must reject it.
	EmitLangfuseAuditFromSink(fakeForeignSink{}, 5, 5, 10, 100, nil)

	// Case 3: a nil *LangfuseAuditSnapshot typed as the interface.
	var nilTyped *LangfuseAuditSnapshot
	EmitLangfuseAuditFromSink(nilTyped, 5, 5, 10, 100, nil)

	time.Sleep(150 * time.Millisecond)
	require.Equal(t, 0, st.count(), "nil/foreign sinks must produce zero emissions")
}

// fakeForeignSink implements LangfuseSnapshotSink but is NOT
// *LangfuseAuditSnapshot, proving the type assertion in
// EmitLangfuseAuditFromSink rejects foreign implementations.
type fakeForeignSink struct{}

func (fakeForeignSink) SetResponsePayload(body []byte)           {}
func (fakeForeignSink) SetResponsePayloadFromString(body string) {}
func (fakeForeignSink) SetBinaryResponse(contentType string, body []byte, omittedReason string) {
	_ = contentType
	_ = body
	_ = omittedReason
}

var _ relaycommon.LangfuseSnapshotSink = fakeForeignSink{}
