package transformer

import "github.com/QuantumNous/new-api/dto"

func NormalizeUsage(usage *PivotUsage) *PivotUsage {
	if usage == nil {
		return nil
	}
	n := clonePivotUsage(usage)

	if n.PromptTokens == nil && n.InputTokens != nil {
			n.PromptTokens = usageIntPtr(*n.InputTokens)
	}
	if n.InputTokens == nil && n.PromptTokens != nil {
		n.InputTokens = usageIntPtr(*n.PromptTokens)
	}
	if n.CompletionTokens == nil && n.OutputTokens != nil {
		n.CompletionTokens = usageIntPtr(*n.OutputTokens)
	}
	if n.OutputTokens == nil && n.CompletionTokens != nil {
		n.OutputTokens = usageIntPtr(*n.CompletionTokens)
	}

	if n.PromptTokensDetails != nil {
		if n.CacheReadInputTokens == nil && n.PromptTokensDetails.CachedTokens != nil {
			n.CacheReadInputTokens = usageIntPtr(*n.PromptTokensDetails.CachedTokens)
		}
		if n.CacheCreationInputTokens == nil && n.PromptTokensDetails.CachedCreationTokens != nil {
			n.CacheCreationInputTokens = usageIntPtr(*n.PromptTokensDetails.CachedCreationTokens)
		}
	}
	if n.PromptTokensDetails == nil && (n.CacheReadInputTokens != nil || n.CacheCreationInputTokens != nil) {
		n.PromptTokensDetails = &PivotInputTokenStats{}
		if n.CacheReadInputTokens != nil {
			n.PromptTokensDetails.CachedTokens = usageIntPtr(*n.CacheReadInputTokens)
		}
		if n.CacheCreationInputTokens != nil {
			n.PromptTokensDetails.CachedCreationTokens = usageIntPtr(*n.CacheCreationInputTokens)
		}
	}

	cacheCreationTotal := intValue(n.CacheCreationInputTokens)
	cacheCreation5m := intValue(n.ClaudeCacheCreation5mTokens)
	cacheCreation1h := intValue(n.ClaudeCacheCreation1hTokens)
	if cacheCreationTotal == 0 && (cacheCreation5m > 0 || cacheCreation1h > 0) {
		cacheCreationTotal = cacheCreation5m + cacheCreation1h
		n.CacheCreationInputTokens = usageIntPtr(cacheCreationTotal)
	}
	if cacheCreationTotal > 0 {
		remainder := cacheCreationTotal - cacheCreation5m - cacheCreation1h
		if remainder < 0 {
			remainder = 0
		}
		cacheCreation5m += remainder
		n.ClaudeCacheCreation5mTokens = usageIntPtr(cacheCreation5m)
		n.ClaudeCacheCreation1hTokens = usageIntPtr(cacheCreation1h)
		if n.PromptTokensDetails == nil {
			n.PromptTokensDetails = &PivotInputTokenStats{}
		}
		n.PromptTokensDetails.CachedCreationTokens = usageIntPtr(cacheCreationTotal)
	}

	if n.TotalTokens == nil && n.PromptTokens != nil && n.CompletionTokens != nil {
		n.TotalTokens = usageIntPtr(*n.PromptTokens + *n.CompletionTokens)
	}

	return n
}

func MergeUsage(base, delta *PivotUsage) *PivotUsage {
	if base == nil && delta == nil {
		return nil
	}
	if base == nil {
		return NormalizeUsage(delta)
	}
	if delta == nil {
		return NormalizeUsage(base)
	}

	b := NormalizeUsage(base)
	d := NormalizeUsage(delta)
	out := clonePivotUsage(b)

	out.PromptTokens = addIntPtrs(b.PromptTokens, d.PromptTokens)
	out.CompletionTokens = addIntPtrs(b.CompletionTokens, d.CompletionTokens)
	out.TotalTokens = addIntPtrs(b.TotalTokens, d.TotalTokens)
	out.InputTokens = addIntPtrs(b.InputTokens, d.InputTokens)
	out.OutputTokens = addIntPtrs(b.OutputTokens, d.OutputTokens)
	out.PromptCacheHitTokens = addIntPtrs(b.PromptCacheHitTokens, d.PromptCacheHitTokens)
	out.CacheCreationInputTokens = addIntPtrs(b.CacheCreationInputTokens, d.CacheCreationInputTokens)
	out.CacheReadInputTokens = addIntPtrs(b.CacheReadInputTokens, d.CacheReadInputTokens)
	out.ClaudeCacheCreation5mTokens = addIntPtrs(b.ClaudeCacheCreation5mTokens, d.ClaudeCacheCreation5mTokens)
	out.ClaudeCacheCreation1hTokens = addIntPtrs(b.ClaudeCacheCreation1hTokens, d.ClaudeCacheCreation1hTokens)
	out.ThoughtsTokenCount = addIntPtrs(b.ThoughtsTokenCount, d.ThoughtsTokenCount)
	out.ToolUsePromptTokenCount = addIntPtrs(b.ToolUsePromptTokenCount, d.ToolUsePromptTokenCount)
	out.PromptTokensDetails = mergeInputStats(b.PromptTokensDetails, d.PromptTokensDetails)
	out.CompletionTokenDetails = mergeOutputStats(b.CompletionTokenDetails, d.CompletionTokenDetails)
	out.InputTokensDetails = mergeInputStats(b.InputTokensDetails, d.InputTokensDetails)

	return NormalizeUsage(out)
}

