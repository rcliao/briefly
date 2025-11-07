package pipeline

import (
	"briefly/internal/core"
	"briefly/internal/narrative"
	"briefly/internal/quality"
	"context"
	"fmt"
	"time"
)

// Pipeline orchestrates the end-to-end digest generation workflow
// It coordinates all components according to the simplified architecture
type Pipeline struct {
	// Core components
	parser        URLParser
	fetcher       ContentFetcher
	summarizer    ArticleSummarizer
	categorizer   ArticleCategorizer // NEW: Categorizes articles
	embedder      EmbeddingGenerator
	clusterer     TopicClusterer
	orderer       ArticleOrderer
	narrative     NarrativeGenerator
	renderer      MarkdownRenderer
	cache         CacheManager
	banner        BannerGenerator   // Optional
	citationTracker CitationTracker // Phase 1: Track citations for articles
	digestRepo    DigestRepository  // Optional: For storing digests in database (v2.0)

	// Configuration
	config *Config
}

// Config holds pipeline configuration
type Config struct {
	// Cache settings
	CacheEnabled bool
	CacheTTL     time.Duration

	// Processing settings
	MaxConcurrentFetches int
	RetryAttempts        int
	RequestTimeout       time.Duration

	// Output settings
	OutputFormat    string // Always "markdown" for now
	GenerateBanner  bool
	BannerStyle     string

	// Quality settings
	MinArticleLength  int     // Minimum chars for valid article
	MinSummaryQuality float64 // 0-1 quality threshold

	// Phase 1: Summary settings
	UseStructuredSummaries bool // Use structured summaries with sections (default: false)
}

// DefaultConfig returns sensible default configuration
func DefaultConfig() *Config {
	return &Config{
		CacheEnabled:         true,
		CacheTTL:             7 * 24 * time.Hour, // 7 days
		MaxConcurrentFetches: 5,
		RetryAttempts:        3,
		RequestTimeout:       30 * time.Second,
		OutputFormat:         "markdown",
		GenerateBanner:       false,
		BannerStyle:          "tech",
		MinArticleLength:     100,
		MinSummaryQuality:    0.5,
		UseStructuredSummaries: false, // Default to simple summaries for backward compatibility
	}
}

// NewPipeline creates a new pipeline with all dependencies
func NewPipeline(
	parser URLParser,
	fetcher ContentFetcher,
	summarizer ArticleSummarizer,
	categorizer ArticleCategorizer,
	embedder EmbeddingGenerator,
	clusterer TopicClusterer,
	orderer ArticleOrderer,
	narrative NarrativeGenerator,
	renderer MarkdownRenderer,
	cache CacheManager,
	banner BannerGenerator,
	citationTracker CitationTracker,
	digestRepo DigestRepository,  // v2.0: Optional digest repository for database storage
	config *Config,
) *Pipeline {
	if config == nil {
		config = DefaultConfig()
	}

	return &Pipeline{
		parser:          parser,
		fetcher:         fetcher,
		summarizer:      summarizer,
		categorizer:     categorizer,
		embedder:        embedder,
		clusterer:       clusterer,
		orderer:         orderer,
		narrative:       narrative,
		renderer:        renderer,
		cache:           cache,
		banner:          banner,
		citationTracker: citationTracker,
		digestRepo:      digestRepo,
		config:          config,
	}
}

// DigestOptions configures digest generation
type DigestOptions struct {
	InputFile      string
	OutputPath     string
	GenerateBanner bool
	BannerStyle    string
	DryRun         bool
}

// DigestResult contains the output of digest generation
type DigestResult struct {
	Digest       *core.Digest
	MarkdownPath string
	BannerPath   string
	Stats        ProcessingStats
}

// ProcessingStats tracks pipeline execution metrics
type ProcessingStats struct {
	TotalURLs          int
	SuccessfulArticles int
	FailedArticles     int
	CacheHits          int
	CacheMisses        int
	ClustersGenerated  int
	ProcessingTime     time.Duration
	StartTime          time.Time
	EndTime            time.Time
}

