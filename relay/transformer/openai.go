package transformer

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
)

type OpenAITransformer struct{}
type OpenAIResponseTransformer struct{}
type OpenAIStreamTransformer struct{}

func init() {
	req := OpenAITransformer{}
	resp := OpenAIResponseTransformer{}
	stream := OpenAIStreamTransformer{}
	Register(types.RelayFormatOpenAI, req, resp, stream)
	Register(types.RelayFormatOpenAIResponses, req, resp, stream)
	Register(types.RelayFormatOpenAIResponsesCompaction, req, resp, stream)
}

func (t OpenAITransformer) Inbound(raw []byte) (*PivotRequest, error) {
	var generic dto.GeneralOpenAIRequest
	if err := common.Unmarshal(raw, &generic); err != nil {
		return nil, err
	}

	p := &PivotRequest{
		RelayFormat:         types.RelayFormatOpenAI,
		Model:               generic.Model,
		Prompt:              generic.Prompt,
		Prefix:              generic.Prefix,
		Suffix:              generic.Suffix,
		Stream:              generic.Stream,
		MaxTokens:           generic.MaxTokens,
		MaxCompletionTokens: generic.MaxCompletionTokens,
		Temperature:         generic.Temperature,
		TopP:                generic.TopP,
		TopK:                generic.TopK,
		N:                   generic.N,
		Input:               generic.Input,
		FrequencyPenalty:    generic.FrequencyPenalty,
		PresencePenalty:     generic.PresencePenalty,
		Seed:                generic.Seed,
		LogProbs:            generic.LogProbs,
		TopLogProbs:         generic.TopLogProbs,
		ParallelToolCalls:   generic.ParallelTooCalls,
	}
	if generic.Instruction != "" {
		v := generic.Instruction
		p.Instruction = &v
	}
	if generic.ReasoningEffort != "" {
		v := generic.ReasoningEffort
		p.ReasoningEffort = &v
	}
	if len(generic.Messages) > 0 {
		p.Messages = make([]PivotMessage, 0, len(generic.Messages))
		for _, m := range generic.Messages {
			p.Messages = append(p.Messages, openAIMessageToPivot(m))
		}
	}
	if generic.StreamOptions != nil {
		p.StreamOptions = &PivotStreamOptions{IncludeUsage: &generic.StreamOptions.IncludeUsage, IncludeObfuscation: &generic.StreamOptions.IncludeObfuscation}
	}
	if generic.ResponseFormat != nil {
		p.ResponseFormat = &PivotResponseFormat{Type: &generic.ResponseFormat.Type, JSONSchema: generic.ResponseFormat.JsonSchema}
	}
	if len(generic.Tools) > 0 {
		p.Tools = make([]PivotTool, 0, len(generic.Tools))
		for _, tool := range generic.Tools {
			pt := PivotTool{Type: tool.Type, Name: tool.Function.Name, Parameters: tool.Function.Parameters}
			if tool.Function.Description != "" {
				d := tool.Function.Description
				pt.Description = &d
			}
			p.Tools = append(p.Tools, pt)
		}
	}

	if generic.ToolChoice != nil {
		p.ToolChoice = &PivotToolChoice{}
		switch v := generic.ToolChoice.(type) {
		case string:
			p.ToolChoice.Type = &v
		case map[string]any:
			if tv, ok := v["type"].(string); ok {
				p.ToolChoice.Type = &tv
			}
			if fn, ok := v["function"].(map[string]any); ok {
				if n, ok := fn["name"].(string); ok {
					p.ToolChoice.Name = &n
				}
			}
		}
	}

	if stop, ok := generic.Stop.(string); ok && stop != "" {
		p.StopSequences = []string{stop}
	} else if arr, ok := generic.Stop.([]any); ok {
		for _, item := range arr {
			if s, ok := item.(string); ok {
				p.StopSequences = append(p.StopSequences, s)
			}
		}
	}

	// responses variant override
	var responses dto.OpenAIResponsesRequest
	var rawMap map[string]any
	_ = common.Unmarshal(raw, &rawMap)
	if err := common.Unmarshal(raw, &responses); err == nil && responses.Model != "" && isOpenAIResponsesPayload(rawMap) {
		p.RelayFormat = types.RelayFormatOpenAIResponses
		p.Model = responses.Model
		p.Stream = responses.Stream
		p.StreamOptions = nil
		if responses.StreamOptions != nil {
			p.StreamOptions = &PivotStreamOptions{IncludeUsage: &responses.StreamOptions.IncludeUsage, IncludeObfuscation: &responses.StreamOptions.IncludeObfuscation}
		}
		p.Temperature = responses.Temperature
		p.TopP = responses.TopP
		p.MaxTokens = responses.MaxOutputTokens
		p.Input = responses.Input
		if responses.Reasoning != nil && responses.Reasoning.Effort != "" {
			e := responses.Reasoning.Effort
			p.ReasoningEffort = &e
		}
	}

	p.ExtraFields = map[string]any{}
	b, _ := common.Marshal(generic)
	_ = common.Unmarshal(b, &p.ExtraFields)
	return p, nil
}

