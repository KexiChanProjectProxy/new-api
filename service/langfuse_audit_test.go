package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newLangfuseTestContext(t *testing.T) (*gin.Context, *relaycommon.RelayInfo) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	c.Set("username", "alice")
	c.Set(common.RequestIdKey, "req-123")
	c.Set(common.UpstreamRequestIdKey, "up-456")
	c.Set("use_channel", []string{"10", "11", "12"})

	info := &relaycommon.RelayInfo{
		StartTime:         time.Now().Add(-250 * time.Millisecond),
		FirstResponseTime: time.Now().Add(-100 * time.Millisecond),
		TokenKey:          "sk-proj-abcdef1234567890",
		UserId:            42,
		OriginModelName:   "gpt-4o",
		TokenGroup:        "default",
		UsingGroup:        "vip",
		RequestId:         "req-123",
		ResolvedClientIP:  "10.0.0.5",
		IsStream:          true,
		IsPlayground:      false,
		RetryIndex:        1,
		RelayFormat:       types.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       1,
			ChannelId:         10,
			UpstreamModelName: "gpt-4o-2024-08-06",
			IsModelMapped:     true,
		},
		Request: &dto.GeneralOpenAIRequest{
			Model: "gpt-4o",
		},
	}
	return c, info
}

func TestLangfuseBuildsValidatedRequestSnapshot(t *testing.T) {
	c, info := newLangfuseTestContext(t)

	snapshot, err := BuildLangfuseAudit(info, c)
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.NotEmpty(t, snapshot.RequestPayload)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(snapshot.RequestPayload, &parsed))
	require.Equal(t, "gpt-4o", parsed["model"])

	meta := snapshot.Metadata
	require.Equal(t, "alice", meta.Username)
	require.Equal(t, 42, meta.UserId)
	require.Equal(t, "req-123", meta.RequestId)
	require.Equal(t, "up-456", meta.UpstreamRequestId)
	require.Equal(t, "gpt-4o", meta.ModelName)
	require.Equal(t, "default", meta.GroupName)
	require.Equal(t, "vip", meta.UsingGroup)
	require.Equal(t, []int{10, 11, 12}, meta.RetryChain)
	require.Equal(t, 1, meta.RetryIndex)
	require.Equal(t, "gpt-4o-2024-08-06", meta.UpstreamModelName)
	require.True(t, meta.IsModelMapped)
	require.True(t, meta.IsStream)
	require.False(t, meta.IsPlayground)
	// FirstResponseLatencyMs should be roughly ~150ms (allow wide tolerance).
	require.Greater(t, meta.FirstResponseLatencyMs, int64(50))
	require.Less(t, meta.FirstResponseLatencyMs, int64(5000))
}

func TestLangfuseMasksTokenAndSensitiveFields(t *testing.T) {
	c, info := newLangfuseTestContext(t)
	info.TokenKey = "sk-super-secret-key-1234567890"

	snapshot, err := BuildLangfuseAudit(info, c)
	require.NoError(t, err)

	// 1. Token key is masked, never raw.
	require.NotContains(t, snapshot.Metadata.MaskedTokenKey, "super-secret")
	require.NotEqual(t, "sk-super-secret-key-1234567890", snapshot.Metadata.MaskedTokenKey)
	require.NotEmpty(t, snapshot.Metadata.MaskedTokenKey)

	// 2. Raw token key must not appear anywhere in the serialized snapshot.
	raw, err := common.Marshal(snapshot)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "sk-super-secret-key-1234567890")

	// 3. MaskSensitiveInfo is applied to request payload: embed a URL + IP
	//    into the request body and confirm they get redacted.
	info.Request = &dto.GeneralOpenAIRequest{
		Model: "gpt-4o",
	}
	snapshot2 := &LangfuseAuditSnapshot{
		RequestPayload: []byte(`{"url":"https://api.openai.com/v1/secret","ip":"192.168.1.1"}`),
		Metadata: LangfuseAuditMetadata{
			MaskedTokenKey: "sk-leaked-key-99999",
		},
	}
	MaskLangfuseAudit(snapshot2)
	require.NotContains(t, string(snapshot2.RequestPayload), "api.openai.com")
	require.NotContains(t, string(snapshot2.RequestPayload), "192.168.1.1")
	// Defensive re-mask overwrites a raw-looking token key with the masked form.
	require.NotEqual(t, "sk-leaked-key-99999", snapshot2.Metadata.MaskedTokenKey)
}

func TestLangfuseTruncatesLargeAuditPayloads(t *testing.T) {
	huge := strings.Repeat("a", 100*1024) // 100 KiB, well over the 64 KiB cap.
	snapshot := &LangfuseAuditSnapshot{
		RequestPayload:  []byte(huge),
		ResponsePayload: []byte(huge),
		ErrorPayload:    []byte(huge),
	}
	MaskLangfuseAudit(snapshot)

	require.Equal(t, 100*1024, snapshot.TruncationMarkers.RequestOriginal)
	require.True(t, snapshot.TruncationMarkers.RequestTruncated)
	require.Equal(t, langfuseMaxPayloadBytes, len(snapshot.RequestPayload))
	require.True(t, strings.HasSuffix(string(snapshot.RequestPayload), langfuseTruncationMarker))

	require.Equal(t, 100*1024, snapshot.TruncationMarkers.ResponseOriginal)
	require.True(t, snapshot.TruncationMarkers.ResponseTruncated)
	require.Equal(t, langfuseMaxPayloadBytes, len(snapshot.ResponsePayload))

	require.Equal(t, 100*1024, snapshot.TruncationMarkers.ErrorOriginal)
	require.True(t, snapshot.TruncationMarkers.ErrorTruncated)
	require.Equal(t, langfuseMaxPayloadBytes, len(snapshot.ErrorPayload))

	// Small payloads are not truncated.
	small := []byte(`{"ok":true}`)
	snapshot2 := &LangfuseAuditSnapshot{RequestPayload: small}
	MaskLangfuseAudit(snapshot2)
	require.False(t, snapshot2.TruncationMarkers.RequestTruncated)
	require.Equal(t, len(small), snapshot2.TruncationMarkers.RequestOriginal)
}

func TestLangfuseBinaryPlaceholderIncludesHash(t *testing.T) {
	snapshot := &LangfuseAuditSnapshot{}
	body := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0xFF, 0xFE}
	snapshot.SetBinaryResponse("image/png", body, "image/png response")

	require.NotNil(t, snapshot.BinaryResponse)
	require.Equal(t, "image/png", snapshot.BinaryResponse.ContentType)
	require.Equal(t, int64(len(body)), snapshot.BinaryResponse.ContentLength)
	require.NotEmpty(t, snapshot.BinaryResponse.SHA256)
	require.Equal(t, "image/png response", snapshot.BinaryResponse.OmittedReason)
	// Raw binary bytes must not be on the snapshot.
	require.Nil(t, snapshot.ResponsePayload)

	// Verify the hash matches an independent computation.
	expectedSum := sha256.Sum256(body)
	require.Equal(t, hex.EncodeToString(expectedSum[:]), snapshot.BinaryResponse.SHA256)

	// Binary content-type detection.
	require.True(t, IsBinaryContentType("image/png"))
	require.True(t, IsBinaryContentType("audio/mpeg; charset=binary"))
	require.True(t, IsBinaryContentType("application/octet-stream"))
	require.False(t, IsBinaryContentType("application/json"))
	require.False(t, IsBinaryContentType("text/html"))
	require.False(t, IsBinaryContentType(""))
}
