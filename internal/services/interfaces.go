package services

import (
	"briefly/internal/core"
	"context"
)

// DigestService handles digest generation and processing
type DigestService interface {
	GenerateDigest(ctx context.Context, urls []string, format string) (*core.Digest, error)
	GenerateDigestFromFile(ctx context.Context, inputFile string, format string) (*core.Digest, error)
	ProcessSingleArticle(ctx context.Context, url string, format string) (*core.Article, *core.Summary, error)
}

// ArticleProcessor handles article fetching and processing
type ArticleProcessor interface {
	ProcessArticle(ctx context.Context, url string) (*core.Article, error)
	ProcessArticles(ctx context.Context, urls []string) ([]core.Article, error)
	CleanAndExtractContent(ctx context.Context, article *core.Article) error
}

// TemplateRenderer handles digest output generation
type TemplateRenderer interface {
	Render(ctx context.Context, digest *core.Digest, format string) (string, error)
	RenderToFile(ctx context.Context, digest *core.Digest, format string, outputPath string) error
	GetAvailableFormats() []string
}

// ResearchService handles content discovery and research
type ResearchService interface {
	PerformResearch(ctx context.Context, query string, depth int) (*core.ResearchReport, error)
	GenerateResearchQueries(ctx context.Context, article core.Article) ([]string, error)
	AnalyzeTopics(ctx context.Context, articles []core.Article) ([]string, error)
}

// FeedService handles RSS/Atom feed management
type FeedService interface {
	AddFeed(ctx context.Context, feedURL string) error
	ListFeeds(ctx context.Context) ([]core.Feed, error)
	RefreshFeeds(ctx context.Context) error
	AnalyzeFeedContent(ctx context.Context) (*core.FeedAnalysisReport, error)
	DiscoverFeeds(ctx context.Context, websiteURL string) ([]string, error)
}

// CacheService handles caching operations
type CacheService interface {
	GetCachedArticle(ctx context.Context, url string) (*core.Article, error)
	CacheArticle(ctx context.Context, article core.Article) error
	GetCachedSummary(ctx context.Context, url string, contentHash string) (*core.Summary, error)
	CacheSummary(ctx context.Context, summary core.Summary, url string, contentHash string) error
	ClearCache(ctx context.Context) error
	GetCacheStats(ctx context.Context) (*CacheStats, error)
}

// LLMService handles all LLM operations
type LLMService interface {
	SummarizeArticle(ctx context.Context, article core.Article, format string) (*core.Summary, error)
	GenerateDigestTitle(ctx context.Context, content string, format string) (string, error)
	GenerateEmbedding(ctx context.Context, text string) ([]float64, error)
	GenerateResearchQueries(ctx context.Context, article core.Article, depth int) ([]string, error)
	AnalyzeSentiment(ctx context.Context, text string) (float64, string, string, error)
}

// MessagingService handles multi-channel output
type MessagingService interface {
	SendSlackMessage(ctx context.Context, digest *core.Digest, webhookURL string, format string) error
	SendDiscordMessage(ctx context.Context, digest *core.Digest, webhookURL string, format string) error
	ValidateWebhookURL(platform string, url string) error
}

// TTSService handles text-to-speech generation
type TTSService interface {
	GenerateAudio(ctx context.Context, digest *core.Digest, config TTSConfig) (string, error)
	GetAvailableVoices(provider string) ([]string, error)
	EstimateAudioLength(text string, speed float64) float64
}

// CacheStats represents cache statistics
type CacheStats struct {
	ArticleCount       int
	SummaryCount       int
	DigestCount        int
	CacheSize          int64
	LastUpdated        string
	FeedCount          int
	ActiveFeedCount    int
	FeedItemCount      int
	ProcessedItemCount int
	TopicClusters      map[string]int
}

// TTSConfig represents TTS configuration
type TTSConfig struct {
	Provider    string
	Voice       string
	Speed       float64
	OutputDir   string
	MaxArticles int
}
