package handlers

import (
	"briefly/internal/config"
	"briefly/internal/logger"
	"briefly/internal/persistence"
	"briefly/internal/sources"
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// NewFeedCmd creates the feed management command
func NewFeedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feed",
		Short: "Manage RSS/Atom feed sources",
		Long: `Manage RSS/Atom feed sources for news aggregation.

Subcommands:
  add       Add a new feed source
  remove    Remove a feed source
  list      List all feed sources
  enable    Enable a feed
  disable   Disable a feed
  stats     Show statistics for feeds`,
	}

	cmd.AddCommand(newFeedAddCmd())
	cmd.AddCommand(newFeedRemoveCmd())
	cmd.AddCommand(newFeedListCmd())
	cmd.AddCommand(newFeedEnableCmd())
	cmd.AddCommand(newFeedDisableCmd())
	cmd.AddCommand(newFeedStatsCmd())

	return cmd
}

func newFeedAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <feed-url>",
		Short: "Add a new RSS/Atom feed source",
		Long: `Add a new feed source for news aggregation.

The feed URL must be a valid RSS or Atom feed. The command will:
  • Validate the feed format
  • Fetch initial metadata
  • Store feed in database
  • Activate feed for aggregation

Examples:
  briefly feed add https://hnrss.org/newest
  briefly feed add https://arxiv.org/rss/cs.AI`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			feedURL := args[0]
			return runFeedAdd(cmd.Context(), feedURL)
		},
	}
}

func newFeedRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <feed-id>",
		Short: "Remove a feed source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			feedID := args[0]
			return runFeedRemove(cmd.Context(), feedID)
		},
	}
}

func newFeedListCmd() *cobra.Command {
	var showInactive bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all feed sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFeedList(cmd.Context(), showInactive)
		},
	}

	cmd.Flags().BoolVar(&showInactive, "all", false, "Show inactive feeds as well")

	return cmd
}

func newFeedEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable <feed-id>",
		Short: "Enable a feed source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			feedID := args[0]
			return runFeedToggle(cmd.Context(), feedID, true)
		},
	}
}

func newFeedDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <feed-id>",
		Short: "Disable a feed source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			feedID := args[0]
			return runFeedToggle(cmd.Context(), feedID, false)
		},
	}
}

