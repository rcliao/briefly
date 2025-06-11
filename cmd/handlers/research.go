package handlers

import (
	"briefly/internal/config"
	"briefly/internal/core"
	"briefly/internal/llm"
	"briefly/internal/search"
	"briefly/internal/services"
	"briefly/internal/store"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// NewResearchCmd creates the consolidated research command
func NewResearchCmd() *cobra.Command {
	researchCmd := &cobra.Command{
		Use:   "research [topic|URL]",
		Short: "Perform research on topics or manage RSS feeds",
		Long: `Consolidated research command that handles:
- Topic research with configurable depth
- RSS feed subscription and management  
- Feed content analysis and report generation
- Research report output for manual curation

Examples:
  # Core research functionality
  briefly research "AI coding tools"           # Generate research report
  briefly research "AI coding tools" --depth 3 # Deep research with iterations
  
  # Feed management
  briefly research --add-feed URL              # Subscribe to RSS feed
  briefly research --list-feeds                # Show subscribed feeds
  briefly research --from-feeds                # Analyze feed content → report
  briefly research --refresh-feeds             # Update all feeds
  briefly research --discover-feeds URL        # Auto-discover feeds from site`,
		Run: researchRunFunc,
	}

	// Add flags for different research modes
	researchCmd.Flags().Int("depth", 1, "Research depth (1-5, higher = more comprehensive)")
	researchCmd.Flags().String("output", "research", "Output directory for research reports")

	// Feed management flags
	researchCmd.Flags().String("add-feed", "", "Subscribe to RSS feed URL")
	researchCmd.Flags().Bool("list-feeds", false, "List all subscribed feeds")
	researchCmd.Flags().Bool("from-feeds", false, "Analyze recent feed content")
	researchCmd.Flags().Bool("refresh-feeds", false, "Update all feeds")
	researchCmd.Flags().String("discover-feeds", "", "Auto-discover feeds from website URL")

	// Research configuration
	researchCmd.Flags().Int("max-results", 20, "Maximum search results per query")
	researchCmd.Flags().String("format", "markdown", "Report format: markdown, json")

	return researchCmd
}

func researchRunFunc(cmd *cobra.Command, args []string) {
	// Check for feed management flags first
	if addFeed, _ := cmd.Flags().GetString("add-feed"); addFeed != "" {
		handleAddFeed(addFeed)
		return
	}

	if listFeeds, _ := cmd.Flags().GetBool("list-feeds"); listFeeds {
		handleListFeeds()
		return
	}

	if fromFeeds, _ := cmd.Flags().GetBool("from-feeds"); fromFeeds {
		handleAnalyzeFeeds(cmd)
		return
	}

	if refreshFeeds, _ := cmd.Flags().GetBool("refresh-feeds"); refreshFeeds {
		handleRefreshFeeds()
		return
	}

	if discoverURL, _ := cmd.Flags().GetString("discover-feeds"); discoverURL != "" {
		handleDiscoverFeeds(discoverURL)
		return
	}

	// Handle topic research
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: research command requires a topic or feed management flag\n")
		_ = cmd.Help()
		os.Exit(1)
	}

	topic := args[0]
	depth, _ := cmd.Flags().GetInt("depth")
	outputDir, _ := cmd.Flags().GetString("output")

	handleTopicResearch(topic, depth, outputDir)
}

func handleAddFeed(feedURL string) {
	fmt.Printf("🔗 Adding RSS feed: %s\n", feedURL)

	// Initialize services
	dataDir := ".briefly-cache"
	dataStore, err := store.NewStore(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize storage: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = dataStore.Close() }()

	llmClient, err := llm.NewClient("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize LLM client: %v\n", err)
		os.Exit(1)
	}

	feedService := services.NewFeedService(dataStore, llmClient)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := feedService.AddFeed(ctx, feedURL); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to add feed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Feed added successfully")
}

func handleListFeeds() {
	fmt.Println("📋 Subscribed RSS Feeds:")

	// Initialize services
	dataDir := ".briefly-cache"
	dataStore, err := store.NewStore(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize storage: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = dataStore.Close() }()

	llmClient, err := llm.NewClient("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize LLM client: %v\n", err)
		os.Exit(1)
	}

	feedService := services.NewFeedService(dataStore, llmClient)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	feeds, err := feedService.ListFeeds(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to list feeds: %v\n", err)
		os.Exit(1)
	}

	if len(feeds) == 0 {
		fmt.Println("  (No feeds configured yet)")
		return
	}

	for i, feed := range feeds {
		status := "🟢"
		if !feed.Active {
			status = "🔴"
		} else if feed.ErrorCount > 0 {
			status = "🟡"
		}

		fmt.Printf("  %d. %s %s\n", i+1, status, feed.Title)
		fmt.Printf("     URL: %s\n", feed.URL)
		if !feed.LastFetched.IsZero() {
			fmt.Printf("     Last fetched: %s\n", feed.LastFetched.Format("2006-01-02 15:04"))
		}
		if feed.ErrorCount > 0 {
			fmt.Printf("     Errors: %d (last: %s)\n", feed.ErrorCount, feed.LastError)
		}
		fmt.Println()
	}
}

