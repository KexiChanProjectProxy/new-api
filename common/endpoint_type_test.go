package common_test

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relay"
	"github.com/stretchr/testify/require"
)

func TestOpenAIResponsesChannelType(t *testing.T) {
	apiType, success := common.ChannelType2APIType(constant.ChannelTypeOpenAIResponses)

	require.True(t, success)
	require.Equal(t, constant.APITypeOpenAI, apiType)
	require.Equal(t, "https://api.openai.com", constant.ChannelBaseURLs[constant.ChannelTypeOpenAIResponses])
	require.Equal(t, 1, constant.ChannelTypeOpenAI)
	require.Equal(t, 59, constant.ChannelTypeDummy)
	require.Equal(t, "OpenAI Chat", constant.GetChannelTypeName(constant.ChannelTypeOpenAI))
	require.Equal(t, "OpenAI Responses", constant.GetChannelTypeName(constant.ChannelTypeOpenAIResponses))
	require.Equal(t, []constant.EndpointType{constant.EndpointTypeOpenAIResponse}, common.GetEndpointTypesByChannelType(constant.ChannelTypeOpenAIResponses, "gpt-4o"))
	require.Equal(t, []constant.EndpointType{constant.EndpointTypeOpenAI}, common.GetEndpointTypesByChannelType(constant.ChannelTypeOpenAI, "gpt-4o"))
	require.Nil(t, relay.GetTaskAdaptor(constant.TaskPlatform("58")))
}
