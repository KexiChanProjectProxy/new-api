package transformer

import (
	"encoding/json"

	"github.com/QuantumNous/new-api/types"
)

type PivotRequest struct {
	RelayFormat         types.RelayFormat    `json:"relay_format,omitempty"`
	Model               string               `json:"model,omitempty"`
	Messages            []PivotMessage       `json:"messages,omitempty"`
	System              *PivotSystemPrompt   `json:"system,omitempty"`
	Prompt              any                  `json:"prompt,omitempty"`
	Input               any                  `json:"input,omitempty"`
	Instruction         *string              `json:"instruction,omitempty"`
	Prefix              any                  `json:"prefix,omitempty"`
	Suffix              any                  `json:"suffix,omitempty"`
	Stream              *bool                `json:"stream,omitempty"`
	StreamOptions       *PivotStreamOptions  `json:"stream_options,omitempty"`
	Temperature         *float64             `json:"temperature,omitempty"`
	TopP                *float64             `json:"top_p,omitempty"`
	TopK                *int                 `json:"top_k,omitempty"`
	MaxTokens           *uint                `json:"max_tokens,omitempty"`
	MaxCompletionTokens *uint                `json:"max_completion_tokens,omitempty"`
	N                   *int                 `json:"n,omitempty"`
	StopSequences       []string             `json:"stop_sequences,omitempty"`
	FrequencyPenalty    *float64             `json:"frequency_penalty,omitempty"`
	PresencePenalty     *float64             `json:"presence_penalty,omitempty"`
	Seed                *float64             `json:"seed,omitempty"`
	LogProbs            *bool                `json:"logprobs,omitempty"`
	TopLogProbs         *int                 `json:"top_logprobs,omitempty"`
	Tools               []PivotTool          `json:"tools,omitempty"`
	Functions           []PivotTool          `json:"functions,omitempty"`
	ToolChoice          *PivotToolChoice     `json:"tool_choice,omitempty"`
	ResponseFormat      *PivotResponseFormat `json:"response_format,omitempty"`
	Thinking            *PivotThinkingConfig `json:"thinking,omitempty"`
	ReasoningEffort     *string              `json:"reasoning_effort,omitempty"`
	ServiceTier         *string              `json:"service_tier,omitempty"`
	ParallelToolCalls   *bool                `json:"parallel_tool_calls,omitempty"`
	Modalities          []string             `json:"modalities,omitempty"`
	Audio               *PivotAudioConfig    `json:"audio,omitempty"`
	Prediction          *PivotPrediction     `json:"prediction,omitempty"`
	ToolConfig          *PivotToolConfig     `json:"tool_config,omitempty"`
	SafetySettings      []PivotSafetySetting `json:"safety_settings,omitempty"`
	CachedContent       *string              `json:"cached_content,omitempty"`
	Metadata            map[string]any       `json:"metadata,omitempty"`
	ProviderExtensions  map[string]any       `json:"provider_extensions,omitempty"`
	ExtraFields         map[string]any       `json:"extra_fields,omitempty"`
}

type PivotResponse struct {
	ID                 string            `json:"id,omitempty"`
	Object             string            `json:"object,omitempty"`
	Created            *int64            `json:"created,omitempty"`
	Model              string            `json:"model,omitempty"`
	Choices            []PivotChoice     `json:"choices,omitempty"`
	Content            []PivotContent    `json:"content,omitempty"`
	Usage              *PivotUsage       `json:"usage,omitempty"`
	StreamState        *PivotStreamState `json:"stream_state,omitempty"`
	FinishReason       *string           `json:"finish_reason,omitempty"`
	SystemFingerprint  *string           `json:"system_fingerprint,omitempty"`
	Status             *string           `json:"status,omitempty"`
	IncompleteReason   *string           `json:"incomplete_reason,omitempty"`
	ProviderMetadata   map[string]any    `json:"provider_metadata,omitempty"`
	ProviderExtensions map[string]any    `json:"provider_extensions,omitempty"`
	Error              any               `json:"error,omitempty"`
}

type PivotMessage struct {
	Role               string                 `json:"role,omitempty"`
	Name               *string                `json:"name,omitempty"`
	Parts              []PivotContent         `json:"parts,omitempty"`
	ToolCalls          []PivotToolCall        `json:"tool_calls,omitempty"`
	ToolCallID         *string                `json:"tool_call_id,omitempty"`
	Thinking           *PivotThinkingBlock    `json:"thinking,omitempty"`
	Prefix             *bool                  `json:"prefix,omitempty"`
	Metadata           map[string]any         `json:"metadata,omitempty"`
	ProviderExtensions map[string]any         `json:"provider_extensions,omitempty"`
}

type PivotSystemPrompt struct {
	Parts              []PivotContent  `json:"parts,omitempty"`
	Text               *string         `json:"text,omitempty"`
	ProviderExtensions map[string]any  `json:"provider_extensions,omitempty"`
}

