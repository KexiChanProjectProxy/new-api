package dto

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGeneralOpenAIRequestPreserveExplicitZeroValues(t *testing.T) {
	raw := []byte(`{
		"model":"gpt-4.1",
		"stream":false,
		"max_tokens":0,
		"max_completion_tokens":0,
		"top_p":0,
		"top_k":0,
		"n":0,
		"frequency_penalty":0,
		"presence_penalty":0,
		"seed":0,
		"logprobs":false,
		"top_logprobs":0,
		"dimensions":0,
		"return_images":false,
		"return_related_questions":false
	}`)

	var req GeneralOpenAIRequest
	err := common.Unmarshal(raw, &req)
	require.NoError(t, err)

	encoded, err := common.Marshal(req)
	require.NoError(t, err)

	require.True(t, gjson.GetBytes(encoded, "stream").Exists())
	require.True(t, gjson.GetBytes(encoded, "max_tokens").Exists())
	require.True(t, gjson.GetBytes(encoded, "max_completion_tokens").Exists())
	require.True(t, gjson.GetBytes(encoded, "top_p").Exists())
	require.True(t, gjson.GetBytes(encoded, "top_k").Exists())
	require.True(t, gjson.GetBytes(encoded, "n").Exists())
	require.True(t, gjson.GetBytes(encoded, "frequency_penalty").Exists())
	require.True(t, gjson.GetBytes(encoded, "presence_penalty").Exists())
	require.True(t, gjson.GetBytes(encoded, "seed").Exists())
	require.True(t, gjson.GetBytes(encoded, "logprobs").Exists())
	require.True(t, gjson.GetBytes(encoded, "top_logprobs").Exists())
	require.True(t, gjson.GetBytes(encoded, "dimensions").Exists())
	require.True(t, gjson.GetBytes(encoded, "return_images").Exists())
	require.True(t, gjson.GetBytes(encoded, "return_related_questions").Exists())
}

func TestOpenAIResponsesRequestPreserveExplicitZeroValues(t *testing.T) {
	raw := []byte(`{
		"model":"gpt-4.1",
		"max_output_tokens":0,
		"max_tool_calls":0,
		"stream":false,
		"top_p":0
	}`)

	var req OpenAIResponsesRequest
	err := common.Unmarshal(raw, &req)
	require.NoError(t, err)

	encoded, err := common.Marshal(req)
	require.NoError(t, err)

	require.True(t, gjson.GetBytes(encoded, "max_output_tokens").Exists())
	require.True(t, gjson.GetBytes(encoded, "max_tool_calls").Exists())
	require.True(t, gjson.GetBytes(encoded, "stream").Exists())
	require.True(t, gjson.GetBytes(encoded, "top_p").Exists())
}

func TestGeneralOpenAIRequestOmitAbsentOptionalFields(t *testing.T) {
	// Request with only required fields, no optional pointer fields set
	raw := []byte(`{
		"model":"gpt-4.1"
	}`)

	var req GeneralOpenAIRequest
	err := common.Unmarshal(raw, &req)
	require.NoError(t, err)

	encoded, err := common.Marshal(req)
	require.NoError(t, err)

	// Verify absent optional fields are NOT present in marshaled JSON
	require.False(t, gjson.GetBytes(encoded, "stream").Exists(), "stream should be omitted when absent")
	require.False(t, gjson.GetBytes(encoded, "max_tokens").Exists(), "max_tokens should be omitted when absent")
	require.False(t, gjson.GetBytes(encoded, "max_completion_tokens").Exists(), "max_completion_tokens should be omitted when absent")
	require.False(t, gjson.GetBytes(encoded, "top_p").Exists(), "top_p should be omitted when absent")
	require.False(t, gjson.GetBytes(encoded, "top_k").Exists(), "top_k should be omitted when absent")
	require.False(t, gjson.GetBytes(encoded, "n").Exists(), "n should be omitted when absent")
	require.False(t, gjson.GetBytes(encoded, "temperature").Exists(), "temperature should be omitted when absent")
	require.False(t, gjson.GetBytes(encoded, "frequency_penalty").Exists(), "frequency_penalty should be omitted when absent")
	require.False(t, gjson.GetBytes(encoded, "presence_penalty").Exists(), "presence_penalty should be omitted when absent")
	require.False(t, gjson.GetBytes(encoded, "seed").Exists(), "seed should be omitted when absent")
}

func TestGeneralOpenAIRequestAbsentVsExplicitZero(t *testing.T) {
	t.Run("stream absent vs explicit false", func(t *testing.T) {
		// Test absent stream field
		rawAbsent := []byte(`{"model":"g"}`)
		var reqAbsent GeneralOpenAIRequest
		require.NoError(t, common.Unmarshal(rawAbsent, &reqAbsent))
		encodedAbsent, _ := common.Marshal(reqAbsent)
		require.False(t, gjson.GetBytes(encodedAbsent, "stream").Exists())

		// Test explicit stream=false
		rawZero := []byte(`{"model":"g","stream":false}`)
		var reqZero GeneralOpenAIRequest
		require.NoError(t, common.Unmarshal(rawZero, &reqZero))
		encodedZero, _ := common.Marshal(reqZero)
		require.True(t, gjson.GetBytes(encodedZero, "stream").Exists())
		require.False(t, gjson.GetBytes(encodedZero, "stream").Bool())
	})

	t.Run("max_tokens absent vs explicit zero", func(t *testing.T) {
		// Test absent max_tokens field
		rawAbsent := []byte(`{"model":"g"}`)
		var reqAbsent GeneralOpenAIRequest
		require.NoError(t, common.Unmarshal(rawAbsent, &reqAbsent))
		encodedAbsent, _ := common.Marshal(reqAbsent)
		require.False(t, gjson.GetBytes(encodedAbsent, "max_tokens").Exists())

		// Test explicit max_tokens=0
		rawZero := []byte(`{"model":"g","max_tokens":0}`)
		var reqZero GeneralOpenAIRequest
		require.NoError(t, common.Unmarshal(rawZero, &reqZero))
		encodedZero, _ := common.Marshal(reqZero)
		require.True(t, gjson.GetBytes(encodedZero, "max_tokens").Exists())
		require.Equal(t, uint64(0), gjson.GetBytes(encodedZero, "max_tokens").Uint())
	})

	t.Run("top_p absent vs explicit zero", func(t *testing.T) {
		// Test absent top_p field
		rawAbsent := []byte(`{"model":"g"}`)
		var reqAbsent GeneralOpenAIRequest
		require.NoError(t, common.Unmarshal(rawAbsent, &reqAbsent))
		encodedAbsent, _ := common.Marshal(reqAbsent)
		require.False(t, gjson.GetBytes(encodedAbsent, "top_p").Exists())

		// Test explicit top_p=0.0
		rawZero := []byte(`{"model":"g","top_p":0.0}`)
		var reqZero GeneralOpenAIRequest
		require.NoError(t, common.Unmarshal(rawZero, &reqZero))
		encodedZero, _ := common.Marshal(reqZero)
		require.True(t, gjson.GetBytes(encodedZero, "top_p").Exists())
		require.Equal(t, 0.0, gjson.GetBytes(encodedZero, "top_p").Float())
	})
}
