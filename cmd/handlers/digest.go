package handlers

import (
	"briefly/internal/alerts"
	"briefly/internal/categorization"
	"briefly/internal/clustering"
	"briefly/internal/config"
	"briefly/internal/core"
	"briefly/internal/cost"
	"briefly/internal/fetch"
	"briefly/internal/llm"
	"briefly/internal/logger"
	"briefly/internal/messaging"
	"briefly/internal/relevance"
	"briefly/internal/render"
	"briefly/internal/sentiment"
	"briefly/internal/services"
	"briefly/internal/store"
	"briefly/internal/templates"
	"briefly/internal/tts"
	"briefly/internal/visual"
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewDigestCmd creates the consolidated digest command with all format options
func NewDigestCmd() *cobra.Command {
	digestCmd := &cobra.Command{
		Use:   "digest [input-file|URL]",
		Short: "Generate digest with Smart Headlines from URLs or single article",
		Long: `Process URLs from a markdown file or summarize a single URL, with multiple output formats.

Features:
- Smart Headlines: Automatically generates compelling, content-based titles
- AI-powered insights: Sentiment analysis, alerts, and trend analysis  
- AI banner images: Generate visual banners using DALL-E for enhanced presentation
- Multiple formats: brief, standard, detailed, newsletter, scannable, email, slack, discord, audio, condensed
- Interactive my-take workflow: Add personal commentary
- Single article mode: Process just one URL

Examples:
  # Standard digest generation
  briefly digest input/links.md
  briefly digest --format newsletter --output digests input/links.md
  
  # Generate digest with AI banner image
  briefly digest --with-banner --format newsletter input/links.md
  
  # Single article processing  
  briefly digest --single https://example.com/article
  
  # Multi-channel outputs
  briefly digest --format slack input/links.md
  briefly digest --format discord input/links.md  
  briefly digest --format audio input/links.md
  
  # Interactive workflow
  briefly digest --interactive input/links.md
  
  # Cost estimation
  briefly digest --dry-run input/links.md
  
  # List available formats
  briefly digest --list-formats`,
		Run: digestRunFunc,
	}

	// Add flags
	digestCmd.Flags().StringP("output", "o", "digests", "Output directory for digest file")
	digestCmd.Flags().StringP("format", "f", "standard", "Digest format: brief, standard, detailed, newsletter, scannable, email, slack, discord, audio, condensed")
	digestCmd.Flags().Bool("dry-run", false, "Estimate costs without making API calls")
	digestCmd.Flags().Bool("list-formats", false, "List available output formats")
	digestCmd.Flags().Bool("single", false, "Process single URL instead of input file")
	digestCmd.Flags().Bool("interactive", false, "Interactive my-take workflow")
	digestCmd.Flags().Bool("with-banner", false, "Generate AI banner image using DALL-E")
	digestCmd.Flags().String("style-guide", "", "Path to personal style guide file")

	// Messaging platform flags
	digestCmd.Flags().String("slack-webhook", "", "Slack webhook URL for slack format")
	digestCmd.Flags().String("discord-webhook", "", "Discord webhook URL for discord format")
	digestCmd.Flags().String("messaging-format", "bullets", "Messaging format: bullets, summary, highlights")

	// TTS flags
	digestCmd.Flags().String("tts-provider", "openai", "TTS provider: openai, elevenlabs, google, mock")
	digestCmd.Flags().String("tts-voice", "alloy", "Voice for TTS generation")
	digestCmd.Flags().Float32("tts-speed", 1.0, "Speech speed (0.25-4.0)")
	digestCmd.Flags().Int("tts-max-articles", 10, "Maximum articles for TTS")

	// v2.0 Relevance filtering flags
	digestCmd.Flags().Float64("min-relevance", 0.4, "Minimum relevance threshold for article inclusion (0.0-1.0)")
	digestCmd.Flags().Int("max-words", 0, "Maximum words for entire digest (0 for template default)")
	digestCmd.Flags().Bool("enable-filtering", true, "Enable relevance-based content filtering")

	return digestCmd
}

func digestRunFunc(cmd *cobra.Command, args []string) {
	// Handle list-formats flag
	if listFormats, _ := cmd.Flags().GetBool("list-formats"); listFormats {
		printAvailableFormats()
		return
	}

	// Validate arguments
	single, _ := cmd.Flags().GetBool("single")
	if single {
		if len(args) != 1 {
			fmt.Fprintf(os.Stderr, "Error: single mode requires exactly one URL argument\n")
			os.Exit(1)
		}
		if err := runSingleArticle(cmd, args[0]); err != nil {
			logger.Error("Failed to process single article", err)
			os.Exit(1)
		}
		return
	}

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "Error: digest command requires exactly one input file argument\n")
		os.Exit(1)
	}

	inputFile := args[0]
	outputDir, _ := cmd.Flags().GetString("output")
	format, _ := cmd.Flags().GetString("format")

	// If no output directory specified via flag, try config then default
	if outputDir == "digests" { // Default value
		configOutputDir := viper.GetString("output.directory")
		if configOutputDir != "" {
			outputDir = configOutputDir
		}
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	interactive, _ := cmd.Flags().GetBool("interactive")
	withBanner, _ := cmd.Flags().GetBool("with-banner")

	if err := runDigest(cmd, inputFile, outputDir, format, dryRun, interactive, withBanner); err != nil {
		logger.Error("Failed to generate digest", err)
		os.Exit(1)
	}
}

func printAvailableFormats() {
	fmt.Println("Available digest formats:")
	fmt.Println("  brief      - Short summaries with key points")
	fmt.Println("  standard   - Standard digest with full summaries")
	fmt.Println("  detailed   - Detailed digest with analysis")
	fmt.Println("  newsletter - Newsletter format with executive summary")
	fmt.Println("  condensed  - Bite-size format for 30-second reading")
	fmt.Println("  email      - HTML email format")
	fmt.Println("  slack      - Slack messaging format")
	fmt.Println("  discord    - Discord messaging format")
	fmt.Println("  audio      - Text-to-Speech audio generation")
}

func runSingleArticle(cmd *cobra.Command, url string) error {
	logger.Info("Processing single article", "url", url)
	fmt.Printf("Processing single article: %s\n", url)

	// Initialize LLM client
	llmClient, err := llm.NewClient("")
	if err != nil {
		return fmt.Errorf("failed to initialize LLM client: %w", err)
	}
	defer llmClient.Close()

	// Create content processor
	processor := fetch.NewContentProcessor()

	// Process article using proper content type detection (handles YouTube, PDF, HTML)
	article, err := processor.ProcessArticle(cmd.Context(), url)
	if err != nil {
		return fmt.Errorf("failed to process article: %w", err)
	}

	// Get format for summarization
	format, _ := cmd.Flags().GetString("format")
	if format == "slack" || format == "discord" || format == "audio" {
		format = "standard" // Use standard format for LLM processing
	}

	// Generate summary
	// Use scannable format for signal digests to get concise 20-40 word summaries
	summaryFormat := format
	if format == "signal" {
		summaryFormat = "scannable"
	}
	summary, err := llmClient.SummarizeArticleTextWithFormat(*article, summaryFormat)
	if err != nil {
		return fmt.Errorf("failed to summarize article: %w", err)
	}

	// Output to terminal
	fmt.Printf("\n✅ %s\n\n", article.Title)
	fmt.Printf("Summary: %s\n\n", summary.SummaryText)
	fmt.Printf("Source: %s\n", url)

	return nil
}

func runDigest(cmd *cobra.Command, inputFile, outputDir, format string, dryRun, interactive, withBanner bool) error {
	logger.Info("Starting digest generation", "input_file", inputFile, "format", format, "dry_run", dryRun, "interactive", interactive)

	// Read links from input file
	links, err := fetch.ReadLinksFromFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read links from file: %w", err)
	}

	logger.Info("Found links", "count", len(links))
	if len(links) == 0 {
		logger.Warn("No valid links found in input file")
		return nil
	}

	if dryRun {
		return runCostEstimation(links)
	}

	// Process articles and generate digest with relevance filtering
	digestItems, processedArticles, err := processArticles(cmd, links, format)
	if err != nil {
		return fmt.Errorf("failed to process articles: %w", err)
	}

	if len(digestItems) == 0 {
		logger.Warn("No articles were successfully processed")
		fmt.Println("⚠️  No articles were successfully processed")
		return nil
	}

	// Generate insights
	insightsData, err := generateInsights(processedArticles)
	if err != nil {
		logger.Warn("Failed to generate insights", "error", err)
	}

	// Load style guide if specified
	styleGuide, err := loadStyleGuide(cmd)
	if err != nil {
		logger.Warn("Failed to load style guide", "error", err)
	}

	// Generate final digest based on format
	digestPath, err := generateOutput(cmd, digestItems, processedArticles, insightsData, outputDir, format, styleGuide, withBanner, interactive)
	if err != nil {
		return fmt.Errorf("failed to generate output: %w", err)
	}

	// Handle interactive my-take workflow
	if interactive && digestPath != "" {
		return handleInteractiveMyTake(digestPath, styleGuide)
	}

	fmt.Printf("✅ %s digest generated: %s\n", format, digestPath)
	return nil
}

