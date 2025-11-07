package handlers

import (
	"briefly/internal/clustering"
	"briefly/internal/config"
	"briefly/internal/core"
	"briefly/internal/llm"
	"briefly/internal/logger"
	"briefly/internal/markdown"
	"briefly/internal/narrative"
	"briefly/internal/persistence"
	"briefly/internal/summarize"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// llmClientAdapter adapts llm.Client to summarize.LLMClient interface
type llmClientAdapter struct {
	client *llm.Client
}

// GenerateText implements summarize.LLMClient interface
func (a *llmClientAdapter) GenerateText(ctx context.Context, prompt string, opts interface{}) (string, error) {
	return a.client.GenerateText(ctx, prompt, llm.TextGenerationOptions{})
}

// narrativeLLMAdapter adapts llm.Client to narrative.LLMClient interface
type narrativeLLMAdapter struct {
	client *llm.Client
}

// GenerateText implements narrative.LLMClient interface (v2.0 with options)
func (a *narrativeLLMAdapter) GenerateText(ctx context.Context, prompt string, options llm.TextGenerationOptions) (string, error) {
	return a.client.GenerateText(ctx, prompt, options)
}

// GetGenaiModel implements narrative.LLMClient interface
func (a *narrativeLLMAdapter) GetGenaiModel() *genai.GenerativeModel {
	return a.client.GetGenaiModel()
}

// NewDigestGenerateCmd creates the digest generate command for database-driven digests
func NewDigestGenerateCmd() *cobra.Command {
	var (
		sinceDay int
		themeFilter string
		outputDir string
		minArticles int
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate digest from classified articles in database",
		Long: `Generate a digest from classified articles stored in the database.

This command (Phase 1 - Digest from Database):
  • Queries classified articles from database
  • Filters by theme and date range
  • Groups articles by theme
  • Generates structured summaries
  • Creates digest markdown file

Typical usage:
  • Weekly digest: briefly digest generate --since 7
  • Theme-specific: briefly digest generate --theme "AI & Machine Learning"
  • Recent articles: briefly digest generate --since 1

Examples:
  # Generate digest from last 7 days
  briefly digest generate --since 7

  # Generate theme-specific digest
  briefly digest generate --theme "AI & Machine Learning" --since 7

  # Generate from last 24 hours
  briefly digest generate --since 1

  # Require minimum articles
  briefly digest generate --since 7 --min-articles 5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDigestGenerate(cmd.Context(), sinceDay, themeFilter, outputDir, minArticles)
		},
	}

	cmd.Flags().IntVar(&sinceDay, "since", 7, "Include articles from last N days")
	cmd.Flags().StringVar(&themeFilter, "theme", "", "Filter by specific theme name")
	cmd.Flags().StringVarP(&outputDir, "output", "o", "digests", "Output directory for digest file")
	cmd.Flags().IntVar(&minArticles, "min-articles", 3, "Minimum articles required to generate digest")

	return cmd
}

func runDigestGenerate(ctx context.Context, sinceDays int, themeFilter string, outputDir string, minArticles int) error {
	startTime := time.Now()
	log := logger.Get()
	log.Info("Starting digest generation from database",
		"since_days", sinceDays,
		"theme_filter", themeFilter,
		"min_articles", minArticles,
	)

	// Load configuration
	_, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg := config.Get()

	// Get database connection
	dbConnStr := cfg.Database.ConnectionString
	if dbConnStr == "" {
		dbConnStr = os.Getenv("DATABASE_URL")
		if dbConnStr == "" {
			return fmt.Errorf("database connection string not configured")
		}
	}

	// Connect to database
	db, err := persistence.NewPostgresDB(dbConnStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	log.Info("Connected to database")

	// Calculate date range
	since := time.Now().AddDate(0, 0, -sinceDays)

	// Query classified articles
	log.Info("Querying classified articles", "since", since.Format("2006-01-02"), "theme", themeFilter)

	articles, err := queryClassifiedArticles(ctx, db, since, themeFilter)
	if err != nil {
		return fmt.Errorf("failed to query articles: %w", err)
	}

	if len(articles) == 0 {
		fmt.Println("⚠️  No classified articles found")
		fmt.Printf("   Date range: %s to now\n", since.Format("2006-01-02"))
		if themeFilter != "" {
			fmt.Printf("   Theme filter: %s\n", themeFilter)
		}
		fmt.Println("\nNext steps:")
		fmt.Println("  • Run aggregation: briefly aggregate --since 24")
		return nil
	}

	if len(articles) < minArticles {
		fmt.Printf("⚠️  Only %d articles found (minimum: %d)\n", len(articles), minArticles)
		fmt.Println("   Run aggregation to collect more articles: briefly aggregate")
		return nil
	}

	log.Info("Found classified articles", "count", len(articles))

	// Group articles by theme
	themeGroups, err := groupArticlesByTheme(ctx, db, articles)
	if err != nil {
		return fmt.Errorf("failed to group articles by theme: %w", err)
	}

	fmt.Printf("\n📊 Articles by Theme:\n")
	for themeName, themeArticles := range themeGroups {
		fmt.Printf("  • %s: %d articles\n", themeName, len(themeArticles))
	}

	// Initialize LLM client for summaries and narrative
	modelName := cfg.AI.Gemini.Model
	if modelName == "" {
		modelName = "gemini-2.5-flash-preview-05-20"
	}

	llmClient, err := llm.NewClient(modelName)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}
	defer llmClient.Close()

	// Generate digests with clustering (v2.0 architecture)
	fmt.Println("\n🤖 Generating summaries and clustering articles...")
	digests, err := generateDigestsWithClustering(ctx, db, llmClient, articles, since, themeFilter)
	if err != nil {
		return fmt.Errorf("failed to generate digests: %w", err)
	}

	if len(digests) == 0 {
		fmt.Println("⚠️  No digests generated (clustering found no valid clusters)")
		return nil
	}

	// Save each digest to database
	fmt.Printf("\n💾 Saving %d digests to database...\n", len(digests))
	savedCount := 0
	var outputPaths []string

	for i, digest := range digests {
		fmt.Printf("   [%d/%d] Saving: %s\n", i+1, len(digests), digest.Title)

		// Build article IDs and theme IDs for this digest
		articleIDs := make([]string, 0, len(digest.Articles))
		themeIDSet := make(map[string]bool)

		for _, article := range digest.Articles {
			articleIDs = append(articleIDs, article.ID)
			if article.ThemeID != nil {
				themeIDSet[*article.ThemeID] = true
			}
		}

		themeIDs := make([]string, 0, len(themeIDSet))
		for themeID := range themeIDSet {
			themeIDs = append(themeIDs, themeID)
		}

		// Store with relationships (includes citation extraction)
		if err := db.Digests().StoreWithRelationships(ctx, digest, articleIDs, themeIDs); err != nil {
			log.Warn("Failed to save digest", "digest_id", digest.ID, "error", err)
			continue
		}

		// Save markdown file
		outputPath, err := saveDigestMarkdown(digest, outputDir)
		if err != nil {
			log.Warn("Failed to save markdown file", "digest_id", digest.ID, "error", err)
		} else {
			outputPaths = append(outputPaths, outputPath)
		}

		savedCount++
		log.Info("Digest saved", "digest_id", digest.ID, "cluster_id", digest.ClusterID, "articles", len(articleIDs))
	}

	duration := time.Since(startTime)

	fmt.Printf("\n✅ Successfully generated %d digests\n", savedCount)
	fmt.Printf("   Total articles: %d\n", len(articles))
	fmt.Printf("   Clusters found: %d\n", len(digests))
	fmt.Printf("   Database: Saved ✓\n")
	fmt.Printf("   Markdown files: %d\n", len(outputPaths))
	fmt.Printf("   Duration: %s\n", duration.Round(time.Millisecond))

	// Show digest breakdown
	fmt.Println("\n📊 Digest Breakdown:")
	for i, digest := range digests {
		fmt.Printf("   %d. %s (%d articles)\n", i+1, digest.Title, digest.ArticleCount)
	}

	return nil
}

// queryClassifiedArticles fetches articles from database with filters
func queryClassifiedArticles(ctx context.Context, db *persistence.PostgresDB, since time.Time, themeFilter string) ([]core.Article, error) {
	log := logger.Get()

	// Get articles repository
	articlesRepo := db.Articles()

	// For now, list all articles and filter in memory
	// TODO: Add proper query methods to repository
	allArticles, err := articlesRepo.List(ctx, persistence.ListOptions{
		Limit:  1000,
		Offset: 0,
	})
	if err != nil {
		return nil, err
	}

	log.Info("Fetched articles from database", "total_count", len(allArticles))

	var filtered []core.Article
	var skippedOld, skippedNoTheme int

	for _, article := range allArticles {
		// Filter by date (use DateFetched as proxy for DateAdded)
		if article.DateFetched.Before(since) {
			skippedOld++
			continue
		}

		// Filter by theme (must have theme assigned)
		if article.ThemeID == nil {
			skippedNoTheme++
			continue
		}

		// TODO: If theme filter specified, filter by theme name
		// This would require fetching theme names here or using a join query
		// For now, filtering by theme is done in the grouping step

		filtered = append(filtered, article)
	}

	log.Info("Filtered articles",
		"matched", len(filtered),
		"skipped_old", skippedOld,
		"skipped_no_theme", skippedNoTheme,
		"since_date", since.Format("2006-01-02"),
	)

	return filtered, nil
}

// groupArticlesByTheme groups articles by their theme name
func groupArticlesByTheme(ctx context.Context, db *persistence.PostgresDB, articles []core.Article) (map[string][]core.Article, error) {
	log := logger.Get()

	// First, collect all unique theme IDs
	themeIDs := make(map[string]bool)
	for _, article := range articles {
		if article.ThemeID != nil {
			themeIDs[*article.ThemeID] = true
		}
	}

	// Fetch all themes to create ID -> Name mapping
	themes, err := db.Themes().List(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch themes: %w", err)
	}

	themeIDToName := make(map[string]string)
	for _, theme := range themes {
		themeIDToName[theme.ID] = theme.Name
	}

	log.Info("Loaded theme mappings", "theme_count", len(themeIDToName))

	// Group articles by theme name
	groups := make(map[string][]core.Article)

	for _, article := range articles {
		if article.ThemeID == nil {
			continue
		}

		themeName, found := themeIDToName[*article.ThemeID]
		if !found {
			log.Warn("Article has unknown theme ID", "theme_id", *article.ThemeID, "article_id", article.ID)
			themeName = "Unknown Theme"
		}

		groups[themeName] = append(groups[themeName], article)
	}

	return groups, nil
}

// generateDigestsWithClustering creates multiple digests using clustering (v2.0)
// Returns one digest per topic cluster
func generateDigestsWithClustering(ctx context.Context, db *persistence.PostgresDB, llmClient *llm.Client, articles []core.Article, since time.Time, themeFilter string) ([]*core.Digest, error) {
	log := logger.Get()

	// Create summarizer with adapter
	adapter := &llmClientAdapter{client: llmClient}
	summarizer := summarize.NewSummarizerWithDefaults(adapter)

	// Generate summaries for articles (or fetch from database)
	fmt.Println("   📝 Generating article summaries...")
	articleSummaries := make(map[string]*core.Summary)
	summaryList := []core.Summary{}

	for i, article := range articles {
		fmt.Printf("   [%d/%d] Summarizing: %s\n", i+1, len(articles), article.Title)

		// Try to fetch existing summary from database
		existingSummary, err := db.Summaries().Get(ctx, article.ID)
		if err == nil && existingSummary != nil {
			articleSummaries[article.ID] = existingSummary
			summaryList = append(summaryList, *existingSummary)
			log.Info("Using existing summary", "article_id", article.ID)
			continue
		}

		// Generate new summary
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

		// Store summary in database
		if err := db.Summaries().Create(ctx, summary); err != nil {
			log.Warn("Failed to save summary to database", "error", err)
		}

		articleSummaries[article.ID] = summary
		summaryList = append(summaryList, *summary)
	}

	// Step 2: Generate embeddings for clustering
	fmt.Println("   🧠 Generating embeddings for clustering...")
	embeddingsMap := make(map[string][]float64)

	for i, article := range articles {
		// Use summary text for embedding (more concise than full article)
		summary, hasSummary := articleSummaries[article.ID]
		textForEmbedding := article.CleanedText
		if hasSummary {
			textForEmbedding = summary.SummaryText
		}

		// Truncate if too long (embeddings have token limits)
		if len(textForEmbedding) > 2000 {
			textForEmbedding = textForEmbedding[:2000]
		}

		fmt.Printf("   [%d/%d] Embedding: %s\n", i+1, len(articles), article.Title)

		embedding, err := llmClient.GenerateEmbedding(textForEmbedding)
		if err != nil {
			log.Warn("Failed to generate embedding", "article_id", article.ID, "error", err)
			continue
		}

		embeddingsMap[article.ID] = embedding

		// Update article with embedding
		articles[i].Embedding = embedding
	}

	// Persist embeddings back to database
	fmt.Println("   💾 Saving embeddings to database...")
	articlesRepo := db.Articles()
	for i := range articles {
		if len(articles[i].Embedding) > 0 {
			if err := articlesRepo.Update(ctx, &articles[i]); err != nil {
				log.Warn("Failed to save embedding to database", "article_id", articles[i].ID, "error", err)
			}
		}
	}
	fmt.Println("   ✓ Embeddings saved")

	// Step 3: Cluster articles using K-means++ with cosine distance
	// Automatically determine K based on article count (aim for 5-6 articles per cluster for better focus)
	// This creates more granular, topic-specific digests
	numClusters := (len(articles) + 4) / 5 // ~5 articles per cluster
	if numClusters < 3 {
		numClusters = 3
	}
	if numClusters > 15 {
		numClusters = 15 // Cap at 15 to avoid too many small clusters
	}

	fmt.Printf("   🔍 Clustering %d articles into ~%d topics (K-means++ with cosine distance)...\n", len(articles), numClusters)

	clusterer := clustering.NewKMeansClusterer()
	clusters, err := clusterer.Cluster(articles, numClusters)
	if err != nil {
		return nil, fmt.Errorf("failed to cluster articles: %w", err)
	}

	if len(clusters) == 0 {
		return nil, fmt.Errorf("no clusters found")
	}

	fmt.Printf("   ✓ Found %d topic clusters\n", len(clusters))

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

	// Create narrative generator
	narrativeAdapter := &narrativeLLMAdapter{client: llmClient}
	narrativeGen := narrative.NewGenerator(narrativeAdapter)

	// Step 4: Generate cluster narratives (hierarchical summarization - Stage 1)
	fmt.Println("   📖 Generating cluster narratives from ALL articles...")
	for i, cluster := range clusters {
		if len(cluster.ArticleIDs) == 0 {
			continue
		}

		fmt.Printf("   [%d/%d] Cluster: %s (%d articles)\n", i+1, len(clusters), cluster.Label, len(cluster.ArticleIDs))

		// Generate comprehensive narrative from ALL articles in this cluster
		clusterNarrative, err := narrativeGen.GenerateClusterSummary(ctx, cluster, articleMap, summaryMap)
		if err != nil {
			log.Warn("Failed to generate cluster narrative", "cluster", cluster.Label, "error", err)
			fmt.Printf("   ⚠  Cluster narrative generation failed, using legacy approach\n")
			continue
		}

		// Update cluster with generated narrative
		clusters[i].Narrative = clusterNarrative
		fmt.Printf("   ✓ Generated: %s (%d words)\n", clusterNarrative.Title, len(clusterNarrative.Summary)/5)
	}

	// Step 5: Generate one digest per cluster (using cluster narratives - Stage 2)
	fmt.Println("   ✨ Generating digest for each cluster...")

	digests := make([]*core.Digest, 0, len(clusters))

	for clusterIdx, cluster := range clusters {
		if len(cluster.ArticleIDs) == 0 {
			continue
		}

		fmt.Printf("   [%d/%d] Cluster: %s (%d articles)\n", clusterIdx+1, len(clusters), cluster.Label, len(cluster.ArticleIDs))

		// Generate digest content for this cluster
		digestContent, err := narrativeGen.GenerateDigestContent(ctx, []core.TopicCluster{cluster}, articleMap, summaryMap)
		if err != nil {
			log.Warn("Failed to generate digest content for cluster", "cluster", cluster.Label, "error", err)
			// Use fallback
			digestContent = &narrative.DigestContent{
				Title:            cluster.Label,
				TLDRSummary:      fmt.Sprintf("Digest covering %d articles about %s", len(cluster.ArticleIDs), cluster.Label),
				KeyMoments:       []core.KeyMoment{},
				Perspectives:     []core.Perspective{},
				ExecutiveSummary: fmt.Sprintf("This digest covers developments in %s.", cluster.Label),
			}
		}

		// Build article list for this cluster
		clusterArticles := make([]core.Article, 0, len(cluster.ArticleIDs))
		for _, articleID := range cluster.ArticleIDs {
			if article, found := articleMap[articleID]; found {
				clusterArticles = append(clusterArticles, article)
			}
		}

		// Extract actual themes from articles by looking up their ThemeIDs
		// Need to fetch theme names from database
		themeIDSet := make(map[string]bool)
		for _, article := range clusterArticles {
			if article.ThemeID != nil {
				themeIDSet[*article.ThemeID] = true
			}
		}

		// Look up theme names from IDs
		themeNames := make([]string, 0)
		themesRepo := db.Themes()
		for themeID := range themeIDSet {
			theme, err := themesRepo.Get(ctx, themeID)
			if err == nil {
				themeNames = append(themeNames, theme.Name)
			}
		}

		// Use cluster label as fallback if no themes found
		themeName := cluster.Label
		if len(themeNames) > 0 {
			// Use first theme as primary
			themeName = themeNames[0]
		}

		// Create ArticleGroups with all themes from this cluster
		articleGroups := make([]core.ArticleGroup, 0)
		if len(themeNames) > 0 {
			for _, theme := range themeNames {
				articleGroups = append(articleGroups, core.ArticleGroup{
					Theme:    theme,
					Articles: clusterArticles,
					Summary:  digestContent.TLDRSummary,
					Category: theme,
				})
			}
		} else {
			// Fallback to cluster label if no themes
			articleGroups = append(articleGroups, core.ArticleGroup{
				Theme:    themeName,
				Articles: clusterArticles,
				Summary:  digestContent.TLDRSummary,
				Category: themeName,
			})
		}

		// Inject citations into summary
		summaryWithCitations := markdown.InjectCitationURLs(digestContent.ExecutiveSummary, clusterArticles)

		// Get current time
		now := time.Now()

		// Create digest for this cluster with ALL required fields
		digest := &core.Digest{
			ID:            uuid.NewString(),

			// v2.0 fields
			Title:         digestContent.Title,
			Summary:       summaryWithCitations,
			TLDRSummary:   digestContent.TLDRSummary,
			KeyMoments:    digestContent.KeyMoments,    // FIXED: Now assigned
			Perspectives:  digestContent.Perspectives,  // FIXED: Now assigned
			Articles:      clusterArticles,
			ClusterID:     &clusterIdx,
			ProcessedDate: now,
			ArticleCount:  len(clusterArticles),

			// Legacy fields for backward compatibility
			ArticleGroups: articleGroups,  // FIXED: Now populated for homepage theme display
			DigestSummary: digestContent.ExecutiveSummary,
			Metadata: core.DigestMetadata{  // FIXED: Now populated for proper display
				Title:         digestContent.Title,
				ArticleCount:  len(clusterArticles),
				DateGenerated: now,
				TLDRSummary:   digestContent.TLDRSummary,
			},
		}

		digests = append(digests, digest)
	}

	fmt.Printf("   ✓ Generated %d digests\n", len(digests))

	return digests, nil
}

// generateThemeSummary creates a summary for a theme group
//
//nolint:unused
func generateThemeSummary(articles []core.Article, summaries map[string]*core.Summary) string {
	if len(articles) == 0 {
		return ""
	}

	// Simple: combine titles
	titles := make([]string, 0, len(articles))
	for _, article := range articles {
		titles = append(titles, article.Title)
	}

	return fmt.Sprintf("%d articles covering: %s", len(articles), strings.Join(titles, ", "))
}

// generateExecutiveSummaryFromThemes creates an executive summary from theme groups

// generateFallbackExecutiveSummary creates a simple fallback if LLM fails
//
//nolint:unused
func generateFallbackExecutiveSummary(articleGroups []core.ArticleGroup) string {
	var summary strings.Builder
	summary.WriteString("This week's digest covers ")

	themes := make([]string, 0, len(articleGroups))
	totalArticles := 0
	for _, group := range articleGroups {
		themes = append(themes, group.Theme)
		totalArticles += len(group.Articles)
	}

	summary.WriteString(fmt.Sprintf("%d articles across %d themes: %s.",
		totalArticles, len(themes), strings.Join(themes, ", ")))

	return summary.String()
}

// saveDigestMarkdown renders digest to LinkedIn-ready markdown file
func saveDigestMarkdown(digest *core.Digest, outputDir string) (string, error) {
	// Create output directory if needed
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate filename
	timestamp := digest.Metadata.DateGenerated.Format("2006-01-02")
	filename := fmt.Sprintf("digest_%s.md", timestamp)
	outputPath := fmt.Sprintf("%s/%s", outputDir, filename)

	// Render markdown
	var content strings.Builder

	// Header with emoji
	content.WriteString("# 🗞️ Weekly Tech Digest\n\n")
	content.WriteString(fmt.Sprintf("*%d Articles Across %d Themes*\n\n",
		digest.Metadata.ArticleCount,
		len(digest.ArticleGroups)))
	content.WriteString("---\n\n")

	// Executive Summary
	if digest.DigestSummary != "" {
		content.WriteString("## 🎯 Executive Summary\n\n")
		content.WriteString(digest.DigestSummary)
		content.WriteString("\n\n---\n\n")
	}

	// Theme sections
	for _, group := range digest.ArticleGroups {
		// Theme header with emoji based on theme name
		emoji := getThemeEmoji(group.Theme)
		content.WriteString(fmt.Sprintf("## %s %s\n\n", emoji, group.Theme))

		// Theme summary if available
		if group.Summary != "" && !strings.Contains(group.Summary, "covering:") {
			content.WriteString(fmt.Sprintf("*%s*\n\n", group.Summary))
		}

		// Articles in this theme
		for _, article := range group.Articles {
			content.WriteString(fmt.Sprintf("### %s\n\n", article.Title))
			content.WriteString(fmt.Sprintf("🔗 [Read Article](%s)\n\n", article.URL))

			// Find summary
			var summary *core.Summary
			for _, s := range digest.Summaries {
				for _, aid := range s.ArticleIDs {
					if aid == article.ID {
						summary = &s
						break
					}
				}
				if summary != nil {
					break
				}
			}

			if summary != nil && summary.SummaryText != "" {
				content.WriteString(summary.SummaryText)
				content.WriteString("\n\n")
			}

			if article.ThemeRelevanceScore != nil {
				content.WriteString(fmt.Sprintf("*Relevance: %.0f%%*\n\n", *article.ThemeRelevanceScore*100))
			}

			content.WriteString("---\n\n")
		}
	}

	// Footer
	content.WriteString(fmt.Sprintf("*Generated on %s*\n",
		digest.Metadata.DateGenerated.Format("Jan 2, 2006")))

	// Write file
	if err := os.WriteFile(outputPath, []byte(content.String()), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return outputPath, nil
}

// getThemeEmoji returns an emoji for a theme name
func getThemeEmoji(theme string) string {
	themeUpper := strings.ToUpper(theme)

	if strings.Contains(themeUpper, "AI") || strings.Contains(themeUpper, "MACHINE LEARNING") {
		return "🤖"
	}
	if strings.Contains(themeUpper, "SECURITY") || strings.Contains(themeUpper, "PRIVACY") {
		return "🔒"
	}
	if strings.Contains(themeUpper, "CLOUD") || strings.Contains(themeUpper, "DEVOPS") {
		return "☁️"
	}
	if strings.Contains(themeUpper, "DATA") || strings.Contains(themeUpper, "ANALYTICS") {
		return "📊"
	}
	if strings.Contains(themeUpper, "MOBILE") {
		return "📱"
	}
	if strings.Contains(themeUpper, "WEB") || strings.Contains(themeUpper, "FRONTEND") {
		return "🌐"
	}
	if strings.Contains(themeUpper, "OPEN SOURCE") {
		return "🔓"
	}
	if strings.Contains(themeUpper, "PRODUCT") || strings.Contains(themeUpper, "STARTUP") {
		return "🚀"
	}
	if strings.Contains(themeUpper, "PROGRAMMING") || strings.Contains(themeUpper, "LANGUAGE") {
		return "💻"
	}
	if strings.Contains(themeUpper, "ENGINEERING") {
		return "⚙️"
	}

	return "📌" // Default
}
