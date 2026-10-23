package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeSiteMode(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "empty falls back to beta", value: "", want: SiteModeBeta},
		{name: "beta stays beta", value: "beta", want: SiteModeBeta},
		{name: "beta_plus is kept", value: "beta_plus", want: SiteModeBetaPlus},
		{name: "release is kept", value: "release", want: SiteModeRelease},
		{name: "unknown falls back to beta", value: "gamma", want: SiteModeBeta},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NormalizeSiteMode(tt.value))
		})
	}
}

func TestGetSiteModeReadsOptionValue(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	original, existed := common.OptionMap[SiteModeOptionKey]
	common.OptionMap[SiteModeOptionKey] = SiteModeBetaPlus
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		defer common.OptionMapRWMutex.Unlock()
		if existed {
			common.OptionMap[SiteModeOptionKey] = original
		} else {
			delete(common.OptionMap, SiteModeOptionKey)
		}
	})

	assert.Equal(t, SiteModeBetaPlus, GetSiteMode())

	common.OptionMapRWMutex.Lock()
	common.OptionMap[SiteModeOptionKey] = "not-a-mode"
	common.OptionMapRWMutex.Unlock()
	assert.Equal(t, SiteModeBeta, GetSiteMode())
}

func TestValidateSiteModeOption(t *testing.T) {
	for _, value := range []string{SiteModeBeta, SiteModeBetaPlus, SiteModeRelease} {
		assert.NoError(t, validateOptionValue(SiteModeOptionKey, value))
	}
	assert.Error(t, validateOptionValue(SiteModeOptionKey, "gamma"))
	assert.Error(t, validateOptionValue(SiteModeOptionKey, ""))
}

func TestGetConnectedModels(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.Exec("DELETE FROM abilities").Error)
	require.NoError(t, DB.Exec("DELETE FROM channels").Error)

	enabledChannel := Channel{Name: "enabled", Key: "k1", Status: common.ChannelStatusEnabled}
	require.NoError(t, DB.Create(&enabledChannel).Error)
	disabledChannel := Channel{Name: "disabled", Key: "k2", Status: common.ChannelStatusManuallyDisabled}
	require.NoError(t, DB.Create(&disabledChannel).Error)

	abilities := []Ability{
		{Group: "default", Model: "model-b", ChannelId: enabledChannel.Id, Enabled: true},
		{Group: "vip", Model: "model-a", ChannelId: enabledChannel.Id, Enabled: true},
		{Group: "default", Model: "model-a", ChannelId: enabledChannel.Id, Enabled: true},
		{Group: "default", Model: "model-on-disabled-channel", ChannelId: disabledChannel.Id, Enabled: true},
		{Group: "default", Model: "model-ability-off", ChannelId: enabledChannel.Id, Enabled: false},
	}
	for i := range abilities {
		require.NoError(t, DB.Create(&abilities[i]).Error)
	}

	assert.Equal(t, []string{"model-a", "model-b"}, GetConnectedModels())
}

func TestGetConnectedModelsEmpty(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.Exec("DELETE FROM abilities").Error)
	require.NoError(t, DB.Exec("DELETE FROM channels").Error)

	assert.Equal(t, []string{}, GetConnectedModels())
}
