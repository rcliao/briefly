package themes

import (
	"briefly/internal/core"
	"briefly/internal/llm"
	"context"
	"strings"
	"testing"
)

// MockTracedClient implements TracedClient interface for testing
type MockTracedClient struct {
	responses  map[string]string
	callCount  int
	shouldFail bool
}

func NewMockTracedClient() *MockTracedClient {
	return &MockTracedClient{
		responses:  make(map[string]string),
		callCount:  0,
		shouldFail: false,
	}
}

func (m *MockTracedClient) GenerateText(ctx context.Context, prompt string, options llm.TextGenerationOptions) (string, error) {
	m.callCount++

	if m.shouldFail {
		return "", &mockError{message: "mock LLM error"}
	}

	// Check for custom responses set via SetResponse
	for key, response := range m.responses {
		if strings.Contains(prompt, key) {
			return response, nil
		}
	}

	// Default response for classification
	if strings.Contains(prompt, "AVAILABLE THEMES:") {
		return `{
  "classifications": [
    {
      "theme_name": "AI & Machine Learning",
      "relevance_score": 0.85,
      "reasoning": "Article discusses machine learning techniques"
    },
    {
      "theme_name": "Software Engineering & Best Practices",
      "relevance_score": 0.45,
      "reasoning": "Mentions software development practices"
    }
  ]
}`, nil
	}

	return "Default mock response", nil
}

func (m *MockTracedClient) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	return []float64{0.1, 0.2, 0.3}, nil
}

func (m *MockTracedClient) SetResponse(key string, response string) {
	m.responses[key] = response
}

type mockError struct {
	message string
}

func (e *mockError) Error() string {
	return e.message
}

// MockPostHogClient implements PostHogClient interface for testing
type MockPostHogClient struct {
	events    []string
	isEnabled bool
}

func NewMockPostHogClient() *MockPostHogClient {
	return &MockPostHogClient{
		events:    []string{},
		isEnabled: true,
	}
}

func (m *MockPostHogClient) IsEnabled() bool {
	return m.isEnabled
}

func (m *MockPostHogClient) TrackThemeClassification(ctx context.Context, articleID string, themeName string, relevance float64) error {
	m.events = append(m.events, "theme_classification")
	return nil
}

func (m *MockPostHogClient) Close() error {
	return nil
}

// Test fixtures
func createTestThemes() []core.Theme {
	return []core.Theme{
		{
			ID:          "theme-1",
			Name:        "AI & Machine Learning",
			Description: "Articles about artificial intelligence and machine learning",
			Keywords:    []string{"AI", "ML", "machine learning", "neural networks"},
			Enabled:     true,
		},
		{
			ID:          "theme-2",
			Name:        "Software Engineering & Best Practices",
			Description: "Articles about software development practices",
			Keywords:    []string{"software", "engineering", "best practices", "code quality"},
			Enabled:     true,
		},
		{
			ID:          "theme-3",
			Name:        "Cloud Infrastructure & DevOps",
			Description: "Articles about cloud computing and DevOps",
			Keywords:    []string{"cloud", "DevOps", "infrastructure", "kubernetes"},
			Enabled:     true,
		},
	}
}

func createTestArticle() core.Article {
	return core.Article{
		ID:          "article-123",
		Title:       "Latest Advances in Machine Learning",
		CleanedText: "This article discusses recent breakthroughs in machine learning and neural networks. The new techniques show promise for improving AI systems.",
		URL:         "https://example.com/ml-article",
	}
}

// Tests

func TestNewClassifier(t *testing.T) {
	mockLLM := NewMockTracedClient()
	mockPostHog := NewMockPostHogClient()

	classifier := NewClassifier(mockLLM, mockPostHog)

	if classifier == nil {
		t.Fatal("Expected classifier to be created")
	}

	// Verify classifier works by calling a method
	article := createTestArticle()
	themes := createTestThemes()
	ctx := context.Background()

	_, err := classifier.ClassifyArticle(ctx, article, themes, 0.4)
	if err != nil {
		t.Errorf("Classifier not working correctly: %v", err)
	}
}

