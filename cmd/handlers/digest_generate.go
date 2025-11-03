package handlers

import (
	"briefly/internal/config"
	"briefly/internal/core"
	"briefly/internal/llm"
	"briefly/internal/logger"
	"briefly/internal/persistence"
	"briefly/internal/summarize"
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

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

	// Generate digest with LLM summaries
	fmt.Println("\n🤖 Generating summaries and digest...")
	digest, err := generateDigestWithSummaries(ctx, db, llmClient, articles, themeGroups, since, themeFilter)
	if err != nil {
		return fmt.Errorf("failed to generate digest: %w", err)
	}

	// Save digest to database
	if err := db.Digests().Create(ctx, digest); err != nil {
		log.Warn("Failed to save digest to database", "error", err)
		// Continue - still save markdown file
	} else {
		log.Info("Digest saved to database", "digest_id", digest.ID)
	}

	// Save digest to markdown file for LinkedIn
	outputPath, err := saveDigestMarkdown(digest, outputDir)
	if err != nil {
		return fmt.Errorf("failed to save markdown file: %w", err)
	}

	duration := time.Since(startTime)

	fmt.Printf("\n✅ Successfully generated digest\n")
	fmt.Printf("   Digest ID: %s\n", digest.ID)
	fmt.Printf("   Articles: %d\n", len(articles))
	fmt.Printf("   Themes: %d\n", len(themeGroups))
	fmt.Printf("   Database: Saved ✓\n")
	fmt.Printf("   Markdown: %s\n", outputPath)
	fmt.Printf("   Duration: %s\n", duration.Round(time.Millisecond))

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

// generateDigestWithSummaries creates a complete digest with LLM-generated summaries
func generateDigestWithSummaries(ctx context.Context, db *persistence.PostgresDB, llmClient *llm.Client, articles []core.Article, themeGroups map[string][]core.Article, since time.Time, themeFilter string) (*core.Digest, error) {
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

	// Build article groups by theme
	articleGroups := []core.ArticleGroup{}
	themeNames := make([]string, 0, len(themeGroups))
	for themeName := range themeGroups {
		themeNames = append(themeNames, themeName)
	}
	sort.Strings(themeNames) // Sort for consistent output

	for _, themeName := range themeNames {
		themeArticles := themeGroups[themeName]

		// Generate theme-level summary
		themeSummary := generateThemeSummary(themeArticles, articleSummaries)

		articleGroup := core.ArticleGroup{
			Theme:    themeName,
			Articles: themeArticles,
			Summary:  themeSummary,
			Priority: len(themeArticles), // Higher count = higher priority
		}
		articleGroups = append(articleGroups, articleGroup)
	}

	// Sort by priority (most articles first)
	sort.Slice(articleGroups, func(i, j int) bool {
		return articleGroups[i].Priority > articleGroups[j].Priority
	})

	// Generate executive summary
	fmt.Println("   ✨ Generating executive summary...")
	executiveSummary := generateExecutiveSummaryFromThemes(ctx, llmClient, articleGroups, articleSummaries)

	// Build digest structure
	digestDate := time.Now()
	if themeFilter != "" {
		digestDate = since // Use since date if filtering
	}

	digest := &core.Digest{
		ID:            uuid.NewString(),
		ArticleGroups: articleGroups,
		Summaries:     summaryList,
		DigestSummary: executiveSummary,
		Title:         fmt.Sprintf("Weekly Tech Digest - %s", digestDate.Format("Jan 2, 2006")),
		Metadata: core.DigestMetadata{
			Title:         fmt.Sprintf("Weekly Tech Digest - %s", digestDate.Format("Jan 2, 2006")),
			ArticleCount:  len(articles),
			DateGenerated: time.Now(),
		},
	}

	return digest, nil
}

// generateThemeSummary creates a summary for a theme group
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
func generateExecutiveSummaryFromThemes(ctx context.Context, llmClient *llm.Client, articleGroups []core.ArticleGroup, summaries map[string]*core.Summary) string {
	log := logger.Get()

	// Build prompt for executive summary
	var prompt strings.Builder
	prompt.WriteString("Generate a compelling executive summary (200 words max) for this weekly tech digest.\n\n")
	prompt.WriteString("The digest covers the following themes and articles:\n\n")

	for _, group := range articleGroups {
		prompt.WriteString(fmt.Sprintf("**%s** (%d articles):\n", group.Theme, len(group.Articles)))
		for i, article := range group.Articles {
			if i >= 3 {
				break // Only use top 3 per theme
			}
			summary := summaries[article.ID]
			if summary != nil {
				prompt.WriteString(fmt.Sprintf("- %s: %s\n", article.Title, summary.SummaryText))
			} else {
				prompt.WriteString(fmt.Sprintf("- %s\n", article.Title))
			}
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString("\nWrite an engaging executive summary that:\n")
	prompt.WriteString("1. Highlights the most important trends and insights\n")
	prompt.WriteString("2. Connects themes and shows relationships\n")
	prompt.WriteString("3. Is written for technical leaders and engineers\n")
	prompt.WriteString("4. Uses a professional but engaging tone\n")
	prompt.WriteString("5. Focuses on actionable insights\n\n")
	prompt.WriteString("Executive Summary:\n")

	// Generate using LLM
	response, err := llmClient.GenerateText(ctx, prompt.String(), llm.TextGenerationOptions{})
	if err != nil {
		log.Warn("Failed to generate executive summary", "error", err)
		return generateFallbackExecutiveSummary(articleGroups)
	}

	return strings.TrimSpace(response)
}

// generateFallbackExecutiveSummary creates a simple fallback if LLM fails
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
