package transformer

import (
	"encoding/json"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
)

// EmbeddingTransformer uses PivotRequest as a transport envelope, storing
// format-specific fields in ProviderExtensions rather than forcing them
// into chat-oriented pivot fields.
type EmbeddingTransformer struct{}

type RerankTransformer struct{}

type ImageTransformer struct{}

type AudioTransformer struct{}

type NonChatNoopResponseTransformer struct{}

type NonChatNoopStreamTransformer struct{}

func init() {
	embReq := EmbeddingTransformer{}
	rerankReq := RerankTransformer{}
	imageReq := ImageTransformer{}
	audioReq := AudioTransformer{}
	noopResp := NonChatNoopResponseTransformer{}
	noopStream := NonChatNoopStreamTransformer{}

	Register(types.RelayFormatEmbedding, embReq, noopResp, noopStream)
	Register(types.RelayFormatRerank, rerankReq, noopResp, noopStream)
	Register(types.RelayFormatOpenAIImage, imageReq, noopResp, noopStream)
	Register(types.RelayFormatOpenAIAudio, audioReq, noopResp, noopStream)
}

func (t EmbeddingTransformer) Inbound(raw []byte) (*PivotRequest, error) {
	var req dto.EmbeddingRequest
	if err := common.Unmarshal(raw, &req); err != nil {
		return nil, fmt.Errorf("embedding transformer: inbound parse: %w", err)
	}

	p := &PivotRequest{
		RelayFormat: types.RelayFormatEmbedding,
		Model:       req.Model,
		Input:       req.Input,
		ProviderExtensions: map[string]any{
			"embedding_encoding_format":   req.EncodingFormat,
			"embedding_dimensions":        req.Dimensions,
			"embedding_user":              req.User,
			"embedding_seed":              req.Seed,
			"embedding_temperature":       req.Temperature,
			"embedding_top_p":             req.TopP,
			"embedding_frequency_penalty": req.FrequencyPenalty,
			"embedding_presence_penalty":  req.PresencePenalty,
		},
	}

	return p, nil
}

func (t EmbeddingTransformer) Outbound(pivot *PivotRequest) ([]byte, error) {
	if pivot == nil {
		return nil, fmt.Errorf("embedding transformer: nil pivot request")
	}
	if pivot.RelayFormat != types.RelayFormatEmbedding {
		return nil, fmt.Errorf("embedding transformer: unexpected relay format %q", pivot.RelayFormat)
	}

	req := dto.EmbeddingRequest{
		Model: pivot.Model,
		Input: pivot.Input,
	}

	if ext := pivot.ProviderExtensions; ext != nil {
		if v, ok := ext["embedding_encoding_format"].(string); ok {
			req.EncodingFormat = v
		}
		if v, ok := ext["embedding_dimensions"].(*int); ok {
			req.Dimensions = v
		}
		if v, ok := ext["embedding_user"].(string); ok {
			req.User = v
		}
		if v, ok := ext["embedding_seed"].(*float64); ok {
			req.Seed = v
		}
		if v, ok := ext["embedding_temperature"].(*float64); ok {
			req.Temperature = v
		}
		if v, ok := ext["embedding_top_p"].(*float64); ok {
			req.TopP = v
		}
		if v, ok := ext["embedding_frequency_penalty"].(*float64); ok {
			req.FrequencyPenalty = v
		}
		if v, ok := ext["embedding_presence_penalty"].(*float64); ok {
			req.PresencePenalty = v
		}
	}

	return common.Marshal(req)
}

func (t RerankTransformer) Inbound(raw []byte) (*PivotRequest, error) {
	var req dto.RerankRequest
	if err := common.Unmarshal(raw, &req); err != nil {
		return nil, fmt.Errorf("rerank transformer: inbound parse: %w", err)
	}

	p := &PivotRequest{
		RelayFormat: types.RelayFormatRerank,
		Model:       req.Model,
		ProviderExtensions: map[string]any{
			"rerank_documents":         req.Documents,
			"rerank_query":             req.Query,
			"rerank_top_n":             req.TopN,
			"rerank_return_documents":  req.ReturnDocuments,
			"rerank_max_chunk_per_doc": req.MaxChunkPerDoc,
			"rerank_overlap_tokens":    req.OverLapTokens,
		},
	}

	return p, nil
}

