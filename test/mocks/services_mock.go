package mocks

import (
	"briefly/internal/core"
	"briefly/internal/services"
	"context"
	"fmt"
	"time"
)

// MockDigestService provides a mock implementation of DigestService
type MockDigestService struct {
	GenerateDigestFunc         func(ctx context.Context, urls []string, format string) (*core.Digest, error)
	GenerateDigestFromFileFunc func(ctx context.Context, inputFile string, format string) (*core.Digest, error)
	ProcessSingleArticleFunc   func(ctx context.Context, url string, format string) (*core.Article, *core.Summary, error)
}

func (m *MockDigestService) GenerateDigest(ctx context.Context, urls []string, format string) (*core.Digest, error) {
	if m.GenerateDigestFunc != nil {
		return m.GenerateDigestFunc(ctx, urls, format)
	}
	return &core.Digest{
		ID:            "mock-digest-1",
		Title:         "Mock Digest",
		Content:       "Mock digest content",
		DateGenerated: time.Now(),
		Format:        format,
	}, nil
}

func (m *MockDigestService) GenerateDigestFromFile(ctx context.Context, inputFile string, format string) (*core.Digest, error) {
	if m.GenerateDigestFromFileFunc != nil {
		return m.GenerateDigestFromFileFunc(ctx, inputFile, format)
	}
	return m.GenerateDigest(ctx, []string{}, format)
}

func (m *MockDigestService) ProcessSingleArticle(ctx context.Context, url string, format string) (*core.Article, *core.Summary, error) {
	if m.ProcessSingleArticleFunc != nil {
		return m.ProcessSingleArticleFunc(ctx, url, format)
	}
	article := &core.Article{
		ID:          "mock-article-1",
		Title:       "Mock Article",
		LinkID:      url,
		CleanedText: "Mock article content",
		DateFetched: time.Now(),
	}
	summary := &core.Summary{
		ID:            "mock-summary-1",
		SummaryText:   "Mock summary",
		DateGenerated: time.Now(),
	}
	return article, summary, nil
}

// MockLLMService provides a mock implementation of LLMService
type MockLLMService struct {
	SummarizeArticleFunc        func(ctx context.Context, article core.Article, format string) (*core.Summary, error)
	GenerateDigestTitleFunc     func(ctx context.Context, content string, format string) (string, error)
	GenerateEmbeddingFunc       func(ctx context.Context, text string) ([]float64, error)
	GenerateResearchQueriesFunc func(ctx context.Context, article core.Article, depth int) ([]string, error)
	AnalyzeSentimentFunc        func(ctx context.Context, text string) (float64, string, string, error)
}

func (m *MockLLMService) SummarizeArticle(ctx context.Context, article core.Article, format string) (*core.Summary, error) {
	if m.SummarizeArticleFunc != nil {
		return m.SummarizeArticleFunc(ctx, article, format)
	}
	return &core.Summary{
		ID:            "mock-summary-1",
		SummaryText:   "Mock summary of " + article.Title,
		DateGenerated: time.Now(),
	}, nil
}

func (m *MockLLMService) GenerateDigestTitle(ctx context.Context, content string, format string) (string, error) {
	if m.GenerateDigestTitleFunc != nil {
		return m.GenerateDigestTitleFunc(ctx, content, format)
	}
	return "Mock Digest Title - " + format, nil
}

func (m *MockLLMService) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	if m.GenerateEmbeddingFunc != nil {
		return m.GenerateEmbeddingFunc(ctx, text)
	}
	// Return a mock embedding vector
	return []float64{0.1, 0.2, 0.3, 0.4, 0.5}, nil
}

func (m *MockLLMService) GenerateResearchQueries(ctx context.Context, article core.Article, depth int) ([]string, error) {
	if m.GenerateResearchQueriesFunc != nil {
		return m.GenerateResearchQueriesFunc(ctx, article, depth)
	}
	return []string{
		"Mock research query 1 for " + article.Title,
		"Mock research query 2 for " + article.Title,
	}, nil
}

func (m *MockLLMService) AnalyzeSentiment(ctx context.Context, text string) (float64, string, string, error) {
	if m.AnalyzeSentimentFunc != nil {
		return m.AnalyzeSentimentFunc(ctx, text)
	}
	return 0.7, "positive", "😊", nil
}

