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

// ArticleCategorizer assigns articles to categories
type ArticleCategorizer interface {
	// CategorizeArticle assigns a category to an article based on its content
	// Returns category name (e.g., "Platform Updates", "From the Field")
	CategorizeArticle(ctx context.Context, article *core.Article, summary *core.Summary) (string, error)
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

// CitationTracker handles citation extraction and storage (Phase 1)
type CitationTracker interface {
	// TrackArticle creates a citation record for an article
	TrackArticle(ctx context.Context, article *core.Article) (*core.Citation, error)

	// TrackBatch creates citation records for multiple articles
	TrackBatch(ctx context.Context, articles []core.Article) (map[string]*core.Citation, error)

	// GetCitation retrieves a citation by article ID
	GetCitation(ctx context.Context, articleID string) (*core.Citation, error)
}

// DigestRepository handles digest persistence (v2.0)
// This interface provides a subset of persistence.DigestRepository needed by the pipeline
type DigestRepository interface {
	// StoreWithRelationships stores a digest with article and theme relationships
	// Performs all operations in a transaction for atomicity
	StoreWithRelationships(ctx context.Context, digest *core.Digest, articleIDs []string, themeIDs []string) error
}

// ArticleRepository provides persistence for articles (Phase 1)
type ArticleRepository interface {
	// UpdateClusterAssignment updates cluster assignment for an article
	UpdateClusterAssignment(ctx context.Context, articleID string, clusterLabel string, confidence float64) error

	// UpdateEmbedding updates the embedding vector for an article
	// This is called after generating embeddings to persist them for semantic search
	UpdateEmbedding(ctx context.Context, articleID string, embedding []float64) error
}

// TagClassifier assigns multiple tags to articles (Phase 1)
type TagClassifier interface {
	// ClassifyArticle assigns 3-5 most relevant tags to an article
	ClassifyArticle(ctx context.Context, article core.Article, summary *core.Summary, tags []core.Tag, minRelevance float64) (*TagClassificationResult, error)

	// ClassifyWithinTheme classifies an article using only tags from a specific theme
	ClassifyWithinTheme(ctx context.Context, article core.Article, summary *core.Summary, themeID string, allTags []core.Tag, minRelevance float64) (*TagClassificationResult, error)

	// ClassifyBatch classifies multiple articles in batch
	ClassifyBatch(ctx context.Context, articles []core.Article, summaries map[string]*core.Summary, tags []core.Tag, minRelevance float64) (map[string]*TagClassificationResult, error)
}

// TagClassificationResult contains all tag classifications for an article
type TagClassificationResult struct {
	ArticleID string                        // Article being classified
	Tags      []TagClassificationResultItem // Assigned tags (3-5 recommended)
	ThemeID   string                        // Parent theme (for filtering)
}

// TagClassificationResultItem contains a single tag classification
type TagClassificationResultItem struct {
	TagID          string  // ID of the matched tag (e.g., "tag-llm")
	TagName        string  // Name of the matched tag (e.g., "Large Language Models")
	RelevanceScore float64 // Relevance score (0.0-1.0)
	Reasoning      string  // Why this tag was chosen
}

// TagRepository handles tag persistence operations (Phase 1)
type TagRepository interface {
	// ListEnabled retrieves all enabled tags
	ListEnabled(ctx context.Context) ([]core.Tag, error)

	// AssignTagsToArticle assigns multiple tags to an article (batch operation)
	AssignTagsToArticle(ctx context.Context, articleID string, tags map[string]float64) error

	// GetArticleTags retrieves all tags assigned to an article (with relevance scores)
	GetArticleTags(ctx context.Context, articleID string) ([]core.Tag, map[string]float64, error)
}

// VectorStore provides semantic search operations for article embeddings (Phase 2)
// Using pgvector for production-scale similarity search with cosine distance
type VectorStore interface {
	// Store saves or updates an embedding for an article
	Store(ctx context.Context, articleID string, embedding []float64) error

	// Search finds articles similar to the query embedding
	Search(ctx context.Context, query VectorSearchQuery) ([]VectorSearchResult, error)

	// SearchByTag performs semantic search within a specific tag (KEY for tag-aware clustering)
	SearchByTag(ctx context.Context, query VectorSearchQuery, tagID string) ([]VectorSearchResult, error)

	// SearchByTags searches across multiple tags (OR operation)
	SearchByTags(ctx context.Context, query VectorSearchQuery, tagIDs []string) ([]VectorSearchResult, error)

	// Delete removes an embedding
	Delete(ctx context.Context, articleID string) error

	// CreateIndex creates pgvector indexes for performance
	CreateIndex(ctx context.Context) error

	// GetStats returns statistics about the vector store
	GetStats(ctx context.Context) (*VectorStoreStats, error)
}

// VectorSearchQuery configures semantic search parameters (Phase 2)
type VectorSearchQuery struct {
	Embedding           []float64 // Query vector (768-dim for Gemini)
	Limit               int       // Max results (default: 10)
	SimilarityThreshold float64   // Min cosine similarity (default: 0.7)
	IncludeArticle      bool      // Populate full article data
	ExcludeIDs          []string  // Filter out specific articles
}

// VectorSearchResult contains a similar article and its similarity score (Phase 2)
type VectorSearchResult struct {
	ArticleID  string        // Article ID
	Similarity float64       // Cosine similarity (0.0-1.0)
	Article    *core.Article // Full article data (if IncludeArticle=true)
	TagIDs     []string      // Assigned tags
	Distance   float64       // Raw cosine distance
}

// VectorStoreStats provides metrics about the vector store (Phase 2)
type VectorStoreStats struct {
	TotalEmbeddings   int64   // Count of stored embeddings
	EmbeddingDimensions int     // Vector size (768 for Gemini)
	IndexType         string  // pgvector index type (ivfflat, hnsw)
	IndexSize         int64   // Disk space used by indexes
	AvgSearchLatency  float64 // Average search query time (ms)
}
