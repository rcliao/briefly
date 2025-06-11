package store

import (
	"briefly/internal/core"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewStore(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	if store.db == nil {
		t.Error("Store database should not be nil")
	}

	// Check that database file was created
	dbPath := filepath.Join(tmpDir, "briefly.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file should be created")
	}
}

func TestNewStore_InvalidDirectory(t *testing.T) {
	// Try to create store in a file (not directory)
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "file.txt")
	_ = os.WriteFile(invalidPath, []byte("test"), 0644)

	_, err := NewStore(invalidPath)
	if err == nil {
		t.Error("Expected error when creating store in invalid directory")
	}
}

func TestCacheArticle_GetCachedArticle(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create test article
	article := core.Article{
		ID:              uuid.NewString(),
		LinkID:          "test-link-id",
		Title:           "Test Article",
		CleanedText:     "This is a test article content.",
		FetchedHTML:     "<html><body>Test content</body></html>",
		MyTake:          "My personal thoughts",
		DateFetched:     time.Now().UTC(),
		Embedding:       []float64{0.1, 0.2, 0.3},
		TopicCluster:    "Technology",
		TopicConfidence: 0.95,
		SentimentScore:  0.7,
		SentimentLabel:  "positive",
		SentimentEmoji:  "😊",
		AlertTriggered:  true,
		AlertConditions: []string{"condition1", "condition2"},
		ResearchQueries: []string{"query1", "query2"},
	}

	// Cache the article
	err = store.CacheArticle(article)
	if err != nil {
		t.Fatalf("CacheArticle failed: %v", err)
	}

	// Retrieve the cached article
	cachedArticle, err := store.GetCachedArticle("test-link-id", 24*time.Hour)
	if err != nil {
		t.Fatalf("GetCachedArticle failed: %v", err)
	}

	if cachedArticle == nil {
		t.Fatal("Expected cached article, got nil")
	}

	// Verify article data
	if cachedArticle.LinkID != article.LinkID {
		t.Errorf("Expected LinkID %s, got %s", article.LinkID, cachedArticle.LinkID)
	}
	if cachedArticle.Title != article.Title {
		t.Errorf("Expected title %s, got %s", article.Title, cachedArticle.Title)
	}
	if cachedArticle.CleanedText != article.CleanedText {
		t.Errorf("Expected content %s, got %s", article.CleanedText, cachedArticle.CleanedText)
	}
	if cachedArticle.MyTake != article.MyTake {
		t.Errorf("Expected MyTake %s, got %s", article.MyTake, cachedArticle.MyTake)
	}
	if cachedArticle.TopicCluster != article.TopicCluster {
		t.Errorf("Expected TopicCluster %s, got %s", article.TopicCluster, cachedArticle.TopicCluster)
	}
	if cachedArticle.TopicConfidence != article.TopicConfidence {
		t.Errorf("Expected TopicConfidence %f, got %f", article.TopicConfidence, cachedArticle.TopicConfidence)
	}
	if cachedArticle.SentimentScore != article.SentimentScore {
		t.Errorf("Expected SentimentScore %f, got %f", article.SentimentScore, cachedArticle.SentimentScore)
	}
	if cachedArticle.SentimentLabel != article.SentimentLabel {
		t.Errorf("Expected SentimentLabel %s, got %s", article.SentimentLabel, cachedArticle.SentimentLabel)
	}
	if cachedArticle.AlertTriggered != article.AlertTriggered {
		t.Errorf("Expected AlertTriggered %t, got %t", article.AlertTriggered, cachedArticle.AlertTriggered)
	}

	// Check embedding
	if len(cachedArticle.Embedding) != len(article.Embedding) {
		t.Errorf("Expected embedding length %d, got %d", len(article.Embedding), len(cachedArticle.Embedding))
	}
	for i, val := range article.Embedding {
		if len(cachedArticle.Embedding) > i && cachedArticle.Embedding[i] != val {
			t.Errorf("Expected embedding[%d] %f, got %f", i, val, cachedArticle.Embedding[i])
		}
	}

	// Check alert conditions
	if len(cachedArticle.AlertConditions) != len(article.AlertConditions) {
		t.Errorf("Expected %d alert conditions, got %d", len(article.AlertConditions), len(cachedArticle.AlertConditions))
	}

	// Check research queries
	if len(cachedArticle.ResearchQueries) != len(article.ResearchQueries) {
		t.Errorf("Expected %d research queries, got %d", len(article.ResearchQueries), len(cachedArticle.ResearchQueries))
	}
}

