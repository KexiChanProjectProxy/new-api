package mcp

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	service.InitHttpClient()
	gin.SetMode(gin.TestMode)
	m.Run()
}

func mockMCPServer(t *testing.T) *httptest.Server {
	_ = t
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			contentType := r.Header.Get("Content-Type")
			accept := r.Header.Get("Accept")

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read body", http.StatusBadRequest)
				return
			}

			method := ExtractMCPMethod(body)

			if strings.Contains(accept, "text/event-stream") || strings.Contains(contentType, "text/event-stream") {
				w.Header().Set("Content-Type", "text/event-stream")
				w.Header().Set("MCP-Session-Id", "sse-session-123")
				w.WriteHeader(http.StatusOK)

				events := []string{
					"event: message\ndata: {\"type\":\"tool_call\",\"id\":\"call_1\"}\n\n",
					"event: message\ndata: {\"type\":\"progress\",\"progress\":50}\n\n",
					"event: message\ndata: {\"type\":\"result\",\"result\":\"done\"}\n\n",
				}
				for _, event := range events {
					_, _ = w.Write([]byte(event))
					if f, ok := w.(http.Flusher); ok {
						f.Flush()
					}
				}
				return
			}

			switch method {
			case MethodInitialize:
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("MCP-Session-Id", "test-session-123")
				w.Header().Set("MCP-Protocol-Version", "2025-03-26")
				w.WriteHeader(http.StatusOK)
				response := `{"jsonrpc":"2.0","id":null,"result":{"protocolVersion":"2025-03-26","capabilities":{"tools":{}},"serverInfo":{"name":"mock-mcp-server","version":"1.0.0"}}}`
				_, _ = w.Write([]byte(response))

			case MethodToolsCall:
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				response := `{"jsonrpc":"2.0","id":"call_123","result":{"content":[{"type":"text","text":"tool call successful"}]}}`
				_, _ = w.Write([]byte(response))

			case MethodPing:
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				response := `{"jsonrpc":"2.0","id":null,"result":{}}`
				_, _ = w.Write([]byte(response))

			default:
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				response := `{"jsonrpc":"2.0","id":null,"error":{"code":-32601,"message":"Method not found: ` + method + `"}}`
				_, _ = w.Write([]byte(response))
			}
			return
		}

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}))
}

func createTestContext(method, target string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}
	ctx.Request = httptest.NewRequest(method, target, reader)
	if body != nil {
		ctx.Request.Header.Set("Content-Type", "application/json")
	}
	return ctx, recorder
}

func buildRelayInfo(baseURL string) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: baseURL,
			ApiKey:        "test-api-key",
		},
	}
}

func TestMCPIntegration_Initialize(t *testing.T) {
	server := mockMCPServer(t)
	defer server.Close()

	adaptor := &Adaptor{}
	info := buildRelayInfo(server.URL + "/")
	adaptor.Init(info)

	assert.True(t, info.IsStream, "IsStream should be true after Init")

	initializeReq := `{"jsonrpc":"2.0","id":null,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}`
	ctx, recorder := createTestContext(http.MethodPost, "/mcp", []byte(initializeReq))
	ctx.Request.Header.Set("Accept", "application/json")

	respInterface, err := adaptor.DoRequest(ctx, info, bytes.NewReader([]byte(initializeReq)))
	require.NoError(t, err, "DoRequest should not fail")
	require.NotNil(t, respInterface, "Response should not be nil")
	resp := respInterface.(*http.Response)

	_, errResp := adaptor.DoResponse(ctx, resp, info)
	assert.Nil(t, errResp, "DoResponse should not return error")

	assert.Equal(t, http.StatusOK, recorder.Code, "Status should be 200")
	assert.Equal(t, "test-session-123", recorder.Header().Get("MCP-Session-Id"), "MCP-Session-Id should be forwarded")
	assert.Equal(t, "2025-03-26", recorder.Header().Get("MCP-Protocol-Version"), "MCP-Protocol-Version should be forwarded")

	body := recorder.Body.String()
	assert.Contains(t, body, "mock-mcp-server", "Response should contain server info")
	assert.Contains(t, body, "protocolVersion", "Response should contain protocol version")
	assert.False(t, IsBillableMethod(MethodInitialize), "initialize should be non-billable")
}

