package handlers

import (
	"briefly/internal/clustering"
	"briefly/internal/config"
	"briefly/internal/core"
	"briefly/internal/fetch"
	"briefly/internal/llm"
	"briefly/internal/logger"
	"briefly/internal/markdown"
	"briefly/internal/narrative"
	"briefly/internal/parser"
	"briefly/internal/store"
	"briefly/internal/summarize"
	"briefly/internal/themes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// NewDigestFromFileCmd creates the digest from-file command for processing curated markdown files
func NewDigestFromFileCmd() *cobra.Command {
	var (
		outputDir      string
		numClusters    int
		noCache        bool
		themeThreshold float64
	)

	cmd := &cobra.Command{
		Use:   "from-file <input.md>",
		Short: "Generate digest from curated markdown file",
		Long: `Generate a digest from a curated markdown file containing URLs.

This command (Phase 1.5 - Digest from File):
  • Parses URLs from a markdown file
  • Fetches articles (HTML, PDF, YouTube)
  • Generates summaries using LLM
  • Classifies articles by theme
  • Clusters articles by topic similarity
  • Creates hierarchical summaries (cluster narratives → executive summary)
  • Renders LinkedIn-ready markdown
  • No database persistence (lightweight, in-memory processing)

Perfect for weekly digests from manually curated URLs.

Examples:
  # Generate digest from curated file
  briefly digest from-file input/weekly.md

  # Custom output directory
  briefly digest from-file input/weekly.md --output my-digests

  # Disable caching (fresh fetch)
  briefly digest from-file input/weekly.md --no-cache

  # Specify number of clusters
  briefly digest from-file input/weekly.md --clusters 5`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDigestFromFile(cmd.Context(), args[0], outputDir, numClusters, noCache, themeThreshold)
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output", "o", "digests", "Output directory for digest file")
	cmd.Flags().IntVar(&numClusters, "clusters", 0, "Number of clusters (0 = auto-determine)")
	cmd.Flags().BoolVar(&noCache, "no-cache", false, "Disable caching (fetch fresh content)")
	cmd.Flags().Float64Var(&themeThreshold, "theme-threshold", 0.4, "Minimum theme relevance score (0.0-1.0)")

	return cmd
}

func runDigestFromFile(ctx context.Context, inputFile string, outputDir string, numClusters int, noCache bool, themeThreshold float64) error {
	startTime := time.Now()
	log := logger.Get()
	log.Info("Starting digest generation from file",
		"input_file", inputFile,
		"output_dir", outputDir,
		"clusters", numClusters,
		"no_cache", noCache,
	)

	// Load configuration
	_, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg := config.Get()

	// Validate input file
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file not found: %s", inputFile)
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Initialize LLM client
	modelName := cfg.AI.Gemini.Model
	if modelName == "" {
		modelName = "gemini-flash-lite-latest"
	}

	fmt.Printf("🔧 Initializing AI client (model: %s)...\n", modelName)
	llmClient, err := llm.NewClient(modelName)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}
	defer llmClient.Close()

	// Initialize cache (unless disabled)
	var cache *store.Store
	if !noCache {
		cacheDir := cfg.Cache.Directory
		if cacheDir == "" {
			cacheDir = ".briefly-cache"
		}
		cache, err = store.NewStore(cacheDir)
		if err != nil {
			log.Warn("Failed to initialize cache, continuing without cache", "error", err)
		} else {
			defer cache.Close()
			fmt.Println("   ✓ Cache initialized")
		}
	}

	// Step 1: Parse URLs from markdown file
	fmt.Printf("\n📄 Step 1/9: Parsing URLs from %s...\n", inputFile)
	urlParser := parser.NewParser()
	links, err := urlParser.ParseMarkdownFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to parse markdown file: %w", err)
	}

	if len(links) == 0 {
		fmt.Println("⚠️  No URLs found in markdown file")
		return nil
	}

	fmt.Printf("   ✓ Found %d URLs\n", len(links))

	// Step 2: Fetch articles
	fmt.Printf("\n🔍 Step 2/9: Fetching and processing articles...\n")
	processor := fetch.NewContentProcessor()
	articles := make([]core.Article, 0, len(links))

	for i, link := range links {
		fmt.Printf("   [%d/%d] Fetching: %s\n", i+1, len(links), link.URL)

		// Check cache first
		var article *core.Article
		if cache != nil {
			cachedArticle, err := cache.GetCachedArticle(link.URL, 24*time.Hour)
			if err == nil && cachedArticle != nil {
				article = cachedArticle
				fmt.Println("           ✓ Cache hit")
			}
		}

		// Fetch if not cached
		if article == nil {
			fetchedArticle, err := processor.ProcessArticle(ctx, link.URL)
			if err != nil {
				log.Warn("Failed to fetch article", "url", link.URL, "error", err)
				fmt.Printf("           ⚠ Fetch failed: %v\n", err)
				continue
			}
			article = fetchedArticle

			// Save to cache
			if cache != nil {
				if err := cache.SaveArticle(article); err != nil {
					log.Warn("Failed to cache article", "url", link.URL, "error", err)
				}
			}
			fmt.Println("           ✓ Fetched and processed")
		}

		articles = append(articles, *article)
	}

	if len(articles) == 0 {
		fmt.Println("\n⚠️  No articles could be fetched")
		return nil
	}

	fmt.Printf("   ✓ Successfully fetched %d/%d articles\n", len(articles), len(links))

	// Step 3: Generate summaries
	fmt.Printf("\n📝 Step 3/9: Generating article summaries...\n")
	adapter := &llmClientAdapter{client: llmClient}
	summarizer := summarize.NewSummarizerWithDefaults(adapter)

	articleSummaries := make(map[string]*core.Summary)
	summaryList := make([]core.Summary, 0, len(articles))

	for i, article := range articles {
		fmt.Printf("   [%d/%d] Summarizing: %s\n", i+1, len(articles), article.Title)

		// Generate summary (cache lookup is complex, skip for now)
		summary, err := summarizer.SummarizeArticle(ctx, &article)
		if err != nil {
			log.Warn("Failed to generate summary", "article_id", article.ID, "error", err)
			// Create fallback summary
			summary = &core.Summary{
				ID:          uuid.NewString(),
				ArticleIDs:  []string{article.ID},
				SummaryText: fmt.Sprintf("Summary for: %s", article.Title),
				ModelUsed:   "fallback",
			}
		}
		fmt.Println("           ✓ Generated")

		articleSummaries[article.ID] = summary
		summaryList = append(summaryList, *summary)
	}

	// Step 4: Classify articles by theme
	fmt.Printf("\n🏷️  Step 4/9: Classifying articles by theme...\n")

	// Load themes (we'll use hardcoded defaults for file-based mode)
	defaultThemes := []core.Theme{
		{ID: uuid.NewString(), Name: "AI & Machine Learning", Keywords: []string{"ai", "machine learning", "llm", "gpt"}},
		{ID: uuid.NewString(), Name: "Cloud & DevOps", Keywords: []string{"cloud", "kubernetes", "docker", "devops"}},
		{ID: uuid.NewString(), Name: "Software Engineering", Keywords: []string{"programming", "software", "development", "code"}},
		{ID: uuid.NewString(), Name: "Web Development", Keywords: []string{"web", "javascript", "react", "frontend"}},
		{ID: uuid.NewString(), Name: "Data & Analytics", Keywords: []string{"data", "analytics", "database", "sql"}},
	}

	themeClassifier := themes.NewClassifier(llmClient, nil) // Pass nil for PostHog (lightweight mode)

	for i := range articles {
		fmt.Printf("   [%d/%d] Classifying: %s\n", i+1, len(articles), articles[i].Title)

		classification, err := themeClassifier.GetBestMatch(ctx, articles[i], defaultThemes, themeThreshold)
		if err != nil {
			log.Warn("Failed to classify article", "article_id", articles[i].ID, "error", err)
			fmt.Println("           ⚠ Classification failed")
			continue
		}

		if classification != nil {
			articles[i].ThemeID = &classification.ThemeID
			fmt.Printf("           ✓ Theme: %s (score: %.2f)\n", classification.ThemeName, classification.RelevanceScore)
		} else {
			fmt.Println("           ⚠ No theme match above threshold")
		}
	}

	// Step 5: Generate embeddings
	fmt.Printf("\n🧠 Step 5/9: Generating embeddings for clustering...\n")
	embeddingsMap := make(map[string][]float64)

	for i, article := range articles {
		summary, hasSummary := articleSummaries[article.ID]
		textForEmbedding := article.CleanedText
		if hasSummary {
			textForEmbedding = summary.SummaryText
		}

		// Truncate if too long
		if len(textForEmbedding) > 2000 {
			textForEmbedding = textForEmbedding[:2000]
		}

		fmt.Printf("   [%d/%d] Embedding: %s\n", i+1, len(articles), article.Title)

		embedding, err := llmClient.GenerateEmbedding(textForEmbedding)
		if err != nil {
			log.Warn("Failed to generate embedding", "article_id", article.ID, "error", err)
			fmt.Println("           ⚠ Failed")
			continue
		}

		embeddingsMap[article.ID] = embedding
		articles[i].Embedding = embedding
		fmt.Printf("           ✓ Generated (%d dimensions)\n", len(embedding))
	}

	// Step 6: Cluster articles
	fmt.Printf("\n🔍 Step 6/9: Clustering articles by topic...\n")

	// Auto-determine clusters if not specified
	if numClusters == 0 {
		numClusters = (len(articles) + 4) / 5 // ~5 articles per cluster
		if numClusters < 3 {
			numClusters = 3
		}
		if numClusters > 15 {
			numClusters = 15
		}
	}

	fmt.Printf("   🔍 Clustering %d articles into ~%d topics (K-means++ with cosine distance)...\n", len(articles), numClusters)

	clusterer := clustering.NewKMeansClusterer()
	clusters, err := clusterer.Cluster(articles, numClusters)
	if err != nil {
		return fmt.Errorf("failed to cluster articles: %w", err)
	}

	if len(clusters) == 0 {
		return fmt.Errorf("no clusters found")
	}

	fmt.Printf("   ✓ Found %d topic clusters\n", len(clusters))
	for i, cluster := range clusters {
		fmt.Printf("      %d. %s (%d articles)\n", i+1, cluster.Label, len(cluster.ArticleIDs))
	}

	// Create article and summary maps
	articleMap := make(map[string]core.Article)
	summaryMap := make(map[string]core.Summary)
	for i, article := range articles {
		articleMap[article.ID] = articles[i]
	}
	for _, summary := range summaryList {
		for _, articleID := range summary.ArticleIDs {
			summaryMap[articleID] = summary
		}
	}

	// Step 7: Generate cluster narratives (hierarchical stage 1)
	fmt.Printf("\n📖 Step 7/9: Generating cluster narratives from ALL articles...\n")
	narrativeAdapter := &narrativeLLMAdapter{client: llmClient}
	narrativeGen := narrative.NewGenerator(narrativeAdapter)

	for i, cluster := range clusters {
		if len(cluster.ArticleIDs) == 0 {
			continue
		}

		fmt.Printf("   [%d/%d] Cluster: %s (%d articles)\n", i+1, len(clusters), cluster.Label, len(cluster.ArticleIDs))

		clusterNarrative, err := narrativeGen.GenerateClusterSummary(ctx, cluster, articleMap, summaryMap)
		if err != nil {
			log.Warn("Failed to generate cluster narrative", "cluster", cluster.Label, "error", err)
			fmt.Println("           ⚠ Narrative generation failed")
			continue
		}

		clusters[i].Narrative = clusterNarrative
		fmt.Printf("   ✓ Generated: %s (%d words)\n", clusterNarrative.Title, len(clusterNarrative.Summary)/5)
	}

	// Step 8: Generate unified executive summary from ALL cluster narratives
	fmt.Printf("\n✨ Step 8/9: Generating unified executive summary from all clusters...\n")

	// Generate ONE digest content from ALL clusters (hierarchical summarization)
	critiqueConfig := narrative.DefaultCritiqueConfig()
	digestContent, err := narrativeGen.GenerateDigestContentWithCritique(ctx, clusters, articleMap, summaryMap, critiqueConfig)
	if err != nil {
		log.Warn("Failed to generate unified digest content", "error", err)
		// Use fallback
		digestContent = &narrative.DigestContent{
			Title:            fmt.Sprintf("Weekly Tech Digest - %d Articles", len(articles)),
			TLDRSummary:      fmt.Sprintf("Digest covering %d articles across %d topics", len(articles), len(clusters)),
			KeyMoments:       []core.KeyMoment{},
			Perspectives:     []core.Perspective{},
			ExecutiveSummary: "This digest covers recent developments in technology.",
		}
	}

	fmt.Printf("   ✓ Generated unified digest: %s\n", digestContent.Title)

	// Build article groups organized by cluster
	articleGroups := make([]core.ArticleGroup, 0, len(clusters))
	for _, cluster := range clusters {
		if len(cluster.ArticleIDs) == 0 {
			continue
		}

		// Build article list for this cluster
		clusterArticles := make([]core.Article, 0, len(cluster.ArticleIDs))
		for _, articleID := range cluster.ArticleIDs {
			if article, found := articleMap[articleID]; found {
				clusterArticles = append(clusterArticles, article)
			}
		}

		// Get theme/cluster name
		themeName := cluster.Label
		if cluster.Narrative != nil && cluster.Narrative.Title != "" {
			themeName = cluster.Narrative.Title
		}

		// Use cluster narrative as the summary
		clusterSummary := ""
		if cluster.Narrative != nil {
			clusterSummary = cluster.Narrative.Summary
		}

		articleGroups = append(articleGroups, core.ArticleGroup{
			Theme:            themeName,
			Articles:         clusterArticles,
			Summary:          clusterSummary,
			ClusterNarrative: cluster.Narrative, // NEW v3.1: Include cluster narrative for bullet rendering
			Category:         themeName,
		})
	}

	// Inject citations into executive summary
	summaryWithCitations := markdown.InjectCitationURLs(digestContent.ExecutiveSummary, articles)

	now := time.Now()

	// Create ONE unified digest with all articles
	digest := &core.Digest{
		ID:            uuid.NewString(),
		Title:         digestContent.Title,
		Summary:       summaryWithCitations,
		TLDRSummary:   digestContent.TLDRSummary,
		KeyMoments:    digestContent.KeyMoments,
		Perspectives:  digestContent.Perspectives,
		Articles:      articles,
		ProcessedDate: now,
		ArticleCount:  len(articles),

		// v3.0 scannable format fields (NEW)
		TopDevelopments: digestContent.TopDevelopments,
		ByTheNumbers:    convertStatistics(digestContent.ByTheNumbers),
		WhyItMatters:    digestContent.WhyItMatters,

		ArticleGroups: articleGroups,
		DigestSummary: digestContent.ExecutiveSummary,
		Metadata: core.DigestMetadata{
			Title:         digestContent.Title,
			ArticleCount:  len(articles),
			DateGenerated: now,
			TLDRSummary:   digestContent.TLDRSummary,
		},
	}

	// Step 9: Render unified markdown file
	fmt.Printf("\n📄 Step 9/9: Rendering unified markdown digest...\n")

	outputPath, err := saveDigestMarkdown(digest, outputDir)
	if err != nil {
		return fmt.Errorf("failed to save digest markdown: %w", err)
	}

	fmt.Printf("   ✓ Saved: %s\n", outputPath)

	duration := time.Since(startTime)

	// Print summary
	fmt.Printf("\n✅ Successfully generated unified digest!\n")
	fmt.Printf("   Title: %s\n", digest.Title)
	fmt.Printf("   Input file: %s\n", inputFile)
	fmt.Printf("   Total URLs: %d\n", len(links))
	fmt.Printf("   Articles fetched: %d\n", len(articles))
	fmt.Printf("   Topic clusters: %d\n", len(clusters))
	fmt.Printf("   Output file: %s\n", outputPath)
	fmt.Printf("   Duration: %s\n", duration.Round(time.Millisecond))

	// Show cluster breakdown
	fmt.Println("\n📊 Cluster Breakdown:")
	for i, group := range articleGroups {
		fmt.Printf("   %d. %s (%d articles)\n", i+1, group.Theme, len(group.Articles))
	}

	fmt.Println("\n💡 Next steps:")
	fmt.Println("   • Review the digest:", outputPath)
	fmt.Println("   • Edit and refine as needed")
	fmt.Println("   • Share on LinkedIn or your preferred platform")

	return nil
}
