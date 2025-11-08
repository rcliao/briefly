package quality

import (
	"fmt"
	"strings"

	"briefly/internal/core"
)

// DigestEvaluator evaluates the quality of generated digests
type DigestEvaluator struct {
	thresholds QualityThresholds
}

// NewDigestEvaluator creates a new digest evaluator with default thresholds
func NewDigestEvaluator() *DigestEvaluator {
	return &DigestEvaluator{
		thresholds: DefaultThresholds(),
	}
}

// NewDigestEvaluatorWithThresholds creates an evaluator with custom thresholds
func NewDigestEvaluatorWithThresholds(thresholds QualityThresholds) *DigestEvaluator {
	return &DigestEvaluator{
		thresholds: thresholds,
	}
}

// EvaluateDigest performs comprehensive quality evaluation on a digest
func (e *DigestEvaluator) EvaluateDigest(digest *core.Digest, articles []core.Article) *DigestQualityMetrics {
	metrics := &DigestQualityMetrics{
		ArticleCount: len(articles),
		Warnings:     []string{},
	}

	// Primary summary text for evaluation
	summaryText := digest.Summary
	if summaryText == "" {
		summaryText = digest.DigestSummary // Fall back to legacy field
	}

	// Check 1: Citation coverage
	citations := ExtractCitations(summaryText)
	metrics.CitationsFound = len(citations)
	if metrics.ArticleCount > 0 {
		metrics.CoveragePct = float64(metrics.CitationsFound) / float64(metrics.ArticleCount)
	}

	// Identify uncited articles
	citationMap := make(map[int]bool)
	for _, c := range citations {
		citationMap[c] = true
	}
	for i := 1; i <= metrics.ArticleCount; i++ {
		if !citationMap[i] {
			metrics.UncitedArticles = append(metrics.UncitedArticles, i)
		}
	}

	// Add warning if coverage is low
	if metrics.CoveragePct < e.thresholds.MinCoveragePct {
		metrics.Warnings = append(metrics.Warnings,
			fmt.Sprintf("Low coverage: only %d/%d articles cited (%.0f%%)",
				metrics.CitationsFound, metrics.ArticleCount, metrics.CoveragePct*100))
		if len(metrics.UncitedArticles) > 0 {
			metrics.Warnings = append(metrics.Warnings,
				fmt.Sprintf("Missing article citations: %v", metrics.UncitedArticles))
		}
	}

	// Check 2: Vagueness detection
	vaguePhraseCount, foundPhrases := DetectVaguePhrases(summaryText)
	metrics.VaguePhrases = vaguePhraseCount
	metrics.VaguePhrasesList = foundPhrases

	if metrics.VaguePhrases > e.thresholds.MaxVaguePhrases {
		metrics.Warnings = append(metrics.Warnings,
			fmt.Sprintf("Too vague: %d generic phrases found (%v)",
				metrics.VaguePhrases, foundPhrases))
	}

	// Check 3: Length check
	metrics.WordCount = len(strings.Fields(summaryText))

	if metrics.WordCount < e.thresholds.MinWordCount {
		metrics.Warnings = append(metrics.Warnings,
			fmt.Sprintf("Too short: %d words (min: %d)",
				metrics.WordCount, e.thresholds.MinWordCount))
	} else if metrics.WordCount > e.thresholds.MaxWordCount {
		metrics.Warnings = append(metrics.Warnings,
			fmt.Sprintf("Too long: %d words (max: %d)",
				metrics.WordCount, e.thresholds.MaxWordCount))
	}

	// Check 4: Specificity checks
	metrics.NumberCount, metrics.HasNumbers = DetectNumbers(summaryText)
	metrics.ProperNounCount, metrics.HasProperNouns = DetectProperNouns(summaryText)
	metrics.SpecificityScore = CalculateSpecificityScore(
		metrics.NumberCount,
		metrics.ProperNounCount,
		metrics.VaguePhrases,
	)

	if !metrics.HasNumbers {
		metrics.Warnings = append(metrics.Warnings,
			"No specific metrics/numbers found in summary")
	}
	if !metrics.HasProperNouns {
		metrics.Warnings = append(metrics.Warnings,
			"No people/company names found in summary")
	}
	if metrics.SpecificityScore < e.thresholds.MinSpecificityScore {
		metrics.Warnings = append(metrics.Warnings,
			fmt.Sprintf("Low specificity score: %d (min: %d)",
				metrics.SpecificityScore, e.thresholds.MinSpecificityScore))
	}

	// Check 5: Citation density
	if metrics.WordCount > 0 {
		metrics.CitationDensity = float64(metrics.CitationsFound) * 100.0 / float64(metrics.WordCount)
	}

	if metrics.CitationDensity < e.thresholds.MinCitationDensity {
		metrics.Warnings = append(metrics.Warnings,
			fmt.Sprintf("Low citation density: %.1f citations per 100 words (min: %.1f)",
				metrics.CitationDensity, e.thresholds.MinCitationDensity))
	}

	// Assign grade based on all metrics
	metrics.Grade = GradeDigestQuality(metrics, e.thresholds)

	// Overall pass/fail
	metrics.Passed = metrics.CoveragePct >= e.thresholds.MinCoveragePct &&
		metrics.VaguePhrases <= e.thresholds.MaxVaguePhrases &&
		metrics.SpecificityScore >= e.thresholds.MinSpecificityScore

	return metrics
}

