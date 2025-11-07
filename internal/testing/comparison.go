package testing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"briefly/internal/core"
	"briefly/internal/quality"
)

// ComparisonResult holds results from comparing two digests
type ComparisonResult struct {
	DigestA         *core.Digest
	DigestB         *core.Digest
	MetricsA        *quality.DigestQualityMetrics
	MetricsB        *quality.DigestQualityMetrics
	Winner          string // "A", "B", or "Tie"
	QualityDelta    *QualityDelta
	Recommendation  string
	ComparisonDate  time.Time
}

// QualityDelta represents the difference between two digest quality metrics
type QualityDelta struct {
	CoverageDelta      float64 // Percentage point difference
	VaguenessDelta     int     // Lower is better (negative = improvement)
	SpecificityDelta   int     // Higher is better (positive = improvement)
	CitationDelta      int     // Absolute citation count difference
	GradeImprovement   int     // Letter grade improvement (A=4, B=3, C=2, D=1)
}

// ComparisonFramework provides A/B testing capabilities for digests
type ComparisonFramework struct {
	evaluator *quality.DigestEvaluator
}

// NewComparisonFramework creates a new comparison framework
func NewComparisonFramework() *ComparisonFramework {
	return &ComparisonFramework{
		evaluator: quality.NewDigestEvaluator(),
	}
}

// CompareDigests performs comprehensive comparison between two digests
func (cf *ComparisonFramework) CompareDigests(
	ctx context.Context,
	digestA *core.Digest,
	digestB *core.Digest,
	articles []core.Article,
) (*ComparisonResult, error) {
	// Evaluate both digests
	metricsA := cf.evaluator.EvaluateDigest(digestA, articles)
	metricsB := cf.evaluator.EvaluateDigest(digestB, articles)

	// Calculate deltas
	delta := cf.calculateDelta(metricsA, metricsB)

	// Determine winner
	winner := cf.determineWinner(metricsA, metricsB, delta)

	// Generate recommendation
	recommendation := cf.generateRecommendation(winner, delta, metricsA, metricsB)

	return &ComparisonResult{
		DigestA:        digestA,
		DigestB:        digestB,
		MetricsA:       metricsA,
		MetricsB:       metricsB,
		Winner:         winner,
		QualityDelta:   delta,
		Recommendation: recommendation,
		ComparisonDate: time.Now().UTC(),
	}, nil
}

// calculateDelta computes quality differences between two metrics
func (cf *ComparisonFramework) calculateDelta(
	metricsA *quality.DigestQualityMetrics,
	metricsB *quality.DigestQualityMetrics,
) *QualityDelta {
	return &QualityDelta{
		CoverageDelta:    (metricsB.CoveragePct - metricsA.CoveragePct) * 100, // Percentage points
		VaguenessDelta:   metricsB.VaguePhrases - metricsA.VaguePhrases,       // Negative = improvement
		SpecificityDelta: metricsB.SpecificityScore - metricsA.SpecificityScore, // Positive = improvement
		CitationDelta:    metricsB.CitationsFound - metricsA.CitationsFound,
		GradeImprovement: cf.gradeToNumeric(metricsB.Grade) - cf.gradeToNumeric(metricsA.Grade),
	}
}

// gradeToNumeric converts letter grade to numeric (A=4, B=3, C=2, D=1, F=0)
func (cf *ComparisonFramework) gradeToNumeric(grade string) int {
	switch grade {
	case "A":
		return 4
	case "B":
		return 3
	case "C":
		return 2
	case "D":
		return 1
	default:
		return 0
	}
}

