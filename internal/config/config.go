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
	App           App           `mapstructure:"app"`
	AI            AI            `mapstructure:"ai"`
	Database      Database      `mapstructure:"database"`
	Server        Server        `mapstructure:"server"`
	Search        Search        `mapstructure:"search"`
	Output        Output        `mapstructure:"output"`
	Cache         Cache         `mapstructure:"cache"`
	Visual        Visual        `mapstructure:"visual"`
	TTS           TTS           `mapstructure:"tts"`
	Messaging     Messaging     `mapstructure:"messaging"`
	Email         Email         `mapstructure:"email"`
	Feeds         Feeds         `mapstructure:"feeds"`
	Research      Research      `mapstructure:"research"`
	Filtering     Filtering     `mapstructure:"filtering"`
	Team          Team          `mapstructure:"team"`
	Logging       Logging       `mapstructure:"logging"`
	CLI           CLI           `mapstructure:"cli"`
	Observability Observability `mapstructure:"observability"`
	Themes        Themes        `mapstructure:"themes"`
}

// Database holds database configuration
type Database struct {
	ConnectionString string `mapstructure:"connection_string"`
	MaxConnections   int    `mapstructure:"max_connections"`
	IdleConnections  int    `mapstructure:"idle_connections"`
}

// Server holds HTTP server configuration
type Server struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	StaticDir       string        `mapstructure:"static_dir"`
	TemplateDir     string        `mapstructure:"template_dir"`
	CORS            CORSConfig    `mapstructure:"cors"`
	RateLimit       RateLimitConfig `mapstructure:"rate_limit"`
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	Enabled        bool     `mapstructure:"enabled"`
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled           bool `mapstructure:"enabled"`
	RequestsPerMinute int  `mapstructure:"requests_per_minute"`
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
	MaxDepth           int        `mapstructure:"max_depth"`
	MaxQueries         int        `mapstructure:"max_queries"`
	Timeout            string     `mapstructure:"timeout"`
	ConcurrentSearches int        `mapstructure:"concurrent_searches"`
	V2                 ResearchV2 `mapstructure:"v2"`
}

// ResearchV2 holds research v2 enhanced features configuration
type ResearchV2 struct {
	Enabled         bool                  `mapstructure:"enabled"`
	QueryGeneration QueryGenerationConfig `mapstructure:"query_generation"`
	Scoring         ScoringConfig         `mapstructure:"scoring"`
	Clustering      ClusteringConfig      `mapstructure:"clustering"`
	Insights        InsightsConfig        `mapstructure:"insights"`
	Sources         SourcesConfig         `mapstructure:"sources"`
}

// QueryGenerationConfig holds query generation settings
type QueryGenerationConfig struct {
	CompetitiveAnalysis bool `mapstructure:"competitive_analysis"`
	TechnicalDepth      bool `mapstructure:"technical_depth"`
	MaxIterations       int  `mapstructure:"max_iterations"`
}

// ScoringConfig holds scoring configuration
type ScoringConfig struct {
	Profile           string  `mapstructure:"profile"` // research, competitive, technical
	SemanticThreshold float64 `mapstructure:"semantic_threshold"`
	AuthorityWeight   float64 `mapstructure:"authority_weight"`
}

// ClusteringConfig holds clustering settings
type ClusteringConfig struct {
	AutoCategorize   bool    `mapstructure:"auto_categorize"`
	MinClusterSize   int     `mapstructure:"min_cluster_size"`
	BalanceThreshold float64 `mapstructure:"balance_threshold"`
}

// InsightsConfig holds insights generation settings
type InsightsConfig struct {
	CompetitiveIntelligence  bool `mapstructure:"competitive_intelligence"`
	TechnicalAssessment      bool `mapstructure:"technical_assessment"`
	StrategicRecommendations bool `mapstructure:"strategic_recommendations"`
}

