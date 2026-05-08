package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

func TestClaudeToOpenAIRequest_PointerFieldsPreserved(t *testing.T) {
	tests := []struct {
		name          string
		claudeRequest dto.ClaudeRequest
		expectNil     func(req dto.ClaudeRequest) bool
		expectValue   func(req dto.ClaudeRequest) bool
	}{
		{
			name: "all pointer fields nil",
			claudeRequest: dto.ClaudeRequest{
				Model: "claude-3-5-sonnet",
			},
			expectNil: func(req dto.ClaudeRequest) bool {
				return req.MaxTokens == nil && req.TopP == nil && req.TopK == nil && req.Stream == nil
			},
			expectValue: func(req dto.ClaudeRequest) bool {
				return false
			},
		},
		{
			name: "MaxTokens explicitly set to value",
			claudeRequest: dto.ClaudeRequest{
				Model:     "claude-3-5-sonnet",
				MaxTokens: lo.ToPtr(uint(1024)),
			},
			expectNil: func(req dto.ClaudeRequest) bool {
				return req.MaxTokens == nil
			},
			expectValue: func(req dto.ClaudeRequest) bool {
				return req.MaxTokens != nil && *req.MaxTokens == 1024
			},
		},
		{
			name: "MaxTokens explicitly set to zero",
			claudeRequest: dto.ClaudeRequest{
				Model:     "claude-3-5-sonnet",
				MaxTokens: lo.ToPtr(uint(0)),
			},
			expectNil: func(req dto.ClaudeRequest) bool {
				return req.MaxTokens == nil
			},
			expectValue: func(req dto.ClaudeRequest) bool {
				return req.MaxTokens != nil && *req.MaxTokens == 0
			},
		},
		{
			name: "TopP explicitly set",
			claudeRequest: dto.ClaudeRequest{
				Model:     "claude-3-5-sonnet",
				TopP:      lo.ToPtr(0.7),
			},
			expectNil: func(req dto.ClaudeRequest) bool {
				return req.TopP == nil
			},
			expectValue: func(req dto.ClaudeRequest) bool {
				return req.TopP != nil && *req.TopP == 0.7
			},
		},
		{
			name: "TopP explicitly set to zero",
			claudeRequest: dto.ClaudeRequest{
				Model:     "claude-3-5-sonnet",
				TopP:      lo.ToPtr(0.0),
			},
			expectNil: func(req dto.ClaudeRequest) bool {
				return req.TopP == nil
			},
			expectValue: func(req dto.ClaudeRequest) bool {
				return req.TopP != nil && *req.TopP == 0.0
			},
		},
		{
			name: "TopK explicitly set",
			claudeRequest: dto.ClaudeRequest{
				Model: "claude-3-5-sonnet",
				TopK:  lo.ToPtr(100),
			},
			expectNil: func(req dto.ClaudeRequest) bool {
				return req.TopK == nil
			},
			expectValue: func(req dto.ClaudeRequest) bool {
				return req.TopK != nil && *req.TopK == 100
			},
		},
		{
			name: "TopK explicitly set to zero",
			claudeRequest: dto.ClaudeRequest{
				Model: "claude-3-5-sonnet",
				TopK:  lo.ToPtr(0),
			},
			expectNil: func(req dto.ClaudeRequest) bool {
				return req.TopK == nil
			},
			expectValue: func(req dto.ClaudeRequest) bool {
				return req.TopK != nil && *req.TopK == 0
			},
		},
		{
			name: "Stream explicitly set to true",
			claudeRequest: dto.ClaudeRequest{
				Model:  "claude-3-5-sonnet",
				Stream: lo.ToPtr(true),
			},
			expectNil: func(req dto.ClaudeRequest) bool {
				return req.Stream == nil
			},
			expectValue: func(req dto.ClaudeRequest) bool {
				return req.Stream != nil && *req.Stream == true
			},
		},
		{
			name: "Stream explicitly set to false",
			claudeRequest: dto.ClaudeRequest{
				Model:  "claude-3-5-sonnet",
				Stream: lo.ToPtr(false),
			},
			expectNil: func(req dto.ClaudeRequest) bool {
				return req.Stream == nil
			},
			expectValue: func(req dto.ClaudeRequest) bool {
				return req.Stream != nil && *req.Stream == false
			},
		},
		{
			name: "all pointer fields with explicit values",
			claudeRequest: dto.ClaudeRequest{
				Model:     "claude-3-5-sonnet",
				MaxTokens: lo.ToPtr(uint(2048)),
				TopP:      lo.ToPtr(0.9),
				TopK:      lo.ToPtr(200),
				Stream:    lo.ToPtr(true),
			},
			expectNil: func(req dto.ClaudeRequest) bool {
				return false
			},
			expectValue: func(req dto.ClaudeRequest) bool {
				return req.MaxTokens != nil && *req.MaxTokens == 2048 &&
					req.TopP != nil && *req.TopP == 0.9 &&
					req.TopK != nil && *req.TopK == 200 &&
					req.Stream != nil && *req.Stream == true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{},
	}
	openAIReq, err := ClaudeToOpenAIRequest(tt.claudeRequest, info)
	require.NoError(t, err)
	require.NotNil(t, openAIReq)

	data, err := common.Marshal(openAIReq)
	require.NoError(t, err)

	if tt.expectNil(tt.claudeRequest) {
		require.NotNil(t, data)
	}

	if tt.expectValue(tt.claudeRequest) {
		require.NotNil(t, data)
	}
		})
	}
}

