package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func resetPassthroughFormatFixture(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&Ability{}).Error)
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&Channel{}).Error)
}

func insertPassthroughFormatCandidate(t *testing.T, channelType int, channelPassThrough bool) {
	t.Helper()
	const (
		channelID = 1
		group     = "default"
		modelName = "passthrough-format-model"
	)

	setting, err := common.Marshal(dto.ChannelSettings{PassThroughBodyEnabled: channelPassThrough})
	require.NoError(t, err)
	settingValue := string(setting)
	priority := int64(0)
	weight := uint(0)

	require.NoError(t, DB.Create(&Channel{
		Id:       channelID,
		Type:     channelType,
		Key:      "passthrough-format-key",
		Status:   common.ChannelStatusEnabled,
		Name:     "passthrough-format-channel",
		Weight:   &weight,
		Priority: &priority,
		Models:   modelName,
		Group:    group,
		Setting:  &settingValue,
	}).Error)
	require.NoError(t, DB.Create(&Ability{
		Group:     group,
		Model:     modelName,
		ChannelId: channelID,
		Enabled:   true,
		Priority:  &priority,
		Weight:    weight,
	}).Error)
}

func TestPassthroughFormatSelectors(t *testing.T) {
	globalSettings := model_setting.GetGlobalSettings()
	originalSettings := *globalSettings
	t.Cleanup(func() {
		*globalSettings = originalSettings
		common.MemoryCacheEnabled = false
	})
	globalSettings.PassThroughRequestEnabled = false

	tests := []struct {
		name               string
		channelType        int
		channelPassThrough bool
		globalPassThrough  bool
		relayFormat        types.RelayFormat
		wantSelectable     bool
	}{
		{
			name:               "passthrough chat accepts chat",
			channelType:        constant.ChannelTypeOpenAI,
			channelPassThrough: true,
			relayFormat:        types.RelayFormatOpenAI,
			wantSelectable:     true,
		},
		{
			name:               "passthrough chat accepts audio",
			channelType:        constant.ChannelTypeOpenAI,
			channelPassThrough: true,
			relayFormat:        types.RelayFormatOpenAIAudio,
			wantSelectable:     true,
		},
		{
			name:               "passthrough chat accepts image",
			channelType:        constant.ChannelTypeOpenAI,
			channelPassThrough: true,
			relayFormat:        types.RelayFormatOpenAIImage,
			wantSelectable:     true,
		},
		{
			name:               "passthrough chat accepts embedding",
			channelType:        constant.ChannelTypeOpenAI,
			channelPassThrough: true,
			relayFormat:        types.RelayFormatEmbedding,
			wantSelectable:     true,
		},
		{
			name:               "passthrough chat rejects responses",
			channelType:        constant.ChannelTypeOpenAI,
			channelPassThrough: true,
			relayFormat:        types.RelayFormatOpenAIResponses,
			wantSelectable:     false,
		},
		{
			name:               "passthrough chat rejects compaction",
			channelType:        constant.ChannelTypeOpenAI,
			channelPassThrough: true,
			relayFormat:        types.RelayFormatOpenAIResponsesCompaction,
			wantSelectable:     false,
		},
		{
			name:               "passthrough responses accepts responses",
			channelType:        constant.ChannelTypeOpenAIResponses,
			channelPassThrough: true,
			relayFormat:        types.RelayFormatOpenAIResponses,
			wantSelectable:     true,
		},
		{
			name:               "passthrough responses accepts compaction",
			channelType:        constant.ChannelTypeOpenAIResponses,
			channelPassThrough: true,
			relayFormat:        types.RelayFormatOpenAIResponsesCompaction,
			wantSelectable:     true,
		},
		{
			name:               "passthrough responses rejects chat",
			channelType:        constant.ChannelTypeOpenAIResponses,
			channelPassThrough: true,
			relayFormat:        types.RelayFormatOpenAI,
			wantSelectable:     false,
		},
		{
			name:           "non passthrough chat accepts responses",
			channelType:    constant.ChannelTypeOpenAI,
			relayFormat:    types.RelayFormatOpenAIResponses,
			wantSelectable: true,
		},
		{
			name:           "non passthrough chat accepts chat",
			channelType:    constant.ChannelTypeOpenAI,
			relayFormat:    types.RelayFormatOpenAI,
			wantSelectable: true,
		},
		{
			name:           "non passthrough chat accepts compaction",
			channelType:    constant.ChannelTypeOpenAI,
			relayFormat:    types.RelayFormatOpenAIResponsesCompaction,
			wantSelectable: true,
		},
		{
			name:           "non passthrough responses accepts chat",
			channelType:    constant.ChannelTypeOpenAIResponses,
			relayFormat:    types.RelayFormatOpenAI,
			wantSelectable: true,
		},
		{
			name:           "non passthrough responses accepts responses",
			channelType:    constant.ChannelTypeOpenAIResponses,
			relayFormat:    types.RelayFormatOpenAIResponses,
			wantSelectable: true,
		},
		{
			name:           "non passthrough responses accepts compaction",
			channelType:    constant.ChannelTypeOpenAIResponses,
			relayFormat:    types.RelayFormatOpenAIResponsesCompaction,
			wantSelectable: true,
		},
		{
			name:              "global passthrough chat rejects responses",
			channelType:       constant.ChannelTypeOpenAI,
			globalPassThrough: true,
			relayFormat:       types.RelayFormatOpenAIResponses,
			wantSelectable:    false,
		},
		{
			name:              "global passthrough responses rejects chat",
			channelType:       constant.ChannelTypeOpenAIResponses,
			globalPassThrough: true,
			relayFormat:       types.RelayFormatOpenAI,
			wantSelectable:    false,
		},
		{
			name:               "empty format remains unfiltered",
			channelType:        constant.ChannelTypeOpenAIResponses,
			channelPassThrough: true,
			relayFormat:        "",
			wantSelectable:     true,
		},
	}

	for _, cacheEnabled := range []bool{true, false} {
		mode := "database"
		if cacheEnabled {
			mode = "memory cache"
		}
		t.Run(mode, func(t *testing.T) {
			common.MemoryCacheEnabled = cacheEnabled
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					resetPassthroughFormatFixture(t)
					globalSettings.PassThroughRequestEnabled = test.globalPassThrough
					insertPassthroughFormatCandidate(t, test.channelType, test.channelPassThrough)
					if cacheEnabled {
						InitChannelCache()
					}

					var (
						channel *Channel
						err     error
					)
					if cacheEnabled {
						channel, err = GetRandomSatisfiedChannel("default", "passthrough-format-model", 0, test.relayFormat)
					} else {
						channel, err = GetChannel("default", "passthrough-format-model", 0, test.relayFormat)
					}
					require.NoError(t, err)
					if test.wantSelectable {
						require.NotNil(t, channel)
						require.Equal(t, 1, channel.Id)
					} else {
						require.Nil(t, channel)
					}
				})
			}
		})
	}
}