// determineWinner decides which digest is higher quality
func (cf *ComparisonFramework) determineWinner(
	metricsA *quality.DigestQualityMetrics,
	metricsB *quality.DigestQualityMetrics,
	delta *QualityDelta,
) string {
	// Weight different factors
	scoreA := 0.0
	scoreB := 0.0

	// Coverage weight: 30%
	scoreA += metricsA.CoveragePct * 0.30
	scoreB += metricsB.CoveragePct * 0.30

	// Vagueness weight: 25% (inverted - fewer is better)
	// Normalize to 0-1 scale (assume max vagueness = 10)
	vagueScoreA := float64(10-metricsA.VaguePhrases) / 10.0
	vagueScoreB := float64(10-metricsB.VaguePhrases) / 10.0
	if vagueScoreA < 0 {
		vagueScoreA = 0
	}
	if vagueScoreB < 0 {
		vagueScoreB = 0
	}
	scoreA += vagueScoreA * 0.25
	scoreB += vagueScoreB * 0.25

	// Specificity weight: 25%
	// Normalize to 0-1 scale (0-100)
	scoreA += float64(metricsA.SpecificityScore) / 100.0 * 0.25
	scoreB += float64(metricsB.SpecificityScore) / 100.0 * 0.25

	// Grade weight: 20%
	gradeScoreA := float64(cf.gradeToNumeric(metricsA.Grade)) / 4.0
	gradeScoreB := float64(cf.gradeToNumeric(metricsB.Grade)) / 4.0
	scoreA += gradeScoreA * 0.20
	scoreB += gradeScoreB * 0.20

	// Determine winner (threshold: 3% difference required)
	diff := scoreB - scoreA
	if diff > 0.03 {
		return "B"
	} else if diff < -0.03 {
		return "A"
	}
	return "Tie"
}

// generateRecommendation creates actionable recommendation text
func (cf *ComparisonFramework) generateRecommendation(
	winner string,
	delta *QualityDelta,
	metricsA *quality.DigestQualityMetrics,
	metricsB *quality.DigestQualityMetrics,
) string {
	if winner == "Tie" {
		return "Both digests are of similar quality. Consider user preference or other factors."
	}

	winnerMetrics := metricsA
	loserMetrics := metricsB
	winnerName := "Digest A"
	if winner == "B" {
		winnerMetrics = metricsB
		loserMetrics = metricsA
		winnerName = "Digest B"
	}

	recommendation := fmt.Sprintf("✅ **%s** is recommended (Grade: %s vs %s)\n\n",
		winnerName, winnerMetrics.Grade, loserMetrics.Grade)

	recommendation += "**Key Improvements:**\n"

	if delta.CoverageDelta > 5 {
		recommendation += fmt.Sprintf("- Better coverage: +%.1f%% more articles cited\n", delta.CoverageDelta)
	} else if delta.CoverageDelta < -5 {
		recommendation += fmt.Sprintf("- Better coverage: +%.1f%% more articles cited\n", -delta.CoverageDelta)
	}

	if delta.VaguenessDelta < -1 {
		recommendation += fmt.Sprintf("- Less vague: %d fewer generic phrases\n", -delta.VaguenessDelta)
	} else if delta.VaguenessDelta > 1 {
		recommendation += fmt.Sprintf("- Less vague: %d fewer generic phrases\n", delta.VaguenessDelta)
	}

	if delta.SpecificityDelta > 5 {
		recommendation += fmt.Sprintf("- More specific: +%d specificity score\n", delta.SpecificityDelta)
	} else if delta.SpecificityDelta < -5 {
		recommendation += fmt.Sprintf("- More specific: +%d specificity score\n", -delta.SpecificityDelta)
	}

	if delta.GradeImprovement > 0 {
		recommendation += fmt.Sprintf("- Quality grade improved by %d letter(s)\n", delta.GradeImprovement)
	} else if delta.GradeImprovement < 0 {
		recommendation += fmt.Sprintf("- Quality grade improved by %d letter(s)\n", -delta.GradeImprovement)
	}

	return recommendation
}