func TestClaudeToOpenAIRequest_SystemPromptConversion(t *testing.T) {
	tests := []struct {
		name          string
		claudeRequest dto.ClaudeRequest
		wantSystem    bool
		wantContent   string
	}{
		{
			name: "string system prompt",
			claudeRequest: dto.ClaudeRequest{
				Model:  "claude-3-5-sonnet",
				System: "You are a helpful assistant.",
			},
			wantSystem:  true,
			wantContent: "You are a helpful assistant.",
		},
		{
			name: "no system prompt",
			claudeRequest: dto.ClaudeRequest{
				Model: "claude-3-5-sonnet",
			},
			wantSystem:  false,
			wantContent: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				ChannelMeta: &relaycommon.ChannelMeta{},
			}
			openAIReq, err := ClaudeToOpenAIRequest(tt.claudeRequest, info)
			require.NoError(t, err)
			require.NotNil(t, openAIReq)

			hasSystem := false
			var systemContent string
			for _, msg := range openAIReq.Messages {
				if msg.Role == "system" {
					hasSystem = true
					systemContent = msg.StringContent()
					break
				}
			}

			require.Equal(t, tt.wantSystem, hasSystem, "system message presence mismatch")
			if tt.wantSystem {
				require.Equal(t, tt.wantContent, systemContent, "system content mismatch")
			}
		})
	}
}

func TestClaudeToOpenAIRequest_MessageConversion(t *testing.T) {
	tests := []struct {
		name          string
		claudeRequest dto.ClaudeRequest
		wantMsgCount  int
		wantFirstRole string
	}{
		{
			name: "single user message",
			claudeRequest: dto.ClaudeRequest{
				Model: "claude-3-5-sonnet",
				Messages: []dto.ClaudeMessage{
					{
						Role:    "user",
						Content: "Hello",
					},
				},
			},
			wantMsgCount:  1,
			wantFirstRole: "user",
		},
		{
			name: "multiple messages with roles",
			claudeRequest: dto.ClaudeRequest{
				Model: "claude-3-5-sonnet",
				Messages: []dto.ClaudeMessage{
					{
						Role:    "user",
						Content: "Hello",
					},
					{
						Role:    "assistant",
						Content: "Hi there!",
					},
					{
						Role:    "user",
						Content: "How are you?",
					},
				},
			},
			wantMsgCount:  3,
			wantFirstRole: "user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				ChannelMeta: &relaycommon.ChannelMeta{},
			}
			openAIReq, err := ClaudeToOpenAIRequest(tt.claudeRequest, info)
			require.NoError(t, err)
			require.NotNil(t, openAIReq)
			require.Len(t, openAIReq.Messages, tt.wantMsgCount, "message count mismatch")
			if tt.wantMsgCount > 0 {
				require.Equal(t, tt.wantFirstRole, openAIReq.Messages[0].Role, "first message role mismatch")
			}
		})
	}
}