func newFeedStatsCmd() *cobra.Command {
	var feedID string

	cmd := &cobra.Command{
		Use:   "stats [feed-id]",
		Short: "Show feed statistics",
		Long: `Show statistics for feeds.

If no feed ID is provided, shows summary statistics for all feeds.
Otherwise, shows detailed statistics for the specified feed.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				feedID = args[0]
			}
			return runFeedStats(cmd.Context(), feedID)
		},
	}

	return cmd
}

// Implementation functions

// getDatabase is a helper function to load config and connect to database
func getDatabase() (persistence.Database, error) {
	_, err := config.Load(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	cfg := config.Get()
	dbConnStr := cfg.Database.ConnectionString
	if dbConnStr == "" {
		// Try environment variable fallback
		dbConnStr = os.Getenv("DATABASE_URL")
		if dbConnStr == "" {
			return nil, fmt.Errorf("database connection string not configured (set database.connection_string in config or DATABASE_URL env var)")
		}
	}

	db, err := persistence.NewPostgresDB(dbConnStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

func runFeedAdd(ctx context.Context, feedURL string) error {
	log := logger.Get()
	log.Info("Adding new feed", "url", feedURL)

	db, err := getDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	sourceMgr := sources.NewManager(db)
	feed, err := sourceMgr.AddFeed(ctx, feedURL)
	if err != nil {
		return fmt.Errorf("failed to add feed: %w", err)
	}

	fmt.Println("✅ Feed added successfully")
	fmt.Printf("   ID:    %s\n", feed.ID)
	fmt.Printf("   Title: %s\n", feed.Title)
	fmt.Printf("   URL:   %s\n", feed.URL)
	fmt.Println("\nNext steps:")
	fmt.Println("  • Run aggregation: briefly aggregate")
	fmt.Println("  • View feed stats: briefly feed stats", feed.ID)

	return nil
}

func runFeedRemove(ctx context.Context, feedID string) error {
	log := logger.Get()
	log.Info("Removing feed", "id", feedID)

	db, err := getDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	sourceMgr := sources.NewManager(db)
	if err := sourceMgr.RemoveFeed(ctx, feedID); err != nil {
		return fmt.Errorf("failed to remove feed: %w", err)
	}

	fmt.Println("✅ Feed removed successfully")
	return nil
}

func runFeedList(ctx context.Context, showInactive bool) error {
	db, err := getDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	sourceMgr := sources.NewManager(db)
	feeds, err := sourceMgr.ListFeeds(ctx, !showInactive)
	if err != nil {
		return fmt.Errorf("failed to list feeds: %w", err)
	}

	if len(feeds) == 0 {
		fmt.Println("No feeds found")
		fmt.Println("\nAdd your first feed:")
		fmt.Println("  briefly feed add <feed-url>")
		return nil
	}

	// Display feeds in a table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tTitle\tActive\tLast Fetched\tError Count\n")
	fmt.Fprintf(w, "━━━━━━━━━━\t━━━━━━━━━━━━━━━━━━━━\t━━━━━━\t━━━━━━━━━━━━\t━━━━━━━━━━━\n")

	for _, feed := range feeds {
		status := "✓"
		if !feed.Active {
			status = "✗"
		}

		lastFetched := "Never"
		if !feed.LastFetched.IsZero() {
			lastFetched = feed.LastFetched.Format("2006-01-02 15:04")
		}

		titleShort := feed.Title
		if len(titleShort) > 40 {
			titleShort = titleShort[:37] + "..."
		}

		errorCount := fmt.Sprintf("%d", feed.ErrorCount)
		if feed.ErrorCount > 0 {
			errorCount = fmt.Sprintf("⚠️  %d", feed.ErrorCount)
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			feed.ID[:8]+"...", titleShort, status, lastFetched, errorCount,
		)
	}
	w.Flush()

	fmt.Printf("\nTotal feeds: %d\n", len(feeds))
	if !showInactive {
		fmt.Println("Use --all to show inactive feeds")
	}

	return nil
}

func runFeedToggle(ctx context.Context, feedID string, active bool) error {
	log := logger.Get()
	action := "Enabling"
	if !active {
		action = "Disabling"
	}
	log.Info(action+" feed", "id", feedID)

	db, err := getDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	sourceMgr := sources.NewManager(db)
	if err := sourceMgr.ToggleFeed(ctx, feedID, active); err != nil {
		return fmt.Errorf("failed to toggle feed: %w", err)
	}

	status := "enabled"
	if !active {
		status = "disabled"
	}
	fmt.Printf("✅ Feed %s\n", status)
	return nil
}

func runFeedStats(ctx context.Context, feedID string) error {
	db, err := getDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	if feedID == "" {
		// Show summary statistics for all feeds
		return showAllFeedStats(ctx, db)
	}

	// Show detailed statistics for specific feed
	sourceMgr := sources.NewManager(db)
	stats, err := sourceMgr.GetFeedStats(ctx, feedID)
	if err != nil {
		return fmt.Errorf("failed to get feed stats: %w", err)
	}

	fmt.Println("📊 Feed Statistics")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Title:            %s\n", stats.Feed.Title)
	fmt.Printf("URL:              %s\n", stats.Feed.URL)
	fmt.Printf("Active:           %v\n", stats.Feed.Active)
	fmt.Printf("Total Items:      %d\n", stats.TotalItems)
	fmt.Printf("Processed Items:  %d\n", stats.ProcessedItems)
	fmt.Printf("Unprocessed:      %d\n", stats.UnprocessedItems)
	if !stats.LatestItem.IsZero() {
		fmt.Printf("Latest Item:      %s\n", stats.LatestItem.Format("2006-01-02 15:04"))
	}
	if !stats.OldestItem.IsZero() {
		fmt.Printf("Oldest Item:      %s\n", stats.OldestItem.Format("2006-01-02 15:04"))
	}
	if stats.Feed.ErrorCount > 0 {
		fmt.Printf("\n⚠️  Error Count:    %d\n", stats.Feed.ErrorCount)
		fmt.Printf("   Last Error:     %s\n", stats.Feed.LastError)
	}

	return nil
}

func showAllFeedStats(ctx context.Context, db persistence.Database) error {
	feeds, err := db.Feeds().List(ctx, persistence.ListOptions{Limit: 1000})
	if err != nil {
		return fmt.Errorf("failed to list feeds: %w", err)
	}

	if len(feeds) == 0 {
		fmt.Println("No feeds configured")
		return nil
	}

	fmt.Println("📊 Feed Statistics Summary")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	totalItems := 0
	activeFeeds := 0
	errorFeeds := 0

	for _, feed := range feeds {
		if feed.Active {
			activeFeeds++
		}
		if feed.ErrorCount > 0 {
			errorFeeds++
		}

		// Get item count for this feed
		items, _ := db.FeedItems().GetByFeedID(ctx, feed.ID, 10000)
		totalItems += len(items)
	}

	fmt.Printf("Total Feeds:      %d\n", len(feeds))
	fmt.Printf("Active Feeds:     %d\n", activeFeeds)
	fmt.Printf("Inactive Feeds:   %d\n", len(feeds)-activeFeeds)
	fmt.Printf("Feeds with Errors: %d\n", errorFeeds)
	fmt.Printf("Total Items:      %d\n", totalItems)

	fmt.Println("\nUse 'briefly feed stats <feed-id>' for detailed statistics")

	return nil
}