// PrintComparisonReport generates a formatted comparison report
func (cf *ComparisonFramework) PrintComparisonReport(result *ComparisonResult) {
	fmt.Println("\n" + strings.Repeat("═", 80))
	fmt.Println("📊 DIGEST A/B COMPARISON REPORT")
	fmt.Println(strings.Repeat("═", 80) + "\n")

	// Winner announcement
	fmt.Printf("🏆 Winner: %s\n\n", result.Winner)

	// Side-by-side metrics
	fmt.Println("┌─────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│                          QUALITY METRICS COMPARISON                         │")
	fmt.Println("├─────────────────────────────────┬───────────────┬───────────────┬───────────┤")
	fmt.Println("│ Metric                          │   Digest A    │   Digest B    │   Delta   │")
	fmt.Println("├─────────────────────────────────┼───────────────┼───────────────┼───────────┤")

	// Coverage
	fmt.Printf("│ Coverage                        │    %.0f%%       │    %.0f%%       │   %+.0f%%    │\n",
		result.MetricsA.CoveragePct*100,
		result.MetricsB.CoveragePct*100,
		result.QualityDelta.CoverageDelta)

	// Citations
	fmt.Printf("│ Citations Found                 │      %2d       │      %2d       │   %+3d     │\n",
		result.MetricsA.CitationsFound,
		result.MetricsB.CitationsFound,
		result.QualityDelta.CitationDelta)

	// Vagueness
	fmt.Printf("│ Vague Phrases (lower=better)    │      %2d       │      %2d       │   %+3d     │\n",
		result.MetricsA.VaguePhrases,
		result.MetricsB.VaguePhrases,
		result.QualityDelta.VaguenessDelta)

	// Specificity
	fmt.Printf("│ Specificity Score (0-100)       │      %2d       │      %2d       │   %+3d     │\n",
		result.MetricsA.SpecificityScore,
		result.MetricsB.SpecificityScore,
		result.QualityDelta.SpecificityDelta)

	// Grade
	fmt.Printf("│ Quality Grade                   │      %s        │      %s        │    %s      │\n",
		result.MetricsA.Grade,
		result.MetricsB.Grade,
		cf.formatGradeDelta(result.QualityDelta.GradeImprovement))

	// Pass/Fail
	passA := "PASS"
	if !result.MetricsA.Passed {
		passA = "FAIL"
	}
	passB := "PASS"
	if !result.MetricsB.Passed {
		passB = "FAIL"
	}
	fmt.Printf("│ Overall Status                  │     %s      │     %s      │           │\n",
		passA, passB)

	fmt.Println("└─────────────────────────────────┴───────────────┴───────────────┴───────────┘\n")

	// Recommendation
	fmt.Println("📝 RECOMMENDATION:")
	fmt.Println(result.Recommendation)

	// Warnings (if any)
	if len(result.MetricsA.Warnings) > 0 || len(result.MetricsB.Warnings) > 0 {
		fmt.Println("\n⚠️  WARNINGS:")
		if len(result.MetricsA.Warnings) > 0 {
			fmt.Println("\nDigest A:")
			for _, warning := range result.MetricsA.Warnings {
				fmt.Printf("  - %s\n", warning)
			}
		}
		if len(result.MetricsB.Warnings) > 0 {
			fmt.Println("\nDigest B:")
			for _, warning := range result.MetricsB.Warnings {
				fmt.Printf("  - %s\n", warning)
			}
		}
	}

	fmt.Println("\n" + strings.Repeat("═", 80) + "\n")
}

// formatGradeDelta formats grade improvement as +/- notation
func (cf *ComparisonFramework) formatGradeDelta(improvement int) string {
	if improvement > 0 {
		return fmt.Sprintf("+%d", improvement)
	} else if improvement < 0 {
		return fmt.Sprintf("%d", improvement)
	}
	return "="
}

// BatchCompare compares multiple digest versions against a baseline
type BatchCompareResult struct {
	Baseline       *core.Digest
	BaselineMetrics *quality.DigestQualityMetrics
	Variants       []VariantResult
	BestVariant    string
}

// VariantResult holds comparison results for a single variant
type VariantResult struct {
	VariantName string
	Digest      *core.Digest
	Metrics     *quality.DigestQualityMetrics
	Delta       *QualityDelta
	Score       float64 // Composite quality score
}