func runCostEstimation(links []core.Link) error {
	logger.Info("Dry run mode - performing cost estimation", "links_count", len(links))

	model := viper.GetString("gemini.model")
	if model == "" {
		model = "gemini-2.5-flash-preview-05-20"
	}

	estimate, err := cost.EstimateDigestCost(links, model)
	if err != nil {
		return fmt.Errorf("failed to estimate costs: %w", err)
	}

	fmt.Print(estimate.FormatEstimate())
	return nil
}

func processArticles(cmd *cobra.Command, links []core.Link, format string) ([]render.DigestData, []core.Article, error) {
	// Initialize cache store
	cacheStore, err := store.NewStore(".briefly-cache")
	if err != nil {
		logger.Error("Failed to initialize cache store", err)
		fmt.Printf("⚠️  Cache disabled due to error: %s\n", err)
		cacheStore = nil
	} else {
		defer func() {
			if err := cacheStore.Close(); err != nil {
				logger.Error("Failed to close cache store", err)
			}
		}()
		logger.Info("Cache store initialized")
	}

	// Initialize LLM client
	llmClient, err := llm.NewClient("")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize LLM client: %w", err)
	}
	defer llmClient.Close()

	var digestItems []render.DigestData
	var processedArticles []core.Article
	cacheHits := 0
	cacheMisses := 0

	// Process each link
	for i, link := range links {
		logger.Info("Processing link", "index", i+1, "total", len(links), "url", link.URL)
		fmt.Printf("Processing %d/%d: %s\n", i+1, len(links), link.URL)

		var article core.Article
		var summary core.Summary
		var usedCache bool

		// Try to get article from cache first
		if cacheStore != nil {
			cachedArticle, err := cacheStore.GetCachedArticle(link.URL, 24*time.Hour)
			if err != nil {
				logger.Error("Cache lookup error", err)
			} else if cachedArticle != nil {
				article = *cachedArticle
				usedCache = true
				cacheHits++
				fmt.Printf("📦 Using cached article: %s\n", cachedArticle.Title)
			}
		}

		// Fetch and process article if not in cache
		if !usedCache {
			// Use new ContentProcessor for multi-format support
			processor := fetch.NewContentProcessor()
			ctx := context.Background()
			fetchedArticle, err := processor.ProcessArticle(ctx, link.URL)
			if err != nil {
				logger.Error("Failed to process content", err, "url", link.URL)
				fmt.Printf("❌ Failed to process: %s\n", link.URL)
				continue
			}
			article = *fetchedArticle
			cacheMisses++

			// Cache the cleaned article
			if cacheStore != nil {
				article.ID = uuid.NewString()
				article.LinkID = link.URL
				article.DateFetched = time.Now().UTC()

				if err := cacheStore.CacheArticle(article); err != nil {
					logger.Error("Failed to cache article", err)
				}
			}
		}

		// Check for cached summary
		contentHash := fmt.Sprintf("%d-%s-%s", len(article.CleanedText), link.URL, format)
		var summaryFromCache bool

		if cacheStore != nil && usedCache {
			cachedSummary, err := cacheStore.GetCachedSummary(link.URL, contentHash, 7*24*time.Hour)
			if err != nil {
				logger.Error("Summary cache lookup error", err)
			} else if cachedSummary != nil {
				summary = *cachedSummary
				summaryFromCache = true
				fmt.Printf("📦 Using cached summary\n")
			}
		}

		// Generate summary if not cached
		if !summaryFromCache {
			// Use scannable format for signal digests to get concise 20-40 word summaries
			summaryFormat := format
			if format == "signal" {
				summaryFormat = "scannable"
			}
			generatedSummary, err := llmClient.SummarizeArticleTextWithFormat(article, summaryFormat)
			if err != nil {
				logger.Error("Failed to summarize article", err, "url", link.URL)
				fmt.Printf("❌ Failed to summarize: %s\n", link.URL)
				continue
			}
			summary = generatedSummary
			summary.ID = uuid.NewString()
			summary.DateGenerated = time.Now().UTC()

			// Cache the summary
			if cacheStore != nil {
				if err := cacheStore.CacheSummary(summary, link.URL, contentHash); err != nil {
					logger.Error("Failed to cache summary", err)
				}
			}
		}

		// Create digest item with content type information
		digestItem := render.DigestData{
			Title:           article.Title,
			URL:             link.URL,
			SummaryText:     summary.SummaryText,
			MyTake:          article.MyTake,
			TopicCluster:    "",  // Will be populated after clustering
			TopicConfidence: 0.0, // Will be populated after clustering
			// Multi-format content support
			ContentType:  string(article.ContentType),
			ContentIcon:  fetch.GetContentTypeIcon(article.ContentType),
			ContentLabel: fetch.GetContentTypeLabel(article.ContentType),
			Duration:     article.Duration,
			Channel:      article.Channel,
			FileSize:     article.FileSize,
			PageCount:    article.PageCount,
		}

		digestItems = append(digestItems, digestItem)
		processedArticles = append(processedArticles, article)
		logger.Info("Successfully processed article", "title", article.Title)
		fmt.Printf("✅ %s\n", article.Title)
	}

	// Display cache statistics
	if cacheStore != nil {
		fmt.Printf("\n📊 Cache Statistics: %d hits, %d misses (%.1f%% hit rate)\n",
			cacheHits, cacheMisses, float64(cacheHits)/float64(cacheHits+cacheMisses)*100)
	}

	// Step 2: Perform clustering first (before filtering)
	if len(digestItems) > 1 && llmClient != nil {
		if err := performClustering(digestItems, processedArticles, llmClient, cacheStore); err != nil {
			logger.Warn("Failed to perform clustering", "error", err)
		}
	}

	// Step 2.5: Categorize and sort articles (for scannable format and better organization)
	if format == "scannable" || format == "newsletter" {
		fmt.Println("\n🏷️ Categorizing articles for enhanced organization...")
		categorizationService := categorization.NewService(llmClient)
		categorizedItems, err := categorizationService.CategorizeArticles(context.Background(), digestItems, processedArticles)
		if err != nil {
			logger.Warn("Failed to categorize articles", "error", err)
		} else {
			// Sort by category priority and relevance
			sortedItems := categorization.SortCategorizedItems(categorizedItems)
			
			// Replace digest items and articles with sorted categorized versions
			if len(sortedItems) == len(digestItems) {
				// Update all items with category information
				for i, catItem := range sortedItems {
					digestItems[i] = catItem.DigestItem
					processedArticles[i] = catItem.Article
					// Store category info in MyTake for now (can be enhanced later)
					categoryInfo := fmt.Sprintf("%s %s", catItem.Category.Category.Emoji, catItem.Category.Category.Name)
					if digestItems[i].MyTake != "" {
						digestItems[i].MyTake = fmt.Sprintf("%s | %s", categoryInfo, digestItems[i].MyTake)
					} else {
						digestItems[i].MyTake = categoryInfo
					}
				}
				fmt.Printf("   ✅ Categorized and sorted %d articles across %d categories\n", 
					len(sortedItems), len(categorization.Categories))
			} else {
				logger.Warn("Categorization result count mismatch, skipping category sorting", 
					"expected", len(digestItems), "got", len(sortedItems))
			}
		}
	}

	// Step 3: Generate team-relevant "Why it matters" insights
	fmt.Println("\n💡 Generating team-relevant insights...")
	teamContext := config.GenerateTeamContextPrompt()
	if teamContext != "" && llmClient != nil {
		insights, err := llmClient.GenerateWhyItMatters(processedArticles, teamContext)
		if err != nil {
			logger.Warn("Failed to generate why it matters insights", "error", err)
		} else {
			// Apply insights to digest items while preserving category information
			for i := range digestItems {
				if i < len(processedArticles) {
					if insight, exists := insights[processedArticles[i].ID]; exists {
						// Preserve category information if present
						if digestItems[i].MyTake != "" && strings.Contains(digestItems[i].MyTake, " | ") {
							// Category info exists, append the insight
							digestItems[i].MyTake = fmt.Sprintf("%s | %s", digestItems[i].MyTake, insight)
						} else if digestItems[i].MyTake != "" && (strings.Contains(digestItems[i].MyTake, "🔥") || strings.Contains(digestItems[i].MyTake, "🚀") || strings.Contains(digestItems[i].MyTake, "🛠️") || strings.Contains(digestItems[i].MyTake, "📊") || strings.Contains(digestItems[i].MyTake, "💡") || strings.Contains(digestItems[i].MyTake, "🔍")) {
							// Category info exists without separator, add separator
							digestItems[i].MyTake = fmt.Sprintf("%s | %s", digestItems[i].MyTake, insight)
						} else {
							// No category info, just set the insight
							digestItems[i].MyTake = insight
						}
						fmt.Printf("   ✅ Generated insight for: %s\n", digestItems[i].Title)
					}
				}
			}
			fmt.Printf("   📝 Generated %d team-relevant insights\n", len(insights))
		}
	}

	// Step 4: Enhanced relevance filtering (now with team context)
	enableFiltering := config.IsFilteringEnabled()
	if cmd.Flags().Changed("enable-filtering") {
		if flagVal, err := cmd.Flags().GetBool("enable-filtering"); err == nil {
			enableFiltering = flagVal
		}
	}
	if enableFiltering && len(digestItems) > 0 {
		originalCount := len(digestItems)
		filteredItems, filteredArticles, err := applyEnhancedRelevanceFiltering(cmd, digestItems, processedArticles, format, teamContext, llmClient)
		if err != nil {
			logger.Warn("Enhanced relevance filtering failed, falling back to basic filtering", "error", err)
			// Fallback to original filtering
			filteredItems, filteredArticles, err = applyRelevanceFiltering(cmd, digestItems, processedArticles, format)
			if err != nil {
				logger.Warn("Relevance filtering failed completely, continuing with all articles", "error", err)
			}
		}

		if err == nil {
			digestItems = filteredItems
			processedArticles = filteredArticles
			fmt.Printf("\n🎯 Enhanced Relevance Filtering: %d → %d articles (%.1f%% included)\n",
				originalCount, len(digestItems), float64(len(digestItems))/float64(originalCount)*100)
		}
	}

	return digestItems, processedArticles, nil
}

