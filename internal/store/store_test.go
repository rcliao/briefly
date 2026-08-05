package store

import (
	"briefly/internal/core"
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

	// Get stats
	stats, err := store.GetCacheStats()
	if err != nil {
		t.Fatalf("GetCacheStats failed: %v", err)
	}

	if stats.ArticleCount != 1 {
		t.Errorf("Expected 1 article, got %d", stats.ArticleCount)
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
}

func TestContentHash(t *testing.T) {
	if ContentHash("") != "empty" {
		t.Errorf("ContentHash(\"\") = %q, expected \"empty\"", ContentHash(""))
	}
	// Stable for identical content
	first := ContentHash("hello")
	if second := ContentHash("hello"); first != second {
		t.Errorf("ContentHash should be deterministic: %q != %q", first, second)
	}
	// Distinct for content the old length-based hash could not tell apart
	if ContentHash("hxxxo") == ContentHash("hello") {
		t.Error("ContentHash should differ for different content of same length/ends")
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

func TestSaveArticle(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	article := &core.Article{
		ID:          uuid.NewString(),
		URL:         "https://example.com/article",
		Title:       "Test Article",
		CleanedText: "Test content",
		DateFetched: time.Now().UTC(),
	}

	err = store.SaveArticle(article)
	if err != nil {
		t.Fatalf("SaveArticle failed: %v", err)
	}

	// Regression: articles must be cached under their URL (an earlier bug
	// keyed them on the usually-empty LinkID) and must round-trip URL and ID.
	retrieved, err := store.GetCachedArticle("https://example.com/article", time.Hour)
	if err != nil {
		t.Fatalf("GetCachedArticle failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected cache hit for saved article URL")
	}
	if retrieved.URL != article.URL {
		t.Errorf("cached article URL = %q, want %q", retrieved.URL, article.URL)
	}
	if retrieved.ID != article.ID {
		t.Errorf("cached article ID = %q, want %q", retrieved.ID, article.ID)
	}

	// An article with neither URL nor LinkID must be rejected, not silently
	// collapsed into a single empty-key row.
	if err := store.SaveArticle(&core.Article{ID: uuid.NewString(), Title: "no url"}); err == nil {
		t.Error("expected error when caching an article without URL")
	}
}
