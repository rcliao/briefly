package handlers

import (
	"briefly/internal/config"
	"briefly/internal/core"
	"briefly/internal/fetch"
	"briefly/internal/interactive"
	"briefly/internal/llm"
	"briefly/internal/logger"
	"briefly/internal/store"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// SummarizeResult represents the output structure for the summarize command
type SummarizeResult struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Summary     string    `json:"summary"`
	Highlights  string    `json:"highlights,omitempty"`
	ContentType string    `json:"content_type"`
	ProcessedAt time.Time `json:"processed_at"`
}

// NewSummarizeCmd creates the summarize command for quick article analysis
func NewSummarizeCmd() *cobra.Command {
	summarizeCmd := &cobra.Command{
		Use:   "summarize [URL]",
		Short: "Quickly summarize a single article with key highlights",
		Long: `Summarize a single article or webpage with AI-powered analysis.

This command fetches content from a URL, extracts the main text, and generates
a detailed summary with key highlights/moments from the article by default.

Features:
- Smart content extraction from web pages, PDFs, and YouTube videos
- Detailed summary generation with key highlights (default)
- Key moments extraction with quotes and explanations
- Multiple output formats: terminal display, JSON, or markdown file
- Intelligent caching to avoid redundant API calls

Examples:
  # Basic summarization with highlights and detailed style (default)
  briefly summarize https://example.com/article

  # Disable highlights for a simpler summary
  briefly summarize https://example.com/article --highlights=false

  # Output as JSON for integration
  briefly summarize https://example.com/article --format json

  # Save to markdown file
  briefly summarize https://example.com/article --format markdown --output summary.md

  # Use a different summary style (only applies when highlights are disabled)
  briefly summarize https://example.com/article --highlights=false --style brief
  briefly summarize https://example.com/article --highlights=false --style standard`,
		Args: cobra.ExactArgs(1),
		Run:  summarizeRunFunc,
	}

	// Add flags
	summarizeCmd.Flags().Bool("highlights", true, "Extract key moments and highlights with quotes")
	summarizeCmd.Flags().StringP("format", "f", "terminal", "Output format: terminal, json, markdown")
	summarizeCmd.Flags().StringP("output", "o", "", "Output file path (for json/markdown formats)")
	summarizeCmd.Flags().String("style", "detailed", "Summary style: brief, standard, detailed")
	summarizeCmd.Flags().Bool("no-cache", false, "Skip cache and force fresh content fetch")
	summarizeCmd.Flags().Bool("chat", false, "Start interactive chat session after summary")

	return summarizeCmd
}

