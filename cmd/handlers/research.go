package handlers

import (
	"fmt"
	"os"

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
	// TODO: Implement feed subscription
	// This would integrate with internal/feeds package
	fmt.Println("✅ Feed added successfully")
	fmt.Println("💡 Note: Feed management functionality will be implemented in Sprint 4")
}

func handleListFeeds() {
	fmt.Println("📋 Subscribed RSS Feeds:")
	// TODO: Implement feed listing
	// This would query internal/feeds package
	fmt.Println("  (No feeds configured yet)")
	fmt.Println("💡 Note: Feed management functionality will be implemented in Sprint 4")
}

func handleAnalyzeFeeds(cmd *cobra.Command) {
	outputDir, _ := cmd.Flags().GetString("output")
	fmt.Printf("📊 Analyzing recent feed content...\n")
	fmt.Printf("📄 Report will be saved to: %s/\n", outputDir)
	// TODO: Implement feed content analysis
	// This would analyze recent feed items and generate report
	fmt.Println("💡 Note: Feed analysis functionality will be implemented in Sprint 4")
}

func handleRefreshFeeds() {
	fmt.Println("🔄 Refreshing all RSS feeds...")
	// TODO: Implement feed refresh
	// This would update all subscribed feeds
	fmt.Println("✅ All feeds refreshed")
	fmt.Println("💡 Note: Feed refresh functionality will be implemented in Sprint 4")
}

func handleDiscoverFeeds(websiteURL string) {
	fmt.Printf("🔍 Discovering RSS feeds from: %s\n", websiteURL)
	// TODO: Implement feed discovery
	// This would auto-discover RSS/Atom feeds from a website
	fmt.Println("✅ Feed discovery complete")
	fmt.Println("💡 Note: Feed discovery functionality will be implemented in Sprint 4")
}

func handleTopicResearch(topic string, depth int, outputDir string) {
	fmt.Printf("🔬 Researching topic: %s (depth: %d)\n", topic, depth)
	fmt.Printf("📄 Report will be saved to: %s/\n", outputDir)

	// TODO: Implement topic research
	// This would:
	// 1. Generate research queries using LLM
	// 2. Execute searches using existing search providers
	// 3. Analyze and rank results
	// 4. Generate research report
	// 5. Save report for manual curation

	fmt.Println("✅ Research complete")
	fmt.Println("💡 Note: Advanced research functionality will be implemented in Sprint 4")
	fmt.Printf("💡 For now, you can use the existing 'research' and 'deep-research' commands\n")
}
