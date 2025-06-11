package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	App       App       `mapstructure:"app"`
	AI        AI        `mapstructure:"ai"`
	Search    Search    `mapstructure:"search"`
	Output    Output    `mapstructure:"output"`
	Cache     Cache     `mapstructure:"cache"`
	Visual    Visual    `mapstructure:"visual"`
	TTS       TTS       `mapstructure:"tts"`
	Messaging Messaging `mapstructure:"messaging"`
	Email     Email     `mapstructure:"email"`
	Feeds     Feeds     `mapstructure:"feeds"`
	Research  Research  `mapstructure:"research"`
	Logging   Logging   `mapstructure:"logging"`
	CLI       CLI       `mapstructure:"cli"`
}

// App holds general application configuration
type App struct {
	Debug      bool   `mapstructure:"debug"`
	LogLevel   string `mapstructure:"log_level"`
	DataDir    string `mapstructure:"data_dir"`
	ConfigFile string `mapstructure:"config_file"`
}

// AI holds AI/LLM configuration
type AI struct {
	Gemini GeminiConfig `mapstructure:"gemini"`
	OpenAI OpenAIConfig `mapstructure:"openai"`
}

// GeminiConfig holds Google Gemini configuration
type GeminiConfig struct {
	APIKey         string  `mapstructure:"api_key"`
	Model          string  `mapstructure:"model"`
	Timeout        string  `mapstructure:"timeout"`
	MaxTokens      int32   `mapstructure:"max_tokens"`
	Temperature    float32 `mapstructure:"temperature"`
	EmbeddingModel string  `mapstructure:"embedding_model"`
}

// OpenAIConfig holds OpenAI configuration
type OpenAIConfig struct {
	APIKey  string `mapstructure:"api_key"`
	Model   string `mapstructure:"model"`
	BaseURL string `mapstructure:"base_url"`
	Timeout string `mapstructure:"timeout"`
}

// Search holds search provider configuration
type Search struct {
	DefaultProvider string          `mapstructure:"default_provider"`
	MaxResults      int             `mapstructure:"max_results"`
	Timeout         string          `mapstructure:"timeout"`
	Language        string          `mapstructure:"language"`
	Providers       SearchProviders `mapstructure:"providers"`
}

// SearchProviders holds configuration for all search providers
type SearchProviders struct {
	Google     GoogleSearchConfig `mapstructure:"google"`
	SerpAPI    SerpAPIConfig      `mapstructure:"serpapi"`
	DuckDuckGo DuckDuckGoConfig   `mapstructure:"duckduckgo"`
}

// GoogleSearchConfig holds Google Custom Search configuration
type GoogleSearchConfig struct {
	APIKey   string `mapstructure:"api_key"`
	SearchID string `mapstructure:"search_id"`
}

// SerpAPIConfig holds SerpAPI configuration
type SerpAPIConfig struct {
	APIKey string `mapstructure:"api_key"`
}

// DuckDuckGoConfig holds DuckDuckGo configuration
type DuckDuckGoConfig struct {
	RateLimit string `mapstructure:"rate_limit"`
}

// Output holds output configuration
type Output struct {
	Directory    string `mapstructure:"directory"`
	Format       string `mapstructure:"format"`
	TemplatesDir string `mapstructure:"templates_dir"`
}