func summarizeRunFunc(cmd *cobra.Command, args []string) {
	url := args[0]
	
	// Validate URL
	if !isValidURL(url) {
		fmt.Fprintf(os.Stderr, "Error: Invalid URL provided: %s\n", url)
		os.Exit(1)
	}

	// Get flags
	highlights, _ := cmd.Flags().GetBool("highlights")
	format, _ := cmd.Flags().GetString("format")
	outputFile, _ := cmd.Flags().GetString("output")
	style, _ := cmd.Flags().GetString("style")
	noCache, _ := cmd.Flags().GetBool("no-cache")
	chatMode, _ := cmd.Flags().GetBool("chat")

	// Validate format
	validFormats := []string{"terminal", "json", "markdown"}
	if !contains(validFormats, format) {
		fmt.Fprintf(os.Stderr, "Error: Invalid format '%s'. Valid formats: %s\n", 
			format, strings.Join(validFormats, ", "))
		os.Exit(1)
	}

	// Validate style
	validStyles := []string{"brief", "standard", "detailed"}
	if !contains(validStyles, style) {
		fmt.Fprintf(os.Stderr, "Error: Invalid style '%s'. Valid styles: %s\n",
			style, strings.Join(validStyles, ", "))
		os.Exit(1)
	}

	logger.Info("Starting article summarization", "url", url, "style", style, "highlights", highlights)

	// Initialize services
	cfg := config.Get()
	
	// Initialize cache store
	var cacheStore *store.Store
	if !noCache {
		var err error
		cacheStore, err = store.NewStore(cfg.Cache.Directory)
		if err != nil {
			logger.Error("Failed to initialize cache store", err)
			// Continue without cache rather than failing
		}
	}

	// Initialize LLM client
	llmClient, err := llm.NewClient(cfg.AI.Gemini.Model)
	if err != nil {
		logger.Error("Failed to initialize LLM client", err)
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize AI client: %v\n", err)
		os.Exit(1)
	}
	defer llmClient.Close()

	// Initialize fetcher
	fetcher := fetch.NewContentProcessor()

	if err := runSummarization(url, style, highlights, format, outputFile, chatMode, fetcher, llmClient, cacheStore); err != nil {
		logger.Error("Failed to summarize article", err, "url", url)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runSummarization(url, style string, highlights bool, format, outputFile string, chatMode bool,
	fetcher *fetch.ContentProcessor, llmClient *llm.Client, cacheStore *store.Store) error {
	
	fmt.Printf("🔍 Fetching content from: %s\n", url)

	// Check cache first
	var article *core.Article
	var contentHash string
	var fromCache bool

	if cacheStore != nil {
		cachedArticle, err := cacheStore.GetCachedArticle(url, 24*time.Hour)
		if err == nil && cachedArticle != nil {
			article = cachedArticle
			// Generate content hash for cached article
			contentHash = generateContentHash(article.CleanedText)
			fromCache = true
			fmt.Printf("📋 Using cached content\n")
		}
	}

	// Fetch content if not in cache
	if article == nil {
		fetchedArticle, err := fetcher.ProcessArticle(context.Background(), url)
		if err != nil {
			return fmt.Errorf("failed to fetch article content: %w", err)
		}
		article = fetchedArticle
		contentHash = generateContentHash(article.CleanedText)

		// Cache the article
		if cacheStore != nil {
			if err := cacheStore.CacheArticle(*article); err != nil {
				logger.Error("Failed to cache article", err)
				// Continue without caching rather than failing
			}
		}
		fmt.Printf("✅ Content fetched successfully\n")
	}

	// Check for cached summary
	var summary *core.Summary
	var summaryFromCache bool

	if cacheStore != nil {
		cachedSummary, err := cacheStore.GetCachedSummary(url, contentHash, 7*24*time.Hour) // 7 day cache
		if err == nil && cachedSummary != nil {
			summary = cachedSummary
			summaryFromCache = true
			fmt.Printf("📋 Using cached summary\n")
		}
	}

	// Generate summary if not cached
	if summary == nil {
		// When highlights are enabled, always use detailed style
		actualStyle := style
		if highlights {
			actualStyle = "detailed"
		}
		
		fmt.Printf("🤖 Generating %s summary with highlights...\n", actualStyle)
		
		var generatedSummary core.Summary
		var err error

		if highlights {
			// Use key moments extraction with detailed format
			generatedSummary, err = llmClient.SummarizeArticleWithKeyMoments(*article)
		} else {
			// Use format-specific summarization
			generatedSummary, err = llmClient.SummarizeArticleTextWithFormat(*article, actualStyle)
		}

		if err != nil {
			return fmt.Errorf("failed to generate summary: %w", err)
		}

		summary = &generatedSummary
		summary.ID = uuid.NewString()
		summary.DateGenerated = time.Now().UTC()

		// Cache the summary
		if cacheStore != nil {
			if err := cacheStore.CacheSummary(*summary, url, contentHash); err != nil {
				logger.Error("Failed to cache summary", err)
				// Continue without caching rather than failing
			}
		}
		fmt.Printf("✅ Summary generated successfully\n")
	}

	// Prepare result
	result := SummarizeResult{
		Title:       article.Title,
		URL:         url,
		Summary:     summary.SummaryText,
		ContentType: getContentTypeFromURL(url),
		ProcessedAt: time.Now().UTC(),
	}

	// Add highlights if requested and available
	if highlights {
		result.Highlights = summary.SummaryText // Key moments are in SummaryText for this format
	}

	// Output based on format
	switch format {
	case "json":
		if err := outputJSON(result, outputFile); err != nil {
			return err
		}
	case "markdown":
		if err := outputMarkdown(result, outputFile, highlights); err != nil {
			return err
		}
	default: // terminal
		if err := outputTerminal(result, highlights, fromCache, summaryFromCache); err != nil {
			return err
		}
	}

	// Start interactive chat if requested
	if chatMode && format == "terminal" {
		chatHandler := interactive.NewChatHandler(llmClient)
		if err := chatHandler.StartChatSession(article, url, summary); err != nil {
			return fmt.Errorf("failed to start chat session: %w", err)
		}
		if err := chatHandler.RunChatLoop(); err != nil {
			return fmt.Errorf("chat session error: %w", err)
		}
	}

	return nil
}

func outputTerminal(result SummarizeResult, highlights bool, fromCache, summaryFromCache bool) error {
	fmt.Printf("\n")
	fmt.Printf("📰 %s\n", result.Title)
	fmt.Printf("🔗 %s\n", result.URL)
	if result.ContentType != "web" {
		fmt.Printf("📄 Content Type: %s\n", result.ContentType)
	}
	fmt.Printf("\n")

	if highlights {
		fmt.Printf("✨ Key Highlights:\n")
		fmt.Printf("%s\n", result.Summary)
	} else {
		fmt.Printf("📝 Summary:\n")
		fmt.Printf("%s\n", result.Summary)
	}

	// Show cache status for transparency
	cacheStatus := ""
	if fromCache && summaryFromCache {
		cacheStatus = " (fully cached)"
	} else if fromCache {
		cacheStatus = " (content cached)"
	} else if summaryFromCache {
		cacheStatus = " (summary cached)"
	}
	
	if cacheStatus != "" {
		fmt.Printf("\n💾 Cache status: %s\n", cacheStatus[2:]) // Remove leading " ("
	}

	fmt.Printf("\n")
	return nil
}

func outputJSON(result SummarizeResult, outputFile string) error {
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if outputFile != "" {
		err = os.WriteFile(outputFile, jsonData, 0644)
		if err != nil {
			return fmt.Errorf("failed to write JSON file: %w", err)
		}
		fmt.Printf("💾 JSON output saved to: %s\n", outputFile)
	} else {
		fmt.Printf("%s\n", jsonData)
	}

	return nil
}

func outputMarkdown(result SummarizeResult, outputFile string, highlights bool) error {
	var content strings.Builder
	
	content.WriteString(fmt.Sprintf("# %s\n\n", result.Title))
	content.WriteString(fmt.Sprintf("**Source:** [%s](%s)\n", result.URL, result.URL))
	content.WriteString(fmt.Sprintf("**Processed:** %s\n\n", result.ProcessedAt.Format("2006-01-02 15:04:05 UTC")))
	
	if highlights {
		content.WriteString("## Key Highlights\n\n")
	} else {
		content.WriteString("## Summary\n\n")
	}
	content.WriteString(fmt.Sprintf("%s\n", result.Summary))

	markdownContent := content.String()

	if outputFile != "" {
		err := os.WriteFile(outputFile, []byte(markdownContent), 0644)
		if err != nil {
			return fmt.Errorf("failed to write markdown file: %w", err)
		}
		fmt.Printf("💾 Markdown output saved to: %s\n", outputFile)
	} else {
		fmt.Printf("%s", markdownContent)
	}

	return nil
}

// Helper functions
func isValidURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func getContentTypeFromURL(url string) string {
	url = strings.ToLower(url)
	if strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be") {
		return "youtube"
	}
	if strings.HasSuffix(url, ".pdf") {
		return "pdf"
	}
	return "web"
}

// generateContentHash creates a SHA-256 hash of the content for cache validation
func generateContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}