// GenerateDigests executes the full digest generation pipeline (v2.0)
// Returns multiple digests - one per topic cluster (Kagi News style)
func (p *Pipeline) GenerateDigests(ctx context.Context, opts DigestOptions) ([]DigestResult, error) {
	startTime := time.Now()
	stats := ProcessingStats{
		StartTime: startTime,
	}

	// Step 1: Parse URLs from markdown file
	fmt.Printf("📄 Step 1/9: Parsing URLs from %s...\n", opts.InputFile)
	links, err := p.parser.ParseMarkdownFile(opts.InputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URLs: %w", err)
	}

	stats.TotalURLs = len(links)
	if stats.TotalURLs == 0 {
		return nil, fmt.Errorf("no valid URLs found in input file")
	}
	fmt.Printf("   ✓ Found %d URLs\n\n", stats.TotalURLs)

	// Step 2: Fetch and summarize articles (with caching)
	fmt.Printf("🔍 Step 2/9: Fetching and summarizing articles...\n")
	articles, summaries, err := p.processArticles(ctx, links, &stats)
	if err != nil {
		return nil, fmt.Errorf("failed to process articles: %w", err)
	}

	if len(articles) == 0 {
		return nil, fmt.Errorf("no articles were successfully processed")
	}

	stats.SuccessfulArticles = len(articles)
	stats.FailedArticles = stats.TotalURLs - stats.SuccessfulArticles
	fmt.Printf("   ✓ Successfully processed %d/%d articles\n", stats.SuccessfulArticles, stats.TotalURLs)
	fmt.Printf("   • Cache hits: %d, Cache misses: %d\n\n", stats.CacheHits, stats.CacheMisses)

	// Step 3: Generate embeddings for clustering
	fmt.Printf("🧠 Step 3/9: Generating embeddings for clustering...\n")
	embeddings, err := p.generateEmbeddings(ctx, articles, summaries)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}
	fmt.Printf("   ✓ Generated %d embeddings\n\n", len(embeddings))

	// Step 4: Cluster articles by topic
	fmt.Printf("🔗 Step 4/9: Clustering articles by topic...\n")
	clusters, err := p.clusterer.ClusterArticles(ctx, articles, summaries, embeddings)
	if err != nil {
		return nil, fmt.Errorf("failed to cluster articles: %w", err)
	}

	stats.ClustersGenerated = len(clusters)
	fmt.Printf("   ✓ Created %d topic clusters\n\n", stats.ClustersGenerated)

	// Quality Gate: Validate clustering quality
	clusteringGate := NewClusteringQualityGate(
		DefaultQualityGateConfig(),
		clusters,
		embeddings,
	)
	if err := clusteringGate.Validate(ctx); err != nil {
		return nil, fmt.Errorf("clustering quality gate failed: %w", err)
	}

	// Step 5: Generate cluster narratives (hierarchical summarization)
	fmt.Printf("📖 Step 5/9: Generating cluster narratives from ALL articles...\n")
	clusters, err = p.generateClusterNarratives(ctx, clusters, articles, summaries)
	if err != nil {
		// Non-fatal: log warning and continue without cluster narratives
		fmt.Printf("   ⚠️  Cluster narrative generation failed: %v\n", err)
		fmt.Printf("   • Continuing with legacy top-3 article summarization\n\n")
	} else {
		narrativeCount := 0
		for _, cluster := range clusters {
			if cluster.Narrative != nil {
				narrativeCount++
			}
		}
		fmt.Printf("   ✓ Generated %d cluster narratives\n", narrativeCount)
		fmt.Printf("   ✓ Each narrative synthesizes ALL articles in its cluster\n\n")

		// Quality Gate: Validate cluster narratives
		narrativeGate := NewNarrativeQualityGate(
			DefaultQualityGateConfig(),
			clusters,
		)
		if err := narrativeGate.Validate(ctx); err != nil {
			// Non-blocking: log warning but continue
			fmt.Printf("   ⚠️  Narrative quality gate warning (non-blocking)\n")
		}
	}

	// Step 6-9: Generate one digest per cluster (v2.0 architecture)
	fmt.Printf("📝 Step 6/9: Generating %d digests (one per cluster)...\n", len(clusters))

	results := make([]DigestResult, 0, len(clusters))
	articleMap := articlesToMap(articles)
	summaryMap := summariesToMap(summaries)

	for i, cluster := range clusters {
		fmt.Printf("\n   [Cluster %d/%d] Label: %s (%d articles)\n", i+1, len(clusters), cluster.Label, len(cluster.ArticleIDs))

		// Build digest for this cluster
		clusterArticles := make([]core.Article, 0, len(cluster.ArticleIDs))
		clusterSummaries := make([]core.Summary, 0, len(cluster.ArticleIDs))

		for _, articleID := range cluster.ArticleIDs {
			if article, found := articleMap[articleID]; found {
				clusterArticles = append(clusterArticles, article)
			}
			if summary, found := summaryMap[articleID]; found {
				clusterSummaries = append(clusterSummaries, summary)
			}
		}

		// Generate digest content for this cluster using hierarchical summarization
		digest := p.buildDigestForCluster(cluster, clusterArticles, clusterSummaries)

		// Generate title, TLDR, and summary using cluster narrative (hierarchical approach)
		// Pass this single cluster to the narrative generator
		singleClusterSlice := []core.TopicCluster{cluster}
		digestContent, err := p.generateDigestContentWithNarratives(ctx, singleClusterSlice, clusterArticles, clusterSummaries)
		if err != nil {
			fmt.Printf("   ⚠️  Digest content generation failed: %v\n", err)
			digestContent = &narrative.DigestContent{
				Title:            fmt.Sprintf("%s - %s", cluster.Label, time.Now().Format("Jan 2")),
				TLDRSummary:      "",
				KeyMoments:       []core.KeyMoment{},
				ExecutiveSummary: "",
			}
		} else {
			fmt.Printf("   ✓ Generated: %s\n", digestContent.Title)
		}

		// Update digest with generated content
		digest.Title = digestContent.Title
		digest.TLDRSummary = digestContent.TLDRSummary
		digest.KeyMoments = digestContent.KeyMoments // v2.0 structured format
		digest.Perspectives = digestContent.Perspectives // v2.0 structured format
		digest.Metadata.Title = digestContent.Title
		digest.Metadata.TLDRSummary = digestContent.TLDRSummary
		// Note: Metadata.KeyMoments is deprecated (legacy []string format)

		// Store digest in database with relationships (v2.0)
		if p.digestRepo != nil {
			// Extract article IDs
			articleIDs := make([]string, len(clusterArticles))
			for idx, article := range clusterArticles {
				articleIDs[idx] = article.ID
			}

			// Extract theme IDs from articles
			themeIDMap := make(map[string]bool)
			for _, article := range clusterArticles {
				if article.ThemeID != nil && *article.ThemeID != "" {
					themeIDMap[*article.ThemeID] = true
				}
			}
			themeIDs := make([]string, 0, len(themeIDMap))
			for themeID := range themeIDMap {
				themeIDs = append(themeIDs, themeID)
			}

			// Store with relationships
			if err := p.digestRepo.StoreWithRelationships(ctx, digest, articleIDs, themeIDs); err != nil {
				// Non-fatal: log warning and continue
				fmt.Printf("   ⚠️  Database storage failed: %v\n", err)
				fmt.Printf("   • Continuing with markdown generation\n")
			} else {
				fmt.Printf("   ✓ Stored in database\n")

				// Calculate and store quality metrics
				if err := p.storeQualityMetrics(ctx, digest, clusterArticles); err != nil {
					// Non-fatal: log warning but continue
					fmt.Printf("   ⚠️  Quality metrics storage failed: %v\n", err)
				}
			}
		}

		// Render markdown for this digest
		markdownPath := ""
		if opts.OutputPath != "" {
			markdownPath, err = p.renderer.RenderDigest(ctx, digest, opts.OutputPath)
			if err != nil {
				fmt.Printf("   ⚠️  Rendering failed: %v\n", err)
			} else {
				fmt.Printf("   ✓ Saved to %s\n", markdownPath)
			}
		}

		results = append(results, DigestResult{
			Digest:       digest,
			MarkdownPath: markdownPath,
			BannerPath:   "",
			Stats:        stats,
		})
	}

	stats.EndTime = time.Now()
	stats.ProcessingTime = stats.EndTime.Sub(startTime)

	fmt.Printf("\n✅ Generated %d digests successfully\n", len(results))
	fmt.Printf("⏱️  Total processing time: %v\n\n", stats.ProcessingTime)

	return results, nil
}

