package pipeline

import (
	"briefly/internal/core"
	"context"
	"time"
)

// URLParser extracts and validates URLs from markdown files
type URLParser interface {
	// ParseMarkdownFile reads a markdown file and extracts all URLs
	ParseMarkdownFile(filePath string) ([]core.Link, error)

	// ParseMarkdownContent extracts URLs from markdown content string
	ParseMarkdownContent(content string) []string

	// ValidateURL checks if a URL is valid
	ValidateURL(url string) error

	// NormalizeURL removes tracking parameters and normalizes format
	NormalizeURL(url string) string
}

// ContentFetcher retrieves content from URLs
type ContentFetcher interface {
	// FetchArticle fetches and extracts content from a URL
	// Handles HTML, PDF, and YouTube content types
	FetchArticle(ctx context.Context, url string) (*core.Article, error)
}

// ArticleSummarizer generates summaries from articles
type ArticleSummarizer interface {
	// SummarizeArticle creates a structured summary with key points
	SummarizeArticle(ctx context.Context, article *core.Article) (*core.Summary, error)

	// GenerateKeyPoints extracts key points from content
	GenerateKeyPoints(ctx context.Context, content string) ([]string, error)

	// ExtractTitle generates or extracts a title from content
	ExtractTitle(ctx context.Context, content string) (string, error)
}

// EmbeddingGenerator creates vector embeddings for text
type EmbeddingGenerator interface {
	// GenerateEmbedding creates a 768-dimensional embedding vector
	GenerateEmbedding(ctx context.Context, text string) ([]float64, error)

	// GenerateEmbeddings batch processes multiple texts
	GenerateEmbeddings(ctx context.Context, texts []string) ([][]float64, error)
}

// TopicClusterer groups similar articles
type TopicClusterer interface {
	// ClusterArticles groups articles by topic similarity
	// Returns clusters with similarity threshold of 0.7
	ClusterArticles(ctx context.Context, articles []core.Article, summaries []core.Summary, embeddings map[string][]float64) ([]core.TopicCluster, error)

	// CalculateSimilarity computes cosine similarity between embeddings
	CalculateSimilarity(embedding1, embedding2 []float64) float64
}

// ArticleOrderer organizes articles for optimal reading
type ArticleOrderer interface {
	// OrderClusters orders clusters by importance and articles within clusters
	OrderClusters(ctx context.Context, clusters []core.TopicCluster, articles []core.Article) ([]core.TopicCluster, error)

	// OrderArticlesInCluster orders articles within a single cluster
	OrderArticlesInCluster(cluster *core.TopicCluster, articles []core.Article) error
}

// NarrativeGenerator creates executive summaries
type NarrativeGenerator interface {
	// GenerateExecutiveSummary creates a story-driven narrative from clusters
	// Takes top 3 articles from each cluster and synthesizes into 200 words
	GenerateExecutiveSummary(ctx context.Context, clusters []core.TopicCluster, articles map[string]core.Article, summaries map[string]core.Summary) (string, error)

	// IdentifyClusterTheme generates a descriptive theme for a cluster
	IdentifyClusterTheme(ctx context.Context, cluster core.TopicCluster, articles []core.Article) (string, error)

	// SelectTopArticles selects the top N articles from a cluster
	SelectTopArticles(cluster core.TopicCluster, articles []core.Article, n int) []core.Article
}

// MarkdownRenderer formats output as markdown
type MarkdownRenderer interface {
	// RenderDigest renders a complete digest to markdown file
	RenderDigest(ctx context.Context, digest *core.Digest, outputPath string) (string, error)

	// RenderQuickRead renders a single article summary to markdown
	RenderQuickRead(ctx context.Context, article *core.Article, summary *core.Summary) (string, error)

	// FormatForLinkedIn applies LinkedIn-specific formatting
	FormatForLinkedIn(markdown string) string
}

// CacheManager handles content caching
type CacheManager interface {
	// GetArticleWithSummary retrieves cached article and summary
	GetArticleWithSummary(url string, ttl time.Duration) (*core.Article, *core.Summary, error)

	// StoreArticleWithSummary caches article and summary
	StoreArticleWithSummary(article *core.Article, summary *core.Summary, ttl time.Duration) error

	// GetCachedArticle retrieves just the article (legacy compatibility)
	GetCachedArticle(url string, ttl time.Duration) (*core.Article, error)

	// CacheArticle stores just the article (legacy compatibility)
	CacheArticle(article *core.Article, ttl time.Duration) error

	// Clear removes all cached data
	Clear() error

	// Stats returns cache statistics
	Stats() (*core.CacheStats, error)
}

// BannerGenerator creates banner images (optional)
type BannerGenerator interface {
	// GenerateBanner creates a social media banner image
	GenerateBanner(ctx context.Context, digest *core.Digest, style string) (string, error)

	// AnalyzeThemes identifies content themes for banner generation
	AnalyzeThemes(digest *core.Digest) ([]core.ContentTheme, error)
}