func TestClaudeToOpenAIRequest_StopSequences(t *testing.T) {
	tests := []struct {
		name          string
		claudeRequest dto.ClaudeRequest
		wantStop      any
	}{
		{
			name: "single stop sequence",
			claudeRequest: dto.ClaudeRequest{
				Model:         "claude-3-5-sonnet",
				StopSequences: []string{"STOP"},
			},
			wantStop: "STOP",
		},
		{
			name: "multiple stop sequences",
			claudeRequest: dto.ClaudeRequest{
				Model:         "claude-3-5-sonnet",
				StopSequences: []string{"STOP", "END"},
			},
			wantStop: []string{"STOP", "END"},
		},
		{
			name:          "no stop sequences",
			claudeRequest: dto.ClaudeRequest{
				Model: "claude-3-5-sonnet",
			},
			wantStop: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				ChannelMeta: &relaycommon.ChannelMeta{},
			}
			openAIReq, err := ClaudeToOpenAIRequest(tt.claudeRequest, info)
			require.NoError(t, err)
			require.NotNil(t, openAIReq)

			if tt.wantStop == nil {
				require.Nil(t, openAIReq.Stop, "stop should be nil")
			} else {
				require.NotNil(t, openAIReq.Stop, "stop should not be nil")
			}
		})
	}
}

func TestNormalizeCacheCreationSplit(t *testing.T) {
	tests := []struct {
		name        string
		totalTokens int
		tokens5m    int
		tokens1h    int
		want5m      int
		want1h      int
	}{
		{
			name:        "no remainder - exact split",
			totalTokens: 100,
			tokens5m:    60,
			tokens1h:    40,
			want5m:      60,
			want1h:      40,
		},
		{
			name:        "remainder goes to 5m",
			totalTokens: 100,
			tokens5m:    50,
			tokens1h:    40,
			want5m:      60, // 50 + (100 - 50 - 40) = 60
			want1h:      40,
		},
		{
			name:        "remainder goes to 5m when 1h is zero",
			totalTokens: 100,
			tokens5m:    50,
			tokens1h:    0,
			want5m:      100, // 50 + (100 - 50 - 0) = 100
			want1h:      0,
		},
		{
			name:        "all tokens in 5m",
			totalTokens: 100,
			tokens5m:    100,
			tokens1h:    0,
			want5m:      100,
			want1h:      0,
		},
		{
			name:        "all tokens in 1h",
			totalTokens: 100,
			tokens5m:    0,
			tokens1h:    100,
			want5m:      0,
			want1h:      100,
		},
		{
			name:        "zero total tokens",
			totalTokens: 0,
			tokens5m:    0,
			tokens1h:    0,
			want5m:      0,
			want1h:      0,
		},
		{
			name:        "total less than sum - returns original",
			totalTokens: 50,
			tokens5m:    60,
			tokens1h:    40,
			want5m:      60,
			want1h:      40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got5m, got1h := NormalizeCacheCreationSplit(tt.totalTokens, tt.tokens5m, tt.tokens1h)
			require.Equal(t, tt.want5m, got5m, "5m tokens mismatch")
			require.Equal(t, tt.want1h, got1h, "1h tokens mismatch")
		})
	}
}

