package common

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadJsonConfigFile(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantConfig JsonConfig
		wantErr    bool
	}{
		{
			name:       "valid JSON with single key-value",
			content:    `{"KEY1": "value1"}`,
			wantConfig: JsonConfig{"KEY1": "value1"},
			wantErr:    false,
		},
		{
			name:       "valid JSON with multiple key-values",
			content:    `{"KEY1": "value1", "KEY2": "value2", "KEY3": "value3"}`,
			wantConfig: JsonConfig{"KEY1": "value1", "KEY2": "value2", "KEY3": "value3"},
			wantErr:    false,
		},
		{
			name:       "valid JSON with empty object",
			content:    `{}`,
			wantConfig: JsonConfig{},
			wantErr:    false,
		},
		{
			name:    "invalid JSON - not an object",
			content: `["array", "not", "object"]`,
			wantErr: true,
		},
		{
			name:    "invalid JSON - malformed",
			content: `{invalid json`,
			wantErr: true,
		},
		{
			name:    "invalid JSON - number as value",
			content: `{"KEY": 123}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON - boolean as value",
			content: `{"KEY": true}`,
			wantErr: true,
		},
		{
			name:       "valid JSON with null value (becomes empty string)",
			content:    `{"KEY": null}`,
			wantConfig: JsonConfig{"KEY": ""},
			wantErr:    false,
		},
		{
			name:    "invalid JSON - nested object",
			content: `{"KEY": {"nested": "value"}}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.json")

			if tt.content != "" {
				if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
					t.Fatalf("failed to write temp config file: %v", err)
				}
			}

			config, err := LoadJsonConfigFile(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadJsonConfigFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(config) != len(tt.wantConfig) {
					t.Errorf("LoadJsonConfigFile() got %d keys, want %d keys", len(config), len(tt.wantConfig))
					return
				}
				for key, wantVal := range tt.wantConfig {
					if gotVal, ok := config[key]; !ok || gotVal != wantVal {
						t.Errorf("LoadJsonConfigFile()[%q] = %q, want %q", key, gotVal, wantVal)
					}
				}
			}
		})
	}
}

func TestLoadJsonConfigFile_MissingFile(t *testing.T) {
	_, err := LoadJsonConfigFile("/nonexistent/path/to/config.json")
	if err == nil {
		t.Error("LoadJsonConfigFile() expected error for missing file, got nil")
	}
}

func TestJsonConfig_ApplyToEnv(t *testing.T) {
	tests := []struct {
		name          string
		config        JsonConfig
		preExistingEnv map[string]string
		expectedEnv   map[string]string
	}{
		{
			name:   "no existing env vars",
			config: JsonConfig{"KEY1": "value1", "KEY2": "value2"},
			preExistingEnv: map[string]string{},
			expectedEnv: map[string]string{"KEY1": "value1", "KEY2": "value2"},
		},
		{
			name:   "some existing env vars",
			config: JsonConfig{"KEY1": "value1", "KEY2": "value2"},
			preExistingEnv: map[string]string{"KEY1": "existing_value1"},
			expectedEnv: map[string]string{"KEY1": "existing_value1", "KEY2": "value2"},
		},
		{
			name:   "all env vars already exist",
			config: JsonConfig{"KEY1": "value1", "KEY2": "value2"},
			preExistingEnv: map[string]string{"KEY1": "existing1", "KEY2": "existing2"},
			expectedEnv: map[string]string{"KEY1": "existing1", "KEY2": "existing2"},
		},
		{
			name:   "empty config",
			config: JsonConfig{},
			preExistingEnv: map[string]string{},
			expectedEnv: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for key := range tt.preExistingEnv {
				os.Setenv(key, tt.preExistingEnv[key])
				defer os.Unsetenv(key)
			}
			for key := range tt.config {
				defer os.Unsetenv(key)
			}

			tt.config.ApplyToEnv()

			for key, expectedVal := range tt.expectedEnv {
				if gotVal := os.Getenv(key); gotVal != expectedVal {
					t.Errorf("ApplyToEnv() env[%q] = %q, want %q", key, gotVal, expectedVal)
				}
			}
		})
	}
}

