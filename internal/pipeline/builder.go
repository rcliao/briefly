package pipeline

import (
	"briefly/internal/categorization"
	"briefly/internal/core"
	"briefly/internal/llm"
	"briefly/internal/observability"
	"briefly/internal/persistence"
	"briefly/internal/summarize"
	"briefly/internal/tags" // Phase 1: For tag classification
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai" // Phase 1: For structured summaries
)

// Builder helps construct a fully configured Pipeline
type Builder struct {
	cacheDir       string
	llmClient      *llm.Client
	tracedClient   *llm.TracedClient    // For theme classification with observability
	db             persistence.Database // Optional: for theme-based categorization
	langfuse       *observability.LangFuseClient
	posthog        *observability.PostHogClient
	config         *Config
	skipCache      bool
	skipBanner     bool
	useThemeSystem bool // Enable theme-based categorization
	vectorStore    VectorStore // Phase 2: Optional vector store for semantic search
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

// WithDatabase sets the database for theme-based categorization
func (b *Builder) WithDatabase(db persistence.Database) *Builder {
	b.db = db
	b.useThemeSystem = true
	return b
}

// WithTracedClient sets the traced LLM client with observability
func (b *Builder) WithTracedClient(client *llm.TracedClient) *Builder {
	b.tracedClient = client
	return b
}

// WithObservability sets the observability clients
func (b *Builder) WithObservability(langfuse *observability.LangFuseClient, posthog *observability.PostHogClient) *Builder {
	b.langfuse = langfuse
	b.posthog = posthog
	return b
}

// WithoutThemes disables theme-based categorization (use legacy categories)
func (b *Builder) WithoutThemes() *Builder {
	b.useThemeSystem = false
	return b
}

// WithVectorStore sets the vector store for semantic search (Phase 2)
func (b *Builder) WithVectorStore(store VectorStore) *Builder {
	b.vectorStore = store
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
	var summarizerCore summarize.SummarizerInterface
	summarizerCore = summarize.NewSummarizerWithDefaults(llmClientForSummarize)

	// Phase 1: Wrap with LangFuse tracking if available
	if b.langfuse != nil && b.langfuse.IsEnabled() {
		fmt.Println("📊 LangFuse tracking enabled for summarization")
		summarizerCore = summarize.NewTracedSummarizer(summarizerCore.(*summarize.Summarizer), b.langfuse)
	}

	summarizer := &SummarizerAdapter{
		summarizer:    summarizerCore,
		useStructured: b.config.UseStructuredSummaries, // Phase 1
	}

	// Create categorizer: use theme-based if database is available, otherwise use legacy
	var categorizer ArticleCategorizer
	if b.useThemeSystem && b.db != nil && b.tracedClient != nil {
		// Use theme-based categorization with database
		fmt.Println("🎨 Using theme-based categorization")
		themeCategorizer := NewThemeCategorizer(b.db, b.tracedClient, b.posthog)
		categorizer = themeCategorizer
	} else {
		// Fall back to legacy categorization (uses old interface)
		fmt.Println("📁 Using legacy categorization (no database or theme system disabled)")
		legacyAdapter := NewLegacyLLMClientAdapter(b.llmClient)
		categorizerCore := categorization.NewCategorizer(legacyAdapter, categorization.DefaultCategories())
		categorizer = NewCategorizerAdapter(categorizerCore)
	}

	// Create citation tracker if database is available (Phase 1)
	var citationTracker CitationTracker
	if b.db != nil {
		fmt.Println("📚 Citation tracking enabled")
		citationTracker = NewCitationTrackerAdapter(b.db)
	}

	// Phase 1: Create article repository for cluster persistence
	var articleRepo ArticleRepository
	if b.db != nil {
		articleRepo = b.db.Articles()
	}

	// Phase 1: Create tag classifier and repository for hierarchical clustering
	var tagClassifier TagClassifier
	var tagRepo TagRepository
	if b.db != nil && b.tracedClient != nil {
		fmt.Println("🏷️  Tag classification enabled")
		tagClassifier = NewTagClassifierAdapter(b.tracedClient, b.posthog)
		tagRepo = b.db.Tags()
	}

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
		citationTracker,
		nil,             // digestRepo: Optional, will be wired up when needed (v2.0)
		articleRepo,     // Phase 1: For persisting cluster assignments
		tagClassifier,   // Phase 1: For multi-label tag classification
		tagRepo,         // Phase 1: For tag persistence
		b.vectorStore,   // Phase 2: Vector store for semantic search
		b.config,
	)

	return pipeline, nil
}

// SummarizerAdapter wraps internal/summarize to implement pipeline.ArticleSummarizer
type SummarizerAdapter struct {
	summarizer    summarize.SummarizerInterface // Phase 1: Use interface for flexibility
	useStructured bool                          // Phase 1: Use structured summaries
}

