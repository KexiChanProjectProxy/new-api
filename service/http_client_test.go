package service

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/setting/network_setting"
)

func TestInitHttpClient_UsesNetworkSettingProxy(t *testing.T) {
	original := *network_setting.GetNetworkSetting()
	t.Cleanup(func() {
		*network_setting.GetNetworkSetting() = original
	})

	network_setting.GetNetworkSetting().ProxyEnabled = true
	network_setting.GetNetworkSetting().ProxyURL = "http://127.0.0.1:7890"

	err := InitHttpClient()
	if err != nil {
		t.Fatalf("InitHttpClient failed: %v", err)
	}

	transport, ok := GetHttpClient().Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", GetHttpClient().Transport)
	}
	if transport.Proxy == nil {
		t.Fatal("transport.Proxy is nil, want configured proxy")
	}
	req, err := http.NewRequest("GET", "https://example.com", nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	proxyURL, err := transport.Proxy(req)
	if err != nil {
		t.Fatalf("transport.Proxy returned error: %v", err)
	}
	if proxyURL == nil || proxyURL.String() != "http://127.0.0.1:7890" {
		t.Fatalf("proxyURL = %v, want http://127.0.0.1:7890", proxyURL)
	}
}

func TestInitHttpClient_DisabledProxyUsesEnvFallback(t *testing.T) {
	original := *network_setting.GetNetworkSetting()
	t.Cleanup(func() {
		*network_setting.GetNetworkSetting() = original
	})

	network_setting.GetNetworkSetting().ProxyEnabled = false
	network_setting.GetNetworkSetting().ProxyURL = "http://127.0.0.1:7890"

	err := InitHttpClient()
	if err != nil {
		t.Fatalf("InitHttpClient failed: %v", err)
	}

	transport, ok := GetHttpClient().Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", GetHttpClient().Transport)
	}
	// When ProxyEnabled=false, the transport should fall back to http.ProxyFromEnvironment
	// (respecting HTTP_PROXY/HTTPS_PROXY/NO_PROXY env vars) rather than being nil
	if transport.Proxy == nil {
		t.Fatal("transport.Proxy is nil, want http.ProxyFromEnvironment when proxy disabled")
	}
	// Verify that the Proxy function is http.ProxyFromEnvironment by calling it
	// (without env vars set, it should return nil URL = no proxy)
	req, err := http.NewRequest("GET", "https://example.com", nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	proxyURL, err := transport.Proxy(req)
	if err != nil {
		t.Fatalf("transport.Proxy returned error: %v", err)
	}
	// Without HTTP_PROXY/HTTPS_PROXY env vars, ProxyFromEnvironment returns nil
	if proxyURL != nil {
		t.Fatalf("proxyURL = %v, want nil (no env proxy configured)", proxyURL)
	}
}