func ConvertPivotUsageToOpenAI(usage *PivotUsage) *dto.Usage {
	if usage == nil {
		return nil
	}
	n := NormalizeUsage(usage)
	out := &dto.Usage{}
	out.PromptTokens = intValue(n.PromptTokens)
	out.CompletionTokens = intValue(n.CompletionTokens)
	out.TotalTokens = intValue(n.TotalTokens)
	out.InputTokens = intValue(n.InputTokens)
	out.OutputTokens = intValue(n.OutputTokens)
	out.PromptCacheHitTokens = intValue(n.PromptCacheHitTokens)
	out.ClaudeCacheCreation5mTokens = intValue(n.ClaudeCacheCreation5mTokens)
	out.ClaudeCacheCreation1hTokens = intValue(n.ClaudeCacheCreation1hTokens)
	if n.UsageSemantic != nil {
		out.UsageSemantic = *n.UsageSemantic
	}
	if n.UsageSource != nil {
		out.UsageSource = *n.UsageSource
	}
	if n.PromptTokensDetails != nil {
		out.PromptTokensDetails = dto.InputTokenDetails{
			CachedTokens:         intValue(n.PromptTokensDetails.CachedTokens),
			CachedCreationTokens: intValue(n.PromptTokensDetails.CachedCreationTokens),
			TextTokens:           intValue(n.PromptTokensDetails.TextTokens),
			AudioTokens:          intValue(n.PromptTokensDetails.AudioTokens),
			ImageTokens:          intValue(n.PromptTokensDetails.ImageTokens),
		}
	}
	if n.CompletionTokenDetails != nil {
		out.CompletionTokenDetails = dto.OutputTokenDetails{
			TextTokens:      intValue(n.CompletionTokenDetails.TextTokens),
			AudioTokens:     intValue(n.CompletionTokenDetails.AudioTokens),
			ReasoningTokens: intValue(n.CompletionTokenDetails.ReasoningTokens),
		}
	}
	if n.InputTokensDetails != nil {
		out.InputTokensDetails = &dto.InputTokenDetails{
			CachedTokens:         intValue(n.InputTokensDetails.CachedTokens),
			CachedCreationTokens: intValue(n.InputTokensDetails.CachedCreationTokens),
			TextTokens:           intValue(n.InputTokensDetails.TextTokens),
			AudioTokens:          intValue(n.InputTokensDetails.AudioTokens),
			ImageTokens:          intValue(n.InputTokensDetails.ImageTokens),
		}
	}
	return out
}