// EvaluateClusterNarrative evaluates a single cluster narrative for quality
func (e *DigestEvaluator) EvaluateClusterNarrative(narrative *core.ClusterNarrative, clusterSize int) *DigestQualityMetrics {
	metrics := &DigestQualityMetrics{
		ArticleCount: clusterSize,
		Warnings:     []string{},
	}

	// Check citation coverage within cluster
	if len(narrative.ArticleRefs) < clusterSize {
		metrics.Warnings = append(metrics.Warnings,
			fmt.Sprintf("Cluster narrative only references %d/%d articles",
				len(narrative.ArticleRefs), clusterSize))
	}
	metrics.CitationsFound = len(narrative.ArticleRefs)
	if clusterSize > 0 {
		metrics.CoveragePct = float64(metrics.CitationsFound) / float64(clusterSize)
	}

	// Check vagueness
	vaguePhraseCount, foundPhrases := DetectVaguePhrases(narrative.Summary)
	metrics.VaguePhrases = vaguePhraseCount
	metrics.VaguePhrasesList = foundPhrases

	if metrics.VaguePhrases > e.thresholds.MaxVaguePhrases {
		metrics.Warnings = append(metrics.Warnings,
			fmt.Sprintf("Cluster narrative too vague: %d generic phrases (%v)",
				metrics.VaguePhrases, foundPhrases))
	}

	// Check specificity
	metrics.NumberCount, metrics.HasNumbers = DetectNumbers(narrative.Summary)
	metrics.ProperNounCount, metrics.HasProperNouns = DetectProperNouns(narrative.Summary)
	metrics.SpecificityScore = CalculateSpecificityScore(
		metrics.NumberCount,
		metrics.ProperNounCount,
		metrics.VaguePhrases,
	)

	// Word count
	metrics.WordCount = len(strings.Fields(narrative.Summary))

	// Citation density
	if metrics.WordCount > 0 {
		// For cluster narratives, citations might be embedded differently
		// Count both [N] style and article references
		allRefs := ExtractCitations(narrative.Summary)
		metrics.CitationDensity = float64(len(allRefs)) * 100.0 / float64(metrics.WordCount)
	}

	// Grade
	metrics.Grade = GradeDigestQuality(metrics, e.thresholds)

	// Pass/fail
	metrics.Passed = metrics.CoveragePct >= e.thresholds.MinCoveragePct &&
		metrics.VaguePhrases <= e.thresholds.MaxVaguePhrases &&
		metrics.SpecificityScore >= e.thresholds.MinSpecificityScore

	return metrics
}

