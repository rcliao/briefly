package core

import (
	"testing"
	"time"
)

func TestLinkCreation(t *testing.T) {
	now := time.Now()
	link := Link{
		ID:        "test-id",
		URL:       "https://example.com",
		DateAdded: now,
		Source:    "test",
	}

	if link.ID != "test-id" {
		t.Errorf("Expected ID to be 'test-id', got %s", link.ID)
	}
	if link.URL != "https://example.com" {
		t.Errorf("Expected URL to be 'https://example.com', got %s", link.URL)
	}
	if link.Source != "test" {
		t.Errorf("Expected Source to be 'test', got %s", link.Source)
	}
}

func TestArticleCreation(t *testing.T) {
	now := time.Now()
	article := Article{
		ID:              "article-1",
		LinkID:          "link-1",
		Title:           "Test Article",
		FetchedHTML:     "<html><body>Test content</body></html>",
		CleanedText:     "Test content",
		DateFetched:     now,
		MyTake:          "Interesting article",
		Embedding:       []float64{0.1, 0.2, 0.3},
		TopicCluster:    "Technology",
		TopicConfidence: 0.85,
		SentimentScore:  0.7,
		SentimentLabel:  "positive",
		SentimentEmoji:  "😊",
		AlertTriggered:  false,
		AlertConditions: []string{},
		ResearchQueries: []string{"deep learning", "AI"},
	}

	if article.ID != "article-1" {
		t.Errorf("Expected ID to be 'article-1', got %s", article.ID)
	}
	if article.Title != "Test Article" {
		t.Errorf("Expected Title to be 'Test Article', got %s", article.Title)
	}
	if article.TopicConfidence != 0.85 {
		t.Errorf("Expected TopicConfidence to be 0.85, got %f", article.TopicConfidence)
	}
	if article.SentimentScore != 0.7 {
		t.Errorf("Expected SentimentScore to be 0.7, got %f", article.SentimentScore)
	}
	if len(article.Embedding) != 3 {
		t.Errorf("Expected Embedding to have 3 elements, got %d", len(article.Embedding))
	}
	if len(article.ResearchQueries) != 2 {
		t.Errorf("Expected ResearchQueries to have 2 elements, got %d", len(article.ResearchQueries))
	}
}

func TestSummaryCreation(t *testing.T) {
	now := time.Now()
	summary := Summary{
		ID:              "summary-1",
		ArticleIDs:      []string{"article-1", "article-2"},
		SummaryText:     "This is a test summary",
		ModelUsed:       "gemini-1.5-flash",
		Length:          "short",
		Instructions:    "Summarize briefly",
		MyTake:          "Good summary",
		DateGenerated:   now,
		Embedding:       []float64{0.4, 0.5, 0.6},
		TopicCluster:    "Technology",
		TopicConfidence: 0.9,
	}

	if summary.ID != "summary-1" {
		t.Errorf("Expected ID to be 'summary-1', got %s", summary.ID)
	}
	if len(summary.ArticleIDs) != 2 {
		t.Errorf("Expected ArticleIDs to have 2 elements, got %d", len(summary.ArticleIDs))
	}
	if summary.ModelUsed != "gemini-1.5-flash" {
		t.Errorf("Expected ModelUsed to be 'gemini-1.5-flash', got %s", summary.ModelUsed)
	}
	if summary.TopicConfidence != 0.9 {
		t.Errorf("Expected TopicConfidence to be 0.9, got %f", summary.TopicConfidence)
	}
}

func TestDigestCreation(t *testing.T) {
	now := time.Now()
	digest := Digest{
		ID:                  "digest-1",
		Title:               "Test Digest",
		Content:             "This is test digest content",
		DigestSummary:       "Executive summary",
		MyTake:              "Great digest",
		ArticleURLs:         []string{"https://example.com/1", "https://example.com/2"},
		ModelUsed:           "gemini-1.5-pro",
		Format:              "standard",
		DateGenerated:       now,
		OverallSentiment:    "positive",
		AlertsSummary:       "No alerts triggered",
		TrendsSummary:       "Trending topics include AI and tech",
		ResearchSuggestions: []string{"machine learning trends", "AI ethics"},
	}

	if digest.ID != "digest-1" {
		t.Errorf("Expected ID to be 'digest-1', got %s", digest.ID)
	}
	if digest.Title != "Test Digest" {
		t.Errorf("Expected Title to be 'Test Digest', got %s", digest.Title)
	}
	if len(digest.ArticleURLs) != 2 {
		t.Errorf("Expected ArticleURLs to have 2 elements, got %d", len(digest.ArticleURLs))
	}
	if digest.Format != "standard" {
		t.Errorf("Expected Format to be 'standard', got %s", digest.Format)
	}
	if len(digest.ResearchSuggestions) != 2 {
		t.Errorf("Expected ResearchSuggestions to have 2 elements, got %d", len(digest.ResearchSuggestions))
	}
}

