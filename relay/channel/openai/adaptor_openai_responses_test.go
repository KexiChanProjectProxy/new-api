package openai

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIResponsesOrganizationHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	header := make(http.Header)
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:  constant.ChannelTypeOpenAIResponses,
			ApiKey:       "test-key",
			Organization: "test-organization",
		},
	}

	err := (&Adaptor{}).SetupRequestHeader(c, &header, info)

	require.NoError(t, err)
	require.Equal(t, "test-organization", header.Get("OpenAI-Organization"))
}

func TestOpenAIResponsesStreamOptions(t *testing.T) {
	streamOptions := &dto.StreamOptions{IncludeUsage: true, IncludeObfuscation: true}
	request := &dto.GeneralOpenAIRequest{StreamOptions: streamOptions}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeOpenAIResponses,
			UpstreamModelName: "gpt-4o",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)

	require.NoError(t, err)
	convertedRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Same(t, streamOptions, convertedRequest.StreamOptions)
}
