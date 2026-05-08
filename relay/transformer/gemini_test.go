package transformer

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGeminiTransformerInboundOutboundRoundTrip(t *testing.T) {
	raw := []byte(`{"contents":[{"role":"user","parts":[{"text":"hello"},{"inlineData":{"mimeType":"image/png","data":"aGVsbG8="}},{"functionCall":{"name":"sum","args":{"a":1}}}]},{"role":"model","parts":[{"functionResponse":{"name":"sum","response":{"result":2}}}]}],"systemInstruction":{"parts":[{"text":"sys"}]},"safetySettings":[{"category":"HARM_CATEGORY_HATE_SPEECH","threshold":"BLOCK_NONE"}],"generationConfig":{"temperature":0,"topP":0,"topK":0,"maxOutputTokens":0,"candidateCount":0,"stopSequences":["END"],"presencePenalty":0,"frequencyPenalty":0,"responseLogprobs":false,"logprobs":0,"seed":0,"thinkingConfig":{"includeThoughts":false,"thinkingBudget":0,"thinkingLevel":"low"}},"tools":[{"functionDeclarations":[{"name":"sum","description":"add","parameters":{"type":"object"}}]}],"toolConfig":{"functionCallingConfig":{"mode":"AUTO","allowedFunctionNames":["sum"]}},"cachedContent":"cache-1"}`)

	tf := GeminiTransformer{}
	pivot, err := tf.Inbound(raw)
	require.NoError(t, err)
	require.Equal(t, string(types.RelayFormatGemini), string(pivot.RelayFormat))
	require.Len(t, pivot.Messages, 2)
	require.Equal(t, "user", pivot.Messages[0].Role)
	require.Equal(t, "assistant", pivot.Messages[1].Role)
	require.NotNil(t, pivot.MaxCompletionTokens)
	require.Equal(t, uint(0), *pivot.MaxCompletionTokens)
	require.NotNil(t, pivot.TopP)
	require.Equal(t, float64(0), *pivot.TopP)
	require.NotNil(t, pivot.TopK)
	require.Equal(t, 0, *pivot.TopK)
	require.NotNil(t, pivot.PresencePenalty)
	require.Equal(t, float64(0), *pivot.PresencePenalty)
	require.NotNil(t, pivot.FrequencyPenalty)
	require.Equal(t, float64(0), *pivot.FrequencyPenalty)
	require.NotNil(t, pivot.LogProbs)
	require.False(t, *pivot.LogProbs)
	require.NotNil(t, pivot.TopLogProbs)
	require.Equal(t, 0, *pivot.TopLogProbs)
	require.NotNil(t, pivot.Seed)
	require.Equal(t, float64(0), *pivot.Seed)
	require.NotNil(t, pivot.System)
	require.Equal(t, "sys", *pivot.System.Parts[0].Text)
	require.Len(t, pivot.Tools, 1)
	require.Equal(t, "sum", pivot.Tools[0].Name)
	require.NotNil(t, pivot.ToolConfig)
	require.NotNil(t, pivot.ToolConfig.FunctionCalling)
	require.Equal(t, "AUTO", *pivot.ToolConfig.FunctionCalling.Mode)
	require.NotNil(t, pivot.CachedContent)
	require.Equal(t, "cache-1", *pivot.CachedContent)

	out, err := tf.Outbound(pivot)
	require.NoError(t, err)
	require.True(t, gjson.GetBytes(out, "generationConfig.temperature").Exists())
	require.True(t, gjson.GetBytes(out, "generationConfig.topP").Exists())
	require.True(t, gjson.GetBytes(out, "generationConfig.topK").Exists())
	require.True(t, gjson.GetBytes(out, "generationConfig.maxOutputTokens").Exists())
	require.True(t, gjson.GetBytes(out, "generationConfig.candidateCount").Exists())
	require.True(t, gjson.GetBytes(out, "generationConfig.presencePenalty").Exists())
	require.True(t, gjson.GetBytes(out, "generationConfig.frequencyPenalty").Exists())
	require.True(t, gjson.GetBytes(out, "generationConfig.responseLogprobs").Exists())
	require.True(t, gjson.GetBytes(out, "generationConfig.logprobs").Exists())
	require.True(t, gjson.GetBytes(out, "generationConfig.seed").Exists())
	require.Equal(t, "model", gjson.GetBytes(out, "contents.1.role").String())
	require.Equal(t, "sum", gjson.GetBytes(out, "tools.0.functionDeclarations.0.name").String())
}

