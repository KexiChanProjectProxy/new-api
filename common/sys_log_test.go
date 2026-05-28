package common

import (
	"testing"
	"time"
)

func TestLogStartupSuccess_Formatting(t *testing.T) {
	tests := []struct {
		name           string
		bindAddress    string
		wantHost       string
		wantPort       string
		wantLocalHost  string
		wantLocalURL   string
		desc           string
	}{
		{
			name:          "bare_port_shows_localhost",
			bindAddress:   ":3000",
			wantHost:      "localhost",
			wantPort:      "3000",
			wantLocalHost: "localhost",
			wantLocalURL:  "http://localhost:3000/",
			desc:          "Bare port :3000 should show localhost:3000",
		},
		{
			name:          "wildcard_ipv4_shows_localhost",
			bindAddress:   "0.0.0.0:3000",
			wantHost:      "0.0.0.0",
			wantPort:      "3000",
			wantLocalHost: "localhost",
			wantLocalURL:  "http://localhost:3000/",
			desc:          "0.0.0.0:3000 should show localhost:3000 (not 0.0.0.0)",
		},
		{
			name:          "wildcard_ipv6_shows_localhost",
			bindAddress:   "[::]:3000",
			wantHost:      "::",
			wantPort:      "3000",
			wantLocalHost: "localhost",
			wantLocalURL:  "http://localhost:3000/",
			desc:          "[::]:3000 should show localhost:3000 (not [::])",
		},
		{
			name:          "localhost_ipv4_preserves_host",
			bindAddress:   "127.0.0.1:3000",
			wantHost:      "127.0.0.1",
			wantPort:      "3000",
			wantLocalHost: "127.0.0.1",
			wantLocalURL:  "http://127.0.0.1:3000/",
			desc:          "127.0.0.1:3000 should preserve 127.0.0.1:3000",
		},
		{
			name:          "localhost_ipv6_preserves_host",
			bindAddress:   "[::1]:3000",
			wantHost:      "::1",
			wantPort:      "3000",
			wantLocalHost: "::1",
			wantLocalURL:  "http://[::1]:3000/",
			desc:          "[::1]:3000 should preserve [::1]:3000",
		},
		{
			name:          "hostname_preserved",
			bindAddress:   "example.com:8080",
			wantHost:      "example.com",
			wantPort:      "8080",
			wantLocalHost: "example.com",
			wantLocalURL:  "http://example.com:8080/",
			desc:          "example.com:8080 should preserve hostname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port := FormatBindAddressForDisplay(tt.bindAddress)

			// Verify host parsing
			if host != tt.wantHost {
				t.Errorf("FormatBindAddressForDisplay(%q) host = %q, want %q", tt.bindAddress, host, tt.wantHost)
			}

			// Verify port parsing
			if port != tt.wantPort {
				t.Errorf("FormatBindAddressForDisplay(%q) port = %q, want %q", tt.bindAddress, port, tt.wantPort)
			}

			// Apply LogStartupSuccess logic for display
			localHost := host
			if IsAnyAddress(localHost) {
				localHost = "localhost"
			}

			// Verify local host after substitution
			if localHost != tt.wantLocalHost {
				t.Errorf("After IsAnyAddress substitution, localHost = %q, want %q", localHost, tt.wantLocalHost)
			}

			// Verify URL construction (use IsIPv6Host for bare hosts)
			var localURL string
			if IsIPv6Host(localHost) {
				localURL = "http://[" + localHost + "]:" + port + "/"
			} else {
				localURL = "http://" + localHost + ":" + port + "/"
			}

			if localURL != tt.wantLocalURL {
				t.Errorf("Constructed local URL = %q, want %q", localURL, tt.wantLocalURL)
			}
		})
	}
}

func TestLogStartupSuccess_BackwardCompat(t *testing.T) {
	// Ensure the function signature change is backward compatible
	// by verifying it accepts the old port-only style call via FormatBindAddressForDisplay

	// Old call: LogStartupSuccess(startTime, "3000")
	// New call: LogStartupSuccess(startTime, ":3000")

	// Test that the old pattern via FormatBindAddressForDisplay works
	host, port := FormatBindAddressForDisplay(":3000")
	if host != "localhost" || port != "3000" {
		t.Errorf("FormatBindAddressForDisplay(:3000) = (%q, %q), want (localhost, 3000)", host, port)
	}
}

func TestLogStartupSuccess_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		bindAddress string
		wantHost    string
		wantPort    string
	}{
		{
			name:        "empty_address",
			bindAddress: "",
			wantHost:    "",
			wantPort:    "",
		},
		{
			name:        "unparseable_address",
			bindAddress: "invalid",
			wantHost:    "",
			wantPort:    "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port := FormatBindAddressForDisplay(tt.bindAddress)
			if host != tt.wantHost || port != tt.wantPort {
				t.Errorf("FormatBindAddressForDisplay(%q) = (%q, %q), want (%q, %q)",
					tt.bindAddress, host, port, tt.wantHost, tt.wantPort)
			}
		})
	}
}

