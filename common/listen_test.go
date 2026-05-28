package common

import (
	"strings"
	"testing"
)

func TestValidateListenAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		// Valid addresses
		{name: "bare_port_with_colon", address: ":3000", wantErr: false},
		{name: "all_interfaces_ipv4", address: "0.0.0.0:3000", wantErr: false},
		{name: "all_interfaces_ipv6", address: "[::]:3000", wantErr: false},
		{name: "localhost_ipv4", address: "127.0.0.1:3000", wantErr: false},
		{name: "localhost_ipv6", address: "[::1]:3000", wantErr: false},
		{name: "hostname_with_port", address: "example.com:3000", wantErr: false},
		{name: "hostname_with_subdomain", address: "api.example.com:8080", wantErr: false},
		{name: "single_char_hostname", address: "a:3000", wantErr: false},
		{name: "localhost_standard_port", address: "localhost:80", wantErr: false},
		{name: "localhost_max_port", address: "localhost:65535", wantErr: false},
		{name: "empty_address", address: "", wantErr: true},
		// Invalid addresses
		{name: "no_port_ipv4", address: "127.0.0.1", wantErr: true},
		{name: "no_port_hostname", address: "example.com", wantErr: true},
		{name: "too_many_colons", address: "bad:addr:3000", wantErr: true},
		{name: "just_port_number", address: "3000", wantErr: true},
		{name: "empty_host_no_port", address: ":", wantErr: false}, // Go's net.SplitHostPort accepts ":" as valid (all interfaces, default port)
		{name: "invalid_ipv6_unclosed_bracket", address: "[::1:3000", wantErr: true},
		{name: "non_numeric_port", address: "localhost:abc", wantErr: true},
		{name: "port_out_of_range_high", address: "localhost:65536", wantErr: true},
		{name: "port_out_of_range_zero", address: "localhost:0", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateListenAddress(tt.address)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateListenAddress(%q) error = %v, wantErr %v", tt.address, err, tt.wantErr)
			}
		})
	}
}

func TestResolveBindAddress(t *testing.T) {
	tests := []struct {
		name      string
		custom    string
		portEnv   string
		portFlag  int
		wantAddr  string
		wantErr   bool
		errSubstr string
	}{
		// Empty custom and PORT → fall back to port flag
		{
			name:     "empty_custom_and_PORT_uses_flag",
			custom:   "",
			portEnv:  "",
			portFlag: 3000,
			wantAddr: ":3000",
			wantErr:  false,
		},
		// PORT env takes precedence over flag
		{
			name:     "PORT_env_takes_precedence",
			custom:   "",
			portEnv:  "8080",
			portFlag: 3000,
			wantAddr: ":8080",
			wantErr:  false,
		},
		// Custom takes precedence over PORT and flag
		{
			name:     "custom_takes_precedence",
			custom:   "0.0.0.0:9000",
			portEnv:  "8080",
			portFlag: 3000,
			wantAddr: "0.0.0.0:9000",
			wantErr:  false,
		},
		// Custom IPv6
		{
			name:     "custom_ipv6",
			custom:   "[::]:3000",
			portEnv:  "",
			portFlag: 3000,
			wantAddr: "[::]:3000",
			wantErr:  false,
		},
		// Custom hostname
		{
			name:     "custom_hostname",
			custom:   "example.com:3000",
			portEnv:  "",
			portFlag: 3000,
			wantAddr: "example.com:3000",
			wantErr:  false,
		},
		// Invalid custom address
		{
			name:      "invalid_custom_no_port",
			custom:    "127.0.0.1",
			portEnv:   "",
			portFlag:  3000,
			wantAddr:  "",
			wantErr:   true,
			errSubstr: "invalid custom listen address",
		},
		// Invalid PORT env
		{
			name:      "invalid_PORT_no_port",
			custom:    "",
			portEnv:   "example.com",
			portFlag:  3000,
			wantAddr:  "",
			wantErr:   true,
			errSubstr: "invalid PORT environment variable",
		},
		// Invalid PORT too many colons
		{
			name:      "invalid_PORT_too_many_colons",
			custom:    "",
			portEnv:   "bad:addr:3000",
			portFlag:  3000,
			wantAddr:  "",
			wantErr:   true,
			errSubstr: "invalid PORT environment variable",
		},
		// Empty custom with invalid PORT errors (PORT has higher precedence than flag)
		{
			name:      "empty_custom_invalid_PORT_errors",
			custom:    "",
			portEnv:   "invalid",
			portFlag:  3000,
			wantAddr:  "",
			wantErr:   true,
			errSubstr: "invalid PORT environment variable",
		},
		// Flag format: bare number becomes :port
		{
			name:     "flag_becomes_colon_port",
			custom:   "",
			portEnv:  "",
			portFlag: 6379,
			wantAddr: ":6379",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveBindAddress(tt.custom, tt.portEnv, tt.portFlag)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveBindAddress(%q, %q, %d) error = %v, wantErr %v",
					tt.custom, tt.portEnv, tt.portFlag, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.wantAddr {
				t.Errorf("ResolveBindAddress(%q, %q, %d) = %q, want %q",
					tt.custom, tt.portEnv, tt.portFlag, got, tt.wantAddr)
			}
			if tt.wantErr && tt.errSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("ResolveBindAddress(%q, %q, %d) error = %v, want error containing %q",
						tt.custom, tt.portEnv, tt.portFlag, err, tt.errSubstr)
				}
			}
		})
	}
}

