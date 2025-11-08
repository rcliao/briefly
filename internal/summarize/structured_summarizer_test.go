package summarize

import (
	"briefly/internal/core"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/generative-ai-go/genai"
)

// MockLLMClientStructured implements LLMClient for testing structured summaries
type MockLLMClientStructured struct {
	response   string
	shouldFail bool
	callCount  int
	failUntil  int // Fail until this attempt number
}

func (m *MockLLMClientStructured) GenerateText(ctx context.Context, prompt string, options interface{}) (string, error) {
	m.callCount++

	// Fail for testing retry logic
	if m.shouldFail || (m.failUntil > 0 && m.callCount <= m.failUntil) {
		return "", fmt.Errorf("mock LLM error")
	}

	// Return mock structured response
	if m.response != "" {
		return m.response, nil
	}

	// Default valid structured response
	defaultResponse := core.StructuredSummaryContent{
		KeyPoints: []string{
			"Point 1: First key point",
			"Point 2: Second key point",
			"Point 3: Third key point",
		},
		Context:          "This is the background context explaining why this matters.",
		MainInsight:      "The core takeaway is that this demonstrates structured summarization.",
		TechnicalDetails: "Technical implementation uses Gemini response_schema API.",
		Impact:           "This improves readability and enables programmatic processing of summaries.",
	}

	jsonBytes, _ := json.Marshal(defaultResponse)
	return string(jsonBytes), nil
}

// Test CreateStructuredSummarySchema
func TestCreateStructuredSummarySchema(t *testing.T) {
	schema := CreateStructuredSummarySchema()

	if schema == nil {
		t.Fatal("Schema should not be nil")
	}

	if schema.Type != genai.TypeObject {
		t.Errorf("Expected schema type Object, got %v", schema.Type)
	}

	// Check required fields
	requiredFields := map[string]bool{
		"key_points":   false,
		"context":      false,
		"main_insight": false,
	}

	for _, field := range schema.Required {
		if _, exists := requiredFields[field]; exists {
			requiredFields[field] = true
		}
	}

	for field, found := range requiredFields {
		if !found {
			t.Errorf("Required field '%s' not found in schema.Required", field)
		}
	}

	// Check properties exist
	expectedProps := []string{"key_points", "context", "main_insight", "technical_details", "impact"}
	for _, prop := range expectedProps {
		if _, exists := schema.Properties[prop]; !exists {
			t.Errorf("Property '%s' not found in schema", prop)
		}
	}

	// Check key_points is array type
	if keyPointsProp := schema.Properties["key_points"]; keyPointsProp.Type != genai.TypeArray {
		t.Errorf("key_points should be array type, got %v", keyPointsProp.Type)
	}
}

// Test BuildStructuredSummaryPrompt
func TestBuildStructuredSummaryPrompt(t *testing.T) {
	title := "Test Article"
	content := "This is test content about a technical topic."

	prompt := BuildStructuredSummaryPrompt(title, content)

	if prompt == "" {
		t.Fatal("Prompt should not be empty")
	}

	// Prompt should contain title
	if !contains(prompt, title) {
		t.Error("Prompt should contain article title")
	}

	// Prompt should contain content
	if !contains(prompt, content) {
		t.Error("Prompt should contain article content")
	}

	// Prompt should mention structured sections
	if !contains(prompt, "KEY POINTS") {
		t.Error("Prompt should mention KEY POINTS section")
	}
	if !contains(prompt, "CONTEXT") {
		t.Error("Prompt should mention CONTEXT section")
	}
	if !contains(prompt, "MAIN INSIGHT") {
		t.Error("Prompt should mention MAIN INSIGHT section")
	}
}

// Test SummarizeArticleStructured - success case
func TestSummarizeArticleStructured_Success(t *testing.T) {
	mockClient := &MockLLMClientStructured{}
	summarizer := NewSummarizerWithDefaults(mockClient)

	article := &core.Article{
		ID:          "test-123",
		Title:       "Test Article",
		CleanedText: "This is a test article about structured summaries.",
		URL:         "https://example.com/test",
	}

	summary, err := summarizer.SummarizeArticleStructured(context.Background(), article)
	if err != nil {
		t.Fatalf("SummarizeArticleStructured failed: %v", err)
	}

	if summary == nil {
		t.Fatal("Summary should not be nil")
	}

	// Check summary fields
	if summary.SummaryType != "structured" {
		t.Errorf("Expected SummaryType 'structured', got '%s'", summary.SummaryType)
	}

	if summary.StructuredContent == nil {
		t.Fatal("StructuredContent should not be nil")
	}

	// Check structured content
	if len(summary.StructuredContent.KeyPoints) == 0 {
		t.Error("KeyPoints should not be empty")
	}

	if summary.StructuredContent.Context == "" {
		t.Error("Context should not be empty")
	}

	if summary.StructuredContent.MainInsight == "" {
		t.Error("MainInsight should not be empty")
	}

	// Check that plain text version was generated
	if summary.SummaryText == "" {
		t.Error("SummaryText should not be empty (should have rendered version)")
	}

	// Check article ID was set
	if len(summary.ArticleIDs) != 1 || summary.ArticleIDs[0] != article.ID {
		t.Errorf("Expected ArticleIDs to contain '%s'", article.ID)
	}
}