// QuickReadOptions configures quick read operation
type QuickReadOptions struct {
	URL string
}

// QuickReadResult contains the output of quick read
type QuickReadResult struct {
	Article     *core.Article
	Summary     *core.Summary
	Markdown    string
	WasCached   bool
	ProcessTime time.Duration
}

// QuickRead processes a single URL and returns a summary
func (p *Pipeline) QuickRead(ctx context.Context, opts QuickReadOptions) (*QuickReadResult, error) {
	startTime := time.Now()

	// Step 1: Check cache
	cachedArticle, cachedSummary, err := p.checkQuickReadCache(opts.URL)
	if err == nil && cachedArticle != nil && cachedSummary != nil {
		// Render cached result
		markdown, err := p.renderer.RenderQuickRead(ctx, cachedArticle, cachedSummary)
		if err != nil {
			return nil, fmt.Errorf("failed to render cached summary: %w", err)
		}

		return &QuickReadResult{
			Article:     cachedArticle,
			Summary:     cachedSummary,
			Markdown:    markdown,
			WasCached:   true,
			ProcessTime: time.Since(startTime),
		}, nil
	}

	// Step 2: Fetch article
	article, err := p.fetcher.FetchArticle(ctx, opts.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch article: %w", err)
	}

	// Step 3: Summarize
	summary, err := p.summarizer.SummarizeArticle(ctx, article)
	if err != nil {
		return nil, fmt.Errorf("failed to summarize article: %w", err)
	}

	// Step 4: Cache result
	if p.config.CacheEnabled {
		_ = p.cacheQuickRead(article, summary)
	}

	// Step 5: Render
	markdown, err := p.renderer.RenderQuickRead(ctx, article, summary)
	if err != nil {
		return nil, fmt.Errorf("failed to render summary: %w", err)
	}

	return &QuickReadResult{
		Article:     article,
		Summary:     summary,
		Markdown:    markdown,
		WasCached:   false,
		ProcessTime: time.Since(startTime),
	}, nil
}

