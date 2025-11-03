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

// ClassificationOptions configures the classification process
type ClassificationOptions struct {
	MaxArticles    int     // Maximum number of articles to classify (0 = no limit)
	MinRelevance   float64 // Minimum relevance score to assign a theme (0.0-1.0)
	ThemeFilter    string  // Optional: Only classify articles matching this theme
	SkipProcessed  bool    // Skip articles that already have a theme assigned
	FetchContent   bool    // Whether to fetch full content for classification
	MaxConcurrency int     // Number of articles to process concurrently
}

// DefaultClassificationOptions returns sensible defaults
func DefaultClassificationOptions() ClassificationOptions {
	return ClassificationOptions{
		MaxArticles:    100,
		MinRelevance:   0.4, // 40% relevance threshold (same as Phase 0)
		ThemeFilter:    "",
		SkipProcessed:  true,
		FetchContent:   true,
		MaxConcurrency: 5,
	}
}

// ArticleClassificationResult contains classification statistics
type ArticleClassificationResult struct {
	ArticlesProcessed   int
	ArticlesClassified  int
	ArticlesFiltered    int // Below relevance threshold
	ArticlesFailed      int
	ThemeDistribution   map[string]int // theme_name -> count
	Errors              []error
}

// ClassifyFeedItems processes unprocessed feed items and classifies them by theme
// This is the Phase 1 RSS enhancement feature that adds theme-based filtering during aggregation
func (m *Manager) ClassifyFeedItems(ctx context.Context, processor ArticleProcessor, classifier ThemeClassifier, opts ClassificationOptions) (*ArticleClassificationResult, error) {
	result := &ArticleClassificationResult{
		ThemeDistribution: make(map[string]int),
	}

	// Get unprocessed feed items
	limit := opts.MaxArticles
	if limit == 0 {
		limit = 1000 // Default max
	}

	feedItems, err := m.db.FeedItems().GetUnprocessed(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get unprocessed feed items: %w", err)
	}

	if len(feedItems) == 0 {
		m.log.Info("No unprocessed feed items to classify")
		return result, nil
	}

	m.log.Info("Starting theme classification", "item_count", len(feedItems), "min_relevance", opts.MinRelevance)

	// Get available themes (enabled only)
	themes, err := m.db.Themes().List(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get active themes: %w", err)
	}

	if len(themes) == 0 {
		m.log.Warn("No active themes found - articles will be stored without classification")
	}

	// Process items with concurrency control
	sem := make(chan struct{}, opts.MaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, item := range feedItems {
		select {
		case <-ctx.Done():
			m.log.Warn("Classification cancelled", "reason", ctx.Err())
			return result, ctx.Err()
		default:
		}

		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(feedItem core.FeedItem) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			itemResult := m.classifyFeedItem(ctx, feedItem, processor, classifier, themes, opts)

			mu.Lock()
			result.ArticlesProcessed++
			if itemResult.Classified {
				result.ArticlesClassified++
				if itemResult.ThemeName != "" {
					result.ThemeDistribution[itemResult.ThemeName]++
				}
			} else if itemResult.Filtered {
				result.ArticlesFiltered++
			} else if itemResult.Error != nil {
				result.ArticlesFailed++
				result.Errors = append(result.Errors, itemResult.Error)
			}
			mu.Unlock()
		}(item)
	}

	wg.Wait()

	m.log.Info("Classification completed",
		"processed", result.ArticlesProcessed,
		"classified", result.ArticlesClassified,
		"filtered", result.ArticlesFiltered,
		"failed", result.ArticlesFailed,
	)

	return result, nil
}

// ItemClassificationResult contains results for a single item classification
type ItemClassificationResult struct {
	Classified bool
	Filtered   bool   // True if below relevance threshold
	ThemeName  string // Name of assigned theme
	Error      error
}