// applyRelevanceFiltering applies relevance scoring and filtering to articles
func applyRelevanceFiltering(cmd *cobra.Command, digestItems []render.DigestData, processedArticles []core.Article, format string) ([]render.DigestData, []core.Article, error) {
	fmt.Println("\n🎯 Applying relevance filtering...")

	// Get filtering parameters from configuration with command flag overrides
	minRelevance := config.GetFilteringMinRelevance()
	if cmd.Flags().Changed("min-relevance") {
		if flagVal, err := cmd.Flags().GetFloat64("min-relevance"); err == nil {
			minRelevance = flagVal
		}
	}

	// Get template-specific filtering settings
	templateFilter := config.GetTemplateFilter(format)

	// Use template-specific min-relevance if not overridden by command flag
	if !cmd.Flags().Changed("min-relevance") && templateFilter.MinRelevance > 0 {
		minRelevance = templateFilter.MinRelevance
	}

	// Get max words from template config with command flag override
	maxWords := templateFilter.MaxWords
	if cmd.Flags().Changed("max-words") {
		if flagVal, err := cmd.Flags().GetInt("max-words"); err == nil {
			maxWords = flagVal
		}
	}

	// Create relevance scorer
	scorer := relevance.NewKeywordScorer()

	// Infer digest theme from articles
	var scorableContents []relevance.Scorable
	for i, item := range digestItems {
		metadata := make(map[string]interface{})
		if i < len(processedArticles) {
			// Add article metadata
			if !processedArticles[i].DateFetched.IsZero() {
				metadata["date_published"] = processedArticles[i].DateFetched
			}
			metadata["content_type"] = processedArticles[i].ContentType
			metadata["content_length"] = len(processedArticles[i].CleanedText)
		}

		adapter := relevance.ArticleAdapter{
			Title:    item.Title,
			Content:  item.SummaryText, // Use summary for relevance scoring to be consistent
			URL:      item.URL,
			Metadata: metadata,
		}
		scorableContents = append(scorableContents, adapter)
	}

	// Infer the main theme for the digest
	digestTheme := relevance.InferDigestTheme(scorableContents)
	fmt.Printf("   📝 Detected theme: %s\n", digestTheme)

	// Create criteria for relevance scoring
	criteria := relevance.DefaultCriteria("digest", digestTheme)
	criteria.Threshold = minRelevance

	// Apply configured scoring weights
	configWeights := config.GetFilteringWeights()
	criteria.Weights = relevance.ScoringWeights{
		ContentRelevance: configWeights.ContentRelevance,
		TitleRelevance:   configWeights.TitleRelevance,
		Authority:        configWeights.Authority,
		Recency:          configWeights.Recency,
		Quality:          configWeights.Quality,
	}

	// Apply basic quality filters
	commonFilters := relevance.CommonFilters()
	criteria.Filters = []relevance.Filter{
		commonFilters["min_content_length"],
		commonFilters["has_title"],
		commonFilters["valid_url"],
	}

	// Use template's max words if flag not set
	if maxWords == 0 {
		template := templates.GetTemplate(templates.DigestFormat(format))
		if template != nil {
			maxWords = template.MaxDigestWords
		}
	}

	ctx := context.Background()

	// Apply filtering with word budget considerations
	var filterResults []relevance.FilterResult
	var err error
	if maxWords > 0 {
		filterResults, err = relevance.FilterForDigest(ctx, scorer, scorableContents, criteria, maxWords)
	} else {
		filterResults, err = relevance.FilterByThreshold(ctx, scorer, scorableContents, criteria)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("relevance filtering failed: %w", err)
	}

	// Get statistics
	stats := relevance.GetFilterStats(filterResults, minRelevance)
	fmt.Printf("   📊 Filter stats: %.2f avg score, %.2f max, %.2f min\n",
		stats.AvgScore, stats.MaxScore, stats.MinScore)

	// Build filtered results
	var filteredItems []render.DigestData
	var filteredArticles []core.Article

	for i, result := range filterResults {
		if result.Included {
			// Update the digest item with relevance score
			if i < len(digestItems) {
				digestItems[i].MyTake = fmt.Sprintf("Relevance: %.2f - %s", result.Score.Value, result.Score.Reasoning)
				filteredItems = append(filteredItems, digestItems[i])
			}

			// Update the article with relevance score
			if i < len(processedArticles) {
				processedArticles[i].RelevanceScore = result.Score.Value
				filteredArticles = append(filteredArticles, processedArticles[i])
			}
		} else {
			fmt.Printf("   🔍 Excluded: %s (%.2f) - %s\n",
				digestItems[i].Title, result.Score.Value, result.Reason)
		}
	}

	if len(filteredItems) == 0 {
		fmt.Printf("   ⚠️ All articles filtered out, keeping top 3 by relevance score\n")
		// Keep top 3 articles if everything gets filtered out
		topCount := 3
		if len(filterResults) < topCount {
			topCount = len(filterResults)
		}

		for i := 0; i < topCount; i++ {
			if i < len(digestItems) {
				filteredItems = append(filteredItems, digestItems[i])
			}
			if i < len(processedArticles) {
				processedArticles[i].RelevanceScore = filterResults[i].Score.Value
				filteredArticles = append(filteredArticles, processedArticles[i])
			}
		}
	}

	fmt.Printf("   ✅ Filtering complete: %d articles included\n", len(filteredItems))
	return filteredItems, filteredArticles, nil
}

// applyEnhancedRelevanceFiltering applies team context-aware relevance filtering
func applyEnhancedRelevanceFiltering(cmd *cobra.Command, digestItems []render.DigestData, processedArticles []core.Article, format string, teamContext string, llmClient *llm.Client) ([]render.DigestData, []core.Article, error) {
	fmt.Println("\n🎯 Applying enhanced relevance filtering with team context...")

	// Get filtering parameters
	minRelevance := config.GetFilteringMinRelevance()
	if cmd.Flags().Changed("min-relevance") {
		if flagVal, err := cmd.Flags().GetFloat64("min-relevance"); err == nil {
			minRelevance = flagVal
		}
	}

	templateFilter := config.GetTemplateFilter(format)
	if !cmd.Flags().Changed("min-relevance") && templateFilter.MinRelevance > 0 {
		minRelevance = templateFilter.MinRelevance
	}

	var filteredItems []render.DigestData
	var filteredArticles []core.Article

	fmt.Printf("   📊 Using team context for enhanced relevance scoring...\n")

	// Generate LLM-based relevance scores using team context
	for i, item := range digestItems {
		if i >= len(processedArticles) {
			continue
		}

		article := processedArticles[i]

		// Use LLM to generate team-aware relevance score
		relevanceScore, reasoning, err := llmClient.GenerateTeamRelevanceScore(article, teamContext)
		if err != nil {
			logger.Warn("Failed to generate team relevance score", "article", item.Title, "error", err)
			// Fallback to basic inclusion
			relevanceScore = 0.5
			reasoning = "LLM scoring failed, using default"
		}

		// Apply threshold
		if relevanceScore >= minRelevance {
			// Update digest item with enhanced relevance info
			digestItems[i].MyTake = fmt.Sprintf("%s (Team relevance: %.2f - %s)",
				digestItems[i].MyTake, relevanceScore, reasoning)
			filteredItems = append(filteredItems, digestItems[i])

			// Update article with relevance score
			processedArticles[i].RelevanceScore = relevanceScore
			filteredArticles = append(filteredArticles, processedArticles[i])

			fmt.Printf("   ✅ Included: %s (%.2f) - %s\n",
				item.Title, relevanceScore, reasoning)
		} else {
			fmt.Printf("   🔍 Excluded: %s (%.2f) - %s\n",
				item.Title, relevanceScore, reasoning)
		}
	}

	// Ensure we don't filter out everything
	if len(filteredItems) == 0 && len(digestItems) > 0 {
		fmt.Printf("   ⚠️ All articles filtered out, keeping top 3 by team relevance\n")

		// Keep top 3 articles by team relevance
		type scoredItem struct {
			digestItem render.DigestData
			article    core.Article
			score      float64
		}

		var scoredItems []scoredItem
		for i, item := range digestItems {
			if i < len(processedArticles) {
				score := processedArticles[i].RelevanceScore
				if score == 0 {
					score = 0.3 // Default for items without scores
				}
				scoredItems = append(scoredItems, scoredItem{
					digestItem: item,
					article:    processedArticles[i],
					score:      score,
				})
			}
		}

		// Sort by score descending
		for i := 0; i < len(scoredItems)-1; i++ {
			for j := i + 1; j < len(scoredItems); j++ {
				if scoredItems[j].score > scoredItems[i].score {
					scoredItems[i], scoredItems[j] = scoredItems[j], scoredItems[i]
				}
			}
		}

		// Keep top 3
		maxToKeep := 3
		if len(scoredItems) < maxToKeep {
			maxToKeep = len(scoredItems)
		}

		for i := 0; i < maxToKeep; i++ {
			filteredItems = append(filteredItems, scoredItems[i].digestItem)
			filteredArticles = append(filteredArticles, scoredItems[i].article)
		}
	}

	fmt.Printf("   ✅ Enhanced filtering complete: %d articles included\n", len(filteredItems))
	return filteredItems, filteredArticles, nil
}

