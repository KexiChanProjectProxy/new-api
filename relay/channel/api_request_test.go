package channel

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

type testAdaptor struct {
	requestURL      string
	setupHeaderFunc func(c *gin.Context, header *http.Header, info *relaycommon.RelayInfo) error
}

type testTaskAdaptor struct {
	requestURL  string
	buildHeader func(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error
}

func (a *testAdaptor) Init(info *relaycommon.RelayInfo) {}

func (a *testAdaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return a.requestURL, nil
}

func (a *testAdaptor) SetupRequestHeader(c *gin.Context, header *http.Header, info *relaycommon.RelayInfo) error {
	if a.setupHeaderFunc != nil {
		return a.setupHeaderFunc(c, header, info)
	}
	return nil
}

func (a *testAdaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	return nil, nil
}

func (a *testAdaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, nil
}

func (a *testAdaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return nil, nil
}

func (a *testAdaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	return nil, nil
}

func (a *testAdaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	return nil, nil
}

func (a *testAdaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	return nil, nil
}

func (a *testAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return nil, nil
}

func (a *testAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	return nil, nil
}

func (a *testAdaptor) GetModelList() []string {
	return nil
}

func (a *testAdaptor) GetChannelName() string {
	return "test"
}

func (a *testAdaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ClaudeRequest) (any, error) {
	return nil, nil
}

func (a *testAdaptor) ConvertGeminiRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeminiChatRequest) (any, error) {
	return nil, nil
}

func (a *testTaskAdaptor) Init(info *relaycommon.RelayInfo) {}

func (a *testTaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	return nil
}

func (a *testTaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	return nil
}

func (a *testTaskAdaptor) AdjustBillingOnSubmit(info *relaycommon.RelayInfo, taskData []byte) map[string]float64 {
	return nil
}

func (a *testTaskAdaptor) AdjustBillingOnComplete(task *model.Task, taskResult *relaycommon.TaskInfo) int {
	return 0
}

func (a *testTaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return a.requestURL, nil
}

func (a *testTaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	if a.buildHeader != nil {
		return a.buildHeader(c, req, info)
	}
	return nil
}

func (a *testTaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	return nil, nil
}

func (a *testTaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return nil, nil
}