func TestMCPIntegration_ToolsCall(t *testing.T) {
	server := mockMCPServer(t)
	defer server.Close()

	adaptor := &Adaptor{}
	info := buildRelayInfo(server.URL + "/")
	adaptor.Init(info)

	toolsCallReq := `{"jsonrpc":"2.0","id":"call_123","method":"tools/call","params":{"name":"test_tool","arguments":{"arg1":"value1"}}}`
	ctx, recorder := createTestContext(http.MethodPost, "/mcp", []byte(toolsCallReq))
	ctx.Request.Header.Set("Accept", "application/json")

	respInterface, err := adaptor.DoRequest(ctx, info, bytes.NewReader([]byte(toolsCallReq)))
	require.NoError(t, err, "DoRequest should not fail")
	require.NotNil(t, respInterface, "Response should not be nil")
	resp := respInterface.(*http.Response)

	_, errResp := adaptor.DoResponse(ctx, resp, info)
	assert.Nil(t, errResp, "DoResponse should not fail")

	assert.Equal(t, http.StatusOK, recorder.Code, "Status should be 200")

	body := recorder.Body.String()
	assert.Contains(t, body, "tool call successful", "Response should contain tool call result")
	assert.Contains(t, body, "call_123", "Response should contain call ID")
	assert.True(t, IsBillableMethod(MethodToolsCall), "tools/call should be billable")
}

func TestMCPIntegration_SSEStreaming(t *testing.T) {
	server := mockMCPServer(t)
	defer server.Close()

	adaptor := &Adaptor{}
	info := buildRelayInfo(server.URL + "/")
	adaptor.Init(info)

	toolsCallReq := `{"jsonrpc":"2.0","id":"call_sse","method":"tools/call","params":{"name":"test_tool"}}`
	ctx, recorder := createTestContext(http.MethodPost, "/mcp", []byte(toolsCallReq))
	ctx.Request.Header.Set("Accept", "text/event-stream")

	respInterface, err := adaptor.DoRequest(ctx, info, bytes.NewReader([]byte(toolsCallReq)))
	require.NoError(t, err, "DoRequest should not fail")
	require.NotNil(t, respInterface, "Response should not be nil")
	resp := respInterface.(*http.Response)

	_, errResp := adaptor.DoResponse(ctx, resp, info)
	assert.Nil(t, errResp, "DoResponse should not fail")

	assert.Equal(t, "text/event-stream", recorder.Header().Get("Content-Type"), "Content-Type should be text/event-stream")
	assert.Equal(t, "sse-session-123", recorder.Header().Get("MCP-Session-Id"), "MCP-Session-Id should be forwarded")

	body := recorder.Body.String()
	assert.Contains(t, body, "tool_call", "Response should contain tool_call event")
	assert.Contains(t, body, "progress", "Response should contain progress event")
	assert.Contains(t, body, "result", "Response should contain result event")
}

func TestMCPIntegration_UnknownMethod(t *testing.T) {
	server := mockMCPServer(t)
	defer server.Close()

	adaptor := &Adaptor{}
	info := buildRelayInfo(server.URL + "/")
	adaptor.Init(info)

	unknownReq := `{"jsonrpc":"2.0","id":null,"method":"unknown/method","params":{}}`
	ctx, recorder := createTestContext(http.MethodPost, "/mcp", []byte(unknownReq))
	ctx.Request.Header.Set("Accept", "application/json")

	respInterface, err := adaptor.DoRequest(ctx, info, bytes.NewReader([]byte(unknownReq)))
	require.NoError(t, err, "DoRequest should not fail")
	require.NotNil(t, respInterface, "Response should not be nil")
	resp := respInterface.(*http.Response)

	_, errResp := adaptor.DoResponse(ctx, resp, info)
	assert.Nil(t, errResp, "DoResponse should not fail")

	assert.Equal(t, http.StatusOK, recorder.Code, "Status should be 200 for JSON-RPC error response")

	body := recorder.Body.String()
	assert.Contains(t, body, "Method not found", "Response should contain error message")
	assert.Contains(t, body, "-32601", "Response should contain error code")
}

func TestMCPIntegration_HeaderForwarding(t *testing.T) {
	server := mockMCPServer(t)
	defer server.Close()

	adaptor := &Adaptor{}
	info := buildRelayInfo(server.URL + "/")
	adaptor.Init(info)

	initializeReq := `{"jsonrpc":"2.0","id":null,"method":"initialize","params":{}}`
	ctx, recorder := createTestContext(http.MethodPost, "/mcp", []byte(initializeReq))
	ctx.Request.Header.Set("Accept", "application/json")
	ctx.Request.Header.Set("MCP-Session-Id", "client-session-456")
	ctx.Request.Header.Set("MCP-Protocol-Version", "2025-03-26")
	ctx.Request.Header.Set("Last-Event-ID", "event-789")

	respInterface, err := adaptor.DoRequest(ctx, info, bytes.NewReader([]byte(initializeReq)))
	require.NoError(t, err, "DoRequest should not fail")
	require.NotNil(t, respInterface, "Response should not be nil")
	resp := respInterface.(*http.Response)

	_, errResp := adaptor.DoResponse(ctx, resp, info)
	assert.Nil(t, errResp, "DoResponse should not fail")

	assert.Equal(t, "test-session-123", recorder.Header().Get("MCP-Session-Id"), "MCP-Session-Id should be from server")
	assert.Equal(t, "2025-03-26", recorder.Header().Get("MCP-Protocol-Version"), "MCP-Protocol-Version should be forwarded")
}

