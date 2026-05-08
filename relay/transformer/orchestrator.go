package transformer

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
)

const relayFormatPivot types.RelayFormat = "pivot"

type Orchestrator struct {
	registry *Registry
	info     *relaycommon.RelayInfo
}

func NewOrchestrator(registry *Registry, info *relaycommon.RelayInfo) *Orchestrator {
	if registry == nil {
		registry = GlobalRegistry()
	}
	return &Orchestrator{registry: registry, info: info}
}

// TransformRequest performs: raw request → inbound transformer → PivotRequest → cross-cutting → outbound transformer → target bytes
func (o *Orchestrator) TransformRequest(rawBody []byte, sourceFormat, targetFormat types.RelayFormat) ([]byte, *ConversionChain, error) {
	if o == nil || o.registry == nil {
		return nil, nil, fmt.Errorf("transformer orchestrator is not initialized")
	}
	if sourceFormat == "" || targetFormat == "" {
		return nil, nil, fmt.Errorf("source and target formats are required")
	}

	inbound, ok := o.registry.GetTransformer(sourceFormat)
	if !ok {
		return nil, nil, fmt.Errorf("inbound transformer not found for format %q", sourceFormat)
	}
	outbound, ok := o.registry.GetTransformer(targetFormat)
	if !ok {
		return nil, nil, fmt.Errorf("outbound transformer not found for format %q", targetFormat)
	}

	pivot, err := inbound.Inbound(rawBody)
	if err != nil {
		return nil, nil, fmt.Errorf("inbound transform failed: %w", err)
	}
	if pivot == nil {
		return nil, nil, fmt.Errorf("inbound transform returned nil pivot request")
	}

	chain := make(ConversionChain, 0, 2)
	chain.Append(sourceFormat, relayFormatPivot, fmt.Sprintf("%T", inbound))

	if err = applyPivotCrossCutting(o.info, pivot); err != nil {
		return nil, nil, fmt.Errorf("apply pivot cross-cutting failed: %w", err)
	}

	out, err := outbound.Outbound(pivot)
	if err != nil {
		return nil, nil, fmt.Errorf("outbound transform failed: %w", err)
	}
	chain.Append(relayFormatPivot, targetFormat, fmt.Sprintf("%T", outbound))

	if o.info != nil {
		o.info.AppendRequestConversion(targetFormat)
		o.info.FinalRequestRelayFormat = targetFormat
	}

	return out, &chain, nil
}

func applyPivotCrossCutting(info *relaycommon.RelayInfo, pivot *PivotRequest) error {
	if pivot == nil {
		return fmt.Errorf("nil pivot request")
	}

	if info != nil && info.ChannelMeta != nil && info.ChannelSetting.SystemPrompt != "" {
		applyPivotSystemPrompt(info, pivot)
	}

	jsonData, err := common.Marshal(pivot)
	if err != nil {
		return err
	}

	if info != nil && info.ChannelMeta != nil {
		passThroughBodyEnabled := false
		passThroughBodyEnabled = info.ChannelSetting.PassThroughBodyEnabled
		jsonData, err = relaycommon.RemoveDisabledFields(jsonData, info.ChannelOtherSettings, passThroughBodyEnabled)
		if err != nil {
			return err
		}
		if len(info.ParamOverride) > 0 {
			jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
			if err != nil {
				return err
			}
		}
	}

	if err = common.Unmarshal(jsonData, pivot); err != nil {
		return err
	}
	return nil
}

func applyPivotSystemPrompt(info *relaycommon.RelayInfo, pivot *PivotRequest) {
	if info == nil || pivot == nil || info.ChannelSetting.SystemPrompt == "" {
		return
	}

	systemText := info.ChannelSetting.SystemPrompt
	hasSystem := false
	for idx := range pivot.Messages {
		if pivot.Messages[idx].Role != "system" {
			continue
		}
		hasSystem = true
		if !info.ChannelSetting.SystemPromptOverride {
			break
		}
		if len(pivot.Messages[idx].Parts) == 0 {
			pivot.Messages[idx].Parts = []PivotContent{{Type: "text", Text: &systemText}}
			break
		}
		for j := range pivot.Messages[idx].Parts {
			part := &pivot.Messages[idx].Parts[j]
			if part.Type == "text" && part.Text != nil {
				combined := systemText + "\n" + *part.Text
				part.Text = &combined
				break
			}
		}
		break
	}

	if !hasSystem {
		pivot.Messages = append([]PivotMessage{{
			Role:  "system",
			Parts: []PivotContent{{Type: "text", Text: &systemText}},
		}}, pivot.Messages...)
	}
}
