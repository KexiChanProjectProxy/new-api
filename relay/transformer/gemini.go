package transformer

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
)

type GeminiTransformer struct{}
type GeminiResponseTransformer struct{}
type GeminiStreamTransformer struct{}

func init() {
	req := GeminiTransformer{}
	resp := GeminiResponseTransformer{}
	stream := GeminiStreamTransformer{}
	Register(types.RelayFormatGemini, req, resp, stream)
}

func (t GeminiTransformer) Inbound(raw []byte) (*PivotRequest, error) {
	var req dto.GeminiChatRequest
	if err := common.Unmarshal(raw, &req); err != nil {
		return nil, err
	}

	p := &PivotRequest{
		RelayFormat:         types.RelayFormatGemini,
		Temperature:         req.GenerationConfig.Temperature,
		TopP:                req.GenerationConfig.TopP,
		MaxCompletionTokens: req.GenerationConfig.MaxOutputTokens,
		N:                   req.GenerationConfig.CandidateCount,
		StopSequences:       req.GenerationConfig.StopSequences,
	}

	if req.GenerationConfig.TopK != nil {
		tk := int(*req.GenerationConfig.TopK)
		p.TopK = &tk
	}
	if req.GenerationConfig.PresencePenalty != nil {
		v := float64(*req.GenerationConfig.PresencePenalty)
		p.PresencePenalty = &v
	}
	if req.GenerationConfig.FrequencyPenalty != nil {
		v := float64(*req.GenerationConfig.FrequencyPenalty)
		p.FrequencyPenalty = &v
	}
	if req.GenerationConfig.ResponseLogprobs != nil {
		p.LogProbs = req.GenerationConfig.ResponseLogprobs
	}
	if req.GenerationConfig.Logprobs != nil {
		v := int(*req.GenerationConfig.Logprobs)
		p.TopLogProbs = &v
	}
	if req.GenerationConfig.Seed != nil {
		v := float64(*req.GenerationConfig.Seed)
		p.Seed = &v
	}

	if req.GenerationConfig.ThinkingConfig != nil {
		p.Thinking = &PivotThinkingConfig{
			IncludeThoughts: &req.GenerationConfig.ThinkingConfig.IncludeThoughts,
			BudgetTokens:    req.GenerationConfig.ThinkingConfig.ThinkingBudget,
		}
		if req.GenerationConfig.ThinkingConfig.ThinkingLevel != "" {
			lvl := req.GenerationConfig.ThinkingConfig.ThinkingLevel
			p.Thinking.Level = &lvl
		}
	}

	if req.GenerationConfig.ResponseMimeType != "" || len(req.GenerationConfig.ResponseJsonSchema) > 0 || req.GenerationConfig.ResponseSchema != nil {
		p.ResponseFormat = &PivotResponseFormat{}
		if req.GenerationConfig.ResponseMimeType != "" {
			mt := req.GenerationConfig.ResponseMimeType
			p.ResponseFormat.MimeType = &mt
		}
		if len(req.GenerationConfig.ResponseJsonSchema) > 0 {
			p.ResponseFormat.JSONSchema = req.GenerationConfig.ResponseJsonSchema
		}
		if req.GenerationConfig.ResponseSchema != nil {
			p.ResponseFormat.Schema = req.GenerationConfig.ResponseSchema
		}
	}

	if req.SystemInstructions != nil && len(req.SystemInstructions.Parts) > 0 {
		p.System = &PivotSystemPrompt{Parts: geminiPartsToPivot(req.SystemInstructions.Parts)}
	}

	if len(req.Contents) > 0 {
		p.Messages = make([]PivotMessage, 0, len(req.Contents))
		for _, content := range req.Contents {
			p.Messages = append(p.Messages, PivotMessage{Role: geminiRoleToPivot(content.Role), Parts: geminiPartsToPivot(content.Parts)})
		}
	}

	if len(req.SafetySettings) > 0 {
		p.SafetySettings = make([]PivotSafetySetting, 0, len(req.SafetySettings))
		for _, s := range req.SafetySettings {
			threshold := s.Threshold
			p.SafetySettings = append(p.SafetySettings, PivotSafetySetting{Category: s.Category, Threshold: &threshold})
		}
	}

	tools := req.GetTools()
	if len(tools) > 0 {
		p.Tools = make([]PivotTool, 0)
		for _, tool := range tools {
			if tool.FunctionDeclarations != nil {
				decls := []map[string]any{}
				if b, err := common.Marshal(tool.FunctionDeclarations); err == nil && common.Unmarshal(b, &decls) == nil {
					for _, d := range decls {
						name, _ := d["name"].(string)
						if name == "" {
							continue
						}
						pt := PivotTool{Type: "function", Name: name, Parameters: d["parameters"]}
						if desc, ok := d["description"].(string); ok && desc != "" {
							pt.Description = &desc
						}
						p.Tools = append(p.Tools, pt)
					}
				}
			}
		}
	}

	if req.ToolConfig != nil {
		tc := &PivotToolConfig{}
		if req.ToolConfig.FunctionCallingConfig != nil {
			mode := string(req.ToolConfig.FunctionCallingConfig.Mode)
			tc.FunctionCalling = &PivotFunctionCallingConfig{Mode: &mode, AllowedFunctionNames: req.ToolConfig.FunctionCallingConfig.AllowedFunctionNames}
		}
		if req.ToolConfig.RetrievalConfig != nil {
			tc.Retrieval = &PivotRetrievalConfig{LanguageCode: usageStrPtr(req.ToolConfig.RetrievalConfig.LanguageCode)}
			if req.ToolConfig.RetrievalConfig.LatLng != nil {
				tc.Retrieval.Latitude = req.ToolConfig.RetrievalConfig.LatLng.Latitude
				tc.Retrieval.Longitude = req.ToolConfig.RetrievalConfig.LatLng.Longitude
			}
		}
		p.ToolConfig = tc
	}

	if req.CachedContent != "" {
		cc := req.CachedContent
		p.CachedContent = &cc
	}

	return p, nil
}

