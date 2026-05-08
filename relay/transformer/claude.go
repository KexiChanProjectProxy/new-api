package transformer

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
)

type ClaudeTransformer struct{}
type ClaudeResponseTransformer struct{}
type ClaudeStreamTransformer struct{}

func init() {
	req := ClaudeTransformer{}
	resp := ClaudeResponseTransformer{}
	stream := ClaudeStreamTransformer{}
	Register(types.RelayFormatClaude, req, resp, stream)
}

func (t ClaudeTransformer) Inbound(raw []byte) (*PivotRequest, error) {
	var req dto.ClaudeRequest
	if err := common.Unmarshal(raw, &req); err != nil {
		return nil, err
	}

	p := &PivotRequest{
		RelayFormat:   types.RelayFormatClaude,
		Model:         req.Model,
		Stream:        req.Stream,
		MaxTokens:     req.MaxTokens,
		Temperature:   req.Temperature,
		TopP:          req.TopP,
		TopK:          req.TopK,
		StopSequences: req.StopSequences,
	}

	if req.Thinking != nil {
		thinkingType := req.Thinking.Type
		p.Thinking = &PivotThinkingConfig{Type: &thinkingType, BudgetTokens: req.Thinking.BudgetTokens}
	}

	if req.Metadata != nil {
		p.ProviderExtensions = map[string]any{"claude_metadata": req.Metadata}
	}

	if req.System != nil {
		if req.IsStringSystem() {
			s := req.GetStringSystem()
			p.System = &PivotSystemPrompt{Text: &s}
		} else {
			parts := claudeContentBlocksToPivot(req.ParseSystem())
			if len(parts) > 0 {
				p.System = &PivotSystemPrompt{Parts: parts}
			}
		}
	}

	if len(req.Messages) > 0 {
		p.Messages = make([]PivotMessage, 0, len(req.Messages))
		for _, msg := range req.Messages {
			pivotMsg := PivotMessage{Role: msg.Role}
			if s, ok := msg.Content.(string); ok {
				pivotMsg.Parts = []PivotContent{{Type: dto.ContentTypeText, Text: &s}}
			} else {
				blocks, err := msg.ParseContent()
				if err != nil {
					return nil, err
				}
				pivotMsg.Parts = claudeContentBlocksToPivot(blocks)
			}
			p.Messages = append(p.Messages, pivotMsg)
		}
	}

	if req.Tools != nil {
		var tools []dto.Tool
		if b, err := common.Marshal(req.Tools); err == nil && common.Unmarshal(b, &tools) == nil {
			p.Tools = make([]PivotTool, 0, len(tools))
			for _, tool := range tools {
				pt := PivotTool{Type: "function", Name: tool.Name, Parameters: tool.InputSchema}
				if tool.Description != "" {
					d := tool.Description
					pt.Description = &d
				}
				p.Tools = append(p.Tools, pt)
			}
		}
	}

	if req.ToolChoice != nil {
		var toolChoice dto.ClaudeToolChoice
		if b, err := common.Marshal(req.ToolChoice); err == nil && common.Unmarshal(b, &toolChoice) == nil {
			tc := &PivotToolChoice{}
			if toolChoice.Type != "" {
				v := toolChoice.Type
				tc.Type = &v
			}
			if toolChoice.Name != "" {
				v := toolChoice.Name
				tc.Name = &v
			}
			if toolChoice.DisableParallelToolUse {
				v := toolChoice.DisableParallelToolUse
				tc.DisableParallelToolUse = &v
			}
			p.ToolChoice = tc
		}
	}

	return p, nil
}

