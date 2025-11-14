// Package persistence provides database abstraction interfaces for storing articles, feeds, and digests
package persistence

import (
	"briefly/internal/core"
	"context"
	"time"
)

// ArticleRepository handles article persistence operations
type ArticleRepository interface {
	// Create inserts a new article
	Create(ctx context.Context, article *core.Article) error

	// Get retrieves an article by ID
	Get(ctx context.Context, id string) (*core.Article, error)

	// GetByURL retrieves an article by its URL
	GetByURL(ctx context.Context, url string) (*core.Article, error)

	// List retrieves articles with pagination and filtering
	List(ctx context.Context, opts ListOptions) ([]core.Article, error)

	// Update updates an existing article
	Update(ctx context.Context, article *core.Article) error

	// Delete removes an article by ID
	Delete(ctx context.Context, id string) error

	// GetRecent retrieves articles published after a given date
	GetRecent(ctx context.Context, since time.Time, limit int) ([]core.Article, error)

	// GetByCluster retrieves articles belonging to a specific cluster
	GetByCluster(ctx context.Context, clusterLabel string, limit int) ([]core.Article, error)

	// UpdateClusterAssignment updates the cluster assignment for an article (Phase 1)
	// This is called after clustering to persist cluster labels and confidence scores
	UpdateClusterAssignment(ctx context.Context, articleID string, clusterLabel string, confidence float64) error

	// UpdateEmbedding updates the embedding vector for an article
	// This is called after generating embeddings to persist them for semantic search
	UpdateEmbedding(ctx context.Context, articleID string, embedding []float64) error
}

// SummaryRepository handles summary persistence operations
type SummaryRepository interface {
	// Create inserts a new summary
	Create(ctx context.Context, summary *core.Summary) error

	// Get retrieves a summary by ID
	Get(ctx context.Context, id string) (*core.Summary, error)

	// GetByArticleID retrieves summaries for a specific article
	GetByArticleID(ctx context.Context, articleID string) ([]core.Summary, error)

	// List retrieves summaries with pagination
	List(ctx context.Context, opts ListOptions) ([]core.Summary, error)

	// Update updates an existing summary
	Update(ctx context.Context, summary *core.Summary) error

	// Delete removes a summary by ID
	Delete(ctx context.Context, id string) error
}

// FeedRepository handles RSS/Atom feed persistence operations
type FeedRepository interface {
	// Create inserts a new feed
	Create(ctx context.Context, feed *core.Feed) error

	// Get retrieves a feed by ID
	Get(ctx context.Context, id string) (*core.Feed, error)

	// GetByURL retrieves a feed by its URL
	GetByURL(ctx context.Context, url string) (*core.Feed, error)

	// ListActive retrieves all active feeds
	ListActive(ctx context.Context) ([]core.Feed, error)

	// List retrieves feeds with pagination
	List(ctx context.Context, opts ListOptions) ([]core.Feed, error)

	// Update updates an existing feed
	Update(ctx context.Context, feed *core.Feed) error

	// Delete removes a feed by ID
	Delete(ctx context.Context, id string) error

	// UpdateLastFetched updates the last fetched timestamp and caching headers
	UpdateLastFetched(ctx context.Context, id string, lastModified, etag string) error
}

// FeedItemRepository handles feed item persistence operations
type FeedItemRepository interface {
	// Create inserts a new feed item
	Create(ctx context.Context, item *core.FeedItem) error

	// CreateBatch inserts multiple feed items efficiently
	CreateBatch(ctx context.Context, items []core.FeedItem) error

	// Get retrieves a feed item by ID
	Get(ctx context.Context, id string) (*core.FeedItem, error)

	// GetByFeedID retrieves items for a specific feed
	GetByFeedID(ctx context.Context, feedID string, limit int) ([]core.FeedItem, error)

	// GetUnprocessed retrieves unprocessed feed items
	GetUnprocessed(ctx context.Context, limit int) ([]core.FeedItem, error)

	// List retrieves feed items with pagination
	List(ctx context.Context, opts ListOptions) ([]core.FeedItem, error)

	// MarkProcessed marks a feed item as processed
	MarkProcessed(ctx context.Context, id string) error

	// Delete removes a feed item by ID
	Delete(ctx context.Context, id string) error
}

