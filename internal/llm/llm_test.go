package llm

import (
	"briefly/internal/core"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestNewClient_Success(t *testing.T) {
	// Skip if no API key available (for CI/CD)
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	client, err := NewClient("")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	if client.apiKey == "" {
		t.Error("Client API key should not be empty")
	}
	if client.modelName == "" {
		t.Error("Client model name should not be empty")
	}
	if client.gClient == nil {
		t.Error("Client gClient should not be nil")
	}
}

func TestNewClient_NoAPIKey(t *testing.T) {
	// Temporarily unset API key
	originalKey := os.Getenv("GEMINI_API_KEY")
	_ = os.Unsetenv("GEMINI_API_KEY")
	viper.Set("gemini.api_key", "") // Clear viper as well
	defer func() {
		if originalKey != "" {
			_ = os.Setenv("GEMINI_API_KEY", originalKey)
		}
	}()

	_, err := NewClient("")
	if err == nil {
		t.Error("Expected error when no API key is available")
	}
	if !strings.Contains(err.Error(), "gemini API key is required") {
		t.Errorf("Expected API key error, got: %v", err)
	}
}

func TestNewClient_WithViperConfig(t *testing.T) {
	// Skip if no API key available
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	// Temporarily unset env var and use viper
	_ = os.Unsetenv("GEMINI_API_KEY")
	viper.Set("gemini.api_key", apiKey)
	viper.Set("gemini.model", "gemini-1.5-flash")
	defer func() {
		_ = os.Setenv("GEMINI_API_KEY", apiKey)
		viper.Set("gemini.api_key", "")
		viper.Set("gemini.model", "")
	}()

	client, err := NewClient("")
	if err != nil {
		t.Fatalf("NewClient with viper config failed: %v", err)
	}
	defer client.Close()

	if client.modelName != "gemini-1.5-flash" {
		t.Errorf("Expected model 'gemini-1.5-flash', got '%s'", client.modelName)
	}
}

func TestNewClient_CustomModel(t *testing.T) {
	// Skip if no API key available
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	customModel := "gemini-1.5-flash"
	client, err := NewClient(customModel)
	if err != nil {
		t.Fatalf("NewClient with custom model failed: %v", err)
	}
	defer client.Close()

	if client.modelName != customModel {
		t.Errorf("Expected model '%s', got '%s'", customModel, client.modelName)
	}
}

func TestSummarizeArticleText_EmptyContent(t *testing.T) {
	// Skip if no API key available
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	client, err := NewClient("")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	article := core.Article{
		ID:          "test-id",
		CleanedText: "", // Empty content
	}

	_, err = client.SummarizeArticleText(article)
	if err == nil {
		t.Error("Expected error for empty CleanedText")
	}
	if !strings.Contains(err.Error(), "no CleanedText to summarize") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestSummarizeArticleTextWithFormat_EmptyContent(t *testing.T) {
	// Skip if no API key available
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	client, err := NewClient("")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	article := core.Article{
		ID:          "test-id",
		CleanedText: "", // Empty content
	}

	_, err = client.SummarizeArticleTextWithFormat(article, "brief")
	if err == nil {
		t.Error("Expected error for empty CleanedText")
	}
	if !strings.Contains(err.Error(), "no CleanedText to summarize") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestSummarizeArticleWithKeyMoments_EmptyContent(t *testing.T) {
	// Skip if no API key available
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	client, err := NewClient("")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	article := core.Article{
		ID:          "test-id",
		CleanedText: "", // Empty content
	}

	_, err = client.SummarizeArticleWithKeyMoments(article)
	if err == nil {
		t.Error("Expected error for empty CleanedText")
	}
	if !strings.Contains(err.Error(), "no CleanedText to summarize") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestGenerateSummary_NoAPIKey(t *testing.T) {
	// Temporarily unset API key
	originalKey := os.Getenv("GEMINI_API_KEY")
	_ = os.Unsetenv("GEMINI_API_KEY")
	defer func() {
		if originalKey != "" {
			_ = os.Setenv("GEMINI_API_KEY", originalKey)
		}
	}()

	_, err := GenerateSummary("test content")
	if err == nil {
		t.Error("Expected error when GEMINI_API_KEY is not set")
	}
	if !strings.Contains(err.Error(), "GEMINI_API_KEY environment variable not set") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestRegenerateDigestWithMyTake_NoAPIKey(t *testing.T) {
	// Temporarily unset API key
	originalKey := os.Getenv("GEMINI_API_KEY")
	_ = os.Unsetenv("GEMINI_API_KEY")
	defer func() {
		if originalKey != "" {
			_ = os.Setenv("GEMINI_API_KEY", originalKey)
		}
	}()

	_, err := RegenerateDigestWithMyTake("digest content", "my take", "standard")
	if err == nil {
		t.Error("Expected error when GEMINI_API_KEY is not set")
	}
	if !strings.Contains(err.Error(), "GEMINI_API_KEY environment variable not set") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestGeneratePromptCorner_NoAPIKey(t *testing.T) {
	// Temporarily unset API key
	originalKey := os.Getenv("GEMINI_API_KEY")
	_ = os.Unsetenv("GEMINI_API_KEY")
	defer func() {
		if originalKey != "" {
			_ = os.Setenv("GEMINI_API_KEY", originalKey)
		}
	}()

	_, err := GeneratePromptCorner("digest content")
	if err == nil {
		t.Error("Expected error when GEMINI_API_KEY is not set")
	}
	if !strings.Contains(err.Error(), "GEMINI_API_KEY environment variable not set") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestGenerateDigestTitle_NoAPIKey(t *testing.T) {
	// Temporarily unset API key
	originalKey := os.Getenv("GEMINI_API_KEY")
	_ = os.Unsetenv("GEMINI_API_KEY")
	defer func() {
		if originalKey != "" {
			_ = os.Setenv("GEMINI_API_KEY", originalKey)
		}
	}()

	_, err := GenerateDigestTitle("digest content", "standard")
	if err == nil {
		t.Error("Expected error when GEMINI_API_KEY is not set")
	}
	if !strings.Contains(err.Error(), "GEMINI_API_KEY environment variable not set") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestGenerateDigestTitle_EmptyContent(t *testing.T) {
	// Skip if no API key available
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	client, err := NewClient("")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	_, err = client.GenerateDigestTitle("", "standard")
	if err == nil {
		t.Error("Expected error for empty digest content")
	}
	if !strings.Contains(err.Error(), "cannot generate title for empty digest content") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestGenerateEmbeddingForArticle_EmptyContent(t *testing.T) {
	// Skip if no API key available
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	client, err := NewClient("")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	article := core.Article{
		ID:          "test-id",
		Title:       "",
		CleanedText: "",
	}

	// Should handle empty content gracefully
	embedding, err := client.GenerateEmbeddingForArticle(article)
	if err != nil {
		t.Fatalf("GenerateEmbeddingForArticle failed: %v", err)
	}

	if len(embedding) == 0 {
		t.Error("Expected non-empty embedding even for empty content")
	}
}

func TestGenerateResearchQueries_EmptyContent(t *testing.T) {
	// Skip if no API key available
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	client, err := NewClient("")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	article := core.Article{
		ID:          "test-id",
		CleanedText: "", // Empty content
	}

	_, err = client.GenerateResearchQueries(article, 3)
	if err == nil {
		t.Error("Expected error for empty CleanedText")
	}
	if !strings.Contains(err.Error(), "no CleanedText for research query generation") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestCosineSimilarity(t *testing.T) {
	testCases := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float64{1.0, 2.0, 3.0},
			b:        []float64{1.0, 2.0, 3.0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float64{1.0, 0.0},
			b:        []float64{0.0, 1.0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float64{1.0, 0.0},
			b:        []float64{-1.0, 0.0},
			expected: -1.0,
		},
		{
			name:     "different lengths",
			a:        []float64{1.0, 2.0},
			b:        []float64{1.0, 2.0, 3.0},
			expected: 0.0,
		},
		{
			name:     "zero vector",
			a:        []float64{0.0, 0.0},
			b:        []float64{1.0, 2.0},
			expected: 0.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := CosineSimilarity(tc.a, tc.b)
			if fmt.Sprintf("%.6f", result) != fmt.Sprintf("%.6f", tc.expected) {
				t.Errorf("Expected %.6f, got %.6f", tc.expected, result)
			}
		})
	}
}

func TestGenerateTrendAnalysisPrompt(t *testing.T) {
	// Skip if no API key available
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	client, err := NewClient("")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	currentTopics := []string{"AI", "machine learning", "cloud computing"}
	previousTopics := []string{"blockchain", "machine learning", "mobile development"}
	timeframe := "this week"

	prompt := client.GenerateTrendAnalysisPrompt(currentTopics, previousTopics, timeframe)

	if prompt == "" {
		t.Error("Generated prompt should not be empty")
	}
	if !strings.Contains(prompt, "AI") {
		t.Errorf("Prompt should contain current topics, got: %s", prompt)
	}
	if !strings.Contains(prompt, "blockchain") {
		t.Errorf("Prompt should contain previous topics, got: %s", prompt)
	}
	if !strings.Contains(prompt, timeframe) {
		t.Errorf("Prompt should contain timeframe, got: %s", prompt)
	}
	if !strings.Contains(prompt, "trend analysis") {
		t.Error("Prompt should mention trend analysis")
	}
}

func TestConstants(t *testing.T) {
	if DefaultModel == "" {
		t.Error("DefaultModel should not be empty")
	}
	if DefaultEmbeddingModel == "" {
		t.Error("DefaultEmbeddingModel should not be empty")
	}
	if SummarizeTextPromptTemplate == "" {
		t.Error("SummarizeTextPromptTemplate should not be empty")
	}
	if SummarizeTextWithFormatPromptTemplate == "" {
		t.Error("SummarizeTextWithFormatPromptTemplate should not be empty")
	}

	// Check that the prompt templates contain expected placeholders
	if !strings.Contains(SummarizeTextPromptTemplate, "%s") {
		t.Errorf("SummarizeTextPromptTemplate should contain %%s placeholder")
	}
	if strings.Count(SummarizeTextWithFormatPromptTemplate, "%s") < 2 {
		t.Errorf("SummarizeTextWithFormatPromptTemplate should contain at least 2 %%s placeholders, got %d", strings.Count(SummarizeTextWithFormatPromptTemplate, "%s"))
	}
}

func TestClientClose(t *testing.T) {
	// Skip if no API key available
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	client, err := NewClient("")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// Should not panic
	client.Close()
	// Should be safe to call multiple times
	client.Close()
}

// Mock tests for functions that require API calls
func TestSummaryStructure(t *testing.T) {
	// Test that the Summary struct is properly populated (without making API calls)

	// Skip if no API key available
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	client, err := NewClient("")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	article := core.Article{
		ID:          "test-article-id",
		CleanedText: "This is a short test article for summarization testing purposes.",
		Title:       "Test Article",
	}

	// Test basic summarization
	summary, err := client.SummarizeArticleText(article)
	if err != nil {
		t.Fatalf("SummarizeArticleText failed: %v", err)
	}

	if len(summary.ArticleIDs) != 1 || summary.ArticleIDs[0] != article.ID {
		t.Error("Summary should contain the correct article ID")
	}
	if summary.SummaryText == "" {
		t.Error("Summary text should not be empty")
	}
	if summary.ModelUsed != client.modelName {
		t.Errorf("Expected model used '%s', got '%s'", client.modelName, summary.ModelUsed)
	}
	if summary.Instructions == "" {
		t.Error("Instructions should not be empty")
	}
}

func TestSummaryWithKeyMomentsStructure(t *testing.T) {
	// Skip if no API key available
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	client, err := NewClient("")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	article := core.Article{
		ID:          "test-article-id",
		CleanedText: "This is a test article about artificial intelligence breakthroughs. The technology has shown significant improvements in performance metrics. New developments include faster processing and better accuracy.",
		Title:       "AI Breakthrough Article",
	}

	summary, err := client.SummarizeArticleWithKeyMoments(article)
	if err != nil {
		t.Fatalf("SummarizeArticleWithKeyMoments failed: %v", err)
	}

	if summary.DateGenerated.IsZero() {
		t.Error("DateGenerated should be set for key moments summary")
	}
	if !summary.DateGenerated.Before(time.Now().Add(time.Minute)) {
		t.Error("DateGenerated should be close to current time")
	}
}

// Integration test for actual API functionality (when API key is available)
func TestLiveAPIIntegration(t *testing.T) {
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("GEMINI_API_KEY not set, skipping live API integration test")
	}

	client, err := NewClient("")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Test with real but simple content
	testContent := "Artificial intelligence has made significant progress in recent years. Machine learning models are becoming more sophisticated and capable of handling complex tasks. This advancement is transforming various industries including healthcare, finance, and technology."

	article := core.Article{
		ID:          "integration-test-id",
		CleanedText: testContent,
		Title:       "AI Progress Update",
	}

	// Test basic summarization
	summary, err := client.SummarizeArticleText(article)
	if err != nil {
		t.Fatalf("Live API summarization failed: %v", err)
	}

	if summary.SummaryText == "" {
		t.Error("Live API should return non-empty summary")
	}
	if len(summary.SummaryText) >= len(testContent) {
		t.Error("Summary should be shorter than original content")
	}

	// Test embedding generation
	embedding, err := client.GenerateEmbeddingForArticle(article)
	if err != nil {
		t.Fatalf("Live API embedding generation failed: %v", err)
	}

	if len(embedding) == 0 {
		t.Error("Live API should return non-empty embedding")
	}

	// text-embedding-004 typically returns 768-dimensional embeddings
	if len(embedding) < 100 {
		t.Error("Embedding dimension seems too small")
	}
}
