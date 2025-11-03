package summarize

import (
	"briefly/internal/core"
	"briefly/internal/observability"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// TracedSummarizer wraps a Summarizer with LangFuse observability (Phase 1)
// Provides section-level tracking for structured summaries
type TracedSummarizer struct {
	summarizer *Summarizer
	langfuse   *observability.LangFuseClient
}

// NewTracedSummarizer creates a summarizer with LangFuse tracking
func NewTracedSummarizer(summarizer *Summarizer, langfuse *observability.LangFuseClient) *TracedSummarizer {
	return &TracedSummarizer{
		summarizer: summarizer,
		langfuse:   langfuse,
	}
}

// SummarizeArticle wraps the standard summarizer with tracking
func (t *TracedSummarizer) SummarizeArticle(ctx context.Context, article *core.Article) (*core.Summary, error) {
	if t.langfuse == nil || !t.langfuse.IsEnabled() {
		return t.summarizer.SummarizeArticle(ctx, article)
	}

	// Create trace for the summarization
	trace, err := t.langfuse.CreateTrace(ctx, observability.TraceOptions{
		Name: "article_summarization_simple",
		Tags: []string{"summarization", "simple"},
		Metadata: map[string]string{
			"article_id":    article.ID,
			"article_title": article.Title,
			"content_type":  string(article.ContentType),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create trace: %w", err)
	}

	startTime := time.Now()
	summary, summaryErr := t.summarizer.SummarizeArticle(ctx, article)
	latency := time.Since(startTime).Milliseconds()

	// Track generation metrics
	if summaryErr == nil && summary != nil {
		_ = t.langfuse.TrackGeneration(trace, observability.GenerationOptions{
			Model:      summary.ModelUsed,
			Prompt:     fmt.Sprintf("Summarize: %s", article.Title),
			Completion: summary.SummaryText,
			LatencyMs:  latency,
			// Token counts would be tracked if available from LLM client
		})
	}

	return summary, summaryErr
}

// SummarizeArticleStructured wraps structured summarization with section-level tracking
func (t *TracedSummarizer) SummarizeArticleStructured(ctx context.Context, article *core.Article) (*core.Summary, error) {
	if t.langfuse == nil || !t.langfuse.IsEnabled() {
		return t.summarizer.SummarizeArticleStructured(ctx, article)
	}

	// Create trace for structured summarization
	trace, err := t.langfuse.CreateTrace(ctx, observability.TraceOptions{
		Name: "article_summarization_structured",
		Tags: []string{"summarization", "structured", "phase1"},
		Metadata: map[string]string{
			"article_id":     article.ID,
			"article_title":  article.Title,
			"content_type":   string(article.ContentType),
			"content_length": fmt.Sprintf("%d", len(article.CleanedText)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create trace: %w", err)
	}

	// Track overall generation
	overallStart := time.Now()
	summary, summaryErr := t.summarizer.SummarizeArticleStructured(ctx, article)
	overallLatency := time.Since(overallStart).Milliseconds()

	if summaryErr != nil {
		// Track failed generation
		span := t.langfuse.CreateSpan(trace, observability.SpanOptions{
			Name: "structured_summary_generation",
		})
		t.langfuse.EndSpan(span, nil, summaryErr)
		return nil, summaryErr
	}

	// Track successful generation with overall metrics
	_ = t.langfuse.TrackGeneration(trace, observability.GenerationOptions{
		Model:      summary.ModelUsed,
		Prompt:     BuildStructuredSummaryPrompt(article.Title, article.CleanedText),
		Completion: summary.SummaryText,
		LatencyMs:  overallLatency,
	})

	// Track individual sections (Phase 1: Section-level observability)
	if summary.StructuredContent != nil {
		t.trackStructuredSections(trace, summary.StructuredContent)
	}

	return summary, nil
}

// trackStructuredSections tracks metrics for each section of a structured summary
func (t *TracedSummarizer) trackStructuredSections(trace *observability.TraceClient, content *core.StructuredSummaryContent) {
	if trace == nil || content == nil {
		return
	}

	// Track key points section
	keyPointsSpan := t.langfuse.CreateSpan(trace, observability.SpanOptions{
		Name: "section_key_points",
		Metadata: map[string]string{
			"count":          fmt.Sprintf("%d", len(content.KeyPoints)),
			"total_length":   fmt.Sprintf("%d", totalLength(content.KeyPoints)),
			"avg_length":     fmt.Sprintf("%.1f", avgLength(content.KeyPoints)),
			"section_type":   "required",
		},
	})
	t.langfuse.EndSpan(keyPointsSpan, map[string]interface{}{
		"key_points": content.KeyPoints,
		"count":      len(content.KeyPoints),
	}, nil)

	// Track context section
	contextSpan := t.langfuse.CreateSpan(trace, observability.SpanOptions{
		Name: "section_context",
		Metadata: map[string]string{
			"length":       fmt.Sprintf("%d", len(content.Context)),
			"word_count":   fmt.Sprintf("%d", wordCount(content.Context)),
			"section_type": "required",
		},
	})
	t.langfuse.EndSpan(contextSpan, map[string]interface{}{
		"context": content.Context,
		"length":  len(content.Context),
	}, nil)

	// Track main insight section
	insightSpan := t.langfuse.CreateSpan(trace, observability.SpanOptions{
		Name: "section_main_insight",
		Metadata: map[string]string{
			"length":       fmt.Sprintf("%d", len(content.MainInsight)),
			"word_count":   fmt.Sprintf("%d", wordCount(content.MainInsight)),
			"section_type": "required",
		},
	})
	t.langfuse.EndSpan(insightSpan, map[string]interface{}{
		"main_insight": content.MainInsight,
		"length":       len(content.MainInsight),
	}, nil)

	// Track technical details section (optional)
	if content.TechnicalDetails != "" {
		techSpan := t.langfuse.CreateSpan(trace, observability.SpanOptions{
			Name: "section_technical_details",
			Metadata: map[string]string{
				"length":       fmt.Sprintf("%d", len(content.TechnicalDetails)),
				"word_count":   fmt.Sprintf("%d", wordCount(content.TechnicalDetails)),
				"section_type": "optional",
				"present":      "true",
			},
		})
		t.langfuse.EndSpan(techSpan, map[string]interface{}{
			"technical_details": content.TechnicalDetails,
			"length":            len(content.TechnicalDetails),
		}, nil)
	}

	// Track impact section (optional)
	if content.Impact != "" {
		impactSpan := t.langfuse.CreateSpan(trace, observability.SpanOptions{
			Name: "section_impact",
			Metadata: map[string]string{
				"length":       fmt.Sprintf("%d", len(content.Impact)),
				"word_count":   fmt.Sprintf("%d", wordCount(content.Impact)),
				"section_type": "optional",
				"present":      "true",
			},
		})
		t.langfuse.EndSpan(impactSpan, map[string]interface{}{
			"impact": content.Impact,
			"length": len(content.Impact),
		}, nil)
	}

	// Track overall structure quality metrics
	qualitySpan := t.langfuse.CreateSpan(trace, observability.SpanOptions{
		Name: "structure_quality_metrics",
		Metadata: map[string]string{
			"key_points_count":      fmt.Sprintf("%d", len(content.KeyPoints)),
			"optional_sections":     fmt.Sprintf("%d", optionalSectionsCount(content)),
			"completeness_score":    fmt.Sprintf("%.2f", completenessScore(content)),
			"total_content_length":  fmt.Sprintf("%d", totalContentLength(content)),
		},
	})
	t.langfuse.EndSpan(qualitySpan, map[string]interface{}{
		"quality_metrics": map[string]interface{}{
			"key_points_count":    len(content.KeyPoints),
			"optional_sections":   optionalSectionsCount(content),
			"completeness_score":  completenessScore(content),
			"total_length":        totalContentLength(content),
		},
	}, nil)
}

// GenerateKeyPoints wraps key point generation with tracking
func (t *TracedSummarizer) GenerateKeyPoints(ctx context.Context, content string) ([]string, error) {
	if t.langfuse == nil || !t.langfuse.IsEnabled() {
		return t.summarizer.GenerateKeyPoints(ctx, content)
	}

	var points []string
	err := t.langfuse.SimpleTrace(ctx, "generate_key_points", func(trace *observability.TraceClient) error {
		span := t.langfuse.CreateSpan(trace, observability.SpanOptions{
			Name: "key_points_extraction",
			Metadata: map[string]string{
				"content_length": fmt.Sprintf("%d", len(content)),
			},
		})

		startTime := time.Now()
		var pointsErr error
		points, pointsErr = t.summarizer.GenerateKeyPoints(ctx, content)
		latency := time.Since(startTime).Milliseconds()

		if pointsErr != nil {
			t.langfuse.EndSpan(span, nil, pointsErr)
			return pointsErr
		}

		t.langfuse.EndSpan(span, map[string]interface{}{
			"key_points": points,
			"count":      len(points),
			"latency_ms": latency,
		}, nil)

		return nil
	})

	return points, err
}

// ExtractTitle wraps title extraction with tracking
func (t *TracedSummarizer) ExtractTitle(ctx context.Context, content string) (string, error) {
	if t.langfuse == nil || !t.langfuse.IsEnabled() {
		return t.summarizer.ExtractTitle(ctx, content)
	}

	var title string
	err := t.langfuse.SimpleTrace(ctx, "extract_title", func(trace *observability.TraceClient) error {
		var extractErr error
		title, extractErr = t.summarizer.ExtractTitle(ctx, content)
		return extractErr
	})

	return title, err
}

// Helper functions for metrics calculation

func totalLength(items []string) int {
	total := 0
	for _, item := range items {
		total += len(item)
	}
	return total
}

func avgLength(items []string) float64 {
	if len(items) == 0 {
		return 0
	}
	return float64(totalLength(items)) / float64(len(items))
}

func wordCount(text string) int {
	if text == "" {
		return 0
	}
	// Simple word count (split by spaces)
	count := 1
	for _, char := range text {
		if char == ' ' {
			count++
		}
	}
	return count
}

func optionalSectionsCount(content *core.StructuredSummaryContent) int {
	count := 0
	if content.TechnicalDetails != "" {
		count++
	}
	if content.Impact != "" {
		count++
	}
	return count
}

func completenessScore(content *core.StructuredSummaryContent) float64 {
	// Score based on presence and quality of sections
	score := 0.0

	// Required sections (3 * 25% = 75%)
	if len(content.KeyPoints) >= 3 {
		score += 0.25
	}
	if len(content.Context) > 50 {
		score += 0.25
	}
	if len(content.MainInsight) > 20 {
		score += 0.25
	}

	// Optional sections (2 * 12.5% = 25%)
	if content.TechnicalDetails != "" {
		score += 0.125
	}
	if content.Impact != "" {
		score += 0.125
	}

	return score
}

func totalContentLength(content *core.StructuredSummaryContent) int {
	total := 0
	for _, kp := range content.KeyPoints {
		total += len(kp)
	}
	total += len(content.Context)
	total += len(content.MainInsight)
	total += len(content.TechnicalDetails)
	total += len(content.Impact)
	return total
}

// MarshalStructuredContent converts structured content to JSON for logging
func MarshalStructuredContent(content *core.StructuredSummaryContent) string {
	if content == nil {
		return "{}"
	}
	bytes, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(bytes)
}
