package mcp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSetupRequestHeader_MCPHeadersForwarded(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	ctx.Request = httptest.NewRequest(http.MethodPost, "/mcp", nil)
	ctx.Request.Header.Set("MCP-Session-Id", "test-session-123")
	ctx.Request.Header.Set("MCP-Protocol-Version", "2025-03-26")
	ctx.Request.Header.Set("Last-Event-ID", "event-456")

	adaptor := &Adaptor{}
	req := &http.Header{}

	info := &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ApiKey: "test-api-key",
		},
	}

	err := adaptor.SetupRequestHeader(ctx, req, info)

	assert.NoError(t, err)
	assert.Equal(t, "test-session-123", req.Get("MCP-Session-Id"))
	assert.Equal(t, "2025-03-26", req.Get("MCP-Protocol-Version"))
	assert.Equal(t, "event-456", req.Get("Last-Event-ID"))
	assert.Equal(t, "Bearer test-api-key", req.Get("Authorization"))
}

func TestSetupRequestHeader_NoMCPSessionId(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	ctx.Request = httptest.NewRequest(http.MethodPost, "/mcp", nil)
	ctx.Request.Header.Set("MCP-Protocol-Version", "2025-03-26")

	adaptor := &Adaptor{}
	req := &http.Header{}

	info := &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ApiKey: "test-api-key",
		},
	}

	err := adaptor.SetupRequestHeader(ctx, req, info)

	assert.NoError(t, err)
	assert.Equal(t, "", req.Get("MCP-Session-Id"))
	assert.Equal(t, "2025-03-26", req.Get("MCP-Protocol-Version"))
	assert.Equal(t, "Bearer test-api-key", req.Get("Authorization"))
}

func TestSetupRequestHeader_EmptyMCPSessionId(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	ctx.Request = httptest.NewRequest(http.MethodPost, "/mcp", nil)
	ctx.Request.Header.Set("MCP-Session-Id", "")

	adaptor := &Adaptor{}
	req := &http.Header{}

	info := &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ApiKey: "test-api-key",
		},
	}

	err := adaptor.SetupRequestHeader(ctx, req, info)

	assert.NoError(t, err)
	assert.Equal(t, "", req.Get("MCP-Session-Id"))
	assert.Equal(t, "Bearer test-api-key", req.Get("Authorization"))
}