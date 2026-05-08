package transformer

import (
	"reflect"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
)

func TestPivotRequestPointerFieldsPreserveAbsentVsZeroSemantics(t *testing.T) {
	var empty PivotRequest
	emptyJSON, err := common.Marshal(empty)
	if err != nil {
		t.Fatalf("marshal empty pivot request: %v", err)
	}

	var emptyMap map[string]any
	if err := common.Unmarshal(emptyJSON, &emptyMap); err != nil {
		t.Fatalf("unmarshal empty pivot request: %v", err)
	}

	zero := populatedZeroPivotRequest()
	zeroJSON, err := common.Marshal(zero)
	if err != nil {
		t.Fatalf("marshal zero pivot request: %v", err)
	}

	var zeroMap map[string]any
	if err := common.Unmarshal(zeroJSON, &zeroMap); err != nil {
		t.Fatalf("unmarshal zero pivot request: %v", err)
	}

	assertPointerSemantics(t, reflect.TypeFor[PivotRequest](), emptyMap, zeroMap)

	var roundTrip PivotRequest
	if err := common.Unmarshal(zeroJSON, &roundTrip); err != nil {
		t.Fatalf("round-trip pivot request: %v", err)
	}

	assertPivotRequestZeroPointers(t, roundTrip)
}

func TestPivotResponseMarshalUnmarshalRoundTrip(t *testing.T) {
	response := samplePivotResponse()
	data, err := common.Marshal(response)
	if err != nil {
		t.Fatalf("marshal pivot response: %v", err)
	}

	var decoded PivotResponse
	if err := common.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal pivot response: %v", err)
	}

	if !reflect.DeepEqual(response, decoded) {
		t.Fatalf("pivot response round-trip mismatch\nwant: %#v\n got: %#v", response, decoded)
	}
}

