package transformer

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestNormalizeUsageCacheCreationSplitAndFieldMirroring(t *testing.T) {
	u := &PivotUsage{
		CacheCreationInputTokens:    intPtr(50),
		ClaudeCacheCreation5mTokens: intPtr(10),
		ClaudeCacheCreation1hTokens: intPtr(20),
	}

	n := NormalizeUsage(u)
	require.NotNil(t, n)
	require.Equal(t, 30, *n.ClaudeCacheCreation5mTokens)
	require.Equal(t, 20, *n.ClaudeCacheCreation1hTokens)
	require.Equal(t, 50, *n.CacheCreationInputTokens)
	require.NotNil(t, n.PromptTokensDetails)
	require.Equal(t, 50, *n.PromptTokensDetails.CachedCreationTokens)
}

func TestNormalizeUsageDerivesPromptCompletionTotals(t *testing.T) {
	u := &PivotUsage{
		InputTokens:  intPtr(120),
		OutputTokens: intPtr(30),
	}
	n := NormalizeUsage(u)
	require.Equal(t, 120, *n.PromptTokens)
	require.Equal(t, 30, *n.CompletionTokens)
	require.Equal(t, 150, *n.TotalTokens)
}

func TestMergeUsageAccumulatesTokensAndDetails(t *testing.T) {
	base := &PivotUsage{
		PromptTokens:     intPtr(100),
		CompletionTokens: intPtr(20),
		PromptTokensDetails: &PivotInputTokenStats{
			CachedTokens: intPtr(30),
		},
	}
	delta := &PivotUsage{
		PromptTokens:     intPtr(10),
		CompletionTokens: intPtr(5),
		PromptTokensDetails: &PivotInputTokenStats{
			CachedTokens: intPtr(2),
		},
	}

	m := MergeUsage(base, delta)
	require.Equal(t, 110, *m.PromptTokens)
	require.Equal(t, 25, *m.CompletionTokens)
	require.Equal(t, 135, *m.TotalTokens)
	require.NotNil(t, m.PromptTokensDetails)
	require.Equal(t, 32, *m.PromptTokensDetails.CachedTokens)
}

func TestPivotOpenAIUsageRoundTrip(t *testing.T) {
	in := &PivotUsage{
		PromptTokens:                intPtr(100),
		CompletionTokens:            intPtr(20),
		CacheCreationInputTokens:    intPtr(50),
		CacheReadInputTokens:        intPtr(30),
		ClaudeCacheCreation5mTokens: intPtr(10),
		ClaudeCacheCreation1hTokens: intPtr(20),
		PromptTokensDetails: &PivotInputTokenStats{
			CachedTokens:         intPtr(30),
			CachedCreationTokens: intPtr(50),
		},
	}
	oai := ConvertPivotUsageToOpenAI(in)
	require.NotNil(t, oai)
	out := ConvertOpenAIUsageToPivot(oai)
	require.NotNil(t, out)
	require.Equal(t, 100, *out.PromptTokens)
	require.Equal(t, 20, *out.CompletionTokens)
	require.Equal(t, 30, *out.CacheReadInputTokens)
	require.Equal(t, 50, *out.CacheCreationInputTokens)
	require.Equal(t, 30, *out.ClaudeCacheCreation5mTokens)
	require.Equal(t, 20, *out.ClaudeCacheCreation1hTokens)
}

func TestConvertOpenAIUsageToPivotPreservesExplicitZeroPointers(t *testing.T) {
	u := &dto.Usage{}
	p := ConvertOpenAIUsageToPivot(u)
	require.NotNil(t, p)
	require.NotNil(t, p.PromptTokens)
	require.NotNil(t, p.CompletionTokens)
	require.NotNil(t, p.TotalTokens)
	require.Equal(t, 0, *p.PromptTokens)
	require.Equal(t, 0, *p.CompletionTokens)
	require.Equal(t, 0, *p.TotalTokens)
}