func TestGetCachedArticle_CacheMiss(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Try to get non-existent article
	cachedArticle, err := store.GetCachedArticle("non-existent", 24*time.Hour)
	if err != nil {
		t.Fatalf("GetCachedArticle failed: %v", err)
	}

	if cachedArticle != nil {
		t.Error("Expected nil for cache miss")
	}
}

func TestGetCachedArticle_Expired(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create article with old date
	article := core.Article{
		ID:          uuid.NewString(),
		LinkID:      "test-link-id",
		Title:       "Test Article",
		CleanedText: "Test content",
		DateFetched: time.Now().UTC().Add(-48 * time.Hour), // 2 days old
	}

	err = store.CacheArticle(article)
	if err != nil {
		t.Fatalf("CacheArticle failed: %v", err)
	}

	// Try to get with 24 hour max age
	cachedArticle, err := store.GetCachedArticle("test-link-id", 24*time.Hour)
	if err != nil {
		t.Fatalf("GetCachedArticle failed: %v", err)
	}

	if cachedArticle != nil {
		t.Error("Expected nil for expired cache entry")
	}
}

func TestCacheSummary_GetCachedSummary(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create test summary
	summary := core.Summary{
		ID:              uuid.NewString(),
		ArticleIDs:      []string{"article1", "article2"},
		SummaryText:     "This is a test summary.",
		ModelUsed:       "test-model",
		Instructions:    "Test instructions",
		DateGenerated:   time.Now().UTC(),
		Embedding:       []float64{0.4, 0.5, 0.6},
		TopicCluster:    "Technology",
		TopicConfidence: 0.88,
	}

	articleURL := "test-article-url"
	contentHash := "test-hash"

	// Cache the summary
	err = store.CacheSummary(summary, articleURL, contentHash)
	if err != nil {
		t.Fatalf("CacheSummary failed: %v", err)
	}

	// Retrieve the cached summary
	cachedSummary, err := store.GetCachedSummary(articleURL, contentHash, 24*time.Hour)
	if err != nil {
		t.Fatalf("GetCachedSummary failed: %v", err)
	}

	if cachedSummary == nil {
		t.Fatal("Expected cached summary, got nil")
	}

	// Verify summary data
	if cachedSummary.ID != summary.ID {
		t.Errorf("Expected ID %s, got %s", summary.ID, cachedSummary.ID)
	}
	if cachedSummary.SummaryText != summary.SummaryText {
		t.Errorf("Expected SummaryText %s, got %s", summary.SummaryText, cachedSummary.SummaryText)
	}
	if cachedSummary.ModelUsed != summary.ModelUsed {
		t.Errorf("Expected ModelUsed %s, got %s", summary.ModelUsed, cachedSummary.ModelUsed)
	}
	if cachedSummary.Instructions != summary.Instructions {
		t.Errorf("Expected Instructions %s, got %s", summary.Instructions, cachedSummary.Instructions)
	}
	if len(cachedSummary.ArticleIDs) != len(summary.ArticleIDs) {
		t.Errorf("Expected %d ArticleIDs, got %d", len(summary.ArticleIDs), len(cachedSummary.ArticleIDs))
	}
}

