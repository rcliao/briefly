package handlers

import (
	"briefly/internal/llm"
	"briefly/internal/logger"
	"briefly/internal/pipeline"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// NewReadSimplifiedCmd creates the new simplified read command
func NewReadSimplifiedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read [url]",
		Short: "Quick summary of a single article",
		Long: `Generate a quick summary of a single article for fast reading.

This command provides:
- 200-word summary
- Key takeaways (3-5 points)
- Main theme/category
- Estimated reading time

Perfect for quickly understanding an article without reading the full text.

Examples:
  briefly read https://example.com/article
  briefly read --no-cache https://example.com/fresh-article
  briefly read https://example.com/long-article`,
		Args: cobra.ExactArgs(1),
		Run:  readSimplifiedRun,
	}

	// Flags
	cmd.Flags().Bool("no-cache", false, "Disable caching (fetch fresh content)")
	cmd.Flags().Bool("raw", false, "Output raw markdown without formatting")

	return cmd
}

func readSimplifiedRun(cmd *cobra.Command, args []string) {
	startTime := time.Now()

	// Get flags
	url := args[0]
	noCache, _ := cmd.Flags().GetBool("no-cache")
	raw, _ := cmd.Flags().GetBool("raw")

	logger.Info("Starting quick read", "url", url, "no_cache", noCache)

	// Validate URL
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		fmt.Fprintf(os.Stderr, "❌ Invalid URL: must start with http:// or https://\n")
		os.Exit(1)
	}

	// Initialize LLM client
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

	// Build pipeline
	builder := pipeline.NewBuilder().
		WithLLMClient(llmClient).
		WithCacheDir(".briefly-cache")

	if noCache {
		builder = builder.WithoutCache()
	}

	pipe, err := builder.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to build pipeline: %v\n", err)
		os.Exit(1)
	}

	// Execute quick read
	ctx := context.Background()
	opts := pipeline.QuickReadOptions{
		URL: url,
	}

	if !raw {
		fmt.Printf("📖 Reading: %s\n\n", url)
	}

	result, err := pipe.QuickRead(ctx, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Failed to read article: %v\n", err)
		os.Exit(1)
	}

	elapsed := time.Since(startTime)

	// Display results
	if raw {
		// Raw markdown output
		fmt.Println(result.Markdown)
	} else {
		// Formatted output
		printQuickReadResult(result, elapsed)
	}

	// Log completion
	logger.Info("Quick read completed",
		"url", url,
		"cached", result.WasCached,
		"duration", elapsed)
}

func printQuickReadResult(result *pipeline.QuickReadResult, elapsed time.Duration) {
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("📄 %s\n", result.Article.Title)
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("🔗 Source: %s\n", result.Article.URL)

	if result.Article.ContentType != "" {
		fmt.Printf("📦 Type: %s\n", result.Article.ContentType)
	}

	if result.WasCached {
		fmt.Println("💾 Loaded from cache")
	}

	fmt.Println()

	// Summary
	fmt.Println("📝 Summary:")
	fmt.Println(wrapText(result.Summary.SummaryText, 80))
	fmt.Println()

	// Key points (if available)
	// Note: We'll need to parse these from the summary or add to Summary struct
	fmt.Println("🎯 Key Takeaways:")
	keyPoints := extractKeyPointsFromSummary(result.Summary.SummaryText)
	if len(keyPoints) > 0 {
		for _, point := range keyPoints {
			fmt.Printf("   • %s\n", point)
		}
	} else {
		fmt.Println("   • See summary above for main points")
	}

	fmt.Println()

	// Metadata
	fmt.Println("ℹ️  Info:")
	fmt.Printf("   • Content Length: %d chars\n", len(result.Article.CleanedText))
	fmt.Printf("   • Estimated Read Time: %d min\n", estimateReadTime(result.Article.CleanedText))
	fmt.Printf("   • Processed in: %v\n", elapsed.Round(time.Millisecond))

	if result.Article.Duration > 0 {
		fmt.Printf("   • Video Duration: %d min\n", result.Article.Duration/60)
	}

	fmt.Println("\n═══════════════════════════════════════")
}

// wrapText wraps text to a specified width
func wrapText(text string, width int) string {
	if len(text) <= width {
		return text
	}

	var result strings.Builder
	words := strings.Fields(text)
	lineLength := 0

	for i, word := range words {
		wordLen := len(word)

		if lineLength+wordLen+1 > width && lineLength > 0 {
			result.WriteString("\n")
			lineLength = 0
		}

		if lineLength > 0 {
			result.WriteString(" ")
			lineLength++
		}

		result.WriteString(word)
		lineLength += wordLen

		// Add space after word unless it's the last word
		// Space will be added at the start of next iteration (logic implicit in loop)
		_ = i // Loop variable used for iteration
	}

	return result.String()
}

// estimateReadTime estimates reading time in minutes (assuming 200 words/min)
func estimateReadTime(content string) int {
	words := len(strings.Fields(content))
	minutes := words / 200
	if minutes == 0 {
		minutes = 1
	}
	return minutes
}

// extractKeyPointsFromSummary attempts to extract bullet points from summary text
func extractKeyPointsFromSummary(summary string) []string {
	var points []string
	lines := strings.Split(summary, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for bullet points
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "•") || strings.HasPrefix(line, "*") {
			point := strings.TrimSpace(line[1:])
			if point != "" {
				points = append(points, point)
			}
		} else if len(line) > 2 && line[0] >= '1' && line[0] <= '9' && (line[1] == '.' || line[1] == ')') {
			// Numbered list
			point := strings.TrimSpace(line[2:])
			if point != "" {
				points = append(points, point)
			}
		}
	}

	return points
}