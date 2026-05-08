package transformer

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
)

func ConvertRequestBetweenFormats(raw []byte, sourceFormat, targetFormat types.RelayFormat) ([]byte, error) {
	inbound, sourceOK := GetTransformer(sourceFormat)
	outbound, targetOK := GetTransformer(targetFormat)
	if !sourceOK || !targetOK {
		return nil, fmt.Errorf("request transformer not found for source=%q target=%q", sourceFormat, targetFormat)
	}
	pivot, err := inbound.Inbound(raw)
	if err != nil {
		return nil, err
	}
	if pivot == nil {
		return nil, fmt.Errorf("request inbound transformer returned nil pivot")
	}
	return outbound.Outbound(pivot)
}

func ConvertOpenAIResponseToFormat(openAIResponse *dto.OpenAITextResponse, targetFormat types.RelayFormat) ([]byte, error) {
	if openAIResponse == nil {
		return nil, fmt.Errorf("openai response is nil")
	}
	inbound, sourceOK := GetResponseTransformer(types.RelayFormatOpenAI)
	outbound, targetOK := GetResponseTransformer(targetFormat)
	if !sourceOK || !targetOK {
		return nil, fmt.Errorf("response transformer not found for target=%q", targetFormat)
	}
	raw, err := common.Marshal(openAIResponse)
	if err != nil {
		return nil, err
	}
	pivot, err := inbound.InboundResponse(raw)
	if err != nil {
		return nil, err
	}
	if pivot == nil {
		return nil, fmt.Errorf("response inbound transformer returned nil pivot")
	}
	return outbound.OutboundResponse(pivot)
}

func ConvertOpenAIStreamToFormat(openAIResponse *dto.ChatCompletionsStreamResponse, targetFormat types.RelayFormat) ([]byte, error) {
	if openAIResponse == nil {
		return nil, fmt.Errorf("openai stream response is nil")
	}
	inbound, sourceOK := GetStreamTransformer(types.RelayFormatOpenAI)
	outbound, targetOK := GetStreamTransformer(targetFormat)
	if !sourceOK || !targetOK {
		return nil, fmt.Errorf("stream transformer not found for target=%q", targetFormat)
	}
	raw, err := common.Marshal(openAIResponse)
	if err != nil {
		return nil, err
	}
	pivot, err := inbound.InboundStream(raw)
	if err != nil {
		return nil, err
	}
	if pivot == nil {
		return nil, fmt.Errorf("stream inbound transformer returned nil pivot")
	}
	return outbound.OutboundStream(pivot)
}