func TestGetCachedSummary_CacheMiss(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Try to get non-existent summary
	cachedSummary, err := store.GetCachedSummary("non-existent", "hash", 24*time.Hour)
	if err != nil {
		t.Fatalf("GetCachedSummary failed: %v", err)
	}

	if cachedSummary != nil {
		t.Error("Expected nil for cache miss")
	}
}

func TestCacheDigest_GetCachedDigest(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	digestID := uuid.NewString()
	title := "Test Digest"
	content := "This is a test digest content."
	digestSummary := "Test digest summary"
	articleURLs := []string{"url1", "url2"}
	modelUsed := "test-model"

	// Cache the digest
	err = store.CacheDigest(digestID, title, content, digestSummary, articleURLs, modelUsed)
	if err != nil {
		t.Fatalf("CacheDigest failed: %v", err)
	}

	// Retrieve the cached digest
	cachedDigest, err := store.GetCachedDigest(digestID)
	if err != nil {
		t.Fatalf("GetCachedDigest failed: %v", err)
	}

	if cachedDigest == nil {
		t.Fatal("Expected cached digest, got nil")
	}

	// Verify digest data
	if cachedDigest.ID != digestID {
		t.Errorf("Expected ID %s, got %s", digestID, cachedDigest.ID)
	}
	if cachedDigest.Title != title {
		t.Errorf("Expected title %s, got %s", title, cachedDigest.Title)
	}
	if cachedDigest.Content != content {
		t.Errorf("Expected content %s, got %s", content, cachedDigest.Content)
	}
	if cachedDigest.DigestSummary != digestSummary {
		t.Errorf("Expected DigestSummary %s, got %s", digestSummary, cachedDigest.DigestSummary)
	}
	if cachedDigest.ModelUsed != modelUsed {
		t.Errorf("Expected ModelUsed %s, got %s", modelUsed, cachedDigest.ModelUsed)
	}
	if len(cachedDigest.ArticleURLs) != len(articleURLs) {
		t.Errorf("Expected %d ArticleURLs, got %d", len(articleURLs), len(cachedDigest.ArticleURLs))
	}
}

func TestCacheDigestWithFormat(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	digestID := uuid.NewString()
	format := "newsletter"

	err = store.CacheDigestWithFormat(digestID, "Title", "Content", "Summary", format, []string{"url1"}, "model")
	if err != nil {
		t.Fatalf("CacheDigestWithFormat failed: %v", err)
	}

	cachedDigest, err := store.GetCachedDigest(digestID)
	if err != nil {
		t.Fatalf("GetCachedDigest failed: %v", err)
	}

	if cachedDigest.Format != format {
		t.Errorf("Expected format %s, got %s", format, cachedDigest.Format)
	}
}

func TestUpdateDigestMyTake(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	digestID := uuid.NewString()
	newMyTake := "My updated take on this digest"

	// Cache a digest first
	err = store.CacheDigest(digestID, "Title", "Content", "Summary", []string{"url1"}, "model")
	if err != nil {
		t.Fatalf("CacheDigest failed: %v", err)
	}

	// Update the my_take
	err = store.UpdateDigestMyTake(digestID, newMyTake)
	if err != nil {
		t.Fatalf("UpdateDigestMyTake failed: %v", err)
	}

	// Retrieve and verify
	digest, err := store.GetCachedDigest(digestID)
	if err != nil {
		t.Fatalf("GetCachedDigest failed: %v", err)
	}

	if digest.MyTake != newMyTake {
		t.Errorf("Expected MyTake %s, got %s", newMyTake, digest.MyTake)
	}
}

