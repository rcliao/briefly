package handlers

import (
	"briefly/internal/core"
	"briefly/internal/logger"
	"briefly/internal/persistence"
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// NewManualURLCmd creates the manual URL management command
func NewManualURLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "manual-url",
		Short: "Manage manually submitted URLs",
		Long: `Manage manually submitted URLs for digest generation.

This allows you to submit URLs directly for processing, independent of RSS feeds.
URLs can be tracked through their processing lifecycle (pending → processing → processed).

Subcommands:
  add       Add one or more URLs
  list      List submitted URLs and their status
  status    Check status of a specific URL
  retry     Retry failed URLs
  clear     Clear processed/failed URLs`,
	}

	cmd.AddCommand(newManualURLAddCmd())
	cmd.AddCommand(newManualURLListCmd())
	cmd.AddCommand(newManualURLStatusCmd())
	cmd.AddCommand(newManualURLRetryCmd())
	cmd.AddCommand(newManualURLClearCmd())

	return cmd
}

func newManualURLAddCmd() *cobra.Command {
	var submittedBy string

	cmd := &cobra.Command{
		Use:   "add <url> [url...]",
		Short: "Add one or more URLs for processing",
		Long: `Add one or more URLs to be processed in the next digest generation.

URLs will be queued with 'pending' status and processed during aggregation.
You can optionally specify who submitted the URL for tracking purposes.

Examples:
  briefly url add https://example.com/article
  briefly url add https://example.com/article1 https://example.com/article2
  briefly url add https://example.com/article --submitted-by "john@example.com"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			urls := args
			return runManualURLAdd(cmd.Context(), urls, submittedBy)
		},
	}

	cmd.Flags().StringVarP(&submittedBy, "submitted-by", "u", "", "Who submitted this URL")

	return cmd
}

func newManualURLListCmd() *cobra.Command {
	var status string
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List submitted URLs",
		Long: `List manually submitted URLs and their processing status.

You can filter by status: pending, processing, processed, failed
Use --limit to control how many URLs to display (default: 50)

Examples:
  briefly url list
  briefly url list --status pending
  briefly url list --status failed --limit 100`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runManualURLList(cmd.Context(), status, limit)
		},
	}

	cmd.Flags().StringVarP(&status, "status", "s", "", "Filter by status (pending/processing/processed/failed)")
	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Maximum number of URLs to display")

	return cmd
}

func newManualURLStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <url-id>",
		Short: "Check status of a specific URL",
		Long: `Get detailed status information for a specific manually submitted URL.

The URL ID is shown when you add a URL or list URLs.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			urlID := args[0]
			return runManualURLStatus(cmd.Context(), urlID)
		},
	}
}

func newManualURLRetryCmd() *cobra.Command {
	var retryAll bool

	cmd := &cobra.Command{
		Use:   "retry [url-id]",
		Short: "Retry failed URL(s)",
		Long: `Reset failed URLs back to pending status for retry.

You can retry a specific URL by ID, or use --all to retry all failed URLs.

Examples:
  briefly url retry abc123
  briefly url retry --all`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			urlID := ""
			if len(args) > 0 {
				urlID = args[0]
			}
			if urlID == "" && !retryAll {
				return fmt.Errorf("specify a URL ID or use --all to retry all failed URLs")
			}
			return runManualURLRetry(cmd.Context(), urlID, retryAll)
		},
	}

	cmd.Flags().BoolVar(&retryAll, "all", false, "Retry all failed URLs")

	return cmd
}

func newManualURLClearCmd() *cobra.Command {
	var clearProcessed bool
	var clearFailed bool

	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear processed or failed URLs",
		Long: `Remove processed or failed URLs from the database.

This helps keep the URL queue clean by removing already-processed or failed URLs.
Use --processed to clear successfully processed URLs, or --failed to clear failed ones.

Examples:
  briefly url clear --processed
  briefly url clear --failed
  briefly url clear --processed --failed`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !clearProcessed && !clearFailed {
				return fmt.Errorf("specify --processed and/or --failed")
			}
			return runManualURLClear(cmd.Context(), clearProcessed, clearFailed)
		},
	}

	cmd.Flags().BoolVar(&clearProcessed, "processed", false, "Clear successfully processed URLs")
	cmd.Flags().BoolVar(&clearFailed, "failed", false, "Clear failed URLs")

	return cmd
}

