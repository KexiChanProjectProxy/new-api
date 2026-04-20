package mcp

import (
	"io"
	"net/http"

	"github.com/QuantumNous/new-api/relay/helper"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func MCPHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) *types.NewAPIError {
	defer service.CloseResponseBodyGracefully(resp)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.NewError(err, types.ErrorCodeBadResponseBody)
	}

	for k, v := range resp.Header {
		if k == "Content-Length" {
			continue
		}
		c.Writer.Header().Set(k, v[0])
	}

	c.Writer.WriteHeader(resp.StatusCode)
	if _, writeErr := c.Writer.Write(responseBody); writeErr != nil {
		return types.NewError(writeErr, types.ErrorCodeDoRequestFailed)
	}

	return nil
}

func MCPStreamHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) *types.NewAPIError {
	defer service.CloseResponseBodyGracefully(resp)

	helper.SetEventStreamHeaders(c)

	if sessionID := resp.Header.Get("MCP-Session-Id"); sessionID != "" {
		c.Writer.Header().Set("MCP-Session-Id", sessionID)
	}
	if protocolVersion := resp.Header.Get("MCP-Protocol-Version"); protocolVersion != "" {
		c.Writer.Header().Set("MCP-Protocol-Version", protocolVersion)
	}
	if lastEventID := resp.Header.Get("Last-Event-ID"); lastEventID != "" {
		c.Writer.Header().Set("Last-Event-ID", lastEventID)
	}

	c.Writer.WriteHeader(resp.StatusCode)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		_, err := io.Copy(c.Writer, resp.Body)
		if err != nil {
			return types.NewError(err, types.ErrorCodeDoRequestFailed)
		}
		return nil
	}

	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := c.Writer.Write(buf[:n]); writeErr != nil {
				return types.NewError(writeErr, types.ErrorCodeDoRequestFailed)
			}
			flusher.Flush()
		}
		if err != nil {
			if err != io.EOF {
				return types.NewError(err, types.ErrorCodeDoRequestFailed)
			}
			break
		}
	}

	return nil
}