func performClustering(digestItems []render.DigestData, processedArticles []core.Article, llmClient *llm.Client, cacheStore *store.Store) error {
	fmt.Println("\n🧮 Generating embeddings and clustering articles...")

	// Collect articles for clustering
	var articlesForClustering []core.Article
	for i, item := range digestItems {
		if i < len(processedArticles) {
			// Generate embedding for the article if not already present
			if len(processedArticles[i].Embedding) == 0 {
				embedding, err := llmClient.GenerateEmbeddingForArticle(processedArticles[i])
				if err != nil {
					logger.Warn("Failed to generate embedding for article", "title", item.Title, "error", err)
					continue
				}
				processedArticles[i].Embedding = embedding

				// Update the article in cache with embedding
				if cacheStore != nil {
					if err := cacheStore.CacheArticle(processedArticles[i]); err != nil {
						logger.Warn("Failed to update article cache with embedding", "error", err)
					}
				}
			}
			articlesForClustering = append(articlesForClustering, processedArticles[i])
		}
	}

	// Perform clustering if we have enough articles
	if len(articlesForClustering) >= 2 {
		fmt.Printf("🔍 Clustering %d articles...\n", len(articlesForClustering))

		// Auto-detect optimal number of clusters (max 5 for readability)
		maxClusters := len(articlesForClustering) / 2
		if maxClusters > 5 {
			maxClusters = 5
		}
		if maxClusters < 2 {
			maxClusters = 2
		}

		optimalClusters, err := clustering.AutoDetectOptimalClusters(articlesForClustering, maxClusters)
		if err != nil {
			logger.Warn("Failed to detect optimal clusters, using 3 clusters", "error", err)
			optimalClusters = 3
		}

		// Perform clustering
		clusterer := clustering.NewKMeansClusterer()
		clusters, err := clusterer.Cluster(articlesForClustering, optimalClusters)
		if err != nil {
			return fmt.Errorf("failed to cluster articles: %w", err)
		}

		fmt.Printf("✅ Created %d topic clusters\n", len(clusters))

		// Update articles with cluster assignments
		for _, cluster := range clusters {
			for _, articleID := range cluster.ArticleIDs {
				for j := range articlesForClustering {
					if articlesForClustering[j].ID == articleID {
						articlesForClustering[j].TopicCluster = cluster.Label
						articlesForClustering[j].TopicConfidence = 0.8
						break
					}
				}
			}
		}

		// Update cache with cluster assignments
		if cacheStore != nil {
			for _, article := range articlesForClustering {
				if article.TopicCluster != "" {
					if err := cacheStore.CacheArticle(article); err != nil {
						logger.Warn("Failed to update article cache with cluster", "error", err)
					}
				}
			}
		}

		// Print cluster summary
		for _, cluster := range clusters {
			fmt.Printf("  📂 %s: %d articles\n", cluster.Label, len(cluster.ArticleIDs))
		}

		// Update digestItems with cluster information
		for i := range digestItems {
			if i < len(articlesForClustering) {
				digestItems[i].TopicCluster = articlesForClustering[i].TopicCluster
				digestItems[i].TopicConfidence = articlesForClustering[i].TopicConfidence
			}
		}

		// Update processedArticles reference
		copy(processedArticles, articlesForClustering)
	}

	return nil
}

type InsightsData struct {
	OverallSentiment    sentiment.DigestSentiment
	TriggeredAlerts     []alerts.Alert
	ResearchSuggestions []string
	InsightsContent     string
}

func generateInsights(processedArticles []core.Article) (*InsightsData, error) {
	if len(processedArticles) == 0 {
		return &InsightsData{}, nil
	}

	fmt.Println("\n🧠 Generating insights...")

	var insightsContent strings.Builder
	insights := &InsightsData{}

	// Initialize analyzers
	sentimentAnalyzer := sentiment.NewSentimentAnalyzer()
	alertManager := alerts.NewAlertManager()

	// 1. Sentiment Analysis
	fmt.Printf("📊 Analyzing sentiment...\n")
	digestSentiment, err := sentimentAnalyzer.AnalyzeDigest(processedArticles, "digest-"+time.Now().Format("20060102"))
	if err == nil {
		insights.OverallSentiment = *digestSentiment
		sentimentSection := sentimentAnalyzer.FormatSentimentSummary(digestSentiment)
		insightsContent.WriteString(sentimentSection)
		insightsContent.WriteString("\n")
	}

	// 2. Alert Evaluation
	fmt.Printf("🚨 Evaluating alerts...\n")
	alertContext := alerts.AlertContext{
		Articles:      processedArticles,
		EstimatedCost: 0.0,
	}

	// Get current topics for alert context
	var currentTopics []string
	for _, article := range processedArticles {
		if article.TopicCluster != "" {
			currentTopics = append(currentTopics, article.TopicCluster)
		}
	}
	alertContext.CurrentTopics = currentTopics

	insights.TriggeredAlerts = alertManager.CheckConditions(alertContext)
	if len(insights.TriggeredAlerts) > 0 {
		alertSection := alertManager.FormatAlertsSection(insights.TriggeredAlerts)
		insightsContent.WriteString(alertSection)
		insightsContent.WriteString("\n")
		fmt.Printf("   ⚠️ %d alerts triggered\n", len(insights.TriggeredAlerts))
	} else {
		fmt.Printf("   ✅ No alerts triggered\n")
	}

	// 3. Research Query Generation (using LLM client)
	fmt.Printf("🔍 Generating research queries...\n")
	llmClient, err := llm.NewClient("")
	if err == nil {
		defer llmClient.Close()
		for _, article := range processedArticles {
			queries, err := llmClient.GenerateResearchQueries(article, 3)
			if err == nil {
				insights.ResearchSuggestions = append(insights.ResearchSuggestions, queries...)
			}
		}
	}

	insights.InsightsContent = insightsContent.String()
	fmt.Printf("✅ Insights generation complete\n")

	return insights, nil
}

func loadStyleGuide(cmd *cobra.Command) (string, error) {
	styleGuidePath, _ := cmd.Flags().GetString("style-guide")
	if styleGuidePath == "" {
		// Try to get from config
		styleGuidePath = viper.GetString("style.default_guide")
	}

	if styleGuidePath == "" {
		return "", nil // No style guide specified
	}

	// Check if file exists
	if _, err := os.Stat(styleGuidePath); os.IsNotExist(err) {
		return "", fmt.Errorf("style guide file not found: %s", styleGuidePath)
	}

	// Read style guide content
	content, err := os.ReadFile(styleGuidePath)
	if err != nil {
		return "", fmt.Errorf("failed to read style guide: %w", err)
	}

	return string(content), nil
}

func generateOutput(cmd *cobra.Command, digestItems []render.DigestData, processedArticles []core.Article, insightsData *InsightsData, outputDir, format, styleGuide string, withBanner, interactive bool) (string, error) {
	switch format {
	case "brief":
		return generateTeamFriendlyBrief(digestItems, outputDir)
	case "slack":
		return generateSlackOutput(cmd, digestItems, insightsData)
	case "discord":
		return generateDiscordOutput(cmd, digestItems, insightsData)
	case "audio":
		return generateTTSOutput(cmd, digestItems, outputDir)
	case "condensed":
		return generateCondensedOutput(digestItems, outputDir, insightsData)
	default:
		return generateStandardOutput(digestItems, processedArticles, insightsData, outputDir, format, styleGuide, withBanner, interactive)
	}
}

func generateTeamFriendlyBrief(digestItems []render.DigestData, outputDir string) (string, error) {
	fmt.Printf("📝 Generating team-friendly brief format...\n")

	// Get team context for the title
	teamContext := config.GenerateTeamContextPrompt()

	// Use custom title with team context
	customTitle := "Weekly Tech Radar"

	// Generate the team-friendly brief using our new template
	_, filePath, err := templates.RenderTeamFriendlyBrief(digestItems, outputDir, customTitle, teamContext)
	if err != nil {
		return "", fmt.Errorf("failed to render team-friendly brief: %w", err)
	}

	return filePath, nil
}

func generateSlackOutput(cmd *cobra.Command, digestItems []render.DigestData, insightsData *InsightsData) (string, error) {
	webhookURL, _ := cmd.Flags().GetString("slack-webhook")
	if webhookURL == "" {
		return "", fmt.Errorf("slack webhook URL required for slack format (use --slack-webhook)")
	}

	msgFormat, _ := cmd.Flags().GetString("messaging-format")

	// Validate message format
	var messageFormat messaging.MessageFormat
	switch msgFormat {
	case "bullets":
		messageFormat = messaging.FormatBullets
	case "summary":
		messageFormat = messaging.FormatSummary
	case "highlights":
		messageFormat = messaging.FormatHighlights
	default:
		messageFormat = messaging.FormatBullets
	}

	// Create messaging client
	client := messaging.NewMessagingClient(webhookURL, "")

	// Generate title
	title := fmt.Sprintf("Digest - %s", time.Now().Format("Jan 2, 2006"))

	// Send message
	err := client.SendMessage(messaging.PlatformSlack, digestItems, title, messageFormat, true)
	if err != nil {
		return "", fmt.Errorf("failed to send Slack message: %w", err)
	}

	fmt.Printf("✅ Slack message sent successfully\n")
	return "slack://sent", nil
}

