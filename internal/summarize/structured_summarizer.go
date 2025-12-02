package summarize

import (
	"briefly/internal/core"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"google.golang.org/genai"
	"github.com/google/uuid"
)

// CreateStructuredSummarySchema returns the Gemini response_schema for structured summaries (Phase 1)
// This schema enforces structured JSON output from the LLM
func CreateStructuredSummarySchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"key_points": {
				Type:        genai.TypeArray,
				Description: "3-5 key bullet points that capture the essential information",
				Items: &genai.Schema{
					Type: genai.TypeString,
				},
			},
			"context": {
				Type:        genai.TypeString,
				Description: "Background information explaining why this article matters (2-3 sentences)",
			},
			"main_insight": {
				Type:        genai.TypeString,
				Description: "The core takeaway or most important finding (1-2 sentences)",
			},
			"technical_details": {
				Type:        genai.TypeString,
				Description: "Technical aspects, methodologies, or specific details for those who want deeper understanding (optional, can be empty if not applicable)",
			},
			"impact": {
				Type:        genai.TypeString,
				Description: "Who this affects and how - practical implications (optional, can be empty if not applicable)",
			},
		},
		Required: []string{"key_points", "context", "main_insight"},
	}
}

// BuildStructuredSummaryPrompt creates a prompt optimized for generating structured summaries
func BuildStructuredSummaryPrompt(title, content string) string {
	return fmt.Sprintf(`Analyze the following article and create a structured summary.

Article Title: %s

Article Content:
%s

Create a comprehensive structured summary with the following sections:

1. KEY POINTS: Extract 3-5 key bullet points that capture the essential information
2. CONTEXT: Explain the background and why this article matters (2-3 sentences)
3. MAIN INSIGHT: Identify the core takeaway or most important finding (1-2 sentences)
4. TECHNICAL DETAILS: Include technical aspects, methodologies, or specific details (if applicable)
5. IMPACT: Describe who this affects and how - practical implications (if applicable)

Focus on clarity, accuracy, and providing value to readers who want to quickly understand the article's significance.`, title, content)
}

// SummarizeArticleStructured creates a structured summary using Gemini's response_schema (Phase 1)
// Returns a Summary with both structured content and rendered plain text
func (s *Summarizer) SummarizeArticleStructured(ctx context.Context, article *core.Article) (*core.Summary, error) {
	if article == nil {
		return nil, fmt.Errorf("article is nil")
	}

	if article.CleanedText == "" {
		return nil, fmt.Errorf("article has no content to summarize")
	}

	// Build structured summary prompt
	prompt := BuildStructuredSummaryPrompt(article.Title, article.CleanedText)

	// Create response schema
	schema := CreateStructuredSummarySchema()

	// Generate structured summary with retries
	var response string
	var err error

	// Generate with response schema
	for attempt := 0; attempt <= s.options.MaxRetries; attempt++ {
		// Note: The options parameter is interface{} in the LLMClient interface
		// The actual implementation will handle the type conversion
		options := struct {
			ResponseSchema *genai.Schema
			Temperature    float32
		}{
			ResponseSchema: schema,
			Temperature:    s.options.Temperature,
		}

		response, err = s.llmClient.GenerateText(ctx, prompt, options)
		if err == nil {
			break
		}

		if attempt < s.options.MaxRetries {
			time.Sleep(s.options.RetryDelay * time.Duration(attempt+1))
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate structured summary after %d attempts: %w", s.options.MaxRetries+1, err)
	}

	// Parse JSON response into StructuredSummaryContent
	var structuredContent core.StructuredSummaryContent
	if err := json.Unmarshal([]byte(response), &structuredContent); err != nil {
		return nil, fmt.Errorf("failed to parse structured summary JSON: %w", err)
	}

	// Validate structured content
	if len(structuredContent.KeyPoints) == 0 {
		return nil, fmt.Errorf("structured summary has no key points")
	}
	if structuredContent.Context == "" {
		return nil, fmt.Errorf("structured summary has no context")
	}
	if structuredContent.MainInsight == "" {
		return nil, fmt.Errorf("structured summary has no main insight")
	}

	// Render structured content to plain text for backward compatibility
	plainText := RenderStructuredSummary(&structuredContent)

	// Create summary object
	summary := &core.Summary{
		ID:            uuid.NewString(),
		ArticleIDs:    []string{article.ID},
		SummaryText:   plainText,
		ModelUsed:     s.options.ModelName,
		DateGenerated: time.Now().UTC(),

		// Phase 1: Structured summary fields
		SummaryType:       "structured",
		StructuredContent: &structuredContent,
	}

	return summary, nil
}

// RenderStructuredSummary converts a StructuredSummaryContent to plain text
// This provides backward compatibility and a readable format
func RenderStructuredSummary(content *core.StructuredSummaryContent) string {
	var parts []string

	// Main insight (headline)
	if content.MainInsight != "" {
		parts = append(parts, fmt.Sprintf("**%s**\n", content.MainInsight))
	}

	// Context
	if content.Context != "" {
		parts = append(parts, fmt.Sprintf("%s\n", content.Context))
	}

	// Key points
	if len(content.KeyPoints) > 0 {
		parts = append(parts, "**Key Points:**")
		for _, point := range content.KeyPoints {
			parts = append(parts, fmt.Sprintf("• %s", point))
		}
		parts = append(parts, "")
	}

	// Technical details (if present)
	if content.TechnicalDetails != "" {
		parts = append(parts, fmt.Sprintf("**Technical Details:** %s\n", content.TechnicalDetails))
	}

	// Impact (if present)
	if content.Impact != "" {
		parts = append(parts, fmt.Sprintf("**Impact:** %s", content.Impact))
	}

	return strings.Join(parts, "\n")
}

// RenderStructuredSummaryPlain converts to plain text without markdown
// Useful for contexts where markdown isn't supported
func RenderStructuredSummaryPlain(content *core.StructuredSummaryContent) string {
	var parts []string

	// Main insight
	if content.MainInsight != "" {
		parts = append(parts, content.MainInsight)
		parts = append(parts, "")
	}

	// Context
	if content.Context != "" {
		parts = append(parts, content.Context)
		parts = append(parts, "")
	}

	// Key points
	if len(content.KeyPoints) > 0 {
		parts = append(parts, "Key Points:")
		for _, point := range content.KeyPoints {
			parts = append(parts, fmt.Sprintf("- %s", point))
		}
		parts = append(parts, "")
	}

	// Technical details
	if content.TechnicalDetails != "" {
		parts = append(parts, fmt.Sprintf("Technical Details: %s", content.TechnicalDetails))
		parts = append(parts, "")
	}

	// Impact
	if content.Impact != "" {
		parts = append(parts, fmt.Sprintf("Impact: %s", content.Impact))
	}

	return strings.Join(parts, "\n")
}