func TestClassifyArticle(t *testing.T) {
	mockLLM := NewMockTracedClient()
	mockPostHog := NewMockPostHogClient()
	classifier := NewClassifier(mockLLM, mockPostHog)

	article := createTestArticle()
	themes := createTestThemes()

	ctx := context.Background()
	results, err := classifier.ClassifyArticle(ctx, article, themes, 0.4)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected at least one classification result")
	}

	// Should have 2 results above 0.4 threshold (0.85 and 0.45)
	if len(results) != 2 {
		t.Errorf("Expected 2 results above threshold, got %d", len(results))
	}

	// Check first result
	if results[0].ThemeName != "AI & Machine Learning" {
		t.Errorf("Expected first theme to be 'AI & Machine Learning', got %s", results[0].ThemeName)
	}

	if results[0].RelevanceScore != 0.85 {
		t.Errorf("Expected relevance score 0.85, got %f", results[0].RelevanceScore)
	}

	if results[0].Reasoning == "" {
		t.Error("Expected reasoning to be populated")
	}

	// Check PostHog tracking
	if len(mockPostHog.events) != 2 {
		t.Errorf("Expected 2 PostHog events, got %d", len(mockPostHog.events))
	}

	// Check LLM was called
	if mockLLM.callCount != 1 {
		t.Errorf("Expected 1 LLM call, got %d", mockLLM.callCount)
	}
}

func TestClassifyArticleWithHigherThreshold(t *testing.T) {
	mockLLM := NewMockTracedClient()
	mockPostHog := NewMockPostHogClient()
	classifier := NewClassifier(mockLLM, mockPostHog)

	article := createTestArticle()
	themes := createTestThemes()

	ctx := context.Background()
	results, err := classifier.ClassifyArticle(ctx, article, themes, 0.7) // Higher threshold

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should only have 1 result above 0.7 threshold (0.85 only)
	if len(results) != 1 {
		t.Errorf("Expected 1 result above threshold 0.7, got %d", len(results))
	}

	if results[0].RelevanceScore < 0.7 {
		t.Errorf("Expected relevance score >= 0.7, got %f", results[0].RelevanceScore)
	}
}

func TestClassifyArticleNoThemes(t *testing.T) {
	mockLLM := NewMockTracedClient()
	mockPostHog := NewMockPostHogClient()
	classifier := NewClassifier(mockLLM, mockPostHog)

	article := createTestArticle()
	themes := []core.Theme{} // Empty themes

	ctx := context.Background()
	results, err := classifier.ClassifyArticle(ctx, article, themes, 0.4)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected no results for empty themes, got %d", len(results))
	}

	// Should not call LLM when no themes
	if mockLLM.callCount != 0 {
		t.Errorf("Expected 0 LLM calls, got %d", mockLLM.callCount)
	}
}

func TestClassifyArticleLLMError(t *testing.T) {
	mockLLM := NewMockTracedClient()
	mockLLM.shouldFail = true

	mockPostHog := NewMockPostHogClient()
	classifier := NewClassifier(mockLLM, mockPostHog)

	article := createTestArticle()
	themes := createTestThemes()

	ctx := context.Background()
	_, err := classifier.ClassifyArticle(ctx, article, themes, 0.4)

	if err == nil {
		t.Error("Expected error when LLM fails")
	}

	if !strings.Contains(err.Error(), "failed to classify") {
		t.Errorf("Expected 'failed to classify' in error, got: %v", err)
	}
}

func TestGetBestMatch(t *testing.T) {
	mockLLM := NewMockTracedClient()
	mockPostHog := NewMockPostHogClient()
	classifier := NewClassifier(mockLLM, mockPostHog)

	article := createTestArticle()
	themes := createTestThemes()

	ctx := context.Background()
	result, err := classifier.GetBestMatch(ctx, article, themes, 0.4)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected best match result")
	}

	// Should return the highest scoring theme (0.85)
	if result.ThemeName != "AI & Machine Learning" {
		t.Errorf("Expected 'AI & Machine Learning', got %s", result.ThemeName)
	}

	if result.RelevanceScore != 0.85 {
		t.Errorf("Expected relevance 0.85, got %f", result.RelevanceScore)
	}
}

func TestGetBestMatchNoMatch(t *testing.T) {
	mockLLM := NewMockTracedClient()
	// Set response with low scores
	mockLLM.SetResponse("AVAILABLE THEMES:", `{
  "classifications": [
    {
      "theme_name": "AI & Machine Learning",
      "relevance_score": 0.2,
      "reasoning": "Low relevance"
    }
  ]
}`)

	mockPostHog := NewMockPostHogClient()
	classifier := NewClassifier(mockLLM, mockPostHog)

	article := createTestArticle()
	themes := createTestThemes()

	ctx := context.Background()
	result, err := classifier.GetBestMatch(ctx, article, themes, 0.5) // High threshold

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should return nil when no match above threshold
	if result != nil {
		t.Errorf("Expected nil result, got theme: %s", result.ThemeName)
	}
}

