// Package network_setting provides main-server network controls including listen address,
// trusted reverse proxy list, and outbound proxy configuration.
//
// NOTE: Changes to ListenAddress and TrustedProxies are RESTART-REQUIRED.
// This module does NOT hot-reconfigure the active Gin engine or TCP listener at runtime.
// Empty ListenAddress means "use existing fallback". Empty TrustedProxies means "trust none"
// (SetTrustedProxies(nil)), NOT trust all.
package network_setting

import (
	"github.com/QuantumNous/new-api/setting/config"
)

// NetworkSetting holds main-server network control parameters.
// These values are persisted to the database and loaded on startup.
// Runtime changes require a server restart to take effect.
type NetworkSetting struct {
	// ListenAddress is the full main server listen address (e.g., ":3000", "0.0.0.0:8080").
	// Empty string means "use existing fallback" (do not override any previously configured bind).
	// This is NOT the public callback URL — see system_setting.ServerAddress for that.
	ListenAddress string `json:"listen_address"`

	// TrustedProxies is a list of trusted reverse proxy IP addresses or CIDR ranges.
	// When empty, no proxies are trusted (Gin's SetTrustedProxies(nil) is used).
	// This controls X-Forwarded-For / X-Real-IP header trust in Gin.
	TrustedProxies []string `json:"trusted_proxies"`

	ProxyURL string `json:"proxy_url"`

	ProxyEnabled bool `json:"proxy_enabled"`
}

// defaultNetworkSetting holds the compiled default values.
var defaultNetworkSetting = NetworkSetting{
	ListenAddress:  "",         // empty = use existing fallback
	TrustedProxies: []string{}, // empty = trust none (not trust all)
	ProxyURL:       "",
	ProxyEnabled:   false,
}

// networkSetting is the module-level singleton used by the config manager.
var networkSetting = defaultNetworkSetting

func init() {
	config.GlobalConfig.Register("network_setting", &networkSetting)
}

// GetNetworkSetting returns a pointer to the current network setting values.
func GetNetworkSetting() *NetworkSetting {
	return &networkSetting
}

func GetProxyURL() string {
	return networkSetting.ProxyURL
}

func IsProxyEnabled() bool {
	return networkSetting.ProxyEnabled
}

// UpdateAndSync is a no-op placeholder for post-load hook consistency.
// Unlike performance_setting or theme, network_setting changes are NOT
// applied at runtime — they require a server restart. This function exists
// to satisfy any generic post-load hook patterns but has no effect on the
// running server.
func UpdateAndSync() {
	// No-op: network settings require restart to take effect.
	// This function exists for interface consistency with other setting modules.
}
