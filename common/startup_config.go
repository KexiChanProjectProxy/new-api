package common

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/QuantumNous/new-api/constant"
)

var globalStartupConfig *StartupConfig

func SetStartupConfig(cfg *StartupConfig) {
	globalStartupConfig = cfg
}

func GetStartupConfig() *StartupConfig {
	return globalStartupConfig
}

// StartupConfig represents the typed nested configuration for application startup.
// This is loaded exclusively from --config path and covers all bootstrap configuration
// that must be present before or during bootstrap, with no DB dependency.
type StartupConfig struct {
	Bootstrap     BootstrapConfig      `json:"bootstrap"`
	Server        ServerConfig        `json:"server"`
	Database      DatabaseConfig      `json:"database"`
	Redis         RedisConfig         `json:"redis"`
	Logging       LoggingConfig       `json:"logging"`
	Observability ObservabilityConfig `json:"observability"`
	Relay         RelayConfig         `json:"relay"`
	Limits        LimitsConfig        `json:"limits"`
	Schedulers    SchedulersConfig    `json:"schedulers"`
	Sync          SyncConfig          `json:"sync"`
	OAuth         OAuthConfig         `json:"oauth"`
	Analytics     *AnalyticsConfig    `json:"analytics,omitempty"`
}

type BootstrapConfig struct {
	SessionSecret      string `json:"session_secret"`
	CryptoSecret       string `json:"crypto_secret,omitempty"`
	NodeType           string `json:"node_type,omitempty"`
	Debug              bool   `json:"debug"`
	MemoryCacheEnabled bool   `json:"memory_cache_enabled"`
	SQLitePath         string `json:"sqlite_path,omitempty"`
}

type ServerConfig struct {
	Port            int    `json:"port,omitempty"`
	GinMode         string `json:"gin_mode,omitempty"`
	FrontendBaseURL string `json:"frontend_base_url,omitempty"`
}

type DatabaseConfig struct {
	SQLDSN             string `json:"sql_dsn,omitempty"`
	LogDSN             string `json:"log_sql_dsn,omitempty"`
	MaxIdleConns       int    `json:"sql_max_idle_conns,omitempty"`
	MaxOpenConns       int    `json:"sql_max_open_conns,omitempty"`
	MaxLifetimeSeconds int    `json:"sql_max_lifetime_seconds,omitempty"`
}

type RedisConfig struct {
	ConnString string `json:"redis_conn_string,omitempty"`
	PoolSize   int    `json:"redis_pool_size,omitempty"`
}

type LoggingConfig struct {
	Dir             string `json:"log_dir,omitempty"`
	ErrorLogEnabled bool   `json:"error_log_enabled"`
}

type ObservabilityConfig struct {
	Pyroscope PyroscopeConfig `json:"pyroscope"`
	Pprof     PprofConfig     `json:"pprof"`
}

type PyroscopeConfig struct {
	URL               string `json:"url,omitempty"`
	AppName           string `json:"app_name,omitempty"`
	BasicAuthUser     string `json:"basic_auth_user,omitempty"`
	BasicAuthPassword string `json:"basic_auth_password,omitempty"`
	MutexRate         int    `json:"mutex_rate,omitempty"`
	BlockRate         int    `json:"block_rate,omitempty"`
	Hostname          string `json:"hostname,omitempty"`
}

type PprofConfig struct {
	Enabled bool `json:"enabled"`
}

type RelayConfig struct {
	TLSInsecureSkipVerify    bool     `json:"tls_insecure_skip_verify"`
	Timeout                  int      `json:"relay_timeout,omitempty"`
	MaxIdleConns             int      `json:"relay_max_idle_conns,omitempty"`
	MaxIdleConnsPerHost      int      `json:"relay_max_idle_conns_per_host,omitempty"`
	StreamingTimeout         int      `json:"streaming_timeout,omitempty"`
	MaxFileDownloadMB        int      `json:"max_file_download_mb,omitempty"`
	StreamScannerMaxBufferMB int      `json:"stream_scanner_max_buffer_mb,omitempty"`
	MaxRequestBodyMB         int      `json:"max_request_body_mb,omitempty"`
	AzureDefaultAPIVersion   string   `json:"azure_default_api_version,omitempty"`
	GeminiSafetySetting      string   `json:"gemini_safety_setting,omitempty"`
	CohereSafetySetting      string   `json:"cohere_safety_setting,omitempty"`
	RelaySubpaths            []string `json:"relay_subpaths,omitempty"`
	TrustedRedirectDomains   []string `json:"trusted_redirect_domains,omitempty"`
	ForceStreamOption        bool     `json:"force_stream_option"`
	CountToken               bool     `json:"count_token"`
	GetMediaToken            bool     `json:"get_media_token"`
	GetMediaTokenNotStream   bool     `json:"get_media_token_not_stream"`
	DifyDebug                bool     `json:"dify_debug"`
}