func TestGetJsonConfigPath(t *testing.T) {
	tests := []struct {
		name          string
		flagValue     *string
		envValue      string
		defaultExists bool
		want          string
	}{
		{
			name:          "flag takes precedence",
			flagValue:     stringPtr("/path/from/flag"),
			envValue:      "/path/from/env",
			defaultExists: true,
			want:          "/path/from/flag",
		},
		{
			name:          "env takes precedence over default when flag is empty",
			flagValue:     stringPtr(""),
			envValue:      "/path/from/env",
			defaultExists: true,
			want:          "/path/from/env",
		},
		{
			name:          "env takes precedence over default when flag is nil",
			flagValue:     nil,
			envValue:      "/path/from/env",
			defaultExists: true,
			want:          "/path/from/env",
		},
		{
			name:          "default used when no flag and no env",
			flagValue:     nil,
			envValue:      "",
			defaultExists: true,
			want:          "config.json",
		},
		{
			name:          "empty when no flag, no env, and default does not exist",
			flagValue:     nil,
			envValue:      "",
			defaultExists: false,
			want:          "",
		},
		{
			name:          "empty when flag is empty and env is empty and default does not exist",
			flagValue:     stringPtr(""),
			envValue:      "",
			defaultExists: false,
			want:          "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalFlagValue := ConfigFile
			originalEnvValue := os.Getenv("CONFIG_FILE")

			if tt.flagValue != nil {
				ConfigFile = tt.flagValue
			} else {
				ConfigFile = nil
			}
			if tt.envValue != "" {
				os.Setenv("CONFIG_FILE", tt.envValue)
				defer os.Unsetenv("CONFIG_FILE")
			} else {
				os.Unsetenv("CONFIG_FILE")
			}

			if tt.defaultExists {
				origWd, _ := os.Getwd()
				tmpDir := t.TempDir()
				os.Chdir(tmpDir)
				configPath := filepath.Join(tmpDir, "config.json")
				os.WriteFile(configPath, []byte(`{}`), 0644)
				defer os.Chdir(origWd)
				_ = configPath
			}

			got := GetJsonConfigPath()
			if got != tt.want {
				t.Errorf("GetJsonConfigPath() = %q, want %q", got, tt.want)
			}

			ConfigFile = originalFlagValue
			if originalEnvValue != "" {
				os.Setenv("CONFIG_FILE", originalEnvValue)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}

// TestJsonConfigPrecedence_EnvOverJson tests that environment variables
// take precedence over JSON config values (env > json).
func TestJsonConfigPrecedence_EnvOverJson(t *testing.T) {
	// Set env var BEFORE loading JSON config
	os.Setenv("TEST_JSONCONFIG_KEY", "from-env")
	t.Cleanup(func() {
		os.Unsetenv("TEST_JSONCONFIG_KEY")
	})

	// Create temp JSON file with the same key but different value
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configData := map[string]string{"TEST_JSONCONFIG_KEY": "from-json"}
	jsonBytes, err := Marshal(configData)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	if err := os.WriteFile(configPath, jsonBytes, 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	// Load and apply
	config, err := LoadJsonConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadJsonConfigFile() error = %v", err)
	}
	config.ApplyToEnv()

	// Env should win over JSON
	if got := os.Getenv("TEST_JSONCONFIG_KEY"); got != "from-env" {
		t.Errorf("ApplyToEnv() env[%q] = %q, want %q (env should win over json)", "TEST_JSONCONFIG_KEY", got, "from-env")
	}
}

// TestJsonConfigPrecedence_JsonFillsGap tests that JSON config values
// are applied when no environment variable exists (json fills the gap).
func TestJsonConfigPrecedence_JsonFillsGap(t *testing.T) {
	// Ensure the env var is NOT set
	os.Unsetenv("TEST_JSONCONFIG_GAP")
	t.Cleanup(func() {
		os.Unsetenv("TEST_JSONCONFIG_GAP")
	})

	// Create temp JSON file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configData := map[string]string{"TEST_JSONCONFIG_GAP": "from-json"}
	jsonBytes, err := Marshal(configData)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	if err := os.WriteFile(configPath, jsonBytes, 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	// Load and apply
	config, err := LoadJsonConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadJsonConfigFile() error = %v", err)
	}
	config.ApplyToEnv()

	// JSON should fill the gap
	if got := os.Getenv("TEST_JSONCONFIG_GAP"); got != "from-json" {
		t.Errorf("ApplyToEnv() env[%q] = %q, want %q (json should fill gap)", "TEST_JSONCONFIG_GAP", got, "from-json")
	}
}

// TestJsonConfigPrecedence_FlagOverEnv documents that flag > env precedence
// is handled by flag.Parse() in InitEnv(), which runs before this code path.
// The JSON config loader sets defaults via ApplyToEnv(), which does NOT override
// existing env vars. Since flag.Parse() binds to env vars (e.g., via pflag's
// BindEnv), the precedence chain is: flag > env > json > default.
func TestJsonConfigPrecedence_FlagOverEnv(t *testing.T) {
	// This test documents the design:
	// - flag.Parse() in InitEnv() handles flag > env precedence
	// - ApplyToEnv() only sets env vars that are NOT already set
	// - Therefore: flag > env > json > default
	//
	// We verify the mechanism: ApplyToEnv skips keys where env already exists.
	os.Setenv("TEST_FLAG_OVER_ENV", "from-env")
	t.Cleanup(func() {
		os.Unsetenv("TEST_FLAG_OVER_ENV")
	})

	config := JsonConfig{"TEST_FLAG_OVER_ENV": "from-json"}
	config.ApplyToEnv()

	// Env should remain unchanged (ApplyToEnv respects existing env)
	if got := os.Getenv("TEST_FLAG_OVER_ENV"); got != "from-env" {
		t.Errorf("ApplyToEnv() should not override existing env, got %q, want %q", got, "from-env")
	}
}

// TestJsonConfigInvalidJSON tests that invalid JSON content returns an error.
func TestJsonConfigInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.json")

	// Write invalid JSON
	invalidContent := "{{invalid}"
	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	_, err := LoadJsonConfigFile(configPath)
	if err == nil {
		t.Error("LoadJsonConfigFile() expected error for invalid JSON, got nil")
	}
}

// TestJsonConfigMissingFile tests that a non-existent file returns an error.
func TestJsonConfigMissingFile(t *testing.T) {
	_, err := LoadJsonConfigFile("/nonexistent/path/to/missing_config.json")
	if err == nil {
		t.Error("LoadJsonConfigFile() expected error for missing file, got nil")
	}
}

// TestJsonConfigEmptyFile tests that an empty JSON object returns
// an empty but non-nil JsonConfig.
func TestJsonConfigEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "empty.json")

	// Write empty JSON object
	emptyContent := "{}"
	if err := os.WriteFile(configPath, []byte(emptyContent), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	config, err := LoadJsonConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadJsonConfigFile() unexpected error: %v", err)
	}

	// Should be non-nil but empty
	if config == nil {
		t.Error("LoadJsonConfigFile() returned nil, want empty non-nil JsonConfig")
	}
	if len(config) != 0 {
		t.Errorf("LoadJsonConfigFile() got %d keys, want 0 keys", len(config))
	}
}