// processArticles fetches and summarizes all articles with cache support
func (p *Pipeline) processArticles(ctx context.Context, links []core.Link, stats *ProcessingStats) ([]core.Article, []core.Summary, error) {
	articles := make([]core.Article, 0, len(links))
	summaries := make([]core.Summary, 0, len(links))

	// Process each link (TODO: Add concurrency control)
	for i, link := range links {
		fmt.Printf("   [%d/%d] Processing: %s\n", i+1, len(links), link.URL)

		// Check cache first
		if p.config.CacheEnabled {
			cachedArticle, cachedSummary, err := p.checkArticleCache(link.URL)
			if err == nil && cachedArticle != nil && cachedSummary != nil {
				fmt.Printf("           ✓ Cache hit\n")
				articles = append(articles, *cachedArticle)
				summaries = append(summaries, *cachedSummary)
				stats.CacheHits++
				continue
			}
		}

		stats.CacheMisses++

		// Fetch article
		article, err := p.fetcher.FetchArticle(ctx, link.URL)
		if err != nil {
			// Log error but continue with other articles
			fmt.Printf("           ✗ Fetch failed: %v\n", err)
			continue
		}

		// Validate article quality
		if len(article.CleanedText) < p.config.MinArticleLength {
			// Skip articles that are too short
			fmt.Printf("           ✗ Article too short (%d chars)\n", len(article.CleanedText))
			continue
		}

		// Summarize article
		summary, err := p.summarizer.SummarizeArticle(ctx, article)
		if err != nil {
			// Log error but continue with other articles
			fmt.Printf("           ✗ Summarization failed: %v\n", err)
			continue
		}

		// Track citation (Phase 1) - non-fatal if it fails
		if p.citationTracker != nil {
			_, err := p.citationTracker.TrackArticle(ctx, article)
			if err != nil {
				// Log warning but continue - citation tracking is not critical
				fmt.Printf("           ⚠️  Citation tracking failed: %v\n", err)
			}
		}

		// Cache result
		if p.config.CacheEnabled {
			_ = p.cacheArticle(article, summary)
		}

		fmt.Printf("           ✓ Fetched and summarized\n")
		articles = append(articles, *article)
		summaries = append(summaries, *summary)
	}

	return articles, summaries, nil
}