func (t GeminiTransformer) Outbound(pivot *PivotRequest) ([]byte, error) {
	if pivot == nil {
		return nil, fmt.Errorf("nil pivot request")
	}

	req := dto.GeminiChatRequest{}
	req.GenerationConfig = dto.GeminiChatGenerationConfig{
		Temperature:     pivot.Temperature,
		TopP:            pivot.TopP,
		MaxOutputTokens: pivot.MaxCompletionTokens,
		CandidateCount:  pivot.N,
		StopSequences:   pivot.StopSequences,
	}

	if pivot.TopK != nil {
		tk := float64(*pivot.TopK)
		req.GenerationConfig.TopK = &tk
	}
	if pivot.PresencePenalty != nil {
		v := float32(*pivot.PresencePenalty)
		req.GenerationConfig.PresencePenalty = &v
	}
	if pivot.FrequencyPenalty != nil {
		v := float32(*pivot.FrequencyPenalty)
		req.GenerationConfig.FrequencyPenalty = &v
	}
	if pivot.LogProbs != nil {
		req.GenerationConfig.ResponseLogprobs = pivot.LogProbs
	}
	if pivot.TopLogProbs != nil {
		v := int32(*pivot.TopLogProbs)
		req.GenerationConfig.Logprobs = &v
	}
	if pivot.Seed != nil {
		v := int64(*pivot.Seed)
		req.GenerationConfig.Seed = &v
	}
	if pivot.Thinking != nil {
		req.GenerationConfig.ThinkingConfig = &dto.GeminiThinkingConfig{}
		if pivot.Thinking.IncludeThoughts != nil {
			req.GenerationConfig.ThinkingConfig.IncludeThoughts = *pivot.Thinking.IncludeThoughts
		}
		req.GenerationConfig.ThinkingConfig.ThinkingBudget = pivot.Thinking.BudgetTokens
		if pivot.Thinking.Level != nil {
			req.GenerationConfig.ThinkingConfig.ThinkingLevel = *pivot.Thinking.Level
		}
	}
	if pivot.ResponseFormat != nil {
		if pivot.ResponseFormat.MimeType != nil {
			req.GenerationConfig.ResponseMimeType = *pivot.ResponseFormat.MimeType
		}
		req.GenerationConfig.ResponseJsonSchema = pivot.ResponseFormat.JSONSchema
		req.GenerationConfig.ResponseSchema = pivot.ResponseFormat.Schema
	}

	if pivot.System != nil {
		req.SystemInstructions = &dto.GeminiChatContent{Role: "user", Parts: pivotPartsToGemini(pivot.System.Parts)}
		if len(req.SystemInstructions.Parts) == 0 && pivot.System.Text != nil {
			req.SystemInstructions.Parts = []dto.GeminiPart{{Text: *pivot.System.Text}}
		}
	}

	if len(pivot.Messages) > 0 {
		req.Contents = make([]dto.GeminiChatContent, 0, len(pivot.Messages))
		for _, msg := range pivot.Messages {
			req.Contents = append(req.Contents, dto.GeminiChatContent{Role: pivotRoleToGemini(msg.Role), Parts: pivotPartsToGemini(msg.Parts)})
		}
	}

	if len(pivot.SafetySettings) > 0 {
		req.SafetySettings = make([]dto.GeminiChatSafetySettings, 0, len(pivot.SafetySettings))
		for _, s := range pivot.SafetySettings {
			reqS := dto.GeminiChatSafetySettings{Category: s.Category}
			if s.Threshold != nil {
				reqS.Threshold = *s.Threshold
			}
			req.SafetySettings = append(req.SafetySettings, reqS)
		}
	}

	if len(pivot.Tools) > 0 {
		decls := make([]map[string]any, 0, len(pivot.Tools))
		for _, tool := range pivot.Tools {
			decl := map[string]any{"name": tool.Name}
			if tool.Description != nil {
				decl["description"] = *tool.Description
			}
			if tool.Parameters != nil {
				decl["parameters"] = tool.Parameters
			}
			decls = append(decls, decl)
		}
		req.SetTools([]dto.GeminiChatTool{{FunctionDeclarations: decls}})
	}

	if pivot.ToolConfig != nil {
		req.ToolConfig = &dto.ToolConfig{}
		if pivot.ToolConfig.FunctionCalling != nil {
			req.ToolConfig.FunctionCallingConfig = &dto.FunctionCallingConfig{AllowedFunctionNames: pivot.ToolConfig.FunctionCalling.AllowedFunctionNames}
			if pivot.ToolConfig.FunctionCalling.Mode != nil {
				req.ToolConfig.FunctionCallingConfig.Mode = dto.FunctionCallingConfigMode(*pivot.ToolConfig.FunctionCalling.Mode)
			}
		}
		if pivot.ToolConfig.Retrieval != nil {
			req.ToolConfig.RetrievalConfig = &dto.RetrievalConfig{}
			if pivot.ToolConfig.Retrieval.LanguageCode != nil {
				req.ToolConfig.RetrievalConfig.LanguageCode = *pivot.ToolConfig.Retrieval.LanguageCode
			}
			if pivot.ToolConfig.Retrieval.Latitude != nil || pivot.ToolConfig.Retrieval.Longitude != nil {
				req.ToolConfig.RetrievalConfig.LatLng = &dto.LatLng{Latitude: pivot.ToolConfig.Retrieval.Latitude, Longitude: pivot.ToolConfig.Retrieval.Longitude}
			}
		}
	}

	if pivot.CachedContent != nil {
		req.CachedContent = *pivot.CachedContent
	}

	return common.Marshal(req)
}

