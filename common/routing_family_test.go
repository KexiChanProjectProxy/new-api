package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
)

func TestChannelTypeToRoutingFamily(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		channelType int
		want        RoutingFamily
	}{
		{
			name:        "anthropic channel maps to anthropic family",
			channelType: constant.ChannelTypeAnthropic,
			want:        RoutingFamilyAnthropic,
		},
		{
			name:        "aws bedrock channel maps to anthropic family",
			channelType: constant.ChannelTypeAws,
			want:        RoutingFamilyAnthropic,
		},
		{
			name:        "openai channel maps to openai family",
			channelType: constant.ChannelTypeOpenAI,
			want:        RoutingFamilyOpenAI,
		},
		{
			name:        "azure channel maps to openai family",
			channelType: constant.ChannelTypeAzure,
			want:        RoutingFamilyOpenAI,
		},
		{
			name:        "gemini channel maps to openai family",
			channelType: constant.ChannelTypeGemini,
			want:        RoutingFamilyOpenAI,
		},
		{
			name:        "deepseek channel maps to openai family",
			channelType: constant.ChannelTypeDeepSeek,
			want:        RoutingFamilyOpenAI,
		},
		{
			name:        "ollama channel maps to openai family",
			channelType: constant.ChannelTypeOllama,
			want:        RoutingFamilyOpenAI,
		},
		{
			name:        "vertex ai channel maps to openai family",
			channelType: constant.ChannelTypeVertexAi,
			want:        RoutingFamilyOpenAI,
		},
		{
			name:        "cohere channel maps to openai family",
			channelType: constant.ChannelTypeCohere,
			want:        RoutingFamilyOpenAI,
		},
		{
			name:        "unknown channel type defaults to openai family",
			channelType: 9999,
			want:        RoutingFamilyOpenAI,
		},
		{
			name:        "zero channel type defaults to openai family",
			channelType: constant.ChannelTypeUnknown,
			want:        RoutingFamilyOpenAI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ChannelTypeToRoutingFamily(tt.channelType)
			if got != tt.want {
				t.Errorf("ChannelTypeToRoutingFamily(%d) = %q, want %q", tt.channelType, got, tt.want)
			}
		})
	}
}

func TestChannelTypeToRoutingFamily_OnlyTwoFamilies(t *testing.T) {
	t.Parallel()

	for ch := 1; ch < int(constant.ChannelTypeDummy); ch++ {
		family := ChannelTypeToRoutingFamily(ch)
		if family != RoutingFamilyOpenAI && family != RoutingFamilyAnthropic && family != RoutingFamilyNone {
			t.Errorf("channel type %d produced unexpected family %q; only openai/anthropic/none allowed", ch, family)
		}
	}
}

func TestPathToRoutingFamily(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want RoutingFamily
	}{
		{
			name: "v1 messages maps to anthropic family",
			path: "/v1/messages",
			want: RoutingFamilyAnthropic,
		},
		{
			name: "v1 messages with query maps to anthropic family",
			path: "/v1/messages?beta=true",
			want: RoutingFamilyAnthropic,
		},
		{
			name: "v1 chat completions maps to openai family",
			path: "/v1/chat/completions",
			want: RoutingFamilyOpenAI,
		},
		{
			name: "v1 completions maps to openai family",
			path: "/v1/completions",
			want: RoutingFamilyOpenAI,
		},
		{
			name: "v1 embeddings maps to openai family",
			path: "/v1/embeddings",
			want: RoutingFamilyOpenAI,
		},
		{
			name: "v1 responses maps to openai family",
			path: "/v1/responses",
			want: RoutingFamilyOpenAI,
		},
		{
			name: "pg chat completions maps to openai family",
			path: "/pg/chat/completions",
			want: RoutingFamilyOpenAI,
		},
		{
			name: "unknown path maps to none",
			path: "/something/else",
			want: RoutingFamilyNone,
		},
		{
			name: "empty path maps to none",
			path: "",
			want: RoutingFamilyNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := PathToRoutingFamily(tt.path)
			if got != tt.want {
				t.Errorf("PathToRoutingFamily(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestRelayFormatToRoutingFamily(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		relayFormat string
		want        RoutingFamily
	}{
		{
			name:        "claude format maps to anthropic family",
			relayFormat: "claude",
			want:        RoutingFamilyAnthropic,
		},
		{
			name:        "openai format maps to openai family",
			relayFormat: "openai",
			want:        RoutingFamilyOpenAI,
		},
		{
			name:        "openai_responses format maps to openai family",
			relayFormat: "openai_responses",
			want:        RoutingFamilyOpenAI,
		},
		{
			name:        "openai_responses_compaction format maps to openai family",
			relayFormat: "openai_responses_compaction",
			want:        RoutingFamilyOpenAI,
		},
		{
			name:        "openai_audio format maps to openai family",
			relayFormat: "openai_audio",
			want:        RoutingFamilyOpenAI,
		},
		{
			name:        "openai_image format maps to openai family",
			relayFormat: "openai_image",
			want:        RoutingFamilyOpenAI,
		},
		{
			name:        "openai_realtime format maps to openai family",
			relayFormat: "openai_realtime",
			want:        RoutingFamilyOpenAI,
		},
		{
			name:        "gemini format maps to openai family",
			relayFormat: "gemini",
			want:        RoutingFamilyOpenAI,
		},
		{
			name:        "embedding format maps to openai family",
			relayFormat: "embedding",
			want:        RoutingFamilyOpenAI,
		},
		{
			name:        "rerank format maps to openai family",
			relayFormat: "rerank",
			want:        RoutingFamilyOpenAI,
		},
		{
			name:        "mcp format maps to openai family",
			relayFormat: "mcp",
			want:        RoutingFamilyOpenAI,
		},
		{
			name:        "empty string maps to none",
			relayFormat: "",
			want:        RoutingFamilyNone,
		},
		{
			name:        "unknown format maps to none",
			relayFormat: "something_new",
			want:        RoutingFamilyNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := RelayFormatToRoutingFamily(tt.relayFormat)
			if got != tt.want {
				t.Errorf("RelayFormatToRoutingFamily(%q) = %q, want %q", tt.relayFormat, got, tt.want)
			}
		})
	}
}

func TestRelayFormatToRoutingFamily_OnlyTwoFamilies(t *testing.T) {
	t.Parallel()

	knownFormats := []string{
		"openai", "claude", "gemini", "openai_responses",
		"openai_responses_compaction", "openai_audio", "openai_image",
		"openai_realtime", "rerank", "embedding", "mcp", "task", "mj_proxy",
	}
	for _, f := range knownFormats {
		family := RelayFormatToRoutingFamily(f)
		if family != RoutingFamilyOpenAI && family != RoutingFamilyAnthropic {
			t.Errorf("relay format %q produced family %q; only openai/anthropic allowed for known formats", f, family)
		}
	}
}