// generateEmbeddings creates vector embeddings for all articles using FULL CONTENT
// This provides richer semantic information for clustering compared to using just summaries
func (p *Pipeline) generateEmbeddings(ctx context.Context, articles []core.Article, summaries []core.Summary) (map[string][]float64, error) {
	embeddings := make(map[string][]float64)
	var failedCount int

	// Build article map for fast lookup
	articleMap := make(map[string]*core.Article)
	for i := range articles {
		articleMap[articles[i].ID] = &articles[i]
	}

	for i, summary := range summaries {
		fmt.Printf("   [%d/%d] Generating embedding for article %s\n", i+1, len(summaries), summary.ID)

		// Get corresponding article
		article, found := articleMap[summary.ArticleIDs[0]] // Summary.ArticleIDs[0] is the article ID
		if !found {
			fmt.Printf("           ✗ Article not found for summary %s\n", summary.ID)
			failedCount++
			continue
		}

		// Use first 2000 words of article content for richer semantics
		// This provides much better clustering than using just 150-word summaries
		embeddingText := article.CleanedText
		maxChars := 2000 * 5 // ~2000 words (assuming avg 5 chars/word)
		if len(embeddingText) > maxChars {
			embeddingText = embeddingText[:maxChars]
		}

		// Fallback to summary if article content is too short
		if len(embeddingText) < 200 {
			fmt.Printf("           ⚠️  Article content too short (%d chars), using summary\n", len(embeddingText))
			embeddingText = summary.SummaryText
		}

		embedding, err := p.embedder.GenerateEmbedding(ctx, embeddingText)
		if err != nil {
			// Log error but continue with other articles
			fmt.Printf("           ✗ Embedding generation failed: %v\n", err)
			failedCount++
			continue
		}

		embeddings[summary.ID] = embedding
		fmt.Printf("           ✓ Embedding generated (%d dimensions, %d chars)\n", len(embedding), len(embeddingText))
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("failed to generate any embeddings (all %d attempts failed)", failedCount)
	}

	if failedCount > 0 {
		fmt.Printf("   ⚠️  Warning: %d/%d embeddings failed to generate\n", failedCount, len(summaries))
	}

	return embeddings, nil
}

// generateClusterNarratives generates comprehensive narratives for each cluster using ALL articles
// This implements hierarchical summarization: cluster summary → executive summary
func (p *Pipeline) generateClusterNarratives(ctx context.Context, clusters []core.TopicCluster, articles []core.Article, summaries []core.Summary) ([]core.TopicCluster, error) {
	// Build maps for fast lookup
	articleMap := articlesToMap(articles)
	summaryMap := summariesToMap(summaries)

	// Define the interface we need from narrative generator
	type ClusterSummarizer interface {
		GenerateClusterSummary(ctx context.Context, cluster core.TopicCluster, articles map[string]core.Article, summaries map[string]core.Summary) (*core.ClusterNarrative, error)
	}

	summarizer, ok := p.narrative.(ClusterSummarizer)
	if !ok {
		return nil, fmt.Errorf("narrative generator does not support cluster summarization")
	}

	// Generate narrative for each cluster
	updatedClusters := make([]core.TopicCluster, 0, len(clusters))
	var failedCount int

	for i, cluster := range clusters {
		fmt.Printf("   [%d/%d] Generating narrative for cluster: %s (%d articles)\n",
			i+1, len(clusters), cluster.Label, len(cluster.ArticleIDs))

		narrative, err := summarizer.GenerateClusterSummary(ctx, cluster, articleMap, summaryMap)
		if err != nil {
			fmt.Printf("           ✗ Narrative generation failed: %v\n", err)
			failedCount++
			// Keep cluster without narrative
			updatedClusters = append(updatedClusters, cluster)
			continue
		}

		// Update cluster with generated narrative
		cluster.Narrative = narrative
		updatedClusters = append(updatedClusters, cluster)

		fmt.Printf("           ✓ Generated narrative: %s\n", narrative.Title)
		fmt.Printf("           ✓ Synthesized %d articles into %d words\n",
			len(narrative.ArticleRefs), len(narrative.Summary)/5)
	}

	if len(updatedClusters) == 0 {
		return nil, fmt.Errorf("failed to process any clusters")
	}

	if failedCount > 0 {
		fmt.Printf("   ⚠️  Warning: %d/%d cluster narratives failed to generate\n", failedCount, len(clusters))
	}

	return updatedClusters, nil
}