// Test SummarizeArticleStructured - nil article
func TestSummarizeArticleStructured_NilArticle(t *testing.T) {
	mockClient := &MockLLMClientStructured{}
	summarizer := NewSummarizerWithDefaults(mockClient)

	_, err := summarizer.SummarizeArticleStructured(context.Background(), nil)
	if err == nil {
		t.Fatal("Expected error for nil article")
	}
}

// Test SummarizeArticleStructured - empty content
func TestSummarizeArticleStructured_EmptyContent(t *testing.T) {
	mockClient := &MockLLMClientStructured{}
	summarizer := NewSummarizerWithDefaults(mockClient)

	article := &core.Article{
		ID:          "test-123",
		Title:       "Test Article",
		CleanedText: "", // Empty
	}

	_, err := summarizer.SummarizeArticleStructured(context.Background(), article)
	if err == nil {
		t.Fatal("Expected error for empty content")
	}
}

// Test SummarizeArticleStructured - LLM failure
func TestSummarizeArticleStructured_LLMFailure(t *testing.T) {
	mockClient := &MockLLMClientStructured{
		shouldFail: true,
	}
	summarizer := NewSummarizerWithDefaults(mockClient)

	article := &core.Article{
		ID:          "test-123",
		Title:       "Test Article",
		CleanedText: "Test content",
	}

	_, err := summarizer.SummarizeArticleStructured(context.Background(), article)
	if err == nil {
		t.Fatal("Expected error when LLM fails")
	}

	// Should have retried
	expectedAttempts := summarizer.options.MaxRetries + 1
	if mockClient.callCount != expectedAttempts {
		t.Errorf("Expected %d retry attempts, got %d", expectedAttempts, mockClient.callCount)
	}
}

// Test SummarizeArticleStructured - retry success
func TestSummarizeArticleStructured_RetrySuccess(t *testing.T) {
	mockClient := &MockLLMClientStructured{
		failUntil: 2, // Fail first 2 attempts, succeed on 3rd
	}
	summarizer := NewSummarizerWithDefaults(mockClient)

	article := &core.Article{
		ID:          "test-123",
		Title:       "Test Article",
		CleanedText: "Test content",
	}

	summary, err := summarizer.SummarizeArticleStructured(context.Background(), article)
	if err != nil {
		t.Fatalf("Expected success after retries, got error: %v", err)
	}

	if summary == nil {
		t.Fatal("Summary should not be nil")
	}

	// Should have tried 3 times
	if mockClient.callCount != 3 {
		t.Errorf("Expected 3 attempts (2 failures + 1 success), got %d", mockClient.callCount)
	}
}

// Test SummarizeArticleStructured - invalid JSON response
func TestSummarizeArticleStructured_InvalidJSON(t *testing.T) {
	mockClient := &MockLLMClientStructured{
		response: "This is not valid JSON",
	}
	summarizer := NewSummarizerWithDefaults(mockClient)

	article := &core.Article{
		ID:          "test-123",
		Title:       "Test Article",
		CleanedText: "Test content",
	}

	_, err := summarizer.SummarizeArticleStructured(context.Background(), article)
	if err == nil {
		t.Fatal("Expected error for invalid JSON response")
	}
}

// Test SummarizeArticleStructured - missing required fields
func TestSummarizeArticleStructured_MissingKeyPoints(t *testing.T) {
	// Response with empty key_points
	invalidResponse := core.StructuredSummaryContent{
		KeyPoints:   []string{}, // Empty!
		Context:     "Context text",
		MainInsight: "Insight text",
	}
	jsonBytes, _ := json.Marshal(invalidResponse)

	mockClient := &MockLLMClientStructured{
		response: string(jsonBytes),
	}
	summarizer := NewSummarizerWithDefaults(mockClient)

	article := &core.Article{
		ID:          "test-123",
		Title:       "Test Article",
		CleanedText: "Test content",
	}

	_, err := summarizer.SummarizeArticleStructured(context.Background(), article)
	if err == nil {
		t.Fatal("Expected error for missing key points")
	}
}