// DigestRepository handles digest persistence operations
type DigestRepository interface {
	// Create inserts a new digest
	Create(ctx context.Context, digest *core.Digest) error

	// StoreWithRelationships stores a digest with article and theme relationships (v2.0)
	// This method handles:
	// - Inserting the digest
	// - Creating digest_articles relationships with citation order
	// - Creating digest_themes relationships
	// - Extracting and storing citations from summary markdown
	// All operations are performed in a transaction for atomicity
	StoreWithRelationships(ctx context.Context, digest *core.Digest, articleIDs []string, themeIDs []string) error

	// Get retrieves a digest by ID
	Get(ctx context.Context, id string) (*core.Digest, error)

	// GetWithArticles retrieves a digest with all associated articles loaded (v2.0)
	GetWithArticles(ctx context.Context, id string) (*core.Digest, error)

	// GetWithThemes retrieves a digest with all associated themes loaded (v2.0)
	GetWithThemes(ctx context.Context, id string) (*core.Digest, error)

	// GetFull retrieves a digest with articles, themes, and citations loaded (v2.0)
	GetFull(ctx context.Context, id string) (*core.Digest, error)

	// GetByDate retrieves a digest for a specific date (legacy, v1.0 behavior)
	GetByDate(ctx context.Context, date time.Time) (*core.Digest, error)

	// ListRecent retrieves digests processed since a given date (v2.0)
	// Used for homepage digest list with time window filtering
	ListRecent(ctx context.Context, since time.Time, limit int) ([]core.Digest, error)

	// ListByTheme retrieves digests associated with a specific theme (v2.0)
	// Used for theme-based filtering on homepage
	ListByTheme(ctx context.Context, themeID string, since time.Time, limit int) ([]core.Digest, error)

	// ListByCluster retrieves digests for a specific HDBSCAN cluster (v2.0)
	ListByCluster(ctx context.Context, clusterID int, limit int) ([]core.Digest, error)

	// List retrieves digests with pagination
	List(ctx context.Context, opts ListOptions) ([]core.Digest, error)

	// Update updates an existing digest
	Update(ctx context.Context, digest *core.Digest) error

	// Delete removes a digest by ID (also removes relationships via CASCADE)
	Delete(ctx context.Context, id string) error

	// GetLatest retrieves the most recent digests
	GetLatest(ctx context.Context, limit int) ([]core.Digest, error)

	// GetByID retrieves a digest by ID (alias for Get, for consistency)
	GetByID(ctx context.Context, id string) (*core.Digest, error)

	// GetDigestArticles retrieves all articles associated with a digest
	GetDigestArticles(ctx context.Context, digestID string) ([]core.Article, error)
}

// ThemeRepository handles theme persistence operations (Phase 0)
type ThemeRepository interface {
	// Create inserts a new theme
	Create(ctx context.Context, theme *core.Theme) error

	// Get retrieves a theme by ID
	Get(ctx context.Context, id string) (*core.Theme, error)

	// GetByName retrieves a theme by its name
	GetByName(ctx context.Context, name string) (*core.Theme, error)

	// List retrieves all themes with optional enabled filter
	List(ctx context.Context, enabledOnly bool) ([]core.Theme, error)

	// Update updates an existing theme
	Update(ctx context.Context, theme *core.Theme) error

	// Delete removes a theme by ID
	Delete(ctx context.Context, id string) error

	// ListEnabled retrieves all enabled themes
	ListEnabled(ctx context.Context) ([]core.Theme, error)
}

// ManualURLRepository handles manual URL submission persistence operations (Phase 0)
type ManualURLRepository interface {
	// Create inserts a new manual URL
	Create(ctx context.Context, manualURL *core.ManualURL) error

	// CreateBatch inserts multiple manual URLs efficiently
	CreateBatch(ctx context.Context, urls []string, submittedBy string) error

	// Get retrieves a manual URL by ID
	Get(ctx context.Context, id string) (*core.ManualURL, error)

	// List retrieves manual URLs with pagination
	List(ctx context.Context, opts ListOptions) ([]core.ManualURL, error)

	// GetPending retrieves all pending manual URLs
	GetPending(ctx context.Context, limit int) ([]core.ManualURL, error)

	// GetByURL retrieves a manual URL by its URL
	GetByURL(ctx context.Context, url string) (*core.ManualURL, error)

	// GetByStatus retrieves manual URLs by status
	GetByStatus(ctx context.Context, status string, limit int) ([]core.ManualURL, error)

	// UpdateStatus updates the status of a manual URL
	UpdateStatus(ctx context.Context, id string, status string, errorMessage string) error

	// MarkProcessed marks a manual URL as successfully processed
	MarkProcessed(ctx context.Context, id string) error

	// MarkFailed marks a manual URL as failed with an error message
	MarkFailed(ctx context.Context, id string, errorMessage string) error

	// Delete removes a manual URL by ID
	Delete(ctx context.Context, id string) error
}

