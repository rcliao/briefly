// Package observability provides observability and analytics integrations (Phase 0)
package observability

import (
	"briefly/internal/config"
	"context"
	"fmt"
	"log/slog"
	"time"
)

// LangFuseClient wraps the LangFuse SDK for LLM observability
// Note: This is a simplified implementation that logs traces locally
// Full SDK integration can be added later when langfuse-go is stable
type LangFuseClient struct {
	enabled bool
	log     *slog.Logger
	config  config.LangFuseConfig
}

// TraceClient represents a trace for tracking operations
type TraceClient struct {
	id   string
	name string
	log  *slog.Logger
}

// SpanClient represents a span within a trace
type SpanClient struct {
	id   string
	name string
	log  *slog.Logger
}

// TraceOptions contains options for creating traces
type TraceOptions struct {
	Name      string            // Trace name (e.g., "article_summarization")
	UserID    string            // Optional user identifier
	SessionID string            // Optional session identifier
	Tags      []string          // Optional tags for filtering
	Metadata  map[string]string // Optional metadata
}

// SpanOptions contains options for creating spans within a trace
type SpanOptions struct {
	Name     string            // Span name (e.g., "llm_call", "embedding_generation")
	Input    interface{}       // Input data
	Metadata map[string]string // Optional metadata
}

// GenerationOptions contains options for LLM generation tracking
type GenerationOptions struct {
	Model       string  // Model name (e.g., "gemini-flash-lite-latest")
	Prompt      string  // The prompt sent to the LLM
	Completion  string  // The completion received from the LLM
	Temperature float32 // Temperature parameter
	MaxTokens   int32   // Max tokens parameter

	// Token usage
	PromptTokens     int // Tokens in prompt
	CompletionTokens int // Tokens in completion
	TotalTokens      int // Total tokens used

	// Performance
	LatencyMs int64 // Latency in milliseconds

	// Cost (optional)
	TotalCost float64 // Total cost in USD
}

// NewLangFuseClient creates a new LangFuse observability client
func NewLangFuseClient() (*LangFuseClient, error) {
	cfg := config.GetLangFuseConfig()

	if !cfg.Enabled {
		return &LangFuseClient{
			enabled: false,
			log:     slog.Default(),
			config:  cfg,
		}, nil
	}

	if cfg.PublicKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("LangFuse enabled but missing credentials (public_key or secret_key)")
	}

	// TODO: Implement actual HTTP-based LangFuse API integration
	// For now, we log locally
	slog.Info("LangFuse observability enabled (local logging mode)",
		"host", cfg.Host)

	return &LangFuseClient{
		enabled: true,
		log:     slog.Default(),
		config:  cfg,
	}, nil
}

// IsEnabled returns whether LangFuse tracking is enabled
func (l *LangFuseClient) IsEnabled() bool {
	return l.enabled
}

// CreateTrace creates a new trace for tracking a complete operation
func (l *LangFuseClient) CreateTrace(ctx context.Context, opts TraceOptions) (*TraceClient, error) {
	if !l.enabled {
		return nil, nil
	}

	// Generate a simple trace ID
	traceID := fmt.Sprintf("trace_%d", time.Now().UnixNano())

	l.log.Info("LangFuse trace created",
		"trace_id", traceID,
		"name", opts.Name,
		"user_id", opts.UserID,
		"tags", opts.Tags)

	return &TraceClient{
		id:   traceID,
		name: opts.Name,
		log:  l.log,
	}, nil
}

// CreateSpan creates a span within a trace for tracking sub-operations
func (l *LangFuseClient) CreateSpan(trace *TraceClient, opts SpanOptions) *SpanClient {
	if !l.enabled || trace == nil {
		return nil
	}

	spanID := fmt.Sprintf("span_%d", time.Now().UnixNano())

	l.log.Info("LangFuse span created",
		"span_id", spanID,
		"trace_id", trace.id,
		"name", opts.Name)

	return &SpanClient{
		id:   spanID,
		name: opts.Name,
		log:  l.log,
	}
}

// TrackGeneration tracks an LLM generation with detailed metrics
func (l *LangFuseClient) TrackGeneration(trace *TraceClient, opts GenerationOptions) error {
	if !l.enabled || trace == nil {
		return nil
	}

	// Calculate cost if not provided (example rates for Gemini)
	if opts.TotalCost == 0 && opts.TotalTokens > 0 {
		opts.TotalCost = l.estimateCost(opts.Model, opts.PromptTokens, opts.CompletionTokens)
	}

	l.log.Info("LangFuse generation tracked",
		"trace_id", trace.id,
		"model", opts.Model,
		"tokens", opts.TotalTokens,
		"latency_ms", opts.LatencyMs,
		"cost", fmt.Sprintf("$%.6f", opts.TotalCost))

	return nil
}

// EndSpan marks a span as complete with optional output
func (l *LangFuseClient) EndSpan(span *SpanClient, output interface{}, err error) {
	if !l.enabled || span == nil {
		return
	}

	if err != nil {
		l.log.Info("LangFuse span ended with error",
			"span_id", span.id,
			"error", err.Error())
	} else {
		l.log.Info("LangFuse span ended",
			"span_id", span.id)
	}
}

// Flush ensures all pending traces are sent to LangFuse
func (l *LangFuseClient) Flush() error {
	if !l.enabled {
		return nil
	}

	// TODO: Implement actual flush when HTTP API is integrated
	l.log.Info("LangFuse flush called")
	return nil
}

// Shutdown gracefully shuts down the LangFuse client
func (l *LangFuseClient) Shutdown(ctx context.Context) error {
	if !l.enabled {
		return nil
	}

	// TODO: Implement actual shutdown when HTTP API is integrated
	l.log.Info("LangFuse shutdown")
	return nil
}

// estimateCost estimates the cost of an LLM call based on token usage
// These are example rates and should be updated based on actual pricing
func (l *LangFuseClient) estimateCost(model string, promptTokens, completionTokens int) float64 {
	// Example pricing (per 1M tokens) - update with actual rates
	var promptCostPer1M, completionCostPer1M float64

	switch {
	case model == "gemini-flash-lite-latest":
		// Gemini Flash pricing example
		promptCostPer1M = 0.075    // $0.075 per 1M input tokens
		completionCostPer1M = 0.30 // $0.30 per 1M output tokens
	case model == "text-embedding-004":
		// Embedding pricing example
		promptCostPer1M = 0.00001 // Very cheap for embeddings
		completionCostPer1M = 0.0
	default:
		// Default conservative estimate
		promptCostPer1M = 0.50
		completionCostPer1M = 1.50
	}

	promptCost := float64(promptTokens) / 1_000_000.0 * promptCostPer1M
	completionCost := float64(completionTokens) / 1_000_000.0 * completionCostPer1M

	return promptCost + completionCost
}

// Helper function to create a simple trace for one-off operations
func (l *LangFuseClient) SimpleTrace(ctx context.Context, name string, fn func(*TraceClient) error) error {
	if !l.enabled {
		return fn(nil)
	}

	trace, err := l.CreateTrace(ctx, TraceOptions{
		Name: name,
	})
	if err != nil {
		return fmt.Errorf("failed to create trace: %w", err)
	}

	return fn(trace)
}