// categorizeArticles assigns categories to all articles using LLM categorization
func (p *Pipeline) categorizeArticles(ctx context.Context, articles []core.Article, summaries []core.Summary) ([]core.Article, error) {
	if p.categorizer == nil {
		return articles, fmt.Errorf("categorizer not available")
	}

	// Build a map of article ID to summary for faster lookup
	summaryMap := summariesToMap(summaries)

	categorizedArticles := make([]core.Article, 0, len(articles))
	var failedCount int

	for i, article := range articles {
		fmt.Printf("   [%d/%d] Categorizing: %s\n", i+1, len(articles), article.Title)

		// Get corresponding summary
		summary, hasSummary := summaryMap[article.ID]
		var summaryPtr *core.Summary
		if hasSummary {
			summaryPtr = &summary
		}

		// Categorize article
		category, err := p.categorizer.CategorizeArticle(ctx, &article, summaryPtr)
		if err != nil {
			// Log error but continue with default category
			fmt.Printf("           ✗ Categorization failed: %v (using 'Miscellaneous')\n", err)
			category = "Miscellaneous"
			failedCount++
		} else {
			fmt.Printf("           ✓ Category: %s\n", category)
		}

		// Update article with category
		article.Category = category
		categorizedArticles = append(categorizedArticles, article)
	}

	if failedCount > 0 {
		fmt.Printf("   ⚠️  Warning: %d/%d articles failed to categorize\n", failedCount, len(articles))
	}

	return categorizedArticles, nil
}

// buildDigest constructs the final digest structure
// Groups articles by category first, then by cluster theme within each category
func (p *Pipeline) buildDigest(clusters []core.TopicCluster, articles []core.Article, summaries []core.Summary, executiveSummary string) *core.Digest {
	digest := &core.Digest{
		ID:            generateID(),
		DigestSummary: executiveSummary,
		DateGenerated: time.Now(),
	}

	// Build a map of article IDs to full articles for quick lookup
	articleMap := articlesToMap(articles)

	// Group articles by category
	categoryGroups := make(map[string][]core.Article)
	articleURLs := make([]string, 0, len(articles))

	// First pass: organize articles by category
	for _, cluster := range clusters {
		for _, articleID := range cluster.ArticleIDs {
			if article, found := articleMap[articleID]; found {
				category := article.Category
				if category == "" {
					category = "Miscellaneous"
				}
				categoryGroups[category] = append(categoryGroups[category], article)
				articleURLs = append(articleURLs, article.URL)
			}
		}
	}

	// Build article groups by category
	articleGroups := make([]core.ArticleGroup, 0, len(categoryGroups))
	for category, categoryArticles := range categoryGroups {
		group := core.ArticleGroup{
			Category: category,
			Theme:    category, // Use category as theme for now
			Articles: categoryArticles,
			Priority: p.getCategoryPriority(category),
		}
		articleGroups = append(articleGroups, group)
	}

	// Sort groups by priority (lower number = higher priority)
	// This ensures Platform Updates comes before Miscellaneous, etc.
	for i := 0; i < len(articleGroups); i++ {
		for j := i + 1; j < len(articleGroups); j++ {
			if articleGroups[i].Priority > articleGroups[j].Priority {
				articleGroups[i], articleGroups[j] = articleGroups[j], articleGroups[i]
			}
		}
	}

	digest.ArticleGroups = articleGroups
	digest.ArticleURLs = articleURLs
	digest.Summaries = summaries // Store summaries for rendering

	// Set metadata (title and TL;DR will be generated later)
	digest.Metadata = core.DigestMetadata{
		Title:         "", // Will be generated by narrative generator
		TLDRSummary:   "", // Will be generated by narrative generator
		DateGenerated: time.Now(),
		ArticleCount:  len(articles),
	}

	return digest
}

