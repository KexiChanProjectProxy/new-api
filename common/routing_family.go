package common

import (
	"strings"

	"github.com/QuantumNous/new-api/constant"
)

// RoutingFamily classifies channels and requests into one of two protocol
// families for routing preference: OpenAI-compatible or Anthropic-compatible.
type RoutingFamily string

const (
	RoutingFamilyOpenAI    RoutingFamily = "openai"
	RoutingFamilyAnthropic RoutingFamily = "anthropic"
	RoutingFamilyNone      RoutingFamily = ""
)

// ChannelTypeToRoutingFamily derives the routing family from a channel type,
// reusing ChannelType2APIType as the canonical source of truth.
func ChannelTypeToRoutingFamily(channelType int) RoutingFamily {
	apiType, _ := ChannelType2APIType(channelType)
	return apiTypeToRoutingFamily(apiType)
}

// PathToRoutingFamily derives the routing family from a request URL path.
// Paths under /v1/messages (the Anthropic Messages API) map to the Anthropic
// family; all other /v1/... paths map to the OpenAI family.
func PathToRoutingFamily(path string) RoutingFamily {
	if strings.HasSuffix(path, "/v1/messages") || strings.Contains(path, "/v1/messages?") {
		return RoutingFamilyAnthropic
	}
	if strings.HasPrefix(path, "/v1/") || strings.HasPrefix(path, "/pg/") || path == "/v1" {
		return RoutingFamilyOpenAI
	}
	for _, prefix := range RelaySubpaths {
		if strings.HasPrefix(path, prefix+"/v1/messages") {
			return RoutingFamilyAnthropic
		}
		if strings.HasPrefix(path, prefix+"/v1/") {
			return RoutingFamilyOpenAI
		}
	}
	return RoutingFamilyNone
}

// RelayFormatToRoutingFamily derives the routing family from a relay format
// string, the strongest request-side signal for protocol family.
func RelayFormatToRoutingFamily(relayFormat string) RoutingFamily {
	switch relayFormat {
	case "claude":
		return RoutingFamilyAnthropic
	case "openai", "openai_responses", "openai_responses_compaction",
		"openai_audio", "openai_image", "openai_realtime",
		"rerank", "embedding", "mcp",
		"gemini", "task", "mj_proxy":
		return RoutingFamilyOpenAI
	default:
		return RoutingFamilyNone
	}
}

func apiTypeToRoutingFamily(apiType int) RoutingFamily {
	switch apiType {
	case constant.APITypeAnthropic, constant.APITypeAws:
		return RoutingFamilyAnthropic
	default:
		return RoutingFamilyOpenAI
	}
}