func (t OpenAITransformer) Outbound(pivot *PivotRequest) ([]byte, error) {
	if pivot == nil {
		return nil, fmt.Errorf("nil pivot request")
	}
	if pivot.RelayFormat == types.RelayFormatOpenAIResponses {
		r := dto.OpenAIResponsesRequest{Model: pivot.Model, Stream: pivot.Stream, Temperature: pivot.Temperature, TopP: pivot.TopP, MaxOutputTokens: pivot.MaxTokens}
		if pivot.StreamOptions != nil {
			r.StreamOptions = &dto.StreamOptions{}
			if pivot.StreamOptions.IncludeUsage != nil {
				r.StreamOptions.IncludeUsage = *pivot.StreamOptions.IncludeUsage
			}
			if pivot.StreamOptions.IncludeObfuscation != nil {
				r.StreamOptions.IncludeObfuscation = *pivot.StreamOptions.IncludeObfuscation
			}
		}
		if pivot.Input != nil {
			if raw, ok := pivot.Input.([]byte); ok {
				r.Input = raw
			} else {
				encoded, err := common.Marshal(pivot.Input)
				if err != nil {
					return nil, err
				}
				r.Input = encoded
			}
		}
		return common.Marshal(r)
	}

	req := dto.GeneralOpenAIRequest{
		Model:               pivot.Model,
		Prompt:              pivot.Prompt,
		Prefix:              pivot.Prefix,
		Suffix:              pivot.Suffix,
		Stream:              pivot.Stream,
		MaxTokens:           pivot.MaxTokens,
		MaxCompletionTokens: pivot.MaxCompletionTokens,
		Temperature:         pivot.Temperature,
		TopP:                pivot.TopP,
		TopK:                pivot.TopK,
		N:                   pivot.N,
		Input:               pivot.Input,
		FrequencyPenalty:    pivot.FrequencyPenalty,
		PresencePenalty:     pivot.PresencePenalty,
		Seed:                pivot.Seed,
		LogProbs:            pivot.LogProbs,
		TopLogProbs:         pivot.TopLogProbs,
		ParallelTooCalls:    pivot.ParallelToolCalls,
	}
	if pivot.Instruction != nil {
		req.Instruction = *pivot.Instruction
	}
	if pivot.ReasoningEffort != nil {
		req.ReasoningEffort = *pivot.ReasoningEffort
	}
	if pivot.StreamOptions != nil {
		req.StreamOptions = &dto.StreamOptions{}
		if pivot.StreamOptions.IncludeUsage != nil {
			req.StreamOptions.IncludeUsage = *pivot.StreamOptions.IncludeUsage
		}
		if pivot.StreamOptions.IncludeObfuscation != nil {
			req.StreamOptions.IncludeObfuscation = *pivot.StreamOptions.IncludeObfuscation
		}
	}
	if len(pivot.StopSequences) == 1 {
		req.Stop = pivot.StopSequences[0]
	} else if len(pivot.StopSequences) > 1 {
		req.Stop = pivot.StopSequences
	}
	if len(pivot.Messages) > 0 {
		req.Messages = make([]dto.Message, 0, len(pivot.Messages))
		for _, m := range pivot.Messages {
			req.Messages = append(req.Messages, pivotMessageToOpenAI(m))
		}
	}
	if len(pivot.Tools) > 0 {
		req.Tools = make([]dto.ToolCallRequest, 0, len(pivot.Tools))
		for _, tool := range pivot.Tools {
			tr := dto.ToolCallRequest{Type: tool.Type, Function: dto.FunctionRequest{Name: tool.Name, Parameters: tool.Parameters}}
			if tool.Description != nil {
				tr.Function.Description = *tool.Description
			}
			req.Tools = append(req.Tools, tr)
		}
	}
	if pivot.ToolChoice != nil {
		if pivot.ToolChoice.Name != nil {
			req.ToolChoice = map[string]any{"type": "function", "function": map[string]any{"name": *pivot.ToolChoice.Name}}
		} else if pivot.ToolChoice.Type != nil {
			req.ToolChoice = *pivot.ToolChoice.Type
		}
	}
	if pivot.ResponseFormat != nil {
		req.ResponseFormat = &dto.ResponseFormat{}
		if pivot.ResponseFormat.Type != nil {
			req.ResponseFormat.Type = *pivot.ResponseFormat.Type
		}
		req.ResponseFormat.JsonSchema = pivot.ResponseFormat.JSONSchema
	}
	return common.Marshal(req)
}