func ConvertOpenAIUsageToPivot(usage *dto.Usage) *PivotUsage {
	if usage == nil {
		return nil
	}
	out := &PivotUsage{
		PromptTokens:                usageIntPtr(usage.PromptTokens),
		CompletionTokens:            usageIntPtr(usage.CompletionTokens),
		TotalTokens:                 usageIntPtr(usage.TotalTokens),
		InputTokens:                 usageIntPtr(usage.InputTokens),
		OutputTokens:                usageIntPtr(usage.OutputTokens),
		PromptCacheHitTokens:        usageIntPtr(usage.PromptCacheHitTokens),
		CacheCreationInputTokens:    usageIntPtr(usage.PromptTokensDetails.CachedCreationTokens),
		CacheReadInputTokens:        usageIntPtr(usage.PromptTokensDetails.CachedTokens),
		ClaudeCacheCreation5mTokens: usageIntPtr(usage.ClaudeCacheCreation5mTokens),
		ClaudeCacheCreation1hTokens: usageIntPtr(usage.ClaudeCacheCreation1hTokens),
		PromptTokensDetails: &PivotInputTokenStats{
			CachedTokens:         usageIntPtr(usage.PromptTokensDetails.CachedTokens),
			CachedCreationTokens: usageIntPtr(usage.PromptTokensDetails.CachedCreationTokens),
			TextTokens:           usageIntPtr(usage.PromptTokensDetails.TextTokens),
			AudioTokens:          usageIntPtr(usage.PromptTokensDetails.AudioTokens),
			ImageTokens:          usageIntPtr(usage.PromptTokensDetails.ImageTokens),
		},
		CompletionTokenDetails: &PivotOutputTokenStats{
			TextTokens:      usageIntPtr(usage.CompletionTokenDetails.TextTokens),
			AudioTokens:     usageIntPtr(usage.CompletionTokenDetails.AudioTokens),
			ReasoningTokens: usageIntPtr(usage.CompletionTokenDetails.ReasoningTokens),
		},
		UsageSemantic: usageStrPtr(usage.UsageSemantic),
		UsageSource:   usageStrPtr(usage.UsageSource),
	}
	if usage.InputTokensDetails != nil {
		out.InputTokensDetails = &PivotInputTokenStats{
			CachedTokens:         usageIntPtr(usage.InputTokensDetails.CachedTokens),
			CachedCreationTokens: usageIntPtr(usage.InputTokensDetails.CachedCreationTokens),
			TextTokens:           usageIntPtr(usage.InputTokensDetails.TextTokens),
			AudioTokens:          usageIntPtr(usage.InputTokensDetails.AudioTokens),
			ImageTokens:          usageIntPtr(usage.InputTokensDetails.ImageTokens),
		}
	}
	return NormalizeUsage(out)
}

func clonePivotUsage(in *PivotUsage) *PivotUsage {
	if in == nil {
		return nil
	}
	out := *in
	if in.PromptTokensDetails != nil {
		d := *in.PromptTokensDetails
		out.PromptTokensDetails = &d
	}
	if in.CompletionTokenDetails != nil {
		d := *in.CompletionTokenDetails
		out.CompletionTokenDetails = &d
	}
	if in.InputTokensDetails != nil {
		d := *in.InputTokensDetails
		out.InputTokensDetails = &d
	}
	return &out
}

func mergeInputStats(a, b *PivotInputTokenStats) *PivotInputTokenStats {
	if a == nil && b == nil {
		return nil
	}
	out := &PivotInputTokenStats{}
	if a != nil {
		*out = *a
	}
	out.CachedTokens = addIntPtrs(nilIntStat(a, "cached"), nilIntStat(b, "cached"))
	out.CachedCreationTokens = addIntPtrs(nilIntStat(a, "cachedCreation"), nilIntStat(b, "cachedCreation"))
	out.TextTokens = addIntPtrs(nilIntStat(a, "text"), nilIntStat(b, "text"))
	out.AudioTokens = addIntPtrs(nilIntStat(a, "audio"), nilIntStat(b, "audio"))
	out.ImageTokens = addIntPtrs(nilIntStat(a, "image"), nilIntStat(b, "image"))
	return out
}

func mergeOutputStats(a, b *PivotOutputTokenStats) *PivotOutputTokenStats {
	if a == nil && b == nil {
		return nil
	}
	out := &PivotOutputTokenStats{}
	if a != nil {
		*out = *a
	}
	out.TextTokens = addIntPtrs(nilOutIntStat(a, "text"), nilOutIntStat(b, "text"))
	out.AudioTokens = addIntPtrs(nilOutIntStat(a, "audio"), nilOutIntStat(b, "audio"))
	out.ReasoningTokens = addIntPtrs(nilOutIntStat(a, "reasoning"), nilOutIntStat(b, "reasoning"))
	return out
}

func nilIntStat(s *PivotInputTokenStats, key string) *int {
	if s == nil {
		return nil
	}
	switch key {
	case "cached":
		return s.CachedTokens
	case "cachedCreation":
		return s.CachedCreationTokens
	case "text":
		return s.TextTokens
	case "audio":
		return s.AudioTokens
	case "image":
		return s.ImageTokens
	default:
		return nil
	}
}

func nilOutIntStat(s *PivotOutputTokenStats, key string) *int {
	if s == nil {
		return nil
	}
	switch key {
	case "text":
		return s.TextTokens
	case "audio":
		return s.AudioTokens
	case "reasoning":
		return s.ReasoningTokens
	default:
		return nil
	}
}

func addIntPtrs(a, b *int) *int {
	if a == nil && b == nil {
		return nil
	}
	return usageIntPtr(intValue(a) + intValue(b))
}

func intValue(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func usageIntPtr(v int) *int { return &v }

func usageStrPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