type PivotContent struct {
	Type               string                `json:"type,omitempty"`
	Text               *string               `json:"text,omitempty"`
	Media              *PivotMedia           `json:"media,omitempty"`
	ToolCall           *PivotToolCall        `json:"tool_call,omitempty"`
	ToolResult         *PivotToolResult      `json:"tool_result,omitempty"`
	FunctionCall       *PivotFunctionCall    `json:"function_call,omitempty"`
	FunctionResponse   *PivotFunctionResult  `json:"function_response,omitempty"`
	Thinking           *PivotThinkingBlock   `json:"thinking,omitempty"`
	JSON               json.RawMessage       `json:"json,omitempty"`
	Data               any                   `json:"data,omitempty"`
	ProviderExtensions map[string]any        `json:"provider_extensions,omitempty"`
}

type PivotMedia struct {
	Kind               string          `json:"kind,omitempty"`
	URL                *string         `json:"url,omitempty"`
	MimeType           *string         `json:"mime_type,omitempty"`
	Data               *string         `json:"data,omitempty"`
	FileName           *string         `json:"file_name,omitempty"`
	Detail             *string         `json:"detail,omitempty"`
	ProviderExtensions map[string]any  `json:"provider_extensions,omitempty"`
}

type PivotTool struct {
	Type               string          `json:"type,omitempty"`
	Name               string          `json:"name,omitempty"`
	Description        *string         `json:"description,omitempty"`
	Parameters         any             `json:"parameters,omitempty"`
	InputSchema        any             `json:"input_schema,omitempty"`
	Strict             *bool           `json:"strict,omitempty"`
	ProviderExtensions map[string]any  `json:"provider_extensions,omitempty"`
}

type PivotToolChoice struct {
	Type                   *string        `json:"type,omitempty"`
	Name                   *string        `json:"name,omitempty"`
	DisableParallelToolUse *bool          `json:"disable_parallel_tool_use,omitempty"`
	Required               *bool          `json:"required,omitempty"`
	ProviderExtensions     map[string]any `json:"provider_extensions,omitempty"`
}

type PivotToolCall struct {
	ID                 *string           `json:"id,omitempty"`
	Type               *string           `json:"type,omitempty"`
	Name               *string           `json:"name,omitempty"`
	Arguments          json.RawMessage   `json:"arguments,omitempty"`
	Input              any               `json:"input,omitempty"`
	Index              *int              `json:"index,omitempty"`
	ProviderExtensions map[string]any    `json:"provider_extensions,omitempty"`
}

type PivotToolResult struct {
	ToolCallID         *string         `json:"tool_call_id,omitempty"`
	Name               *string         `json:"name,omitempty"`
	Content            any             `json:"content,omitempty"`
	IsError            *bool           `json:"is_error,omitempty"`
	ProviderExtensions map[string]any  `json:"provider_extensions,omitempty"`
}

type PivotFunctionCall struct {
	Name               *string         `json:"name,omitempty"`
	Arguments          json.RawMessage `json:"arguments,omitempty"`
	ProviderExtensions map[string]any  `json:"provider_extensions,omitempty"`
}

type PivotFunctionResult struct {
	Name               *string         `json:"name,omitempty"`
	Response           any             `json:"response,omitempty"`
	WillContinue       *bool           `json:"will_continue,omitempty"`
	ProviderExtensions map[string]any  `json:"provider_extensions,omitempty"`
}

type PivotThinkingConfig struct {
	Type               *string         `json:"type,omitempty"`
	Enabled            *bool           `json:"enabled,omitempty"`
	BudgetTokens       *int            `json:"budget_tokens,omitempty"`
	Effort             *string         `json:"effort,omitempty"`
	Level              *string         `json:"level,omitempty"`
	IncludeThoughts    *bool           `json:"include_thoughts,omitempty"`
	ProviderExtensions map[string]any  `json:"provider_extensions,omitempty"`
}

type PivotThinkingBlock struct {
	Text               *string         `json:"text,omitempty"`
	Signature          *string         `json:"signature,omitempty"`
	Redacted           *bool           `json:"redacted,omitempty"`
	ProviderExtensions map[string]any  `json:"provider_extensions,omitempty"`
}

type PivotResponseFormat struct {
	Type               *string         `json:"type,omitempty"`
	JSONSchema         json.RawMessage `json:"json_schema,omitempty"`
	Schema             any             `json:"schema,omitempty"`
	Name               *string         `json:"name,omitempty"`
	Description        *string         `json:"description,omitempty"`
	Strict             *bool           `json:"strict,omitempty"`
	MimeType           *string         `json:"mime_type,omitempty"`
	ProviderExtensions map[string]any  `json:"provider_extensions,omitempty"`
}

type PivotStreamOptions struct {
	IncludeUsage        *bool          `json:"include_usage,omitempty"`
	IncludeObfuscation  *bool          `json:"include_obfuscation,omitempty"`
	ProviderExtensions  map[string]any `json:"provider_extensions,omitempty"`
}

type PivotAudioConfig struct {
	Voice              *string         `json:"voice,omitempty"`
	Format             *string         `json:"format,omitempty"`
	SampleRate         *int            `json:"sample_rate,omitempty"`
	ProviderExtensions map[string]any  `json:"provider_extensions,omitempty"`
}

