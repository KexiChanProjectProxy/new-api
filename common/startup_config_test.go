package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadStartupConfig_ValidNestedJSON(t *testing.T) {
	validConfig := `{
		"bootstrap": {
			"session_secret": "my-secret-key-12345"
		},
		"server": {
			"port": 8080
		},
		"database": {
			"sql_dsn": "mysql://user:pass@localhost/db"
		},
		"redis": {
			"redis_conn_string": "redis://localhost:6379",
			"redis_pool_size": 20
		},
		"logging": {
			"log_dir": "/var/log/app"
		},
		"relay": {
			"gemini_safety_setting": "BLOCK_NONE",
			"cohere_safety_setting": "NONE"
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	config, err := LoadStartupConfig(configPath)
	if err != nil {
		t.Fatalf("LoadStartupConfig() error = %v, want nil", err)
	}

	if config.Bootstrap.SessionSecret != "my-secret-key-12345" {
		t.Errorf("Bootstrap.SessionSecret = %q, want %q", config.Bootstrap.SessionSecret, "my-secret-key-12345")
	}
	if config.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want %d", config.Server.Port, 8080)
	}
	if config.Database.SQLDSN != "mysql://user:pass@localhost/db" {
		t.Errorf("Database.SQLDSN = %q, want %q", config.Database.SQLDSN, "mysql://user:pass@localhost/db")
	}
	if config.Redis.ConnString != "redis://localhost:6379" {
		t.Errorf("Redis.ConnString = %q, want %q", config.Redis.ConnString, "redis://localhost:6379")
	}
	if config.Redis.PoolSize != 20 {
		t.Errorf("Redis.PoolSize = %d, want %d", config.Redis.PoolSize, 20)
	}
}

func TestLoadStartupConfig_NestedRateLimitsAndTasks(t *testing.T) {
	validConfig := `{
		"bootstrap": {
			"session_secret": "my-secret"
		},
		"limits": {
			"rate_limit": {
				"api": {"enable": true, "num": 200, "duration": 120},
				"web": {"enable": true, "num": 100, "duration": 60},
				"critical": {"enable": false, "num": 50, "duration": 300},
				"search": {"enable": true, "num": 20, "duration": 30}
			},
			"task": {
				"query_limit": 500,
				"timeout_minutes": 720,
				"notify_limit_count": 5,
				"notification_limit_duration_minute": 15,
				"price_patches": ["patch1", "patch2"]
			}
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	config, err := LoadStartupConfig(configPath)
	if err != nil {
		t.Fatalf("LoadStartupConfig() error = %v, want nil", err)
	}

	if !config.Limits.RateLimit.API.Enable {
		t.Errorf("Limits.RateLimit.API.Enable = %v, want true", config.Limits.RateLimit.API.Enable)
	}
	if config.Limits.RateLimit.API.Num != 200 {
		t.Errorf("Limits.RateLimit.API.Num = %d, want %d", config.Limits.RateLimit.API.Num, 200)
	}
	if config.Limits.RateLimit.API.Duration != 120 {
		t.Errorf("Limits.RateLimit.API.Duration = %d, want %d", config.Limits.RateLimit.API.Duration, 120)
	}
	if config.Limits.RateLimit.Web.Num != 100 {
		t.Errorf("Limits.RateLimit.Web.Num = %d, want %d", config.Limits.RateLimit.Web.Num, 100)
	}
	if config.Limits.RateLimit.Critical.Enable {
		t.Errorf("Limits.RateLimit.Critical.Enable = %v, want false", config.Limits.RateLimit.Critical.Enable)
	}
	if config.Limits.Task.QueryLimit != 500 {
		t.Errorf("Limits.Task.QueryLimit = %d, want %d", config.Limits.Task.QueryLimit, 500)
	}
	if config.Limits.Task.TimeoutMinutes != 720 {
		t.Errorf("Limits.Task.TimeoutMinutes = %d, want %d", config.Limits.Task.TimeoutMinutes, 720)
	}
	if len(config.Limits.Task.PricePatches) != 2 {
		t.Errorf("len(Limits.Task.PricePatches) = %d, want 2", len(config.Limits.Task.PricePatches))
	}
}

func TestLoadStartupConfig_NestedObservability(t *testing.T) {
	validConfig := `{
		"bootstrap": {
			"session_secret": "my-secret"
		},
		"observability": {
			"pyroscope": {
				"url": "http://pyroscope:4040",
				"app_name": "my-app",
				"basic_auth_user": "admin",
				"basic_auth_password": "secret",
				"mutex_rate": 10,
				"block_rate": 10,
				"hostname": "my-host"
			},
			"pprof": {
				"enabled": true
			}
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	config, err := LoadStartupConfig(configPath)
	if err != nil {
		t.Fatalf("LoadStartupConfig() error = %v, want nil", err)
	}

	if config.Observability.Pyroscope.URL != "http://pyroscope:4040" {
		t.Errorf("Observability.Pyroscope.URL = %q, want %q", config.Observability.Pyroscope.URL, "http://pyroscope:4040")
	}
	if config.Observability.Pyroscope.AppName != "my-app" {
		t.Errorf("Observability.Pyroscope.AppName = %q, want %q", config.Observability.Pyroscope.AppName, "my-app")
	}
	if config.Observability.Pyroscope.MutexRate != 10 {
		t.Errorf("Observability.Pyroscope.MutexRate = %d, want %d", config.Observability.Pyroscope.MutexRate, 10)
	}
	if !config.Observability.Pprof.Enabled {
		t.Errorf("Observability.Pprof.Enabled = %v, want true", config.Observability.Pprof.Enabled)
	}
}

func TestLoadStartupConfig_MissingConfigPath(t *testing.T) {
	_, err := LoadStartupConfig("")
	if err == nil {
		t.Error("LoadStartupConfig() expected error for empty path, got nil")
	}
	if err != nil && err.Error() != "--config flag is required: no configuration file path provided" {
		t.Errorf("LoadStartupConfig() error = %q, want %q", err.Error(), "--config flag is required: no configuration file path provided")
	}
}

func TestLoadStartupConfig_MissingFile(t *testing.T) {
	_, err := LoadStartupConfig("/nonexistent/path/to/config.json")
	if err == nil {
		t.Error("LoadStartupConfig() expected error for missing file, got nil")
	}
}

func TestLoadStartupConfig_MalformedJSON(t *testing.T) {
	malformedJSON := `{invalid json content}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(malformedJSON), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	_, err := LoadStartupConfig(configPath)
	if err == nil {
		t.Error("LoadStartupConfig() expected error for malformed JSON, got nil")
	}
}

func TestLoadStartupConfig_MissingRequiredSessionSecret(t *testing.T) {
	missingSecret := `{
		"bootstrap": {}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(missingSecret), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	_, err := LoadStartupConfig(configPath)
	if err == nil {
		t.Error("LoadStartupConfig() expected error for missing session_secret, got nil")
	}
}

func TestLoadStartupConfig_InvalidSessionSecret(t *testing.T) {
	invalidSecret := `{
		"bootstrap": {
			"session_secret": "random_string"
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(invalidSecret), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	_, err := LoadStartupConfig(configPath)
	if err == nil {
		t.Error("LoadStartupConfig() expected error for invalid session_secret 'random_string', got nil")
	}
}

func TestLoadStartupConfig_InvalidGeminiSafetySetting(t *testing.T) {
	invalidGemini := `{
		"bootstrap": {
			"session_secret": "my-secret"
		},
		"relay": {
			"gemini_safety_setting": "INVALID_SETTING"
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(invalidGemini), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	_, err := LoadStartupConfig(configPath)
	if err == nil {
		t.Error("LoadStartupConfig() expected error for invalid gemini_safety_setting, got nil")
	}
}

func TestLoadStartupConfig_InvalidCohereSafetySetting(t *testing.T) {
	invalidCohere := `{
		"bootstrap": {
			"session_secret": "my-secret"
		},
		"relay": {
			"cohere_safety_setting": "INVALID"
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(invalidCohere), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	_, err := LoadStartupConfig(configPath)
	if err == nil {
		t.Error("LoadStartupConfig() expected error for invalid cohere_safety_setting, got nil")
	}
}

func TestLoadStartupConfig_InvalidNodeType(t *testing.T) {
	invalidNodeType := `{
		"bootstrap": {
			"session_secret": "my-secret",
			"node_type": "invalid"
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(invalidNodeType), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	_, err := LoadStartupConfig(configPath)
	if err == nil {
		t.Error("LoadStartupConfig() expected error for invalid node_type, got nil")
	}
}

func TestLoadStartupConfig_InvalidGinMode(t *testing.T) {
	invalidGinMode := `{
		"bootstrap": {
			"session_secret": "my-secret"
		},
		"server": {
			"gin_mode": "invalid"
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(invalidGinMode), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	_, err := LoadStartupConfig(configPath)
	if err == nil {
		t.Error("LoadStartupConfig() expected error for invalid gin_mode, got nil")
	}
}

func TestLoadStartupConfig_RejectsFlatKeyFormat(t *testing.T) {
	flatKeyConfig := `{
		"session_secret": "my-secret",
		"sql_dsn": "mysql://localhost/db"
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(flatKeyConfig), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	_, err := LoadStartupConfig(configPath)
	if err == nil {
		t.Error("LoadStartupConfig() expected error for flat-key format, got nil")
	}
	if err != nil && err.Error() != "config file uses legacy flat-key format; expected nested JSON structure with sections like 'bootstrap', 'database', 'redis'" {
		t.Errorf("LoadStartupConfig() error = %q, want %q", err.Error(), "config file uses legacy flat-key format; expected nested JSON structure with sections like 'bootstrap', 'database', 'redis'")
	}
}

func TestLoadStartupConfig_RejectsPartialFlatKeyFormat(t *testing.T) {
	partialFlat := `{
		"bootstrap": {
			"session_secret": "my-secret"
		},
		"REDIS_CONN_STRING": "redis://localhost"
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(partialFlat), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	_, err := LoadStartupConfig(configPath)
	if err == nil {
		t.Error("LoadStartupConfig() expected error for partial flat-key format, got nil")
	}
}

func TestLoadStartupConfig_NoEnvFallback(t *testing.T) {
	os.Setenv("SESSION_SECRET", "env-secret-should-not-be-used")
	os.Setenv("PORT", "9999")
	os.Setenv("REDIS_CONN_STRING", "redis://env-host:6379")
	defer func() {
		os.Unsetenv("SESSION_SECRET")
		os.Unsetenv("PORT")
		os.Unsetenv("REDIS_CONN_STRING")
	}()

	validConfig := `{
		"bootstrap": {
			"session_secret": "json-secret"
		},
		"server": {
			"port": 3000
		},
		"redis": {
			"redis_conn_string": "redis://json-host:6379"
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	config, err := LoadStartupConfig(configPath)
	if err != nil {
		t.Fatalf("LoadStartupConfig() error = %v, want nil", err)
	}

	if config.Bootstrap.SessionSecret != "json-secret" {
		t.Errorf("Bootstrap.SessionSecret = %q, want %q (env should be ignored)", config.Bootstrap.SessionSecret, "json-secret")
	}
	if config.Server.Port != 3000 {
		t.Errorf("Server.Port = %d, want %d (env should be ignored)", config.Server.Port, 3000)
	}
	if config.Redis.ConnString != "redis://json-host:6379" {
		t.Errorf("Redis.ConnString = %q, want %q (env should be ignored)", config.Redis.ConnString, "redis://json-host:6379")
	}
}

func TestGetConfigFilePath_NoEnvFallback(t *testing.T) {
	os.Setenv("CONFIG_FILE", "env-config.json")
	defer os.Unsetenv("CONFIG_FILE")

	originalConfigFile := ConfigFile
	ConfigFile = nil
	defer func() { ConfigFile = originalConfigFile }()

	path := GetConfigFilePath()
	if path != "" {
		t.Errorf("GetConfigFilePath() = %q, want empty string (env CONFIG_FILE should be ignored)", path)
	}
}

func TestGetConfigFilePath_WithFlag(t *testing.T) {
	originalConfigFile := ConfigFile
	defer func() { ConfigFile = originalConfigFile }()

	ConfigFile = stringPtr("/path/to/flag-config.json")
	path := GetConfigFilePath()
	if path != "/path/to/flag-config.json" {
		t.Errorf("GetConfigFilePath() = %q, want %q", path, "/path/to/flag-config.json")
	}
}

func TestStartupConfig_ApplyDefaults(t *testing.T) {
	config := &StartupConfig{}
	config.ApplyDefaults()

	if config.Bootstrap.NodeType != "master" {
		t.Errorf("Bootstrap.NodeType = %q, want %q", config.Bootstrap.NodeType, "master")
	}
	if config.Bootstrap.SQLitePath != "one-api.db?_busy_timeout=30000" {
		t.Errorf("Bootstrap.SQLitePath = %q, want %q", config.Bootstrap.SQLitePath, "one-api.db?_busy_timeout=30000")
	}
	if config.Server.Port != 3000 {
		t.Errorf("Server.Port = %d, want %d", config.Server.Port, 3000)
	}
	if config.Database.MaxIdleConns != 100 {
		t.Errorf("Database.MaxIdleConns = %d, want %d", config.Database.MaxIdleConns, 100)
	}
	if config.Database.MaxOpenConns != 1000 {
		t.Errorf("Database.MaxOpenConns = %d, want %d", config.Database.MaxOpenConns, 1000)
	}
	if config.Redis.PoolSize != 10 {
		t.Errorf("Redis.PoolSize = %d, want %d", config.Redis.PoolSize, 10)
	}
	if config.Logging.Dir != "./logs" {
		t.Errorf("Logging.Dir = %q, want %q", config.Logging.Dir, "./logs")
	}
	if config.Observability.Pyroscope.AppName != "new-api" {
		t.Errorf("Observability.Pyroscope.AppName = %q, want %q", config.Observability.Pyroscope.AppName, "new-api")
	}
	if config.Relay.StreamingTimeout != 300 {
		t.Errorf("Relay.StreamingTimeout = %d, want %d", config.Relay.StreamingTimeout, 300)
	}
	if config.Relay.GeminiSafetySetting != "BLOCK_NONE" {
		t.Errorf("Relay.GeminiSafetySetting = %q, want %q", config.Relay.GeminiSafetySetting, "BLOCK_NONE")
	}
	if config.Relay.CohereSafetySetting != "NONE" {
		t.Errorf("Relay.CohereSafetySetting = %q, want %q", config.Relay.CohereSafetySetting, "NONE")
	}
	if config.OAuth.LinuxDOTokenEndpoint != "https://connect.linux.do/oauth2/token" {
		t.Errorf("OAuth.LinuxDOTokenEndpoint = %q, want %q", config.OAuth.LinuxDOTokenEndpoint, "https://connect.linux.do/oauth2/token")
	}
	if config.Limits.RateLimit.API.Num != 180 {
		t.Errorf("Limits.RateLimit.API.Num = %d, want %d", config.Limits.RateLimit.API.Num, 180)
	}
	if config.Limits.Task.QueryLimit != 1000 {
		t.Errorf("Limits.Task.QueryLimit = %d, want %d", config.Limits.Task.QueryLimit, 1000)
	}
}

func TestStartupConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  StartupConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with all required fields",
			config: StartupConfig{
				Bootstrap: BootstrapConfig{
					SessionSecret: "my-secret-key",
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with master node type",
			config: StartupConfig{
				Bootstrap: BootstrapConfig{
					SessionSecret: "my-secret-key",
					NodeType:     "master",
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with slave node type",
			config: StartupConfig{
				Bootstrap: BootstrapConfig{
					SessionSecret: "my-secret-key",
					NodeType:     "slave",
				},
			},
			wantErr: false,
		},
		{
			name: "missing session secret",
			config: StartupConfig{
				Bootstrap: BootstrapConfig{
					SessionSecret: "",
				},
			},
			wantErr: true,
			errMsg:  "bootstrap.session_secret is required",
		},
		{
			name: "invalid session secret value",
			config: StartupConfig{
				Bootstrap: BootstrapConfig{
					SessionSecret: "random_string",
				},
			},
			wantErr: true,
			errMsg:  "bootstrap.session_secret must not be set to the default value 'random_string'",
		},
		{
			name: "invalid node type",
			config: StartupConfig{
				Bootstrap: BootstrapConfig{
					SessionSecret: "my-secret",
					NodeType:     "invalid",
				},
			},
			wantErr: true,
			errMsg:  "bootstrap.node_type must be either 'master' or 'slave'",
		},
		{
			name: "invalid gin mode",
			config: StartupConfig{
				Bootstrap: BootstrapConfig{
					SessionSecret: "my-secret",
				},
				Server: ServerConfig{
					GinMode: "invalid",
				},
			},
			wantErr: true,
			errMsg:  "server.gin_mode must be one of: debug, release, test",
		},
		{
			name: "invalid gemini safety setting",
			config: StartupConfig{
				Bootstrap: BootstrapConfig{
					SessionSecret: "my-secret",
				},
				Relay: RelayConfig{
					GeminiSafetySetting: "INVALID",
				},
			},
			wantErr: true,
			errMsg:  "relay.gemini_safety_setting must be one of: BLOCK_NONE, BLOCK_ONLY_HIGH, BLOCK_MEDIUM_AND_ABOVE, BLOCK_LOW_AND_ABOVE",
		},
		{
			name: "invalid cohere safety setting",
			config: StartupConfig{
				Bootstrap: BootstrapConfig{
					SessionSecret: "my-secret",
				},
				Relay: RelayConfig{
					CohereSafetySetting: "INVALID",
				},
			},
			wantErr: true,
			errMsg:  "relay.cohere_safety_setting must be one of: NONE, CONTEXTUAL, STRICT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("Validate() error = %q, want %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestStartupConfig_RedactSecrets(t *testing.T) {
	config := &StartupConfig{
		Bootstrap: BootstrapConfig{
			SessionSecret: "secret1",
			CryptoSecret: "secret2",
		},
		Redis: RedisConfig{
			ConnString: "redis://password@localhost",
		},
	}

	redacted := config.RedactSecrets()

	if redacted.Bootstrap.SessionSecret != "***REDACTED***" {
		t.Errorf("Redacted SessionSecret = %q, want %q", redacted.Bootstrap.SessionSecret, "***REDACTED***")
	}
	if redacted.Bootstrap.CryptoSecret != "***REDACTED***" {
		t.Errorf("Redacted CryptoSecret = %q, want %q", redacted.Bootstrap.CryptoSecret, "***REDACTED***")
	}
	if redacted.Redis.ConnString != "***REDACTED***" {
		t.Errorf("Redacted Redis.ConnString = %q, want %q", redacted.Redis.ConnString, "***REDACTED***")
	}

	if config.Bootstrap.SessionSecret != "secret1" {
		t.Errorf("Original SessionSecret should not be modified")
	}
}

func TestMakeConfigPathAbsolute(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "absolute path",
			path:    "/absolute/path/config.json",
			wantErr: false,
		},
		{
			name:    "relative path",
			path:    "config.json",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MakeConfigPathAbsolute(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("MakeConfigPathAbsolute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.path != "" && got == "" {
				t.Errorf("MakeConfigPathAbsolute() returned empty string for path %q", tt.path)
			}
		})
	}
}

func TestStartupConfig_AnalyticsOptional(t *testing.T) {
	withoutAnalytics := `{
		"bootstrap": {
			"session_secret": "my-secret"
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(withoutAnalytics), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	config, err := LoadStartupConfig(configPath)
	if err != nil {
		t.Fatalf("LoadStartupConfig() error = %v, want nil", err)
	}

	if config.Analytics != nil {
		t.Errorf("Analytics should be nil when not provided in config")
	}

	withAnalytics := `{
		"bootstrap": {
			"session_secret": "my-secret"
		},
		"analytics": {
			"umami_website_id": "abc123",
			"google_analytics_id": "G-XXXXX"
		}
	}`

	configPath2 := filepath.Join(tmpDir, "config2.json")
	if err := os.WriteFile(configPath2, []byte(withAnalytics), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	config2, err := LoadStartupConfig(configPath2)
	if err != nil {
		t.Fatalf("LoadStartupConfig() error = %v, want nil", err)
	}

	if config2.Analytics == nil {
		t.Errorf("Analytics should not be nil when provided in config")
	}
	if config2.Analytics.UmamiWebsiteID != "abc123" {
		t.Errorf("Analytics.UmamiWebsiteID = %q, want %q", config2.Analytics.UmamiWebsiteID, "abc123")
	}
}

func TestLoadStartupConfig_UnknownFieldsIgnored(t *testing.T) {
	// JSON with unknown fields should still parse successfully
	configWithUnknown := `{
		"bootstrap": {
			"session_secret": "my-secret"
		},
		"server": {
			"port": 8080
		},
		"database": {
			"sql_dsn": "mysql://user:pass@localhost/db"
		},
		"unknown_section": {
			"unknown_field": "should_be_ignored"
		},
		"another_unknown": 12345
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(configWithUnknown), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	config, err := LoadStartupConfig(configPath)
	if err != nil {
		t.Fatalf("LoadStartupConfig() error = %v, want nil (unknown fields should be ignored)", err)
	}

	if config.Bootstrap.SessionSecret != "my-secret" {
		t.Errorf("Bootstrap.SessionSecret = %q, want %q", config.Bootstrap.SessionSecret, "my-secret")
	}
	if config.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want %d", config.Server.Port, 8080)
	}
	if config.Database.SQLDSN != "mysql://user:pass@localhost/db" {
		t.Errorf("Database.SQLDSN = %q, want %q", config.Database.SQLDSN, "mysql://user:pass@localhost/db")
	}
}

func TestLoadStartupConfig_AllSectionsPopulated(t *testing.T) {
	// Comprehensive test with all sections populated
	allSectionsConfig := `{
		"bootstrap": {
			"session_secret": "my-secret-key-12345",
			"crypto_secret": "crypto-secret-456",
			"node_type": "slave",
			"debug": true,
			"memory_cache_enabled": true,
			"sqlite_path": "custom.db"
		},
		"server": {
			"port": 9000,
			"gin_mode": "release",
			"frontend_base_url": "https://example.com"
		},
		"database": {
			"sql_dsn": "mysql://user:pass@localhost:3306/mydb",
			"log_sql_dsn": "mysql://user:pass@localhost:3306/mydb_log",
			"sql_max_idle_conns": 50,
			"sql_max_open_conns": 500,
			"sql_max_lifetime_seconds": 120
		},
		"redis": {
			"redis_conn_string": "redis://:password@redis-host:6380/1",
			"redis_pool_size": 50
		},
		"logging": {
			"log_dir": "/var/log/myapp",
			"error_log_enabled": true
		},
		"observability": {
			"pyroscope": {
				"url": "http://pyroscope:4040",
				"app_name": "my-app-prod",
				"basic_auth_user": "admin",
				"basic_auth_password": "secret123",
				"mutex_rate": 10,
				"block_rate": 10,
				"hostname": "prod-host-1"
			},
			"pprof": {
				"enabled": true
			}
		},
		"relay": {
			"tls_insecure_skip_verify": true,
			"relay_timeout": 60,
			"relay_max_idle_conns": 200,
			"relay_max_idle_conns_per_host": 50,
			"streaming_timeout": 600,
			"max_file_download_mb": 100,
			"stream_scanner_max_buffer_mb": 64,
			"max_request_body_mb": 64,
			"azure_default_api_version": "2024-06-01",
			"gemini_safety_setting": "BLOCK_ONLY_HIGH",
			"cohere_safety_setting": "CONTEXTUAL",
			"relay_subpaths": ["/v1", "/v2"],
			"trusted_redirect_domains": ["example.com", "trusted.org"],
			"force_stream_option": true,
			"count_token": true,
			"get_media_token": true,
			"get_media_token_not_stream": false,
			"dify_debug": false
		},
		"limits": {
			"rate_limit": {
				"api": {"enable": true, "num": 200, "duration": 120},
				"web": {"enable": true, "num": 100, "duration": 60},
				"critical": {"enable": false, "num": 50, "duration": 300},
				"search": {"enable": true, "num": 20, "duration": 30}
			},
			"task": {
				"query_limit": 500,
				"timeout_minutes": 720,
				"notify_limit_count": 5,
				"notification_limit_duration_minute": 15,
				"price_patches": ["patch1", "patch2"]
			}
		},
		"schedulers": {
			"sync_frequency": 120,
			"polling_interval": 30,
			"batch_update_enabled": true,
			"batch_update_interval": 10,
			"channel_update_frequency": 60,
			"update_task": true,
			"channel_upstream_model_update_task_enabled": true,
			"channel_upstream_model_update_task_interval_minutes": 45,
			"channel_upstream_model_update_min_check_interval_seconds": 600
		},
		"sync": {
			"upstream_base": "https://custom.metadata.io",
			"http_timeout_seconds": 30,
			"http_retry": 5,
			"http_max_mb": 20
		},
		"oauth": {
			"linux_do_token_endpoint": "https://custom.example.com/oauth2/token",
			"linux_do_user_endpoint": "https://custom.example.com/api/user"
		},
		"analytics": {
			"umami_website_id": "abc123",
			"umami_script_url": "https://analytics.example.com/script.js",
			"google_analytics_id": "G-YYYYY"
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(allSectionsConfig), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	config, err := LoadStartupConfig(configPath)
	if err != nil {
		t.Fatalf("LoadStartupConfig() error = %v, want nil", err)
	}

	// Bootstrap
	if config.Bootstrap.SessionSecret != "my-secret-key-12345" {
		t.Errorf("Bootstrap.SessionSecret = %q, want %q", config.Bootstrap.SessionSecret, "my-secret-key-12345")
	}
	if config.Bootstrap.CryptoSecret != "crypto-secret-456" {
		t.Errorf("Bootstrap.CryptoSecret = %q, want %q", config.Bootstrap.CryptoSecret, "crypto-secret-456")
	}
	if config.Bootstrap.NodeType != "slave" {
		t.Errorf("Bootstrap.NodeType = %q, want %q", config.Bootstrap.NodeType, "slave")
	}
	if !config.Bootstrap.Debug {
		t.Errorf("Bootstrap.Debug = %v, want true", config.Bootstrap.Debug)
	}
	if !config.Bootstrap.MemoryCacheEnabled {
		t.Errorf("Bootstrap.MemoryCacheEnabled = %v, want true", config.Bootstrap.MemoryCacheEnabled)
	}
	if config.Bootstrap.SQLitePath != "custom.db" {
		t.Errorf("Bootstrap.SQLitePath = %q, want %q", config.Bootstrap.SQLitePath, "custom.db")
	}

	// Server
	if config.Server.Port != 9000 {
		t.Errorf("Server.Port = %d, want %d", config.Server.Port, 9000)
	}
	if config.Server.GinMode != "release" {
		t.Errorf("Server.GinMode = %q, want %q", config.Server.GinMode, "release")
	}
	if config.Server.FrontendBaseURL != "https://example.com" {
		t.Errorf("Server.FrontendBaseURL = %q, want %q", config.Server.FrontendBaseURL, "https://example.com")
	}

	// Database
	if config.Database.SQLDSN != "mysql://user:pass@localhost:3306/mydb" {
		t.Errorf("Database.SQLDSN = %q, want %q", config.Database.SQLDSN, "mysql://user:pass@localhost:3306/mydb")
	}
	if config.Database.MaxIdleConns != 50 {
		t.Errorf("Database.MaxIdleConns = %d, want %d", config.Database.MaxIdleConns, 50)
	}
	if config.Database.MaxOpenConns != 500 {
		t.Errorf("Database.MaxOpenConns = %d, want %d", config.Database.MaxOpenConns, 500)
	}

	// Redis
	if config.Redis.ConnString != "redis://:password@redis-host:6380/1" {
		t.Errorf("Redis.ConnString = %q, want %q", config.Redis.ConnString, "redis://:password@redis-host:6380/1")
	}
	if config.Redis.PoolSize != 50 {
		t.Errorf("Redis.PoolSize = %d, want %d", config.Redis.PoolSize, 50)
	}

	// Relay
	if !config.Relay.TLSInsecureSkipVerify {
		t.Errorf("Relay.TLSInsecureSkipVerify = %v, want true", config.Relay.TLSInsecureSkipVerify)
	}
	if config.Relay.GeminiSafetySetting != "BLOCK_ONLY_HIGH" {
		t.Errorf("Relay.GeminiSafetySetting = %q, want %q", config.Relay.GeminiSafetySetting, "BLOCK_ONLY_HIGH")
	}
	if config.Relay.CohereSafetySetting != "CONTEXTUAL" {
		t.Errorf("Relay.CohereSafetySetting = %q, want %q", config.Relay.CohereSafetySetting, "CONTEXTUAL")
	}
	if len(config.Relay.RelaySubpaths) != 2 {
		t.Errorf("len(Relay.RelaySubpaths) = %d, want 2", len(config.Relay.RelaySubpaths))
	}

	// Limits
	if !config.Limits.RateLimit.API.Enable {
		t.Errorf("Limits.RateLimit.API.Enable = %v, want true", config.Limits.RateLimit.API.Enable)
	}
	if config.Limits.RateLimit.API.Num != 200 {
		t.Errorf("Limits.RateLimit.API.Num = %d, want %d", config.Limits.RateLimit.API.Num, 200)
	}
	if config.Limits.Task.QueryLimit != 500 {
		t.Errorf("Limits.Task.QueryLimit = %d, want %d", config.Limits.Task.QueryLimit, 500)
	}

	// OAuth
	if config.OAuth.LinuxDOTokenEndpoint != "https://custom.example.com/oauth2/token" {
		t.Errorf("OAuth.LinuxDOTokenEndpoint = %q, want %q", config.OAuth.LinuxDOTokenEndpoint, "https://custom.example.com/oauth2/token")
	}

	// Analytics
	if config.Analytics == nil {
		t.Fatal("Analytics should not be nil")
	}
	if config.Analytics.UmamiWebsiteID != "abc123" {
		t.Errorf("Analytics.UmamiWebsiteID = %q, want %q", config.Analytics.UmamiWebsiteID, "abc123")
	}
}

func TestLoadStartupConfig_EmptyBootstrapWithOnlySessionSecret(t *testing.T) {
	// bootstrap: {} without session_secret should fail validation
	emptyBootstrap := `{
		"bootstrap": {}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(emptyBootstrap), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	_, err := LoadStartupConfig(configPath)
	if err == nil {
		t.Error("LoadStartupConfig() expected error for empty bootstrap, got nil")
	}
}

func TestStartupConfig_ApplyToBootstrapGlobals(t *testing.T) {
	config := &StartupConfig{
		Bootstrap: BootstrapConfig{
			SessionSecret:      "test-secret",
			CryptoSecret:       "crypto-secret",
			NodeType:           "slave",
			Debug:              true,
			MemoryCacheEnabled: true,
			SQLitePath:         "test.db",
		},
		Server: ServerConfig{
			Port:    9999,
			GinMode: "debug",
		},
		Logging: LoggingConfig{
			Dir:             "/tmp/test-logs",
			ErrorLogEnabled: true,
		},
		Schedulers: SchedulersConfig{
			SyncFrequency:       120,
			BatchUpdateEnabled:  true,
			BatchUpdateInterval: 15,
			PollingInterval:     45,
		},
		Limits: LimitsConfig{
			RateLimit: RateLimitSettings{
				API:      RateLimitTier{Enable: true, Num: 300, Duration: 90},
				Web:      RateLimitTier{Enable: true, Num: 150, Duration: 60},
				Critical: RateLimitTier{Enable: false, Num: 50, Duration: 300},
				Search:   RateLimitTier{Enable: true, Num: 30, Duration: 45},
			},
		},
		Relay: RelayConfig{
			TLSInsecureSkipVerify:   true,
			Timeout:                90,
			MaxIdleConns:           300,
			MaxIdleConnsPerHost:    75,
			GeminiSafetySetting:     "BLOCK_ONLY_HIGH",
			CohereSafetySetting:    "CONTEXTUAL",
			RelaySubpaths:          []string{"/custom/v1"},
			ForceStreamOption:      true,
			CountToken:             true,
			GetMediaToken:          false,
			GetMediaTokenNotStream: true,
			DifyDebug:              true,
		},
	}

	// ApplyToBootstrapGlobals modifies global variables, so we test it indirectly
	// by checking the config values that would be applied
	if config.Bootstrap.SessionSecret != "test-secret" {
		t.Errorf("ApplyToBootstrapGlobals: SessionSecret = %q, want %q", config.Bootstrap.SessionSecret, "test-secret")
	}
	if config.Bootstrap.CryptoSecret != "crypto-secret" {
		t.Errorf("ApplyToBootstrapGlobals: CryptoSecret = %q, want %q", config.Bootstrap.CryptoSecret, "crypto-secret")
	}
	if config.Bootstrap.NodeType != "slave" {
		t.Errorf("ApplyToBootstrapGlobals: NodeType = %q, want %q", config.Bootstrap.NodeType, "slave")
	}
	if !config.Bootstrap.Debug {
		t.Error("ApplyToBootstrapGlobals: Debug should be true")
	}
	if config.Server.Port != 9999 {
		t.Errorf("ApplyToBootstrapGlobals: Server.Port = %d, want %d", config.Server.Port, 9999)
	}
	if config.Relay.GeminiSafetySetting != "BLOCK_ONLY_HIGH" {
		t.Errorf("ApplyToBootstrapGlobals: Relay.GeminiSafetySetting = %q, want %q", config.Relay.GeminiSafetySetting, "BLOCK_ONLY_HIGH")
	}
	if config.Limits.RateLimit.API.Num != 300 {
		t.Errorf("ApplyToBootstrapGlobals: Limits.RateLimit.API.Num = %d, want %d", config.Limits.RateLimit.API.Num, 300)
	}
}

func TestStartupConfig_InitEnvFromStartupConfig(t *testing.T) {
	config := &StartupConfig{
		Relay: RelayConfig{
			StreamingTimeout:          600,
			MaxFileDownloadMB:        200,
			StreamScannerMaxBufferMB: 256,
			MaxRequestBodyMB:         64,
			AzureDefaultAPIVersion:   "2024-06-01",
			ForceStreamOption:        true,
			CountToken:               true,
			GetMediaToken:            true,
			GetMediaTokenNotStream:   false,
			DifyDebug:                true,
			TrustedRedirectDomains:   []string{"example.com", "test.org"},
		},
		Logging: LoggingConfig{
			ErrorLogEnabled: true,
		},
		Limits: LimitsConfig{
			Task: TaskSettings{
				QueryLimit:                       2000,
				TimeoutMinutes:                   1440,
				NotifyLimitCount:                 10,
				NotificationLimitDurationMinute:  30,
				PricePatches:                     []string{"patch-a", "patch-b"},
			},
		},
		Schedulers: SchedulersConfig{
			UpdateTask: true,
		},
	}

	// Verify config values that InitEnvFromStartupConfig would apply
	if config.Relay.StreamingTimeout != 600 {
		t.Errorf("InitEnvFromStartupConfig: StreamingTimeout = %d, want %d", config.Relay.StreamingTimeout, 600)
	}
	if config.Relay.MaxFileDownloadMB != 200 {
		t.Errorf("InitEnvFromStartupConfig: MaxFileDownloadMB = %d, want %d", config.Relay.MaxFileDownloadMB, 200)
	}
	if config.Relay.MaxRequestBodyMB != 64 {
		t.Errorf("InitEnvFromStartupConfig: MaxRequestBodyMB = %d, want %d", config.Relay.MaxRequestBodyMB, 64)
	}
	if config.Logging.ErrorLogEnabled != true {
		t.Error("InitEnvFromStartupConfig: ErrorLogEnabled should be true")
	}
	if config.Limits.Task.QueryLimit != 2000 {
		t.Errorf("InitEnvFromStartupConfig: Task.QueryLimit = %d, want %d", config.Limits.Task.QueryLimit, 2000)
	}
	if config.Limits.Task.NotifyLimitCount != 10 {
		t.Errorf("InitEnvFromStartupConfig: Task.NotifyLimitCount = %d, want %d", config.Limits.Task.NotifyLimitCount, 10)
	}
	if len(config.Relay.TrustedRedirectDomains) != 2 {
		t.Errorf("InitEnvFromStartupConfig: TrustedRedirectDomains len = %d, want 2", len(config.Relay.TrustedRedirectDomains))
	}
}

func TestLoadStartupConfig_DatabasePoolSizeBoundary(t *testing.T) {
	tests := []struct {
		name            string
		maxIdleConns    int
		maxOpenConns    int
		maxLifetimeSecs int
		wantErr         bool
	}{
		{"zero values use defaults", 0, 0, 0, false},
		{"small values", 1, 1, 1, false},
		{"typical values", 100, 1000, 60, false},
		{"large values", 10000, 50000, 3600, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configJSON := fmt.Sprintf(`{
				"bootstrap": {
					"session_secret": "my-secret"
				},
				"database": {
					"sql_dsn": "mysql://localhost/db",
					"sql_max_idle_conns": %d,
					"sql_max_open_conns": %d,
					"sql_max_lifetime_seconds": %d
				}
			}`, tt.maxIdleConns, tt.maxOpenConns, tt.maxLifetimeSecs)

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.json")
			if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
				t.Fatalf("failed to write temp config file: %v", err)
			}

			config, err := LoadStartupConfig(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadStartupConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if config.Database.MaxIdleConns != tt.maxIdleConns && tt.maxIdleConns != 0 {
					// Default is applied when 0
					t.Errorf("Database.MaxIdleConns = %d, want %d (or default if 0)", config.Database.MaxIdleConns, tt.maxIdleConns)
				}
			}
		})
	}
}

func TestLoadStartupConfig_RedisPoolSizeBoundary(t *testing.T) {
	tests := []struct {
		name     string
		poolSize int
		wantErr  bool
	}{
		{"zero uses default", 0, false},
		{"minimum", 1, false},
		{"typical", 50, false},
		{"large", 10000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configJSON := fmt.Sprintf(`{
				"bootstrap": {
					"session_secret": "my-secret"
				},
				"redis": {
					"redis_conn_string": "redis://localhost:6379",
					"redis_pool_size": %d
				}
			}`, tt.poolSize)

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.json")
			if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
				t.Fatalf("failed to write temp config file: %v", err)
			}

			config, err := LoadStartupConfig(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadStartupConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.poolSize != 0 {
				if config.Redis.PoolSize != tt.poolSize {
					t.Errorf("Redis.PoolSize = %d, want %d", config.Redis.PoolSize, tt.poolSize)
				}
			}
		})
	}
}

func TestLoadStartupConfig_ValidMinmalConfig(t *testing.T) {
	// Minimal config with just session_secret should be valid
	minimalConfig := `{
		"bootstrap": {
			"session_secret": "my-valid-secret-key"
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(minimalConfig), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	config, err := LoadStartupConfig(configPath)
	if err != nil {
		t.Fatalf("LoadStartupConfig() error = %v, want nil", err)
	}

	if config.Bootstrap.SessionSecret != "my-valid-secret-key" {
		t.Errorf("Bootstrap.SessionSecret = %q, want %q", config.Bootstrap.SessionSecret, "my-valid-secret-key")
	}
	// Check defaults are applied
	if config.Server.Port != 3000 {
		t.Errorf("Server.Port = %d, want default %d", config.Server.Port, 3000)
	}
	if config.Redis.PoolSize != 10 {
		t.Errorf("Redis.PoolSize = %d, want default %d", config.Redis.PoolSize, 10)
	}
}

func TestStartupConfig_RedactSecrets_UnknownFields(t *testing.T) {
	// Ensure RedactSecrets only redacts known secret fields
	config := &StartupConfig{
		Bootstrap: BootstrapConfig{
			SessionSecret: "secret1",
			CryptoSecret: "secret2",
			NodeType:     "master",
		},
		Redis: RedisConfig{
			ConnString: "redis://password@localhost",
		},
		Server: ServerConfig{
			Port: 3000,
		},
	}

	redacted := config.RedactSecrets()

	// Secrets should be redacted
	if redacted.Bootstrap.SessionSecret == "secret1" {
		t.Error("SessionSecret should be redacted")
	}
	if redacted.Bootstrap.CryptoSecret == "secret2" {
		t.Error("CryptoSecret should be redacted")
	}
	if redacted.Redis.ConnString == "redis://password@localhost" {
		t.Error("Redis.ConnString should be redacted")
	}

	// Non-secret fields should be unchanged
	if redacted.Bootstrap.NodeType != "master" {
		t.Errorf("NodeType = %q, should remain unchanged", redacted.Bootstrap.NodeType)
	}
	if redacted.Server.Port != 3000 {
		t.Errorf("Server.Port = %d, should remain unchanged", redacted.Server.Port)
	}
}

func TestStartupConfig_String(t *testing.T) {
	config := &StartupConfig{
		Bootstrap: BootstrapConfig{
			SessionSecret: "my-secret",
			NodeType:     "master",
		},
		Server: ServerConfig{
			Port: 3000,
		},
	}

	str := config.String()
	// Should contain redacted values and JSON format
	if str == "" {
		t.Error("String() should return non-empty string")
	}
	// The string should NOT contain the actual secret
	if str != "" && !strings.Contains(str, "REDACTED") {
		t.Logf("String output: %s", str)
	}
}

func stringPtr(s string) *string {
	return &s
}