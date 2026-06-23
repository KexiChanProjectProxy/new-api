package openai

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// fakeLangfuseSnapshot records the last capture call for assertion.
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

func newCaptureTestContext(t *testing.T) (*gin.Context, *relaycommon.RelayInfo, *fakeLangfuseSnapshot) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(""))
	snap := &fakeLangfuseSnapshot{}
	info := &relaycommon.RelayInfo{
		OriginModelName:  "gpt-4o",
		RelayFormat:      types.RelayFormatOpenAI,
		LangfuseSnapshot: snap,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-4o",
		},
	}
	return c, info, snap
}

// TestLangfuseCapturesFinalNonStreamResponse verifies that OpenaiHandler
// (non-stream) writes the exact final response body bytes onto the snapshot
// AFTER format conversion / usage patching, before billing runs.
func TestLangfuseCapturesFinalNonStreamResponse(t *testing.T) {
	c, info, snap := newCaptureTestContext(t)

	finalBody := `{"id":"chatcmpl-1","object":"chat.completion","model":"gpt-4o","choices":[{"index":0,"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":1,"total_tokens":4}}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader([]byte(finalBody))),
	}

	usage, apiErr := OpenaiHandler(c, info, resp)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	require.Equal(t, 3, usage.PromptTokens)

	require.Equal(t, []byte(finalBody), snap.payload, "snapshot must capture the exact final body bytes")
	require.False(t, snap.gotBinary, "non-stream text response must not be captured as binary")
}

// TestLangfuseExportsOnlyFinalStreamOutcome verifies that OaiStreamHandler
// captures ONLY the normalized terminal stream payload (the final SSE chunk
// with usage + finish_reason), never per-token deltas.
func TestLangfuseExportsOnlyFinalStreamOutcome(t *testing.T) {
	if constant.StreamingTimeout <= 0 {
		constant.StreamingTimeout = 30
		defer func() { constant.StreamingTimeout = 0 }()
	}
	c, info, snap := newCaptureTestContext(t)
	info.IsStream = true
	info.ShouldIncludeUsage = true

	// Build a minimal SSE stream: one delta chunk then a final chunk with usage.
	deltaChunk := `data: {"id":"chatcmpl-1","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"content":"he"}}]}`
	finalChunk := `data: {"id":"chatcmpl-1","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":2,"completion_tokens":1,"total_tokens":3}}`
	sseBody := deltaChunk + "\n\n" + finalChunk + "\n\ndata: [DONE]\n\n"

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(sseBody)),
	}

	usage, apiErr := OaiStreamHandler(c, info, resp)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)

	require.NotEmpty(t, snap.payload, "snapshot must capture the terminal stream payload")
	require.NotContains(t, string(snap.payload), `"content":"he"`, "per-token delta must NOT be forwarded")
	require.Contains(t, string(snap.payload), `"finish_reason":"stop"`, "terminal payload must carry finish_reason")
}

// TestLangfuseCapturesResponsesCompletedPayload verifies that
// OaiResponsesStreamHandler captures ONLY the response.completed terminal
// event for the Responses API stream, not per-token deltas.
func TestLangfuseCapturesResponsesCompletedPayload(t *testing.T) {
	if constant.StreamingTimeout <= 0 {
		constant.StreamingTimeout = 30
		defer func() { constant.StreamingTimeout = 0 }()
	}
	c, info, snap := newCaptureTestContext(t)
	info.IsStream = true

	deltaEvt := `data: {"type":"response.output_text.delta","delta":"hel"}`
	completedEvt := `data: {"type":"response.completed","response":{"id":"resp_1","usage":{"input_tokens":2,"output_tokens":1,"total_tokens":3}}}`
	sseBody := deltaEvt + "\n\n" + completedEvt + "\n\ndata: [DONE]\n\n"

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(sseBody)),
	}

	usage, apiErr := OaiResponsesStreamHandler(c, info, resp)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)

	require.NotEmpty(t, snap.payload, "snapshot must capture response.completed payload")
	require.NotContains(t, string(snap.payload), `"delta":"hel"`, "per-token delta must NOT be forwarded")
	require.Contains(t, string(snap.payload), `"type":"response.completed"`, "must capture the completed event")
	require.Contains(t, string(snap.payload), `"input_tokens":2`, "terminal payload must carry usage")
}

// TestLangfuseStoresBinaryResponsePlaceholder verifies that OpenaiTTSHandler
// captures audio output via SetBinaryResponse (content_type + sha256 placeholder),
// never the raw audio bytes.
func TestLangfuseStoresBinaryResponsePlaceholder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/speech", strings.NewReader(""))

	snap := &fakeLangfuseSnapshot{}
	info := &relaycommon.RelayInfo{
		OriginModelName:  "tts-1",
		RelayFormat:      types.RelayFormatOpenAIAudio,
		LangfuseSnapshot: snap,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "tts-1",
		},
	}
	info.Request = &dto.AudioRequest{Model: "tts-1", ResponseFormat: "mp3"}

	audioBytes := []byte("ID3FAKEAUDIOBYTESFORTEST")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"audio/mpeg"}},
		Body:       io.NopCloser(bytes.NewReader(audioBytes)),
	}

	usage := OpenaiTTSHandler(c, resp, info)
	require.NotNil(t, usage)

	require.True(t, snap.gotBinary, "audio response must be captured as binary placeholder")
	require.Equal(t, "audio/mpeg", snap.binaryCT)
	require.Equal(t, audioBytes, snap.binaryBody, "binary body bytes are passed for sha256 hashing")
	require.NotEmpty(t, snap.binaryReason, "omitted reason must be set")
	require.Nil(t, snap.payload, "raw binary bytes must NOT leak into the text payload bucket")
}
