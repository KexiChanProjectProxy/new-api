package network_setting

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/setting/config"
)

func TestUpdateAndSync_NoPanic(t *testing.T) {
	// UpdateAndSync must be safe to call even though it is a no-op.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("UpdateAndSync panicked: %v", r)
		}
	}()
	UpdateAndSync()
}

func TestNetworkSetting_Defaults(t *testing.T) {
	ns := GetNetworkSetting()

	if ns.ListenAddress != "" {
		t.Errorf("default ListenAddress = %q, want empty string", ns.ListenAddress)
	}
	if ns.ProxyURL != "" {
		t.Errorf("default ProxyURL = %q, want empty string", ns.ProxyURL)
	}
	if ns.ProxyEnabled {
		t.Errorf("default ProxyEnabled = %v, want false", ns.ProxyEnabled)
	}
	// Empty slice vs nil — both mean "trust none"
	if ns.TrustedProxies == nil {
		t.Errorf("default TrustedProxies = nil, want empty slice or non-nil empty slice")
	}
}

func TestNetworkSetting_UpdateConfigFromMap_ListenAddress(t *testing.T) {
	saved := map[string]string{}
	config.GlobalConfig.SaveToDB(func(key, value string) error {
		if strings.HasPrefix(key, "network_setting.") {
			saved[key] = value
		}
		return nil
	})
	t.Cleanup(func() {
		config.GlobalConfig.LoadFromDB(saved)
	})

	// Reset to defaults before test
	ns := GetNetworkSetting()
	ns.ListenAddress = ""
	ns.TrustedProxies = []string{}
	ns.ProxyURL = ""
	ns.ProxyEnabled = false

	err := config.UpdateConfigFromMap(ns, map[string]string{
		"listen_address": ":9000",
	})
	if err != nil {
		t.Fatalf("UpdateConfigFromMap failed: %v", err)
	}
	if ns.ListenAddress != ":9000" {
		t.Errorf("ListenAddress = %q, want %q", ns.ListenAddress, ":9000")
	}
}

func TestNetworkSetting_UpdateConfigFromMap_EmptyListenAddress(t *testing.T) {
	ns := GetNetworkSetting()
	ns.ListenAddress = ":9000" // set a non-empty value first

	err := config.UpdateConfigFromMap(ns, map[string]string{
		"listen_address": "",
	})
	if err != nil {
		t.Fatalf("UpdateConfigFromMap failed: %v", err)
	}
	if ns.ListenAddress != "" {
		t.Errorf("ListenAddress = %q, want empty string", ns.ListenAddress)
	}
}

func TestNetworkSetting_UpdateConfigFromMap_TrustedProxies(t *testing.T) {
	ns := GetNetworkSetting()
	ns.TrustedProxies = []string{}

	err := config.UpdateConfigFromMap(ns, map[string]string{
		"trusted_proxies": `["10.0.0.1", "10.0.0.2"]`,
	})
	if err != nil {
		t.Fatalf("UpdateConfigFromMap failed: %v", err)
	}
	if len(ns.TrustedProxies) != 2 {
		t.Fatalf("TrustedProxies length = %d, want 2", len(ns.TrustedProxies))
	}
	if ns.TrustedProxies[0] != "10.0.0.1" || ns.TrustedProxies[1] != "10.0.0.2" {
		t.Errorf("TrustedProxies = %v, want [10.0.0.1, 10.0.0.2]", ns.TrustedProxies)
	}
}

func TestNetworkSetting_UpdateConfigFromMap_EmptyTrustedProxies(t *testing.T) {
	ns := GetNetworkSetting()
	ns.TrustedProxies = []string{"10.0.0.1"} // set non-empty first

	err := config.UpdateConfigFromMap(ns, map[string]string{
		"trusted_proxies": `[]`,
	})
	if err != nil {
		t.Fatalf("UpdateConfigFromMap failed: %v", err)
	}
	if len(ns.TrustedProxies) != 0 {
		t.Errorf("TrustedProxies length = %d after setting to [], want 0", len(ns.TrustedProxies))
	}
}