func (t RerankTransformer) Outbound(pivot *PivotRequest) ([]byte, error) {
	if pivot == nil {
		return nil, fmt.Errorf("rerank transformer: nil pivot request")
	}
	if pivot.RelayFormat != types.RelayFormatRerank {
		return nil, fmt.Errorf("rerank transformer: unexpected relay format %q", pivot.RelayFormat)
	}

	req := dto.RerankRequest{
		Model: pivot.Model,
	}

	if ext := pivot.ProviderExtensions; ext != nil {
		if v, ok := ext["rerank_documents"]; ok {
			if arr, ok := v.([]any); ok {
				req.Documents = arr
			}
		}
		if v, ok := ext["rerank_query"].(string); ok {
			req.Query = v
		}
		if v, ok := ext["rerank_top_n"].(*int); ok {
			req.TopN = v
		}
		if v, ok := ext["rerank_return_documents"].(*bool); ok {
			req.ReturnDocuments = v
		}
		if v, ok := ext["rerank_max_chunk_per_doc"].(*int); ok {
			req.MaxChunkPerDoc = v
		}
		if v, ok := ext["rerank_overlap_tokens"].(*int); ok {
			req.OverLapTokens = v
		}
	}

	return common.Marshal(req)
}

func (t ImageTransformer) Inbound(raw []byte) (*PivotRequest, error) {
	var req dto.ImageRequest
	if err := common.Unmarshal(raw, &req); err != nil {
		return nil, fmt.Errorf("image transformer: inbound parse: %w", err)
	}

	p := &PivotRequest{
		RelayFormat: types.RelayFormatOpenAIImage,
		Model:       req.Model,
		ProviderExtensions: map[string]any{
			"image_prompt":             req.Prompt,
			"image_n":                  req.N,
			"image_size":               req.Size,
			"image_quality":            req.Quality,
			"image_response_format":    req.ResponseFormat,
			"image_style":              req.Style,
			"image_user":               req.User,
			"image_extra_fields":       req.ExtraFields,
			"image_background":         req.Background,
			"image_moderation":         req.Moderation,
			"image_output_format":      req.OutputFormat,
			"image_output_compression": req.OutputCompression,
			"image_partial_images":     req.PartialImages,
			"image_watermark":          req.Watermark,
			"image_watermark_enabled":  req.WatermarkEnabled,
			"image_user_id":            req.UserId,
			"image_image":              req.Image,
			"image_extra":              req.Extra,
		},
	}

	return p, nil
}

func (t ImageTransformer) Outbound(pivot *PivotRequest) ([]byte, error) {
	if pivot == nil {
		return nil, fmt.Errorf("image transformer: nil pivot request")
	}
	if pivot.RelayFormat != types.RelayFormatOpenAIImage {
		return nil, fmt.Errorf("image transformer: unexpected relay format %q", pivot.RelayFormat)
	}

	req := dto.ImageRequest{
		Model: pivot.Model,
	}

	if ext := pivot.ProviderExtensions; ext != nil {
		if v, ok := ext["image_prompt"].(string); ok {
			req.Prompt = v
		}
		if v, ok := ext["image_n"].(*uint); ok {
			req.N = v
		}
		if v, ok := ext["image_size"].(string); ok {
			req.Size = v
		}
		if v, ok := ext["image_quality"].(string); ok {
			req.Quality = v
		}
		if v, ok := ext["image_response_format"].(string); ok {
			req.ResponseFormat = v
		}
		if v, ok := ext["image_style"]; ok {
			if b, err := common.Marshal(v); err == nil {
				req.Style = b
			}
		}
		if v, ok := ext["image_user"]; ok {
			if b, err := common.Marshal(v); err == nil {
				req.User = b
			}
		}
		if v, ok := ext["image_extra_fields"]; ok {
			if b, err := common.Marshal(v); err == nil {
				req.ExtraFields = b
			}
		}
		if v, ok := ext["image_background"]; ok {
			if b, err := common.Marshal(v); err == nil {
				req.Background = b
			}
		}
		if v, ok := ext["image_moderation"]; ok {
			if b, err := common.Marshal(v); err == nil {
				req.Moderation = b
			}
		}
		if v, ok := ext["image_output_format"]; ok {
			if b, err := common.Marshal(v); err == nil {
				req.OutputFormat = b
			}
		}
		if v, ok := ext["image_output_compression"]; ok {
			if b, err := common.Marshal(v); err == nil {
				req.OutputCompression = b
			}
		}
		if v, ok := ext["image_partial_images"]; ok {
			if b, err := common.Marshal(v); err == nil {
				req.PartialImages = b
			}
		}
		if v, ok := ext["image_watermark"].(*bool); ok {
			req.Watermark = v
		}
		if v, ok := ext["image_watermark_enabled"]; ok {
			if b, err := common.Marshal(v); err == nil {
				req.WatermarkEnabled = b
			}
		}
		if v, ok := ext["image_user_id"]; ok {
			if b, err := common.Marshal(v); err == nil {
				req.UserId = b
			}
		}
		if v, ok := ext["image_image"]; ok {
			if b, err := common.Marshal(v); err == nil {
				req.Image = b
			}
		}
		if v, ok := ext["image_extra"].(map[string]any); ok {
			req.Extra = make(map[string]json.RawMessage, len(v))
			for k, val := range v {
				if b, err := common.Marshal(val); err == nil {
					req.Extra[k] = b
				}
			}
		}
	}

	return common.Marshal(req)
}