func TestPivotRequestCanRepresentOpenAIClaudeAndGeminiFields(t *testing.T) {
	openAIStream := false
	openAIReq := dto.GeneralOpenAIRequest{
		Model:               "gpt-4.1",
		Stream:              &openAIStream,
		MaxTokens:           uintPtr(256),
		MaxCompletionTokens: uintPtr(128),
		Temperature:         float64Ptr(0),
		TopP:                float64Ptr(0),
		TopK:                intPtr(0),
		N:                   intPtr(0),
		FrequencyPenalty:    float64Ptr(0),
		PresencePenalty:     float64Ptr(0),
		Seed:                float64Ptr(0),
		LogProbs:            boolPtr(false),
		TopLogProbs:         intPtr(0),
		ResponseFormat:      &dto.ResponseFormat{Type: "json_schema"},
	}

	claudeReq := dto.ClaudeRequest{
		Model:          "claude-3-7-sonnet",
		System:         "system prompt",
		Messages:       []dto.ClaudeMessage{{Role: "user", Content: "hello"}},
		MaxTokens:      uintPtr(512),
		Temperature:    float64Ptr(0),
		TopP:           float64Ptr(0),
		TopK:           intPtr(0),
		Stream:         boolPtr(false),
		StopSequences:  []string{"END"},
		Thinking:       &dto.Thinking{Type: "enabled", BudgetTokens: intPtr(0)},
		ServiceTier:    "standard",
	}

	geminiReq := dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{{Role: "user", Parts: []dto.GeminiPart{{Text: "hi"}}}},
		SystemInstructions: &dto.GeminiChatContent{Parts: []dto.GeminiPart{{Text: "system"}}},
		SafetySettings:     []dto.GeminiChatSafetySettings{{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_NONE"}},
		ToolConfig: &dto.ToolConfig{FunctionCallingConfig: &dto.FunctionCallingConfig{
			Mode:                 dto.FunctionCallingConfigMode("AUTO"),
			AllowedFunctionNames: []string{"lookup"},
		}},
		CachedContent: "cache-id",
	}
	geminiReq.GenerationConfig = dto.GeminiChatGenerationConfig{
		Temperature:      float64Ptr(0),
		TopP:             float64Ptr(0),
		MaxOutputTokens:  uintPtr(1024),
		CandidateCount:   intPtr(0),
		StopSequences:    []string{"STOP"},
		ResponseLogprobs: boolPtr(false),
		Logprobs:         int32Ptr(0),
		Seed:             int64Ptr(0),
	}

	pivot := PivotRequest{
		RelayFormat:         types.RelayFormatOpenAI,
		Model:               openAIReq.Model,
		Stream:              openAIReq.Stream,
		MaxTokens:           openAIReq.MaxTokens,
		MaxCompletionTokens: openAIReq.MaxCompletionTokens,
		Temperature:         openAIReq.Temperature,
		TopP:                openAIReq.TopP,
		TopK:                openAIReq.TopK,
		N:                   openAIReq.N,
		FrequencyPenalty:    openAIReq.FrequencyPenalty,
		PresencePenalty:     openAIReq.PresencePenalty,
		Seed:                openAIReq.Seed,
		LogProbs:            openAIReq.LogProbs,
		TopLogProbs:         openAIReq.TopLogProbs,
		ResponseFormat:      &PivotResponseFormat{Type: strPtr(openAIReq.ResponseFormat.Type)},
		System:              &PivotSystemPrompt{Text: strPtr(claudeReq.GetStringSystem())},
		Messages: []PivotMessage{{
			Role:  claudeReq.Messages[0].Role,
			Parts: []PivotContent{{Type: "text", Text: strPtr(claudeReq.Messages[0].GetStringContent())}},
		}},
		StopSequences: claudeReq.StopSequences,
		Thinking: &PivotThinkingConfig{
			Type:         strPtr(claudeReq.Thinking.Type),
			BudgetTokens: claudeReq.Thinking.BudgetTokens,
		},
		ServiceTier:   strPtr(claudeReq.ServiceTier),
		SafetySettings: []PivotSafetySetting{{
			Category:  geminiReq.SafetySettings[0].Category,
			Threshold: strPtr(geminiReq.SafetySettings[0].Threshold),
		}},
		ToolConfig: &PivotToolConfig{FunctionCalling: &PivotFunctionCallingConfig{
			Mode:                 strPtr(string(geminiReq.ToolConfig.FunctionCallingConfig.Mode)),
			AllowedFunctionNames: geminiReq.ToolConfig.FunctionCallingConfig.AllowedFunctionNames,
		}},
		CachedContent: strPtr(geminiReq.CachedContent),
		ProviderExtensions: map[string]any{
			"gemini_generation": map[string]any{
				"candidate_count":     *geminiReq.GenerationConfig.CandidateCount,
				"response_logprobs":   *geminiReq.GenerationConfig.ResponseLogprobs,
				"logprobs":            *geminiReq.GenerationConfig.Logprobs,
				"seed":                *geminiReq.GenerationConfig.Seed,
				"system_instruction":  geminiReq.SystemInstructions.Parts[0].Text,
			},
		},
	}

	if len(geminiReq.Contents) != 1 || len(geminiReq.Contents[0].Parts) != 1 || geminiReq.Contents[0].Parts[0].Text != "hi" {
		t.Fatalf("gemini source request setup invalid")
	}
	if pivot.RelayFormat != types.RelayFormatOpenAI || pivot.Model != openAIReq.Model || pivot.Stream == nil || *pivot.Stream != *openAIReq.Stream {
		t.Fatalf("pivot did not preserve openai request fields")
	}
	if pivot.MaxTokens == nil || *pivot.MaxTokens != *openAIReq.MaxTokens || pivot.MaxCompletionTokens == nil || *pivot.MaxCompletionTokens != *openAIReq.MaxCompletionTokens {
		t.Fatalf("pivot did not preserve openai max token fields")
	}
	if pivot.Temperature == nil || *pivot.Temperature != *openAIReq.Temperature || pivot.TopP == nil || *pivot.TopP != *openAIReq.TopP || pivot.TopK == nil || *pivot.TopK != *openAIReq.TopK {
		t.Fatalf("pivot did not preserve openai sampling fields")
	}
	if pivot.N == nil || *pivot.N != *openAIReq.N || pivot.FrequencyPenalty == nil || *pivot.FrequencyPenalty != *openAIReq.FrequencyPenalty || pivot.PresencePenalty == nil || *pivot.PresencePenalty != *openAIReq.PresencePenalty {
		t.Fatalf("pivot did not preserve openai penalty fields")
	}
	if pivot.Seed == nil || *pivot.Seed != *openAIReq.Seed || pivot.LogProbs == nil || *pivot.LogProbs != *openAIReq.LogProbs || pivot.TopLogProbs == nil || *pivot.TopLogProbs != *openAIReq.TopLogProbs {
		t.Fatalf("pivot did not preserve openai logprob fields")
	}
	if pivot.ResponseFormat == nil || pivot.ResponseFormat.Type == nil || *pivot.ResponseFormat.Type != openAIReq.ResponseFormat.Type {
		t.Fatalf("pivot did not preserve openai response format")
	}
	if pivot.System == nil || pivot.System.Text == nil || *pivot.System.Text != "system prompt" {
		t.Fatalf("pivot did not preserve claude system prompt")
	}
	if len(pivot.Messages) != 1 || len(pivot.Messages[0].Parts) != 1 || pivot.Messages[0].Parts[0].Text == nil || *pivot.Messages[0].Parts[0].Text != "hello" {
		t.Fatalf("pivot did not preserve claude message content")
	}
	if len(pivot.StopSequences) != 1 || pivot.StopSequences[0] != claudeReq.StopSequences[0] || pivot.Thinking == nil || pivot.Thinking.Type == nil || *pivot.Thinking.Type != claudeReq.Thinking.Type || pivot.Thinking.BudgetTokens == nil || *pivot.Thinking.BudgetTokens != *claudeReq.Thinking.BudgetTokens || pivot.ServiceTier == nil || *pivot.ServiceTier != claudeReq.ServiceTier {
		t.Fatalf("pivot did not preserve claude-specific fields")
	}
	if len(pivot.SafetySettings) != 1 || pivot.SafetySettings[0].Threshold == nil || *pivot.SafetySettings[0].Threshold != "BLOCK_NONE" {
		t.Fatalf("pivot did not preserve gemini safety settings")
	}
	if pivot.ToolConfig == nil || pivot.ToolConfig.FunctionCalling == nil || len(pivot.ToolConfig.FunctionCalling.AllowedFunctionNames) != 1 {
		t.Fatalf("pivot did not preserve gemini tool config")
	}
	if pivot.CachedContent == nil || *pivot.CachedContent != geminiReq.CachedContent {
		t.Fatalf("pivot did not preserve gemini cached content")
	}
	if pivot.ProviderExtensions == nil || pivot.ProviderExtensions["gemini_generation"] == nil {
		t.Fatalf("pivot did not preserve provider extension payload")
	}
}

func assertPointerSemantics(t *testing.T, typ reflect.Type, emptyMap, zeroMap map[string]any) {
	t.Helper()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		jsonName := jsonFieldName(field)
		if jsonName == "" {
			continue
		}
		if field.Type.Kind() == reflect.Pointer && isScalarKind(field.Type.Elem().Kind()) {
			if _, ok := emptyMap[jsonName]; ok {
				t.Fatalf("expected absent field %q to be omitted", jsonName)
			}
			if _, ok := zeroMap[jsonName]; !ok {
				t.Fatalf("expected zero-valued pointer field %q to be present", jsonName)
			}
		}
	}
}