func (t ClaudeTransformer) Outbound(pivot *PivotRequest) ([]byte, error) {
	if pivot == nil {
		return nil, fmt.Errorf("nil pivot request")
	}

	req := dto.ClaudeRequest{
		Model:         pivot.Model,
		MaxTokens:     pivot.MaxTokens,
		Temperature:   pivot.Temperature,
		TopP:          pivot.TopP,
		TopK:          pivot.TopK,
		Stream:        pivot.Stream,
		StopSequences: pivot.StopSequences,
	}

	if pivot.System != nil {
		if pivot.System.Text != nil {
			req.System = *pivot.System.Text
		} else if len(pivot.System.Parts) > 0 {
			req.System = pivotContentsToClaudeBlocks(pivot.System.Parts)
		}
	}

	if len(pivot.Messages) > 0 {
		req.Messages = make([]dto.ClaudeMessage, 0, len(pivot.Messages))
		for _, msg := range pivot.Messages {
			out := dto.ClaudeMessage{Role: msg.Role}
			if len(msg.Parts) == 1 && msg.Parts[0].Type == dto.ContentTypeText && msg.Parts[0].Text != nil {
				out.Content = *msg.Parts[0].Text
			} else {
				out.Content = pivotContentsToClaudeBlocks(msg.Parts)
			}
			req.Messages = append(req.Messages, out)
		}
	}

	if len(pivot.Tools) > 0 {
		tools := make([]dto.Tool, 0, len(pivot.Tools))
		for _, tool := range pivot.Tools {
			t := dto.Tool{Name: tool.Name}
			if tool.Description != nil {
				t.Description = *tool.Description
			}
			if schema, ok := tool.Parameters.(map[string]any); ok {
				t.InputSchema = schema
			}
			tools = append(tools, t)
		}
		req.Tools = tools
	}

	if pivot.ToolChoice != nil {
		tc := dto.ClaudeToolChoice{}
		if pivot.ToolChoice.Type != nil {
			tc.Type = *pivot.ToolChoice.Type
		}
		if pivot.ToolChoice.Name != nil {
			tc.Name = *pivot.ToolChoice.Name
		}
		if pivot.ToolChoice.DisableParallelToolUse != nil {
			tc.DisableParallelToolUse = *pivot.ToolChoice.DisableParallelToolUse
		}
		req.ToolChoice = tc
	}

	if pivot.Thinking != nil {
		t := &dto.Thinking{}
		if pivot.Thinking.Type != nil {
			t.Type = *pivot.Thinking.Type
		}
		t.BudgetTokens = pivot.Thinking.BudgetTokens
		req.Thinking = t
	}

	if pivot.ProviderExtensions != nil {
		if raw, ok := pivot.ProviderExtensions["claude_metadata"].([]byte); ok {
			req.Metadata = raw
		} else if v, ok := pivot.ProviderExtensions["claude_metadata"]; ok {
			b, err := common.Marshal(v)
			if err != nil {
				return nil, err
			}
			req.Metadata = b
		}
	}

	return common.Marshal(req)
}

func (t ClaudeResponseTransformer) InboundResponse(raw []byte) (*PivotResponse, error) {
	var response dto.ClaudeResponse
	if err := common.Unmarshal(raw, &response); err != nil {
		return nil, err
	}

	p := &PivotResponse{
		ID:     response.Id,
		Object: response.Type,
		Model:  response.Model,
		Error:  response.Error,
		Usage:  convertClaudeUsageToPivot(response.Usage),
	}

	if response.Type == "message" {
		msg := PivotMessage{Role: response.Role, Parts: claudeContentBlocksToPivot(response.Content)}
		idx := 0
		p.Choices = []PivotChoice{{Index: &idx, Message: &msg}}
		if response.StopReason != "" {
			stop := response.StopReason
			p.Choices[0].FinishReason = &stop
		}
	}

	return p, nil
}

func (t ClaudeResponseTransformer) OutboundResponse(pivot *PivotResponse) ([]byte, error) {
	if pivot == nil {
		return nil, fmt.Errorf("nil pivot response")
	}
	out := dto.ClaudeResponse{Id: pivot.ID, Type: "message", Model: pivot.Model, Error: pivot.Error}
	if pivot.Object != "" {
		out.Type = pivot.Object
	}
	if len(pivot.Choices) > 0 {
		if pivot.Choices[0].Message != nil {
			out.Role = pivot.Choices[0].Message.Role
			out.Content = pivotContentsToClaudeBlocks(pivot.Choices[0].Message.Parts)
		}
		if pivot.Choices[0].FinishReason != nil {
			out.StopReason = *pivot.Choices[0].FinishReason
		}
	}
	out.Usage = convertPivotUsageToClaude(pivot.Usage)
	return common.Marshal(out)
}