func (t OpenAIResponseTransformer) InboundResponse(raw []byte) (*PivotResponse, error) {
	var response dto.OpenAITextResponse
	if err := common.Unmarshal(raw, &response); err != nil {
		return nil, err
	}
	p := &PivotResponse{ID: response.Id, Object: response.Object, Model: response.Model, Error: response.Error}
	if created, ok := response.Created.(float64); ok {
		c := int64(created)
		p.Created = &c
	}
	if len(response.Choices) > 0 {
		p.Choices = make([]PivotChoice, 0, len(response.Choices))
		for _, ch := range response.Choices {
			idx := ch.Index
			finish := ch.FinishReason
			msg := openAIMessageToPivot(ch.Message)
			p.Choices = append(p.Choices, PivotChoice{Index: &idx, FinishReason: &finish, Message: &msg})
		}
	}
	p.Usage = ConvertOpenAIUsageToPivot(&response.Usage)
	return p, nil
}

func (t OpenAIResponseTransformer) OutboundResponse(pivot *PivotResponse) ([]byte, error) {
	if pivot == nil {
		return nil, fmt.Errorf("nil pivot response")
	}
	out := dto.OpenAITextResponse{Id: pivot.ID, Object: pivot.Object, Model: pivot.Model, Error: pivot.Error}
	if pivot.Created != nil {
		out.Created = *pivot.Created
	}
	if len(pivot.Choices) > 0 {
		out.Choices = make([]dto.OpenAITextResponseChoice, 0, len(pivot.Choices))
		for _, c := range pivot.Choices {
			choice := dto.OpenAITextResponseChoice{}
			if c.Index != nil {
				choice.Index = *c.Index
			}
			if c.Message != nil {
				choice.Message = pivotMessageToOpenAI(*c.Message)
			}
			if c.FinishReason != nil {
				choice.FinishReason = *c.FinishReason
			}
			out.Choices = append(out.Choices, choice)
		}
	}
	if pivot.Usage != nil {
		u := ConvertPivotUsageToOpenAI(pivot.Usage)
		if u != nil {
			out.Usage = *u
		}
	}
	return common.Marshal(out)
}

func (t OpenAIStreamTransformer) InboundStream(raw []byte) (*PivotResponse, error) {
	var chunk dto.ChatCompletionsStreamResponse
	if err := common.Unmarshal(raw, &chunk); err != nil {
		return nil, err
	}
	p := &PivotResponse{ID: chunk.Id, Object: chunk.Object, Model: chunk.Model, Usage: ConvertOpenAIUsageToPivot(chunk.Usage), SystemFingerprint: chunk.SystemFingerprint}
	c := chunk.Created
	p.Created = &c
	if len(chunk.Choices) > 0 {
		p.Choices = make([]PivotChoice, 0, len(chunk.Choices))
		for _, ch := range chunk.Choices {
			idx := ch.Index
			delta := PivotMessage{Role: ch.Delta.Role}
			if ch.Delta.Content != nil {
				delta.Parts = []PivotContent{{Type: dto.ContentTypeText, Text: ch.Delta.Content}}
			}
			if len(ch.Delta.ToolCalls) > 0 {
				delta.ToolCalls = make([]PivotToolCall, 0, len(ch.Delta.ToolCalls))
				for _, tc := range ch.Delta.ToolCalls {
					call := PivotToolCall{Arguments: []byte(tc.Function.Arguments)}
					if tc.ID != "" {
						id := tc.ID
						call.ID = &id
					}
					if tc.Function.Name != "" {
						n := tc.Function.Name
						call.Name = &n
					}
					delta.ToolCalls = append(delta.ToolCalls, call)
				}
			}
			p.Choices = append(p.Choices, PivotChoice{Index: &idx, Delta: &delta, FinishReason: ch.FinishReason})
		}
	}
	return p, nil
}

func (t OpenAIStreamTransformer) OutboundStream(pivot *PivotResponse) ([]byte, error) {
	if pivot == nil {
		return nil, fmt.Errorf("nil pivot stream response")
	}
	out := dto.ChatCompletionsStreamResponse{Id: pivot.ID, Object: pivot.Object, Model: pivot.Model, SystemFingerprint: pivot.SystemFingerprint}
	if pivot.Created != nil {
		out.Created = *pivot.Created
	}
	if pivot.Usage != nil {
		out.Usage = ConvertPivotUsageToOpenAI(pivot.Usage)
	}
	if len(pivot.Choices) > 0 {
		out.Choices = make([]dto.ChatCompletionsStreamResponseChoice, 0, len(pivot.Choices))
		for _, c := range pivot.Choices {
			ch := dto.ChatCompletionsStreamResponseChoice{}
			if c.Index != nil {
				ch.Index = *c.Index
			}
			if c.Delta != nil {
				ch.Delta.Role = c.Delta.Role
				for _, part := range c.Delta.Parts {
					if part.Type == dto.ContentTypeText && part.Text != nil {
						ch.Delta.Content = part.Text
						break
					}
				}
			}
			ch.FinishReason = c.FinishReason
			out.Choices = append(out.Choices, ch)
		}
	}
	return common.Marshal(out)
}

