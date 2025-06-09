package integration

import (
	"briefly/internal/core"
	"briefly/test/mocks"
	"context"
	"testing"
	"time"
)

// TestDigestWorkflow tests the complete digest generation workflow using mocks
func TestDigestWorkflow(t *testing.T) {
	ctx := context.Background()

	// Setup mock services
	mockLLM := &mocks.MockLLMService{}
	mockCache := &mocks.MockCacheService{}
	mockProcessor := &mocks.MockArticleProcessor{}
	mockDigest := &mocks.MockDigestService{}

	// Test data
	testURLs := []string{
		"https://example.com/article1",
		"https://example.com/article2",
	}

	// Test single article processing
	t.Run("ProcessSingleArticle", func(t *testing.T) {
		article, summary, err := mockDigest.ProcessSingleArticle(ctx, testURLs[0], "standard")
		if err != nil {
			t.Fatalf("ProcessSingleArticle failed: %v", err)
		}

		if article == nil {
			t.Fatal("Expected article, got nil")
		}

		if summary == nil {
			t.Fatal("Expected summary, got nil")
		}

		if article.LinkID != testURLs[0] {
			t.Errorf("Expected LinkID %s, got %s", testURLs[0], article.LinkID)
		}
	})

	// Test article processing
	t.Run("ProcessArticles", func(t *testing.T) {
		articles, err := mockProcessor.ProcessArticles(ctx, testURLs)
		if err != nil {
			t.Fatalf("ProcessArticles failed: %v", err)
		}

		if len(articles) != len(testURLs) {
			t.Errorf("Expected %d articles, got %d", len(testURLs), len(articles))
		}

		for i, article := range articles {
			if article.LinkID != testURLs[i] {
				t.Errorf("Expected LinkID %s, got %s", testURLs[i], article.LinkID)
			}
		}
	})

	// Test LLM summarization
	t.Run("SummarizeArticle", func(t *testing.T) {
		testArticle := core.Article{
			ID:          "test-1",
			Title:       "Test Article",
			LinkID:      testURLs[0],
			CleanedText: "This is test content",
			DateFetched: time.Now(),
		}

		summary, err := mockLLM.SummarizeArticle(ctx, testArticle, "standard")
		if err != nil {
			t.Fatalf("SummarizeArticle failed: %v", err)
		}

		if summary == nil {
			t.Fatal("Expected summary, got nil")
		}

		if summary.SummaryText == "" {
			t.Error("Expected non-empty summary text")
		}
	})

	// Test caching operations
	t.Run("CacheOperations", func(t *testing.T) {
		// Test cache stats
		stats, err := mockCache.GetCacheStats(ctx)
		if err != nil {
			t.Fatalf("GetCacheStats failed: %v", err)
		}

		if stats == nil {
			t.Fatal("Expected cache stats, got nil")
		}

		if stats.ArticleCount <= 0 {
			t.Error("Expected positive article count")
		}

		// Test cache operations
		testArticle := core.Article{
			ID:          "cache-test-1",
			Title:       "Cache Test Article",
			LinkID:      "https://cache-test.com",
			CleanedText: "Cache test content",
			DateFetched: time.Now(),
		}

		err = mockCache.CacheArticle(ctx, testArticle)
		if err != nil {
			t.Errorf("CacheArticle failed: %v", err)
		}
	})

	// Test digest generation
	t.Run("GenerateDigest", func(t *testing.T) {
		digest, err := mockDigest.GenerateDigest(ctx, testURLs, "standard")
		if err != nil {
			t.Fatalf("GenerateDigest failed: %v", err)
		}

		if digest == nil {
			t.Fatal("Expected digest, got nil")
		}

		if digest.Title == "" {
			t.Error("Expected non-empty digest title")
		}

		if digest.Content == "" {
			t.Error("Expected non-empty digest content")
		}

		if digest.Format != "standard" {
			t.Errorf("Expected format 'standard', got %s", digest.Format)
		}
	})

	// Test LLM embedding generation
	t.Run("GenerateEmbedding", func(t *testing.T) {
		embedding, err := mockLLM.GenerateEmbedding(ctx, "test text for embedding")
		if err != nil {
			t.Fatalf("GenerateEmbedding failed: %v", err)
		}

		if len(embedding) == 0 {
			t.Error("Expected non-empty embedding vector")
		}
	})

	// Test research query generation
	t.Run("GenerateResearchQueries", func(t *testing.T) {
		testArticle := core.Article{
			ID:          "research-test-1",
			Title:       "AI Development Trends",
			LinkID:      "https://research-test.com",
			CleanedText: "Article about AI development trends and future predictions",
			DateFetched: time.Now(),
		}

		queries, err := mockLLM.GenerateResearchQueries(ctx, testArticle, 2)
		if err != nil {
			t.Fatalf("GenerateResearchQueries failed: %v", err)
		}

		if len(queries) == 0 {
			t.Error("Expected non-empty research queries")
		}

		for _, query := range queries {
			if query == "" {
				t.Error("Expected non-empty research query")
			}
		}
	})
}

// TestServiceIntegration tests that services work together correctly
func TestServiceIntegration(t *testing.T) {
	ctx := context.Background()

	// This test demonstrates how the services would work together
	// in a real implementation

	mockLLM := &mocks.MockLLMService{}
	mockCache := &mocks.MockCacheService{}
	mockProcessor := &mocks.MockArticleProcessor{}

	testURL := "https://integration-test.com/article"

	// Step 1: Process article
	article, err := mockProcessor.ProcessArticle(ctx, testURL)
	if err != nil {
		t.Fatalf("Failed to process article: %v", err)
	}

	// Step 2: Cache the article
	err = mockCache.CacheArticle(ctx, *article)
	if err != nil {
		t.Fatalf("Failed to cache article: %v", err)
	}

	// Step 3: Generate summary
	summary, err := mockLLM.SummarizeArticle(ctx, *article, "standard")
	if err != nil {
		t.Fatalf("Failed to summarize article: %v", err)
	}

	// Step 4: Cache the summary
	err = mockCache.CacheSummary(ctx, *summary, testURL, "content-hash-123")
	if err != nil {
		t.Fatalf("Failed to cache summary: %v", err)
	}

	// Step 5: Generate research queries
	queries, err := mockLLM.GenerateResearchQueries(ctx, *article, 3)
	if err != nil {
		t.Fatalf("Failed to generate research queries: %v", err)
	}

	// Verify the integration
	if len(queries) == 0 {
		t.Error("Expected research queries from integration")
	}

	t.Logf("Integration test completed successfully. Generated %d research queries.", len(queries))
}