type LimitsConfig struct {
	RateLimit RateLimitSettings `json:"rate_limit"`
	Task      TaskSettings      `json:"task"`
}

type RateLimitSettings struct {
	API      RateLimitTier `json:"api"`
	Web      RateLimitTier `json:"web"`
	Critical RateLimitTier `json:"critical"`
	Search   RateLimitTier `json:"search"`
}

type RateLimitTier struct {
	Enable   bool  `json:"enable"`
	Num      int   `json:"num,omitempty"`
	Duration int64 `json:"duration,omitempty"`
}

type TaskSettings struct {
	QueryLimit                      int      `json:"query_limit,omitempty"`
	TimeoutMinutes                  int      `json:"timeout_minutes,omitempty"`
	NotifyLimitCount                int      `json:"notify_limit_count,omitempty"`
	NotificationLimitDurationMinute int      `json:"notification_limit_duration_minute,omitempty"`
	PricePatches                    []string `json:"price_patches,omitempty"`
}

type SchedulersConfig struct {
	SyncFrequency                               int  `json:"sync_frequency,omitempty"`
	PollingInterval                             int  `json:"polling_interval,omitempty"`
	BatchUpdateEnabled                          bool `json:"batch_update_enabled"`
	BatchUpdateInterval                         int  `json:"batch_update_interval,omitempty"`
	ChannelUpdateFrequency                      int  `json:"channel_update_frequency,omitempty"`
	UpdateTask                                  bool `json:"update_task"`
	ChannelUpstreamModelUpdateTaskEnabled       bool `json:"channel_upstream_model_update_task_enabled"`
	ChannelUpstreamModelUpdateTaskIntervalMinutes int `json:"channel_upstream_model_update_task_interval_minutes,omitempty"`
	ChannelUpstreamModelUpdateMinCheckIntervalSeconds int `json:"channel_upstream_model_update_min_check_interval_seconds,omitempty"`
}

type SyncConfig struct {
	UpstreamBase     string `json:"upstream_base,omitempty"`
	HTTPTimeout      int    `json:"http_timeout_seconds,omitempty"`
	HTTPRetry        int    `json:"http_retry,omitempty"`
	HTTPMaxMB        int    `json:"http_max_mb,omitempty"`
}

type OAuthConfig struct {
	LinuxDOTokenEndpoint string `json:"linux_do_token_endpoint,omitempty"`
	LinuxDOUserEndpoint  string `json:"linux_do_user_endpoint,omitempty"`
}

type AnalyticsConfig struct {
	UmamiWebsiteID    string `json:"umami_website_id,omitempty"`
	UmamiScriptURL    string `json:"umami_script_url,omitempty"`
	GoogleAnalyticsID string `json:"google_analytics_id,omitempty"`
}

