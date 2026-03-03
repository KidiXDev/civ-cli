package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Manager struct {
	v *viper.Viper
}

const configName = "civitool_config"
const configType = "yaml" // Can be easily edited manually

func NewManager() *Manager {
	v := viper.New()
	v.SetConfigName(configName)
	v.SetConfigType(configType)

	// Environment variables
	v.SetEnvPrefix("CIVITAI")
	v.BindEnv("api_key")

	// Set configuration path to executable's directory
	execPath, err := os.Executable()
	if err == nil {
		v.AddConfigPath(filepath.Dir(execPath))
	} else {
		v.AddConfigPath(".")
	}
	v.AddConfigPath(".")

	return &Manager{v: v}
}

// Load reads the config from file or environment.
func (m *Manager) Load() (*Config, error) {
	// Set defaults
	defaults := DefaultConfig()
	m.v.SetDefault("api_key", defaults.APIKey)
	m.v.SetDefault("default_search_limit", defaults.DefaultSearchLimit)
	m.v.SetDefault("default_download_path", defaults.DefaultDownloadDir)
	m.v.SetDefault("output_format", defaults.OutputFormat)
	m.v.SetDefault("theme", defaults.Theme)
	m.v.SetDefault("timeout", defaults.TimeoutSeconds)
	m.v.SetDefault("retry_count", defaults.RetryCount)

	if err := m.v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// File not found is acceptable, defaults will be used
	}

	var cfg Config
	if err := m.v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &cfg, nil
}

// Save writes the current struct back to the config file
func (m *Manager) Save(cfg *Config) error {
	m.v.Set("api_key", cfg.APIKey)
	m.v.Set("default_search_limit", cfg.DefaultSearchLimit)
	m.v.Set("default_download_path", cfg.DefaultDownloadDir)
	m.v.Set("output_format", cfg.OutputFormat)
	m.v.Set("theme", cfg.Theme)
	m.v.Set("timeout", cfg.TimeoutSeconds)
	m.v.Set("retry_count", cfg.RetryCount)

	// If no config file exists yet, safe write
	if err := m.v.SafeWriteConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileAlreadyExistsError); ok {
			return m.v.WriteConfig()
		}
		return fmt.Errorf("error saving config: %w", err)
	}
	return nil
}

// FileExists checks if the configuration file has been written to disk
func (m *Manager) FileExists() bool {
	// ReadInConfig tells us if file is found or not in Load(), but we can check if viper tracked it
	return m.v.ConfigFileUsed() != "" || m.checkFileManually()
}

func (m *Manager) checkFileManually() bool {
	execPath, err := os.Executable()
	var dir string
	if err == nil {
		dir = filepath.Dir(execPath)
	} else {
		dir = "."
	}
	filePath := filepath.Join(dir, configName+"."+configType)
	_, err = os.Stat(filePath)
	return err == nil
}