// buildDigestForCluster builds a digest for a single cluster (v2.0)
// This method creates one focused digest per topic cluster
func (p *Pipeline) buildDigestForCluster(cluster core.TopicCluster, articles []core.Article, summaries []core.Summary) *core.Digest {
	digest := &core.Digest{
		ID:            generateID(),
		Title:         cluster.Label, // Will be enhanced by narrative generator
		ProcessedDate: time.Now(),
		ArticleCount:  len(articles),
		ClusterID:     nil, // TODO: Set when HDBSCAN is implemented (K-means has string IDs)
		Articles:      articles,
		Summaries:     summaries,
	}

	// Extract article URLs
	articleURLs := make([]string, 0, len(articles))
	for _, article := range articles {
		articleURLs = append(articleURLs, article.URL)
	}
	digest.ArticleURLs = articleURLs

	// Set metadata
	digest.Metadata = core.DigestMetadata{
		Title:         cluster.Label,
		DateGenerated: time.Now(),
		ArticleCount:  len(articles),
	}

	// Create a single ArticleGroup for this cluster (for backward compatibility)
	digest.ArticleGroups = []core.ArticleGroup{
		{
			Category: cluster.Label,
			Theme:    cluster.Label,
			Articles: articles,
			Summary:  "", // Will be generated
			Priority: 1,
		},
	}

	return digest
}

// getCategoryPriority returns the priority of a category for sorting
// Lower numbers appear first in the digest
func (p *Pipeline) getCategoryPriority(category string) int {
	// Map of category to priority (from categorization/categories.go default order)
	priorities := map[string]int{
		"Platform Updates": 1,
		"From the Field":   2,
		"Research":         3,
		"Tutorials":        4,
		"Analysis":         5,
		"Miscellaneous":    99,
	}

	if priority, found := priorities[category]; found {
		return priority
	}
	return 50 // Default priority for unknown categories
}

// generateExecutiveSummaryFromDigest generates executive summary using category-grouped articles
// This ensures article numbering in the prompt matches the final output
// generateDigestContentWithNarratives generates digest content using cluster narratives (hierarchical summarization)
// This is the NEW approach that uses cluster-level summaries instead of individual articles
// NOW WITH SELF-CRITIQUE: Always runs quality refinement pass for maximum quality
func (p *Pipeline) generateDigestContentWithNarratives(ctx context.Context, clusters []core.TopicCluster, articles []core.Article, summaries []core.Summary) (*narrative.DigestContent, error) {
	if len(clusters) == 0 {
		return nil, fmt.Errorf("no clusters provided")
	}

	// Build maps for narrative generator
	articleMap := articlesToMap(articles)
	summaryMap := summariesToMap(summaries)

	// Check if we have a narrative generator that supports cluster narratives with critique
	type ContentGeneratorWithCritique interface {
		GenerateDigestContentWithCritique(ctx context.Context, clusters []core.TopicCluster, articles map[string]core.Article, summaries map[string]core.Summary, config narrative.CritiqueConfig) (*narrative.DigestContent, error)
	}

	gen, ok := p.narrative.(ContentGeneratorWithCritique)
	if !ok {
		// Fallback to base generator without critique (backward compatibility)
		type ContentGenerator interface {
			GenerateDigestContent(ctx context.Context, clusters []core.TopicCluster, articles map[string]core.Article, summaries map[string]core.Summary) (*narrative.DigestContent, error)
		}

		baseGen, ok := p.narrative.(ContentGenerator)
		if !ok {
			return nil, fmt.Errorf("narrative generator does not support digest content generation")
		}

		fmt.Printf("   ⚠️  Using legacy generator without self-critique\n")
		return baseGen.GenerateDigestContent(ctx, clusters, articleMap, summaryMap)
	}

	// Use NEW generator with self-critique refinement pass
	// This ensures quality through always-on critique (signal over noise)
	critiqueConfig := narrative.DefaultCritiqueConfig()
	return gen.GenerateDigestContentWithCritique(ctx, clusters, articleMap, summaryMap, critiqueConfig)
}

