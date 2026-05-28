package common

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// ValidateListenAddress checks if the given address string is a valid listen
// address format accepted by net.Listen. Returns nil if valid, otherwise an
// error describing why validation failed.
//
// Valid forms: :3000, 0.0.0.0:3000, [::]:3000, hostname:port
// Invalid forms: 127.0.0.1 (no port), bad:addr:3000 (too many colons)
func ValidateListenAddress(address string) error {
	if address == "" {
		return fmt.Errorf("listen address cannot be empty")
	}
	// Try to split host:port — this validates the format is parseable.
	// For plain port like "3000", this will fail, which is correct (we require host:port).
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("invalid listen address %q: %w", address, err)
	}
	// Validate port if non-empty (empty port is acceptable — Go uses default)
	if portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid listen address %q: port %q is not a valid number", address, portStr)
		}
		if port < 1 || port > 65535 {
			return fmt.Errorf("invalid listen address %q: port %d is out of range (1-65535)", address, port)
		}
	}
	_ = host // host is not needed for validation
	return nil
}

// ResolveBindAddress resolves the server bind target by precedence:
//  1. custom (highest priority) — a non-empty custom listen address
//  2. portEnv — the PORT environment variable value (bare port number, e.g. "8080")
//  3. portFlag (lowest priority) — the -port CLI flag default value
//
// All three inputs are raw values; ResolveBindAddress applies no environment
// or flag reading itself — callers pass whatever values they have.
//
// If custom is non-empty it is validated and returned verbatim.
// If portEnv is non-empty it is treated as a bare port number and returned as ":port".
// Otherwise portFlag is formatted as a string (":port") and returned.
//
// A non-nil error is returned when the selected address fails validation.
// The returned address is ready to pass to server.Run (e.g. Gin.Run(addr)).
func ResolveBindAddress(custom, portEnv string, portFlag int) (string, error) {
	// Precedence 1: custom listen address
	if custom != "" {
		if err := ValidateListenAddress(custom); err != nil {
			return "", fmt.Errorf("invalid custom listen address %q: %w", custom, err)
		}
		return custom, nil
	}

	// Precedence 2: PORT env var (treated as bare port number)
	if portEnv != "" {
		// PORT is always a bare port number — validate it's a valid port
		port, err := strconv.Atoi(portEnv)
		if err != nil {
			return "", fmt.Errorf("invalid PORT environment variable %q: not a valid port number: %w", portEnv, err)
		}
		if port < 1 || port > 65535 {
			return "", fmt.Errorf("invalid PORT environment variable %q: port out of range (1-65535)", portEnv)
		}
		return ":" + portEnv, nil
	}

	// Precedence 3: -port flag default
	return fmt.Sprintf(":%d", portFlag), nil
}

// FormatBindAddressForDisplay splits a bind address into host and port
// components suitable for startup logging (e.g. constructing http:// URLs).
//
// For addresses like ":3000", "0.0.0.0:3000", "[::]:3000", "example.com:3000":
//   - host = "localhost", "0.0.0.0", "::", or "example.com"
//   - port = "3000"
//
// For bare ports like ":3000" the host is returned as "localhost" for display.
// The returned port is always a string; the returned host may be empty for
// bare-port forms where the caller should substitute "localhost".
func FormatBindAddressForDisplay(address string) (host, port string) {
	if address == "" {
		return "", ""
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		// Fallback: assume entire string is a port (backwards compat for bare ":3000")
		// net.SplitHostPort(":3000") returns ("", "3000", nil), so this path
		// is only hit for genuinely unparseable strings.
		return "", address
	}
	if host == "" {
		// Bare port like ":3000" — host is empty per net.SplitHostPort.
		// For display, callers typically substitute "localhost".
		host = "localhost"
	}
	return host, port
}

// MustResolveBindAddress is like ResolveBindAddress but panics with a FatalLog
// if validation fails. Use this at startup when the address MUST be valid.
func MustResolveBindAddress(custom, portEnv string, portFlag int) string {
	addr, err := ResolveBindAddress(custom, portEnv, portFlag)
	if err != nil {
		FatalLog("listen address resolution failed: " + err.Error())
	}
	return addr
}

// IsAnyAddress returns true if host is a wildcard bind address
// (empty/unspecified, 0.0.0.0, or ::).
func IsAnyAddress(host string) bool {
	return host == "" || host == "0.0.0.0" || host == "::" || host == "[::]"
}

// IsIPv6 returns true if the host portion of the address is an IPv6 address.
// IsIPv6Host returns true if the given bare host string is an IPv6 address.
// Unlike IsIPv6, this function handles hosts without ports (e.g. "::1", "fe80::1").
func IsIPv6Host(host string) bool {
	if host == "" {
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.To4() == nil
}
// Handles both bracketed [::] and plain :: forms.
func IsIPv6(address string) bool {
	if address == "" {
		return false
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return false
	}
	// Check for IPv6 patterns
	return strings.Contains(host, ":")
}
