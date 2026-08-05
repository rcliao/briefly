// Package config loads briefly's configuration from .briefly.yaml, .env,
// and environment variables (flags > env > config file > defaults).
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	AI     AI     `mapstructure:"ai"`
	Cache  Cache  `mapstructure:"cache"`
	Banner Banner `mapstructure:"banner"`
}

// AI holds AI/LLM configuration
type AI struct {
	Gemini GeminiConfig `mapstructure:"gemini"`
}

// GeminiConfig holds Google Gemini configuration
type GeminiConfig struct {
	APIKey string `mapstructure:"api_key"`
	Model  string `mapstructure:"model"`
}

// Cache holds cache configuration
type Cache struct {
	Directory string `mapstructure:"directory"`
}

// Banner holds LinkedIn banner image generation configuration
type Banner struct {
	Enabled     bool   `mapstructure:"enabled"`
	Model       string `mapstructure:"model"`
	Style       string `mapstructure:"style"`
	AspectRatio string `mapstructure:"aspect_ratio"`
}

var globalConfig *Config

// Load reads configuration from .env, the config file, and environment
// variables. Safe to call repeatedly; the first successful load wins.
func Load(configFile string) (*Config, error) {
	if globalConfig != nil {
		return globalConfig, nil
	}

	// Load .env file if it exists (provides GEMINI_API_KEY etc. to os.Getenv)
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(".env"); err != nil {
			fmt.Printf("Warning: Error loading .env file: %v\n", err)
		}
	}

	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME")
		viper.SetConfigName(".briefly")
		viper.SetConfigType("yaml")
	}

	setDefaults()

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	globalConfig = config
	return config, nil
}

// Get returns the global configuration, loading it if necessary.
// A CLI cannot proceed without configuration, so a load failure here exits
// with a clear message rather than panicking with a stack trace.
func Get() *Config {
	if globalConfig == nil {
		config, err := Load("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "briefly: failed to load configuration: %v\n", err)
			os.Exit(1)
		}
		return config
	}
	return globalConfig
}

func setDefaults() {
	viper.SetDefault("ai.gemini.model", "gemini-3.6-flash")
	viper.SetDefault("cache.directory", ".briefly-cache")

	viper.SetDefault("banner.enabled", true)
	viper.SetDefault("banner.model", "gemini-3.1-flash-image")
	viper.SetDefault("banner.style", "HD-2D style, detailed pixel art sprites in a 3D diorama environment, dramatic volumetric lighting, depth of field, tilt-shift, bloom and glow effects, rich atmospheric detail, no text or lettering")
	viper.SetDefault("banner.aspect_ratio", "16:9")
}
