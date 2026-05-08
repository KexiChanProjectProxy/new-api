package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFamilyTestCache(t *testing.T) {
	t.Helper()
	channelSyncLock.Lock()
	defer channelSyncLock.Unlock()

	common.MemoryCacheEnabled = true

	priority0 := int64(0)
	weight10 := uint(10)

	channelsIDM = map[int]*Channel{
		1: {
			Id:       1,
			Type:     constant.ChannelTypeOpenAI,
			Status:   common.ChannelStatusEnabled,
			Name:     "openai-ch",
			Priority: &priority0,
			Weight:   &weight10,
			Group:    "default",
			Models:   "gpt-4",
			Key:      "key-1",
		},
		2: {
			Id:       2,
			Type:     constant.ChannelTypeAnthropic,
			Status:   common.ChannelStatusEnabled,
			Name:     "anthropic-ch",
			Priority: &priority0,
			Weight:   &weight10,
			Group:    "default",
			Models:   "gpt-4",
			Key:      "key-2",
		},
		3: {
			Id:       3,
			Type:     constant.ChannelTypeAzure,
			Status:   common.ChannelStatusEnabled,
			Name:     "azure-ch",
			Priority: &priority0,
			Weight:   &weight10,
			Group:    "default",
			Models:   "gpt-4",
			Key:      "key-3",
		},
		4: {
			Id:       4,
			Type:     constant.ChannelTypeAws,
			Status:   common.ChannelStatusEnabled,
			Name:     "aws-ch",
			Priority: &priority0,
			Weight:   &weight10,
			Group:    "default",
			Models:   "gpt-4",
			Key:      "key-4",
		},
	}

	group2model2channels = map[string]map[string][]int{
		"default": {
			"gpt-4": {1, 2, 3, 4},
		},
	}
}

func TestGetRandomSatisfiedChannel_SameFamilyPreference(t *testing.T) {
	setupFamilyTestCache(t)

	for i := 0; i < 20; i++ {
		ch, err := GetRandomSatisfiedChannel("default", "gpt-4", 0, common.RoutingFamilyOpenAI)
		require.NoError(t, err)
		require.NotNil(t, ch)
		family := common.ChannelTypeToRoutingFamily(ch.Type)
		assert.Equal(t, common.RoutingFamilyOpenAI, family,
			"expected OpenAI-family channel, got type=%d (family=%s)", ch.Type, family)
	}
}

func TestGetRandomSatisfiedChannel_AnthropicFamilyPreference(t *testing.T) {
	setupFamilyTestCache(t)

	for i := 0; i < 20; i++ {
		ch, err := GetRandomSatisfiedChannel("default", "gpt-4", 0, common.RoutingFamilyAnthropic)
		require.NoError(t, err)
		require.NotNil(t, ch)
		family := common.ChannelTypeToRoutingFamily(ch.Type)
		assert.Equal(t, common.RoutingFamilyAnthropic, family,
			"expected Anthropic-family channel, got type=%d (family=%s)", ch.Type, family)
	}
}