func (sc *StartupConfig) ApplyDefaults() {
	if sc.Bootstrap.NodeType == "" {
		sc.Bootstrap.NodeType = "master"
	}
	if sc.Bootstrap.SQLitePath == "" {
		sc.Bootstrap.SQLitePath = "one-api.db?_busy_timeout=30000"
	}
	if sc.Server.Port == 0 {
		sc.Server.Port = 3000
	}
	if sc.Database.MaxIdleConns == 0 {
		sc.Database.MaxIdleConns = 100
	}
	if sc.Database.MaxOpenConns == 0 {
		sc.Database.MaxOpenConns = 1000
	}
	if sc.Database.MaxLifetimeSeconds == 0 {
		sc.Database.MaxLifetimeSeconds = 60
	}
	if sc.Redis.PoolSize == 0 {
		sc.Redis.PoolSize = 10
	}
	if sc.Logging.Dir == "" {
		sc.Logging.Dir = "./logs"
	}
	if sc.Observability.Pyroscope.AppName == "" {
		sc.Observability.Pyroscope.AppName = "new-api"
	}
	if sc.Observability.Pyroscope.Hostname == "" {
		sc.Observability.Pyroscope.Hostname = "new-api"
	}
	if sc.Observability.Pyroscope.MutexRate == 0 {
		sc.Observability.Pyroscope.MutexRate = 5
	}
	if sc.Observability.Pyroscope.BlockRate == 0 {
		sc.Observability.Pyroscope.BlockRate = 5
	}
	if sc.Relay.StreamingTimeout == 0 {
		sc.Relay.StreamingTimeout = 300
	}
	if sc.Relay.MaxFileDownloadMB == 0 {
		sc.Relay.MaxFileDownloadMB = 64
	}
	if sc.Relay.StreamScannerMaxBufferMB == 0 {
		sc.Relay.StreamScannerMaxBufferMB = 128
	}
	if sc.Relay.MaxRequestBodyMB == 0 {
		sc.Relay.MaxRequestBodyMB = 128
	}
	if sc.Relay.AzureDefaultAPIVersion == "" {
		sc.Relay.AzureDefaultAPIVersion = "2025-04-01-preview"
	}
	if sc.Relay.GeminiSafetySetting == "" {
		sc.Relay.GeminiSafetySetting = "BLOCK_NONE"
	}
	if sc.Relay.CohereSafetySetting == "" {
		sc.Relay.CohereSafetySetting = "NONE"
	}
	if sc.Relay.MaxIdleConns == 0 {
		sc.Relay.MaxIdleConns = 500
	}
	if sc.Relay.MaxIdleConnsPerHost == 0 {
		sc.Relay.MaxIdleConnsPerHost = 100
	}
	if sc.Limits.RateLimit.API.Num == 0 {
		sc.Limits.RateLimit.API.Num = 180
	}
	if sc.Limits.RateLimit.API.Duration == 0 {
		sc.Limits.RateLimit.API.Duration = 180
	}
	if sc.Limits.RateLimit.Web.Num == 0 {
		sc.Limits.RateLimit.Web.Num = 60
	}
	if sc.Limits.RateLimit.Web.Duration == 0 {
		sc.Limits.RateLimit.Web.Duration = 180
	}
	if sc.Limits.RateLimit.Critical.Num == 0 {
		sc.Limits.RateLimit.Critical.Num = 20
	}
	if sc.Limits.RateLimit.Critical.Duration == 0 {
		sc.Limits.RateLimit.Critical.Duration = 1200
	}
	if sc.Limits.RateLimit.Search.Num == 0 {
		sc.Limits.RateLimit.Search.Num = 10
	}
	if sc.Limits.RateLimit.Search.Duration == 0 {
		sc.Limits.RateLimit.Search.Duration = 60
	}
	if sc.Limits.Task.QueryLimit == 0 {
		sc.Limits.Task.QueryLimit = 1000
	}
	if sc.Limits.Task.TimeoutMinutes == 0 {
		sc.Limits.Task.TimeoutMinutes = 1440
	}
	if sc.Limits.Task.NotifyLimitCount == 0 {
		sc.Limits.Task.NotifyLimitCount = 2
	}
	if sc.Limits.Task.NotificationLimitDurationMinute == 0 {
		sc.Limits.Task.NotificationLimitDurationMinute = 10
	}
	if sc.Schedulers.SyncFrequency == 0 {
		sc.Schedulers.SyncFrequency = 60
	}
	if sc.Schedulers.PollingInterval == 0 {
		sc.Schedulers.PollingInterval = 60
	}
	if sc.Schedulers.BatchUpdateInterval == 0 {
		sc.Schedulers.BatchUpdateInterval = 5
	}
	if sc.OAuth.LinuxDOTokenEndpoint == "" {
		sc.OAuth.LinuxDOTokenEndpoint = "https://connect.linux.do/oauth2/token"
	}
	if sc.OAuth.LinuxDOUserEndpoint == "" {
		sc.OAuth.LinuxDOUserEndpoint = "https://connect.linux.do/api/user"
	}
	if sc.Sync.UpstreamBase == "" {
		sc.Sync.UpstreamBase = "https://basellm.github.io/llm-metadata"
	}
	if sc.Sync.HTTPTimeout == 0 {
		sc.Sync.HTTPTimeout = 10
	}
	if sc.Sync.HTTPRetry == 0 {
		sc.Sync.HTTPRetry = 3
	}
	if sc.Sync.HTTPMaxMB == 0 {
		sc.Sync.HTTPMaxMB = 10
	}
	if sc.Schedulers.ChannelUpstreamModelUpdateTaskIntervalMinutes == 0 {
		sc.Schedulers.ChannelUpstreamModelUpdateTaskIntervalMinutes = 30
	}
	if sc.Schedulers.ChannelUpstreamModelUpdateMinCheckIntervalSeconds == 0 {
		sc.Schedulers.ChannelUpstreamModelUpdateMinCheckIntervalSeconds = 300
	}
}