func (t GeminiResponseTransformer) InboundResponse(raw []byte) (*PivotResponse, error) {
	var response dto.GeminiChatResponse
	if err := common.Unmarshal(raw, &response); err != nil {
		return nil, err
	}
	p := &PivotResponse{Object: "gemini.response"}
	if len(response.Candidates) > 0 {
		p.Choices = make([]PivotChoice, 0, len(response.Candidates))
		for _, c := range response.Candidates {
			idx := int(c.Index)
			msg := PivotMessage{Role: geminiRoleToPivot(c.Content.Role), Parts: geminiPartsToPivot(c.Content.Parts)}
			choice := PivotChoice{Index: &idx, Message: &msg, SafetyRatings: geminiSafetyRatingsToPivot(c.SafetyRatings)}
			if c.FinishReason != nil {
				finish := *c.FinishReason
				choice.FinishReason = &finish
			}
			p.Choices = append(p.Choices, choice)
		}
	}
	p.Usage = geminiUsageToPivot(response.UsageMetadata)
	if response.PromptFeedback != nil {
		p.ProviderExtensions = map[string]any{"gemini_prompt_feedback": response.PromptFeedback}
	}
	return p, nil
}

func (t GeminiResponseTransformer) OutboundResponse(pivot *PivotResponse) ([]byte, error) {
	if pivot == nil {
		return nil, fmt.Errorf("nil pivot response")
	}
	out := dto.GeminiChatResponse{Candidates: make([]dto.GeminiChatCandidate, 0, len(pivot.Choices)), UsageMetadata: pivotUsageToGemini(pivot.Usage)}
	for _, c := range pivot.Choices {
		candidate := dto.GeminiChatCandidate{SafetyRatings: pivotSafetyRatingsToGemini(c.SafetyRatings)}
		if c.Index != nil {
			candidate.Index = int64(*c.Index)
		}
		if c.Message != nil {
			candidate.Content = dto.GeminiChatContent{Role: pivotRoleToGemini(c.Message.Role), Parts: pivotPartsToGemini(c.Message.Parts)}
		}
		if c.FinishReason != nil {
			finish := *c.FinishReason
			candidate.FinishReason = &finish
		}
		out.Candidates = append(out.Candidates, candidate)
	}
	if pivot.ProviderExtensions != nil && pivot.ProviderExtensions["gemini_prompt_feedback"] != nil {
		if b, err := common.Marshal(pivot.ProviderExtensions["gemini_prompt_feedback"]); err == nil {
			var pf dto.GeminiChatPromptFeedback
			if common.Unmarshal(b, &pf) == nil {
				out.PromptFeedback = &pf
			}
		}
	}
	return common.Marshal(out)
}