func TestGetLatestDigests(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Cache multiple digests
	for i := 0; i < 5; i++ {
		digestID := uuid.NewString()
		title := fmt.Sprintf("Test Digest %d", i)
		err = store.CacheDigest(digestID, title, "Content", "Summary", []string{"url"}, "model")
		if err != nil {
			t.Fatalf("CacheDigest failed: %v", err)
		}
		// Small delay to ensure different timestamps
		time.Sleep(time.Millisecond)
	}

	// Get latest 3 digests
	digests, err := store.GetLatestDigests(3)
	if err != nil {
		t.Fatalf("GetLatestDigests failed: %v", err)
	}

	if len(digests) != 3 {
		t.Errorf("Expected 3 digests, got %d", len(digests))
	}

	// Should be ordered by date descending (most recent first)
	for i := 0; i < len(digests)-1; i++ {
		if digests[i].DateGenerated.Before(digests[i+1].DateGenerated) {
			t.Error("Digests should be ordered by date descending")
		}
	}
}

func TestFindDigestByPartialID(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	digestID := "1234567890abcdef"
	err = store.CacheDigest(digestID, "Test", "Content", "Summary", []string{"url"}, "model")
	if err != nil {
		t.Fatalf("CacheDigest failed: %v", err)
	}

	// Test exact match
	digest, err := store.FindDigestByPartialID(digestID)
	if err != nil {
		t.Fatalf("FindDigestByPartialID failed: %v", err)
	}
	if digest == nil || digest.ID != digestID {
		t.Error("Should find digest with exact ID match")
	}

	// Test partial match
	digest, err = store.FindDigestByPartialID("1234")
	if err != nil {
		t.Fatalf("FindDigestByPartialID failed: %v", err)
	}
	if digest == nil || digest.ID != digestID {
		t.Error("Should find digest with partial ID match")
	}

	// Test non-existent
	digest, err = store.FindDigestByPartialID("xyz")
	if err != nil {
		t.Fatalf("FindDigestByPartialID failed: %v", err)
	}
	if digest != nil {
		t.Error("Should not find digest with non-matching partial ID")
	}
}

func TestGetCacheStats(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add some test data
	article := core.Article{
		ID:          uuid.NewString(),
		LinkID:      "test-url",
		Title:       "Test",
		CleanedText: "Content",
		DateFetched: time.Now().UTC(),
	}
	err = store.CacheArticle(article)
	if err != nil {
		t.Fatalf("CacheArticle failed: %v", err)
	}

	err = store.CacheDigest(uuid.NewString(), "Title", "Content", "Summary", []string{"url"}, "model")
	if err != nil {
		t.Fatalf("CacheDigest failed: %v", err)
	}

	// Get stats
	stats, err := store.GetCacheStats()
	if err != nil {
		t.Fatalf("GetCacheStats failed: %v", err)
	}

	if stats.ArticleCount != 1 {
		t.Errorf("Expected 1 article, got %d", stats.ArticleCount)
	}
	if stats.DigestCount != 1 {
		t.Errorf("Expected 1 digest, got %d", stats.DigestCount)
	}
	if stats.CacheSize <= 0 {
		t.Error("Cache size should be greater than 0")
	}
}

func TestClearCache(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add some test data
	article := core.Article{
		ID:          uuid.NewString(),
		LinkID:      "test-url",
		CleanedText: "Content",
		DateFetched: time.Now().UTC(),
	}
	err = store.CacheArticle(article)
	if err != nil {
		t.Fatalf("CacheArticle failed: %v", err)
	}

	// Clear cache
	err = store.ClearCache()
	if err != nil {
		t.Fatalf("ClearCache failed: %v", err)
	}

	// Verify cache is empty
	stats, err := store.GetCacheStats()
	if err != nil {
		t.Fatalf("GetCacheStats failed: %v", err)
	}

	if stats.ArticleCount != 0 {
		t.Errorf("Expected 0 articles after clear, got %d", stats.ArticleCount)
	}
	if stats.DigestCount != 0 {
		t.Errorf("Expected 0 digests after clear, got %d", stats.DigestCount)
	}
}

