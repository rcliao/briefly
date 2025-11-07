package handlers

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"briefly/internal/core"
	"briefly/internal/persistence"
	testing "briefly/internal/testing"
)

// NewDigestCompareCmd creates the digest compare command
func NewDigestCompareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compare <digest-id-a> <digest-id-b>",
		Short: "Compare two digests for quality differences (A/B testing)",
		Long: `Compare two digests side-by-side to evaluate quality differences.

This command performs comprehensive A/B testing of digest quality, comparing:
- Coverage percentage (articles cited)
- Vagueness (generic phrases)
- Specificity (numbers, proper nouns, concrete facts)
- Overall quality grade

Examples:
  # Compare two specific digests
  briefly digest compare 123 456

  # Compare digest against previous version
  briefly digest compare --digest-id 123 --previous

  # Batch compare multiple variants
  briefly digest compare --baseline 100 --variants 101,102,103`,
		RunE: runDigestCompare,
	}

	// Flags
	cmd.Flags().String("baseline", "", "Baseline digest ID for batch comparison")
	cmd.Flags().StringSlice("variants", []string{}, "Comma-separated variant digest IDs")
	cmd.Flags().Bool("previous", false, "Compare against previous digest (requires --digest-id)")
	cmd.Flags().String("digest-id", "", "Digest ID for --previous comparison")

	return cmd
}

// runDigestCompare executes the digest comparison
func runDigestCompare(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable not set")
	}

	db, err := persistence.NewPostgresDB(dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Create comparison framework
	framework := testing.NewComparisonFramework()

	// Check for batch comparison mode
	baseline, _ := cmd.Flags().GetString("baseline")
	variants, _ := cmd.Flags().GetStringSlice("variants")
	previous, _ := cmd.Flags().GetBool("previous")
	digestID, _ := cmd.Flags().GetString("digest-id")

	// Mode 1: Batch comparison
	if baseline != "" && len(variants) > 0 {
		return runBatchComparison(ctx, framework, db, baseline, variants)
	}

	// Mode 2: Compare with previous digest
	if previous && digestID != "" {
		return runPreviousComparison(ctx, framework, db, digestID)
	}

	// Mode 3: Standard A/B comparison (requires 2 digest IDs)
	if len(args) != 2 {
		return fmt.Errorf("requires exactly 2 digest IDs for standard comparison, or use --baseline/--variants for batch comparison")
	}

	digestIDA := args[0]
	digestIDB := args[1]

	return runStandardComparison(ctx, framework, db, digestIDA, digestIDB)
}

// runStandardComparison performs A/B comparison between two digests
func runStandardComparison(
	ctx context.Context,
	framework *testing.ComparisonFramework,
	db persistence.Database,
	digestIDA string,
	digestIDB string,
) error {
	fmt.Printf("\n🔍 Loading digests for comparison...\n")

	// Load digest A
	digestA, err := db.Digests().GetByID(ctx, digestIDA)
	if err != nil {
		return fmt.Errorf("failed to load digest A (%s): %w", digestIDA, err)
	}

	// Load digest B
	digestB, err := db.Digests().GetByID(ctx, digestIDB)
	if err != nil {
		return fmt.Errorf("failed to load digest B (%s): %w", digestIDB, err)
	}

	// Load articles (use all articles from both digests)
	articlesA, err := db.Digests().GetDigestArticles(ctx, digestIDA)
	if err != nil {
		return fmt.Errorf("failed to load articles for digest A: %w", err)
	}

	articlesB, err := db.Digests().GetDigestArticles(ctx, digestIDB)
	if err != nil {
		return fmt.Errorf("failed to load articles for digest B: %w", err)
	}

	// Merge article lists (use union for fair comparison)
	articles := mergeArticles(articlesA, articlesB)

	fmt.Printf("   ✓ Loaded digest A: %s (%d articles)\n", digestA.Metadata.Title, len(articlesA))
	fmt.Printf("   ✓ Loaded digest B: %s (%d articles)\n\n", digestB.Metadata.Title, len(articlesB))

	// Perform comparison
	result, err := framework.CompareDigests(ctx, digestA, digestB, articles)
	if err != nil {
		return fmt.Errorf("comparison failed: %w", err)
	}

	// Print report
	framework.PrintComparisonReport(result)

	return nil
}