func assertPivotRequestZeroPointers(t *testing.T, got PivotRequest) {
	t.Helper()
	checks := map[string]bool{
		"instruction":           got.Instruction != nil && *got.Instruction == "",
		"stream":                got.Stream != nil && !*got.Stream,
		"temperature":           got.Temperature != nil && *got.Temperature == 0,
		"top_p":                 got.TopP != nil && *got.TopP == 0,
		"top_k":                 got.TopK != nil && *got.TopK == 0,
		"max_tokens":            got.MaxTokens != nil && *got.MaxTokens == 0,
		"max_completion_tokens": got.MaxCompletionTokens != nil && *got.MaxCompletionTokens == 0,
		"n":                     got.N != nil && *got.N == 0,
		"frequency_penalty":     got.FrequencyPenalty != nil && *got.FrequencyPenalty == 0,
		"presence_penalty":      got.PresencePenalty != nil && *got.PresencePenalty == 0,
		"seed":                  got.Seed != nil && *got.Seed == 0,
		"logprobs":              got.LogProbs != nil && !*got.LogProbs,
		"top_logprobs":          got.TopLogProbs != nil && *got.TopLogProbs == 0,
		"reasoning_effort":      got.ReasoningEffort != nil && *got.ReasoningEffort == "",
		"service_tier":          got.ServiceTier != nil && *got.ServiceTier == "",
		"parallel_tool_calls":   got.ParallelToolCalls != nil && !*got.ParallelToolCalls,
		"cached_content":        got.CachedContent != nil && *got.CachedContent == "",
	}
	for field, ok := range checks {
		if !ok {
			t.Fatalf("zero-value pointer field %q not preserved after round-trip", field)
		}
	}
}