func TestPassthroughFormatCompatibilityMatrix(t *testing.T) {
	tests := []struct {
		name           string
		channelType    int
		passthrough    bool
		relayFormat    types.RelayFormat
		wantCompatible bool
	}{
		{
			name:           "chat passthrough chat",
			channelType:    constant.ChannelTypeOpenAI,
			passthrough:    true,
			relayFormat:    types.RelayFormatOpenAI,
			wantCompatible: true,
		},
		{
			name:           "chat passthrough audio",
			channelType:    constant.ChannelTypeOpenAI,
			passthrough:    true,
			relayFormat:    types.RelayFormatOpenAIAudio,
			wantCompatible: true,
		},
		{
			name:           "chat passthrough image",
			channelType:    constant.ChannelTypeOpenAI,
			passthrough:    true,
			relayFormat:    types.RelayFormatOpenAIImage,
			wantCompatible: true,
		},
		{
			name:           "chat passthrough embedding",
			channelType:    constant.ChannelTypeOpenAI,
			passthrough:    true,
			relayFormat:    types.RelayFormatEmbedding,
			wantCompatible: true,
		},
		{
			name:           "chat passthrough responses",
			channelType:    constant.ChannelTypeOpenAI,
			passthrough:    true,
			relayFormat:    types.RelayFormatOpenAIResponses,
			wantCompatible: false,
		},
		{
			name:           "chat passthrough compaction",
			channelType:    constant.ChannelTypeOpenAI,
			passthrough:    true,
			relayFormat:    types.RelayFormatOpenAIResponsesCompaction,
			wantCompatible: false,
		},
		{
			name:           "responses passthrough responses",
			channelType:    constant.ChannelTypeOpenAIResponses,
			passthrough:    true,
			relayFormat:    types.RelayFormatOpenAIResponses,
			wantCompatible: true,
		},
		{
			name:           "responses passthrough compaction",
			channelType:    constant.ChannelTypeOpenAIResponses,
			passthrough:    true,
			relayFormat:    types.RelayFormatOpenAIResponsesCompaction,
			wantCompatible: true,
		},
		{
			name:           "disabled passthrough leaves matrix unfiltered",
			channelType:    constant.ChannelTypeOpenAIResponses,
			passthrough:    false,
			relayFormat:    types.RelayFormatOpenAI,
			wantCompatible: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.wantCompatible, IsPassthroughFormatCompatible(test.channelType, test.passthrough, test.relayFormat), fmt.Sprintf("%s compatibility", test.name))
		})
	}
}