func (t GeminiStreamTransformer) InboundStream(raw []byte) (*PivotResponse, error) {
	return GeminiResponseTransformer{}.InboundResponse(raw)
}

func (t GeminiStreamTransformer) OutboundStream(pivot *PivotResponse) ([]byte, error) {
	return GeminiResponseTransformer{}.OutboundResponse(pivot)
}

func geminiRoleToPivot(role string) string {
	switch role {
	case "model":
		return "assistant"
	case "user":
		return "user"
	default:
		return role
	}
}

func pivotRoleToGemini(role string) string {
	switch role {
	case "assistant":
		return "model"
	case "user":
		return "user"
	default:
		return role
	}
}

func geminiPartsToPivot(parts []dto.GeminiPart) []PivotContent {
	out := make([]PivotContent, 0, len(parts))
	for _, p := range parts {
		switch {
		case p.Text != "":
			t := p.Text
			out = append(out, PivotContent{Type: dto.ContentTypeText, Text: &t})
		case p.InlineData != nil:
			pc := PivotContent{Type: "inline_data", Media: &PivotMedia{Kind: "inline_data"}}
			if p.InlineData.MimeType != "" {
				mt := p.InlineData.MimeType
				pc.Media.MimeType = &mt
			}
			if p.InlineData.Data != "" {
				d := p.InlineData.Data
				pc.Media.Data = &d
			}
			out = append(out, pc)
		case p.FileData != nil:
			pc := PivotContent{Type: "file_data", Media: &PivotMedia{Kind: "file_data"}}
			if p.FileData.FileUri != "" {
				u := p.FileData.FileUri
				pc.Media.URL = &u
			}
			if p.FileData.MimeType != "" {
				mt := p.FileData.MimeType
				pc.Media.MimeType = &mt
			}
			out = append(out, pc)
		case p.FunctionCall != nil:
			pc := PivotContent{Type: "function_call", FunctionCall: &PivotFunctionCall{}}
			if p.FunctionCall.FunctionName != "" {
				n := p.FunctionCall.FunctionName
				pc.FunctionCall.Name = &n
			}
			if p.FunctionCall.Arguments != nil {
				if b, err := common.Marshal(p.FunctionCall.Arguments); err == nil {
					pc.FunctionCall.Arguments = b
				}
			}
			out = append(out, pc)
		case p.FunctionResponse != nil:
			pc := PivotContent{Type: "function_response", FunctionResponse: &PivotFunctionResult{Name: &p.FunctionResponse.Name, Response: p.FunctionResponse.Response}}
			out = append(out, pc)
		}
	}
	return out
}

