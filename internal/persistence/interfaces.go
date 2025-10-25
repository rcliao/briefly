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
}
