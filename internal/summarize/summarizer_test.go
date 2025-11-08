package summarize

import (
	"briefly/internal/core"
	"context"
	"strings"
	"testing"
	"time"
)

// MockLLMClient implements LLMClient for testing
type MockLLMClient struct {
	responses  map[string]string
	callCount  int
	shouldFail bool
}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		responses:  make(map[string]string),
		callCount:  0,
		shouldFail: false,
	}
}

func (m *MockLLMClient) GenerateText(ctx context.Context, prompt string, options interface{}) (string, error) {
	m.callCount++

	if m.shouldFail {
		return "", &mockError{message: "mock LLM error"}
	}

	// Return mock response based on prompt content
	// Check Theme before Title since Theme prompt also contains "Title:"
	if strings.Contains(prompt, "Theme:") {
		return "Technology", nil
	}

	if strings.Contains(prompt, "SUMMARY:") {
		return `SUMMARY:
This is a test summary of approximately 150 words that describes the main points of the article in a concise and clear manner. The summary captures the key insights and findings.

KEY POINTS:
- First important key point about the article
- Second key point highlighting main insight
- Third point discussing implications
- Fourth point about methodology or approach
- Fifth point summarizing conclusions`, nil
	}

	if strings.Contains(prompt, "key points") {
		return `- First key point from the content
- Second key point highlighting important information
- Third key point with actionable insights
- Fourth key point about conclusions
- Fifth key point summarizing takeaways`, nil
	}

	if strings.Contains(prompt, "Title:") {
		return "Generated Test Article Title", nil
	}

	return "Default mock response", nil
}

type mockError struct {
	message string
}

func (e *mockError) Error() string {
	return e.message
}

func TestNewSummarizer(t *testing.T) {
	mockClient := NewMockLLMClient()
	opts := DefaultSummarizerOptions()

	summarizer := NewSummarizer(mockClient, opts)

	if summarizer == nil {
		t.Fatal("Expected summarizer to be created")
	}

	if summarizer.llmClient != mockClient {
		t.Error("LLM client not set correctly")
	}

	if summarizer.options.DefaultMaxWords != 150 {
		t.Errorf("Expected default max words to be 150, got %d", summarizer.options.DefaultMaxWords)
	}
}

func TestSummarizeArticle(t *testing.T) {
	mockClient := NewMockLLMClient()
	summarizer := NewSummarizerWithDefaults(mockClient)

	article := &core.Article{
		ID:          "test-123",
		Title:       "Test Article",
		CleanedText: "This is test article content with meaningful information that needs to be summarized. It contains multiple sentences and paragraphs.",
	}

	ctx := context.Background()
	summary, err := summarizer.SummarizeArticle(ctx, article)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if summary == nil {
		t.Fatal("Expected summary to be created")
	}

	if summary.SummaryText == "" {
		t.Error("Expected summary text to be populated")
	}

	if summary.ModelUsed == "" {
		t.Error("Expected model used to be set")
	}

	if len(summary.ArticleIDs) != 1 || summary.ArticleIDs[0] != article.ID {
		t.Error("Expected article ID to be in summary")
	}

	if mockClient.callCount != 1 {
		t.Errorf("Expected 1 LLM call, got %d", mockClient.callCount)
	}
}

func TestSummarizeArticleWithRetry(t *testing.T) {
	mockClient := NewMockLLMClient()
	mockClient.shouldFail = true

	opts := DefaultSummarizerOptions()
	opts.MaxRetries = 2
	opts.RetryDelay = time.Millisecond

	summarizer := NewSummarizer(mockClient, opts)

	article := &core.Article{
		ID:          "test-123",
		Title:       "Test Article",
		CleanedText: "Content to summarize",
	}

	ctx := context.Background()
	_, err := summarizer.SummarizeArticle(ctx, article)

	if err == nil {
		t.Error("Expected error when LLM fails")
	}

	// Should have tried: initial + 2 retries = 3 calls
	expectedCalls := 3
	if mockClient.callCount != expectedCalls {
		t.Errorf("Expected %d LLM calls with retries, got %d", expectedCalls, mockClient.callCount)
	}
}

