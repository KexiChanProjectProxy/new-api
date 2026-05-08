package transformer

import (
	"strings"

	"github.com/QuantumNous/new-api/types"
)

type ConversionStep struct {
	From            types.RelayFormat `json:"from,omitempty"`
	To              types.RelayFormat `json:"to,omitempty"`
	TransformerName string            `json:"transformer_name,omitempty"`
}

type ConversionChain []ConversionStep

func (c *ConversionChain) Append(from, to types.RelayFormat, name string) {
	if c == nil {
		return
	}
	if from == "" || to == "" {
		return
	}
	*c = append(*c, ConversionStep{From: from, To: to, TransformerName: name})
}

func (c ConversionChain) LastFormat() types.RelayFormat {
	if len(c) == 0 {
		return ""
	}
	return c[len(c)-1].To
}

func (c ConversionChain) String() string {
	if len(c) == 0 {
		return ""
	}
	parts := make([]string, 0, len(c))
	for _, step := range c {
		part := string(step.From) + "->" + string(step.To)
		if step.TransformerName != "" {
			part += "(" + step.TransformerName + ")"
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, " | ")
}

func (c ConversionChain) RelayFormats() []types.RelayFormat {
	if len(c) == 0 {
		return nil
	}
	formats := make([]types.RelayFormat, 0, len(c)+1)
	formats = append(formats, c[0].From)
	for _, step := range c {
		formats = append(formats, step.To)
	}
	return formats
}

func ChainFromRelayFormats(formats []types.RelayFormat) ConversionChain {
	if len(formats) < 2 {
		return nil
	}
	chain := make(ConversionChain, 0, len(formats)-1)
	for i := 1; i < len(formats); i++ {
		chain = append(chain, ConversionStep{From: formats[i-1], To: formats[i]})
	}
	return chain
}
