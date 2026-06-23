package claude

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeLangfuseSnapshot struct {
	payload      []byte
	binaryCT     string
	binaryBody   []byte
	binaryReason string
	gotBinary    bool
}

func (f *fakeLangfuseSnapshot) SetResponsePayload(body []byte) {
	f.payload = append([]byte(nil), body...)
}

func (f *fakeLangfuseSnapshot) SetResponsePayloadFromString(body string) {
	f.payload = []byte(body)
}

func (f *fakeLangfuseSnapshot) SetBinaryResponse(contentType string, body []byte, omittedReason string) {
	f.binaryCT = contentType
	f.binaryBody = append([]byte(nil), body...)
	f.binaryReason = omittedReason
	f.gotBinary = true
}

// TestLangfuseExportsClaudeFinalStreamOutcome verifies ClaudeStreamHandler
// captures ONLY the terminal message_delta event (the SSE chunk that sets
// Done=true), never the per-token text deltas.
func TestLangfuseExportsClaudeFinalStreamOutcome(t *testing.T) {
	if constant.StreamingTimeout <= 0 {
		constant.StreamingTimeout = 30
		defer func() { constant.StreamingTimeout = 0 }()
	}
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(""))

	snap := &fakeLangfuseSnapshot{}
	info := &relaycommon.RelayInfo{
		OriginModelName:  "claude-3-5-sonnet",
		RelayFormat:      types.RelayFormatClaude,
		IsStream:         true,
		LangfuseSnapshot: snap,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "claude-3-5-sonnet",
		},
	}

	// Build a minimal Claude SSE stream: message_start, one content_block_delta,
	// and the terminal message_delta carrying usage.
	startEvt := `event: message_start
data: {"type":"message_start","message":{"id":"msg_1","model":"claude-3-5-sonnet","usage":{"input_tokens":5,"output_tokens":1}}}`

	deltaEvt := `event: content_block_delta
data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"hel"}}`

	finalEvt := `event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"input_tokens":5,"output_tokens":2}}`

	sseBody := startEvt + "\n\n" + deltaEvt + "\n\n" + finalEvt + "\n\n"

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(sseBody)),
	}

	usage, apiErr := ClaudeStreamHandler(c, resp, info)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)

	require.NotEmpty(t, snap.payload, "snapshot must capture the terminal message_delta payload")
	require.NotContains(t, string(snap.payload), `"text":"hel"`, "per-token text delta must NOT be forwarded")
	require.Contains(t, string(snap.payload), `"type":"message_delta"`, "must capture the message_delta event")
	require.Contains(t, string(snap.payload), `"stop_reason":"end_turn"`, "terminal payload must carry stop_reason")
}
