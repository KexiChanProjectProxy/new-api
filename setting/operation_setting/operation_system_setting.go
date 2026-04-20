package operation_setting

import (
	"github.com/QuantumNous/new-api/setting/config"
)

type OperationSystemSetting struct {
	ExposeRatioEnabled bool `json:"expose_ratio_enabled"`
}

var operationSystemSetting = OperationSystemSetting{
	ExposeRatioEnabled: false,
}

func init() {
	config.GlobalConfig.Register("operation_system_setting", &operationSystemSetting)
}

func GetOperationSystemSetting() *OperationSystemSetting {
	return &operationSystemSetting
}

func UpdateOperationSystemSetting(field string, value string) {
	cfg := config.GlobalConfig.Get("operation_system_setting")
	if cfg == nil {
		return
	}
	configMap := map[string]string{field: value}
	config.UpdateConfigFromMap(cfg, configMap)
}