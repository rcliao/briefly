// Package sources provides feed source management and aggregation
package sources

import (
	"briefly/internal/core"
	"briefly/internal/feeds"
	"briefly/internal/logger"
	"briefly/internal/persistence"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Manager handles feed source management and article discovery
type Manager struct {
	db          persistence.Database
	feedManager *feeds.FeedManager
	log         *slog.Logger
}

// NewManager creates a new source manager
func NewManager(db persistence.Database) *Manager {
	return &Manager{
		db:          db,
		feedManager: feeds.NewFeedManager(),
		log:         logger.Get(),
	}
}

// AddFeed adds a new RSS/Atom feed source
func (m *Manager) AddFeed(ctx context.Context, feedURL string) (*core.Feed, error) {
	// Check if feed already exists
	existingFeed, err := m.db.Feeds().GetByURL(ctx, feedURL)
	if err == nil {
		return existingFeed, fmt.Errorf("feed already exists with ID: %s", existingFeed.ID)
	}

	// Validate and fetch feed
	parsedFeed, err := m.feedManager.FetchFeed(feedURL, "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to validate feed: %w", err)
	}

	// Store feed in database
	if err := m.db.Feeds().Create(ctx, &parsedFeed.Feed); err != nil {
		return nil, fmt.Errorf("failed to store feed: %w", err)
	}

	m.log.Info("Added new feed", "id", parsedFeed.Feed.ID, "title", parsedFeed.Feed.Title)
	return &parsedFeed.Feed, nil
}

// RemoveFeed removes a feed source by ID
func (m *Manager) RemoveFeed(ctx context.Context, feedID string) error {
	if err := m.db.Feeds().Delete(ctx, feedID); err != nil {
		return fmt.Errorf("failed to remove feed: %w", err)
	}

	m.log.Info("Removed feed", "id", feedID)
	return nil
}

// ListFeeds returns all registered feeds
func (m *Manager) ListFeeds(ctx context.Context, activeOnly bool) ([]core.Feed, error) {
	if activeOnly {
		return m.db.Feeds().ListActive(ctx)
	}
	return m.db.Feeds().List(ctx, persistence.ListOptions{Limit: 1000})
}

// ToggleFeed activates or deactivates a feed
func (m *Manager) ToggleFeed(ctx context.Context, feedID string, active bool) error {
	feed, err := m.db.Feeds().Get(ctx, feedID)
	if err != nil {
		return fmt.Errorf("feed not found: %w", err)
	}

	feed.Active = active
	if err := m.db.Feeds().Update(ctx, feed); err != nil {
		return fmt.Errorf("failed to update feed: %w", err)
	}

	m.log.Info("Toggled feed", "id", feedID, "active", active)
	return nil
}

// AggregateOptions configures the aggregation process
type AggregateOptions struct {
	MaxArticlesPerFeed int           // Limit articles per feed (0 = no limit)
	MaxConcurrency     int           // Number of feeds to fetch concurrently
	Since              time.Time     // Only fetch items published after this date
	Timeout            time.Duration // Timeout for entire aggregation
}

// DefaultAggregateOptions returns sensible defaults
func DefaultAggregateOptions() AggregateOptions {
	return AggregateOptions{
		MaxArticlesPerFeed: 50,
		MaxConcurrency:     5,
		Since:              time.Now().Add(-24 * time.Hour), // Last 24 hours
		Timeout:            10 * time.Minute,
	}
}

// AggregateResult contains aggregation statistics
type AggregateResult struct {
	FeedsFetched     int
	FeedsSkipped     int
	FeedsFailed      int
	NewArticles      int
	DuplicateArticles int
	Errors           []error
}