func populatedZeroPivotRequest() PivotRequest {
	return PivotRequest{
		Instruction:         strPtr(""),
		Stream:              boolPtr(false),
		StreamOptions:       &PivotStreamOptions{IncludeUsage: boolPtr(false), IncludeObfuscation: boolPtr(false)},
		Temperature:         float64Ptr(0),
		TopP:                float64Ptr(0),
		TopK:                intPtr(0),
		MaxTokens:           uintPtr(0),
		MaxCompletionTokens: uintPtr(0),
		N:                   intPtr(0),
		FrequencyPenalty:    float64Ptr(0),
		PresencePenalty:     float64Ptr(0),
		Seed:                float64Ptr(0),
		LogProbs:            boolPtr(false),
		TopLogProbs:         intPtr(0),
		ResponseFormat:      &PivotResponseFormat{Type: strPtr(""), Name: strPtr(""), Description: strPtr(""), Strict: boolPtr(false), MimeType: strPtr("")},
		Thinking:            &PivotThinkingConfig{Type: strPtr(""), Enabled: boolPtr(false), BudgetTokens: intPtr(0), Effort: strPtr(""), Level: strPtr(""), IncludeThoughts: boolPtr(false)},
		ReasoningEffort:     strPtr(""),
		ServiceTier:         strPtr(""),
		ParallelToolCalls:   boolPtr(false),
		Audio:               &PivotAudioConfig{Voice: strPtr(""), Format: strPtr(""), SampleRate: intPtr(0)},
		Prediction:          &PivotPrediction{Type: strPtr("")},
		ToolConfig:          &PivotToolConfig{FunctionCalling: &PivotFunctionCallingConfig{Mode: strPtr("")}, Retrieval: &PivotRetrievalConfig{Latitude: float64Ptr(0), Longitude: float64Ptr(0), LanguageCode: strPtr("")}},
		CachedContent:       strPtr(""),
	}
}

func samplePivotResponse() PivotResponse {
	return PivotResponse{
		ID:     "resp_123",
		Object: "chat.completion",
		Created: int64Ptr(123456789),
		Model:  "gpt-4.1",
		Choices: []PivotChoice{{
			Index:        intPtr(0),
			FinishReason: strPtr("stop"),
			Message: &PivotMessage{
				Role: "assistant",
				Parts: []PivotContent{{Type: "text", Text: strPtr("hello")}},
				Thinking: &PivotThinkingBlock{Text: strPtr("thought")},
			},
		}},
		Content: []PivotContent{{Type: "text", Text: strPtr("hello")}},
		Usage: &PivotUsage{
			PromptTokens:                intPtr(10),
			CompletionTokens:            intPtr(5),
			TotalTokens:                 intPtr(15),
			PromptCacheHitTokens:        intPtr(2),
			CacheCreationInputTokens:    intPtr(1),
			CacheReadInputTokens:        intPtr(3),
			ClaudeCacheCreation5mTokens: intPtr(4),
			ClaudeCacheCreation1hTokens: intPtr(5),
			ThoughtsTokenCount:          intPtr(6),
			ToolUsePromptTokenCount:     intPtr(7),
			PromptTokensDetails:         &PivotInputTokenStats{CachedTokens: intPtr(2), TextTokens: intPtr(8)},
			CompletionTokenDetails:      &PivotOutputTokenStats{TextTokens: intPtr(5), ReasoningTokens: intPtr(1)},
			InputTokensDetails:          &PivotInputTokenStats{AudioTokens: intPtr(0), ImageTokens: intPtr(0)},
			UsageSemantic:               strPtr("billable"),
			UsageSource:                 strPtr("provider"),
		},
		StreamState:        &PivotStreamState{IsStream: boolPtr(true), Done: boolPtr(false), ChunkIndex: intPtr(1), Sequence: intPtr(2), LastMessageType: strPtr("text")},
		FinishReason:       strPtr("stop"),
		SystemFingerprint:  strPtr("fp_1"),
		Status:             strPtr("completed"),
		IncompleteReason:   strPtr(""),
		ProviderMetadata:   map[string]any{"upstream": "openai"},
		ProviderExtensions: map[string]any{"cache": true},
	}
}

func jsonFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" || tag == "-" {
		return ""
	}
	for i, ch := range tag {
		if ch == ',' {
			return tag[:i]
		}
	}
	return tag
}

func isScalarKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64, reflect.String:
		return true
	default:
		return false
	}
}

func boolPtr(v bool) *bool { return &v }
func intPtr(v int) *int { return &v }
func int32Ptr(v int32) *int32 { return &v }
func int64Ptr(v int64) *int64 { return &v }
func uintPtr(v uint) *uint { return &v }
func float64Ptr(v float64) *float64 { return &v }
func strPtr(v string) *string { return &v }