func setupRetryFamilyTestCache(t *testing.T) {
	t.Helper()
	channelSyncLock.Lock()
	defer channelSyncLock.Unlock()

	common.MemoryCacheEnabled = true

	priority10 := int64(10)
	priority0 := int64(0)
	weight10 := uint(10)

	channelsIDM = map[int]*Channel{
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

	group2model2channels = map[string]map[string][]int{
		"default": {
			"gpt-4": {10, 11},
		},
	}
}

func TestGetRandomSatisfiedChannel_RetrySameFamilyFirstThenCrossFamilyFallback(t *testing.T) {
	setupRetryFamilyTestCache(t)

	// retry=0 with OpenAI family: filters to [10] (OpenAI at priority 10).
	// Only 1 unique priority, so retry=0 returns channel 10.
	ch, err := GetRandomSatisfiedChannel("default", "gpt-4", 0, common.RoutingFamilyOpenAI)
	require.NoError(t, err)
	require.NotNil(t, ch)
	assert.Equal(t, 10, ch.Id, "retry=0 OpenAI family should select openai-p10 (same family)")

	// retry=1 with OpenAI family: still filters to [10]. Only 1 unique priority,
	// so retry=1 is clamped to retry=0 and returns channel 10 again.
	// The model layer does NOT return nil when same-family channels exist at a different priority.
	ch, err = GetRandomSatisfiedChannel("default", "gpt-4", 1, common.RoutingFamilyOpenAI)
	require.NoError(t, err)
	require.NotNil(t, ch)
	assert.Equal(t, 10, ch.Id, "retry=1 OpenAI family should still return openai-p10 (only same-family priority)")

	// retry=0 with RoutingFamilyNone: no family filter, all channels available.
	// Priority 10 has channel 10 (OpenAI), so it's selected.
	ch, err = GetRandomSatisfiedChannel("default", "gpt-4", 0, common.RoutingFamilyNone)
	require.NoError(t, err)
	require.NotNil(t, ch)
	assert.Equal(t, 10, ch.Id, "retry=0 with no family filter should select highest-priority channel")

	// retry=1 with RoutingFamilyNone: all channels, priority 0 has channel 11 (Anthropic).
	ch, err = GetRandomSatisfiedChannel("default", "gpt-4", 1, common.RoutingFamilyNone)
	require.NoError(t, err)
	require.NotNil(t, ch)
	assert.Equal(t, 11, ch.Id, "retry=1 with no family filter should select next-priority channel (cross-family)")
}

func TestGetRandomSatisfiedChannel_RetryAnthropicFamilySameFirst(t *testing.T) {
	setupRetryFamilyTestCache(t)

	// retry=0 with Anthropic family: filters to [11] (Anthropic at priority 0).
	// Only 1 unique priority, so retry=0 returns channel 11.
	ch, err := GetRandomSatisfiedChannel("default", "gpt-4", 0, common.RoutingFamilyAnthropic)
	require.NoError(t, err)
	require.NotNil(t, ch)
	assert.Equal(t, 11, ch.Id, "retry=0 Anthropic family should select anthropic-p0 (same family)")
}

func setupNoSameFamilyTestCache(t *testing.T) {
	t.Helper()
	channelSyncLock.Lock()
	defer channelSyncLock.Unlock()

	common.MemoryCacheEnabled = true

	priority10 := int64(10)
	weight10 := uint(10)

	// Only Anthropic channels exist — no OpenAI channels at all.
	channelsIDM = map[int]*Channel{
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

	group2model2channels = map[string]map[string][]int{
		"default": {
			"gpt-4": {20},
		},
	}
}

func TestGetRandomSatisfiedChannel_NoSameFamilyCrossFamilyFallback(t *testing.T) {
	setupNoSameFamilyTestCache(t)

	// Requesting OpenAI family when no OpenAI channels exist at all.
	// The model layer falls back to cross-family (len(sameFamily) == 0, so channels stays as [20]).
	ch, err := GetRandomSatisfiedChannel("default", "gpt-4", 0, common.RoutingFamilyOpenAI)
	require.NoError(t, err)
	require.NotNil(t, ch, "model layer should fall back to cross-family when no same-family channels exist")
	assert.Equal(t, 20, ch.Id, "should return cross-family channel when no same-family exists")

	// With RoutingFamilyNone, should also return the Anthropic channel.
	ch, err = GetRandomSatisfiedChannel("default", "gpt-4", 0, common.RoutingFamilyNone)
	require.NoError(t, err)
	require.NotNil(t, ch)
	assert.Equal(t, 20, ch.Id, "with no family filter, should return any available channel")
}

func setupMixedFamilySamePriorityCache(t *testing.T) {
	t.Helper()
	channelSyncLock.Lock()
	defer channelSyncLock.Unlock()

	common.MemoryCacheEnabled = true

	priority10 := int64(10)
	weight10 := uint(10)

	// Both OpenAI and Anthropic at the same priority level.
	channelsIDM = map[int]*Channel{
		30: {
			Id:       30,
			Type:     constant.ChannelTypeOpenAI,
			Status:   common.ChannelStatusEnabled,
			Name:     "openai-mixed",
			Priority: &priority10,
			Weight:   &weight10,
			Group:    "default",
			Models:   "gpt-4",
			Key:      "key-30",
		},
		31: {
			Id:       31,
			Type:     constant.ChannelTypeAnthropic,
			Status:   common.ChannelStatusEnabled,
			Name:     "anthropic-mixed",
			Priority: &priority10,
			Weight:   &weight10,
			Group:    "default",
			Models:   "gpt-4",
			Key:      "key-31",
		},
	}

	group2model2channels = map[string]map[string][]int{
		"default": {
			"gpt-4": {30, 31},
		},
	}
}

func TestGetRandomSatisfiedChannel_SameFamilyPreferenceAtSamePriority(t *testing.T) {
	setupMixedFamilySamePriorityCache(t)

	// Requesting OpenAI family: should only return OpenAI-family channels.
	for i := 0; i < 20; i++ {
		ch, err := GetRandomSatisfiedChannel("default", "gpt-4", 0, common.RoutingFamilyOpenAI)
		require.NoError(t, err)
		require.NotNil(t, ch)
		assert.Equal(t, 30, ch.Id, "OpenAI family filter should select openai-mixed (id=30)")
	}

	// Requesting Anthropic family: should only return Anthropic-family channels.
	for i := 0; i < 20; i++ {
		ch, err := GetRandomSatisfiedChannel("default", "gpt-4", 0, common.RoutingFamilyAnthropic)
		require.NoError(t, err)
		require.NotNil(t, ch)
		assert.Equal(t, 31, ch.Id, "Anthropic family filter should select anthropic-mixed (id=31)")
	}

	// No family filter: can return either.
	ch, err := GetRandomSatisfiedChannel("default", "gpt-4", 0, common.RoutingFamilyNone)
	require.NoError(t, err)
	require.NotNil(t, ch)
	assert.Contains(t, []int{30, 31}, ch.Id, "no family filter should select any available channel")
}

func TestGetRandomSatisfiedChannel_CrossFamilyFallback(t *testing.T) {
	channelSyncLock.Lock()
	common.MemoryCacheEnabled = true

	priority0 := int64(0)
	weight10 := uint(10)

	// Only OpenAI-family channels available
	channelsIDM = map[int]*Channel{
		1: {
			Id:       1,
			Type:     constant.ChannelTypeOpenAI,
			Status:   common.ChannelStatusEnabled,
			Name:     "openai-ch",
			Priority: &priority0,
			Weight:   &weight10,
			Group:    "default",
			Models:   "gpt-4",
			Key:      "key-1",
		},
	}
	group2model2channels = map[string]map[string][]int{
		"default": {
			"gpt-4": {1},
		},
	}
	channelSyncLock.Unlock()

	// Request Anthropic family but only OpenAI channels exist — should fall back
	ch, err := GetRandomSatisfiedChannel("default", "gpt-4", 0, common.RoutingFamilyAnthropic)
	require.NoError(t, err)
	require.NotNil(t, ch)
	assert.Equal(t, 1, ch.Id, "cross-family fallback should return the only available channel")
}

func TestGetRandomSatisfiedChannel_NoFamilyFilter(t *testing.T) {
	setupFamilyTestCache(t)

	// RoutingFamilyNone should not filter at all — any channel can be returned
	for i := 0; i < 20; i++ {
		ch, err := GetRandomSatisfiedChannel("default", "gpt-4", 0, common.RoutingFamilyNone)
		require.NoError(t, err)
		require.NotNil(t, ch)
	}
}

func TestGetRandomSatisfiedChannel_PriorityPreservedWithFamilyFilter(t *testing.T) {
	channelSyncLock.Lock()
	common.MemoryCacheEnabled = true

	priorityHigh := int64(10)
	priorityLow := int64(0)
	weight10 := uint(10)

	// OpenAI channel at higher priority, Anthropic channel at lower priority
	channelsIDM = map[int]*Channel{
		1: {
			Id:       1,
			Type:     constant.ChannelTypeOpenAI,
			Status:   common.ChannelStatusEnabled,
			Name:     "openai-high",
			Priority: &priorityHigh,
			Weight:   &weight10,
			Group:    "default",
			Models:   "gpt-4",
			Key:      "key-1",
		},
		2: {
			Id:       2,
			Type:     constant.ChannelTypeAnthropic,
			Status:   common.ChannelStatusEnabled,
			Name:     "anthropic-low",
			Priority: &priorityLow,
			Weight:   &weight10,
			Group:    "default",
			Models:   "gpt-4",
			Key:      "key-2",
		},
	}
	group2model2channels = map[string]map[string][]int{
		"default": {
			"gpt-4": {1, 2},
		},
	}
	channelSyncLock.Unlock()

	// Request OpenAI family with retry=0 (highest priority) — should get high-priority OpenAI channel
	ch, err := GetRandomSatisfiedChannel("default", "gpt-4", 0, common.RoutingFamilyOpenAI)
	require.NoError(t, err)
	require.NotNil(t, ch)
	assert.Equal(t, 1, ch.Id, "should select same-family channel at highest priority")

	// Request Anthropic family with retry=0 — only low-priority Anthropic channel exists
	ch, err = GetRandomSatisfiedChannel("default", "gpt-4", 0, common.RoutingFamilyAnthropic)
	require.NoError(t, err)
	require.NotNil(t, ch)
	assert.Equal(t, 2, ch.Id, "should select same-family Anthropic channel")
}
