package handlers

import (
	"briefly/internal/logger"
	"briefly/internal/store"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewCacheCmd creates the cache management command
func NewCacheCmd() *cobra.Command {
	cacheCmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the article and summary cache",
		Long:  `Inspect, clean, and manage the SQLite cache for articles and summaries.`,
	}

	// Add subcommands
	cacheCmd.AddCommand(newCacheStatsCmd())
	cacheCmd.AddCommand(newCacheClearCmd())

	return cacheCmd
}

func newCacheStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show cache statistics and storage information",
		Long:  `Display detailed statistics about the cache including number of cached articles, summaries, and storage usage.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := runCacheStats(); err != nil {
				logger.Error("Failed to get cache stats", err)
				os.Exit(1)
			}
		},
	}
}

func newCacheClearCmd() *cobra.Command {
	clearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear the cache (removes all cached articles and summaries)",
		Long:  `Remove all cached articles and summaries from the SQLite database.`,
		Run: func(cmd *cobra.Command, args []string) {
			confirm, _ := cmd.Flags().GetBool("confirm")
			if err := runCacheClear(confirm); err != nil {
				logger.Error("Failed to clear cache", err)
				os.Exit(1)
			}
		},
	}

	clearCmd.Flags().Bool("confirm", false, "Skip confirmation prompt")
	return clearCmd
}

func runCacheStats() error {
	fmt.Println("📊 Cache Statistics")
	fmt.Println("==================")

	// Initialize cache store
	cacheStore, err := store.NewStore(".briefly-cache")
	if err != nil {
		return fmt.Errorf("failed to initialize cache store: %w", err)
	}
	defer func() {
		if err := cacheStore.Close(); err != nil {
			logger.Error("Failed to close cache store", err)
		}
	}()

	// Get cache statistics
	stats, err := cacheStore.GetCacheStats()
	if err != nil {
		return fmt.Errorf("failed to get cache statistics: %w", err)
	}

	// Display statistics
	fmt.Printf("📄 Articles cached: %d\n", stats.ArticleCount)
	fmt.Printf("📝 Summaries cached: %d\n", stats.SummaryCount)
	fmt.Printf("📊 Digests cached: %d\n", stats.DigestCount)
	fmt.Printf("💾 Cache size: %.2f MB\n", float64(stats.CacheSize)/1024/1024)
	fmt.Printf("📅 Last updated: %s\n", stats.LastUpdated.Format("2006-01-02 15:04:05"))
	fmt.Printf("📡 RSS feeds: %d\n", stats.FeedCount)

	return nil
}

func runCacheClear(confirm bool) error {
	if !confirm {
		fmt.Print("⚠️  This will remove all cached articles and summaries. Continue? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" && response != "yes" {
			fmt.Println("Cache clear cancelled")
			return nil
		}
	}

	fmt.Println("🗑️  Clearing cache...")

	// Initialize cache store
	cacheStore, err := store.NewStore(".briefly-cache")
	if err != nil {
		return fmt.Errorf("failed to initialize cache store: %w", err)
	}
	defer func() {
		if err := cacheStore.Close(); err != nil {
			logger.Error("Failed to close cache store", err)
		}
	}()

	// Clear the cache
	if err := cacheStore.ClearCache(); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	fmt.Println("✅ Cache cleared successfully")
	return nil
}
