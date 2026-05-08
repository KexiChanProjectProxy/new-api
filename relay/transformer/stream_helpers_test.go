package transformer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStreamAccumulatorAccumulatesPartialChunksToFinalResponse(t *testing.T) {
	acc := NewStreamAccumulator()

	chunk1 := &PivotResponse{
		ID:    "resp_1",
		Model: "test-model",
		Choices: []PivotChoice{{
			Index: intPtr(0),
			Delta: &PivotMessage{
				Role: "assistant",
				Parts: []PivotContent{{
					Type: "text",
					Text: strPtr("Hel"),
				}},
			},
		}},
	}
	require.NoError(t, acc.Accumulate(chunk1))

	chunk2 := &PivotResponse{
		Choices: []PivotChoice{{
			Index: intPtr(0),
			Delta: &PivotMessage{
				Parts: []PivotContent{{
					Type: "text",
					Text: strPtr("lo"),
				}},
			},
			FinishReason: strPtr("stop"),
		}},
	}
	require.NoError(t, acc.Accumulate(chunk2))

	final, err := acc.Finalize()
	require.NoError(t, err)
	require.Equal(t, "resp_1", final.ID)
	require.Len(t, final.Choices, 1)
	require.NotNil(t, final.Choices[0].Message)
	require.Len(t, final.Choices[0].Message.Parts, 1)
	require.NotNil(t, final.Choices[0].Message.Parts[0].Text)
	require.Equal(t, "Hello", *final.Choices[0].Message.Parts[0].Text)
	require.NotNil(t, final.Choices[0].FinishReason)
	require.Equal(t, "stop", *final.Choices[0].FinishReason)
}

func TestStreamAccumulatorUsageAccumulationAcrossChunks(t *testing.T) {
	acc := NewStreamAccumulator()
	require.NoError(t, acc.Accumulate(&PivotResponse{Usage: &PivotUsage{PromptTokens: intPtr(100)}}))
	require.NoError(t, acc.Accumulate(&PivotResponse{Usage: &PivotUsage{CompletionTokens: intPtr(20)}}))

	final, err := acc.Finalize()
	require.NoError(t, err)
	require.NotNil(t, final.Usage)
	require.Equal(t, 100, *final.Usage.PromptTokens)
	require.Equal(t, 20, *final.Usage.CompletionTokens)
	require.Equal(t, 120, *final.Usage.TotalTokens)
}

func TestStreamAccumulatorMissingUsageEdgeCaseReturnsDefaultUsage(t *testing.T) {
	acc := NewStreamAccumulator()
	require.NoError(t, acc.Accumulate(&PivotResponse{Choices: []PivotChoice{{Index: intPtr(0)}}}))
	final, err := acc.Finalize()
	require.NoError(t, err)
	require.NotNil(t, final.Usage)
	require.Nil(t, final.Usage.PromptTokens)
}

func TestStreamAccumulatorMalformedSequenceFailsPredictably(t *testing.T) {
	acc := NewStreamAccumulator()
	err := acc.Accumulate(&PivotResponse{Choices: []PivotChoice{{Index: intPtr(-1)}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "negative choice index")
}

func TestStreamAccumulatorToolCallArgumentMerge(t *testing.T) {
	acc := NewStreamAccumulator()
	require.NoError(t, acc.Accumulate(&PivotResponse{Choices: []PivotChoice{{
		Index: intPtr(0),
		Delta: &PivotMessage{ToolCalls: []PivotToolCall{{
			Index:     intPtr(0),
			ID:        strPtr("call_1"),
			Name:      strPtr("fn"),
			Arguments: []byte("{\"a\":"),
		}}},
	}}}))
	require.NoError(t, acc.Accumulate(&PivotResponse{Choices: []PivotChoice{{
		Index: intPtr(0),
		Delta: &PivotMessage{ToolCalls: []PivotToolCall{{
			Index:     intPtr(0),
			Arguments: []byte("1}"),
		}}},
	}}}))

	final, err := acc.Finalize()
	require.NoError(t, err)
	require.Len(t, final.Choices, 1)
	require.NotNil(t, final.Choices[0].Message)
	require.Len(t, final.Choices[0].Message.ToolCalls, 1)
	require.Equal(t, "{\"a\":1}", string(final.Choices[0].Message.ToolCalls[0].Arguments))
}