// classifyFeedItem processes a single feed item: fetches content, classifies, and stores
func (m *Manager) classifyFeedItem(ctx context.Context, item core.FeedItem, processor ArticleProcessor, classifier ThemeClassifier, themes []core.Theme, opts ClassificationOptions) ItemClassificationResult {
	result := ItemClassificationResult{}

	// Fetch and process article content
	article, err := processor.ProcessArticle(ctx, item.Link)
	if err != nil {
		m.log.Error("Failed to process article", "url", item.Link, "error", err)
		result.Error = fmt.Errorf("process %s: %w", item.Link, err)

		// Mark feed item as processed even if failed (avoid retry loops)
		_ = m.db.FeedItems().MarkProcessed(ctx, item.ID)
		return result
	}

	// Enrich article with feed item metadata
	if article.Title == "" {
		article.Title = item.Title
	}
	article.DatePublished = item.Published

	// Skip classification if no themes available
	if len(themes) == 0 {
		// Store article without theme
		if err := m.db.Articles().Create(ctx, article); err != nil {
			m.log.Error("Failed to store article", "url", item.Link, "error", err)
			result.Error = fmt.Errorf("store %s: %w", item.Link, err)
			return result
		}

		// Mark feed item as processed
		_ = m.db.FeedItems().MarkProcessed(ctx, item.ID)
		result.Classified = false
		return result
	}

	// Classify article
	bestMatch, err := classifier.GetBestMatch(ctx, *article, themes, opts.MinRelevance)
	if err != nil {
		m.log.Error("Failed to classify article", "url", item.Link, "error", err)
		result.Error = fmt.Errorf("classify %s: %w", item.Link, err)
		return result
	}

	// Check if article passes relevance threshold
	if bestMatch == nil {
		m.log.Debug("Article filtered (below relevance threshold)", "url", item.Link, "min_relevance", opts.MinRelevance)
		result.Filtered = true

		// Mark feed item as processed (but don't store article)
		_ = m.db.FeedItems().MarkProcessed(ctx, item.ID)
		return result
	}

	// Check theme filter if specified
	if opts.ThemeFilter != "" && bestMatch.GetThemeName() != opts.ThemeFilter {
		m.log.Debug("Article filtered (theme mismatch)", "url", item.Link, "theme", bestMatch.GetThemeName(), "filter", opts.ThemeFilter)
		result.Filtered = true

		// Mark feed item as processed
		_ = m.db.FeedItems().MarkProcessed(ctx, item.ID)
		return result
	}

	// Assign theme to article (use interface methods)
	themeID := bestMatch.GetThemeID()
	relevanceScore := bestMatch.GetRelevanceScore()
	article.ThemeID = &themeID
	article.ThemeRelevanceScore = &relevanceScore

	// Store article
	if err := m.db.Articles().Create(ctx, article); err != nil {
		m.log.Error("Failed to store article", "url", item.Link, "error", err)
		result.Error = fmt.Errorf("store %s: %w", item.Link, err)
		return result
	}

	// Mark feed item as processed
	if err := m.db.FeedItems().MarkProcessed(ctx, item.ID); err != nil {
		m.log.Error("Failed to mark feed item as processed", "id", item.ID, "error", err)
	}

	result.Classified = true
	result.ThemeName = bestMatch.GetThemeName()
	m.log.Info("Article classified and stored",
		"url", item.Link,
		"theme", bestMatch.GetThemeName(),
		"relevance", fmt.Sprintf("%.2f", bestMatch.GetRelevanceScore()),
	)

	return result
}

// ArticleProcessor interface for article content processing
type ArticleProcessor interface {
	ProcessArticle(ctx context.Context, url string) (*core.Article, error)
}

// ThemeClassifier interface for theme classification
// This is an abstract interface that can be implemented by themes.Classifier
type ThemeClassifier interface {
	GetBestMatch(ctx context.Context, article core.Article, themes []core.Theme, minRelevance float64) (ThemeClassificationResult, error)
}

// ThemeClassificationResult contains the result of theme classification
// This is defined to avoid import cycle with themes package
type ThemeClassificationResult interface {
	GetThemeID() string
	GetThemeName() string
	GetRelevanceScore() float64
	GetReasoning() string
}

// ClassificationResultAdapter adapts themes.ClassificationResult to ThemeClassificationResult
type ClassificationResultAdapter struct {
	ThemeID        string
	ThemeName      string
	RelevanceScore float64
	Reasoning      string
}

func (c ClassificationResultAdapter) GetThemeID() string {
	return c.ThemeID
}

func (c ClassificationResultAdapter) GetThemeName() string {
	return c.ThemeName
}

func (c ClassificationResultAdapter) GetRelevanceScore() float64 {
	return c.RelevanceScore
}

func (c ClassificationResultAdapter) GetReasoning() string {
	return c.Reasoning
}

// ThemeClassifierAdapter wraps a classifier with a concrete getBestMatch implementation
type ThemeClassifierAdapter struct {
	// Store the actual implementation as a function
	getBestMatchFunc func(ctx context.Context, article core.Article, themes []core.Theme, minRelevance float64) (ThemeClassificationResult, error)
}

// NewThemeClassifierAdapter creates an adapter from a classifier
// The classifier must have a GetBestMatch method that returns a result implementing ThemeClassificationResult
func NewThemeClassifierAdapter(classifier interface{}) *ThemeClassifierAdapter {
	// Define the expected interface
	type classifierWithGetBestMatch interface {
		GetBestMatch(ctx context.Context, article core.Article, themes []core.Theme, minRelevance float64) (ThemeClassificationResult, error)
	}

	// Try direct cast first
	if c, ok := classifier.(classifierWithGetBestMatch); ok {
		return &ThemeClassifierAdapter{
			getBestMatchFunc: c.GetBestMatch,
		}
	}

	// Otherwise, we need to wrap the result
	// This handles classifiers that return concrete types (like *themes.ClassificationResult)
	return &ThemeClassifierAdapter{
		getBestMatchFunc: func(ctx context.Context, article core.Article, themes []core.Theme, minRelevance float64) (ThemeClassificationResult, error) {
			// Use reflection-free approach: call the method directly via interface{}
			type anyResultClassifier interface {
				GetBestMatch(ctx context.Context, article core.Article, themes []core.Theme, minRelevance float64) (interface{}, error)
			}

			if c, ok := classifier.(anyResultClassifier); ok {
				result, err := c.GetBestMatch(ctx, article, themes, minRelevance)
				if err != nil {
					return nil, err
				}
				if result == nil {
					return nil, nil
				}
				// The result should implement ThemeClassificationResult
				if tcr, ok := result.(ThemeClassificationResult); ok {
					return tcr, nil
				}
			}

			panic(fmt.Sprintf("classifier %T does not implement expected GetBestMatch signature", classifier))
		},
	}
}

// GetBestMatch implements the ThemeClassifier interface
func (a *ThemeClassifierAdapter) GetBestMatch(ctx context.Context, article core.Article, themes []core.Theme, minRelevance float64) (ThemeClassificationResult, error) {
	return a.getBestMatchFunc(ctx, article, themes, minRelevance)
}
