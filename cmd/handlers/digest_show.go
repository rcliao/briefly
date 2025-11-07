package handlers

import (
	"briefly/internal/core"
	"briefly/internal/persistence"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// NewDigestShowCmd creates the digest show command
func NewDigestShowCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "show <digest-id>",
		Short: "Display a specific digest",
		Long: `Show detailed information about a specific digest.

This command retrieves a digest from the database and displays
its content, including summary, key moments, perspectives, and articles.

Examples:
  # Show digest with default formatting
  briefly digest show abc123

  # Show digest in JSON format
  briefly digest show abc123 --format json

  # Show digest in markdown format
  briefly digest show abc123 --format markdown`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			digestShowRun(cmd, args[0], format)
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json, markdown)")

	return cmd
}

func digestShowRun(cmd *cobra.Command, digestID string, format string) {
	ctx := context.Background()

	// Get database connection string
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Fprintf(os.Stderr, "❌ DATABASE_URL environment variable not set\n")
		fmt.Fprintf(os.Stderr, "💡 Set DATABASE_URL to your PostgreSQL connection string\n")
		os.Exit(1)
	}

	// Connect to database
	db, err := persistence.NewPostgresDB(dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Fetch digest with relationships
	digest, err := db.Digests().GetWithArticles(ctx, digestID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to get digest: %v\n", err)
		fmt.Fprintf(os.Stderr, "💡 Use 'briefly digest list' to see available digests\n")
		os.Exit(1)
	}

	// Display based on format
	switch strings.ToLower(format) {
	case "json":
		displayDigestJSON(digest)
	case "markdown", "md":
		displayDigestMarkdown(digest)
	default:
		displayDigestText(digest)
	}
}

func displayDigestText(digest *core.Digest) {
	// Get title from v2.0 or v1.0
	title := digest.Title
	if title == "" {
		title = digest.Metadata.Title
	}

	fmt.Printf("\n📄 %s\n", title)
	fmt.Println(strings.Repeat("═", 80))

	fmt.Printf("ID:           %s\n", digest.ID)
	fmt.Printf("Date:         %s\n", digest.ProcessedDate.Format("January 2, 2006"))
	fmt.Printf("Articles:     %d\n", digest.ArticleCount)
	if digest.ClusterID != nil {
		fmt.Printf("Cluster ID:   %d\n", *digest.ClusterID)
	}
	fmt.Println()

	// Display TLDR
	if digest.TLDRSummary != "" {
		fmt.Println("📌 TL;DR")
		fmt.Println(strings.Repeat("─", 80))
		fmt.Println(digest.TLDRSummary)
		fmt.Println()
	}

	// Display Summary
	if digest.Summary != "" {
		fmt.Println("📝 Summary")
		fmt.Println(strings.Repeat("─", 80))
		fmt.Println(digest.Summary)
		fmt.Println()
	}

	// Display Key Moments
	if len(digest.KeyMoments) > 0 {
		fmt.Println("💡 Key Moments")
		fmt.Println(strings.Repeat("─", 80))
		for i, moment := range digest.KeyMoments {
			fmt.Printf("%d. \"%s\" [%d]\n", i+1, moment.Quote, moment.CitationNumber)
		}
		fmt.Println()
	}

	// Display Perspectives
	if len(digest.Perspectives) > 0 {
		fmt.Println("🔍 Perspectives")
		fmt.Println(strings.Repeat("─", 80))
		for _, persp := range digest.Perspectives {
			icon := "✓"
			if persp.Type == "opposing" {
				icon = "✗"
			}
			// Capitalize first letter
			capitalizedType := strings.ToUpper(string(persp.Type[0])) + persp.Type[1:]
			fmt.Printf("%s %s View\n", icon, capitalizedType)
			fmt.Printf("  %s\n", persp.Summary)
			fmt.Printf("  Sources: %v\n\n", persp.CitationNumbers)
		}
	}

	// Display Articles
	if len(digest.Articles) > 0 {
		fmt.Println("📚 Articles")
		fmt.Println(strings.Repeat("─", 80))
		for i, article := range digest.Articles {
			fmt.Printf("[%d] %s\n", i+1, article.Title)
			fmt.Printf("    %s\n", article.URL)
			if article.Publisher != "" {
				fmt.Printf("    Publisher: %s\n", article.Publisher)
			}
			fmt.Println()
		}
	}

	fmt.Println(strings.Repeat("═", 80))
}

func displayDigestMarkdown(digest *core.Digest) {
	// Simply output the markdown summary if available
	if digest.Summary != "" {
		fmt.Println(digest.Summary)
	} else {
		fmt.Fprintf(os.Stderr, "No markdown content available for this digest\n")
	}
}

func displayDigestJSON(digest *core.Digest) {
	// TODO: Implement JSON output using encoding/json
	fmt.Fprintf(os.Stderr, "JSON format not yet implemented\n")
	os.Exit(1)
}