func TestGeminiTransformerInboundAbsentFieldsRemainNil(t *testing.T) {
	tf := GeminiTransformer{}
	pivot, err := tf.Inbound([]byte(`{"contents":[{"role":"user","parts":[{"text":"hi"}]}]}`))
	require.NoError(t, err)
	require.Nil(t, pivot.MaxCompletionTokens)
	require.Nil(t, pivot.TopP)
	require.Nil(t, pivot.LogProbs)
	require.Nil(t, pivot.ToolConfig)
	require.Nil(t, pivot.CachedContent)

	out, err := tf.Outbound(pivot)
	require.NoError(t, err)
	require.False(t, gjson.GetBytes(out, "generationConfig.maxOutputTokens").Exists())
	require.False(t, gjson.GetBytes(out, "generationConfig.topP").Exists())
	require.False(t, gjson.GetBytes(out, "generationConfig.responseLogprobs").Exists())
	require.False(t, gjson.GetBytes(out, "toolConfig").Exists())
	require.False(t, gjson.GetBytes(out, "cachedContent").Exists())
}

func TestGeminiResponseTransformerRoundTrip(t *testing.T) {
	raw := []byte(`{"candidates":[{"index":0,"content":{"role":"model","parts":[{"text":"done"}]},"finishReason":"STOP","safetyRatings":[{"category":"HARM_CATEGORY_HATE_SPEECH","probability":"NEGLIGIBLE"}]}],"promptFeedback":{"blockReason":"NONE","safetyRatings":[{"category":"HARM_CATEGORY_HATE_SPEECH","probability":"NEGLIGIBLE"}]},"usageMetadata":{"promptTokenCount":0,"candidatesTokenCount":0,"totalTokenCount":0,"cachedContentTokenCount":0,"thoughtsTokenCount":0}}`)

	tf := GeminiResponseTransformer{}
	pivot, err := tf.InboundResponse(raw)
	require.NoError(t, err)
	require.Len(t, pivot.Choices, 1)
	require.Equal(t, "assistant", pivot.Choices[0].Message.Role)
	require.Equal(t, "done", *pivot.Choices[0].Message.Parts[0].Text)
	require.NotNil(t, pivot.Usage)
	require.Equal(t, 0, *pivot.Usage.PromptTokens)
	require.NotNil(t, pivot.ProviderExtensions)
	require.NotNil(t, pivot.ProviderExtensions["gemini_prompt_feedback"])

	out, err := tf.OutboundResponse(pivot)
	require.NoError(t, err)
	require.Equal(t, "model", gjson.GetBytes(out, "candidates.0.content.role").String())
	require.Equal(t, "done", gjson.GetBytes(out, "candidates.0.content.parts.0.text").String())
	require.True(t, gjson.GetBytes(out, "promptFeedback").Exists())
}

func TestGeminiStreamTransformerRoundTrip(t *testing.T) {
	raw := []byte(`{"candidates":[{"index":0,"content":{"role":"model","parts":[{"text":"hel"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":0,"candidatesTokenCount":0,"totalTokenCount":0}}`)

	tf := GeminiStreamTransformer{}
	pivot, err := tf.InboundStream(raw)
	require.NoError(t, err)
	require.Len(t, pivot.Choices, 1)
	require.Equal(t, "hel", *pivot.Choices[0].Message.Parts[0].Text)

	out, err := tf.OutboundStream(pivot)
	require.NoError(t, err)
	require.Equal(t, "hel", gjson.GetBytes(out, "candidates.0.content.parts.0.text").String())
}

func TestGeminiTransformerRegistration(t *testing.T) {
	req, ok := GetTransformer(types.RelayFormatGemini)
	require.True(t, ok)
	require.NotNil(t, req)

	resp, ok := GetResponseTransformer(types.RelayFormatGemini)
	require.True(t, ok)
	require.NotNil(t, resp)

	stream, ok := GetStreamTransformer(types.RelayFormatGemini)
	require.True(t, ok)
	require.NotNil(t, stream)
}

func TestGeminiTransformerUsesCommonJSONWrappers(t *testing.T) {
	_, err := common.Marshal(GeminiTransformer{})
	require.NoError(t, err)

	var req dto.GeminiChatRequest
	err = common.Unmarshal([]byte(`{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`), &req)
	require.NoError(t, err)
}
