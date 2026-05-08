package transformer

import (
	"testing"

	"github.com/QuantumNous/new-api/types"
)

type stubTransformer struct{}

func (stubTransformer) Inbound(raw []byte) (*PivotRequest, error) { return &PivotRequest{}, nil }
func (stubTransformer) Outbound(pivot *PivotRequest) ([]byte, error) { return []byte("{}"), nil }

type stubResponseTransformer struct{}

func (stubResponseTransformer) InboundResponse(raw []byte) (*PivotResponse, error) {
	return &PivotResponse{}, nil
}
func (stubResponseTransformer) OutboundResponse(pivot *PivotResponse) ([]byte, error) {
	return []byte("{}"), nil
}

type stubStreamTransformer struct{}

func (stubStreamTransformer) InboundStream(raw []byte) (*PivotResponse, error) {
	return &PivotResponse{}, nil
}
func (stubStreamTransformer) OutboundStream(pivot *PivotResponse) ([]byte, error) {
	return []byte("{}"), nil
}

func TestRegistryRegisterAndLookup(t *testing.T) {
	registry := NewRegistry()
	requestTransformer := stubTransformer{}
	responseTransformer := stubResponseTransformer{}
	streamTransformer := stubStreamTransformer{}

	registry.Register(types.RelayFormatOpenAI, requestTransformer, responseTransformer, streamTransformer)

	gotRequest, ok := registry.GetTransformer(types.RelayFormatOpenAI)
	if !ok || gotRequest == nil {
		t.Fatalf("expected request transformer to be registered")
	}
	if gotRequest != requestTransformer {
		t.Fatalf("unexpected request transformer: %#v", gotRequest)
	}

	gotResponse, ok := registry.GetResponseTransformer(types.RelayFormatOpenAI)
	if !ok || gotResponse == nil {
		t.Fatalf("expected response transformer to be registered")
	}
	if gotResponse != responseTransformer {
		t.Fatalf("unexpected response transformer: %#v", gotResponse)
	}

	gotStream, ok := registry.GetStreamTransformer(types.RelayFormatOpenAI)
	if !ok || gotStream == nil {
		t.Fatalf("expected stream transformer to be registered")
	}
	if gotStream != streamTransformer {
		t.Fatalf("unexpected stream transformer: %#v", gotStream)
	}
}

func TestRegistryLookupMissingFormatReturnsFalse(t *testing.T) {
	registry := NewRegistry()

	if transformer, ok := registry.GetTransformer(types.RelayFormatClaude); ok || transformer != nil {
		t.Fatalf("expected missing request transformer to return nil, false")
	}
	if transformer, ok := registry.GetResponseTransformer(types.RelayFormatClaude); ok || transformer != nil {
		t.Fatalf("expected missing response transformer to return nil, false")
	}
	if transformer, ok := registry.GetStreamTransformer(types.RelayFormatClaude); ok || transformer != nil {
		t.Fatalf("expected missing stream transformer to return nil, false")
	}
}

func TestRegistryDoubleRegistrationPanics(t *testing.T) {
	registry := NewRegistry()
	registry.Register(types.RelayFormatOpenAI, stubTransformer{}, stubResponseTransformer{}, stubStreamTransformer{})

	defer func() {
		if recover() == nil {
			t.Fatal("expected duplicate registration to panic")
		}
	}()

	registry.Register(types.RelayFormatOpenAI, stubTransformer{}, stubResponseTransformer{}, stubStreamTransformer{})
}

func TestConversionChainAppendLastFormatAndString(t *testing.T) {
	var chain ConversionChain
	chain.Append(types.RelayFormatOpenAI, types.RelayFormatClaude, "openai-to-claude")
	chain.Append(types.RelayFormatClaude, types.RelayFormatGemini, "claude-to-gemini")

	if got := chain.LastFormat(); got != types.RelayFormatGemini {
		t.Fatalf("unexpected last format: %q", got)
	}

	want := "openai->claude(openai-to-claude) | claude->gemini(claude-to-gemini)"
	if got := chain.String(); got != want {
		t.Fatalf("unexpected chain string: got %q want %q", got, want)
	}

	formats := chain.RelayFormats()
	if len(formats) != 3 {
		t.Fatalf("unexpected relay format count: %d", len(formats))
	}
	if formats[0] != types.RelayFormatOpenAI || formats[1] != types.RelayFormatClaude || formats[2] != types.RelayFormatGemini {
		t.Fatalf("unexpected relay formats: %#v", formats)
	}
}

func TestConversionChainEmptyBehaviors(t *testing.T) {
	var chain ConversionChain

	if got := chain.LastFormat(); got != "" {
		t.Fatalf("expected empty last format, got %q", got)
	}
	if got := chain.String(); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
	if formats := chain.RelayFormats(); formats != nil {
		t.Fatalf("expected nil relay formats, got %#v", formats)
	}
}

func TestChainFromRelayFormats(t *testing.T) {
	chain := ChainFromRelayFormats([]types.RelayFormat{
		types.RelayFormatOpenAI,
		types.RelayFormatOpenAIResponses,
		types.RelayFormatClaude,
	})

	if len(chain) != 2 {
		t.Fatalf("unexpected chain length: %d", len(chain))
	}
	if chain[0].From != types.RelayFormatOpenAI || chain[0].To != types.RelayFormatOpenAIResponses {
		t.Fatalf("unexpected first step: %#v", chain[0])
	}
	if chain[1].From != types.RelayFormatOpenAIResponses || chain[1].To != types.RelayFormatClaude {
		t.Fatalf("unexpected second step: %#v", chain[1])
	}
}