func TestCleanupOldCache(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add old article
	oldArticle := core.Article{
		ID:          uuid.NewString(),
		LinkID:      "old-url",
		CleanedText: "Old content",
		DateFetched: time.Now().UTC().Add(-48 * time.Hour),
	}
	err = store.CacheArticle(oldArticle)
	if err != nil {
		t.Fatalf("CacheArticle failed: %v", err)
	}

	// Add recent article
	recentArticle := core.Article{
		ID:          uuid.NewString(),
		LinkID:      "recent-url",
		CleanedText: "Recent content",
		DateFetched: time.Now().UTC(),
	}
	err = store.CacheArticle(recentArticle)
	if err != nil {
		t.Fatalf("CacheArticle failed: %v", err)
	}

	// Cleanup old cache (older than 24 hours)
	err = store.CleanupOldCache(24*time.Hour, 24*time.Hour)
	if err != nil {
		t.Fatalf("CleanupOldCache failed: %v", err)
	}

	// Verify old article is gone
	cachedOld, err := store.GetCachedArticle("old-url", 72*time.Hour)
	if err != nil {
		t.Fatalf("GetCachedArticle failed: %v", err)
	}
	if cachedOld != nil {
		t.Error("Old article should be cleaned up")
	}

	// Verify recent article remains
	cachedRecent, err := store.GetCachedArticle("recent-url", 24*time.Hour)
	if err != nil {
		t.Fatalf("GetCachedArticle failed: %v", err)
	}
	if cachedRecent == nil {
		t.Error("Recent article should remain after cleanup")
	}
}

func TestGenerateContentHash(t *testing.T) {
	testCases := []struct {
		content  string
		expected string
	}{
		{"", "empty"},
		{"a", "1-a-a"},
		{"hello", "5-h-o"},
		{"hello world", "11-h-d"},
	}

	for _, tc := range testCases {
		result := generateContentHash(tc.content)
		if result != tc.expected {
			t.Errorf("generateContentHash(%q) = %q, expected %q", tc.content, result, tc.expected)
		}
	}
}

func TestSerializeDeserializeEmbedding(t *testing.T) {
	original := []float64{0.1, 0.2, 0.3, -0.5, 1.0}

	// Test serialization
	serialized, err := serializeEmbedding(original)
	if err != nil {
		t.Fatalf("serializeEmbedding failed: %v", err)
	}

	if len(serialized) == 0 {
		t.Error("Serialized embedding should not be empty")
	}

	// Test deserialization
	deserialized, err := deserializeEmbedding(serialized)
	if err != nil {
		t.Fatalf("deserializeEmbedding failed: %v", err)
	}

	if len(deserialized) != len(original) {
		t.Errorf("Expected length %d, got %d", len(original), len(deserialized))
	}

	for i, val := range original {
		if deserialized[i] != val {
			t.Errorf("Expected embedding[%d] = %f, got %f", i, val, deserialized[i])
		}
	}
}

func TestSerializeDeserializeEmbedding_Nil(t *testing.T) {
	// Test nil embedding
	serialized, err := serializeEmbedding(nil)
	if err != nil {
		t.Fatalf("serializeEmbedding failed: %v", err)
	}
	if serialized != nil {
		t.Error("Serialized nil embedding should be nil")
	}

	deserialized, err := deserializeEmbedding(nil)
	if err != nil {
		t.Fatalf("deserializeEmbedding failed: %v", err)
	}
	if deserialized != nil {
		t.Error("Deserialized nil embedding should be nil")
	}
}

