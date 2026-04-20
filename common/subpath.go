package common

import (
	"fmt"
	"strings"
)

// RelaySubpaths holds the validated relay subpath prefixes.
// These prefixes are used for routing relay requests to alternative paths.
var RelaySubpaths []string

// ParseRelaySubpaths parses a comma-separated string of subpath prefixes,
// validates each one, and returns the normalized list.
// Returns an empty slice with nil error if raw is empty.
// Returns an error if any subpath is invalid.
func ParseRelaySubpaths(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return []string{}, nil
	}

	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		normalized, err := validateSubpath(trimmed)
		if err != nil {
			return nil, err
		}
		result = append(result, normalized)
	}

	return result, nil
}

func validateSubpath(prefix string) (string, error) {
	trimmed := strings.TrimSpace(prefix)

	if !strings.HasPrefix(trimmed, "/") {
		return "", fmt.Errorf("subpath must start with /: %q", prefix)
	}

	if strings.HasSuffix(trimmed, "/") {
		return "", fmt.Errorf("subpath %q has trailing slash", prefix)
	}

	if strings.Contains(trimmed, "//") {
		return "", fmt.Errorf("subpath %q contains double slash", prefix)
	}

	normalized := normalizeSubpath(trimmed)

	if normalized == "/" {
		return "", fmt.Errorf("subpath %q is not allowed: conflicts with root route", prefix)
	}

	return normalized, nil
}

func normalizeSubpath(prefix string) string {
	result := strings.TrimSpace(prefix)

	if !strings.HasPrefix(result, "/") {
		result = "/" + result
	}

	result = strings.TrimSuffix(result, "/")

	for strings.Contains(result, "//") {
		result = strings.ReplaceAll(result, "//", "/")
	}

	return result
}