// TagRepository handles tag persistence operations (Phase 1)
type TagRepository interface {
	// Create inserts a new tag
	Create(ctx context.Context, tag *core.Tag) error

	// Get retrieves a tag by ID
	Get(ctx context.Context, id string) (*core.Tag, error)

	// GetByName retrieves a tag by its name
	GetByName(ctx context.Context, name string) (*core.Tag, error)

	// List retrieves all tags with optional enabled filter
	List(ctx context.Context, enabledOnly bool) ([]core.Tag, error)

	// ListByTheme retrieves tags for a specific theme
	ListByTheme(ctx context.Context, themeID string, enabledOnly bool) ([]core.Tag, error)

	// Update updates an existing tag
	Update(ctx context.Context, tag *core.Tag) error

	// Delete removes a tag by ID
	Delete(ctx context.Context, id string) error

	// ListEnabled retrieves all enabled tags
	ListEnabled(ctx context.Context) ([]core.Tag, error)

	// AssignTagToArticle assigns a tag to an article with relevance score
	AssignTagToArticle(ctx context.Context, articleID string, tagID string, relevanceScore float64) error

	// AssignTagsToArticle assigns multiple tags to an article (batch operation)
	AssignTagsToArticle(ctx context.Context, articleID string, tags map[string]float64) error

	// GetArticleTags retrieves all tags assigned to an article (with relevance scores)
	GetArticleTags(ctx context.Context, articleID string) ([]core.Tag, map[string]float64, error)

	// GetTagArticles retrieves all articles assigned to a tag (with relevance scores)
	GetTagArticles(ctx context.Context, tagID string, minRelevance float64) ([]string, map[string]float64, error)

	// RemoveTagFromArticle removes a tag assignment from an article
	RemoveTagFromArticle(ctx context.Context, articleID string, tagID string) error

	// RemoveAllTagsFromArticle removes all tag assignments from an article
	RemoveAllTagsFromArticle(ctx context.Context, articleID string) error
}

// CitationRepository handles citation persistence operations (Phase 1)
// Updated in v2.0 to support both article metadata citations AND digest inline citations
type CitationRepository interface {
	// Create inserts a new citation
	Create(ctx context.Context, citation *core.Citation) error

	// CreateBatch inserts multiple citations efficiently (v2.0)
	// Used when storing digest citations extracted from summary markdown
	CreateBatch(ctx context.Context, citations []core.Citation) error

	// Get retrieves a citation by ID
	Get(ctx context.Context, id string) (*core.Citation, error)

	// GetByArticleID retrieves citation for a specific article (article metadata)
	GetByArticleID(ctx context.Context, articleID string) (*core.Citation, error)

	// GetByArticleIDs retrieves citations for multiple articles (batch lookup)
	GetByArticleIDs(ctx context.Context, articleIDs []string) (map[string]*core.Citation, error)

	// GetByDigestID retrieves all inline citations for a specific digest (v2.0)
	// Returns citations ordered by citation_number ([1], [2], [3], etc.)
	GetByDigestID(ctx context.Context, digestID string) ([]core.Citation, error)

	// List retrieves citations with pagination
	List(ctx context.Context, opts ListOptions) ([]core.Citation, error)

	// Update updates an existing citation
	Update(ctx context.Context, citation *core.Citation) error

	// Delete removes a citation by ID
	Delete(ctx context.Context, id string) error

	// DeleteByArticleID removes citation for a specific article
	DeleteByArticleID(ctx context.Context, articleID string) error

	// DeleteByDigestID removes all citations for a specific digest (v2.0)
	DeleteByDigestID(ctx context.Context, digestID string) error
}

// ListOptions provides common filtering and pagination options
type ListOptions struct {
	Limit  int               // Maximum number of results (0 for no limit)
	Offset int               // Number of results to skip
	SortBy string            // Field to sort by
	Order  string            // "asc" or "desc"
	Filter map[string]string // Key-value filters
}

// Database represents the main database interface that aggregates all repositories
type Database interface {
	// Articles returns the article repository
	Articles() ArticleRepository

	// Summaries returns the summary repository
	Summaries() SummaryRepository

	// Feeds returns the feed repository
	Feeds() FeedRepository

	// FeedItems returns the feed item repository
	FeedItems() FeedItemRepository

	// Digests returns the digest repository
	Digests() DigestRepository

	// Themes returns the theme repository (Phase 0)
	Themes() ThemeRepository

	// ManualURLs returns the manual URL repository (Phase 0)
	ManualURLs() ManualURLRepository

	// Citations returns the citation repository (Phase 1)
	Citations() CitationRepository

	// Tags returns the tag repository (Phase 1)
	Tags() TagRepository

	// Close closes the database connection
	Close() error

	// Ping verifies the database connection
	Ping(ctx context.Context) error

	// BeginTx starts a new transaction (optional, for implementations that support it)
	BeginTx(ctx context.Context) (Transaction, error)
}

// Transaction represents a database transaction
type Transaction interface {
	// Commit commits the transaction
	Commit() error

	// Rollback rolls back the transaction
	Rollback() error

	// Articles returns the article repository within this transaction
	Articles() ArticleRepository

	// Summaries returns the summary repository within this transaction
	Summaries() SummaryRepository

	// Feeds returns the feed repository within this transaction
	Feeds() FeedRepository

	// FeedItems returns the feed item repository within this transaction
	FeedItems() FeedItemRepository

	// Digests returns the digest repository within this transaction
	Digests() DigestRepository

	// Themes returns the theme repository within this transaction (Phase 0)
	Themes() ThemeRepository

	// ManualURLs returns the manual URL repository within this transaction (Phase 0)
	ManualURLs() ManualURLRepository

	// Tags returns the tag repository within this transaction (Phase 1)
	Tags() TagRepository
}
