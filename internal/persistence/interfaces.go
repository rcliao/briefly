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

	// Get retrieves a digest by ID
	Get(ctx context.Context, id string) (*core.Digest, error)

	// GetByDate retrieves a digest for a specific date
	GetByDate(ctx context.Context, date time.Time) (*core.Digest, error)

	// List retrieves digests with pagination
	List(ctx context.Context, opts ListOptions) ([]core.Digest, error)

	// Update updates an existing digest
	Update(ctx context.Context, digest *core.Digest) error

	// Delete removes a digest by ID
	Delete(ctx context.Context, id string) error

	// GetLatest retrieves the most recent digests
	GetLatest(ctx context.Context, limit int) ([]core.Digest, error)
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
}