// Aggregate fetches new articles from all active feeds
func (m *Manager) Aggregate(ctx context.Context, opts AggregateOptions) (*AggregateResult, error) {
	// Create timeout context
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Get all active feeds
	feeds, err := m.db.Feeds().ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list active feeds: %w", err)
	}

	if len(feeds) == 0 {
		m.log.Warn("No active feeds found")
		return &AggregateResult{}, nil
	}

	m.log.Info("Starting aggregation", "feed_count", len(feeds), "max_concurrency", opts.MaxConcurrency)

	// Process feeds with concurrency control
	result := &AggregateResult{}
	sem := make(chan struct{}, opts.MaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, feed := range feeds {
		select {
		case <-ctx.Done():
			m.log.Warn("Aggregation cancelled", "reason", ctx.Err())
			return result, ctx.Err()
		default:
		}

		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(f core.Feed) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			feedResult := m.processFeed(ctx, f, opts)

			mu.Lock()
			result.FeedsFetched += feedResult.FeedsFetched
			result.FeedsSkipped += feedResult.FeedsSkipped
			result.FeedsFailed += feedResult.FeedsFailed
			result.NewArticles += feedResult.NewArticles
			result.DuplicateArticles += feedResult.DuplicateArticles
			result.Errors = append(result.Errors, feedResult.Errors...)
			mu.Unlock()
		}(feed)
	}

	wg.Wait()

	m.log.Info("Aggregation completed",
		"fetched", result.FeedsFetched,
		"skipped", result.FeedsSkipped,
		"failed", result.FeedsFailed,
		"new_articles", result.NewArticles,
		"duplicates", result.DuplicateArticles,
	)

	return result, nil
}

// processFeed fetches and stores items from a single feed
func (m *Manager) processFeed(ctx context.Context, feed core.Feed, opts AggregateOptions) *AggregateResult {
	result := &AggregateResult{}

	// Fetch feed with conditional GET
	parsedFeed, err := m.feedManager.FetchFeed(feed.URL, feed.LastModified, feed.ETag)
	if err != nil {
		m.log.Error("Failed to fetch feed", "feed_id", feed.ID, "error", err)
		result.FeedsFailed++
		result.Errors = append(result.Errors, fmt.Errorf("feed %s: %w", feed.ID, err))

		// Update error count
		feed.ErrorCount++
		feed.LastError = err.Error()
		_ = m.db.Feeds().Update(ctx, &feed)
		return result
	}

	// Check if feed was modified
	if parsedFeed.NotModified {
		m.log.Debug("Feed not modified since last fetch", "feed_id", feed.ID)
		result.FeedsSkipped++
		return result
	}

	result.FeedsFetched++

	// Filter items by publication date
	var newItems []core.FeedItem
	for _, item := range parsedFeed.Items {
		// Skip items older than the "since" threshold
		if !opts.Since.IsZero() && item.Published.Before(opts.Since) {
			continue
		}

		newItems = append(newItems, item)

		// Respect max articles per feed limit
		if opts.MaxArticlesPerFeed > 0 && len(newItems) >= opts.MaxArticlesPerFeed {
			break
		}
	}

	// Store feed items
	if len(newItems) > 0 {
		if err := m.db.FeedItems().CreateBatch(ctx, newItems); err != nil {
			m.log.Error("Failed to store feed items", "feed_id", feed.ID, "error", err)
			result.Errors = append(result.Errors, fmt.Errorf("store items for %s: %w", feed.ID, err))
		} else {
			result.NewArticles += len(newItems)
			m.log.Info("Stored feed items", "feed_id", feed.ID, "count", len(newItems))
		}
	}

	// Update feed metadata
	if err := m.db.Feeds().UpdateLastFetched(ctx, feed.ID, parsedFeed.LastModified, parsedFeed.ETag); err != nil {
		m.log.Error("Failed to update feed metadata", "feed_id", feed.ID, "error", err)
	}

	// Reset error count on successful fetch
	if feed.ErrorCount > 0 {
		feed.ErrorCount = 0
		feed.LastError = ""
		_ = m.db.Feeds().Update(ctx, &feed)
	}

	return result
}

// GetUnprocessedItems returns feed items that haven't been processed yet
func (m *Manager) GetUnprocessedItems(ctx context.Context, limit int) ([]core.FeedItem, error) {
	return m.db.FeedItems().GetUnprocessed(ctx, limit)
}