// runBatchComparison performs batch comparison against baseline
func runBatchComparison(
	ctx context.Context,
	framework *testing.ComparisonFramework,
	db persistence.Database,
	baselineID string,
	variantIDs []string,
) error {
	fmt.Printf("\n🔍 Loading baseline and %d variants...\n", len(variantIDs))

	// Load baseline digest
	baseline, err := db.Digests().GetByID(ctx, baselineID)
	if err != nil {
		return fmt.Errorf("failed to load baseline digest: %w", err)
	}
	fmt.Printf("   ✓ Baseline: %s (ID: %s)\n", baseline.Metadata.Title, baselineID)

	// Load articles for baseline
	articles, err := db.Digests().GetDigestArticles(ctx, baselineID)
	if err != nil {
		return fmt.Errorf("failed to load articles: %w", err)
	}

	// Load variant digests
	variants := make(map[string]*core.Digest)
	for _, variantID := range variantIDs {
		variant, err := db.Digests().GetByID(ctx, variantID)
		if err != nil {
			fmt.Printf("   ⚠️  Failed to load variant %s: %v\n", variantID, err)
			continue
		}

		variantName := fmt.Sprintf("Variant-%s", variantID)
		variants[variantName] = variant
		fmt.Printf("   ✓ Loaded %s: %s\n", variantName, variant.Metadata.Title)
	}

	if len(variants) == 0 {
		return fmt.Errorf("no valid variants loaded")
	}

	fmt.Println()

	// Perform batch comparison
	result, err := framework.BatchCompareDigests(ctx, baseline, variants, articles)
	if err != nil {
		return fmt.Errorf("batch comparison failed: %w", err)
	}

	// Print report
	framework.PrintBatchComparisonReport(result)

	return nil
}

// runPreviousComparison compares a digest with the previous digest
func runPreviousComparison(
	ctx context.Context,
	framework *testing.ComparisonFramework,
	db persistence.Database,
	digestID string,
) error {
	fmt.Printf("\n🔍 Finding previous digest for comparison...\n")

	// Load current digest
	current, err := db.Digests().GetByID(ctx, digestID)
	if err != nil {
		return fmt.Errorf("failed to load digest: %w", err)
	}
	fmt.Printf("   ✓ Current: %s (ID: %s)\n", current.Metadata.Title, digestID)

	// Get all recent digests to find previous
	recents, err := db.Digests().ListRecent(ctx, current.ProcessedDate.AddDate(0, 0, -90), 100)
	if err != nil {
		return fmt.Errorf("failed to list recent digests: %w", err)
	}

	// Find previous digest (by creation date)
	var previous *core.Digest
	for i, digest := range recents {
		if digest.ID == digestID && i+1 < len(recents) {
			previous = &recents[i+1]
			break
		}
	}

	if previous == nil {
		return fmt.Errorf("no previous digest found (digest %s may be the first)", digestID)
	}
	fmt.Printf("   ✓ Previous: %s (ID: %s)\n\n", previous.Metadata.Title, previous.ID)

	// Load articles (merge from both digests)
	currentArticles, err := db.Digests().GetDigestArticles(ctx, digestID)
	if err != nil {
		return fmt.Errorf("failed to load current articles: %w", err)
	}

	prevArticles, err := db.Digests().GetDigestArticles(ctx, previous.ID)
	if err != nil {
		return fmt.Errorf("failed to load previous articles: %w", err)
	}

	articles := mergeArticles(currentArticles, prevArticles)

	// Perform comparison (previous = A, current = B)
	result, err := framework.CompareDigests(ctx, previous, current, articles)
	if err != nil {
		return fmt.Errorf("comparison failed: %w", err)
	}

	// Print report
	framework.PrintComparisonReport(result)

	return nil
}

// mergeArticles combines two article slices, removing duplicates
func mergeArticles(a []core.Article, b []core.Article) []core.Article {
	seen := make(map[string]bool)
	merged := make([]core.Article, 0, len(a)+len(b))

	for _, article := range a {
		if !seen[article.ID] {
			merged = append(merged, article)
			seen[article.ID] = true
		}
	}

	for _, article := range b {
		if !seen[article.ID] {
			merged = append(merged, article)
			seen[article.ID] = true
		}
	}

	return merged
}
