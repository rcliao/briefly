package services

import (
	"briefly/internal/core"
	"briefly/internal/feeds"
	"briefly/internal/llm"
	"briefly/internal/store"
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FeedServiceImpl implements the FeedService interface
type FeedServiceImpl struct {
	feedManager *feeds.FeedManager
	store       *store.Store
	llmClient   *llm.Client
}

// NewFeedService creates a new feed service
func NewFeedService(store *store.Store, llmClient *llm.Client) *FeedServiceImpl {
	return &FeedServiceImpl{
		feedManager: feeds.NewFeedManager(),
		store:       store,
		llmClient:   llmClient,
	}
}

// AddFeed subscribes to an RSS/Atom feed
func (f *FeedServiceImpl) AddFeed(ctx context.Context, feedURL string) error {
	// Validate the feed URL first
	if err := f.feedManager.ValidateFeedURL(feedURL); err != nil {
		return fmt.Errorf("invalid feed URL: %w", err)
	}

	// Fetch and parse the feed
	parsedFeed, err := f.feedManager.FetchFeed(feedURL, "", "")
	if err != nil {
		return fmt.Errorf("failed to fetch feed: %w", err)
	}

	if parsedFeed.NotModified {
		return fmt.Errorf("feed appears to be empty or inaccessible")
	}

	// Check if feed already exists
	existingFeeds, err := f.store.GetFeeds(false) // Get all feeds to check for duplicates
	if err == nil {
		for _, existing := range existingFeeds {
			if existing.URL == feedURL {
				return fmt.Errorf("feed already exists: %s", existing.Title)
			}
		}
	}

	// Store the feed and its items
	if err := f.store.AddFeed(parsedFeed.Feed); err != nil {
		return fmt.Errorf("failed to save feed: %w", err)
	}

	// Store initial feed items
	for _, item := range parsedFeed.Items {
		if err := f.store.AddFeedItem(item); err != nil {
			// Log error but don't fail the entire operation
			continue
		}
	}

	return nil
}

// ListFeeds returns all subscribed feeds
func (f *FeedServiceImpl) ListFeeds(ctx context.Context) ([]core.Feed, error) {
	feeds, err := f.store.GetFeeds(false) // Get all feeds, not just active
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve feeds: %w", err)
	}

	// Sort feeds by title for consistent output
	sort.Slice(feeds, func(i, j int) bool {
		return feeds[i].Title < feeds[j].Title
	})

	return feeds, nil
}

// RefreshFeeds updates all active feeds
func (f *FeedServiceImpl) RefreshFeeds(ctx context.Context) error {
	feeds, err := f.store.GetFeeds(true) // Get only active feeds
	if err != nil {
		return fmt.Errorf("failed to get feeds: %w", err)
	}

	var successCount, errorCount int
	for _, feed := range feeds {
		if !feed.Active {
			continue
		}

		if err := f.refreshSingleFeed(ctx, feed); err != nil {
			errorCount++
			// Update error count in feed
			feed.ErrorCount++
			feed.LastError = err.Error()
		} else {
			successCount++
			// Reset error count on success
			feed.ErrorCount = 0
			feed.LastError = ""
		}

		feed.LastFetched = time.Now().UTC()

		// Disable feed if too many consecutive errors
		if feed.ErrorCount >= 5 {
			feed.Active = false
		}

		// Update feed error info
		if feed.ErrorCount > 0 || feed.LastError != "" {
			if err := f.store.UpdateFeedError(feed.ID, feed.LastError); err != nil {
				// Log error but continue
			}
		}

		// Update feed status
		if err := f.store.SetFeedActive(feed.ID, feed.Active); err != nil {
			// Log error but continue
		}
	}

	if errorCount > 0 && successCount == 0 {
		return fmt.Errorf("failed to refresh any feeds (%d errors)", errorCount)
	}

	return nil
}

// refreshSingleFeed refreshes a single feed
func (f *FeedServiceImpl) refreshSingleFeed(ctx context.Context, feed core.Feed) error {
	parsedFeed, err := f.feedManager.FetchFeed(feed.URL, feed.LastModified, feed.ETag)
	if err != nil {
		return fmt.Errorf("failed to fetch feed %s: %w", feed.Title, err)
	}

	if parsedFeed.NotModified {
		// Feed hasn't changed, no new items
		return nil
	}

	// Update feed metadata
	if parsedFeed.LastModified != "" {
		feed.LastModified = parsedFeed.LastModified
	}
	if parsedFeed.ETag != "" {
		feed.ETag = parsedFeed.ETag
	}

	// Process new items
	newItemCount := 0
	for _, item := range parsedFeed.Items {
		// Check if item already exists
		exists, err := f.store.FeedItemExists(item.ID)
		if err != nil {
			continue
		}
		if exists {
			continue
		}

		// Save new item
		if err := f.store.SaveFeedItem(item); err != nil {
			continue
		}
		newItemCount++
	}

	return nil
}