// Cache holds cache configuration
type Cache struct {
	Directory string         `mapstructure:"directory"`
	Database  DatabaseConfig `mapstructure:"database"`
	TTL       TTLConfig      `mapstructure:"ttl"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Timeout string `mapstructure:"timeout"`
}

// TTLConfig holds TTL configuration for different content types
type TTLConfig struct {
	Articles  string `mapstructure:"articles"`
	Summaries string `mapstructure:"summaries"`
	Digests   string `mapstructure:"digests"`
	Feeds     string `mapstructure:"feeds"`
}

// Visual holds visual/banner configuration
type Visual struct {
	Banners BannerConfig `mapstructure:"banners"`
}

// BannerConfig holds banner generation configuration
type BannerConfig struct {
	DefaultStyle    string `mapstructure:"default_style"`
	Width           int    `mapstructure:"width"`
	Height          int    `mapstructure:"height"`
	OutputDirectory string `mapstructure:"output_directory"`
	Quality         string `mapstructure:"quality"`
	Format          string `mapstructure:"format"`
}

// TTS holds text-to-speech configuration
type TTS struct {
	DefaultProvider string       `mapstructure:"default_provider"`
	DefaultVoice    string       `mapstructure:"default_voice"`
	DefaultSpeed    float32      `mapstructure:"default_speed"`
	OutputDirectory string       `mapstructure:"output_directory"`
	Timeout         string       `mapstructure:"timeout"`
	Providers       TTSProviders `mapstructure:"providers"`
}

// TTSProviders holds configuration for TTS providers
type TTSProviders struct {
	OpenAI     TTSOpenAIConfig     `mapstructure:"openai"`
	ElevenLabs TTSElevenLabsConfig `mapstructure:"elevenlabs"`
	Google     TTSGoogleConfig     `mapstructure:"google"`
}

// TTSOpenAIConfig holds OpenAI TTS configuration
type TTSOpenAIConfig struct {
	APIKey string `mapstructure:"api_key"`
	Model  string `mapstructure:"model"`
}

// TTSElevenLabsConfig holds ElevenLabs configuration
type TTSElevenLabsConfig struct {
	APIKey string `mapstructure:"api_key"`
}

// TTSGoogleConfig holds Google TTS configuration
type TTSGoogleConfig struct {
	APIKey string `mapstructure:"api_key"`
}

// Messaging holds messaging platform configuration
type Messaging struct {
	DefaultFormat string        `mapstructure:"default_format"`
	Timeout       string        `mapstructure:"timeout"`
	Slack         SlackConfig   `mapstructure:"slack"`
	Discord       DiscordConfig `mapstructure:"discord"`
}

// SlackConfig holds Slack configuration
type SlackConfig struct {
	WebhookURL     string `mapstructure:"webhook_url"`
	DefaultChannel string `mapstructure:"default_channel"`
	Username       string `mapstructure:"username"`
	IconEmoji      string `mapstructure:"icon_emoji"`
}

// DiscordConfig holds Discord configuration
type DiscordConfig struct {
	WebhookURL string `mapstructure:"webhook_url"`
	Username   string `mapstructure:"username"`
	AvatarURL  string `mapstructure:"avatar_url"`
}

// Email holds email configuration
type Email struct {
	SMTP            SMTPConfig `mapstructure:"smtp"`
	DefaultTemplate string     `mapstructure:"default_template"`
	FromAddress     string     `mapstructure:"from_address"`
	FromName        string     `mapstructure:"from_name"`
}

// SMTPConfig holds SMTP configuration
type SMTPConfig struct {
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	Username   string `mapstructure:"username"`
	Password   string `mapstructure:"password"`
	TLSEnabled bool   `mapstructure:"tls_enabled"`
}

// Feeds holds RSS/feed configuration
type Feeds struct {
	FetchInterval   string `mapstructure:"fetch_interval"`
	UserAgent       string `mapstructure:"user_agent"`
	Timeout         string `mapstructure:"timeout"`
	MaxItemsPerFeed int    `mapstructure:"max_items_per_feed"`
	CleanupInterval string `mapstructure:"cleanup_interval"`
}

// Research holds research configuration
type Research struct {
	MaxDepth           int    `mapstructure:"max_depth"`
	MaxQueries         int    `mapstructure:"max_queries"`
	Timeout            string `mapstructure:"timeout"`
	ConcurrentSearches int    `mapstructure:"concurrent_searches"`
}

// Logging holds logging configuration
type Logging struct {
	Level    string `mapstructure:"level"`
	Format   string `mapstructure:"format"`
	Output   string `mapstructure:"output"`
	FilePath string `mapstructure:"file_path"`
}

// CLI holds CLI-specific configuration
type CLI struct {
	Editor         string `mapstructure:"editor"`
	Interactive    bool   `mapstructure:"interactive"`
	DefaultFormat  string `mapstructure:"default_format"`
	StyleGuidePath string `mapstructure:"style_guide_path"`
}

var globalConfig *Config

// Load loads the configuration from various sources
func Load(configFile string) (*Config, error) {
	if globalConfig != nil {
		return globalConfig, nil
	}

	// Load .env file if it exists
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(".env"); err != nil {
			fmt.Printf("Warning: Error loading .env file: %v\n", err)
		}
	}

	// Configure viper
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME")
		viper.SetConfigName(".briefly")
		viper.SetConfigType("yaml")
	}

	// Set defaults
	setDefaults()

	// Bind environment variables
	bindEnvironmentVariables()

	// Enable automatic environment variable reading
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal into struct
	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Apply post-processing
	if err := postProcessConfig(config); err != nil {
		return nil, fmt.Errorf("error post-processing config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	globalConfig = config
	return config, nil
}

// Get returns the global configuration, loading it if necessary
func Get() *Config {
	if globalConfig == nil {
		config, err := Load("")
		if err != nil {
			panic(fmt.Sprintf("Failed to load configuration: %v", err))
		}
		return config
	}
	return globalConfig
}

// setDefaults sets default configuration values
func setDefaults() {
	// App defaults
	viper.SetDefault("app.debug", false)
	viper.SetDefault("app.log_level", "info")
	viper.SetDefault("app.data_dir", ".briefly-cache")

	// AI defaults
	viper.SetDefault("ai.gemini.model", "gemini-2.5-flash-preview-05-20")
	viper.SetDefault("ai.gemini.timeout", "30s")
	viper.SetDefault("ai.gemini.max_tokens", 8192)
	viper.SetDefault("ai.gemini.temperature", 0.7)
	viper.SetDefault("ai.gemini.embedding_model", "text-embedding-004")
	viper.SetDefault("ai.openai.model", "gpt-image-1")
	viper.SetDefault("ai.openai.base_url", "https://api.openai.com/v1")
	viper.SetDefault("ai.openai.timeout", "30s")

	// Search defaults
	viper.SetDefault("search.default_provider", "duckduckgo")
	viper.SetDefault("search.max_results", 10)
	viper.SetDefault("search.timeout", "15s")
	viper.SetDefault("search.language", "en")
	viper.SetDefault("search.providers.duckduckgo.rate_limit", "1s")

	// Output defaults
	viper.SetDefault("output.directory", "digests")
	viper.SetDefault("output.format", "standard")
	viper.SetDefault("output.templates_dir", "templates")

	// Cache defaults
	viper.SetDefault("cache.directory", ".briefly-cache")
	viper.SetDefault("cache.database.timeout", "5s")
	viper.SetDefault("cache.ttl.articles", "24h")
	viper.SetDefault("cache.ttl.summaries", "168h")
	viper.SetDefault("cache.ttl.digests", "720h")
	viper.SetDefault("cache.ttl.feeds", "1h")

	// Visual defaults
	viper.SetDefault("visual.banners.default_style", "tech")
	viper.SetDefault("visual.banners.width", 1792)
	viper.SetDefault("visual.banners.height", 1024)
	viper.SetDefault("visual.banners.output_directory", "banners")
	viper.SetDefault("visual.banners.quality", "high")
	viper.SetDefault("visual.banners.format", "PNG")

	// TTS defaults
	viper.SetDefault("tts.default_provider", "openai")
	viper.SetDefault("tts.default_voice", "alloy")
	viper.SetDefault("tts.default_speed", 1.0)
	viper.SetDefault("tts.output_directory", "audio")
	viper.SetDefault("tts.timeout", "60s")
	viper.SetDefault("tts.providers.openai.model", "tts-1")

	// Messaging defaults
	viper.SetDefault("messaging.default_format", "summary")
	viper.SetDefault("messaging.timeout", "10s")
	viper.SetDefault("messaging.slack.username", "Briefly")
	viper.SetDefault("messaging.slack.icon_emoji", ":newspaper:")
	viper.SetDefault("messaging.discord.username", "Briefly")

	// Email defaults
	viper.SetDefault("email.smtp.port", 587)
	viper.SetDefault("email.smtp.tls_enabled", true)
	viper.SetDefault("email.default_template", "default")
	viper.SetDefault("email.from_name", "Briefly")

	// Feeds defaults
	viper.SetDefault("feeds.fetch_interval", "1h")
	viper.SetDefault("feeds.user_agent", "Briefly/1.0")
	viper.SetDefault("feeds.timeout", "30s")
	viper.SetDefault("feeds.max_items_per_feed", 50)
	viper.SetDefault("feeds.cleanup_interval", "24h")

	// Research defaults
	viper.SetDefault("research.max_depth", 3)
	viper.SetDefault("research.max_queries", 5)
	viper.SetDefault("research.timeout", "60s")
	viper.SetDefault("research.concurrent_searches", 3)

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "text")
	viper.SetDefault("logging.output", "stdout")

	// CLI defaults
	viper.SetDefault("cli.editor", os.Getenv("EDITOR"))
	viper.SetDefault("cli.interactive", false)
	viper.SetDefault("cli.default_format", "standard")
}

// bindEnvironmentVariables sets up flexible environment variable binding
func bindEnvironmentVariables() {
	// Gemini API key - support multiple formats
	bindEnvKeys("ai.gemini.api_key", []string{
		"GEMINI_API_KEY",
		"GOOGLE_GEMINI_API_KEY",
		"GOOGLE_AI_API_KEY",
	})

	// OpenAI API key
	bindEnvKeys("ai.openai.api_key", []string{
		"OPENAI_API_KEY",
	})

	// Google Custom Search - support multiple formats
	bindEnvKeys("search.providers.google.api_key", []string{
		"GOOGLE_CUSTOM_SEARCH_API_KEY",
		"GOOGLE_CSE_API_KEY",
		"GOOGLE_SEARCH_API_KEY",
	})

	bindEnvKeys("search.providers.google.search_id", []string{
		"GOOGLE_CUSTOM_SEARCH_ID",
		"GOOGLE_CSE_ID",
		"GOOGLE_SEARCH_ENGINE_ID",
	})

	// SerpAPI
	bindEnvKeys("search.providers.serpapi.api_key", []string{
		"SERPAPI_API_KEY",
		"SERPAPI_KEY",
	})

	// TTS providers
	bindEnvKeys("tts.providers.openai.api_key", []string{
		"OPENAI_API_KEY",
	})

	bindEnvKeys("tts.providers.elevenlabs.api_key", []string{
		"ELEVENLABS_API_KEY",
		"ELEVEN_LABS_API_KEY",
	})

	bindEnvKeys("tts.providers.google.api_key", []string{
		"GOOGLE_TTS_API_KEY",
		"GOOGLE_CLOUD_API_KEY",
	})

	// Messaging webhooks
	bindEnvKeys("messaging.slack.webhook_url", []string{
		"SLACK_WEBHOOK_URL",
		"SLACK_WEBHOOK",
	})

	bindEnvKeys("messaging.discord.webhook_url", []string{
		"DISCORD_WEBHOOK_URL",
		"DISCORD_WEBHOOK",
	})

	// Email SMTP
	bindEnvKeys("email.smtp.host", []string{
		"SMTP_HOST",
		"EMAIL_SMTP_HOST",
	})

	bindEnvKeys("email.smtp.username", []string{
		"SMTP_USERNAME",
		"EMAIL_USERNAME",
	})

	bindEnvKeys("email.smtp.password", []string{
		"SMTP_PASSWORD",
		"EMAIL_PASSWORD",
	})

	// General settings
	bindEnvKeys("app.debug", []string{
		"DEBUG",
		"BRIEFLY_DEBUG",
	})

	bindEnvKeys("search.default_provider", []string{
		"SEARCH_PROVIDER",
		"DEFAULT_SEARCH_PROVIDER",
	})

	bindEnvKeys("cli.editor", []string{
		"EDITOR",
		"VISUAL",
	})
}

// bindEnvKeys binds the first found environment variable to a viper key
func bindEnvKeys(viperKey string, envKeys []string) {
	for _, envKey := range envKeys {
		if value := os.Getenv(envKey); value != "" {
			viper.Set(viperKey, value)
			return
		}
	}
}

// postProcessConfig applies post-processing to configuration values
func postProcessConfig(config *Config) error {
	// Expand paths
	if config.Cache.Directory != "" {
		config.Cache.Directory = expandPath(config.Cache.Directory)
	}
	if config.Output.Directory != "" {
		config.Output.Directory = expandPath(config.Output.Directory)
	}
	if config.Visual.Banners.OutputDirectory != "" {
		config.Visual.Banners.OutputDirectory = expandPath(config.Visual.Banners.OutputDirectory)
	}
	if config.TTS.OutputDirectory != "" {
		config.TTS.OutputDirectory = expandPath(config.TTS.OutputDirectory)
	}

	// Validate durations
	durations := map[string]string{
		"ai.gemini.timeout":      config.AI.Gemini.Timeout,
		"ai.openai.timeout":      config.AI.OpenAI.Timeout,
		"search.timeout":         config.Search.Timeout,
		"cache.database.timeout": config.Cache.Database.Timeout,
		"cache.ttl.articles":     config.Cache.TTL.Articles,
		"cache.ttl.summaries":    config.Cache.TTL.Summaries,
		"cache.ttl.digests":      config.Cache.TTL.Digests,
		"cache.ttl.feeds":        config.Cache.TTL.Feeds,
		"tts.timeout":            config.TTS.Timeout,
		"messaging.timeout":      config.Messaging.Timeout,
		"feeds.fetch_interval":   config.Feeds.FetchInterval,
		"feeds.timeout":          config.Feeds.Timeout,
		"feeds.cleanup_interval": config.Feeds.CleanupInterval,
		"research.timeout":       config.Research.Timeout,
	}

	for key, duration := range durations {
		if duration != "" {
			if _, err := time.ParseDuration(duration); err != nil {
				return fmt.Errorf("invalid duration for %s: %s", key, duration)
			}
		}
	}

	return nil
}

// expandPath expands ~ and environment variables in paths
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return os.ExpandEnv(path)
}

// validateConfig ensures required configuration is present
func validateConfig(config *Config) error {
	var errors []string

	// Gemini API key is required for most operations
	if config.AI.Gemini.APIKey == "" {
		errors = append(errors, "Gemini API key is required. Set GEMINI_API_KEY environment variable or ai.gemini.api_key in config file.\nGet your API key from: https://makersuite.google.com/app/apikey")
	}

	// Validate search provider configuration
	if config.Search.DefaultProvider != "" {
		switch config.Search.DefaultProvider {
		case "google":
			if config.Search.Providers.Google.APIKey == "" || config.Search.Providers.Google.SearchID == "" {
				errors = append(errors, "Google Custom Search requires both API key and Search ID. Set GOOGLE_CUSTOM_SEARCH_API_KEY and GOOGLE_CUSTOM_SEARCH_ID")
			}
		case "serpapi":
			if config.Search.Providers.SerpAPI.APIKey == "" {
				errors = append(errors, "SerpAPI requires API key. Set SERPAPI_API_KEY environment variable")
			}
		case "duckduckgo", "mock":
			// No validation needed for these providers
		default:
			errors = append(errors, fmt.Sprintf("Unknown search provider: %s. Supported: google, serpapi, duckduckgo, mock", config.Search.DefaultProvider))
		}
	}

	// Validate email SMTP configuration if any email settings are provided
	if config.Email.SMTP.Host != "" || config.Email.SMTP.Username != "" {
		if config.Email.SMTP.Host == "" {
			errors = append(errors, "SMTP host is required when email is configured")
		}
		if config.Email.SMTP.Username == "" {
			errors = append(errors, "SMTP username is required when email is configured")
		}
		if config.Email.SMTP.Password == "" {
			errors = append(errors, "SMTP password is required when email is configured")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration errors:\n- %s", strings.Join(errors, "\n- "))
	}

	return nil
}

// Convenience getters for commonly used configuration values
func GetApp() App             { return Get().App }
func GetAI() AI               { return Get().AI }
func GetSearch() Search       { return Get().Search }
func GetOutput() Output       { return Get().Output }
func GetCache() Cache         { return Get().Cache }
func GetVisual() Visual       { return Get().Visual }
func GetTTS() TTS             { return Get().TTS }
func GetMessaging() Messaging { return Get().Messaging }
func GetEmail() Email         { return Get().Email }
func GetFeeds() Feeds         { return Get().Feeds }
func GetResearch() Research   { return Get().Research }
func GetLogging() Logging     { return Get().Logging }
func GetCLI() CLI             { return Get().CLI }

// Specific convenience getters for frequently accessed values
func GetGeminiAPIKey() string   { return Get().AI.Gemini.APIKey }
func GetGeminiModel() string    { return Get().AI.Gemini.Model }
func GetOpenAIAPIKey() string   { return Get().AI.OpenAI.APIKey }
func GetSearchProvider() string { return Get().Search.DefaultProvider }
func GetGoogleSearchConfig() (string, string) {
	c := Get().Search.Providers.Google
	return c.APIKey, c.SearchID
}
func GetSerpAPIKey() string      { return Get().Search.Providers.SerpAPI.APIKey }
func GetOutputDirectory() string { return Get().Output.Directory }
func GetCacheDirectory() string  { return Get().Cache.Directory }
func IsDebugMode() bool          { return Get().App.Debug }

// HasValidGoogleSearch returns true if Google Custom Search is properly configured
func HasValidGoogleSearch() bool {
	apiKey, searchID := GetGoogleSearchConfig()
	return isValidAPIKey(apiKey) && isValidSearchID(searchID)
}

// HasValidSerpAPI returns true if SerpAPI is properly configured
func HasValidSerpAPI() bool {
	return isValidAPIKey(GetSerpAPIKey())
}

// GetSearchProviderConfig returns configuration for creating a search provider
func GetSearchProviderConfig(providerType string) map[string]string {
	config := Get()

	switch providerType {
	case "google":
		return map[string]string{
			"api_key":   config.Search.Providers.Google.APIKey,
			"search_id": config.Search.Providers.Google.SearchID,
		}
	case "serpapi":
		return map[string]string{
			"api_key": config.Search.Providers.SerpAPI.APIKey,
		}
	default:
		return map[string]string{}
	}
}

// isValidAPIKey checks if an API key is valid (not empty and not a placeholder)
func isValidAPIKey(apiKey string) bool {
	if apiKey == "" {
		return false
	}

	// Check for common placeholder values
	placeholders := []string{
		"your-api-key", "your-google-key", "your-google-api-key", "your-serpapi-key",
		"your-openai-key", "YOUR_API_KEY", "PLACEHOLDER", "TODO", "CHANGE_ME",
	}

	for _, placeholder := range placeholders {
		if apiKey == placeholder {
			return false
		}
	}

	return true
}

// isValidSearchID checks if a search ID is valid (not empty and not a placeholder)
func isValidSearchID(searchID string) bool {
	if searchID == "" {
		return false
	}

	// Check for common placeholder values
	placeholders := []string{
		"your-search-engine-id", "your-search-id", "your-cse-id",
		"YOUR_SEARCH_ID", "PLACEHOLDER", "TODO", "CHANGE_ME",
	}

	for _, placeholder := range placeholders {
		if searchID == placeholder {
			return false
		}
	}

	return true
}

// Reset clears the global configuration (useful for testing)
func Reset() {
	globalConfig = nil
	viper.Reset()
}
