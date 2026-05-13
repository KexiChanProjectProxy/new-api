package dto

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestChannelOtherSettingsForwardClientIP_DefaultFalse(t *testing.T) {
	settings := ChannelOtherSettings{}
	require.False(t, settings.ForwardClientIP)
}

func TestChannelOtherSettingsForwardClientIP_MarshalOmitsWhenFalse(t *testing.T) {
	settings := ChannelOtherSettings{
		ForwardClientIP: false,
	}
	data, err := common.Marshal(settings)
	require.NoError(t, err)
	require.False(t, gjson.GetBytes(data, "forward_client_ip").Exists(), "forward_client_ip should be omitted when false")
}

func TestChannelOtherSettingsForwardClientIP_MarshalIncludesWhenTrue(t *testing.T) {
	settings := ChannelOtherSettings{
		ForwardClientIP: true,
	}
	data, err := common.Marshal(settings)
	require.NoError(t, err)
	require.True(t, gjson.GetBytes(data, "forward_client_ip").Exists())
	require.Equal(t, "true", gjson.GetBytes(data, "forward_client_ip").Raw)
}

func TestChannelOtherSettingsForwardClientIP_UnmarshalAbsenceYieldsFalse(t *testing.T) {
	raw := []byte(`{}`)
	var settings ChannelOtherSettings
	err := common.Unmarshal(raw, &settings)
	require.NoError(t, err)
	require.False(t, settings.ForwardClientIP)
}

func TestChannelOtherSettingsForwardClientIP_UnmarshalTrueYieldsTrue(t *testing.T) {
	raw := []byte(`{"forward_client_ip":true}`)
	var settings ChannelOtherSettings
	err := common.Unmarshal(raw, &settings)
	require.NoError(t, err)
	require.True(t, settings.ForwardClientIP)
}

func TestChannelOtherSettingsForwardClientIP_UnmarshalFalseYieldsFalse(t *testing.T) {
	raw := []byte(`{"forward_client_ip":false}`)
	var settings ChannelOtherSettings
	err := common.Unmarshal(raw, &settings)
	require.NoError(t, err)
	require.False(t, settings.ForwardClientIP)
}

func TestChannelOtherSettingsForwardClientIP_RoundtripTruePreservesTrue(t *testing.T) {
	settings := ChannelOtherSettings{
		ForwardClientIP: true,
	}
	data, err := common.Marshal(settings)
	require.NoError(t, err)
	var restored ChannelOtherSettings
	err = common.Unmarshal(data, &restored)
	require.NoError(t, err)
	require.True(t, restored.ForwardClientIP)
}

func TestChannelOtherSettingsForwardClientIP_RoundtripFalsePreservesFalse(t *testing.T) {
	settings := ChannelOtherSettings{
		ForwardClientIP: false,
	}
	data, err := common.Marshal(settings)
	require.NoError(t, err)
	var restored ChannelOtherSettings
	err = common.Unmarshal(data, &restored)
	require.NoError(t, err)
	require.False(t, restored.ForwardClientIP)
}