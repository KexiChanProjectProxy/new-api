package transformer

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestClaudeTransformerInboundOutboundRoundTrip(t *testing.T) {
	raw := []byte(`{"model":"claude-3-7-sonnet","system":[{"type":"text","text":"sys"}],"messages":[{"role":"user","content":[{"type":"text","text":"hello"},{"type":"tool_use","id":"tool_1","name":"sum","input":{"a":1}}]},{"role":"assistant","content":[{"type":"thinking","thinking":"hmm","signature":"sig"},{"type":"tool_result","tool_use_id":"tool_1","content":"ok"}]}],"stream":false,"max_tokens":0,"temperature":0,"top_p":0,"top_k":0,"stop_sequences":["END"],"tools":[{"name":"sum","description":"add","input_schema":{"type":"object"}}],"tool_choice":{"type":"tool","name":"sum","disable_parallel_tool_use":false},"thinking":{"type":"enabled","budget_tokens":0},"metadata":{"user_id":"u1"}}`)

	tf := ClaudeTransformer{}
	pivot, err := tf.Inbound(raw)
	require.NoError(t, err)
	require.Equal(t, string(types.RelayFormatClaude), string(pivot.RelayFormat))
	require.NotNil(t, pivot.Stream)
	require.False(t, *pivot.Stream)
	require.NotNil(t, pivot.MaxTokens)
	require.Equal(t, uint(0), *pivot.MaxTokens)
	require.NotNil(t, pivot.Thinking)
	require.Equal(t, "enabled", *pivot.Thinking.Type)
	require.Len(t, pivot.Messages, 2)
	require.Equal(t, "hello", *pivot.Messages[0].Parts[0].Text)
	require.NotNil(t, pivot.Messages[0].Parts[1].ToolCall)
	require.Len(t, pivot.Tools, 1)
	require.Equal(t, "sum", pivot.Tools[0].Name)
	require.NotNil(t, pivot.ProviderExtensions)
	require.NotNil(t, pivot.ProviderExtensions["claude_metadata"])

	out, err := tf.Outbound(pivot)
	require.NoError(t, err)
	require.True(t, gjson.GetBytes(out, "stream").Exists())
	require.True(t, gjson.GetBytes(out, "max_tokens").Exists())
	require.True(t, gjson.GetBytes(out, "top_p").Exists())
	require.True(t, gjson.GetBytes(out, "top_k").Exists())
	require.True(t, gjson.GetBytes(out, "tool_choice").Exists())
	require.True(t, gjson.GetBytes(out, "thinking").Exists())
	require.True(t, gjson.GetBytes(out, "metadata").Exists())
}

func TestClaudeTransformerInboundAbsentFieldsRemainNil(t *testing.T) {
	tf := ClaudeTransformer{}
	pivot, err := tf.Inbound([]byte(`{"model":"claude-3-7-sonnet"}`))
	require.NoError(t, err)
	require.Nil(t, pivot.Stream)
	require.Nil(t, pivot.MaxTokens)
	require.Nil(t, pivot.TopP)
	require.Nil(t, pivot.Thinking)

	out, err := tf.Outbound(pivot)
	require.NoError(t, err)
	require.False(t, gjson.GetBytes(out, "stream").Exists())
	require.False(t, gjson.GetBytes(out, "max_tokens").Exists())
	require.False(t, gjson.GetBytes(out, "top_p").Exists())
	require.False(t, gjson.GetBytes(out, "thinking").Exists())
}

func TestClaudeResponseTransformerRoundTrip(t *testing.T) {
	raw := []byte(`{"id":"msg_1","type":"message","role":"assistant","model":"claude-3-7-sonnet","content":[{"type":"text","text":"done"}],"stop_reason":"end_turn","usage":{"input_tokens":10,"output_tokens":5,"cache_creation_input_tokens":2,"cache_read_input_tokens":3}}`)

	tf := ClaudeResponseTransformer{}
	pivot, err := tf.InboundResponse(raw)
	require.NoError(t, err)
	require.Equal(t, "msg_1", pivot.ID)
	require.Equal(t, "message", pivot.Object)
	require.Len(t, pivot.Choices, 1)
	require.Equal(t, "assistant", pivot.Choices[0].Message.Role)
	require.Equal(t, "done", *pivot.Choices[0].Message.Parts[0].Text)
	require.NotNil(t, pivot.Usage)
	require.Equal(t, 10, *pivot.Usage.InputTokens)
	require.Equal(t, 5, *pivot.Usage.OutputTokens)

	out, err := tf.OutboundResponse(pivot)
	require.NoError(t, err)
	require.Equal(t, "message", gjson.GetBytes(out, "type").String())
	require.Equal(t, "assistant", gjson.GetBytes(out, "role").String())
	require.Equal(t, "done", gjson.GetBytes(out, "content.0.text").String())
	require.Equal(t, int64(10), gjson.GetBytes(out, "usage.input_tokens").Int())
	require.Equal(t, int64(5), gjson.GetBytes(out, "usage.output_tokens").Int())
}

