package common

import "os"

// JsonConfig is a flat key-value representation of JSON configuration.
// The JSON file format is a simple flat object: {"KEY": "value", ...}
// that maps directly to environment variable names.
type JsonConfig map[string]string

// LoadJsonConfigFile reads and parses a JSON configuration file.
// The JSON file must be a flat object with string values only.
// Uses common.DecodeJson internally.
func LoadJsonConfigFile(path string) (JsonConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config JsonConfig
	if err := DecodeJson(file, &config); err != nil {
		return nil, err
	}

	return config, nil
}

func (jc JsonConfig) ApplyToEnv() {
	for key, value := range jc {
		if _, ok := os.LookupEnv(key); !ok {
			os.Setenv(key, value)
		}
	}
}

func GetJsonConfigPath() string {
	if ConfigFile != nil && *ConfigFile != "" {
		return *ConfigFile
	}

	if configFile := os.Getenv("CONFIG_FILE"); configFile != "" {
		return configFile
	}

	if _, err := os.Stat("config.json"); err == nil {
		return "config.json"
	}

	return ""
}