// MockCacheService provides a mock implementation of CacheService
type MockCacheService struct {
	GetCachedArticleFunc func(ctx context.Context, url string) (*core.Article, error)
	CacheArticleFunc     func(ctx context.Context, article core.Article) error
	GetCachedSummaryFunc func(ctx context.Context, url string, contentHash string) (*core.Summary, error)
	CacheSummaryFunc     func(ctx context.Context, summary core.Summary, url string, contentHash string) error
	ClearCacheFunc       func(ctx context.Context) error
	GetCacheStatsFunc    func(ctx context.Context) (*services.CacheStats, error)
}

func (m *MockCacheService) GetCachedArticle(ctx context.Context, url string) (*core.Article, error) {
	if m.GetCachedArticleFunc != nil {
		return m.GetCachedArticleFunc(ctx, url)
	}
	return nil, nil // No cached article by default
}

func (m *MockCacheService) CacheArticle(ctx context.Context, article core.Article) error {
	if m.CacheArticleFunc != nil {
		return m.CacheArticleFunc(ctx, article)
	}
	return nil
}

func (m *MockCacheService) GetCachedSummary(ctx context.Context, url string, contentHash string) (*core.Summary, error) {
	if m.GetCachedSummaryFunc != nil {
		return m.GetCachedSummaryFunc(ctx, url, contentHash)
	}
	return nil, nil // No cached summary by default
}

func (m *MockCacheService) CacheSummary(ctx context.Context, summary core.Summary, url string, contentHash string) error {
	if m.CacheSummaryFunc != nil {
		return m.CacheSummaryFunc(ctx, summary, url, contentHash)
	}
	return nil
}

func (m *MockCacheService) ClearCache(ctx context.Context) error {
	if m.ClearCacheFunc != nil {
		return m.ClearCacheFunc(ctx)
	}
	return nil
}

func (m *MockCacheService) GetCacheStats(ctx context.Context) (*services.CacheStats, error) {
	if m.GetCacheStatsFunc != nil {
		return m.GetCacheStatsFunc(ctx)
	}
	return &services.CacheStats{
		ArticleCount:       10,
		SummaryCount:       8,
		DigestCount:        3,
		CacheSize:          1024 * 1024, // 1MB
		LastUpdated:        time.Now().Format("2006-01-02 15:04:05"),
		FeedCount:          2,
		ActiveFeedCount:    1,
		FeedItemCount:      50,
		ProcessedItemCount: 25,
		TopicClusters:      map[string]int{"AI": 3, "Tech": 2, "Development": 5},
	}, nil
}

// MockArticleProcessor provides a mock implementation of ArticleProcessor
type MockArticleProcessor struct {
	ProcessArticleFunc         func(ctx context.Context, url string) (*core.Article, error)
	ProcessArticlesFunc        func(ctx context.Context, urls []string) ([]core.Article, error)
	CleanAndExtractContentFunc func(ctx context.Context, article *core.Article) error
}

func (m *MockArticleProcessor) ProcessArticle(ctx context.Context, url string) (*core.Article, error) {
	if m.ProcessArticleFunc != nil {
		return m.ProcessArticleFunc(ctx, url)
	}
	return &core.Article{
		ID:          "mock-article-1",
		Title:       "Mock Article from " + url,
		LinkID:      url,
		CleanedText: "Mock cleaned content for " + url,
		DateFetched: time.Now(),
	}, nil
}

func (m *MockArticleProcessor) ProcessArticles(ctx context.Context, urls []string) ([]core.Article, error) {
	if m.ProcessArticlesFunc != nil {
		return m.ProcessArticlesFunc(ctx, urls)
	}
	var articles []core.Article
	for i, url := range urls {
		article, _ := m.ProcessArticle(ctx, url)
		article.ID = fmt.Sprintf("mock-article-%d", i+1)
		articles = append(articles, *article)
	}
	return articles, nil
}

func (m *MockArticleProcessor) CleanAndExtractContent(ctx context.Context, article *core.Article) error {
	if m.CleanAndExtractContentFunc != nil {
		return m.CleanAndExtractContentFunc(ctx, article)
	}
	article.CleanedText = "Cleaned: " + article.CleanedText
	return nil
}
