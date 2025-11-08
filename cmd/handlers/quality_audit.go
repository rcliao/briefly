package handlers

import (
	"context"
	"fmt"
	"os"
	"time"

	"briefly/internal/core"
	"briefly/internal/persistence"
	"briefly/internal/quality"
	"github.com/spf13/cobra"
)

// NewQualityAuditCmd creates the quality audit command
func NewQualityAuditCmd() *cobra.Command {
	var limit int
	var since int
	var verbose bool

	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Audit quality of recent digests",
		Long: `Perform comprehensive quality audit on recent digests.

This command evaluates digests for:
  - Coverage: % of articles cited in summary
  - Vagueness: Generic phrases like "several", "various", "many"
  - Specificity: Presence of numbers, names, dates
  - Citation density: Citations per 100 words

Each digest is graded A (excellent), B (good), C (fair), or D (poor).

Examples:
  # Audit last 10 digests
  briefly quality audit

  # Audit last 20 digests with verbose output
  briefly quality audit --limit 20 --verbose

  # Audit digests from last 30 days
  briefly quality audit --since 30`,
		Run: func(cmd *cobra.Command, args []string) {
			qualityAuditRun(cmd, limit, since, verbose)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Maximum number of digests to audit")
	cmd.Flags().IntVarP(&since, "since", "s", 30, "Audit digests from last N days")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed metrics for each digest")

	return cmd
}

func qualityAuditRun(cmd *cobra.Command, limit int, since int, verbose bool) {
	ctx := context.Background()

	// Get database connection
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
	fmt.Printf("🔍 Fetching digests from last %d days...\n", since)
	digests, err := db.Digests().ListRecent(ctx, sinceDate, limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to list digests: %v\n", err)
		os.Exit(1)
	}

	if len(digests) == 0 {
		fmt.Println("\n⚠️  No digests found in the specified time range")
		fmt.Printf("💡 Try increasing --since (currently: %d days)\n", since)
		return
	}

	fmt.Printf("✓ Found %d digests to audit\n\n", len(digests))

	// Fetch articles for each digest
	articlesMap := make(map[string][]core.Article)
	for i, digest := range digests {
		fmt.Printf("\r   Fetching articles [%d/%d]...", i+1, len(digests))
		articles, err := db.Digests().GetDigestArticles(ctx, digest.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ Failed to fetch articles for digest %s: %v\n", digest.ID, err)
			continue
		}
		articlesMap[digest.ID] = articles
	}
	fmt.Printf("\r   ✓ Fetched articles for %d digests\n\n", len(articlesMap))

	// Create evaluator
	evaluator := quality.NewDigestEvaluator()

	// Perform audit
	if verbose {
		// Verbose mode: show detailed metrics for each digest
		fmt.Println("═══════════════════════════════════════════════════════════════════")
		fmt.Println("DETAILED QUALITY REPORT (per digest)")
		fmt.Println("═══════════════════════════════════════════════════════════════════")

		for _, digest := range digests {
			articles := articlesMap[digest.ID]
			evaluator.PrintDigestReport(&digest, articles)
		}
	}

	// Print aggregate audit report
	report := evaluator.AuditRecentDigests(digests, articlesMap)
	evaluator.PrintAuditReport(report)

	// Print detailed recommendations based on findings
	printRecommendations(report)
}

// printRecommendations provides actionable recommendations based on audit results
func printRecommendations(report *quality.AuditReport) {
	fmt.Println("\n📊 DETAILED RECOMMENDATIONS")
	fmt.Println("═══════════════════════════════════════════════════════════════════")

	hasIssues := false

	// Check coverage
	if report.AvgCoverage < 0.80 {
		hasIssues = true
		fmt.Println("\n🔴 LOW ARTICLE COVERAGE")
		fmt.Printf("   Current: %.0f%% | Target: 80%%+\n", report.AvgCoverage*100)
		fmt.Println("   Actions:")
		fmt.Println("   1. Review cluster narrative generation prompts")
		fmt.Println("   2. Ensure all articles are included in clustering")
		fmt.Println("   3. Consider per-article extraction approach (see docs/digest-improvement-plan.md)")
		fmt.Println("   4. Check if some articles are being filtered out")
	}

	// Check vagueness
	if report.AvgVagueness > 2.0 {
		hasIssues = true
		fmt.Println("\n🟡 HIGH VAGUENESS SCORE")
		fmt.Printf("   Current: %.1f phrases/digest | Target: ≤ 2.0\n", report.AvgVagueness)
		fmt.Println("   Actions:")
		fmt.Println("   1. Add fact-extraction prompts to article summarization")
		fmt.Println("   2. Ban vague phrases in prompts: 'several', 'various', 'many'")
		fmt.Println("   3. Require specific numbers, names, and dates")
		fmt.Println("   4. Implement self-critique refinement pass")
	}

	// Check specificity
	if report.AvgSpecificity < 50.0 {
		hasIssues = true
		fmt.Println("\n🟡 LOW SPECIFICITY SCORE")
		fmt.Printf("   Current: %.0f/100 | Target: 50+\n", report.AvgSpecificity)
		fmt.Println("   Actions:")
		fmt.Println("   1. Enforce fact extraction in article summarization")
		fmt.Println("   2. Require minimum number of specific facts per digest")
		fmt.Println("   3. Add validation: numbers/percentages, proper nouns, dates")
		fmt.Println("   4. Review prompts for specificity requirements")
	}

	// Check grade distribution
	dGradeCount := report.GradeCounts["D"]
	totalCount := report.TotalDigests
	if dGradeCount > 0 {
		dPct := float64(dGradeCount) * 100.0 / float64(totalCount)
		if dPct > 20.0 { // More than 20% D grades
			hasIssues = true
			fmt.Println("\n🔴 HIGH FAILURE RATE")
			fmt.Printf("   %d/%d digests (%.0f%%) graded D (poor)\n", dGradeCount, totalCount, dPct)
			fmt.Println("   Actions:")
			fmt.Println("   1. Run full diagnostic using improvement plan (docs/digest-improvement-plan.md)")
			fmt.Println("   2. Consider implementing quality gates in pipeline")
			fmt.Println("   3. Add retry logic for failed digests")
			fmt.Println("   4. Review clustering quality (poor clusters → poor digests)")
		}
	}

	if !hasIssues {
		fmt.Println("\n🟢 Quality is within acceptable ranges!")
		fmt.Println("   Continue monitoring and consider incremental improvements:")
		fmt.Println("   - Track quality trends over time")
		fmt.Println("   - A/B test new prompts before deploying")
		fmt.Println("   - Aim for 80%+ A/B grades")
	}

	fmt.Println("\n═══════════════════════════════════════════════════════════════════")
	fmt.Println("💡 For comprehensive improvement strategies, see:")
	fmt.Println("   docs/digest-improvement-plan.md")
	fmt.Println("═══════════════════════════════════════════════════════════════════")
}

// NewQualityReportCmd creates the quality report command for a specific digest
func NewQualityReportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report <digest-id>",
		Short: "Generate detailed quality report for a specific digest",
		Long: `Generate comprehensive quality report for a single digest.

Shows detailed metrics including:
  - Article coverage and uncited articles
  - Vague phrases found
  - Specificity score breakdown
  - Citation density
  - Letter grade with justification

Examples:
  # Get detailed report for digest
  briefly quality report abc123

  # Get report and see cluster coherence
  briefly quality report abc123 --with-clustering`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			qualityReportRun(cmd, args[0])
		},
	}

	return cmd
}

