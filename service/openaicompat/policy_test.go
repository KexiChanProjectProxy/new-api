package openaicompat

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/stretchr/testify/require"
)

func TestOpenAIResponsesChannelUsesChatToResponsesConversion(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	originalSettings := *settings
	t.Cleanup(func() {
		*settings = originalSettings
	})

	settings.PassThroughRequestEnabled = false
	settings.ChatCompletionsToResponsesPolicy = model_setting.ChatCompletionsToResponsesPolicy{}

	require.True(t, ShouldChatCompletionsUseResponsesGlobal(58, constant.ChannelTypeOpenAIResponses, "gpt-5"))

	require.False(t, ShouldChatCompletionsUseResponsesGlobal(1, constant.ChannelTypeOpenAI, "gpt-5"))
}
