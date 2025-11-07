package handlers

import (
	"briefly/internal/persistence"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// NewDigestListCmd creates the digest list command
func NewDigestListCmd() *cobra.Command {
	var limit int
	var since int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent digests from database",
		Long: `List recent digests stored in the database.

This command queries the database for recently generated digests
and displays their metadata in a table format.

Examples:
  # List last 10 digests
  briefly digest list

  # List last 20 digests
  briefly digest list --limit 20

  # List digests from last 7 days
  briefly digest list --since 7`,
		Run: func(cmd *cobra.Command, args []string) {
			digestListRun(cmd, limit, since)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Maximum number of digests to list")
	cmd.Flags().IntVarP(&since, "since", "s", 30, "List digests from last N days")

	return cmd
}

func digestListRun(cmd *cobra.Command, limit int, since int) {
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

	// Calculate since date
	sinceDate := time.Now().AddDate(0, 0, -since)

	// Fetch recent digests
	digests, err := db.Digests().ListRecent(ctx, sinceDate, limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to list digests: %v\n", err)
		os.Exit(1)
	}

	if len(digests) == 0 {
		fmt.Println("No digests found in the specified time range")
		fmt.Printf("💡 Try increasing --since (currently: %d days)\n", since)
		return
	}

	// Display results
	fmt.Printf("\n📄 Recent Digests (last %d days)\n", since)
	fmt.Println("═══════════════════════════════════════════════════════════════════")
	fmt.Printf("%-12s  %-40s  %s\n", "Date", "Title", "Articles")
	fmt.Println("───────────────────────────────────────────────────────────────────")

	for _, digest := range digests {
		date := digest.ProcessedDate.Format("Jan 02, 2006")
		title := digest.Title
		if len(title) > 40 {
			title = title[:37] + "..."
		}

		// Get article count from v2.0 or v1.0 field
		articleCount := digest.ArticleCount
		if articleCount == 0 {
			articleCount = digest.Metadata.ArticleCount
		}

		fmt.Printf("%-12s  %-40s  %d\n", date, title, articleCount)
	}

	fmt.Println("═══════════════════════════════════════════════════════════════════")
	fmt.Printf("\n💡 Use 'briefly digest show <id>' to view a specific digest\n")
}
