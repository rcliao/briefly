package handlers

import (
	"briefly/internal/config"
	"briefly/internal/logger"
	"briefly/internal/persistence"
	"briefly/internal/sources"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// NewAggregateCmd creates the aggregate command for news aggregation
func NewAggregateCmd() *cobra.Command {
	var (
		maxArticles int
		concurrency int
		sinceHours  int
		dryRun      bool
	)

	cmd := &cobra.Command{
		Use:   "aggregate",
		Short: "Aggregate news from configured RSS feeds",
		Long: `Aggregate fetches and processes articles from all active RSS/Atom feeds.

This command:
  • Fetches new items from all active feeds
  • Stores articles and metadata in the database
  • Respects feed update frequencies (uses conditional GET)
  • Runs concurrently for better performance

Typical usage:
  • Run daily via cron: briefly aggregate --since 24
  • Test new feeds: briefly aggregate --dry-run
  • Fetch many articles: briefly aggregate --max-articles 100

Examples:
  # Aggregate last 24 hours of articles
  briefly aggregate --since 24

  # Limit to 20 articles per feed with 10 concurrent requests
  briefly aggregate --max-articles 20 --concurrency 10

  # Dry run to see what would be fetched
  briefly aggregate --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAggregate(cmd.Context(), maxArticles, concurrency, sinceHours, dryRun)
		},
	}

	cmd.Flags().IntVar(&maxArticles, "max-articles", 50, "Maximum articles to fetch per feed")
	cmd.Flags().IntVar(&concurrency, "concurrency", 5, "Number of feeds to fetch concurrently")
	cmd.Flags().IntVar(&sinceHours, "since", 24, "Only fetch articles published in the last N hours")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be fetched without storing")

	return cmd
}

func runAggregate(ctx context.Context, maxArticles, concurrency, sinceHours int, dryRun bool) error {
	log := logger.Get()
	log.Info("Starting news aggregation",
		"max_articles", maxArticles,
		"concurrency", concurrency,
		"since_hours", sinceHours,
		"dry_run", dryRun,
	)

	// Load configuration
	_, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get database connection string
	cfg := config.Get()
	dbConnStr := cfg.Database.ConnectionString
	if dbConnStr == "" {
		// Try environment variable fallback
		dbConnStr = os.Getenv("DATABASE_URL")
		if dbConnStr == "" {
			return fmt.Errorf("database connection string not configured (set database.connection_string in config or DATABASE_URL env var)")
		}
	}

	// Connect to database
	db, err := persistence.NewPostgresDB(dbConnStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	log.Info("Connected to database")

	// Create source manager
	sourceMgr := sources.NewManager(db)

	// Check if there are any active feeds
	feeds, err := sourceMgr.ListFeeds(ctx, true)
	if err != nil {
		return fmt.Errorf("failed to list feeds: %w", err)
	}

	if len(feeds) == 0 {
		log.Warn("No active feeds found. Add feeds first using: briefly feed add <url>")
		return nil
	}

	log.Info("Found active feeds", "count", len(feeds))
	for i, feed := range feeds {
		log.Info(fmt.Sprintf("  [%d] %s", i+1, feed.Title), "url", feed.URL)
	}

	if dryRun {
		log.Info("Dry run mode - no articles will be stored")
		return nil
	}

	// Prepare aggregation options
	opts := sources.AggregateOptions{
		MaxArticlesPerFeed: maxArticles,
		MaxConcurrency:     concurrency,
		Since:              time.Now().Add(-time.Duration(sinceHours) * time.Hour),
		Timeout:            10 * time.Minute,
	}

	// Run aggregation
	startTime := time.Now()
	result, err := sourceMgr.Aggregate(ctx, opts)
	duration := time.Since(startTime)

	if err != nil {
		return fmt.Errorf("aggregation failed: %w", err)
	}

	// Display results
	log.Info("Aggregation completed",
		"duration", duration.String(),
		"feeds_fetched", result.FeedsFetched,
		"feeds_skipped", result.FeedsSkipped,
		"feeds_failed", result.FeedsFailed,
		"new_articles", result.NewArticles,
		"duplicates", result.DuplicateArticles,
	)

	fmt.Println("\n📊 Aggregation Summary")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Duration:         %s\n", duration.Round(time.Millisecond))
	fmt.Printf("Feeds Fetched:    %d\n", result.FeedsFetched)
	fmt.Printf("Feeds Skipped:    %d (not modified)\n", result.FeedsSkipped)
	fmt.Printf("Feeds Failed:     %d\n", result.FeedsFailed)
	fmt.Printf("New Articles:     %d\n", result.NewArticles)
	fmt.Printf("Duplicate Articles: %d\n", result.DuplicateArticles)

	if len(result.Errors) > 0 {
		fmt.Println("\n⚠️  Errors:")
		for i, err := range result.Errors {
			fmt.Printf("  [%d] %v\n", i+1, err)
		}
	}

	if result.NewArticles > 0 {
		fmt.Printf("\n✅ Successfully aggregated %d new articles\n", result.NewArticles)
		fmt.Println("Next steps:")
		fmt.Println("  • View unprocessed articles: briefly feed list-items --unprocessed")
		fmt.Println("  • Generate digest: briefly digest --from-feeds")
	} else {
		fmt.Println("\nℹ️  No new articles found")
		fmt.Println("   Try adjusting --since or --max-articles parameters")
	}

	return nil
}