type PivotPrediction struct {
	Type               *string         `json:"type,omitempty"`
	Content            any             `json:"content,omitempty"`
	ProviderExtensions map[string]any  `json:"provider_extensions,omitempty"`
}

type PivotToolConfig struct {
	FunctionCalling    *PivotFunctionCallingConfig `json:"function_calling,omitempty"`
	Retrieval          *PivotRetrievalConfig       `json:"retrieval,omitempty"`
	ProviderExtensions map[string]any              `json:"provider_extensions,omitempty"`
}

type PivotFunctionCallingConfig struct {
	Mode                 *string        `json:"mode,omitempty"`
	AllowedFunctionNames []string       `json:"allowed_function_names,omitempty"`
	ProviderExtensions   map[string]any `json:"provider_extensions,omitempty"`
}

type PivotRetrievalConfig struct {
	Latitude           *float64       `json:"latitude,omitempty"`
	Longitude          *float64       `json:"longitude,omitempty"`
	LanguageCode       *string        `json:"language_code,omitempty"`
	ProviderExtensions map[string]any `json:"provider_extensions,omitempty"`
}

type PivotSafetySetting struct {
	Category           string         `json:"category,omitempty"`
	Threshold          *string        `json:"threshold,omitempty"`
	Probability        *string        `json:"probability,omitempty"`
	Blocked            *bool          `json:"blocked,omitempty"`
	ProviderExtensions map[string]any `json:"provider_extensions,omitempty"`
}

type PivotChoice struct {
	Index              *int            `json:"index,omitempty"`
	Message            *PivotMessage   `json:"message,omitempty"`
	Delta              *PivotMessage   `json:"delta,omitempty"`
	FinishReason       *string         `json:"finish_reason,omitempty"`
	LogProbs           any             `json:"logprobs,omitempty"`
	StopReason         *string         `json:"stop_reason,omitempty"`
	SafetyRatings      []PivotSafetySetting `json:"safety_ratings,omitempty"`
	ProviderExtensions map[string]any  `json:"provider_extensions,omitempty"`
}

type PivotUsage struct {
	PromptTokens                *int                  `json:"prompt_tokens,omitempty"`
	CompletionTokens            *int                  `json:"completion_tokens,omitempty"`
	TotalTokens                 *int                  `json:"total_tokens,omitempty"`
	InputTokens                 *int                  `json:"input_tokens,omitempty"`
	OutputTokens                *int                  `json:"output_tokens,omitempty"`
	PromptCacheHitTokens        *int                  `json:"prompt_cache_hit_tokens,omitempty"`
	CacheCreationInputTokens    *int                  `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens        *int                  `json:"cache_read_input_tokens,omitempty"`
	ClaudeCacheCreation5mTokens *int                  `json:"claude_cache_creation_5_m_tokens,omitempty"`
	ClaudeCacheCreation1hTokens *int                  `json:"claude_cache_creation_1_h_tokens,omitempty"`
	ThoughtsTokenCount          *int                  `json:"thoughts_token_count,omitempty"`
	ToolUsePromptTokenCount     *int                  `json:"tool_use_prompt_token_count,omitempty"`
	PromptTokensDetails         *PivotInputTokenStats `json:"prompt_tokens_details,omitempty"`
	CompletionTokenDetails      *PivotOutputTokenStats `json:"completion_tokens_details,omitempty"`
	InputTokensDetails          *PivotInputTokenStats `json:"input_tokens_details,omitempty"`
	Cost                        any                   `json:"cost,omitempty"`
	UsageSemantic               *string               `json:"usage_semantic,omitempty"`
	UsageSource                 *string               `json:"usage_source,omitempty"`
	ProviderExtensions          map[string]any        `json:"provider_extensions,omitempty"`
}

type PivotInputTokenStats struct {
	CachedTokens         *int          `json:"cached_tokens,omitempty"`
	CachedCreationTokens *int          `json:"cached_creation_tokens,omitempty"`
	TextTokens           *int          `json:"text_tokens,omitempty"`
	AudioTokens          *int          `json:"audio_tokens,omitempty"`
	ImageTokens          *int          `json:"image_tokens,omitempty"`
	ProviderExtensions   map[string]any `json:"provider_extensions,omitempty"`
}

type PivotOutputTokenStats struct {
	TextTokens         *int          `json:"text_tokens,omitempty"`
	AudioTokens        *int          `json:"audio_tokens,omitempty"`
	ReasoningTokens    *int          `json:"reasoning_tokens,omitempty"`
	ProviderExtensions map[string]any `json:"provider_extensions,omitempty"`
}

type PivotStreamState struct {
	IsStream           *bool          `json:"is_stream,omitempty"`
	Done               *bool          `json:"done,omitempty"`
	Event              *string        `json:"event,omitempty"`
	ChunkIndex         *int           `json:"chunk_index,omitempty"`
	Sequence           *int           `json:"sequence,omitempty"`
	LastMessageType    *string        `json:"last_message_type,omitempty"`
	ProviderExtensions map[string]any `json:"provider_extensions,omitempty"`
}