// PrintDigestReport prints a formatted quality report for a digest
func (e *DigestEvaluator) PrintDigestReport(digest *core.Digest, articles []core.Article) *DigestQualityMetrics {
	metrics := e.EvaluateDigest(digest, articles)

	title := digest.Title
	if title == "" {
		title = "Untitled Digest"
	}

	fmt.Println("============================================================")
	fmt.Printf("DIGEST QUALITY REPORT: %s\n", title)
	fmt.Println("============================================================")
	fmt.Printf("Grade: %s\n", metrics.Grade)
	fmt.Printf("Coverage: %.0f%% (%d/%d articles cited)\n",
		metrics.CoveragePct*100, metrics.CitationsFound, metrics.ArticleCount)
	fmt.Printf("Vagueness: %d generic phrases", metrics.VaguePhrases)
	if len(metrics.VaguePhrasesList) > 0 {
		fmt.Printf(" (%v)", metrics.VaguePhrasesList)
	}
	fmt.Println()
	fmt.Printf("Specificity Score: %d/100\n", metrics.SpecificityScore)
	fmt.Printf("  - Numbers/metrics: %d\n", metrics.NumberCount)
	fmt.Printf("  - Proper nouns: %d\n", metrics.ProperNounCount)
	fmt.Printf("Length: %d words\n", metrics.WordCount)
	fmt.Printf("Citation Density: %.1f citations per 100 words\n", metrics.CitationDensity)

	if len(metrics.Warnings) > 0 {
		fmt.Println("\n⚠️  WARNINGS:")
		for _, warning := range metrics.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	} else {
		fmt.Println("\n✅ No issues detected")
	}

	fmt.Println("============================================================")
	fmt.Println()

	return metrics
}

// AuditRecentDigests audits multiple digests and provides aggregate statistics
func (e *DigestEvaluator) AuditRecentDigests(digests []core.Digest, articlesMap map[string][]core.Article) *AuditReport {
	report := &AuditReport{
		TotalDigests:  len(digests),
		GradeCounts:   make(map[string]int),
		DigestMetrics: []DigestQualityMetrics{},
	}

	for _, digest := range digests {
		articles := articlesMap[digest.ID]
		metrics := e.EvaluateDigest(&digest, articles)

		report.DigestMetrics = append(report.DigestMetrics, *metrics)
		report.TotalCoverage += metrics.CoveragePct
		report.TotalVagueness += float64(metrics.VaguePhrases)
		report.TotalSpecificity += float64(metrics.SpecificityScore)

		// Count grades (extract letter only: "A - EXCELLENT" -> "A")
		gradeLetter := strings.Split(metrics.Grade, " ")[0]
		report.GradeCounts[gradeLetter]++
	}

	// Calculate averages
	if report.TotalDigests > 0 {
		report.AvgCoverage = report.TotalCoverage / float64(report.TotalDigests)
		report.AvgVagueness = report.TotalVagueness / float64(report.TotalDigests)
		report.AvgSpecificity = report.TotalSpecificity / float64(report.TotalDigests)
	}

	// Determine recommendation
	if report.AvgCoverage < 0.8 {
		report.Recommendation = "🔴 CRITICAL: Low coverage - consider per-article extraction approach"
	} else if report.AvgVagueness > 2.0 {
		report.Recommendation = "🟡 WARNING: High vagueness - improve specificity in prompts"
	} else if report.AvgSpecificity < 50.0 {
		report.Recommendation = "🟡 WARNING: Low specificity - enforce fact extraction in prompts"
	} else {
		report.Recommendation = "🟢 GOOD: Current approach producing acceptable quality"
	}

	return report
}

// PrintAuditReport prints a formatted audit report
func (e *DigestEvaluator) PrintAuditReport(report *AuditReport) {
	fmt.Println("============================================================")
	fmt.Printf("DIGEST QUALITY AUDIT REPORT (%d digests)\n", report.TotalDigests)
	fmt.Println("============================================================")
	fmt.Printf("Average Coverage: %.0f%%\n", report.AvgCoverage*100)
	fmt.Printf("Average Vagueness: %.1f phrases per digest\n", report.AvgVagueness)
	fmt.Printf("Average Specificity: %.0f/100\n", report.AvgSpecificity)
	fmt.Println("\nGrade Distribution:")
	for _, grade := range []string{"A", "B", "C", "D"} {
		count := report.GradeCounts[grade]
		if count > 0 {
			pct := float64(count) * 100.0 / float64(report.TotalDigests)
			fmt.Printf("  %s: %d digests (%.0f%%)\n", grade, count, pct)
		}
	}
	fmt.Printf("\n%s\n", report.Recommendation)
	fmt.Println("============================================================")
}

// AuditReport contains aggregate statistics from auditing multiple digests
type AuditReport struct {
	TotalDigests   int
	AvgCoverage    float64
	AvgVagueness   float64
	AvgSpecificity float64
	GradeCounts    map[string]int
	Recommendation string
	DigestMetrics  []DigestQualityMetrics

	// Totals for averaging
	TotalCoverage    float64
	TotalVagueness   float64
	TotalSpecificity float64
}
