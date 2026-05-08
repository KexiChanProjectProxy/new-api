package transformer

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenAITransformerInboundOutboundRoundTrip(t *testing.T) {
	raw := []byte(`{"model":"gpt-4.1","messages":[{"role":"user","content":"hello"}],"stream":false,"stream_options":{"include_usage":false},"max_tokens":0,"top_p":0,"top_k":0,"n":0,"frequency_penalty":0,"presence_penalty":0,"seed":0,"logprobs":false,"top_logprobs":0,"tools":[{"type":"function","function":{"name":"sum","description":"add","parameters":{"type":"object"}}}],"response_format":{"type":"json_schema","json_schema":{"name":"x"}}}`)

	tf := OpenAITransformer{}
	pivot, err := tf.Inbound(raw)
	require.NoError(t, err)
	require.Equal(t, types.RelayFormatOpenAI, pivot.RelayFormat)
	require.NotNil(t, pivot.Stream)
	require.False(t, *pivot.Stream)
	require.NotNil(t, pivot.MaxTokens)
	require.Equal(t, uint(0), *pivot.MaxTokens)
	require.NotNil(t, pivot.LogProbs)
	require.False(t, *pivot.LogProbs)
	require.Len(t, pivot.Messages, 1)
	require.Equal(t, "hello", *pivot.Messages[0].Parts[0].Text)
	require.Len(t, pivot.Tools, 1)
	require.Equal(t, "sum", pivot.Tools[0].Name)

	out, err := tf.Outbound(pivot)
	require.NoError(t, err)
	require.True(t, gjson.GetBytes(out, "stream").Exists())
	require.True(t, gjson.GetBytes(out, "max_tokens").Exists())
	require.True(t, gjson.GetBytes(out, "top_p").Exists())
	require.True(t, gjson.GetBytes(out, "top_k").Exists())
	require.True(t, gjson.GetBytes(out, "n").Exists())
	require.True(t, gjson.GetBytes(out, "frequency_penalty").Exists())
	require.True(t, gjson.GetBytes(out, "presence_penalty").Exists())
	require.True(t, gjson.GetBytes(out, "seed").Exists())
	require.True(t, gjson.GetBytes(out, "logprobs").Exists())
	require.True(t, gjson.GetBytes(out, "top_logprobs").Exists())
}

func TestOpenAITransformerInboundAbsentFieldsRemainNil(t *testing.T) {
	tf := OpenAITransformer{}
	pivot, err := tf.Inbound([]byte(`{"model":"gpt-4.1"}`))
	require.NoError(t, err)
	require.Nil(t, pivot.Stream)
	require.Nil(t, pivot.MaxTokens)
	require.Nil(t, pivot.TopP)
	require.Nil(t, pivot.LogProbs)

	out, err := tf.Outbound(pivot)
	require.NoError(t, err)
	require.False(t, gjson.GetBytes(out, "stream").Exists())
	require.False(t, gjson.GetBytes(out, "max_tokens").Exists())
	require.False(t, gjson.GetBytes(out, "top_p").Exists())
	require.False(t, gjson.GetBytes(out, "logprobs").Exists())
}

func TestOpenAITransformerResponsesVariant(t *testing.T) {
	raw := []byte(`{"model":"gpt-4.1","max_output_tokens":0,"stream":false,"top_p":0,"input":[{"type":"message","role":"user","content":"hi"}]}`)
	tf := OpenAITransformer{}
	pivot, err := tf.Inbound(raw)
	require.NoError(t, err)
	require.Equal(t, types.RelayFormat(types.RelayFormatOpenAIResponses), pivot.RelayFormat)
	require.NotNil(t, pivot.MaxTokens)
	require.Equal(t, uint(0), *pivot.MaxTokens)
	require.NotNil(t, pivot.Stream)
	require.False(t, *pivot.Stream)

	out, err := tf.Outbound(pivot)
	require.NoError(t, err)
	require.True(t, gjson.GetBytes(out, "max_output_tokens").Exists())
	require.True(t, gjson.GetBytes(out, "stream").Exists())
	require.True(t, gjson.GetBytes(out, "top_p").Exists())
}

func TestOpenAIResponseTransformerRoundTrip(t *testing.T) {
	raw := []byte(`{"id":"chatcmpl-1","object":"chat.completion","created":1710000000,"model":"gpt-4.1","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`)
	rt := OpenAIResponseTransformer{}
	pivot, err := rt.InboundResponse(raw)
	require.NoError(t, err)
	require.Equal(t, "chatcmpl-1", pivot.ID)
	require.NotNil(t, pivot.Usage)

	out, err := rt.OutboundResponse(pivot)
	require.NoError(t, err)
	require.Equal(t, "chatcmpl-1", gjson.GetBytes(out, "id").String())
	require.Equal(t, "ok", gjson.GetBytes(out, "choices.0.message.content").String())
}

func TestOpenAIStreamTransformerRoundTrip(t *testing.T) {
	raw := []byte(`{"id":"chatcmpl-s","object":"chat.completion.chunk","created":1710000001,"model":"gpt-4.1","choices":[{"index":0,"delta":{"role":"assistant","content":"hel"},"finish_reason":null}],"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`)
	st := OpenAIStreamTransformer{}
	pivot, err := st.InboundStream(raw)
	require.NoError(t, err)
	require.Len(t, pivot.Choices, 1)
	require.Equal(t, "hel", *pivot.Choices[0].Delta.Parts[0].Text)

	out, err := st.OutboundStream(pivot)
	require.NoError(t, err)
	require.Equal(t, "hel", gjson.GetBytes(out, "choices.0.delta.content").String())
}

func TestOpenAIRegistryRegistration(t *testing.T) {
	req, ok := GetTransformer(types.RelayFormatOpenAI)
	require.True(t, ok)
	require.NotNil(t, req)

	resp, ok := GetResponseTransformer(types.RelayFormatOpenAI)
	require.True(t, ok)
	require.NotNil(t, resp)

	stream, ok := GetStreamTransformer(types.RelayFormatOpenAI)
	require.True(t, ok)
	require.NotNil(t, stream)

	req2, ok := GetTransformer(types.RelayFormatOpenAIResponses)
	require.True(t, ok)
	require.NotNil(t, req2)
}

func TestOpenAIMessageToolCallRoundTrip(t *testing.T) {
	msg := PivotMessage{
		Role: "assistant",
		ToolCalls: []PivotToolCall{{
			ID:        openAIStrPtr("call_1"),
			Type:      openAIStrPtr("function"),
			Name:      openAIStrPtr("sum"),
			Arguments: []byte(`{"a":1}`),
		}},
	}
	out := pivotMessageToOpenAI(msg)
	require.NotNil(t, out.ToolCalls)
	back := openAIMessageToPivot(out)
	require.Len(t, back.ToolCalls, 1)
	require.Equal(t, "call_1", *back.ToolCalls[0].ID)
	require.Equal(t, `{"a":1}`, string(back.ToolCalls[0].Arguments))
}

func TestOpenAITransformerUsesCommonJSONWrappers(t *testing.T) {
	_, err := common.Marshal(OpenAITransformer{})
	require.NoError(t, err)
	var req dto.GeneralOpenAIRequest
	err = common.Unmarshal([]byte(`{"model":"gpt-4.1"}`), &req)
	require.NoError(t, err)
}

func openAIStrPtr(s string) *string { return &s }