func TestSummarizeArticleNilArticle(t *testing.T) {
	mockClient := NewMockLLMClient()
	summarizer := NewSummarizerWithDefaults(mockClient)

	ctx := context.Background()
	_, err := summarizer.SummarizeArticle(ctx, nil)

	if err == nil {
		t.Error("Expected error for nil article")
	}

	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("Expected 'nil' in error message, got: %v", err)
	}
}

func TestSummarizeArticleEmptyContent(t *testing.T) {
	mockClient := NewMockLLMClient()
	summarizer := NewSummarizerWithDefaults(mockClient)

	article := &core.Article{
		ID:          "test-123",
		Title:       "Test Article",
		CleanedText: "", // Empty content
	}

	ctx := context.Background()
	_, err := summarizer.SummarizeArticle(ctx, article)

	if err == nil {
		t.Error("Expected error for empty content")
	}

	if !strings.Contains(err.Error(), "no content") {
		t.Errorf("Expected 'no content' in error message, got: %v", err)
	}
}

func TestGenerateKeyPoints(t *testing.T) {
	mockClient := NewMockLLMClient()
	summarizer := NewSummarizerWithDefaults(mockClient)

	content := "Long article content with multiple important points that need to be extracted."

	ctx := context.Background()
	keyPoints, err := summarizer.GenerateKeyPoints(ctx, content)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(keyPoints) == 0 {
		t.Error("Expected key points to be extracted")
	}

	if len(keyPoints) != 5 {
		t.Errorf("Expected 5 key points, got %d", len(keyPoints))
	}
}

func TestExtractTitle(t *testing.T) {
	mockClient := NewMockLLMClient()
	summarizer := NewSummarizerWithDefaults(mockClient)

	content := "Article content about an interesting topic that needs a title."

	ctx := context.Background()
	title, err := summarizer.ExtractTitle(ctx, content)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if title == "" {
		t.Error("Expected title to be generated")
	}

	if title != "Generated Test Article Title" {
		t.Errorf("Expected specific title, got: %s", title)
	}
}

func TestIdentifyTheme(t *testing.T) {
	mockClient := NewMockLLMClient()
	summarizer := NewSummarizerWithDefaults(mockClient)

	article := &core.Article{
		ID:          "test-123",
		Title:       "AI and Machine Learning Advances",
		CleanedText: "Article about recent developments in AI and ML...",
	}

	summary := &core.Summary{
		SummaryText: "Summary discussing AI innovations...",
	}

	ctx := context.Background()
	theme, err := summarizer.IdentifyTheme(ctx, article, summary)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if theme == "" {
		t.Error("Expected theme to be identified")
	}

	if theme != "Technology" {
		t.Errorf("Expected 'Technology' theme, got: %s", theme)
	}
}

func TestSummarizeBatch(t *testing.T) {
	mockClient := NewMockLLMClient()
	summarizer := NewSummarizerWithDefaults(mockClient)

	articles := []*core.Article{
		{
			ID:          "article-1",
			Title:       "First Article",
			CleanedText: "Content for first article with meaningful information.",
		},
		{
			ID:          "article-2",
			Title:       "Second Article",
			CleanedText: "Content for second article with different information.",
		},
		{
			ID:          "article-3",
			Title:       "Third Article",
			CleanedText: "Content for third article with more details.",
		},
	}

	ctx := context.Background()
	summaries, err := summarizer.SummarizeBatch(ctx, articles)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(summaries) != 3 {
		t.Errorf("Expected 3 summaries, got %d", len(summaries))
	}

	// Verify each summary is valid
	for i, summary := range summaries {
		if summary == nil {
			t.Errorf("Summary %d is nil", i)
			continue
		}

		if summary.SummaryText == "" {
			t.Errorf("Summary %d has empty text", i)
		}

		if len(summary.ArticleIDs) != 1 || summary.ArticleIDs[0] != articles[i].ID {
			t.Errorf("Summary %d has incorrect article ID", i)
		}
	}

	expectedCalls := 3
	if mockClient.callCount != expectedCalls {
		t.Errorf("Expected %d LLM calls, got %d", expectedCalls, mockClient.callCount)
	}
}