func pivotPartsToGemini(parts []PivotContent) []dto.GeminiPart {
	out := make([]dto.GeminiPart, 0, len(parts))
	for _, p := range parts {
		switch {
		case p.Text != nil:
			out = append(out, dto.GeminiPart{Text: *p.Text})
		case p.FunctionCall != nil:
			call := &dto.FunctionCall{}
			if p.FunctionCall.Name != nil {
				call.FunctionName = *p.FunctionCall.Name
			}
			if len(p.FunctionCall.Arguments) > 0 {
				var args any
				if common.Unmarshal(p.FunctionCall.Arguments, &args) == nil {
					call.Arguments = args
				}
			}
			out = append(out, dto.GeminiPart{FunctionCall: call})
		case p.FunctionResponse != nil:
			resp := &dto.GeminiFunctionResponse{Response: map[string]interface{}{}}
			if p.FunctionResponse.Name != nil {
				resp.Name = *p.FunctionResponse.Name
			}
			if m, ok := p.FunctionResponse.Response.(map[string]interface{}); ok {
				resp.Response = m
			}
			out = append(out, dto.GeminiPart{FunctionResponse: resp})
		case p.Media != nil:
			kind := p.Media.Kind
			if kind == "" {
				kind = p.Type
			}
			if kind == "inline_data" {
				part := dto.GeminiPart{InlineData: &dto.GeminiInlineData{}}
				if p.Media.MimeType != nil {
					part.InlineData.MimeType = *p.Media.MimeType
				}
				if p.Media.Data != nil {
					part.InlineData.Data = *p.Media.Data
				}
				out = append(out, part)
			} else if kind == "file_data" {
				part := dto.GeminiPart{FileData: &dto.GeminiFileData{}}
				if p.Media.MimeType != nil {
					part.FileData.MimeType = *p.Media.MimeType
				}
				if p.Media.URL != nil {
					part.FileData.FileUri = *p.Media.URL
				}
				out = append(out, part)
			}
		}
	}
	return out
}

func geminiSafetyRatingsToPivot(ratings []dto.GeminiChatSafetyRating) []PivotSafetySetting {
	out := make([]PivotSafetySetting, 0, len(ratings))
	for _, r := range ratings {
		probability := r.Probability
		out = append(out, PivotSafetySetting{Category: r.Category, Probability: &probability})
	}
	return out
}

func pivotSafetyRatingsToGemini(ratings []PivotSafetySetting) []dto.GeminiChatSafetyRating {
	out := make([]dto.GeminiChatSafetyRating, 0, len(ratings))
	for _, r := range ratings {
		item := dto.GeminiChatSafetyRating{Category: r.Category}
		if r.Probability != nil {
			item.Probability = *r.Probability
		}
		out = append(out, item)
	}
	return out
}

func geminiUsageToPivot(usage dto.GeminiUsageMetadata) *PivotUsage {
		u := &PivotUsage{
		PromptTokens:             usageIntPtr(usage.PromptTokenCount),
		CompletionTokens:         usageIntPtr(usage.CandidatesTokenCount),
		TotalTokens:              usageIntPtr(usage.TotalTokenCount),
		InputTokens:              usageIntPtr(usage.PromptTokenCount),
		OutputTokens:             usageIntPtr(usage.CandidatesTokenCount),
		PromptCacheHitTokens:     usageIntPtr(usage.CachedContentTokenCount),
		CacheReadInputTokens:     usageIntPtr(usage.CachedContentTokenCount),
		ThoughtsTokenCount:       usageIntPtr(usage.ThoughtsTokenCount),
	}
	return NormalizeUsage(u)
}

func pivotUsageToGemini(usage *PivotUsage) dto.GeminiUsageMetadata {
	n := NormalizeUsage(usage)
	if n == nil {
		return dto.GeminiUsageMetadata{}
	}
	return dto.GeminiUsageMetadata{
		PromptTokenCount:        intValue(n.PromptTokens),
		CandidatesTokenCount:    intValue(n.CompletionTokens),
		TotalTokenCount:         intValue(n.TotalTokens),
		CachedContentTokenCount: intValue(n.CacheReadInputTokens),
		ThoughtsTokenCount:      intValue(n.ThoughtsTokenCount),
	}
}
