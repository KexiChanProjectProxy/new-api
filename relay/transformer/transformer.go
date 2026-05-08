package transformer

import (
	"fmt"
	"sync"

	"github.com/QuantumNous/new-api/types"
)

type Transformer interface {
	Inbound(raw []byte) (*PivotRequest, error)
	Outbound(pivot *PivotRequest) ([]byte, error)
}

type ResponseTransformer interface {
	InboundResponse(raw []byte) (*PivotResponse, error)
	OutboundResponse(pivot *PivotResponse) ([]byte, error)
}

type StreamTransformer interface {
	InboundStream(raw []byte) (*PivotResponse, error)
	OutboundStream(pivot *PivotResponse) ([]byte, error)
}

type Registry struct {
	mu                   sync.RWMutex
	transformers         map[types.RelayFormat]Transformer
	responseTransformers map[types.RelayFormat]ResponseTransformer
	streamTransformers   map[types.RelayFormat]StreamTransformer
}

func NewRegistry() *Registry {
	return &Registry{
		transformers:         make(map[types.RelayFormat]Transformer),
		responseTransformers: make(map[types.RelayFormat]ResponseTransformer),
		streamTransformers:   make(map[types.RelayFormat]StreamTransformer),
	}
}

func (r *Registry) Register(format types.RelayFormat, transformer Transformer, responseTransformer ResponseTransformer, streamTransformer StreamTransformer) {
	if r == nil {
		panic("transformer: register on nil registry")
	}
	if format == "" {
		panic("transformer: register empty relay format")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.transformers[format]; exists {
		panic(fmt.Sprintf("transformer: duplicate request transformer registration for format %q", format))
	}
	if _, exists := r.responseTransformers[format]; exists {
		panic(fmt.Sprintf("transformer: duplicate response transformer registration for format %q", format))
	}
	if _, exists := r.streamTransformers[format]; exists {
		panic(fmt.Sprintf("transformer: duplicate stream transformer registration for format %q", format))
	}

	r.transformers[format] = transformer
	r.responseTransformers[format] = responseTransformer
	r.streamTransformers[format] = streamTransformer
}

func (r *Registry) GetTransformer(format types.RelayFormat) (Transformer, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	transformer, ok := r.transformers[format]
	if !ok || transformer == nil {
		return nil, false
	}
	return transformer, true
}

func (r *Registry) GetResponseTransformer(format types.RelayFormat) (ResponseTransformer, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	transformer, ok := r.responseTransformers[format]
	if !ok || transformer == nil {
		return nil, false
	}
	return transformer, true
}

func (r *Registry) GetStreamTransformer(format types.RelayFormat) (StreamTransformer, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	transformer, ok := r.streamTransformers[format]
	if !ok || transformer == nil {
		return nil, false
	}
	return transformer, true
}

var defaultRegistry = NewRegistry()

func GlobalRegistry() *Registry {
	return defaultRegistry
}

func Register(format types.RelayFormat, transformer Transformer, responseTransformer ResponseTransformer, streamTransformer StreamTransformer) {
	defaultRegistry.Register(format, transformer, responseTransformer, streamTransformer)
}

func GetTransformer(format types.RelayFormat) (Transformer, bool) {
	return defaultRegistry.GetTransformer(format)
}

func GetResponseTransformer(format types.RelayFormat) (ResponseTransformer, bool) {
	return defaultRegistry.GetResponseTransformer(format)
}

func GetStreamTransformer(format types.RelayFormat) (StreamTransformer, bool) {
	return defaultRegistry.GetStreamTransformer(format)
}