func handleAnalyzeFeeds(cmd *cobra.Command) {
	outputDir, _ := cmd.Flags().GetString("output")
	fmt.Printf("📊 Analyzing recent feed content...\n")

	// Initialize services
	dataDir := ".briefly-cache"
	dataStore, err := store.NewStore(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize storage: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = dataStore.Close() }()

	llmClient, err := llm.NewClient("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize LLM client: %v\n", err)
		os.Exit(1)
	}

	feedService := services.NewFeedService(dataStore, llmClient)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	report, err := feedService.AnalyzeFeedContent(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to analyze feed content: %v\n", err)
		os.Exit(1)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	// Generate report filename
	filename := fmt.Sprintf("feed-analysis-%s.md", report.DateGenerated.Format("2006-01-02-15-04"))
	reportPath := filepath.Join(outputDir, filename)

	// Generate report content
	reportContent := generateFeedAnalysisReport(report)

	// Write report to file
	if err := os.WriteFile(reportPath, []byte(reportContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to write report: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📄 Report saved to: %s\n", reportPath)
	fmt.Printf("✅ Analysis complete: %d feeds, %d items analyzed\n", report.FeedsAnalyzed, report.ItemsAnalyzed)
}

func handleRefreshFeeds() {
	fmt.Println("🔄 Refreshing all RSS feeds...")

	// Initialize services
	dataDir := ".briefly-cache"
	dataStore, err := store.NewStore(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize storage: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = dataStore.Close() }()

	llmClient, err := llm.NewClient("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize LLM client: %v\n", err)
		os.Exit(1)
	}

	feedService := services.NewFeedService(dataStore, llmClient)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if err := feedService.RefreshFeeds(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to refresh feeds: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ All feeds refreshed")
}

func handleDiscoverFeeds(websiteURL string) {
	fmt.Printf("🔍 Discovering RSS feeds from: %s\n", websiteURL)

	// Initialize services
	dataDir := ".briefly-cache"
	dataStore, err := store.NewStore(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize storage: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = dataStore.Close() }()

	llmClient, err := llm.NewClient("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize LLM client: %v\n", err)
		os.Exit(1)
	}

	feedService := services.NewFeedService(dataStore, llmClient)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	discoveredFeeds, err := feedService.DiscoverFeeds(ctx, websiteURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to discover feeds: %v\n", err)
		os.Exit(1)
	}

	if len(discoveredFeeds) == 0 {
		fmt.Println("❌ No RSS/Atom feeds found on this website")
		return
	}

	fmt.Printf("✅ Found %d potential feed(s):\n", len(discoveredFeeds))
	for i, feedURL := range discoveredFeeds {
		fmt.Printf("  %d. %s\n", i+1, feedURL)
	}
	fmt.Println("\nUse 'briefly research --add-feed URL' to subscribe to any of these feeds.")
}

func handleTopicResearch(topic string, depth int, outputDir string) {
	fmt.Printf("🔬 Researching topic: %s (depth: %d)\n", topic, depth)

	// Initialize services
	dataDir := ".briefly-cache"
	dataStore, err := store.NewStore(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize storage: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = dataStore.Close() }()

	llmClient, err := llm.NewClient("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize LLM client: %v\n", err)
		os.Exit(1)
	}

	// Initialize search provider with improved configuration
	searchProvider, err := createSearchProvider()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize search provider: %v\n", err)
		os.Exit(1)
	}

	researchService := services.NewResearchService(llmClient, searchProvider)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second) // 3 minutes for research
	defer cancel()

	fmt.Println("🔍 Performing research...")
	report, err := researchService.PerformResearch(ctx, topic, depth)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to perform research: %v\n", err)
		os.Exit(1)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	// Generate report filename
	filename := fmt.Sprintf("research-%s-%s.md",
		sanitizeFilename(topic),
		report.DateGenerated.Format("2006-01-02-15-04"))
	reportPath := filepath.Join(outputDir, filename)

	// Generate report content
	reportContent := generateResearchReport(report)

	// Write report to file
	if err := os.WriteFile(reportPath, []byte(reportContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to write report: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📄 Report saved to: %s\n", reportPath)
	fmt.Printf("✅ Research complete: %d results found (relevance: %.2f)\n",
		report.TotalResults, report.RelevanceScore)
}

// generateResearchReport creates a markdown report from research results
func generateResearchReport(report *core.ResearchReport) string {
	content := fmt.Sprintf(`# Research Report: %s

**Generated:** %s
**Depth:** %d
**Total Results:** %d
**Relevance Score:** %.2f

## Summary

%s

## Generated Queries

`, report.Query, report.DateGenerated.Format("2006-01-02 15:04 MST"),
		report.Depth, report.TotalResults, report.RelevanceScore, report.Summary)

	for i, query := range report.GeneratedQueries {
		content += fmt.Sprintf("%d. %s\n", i+1, query)
	}

	content += "\n## Research Results\n\n"

	for i, result := range report.Results {
		if i >= 20 { // Limit to top 20 results in the report
			break
		}

		content += fmt.Sprintf("### %d. %s\n\n", i+1, result.Title)
		content += fmt.Sprintf("**URL:** %s\n", result.URL)
		content += fmt.Sprintf("**Relevance:** %.2f | **Source:** %s\n\n", result.Relevance, result.Source)
		content += fmt.Sprintf("%s\n\n", result.Snippet)

		if len(result.Keywords) > 0 {
			content += fmt.Sprintf("**Keywords:** %s\n\n", strings.Join(result.Keywords, ", "))
		}

		content += "---\n\n"
	}

	content += "## Manual Curation URLs\n\n"
	content += "Copy and paste interesting URLs below into your digest input file:\n\n"

	for i, result := range report.Results {
		if i >= 10 { // Top 10 URLs for easy copy-paste
			break
		}
		content += fmt.Sprintf("- %s\n", result.URL)
	}

	return content
}

// generateFeedAnalysisReport creates a markdown report from feed analysis
func generateFeedAnalysisReport(report *core.FeedAnalysisReport) string {
	content := fmt.Sprintf(`# Feed Analysis Report

**Generated:** %s
**Feeds Analyzed:** %d
**Items Analyzed:** %d
**Quality Score:** %.2f

## Summary

%s

## Top Topics

`, report.DateGenerated.Format("2006-01-02 15:04 MST"),
		report.FeedsAnalyzed, report.ItemsAnalyzed, report.QualityScore, report.Summary)

	for i, topic := range report.TopTopics {
		content += fmt.Sprintf("%d. %s\n", i+1, topic)
	}

	content += "\n## Trending Keywords\n\n"

	for i, keyword := range report.TrendingKeywords {
		if i >= 15 { // Limit to top 15 keywords
			break
		}
		content += fmt.Sprintf("- %s\n", keyword)
	}

	content += "\n## Recommended Items\n\n"

	for i, item := range report.RecommendedItems {
		if i >= 10 { // Limit to top 10 recommendations
			break
		}

		content += fmt.Sprintf("### %d. %s\n\n", i+1, item.Title)
		content += fmt.Sprintf("**URL:** %s\n", item.Link)
		content += fmt.Sprintf("**Published:** %s\n\n", item.Published.Format("2006-01-02 15:04"))
		content += fmt.Sprintf("%s\n\n", item.Description)
		content += "---\n\n"
	}

	content += "## Manual Curation URLs\n\n"
	content += "Copy and paste interesting URLs below into your digest input file:\n\n"

	for i, item := range report.RecommendedItems {
		if i >= 10 { // Top 10 URLs for easy copy-paste
			break
		}
		content += fmt.Sprintf("- %s\n", item.Link)
	}

	return content
}

// sanitizeFilename removes unsafe characters from filenames
func sanitizeFilename(filename string) string {
	// Replace unsafe characters with hyphens
	unsafe := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	clean := filename
	for _, char := range unsafe {
		clean = strings.ReplaceAll(clean, char, "-")
	}

	// Remove multiple consecutive hyphens
	for strings.Contains(clean, "--") {
		clean = strings.ReplaceAll(clean, "--", "-")
	}

	// Trim hyphens from start and end
	clean = strings.Trim(clean, "-")

	// Limit length
	if len(clean) > 50 {
		clean = clean[:50]
	}

	return clean
}

// createSearchProvider creates a search provider based on centralized configuration
func createSearchProvider() (search.Provider, error) {
	factory := search.NewProviderFactory()

	// Try Google Custom Search first
	if config.HasValidGoogleSearch() {
		provider, err := factory.CreateProvider(search.ProviderTypeGoogle, config.GetSearchProviderConfig("google"))
		if err != nil {
			return nil, fmt.Errorf("failed to create Google Custom Search provider: %w", err)
		}
		fmt.Println("🔍 Using Google Custom Search")
		return provider, nil
	}

	// Try SerpAPI next
	if config.HasValidSerpAPI() {
		provider, err := factory.CreateProvider(search.ProviderTypeSerpAPI, config.GetSearchProviderConfig("serpapi"))
		if err != nil {
			return nil, fmt.Errorf("failed to create SerpAPI provider: %w", err)
		}
		fmt.Println("🔍 Using SerpAPI")
		return provider, nil
	}

	// Fallback to DuckDuckGo with helpful guidance
	provider, err := factory.CreateProvider(search.ProviderTypeDuckDuckGo, map[string]string{})
	if err != nil {
		return nil, fmt.Errorf("failed to create DuckDuckGo provider: %w", err)
	}

	fmt.Println("⚠️  Using DuckDuckGo search (may be limited by rate limiting)")
	fmt.Println()
	fmt.Println("💡 For better search results, configure one of these providers:")
	fmt.Println("   Google Custom Search: Set GOOGLE_CUSTOM_SEARCH_API_KEY and GOOGLE_CUSTOM_SEARCH_ID")
	fmt.Println("   SerpAPI: Set SERPAPI_API_KEY")
	fmt.Println()
	fmt.Println("   Setup guides:")
	fmt.Println("   - Google: https://developers.google.com/custom-search/v1/introduction")
	fmt.Println("   - SerpAPI: https://serpapi.com/dashboard")
	fmt.Println()

	return provider, nil
}
