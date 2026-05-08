package transformer

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
)

type StreamAccumulator struct {
	result *PivotResponse
	usage  *PivotUsage
	closed bool
}

func NewStreamAccumulator() *StreamAccumulator {
	return &StreamAccumulator{}
}

func (a *StreamAccumulator) Accumulate(chunk *PivotResponse) error {
	if a == nil {
		return fmt.Errorf("nil stream accumulator")
	}
	if a.closed {
		return fmt.Errorf("stream already finalized")
	}
	if chunk == nil {
		return fmt.Errorf("nil chunk")
	}
	if chunk.Error != nil {
		return fmt.Errorf("chunk contains error")
	}

	if a.result == nil {
		a.result = &PivotResponse{}
	}
	if a.result.ID == "" {
		a.result.ID = chunk.ID
	}
	if a.result.Object == "" {
		a.result.Object = chunk.Object
	}
	if a.result.Created == nil && chunk.Created != nil {
		a.result.Created = chunk.Created
	}
	if a.result.Model == "" {
		a.result.Model = chunk.Model
	}
	if a.result.SystemFingerprint == nil && chunk.SystemFingerprint != nil {
		a.result.SystemFingerprint = chunk.SystemFingerprint
	}
	if a.result.Status == nil && chunk.Status != nil {
		a.result.Status = chunk.Status
	}
	if a.result.IncompleteReason == nil && chunk.IncompleteReason != nil {
		a.result.IncompleteReason = chunk.IncompleteReason
	}
	if chunk.FinishReason != nil {
		a.result.FinishReason = chunk.FinishReason
	}

	for _, ch := range chunk.Choices {
		if err := a.mergeChoice(ch); err != nil {
			return err
		}
	}

	a.usage = MergeUsage(a.usage, chunk.Usage)
	return nil
}

func (a *StreamAccumulator) Finalize() (*PivotResponse, error) {
	if a == nil {
		return nil, fmt.Errorf("nil stream accumulator")
	}
	if a.result == nil {
		return nil, fmt.Errorf("no chunks accumulated")
	}
	a.closed = true
	a.result.Usage = NormalizeUsage(a.usage)
	if a.result.Usage == nil {
		a.result.Usage = &PivotUsage{}
	}
	return a.result, nil
}

func (a *StreamAccumulator) mergeChoice(in PivotChoice) error {
	idx := 0
	if in.Index != nil {
		idx = *in.Index
	}
	if idx < 0 {
		return fmt.Errorf("invalid negative choice index")
	}
	for len(a.result.Choices) <= idx {
		i := len(a.result.Choices)
		a.result.Choices = append(a.result.Choices, PivotChoice{Index: usageIntPtr(i)})
	}
	target := &a.result.Choices[idx]
	if target.Index == nil {
		target.Index = usageIntPtr(idx)
	}
	if in.FinishReason != nil {
		target.FinishReason = in.FinishReason
		a.result.FinishReason = in.FinishReason
	}

	if in.Message != nil {
		target.Message = mergePivotMessage(target.Message, in.Message)
	}
	if in.Delta != nil {
		target.Delta = mergePivotMessage(target.Delta, in.Delta)
		target.Message = mergePivotMessage(target.Message, in.Delta)
	}
	return nil
}

func mergePivotMessage(base *PivotMessage, delta *PivotMessage) *PivotMessage {
	if delta == nil {
		return base
	}
	if base == nil {
		cloned := *delta
		cloned.Parts = cloneParts(delta.Parts)
		cloned.ToolCalls = cloneToolCalls(delta.ToolCalls)
		return &cloned
	}

	out := *base
	if out.Role == "" && delta.Role != "" {
		out.Role = delta.Role
	}
	if out.Name == nil && delta.Name != nil {
		out.Name = delta.Name
	}
	if out.ToolCallID == nil && delta.ToolCallID != nil {
		out.ToolCallID = delta.ToolCallID
	}
	out.Parts = mergeParts(out.Parts, delta.Parts)
	out.ToolCalls = mergeToolCalls(out.ToolCalls, delta.ToolCalls)
	return &out
}

func cloneParts(parts []PivotContent) []PivotContent {
	if len(parts) == 0 {
		return nil
	}
	out := make([]PivotContent, len(parts))
	copy(out, parts)
	return out
}

func mergeParts(base []PivotContent, delta []PivotContent) []PivotContent {
	if len(delta) == 0 {
		return base
	}
	if len(base) == 0 {
		return cloneParts(delta)
	}
	out := cloneParts(base)
	for _, d := range delta {
		if d.Type == "text" && d.Text != nil && len(out) > 0 {
			last := &out[len(out)-1]
			if last.Type == "text" && last.Text != nil {
				merged := *last.Text + *d.Text
				last.Text = &merged
				continue
			}
		}
		out = append(out, d)
	}
	return out
}

func cloneToolCalls(calls []PivotToolCall) []PivotToolCall {
	if len(calls) == 0 {
		return nil
	}
	out := make([]PivotToolCall, len(calls))
	copy(out, calls)
	return out
}

func mergeToolCalls(base []PivotToolCall, delta []PivotToolCall) []PivotToolCall {
	if len(delta) == 0 {
		return base
	}
	out := cloneToolCalls(base)
	for _, d := range delta {
		idx := -1
		if d.Index != nil {
			idx = *d.Index
		}
		if idx >= 0 {
			for len(out) <= idx {
				i := len(out)
				out = append(out, PivotToolCall{Index: usageIntPtr(i)})
			}
			tc := &out[idx]
			mergeToolCallInto(tc, d)
			continue
		}

		mergedByID := false
		if d.ID != nil {
			for i := range out {
				if out[i].ID != nil && *out[i].ID == *d.ID {
					mergeToolCallInto(&out[i], d)
					mergedByID = true
					break
				}
			}
		}
		if !mergedByID {
			out = append(out, d)
		}
	}
	return out
}

func mergeToolCallInto(base *PivotToolCall, delta PivotToolCall) {
	if base.ID == nil && delta.ID != nil {
		base.ID = delta.ID
	}
	if base.Type == nil && delta.Type != nil {
		base.Type = delta.Type
	}
	if base.Name == nil && delta.Name != nil {
		base.Name = delta.Name
	}
	if base.Index == nil && delta.Index != nil {
		base.Index = delta.Index
	}
	if len(delta.Arguments) > 0 {
		if len(base.Arguments) == 0 {
			base.Arguments = delta.Arguments
		} else {
			joined := string(base.Arguments) + string(delta.Arguments)
			base.Arguments = []byte(joined)
		}
	}
	if delta.Input != nil {
		base.Input = delta.Input
	}
}

func (a *StreamAccumulator) MarshalFinal() ([]byte, error) {
	final, err := a.Finalize()
	if err != nil {
		return nil, err
	}
	return common.Marshal(final)
}
