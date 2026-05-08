package service

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
)

// setupRetryFamilyCache populates the model channel cache with:
//   - Channel 10 (OpenAI) at priority 10
//   - Channel 11 (Anthropic) at priority 0
//   Both serve model "gpt-4" in group "default".
func setupRetryFamilyCache(t *testing.T) {
	t.Helper()

	priority10 := int64(10)
	priority0 := int64(0)
	weight10 := uint(10)

	idm := map[int]*model.Channel{
		10: {
			Id:       10,
			Type:     constant.ChannelTypeOpenAI,
			Status:   common.ChannelStatusEnabled,
			Name:     "openai-p10",
			Priority: &priority10,
			Weight:   &weight10,
			Group:    "default",
			Models:   "gpt-4",
			Key:      "key-10",
		},
		11: {
			Id:       11,
			Type:     constant.ChannelTypeAnthropic,
			Status:   common.ChannelStatusEnabled,
			Name:     "anthropic-p0",
			Priority: &priority0,
			Weight:   &weight10,
			Group:    "default",
			Models:   "gpt-4",
			Key:      "key-11",
		},
	}

	g2m2c := map[string]map[string][]int{
		"default": {
			"gpt-4": {10, 11},
		},
	}

	model.SetupTestChannelCache(idm, g2m2c)
}

// setupNoSameFamilyCache has only Anthropic channels — no OpenAI channels at all.
func setupNoSameFamilyCache(t *testing.T) {
	t.Helper()

	priority10 := int64(10)
	weight10 := uint(10)

	idm := map[int]*model.Channel{
		20: {
			Id:       20,
			Type:     constant.ChannelTypeAnthropic,
			Status:   common.ChannelStatusEnabled,
			Name:     "anthropic-only",
			Priority: &priority10,
			Weight:   &weight10,
			Group:    "default",
			Models:   "gpt-4",
			Key:      "key-20",
		},
	}

	g2m2c := map[string]map[string][]int{
		"default": {
			"gpt-4": {20},
		},
	}

	model.SetupTestChannelCache(idm, g2m2c)
}

func TestCacheGetRandomSatisfiedChannel_RetrySameFamilyFirst(t *testing.T) {
	setupRetryFamilyCache(t)

	c, _ := gin.CreateTestContext(nil)

	retry := 0
	param := &RetryParam{
		Ctx:           c,
		TokenGroup:    "default",
		ModelName:     "gpt-4",
		Retry:         &retry,
		RoutingFamily: common.RoutingFamilyOpenAI,
	}

	ch, group, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, ch, "retry=0 with OpenAI family should find same-family channel")
	assert.Equal(t, "default", group)
	assert.Equal(t, 10, ch.Id, "should select OpenAI-family channel (id=10)")
}

func TestCacheGetRandomSatisfiedChannel_RetrySameFamilyAcrossPriorities(t *testing.T) {
	setupRetryFamilyCache(t)

	c, _ := gin.CreateTestContext(nil)

	// retry=1 with OpenAI family: model layer filters to [10] (only OpenAI channel),
	// which has 1 unique priority. retry=1 is clamped to retry=0, returns channel 10.
	retry := 1
	param := &RetryParam{
		Ctx:           c,
		TokenGroup:    "default",
		ModelName:     "gpt-4",
		Retry:         &retry,
		RoutingFamily: common.RoutingFamilyOpenAI,
	}

	ch, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, ch, "retry=1 with OpenAI family should still find same-family channel")
	assert.Equal(t, 10, ch.Id, "should return same-family channel even at retry=1 (priority clamp)")
}