// Test RenderStructuredSummary
func TestRenderStructuredSummary(t *testing.T) {
	content := &core.StructuredSummaryContent{
		KeyPoints: []string{
			"First point",
			"Second point",
			"Third point",
		},
		Context:          "Background information",
		MainInsight:      "Core takeaway",
		TechnicalDetails: "Technical aspects",
		Impact:           "Who it affects",
	}

	rendered := RenderStructuredSummary(content)

	if rendered == "" {
		t.Fatal("Rendered text should not be empty")
	}

	// Should contain main insight
	if !contains(rendered, content.MainInsight) {
		t.Error("Rendered text should contain main insight")
	}

	// Should contain context
	if !contains(rendered, content.Context) {
		t.Error("Rendered text should contain context")
	}

	// Should contain all key points
	for _, point := range content.KeyPoints {
		if !contains(rendered, point) {
			t.Errorf("Rendered text should contain key point: %s", point)
		}
	}

	// Should contain technical details
	if !contains(rendered, content.TechnicalDetails) {
		t.Error("Rendered text should contain technical details")
	}

	// Should contain impact
	if !contains(rendered, content.Impact) {
		t.Error("Rendered text should contain impact")
	}
}

// Test RenderStructuredSummaryPlain
func TestRenderStructuredSummaryPlain(t *testing.T) {
	content := &core.StructuredSummaryContent{
		KeyPoints: []string{
			"First point",
			"Second point",
		},
		Context:     "Background",
		MainInsight: "Core idea",
	}

	rendered := RenderStructuredSummaryPlain(content)

	if rendered == "" {
		t.Fatal("Rendered plain text should not be empty")
	}

	// Plain text should not contain markdown
	if contains(rendered, "**") {
		t.Error("Plain text should not contain markdown bold markers")
	}

	// Should still contain content
	if !contains(rendered, content.MainInsight) {
		t.Error("Plain text should contain main insight")
	}

	if !contains(rendered, content.Context) {
		t.Error("Plain text should contain context")
	}
}

// Test RenderStructuredSummary - minimal content
func TestRenderStructuredSummary_Minimal(t *testing.T) {
	content := &core.StructuredSummaryContent{
		KeyPoints:   []string{"Only point"},
		Context:     "Minimal context",
		MainInsight: "Minimal insight",
		// No technical details or impact
	}

	rendered := RenderStructuredSummary(content)

	if rendered == "" {
		t.Fatal("Rendered text should not be empty")
	}

	// Should handle optional fields gracefully
	if !contains(rendered, content.MainInsight) {
		t.Error("Should contain main insight")
	}
}

// Test that structured summaries have proper timestamps
func TestSummarizeArticleStructured_Timestamps(t *testing.T) {
	mockClient := &MockLLMClientStructured{}
	summarizer := NewSummarizerWithDefaults(mockClient)

	article := &core.Article{
		ID:          "test-123",
		Title:       "Test Article",
		CleanedText: "Test content",
	}

	beforeTime := time.Now().UTC()
	summary, err := summarizer.SummarizeArticleStructured(context.Background(), article)
	afterTime := time.Now().UTC()

	if err != nil {
		t.Fatalf("SummarizeArticleStructured failed: %v", err)
	}

	// Check timestamp is within expected range
	if summary.DateGenerated.Before(beforeTime) || summary.DateGenerated.After(afterTime) {
		t.Errorf("DateGenerated %v should be between %v and %v", summary.DateGenerated, beforeTime, afterTime)
	}
}

// Test structured summary with context cancellation
func TestSummarizeArticleStructured_ContextCancellation(t *testing.T) {
	mockClient := &MockLLMClientStructured{
		// Simulate slow response
	}
	summarizer := NewSummarizerWithDefaults(mockClient)

	article := &core.Article{
		ID:          "test-123",
		Title:       "Test Article",
		CleanedText: "Test content",
	}

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := summarizer.SummarizeArticleStructured(ctx, article)
	// The actual behavior depends on implementation
	// For now, just verify it doesn't panic
	_ = err
}

// Helper function for string contains (same as in tracker_test.go)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			len(s) > len(substr)+1 && anySubstring(s[1:len(s)-1], substr)))
}

func anySubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