// AnalyzeFeedContent analyzes recent feed content and generates a report
func (f *FeedServiceImpl) AnalyzeFeedContent(ctx context.Context) (*core.FeedAnalysisReport, error) {
	// Get all active feeds
	feeds, err := f.store.GetFeeds(false) // Get all feeds
	if err != nil {
		return nil, fmt.Errorf("failed to get feeds: %w", err)
	}

	activeFeeds := make([]core.Feed, 0)
	for _, feed := range feeds {
		if feed.Active {
			activeFeeds = append(activeFeeds, feed)
		}
	}

	if len(activeFeeds) == 0 {
		return &core.FeedAnalysisReport{
			ID:            uuid.New().String(),
			DateGenerated: time.Now().UTC(),
			Summary:       "No active feeds to analyze",
		}, nil
	}

	// Get recent feed items (last 7 days)
	since := time.Now().AddDate(0, 0, -7)
	recentItems, err := f.store.GetRecentFeedItems(since)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent feed items: %w", err)
	}

	if len(recentItems) == 0 {
		return &core.FeedAnalysisReport{
			ID:            uuid.New().String(),
			DateGenerated: time.Now().UTC(),
			FeedsAnalyzed: len(activeFeeds),
			Summary:       "No recent feed items to analyze",
		}, nil
	}

	// Analyze topics and trends
	topTopics, err := f.analyzeTopics(ctx, recentItems)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze topics: %w", err)
	}

	trendingKeywords := f.extractTrendingKeywords(recentItems)
	recommendedItems := f.selectRecommendedItems(recentItems, 10)
	qualityScore := f.calculateQualityScore(recentItems)

	// Generate overall summary
	summary, err := f.generateAnalysisSummary(ctx, recentItems, topTopics, trendingKeywords)
	if err != nil {
		summary = "Unable to generate analysis summary"
	}

	report := &core.FeedAnalysisReport{
		ID:               uuid.New().String(),
		DateGenerated:    time.Now().UTC(),
		FeedsAnalyzed:    len(activeFeeds),
		ItemsAnalyzed:    len(recentItems),
		TopTopics:        topTopics,
		TrendingKeywords: trendingKeywords,
		RecommendedItems: recommendedItems,
		QualityScore:     qualityScore,
		Summary:          summary,
	}

	return report, nil
}

// DiscoverFeeds auto-discovers RSS/Atom feeds from a website
func (f *FeedServiceImpl) DiscoverFeeds(ctx context.Context, websiteURL string) ([]string, error) {
	discoveredFeeds, err := f.feedManager.DiscoverFeedURL(websiteURL)
	if err != nil {
		return nil, fmt.Errorf("failed to discover feeds: %w", err)
	}

	return discoveredFeeds, nil
}

// analyzeTopics identifies common topics in feed items using LLM
func (f *FeedServiceImpl) analyzeTopics(ctx context.Context, items []core.FeedItem) ([]string, error) {
	if len(items) == 0 {
		return []string{}, nil
	}

	// Prepare content for analysis
	var content strings.Builder
	content.WriteString("Recent feed items:\n\n")

	// Include up to 20 items for analysis
	maxItems := 20
	if len(items) < maxItems {
		maxItems = len(items)
	}

	for i := 0; i < maxItems; i++ {
		item := items[i]
		content.WriteString(fmt.Sprintf("- %s: %s\n",
			item.Title, f.truncateText(item.Description, 100)))
	}

	prompt := fmt.Sprintf(`Analyze these recent feed items and identify the top 5 recurring topics or themes:

%s

For each topic, provide:
1. A clear topic name (2-3 words)
2. Why it's trending based on the items

Format: Return as "Topic Name: Brief explanation", one per line.`, content.String())

	response, err := f.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		MaxTokens:   400,
		Temperature: 0.5,
		Model:       "gemini-1.5-flash",
	})
	if err != nil {
		return []string{}, fmt.Errorf("failed to analyze topics: %w", err)
	}

	topics := strings.Split(strings.TrimSpace(response), "\n")
	var cleanTopics []string
	for _, topic := range topics {
		topic = strings.TrimSpace(topic)
		if topic != "" && strings.Contains(topic, ":") {
			cleanTopics = append(cleanTopics, topic)
		}
	}

	return cleanTopics, nil
}

