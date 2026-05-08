package transformer

import (
	"errors"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
)

type orchestratorStubTransformer struct {
	inboundFn  func(raw []byte) (*PivotRequest, error)
	outboundFn func(pivot *PivotRequest) ([]byte, error)
}

func (s orchestratorStubTransformer) Inbound(raw []byte) (*PivotRequest, error) {
	if s.inboundFn != nil {
		return s.inboundFn(raw)
	}
	return &PivotRequest{}, nil
}

func (s orchestratorStubTransformer) Outbound(pivot *PivotRequest) ([]byte, error) {
	if s.outboundFn != nil {
		return s.outboundFn(pivot)
	}
	return []byte(`{"ok":true}`), nil
}

func TestNewOrchestratorUsesGlobalRegistryWhenNil(t *testing.T) {
	orchestrator := NewOrchestrator(nil, nil)
	if orchestrator == nil {
		t.Fatalf("expected orchestrator")
	}
	if orchestrator.registry != GlobalRegistry() {
		t.Fatalf("expected global registry fallback")
	}
}

func TestTransformRequestSuccess(t *testing.T) {
	registry := NewRegistry()
	registry.Register(types.RelayFormatOpenAI, orchestratorStubTransformer{
		inboundFn: func(raw []byte) (*PivotRequest, error) {
			if string(raw) != `{"source":"openai"}` {
				t.Fatalf("unexpected raw: %s", string(raw))
			}
			return &PivotRequest{Model: "m1"}, nil
		},
	}, nil, nil)
	registry.Register(types.RelayFormatClaude, orchestratorStubTransformer{
		outboundFn: func(pivot *PivotRequest) ([]byte, error) {
			if pivot == nil || pivot.Model != "m1" {
				t.Fatalf("unexpected pivot: %#v", pivot)
			}
			return []byte(`{"target":"claude"}`), nil
		},
	}, nil, nil)

	info := &relaycommon.RelayInfo{RelayFormat: types.RelayFormatOpenAI}
	orchestrator := NewOrchestrator(registry, info)
	out, chain, err := orchestrator.TransformRequest([]byte(`{"source":"openai"}`), types.RelayFormatOpenAI, types.RelayFormatClaude)
	if err != nil {
		t.Fatalf("transform request: %v", err)
	}
	if string(out) != `{"target":"claude"}` {
		t.Fatalf("unexpected output: %s", string(out))
	}
	if chain == nil || len(*chain) != 2 {
		t.Fatalf("expected 2-step chain, got: %#v", chain)
	}
	if (*chain)[0].From != types.RelayFormatOpenAI || (*chain)[0].To != relayFormatPivot {
		t.Fatalf("unexpected first chain step: %#v", (*chain)[0])
	}
	if (*chain)[1].From != relayFormatPivot || (*chain)[1].To != types.RelayFormatClaude {
		t.Fatalf("unexpected second chain step: %#v", (*chain)[1])
	}
	if info.FinalRequestRelayFormat != types.RelayFormatClaude {
		t.Fatalf("expected final request format to be claude, got %q", info.FinalRequestRelayFormat)
	}
	if got := info.GetFinalRequestRelayFormat(); got != types.RelayFormatClaude {
		t.Fatalf("unexpected final request format getter: %q", got)
	}
}

func TestTransformRequestMissingTransformer(t *testing.T) {
	orchestrator := NewOrchestrator(NewRegistry(), nil)
	_, _, err := orchestrator.TransformRequest([]byte(`{}`), types.RelayFormatOpenAI, types.RelayFormatClaude)
	if err == nil {
		t.Fatalf("expected error when transformers are missing")
	}
}

func TestTransformRequestInboundError(t *testing.T) {
	registry := NewRegistry()
	registry.Register(types.RelayFormatOpenAI, orchestratorStubTransformer{
		inboundFn: func(raw []byte) (*PivotRequest, error) {
			return nil, errors.New("inbound failed")
		},
	}, nil, nil)
	registry.Register(types.RelayFormatClaude, orchestratorStubTransformer{}, nil, nil)

	orchestrator := NewOrchestrator(registry, nil)
	_, _, err := orchestrator.TransformRequest([]byte(`{}`), types.RelayFormatOpenAI, types.RelayFormatClaude)
	if err == nil {
		t.Fatalf("expected inbound error")
	}
}