// MarkItemProcessed marks a feed item as processed
func (m *Manager) MarkItemProcessed(ctx context.Context, itemID string) error {
	return m.db.FeedItems().MarkProcessed(ctx, itemID)
}

// GetFeedStats returns statistics for a specific feed
func (m *Manager) GetFeedStats(ctx context.Context, feedID string) (*FeedStats, error) {
	feed, err := m.db.Feeds().Get(ctx, feedID)
	if err != nil {
		return nil, fmt.Errorf("feed not found: %w", err)
	}

	items, err := m.db.FeedItems().GetByFeedID(ctx, feedID, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to get feed items: %w", err)
	}

	stats := &FeedStats{
		Feed:       *feed,
		TotalItems: len(items),
	}

	for _, item := range items {
		if item.Processed {
			stats.ProcessedItems++
		}
		if !item.Published.IsZero() && (stats.LatestItem.IsZero() || item.Published.After(stats.LatestItem)) {
			stats.LatestItem = item.Published
		}
		if !item.Published.IsZero() && (stats.OldestItem.IsZero() || item.Published.Before(stats.OldestItem)) {
			stats.OldestItem = item.Published
		}
	}

	stats.UnprocessedItems = stats.TotalItems - stats.ProcessedItems

	return stats, nil
}

// FeedStats contains statistics for a feed
type FeedStats struct {
	Feed             core.Feed
	TotalItems       int
	ProcessedItems   int
	UnprocessedItems int
	LatestItem       time.Time
	OldestItem       time.Time
}

// ManualURLResult contains statistics for manual URL processing
type ManualURLResult struct {
	URLsProcessed int
	URLsFailed    int
	Errors        []error
}

// AggregateManualURLs processes all pending manual URLs
func (m *Manager) AggregateManualURLs(ctx context.Context, maxURLs int) (*ManualURLResult, error) {
	result := &ManualURLResult{}

	// Get pending URLs
	urls, err := m.db.ManualURLs().GetPending(ctx, maxURLs)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending URLs: %w", err)
	}

	if len(urls) == 0 {
		m.log.Info("No pending manual URLs to process")
		return result, nil
	}

	m.log.Info("Processing manual URLs", "count", len(urls))

	// Process each URL
	for _, manualURL := range urls {
		select {
		case <-ctx.Done():
			m.log.Warn("Manual URL processing cancelled", "reason", ctx.Err())
			return result, ctx.Err()
		default:
		}

		// Mark as processing
		if err := m.db.ManualURLs().UpdateStatus(ctx, manualURL.ID, string(core.ManualURLStatusProcessing), ""); err != nil {
			m.log.Error("Failed to update URL status", "url", manualURL.URL, "error", err)
			continue
		}

		// Create a feed item for this URL
		feedItem := &core.FeedItem{
			ID:             manualURL.ID,
			FeedID:         "manual", // Special feed ID for manual URLs
			Title:          manualURL.URL,
			Link:           manualURL.URL,
			Description:    fmt.Sprintf("Manually submitted by %s", manualURL.SubmittedBy),
			Published:      manualURL.CreatedAt,
			GUID:           manualURL.URL,
			Processed:      false,
			DateDiscovered: manualURL.CreatedAt,
		}

		// Store feed item
		if err := m.db.FeedItems().Create(ctx, feedItem); err != nil {
			m.log.Error("Failed to store feed item", "url", manualURL.URL, "error", err)
			result.URLsFailed++
			result.Errors = append(result.Errors, fmt.Errorf("failed to store %s: %w", manualURL.URL, err))

			// Mark as failed
			_ = m.db.ManualURLs().MarkFailed(ctx, manualURL.ID, err.Error())
			continue
		}

		// Mark as processed
		if err := m.db.ManualURLs().MarkProcessed(ctx, manualURL.ID); err != nil {
			m.log.Error("Failed to mark URL as processed", "url", manualURL.URL, "error", err)
		}

		result.URLsProcessed++
		m.log.Info("Processed manual URL", "url", manualURL.URL)
	}

	m.log.Info("Manual URL processing completed",
		"processed", result.URLsProcessed,
		"failed", result.URLsFailed,
	)

	return result, nil
}
