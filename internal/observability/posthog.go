package observability

import (
	"briefly/internal/config"
	"context"
	"fmt"
	"log/slog"

	"github.com/posthog/posthog-go"
)

// PostHogClient wraps the PostHog SDK for product analytics
type PostHogClient struct {
	client  posthog.Client
	enabled bool
	log     *slog.Logger
}

// EventProperties contains properties for an event
type EventProperties map[string]interface{}

// UserProperties contains properties for a user
type UserProperties map[string]interface{}

// NewPostHogClient creates a new PostHog analytics client
func NewPostHogClient() (*PostHogClient, error) {
	cfg := config.GetPostHogConfig()

	if !cfg.Enabled {
		return &PostHogClient{
			enabled: false,
			log:     slog.Default(),
		}, nil
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("PostHog enabled but missing API key")
	}

	// Initialize PostHog client
	client, err := posthog.NewWithConfig(cfg.APIKey, posthog.Config{
		Endpoint: cfg.Host,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create PostHog client: %w", err)
	}

	return &PostHogClient{
		client:  client,
		enabled: true,
		log:     slog.Default(),
	}, nil
}

// IsEnabled returns whether PostHog tracking is enabled
func (p *PostHogClient) IsEnabled() bool {
	return p.enabled
}

// Capture sends an event to PostHog
func (p *PostHogClient) Capture(ctx context.Context, distinctID string, event string, properties EventProperties) error {
	if !p.enabled {
		return nil
	}

	return p.client.Enqueue(posthog.Capture{
		DistinctId: distinctID,
		Event:      event,
		Properties: posthog.NewProperties().
			Set("$set", properties),
	})
}

// Identify associates properties with a user
func (p *PostHogClient) Identify(ctx context.Context, distinctID string, properties UserProperties) error {
	if !p.enabled {
		return nil
	}

	props := posthog.NewProperties()
	for k, v := range properties {
		props.Set(k, v)
	}

	return p.client.Enqueue(posthog.Identify{
		DistinctId: distinctID,
		Properties: props,
	})
}

// PageView tracks a page view event
func (p *PostHogClient) PageView(ctx context.Context, distinctID string, path string, properties EventProperties) error {
	if properties == nil {
		properties = make(EventProperties)
	}
	properties["$current_url"] = path

	return p.Capture(ctx, distinctID, "$pageview", properties)
}

// TrackDigestGeneration tracks when a digest is generated
func (p *PostHogClient) TrackDigestGeneration(ctx context.Context, digestID string, articleCount int, themes []string, durationMs int64) error {
	return p.Capture(ctx, "system", "digest_generated", EventProperties{
		"digest_id":     digestID,
		"article_count": articleCount,
		"themes":        themes,
		"duration_ms":   durationMs,
	})
}

// TrackArticleProcessed tracks when an article is processed
func (p *PostHogClient) TrackArticleProcessed(ctx context.Context, articleID string, source string, contentType string, successful bool) error {
	return p.Capture(ctx, "system", "article_processed", EventProperties{
		"article_id":   articleID,
		"source":       source,      // "rss", "manual", "search"
		"content_type": contentType, // "html", "pdf", "youtube"
		"successful":   successful,
	})
}

// TrackThemeClassification tracks theme classification operations
func (p *PostHogClient) TrackThemeClassification(ctx context.Context, articleID string, themeName string, relevanceScore float64) error {
	return p.Capture(ctx, "system", "theme_classified", EventProperties{
		"article_id":      articleID,
		"theme":           themeName,
		"relevance_score": relevanceScore,
	})
}

// TrackManualURLSubmission tracks when a URL is manually submitted
func (p *PostHogClient) TrackManualURLSubmission(ctx context.Context, submittedBy string, urlCount int) error {
	return p.Capture(ctx, submittedBy, "manual_url_submitted", EventProperties{
		"url_count": urlCount,
	})
}

// TrackArticleClick tracks when a user clicks on an article
func (p *PostHogClient) TrackArticleClick(ctx context.Context, userID string, articleID string, articleTitle string, source string) error {
	return p.Capture(ctx, userID, "article_clicked", EventProperties{
		"article_id":    articleID,
		"article_title": articleTitle,
		"source":        source, // "digest", "archive", "search"
	})
}

// TrackThemeFilter tracks when a user filters by theme
func (p *PostHogClient) TrackThemeFilter(ctx context.Context, userID string, themeName string) error {
	return p.Capture(ctx, userID, "theme_filter_applied", EventProperties{
		"theme": themeName,
	})
}

// TrackDigestView tracks when a user views a digest
func (p *PostHogClient) TrackDigestView(ctx context.Context, userID string, digestID string, digestDate string) error {
	return p.Capture(ctx, userID, "digest_viewed", EventProperties{
		"digest_id":   digestID,
		"digest_date": digestDate,
	})
}

// TrackError tracks when an error occurs
func (p *PostHogClient) TrackError(ctx context.Context, errorType string, errorMessage string, component string) error {
	return p.Capture(ctx, "system", "error_occurred", EventProperties{
		"error_type":    errorType,
		"error_message": errorMessage,
		"component":     component,
	})
}

// TrackLLMCall tracks LLM API calls for cost and performance monitoring
func (p *PostHogClient) TrackLLMCall(ctx context.Context, model string, operation string, tokens int, latencyMs int64, cost float64) error {
	return p.Capture(ctx, "system", "llm_call", EventProperties{
		"model":      model,
		"operation":  operation, // "summarization", "classification", "embedding"
		"tokens":     tokens,
		"latency_ms": latencyMs,
		"cost":       cost,
	})
}

// Flush ensures all pending events are sent to PostHog
func (p *PostHogClient) Flush() error {
	if !p.enabled {
		return nil
	}

	return p.client.Close()
}

// Shutdown gracefully shuts down the PostHog client
func (p *PostHogClient) Shutdown(ctx context.Context) error {
	if !p.enabled {
		return nil
	}

	return p.client.Close()
}
