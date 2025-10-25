package pipeline

import (
	"briefly/internal/categorization"
	"briefly/internal/core"
	"briefly/internal/llm"
	"briefly/internal/summarize"
	"context"
	"fmt"
)

// Builder helps construct a fully configured Pipeline
type Builder struct {
	cacheDir   string
	llmClient  *llm.Client
	config     *Config
	skipCache  bool
	skipBanner bool
}

// NewBuilder creates a new pipeline builder with default settings
func NewBuilder() *Builder {
	return &Builder{
		cacheDir:   ".briefly-cache",
		config:     DefaultConfig(),
		skipCache:  false,
		skipBanner: false,
	}
}

// WithCacheDir sets the cache directory
func (b *Builder) WithCacheDir(dir string) *Builder {
	b.cacheDir = dir
	return b
}

// WithLLMClient sets the LLM client
func (b *Builder) WithLLMClient(client *llm.Client) *Builder {
	b.llmClient = client
	return b
}

// WithConfig sets the pipeline configuration
func (b *Builder) WithConfig(config *Config) *Builder {
	b.config = config
	return b
}

// WithoutCache disables caching
func (b *Builder) WithoutCache() *Builder {
	b.skipCache = true
	if b.config != nil {
		b.config.CacheEnabled = false
	}
	return b
}

// WithoutBanner disables banner generation
func (b *Builder) WithoutBanner() *Builder {
	b.skipBanner = true
	return b
}

// WithBanner enables banner generation
func (b *Builder) WithBanner(style string) *Builder {
	b.skipBanner = false
	if b.config != nil {
		b.config.GenerateBanner = true
		b.config.BannerStyle = style
	}
	return b
}

// Build constructs a fully configured Pipeline
func (b *Builder) Build() (*Pipeline, error) {
	// Validate required components
	if b.llmClient == nil {
		return nil, fmt.Errorf("LLM client is required")
	}

	// Initialize all adapters
	parser := NewParserAdapter()
	fetcher := NewFetcherAdapter()
	embedder := NewLLMAdapter(b.llmClient)
	clusterer := NewClustererAdapter()
	orderer := NewOrdererAdapter()
	renderer := NewRendererAdapter()

	// Create LLM client adapter for narrative generation
	llmClientAdapter := NewLLMClientAdapter(b.llmClient)
	narrative := NewNarrativeAdapter(llmClientAdapter)

	// Initialize cache (optional)
	var cache CacheManager
	if !b.skipCache && b.config.CacheEnabled {
		cacheAdapter, err := NewCacheAdapter(b.cacheDir)
		if err != nil {
			// Non-fatal: log warning and continue without cache
			fmt.Printf("Warning: failed to initialize cache: %v\n", err)
			cache = nil
			b.config.CacheEnabled = false
		} else {
			cache = cacheAdapter
		}
	}

	// Initialize banner generator (optional)
	var banner BannerGenerator
	if !b.skipBanner && b.config.GenerateBanner {
		banner = NewBannerAdapter()
	}

	// Create summarizer adapter using the new summarize package
	llmClientForSummarize := &LLMClientForSummarize{client: b.llmClient}
	summarizerCore := summarize.NewSummarizerWithDefaults(llmClientForSummarize)
	summarizer := &SummarizerAdapter{summarizer: summarizerCore}

	// Create categorizer adapter using the new categorization package
	categorizerCore := categorization.NewCategorizer(llmClientAdapter, categorization.DefaultCategories())
	categorizer := NewCategorizerAdapter(categorizerCore)

	// Build pipeline
	pipeline := NewPipeline(
		parser,
		fetcher,
		summarizer,
		categorizer,
		embedder,
		clusterer,
		orderer,
		narrative,
		renderer,
		cache,
		banner,
		b.config,
	)

	return pipeline, nil
}

// SummarizerAdapter wraps internal/summarize to implement pipeline.ArticleSummarizer
type SummarizerAdapter struct {
	summarizer *summarize.Summarizer
}

func (s *SummarizerAdapter) SummarizeArticle(ctx context.Context, article *core.Article) (*core.Summary, error) {
	return s.summarizer.SummarizeArticle(ctx, article)
}

func (s *SummarizerAdapter) GenerateKeyPoints(ctx context.Context, content string) ([]string, error) {
	return s.summarizer.GenerateKeyPoints(ctx, content)
}

func (s *SummarizerAdapter) ExtractTitle(ctx context.Context, content string) (string, error) {
	return s.summarizer.ExtractTitle(ctx, content)
}

// LLMClientForSummarize adapts llm.Client to summarize.LLMClient interface
type LLMClientForSummarize struct {
	client *llm.Client
}

func (l *LLMClientForSummarize) GenerateText(ctx context.Context, prompt string, options interface{}) (string, error) {
	return l.client.GenerateText(ctx, prompt, llm.TextGenerationOptions{})
}