func (t ClaudeStreamTransformer) InboundStream(raw []byte) (*PivotResponse, error) {
	var chunk dto.ClaudeResponse
	if err := common.Unmarshal(raw, &chunk); err != nil {
		return nil, err
	}

	p := &PivotResponse{ID: chunk.Id, Object: chunk.Type, Model: chunk.Model, Usage: convertClaudeUsageToPivot(chunk.Usage), Error: chunk.Error}
	if chunk.Index != nil {
		idx := *chunk.Index
		p.Choices = []PivotChoice{{Index: &idx}}
	}

	switch chunk.Type {
	case "message_start":
		if chunk.Message != nil {
			p.ID = chunk.Message.Id
			p.Model = chunk.Message.Model
			p.Object = chunk.Message.Type
			p.Usage = convertClaudeUsageToPivot(chunk.Message.Usage)
		}
	case "content_block_delta":
		if chunk.Delta != nil {
			delta := PivotMessage{}
			switch chunk.Delta.Type {
			case "text_delta":
				if chunk.Delta.Text != nil {
					delta.Parts = []PivotContent{{Type: dto.ContentTypeText, Text: chunk.Delta.Text}}
				}
			case "input_json_delta":
				partial := chunk.Delta.PartialJson
				if partial != nil {
					delta.Parts = []PivotContent{{Type: "tool_use", ToolCall: &PivotToolCall{Arguments: []byte(*partial)}}}
				}
			}
			if len(p.Choices) == 0 {
				idx := 0
				p.Choices = []PivotChoice{{Index: &idx, Delta: &delta}}
			} else {
				p.Choices[0].Delta = &delta
			}
		}
	case "message_delta":
		if chunk.Delta != nil {
			if chunk.Delta.StopReason != nil {
				stop := *chunk.Delta.StopReason
				if len(p.Choices) == 0 {
					idx := 0
					p.Choices = []PivotChoice{{Index: &idx, FinishReason: &stop}}
				} else {
					p.Choices[0].FinishReason = &stop
				}
			}
			if chunk.Usage != nil {
				p.Usage = convertClaudeUsageToPivot(chunk.Usage)
			}
		}
	}

	return p, nil
}

func (t ClaudeStreamTransformer) OutboundStream(pivot *PivotResponse) ([]byte, error) {
	if pivot == nil {
		return nil, fmt.Errorf("nil pivot stream response")
	}
	out := dto.ClaudeResponse{Id: pivot.ID, Type: pivot.Object, Model: pivot.Model, Usage: convertPivotUsageToClaude(pivot.Usage), Error: pivot.Error}
	if len(pivot.Choices) > 0 {
		if pivot.Choices[0].Index != nil {
			idx := *pivot.Choices[0].Index
			out.Index = &idx
		}
		if pivot.Choices[0].Delta != nil {
			d := pivot.Choices[0].Delta
			if len(d.Parts) > 0 {
				part := d.Parts[0]
				if part.Type == dto.ContentTypeText && part.Text != nil {
					out.Delta = &dto.ClaudeMediaMessage{Type: "text_delta", Text: part.Text}
				} else if part.ToolCall != nil {
					partial := string(part.ToolCall.Arguments)
					out.Delta = &dto.ClaudeMediaMessage{Type: "input_json_delta", PartialJson: &partial}
				}
			}
		}
		if pivot.Choices[0].FinishReason != nil {
			if out.Delta == nil {
				out.Delta = &dto.ClaudeMediaMessage{}
			}
			out.Delta.StopReason = pivot.Choices[0].FinishReason
		}
	}
	if out.Type == "" {
		out.Type = "message_delta"
	}
	return common.Marshal(out)
}

func claudeContentBlocksToPivot(blocks []dto.ClaudeMediaMessage) []PivotContent {
	out := make([]PivotContent, 0, len(blocks))
	for _, block := range blocks {
		pc := PivotContent{Type: block.Type}
		switch block.Type {
		case dto.ContentTypeText, "text_delta":
			if block.Text != nil {
				t := *block.Text
				pc.Type = dto.ContentTypeText
				pc.Text = &t
			}
		case "image":
			pc.Media = &PivotMedia{Kind: "image"}
			if block.Source != nil {
				if data, ok := block.Source.Data.(string); ok && data != "" {
					u := "data:" + block.Source.MediaType + ";base64," + data
					pc.Media.URL = &u
				}
				if block.Source.MediaType != "" {
					mt := block.Source.MediaType
					pc.Media.MimeType = &mt
				}
			}
		case "tool_use":
			tc := &PivotToolCall{}
			if block.Id != "" {
				id := block.Id
				tc.ID = &id
			}
			if block.Name != "" {
				n := block.Name
				tc.Name = &n
			}
			if block.Input != nil {
				b, err := common.Marshal(block.Input)
				if err == nil {
					tc.Arguments = b
				}
			}
			pc.ToolCall = tc
		case "tool_result":
			tr := &PivotToolResult{Content: block.Content}
			if block.ToolUseId != "" {
				id := block.ToolUseId
				tr.ToolCallID = &id
			}
			pc.ToolResult = tr
		case "thinking", "redacted_thinking":
			pc.Type = "thinking"
			pc.Thinking = &PivotThinkingBlock{Text: block.Thinking}
			if block.Signature != "" {
				s := block.Signature
				pc.Thinking.Signature = &s
			}
		default:
			if block.Text != nil {
				pc.Text = block.Text
			}
		}
		out = append(out, pc)
	}
	return out
}

