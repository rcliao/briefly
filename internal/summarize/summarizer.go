package summarize

import (
	"briefly/internal/core"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// LLMClient defines the interface for LLM operations
type LLMClient interface {
	// GenerateText generates text from a prompt
	GenerateText(ctx context.Context, prompt string, options interface{}) (string, error)
}

// SummarizerInterface defines the interface for article summarization
// Both Summarizer and TracedSummarizer implement this interface
type SummarizerInterface interface {
	SummarizeArticle(ctx context.Context, article *core.Article) (*core.Summary, error)
	SummarizeArticleStructured(ctx context.Context, article *core.Article) (*core.Summary, error)
	GenerateKeyPoints(ctx context.Context, content string) ([]string, error)
	ExtractTitle(ctx context.Context, content string) (string, error)
}

// Summarizer handles article summarization using LLM
type Summarizer struct {
	llmClient LLMClient
	options   SummarizerOptions
}

// SummarizerOptions configures the summarizer behavior
type SummarizerOptions struct {
	// Default settings for summaries
	DefaultMaxWords      int
	DefaultKeyPointCount int

	// Model settings
	ModelName   string
	Temperature float32

	// Retry settings
	MaxRetries int
	RetryDelay time.Duration

	// Quality control
	MinSummaryWords int // Minimum words for valid summary
	MaxSummaryWords int // Maximum words before truncation
}

// DefaultSummarizerOptions returns sensible defaults
func DefaultSummarizerOptions() SummarizerOptions {
	return SummarizerOptions{
		DefaultMaxWords:      150,
		DefaultKeyPointCount: 5,
		ModelName:            "gemini-flash-lite-latest",
		Temperature:          0.3, // Lower temperature for more consistent summaries
		MaxRetries:           2,
		RetryDelay:           time.Second,
		MinSummaryWords:      50,
		MaxSummaryWords:      300,
	}
}

// NewSummarizer creates a new summarizer with the given LLM client
func NewSummarizer(llmClient LLMClient, options SummarizerOptions) *Summarizer {
	return &Summarizer{
		llmClient: llmClient,
		options:   options,
	}
}

// NewSummarizerWithDefaults creates a summarizer with default options
func NewSummarizerWithDefaults(llmClient LLMClient) *Summarizer {
	return NewSummarizer(llmClient, DefaultSummarizerOptions())
}

// SummarizeArticle creates a comprehensive summary of an article
func (s *Summarizer) SummarizeArticle(ctx context.Context, article *core.Article) (*core.Summary, error) {
	if article == nil {
		return nil, fmt.Errorf("article is nil")
	}

	if article.CleanedText == "" {
		return nil, fmt.Errorf("article has no content to summarize")
	}

	// Build prompt
	promptOpts := DefaultDigestOptions()
	promptOpts.MaxWords = s.options.DefaultMaxWords
	promptOpts.KeyPointCount = s.options.DefaultKeyPointCount

	prompt := BuildSummarizationPrompt(article.Title, article.CleanedText, promptOpts)

	// Generate summary with retries
	var response string
	var err error

	for attempt := 0; attempt <= s.options.MaxRetries; attempt++ {
		response, err = s.llmClient.GenerateText(ctx, prompt, nil)
		if err == nil {
			break
		}

		if attempt < s.options.MaxRetries {
			time.Sleep(s.options.RetryDelay * time.Duration(attempt+1))
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate summary after %d attempts: %w", s.options.MaxRetries+1, err)
	}

	// Parse response
	summaryText, keyPoints := ParseSummaryResponse(response)

	// Validate summary
	if err := s.validateSummary(summaryText); err != nil {
		// Try to extract first N words as fallback
		summaryText = s.extractFallbackSummary(article.CleanedText)
		keyPoints = []string{} // Clear key points for fallback
	}

	// Build summary object
	summary := &core.Summary{
		ID:            uuid.NewString(),
		ArticleIDs:    []string{article.ID},
		SummaryText:   summaryText,
		ModelUsed:     s.options.ModelName,
		DateGenerated: time.Now(),
	}

	// Store key points if we got them
	// Key points would go into a KeyPoints field if we add it to Summary struct
	// For now, we can append them to the summary text in a structured way
	// Or wait until we update the core.Summary struct
	_ = keyPoints // Explicitly acknowledge we're not using key points yet

	return summary, nil
}

// GenerateKeyPoints extracts key points from content
func (s *Summarizer) GenerateKeyPoints(ctx context.Context, content string) ([]string, error) {
	if content == "" {
		return nil, fmt.Errorf("content is empty")
	}

	count := s.options.DefaultKeyPointCount
	prompt := BuildKeyPointsPrompt(content, count)

	response, err := s.llmClient.GenerateText(ctx, prompt, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key points: %w", err)
	}

	// Parse bullet points from response
	keyPoints := s.parseKeyPointsFromResponse(response)

	if len(keyPoints) == 0 {
		return nil, fmt.Errorf("no key points extracted from response")
	}

	return keyPoints, nil
}

// ExtractTitle generates or extracts a title from content
func (s *Summarizer) ExtractTitle(ctx context.Context, content string) (string, error) {
	if content == "" {
		return "", fmt.Errorf("content is empty")
	}

	prompt := BuildTitlePrompt(content)

	title, err := s.llmClient.GenerateText(ctx, prompt, nil)
	if err != nil {
		return "", fmt.Errorf("failed to extract title: %w", err)
	}

	title = strings.TrimSpace(title)
	title = strings.Trim(title, `"'`)

	// Validate title length
	if len(strings.Fields(title)) > 15 {
		// Truncate overly long titles
		words := strings.Fields(title)
		title = strings.Join(words[:15], " ") + "..."
	}

	return title, nil
}

// IdentifyTheme identifies the main theme or category of an article
func (s *Summarizer) IdentifyTheme(ctx context.Context, article *core.Article, summary *core.Summary) (string, error) {
	var summaryText string
	if summary != nil {
		summaryText = summary.SummaryText
	} else if len(article.CleanedText) > 500 {
		summaryText = article.CleanedText[:500]
	} else {
		summaryText = article.CleanedText
	}

	prompt := BuildThemePrompt(article.Title, summaryText)

	theme, err := s.llmClient.GenerateText(ctx, prompt, nil)
	if err != nil {
		return "General", nil // Return default theme on error
	}

	theme = strings.TrimSpace(theme)
	theme = strings.Trim(theme, `"'`)

	// Validate theme
	if theme == "" || len(theme) > 50 {
		return "General", nil
	}

	return theme, nil
}

// SummarizeBatch processes multiple articles in batch
// This can be optimized later for concurrent processing
func (s *Summarizer) SummarizeBatch(ctx context.Context, articles []*core.Article) ([]*core.Summary, error) {
	summaries := make([]*core.Summary, 0, len(articles))
	errors := make([]error, 0)

	for _, article := range articles {
		summary, err := s.SummarizeArticle(ctx, article)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to summarize article %s: %w", article.ID, err))
			continue
		}

		summaries = append(summaries, summary)
	}

	// If we have partial success, return what we got
	if len(summaries) > 0 {
		return summaries, nil
	}

	// If everything failed, return first error
	if len(errors) > 0 {
		return nil, errors[0]
	}

	return summaries, nil
}

// RefineSum mary refines an existing summary based on feedback
func (s *Summarizer) RefineSummary(ctx context.Context, originalSummary string, feedback string, targetWords int) (string, error) {
	prompt := BuildRefinePrompt(originalSummary, feedback, targetWords)

	refinedSummary, err := s.llmClient.GenerateText(ctx, prompt, nil)
	if err != nil {
		return "", fmt.Errorf("failed to refine summary: %w", err)
	}

	return strings.TrimSpace(refinedSummary), nil
}

// SimplifyForAudience simplifies technical content for a specific audience
func (s *Summarizer) SimplifyForAudience(ctx context.Context, content string, audience string) (string, error) {
	prompt := BuildSimplificationPrompt(content, audience)

	simplified, err := s.llmClient.GenerateText(ctx, prompt, nil)
	if err != nil {
		return "", fmt.Errorf("failed to simplify content: %w", err)
	}

	return strings.TrimSpace(simplified), nil
}

// validateSummary checks if a summary meets minimum quality requirements
func (s *Summarizer) validateSummary(summary string) error {
	if summary == "" {
		return fmt.Errorf("summary is empty")
	}

	wordCount := len(strings.Fields(summary))

	if wordCount < s.options.MinSummaryWords {
		return fmt.Errorf("summary too short: %d words (minimum: %d)", wordCount, s.options.MinSummaryWords)
	}

	if wordCount > s.options.MaxSummaryWords {
		return fmt.Errorf("summary too long: %d words (maximum: %d)", wordCount, s.options.MaxSummaryWords)
	}

	return nil
}

// extractFallbackSummary creates a simple fallback summary from content
func (s *Summarizer) extractFallbackSummary(content string) string {
	// Take first few sentences up to target word count
	sentences := strings.Split(content, ". ")

	var summary strings.Builder
	wordCount := 0
	targetWords := s.options.DefaultMaxWords

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		words := strings.Fields(sentence)
		if wordCount+len(words) > targetWords && wordCount > 0 {
			break
		}

		if summary.Len() > 0 {
			summary.WriteString(". ")
		}
		summary.WriteString(sentence)
		wordCount += len(words)

		if wordCount >= targetWords {
			break
		}
	}

	result := summary.String()
	if !strings.HasSuffix(result, ".") {
		result += "."
	}

	return result
}

// parseKeyPointsFromResponse extracts bullet points from LLM response
func (s *Summarizer) parseKeyPointsFromResponse(response string) []string {
	var keyPoints []string
	lines := strings.Split(response, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for bullet points
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "•") || strings.HasPrefix(line, "*") {
			point := strings.TrimSpace(line[1:])
			if point != "" {
				keyPoints = append(keyPoints, point)
			}
		} else if len(line) > 2 && line[0] >= '1' && line[0] <= '9' && (line[1] == '.' || line[1] == ')') {
			// Numbered list
			point := strings.TrimSpace(line[2:])
			if point != "" {
				keyPoints = append(keyPoints, point)
			}
		}
	}

	return keyPoints
}

// GetStats returns summarizer statistics
func (s *Summarizer) GetStats() SummarizerStats {
	return SummarizerStats{
		ModelName:            s.options.ModelName,
		DefaultMaxWords:      s.options.DefaultMaxWords,
		DefaultKeyPointCount: s.options.DefaultKeyPointCount,
	}
}

// SummarizerStats holds summarizer statistics
type SummarizerStats struct {
	ModelName            string
	DefaultMaxWords      int
	DefaultKeyPointCount int
}