func (t AudioTransformer) Inbound(raw []byte) (*PivotRequest, error) {
	var req dto.AudioRequest
	if err := common.Unmarshal(raw, &req); err != nil {
		return nil, fmt.Errorf("audio transformer: inbound parse: %w", err)
	}

	p := &PivotRequest{
		RelayFormat: types.RelayFormatOpenAIAudio,
		Model:       req.Model,
		ProviderExtensions: map[string]any{
			"audio_input":                      req.Input,
			"audio_voice":                      req.Voice,
			"audio_instructions":               req.Instructions,
			"audio_response_format":            req.ResponseFormat,
			"audio_speed":                      req.Speed,
			"audio_stream_format":              req.StreamFormat,
			"audio_metadata":                   req.Metadata,
			"audio_task_type":                  req.TaskType,
			"audio_language":                   req.Language,
			"audio_ref_audio":                  req.RefAudio,
			"audio_ref_text":                   req.RefText,
			"audio_x_vector_only_mode":         req.XVectorOnlyMode,
			"audio_max_new_tokens":             req.MaxNewTokens,
			"audio_initial_codec_chunk_frames": req.InitialCodecChunkFrames,
		},
	}

	return p, nil
}

func (t AudioTransformer) Outbound(pivot *PivotRequest) ([]byte, error) {
	if pivot == nil {
		return nil, fmt.Errorf("audio transformer: nil pivot request")
	}
	if pivot.RelayFormat != types.RelayFormatOpenAIAudio {
		return nil, fmt.Errorf("audio transformer: unexpected relay format %q", pivot.RelayFormat)
	}

	req := dto.AudioRequest{
		Model: pivot.Model,
	}

	if ext := pivot.ProviderExtensions; ext != nil {
		if v, ok := ext["audio_input"].(string); ok {
			req.Input = v
		}
		if v, ok := ext["audio_voice"].(string); ok {
			req.Voice = v
		}
		if v, ok := ext["audio_instructions"].(string); ok {
			req.Instructions = v
		}
		if v, ok := ext["audio_response_format"].(string); ok {
			req.ResponseFormat = v
		}
		if v, ok := ext["audio_speed"].(*float64); ok {
			req.Speed = v
		}
		if v, ok := ext["audio_stream_format"].(string); ok {
			req.StreamFormat = v
		}
		if v, ok := ext["audio_metadata"]; ok {
			if b, err := common.Marshal(v); err == nil {
				req.Metadata = b
			}
		}
		if v, ok := ext["audio_task_type"]; ok {
			if b, err := common.Marshal(v); err == nil {
				req.TaskType = b
			}
		}
		if v, ok := ext["audio_language"]; ok {
			if b, err := common.Marshal(v); err == nil {
				req.Language = b
			}
		}
		if v, ok := ext["audio_ref_audio"]; ok {
			if b, err := common.Marshal(v); err == nil {
				req.RefAudio = b
			}
		}
		if v, ok := ext["audio_ref_text"]; ok {
			if b, err := common.Marshal(v); err == nil {
				req.RefText = b
			}
		}
		if v, ok := ext["audio_x_vector_only_mode"]; ok {
			if b, err := common.Marshal(v); err == nil {
				req.XVectorOnlyMode = b
			}
		}
		if v, ok := ext["audio_max_new_tokens"]; ok {
			if b, err := common.Marshal(v); err == nil {
				req.MaxNewTokens = b
			}
		}
		if v, ok := ext["audio_initial_codec_chunk_frames"]; ok {
			if b, err := common.Marshal(v); err == nil {
				req.InitialCodecChunkFrames = b
			}
		}
	}

	return common.Marshal(req)
}

func (t NonChatNoopResponseTransformer) InboundResponse(raw []byte) (*PivotResponse, error) {
	return nil, fmt.Errorf("non-chat formats do not support chat response pivot transformation")
}

func (t NonChatNoopResponseTransformer) OutboundResponse(pivot *PivotResponse) ([]byte, error) {
	return nil, fmt.Errorf("non-chat formats do not support chat response pivot transformation")
}

func (t NonChatNoopStreamTransformer) InboundStream(raw []byte) (*PivotResponse, error) {
	return nil, fmt.Errorf("non-chat formats do not support chat stream pivot transformation")
}

func (t NonChatNoopStreamTransformer) OutboundStream(pivot *PivotResponse) ([]byte, error) {
	return nil, fmt.Errorf("non-chat formats do not support chat stream pivot transformation")
}