// Implementation functions

func runManualURLAdd(ctx context.Context, urls []string, submittedBy string) error {
	log := logger.Get()
	log.Info("Adding manual URLs", "count", len(urls))

	db, err := getDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	addedCount := 0
	for _, url := range urls {
		// Check if URL already exists
		existing, _ := db.ManualURLs().GetByURL(ctx, url)
		if existing != nil {
			fmt.Printf("⚠️  URL already exists: %s (Status: %s)\n", url, existing.Status)
			continue
		}

		manualURL := &core.ManualURL{
			ID:          uuid.NewString(),
			URL:         url,
			SubmittedBy: submittedBy,
			Status:      core.ManualURLStatusPending,
			CreatedAt:   time.Now().UTC(),
		}

		if err := db.ManualURLs().Create(ctx, manualURL); err != nil {
			fmt.Printf("❌ Failed to add URL: %s - %v\n", url, err)
			continue
		}

		fmt.Printf("✅ Added: %s (ID: %s)\n", url, manualURL.ID[:8]+"...")
		addedCount++
	}

	fmt.Printf("\n%d URL(s) added successfully\n", addedCount)
	fmt.Println("\nNext steps:")
	fmt.Println("  • Run aggregation to process: briefly aggregate")
	fmt.Println("  • Check status: briefly url list")

	return nil
}

func runManualURLList(ctx context.Context, statusFilter string, limit int) error {
	db, err := getDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	var urls []core.ManualURL
	if statusFilter != "" {
		urls, err = db.ManualURLs().GetByStatus(ctx, statusFilter, limit)
	} else {
		urls, err = db.ManualURLs().List(ctx, persistence.ListOptions{Limit: limit})
	}
	if err != nil {
		return fmt.Errorf("failed to list URLs: %w", err)
	}

	if len(urls) == 0 {
		fmt.Println("No URLs found")
		if statusFilter != "" {
			fmt.Printf("(filtered by status: %s)\n", statusFilter)
		}
		fmt.Println("\nAdd your first URL:")
		fmt.Println("  briefly url add <url>")
		return nil
	}

	// Display URLs in a table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tURL\tStatus\tSubmitted By\tCreated\n")
	fmt.Fprintf(w, "━━━━━━━━━━\t━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\t━━━━━━━━━━━━\t━━━━━━━━━━━━━━\t━━━━━━━━━━━━\n")

	for _, url := range urls {
		statusIcon := getStatusIcon(url.Status)

		urlShort := url.URL
		if len(urlShort) > 50 {
			urlShort = urlShort[:47] + "..."
		}

		submittedBy := url.SubmittedBy
		if submittedBy == "" {
			submittedBy = "(anonymous)"
		}
		if len(submittedBy) > 20 {
			submittedBy = submittedBy[:17] + "..."
		}

		created := url.CreatedAt.Format("2006-01-02 15:04")

		fmt.Fprintf(w, "%s\t%s\t%s %s\t%s\t%s\n",
			url.ID[:8]+"...", urlShort, statusIcon, url.Status, submittedBy, created,
		)
	}
	w.Flush()

	fmt.Printf("\nTotal URLs: %d\n", len(urls))
	if statusFilter != "" {
		fmt.Printf("(filtered by status: %s)\n", statusFilter)
	}

	return nil
}

