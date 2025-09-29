package pipeline

import (
	"briefly/internal/core"
	"briefly/internal/llm"
	"context"
	"fmt"
	"strings"
	"time"
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

	// Create summarizer adapter
	// Note: We'll need to create this once we build the summarize package
	// For now, use a placeholder that will be implemented
	summarizer := &PlaceholderSummarizer{llmClient: b.llmClient}

	// Build pipeline
	pipeline := NewPipeline(
		parser,
		fetcher,
		summarizer,
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

// PlaceholderSummarizer is a temporary implementation until we create internal/summarize
type PlaceholderSummarizer struct {
	llmClient *llm.Client
}

func (s *PlaceholderSummarizer) SummarizeArticle(ctx context.Context, article *core.Article) (*core.Summary, error) {
	// Temporary implementation using existing LLM client
	// This will be replaced by proper summarize package
	prompt := fmt.Sprintf(`Summarize this article in 150 words and provide 3-5 key points.

Title: %s
Content: %s

Format your response as:
SUMMARY:
[150-word summary here]

KEY POINTS:
- [point 1]
- [point 2]
- [point 3]
`, article.Title, truncateForPrompt(article.CleanedText, 3000))

	response, err := s.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %w", err)
	}

	// Parse response to extract summary and key points
	summary := &core.Summary{
		ID:            generateID(),
		ArticleIDs:    []string{article.ID},
		SummaryText:   response, // Simplified for now
		ModelUsed:     "gemini-2.5-flash-preview-05-20",
		DateGenerated: time.Now(),
	}

	return summary, nil
}

func (s *PlaceholderSummarizer) GenerateKeyPoints(ctx context.Context, content string) ([]string, error) {
	prompt := fmt.Sprintf(`Extract 3-5 key points from this content:

%s

Provide key points as a bulleted list.`, truncateForPrompt(content, 2000))

	response, err := s.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{})
	if err != nil {
		return nil, err
	}

	// Simple parsing - split by lines starting with - or *
	points := []string{}
	for _, line := range strings.Split(response, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") {
			points = append(points, strings.TrimSpace(line[1:]))
		}
	}

	return points, nil
}

func (s *PlaceholderSummarizer) ExtractTitle(ctx context.Context, content string) (string, error) {
	prompt := fmt.Sprintf(`Generate a concise, descriptive title (5-10 words) for this content:

%s

Title:`, truncateForPrompt(content, 1000))

	return s.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{})
}

func truncateForPrompt(text string, maxChars int) string {
	if len(text) <= maxChars {
		return text
	}
	return text[:maxChars] + "..."
}