func TestFeedCreation(t *testing.T) {
	now := time.Now()
	feed := Feed{
		ID:           "feed-1",
		URL:          "https://example.com/rss",
		Title:        "Test Feed",
		Description:  "A test RSS feed",
		LastFetched:  now,
		LastModified: "Wed, 01 Jan 2025 00:00:00 GMT",
		ETag:         "abc123",
		Active:       true,
		ErrorCount:   0,
		LastError:    "",
		DateAdded:    now,
	}

	if feed.ID != "feed-1" {
		t.Errorf("Expected ID to be 'feed-1', got %s", feed.ID)
	}
	if feed.URL != "https://example.com/rss" {
		t.Errorf("Expected URL to be 'https://example.com/rss', got %s", feed.URL)
	}
	if !feed.Active {
		t.Errorf("Expected Active to be true, got %v", feed.Active)
	}
	if feed.ErrorCount != 0 {
		t.Errorf("Expected ErrorCount to be 0, got %d", feed.ErrorCount)
	}
}

func TestFeedItemCreation(t *testing.T) {
	now := time.Now()
	feedItem := FeedItem{
		ID:             "item-1",
		FeedID:         "feed-1",
		Title:          "Test Item",
		Link:           "https://example.com/article",
		Description:    "Test article description",
		Published:      now,
		GUID:           "guid-123",
		Processed:      false,
		DateDiscovered: now,
	}

	if feedItem.ID != "item-1" {
		t.Errorf("Expected ID to be 'item-1', got %s", feedItem.ID)
	}
	if feedItem.FeedID != "feed-1" {
		t.Errorf("Expected FeedID to be 'feed-1', got %s", feedItem.FeedID)
	}
	if feedItem.Processed {
		t.Errorf("Expected Processed to be false, got %v", feedItem.Processed)
	}
}

func TestTopicClusterCreation(t *testing.T) {
	now := time.Now()
	cluster := TopicCluster{
		ID:         "cluster-1",
		Label:      "Technology",
		Keywords:   []string{"AI", "machine learning", "tech"},
		ArticleIDs: []string{"article-1", "article-2", "article-3"},
		Centroid:   []float64{0.1, 0.2, 0.3, 0.4, 0.5},
		CreatedAt:  now,
	}

	if cluster.ID != "cluster-1" {
		t.Errorf("Expected ID to be 'cluster-1', got %s", cluster.ID)
	}
	if cluster.Label != "Technology" {
		t.Errorf("Expected Label to be 'Technology', got %s", cluster.Label)
	}
	if len(cluster.Keywords) != 3 {
		t.Errorf("Expected Keywords to have 3 elements, got %d", len(cluster.Keywords))
	}
	if len(cluster.ArticleIDs) != 3 {
		t.Errorf("Expected ArticleIDs to have 3 elements, got %d", len(cluster.ArticleIDs))
	}
	if len(cluster.Centroid) != 5 {
		t.Errorf("Expected Centroid to have 5 elements, got %d", len(cluster.Centroid))
	}
}

func TestCacheStatsCreation(t *testing.T) {
	now := time.Now()
	stats := CacheStats{
		ArticleCount:  10,
		SummaryCount:  5,
		DigestCount:   2,
		FeedCount:     3,
		FeedItemCount: 25,
		CacheSize:     1024000,
		LastUpdated:   now,
	}

	if stats.ArticleCount != 10 {
		t.Errorf("Expected ArticleCount to be 10, got %d", stats.ArticleCount)
	}
	if stats.SummaryCount != 5 {
		t.Errorf("Expected SummaryCount to be 5, got %d", stats.SummaryCount)
	}
	if stats.CacheSize != 1024000 {
		t.Errorf("Expected CacheSize to be 1024000, got %d", stats.CacheSize)
	}
}

func TestPromptCreation(t *testing.T) {
	creationTime := time.Now()
	lastUsedTime := creationTime.Add(1 * time.Hour)

	prompt := Prompt{
		ID:           "prompt-1",
		Text:         "Summarize this article",
		Category:     "summary",
		CreationDate: creationTime,
		LastUsedDate: lastUsedTime,
		UsageCount:   5,
	}

	if prompt.ID != "prompt-1" {
		t.Errorf("Expected ID to be 'prompt-1', got %s", prompt.ID)
	}
	if prompt.Text != "Summarize this article" {
		t.Errorf("Expected Text to be 'Summarize this article', got %s", prompt.Text)
	}
	if prompt.Category != "summary" {
		t.Errorf("Expected Category to be 'summary', got %s", prompt.Category)
	}
	if prompt.UsageCount != 5 {
		t.Errorf("Expected UsageCount to be 5, got %d", prompt.UsageCount)
	}
}