func generateDiscordOutput(cmd *cobra.Command, digestItems []render.DigestData, insightsData *InsightsData) (string, error) {
	webhookURL, _ := cmd.Flags().GetString("discord-webhook")
	if webhookURL == "" {
		return "", fmt.Errorf("discord webhook URL required for discord format (use --discord-webhook)")
	}

	msgFormat, _ := cmd.Flags().GetString("messaging-format")

	// Validate message format
	var messageFormat messaging.MessageFormat
	switch msgFormat {
	case "bullets":
		messageFormat = messaging.FormatBullets
	case "summary":
		messageFormat = messaging.FormatSummary
	case "highlights":
		messageFormat = messaging.FormatHighlights
	default:
		messageFormat = messaging.FormatBullets
	}

	// Create messaging client
	client := messaging.NewMessagingClient("", webhookURL)

	// Generate title
	title := fmt.Sprintf("Digest - %s", time.Now().Format("Jan 2, 2006"))

	// Send message
	err := client.SendMessage(messaging.PlatformDiscord, digestItems, title, messageFormat, true)
	if err != nil {
		return "", fmt.Errorf("failed to send Discord message: %w", err)
	}

	fmt.Printf("✅ Discord message sent successfully\n")
	return "discord://sent", nil
}

func generateTTSOutput(cmd *cobra.Command, digestItems []render.DigestData, outputDir string) (string, error) {
	provider, _ := cmd.Flags().GetString("tts-provider")
	voice, _ := cmd.Flags().GetString("tts-voice")
	speed, _ := cmd.Flags().GetFloat32("tts-speed")
	maxArticles, _ := cmd.Flags().GetInt("tts-max-articles")

	// Validate provider
	if provider != "openai" && provider != "elevenlabs" && provider != "google" && provider != "mock" {
		return "", fmt.Errorf("unsupported TTS provider: %s (supported: openai, elevenlabs, google, mock)", provider)
	}

	// Limit articles for TTS
	if len(digestItems) > maxArticles {
		digestItems = digestItems[:maxArticles]
		fmt.Printf("ℹ️  Limited to %d articles for TTS generation\n", maxArticles)
	}

	// Create TTS configuration
	voiceConfig := tts.TTSVoice{ID: voice, Name: voice}
	config := tts.TTSConfig{
		Provider:  tts.TTSProvider(provider),
		Voice:     voiceConfig,
		Speed:     float64(speed),
		OutputDir: outputDir,
	}

	// Generate TTS
	fmt.Printf("🎵 Generating TTS audio using %s provider...\n", provider)

	// Create TTS client and generate audio
	client := tts.NewTTSClient(&config)
	ttsText := tts.PrepareTTSText(digestItems, "Weekly Digest", true, maxArticles)
	filename := fmt.Sprintf("digest_%s.mp3", time.Now().Format("2006-01-02"))
	outputPath, err := client.GenerateAudio(ttsText, filename)
	if err != nil {
		return "", fmt.Errorf("failed to generate TTS: %w", err)
	}

	fmt.Printf("✅ TTS audio generated: %s\n", outputPath)
	return outputPath, nil
}

func generateCondensedOutput(digestItems []render.DigestData, outputDir string, insightsData *InsightsData) (string, error) {
	// Implement condensed format according to Sprint 1 requirements
	// This should be a truly bite-size format (150-200 words, 30-second read)

	fmt.Printf("📝 Generating condensed digest format...\n")

	// Create condensed template data
	condensedData := struct {
		Title        string
		Date         string
		Items        []render.DigestData
		ReadingTime  string
		ArticleCount int
	}{
		Title:        generateCondensedTitle(digestItems),
		Date:         time.Now().Format("Jan 2"),
		Items:        digestItems[:min(5, len(digestItems))], // Limit to 5 items max
		ReadingTime:  "30 sec",
		ArticleCount: len(digestItems),
	}

	// Generate condensed content
	var content strings.Builder
	content.WriteString(fmt.Sprintf("# %s - Week of %s\n\n", condensedData.Title, condensedData.Date))
	content.WriteString("## 🎯 This Week's Picks\n\n")

	for i, item := range condensedData.Items {
		if i >= 3 { // Limit to 3 key items for true condensed format
			break
		}

		// Determine category emoji
		category := determineCategory(item.Title, item.SummaryText)

		// Create one-liner insight (max 60 chars)
		insight := createOneLineInsight(item.SummaryText)

		// Create actionable takeaway
		takeaway := createActionableTakeaway(item.SummaryText)

		content.WriteString(fmt.Sprintf("**%s %s**: %s\n", category, "Category", insight))
		content.WriteString(fmt.Sprintf("→ %s\n\n", takeaway))
	}

	content.WriteString("## 🚀 Try This\n")
	content.WriteString(generateCallToAction(digestItems))
	content.WriteString("\n\n---\n")
	content.WriteString(fmt.Sprintf("*%d articles, %s read • Forward to your team*\n",
		condensedData.ArticleCount, condensedData.ReadingTime))

	// Write to file
	filename := fmt.Sprintf("digest_condensed_%s.md", time.Now().Format("2006-01-02"))
	filepath := filepath.Join(outputDir, filename)

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := os.WriteFile(filepath, []byte(content.String()), 0644); err != nil {
		return "", fmt.Errorf("failed to write condensed digest: %w", err)
	}

	return filepath, nil
}