// BatchCompareDigests compares multiple variants against a baseline
func (cf *ComparisonFramework) BatchCompareDigests(
	ctx context.Context,
	baseline *core.Digest,
	variants map[string]*core.Digest, // variant name -> digest
	articles []core.Article,
) (*BatchCompareResult, error) {
	// Evaluate baseline
	baselineMetrics := cf.evaluator.EvaluateDigest(baseline, articles)

	result := &BatchCompareResult{
		Baseline:       baseline,
		BaselineMetrics: baselineMetrics,
		Variants:       make([]VariantResult, 0, len(variants)),
	}

	bestScore := -1.0
	bestVariantName := ""

	// Evaluate each variant
	for name, digest := range variants {
		metrics := cf.evaluator.EvaluateDigest(digest, articles)
		delta := cf.calculateDelta(baselineMetrics, metrics)

		// Calculate composite score
		score := cf.calculateCompositeScore(metrics)

		variant := VariantResult{
			VariantName: name,
			Digest:      digest,
			Metrics:     metrics,
			Delta:       delta,
			Score:       score,
		}

		result.Variants = append(result.Variants, variant)

		if score > bestScore {
			bestScore = score
			bestVariantName = name
		}
	}

	result.BestVariant = bestVariantName

	return result, nil
}

// calculateCompositeScore computes weighted quality score (0-1 scale)
func (cf *ComparisonFramework) calculateCompositeScore(metrics *quality.DigestQualityMetrics) float64 {
	score := 0.0

	// Coverage: 30%
	score += metrics.CoveragePct * 0.30

	// Vagueness: 25% (inverted)
	vagueScore := float64(10-metrics.VaguePhrases) / 10.0
	if vagueScore < 0 {
		vagueScore = 0
	}
	if vagueScore > 1 {
		vagueScore = 1
	}
	score += vagueScore * 0.25

	// Specificity: 25%
	score += float64(metrics.SpecificityScore) / 100.0 * 0.25

	// Grade: 20%
	gradeScore := float64(cf.gradeToNumeric(metrics.Grade)) / 4.0
	score += gradeScore * 0.20

	return score
}

// PrintBatchComparisonReport generates a batch comparison report
func (cf *ComparisonFramework) PrintBatchComparisonReport(result *BatchCompareResult) {
	fmt.Println("\n" + strings.Repeat("═", 100))
	fmt.Println("📊 BATCH DIGEST COMPARISON REPORT")
	fmt.Println(strings.Repeat("═", 100) + "\n")

	fmt.Printf("🏆 Best Variant: %s (Score: %.3f)\n\n", result.BestVariant, cf.getBestScore(result))

	// Baseline metrics
	fmt.Println("📍 BASELINE METRICS:")
	fmt.Printf("   Coverage: %.0f%% | Vagueness: %d | Specificity: %d | Grade: %s\n\n",
		result.BaselineMetrics.CoveragePct*100,
		result.BaselineMetrics.VaguePhrases,
		result.BaselineMetrics.SpecificityScore,
		result.BaselineMetrics.Grade)

	// Variant comparison table
	fmt.Println("┌────────────────────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│                              VARIANT COMPARISON TABLE                                      │")
	fmt.Println("├──────────────┬──────────┬───────────┬────────────┬───────┬───────────────────────────────┤")
	fmt.Println("│ Variant      │ Coverage │ Vagueness │ Specificity│ Grade │ Delta vs Baseline             │")
	fmt.Println("├──────────────┼──────────┼───────────┼────────────┼───────┼───────────────────────────────┤")

	for _, variant := range result.Variants {
		fmt.Printf("│ %-12s │   %.0f%%    │    %2d     │     %2d     │   %s   │ Cov:%+.0f%% Vag:%+d Spec:%+d  │\n",
			variant.VariantName,
			variant.Metrics.CoveragePct*100,
			variant.Metrics.VaguePhrases,
			variant.Metrics.SpecificityScore,
			variant.Metrics.Grade,
			variant.Delta.CoverageDelta,
			variant.Delta.VaguenessDelta,
			variant.Delta.SpecificityDelta)
	}

	fmt.Println("└──────────────┴──────────┴───────────┴────────────┴───────┴───────────────────────────────┘\n")

	fmt.Println(strings.Repeat("═", 100) + "\n")
}

// getBestScore retrieves the highest composite score from batch results
func (cf *ComparisonFramework) getBestScore(result *BatchCompareResult) float64 {
	bestScore := 0.0
	for _, variant := range result.Variants {
		if variant.Score > bestScore {
			bestScore = variant.Score
		}
	}
	return bestScore
}