func (a *testTaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (string, []byte, *dto.TaskError) {
	return "", nil, nil
}

func (a *testTaskAdaptor) GetModelList() []string {
	return nil
}

func (a *testTaskAdaptor) GetChannelName() string {
	return "test-task"
}

func (a *testTaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	return nil, nil
}

func (a *testTaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	return nil, nil
}

func TestProcessHeaderOverride_ChannelTestSkipsPassthroughRules(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Trace-Id", "trace-123")

	info := &relaycommon.RelayInfo{
		IsChannelTest: true,
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"*": "",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Empty(t, headers)
}

func TestProcessHeaderOverride_ChannelTestSkipsClientHeaderPlaceholder(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Trace-Id", "trace-123")

	info := &relaycommon.RelayInfo{
		IsChannelTest: true,
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"X-Upstream-Trace": "{client_header:X-Trace-Id}",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	_, ok := headers["x-upstream-trace"]
	require.False(t, ok)
}

func TestProcessHeaderOverride_NonTestKeepsClientHeaderPlaceholder(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Trace-Id", "trace-123")

	info := &relaycommon.RelayInfo{
		IsChannelTest: false,
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"X-Upstream-Trace": "{client_header:X-Trace-Id}",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Equal(t, "trace-123", headers["x-upstream-trace"])
}

func TestProcessHeaderOverride_RuntimeOverrideIsFinalHeaderMap(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	info := &relaycommon.RelayInfo{
		IsChannelTest:             false,
		UseRuntimeHeadersOverride: true,
		RuntimeHeadersOverride: map[string]any{
			"x-static":  "runtime-value",
			"x-runtime": "runtime-only",
		},
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"X-Static": "legacy-value",
				"X-Legacy": "legacy-only",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Equal(t, "runtime-value", headers["x-static"])
	require.Equal(t, "runtime-only", headers["x-runtime"])
	_, exists := headers["x-legacy"]
	require.False(t, exists)
}

func TestProcessHeaderOverride_PassthroughSkipsAcceptEncoding(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Trace-Id", "trace-123")
	ctx.Request.Header.Set("Accept-Encoding", "gzip")

	info := &relaycommon.RelayInfo{
		IsChannelTest: false,
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"*": "",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Equal(t, "trace-123", headers["x-trace-id"])

	_, hasAcceptEncoding := headers["accept-encoding"]
	require.False(t, hasAcceptEncoding)
}

func TestProcessHeaderOverride_PassHeadersTemplateSetsRuntimeHeaders(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	ctx.Request.Header.Set("Originator", "Codex CLI")
	ctx.Request.Header.Set("Session_id", "sess-123")

	info := &relaycommon.RelayInfo{
		IsChannelTest: false,
		RequestHeaders: map[string]string{
			"Originator": "Codex CLI",
			"Session_id": "sess-123",
		},
		ChannelMeta: &relaycommon.ChannelMeta{
			ParamOverride: map[string]any{
				"operations": []any{
					map[string]any{
						"mode":  "pass_headers",
						"value": []any{"Originator", "Session_id", "X-Codex-Beta-Features"},
					},
				},
			},
			HeadersOverride: map[string]any{
				"X-Static": "legacy-value",
			},
		},
	}

	_, err := relaycommon.ApplyParamOverrideWithRelayInfo([]byte(`{"model":"gpt-4.1"}`), info)
	require.NoError(t, err)
	require.True(t, info.UseRuntimeHeadersOverride)
	require.Equal(t, "Codex CLI", info.RuntimeHeadersOverride["originator"])
	require.Equal(t, "sess-123", info.RuntimeHeadersOverride["session_id"])
	_, exists := info.RuntimeHeadersOverride["x-codex-beta-features"]
	require.False(t, exists)
	require.Equal(t, "legacy-value", info.RuntimeHeadersOverride["x-static"])

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Equal(t, "Codex CLI", headers["originator"])
	require.Equal(t, "sess-123", headers["session_id"])
	_, exists = headers["x-codex-beta-features"]
	require.False(t, exists)

	upstreamReq := httptest.NewRequest(http.MethodPost, "https://example.com/v1/responses", nil)
	applyHeaderOverrideToRequest(upstreamReq, headers)
	require.Equal(t, "Codex CLI", upstreamReq.Header.Get("Originator"))
	require.Equal(t, "sess-123", upstreamReq.Header.Get("Session_id"))
	require.Empty(t, upstreamReq.Header.Get("X-Codex-Beta-Features"))
}

// TestProcessHeaderOverride_PassthroughSkipsXForwardedFor verifies that X-Forwarded-For
// is NOT forwarded via wildcard passthrough (gateway controls the header).
func TestProcessHeaderOverride_PassthroughSkipsXForwardedFor(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	ctx.Request.Header.Set("X-Real-IP", "1.2.3.4")
	ctx.Request.Header.Set("X-Custom", "custom-value")

	info := &relaycommon.RelayInfo{
		IsChannelTest: false,
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"*": "",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	// X-Forwarded-For must be blocked from passthrough
	_, hasXFF := headers["x-forwarded-for"]
	require.False(t, hasXFF, "X-Forwarded-For should not be passed through")
	// X-Real-IP must also be blocked
	_, hasXRI := headers["x-real-ip"]
	require.False(t, hasXRI, "X-Real-IP should not be passed through")
	// Other headers should still be passed through
	require.Equal(t, "custom-value", headers["x-custom"])
}

func TestApplyClientIPForward_Enabled(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	// Simulate a trusted proxy scenario where ClientIP returns a specific IP
	ctx.Request.Header.Set("X-Forwarded-For", "6.7.8.9, 1.2.3.4")

	// Create upstream request
	upstreamReq := httptest.NewRequest(http.MethodPost, "https://example.com/v1/chat/completions", nil)
	// Pre-set X-Forwarded-For to simulate a previous value that should be overwritten
	upstreamReq.Header.Set("X-Forwarded-For", "should-be-overwritten")

	info := &relaycommon.RelayInfo{ResolvedClientIP: ctx.ClientIP(), ChannelMeta: &relaycommon.ChannelMeta{ChannelOtherSettings: dto.ChannelOtherSettings{ForwardClientIP: true}}}
	applyClientIPForward(info, upstreamReq.Header)

	// The gateway XFF should overwrite the previous value
	require.Equal(t, "6.7.8.9", upstreamReq.Header.Get("X-Forwarded-For"))
}

func TestApplyClientIPForward_Disabled(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Forwarded-For", "6.7.8.9")

	// Create upstream request
	upstreamReq := httptest.NewRequest(http.MethodPost, "https://example.com/v1/chat/completions", nil)
	// Pre-set X-Forwarded-For
	upstreamReq.Header.Set("X-Forwarded-For", "original-value")

	info := &relaycommon.RelayInfo{ResolvedClientIP: ctx.ClientIP(), ChannelMeta: &relaycommon.ChannelMeta{ChannelOtherSettings: dto.ChannelOtherSettings{ForwardClientIP: false}}}
	applyClientIPForward(info, upstreamReq.Header)

	// The header should remain unchanged
	require.Equal(t, "original-value", upstreamReq.Header.Get("X-Forwarded-For"))
}

func TestApplyClientIPForward_GatewayValueIsFinal(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Forwarded-For", "6.7.8.9")

	// Create upstream request
	upstreamReq := httptest.NewRequest(http.MethodPost, "https://example.com/v1/chat/completions", nil)

	upstreamReq.Header.Set("X-Forwarded-For", "channel-adapted-value")

	info := &relaycommon.RelayInfo{ResolvedClientIP: ctx.ClientIP(), ChannelMeta: &relaycommon.ChannelMeta{ChannelOtherSettings: dto.ChannelOtherSettings{ForwardClientIP: true}}}
	applyClientIPForward(info, upstreamReq.Header)

	// Gateway value must be the final value
	require.Equal(t, "6.7.8.9", upstreamReq.Header.Get("X-Forwarded-For"))
}

func TestApplyClientIPForward_NilInfoDoesNotPanic(t *testing.T) {
	t.Parallel()

	upstreamReq := httptest.NewRequest(http.MethodPost, "https://example.com/v1/chat/completions", nil)
	applyClientIPForward(nil, upstreamReq.Header)
	// Header should remain unset
	require.Empty(t, upstreamReq.Header.Get("X-Forwarded-For"))
}

func TestApplyClientIPForward_NilHeaderDoesNotPanic(t *testing.T) {
	t.Parallel()

	info := &relaycommon.RelayInfo{ResolvedClientIP: "6.7.8.9", ChannelMeta: &relaycommon.ChannelMeta{ChannelOtherSettings: dto.ChannelOtherSettings{ForwardClientIP: true}}}
	applyClientIPForward(info, nil)
}

func TestDoWssRequest_AppliesGatewayXFFToUpgradeRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/realtime", nil)
	ctx.Request.Header.Set("X-Forwarded-For", "6.7.8.9, 1.2.3.4")

	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "6.7.8.9", r.Header.Get("X-Forwarded-For"))
		require.Equal(t, "override-value", r.Header.Get("X-Test-Header"))
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		_ = conn.Close()
	}))
	defer server.Close()

	adaptor := &testAdaptor{
		requestURL: strings.Replace(server.URL, "http://", "ws://", 1),
		setupHeaderFunc: func(c *gin.Context, header *http.Header, info *relaycommon.RelayInfo) error {
			header.Set("X-Test-Header", "setup-value")
			return nil
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelOtherSettings: dto.ChannelOtherSettings{ForwardClientIP: true},
			HeadersOverride: map[string]any{
				"X-Test-Header": "override-value",
			},
		},
		ResolvedClientIP: ctx.ClientIP(),
	}

	conn, err := DoWssRequest(adaptor, ctx, info, nil)
	require.NoError(t, err)
	require.NotNil(t, conn)
	_ = conn.Close()
}

func TestDoRequest_AppliesGatewayXFFToDirectUpstreamRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/stream", strings.NewReader(""))
	ctx.Request.Header.Set("X-Forwarded-For", "6.7.8.9, 1.2.3.4")
	info := &relaycommon.RelayInfo{
		ChannelMeta:      &relaycommon.ChannelMeta{ChannelOtherSettings: dto.ChannelOtherSettings{ForwardClientIP: true}},
		ResolvedClientIP: ctx.ClientIP(),
	}
	require.NoError(t, service.InitHttpClient())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "6.7.8.9", r.Header.Get("X-Forwarded-For"))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL, strings.NewReader(""))
	require.NoError(t, err)
	req.Header.Set("X-Forwarded-For", "should-be-overwritten")

	resp, err := DoRequest(ctx, req, info)
	require.NoError(t, err)
	require.NotNil(t, resp)
	_ = resp.Body.Close()
}