func generateStandardOutput(digestItems []render.DigestData, processedArticles []core.Article, insightsData *InsightsData, outputDir, format, styleGuide string, withBanner, interactive bool) (string, error) {
	// Initialize LLM client for final digest generation
	llmClient, err := llm.NewClient("")
	if err != nil {
		return "", fmt.Errorf("failed to initialize LLM client: %w", err)
	}
	defer llmClient.Close()

	// Determine which template to use
	var digestFormat templates.DigestFormat
	switch format {
	case "brief":
		digestFormat = templates.FormatBrief
	case "detailed":
		digestFormat = templates.FormatDetailed
	case "newsletter":
		digestFormat = templates.FormatNewsletter
	case "scannable":
		digestFormat = templates.FormatScannableNewsletter
	case "email":
		digestFormat = templates.FormatEmail
	case "standard", "":
		digestFormat = templates.FormatStandard
	default:
		logger.Warn("Unknown format, using standard", "format", format)
		digestFormat = templates.FormatStandard
	}

	template := templates.GetTemplate(digestFormat)

	// Handle interactive article selection if enabled  
	digestItems = handleInteractiveSelection(digestItems, interactive)

	// Generate final digest summary using LLM
	fmt.Printf("Generating final digest summary using %s format...\n", format)
	logger.Info("Generating final digest summary", "format", format)

	// Order items consistently for both LLM input and Sources rendering
	// This ensures reference numbers match between the digest text and Sources section
	orderedItems := orderItemsForSources(digestItems, format)
	
	// Prepare combined summaries for final digest with category information
	var combinedSummaries strings.Builder
	
	// Check if articles are categorized by looking for category info in MyTake
	categorized := checkIfCategorized(orderedItems)
	
	if categorized {
		// Use the SAME categorization logic as the Sources section to ensure reference numbers match
		categoryGroups := groupSignalItemsByCategory(orderedItems)
		// Use the SAME category order as the Sources section
		categoryOrder := []string{"🔥 Breaking & Hot", "🛠️ Tools & Platforms", "📊 Analysis & Research", "💰 Business & Economics", "💡 Additional Items"}
		
		// Create a global reference number counter that matches final Sources section numbering
		globalRefNum := 1
		
		for _, categoryName := range categoryOrder {
			if items, exists := categoryGroups[categoryName]; exists && len(items) > 0 {
				combinedSummaries.WriteString(fmt.Sprintf("**Category: %s** (%d articles)\n", categoryName, len(items)))
				for _, item := range items {
					combinedSummaries.WriteString(fmt.Sprintf("%d. **%s**\n", globalRefNum, item.Title))
					combinedSummaries.WriteString(fmt.Sprintf("   Summary: %s\n", item.SummaryText))
					combinedSummaries.WriteString(fmt.Sprintf("   Reference URL: %s\n\n", item.URL))
					globalRefNum++
				}
				combinedSummaries.WriteString("\n")
			}
		}
		
		// Handle uncategorized items - but note that groupSignalItemsByCategory should categorize all items
		// Keep this as a safety fallback
		uncategorized := getUncategorizedItemsForSignal(orderedItems, categoryGroups)
		if len(uncategorized) > 0 {
			combinedSummaries.WriteString("**Category: Other** (uncategorized)\n")
			for _, item := range uncategorized {
				combinedSummaries.WriteString(fmt.Sprintf("%d. **%s**\n", globalRefNum, item.Title))
				combinedSummaries.WriteString(fmt.Sprintf("   Summary: %s\n", item.SummaryText))
				combinedSummaries.WriteString(fmt.Sprintf("   Reference URL: %s\n\n", item.URL))
				globalRefNum++
			}
		}
	} else {
		// Fallback to original flat format
		for i, item := range orderedItems {
			combinedSummaries.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, item.Title))
			combinedSummaries.WriteString(fmt.Sprintf("   Summary: %s\n", item.SummaryText))
			combinedSummaries.WriteString(fmt.Sprintf("   Reference URL: %s\n\n", item.URL))
		}
	}

	// Use new structured generation approach for newsletter and standard formats
	// For scannable format, we want enhanced summary but keep the categorized Featured Articles section
	var finalDigest string
	var useStructuredGeneration = (format == "newsletter" || format == "standard")
	var useEnhancedSummary = (format == "newsletter" || format == "standard" || format == "scannable")

	if useEnhancedSummary {
		// Enhanced summary generation approach  
		fmt.Printf("🧠 Using enhanced summary generation...\n")

		// Prepare context from insights
		var alertsContext, sentimentContext string
		var researchQueries []string

		if insightsData != nil {
			sentimentContext = fmt.Sprintf("Overall: %s (%.2f)",
				insightsData.OverallSentiment.OverallEmoji, insightsData.OverallSentiment.OverallScore.Overall)

			if len(insightsData.TriggeredAlerts) > 0 {
				alertManager := alerts.NewAlertManager()
				alertsContext = alertManager.FormatAlertsSection(insightsData.TriggeredAlerts)
			}

			researchQueries = insightsData.ResearchSuggestions
		}

		finalDigest, err = llmClient.GenerateStructuredDigest(
			combinedSummaries.String(),
			format,
			alertsContext,
			sentimentContext,
			researchQueries)
	} else if styleGuide != "" {
		// Enhanced prompt with style guide
		finalDigest, err = generateDigestWithStyleGuide(llmClient, combinedSummaries.String(), format, *template, styleGuide)
	} else {
		// Standard digest generation
		finalDigest, err = generateStandardDigest(llmClient, combinedSummaries.String(), format, *template)
	}

	var generatedTitle string
	if err != nil {
		logger.Error("Failed to generate final digest", err)
		fmt.Printf("⚠️  Failed to generate final digest summary, using individual summaries: %s\n", err)
		finalDigest = ""
	}

	// Generate Smart Headline
	fmt.Printf("🎯 Generating Smart Headline...\n")
	contentForTitle := finalDigest
	if contentForTitle == "" {
		contentForTitle = combinedSummaries.String()
	}

	title, titleErr := llmClient.GenerateDigestTitle(contentForTitle, format)
	if titleErr != nil {
		logger.Error("Failed to generate digest title", titleErr)
		fmt.Printf("   ⚠️ Smart Headline generation failed, using default\n")
		generatedTitle = template.Title
	} else {
		generatedTitle = title
		fmt.Printf("   ✅ Smart Headline: \"%s\"\n", generatedTitle)
		logger.Info("Digest title generated", "title", generatedTitle)
	}

	// Generate digest-level research queries as final pipeline step
	fmt.Printf("🔬 Generating digest-level research queries...\n")
	if insightsData != nil {
		teamContext := config.GenerateTeamContextPrompt()
		var articleTitles []string
		for _, item := range digestItems {
			articleTitles = append(articleTitles, item.Title)
		}

		digestResearchQueries, err := llmClient.GenerateDigestResearchQueries(finalDigest, teamContext, articleTitles)
		if err != nil {
			logger.Warn("Failed to generate digest research queries", "error", err)
			fmt.Printf("   ⚠️ Digest research queries generation failed\n")
		} else {
			// Replace individual article queries with digest-level strategic queries
			insightsData.ResearchSuggestions = digestResearchQueries
			fmt.Printf("   ✅ Generated %d strategic research directions\n", len(digestResearchQueries))
		}
	}

	// Prepare insights data for rendering
	var overallSentimentText, alertsSummaryText, trendsSummaryText string
	if insightsData != nil {
		overallSentimentText = fmt.Sprintf("Overall: %s (%.2f)",
			insightsData.OverallSentiment.OverallEmoji, insightsData.OverallSentiment.OverallScore.Overall)

		if len(insightsData.TriggeredAlerts) > 0 {
			alertManager := alerts.NewAlertManager()
			alertsSummaryText = alertManager.FormatAlertsSection(insightsData.TriggeredAlerts)
		} else {
			alertsSummaryText = "### ✅ Alert Monitoring\nNo alerts triggered for this digest. All articles passed through standard monitoring criteria."
		}
	}

	// Generate banner image if requested
	var banner *core.BannerImage
	if withBanner && (template.IncludeBanner || format == "email" || format == "newsletter") {
		banner = generateBannerImage(finalDigest, outputDir, format)
		if banner != nil {
			logger.Info("Banner image generated", "path", banner.ImageURL, "themes", len(banner.Themes))
		}
	}

	// Generate digest output
	var renderedContent, digestPath string
	var renderErr error

	if format == "signal" {
		// Phase 4: Use Signal+Sources template for signal format
		// Note: This is a simplified version using existing data structures
		// Full implementation will use the new core.Digest with Signal structure
		
		// Use orderedItems to ensure Sources section matches LLM input order
		renderedContent, digestPath, renderErr = templates.RenderSignalStyleDigest(orderedItems, outputDir, finalDigest, template, generatedTitle)
	} else if format == "email" {
		renderedContent, digestPath, renderErr = templates.RenderHTMLEmailWithBanner(orderedItems, outputDir, finalDigest, generatedTitle, overallSentimentText, alertsSummaryText, trendsSummaryText, insightsData.ResearchSuggestions, "default", banner)
	} else if useStructuredGeneration {
		// Use new structured rendering approach
		renderedContent, digestPath, renderErr = templates.RenderWithStructuredContent(orderedItems, outputDir, finalDigest, template, generatedTitle, banner)
	} else {
		renderedContent, digestPath, renderErr = templates.RenderWithBannerAndInsights(orderedItems, outputDir, finalDigest, "", template, generatedTitle, overallSentimentText, alertsSummaryText, trendsSummaryText, insightsData.ResearchSuggestions, banner)
	}

	if renderErr != nil {
		return "", fmt.Errorf("failed to render digest with template: %w", renderErr)
	}

	// Cache the digest
	cacheStore, err := store.NewStore(".briefly-cache")
	if err == nil {
		defer func() { _ = cacheStore.Close() }()
		digestID := uuid.NewString()
		var articleURLs []string
		for _, item := range digestItems {
			articleURLs = append(articleURLs, item.URL)
		}

		model := viper.GetString("gemini.model")
		if model == "" {
			model = "gemini-2.5-flash-preview-05-20"
		}

		if cacheErr := cacheStore.CacheDigestWithFormat(digestID, generatedTitle, renderedContent, finalDigest, format, articleURLs, model); cacheErr != nil {
			logger.Error("Failed to cache digest", cacheErr)
		}
	}

	logger.Info("Digest generated successfully", "path", digestPath, "format", format, "title", generatedTitle)
	return digestPath, nil
}

func handleInteractiveMyTake(digestPath, styleGuide string) error {
	fmt.Printf("\n🎯 Enhanced Interactive My-Take Workflow\n")
	fmt.Printf("Digest generated: %s\n", digestPath)

	reader := bufio.NewReader(os.Stdin)

	// Show workflow options
	fmt.Printf("\nWhat would you like to do?\n")
	fmt.Printf("1. 📝 Add personal take and regenerate with LLM\n")
	fmt.Printf("2. 👁️  Just review the digest (open in editor)\n")
	fmt.Printf("3. ⏭️  Skip and finish\n")
	fmt.Print("Enter your choice [1-3]: ")

	choice, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	choice = strings.TrimSpace(choice)
	switch choice {
	case "1":
		return handleLLMRegenerateWorkflow(digestPath, styleGuide, reader)
	case "2":
		return openDigestInEditor(digestPath)
	case "3":
		fmt.Println("✅ Digest workflow complete!")
		return nil
	default:
		fmt.Println("Invalid choice, skipping my-take workflow")
		return nil
	}
}