func TestParseSummaryResponse(t *testing.T) {
	tests := []struct {
		name             string
		response         string
		expectedSummary  string
		expectedKeyCount int
	}{
		{
			name: "well-formed response",
			response: `SUMMARY:
This is a test summary with multiple sentences.

KEY POINTS:
- First key point
- Second key point
- Third key point`,
			expectedSummary:  "This is a test summary with multiple sentences.",
			expectedKeyCount: 3,
		},
		{
			name: "response with numbered list",
			response: `SUMMARY:
Summary text here.

KEY POINTS:
1. First point
2. Second point
3. Third point`,
			expectedSummary:  "Summary text here.",
			expectedKeyCount: 3,
		},
		{
			name: "response without key points section",
			response: `SUMMARY:
Just a summary without key points.`,
			expectedSummary:  "Just a summary without key points.",
			expectedKeyCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary, keyPoints := ParseSummaryResponse(tt.response)

			if !strings.Contains(summary, tt.expectedSummary) {
				t.Errorf("Expected summary to contain %q, got %q", tt.expectedSummary, summary)
			}

			if len(keyPoints) != tt.expectedKeyCount {
				t.Errorf("Expected %d key points, got %d", tt.expectedKeyCount, len(keyPoints))
			}
		})
	}
}

func TestValidateSummary(t *testing.T) {
	opts := DefaultSummarizerOptions()
	opts.MinSummaryWords = 10
	opts.MaxSummaryWords = 200

	mockClient := NewMockLLMClient()
	summarizer := NewSummarizer(mockClient, opts)

	tests := []struct {
		name        string
		summary     string
		shouldError bool
	}{
		{
			name:        "valid summary",
			summary:     strings.Repeat("word ", 50), // 50 words
			shouldError: false,
		},
		{
			name:        "too short",
			summary:     "Too short",
			shouldError: true,
		},
		{
			name:        "too long",
			summary:     strings.Repeat("word ", 250), // 250 words
			shouldError: true,
		},
		{
			name:        "empty",
			summary:     "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := summarizer.validateSummary(tt.summary)

			if tt.shouldError && err == nil {
				t.Error("Expected validation error, got none")
			}

			if !tt.shouldError && err != nil {
				t.Errorf("Expected no validation error, got: %v", err)
			}
		})
	}
}

func TestExtractFallbackSummary(t *testing.T) {
	opts := DefaultSummarizerOptions()
	opts.DefaultMaxWords = 50

	mockClient := NewMockLLMClient()
	summarizer := NewSummarizer(mockClient, opts)

	content := "First sentence. Second sentence with more detail. Third sentence. Fourth sentence. Fifth sentence. Sixth sentence."

	fallback := summarizer.extractFallbackSummary(content)

	if fallback == "" {
		t.Error("Expected fallback summary to be generated")
	}

	wordCount := len(strings.Fields(fallback))
	if wordCount > opts.DefaultMaxWords {
		t.Errorf("Fallback summary too long: %d words (max: %d)", wordCount, opts.DefaultMaxWords)
	}

	if !strings.HasSuffix(fallback, ".") {
		t.Error("Expected fallback summary to end with period")
	}
}

func TestGetStats(t *testing.T) {
	mockClient := NewMockLLMClient()
	opts := DefaultSummarizerOptions()
	opts.DefaultMaxWords = 200
	opts.DefaultKeyPointCount = 7

	summarizer := NewSummarizer(mockClient, opts)

	stats := summarizer.GetStats()

	if stats.DefaultMaxWords != 200 {
		t.Errorf("Expected max words 200, got %d", stats.DefaultMaxWords)
	}

	if stats.DefaultKeyPointCount != 7 {
		t.Errorf("Expected key point count 7, got %d", stats.DefaultKeyPointCount)
	}

	if stats.ModelName == "" {
		t.Error("Expected model name to be set")
	}
}