func runManualURLStatus(ctx context.Context, urlID string) error {
	db, err := getDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	manualURL, err := db.ManualURLs().Get(ctx, urlID)
	if err != nil {
		return fmt.Errorf("URL not found: %w", err)
	}

	fmt.Println("📋 URL Details")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("ID:           %s\n", manualURL.ID)
	fmt.Printf("URL:          %s\n", manualURL.URL)
	fmt.Printf("Status:       %s %s\n", getStatusIcon(manualURL.Status), manualURL.Status)
	if manualURL.SubmittedBy != "" {
		fmt.Printf("Submitted By: %s\n", manualURL.SubmittedBy)
	}
	fmt.Printf("Created:      %s\n", manualURL.CreatedAt.Format("2006-01-02 15:04:05"))
	if manualURL.ProcessedAt != nil {
		fmt.Printf("Processed:    %s\n", manualURL.ProcessedAt.Format("2006-01-02 15:04:05"))
	}
	if manualURL.ErrorMessage != "" {
		fmt.Printf("\n⚠️  Error:      %s\n", manualURL.ErrorMessage)
		fmt.Println("\nTo retry this URL:")
		fmt.Printf("  briefly url retry %s\n", manualURL.ID)
	}

	return nil
}

func runManualURLRetry(ctx context.Context, urlID string, retryAll bool) error {
	log := logger.Get()

	db, err := getDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	if retryAll {
		log.Info("Retrying all failed URLs")
		failedURLs, err := db.ManualURLs().GetByStatus(ctx, string(core.ManualURLStatusFailed), 1000)
		if err != nil {
			return fmt.Errorf("failed to get failed URLs: %w", err)
		}

		if len(failedURLs) == 0 {
			fmt.Println("No failed URLs to retry")
			return nil
		}

		retryCount := 0
		for _, url := range failedURLs {
			if err := db.ManualURLs().UpdateStatus(ctx, url.ID, string(core.ManualURLStatusPending), ""); err != nil {
				fmt.Printf("⚠️  Failed to retry %s: %v\n", url.URL, err)
				continue
			}
			retryCount++
		}

		fmt.Printf("✅ Reset %d failed URL(s) to pending status\n", retryCount)
		return nil
	}

	// Retry specific URL
	log.Info("Retrying URL", "id", urlID)
	manualURL, err := db.ManualURLs().Get(ctx, urlID)
	if err != nil {
		return fmt.Errorf("URL not found: %w", err)
	}

	if manualURL.Status != core.ManualURLStatusFailed {
		return fmt.Errorf("URL is not in failed status (current: %s)", manualURL.Status)
	}

	if err := db.ManualURLs().UpdateStatus(ctx, urlID, string(core.ManualURLStatusPending), ""); err != nil {
		return fmt.Errorf("failed to retry URL: %w", err)
	}

	fmt.Printf("✅ URL reset to pending status: %s\n", manualURL.URL)
	return nil
}

func runManualURLClear(ctx context.Context, clearProcessed, clearFailed bool) error {
	log := logger.Get()
	log.Info("Clearing URLs", "processed", clearProcessed, "failed", clearFailed)

	db, err := getDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	deletedCount := 0

	if clearProcessed {
		processedURLs, err := db.ManualURLs().GetByStatus(ctx, string(core.ManualURLStatusProcessed), 10000)
		if err != nil {
			return fmt.Errorf("failed to get processed URLs: %w", err)
		}

		for _, url := range processedURLs {
			if err := db.ManualURLs().Delete(ctx, url.ID); err != nil {
				fmt.Printf("⚠️  Failed to delete %s: %v\n", url.URL, err)
				continue
			}
			deletedCount++
		}
	}

	if clearFailed {
		failedURLs, err := db.ManualURLs().GetByStatus(ctx, string(core.ManualURLStatusFailed), 10000)
		if err != nil {
			return fmt.Errorf("failed to get failed URLs: %w", err)
		}

		for _, url := range failedURLs {
			if err := db.ManualURLs().Delete(ctx, url.ID); err != nil {
				fmt.Printf("⚠️  Failed to delete %s: %v\n", url.URL, err)
				continue
			}
			deletedCount++
		}
	}

	fmt.Printf("✅ Cleared %d URL(s)\n", deletedCount)
	return nil
}

// Helper function to get status icon
func getStatusIcon(status string) string {
	switch status {
	case string(core.ManualURLStatusPending):
		return "⏳"
	case string(core.ManualURLStatusProcessing):
		return "⚙️"
	case string(core.ManualURLStatusProcessed):
		return "✅"
	case string(core.ManualURLStatusFailed):
		return "❌"
	default:
		return "❓"
	}
}