func TestCacheGetRandomSatisfiedChannel_CrossFamilyFallbackWhenNoSameFamily(t *testing.T) {
	setupNoSameFamilyCache(t)

	c, _ := gin.CreateTestContext(nil)

	// Requesting OpenAI family when no OpenAI channels exist at all.
	// Same-family selection returns nil, then cross-family fallback should find channel 20.
	retry := 0
	param := &RetryParam{
		Ctx:           c,
		TokenGroup:    "default",
		ModelName:     "gpt-4",
		Retry:         &retry,
		RoutingFamily: common.RoutingFamilyOpenAI,
	}

	ch, group, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, ch, "should fall back to cross-family channel when no same-family exists")
	assert.Equal(t, "default", group)
	assert.Equal(t, 20, ch.Id, "should fall back to Anthropic-family channel (id=20)")
}

func TestCacheGetRandomSatisfiedChannel_NoneFamilyNoFilter(t *testing.T) {
	setupRetryFamilyCache(t)

	c, _ := gin.CreateTestContext(nil)

	retry := 0
	param := &RetryParam{
		Ctx:           c,
		TokenGroup:    "default",
		ModelName:     "gpt-4",
		Retry:         &retry,
		RoutingFamily: common.RoutingFamilyNone,
	}

	ch, group, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, ch, "retry=0 with no family filter should find a channel")
	assert.Equal(t, "default", group)
	assert.Equal(t, 10, ch.Id, "should select highest-priority channel regardless of family")
}

func TestCacheGetRandomSatisfiedChannel_RetryPreservesFamilyAcrossAttempts(t *testing.T) {
	setupRetryFamilyCache(t)

	c, _ := gin.CreateTestContext(nil)

	param := &RetryParam{
		Ctx:           c,
		TokenGroup:    "default",
		ModelName:     "gpt-4",
		Retry:         common.GetPointer(0),
		RoutingFamily: common.RoutingFamilyOpenAI,
	}

	// Attempt 1 (retry=0): should find OpenAI at priority 10.
	ch, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, ch)
	assert.Equal(t, 10, ch.Id, "first attempt should find OpenAI-family channel")

	param.IncreaseRetry()

	// Attempt 2 (retry=1): still finds same-family channel (priority clamp).
	ch, _, err = CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, ch, "second attempt should still find same-family channel")
	assert.Equal(t, 10, ch.Id, "second attempt should return same-family channel (priority clamp)")

	assert.Equal(t, common.RoutingFamilyOpenAI, param.RoutingFamily,
		"RoutingFamily on RetryParam should not drift across retries")
}

func TestCacheGetRandomSatisfiedChannel_RetryCrossFamilyFallbackAcrossAttempts(t *testing.T) {
	setupNoSameFamilyCache(t)

	c, _ := gin.CreateTestContext(nil)

	param := &RetryParam{
		Ctx:           c,
		TokenGroup:    "default",
		ModelName:     "gpt-4",
		Retry:         common.GetPointer(0),
		RoutingFamily: common.RoutingFamilyOpenAI,
	}

	// Attempt 1 (retry=0): no OpenAI channels, cross-family fallback to channel 20.
	ch, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, ch)
	assert.Equal(t, 20, ch.Id, "first attempt should fall back to cross-family channel")

	param.IncreaseRetry()

	// Attempt 2 (retry=1): still no OpenAI channels, cross-family fallback again.
	ch, _, err = CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, ch, "second attempt should still fall back to cross-family channel")
	assert.Equal(t, 20, ch.Id, "second attempt should fall back to same cross-family channel")

	assert.Equal(t, common.RoutingFamilyOpenAI, param.RoutingFamily,
		"RoutingFamily on RetryParam should not drift across retries")
}

func TestCacheGetRandomSatisfiedChannel_RoutingFamilyFromContext(t *testing.T) {
	setupRetryFamilyCache(t)

	c, _ := gin.CreateTestContext(nil)
	common.SetContextKey(c, constant.ContextKeyRequestRoutingFamily, common.RoutingFamilyOpenAI)

	retry := 0
	param := &RetryParam{
		Ctx:           c,
		TokenGroup:    "default",
		ModelName:     "gpt-4",
		Retry:         &retry,
		RoutingFamily: "",
	}

	ch, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, ch)
	assert.Equal(t, 10, ch.Id, "should use context RoutingFamily when RetryParam.RoutingFamily is empty")
}