func handleLLMRegenerateWorkflow(digestPath, styleGuide string, reader *bufio.Reader) error {
	fmt.Printf("\n📝 Personal Take Input\n")
	fmt.Printf("Enter your personal take on this week's content.\n")
	fmt.Printf("This will be integrated throughout the digest by the LLM.\n")
	fmt.Printf("You can include:\n")
	fmt.Printf("  • Your perspective on the trends\n")
	fmt.Printf("  • Personal experiences related to the content\n")
	fmt.Printf("  • Disagreements or additional insights\n")
	fmt.Printf("  • Team-specific implications\n\n")

	fmt.Printf("Type your take (press Enter twice when done):\n")
	fmt.Printf("─────────────────────────────────────────\n")

	var myTakeLines []string
	emptyLineCount := 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read my-take input: %w", err)
		}

		line = strings.TrimRight(line, "\n")

		if line == "" {
			emptyLineCount++
			if emptyLineCount >= 2 {
				break
			}
		} else {
			emptyLineCount = 0
		}

		myTakeLines = append(myTakeLines, line)
	}

	// Remove trailing empty lines
	for len(myTakeLines) > 0 && myTakeLines[len(myTakeLines)-1] == "" {
		myTakeLines = myTakeLines[:len(myTakeLines)-1]
	}

	myTake := strings.Join(myTakeLines, "\n")
	myTake = strings.TrimSpace(myTake)

	if myTake == "" {
		fmt.Println("No take provided, skipping regeneration")
		return nil
	}

	fmt.Printf("─────────────────────────────────────────\n")
	fmt.Printf("📝 Take received (%d characters)\n", len(myTake))

	// Confirm before proceeding
	fmt.Print("Proceed with LLM regeneration? [Y/n]: ")
	confirm, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}

	confirm = strings.TrimSpace(strings.ToLower(confirm))
	if confirm == "n" || confirm == "no" {
		fmt.Println("Regeneration cancelled")
		return nil
	}

	// Read current digest content
	content, err := os.ReadFile(digestPath)
	if err != nil {
		return fmt.Errorf("failed to read digest file: %w", err)
	}

	// Generate new filename with enhanced suffix
	dir := filepath.Dir(digestPath)
	filename := filepath.Base(digestPath)
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)
	timestamp := time.Now().Format("150405")
	newFilename := fmt.Sprintf("%s_enhanced_%s%s", nameWithoutExt, timestamp, ext)
	newPath := filepath.Join(dir, newFilename)

	// Regenerate with LLM including my-take and team context
	llmClient, err := llm.NewClient("")
	if err != nil {
		return fmt.Errorf("failed to initialize LLM client: %w", err)
	}
	defer llmClient.Close()

	fmt.Printf("\n🤖 Regenerating digest with LLM integration...\n")
	regeneratedContent, err := regenerateDigestWithMyTake(llmClient, string(content), myTake, styleGuide)
	if err != nil {
		return fmt.Errorf("failed to regenerate digest with my-take: %w", err)
	}

	// Write regenerated content
	if err := os.WriteFile(newPath, []byte(regeneratedContent), 0644); err != nil {
		return fmt.Errorf("failed to write regenerated digest: %w", err)
	}

	fmt.Printf("✅ Enhanced digest generated: %s\n", newPath)
	fmt.Printf("📊 Original: %d chars → Enhanced: %d chars\n", len(content), len(regeneratedContent))

	// Optionally open the new digest
	fmt.Print("Open enhanced digest in editor? [Y/n]: ")
	openResponse, err := reader.ReadString('\n')
	if err == nil {
		openResponse = strings.TrimSpace(strings.ToLower(openResponse))
		if openResponse != "n" && openResponse != "no" {
			return openDigestInEditor(newPath)
		}
	}

	return nil
}

func openDigestInEditor(digestPath string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim" // Default fallback
	}

	fmt.Printf("📖 Opening digest in %s...\n", editor)
	cmd := exec.Command(editor, digestPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}

	fmt.Printf("✅ Editor closed\n")
	return nil
}

// Helper functions for condensed format
func generateCondensedTitle(digestItems []render.DigestData) string {
	// Simple title generation - could be enhanced with LLM later
	return "Dev Insights & Quick Wins"
}

func determineCategory(title, summary string) string {
	// Simple category detection - could be enhanced with LLM
	titleLower := strings.ToLower(title)
	summaryLower := strings.ToLower(summary)

	if strings.Contains(titleLower, "dev") || strings.Contains(summaryLower, "development") {
		return "🔧"
	} else if strings.Contains(titleLower, "research") || strings.Contains(summaryLower, "study") {
		return "📚"
	} else {
		return "💡"
	}
}

func createOneLineInsight(summary string) string {
	// Extract first sentence or create summary
	sentences := strings.Split(summary, ".")
	if len(sentences) > 0 && len(sentences[0]) <= 60 {
		return strings.TrimSpace(sentences[0])
	}

	// Truncate if too long
	if len(summary) > 60 {
		return summary[:57] + "..."
	}
	return summary
}

func createActionableTakeaway(summary string) string {
	// Simple extraction - could be enhanced with LLM
	if strings.Contains(summary, "should") {
		parts := strings.Split(summary, "should")
		if len(parts) > 1 {
			return "Should" + strings.Split(parts[1], ".")[0]
		}
	}
	return "Worth exploring for your next project"
}

func generateCallToAction(digestItems []render.DigestData) string {
	// Simple CTA generation - could be enhanced
	return "Try implementing one of these insights in your current project"
}

func generateDigestWithStyleGuide(llmClient *llm.Client, content, format string, template templates.DigestTemplate, styleGuide string) (string, error) {
	// This would need to be implemented in the LLM client
	// For now, fall back to standard generation
	return generateStandardDigest(llmClient, content, format, template)
}

func generateStandardDigest(llmClient *llm.Client, content, format string, template templates.DigestTemplate) (string, error) {
	// Use the new LLM client method to generate the final digest
	return llmClient.GenerateFinalDigest(content, format)
}

