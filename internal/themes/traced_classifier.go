package themes

import (
	"briefly/internal/core"
	"briefly/internal/observability"
	"context"
	"fmt"
	"time"
)

// TracedClassifier wraps a Classifier with LangFuse observability (Phase 1)
// Provides classification-level tracking for theme assignment
type TracedClassifier struct {
	classifier *Classifier
	langfuse   *observability.LangFuseClient
}

// NewTracedClassifier creates a classifier with LangFuse tracking
func NewTracedClassifier(classifier *Classifier, langfuse *observability.LangFuseClient) *TracedClassifier {
	return &TracedClassifier{
		classifier: classifier,
		langfuse:   langfuse,
	}
}

// GetBestMatch wraps the classifier with LangFuse tracking
func (t *TracedClassifier) GetBestMatch(ctx context.Context, article core.Article, themes []core.Theme, minRelevance float64) (*ClassificationResult, error) {
	if t.langfuse == nil || !t.langfuse.IsEnabled() {
		return t.classifier.GetBestMatch(ctx, article, themes, minRelevance)
	}

	// Create trace for the classification
	trace, err := t.langfuse.CreateTrace(ctx, observability.TraceOptions{
		Name: "theme_classification",
		Tags: []string{"classification", "theme", "phase1"},
		Metadata: map[string]string{
			"article_id":     article.ID,
			"article_title":  article.Title,
			"article_url":    article.URL,
			"content_type":   string(article.ContentType),
			"content_length": fmt.Sprintf("%d", len(article.CleanedText)),
			"theme_count":    fmt.Sprintf("%d", len(themes)),
			"min_relevance":  fmt.Sprintf("%.2f", minRelevance),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create trace: %w", err)
	}

	startTime := time.Now()
	result, classifyErr := t.classifier.GetBestMatch(ctx, article, themes, minRelevance)
	latency := time.Since(startTime).Milliseconds()

	// Track generation metrics
	if classifyErr == nil {
		var completionText string
		if result != nil {
			completionText = fmt.Sprintf("Matched theme: %s (ID: %s, Relevance: %.3f)\nReasoning: %s",
				result.ThemeName, result.ThemeID, result.RelevanceScore, result.Reasoning)
		} else {
			completionText = "No theme matched (below relevance threshold)"
		}

		_ = t.langfuse.TrackGeneration(trace, observability.GenerationOptions{
			Model:      "gemini-3-flash-preview",
			Prompt:     fmt.Sprintf("Classify article: %s", article.Title),
			Completion: completionText,
			LatencyMs:  latency,
		})
	}

	return result, classifyErr
}

// ClassifyArticle wraps multi-theme classification with tracking
func (t *TracedClassifier) ClassifyArticle(ctx context.Context, article core.Article, themes []core.Theme, minRelevance float64) ([]ClassificationResult, error) {
	if t.langfuse == nil || !t.langfuse.IsEnabled() {
		return t.classifier.ClassifyArticle(ctx, article, themes, minRelevance)
	}

	// Create trace for multi-theme classification
	trace, err := t.langfuse.CreateTrace(ctx, observability.TraceOptions{
		Name: "theme_classification_multi",
		Tags: []string{"classification", "theme", "multi", "phase1"},
		Metadata: map[string]string{
			"article_id":    article.ID,
			"article_title": article.Title,
			"theme_count":   fmt.Sprintf("%d", len(themes)),
			"min_relevance": fmt.Sprintf("%.2f", minRelevance),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create trace: %w", err)
	}

	startTime := time.Now()
	results, classifyErr := t.classifier.ClassifyArticle(ctx, article, themes, minRelevance)
	latency := time.Since(startTime).Milliseconds()

	// Track generation with results summary
	if classifyErr == nil {
		matchedThemes := make([]string, 0, len(results))
		for _, result := range results {
			matchedThemes = append(matchedThemes, fmt.Sprintf("%s (%.2f)", result.ThemeName, result.RelevanceScore))
		}

		completionText := fmt.Sprintf("Matched %d themes: %v", len(results), matchedThemes)

		_ = t.langfuse.TrackGeneration(trace, observability.GenerationOptions{
			Model:      "gemini-3-flash-preview",
			Prompt:     fmt.Sprintf("Classify article: %s", article.Title),
			Completion: completionText,
			LatencyMs:  latency,
		})

		// Track individual theme matches as spans
		for i, result := range results {
			span := t.langfuse.CreateSpan(trace, observability.SpanOptions{
				Name: fmt.Sprintf("theme_match_%d", i+1),
				Metadata: map[string]string{
					"theme_id":        result.ThemeID,
					"theme_name":      result.ThemeName,
					"relevance_score": fmt.Sprintf("%.3f", result.RelevanceScore),
					"reasoning":       result.Reasoning,
				},
			})
			t.langfuse.EndSpan(span, map[string]interface{}{
				"theme":     result.ThemeName,
				"relevance": result.RelevanceScore,
			}, nil)
		}
	}

	return results, classifyErr
}

// ClassifyBatch wraps batch classification with tracking
func (t *TracedClassifier) ClassifyBatch(ctx context.Context, articles []core.Article, themes []core.Theme, minRelevance float64) (map[string][]ClassificationResult, error) {
	if t.langfuse == nil || !t.langfuse.IsEnabled() {
		return t.classifier.ClassifyBatch(ctx, articles, themes, minRelevance)
	}

	// Create trace for batch classification
	trace, err := t.langfuse.CreateTrace(ctx, observability.TraceOptions{
		Name: "theme_classification_batch",
		Tags: []string{"classification", "theme", "batch", "phase1"},
		Metadata: map[string]string{
			"article_count": fmt.Sprintf("%d", len(articles)),
			"theme_count":   fmt.Sprintf("%d", len(themes)),
			"min_relevance": fmt.Sprintf("%.2f", minRelevance),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create trace: %w", err)
	}

	startTime := time.Now()
	results, classifyErr := t.classifier.ClassifyBatch(ctx, articles, themes, minRelevance)
	latency := time.Since(startTime).Milliseconds()

	// Track batch generation metrics
	if classifyErr == nil {
		totalMatches := 0
		for _, articleResults := range results {
			totalMatches += len(articleResults)
		}

		avgLatency := float64(latency) / float64(len(articles))
		completionText := fmt.Sprintf("Classified %d articles, %d total theme matches, avg latency: %.0fms per article",
			len(results), totalMatches, avgLatency)

		_ = t.langfuse.TrackGeneration(trace, observability.GenerationOptions{
			Model:      "gemini-3-flash-preview",
			Prompt:     fmt.Sprintf("Classify %d articles", len(articles)),
			Completion: completionText,
			LatencyMs:  latency,
		})

		// Track per-article classification as spans
		for articleID, articleResults := range results {
			span := t.langfuse.CreateSpan(trace, observability.SpanOptions{
				Name: "article_classification",
				Metadata: map[string]string{
					"article_id":  articleID,
					"match_count": fmt.Sprintf("%d", len(articleResults)),
				},
			})

			matchInfo := make(map[string]interface{})
			for i, result := range articleResults {
				matchInfo[fmt.Sprintf("theme_%d", i+1)] = map[string]interface{}{
					"name":      result.ThemeName,
					"relevance": result.RelevanceScore,
				}
			}

			t.langfuse.EndSpan(span, matchInfo, nil)
		}
	}

	return results, classifyErr
}