// SourcesConfig holds source management settings
type SourcesConfig struct {
	AuthorityWeighting   bool    `mapstructure:"authority_weighting"`
	DiversityRequirement int     `mapstructure:"diversity_requirement"`
	QualityThreshold     float64 `mapstructure:"quality_threshold"`
}

// Filtering holds relevance filtering configuration
type Filtering struct {
	Enabled      bool              `mapstructure:"enabled"`       // Enable/disable relevance filtering
	MinRelevance float64           `mapstructure:"min_relevance"` // Minimum relevance threshold (0.0-1.0)
	Method       string            `mapstructure:"method"`        // Scoring method: keyword, embedding, hybrid
	Weights      FilteringWeights  `mapstructure:"weights"`       // Scoring weights configuration
	Templates    TemplateFiltering `mapstructure:"templates"`     // Per-template filtering settings
}

// FilteringWeights holds scoring weight configuration
type FilteringWeights struct {
	ContentRelevance float64 `mapstructure:"content_relevance"` // Weight for content match (0.0-1.0)
	TitleRelevance   float64 `mapstructure:"title_relevance"`   // Weight for title match (0.0-1.0)
	Authority        float64 `mapstructure:"authority"`         // Weight for source authority (0.0-1.0)
	Recency          float64 `mapstructure:"recency"`           // Weight for content freshness (0.0-1.0)
	Quality          float64 `mapstructure:"quality"`           // Weight for content quality (0.0-1.0)
}

// TemplateFiltering holds per-template filtering configuration
type TemplateFiltering struct {
	Brief      TemplateFilter `mapstructure:"brief"`
	Standard   TemplateFilter `mapstructure:"standard"`
	Detailed   TemplateFilter `mapstructure:"detailed"`
	Newsletter TemplateFilter `mapstructure:"newsletter"`
	Email      TemplateFilter `mapstructure:"email"`
}

// TemplateFilter holds filtering settings for a specific template
type TemplateFilter struct {
	MinRelevance float64 `mapstructure:"min_relevance"` // Override global min_relevance for this template
	MaxWords     int     `mapstructure:"max_words"`     // Maximum words for this template (0 for template default)
	MaxArticles  int     `mapstructure:"max_articles"`  // Maximum number of articles (0 for no limit)
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

// Team holds team context configuration for relevance and insights
type Team struct {
	TechStack         []string `mapstructure:"tech_stack"`         // Technologies your team uses
	CurrentChallenges []string `mapstructure:"current_challenges"` // Current problems/focuses
	Interests         []string `mapstructure:"interests"`          // Areas of interest
	ProductType       string   `mapstructure:"product_type"`       // Type of product being built
	CompanySize       string   `mapstructure:"company_size"`       // startup, mid-size, enterprise
	Industry          string   `mapstructure:"industry"`           // Industry vertical
	ExpertiseLevel    string   `mapstructure:"expertise_level"`    // junior, mid, senior, principal
	WorkingStyle      string   `mapstructure:"working_style"`      // agile, waterfall, lean, etc.
	TeamSize          int      `mapstructure:"team_size"`          // Size of the engineering team
	Priority          string   `mapstructure:"priority"`           // Current priority: performance, features, quality, etc.
}

// Observability holds observability and analytics configuration
type Observability struct {
	LangFuse LangFuseConfig `mapstructure:"langfuse"`
	PostHog  PostHogConfig  `mapstructure:"posthog"`
}

// LangFuseConfig holds LangFuse observability configuration
type LangFuseConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	PublicKey string `mapstructure:"public_key"`
	SecretKey string `mapstructure:"secret_key"`
	Host      string `mapstructure:"host"` // Default: https://cloud.langfuse.com
}

// PostHogConfig holds PostHog analytics configuration
type PostHogConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	APIKey  string `mapstructure:"api_key"`
	Host    string `mapstructure:"host"` // Default: https://app.posthog.com
}