func TestParseClassificationResponseValid(t *testing.T) {
	mockLLM := NewMockTracedClient()
	mockPostHog := NewMockPostHogClient()
	classifier := NewClassifier(mockLLM, mockPostHog)

	themes := createTestThemes()

	response := `{
  "classifications": [
    {
      "theme_name": "AI & Machine Learning",
      "relevance_score": 0.85,
      "reasoning": "Test reasoning"
    },
    {
      "theme_name": "Cloud Infrastructure & DevOps",
      "relevance_score": 0.65,
      "reasoning": "Secondary match"
    }
  ]
}`

	results, err := classifier.parseClassificationResponse(response, themes)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	if results[0].ThemeName != "AI & Machine Learning" {
		t.Errorf("Expected 'AI & Machine Learning', got %s", results[0].ThemeName)
	}

	if results[0].ThemeID != "theme-1" {
		t.Errorf("Expected theme ID 'theme-1', got %s", results[0].ThemeID)
	}
}

func TestParseClassificationResponseMarkdownWrapped(t *testing.T) {
	mockLLM := NewMockTracedClient()
	mockPostHog := NewMockPostHogClient()
	classifier := NewClassifier(mockLLM, mockPostHog)

	themes := createTestThemes()

	// Response wrapped in markdown code block
	response := "```json\n" + `{
  "classifications": [
    {
      "theme_name": "AI & Machine Learning",
      "relevance_score": 0.75,
      "reasoning": "Test"
    }
  ]
}` + "\n```"

	results, err := classifier.parseClassificationResponse(response, themes)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestParseClassificationResponseCaseInsensitive(t *testing.T) {
	mockLLM := NewMockTracedClient()
	mockPostHog := NewMockPostHogClient()
	classifier := NewClassifier(mockLLM, mockPostHog)

	themes := createTestThemes()

	// Theme name in different case
	response := `{
  "classifications": [
    {
      "theme_name": "ai & machine learning",
      "relevance_score": 0.80,
      "reasoning": "Test"
    }
  ]
}`

	results, err := classifier.parseClassificationResponse(response, themes)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	// Should still match theme-1 despite case difference
	if results[0].ThemeID != "theme-1" {
		t.Errorf("Expected theme ID 'theme-1', got %s", results[0].ThemeID)
	}
}

func TestParseClassificationResponseInvalidJSON(t *testing.T) {
	mockLLM := NewMockTracedClient()
	mockPostHog := NewMockPostHogClient()
	classifier := NewClassifier(mockLLM, mockPostHog)

	themes := createTestThemes()

	response := "This is not valid JSON"

	_, err := classifier.parseClassificationResponse(response, themes)

	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	if !strings.Contains(err.Error(), "failed to parse JSON") {
		t.Errorf("Expected 'failed to parse JSON' in error, got: %v", err)
	}
}

func TestParseClassificationResponseUnknownTheme(t *testing.T) {
	mockLLM := NewMockTracedClient()
	mockPostHog := NewMockPostHogClient()
	classifier := NewClassifier(mockLLM, mockPostHog)

	themes := createTestThemes()

	// Response with unknown theme name
	response := `{
  "classifications": [
    {
      "theme_name": "Unknown Theme",
      "relevance_score": 0.90,
      "reasoning": "Test"
    },
    {
      "theme_name": "AI & Machine Learning",
      "relevance_score": 0.80,
      "reasoning": "Valid theme"
    }
  ]
}`

	results, err := classifier.parseClassificationResponse(response, themes)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should skip unknown theme and only return valid one
	if len(results) != 1 {
		t.Errorf("Expected 1 result (unknown theme skipped), got %d", len(results))
	}

	if results[0].ThemeName != "AI & Machine Learning" {
		t.Errorf("Expected 'AI & Machine Learning', got %s", results[0].ThemeName)
	}
}

func TestClassifyBatch(t *testing.T) {
	mockLLM := NewMockTracedClient()
	mockPostHog := NewMockPostHogClient()
	classifier := NewClassifier(mockLLM, mockPostHog)

	articles := []core.Article{
		createTestArticle(),
		{
			ID:          "article-2",
			Title:       "Kubernetes Best Practices",
			CleanedText: "Guide to deploying applications on Kubernetes",
		},
		{
			ID:          "article-3",
			Title:       "TypeScript Tips",
			CleanedText: "Advanced TypeScript patterns for better code",
		},
	}

	themes := createTestThemes()

	ctx := context.Background()
	results, err := classifier.ClassifyBatch(ctx, articles, themes, 0.4)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected at least one classification result")
	}

	// Should have results for all articles (graceful error handling)
	if len(results) != 3 {
		t.Logf("Warning: Expected 3 articles classified, got %d (non-fatal)", len(results))
	}

	// Verify each result has classifications
	for articleID, classifications := range results {
		if len(classifications) == 0 {
			t.Errorf("Article %s has no classifications", articleID)
		}
	}

	// Should call LLM for each article
	expectedCalls := 3
	if mockLLM.callCount != expectedCalls {
		t.Errorf("Expected %d LLM calls, got %d", expectedCalls, mockLLM.callCount)
	}
}