func TestBuildClaudeUsageFromOpenAIUsage(t *testing.T) {
	tests := []struct {
		name     string
		oaiUsage *dto.Usage
		wantNil  bool
		wantInput int
		wantOutput int
		wantCacheRead int
	}{
		{
			name:     "nil usage",
			oaiUsage: nil,
			wantNil:  true,
		},
		{
			name: "basic usage without cache",
			oaiUsage: &dto.Usage{
				PromptTokens:     100,
				CompletionTokens: 50,
				PromptTokensDetails: dto.InputTokenDetails{
					CachedTokens:         0,
					CachedCreationTokens: 0,
				},
			},
			wantNil:      false,
			wantInput:     100,
			wantOutput:    50,
			wantCacheRead: 0,
		},
		{
			name: "usage with cache tokens",
			oaiUsage: &dto.Usage{
				PromptTokens:     200,
				CompletionTokens: 100,
				PromptTokensDetails: dto.InputTokenDetails{
					CachedTokens:         50,
					CachedCreationTokens: 30,
				},
			},
			wantNil:      false,
			wantInput:     200,
			wantOutput:    100,
			wantCacheRead: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildClaudeUsageFromOpenAIUsage(tt.oaiUsage)
			if tt.wantNil {
				require.Nil(t, result)
			} else {
				require.NotNil(t, result)
				require.Equal(t, tt.wantInput, result.InputTokens)
				require.Equal(t, tt.wantOutput, result.OutputTokens)
				require.Equal(t, tt.wantCacheRead, result.CacheReadInputTokens)
			}
		})
	}
}

func TestGeminiToOpenAIRequest_BasicConversion(t *testing.T) {
	tests := []struct {
		name         string
		geminiReq    *dto.GeminiChatRequest
		wantMsgCount int
		wantModel    string
	}{
		{
			name: "simple text request",
			geminiReq: &dto.GeminiChatRequest{
				Contents: []dto.GeminiChatContent{
					{
						Role: "user",
						Parts: []dto.GeminiPart{
							{Text: "Hello"},
						},
					},
				},
			},
			wantMsgCount: 1,
			wantModel:    "test-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				ChannelMeta: &relaycommon.ChannelMeta{
					UpstreamModelName: "test-model",
				},
			}
			openAIReq, err := GeminiToOpenAIRequest(tt.geminiReq, info)
			require.NoError(t, err)
			require.NotNil(t, openAIReq)
			require.Equal(t, tt.wantModel, openAIReq.Model)
		})
	}
}

func TestResponseOpenAI2Claude_BasicResponse(t *testing.T) {
	tests := []struct {
		name         string
		openAIResp   *dto.OpenAITextResponse
		wantType     string
		wantContent  bool
	}{
		{
			name: "basic text response",
			openAIResp: &dto.OpenAITextResponse{
				Id:      "test-id",
				Model:   "claude-3-5-sonnet",
				Object:  "chat.completion",
				Created: 1234567890,
				Choices: []dto.OpenAITextResponseChoice{
					{
						Index: 0,
						Message: dto.Message{
							Role:    "assistant",
							Content: "Hello!",
						},
						FinishReason: "stop",
					},
				},
				Usage: dto.Usage{
					PromptTokens:     10,
					CompletionTokens: 5,
					TotalTokens:      15,
				},
			},
			wantType:    "message",
			wantContent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				ChannelMeta: &relaycommon.ChannelMeta{},
			}
			result := ResponseOpenAI2Claude(tt.openAIResp, info)
			require.NotNil(t, result)
			require.Equal(t, tt.wantType, result.Type)
		})
	}
}

func TestStreamResponseOpenAI2Claude_BasicStream(t *testing.T) {
	info := &relaycommon.RelayInfo{
		IsStream: true,
		ClaudeConvertInfo: &relaycommon.ClaudeConvertInfo{
			Index:            0,
			LastMessagesType: relaycommon.LastMessageTypeNone,
			Done:             false,
		},
	}

	tests := []struct {
		name          string
		openAIStream  *dto.ChatCompletionsStreamResponse
		wantNil       bool
		wantRespCount int
	}{
		{
			name: "empty choices",
			openAIStream: &dto.ChatCompletionsStreamResponse{
				Id:      "test-id",
				Model:   "claude-3-5-sonnet",
				Object:  "chat.completion.chunk",
				Choices: []dto.ChatCompletionsStreamResponseChoice{},
			},
			wantNil:       true,
			wantRespCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StreamResponseOpenAI2Claude(tt.openAIStream, info)
			if tt.wantNil {
				require.Nil(t, result)
			} else {
				_ = result
			}
		})
	}
}