func qualityReportRun(cmd *cobra.Command, digestID string) {
	ctx := context.Background()

	// Get database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Fprintf(os.Stderr, "❌ DATABASE_URL environment variable not set\n")
		os.Exit(1)
	}

	db, err := persistence.NewPostgresDB(dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Fetch digest
	digest, err := db.Digests().GetByID(ctx, digestID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to fetch digest: %v\n", err)
		os.Exit(1)
	}

	// Fetch articles
	articles, err := db.Digests().GetDigestArticles(ctx, digestID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to fetch articles: %v\n", err)
		os.Exit(1)
	}

	// Evaluate quality
	evaluator := quality.NewDigestEvaluator()
	evaluator.PrintDigestReport(digest, articles)
}

// NewQualityTrendsCmd creates the quality trends command
func NewQualityTrendsCmd() *cobra.Command {
	var since int

	cmd := &cobra.Command{
		Use:   "trends",
		Short: "Analyze quality trends over time",
		Long: `Analyze how digest quality has changed over time.

Shows:
  - Average quality scores by week/month
  - Grade distribution trends
  - Coverage and specificity trends
  - Regression detection

Examples:
  # Analyze last 90 days
  briefly quality trends --since 90

  # Analyze last 6 months
  briefly quality trends --since 180`,
		Run: func(cmd *cobra.Command, args []string) {
			qualityTrendsRun(cmd, since)
		},
	}

	cmd.Flags().IntVarP(&since, "since", "s", 90, "Analyze trends from last N days")

	return cmd
}

func qualityTrendsRun(cmd *cobra.Command, since int) {
	ctx := context.Background()

	// Get database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Fprintf(os.Stderr, "❌ DATABASE_URL environment variable not set\n")
		os.Exit(1)
	}

	db, err := persistence.NewPostgresDB(dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	sinceDate := time.Now().AddDate(0, 0, -since)

	// Fetch digests
	digests, err := db.Digests().ListRecent(ctx, sinceDate, 1000) // Get all in range
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to list digests: %v\n", err)
		os.Exit(1)
	}

	if len(digests) < 2 {
		fmt.Println("\n⚠️  Not enough digests for trend analysis (need at least 2)")
		fmt.Printf("💡 Found: %d digests in last %d days\n", len(digests), since)
		return
	}

	// Fetch articles for each digest
	articlesMap := make(map[string][]core.Article)
	for i, digest := range digests {
		fmt.Printf("\rFetching articles [%d/%d]...", i+1, len(digests))
		articles, err := db.Digests().GetDigestArticles(ctx, digest.ID)
		if err != nil {
			continue
		}
		articlesMap[digest.ID] = articles
	}
	fmt.Printf("\r✓ Fetched articles for %d digests\n\n", len(articlesMap))

	// Analyze trends
	analyzeTrends(digests, articlesMap)
}