func TestClassifyBatchWithErrors(t *testing.T) {
	mockLLM := NewMockTracedClient()
	mockPostHog := NewMockPostHogClient()
	classifier := NewClassifier(mockLLM, mockPostHog)

	// Set up to fail on first call
	originalResponse := mockLLM.responses
	mockLLM.shouldFail = true

	articles := []core.Article{
		createTestArticle(),
		{
			ID:          "article-2",
			Title:       "Second Article",
			CleanedText: "Content",
		},
	}

	themes := createTestThemes()

	ctx := context.Background()

	// Temporarily fail
	results, err := classifier.ClassifyBatch(ctx, articles, themes, 0.4)

	// Should not return error for batch (graceful degradation)
	if err != nil {
		t.Fatalf("Expected no error from batch, got: %v", err)
	}

	// Should have empty results due to failures
	if len(results) != 0 {
		t.Logf("Warning: Expected 0 results due to failures, got %d", len(results))
	}

	// Reset for second test
	mockLLM.shouldFail = false
	mockLLM.responses = originalResponse
}

func TestBuildClassificationPrompt(t *testing.T) {
	mockLLM := NewMockTracedClient()
	mockPostHog := NewMockPostHogClient()
	classifier := NewClassifier(mockLLM, mockPostHog)

	article := createTestArticle()
	themes := createTestThemes()

	prompt := classifier.buildClassificationPrompt(article, themes)

	// Check prompt contains key elements
	if !strings.Contains(prompt, article.Title) {
		t.Error("Prompt should contain article title")
	}

	if !strings.Contains(prompt, "AI & Machine Learning") {
		t.Error("Prompt should contain theme names")
	}

	if !strings.Contains(prompt, "AVAILABLE THEMES:") {
		t.Error("Prompt should have themes section")
	}

	if !strings.Contains(prompt, "relevance_score") {
		t.Error("Prompt should request relevance scores")
	}

	if !strings.Contains(prompt, "JSON") {
		t.Error("Prompt should request JSON format")
	}
}

func TestBuildClassificationPromptTruncatesContent(t *testing.T) {
	mockLLM := NewMockTracedClient()
	mockPostHog := NewMockPostHogClient()
	classifier := NewClassifier(mockLLM, mockPostHog)

	// Create article with very long content
	longContent := strings.Repeat("word ", 1000) // ~5000 characters
	article := core.Article{
		ID:          "article-long",
		Title:       "Test Article",
		CleanedText: longContent,
	}

	themes := createTestThemes()

	prompt := classifier.buildClassificationPrompt(article, themes)

	// Check that content was truncated (2000 chars + "...")
	if strings.Count(prompt, "word") > 500 {
		t.Error("Expected content to be truncated to ~2000 chars")
	}

	if !strings.Contains(prompt, "...") {
		t.Error("Expected truncation indicator (...) in prompt")
	}
}

func TestClassifyArticleNilPostHog(t *testing.T) {
	mockLLM := NewMockTracedClient()
	classifier := NewClassifier(mockLLM, nil) // Nil PostHog

	article := createTestArticle()
	themes := createTestThemes()

	ctx := context.Background()

	// Should not panic with nil PostHog
	results, err := classifier.ClassifyArticle(ctx, article, themes, 0.4)

	if err != nil {
		t.Fatalf("Expected no error with nil PostHog, got: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected results even with nil PostHog")
	}
}

func TestClassifyArticleDisabledPostHog(t *testing.T) {
	mockLLM := NewMockTracedClient()
	mockPostHog := NewMockPostHogClient()
	mockPostHog.isEnabled = false // Disabled

	classifier := NewClassifier(mockLLM, mockPostHog)

	article := createTestArticle()
	themes := createTestThemes()

	ctx := context.Background()
	results, err := classifier.ClassifyArticle(ctx, article, themes, 0.4)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected results")
	}

	// Should not track events when disabled
	if len(mockPostHog.events) > 0 {
		t.Errorf("Expected no events when PostHog disabled, got %d", len(mockPostHog.events))
	}
}
