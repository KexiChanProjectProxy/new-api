package xunfei

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestXunfeiMakeRequest_AppliesGatewayXFFHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := newXunfeiTestContext("203.0.113.10:4567")
	headers, stopChan := runXunfeiMakeRequestThroughServer(t, ctx, true)

	require.Equal(t, "203.0.113.10", headers.Get("X-Forwarded-For"))
	require.Equal(t, "203.0.113.10", headers.Get("X-Real-IP"))
	require.True(t, <-stopChan)
	require.NotNil(t, recorder)
}

func TestXunfeiMakeRequest_DoesNotInjectHeadersWhenDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := newXunfeiTestContext("203.0.113.10:4567")
	headers, stopChan := runXunfeiMakeRequestThroughServer(t, ctx, false)

	require.Empty(t, headers.Get("X-Forwarded-For"))
	require.Empty(t, headers.Get("X-Real-IP"))
	require.True(t, <-stopChan)
	require.NotNil(t, recorder)
}

func newXunfeiTestContext(remoteAddr string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"SparkDesk","messages":[]}`))
	ctx.Request.RemoteAddr = remoteAddr
	return ctx, recorder
}

func runXunfeiMakeRequestThroughServer(t *testing.T, ctx *gin.Context, forwardXFF bool) (http.Header, chan bool) {
	t.Helper()

	headersCh := make(chan http.Header, 1)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		headersCh <- r.Header.Clone()

		var request XunfeiChatRequest
		require.NoError(t, conn.ReadJSON(&request))
		require.NoError(t, conn.WriteJSON(XunfeiChatResponse{
			Payload: struct {
				Choices struct {
					Status int                          `json:"status"`
					Seq    int                          `json:"seq"`
					Text   []XunfeiChatResponseTextItem `json:"text"`
				} `json:"choices"`
				Usage struct {
					Text dto.Usage `json:"text"`
				} `json:"usage"`
			}{
				Choices: struct {
					Status int                          `json:"status"`
					Seq    int                          `json:"seq"`
					Text   []XunfeiChatResponseTextItem `json:"text"`
				}{
					Status: 2,
					Text: []XunfeiChatResponseTextItem{{
						Content: "done",
						Role:    "assistant",
						Index:   0,
					}},
				},
			},
		}))
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	dataChan, stopChan, err := xunfeiMakeRequest(ctx, dto.GeneralOpenAIRequest{}, "general", wsURL, "app-id", forwardXFF)
	require.NoError(t, err)
	response := <-dataChan
	require.Equal(t, 2, response.Payload.Choices.Status)

	return <-headersCh, stopChan
}