func (sc *StartupConfig) Validate() error {
	if sc.Bootstrap.SessionSecret == "" {
		return fmt.Errorf("bootstrap.session_secret is required")
	}
	if sc.Bootstrap.SessionSecret == "random_string" {
		return fmt.Errorf("bootstrap.session_secret must not be set to the default value 'random_string'")
	}
	validGeminiSettings := map[string]bool{
		"BLOCK_NONE": true, "BLOCK_ONLY_HIGH": true,
		"BLOCK_MEDIUM_AND_ABOVE": true, "BLOCK_LOW_AND_ABOVE": true,
	}
	if sc.Relay.GeminiSafetySetting != "" && !validGeminiSettings[sc.Relay.GeminiSafetySetting] {
		return fmt.Errorf("relay.gemini_safety_setting must be one of: BLOCK_NONE, BLOCK_ONLY_HIGH, BLOCK_MEDIUM_AND_ABOVE, BLOCK_LOW_AND_ABOVE")
	}
	validCohereSettings := map[string]bool{
		"NONE": true, "CONTEXTUAL": true, "STRICT": true,
	}
	if sc.Relay.CohereSafetySetting != "" && !validCohereSettings[sc.Relay.CohereSafetySetting] {
		return fmt.Errorf("relay.cohere_safety_setting must be one of: NONE, CONTEXTUAL, STRICT")
	}
	if sc.Bootstrap.NodeType != "" && sc.Bootstrap.NodeType != "master" && sc.Bootstrap.NodeType != "slave" {
		return fmt.Errorf("bootstrap.node_type must be either 'master' or 'slave'")
	}
	if sc.Server.GinMode != "" && sc.Server.GinMode != "debug" && sc.Server.GinMode != "release" && sc.Server.GinMode != "test" {
		return fmt.Errorf("server.gin_mode must be one of: debug, release, test")
	}
	return nil
}