// Themes holds theme classification configuration
type Themes struct {
	Enabled             bool    `mapstructure:"enabled"`
	MinRelevanceScore   float64 `mapstructure:"min_relevance_score"` // 0.0-1.0, minimum score to assign theme
	ClassificationModel string  `mapstructure:"classification_model"` // LLM model to use for classification
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

	// Server defaults
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout", "15s")
	viper.SetDefault("server.write_timeout", "15s")
	viper.SetDefault("server.shutdown_timeout", "10s")
	viper.SetDefault("server.static_dir", "web/static")
	viper.SetDefault("server.template_dir", "web/templates")
	viper.SetDefault("server.cors.enabled", true)
	viper.SetDefault("server.cors.allowed_origins", []string{"http://localhost:3000", "http://localhost:8080"})
	viper.SetDefault("server.rate_limit.enabled", true)
	viper.SetDefault("server.rate_limit.requests_per_minute", 60)

	// AI defaults
	viper.SetDefault("ai.gemini.model", "gemini-flash-lite-latest")
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

	// Research V2 defaults
	viper.SetDefault("research.v2.enabled", true)
	viper.SetDefault("research.v2.query_generation.competitive_analysis", true)
	viper.SetDefault("research.v2.query_generation.technical_depth", true)
	viper.SetDefault("research.v2.query_generation.max_iterations", 3)
	viper.SetDefault("research.v2.scoring.profile", "research")
	viper.SetDefault("research.v2.scoring.semantic_threshold", 0.7)
	viper.SetDefault("research.v2.scoring.authority_weight", 0.2)
	viper.SetDefault("research.v2.clustering.auto_categorize", true)
	viper.SetDefault("research.v2.clustering.min_cluster_size", 3)
	viper.SetDefault("research.v2.clustering.balance_threshold", 0.6)
	viper.SetDefault("research.v2.insights.competitive_intelligence", true)
	viper.SetDefault("research.v2.insights.technical_assessment", true)
	viper.SetDefault("research.v2.insights.strategic_recommendations", true)
	viper.SetDefault("research.v2.sources.authority_weighting", true)
	viper.SetDefault("research.v2.sources.diversity_requirement", 4)
	viper.SetDefault("research.v2.sources.quality_threshold", 0.5)

	// Filtering defaults
	viper.SetDefault("filtering.enabled", true)
	viper.SetDefault("filtering.min_relevance", 0.4)
	viper.SetDefault("filtering.method", "keyword")

	// Filtering weights defaults (digest profile)
	viper.SetDefault("filtering.weights.content_relevance", 0.6)
	viper.SetDefault("filtering.weights.title_relevance", 0.3)
	viper.SetDefault("filtering.weights.authority", 0.1)
	viper.SetDefault("filtering.weights.recency", 0.0)
	viper.SetDefault("filtering.weights.quality", 0.0)

	// Template-specific filtering defaults
	viper.SetDefault("filtering.templates.brief.min_relevance", 0.6) // Stricter for brief
	viper.SetDefault("filtering.templates.brief.max_words", 200)
	viper.SetDefault("filtering.templates.brief.max_articles", 3)

	viper.SetDefault("filtering.templates.standard.min_relevance", 0.4) // Balanced
	viper.SetDefault("filtering.templates.standard.max_words", 400)
	viper.SetDefault("filtering.templates.standard.max_articles", 5)

	viper.SetDefault("filtering.templates.detailed.min_relevance", 0.2) // Most inclusive
	viper.SetDefault("filtering.templates.detailed.max_words", 0)       // No limit
	viper.SetDefault("filtering.templates.detailed.max_articles", 0)    // No limit

	viper.SetDefault("filtering.templates.newsletter.min_relevance", 0.4) // Balanced
	viper.SetDefault("filtering.templates.newsletter.max_words", 800)
	viper.SetDefault("filtering.templates.newsletter.max_articles", 6)

	viper.SetDefault("filtering.templates.email.min_relevance", 0.4) // Balanced
	viper.SetDefault("filtering.templates.email.max_words", 400)
	viper.SetDefault("filtering.templates.email.max_articles", 5)

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "text")
	viper.SetDefault("logging.output", "stdout")

	// CLI defaults
	viper.SetDefault("cli.editor", os.Getenv("EDITOR"))
	viper.SetDefault("cli.interactive", false)
	viper.SetDefault("cli.default_format", "standard")

	// Team defaults - generic software engineering team
	viper.SetDefault("team.tech_stack", []string{"JavaScript", "Python", "React"})
	viper.SetDefault("team.current_challenges", []string{"Performance optimization", "Code quality", "Scalability"})
	viper.SetDefault("team.interests", []string{"Best practices", "Developer experience", "System design"})
	viper.SetDefault("team.product_type", "Web application")
	viper.SetDefault("team.company_size", "startup")
	viper.SetDefault("team.industry", "technology")
	viper.SetDefault("team.expertise_level", "mid")
	viper.SetDefault("team.working_style", "agile")
	viper.SetDefault("team.team_size", 5)
	viper.SetDefault("team.priority", "features")

	// Observability defaults
	viper.SetDefault("observability.langfuse.enabled", false)
	viper.SetDefault("observability.langfuse.host", "https://cloud.langfuse.com")
	viper.SetDefault("observability.posthog.enabled", false)
	viper.SetDefault("observability.posthog.host", "https://app.posthog.com")

	// Themes defaults
	viper.SetDefault("themes.enabled", true)
	viper.SetDefault("themes.min_relevance_score", 0.6)
	viper.SetDefault("themes.classification_model", "gemini-flash-lite-latest")
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

	// LangFuse observability
	bindEnvKeys("observability.langfuse.public_key", []string{
		"LANGFUSE_PUBLIC_KEY",
		"LANGFUSE_PK",
	})

	bindEnvKeys("observability.langfuse.secret_key", []string{
		"LANGFUSE_SECRET_KEY",
		"LANGFUSE_SK",
	})

	bindEnvKeys("observability.langfuse.host", []string{
		"LANGFUSE_HOST",
		"LANGFUSE_URL",
	})

	// PostHog analytics
	bindEnvKeys("observability.posthog.api_key", []string{
		"POSTHOG_API_KEY",
		"POSTHOG_KEY",
	})

	bindEnvKeys("observability.posthog.host", []string{
		"POSTHOG_HOST",
		"POSTHOG_URL",
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
func GetApp() App                     { return Get().App }
func GetAI() AI                       { return Get().AI }
func GetDatabase() Database           { return Get().Database }
func GetServer() Server               { return Get().Server }
func GetSearch() Search               { return Get().Search }
func GetOutput() Output               { return Get().Output }
func GetCache() Cache                 { return Get().Cache }
func GetVisual() Visual               { return Get().Visual }
func GetTTS() TTS                     { return Get().TTS }
func GetMessaging() Messaging         { return Get().Messaging }
func GetEmail() Email                 { return Get().Email }
func GetFeeds() Feeds                 { return Get().Feeds }
func GetResearch() Research           { return Get().Research }
func GetFiltering() Filtering         { return Get().Filtering }
func GetTeam() Team                   { return Get().Team }
func GetLogging() Logging             { return Get().Logging }
func GetCLI() CLI                     { return Get().CLI }
func GetObservability() Observability { return Get().Observability }
func GetThemes() Themes               { return Get().Themes }

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

// Team context convenience getters
func GetTeamTechStack() []string  { return Get().Team.TechStack }
func GetTeamChallenges() []string { return Get().Team.CurrentChallenges }
func GetTeamInterests() []string  { return Get().Team.Interests }
func GetTeamProductType() string  { return Get().Team.ProductType }
func GetTeamPriority() string     { return Get().Team.Priority }
func GetTeamContext() Team        { return Get().Team }

// Observability convenience getters
func GetLangFuseConfig() LangFuseConfig { return Get().Observability.LangFuse }
func GetPostHogConfig() PostHogConfig   { return Get().Observability.PostHog }
func IsLangFuseEnabled() bool           { return Get().Observability.LangFuse.Enabled }
func IsPostHogEnabled() bool            { return Get().Observability.PostHog.Enabled }

// Themes convenience getters
func IsThemesEnabled() bool            { return Get().Themes.Enabled }
func GetThemeMinRelevance() float64    { return Get().Themes.MinRelevanceScore }
func GetThemeClassificationModel() string { return Get().Themes.ClassificationModel }

// GenerateTeamContextPrompt creates a formatted prompt string for LLM context
func GenerateTeamContextPrompt() string {
	team := GetTeamContext()

	var prompt strings.Builder
	prompt.WriteString("I'm sharing these links with my software engineering team. ")

	if len(team.TechStack) > 0 {
		prompt.WriteString(fmt.Sprintf("We work with %s, ", strings.Join(team.TechStack, ", ")))
	}

	if team.ProductType != "" {
		prompt.WriteString(fmt.Sprintf("and focus on %s. ", team.ProductType))
	}

	prompt.WriteString("Help me write concise \"Why it matters\" insights for each link.\n\n")
	prompt.WriteString("Context about our team:\n")

	if team.ProductType != "" {
		prompt.WriteString(fmt.Sprintf("- We build %s\n", team.ProductType))
	}

	if len(team.CurrentChallenges) > 0 {
		prompt.WriteString("- Current challenges: ")
		prompt.WriteString(strings.Join(team.CurrentChallenges, ", "))
		prompt.WriteString("\n")
	}

	if len(team.TechStack) > 0 {
		prompt.WriteString("- Tech stack: ")
		prompt.WriteString(strings.Join(team.TechStack, ", "))
		prompt.WriteString("\n")
	}

	if len(team.Interests) > 0 {
		prompt.WriteString("- Team interests: ")
		prompt.WriteString(strings.Join(team.Interests, ", "))
		prompt.WriteString("\n")
	}

	if team.Priority != "" {
		prompt.WriteString(fmt.Sprintf("- Current priority: %s\n", team.Priority))
	}

	return prompt.String()
}

// Filtering convenience getters
func IsFilteringEnabled() bool              { return Get().Filtering.Enabled }
func GetFilteringMinRelevance() float64     { return Get().Filtering.MinRelevance }
func GetFilteringMethod() string            { return Get().Filtering.Method }
func GetFilteringWeights() FilteringWeights { return Get().Filtering.Weights }

// Get template-specific filtering settings
func GetTemplateFilter(format string) TemplateFilter {
	templates := Get().Filtering.Templates
	switch format {
	case "brief":
		return templates.Brief
	case "standard":
		return templates.Standard
	case "detailed":
		return templates.Detailed
	case "newsletter":
		return templates.Newsletter
	case "email":
		return templates.Email
	default:
		return templates.Standard // Default fallback
	}
}

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

// SaveTeamContextOverride saves team context overrides to a temporary config file
// This allows TUI changes to override the main config without modifying the original file
func SaveTeamContextOverride(teamContext Team) error {
	// Create/update the team context override in viper
	viper.Set("team.tech_stack", teamContext.TechStack)
	viper.Set("team.current_challenges", teamContext.CurrentChallenges)
	viper.Set("team.interests", teamContext.Interests)
	viper.Set("team.product_type", teamContext.ProductType)
	viper.Set("team.company_size", teamContext.CompanySize)
	viper.Set("team.industry", teamContext.Industry)
	viper.Set("team.expertise_level", teamContext.ExpertiseLevel)
	viper.Set("team.working_style", teamContext.WorkingStyle)
	viper.Set("team.team_size", teamContext.TeamSize)
	viper.Set("team.priority", teamContext.Priority)

	// Update the global config if it exists
	if globalConfig != nil {
		globalConfig.Team = teamContext
	}

	// Write the updated config to an override file
	overrideFile := ".briefly-override.yaml"
	if err := viper.WriteConfigAs(overrideFile); err != nil {
		return fmt.Errorf("failed to write team context override: %w", err)
	}

	return nil
}
