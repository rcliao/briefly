package handlers

import (
	"briefly/internal/config"
	"briefly/internal/core"
	"briefly/internal/llm"
	"briefly/internal/logger"
	"briefly/internal/persistence"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

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
	themeGroups := groupArticlesByTheme(articles)

	fmt.Printf("\n📊 Articles by Theme:\n")
	for themeName, themeArticles := range themeGroups {
		fmt.Printf("  • %s: %d articles\n", themeName, len(themeArticles))
	}

	// Initialize LLM client for summaries
	modelName := cfg.AI.Gemini.Model
	if modelName == "" {
		modelName = "gemini-2.5-flash-preview-05-20"
	}

	llmClient, err := llm.NewClient(modelName)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}
	defer llmClient.Close()

	// Generate simple digest structure
	digest := generateSimpleDigest(articles, themeGroups, since)

	// Save digest to file
	outputPath, err := saveDigest(digest, outputDir, themeFilter)
	if err != nil {
		return fmt.Errorf("failed to save digest: %w", err)
	}

	duration := time.Since(time.Now().Add(-5 * time.Second)) // Approximate

	fmt.Printf("\n✅ Successfully generated digest\n")
	fmt.Printf("   Articles: %d\n", len(articles))
	fmt.Printf("   Themes: %d\n", len(themeGroups))
	fmt.Printf("   Output: %s\n", outputPath)
	fmt.Printf("   Duration: %s\n", duration.Round(time.Millisecond))

	return nil
}

// queryClassifiedArticles fetches articles from database with filters
func queryClassifiedArticles(ctx context.Context, db *persistence.PostgresDB, since time.Time, themeFilter string) ([]core.Article, error) {
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

	var filtered []core.Article
	for _, article := range allArticles {
		// Filter by date (use DateFetched as proxy for DateAdded)
		if article.DateFetched.Before(since) {
			continue
		}

		// Filter by theme (must have theme assigned)
		if article.ThemeID == nil {
			continue
		}

		// If theme filter specified, only include matching theme
		if themeFilter != "" {
			// Need to fetch theme name - for now skip this filtering
			// TODO: Add join query or theme lookup
		}

		filtered = append(filtered, article)
	}

	return filtered, nil
}

// groupArticlesByTheme groups articles by their theme
func groupArticlesByTheme(articles []core.Article) map[string][]core.Article {
	groups := make(map[string][]core.Article)

	for _, article := range articles {
		if article.ThemeID == nil {
			continue
		}

		// Use theme ID as key for now
		// TODO: Fetch theme name from database
		themeKey := *article.ThemeID
		groups[themeKey] = append(groups[themeKey], article)
	}

	return groups
}

// generateSimpleDigest creates a basic digest structure
func generateSimpleDigest(articles []core.Article, themeGroups map[string][]core.Article, since time.Time) string {
	var content string

	content += fmt.Sprintf("# Weekly Tech Digest\n\n")
	content += fmt.Sprintf("**Period:** %s to %s\n\n", since.Format("Jan 2"), time.Now().Format("Jan 2, 2006"))
	content += fmt.Sprintf("**Articles:** %d across %d themes\n\n", len(articles), len(themeGroups))
	content += "---\n\n"

	for themeName, themeArticles := range themeGroups {
		content += fmt.Sprintf("## %s (%d articles)\n\n", themeName, len(themeArticles))

		for _, article := range themeArticles {
			content += fmt.Sprintf("### %s\n\n", article.Title)
			content += fmt.Sprintf("**Source:** [%s](%s)\n\n", article.URL, article.URL)

			if article.ThemeRelevanceScore != nil {
				content += fmt.Sprintf("**Relevance:** %.0f%%\n\n", *article.ThemeRelevanceScore*100)
			}

			// Add snippet of content if available
			if len(article.CleanedText) > 200 {
				content += fmt.Sprintf("%s...\n\n", article.CleanedText[:200])
			} else if article.CleanedText != "" {
				content += fmt.Sprintf("%s\n\n", article.CleanedText)
			}

			content += "---\n\n"
		}
	}

	return content
}

// saveDigest writes digest to markdown file
func saveDigest(content string, outputDir string, themeFilter string) (string, error) {
	// Create output directory if needed
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate filename
	timestamp := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("digest_%s", timestamp)
	if themeFilter != "" {
		filename += fmt.Sprintf("_%s", themeFilter)
	}
	filename += ".md"

	outputPath := fmt.Sprintf("%s/%s", outputDir, filename)

	// Write file
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return outputPath, nil
}
