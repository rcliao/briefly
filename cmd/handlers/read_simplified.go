package handlers

import (
	"briefly/internal/core"
	"briefly/internal/fetch"
	"briefly/internal/llm"
	"briefly/internal/logger"
	"briefly/internal/store"
	"briefly/internal/summarize"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// NewReadSimplifiedCmd creates the read command
func NewReadSimplifiedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read [url]",
		Short: "Quick summary of a single article",
		Long: `Generate a quick summary of a single article for fast reading.

This command provides:
- A concise summary with key points
- Estimated reading time
- Cached results for repeat reads

Examples:
  briefly read https://example.com/article
  briefly read --no-cache https://example.com/fresh-article
  briefly read --raw https://example.com/article`,
		Args: cobra.ExactArgs(1),
		Run:  readSimplifiedRun,
	}

	cmd.Flags().Bool("no-cache", false, "Disable caching (fetch fresh content)")
	cmd.Flags().Bool("raw", false, "Output summary text only, no formatting")

	return cmd
}

func readSimplifiedRun(cmd *cobra.Command, args []string) {
	startTime := time.Now()

	url := args[0]
	noCache, _ := cmd.Flags().GetBool("no-cache")
	raw, _ := cmd.Flags().GetBool("raw")

	logger.Info("Starting quick read", "url", url, "no_cache", noCache)

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		fmt.Fprintf(os.Stderr, "❌ Invalid URL: must start with http:// or https://\n")
		os.Exit(1)
	}

	if !raw {
		fmt.Println("🔧 Initializing AI client...")
	}

	llmClient, err := llm.NewClient("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to initialize AI client: %v\n", err)
		fmt.Fprintf(os.Stderr, "💡 Make sure GEMINI_API_KEY is set in your environment or .env file\n")
		os.Exit(1)
	}
	defer llmClient.Close()

	// Optional cache
	var cache *store.Store
	if !noCache {
		if s, err := store.NewStore(cacheDirFromConfig()); err == nil {
			cache = s
			defer func() { _ = cache.Close() }()
		}
	}

	if !raw {
		fmt.Printf("📖 Reading: %s\n\n", url)
	}

	ctx := cmd.Context()

	// Fetch (cache first)
	var article *core.Article
	wasCached := false
	if cache != nil {
		if cached, err := cache.GetCachedArticle(url, 24*time.Hour); err == nil && cached != nil {
			// Re-extract from stored HTML so old cache entries benefit from parser fixes
			if cached.FetchedHTML != "" {
				if err := fetch.ParseArticleContent(cached); err == nil {
					cached.EstimatedReadMinutes = fetch.CalculateReadingTime(cached)
				}
			}
			article = cached
			wasCached = true
		}
	}
	if article == nil {
		fetched, err := fetch.NewContentProcessor().ProcessArticle(ctx, url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ Failed to fetch article: %v\n", err)
			os.Exit(1)
		}
		article = fetched
		if cache != nil {
			_ = cache.SaveArticle(fetched)
		}
	}

	// Summarize (cached by content hash)
	contentHash := store.ContentHash(article.CleanedText)
	var summary *core.Summary
	if cache != nil {
		if cached, err := cache.GetCachedSummary(article.URL, contentHash, 7*24*time.Hour); err == nil && cached != nil && cached.SummaryText != "" {
			summary = cached
		}
	}
	if summary == nil {
		summarizer := summarize.NewSummarizerWithDefaults(&llmClientAdapter{client: llmClient})
		s, err := summarizer.SummarizeArticle(ctx, article)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ Failed to summarize article: %v\n", err)
			os.Exit(1)
		}
		summary = s
		if cache != nil {
			if summary.DateGenerated.IsZero() {
				summary.DateGenerated = time.Now().UTC()
			}
			_ = cache.CacheSummary(*summary, article.URL, contentHash)
		}
	}

	elapsed := time.Since(startTime)

	if raw {
		fmt.Println(summary.SummaryText)
		return
	}

	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("📄 %s\n", article.Title)
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("🔗 Source: %s\n", article.URL)
	if article.ContentType != "" {
		fmt.Printf("📦 Type: %s\n", article.ContentType)
	}
	if wasCached {
		fmt.Println("💾 Loaded from cache")
	}
	fmt.Println()
	fmt.Println("📝 Summary:")
	fmt.Println(summary.SummaryText)
	fmt.Println()
	fmt.Println("ℹ️  Info:")
	fmt.Printf("   • Estimated Read Time: %d min\n", article.EstimatedReadMinutes)
	fmt.Printf("   • Processed in: %v\n", elapsed.Round(time.Millisecond))
	fmt.Println("═══════════════════════════════════════")

	logger.Info("Quick read completed", "url", url, "cached", wasCached, "duration", elapsed)
}
