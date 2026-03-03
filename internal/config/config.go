package config

// Config represents the core settings for the application.
type Config struct {
	APIKey             string `mapstructure:"api_key" yaml:"api_key" json:"api_key"`
	DefaultSearchLimit int    `mapstructure:"default_search_limit" yaml:"default_search_limit" json:"default_search_limit"`
	DefaultDownloadDir string `mapstructure:"default_download_path" yaml:"default_download_path" json:"default_download_path"`
	OutputFormat       string `mapstructure:"output_format" yaml:"output_format" json:"output_format"`
	Theme              string `mapstructure:"theme" yaml:"theme" json:"theme"`
	TimeoutSeconds     int    `mapstructure:"timeout" yaml:"timeout" json:"timeout"`
	RetryCount         int    `mapstructure:"retry_count" yaml:"retry_count" json:"retry_count"`
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		APIKey:             "",
		DefaultSearchLimit: 20,
		DefaultDownloadDir: "./downloads",
		OutputFormat:       "table",
		Theme:              "default",
		TimeoutSeconds:     30,
		RetryCount:         3,
	}
}
