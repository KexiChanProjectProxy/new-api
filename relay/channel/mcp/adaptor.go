package mcp

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type Adaptor struct {
}

func (a *Adaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeminiChatRequest) (any, error) {
	return nil, errors.New("mcp adaptor: ConvertGeminiRequest not implemented - MCP does not use format conversion")
}

func (a *Adaptor) ConvertClaudeRequest(*gin.Context, *relaycommon.RelayInfo, *dto.ClaudeRequest) (any, error) {
	return nil, errors.New("mcp adaptor: ConvertClaudeRequest not implemented - MCP does not use format conversion")
}

func (a *Adaptor) ConvertAudioRequest(*gin.Context, *relaycommon.RelayInfo, dto.AudioRequest) (io.Reader, error) {
	return nil, errors.New("mcp adaptor: ConvertAudioRequest not implemented - MCP does not use format conversion")
}

func (a *Adaptor) ConvertImageRequest(*gin.Context, *relaycommon.RelayInfo, dto.ImageRequest) (any, error) {
	return nil, errors.New("mcp adaptor: ConvertImageRequest not implemented - MCP does not use format conversion")
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
	info.IsStream = true
}

func (a *Adaptor) ConvertOpenAIRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeneralOpenAIRequest) (any, error) {
	return nil, errors.New("mcp adaptor: ConvertOpenAIRequest not implemented - MCP does not use format conversion")
}

func (a *Adaptor) ConvertRerankRequest(*gin.Context, int, dto.RerankRequest) (any, error) {
	return nil, errors.New("mcp adaptor: ConvertRerankRequest not implemented - MCP does not use format conversion")
}

func (a *Adaptor) ConvertEmbeddingRequest(*gin.Context, *relaycommon.RelayInfo, dto.EmbeddingRequest) (any, error) {
	return nil, errors.New("mcp adaptor: ConvertEmbeddingRequest not implemented - MCP does not use format conversion")
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(*gin.Context, *relaycommon.RelayInfo, dto.OpenAIResponsesRequest) (any, error) {
	return nil, errors.New("mcp adaptor: ConvertOpenAIResponsesRequest not implemented - MCP does not use format conversion")
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	info.FinalRequestRelayFormat = types.RelayFormatMCP
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		return nil, MCPStreamHandler(c, resp, info)
	}
	return nil, MCPHandler(c, resp, info)
}

func (a *Adaptor) GetModelList() []string {
	return []string{}
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return info.ChannelBaseUrl, nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	req.Set("Authorization", "Bearer "+info.ApiKey)

	if sessionID := c.Request.Header.Get("MCP-Session-Id"); sessionID != "" {
		req.Set("MCP-Session-Id", sessionID)
	}
	if protoVersion := c.Request.Header.Get("MCP-Protocol-Version"); protoVersion != "" {
		req.Set("MCP-Protocol-Version", protoVersion)
	}
	if lastEventID := c.Request.Header.Get("Last-Event-ID"); lastEventID != "" {
		req.Set("Last-Event-ID", lastEventID)
	}

	return nil
}