func TestClaudeStreamTransformerInboundEvents(t *testing.T) {
	stream := ClaudeStreamTransformer{}

	startRaw := []byte(`{"type":"message_start","message":{"id":"msg_1","type":"message","model":"claude-3-7-sonnet","usage":{"input_tokens":4,"output_tokens":0}}}`)
	startPivot, err := stream.InboundStream(startRaw)
	require.NoError(t, err)
	require.Equal(t, "msg_1", startPivot.ID)
	require.Equal(t, "message", startPivot.Object)
	require.NotNil(t, startPivot.Usage)
	require.Equal(t, 4, *startPivot.Usage.InputTokens)

	textDeltaRaw := []byte(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"he"}}`)
	textDeltaPivot, err := stream.InboundStream(textDeltaRaw)
	require.NoError(t, err)
	require.Len(t, textDeltaPivot.Choices, 1)
	require.NotNil(t, textDeltaPivot.Choices[0].Delta)
	require.Equal(t, "he", *textDeltaPivot.Choices[0].Delta.Parts[0].Text)

	toolDeltaRaw := []byte(`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"a\":"}}`)
	toolDeltaPivot, err := stream.InboundStream(toolDeltaRaw)
	require.NoError(t, err)
	require.Equal(t, "{\"a\":", string(toolDeltaPivot.Choices[0].Delta.Parts[0].ToolCall.Arguments))

	msgDeltaRaw := []byte(`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":7}}`)
	msgDeltaPivot, err := stream.InboundStream(msgDeltaRaw)
	require.NoError(t, err)
	require.Equal(t, "end_turn", *msgDeltaPivot.Choices[0].FinishReason)
	require.NotNil(t, msgDeltaPivot.Usage)
	require.Equal(t, 7, *msgDeltaPivot.Usage.OutputTokens)
}

func TestClaudeStreamTransformerOutboundDelta(t *testing.T) {
	tf := ClaudeStreamTransformer{}
	idx := 0
	finish := "end_turn"
	text := "ok"
	outRaw, err := tf.OutboundStream(&PivotResponse{
		ID:     "msg_2",
		Object: "message_delta",
		Model:  "claude-3-7-sonnet",
		Choices: []PivotChoice{{
			Index:        &idx,
			FinishReason: &finish,
			Delta: &PivotMessage{Parts: []PivotContent{{Type: dto.ContentTypeText, Text: &text}}},
		}},
	})
	require.NoError(t, err)
	require.Equal(t, "message_delta", gjson.GetBytes(outRaw, "type").String())
	require.Equal(t, "text_delta", gjson.GetBytes(outRaw, "delta.type").String())
	require.Equal(t, "ok", gjson.GetBytes(outRaw, "delta.text").String())
	require.Equal(t, "end_turn", gjson.GetBytes(outRaw, "delta.stop_reason").String())
}

func TestClaudeTransformerRegistration(t *testing.T) {
	req, ok := GetTransformer(types.RelayFormatClaude)
	require.True(t, ok)
	require.NotNil(t, req)
	resp, ok := GetResponseTransformer(types.RelayFormatClaude)
	require.True(t, ok)
	require.NotNil(t, resp)
	stream, ok := GetStreamTransformer(types.RelayFormatClaude)
	require.True(t, ok)
	require.NotNil(t, stream)
}

func TestClaudeTransformerUsesCommonJSONWrappers(t *testing.T) {
	_, err := common.Marshal(ClaudeTransformer{})
	require.NoError(t, err)
	var req dto.ClaudeRequest
	err = common.Unmarshal([]byte(`{"model":"claude-3-7-sonnet"}`), &req)
	require.NoError(t, err)
}