func analyzeTrends(digests []core.Digest, articlesMap map[string][]core.Article) {
	evaluator := quality.NewDigestEvaluator()

	// Group by week
	type weekStats struct {
		weekStart      time.Time
		count          int
		avgCoverage    float64
		avgVagueness   float64
		avgSpecificity float64
		gradeACounts   int
		gradeBCounts   int
		gradeCCounts   int
		gradeDCounts   int
	}

	weekMap := make(map[string]*weekStats)

	for _, digest := range digests {
		articles := articlesMap[digest.ID]
		metrics := evaluator.EvaluateDigest(&digest, articles)

		// Get week start (Monday)
		weekStart := digest.ProcessedDate
		for weekStart.Weekday() != time.Monday {
			weekStart = weekStart.AddDate(0, 0, -1)
		}
		weekKey := weekStart.Format("2006-01-02")

		if _, ok := weekMap[weekKey]; !ok {
			weekMap[weekKey] = &weekStats{weekStart: weekStart}
		}

		ws := weekMap[weekKey]
		ws.count++
		ws.avgCoverage += metrics.CoveragePct
		ws.avgVagueness += float64(metrics.VaguePhrases)
		ws.avgSpecificity += float64(metrics.SpecificityScore)

		grade := metrics.Grade[:1] // Extract letter only
		switch grade {
		case "A":
			ws.gradeACounts++
		case "B":
			ws.gradeBCounts++
		case "C":
			ws.gradeCCounts++
		case "D":
			ws.gradeDCounts++
		}
	}

	// Calculate averages and print
	fmt.Println("═══════════════════════════════════════════════════════════════════")
	fmt.Println("QUALITY TRENDS (by week)")
	fmt.Println("═══════════════════════════════════════════════════════════════════")
	fmt.Printf("%-12s  %5s  %8s  %9s  %11s  %s\n",
		"Week", "Count", "Coverage", "Vagueness", "Specificity", "Grades (A/B/C/D)")
	fmt.Println("───────────────────────────────────────────────────────────────────")

	// Sort weeks chronologically
	weeks := make([]string, 0, len(weekMap))
	for weekKey := range weekMap {
		weeks = append(weeks, weekKey)
	}
	// Simple bubble sort for small datasets
	for i := 0; i < len(weeks); i++ {
		for j := i + 1; j < len(weeks); j++ {
			if weeks[i] > weeks[j] {
				weeks[i], weeks[j] = weeks[j], weeks[i]
			}
		}
	}

	for _, weekKey := range weeks {
		ws := weekMap[weekKey]
		if ws.count == 0 {
			continue
		}

		avgCoverage := ws.avgCoverage / float64(ws.count)
		avgVagueness := ws.avgVagueness / float64(ws.count)
		avgSpecificity := ws.avgSpecificity / float64(ws.count)

		fmt.Printf("%-12s  %5d  %7.0f%%  %9.1f  %11.0f  %d/%d/%d/%d\n",
			ws.weekStart.Format("Jan 02"),
			ws.count,
			avgCoverage*100,
			avgVagueness,
			avgSpecificity,
			ws.gradeACounts, ws.gradeBCounts, ws.gradeCCounts, ws.gradeDCounts)
	}

	fmt.Println("═══════════════════════════════════════════════════════════════════")

	// Simple trend detection
	if len(weeks) >= 4 {
		// Compare first 2 weeks vs last 2 weeks
		firstHalf := weeks[:len(weeks)/2]
		secondHalf := weeks[len(weeks)/2:]

		var firstCoverage, secondCoverage float64
		var firstCount, secondCount int

		for _, weekKey := range firstHalf {
			ws := weekMap[weekKey]
			firstCoverage += ws.avgCoverage
			firstCount += ws.count
		}
		for _, weekKey := range secondHalf {
			ws := weekMap[weekKey]
			secondCoverage += ws.avgCoverage
			secondCount += ws.count
		}

		if firstCount > 0 && secondCount > 0 {
			firstAvg := (firstCoverage / float64(firstCount)) * 100
			secondAvg := (secondCoverage / float64(secondCount)) * 100
			change := secondAvg - firstAvg

			fmt.Println("📈 TREND ANALYSIS")
			fmt.Println("─────────────────────────────────────────────────────────────────")
			if change > 5 {
				fmt.Printf("🟢 Coverage improving: %.0f%% → %.0f%% (+%.1f%%)\n", firstAvg, secondAvg, change)
			} else if change < -5 {
				fmt.Printf("🔴 Coverage declining: %.0f%% → %.0f%% (%.1f%%)\n", firstAvg, secondAvg, change)
			} else {
				fmt.Printf("🟡 Coverage stable: %.0f%% → %.0f%% (%.1f%%)\n", firstAvg, secondAvg, change)
			}
			fmt.Println("═══════════════════════════════════════════════════════════════════")
		}
	}
}