func TestDoTaskApiRequest_AppliesClientIPForward(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/tasks", strings.NewReader(`{}`))
	ctx.Request.Header.Set("X-Forwarded-For", "6.7.8.9, 1.2.3.4")

	adaptor := &testTaskAdaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta:      &relaycommon.ChannelMeta{ChannelOtherSettings: dto.ChannelOtherSettings{ForwardClientIP: true}},
		ResolvedClientIP: ctx.ClientIP(),
	}
	require.NoError(t, service.InitHttpClient())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "6.7.8.9", r.Header.Get("X-Forwarded-For"))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	adaptor.requestURL = server.URL

	resp, err := DoTaskApiRequest(adaptor, ctx, info, strings.NewReader(`{"prompt":"hello"}`))
	require.NoError(t, err)
	require.NotNil(t, resp)
	_ = resp.Body.Close()
}

// =============================================================================
// Regression Tests: XFF Forwarding Precedence (per-channel)
// These tests verify the full XFF forwarding chain for HTTP, Form, WebSocket paths.
// =============================================================================

// TestXFFPrecedence_HTTPPath_ForwardXFFEnabled verifies that when a channel has
func TestXFFPrecedence_HTTPPath_ForwardXFFEnabled(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Forwarded-For", "203.0.113.99, 10.0.0.1")

	info := &relaycommon.RelayInfo{
		ChannelMeta:      &relaycommon.ChannelMeta{ChannelOtherSettings: dto.ChannelOtherSettings{ForwardClientIP: true}},
		ResolvedClientIP: ctx.ClientIP(),
	}
	require.NoError(t, service.InitHttpClient())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// When enabled, gateway XFF should be present and be the client IP
		xff := r.Header.Get("X-Forwarded-For")
		require.NotEmpty(t, xff, "X-Forwarded-For should be set when ForwardClientIP=true")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(`{"model":"gpt-4"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := DoRequest(ctx, req, info)
	require.NoError(t, err)
	require.NotNil(t, resp)
	_ = resp.Body.Close()
}

// TestXFFPrecedence_HTTPPath_ForwardXFFDisabled verifies that when a channel has
func TestXFFPrecedence_HTTPPath_ForwardXFFDisabled(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Forwarded-For", "203.0.113.99")

	info := &relaycommon.RelayInfo{
		ChannelMeta:      &relaycommon.ChannelMeta{ChannelOtherSettings: dto.ChannelOtherSettings{ForwardClientIP: false}},
		ResolvedClientIP: ctx.ClientIP(),
	}
	require.NoError(t, service.InitHttpClient())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// When disabled, X-Forwarded-For should NOT be set by gateway
		xff := r.Header.Get("X-Forwarded-For")
		require.Empty(t, xff, "X-Forwarded-For should NOT be set when ForwardClientIP=false")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(`{"model":"gpt-4"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := DoRequest(ctx, req, info)
	require.NoError(t, err)
	require.NotNil(t, resp)
	_ = resp.Body.Close()
}

// TestXFFPrecedence_RuntimeOverrideDoesNotReplaceGatewayXFF verifies that runtime
// header overrides cannot replace gateway-controlled outbound XFF when the feature is enabled.
func TestXFFPrecedence_RuntimeOverrideDoesNotReplaceGatewayXFF(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Forwarded-For", "203.0.113.50")

	info := &relaycommon.RelayInfo{
		IsChannelTest: false,
		// Runtime headers override attempts to set XFF before gateway applies it
		UseRuntimeHeadersOverride: true,
		RuntimeHeadersOverride: map[string]any{
			"x-forwarded-for": "spoofed-value",
		},
		ChannelMeta:      &relaycommon.ChannelMeta{ChannelOtherSettings: dto.ChannelOtherSettings{ForwardClientIP: true}},
		ResolvedClientIP: ctx.ClientIP(),
	}
	require.NoError(t, service.InitHttpClient())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		xff := r.Header.Get("X-Forwarded-For")
		// Gateway XFF must be final - not the runtime override value
		require.NotEqual(t, "spoofed-value", xff, "Runtime override should not replace gateway XFF")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(`{"model":"gpt-4"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := DoRequest(ctx, req, info)
	require.NoError(t, err)
	require.NotNil(t, resp)
	_ = resp.Body.Close()
}

// TestXFFPrecedence_WebSocket_ForwardXFFEnabled verifies that when a channel has
func TestXFFPrecedence_WebSocket_ForwardXFFEnabled(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/realtime", nil)
	ctx.Request.Header.Set("X-Forwarded-For", "203.0.113.99, 10.0.0.1")

	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		xff := r.Header.Get("X-Forwarded-For")
		require.NotEmpty(t, xff, "WebSocket upgrade should have X-Forwarded-For when enabled")
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		_ = conn.Close()
	}))
	defer server.Close()

	adaptor := &testAdaptor{
		requestURL: strings.Replace(server.URL, "http://", "ws://", 1),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta:      &relaycommon.ChannelMeta{ChannelOtherSettings: dto.ChannelOtherSettings{ForwardClientIP: true}},
		ResolvedClientIP: ctx.ClientIP(),
	}

	conn, err := DoWssRequest(adaptor, ctx, info, nil)
	require.NoError(t, err)
	require.NotNil(t, conn)
	_ = conn.Close()
}

// TestXFFPrecedence_WebSocket_ForwardXFFDisabled verifies that when a channel has
func TestXFFPrecedence_WebSocket_ForwardXFFDisabled(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/realtime", nil)
	ctx.Request.Header.Set("X-Forwarded-For", "203.0.113.99")

	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		xff := r.Header.Get("X-Forwarded-For")
		require.Empty(t, xff, "WebSocket upgrade should NOT have X-Forwarded-For when disabled")
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		_ = conn.Close()
	}))
	defer server.Close()

	adaptor := &testAdaptor{
		requestURL: strings.Replace(server.URL, "http://", "ws://", 1),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta:      &relaycommon.ChannelMeta{ChannelOtherSettings: dto.ChannelOtherSettings{ForwardClientIP: false}},
		ResolvedClientIP: ctx.ClientIP(),
	}

	conn, err := DoWssRequest(adaptor, ctx, info, nil)
	require.NoError(t, err)
	require.NotNil(t, conn)
	_ = conn.Close()
}

// =============================================================================
// Regression Tests: Per-Channel XFF Toggle
// These tests verify that different channels can have different XFF settings
// and that requests via each channel respect their own setting.
// =============================================================================

// TestPerChannelXFFToggle_ChannelAEnabled_ChannelBDisabled verifies that Channel A
func TestPerChannelXFFToggle_ChannelAEnabled_ChannelBDisabled(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	require.NoError(t, service.InitHttpClient())

	// Server that tracks received XFF headers
	type xffResult struct {
		mu  sync.Mutex
		val string
	}
	var result xffResult

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result.mu.Lock()
		result.val = r.Header.Get("X-Forwarded-For")
		result.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channelA := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelOtherSettings: dto.ChannelOtherSettings{ForwardClientIP: true}},
	}

	channelB := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelOtherSettings: dto.ChannelOtherSettings{ForwardClientIP: false}},
	}

	// Make request via Channel A
	ctxA, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctxA.Request = httptest.NewRequest(http.MethodPost, server.URL, strings.NewReader(`{}`))
	ctxA.Request.Header.Set("X-Forwarded-For", "203.0.113.50")
	channelA.ResolvedClientIP = ctxA.ClientIP()

	reqA, _ := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(`{}`))
	respA, err := DoRequest(ctxA, reqA, channelA)
	require.NoError(t, err)
	_ = respA.Body.Close()

	// Make request via Channel B
	ctxB, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctxB.Request = httptest.NewRequest(http.MethodPost, server.URL, strings.NewReader(`{}`))
	ctxB.Request.Header.Set("X-Forwarded-For", "203.0.113.50")
	channelB.ResolvedClientIP = ctxB.ClientIP()

	reqB, _ := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(`{}`))
	respB, err := DoRequest(ctxB, reqB, channelB)
	require.NoError(t, err)
	_ = respB.Body.Close()

	result.mu.Lock()
	xffVal := result.val
	result.mu.Unlock()

	// The last request was via Channel B (ForwardXFF=false), so XFF should be empty
	require.Empty(t, xffVal, "Channel with ForwardClientIP=false should not send XFF")
}

func TestPerChannelXFFToggle_DefaultIsFalse(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	require.NoError(t, service.InitHttpClient())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		xff := r.Header.Get("X-Forwarded-For")
		require.Empty(t, xff, "Default (zero value) should be false, no XFF sent")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Channel with zero value (default)
	channelDefault := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelOtherSettings: dto.ChannelOtherSettings{}},
	}

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, server.URL, strings.NewReader(`{}`))
	ctx.Request.Header.Set("X-Forwarded-For", "203.0.113.50")
	channelDefault.ResolvedClientIP = ctx.ClientIP()

	req, _ := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(`{}`))
	resp, err := DoRequest(ctx, req, channelDefault)
	require.NoError(t, err)
	_ = resp.Body.Close()
}

// =============================================================================
// Regression Tests: Startup Address Resolution
// These tests verify the precedence: custom addr > PORT env > -port flag
// =============================================================================

// TestListenAddressPrecedence_CustomTakesPrecedence verifies that a custom listen
// address takes precedence over PORT env and port flag.
func TestListenAddressPrecedence_CustomTakesPrecedence(t *testing.T) {
	t.Parallel()

	// Custom address is set → should use custom
	addr, err := common.ResolveBindAddress("0.0.0.0:9000", "8080", 3000)
	require.NoError(t, err)
	require.Equal(t, "0.0.0.0:9000", addr)
}

// TestListenAddressPrecedence_PortEnvTakesPrecedenceOverFlag verifies that when
// custom is empty, PORT env takes precedence over port flag.
func TestListenAddressPrecedence_PortEnvTakesPrecedenceOverFlag(t *testing.T) {
	t.Parallel()

	// Custom empty, PORT set → should use PORT
	addr, err := common.ResolveBindAddress("", "8080", 3000)
	require.NoError(t, err)
	require.Equal(t, ":8080", addr)
}

// TestListenAddressPrecedence_FlagUsedWhenBothEmpty verifies that when both custom
// and PORT are empty, the port flag is used.
func TestListenAddressPrecedence_FlagUsedWhenBothEmpty(t *testing.T) {
	t.Parallel()

	// Both empty → should use flag
	addr, err := common.ResolveBindAddress("", "", 3000)
	require.NoError(t, err)
	require.Equal(t, ":3000", addr)
}

// TestListenAddressPrecedence_InvalidCustomAddressCauseFatal verifies that an
// invalid custom address causes an error (not a panic).
func TestListenAddressPrecedence_InvalidCustomAddressCauseFatal(t *testing.T) {
	t.Parallel()

	// Invalid custom (no port) should error
	_, err := common.ResolveBindAddress("127.0.0.1", "", 3000)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid custom listen address")
}

// TestListenAddressValidation_ValidIPv6Forms verifies that valid IPv6 address
// forms are accepted.
func TestListenAddressValidation_ValidIPv6Forms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		address string
	}{
		{address: "[::]:3000"},
		{address: "[::1]:3000"},
		{address: "[2001:db8::1]:3000"},
	}

	for _, tt := range tests {
		t.Run(tt.address, func(t *testing.T) {
			err := common.ValidateListenAddress(tt.address)
			require.NoError(t, err, "IPv6 address %s should be valid", tt.address)
		})
	}
}

// TestListenAddressValidation_InvalidForms verifies that invalid address forms
// are rejected.
func TestListenAddressValidation_InvalidForms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		address string
	}{
		{address: "127.0.0.1"},     // no port
		{address: "example.com"},   // no port
		{address: "bad:addr:3000"}, // too many colons
		{address: "3000"},          // bare port number (needs colon prefix)
		{address: "[::1:3000"},     // unclosed bracket
	}

	for _, tt := range tests {
		t.Run(tt.address, func(t *testing.T) {
			err := common.ValidateListenAddress(tt.address)
			require.Error(t, err, "Address %s should be invalid", tt.address)
		})
	}
}