func pivotContentsToClaudeBlocks(parts []PivotContent) []dto.ClaudeMediaMessage {
	out := make([]dto.ClaudeMediaMessage, 0, len(parts))
	for _, part := range parts {
		block := dto.ClaudeMediaMessage{Type: part.Type}
		switch part.Type {
		case dto.ContentTypeText:
			block.SetText(stringValue(part.Text))
		case "tool_use":
			if part.ToolCall != nil {
				if part.ToolCall.ID != nil {
					block.Id = *part.ToolCall.ID
				}
				if part.ToolCall.Name != nil {
					block.Name = *part.ToolCall.Name
				}
				if len(part.ToolCall.Arguments) > 0 {
					var input any
					if common.Unmarshal(part.ToolCall.Arguments, &input) == nil {
						block.Input = input
					}
				}
			}
		case "tool_result":
			if part.ToolResult != nil {
				if part.ToolResult.ToolCallID != nil {
					block.ToolUseId = *part.ToolResult.ToolCallID
				}
				block.Content = part.ToolResult.Content
			}
		case "thinking":
			if part.Thinking != nil {
				block.Thinking = part.Thinking.Text
				if part.Thinking.Signature != nil {
					block.Signature = *part.Thinking.Signature
				}
			}
		default:
			if part.Media != nil {
				block.Type = part.Media.Kind
				if part.Media.URL != nil {
					block.Source = &dto.ClaudeMessageSource{Type: "base64"}
					if part.Media.MimeType != nil {
						block.Source.MediaType = *part.Media.MimeType
					}
					block.Source.Data = *part.Media.URL
				}
			}
		}
		out = append(out, block)
	}
	return out
}

func convertClaudeUsageToPivot(usage *dto.ClaudeUsage) *PivotUsage {
	if usage == nil {
		return nil
	}
	input := usage.InputTokens
	output := usage.OutputTokens
	cacheRead := usage.CacheReadInputTokens
	cacheCreation := usage.GetCacheCreationTotalTokens()
	cache5m := usage.GetCacheCreation5mTokens()
	cache1h := usage.GetCacheCreation1hTokens()
	total := input + output
	return NormalizeUsage(&PivotUsage{
		InputTokens:                 &input,
		OutputTokens:                &output,
		PromptTokens:                &input,
		CompletionTokens:            &output,
		TotalTokens:                 &total,
		CacheReadInputTokens:        &cacheRead,
		CacheCreationInputTokens:    &cacheCreation,
		ClaudeCacheCreation5mTokens: &cache5m,
		ClaudeCacheCreation1hTokens: &cache1h,
	})
}

func convertPivotUsageToClaude(usage *PivotUsage) *dto.ClaudeUsage {
	n := NormalizeUsage(usage)
	if n == nil {
		return nil
	}
	out := &dto.ClaudeUsage{}
	out.InputTokens = intValue(n.InputTokens)
	out.OutputTokens = intValue(n.OutputTokens)
	out.CacheReadInputTokens = intValue(n.CacheReadInputTokens)
	out.CacheCreationInputTokens = intValue(n.CacheCreationInputTokens)
	cache5m := intValue(n.ClaudeCacheCreation5mTokens)
	cache1h := intValue(n.ClaudeCacheCreation1hTokens)
	if cache5m > 0 || cache1h > 0 {
		out.CacheCreation = &dto.ClaudeCacheCreationUsage{Ephemeral5mInputTokens: cache5m, Ephemeral1hInputTokens: cache1h}
	}
	return out
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