func regenerateDigestWithMyTake(llmClient *llm.Client, originalContent, myTake, styleGuide string) (string, error) {
	// Get team context for enhanced regeneration
	teamContext := config.GenerateTeamContextPrompt()

	// Use LLM to regenerate the entire digest with personal take integrated
	fmt.Printf("🤖 Using LLM to regenerate digest with your personal take...\n")

	regeneratedContent, err := llmClient.RegenerateDigestWithMyTake(originalContent, myTake, teamContext, styleGuide)
	if err != nil {
		// Fallback to simple append if LLM regeneration fails
		fmt.Printf("⚠️  LLM regeneration failed (%s), falling back to simple append\n", err.Error())
		return originalContent + "\n\n## My Take\n\n" + myTake, nil
	}

	fmt.Printf("✅ Successfully regenerated digest with integrated personal insights\n")
	return regeneratedContent, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// generateBannerImage creates an AI-generated banner image for the digest
func generateBannerImage(digestContent, outputDir, format string) *core.BannerImage {
	// Initialize LLM service for content analysis
	llmClient, err := llm.NewClient("")
	if err != nil {
		logger.Warn("Failed to initialize LLM client for banner generation", "error", err)
		return nil
	}
	defer llmClient.Close()

	// Get OpenAI API key for DALL-E (try visual config first, then fallback to env var)
	openAIKey := viper.GetString("visual.openai.api_key")
	if openAIKey == "" {
		openAIKey = viper.GetString("openai.api_key")
	}
	if openAIKey == "" {
		openAIKey = os.Getenv("OPENAI_API_KEY")
	}
	if openAIKey == "" {
		logger.Warn("OpenAI API key not configured (set visual.openai.api_key, openai.api_key, or OPENAI_API_KEY env var), skipping banner generation")
		return nil
	}

	// Initialize visual service (pass nil for LLM service since we'll handle theme analysis differently)
	visualService := visual.NewService(nil, openAIKey, outputDir)

	// Create a digest object for theme analysis
	digest := &core.Digest{
		Title:         "Digest",
		Content:       digestContent,
		DigestSummary: digestContent,
	}

	ctx := context.Background()

	// Analyze content themes
	themes, err := visualService.AnalyzeContentThemes(ctx, digest)
	if err != nil {
		logger.Warn("Failed to analyze content themes", "error", err)
		return nil
	}

	// Determine banner style (use config or format-based default)
	style := viper.GetString("visual.banners.default_style")
	if style == "" {
		switch format {
		case "newsletter":
			style = "minimalist"
		case "email":
			style = "professional"
		default:
			style = "tech"
		}
	}

	// Generate banner prompt
	prompt, err := visualService.GenerateBannerPrompt(ctx, themes, style)
	if err != nil {
		logger.Warn("Failed to generate banner prompt", "error", err)
		return nil
	}

	// Get configuration values with defaults
	width := viper.GetInt("visual.banners.width")
	if width == 0 {
		width = 1792 // Default to DALL-E's 16:9-ish ratio
	}
	height := viper.GetInt("visual.banners.height")
	if height == 0 {
		height = 1024
	}

	// Set up banner output directory
	bannerOutputDir := outputDir
	if bannerSubDir := viper.GetString("visual.banners.output_directory"); bannerSubDir != "" {
		bannerOutputDir = filepath.Join(outputDir, bannerSubDir)
	}

	// Generate banner image
	bannerConfig := services.BannerConfig{
		Style:     style,
		Width:     width,
		Height:    height,
		Quality:   "high", // Keep for backward compatibility, not used by new API
		Format:    "JPEG",
		OutputDir: bannerOutputDir,
	}

	banner, err := visualService.GenerateBannerImage(ctx, prompt, bannerConfig)
	if err != nil {
		logger.Warn("Failed to generate banner image", "error", err)
		return nil
	}

	// Generate alt text
	altText, err := visualService.GenerateAltText(ctx, themes)
	if err == nil {
		banner.AltText = altText
	}

	// Store theme information in banner
	for _, theme := range themes {
		banner.Themes = append(banner.Themes, theme.Theme)
	}

	return banner
}

// orderItemsForSources orders items the same way the Sources section will display them
// This ensures reference numbers in the LLM-generated text match the Sources section
func orderItemsForSources(items []render.DigestData, format string) []render.DigestData {
	// Only signal format currently uses special ordering
	// Other formats keep original order
	if format != "signal" && format != "scannable" && format != "newsletter" {
		return items
	}
	
	// Use the same categorization as Sources section
	categoryGroups := groupSignalItemsByCategory(items)
	categoryOrder := []string{
		"🔥 Breaking & Hot",
		"🛠️ Tools & Platforms",
		"📊 Analysis & Research",
		"💰 Business & Economics",
		"💡 Additional Items",
	}
	
	// Build ordered list
	ordered := []render.DigestData{}
	for _, category := range categoryOrder {
		if categoryItems, exists := categoryGroups[category]; exists {
			ordered = append(ordered, categoryItems...)
		}
	}
	
	// Handle any uncategorized items
	uncategorized := getUncategorizedItemsForSignal(items, categoryGroups)
	ordered = append(ordered, uncategorized...)
	
	return ordered
}

// checkIfCategorized checks if articles have category information in MyTake
func checkIfCategorized(digestItems []render.DigestData) bool {
	for _, item := range digestItems {
		if item.MyTake != "" && (strings.Contains(item.MyTake, "🔥") || strings.Contains(item.MyTake, "🚀") || strings.Contains(item.MyTake, "🛠️") || strings.Contains(item.MyTake, "📊") || strings.Contains(item.MyTake, "💡") || strings.Contains(item.MyTake, "🔍")) {
			return true
		}
	}
	return false
}

// groupItemsByCategory groups digest items by their category information from MyTake
func groupItemsByCategory(digestItems []render.DigestData) map[string][]render.DigestData {
	categoryGroups := make(map[string][]render.DigestData)
	
	for _, item := range digestItems {
		if item.MyTake != "" && strings.Contains(item.MyTake, " ") {
			// Extract category from MyTake (format: "🔥 Breaking & Hot | insight")
			parts := strings.Split(item.MyTake, " | ")
			if len(parts) >= 1 {
				categoryName := strings.TrimSpace(parts[0])
				if categoryName != "" && strings.Contains(categoryName, " ") {
					categoryGroups[categoryName] = append(categoryGroups[categoryName], item)
				}
			}
		}
	}
	
	return categoryGroups
}

// getUncategorizedItems returns items that don't belong to any category group
func getUncategorizedItems(digestItems []render.DigestData, categoryGroups map[string][]render.DigestData) []render.DigestData {
	var uncategorized []render.DigestData
	
	for _, item := range digestItems {
		found := false
		for _, articles := range categoryGroups {
			for _, catItem := range articles {
				if catItem.URL == item.URL {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			uncategorized = append(uncategorized, item)
		}
	}
	
	return uncategorized
}

// handleInteractiveSelection manages the interactive article selection workflow
func handleInteractiveSelection(digestItems []render.DigestData, enableInteractive bool) []render.DigestData {
	if !enableInteractive {
		return digestItems
	}
	
	fmt.Println("\n🎯 Starting interactive article selection...")
	
	// For now, implement a simple interactive flow using built-in Go packages
	// This will avoid the import issue and provide basic functionality
	
	// Sort articles by priority score using proper calculation (highest first)
	digestItems = templates.SortByPriority(digestItems)
	
	fmt.Printf("\n📖 Select Game-Changer Article (%d articles processed)\n", len(digestItems))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	// Display articles with priority scores and summaries
	for i, article := range digestItems {
		// Truncate long titles
		title := article.Title
		if len(title) > 60 {
			title = title[:57] + "..."
		}
		
		// Truncate summary for readability
		summary := article.SummaryText
		if len(summary) > 120 {
			summary = summary[:117] + "..."
		}
		
		// Get category emoji from MyTake
		categoryEmoji := extractCategoryEmoji(article.MyTake)
		
		fmt.Printf("[%d] %s %s (Score: %.2f)\n", 
			i+1, categoryEmoji, title, article.PriorityScore)
		fmt.Printf("    📝 %s\n\n", summary)
	}
	
	fmt.Printf("\nEnter number (1-%d), or 'a' for auto-selection: ", len(digestItems))
	
	scanner := bufio.NewScanner(os.Stdin)
	for {
		if !scanner.Scan() {
			logger.Warn("Failed to read input, using auto-selection")
			break
		}
		
		input := strings.TrimSpace(scanner.Text())
		
		// Handle auto-selection
		if strings.ToLower(input) == "a" || strings.ToLower(input) == "auto" {
			selectedArticle := &digestItems[0] // Highest priority
			selectedArticle.UserSelected = false // Mark as auto-selected
			fmt.Printf("✅ Auto-selected: %s\n", selectedArticle.Title)
			return digestItems
		}
		
		// Parse number selection
		num, err := strconv.Atoi(input)
		if err != nil || num < 1 || num > len(digestItems) {
			fmt.Printf("Invalid selection. Enter number (1-%d) or 'a' for auto: ", len(digestItems))
			continue
		}
		
		// Select the article
		selectedArticle := &digestItems[num-1]
		selectedArticle.UserSelected = true // Mark as user-selected
		selectedArticle.PriorityScore = 1.0 // Maximum priority to ensure it becomes Game-Changer
		
		fmt.Printf("✅ Selected: %s\n", selectedArticle.Title)
		
		// Simple user take input
		fmt.Printf("\n📝 Add your personal take (optional, press Enter to skip):\n> ")
		if scanner.Scan() {
			userTake := strings.TrimSpace(scanner.Text())
			if userTake != "" {
				selectedArticle.UserTakeText = userTake
				fmt.Printf("✅ Take captured (%d characters)\n", len(userTake))
			}
		}
		
		// Update the article in slice
		digestItems[num-1] = *selectedArticle
		
		fmt.Printf("✅ Interactive selection complete. Selected: %s\n", selectedArticle.Title)
		return digestItems
	}
	
	return digestItems
}

// extractCategoryEmoji extracts emoji from category info in MyTake field
func extractCategoryEmoji(myTake string) string {
	if myTake == "" {
		return "📄"
	}
	
	// Extract emoji from format like "🔥 Breaking & Hot | insight"
	parts := strings.Split(myTake, " ")
	if len(parts) > 0 {
		// Check if first part contains emoji
		emoji := parts[0]
		if len(emoji) > 0 {
			// Basic emoji detection (Unicode range for common emojis)
			for _, r := range emoji {
				if r >= 0x1F300 && r <= 0x1F9FF {
					return emoji
				}
			}
		}
	}
	
	return "📄" // Default document emoji
}

// groupSignalItemsByCategory groups digest items by category for Signal format
// This mirrors the logic used in the Sources section to ensure reference numbers match
func groupSignalItemsByCategory(digestItems []render.DigestData) map[string][]render.DigestData {
	categoryGroups := make(map[string][]render.DigestData)
	
	for _, item := range digestItems {
		category := extractCategoryFromItemForSignal(item)
		categoryGroups[category] = append(categoryGroups[category], item)
	}
	
	return categoryGroups
}

// extractCategoryFromItemForSignal extracts category from digest item using the same logic as Sources section
func extractCategoryFromItemForSignal(item render.DigestData) string {
	// Try to extract from MyTake first (which may contain category info)
	if item.MyTake != "" && strings.Contains(item.MyTake, "🔥") {
		return "🔥 Breaking & Hot"
	}
	if item.MyTake != "" && strings.Contains(item.MyTake, "🛠️") {
		return "🛠️ Tools & Platforms"
	}
	if item.MyTake != "" && strings.Contains(item.MyTake, "📊") {
		return "📊 Analysis & Research"
	}
	if item.MyTake != "" && strings.Contains(item.MyTake, "💰") {
		return "💰 Business & Economics"
	}
	
	// Fallback to keyword-based categorization
	title := strings.ToLower(item.Title)
	summary := strings.ToLower(item.SummaryText)
	
	// Breaking news and hot topics
	if strings.Contains(title, "breaking") || strings.Contains(title, "announce") ||
		strings.Contains(title, "launch") || strings.Contains(title, "release") ||
		strings.Contains(summary, "just announced") || strings.Contains(summary, "breaking") {
		return "🔥 Breaking & Hot"
	}
	
	// Tools and platforms
	if strings.Contains(title, "tool") || strings.Contains(title, "platform") ||
		strings.Contains(title, "framework") || strings.Contains(title, "api") ||
		strings.Contains(title, "sdk") || strings.Contains(title, "library") ||
		strings.Contains(summary, "development") || strings.Contains(summary, "coding") {
		return "🛠️ Tools & Platforms"
	}
	
	// Analysis and research
	if strings.Contains(title, "analysis") || strings.Contains(title, "research") ||
		strings.Contains(title, "report") || strings.Contains(title, "study") ||
		strings.Contains(title, "survey") || strings.Contains(summary, "analysis") ||
		strings.Contains(summary, "research") || strings.Contains(summary, "study") {
		return "📊 Analysis & Research"
	}
	
	// Business and economics
	if strings.Contains(title, "business") || strings.Contains(title, "economic") ||
		strings.Contains(title, "funding") || strings.Contains(title, "investment") ||
		strings.Contains(title, "revenue") || strings.Contains(summary, "business") ||
		strings.Contains(summary, "economic") || strings.Contains(summary, "funding") {
		return "💰 Business & Economics"
	}
	
	// Default category
	return "💡 Additional Items"
}

// getUncategorizedItemsForSignal returns items that don't belong to any signal category group
func getUncategorizedItemsForSignal(digestItems []render.DigestData, categoryGroups map[string][]render.DigestData) []render.DigestData {
	var uncategorized []render.DigestData
	
	for _, item := range digestItems {
		found := false
		for _, articles := range categoryGroups {
			for _, catItem := range articles {
				if catItem.URL == item.URL {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			uncategorized = append(uncategorized, item)
		}
	}
	
	return uncategorized
}