func openAIMessageToPivot(m dto.Message) PivotMessage {
	out := PivotMessage{Role: m.Role, Name: m.Name, Prefix: m.Prefix}
	if m.ToolCallId != "" {
		id := m.ToolCallId
		out.ToolCallID = &id
	}
	if s, ok := m.Content.(string); ok {
		out.Parts = []PivotContent{{Type: dto.ContentTypeText, Text: &s}}
	} else {
		for _, part := range m.ParseContent() {
			pc := PivotContent{Type: part.Type}
			if part.Type == dto.ContentTypeText {
				t := part.Text
				pc.Text = &t
			} else {
				pc.Media = &PivotMedia{Kind: part.Type}
				if img := part.GetImageMedia(); img != nil {
					pc.Media.URL = &img.Url
					pc.Media.Detail = &img.Detail
					pc.Media.MimeType = &img.MimeType
				}
			}
			out.Parts = append(out.Parts, pc)
		}
	}
	if len(m.ToolCalls) > 0 {
		var calls []dto.ToolCallResponse
		if common.Unmarshal(m.ToolCalls, &calls) == nil {
			out.ToolCalls = make([]PivotToolCall, 0, len(calls))
			for _, tc := range calls {
				call := PivotToolCall{Arguments: []byte(tc.Function.Arguments)}
				if tc.ID != "" {
					id := tc.ID
					call.ID = &id
				}
				if typeStr, ok := tc.Type.(string); ok {
					call.Type = &typeStr
				}
				if tc.Function.Name != "" {
					n := tc.Function.Name
					call.Name = &n
				}
				if tc.Index != nil {
					call.Index = tc.Index
				}
				out.ToolCalls = append(out.ToolCalls, call)
			}
		}
	}
	return out
}

func pivotMessageToOpenAI(m PivotMessage) dto.Message {
	out := dto.Message{Role: m.Role, Name: m.Name, Prefix: m.Prefix}
	if m.ToolCallID != nil {
		out.ToolCallId = *m.ToolCallID
	}
	if len(m.Parts) == 1 && m.Parts[0].Type == dto.ContentTypeText && m.Parts[0].Text != nil {
		out.Content = *m.Parts[0].Text
	} else {
		parts := make([]dto.MediaContent, 0, len(m.Parts))
		for _, p := range m.Parts {
			mc := dto.MediaContent{Type: p.Type}
			if p.Type == dto.ContentTypeText && p.Text != nil {
				mc.Text = *p.Text
			}
			if p.Media != nil {
				if p.Media.Kind != "" {
					mc.Type = p.Media.Kind
				}
				if p.Media.URL != nil {
					mc.ImageUrl = map[string]any{"url": *p.Media.URL}
					if p.Media.Detail != nil {
						mc.ImageUrl = map[string]any{"url": *p.Media.URL, "detail": *p.Media.Detail}
					}
				}
			}
			parts = append(parts, mc)
		}
		out.Content = parts
	}
	if len(m.ToolCalls) > 0 {
		calls := make([]dto.ToolCallResponse, 0, len(m.ToolCalls))
		for _, tc := range m.ToolCalls {
			c := dto.ToolCallResponse{Function: dto.FunctionResponse{}}
			if tc.ID != nil {
				c.ID = *tc.ID
			}
			if tc.Type != nil {
				c.Type = *tc.Type
			}
			if tc.Name != nil {
				c.Function.Name = *tc.Name
			}
			if len(tc.Arguments) > 0 {
				c.Function.Arguments = string(tc.Arguments)
			}
			if tc.Index != nil {
				c.Index = tc.Index
			}
			calls = append(calls, c)
		}
		raw, _ := common.Marshal(calls)
		out.ToolCalls = raw
	}
	return out
}

func isOpenAIResponsesPayload(raw map[string]any) bool {
	if len(raw) == 0 {
		return false
	}
	responsesOnlyKeys := []string{
		"max_output_tokens",
		"previous_response_id",
		"truncation",
		"context_management",
		"conversation",
		"max_tool_calls",
	}
	for _, k := range responsesOnlyKeys {
		if _, ok := raw[k]; ok {
			return true
		}
	}
	return false
}