func TestMCPIntegration_BillingClassification(t *testing.T) {
	assert.False(t, IsBillableMethod(MethodInitialize), "initialize should be non-billable")
	assert.False(t, IsBillableMethod(MethodPing), "ping should be non-billable")
	assert.False(t, IsBillableMethod("notifications/initialized"), "notifications/initialized should be non-billable")
	assert.False(t, IsBillableMethod("notifications/cancelled"), "notifications/cancelled should be non-billable")

	assert.True(t, IsBillableMethod(MethodToolsCall), "tools/call should be billable")
	assert.True(t, IsBillableMethod("resources/read"), "resources/read should be billable")
	assert.True(t, IsBillableMethod("prompts/get"), "prompts/get should be billable")
	assert.True(t, IsBillableMethod("completion/complete"), "completion/complete should be billable")
	assert.True(t, IsBillableMethod("sampling/createMessage"), "sampling/createMessage should be billable")
	assert.True(t, IsBillableMethod("roots/list"), "roots/list should be billable")

	assert.True(t, IsBillableMethod("unknown/method"), "unknown methods should default to billable")
}

func TestMCPIntegration_JSONResponsePassthrough(t *testing.T) {
	server := mockMCPServer(t)
	defer server.Close()

	adaptor := &Adaptor{}
	info := buildRelayInfo(server.URL + "/")
	adaptor.Init(info)

	pingReq := `{"jsonrpc":"2.0","id":null,"method":"ping","params":{}}`
	ctx, recorder := createTestContext(http.MethodPost, "/mcp", []byte(pingReq))
	ctx.Request.Header.Set("Accept", "application/json")

	respInterface, err := adaptor.DoRequest(ctx, info, bytes.NewReader([]byte(pingReq)))
	require.NoError(t, err, "DoRequest should not fail")
	require.NotNil(t, respInterface, "Response should not be nil")
	resp := respInterface.(*http.Response)

	_, errResp := adaptor.DoResponse(ctx, resp, info)
	assert.Nil(t, errResp, "DoResponse should not fail")

	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"), "Content-Type should be application/json")

	body := recorder.Body.String()
	assert.Contains(t, body, `"jsonrpc"`, "Response should contain jsonrpc field")
	assert.Contains(t, body, `"id"`, "Response should contain id field")
	assert.Contains(t, body, `"result"`, "Response should contain result field")
}

func TestMCPIntegration_UpstreamError(t *testing.T) {
	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "upstream server error"}`))
	}))
	defer errorServer.Close()

	adaptor := &Adaptor{}
	info := buildRelayInfo(errorServer.URL + "/")
	adaptor.Init(info)

	reqBody := `{"jsonrpc":"2.0","id":null,"method":"ping","params":{}}`
	ctx, _ := createTestContext(http.MethodPost, "/mcp", []byte(reqBody))
	ctx.Request.Header.Set("Accept", "application/json")

	respInterface, err := adaptor.DoRequest(ctx, info, bytes.NewReader([]byte(reqBody)))
	require.NoError(t, err, "DoRequest should not fail for upstream error responses")
	require.NotNil(t, respInterface, "Response should not be nil")
	resp := respInterface.(*http.Response)

	_, errResp := adaptor.DoResponse(ctx, resp, info)
	assert.Nil(t, errResp, "DoResponse should not return error for upstream 500")
}

func TestMCPIntegration_ExtractMethod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "initialize method",
			body:     `{"jsonrpc":"2.0","id":null,"method":"initialize","params":{}}`,
			expected: MethodInitialize,
		},
		{
			name:     "tools/call method",
			body:     `{"jsonrpc":"2.0","id":"call_1","method":"tools/call","params":{"name":"test"}}`,
			expected: MethodToolsCall,
		},
		{
			name:     "ping method",
			body:     `{"jsonrpc":"2.0","method":"ping"}`,
			expected: MethodPing,
		},
		{
			name:     "empty body",
			body:     ``,
			expected: "",
		},
		{
			name:     "invalid JSON",
			body:     `not json`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractMCPMethod([]byte(tt.body))
			assert.Equal(t, tt.expected, result, "ExtractMCPMethod should return correct method")
		})
	}
}

var _ channel.Adaptor = (*Adaptor)(nil)