// extractTrendingKeywords extracts trending keywords from feed items
func (f *FeedServiceImpl) extractTrendingKeywords(items []core.FeedItem) []string {
	wordCounts := make(map[string]int)

	// Common stop words to filter out
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"this": true, "that": true, "these": true, "those": true, "will": true,
		"can": true, "could": true, "should": true, "would": true, "have": true,
		"has": true, "had": true, "been": true, "be": true, "do": true, "does": true,
		"did": true, "get": true, "got": true, "new": true, "now": true, "how": true,
		"what": true, "when": true, "where": true, "why": true, "who": true,
	}

	for _, item := range items {
		// Combine title and description for keyword extraction
		text := strings.ToLower(item.Title + " " + item.Description)
		words := strings.Fields(text)

		for _, word := range words {
			// Clean word
			word = strings.Trim(word, ".,!?;:\"'()[]{}|")
			if len(word) > 3 && !stopWords[word] {
				wordCounts[word]++
			}
		}
	}

	// Sort by frequency
	type wordFreq struct {
		word  string
		count int
	}
	var sortedWords []wordFreq
	for word, count := range wordCounts {
		if count >= 2 { // Only include words that appear at least twice
			sortedWords = append(sortedWords, wordFreq{word, count})
		}
	}

	sort.Slice(sortedWords, func(i, j int) bool {
		return sortedWords[i].count > sortedWords[j].count
	})

	// Return top 10 keywords
	var keywords []string
	maxKeywords := 10
	if len(sortedWords) < maxKeywords {
		maxKeywords = len(sortedWords)
	}

	for i := 0; i < maxKeywords; i++ {
		keywords = append(keywords, sortedWords[i].word)
	}

	return keywords
}

// selectRecommendedItems selects the most interesting items for recommendation
func (f *FeedServiceImpl) selectRecommendedItems(items []core.FeedItem, maxItems int) []core.FeedItem {
	if len(items) <= maxItems {
		return items
	}

	// Simple scoring based on recency and title length (as proxy for content quality)
	type scoredItem struct {
		item  core.FeedItem
		score float64
	}

	var scoredItems []scoredItem
	now := time.Now()

	for _, item := range items {
		score := 0.0

		// Recency score (more recent = higher score)
		daysSince := now.Sub(item.Published).Hours() / 24
		if daysSince <= 1 {
			score += 0.5
		} else if daysSince <= 3 {
			score += 0.3
		} else if daysSince <= 7 {
			score += 0.1
		}

		// Title quality score (reasonable length)
		titleLen := len(item.Title)
		if titleLen >= 20 && titleLen <= 100 {
			score += 0.3
		}

		// Description quality score
		if len(item.Description) > 50 {
			score += 0.2
		}

		scoredItems = append(scoredItems, scoredItem{item, score})
	}

	// Sort by score
	sort.Slice(scoredItems, func(i, j int) bool {
		return scoredItems[i].score > scoredItems[j].score
	})

	// Return top items
	var recommended []core.FeedItem
	for i := 0; i < maxItems && i < len(scoredItems); i++ {
		recommended = append(recommended, scoredItems[i].item)
	}

	return recommended
}

// calculateQualityScore calculates an overall quality score for the feed content
func (f *FeedServiceImpl) calculateQualityScore(items []core.FeedItem) float64 {
	if len(items) == 0 {
		return 0.0
	}

	var totalScore float64
	for _, item := range items {
		score := 0.0

		// Title quality
		titleLen := len(item.Title)
		if titleLen >= 10 && titleLen <= 150 {
			score += 0.3
		}

		// Description quality
		if len(item.Description) > 20 {
			score += 0.4
		}

		// Has valid publish date
		if !item.Published.IsZero() {
			score += 0.2
		}

		// Recent content (published within last 30 days)
		if time.Since(item.Published).Hours() < 30*24 {
			score += 0.1
		}

		totalScore += score
	}

	return totalScore / float64(len(items))
}

// generateAnalysisSummary creates a summary of the feed analysis
func (f *FeedServiceImpl) generateAnalysisSummary(ctx context.Context, items []core.FeedItem, topics []string, keywords []string) (string, error) {
	topicsStr := strings.Join(topics, "; ")
	keywordsStr := strings.Join(keywords, ", ")

	prompt := fmt.Sprintf(`Based on this feed analysis data, create a concise summary for manual content curation:

Items Analyzed: %d recent feed items
Top Topics: %s
Trending Keywords: %s

Create a 2-3 paragraph summary that includes:
1. Overview of what's trending in the feeds
2. Key themes worth investigating further
3. Actionable recommendations for content curation

Keep it practical and focused on helping with manual content selection.`,
		len(items), topicsStr, keywordsStr)

	response, err := f.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		MaxTokens:   300,
		Temperature: 0.6,
		Model:       "gemini-1.5-flash",
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	return strings.TrimSpace(response), nil
}

// truncateText truncates text to specified length
func (f *FeedServiceImpl) truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}

	truncated := text[:maxLength]
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > maxLength-20 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}
