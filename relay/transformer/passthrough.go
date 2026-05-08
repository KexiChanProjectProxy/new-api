package transformer

import (
	"fmt"

	"github.com/QuantumNous/new-api/types"
)

// PassthroughTransformer handles relay formats that do not participate in
// cross-format conversion (e.g., realtime WebSocket, MCP, task, mj_proxy).
// It stores raw bytes in ProviderExtensions and round-trips them unchanged.
type PassthroughTransformer struct {
	Format types.RelayFormat
}

func init() {
	noopResp := PassthroughNoopResponseTransformer{}
	noopStream := PassthroughNoopStreamTransformer{}

	Register(types.RelayFormatOpenAIRealtime, PassthroughTransformer{Format: types.RelayFormatOpenAIRealtime}, noopResp, noopStream)
	Register(types.RelayFormatMCP, PassthroughTransformer{Format: types.RelayFormatMCP}, noopResp, noopStream)
	Register(types.RelayFormatTask, PassthroughTransformer{Format: types.RelayFormatTask}, noopResp, noopStream)
	Register(types.RelayFormatMjProxy, PassthroughTransformer{Format: types.RelayFormatMjProxy}, noopResp, noopStream)
}

func (t PassthroughTransformer) Inbound(raw []byte) (*PivotRequest, error) {
	if raw == nil {
		return nil, fmt.Errorf("passthrough transformer: nil input")
	}
	return &PivotRequest{
		RelayFormat: t.Format,
		ProviderExtensions: map[string]any{
			"passthrough_raw": raw,
		},
	}, nil
}

func (t PassthroughTransformer) Outbound(pivot *PivotRequest) ([]byte, error) {
	if pivot == nil {
		return nil, fmt.Errorf("passthrough transformer: nil pivot")
	}
	raw, ok := pivot.ProviderExtensions["passthrough_raw"]
	if !ok {
		return nil, fmt.Errorf("passthrough transformer: missing passthrough_raw in ProviderExtensions")
	}
	b, ok := raw.([]byte)
	if !ok {
		return nil, fmt.Errorf("passthrough transformer: passthrough_raw is not []byte")
	}
	return b, nil
}

// PassthroughNoopResponseTransformer and PassthroughNoopStreamTransformer
// are no-op response/stream transformers for pass-through formats that
// don't participate in cross-format response conversion.
type PassthroughNoopResponseTransformer struct{}

func (t PassthroughNoopResponseTransformer) InboundResponse(raw []byte) (*PivotResponse, error) {
	return nil, nil
}

func (t PassthroughNoopResponseTransformer) OutboundResponse(pivot *PivotResponse) ([]byte, error) {
	return nil, nil
}

type PassthroughNoopStreamTransformer struct{}

func (t PassthroughNoopStreamTransformer) InboundStream(raw []byte) (*PivotResponse, error) {
	return nil, nil
}

func (t PassthroughNoopStreamTransformer) OutboundStream(pivot *PivotResponse) ([]byte, error) {
	return nil, nil
}