func (s *SummarizerAdapter) SummarizeArticle(ctx context.Context, article *core.Article) (*core.Summary, error) {
	// Phase 1: Choose between simple and structured summaries
	if s.useStructured {
		return s.summarizer.SummarizeArticleStructured(ctx, article)
	}
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
	// Phase 1: Handle structured summary options with ResponseSchema
	llmOptions := llm.TextGenerationOptions{}

	// Try to extract options if provided
	if options != nil {
		// Handle struct with ResponseSchema (for structured summaries)
		type StructuredOptions struct {
			ResponseSchema *genai.Schema
			Temperature    float32
		}

		// Type assertion to check if it's our structured options
		if structOpts, ok := options.(StructuredOptions); ok {
			llmOptions.ResponseSchema = structOpts.ResponseSchema
			llmOptions.Temperature = structOpts.Temperature
		}
	}

	return l.client.GenerateText(ctx, prompt, llmOptions)
}

// TagClassifierAdapter adapts internal/tags.Classifier to implement pipeline.TagClassifier
type TagClassifierAdapter struct {
	classifier *tags.Classifier
}

// NewTagClassifierAdapter creates a new tag classifier adapter
func NewTagClassifierAdapter(tracedClient *llm.TracedClient, posthog *observability.PostHogClient) *TagClassifierAdapter {
	classifier := tags.NewClassifierWithClients(tracedClient, posthog)
	return &TagClassifierAdapter{classifier: classifier}
}

func (t *TagClassifierAdapter) ClassifyArticle(ctx context.Context, article core.Article, summary *core.Summary, tagList []core.Tag, minRelevance float64) (*TagClassificationResult, error) {
	result, err := t.classifier.ClassifyArticle(ctx, article, summary, tagList, minRelevance)
	if err != nil {
		return nil, err
	}

	// Convert tags.ClassificationResult to pipeline.TagClassificationResult
	pipelineResult := &TagClassificationResult{
		ArticleID: result.ArticleID,
		ThemeID:   result.ThemeID,
		Tags:      make([]TagClassificationResultItem, len(result.Tags)),
	}

	for i, tag := range result.Tags {
		pipelineResult.Tags[i] = TagClassificationResultItem{
			TagID:          tag.TagID,
			TagName:        tag.TagName,
			RelevanceScore: tag.RelevanceScore,
			Reasoning:      tag.Reasoning,
		}
	}

	return pipelineResult, nil
}

func (t *TagClassifierAdapter) ClassifyWithinTheme(ctx context.Context, article core.Article, summary *core.Summary, themeID string, allTags []core.Tag, minRelevance float64) (*TagClassificationResult, error) {
	result, err := t.classifier.ClassifyWithinTheme(ctx, article, summary, themeID, allTags, minRelevance)
	if err != nil {
		return nil, err
	}

	// Convert tags.ClassificationResult to pipeline.TagClassificationResult
	pipelineResult := &TagClassificationResult{
		ArticleID: result.ArticleID,
		ThemeID:   result.ThemeID,
		Tags:      make([]TagClassificationResultItem, len(result.Tags)),
	}

	for i, tag := range result.Tags {
		pipelineResult.Tags[i] = TagClassificationResultItem{
			TagID:          tag.TagID,
			TagName:        tag.TagName,
			RelevanceScore: tag.RelevanceScore,
			Reasoning:      tag.Reasoning,
		}
	}

	return pipelineResult, nil
}

func (t *TagClassifierAdapter) ClassifyBatch(ctx context.Context, articles []core.Article, summaries map[string]*core.Summary, tagList []core.Tag, minRelevance float64) (map[string]*TagClassificationResult, error) {
	results, err := t.classifier.ClassifyBatch(ctx, articles, summaries, tagList, minRelevance)
	if err != nil {
		return nil, err
	}

	// Convert map[string]*tags.ClassificationResult to map[string]*pipeline.TagClassificationResult
	pipelineResults := make(map[string]*TagClassificationResult)
	for articleID, result := range results {
		pipelineResult := &TagClassificationResult{
			ArticleID: result.ArticleID,
			ThemeID:   result.ThemeID,
			Tags:      make([]TagClassificationResultItem, len(result.Tags)),
		}

		for i, tag := range result.Tags {
			pipelineResult.Tags[i] = TagClassificationResultItem{
				TagID:          tag.TagID,
				TagName:        tag.TagName,
				RelevanceScore: tag.RelevanceScore,
				Reasoning:      tag.Reasoning,
			}
		}

		pipelineResults[articleID] = pipelineResult
	}

	return pipelineResults, nil
}