func TestFormatBindAddressForDisplay(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		wantHost string
		wantPort string
	}{
		{name: "bare_port", address: ":3000", wantHost: "localhost", wantPort: "3000"},
		{name: "all_interfaces", address: "0.0.0.0:3000", wantHost: "0.0.0.0", wantPort: "3000"},
		{name: "all_interfaces_ipv6", address: "[::]:3000", wantHost: "::", wantPort: "3000"},
		{name: "localhost_ipv4", address: "127.0.0.1:3000", wantHost: "127.0.0.1", wantPort: "3000"},
		{name: "localhost_ipv6", address: "[::1]:3000", wantHost: "::1", wantPort: "3000"},
		{name: "hostname", address: "example.com:8080", wantHost: "example.com", wantPort: "8080"},
		{name: "empty_string", address: "", wantHost: "", wantPort: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHost, gotPort := FormatBindAddressForDisplay(tt.address)
			if gotHost != tt.wantHost || gotPort != tt.wantPort {
				t.Errorf("FormatBindAddressForDisplay(%q) = (%q, %q), want (%q, %q)",
					tt.address, gotHost, gotPort, tt.wantHost, tt.wantPort)
			}
		})
	}
}

func TestIsAnyAddress(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{host: "", want: true},
		{host: "0.0.0.0", want: true},
		{host: "::", want: true},
		{host: "[::]", want: true},
		{host: "localhost", want: false},
		{host: "127.0.0.1", want: false},
		{host: "::1", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			if got := IsAnyAddress(tt.host); got != tt.want {
				t.Errorf("IsAnyAddress(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}

func TestIsIPv6(t *testing.T) {
	tests := []struct {
		address string
		want    bool
	}{
		{address: "[::]:3000", want: true},
		{address: "[::1]:3000", want: true},
		{address: "0.0.0.0:3000", want: false},
		{address: "127.0.0.1:3000", want: false},
		{address: "example.com:3000", want: false},
		{address: ":3000", want: false},
		{address: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.address, func(t *testing.T) {
			if got := IsIPv6(tt.address); got != tt.want {
				t.Errorf("IsIPv6(%q) = %v, want %v", tt.address, got, tt.want)
			}
		})
	}
}

func TestResolveBindAddressPrecedence(t *testing.T) {
	portFlag := 3000

	// custom is set → should use custom
	addr, err := ResolveBindAddress("0.0.0.0:9000", "8080", portFlag)
	if err != nil || addr != "0.0.0.0:9000" {
		t.Errorf("custom should take precedence: got %q, err %v", addr, err)
	}

	// custom empty, PORT set → should use PORT
	addr, err = ResolveBindAddress("", "8080", portFlag)
	if err != nil || addr != ":8080" {
		t.Errorf("PORT should be used when custom is empty: got %q, err %v", addr, err)
	}

	// both empty → should use flag
	addr, err = ResolveBindAddress("", "", portFlag)
	if err != nil || addr != ":3000" {
		t.Errorf("flag should be used when both custom and PORT are empty: got %q, err %v", addr, err)
	}
}

func TestResolveBindAddressErrorMessages(t *testing.T) {
	// Verify error messages contain context about which source failed

	// Invalid custom
	_, err := ResolveBindAddress("bad:addr:3000", "", 3000)
	if err == nil || !strings.Contains(err.Error(), "invalid custom listen address") {
		t.Errorf("error should mention 'invalid custom listen address', got: %v", err)
	}

	// Invalid PORT
	_, err = ResolveBindAddress("", "novalid", 3000)
	if err == nil || !strings.Contains(err.Error(), "invalid PORT environment variable") {
		t.Errorf("error should mention 'invalid PORT environment variable', got: %v", err)
	}
}