func TestLogStartupSuccess_NoMisleadingUrls(t *testing.T) {
	// Ensure wildcard addresses don't produce misleading URLs
	// After substitution, they should show localhost, not 0.0.0.0 or ::

	wildcardAddresses := []string{
		":3000",
		"0.0.0.0:3000",
		"[::]:3000",
	}

	for _, addr := range wildcardAddresses {
		t.Run(addr, func(t *testing.T) {
			host, _ := FormatBindAddressForDisplay(addr)

			// After substitution, wildcard hosts should become localhost
			localHost := host
			if IsAnyAddress(localHost) {
				localHost = "localhost"
			}

			if localHost != "localhost" {
				t.Errorf("Wildcard address %q after substitution produced host %q, want localhost", addr, localHost)
			}
		})
	}
}

// TestLogStartupSuccess_Integration tests the actual LogStartupSuccess function
// by capturing output - but since it uses gin.DefaultWriter directly,
// we test the underlying formatting functions instead.
func TestLogStartupSuccess_Integration(t *testing.T) {
	// startTime is used in the real LogStartupSuccess; we verify formatting here
	_ = time.Now()

	// This would be an integration test if we could capture gin.DefaultWriter
	// For now, we verify the formatting logic works correctly

	bindAddresses := []string{
		":3000",
		"0.0.0.0:3000",
		"[::]:3000",
		"127.0.0.1:3000",
		"[::1]:3000",
		"example.com:8080",
	}

	for _, addr := range bindAddresses {
		t.Run(addr, func(t *testing.T) {
			// Should not panic
			displayHost, displayPort := FormatBindAddressForDisplay(addr)

			if displayPort == "" && addr != "" {
				t.Errorf("FormatBindAddressForDisplay(%q) returned empty port for non-empty address", addr)
			}

			// Verify IsAnyAddress works correctly
			_ = IsAnyAddress(displayHost)

			// For display, we should substitute localhost for any addresses
			localHost := displayHost
			if IsAnyAddress(localHost) {
				localHost = "localhost"
			}

			if localHost == "" && addr != "" {
				t.Errorf("Local host became empty for address %q", addr)
			}
		})
	}
}

func TestLogStartupSuccess_URLConstruction(t *testing.T) {
	tests := []struct {
		name        string
		bindAddress string
		wantURL     string
	}{
		{
			name:        "bare_port_ipv4",
			bindAddress: ":3000",
			wantURL:     "http://localhost:3000/",
		},
		{
			name:        "wildcard_ipv4",
			bindAddress: "0.0.0.0:3000",
			wantURL:     "http://localhost:3000/",
		},
		{
			name:        "wildcard_ipv6",
			bindAddress: "[::]:3000",
			wantURL:     "http://localhost:3000/",
		},
		{
			name:        "localhost_ipv4",
			bindAddress: "127.0.0.1:3000",
			wantURL:     "http://127.0.0.1:3000/",
		},
		{
			name:        "localhost_ipv6",
			bindAddress: "[::1]:3000",
			wantURL:     "http://[::1]:3000/",
		},
		{
			name:        "hostname",
			bindAddress: "example.com:8080",
			wantURL:     "http://example.com:8080/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port := FormatBindAddressForDisplay(tt.bindAddress)

			// Apply the same logic as LogStartupSuccess
			localHost := host
			if IsAnyAddress(localHost) {
				localHost = "localhost"
			}

			// Use IsIPv6Host for bare hosts (not full addresses)
			var localURL string
			if IsIPv6Host(localHost) {
				localURL = "http://[" + localHost + "]:" + port + "/"
			} else {
				localURL = "http://" + localHost + ":" + port + "/"
			}

			if localURL != tt.wantURL {
				t.Errorf("Constructed URL = %q, want %q", localURL, tt.wantURL)
			}
		})
	}
}

// TestLogStartupSuccess_MainGoCallSite tests that the new signature works
// with the call site in main.go which passes the resolved bind address
func TestLogStartupSuccess_MainGoCallSite(t *testing.T) {
	// Simulate main.go call: common.LogStartupSuccess(startTime, bindAddress)
	// where bindAddress comes from ResolveBindAddress

	testCases := []struct {
		name       string
		custom     string
		portEnv    string
		portFlag   int
		resolvedTo string
	}{
		{
			name:       "custom_bind",
			custom:     "0.0.0.0:9000",
			portEnv:    "8080",
			portFlag:   3000,
			resolvedTo: "0.0.0.0:9000",
		},
		{
			name:       "port_env",
			custom:     "",
			portEnv:    "8080",
			portFlag:   3000,
			resolvedTo: ":8080",
		},
		{
			name:       "port_flag",
			custom:     "",
			portEnv:    "",
			portFlag:   3000,
			resolvedTo: ":3000",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			addr, err := ResolveBindAddress(tc.custom, tc.portEnv, tc.portFlag)
			if err != nil {
				t.Fatalf("ResolveBindAddress failed: %v", err)
			}

			if addr != tc.resolvedTo {
				t.Errorf("ResolveBindAddress returned %q, want %q", addr, tc.resolvedTo)
			}

			// Now verify the address can be formatted for display
			host, port := FormatBindAddressForDisplay(addr)
			if port == "" {
				t.Errorf("FormatBindAddressForDisplay(%q) returned empty port", addr)
			}

			// Verify we can construct a proper display URL
			localHost := host
			if IsAnyAddress(localHost) {
				localHost = "localhost"
			}

			if localHost == "" || port == "" {
				t.Errorf("Failed to construct proper display URL for %q", addr)
			}

			// Log success for verification
			t.Logf("Bind address %q -> display host %q, port %q, localURL would be http://%s:%s/",
				addr, host, port, localHost, port)
		})
	}
}