func TestGetRecentArticles(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add articles from different dates
	now := time.Now().UTC()

	// Recent article (1 day ago)
	recentArticle := core.Article{
		ID:          uuid.NewString(),
		LinkID:      "recent-url",
		Title:       "Recent Article",
		CleanedText: "Recent content",
		DateFetched: now.AddDate(0, 0, -1),
	}
	err = store.CacheArticle(recentArticle)
	if err != nil {
		t.Fatalf("CacheArticle failed: %v", err)
	}

	// Old article (10 days ago)
	oldArticle := core.Article{
		ID:          uuid.NewString(),
		LinkID:      "old-url",
		Title:       "Old Article",
		CleanedText: "Old content",
		DateFetched: now.AddDate(0, 0, -10),
	}
	err = store.CacheArticle(oldArticle)
	if err != nil {
		t.Fatalf("CacheArticle failed: %v", err)
	}

	// Get articles from last 7 days
	articles, err := store.GetRecentArticles(7)
	if err != nil {
		t.Fatalf("GetRecentArticles failed: %v", err)
	}

	// Should only get the recent article
	if len(articles) != 1 {
		t.Errorf("Expected 1 recent article, got %d", len(articles))
	}
	if len(articles) > 0 && articles[0].Title != "Recent Article" {
		t.Errorf("Expected recent article, got %s", articles[0].Title)
	}
}

func TestGetArticleByURL(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	article := core.Article{
		ID:          uuid.NewString(),
		LinkID:      "test-url",
		Title:       "Test Article",
		CleanedText: "Test content",
		DateFetched: time.Now().UTC(),
	}

	err = store.CacheArticle(article)
	if err != nil {
		t.Fatalf("CacheArticle failed: %v", err)
	}

	// Get article by URL
	retrieved, err := store.GetArticleByURL("test-url")
	if err != nil {
		t.Fatalf("GetArticleByURL failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected article, got nil")
	}
	if retrieved.Title != article.Title {
		t.Errorf("Expected title %s, got %s", article.Title, retrieved.Title)
	}

	// Test non-existent URL
	notFound, err := store.GetArticleByURL("non-existent")
	if err != nil {
		t.Fatalf("GetArticleByURL failed: %v", err)
	}
	if notFound != nil {
		t.Error("Expected nil for non-existent URL")
	}
}

func TestSaveArticle(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	article := &core.Article{
		ID:          uuid.NewString(),
		LinkID:      "test-url",
		Title:       "Test Article",
		CleanedText: "Test content",
		DateFetched: time.Now().UTC(),
	}

	err = store.SaveArticle(article)
	if err != nil {
		t.Fatalf("SaveArticle failed: %v", err)
	}

	// Verify it was saved
	retrieved, err := store.GetArticleByURL("test-url")
	if err != nil {
		t.Fatalf("GetArticleByURL failed: %v", err)
	}
	if retrieved == nil {
		t.Error("Article should be saved")
	}
}

func TestGetArticlesByDateRange(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	now := time.Now().UTC()
	startDate := now.AddDate(0, 0, -5) // 5 days ago
	endDate := now.AddDate(0, 0, -1)   // 1 day ago

	// Article within range (3 days ago)
	inRangeArticle := core.Article{
		ID:          uuid.NewString(),
		LinkID:      "in-range-url",
		Title:       "In Range Article",
		CleanedText: "In range content",
		DateFetched: now.AddDate(0, 0, -3),
	}
	err = store.CacheArticle(inRangeArticle)
	if err != nil {
		t.Fatalf("CacheArticle failed: %v", err)
	}

	// Article outside range (10 days ago)
	outOfRangeArticle := core.Article{
		ID:          uuid.NewString(),
		LinkID:      "out-of-range-url",
		Title:       "Out of Range Article",
		CleanedText: "Out of range content",
		DateFetched: now.AddDate(0, 0, -10),
	}
	err = store.CacheArticle(outOfRangeArticle)
	if err != nil {
		t.Fatalf("CacheArticle failed: %v", err)
	}

	// Get articles by date range
	articles, err := store.GetArticlesByDateRange(startDate, endDate)
	if err != nil {
		t.Fatalf("GetArticlesByDateRange failed: %v", err)
	}

	// Should only get the in-range article
	if len(articles) != 1 {
		t.Errorf("Expected 1 article in range, got %d", len(articles))
	}
	if len(articles) > 0 && articles[0].Title != "In Range Article" {
		t.Errorf("Expected in-range article, got %s", articles[0].Title)
	}
}