func TestNetworkSetting_UpdateConfigFromMap_ProxyURL(t *testing.T) {
	ns := GetNetworkSetting()
	ns.ProxyURL = ""

	err := config.UpdateConfigFromMap(ns, map[string]string{
		"proxy_url": "http://127.0.0.1:7890",
	})
	if err != nil {
		t.Fatalf("UpdateConfigFromMap failed: %v", err)
	}
	if ns.ProxyURL != "http://127.0.0.1:7890" {
		t.Errorf("ProxyURL = %q, want %q", ns.ProxyURL, "http://127.0.0.1:7890")
	}
}

func TestNetworkSetting_UpdateConfigFromMap_ProxyEnabled(t *testing.T) {
	ns := GetNetworkSetting()
	ns.ProxyEnabled = false

	err := config.UpdateConfigFromMap(ns, map[string]string{
		"proxy_enabled": "true",
	})
	if err != nil {
		t.Fatalf("UpdateConfigFromMap failed: %v", err)
	}
	if !ns.ProxyEnabled {
		t.Errorf("ProxyEnabled = %v, want true", ns.ProxyEnabled)
	}
}

func TestNetworkSetting_UpdateConfigFromMap_ProxyEnabled_FalseExplicit(t *testing.T) {
	ns := GetNetworkSetting()
	ns.ProxyEnabled = true

	err := config.UpdateConfigFromMap(ns, map[string]string{
		"proxy_enabled": "false",
	})
	if err != nil {
		t.Fatalf("UpdateConfigFromMap failed: %v", err)
	}
	if ns.ProxyEnabled {
		t.Errorf("ProxyEnabled = %v, want false", ns.ProxyEnabled)
	}
}

func TestNetworkSetting_ConfigToMapRoundTrip(t *testing.T) {
	ns := GetNetworkSetting()
	ns.ListenAddress = ":8080"
	ns.TrustedProxies = []string{"192.168.1.1", "10.0.0.0/8"}
	ns.ProxyURL = "socks5://127.0.0.1:1080"
	ns.ProxyEnabled = true

	configMap, err := config.ConfigToMap(ns)
	if err != nil {
		t.Fatalf("ConfigToMap failed: %v", err)
	}

	// Verify keys are the expected json-tag names
	if _, ok := configMap["listen_address"]; !ok {
		t.Errorf("ConfigToMap missing key 'listen_address', got %v", configMap)
	}
	if _, ok := configMap["trusted_proxies"]; !ok {
		t.Errorf("ConfigToMap missing key 'trusted_proxies', got %v", configMap)
	}
	if _, ok := configMap["proxy_url"]; !ok {
		t.Errorf("ConfigToMap missing key 'proxy_url', got %v", configMap)
	}
	if _, ok := configMap["proxy_enabled"]; !ok {
		t.Errorf("ConfigToMap missing key 'proxy_enabled', got %v", configMap)
	}

	// Reset for next test
	ns2 := GetNetworkSetting()
	ns2.ListenAddress = ""
	ns2.TrustedProxies = []string{}
	ns2.ProxyURL = ""
	ns2.ProxyEnabled = false

	err = config.UpdateConfigFromMap(ns2, configMap)
	if err != nil {
		t.Fatalf("UpdateConfigFromMap round-trip failed: %v", err)
	}
	if ns2.ListenAddress != ":8080" {
		t.Errorf("round-trip ListenAddress = %q, want %q", ns2.ListenAddress, ":8080")
	}
	if len(ns2.TrustedProxies) != 2 {
		t.Errorf("round-trip TrustedProxies length = %d, want 2", len(ns2.TrustedProxies))
	}
	if ns2.ProxyURL != "socks5://127.0.0.1:1080" {
		t.Errorf("round-trip ProxyURL = %q, want %q", ns2.ProxyURL, "socks5://127.0.0.1:1080")
	}
	if !ns2.ProxyEnabled {
		t.Errorf("round-trip ProxyEnabled = %v, want true", ns2.ProxyEnabled)
	}
}