// checkArticleCache checks if an article and its summary are cached
func (p *Pipeline) checkArticleCache(url string) (*core.Article, *core.Summary, error) {
	if p.cache == nil {
		return nil, nil, fmt.Errorf("cache not available")
	}
	return p.cache.GetArticleWithSummary(url, p.config.CacheTTL)
}

// cacheArticle stores an article and its summary in cache
func (p *Pipeline) cacheArticle(article *core.Article, summary *core.Summary) error {
	if p.cache == nil {
		return nil
	}
	return p.cache.StoreArticleWithSummary(article, summary, p.config.CacheTTL)
}

func (p *Pipeline) checkQuickReadCache(url string) (*core.Article, *core.Summary, error) {
	if p.cache == nil {
		return nil, nil, fmt.Errorf("cache not available")
	}
	return p.cache.GetArticleWithSummary(url, 24*time.Hour) // 24 hour TTL for quick reads
}

func (p *Pipeline) cacheQuickRead(article *core.Article, summary *core.Summary) error {
	if p.cache == nil {
		return nil
	}
	return p.cache.StoreArticleWithSummary(article, summary, 24*time.Hour)
}

// Helper functions

func articlesToMap(articles []core.Article) map[string]core.Article {
	result := make(map[string]core.Article)
	for _, article := range articles {
		result[article.ID] = article
	}
	return result
}

func summariesToMap(summaries []core.Summary) map[string]core.Summary {
	result := make(map[string]core.Summary)
	for _, summary := range summaries {
		// Map by article ID, not summary ID, for narrative generator
		// Summaries have ArticleIDs array, use first one
		if len(summary.ArticleIDs) > 0 {
			result[summary.ArticleIDs[0]] = summary
		}
	}
	return result
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// storeQualityMetrics calculates and stores quality metrics for a digest
func (p *Pipeline) storeQualityMetrics(ctx context.Context, digest *core.Digest, articles []core.Article) error {
	// Create quality evaluator
	evaluator := quality.NewDigestEvaluator()

	// Evaluate digest quality
	metrics := evaluator.EvaluateDigest(digest, articles)

	// Log quality metrics to console
	fmt.Printf("   📊 Quality Metrics:\n")
	fmt.Printf("      • Coverage: %.0f%% (%d/%d articles cited)\n",
		metrics.CoveragePct*100, metrics.CitationsFound, metrics.ArticleCount)
	fmt.Printf("      • Vague phrases: %d\n", metrics.VaguePhrases)
	fmt.Printf("      • Specificity: %d/100\n", metrics.SpecificityScore)
	fmt.Printf("      • Citation density: %.2f per 100 words\n", metrics.CitationDensity)
	fmt.Printf("      • Grade: %s\n", metrics.Grade)
	if metrics.Passed {
		fmt.Printf("      • Status: ✓ PASSED\n")
	} else {
		fmt.Printf("      • Status: ⚠️  NEEDS IMPROVEMENT\n")
		if len(metrics.Warnings) > 0 {
			fmt.Printf("      • Warnings:\n")
			for _, warning := range metrics.Warnings {
				fmt.Printf("        - %s\n", warning)
			}
		}
	}

	// TODO: Store metrics in database using quality_thresholds table
	// For now, metrics are logged to console for visibility

	return nil
}