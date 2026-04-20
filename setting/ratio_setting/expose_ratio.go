package ratio_setting

import (
	"strconv"

	"github.com/QuantumNous/new-api/setting/operation_setting"
)

func SetExposeRatioEnabled(enabled bool) {
	cfg := operation_setting.GetOperationSystemSetting()
	if cfg != nil {
		cfg.ExposeRatioEnabled = enabled
	}
	operation_setting.UpdateOperationSystemSetting("expose_ratio_enabled", strconv.FormatBool(enabled))
}

func IsExposeRatioEnabled() bool {
	cfg := operation_setting.GetOperationSystemSetting()
	if cfg != nil {
		return cfg.ExposeRatioEnabled
	}
	return false
}