func LoadStartupConfig(path string) (*StartupConfig, error) {
	if path == "" {
		return nil, fmt.Errorf("--config flag is required: no configuration file path provided")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	type topLevelCheck struct {
		SessionSecret string `json:"session_secret,omitempty"`
		SQLDSN        string `json:"sql_dsn,omitempty"`
		REDISConnStr  string `json:"redis_conn_string,omitempty"`
	}

	var topLevel topLevelCheck
	if err := DecodeJson(file, &topLevel); err != nil {
		return nil, fmt.Errorf("failed to parse config file as JSON: %w", err)
	}
	if topLevel.SessionSecret != "" || topLevel.SQLDSN != "" || topLevel.REDISConnStr != "" {
		return nil, fmt.Errorf("config file uses legacy flat-key format; expected nested JSON structure with sections like 'bootstrap', 'database', 'redis'")
	}

	file.Close()
	file, err = os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to reopen config file: %w", err)
	}
	defer file.Close()

	var config StartupConfig
	if err := DecodeJson(file, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	config.ApplyDefaults()

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

func GetConfigFilePath() string {
	if ConfigFile != nil && *ConfigFile != "" {
		return *ConfigFile
	}
	return ""
}

func MakeConfigPathAbsolute(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}
	return absPath, nil
}

func (sc *StartupConfig) RedactSecrets() StartupConfig {
	copy := *sc
	if copy.Bootstrap.SessionSecret != "" {
		copy.Bootstrap.SessionSecret = "***REDACTED***"
	}
	if copy.Bootstrap.CryptoSecret != "" {
		copy.Bootstrap.CryptoSecret = "***REDACTED***"
	}
	if copy.Redis.ConnString != "" {
		copy.Redis.ConnString = "***REDACTED***"
	}
	return copy
}

func (sc *StartupConfig) String() string {
	redacted := sc.RedactSecrets()
	data, err := Marshal(redacted)
	if err != nil {
		return fmt.Sprintf("StartupConfig{...error marshaling: %v}", err)
	}
	return string(data)
}

func (sc *StartupConfig) ApplyToBootstrapGlobals() {
	SessionSecret = sc.Bootstrap.SessionSecret
	CryptoSecret = sc.Bootstrap.CryptoSecret
	if CryptoSecret == "" {
		CryptoSecret = SessionSecret
	}
	SQLitePath = sc.Bootstrap.SQLitePath
	DebugEnabled = sc.Bootstrap.Debug
	MemoryCacheEnabled = sc.Bootstrap.MemoryCacheEnabled
	if sc.Bootstrap.NodeType == "" {
		IsMasterNode = true
	} else {
		IsMasterNode = sc.Bootstrap.NodeType == "master"
	}

	Port = &sc.Server.Port

	LogDir = sc.Logging.Dir

	SyncFrequency = sc.Schedulers.SyncFrequency
	BatchUpdateEnabled = sc.Schedulers.BatchUpdateEnabled
	BatchUpdateInterval = sc.Schedulers.BatchUpdateInterval

	GlobalApiRateLimitEnable = sc.Limits.RateLimit.API.Enable
	GlobalApiRateLimitNum = sc.Limits.RateLimit.API.Num
	GlobalApiRateLimitDuration = sc.Limits.RateLimit.API.Duration

	GlobalWebRateLimitEnable = sc.Limits.RateLimit.Web.Enable
	GlobalWebRateLimitNum = sc.Limits.RateLimit.Web.Num
	GlobalWebRateLimitDuration = sc.Limits.RateLimit.Web.Duration

	CriticalRateLimitEnable = sc.Limits.RateLimit.Critical.Enable
	CriticalRateLimitNum = sc.Limits.RateLimit.Critical.Num
	CriticalRateLimitDuration = sc.Limits.RateLimit.Critical.Duration

	SearchRateLimitEnable = sc.Limits.RateLimit.Search.Enable
	SearchRateLimitNum = sc.Limits.RateLimit.Search.Num
	SearchRateLimitDuration = sc.Limits.RateLimit.Search.Duration

	TLSInsecureSkipVerify = sc.Relay.TLSInsecureSkipVerify
	if TLSInsecureSkipVerify {
		if tr, ok := http.DefaultTransport.(*http.Transport); ok && tr != nil {
			if tr.TLSClientConfig != nil {
				tr.TLSClientConfig.InsecureSkipVerify = true
			} else {
				tr.TLSClientConfig = InsecureTLSConfig
			}
		}
	}

	RelayTimeout = sc.Relay.Timeout
	RelayMaxIdleConns = sc.Relay.MaxIdleConns
	RelayMaxIdleConnsPerHost = sc.Relay.MaxIdleConnsPerHost
	GeminiSafetySetting = sc.Relay.GeminiSafetySetting
	CohereSafetySetting = sc.Relay.CohereSafetySetting
	RelaySubpaths = sc.Relay.RelaySubpaths

	RequestInterval = time.Duration(sc.Schedulers.PollingInterval) * time.Second
}

func (sc *StartupConfig) InitEnvFromStartupConfig() {
	constant.ErrorLogEnabled = sc.Logging.ErrorLogEnabled
	constant.StreamingTimeout = sc.Relay.StreamingTimeout
	constant.DifyDebug = sc.Relay.DifyDebug
	constant.MaxFileDownloadMB = sc.Relay.MaxFileDownloadMB
	constant.StreamScannerMaxBufferMB = sc.Relay.StreamScannerMaxBufferMB
	constant.MaxRequestBodyMB = sc.Relay.MaxRequestBodyMB
	constant.ForceStreamOption = sc.Relay.ForceStreamOption
	constant.CountToken = sc.Relay.CountToken
	constant.GetMediaToken = sc.Relay.GetMediaToken
	constant.GetMediaTokenNotStream = sc.Relay.GetMediaTokenNotStream
	constant.UpdateTask = sc.Schedulers.UpdateTask
	constant.AzureDefaultAPIVersion = sc.Relay.AzureDefaultAPIVersion
	constant.NotifyLimitCount = sc.Limits.Task.NotifyLimitCount
	constant.NotificationLimitDurationMinute = sc.Limits.Task.NotificationLimitDurationMinute
	constant.TaskQueryLimit = sc.Limits.Task.QueryLimit
	constant.TaskTimeoutMinutes = sc.Limits.Task.TimeoutMinutes
	constant.TaskPricePatches = sc.Limits.Task.PricePatches
	constant.TrustedRedirectDomains = sc.Relay.TrustedRedirectDomains
}
