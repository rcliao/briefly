package pipeline

import (
	"context"
	"fmt"

	"briefly/internal/core"
	"briefly/internal/quality"
)

// QualityGate represents a validation checkpoint in the pipeline
type QualityGate interface {
	// Validate checks if the stage output meets quality requirements
	Validate(ctx context.Context) error

	// Name returns the gate name for logging
	Name() string

	// IsBlocking returns whether failure should stop the pipeline
	IsBlocking() bool
}

// QualityGateConfig holds configuration for quality gates
type QualityGateConfig struct {
	EnableClusteringGate bool    // Validate clustering quality
	EnableNarrativeGate  bool    // Validate cluster narratives
	EnableDigestGate     bool    // Validate final digest
	MinSilhouette        float64 // Minimum silhouette score for clustering
	MinCoverage          float64 // Minimum article coverage for digests
	MaxVagueness         int     // Maximum vague phrases allowed
	MinSpecificity       int     // Minimum specificity score
	BlockOnFailure       bool    // Stop pipeline on gate failure
}

// DefaultQualityGateConfig returns default configuration
func DefaultQualityGateConfig() QualityGateConfig {
	return QualityGateConfig{
		EnableClusteringGate: true,
		EnableNarrativeGate:  true,
		EnableDigestGate:     true,
		MinSilhouette:        0.3,
		MinCoverage:          0.80,
		MaxVagueness:         2,
		MinSpecificity:       50,
		BlockOnFailure:       false, // Non-blocking by default (warn only)
	}
}

// ============================================================================
// Clustering Quality Gate
// ============================================================================

// ClusteringQualityGate validates clustering results
type ClusteringQualityGate struct {
	config             QualityGateConfig
	clusters           []core.TopicCluster
	embeddings         map[string][]float64
	coherenceEvaluator *quality.ClusterCoherenceEvaluator
	lastMetrics        *quality.ClusterCoherenceMetrics // Stores metrics for persistence
}

// NewClusteringQualityGate creates a new clustering quality gate
func NewClusteringQualityGate(
	config QualityGateConfig,
	clusters []core.TopicCluster,
	embeddings map[string][]float64,
) *ClusteringQualityGate {
	return &ClusteringQualityGate{
		config:             config,
		clusters:           clusters,
		embeddings:         embeddings,
		coherenceEvaluator: quality.NewClusterCoherenceEvaluator(),
	}
}

// Name returns the gate name
func (g *ClusteringQualityGate) Name() string {
	return "Clustering Quality Gate"
}

// IsBlocking returns whether this gate blocks the pipeline
func (g *ClusteringQualityGate) IsBlocking() bool {
	return g.config.BlockOnFailure
}

// GetMetrics returns the last evaluated cluster coherence metrics
// Call this after Validate() to retrieve metrics for persistence
func (g *ClusteringQualityGate) GetMetrics() *quality.ClusterCoherenceMetrics {
	return g.lastMetrics
}

// Validate checks clustering quality
func (g *ClusteringQualityGate) Validate(ctx context.Context) error {
	if !g.config.EnableClusteringGate {
		return nil // Gate disabled
	}

	fmt.Printf("\n🔒 %s: Validating clustering quality...\n", g.Name())

	// Evaluate cluster coherence
	metrics := g.coherenceEvaluator.EvaluateClusterCoherence(g.clusters, g.embeddings)
	g.lastMetrics = metrics // Store for later retrieval

	// Print detailed cohesion report
	g.printDetailedCohesionReport(metrics)

	// Check silhouette score
	if metrics.AvgSilhouette < g.config.MinSilhouette {
		err := fmt.Errorf("clustering quality below threshold: silhouette=%.3f (min: %.3f)",
			metrics.AvgSilhouette, g.config.MinSilhouette)

		if g.IsBlocking() {
			fmt.Printf("   ❌ GATE FAILED (blocking): %v\n", err)
			return err
		}

		fmt.Printf("   ⚠️  GATE WARNING (non-blocking): %v\n", err)
		return nil // Non-blocking, continue with warning
	}

	// Check for clustering issues
	if len(metrics.Issues) > 0 {
		fmt.Printf("\n   ⚠️  Clustering issues detected:\n")
		for _, issue := range metrics.Issues {
			fmt.Printf("      - %s\n", issue)
		}

		if g.IsBlocking() && metrics.AvgSilhouette < g.config.MinSilhouette*1.2 {
			return fmt.Errorf("clustering quality issues detected")
		}
	}

	fmt.Printf("\n   ✓ Clustering quality acceptable\n")

	return nil
}

// printDetailedCohesionReport prints per-cluster cohesion breakdown
func (g *ClusteringQualityGate) printDetailedCohesionReport(metrics *quality.ClusterCoherenceMetrics) {
	fmt.Println()
	fmt.Printf("   📊 Cluster Cohesion Metrics:\n")
	fmt.Printf("      Grade: %s\n", metrics.CoherenceGrade)
	fmt.Printf("      Avg Silhouette: %.3f (threshold: %.2f)\n", metrics.AvgSilhouette, g.config.MinSilhouette)
	fmt.Printf("      Avg Intra-Cluster Similarity: %.3f\n", metrics.AvgIntraClusterSimilarity)
	fmt.Printf("      Avg Inter-Cluster Distance: %.3f\n", metrics.AvgInterClusterDistance)
	fmt.Println()

	// Per-cluster breakdown
	fmt.Printf("      Per-Cluster Breakdown:\n")
	fmt.Printf("      %-3s %-30s %-10s %-10s\n", "#", "Cluster Label", "Articles", "Cohesion")
	fmt.Printf("      %-3s %-30s %-10s %-10s\n", "---", "------------------------------", "--------", "--------")

	for i, cluster := range g.clusters {
		// Get cohesion score for this cluster
		cohesion := 0.0
		if i < len(metrics.IntraClusterSimilarities) {
			cohesion = metrics.IntraClusterSimilarities[i]
		}

		// Determine indicator based on cohesion quality
		indicator := "✓"
		if cohesion < 0.5 && cohesion >= 0.3 {
			indicator = "⚠"
		} else if cohesion < 0.3 {
			indicator = "✗"
		}

		// Truncate long labels
		label := cluster.Label
		if len(label) > 28 {
			label = label[:25] + "..."
		}

		fmt.Printf("      %-3d %-30s %-10d %.3f %s\n",
			i+1, label, len(cluster.ArticleIDs), cohesion, indicator)
	}

	fmt.Println()
	fmt.Printf("      Legend: ✓ Good (>=0.5)  ⚠ Fair (>=0.3)  ✗ Poor (<0.3)\n")
}

// ============================================================================
// Narrative Quality Gate
// ============================================================================

// NarrativeQualityGate validates cluster narratives
type NarrativeQualityGate struct {
	config          QualityGateConfig
	clusters        []core.TopicCluster
	digestEvaluator *quality.DigestEvaluator
}

// NewNarrativeQualityGate creates a new narrative quality gate
func NewNarrativeQualityGate(
	config QualityGateConfig,
	clusters []core.TopicCluster,
) *NarrativeQualityGate {
	return &NarrativeQualityGate{
		config:          config,
		clusters:        clusters,
		digestEvaluator: quality.NewDigestEvaluator(),
	}
}

// Name returns the gate name
func (g *NarrativeQualityGate) Name() string {
	return "Narrative Quality Gate"
}

// IsBlocking returns whether this gate blocks the pipeline
func (g *NarrativeQualityGate) IsBlocking() bool {
	return g.config.BlockOnFailure
}

// Validate checks cluster narrative quality
func (g *NarrativeQualityGate) Validate(ctx context.Context) error {
	if !g.config.EnableNarrativeGate {
		return nil // Gate disabled
	}

	fmt.Printf("\n🔒 %s: Validating cluster narratives...\n", g.Name())

	issueCount := 0
	for i, cluster := range g.clusters {
		if cluster.Narrative == nil {
			fmt.Printf("   ⚠️  Cluster %d: Missing narrative\n", i+1)
			issueCount++
			continue
		}

		// Evaluate narrative quality
		metrics := g.digestEvaluator.EvaluateClusterNarrative(cluster.Narrative, len(cluster.ArticleIDs))

		// Check coverage
		if metrics.CoveragePct < g.config.MinCoverage {
			fmt.Printf("   ⚠️  Cluster %d: Low coverage %.0f%% (expected: %.0f%%)\n",
				i+1, metrics.CoveragePct*100, g.config.MinCoverage*100)
			issueCount++
		}

		// Check vagueness
		if metrics.VaguePhrases > g.config.MaxVagueness {
			fmt.Printf("   ⚠️  Cluster %d: Too vague (%d phrases, max: %d)\n",
				i+1, metrics.VaguePhrases, g.config.MaxVagueness)
			issueCount++
		}

		// Check specificity
		if metrics.SpecificityScore < g.config.MinSpecificity {
			fmt.Printf("   ⚠️  Cluster %d: Low specificity (%d, min: %d)\n",
				i+1, metrics.SpecificityScore, g.config.MinSpecificity)
			issueCount++
		}
	}

	if issueCount > 0 {
		err := fmt.Errorf("%d cluster narratives have quality issues", issueCount)

		if g.IsBlocking() {
			fmt.Printf("   ❌ GATE FAILED (blocking): %v\n", err)
			return err
		}

		fmt.Printf("   ⚠️  GATE WARNING (non-blocking): %v\n", err)
		return nil
	}

	fmt.Printf("   ✓ All cluster narratives meet quality standards\n")
	return nil
}

// ============================================================================
// Digest Quality Gate
// ============================================================================

// DigestQualityGate validates final digest quality
type DigestQualityGate struct {
	config          QualityGateConfig
	digest          *core.Digest
	articles        []core.Article
	digestEvaluator *quality.DigestEvaluator
}

// NewDigestQualityGate creates a new digest quality gate
func NewDigestQualityGate(
	config QualityGateConfig,
	digest *core.Digest,
	articles []core.Article,
) *DigestQualityGate {
	return &DigestQualityGate{
		config:          config,
		digest:          digest,
		articles:        articles,
		digestEvaluator: quality.NewDigestEvaluator(),
	}
}

// Name returns the gate name
func (g *DigestQualityGate) Name() string {
	return "Digest Quality Gate"
}

// IsBlocking returns whether this gate blocks the pipeline
func (g *DigestQualityGate) IsBlocking() bool {
	return g.config.BlockOnFailure
}

// Validate checks final digest quality
func (g *DigestQualityGate) Validate(ctx context.Context) error {
	if !g.config.EnableDigestGate {
		return nil // Gate disabled
	}

	fmt.Printf("\n🔒 %s: Validating final digest quality...\n", g.Name())

	// Evaluate digest quality
	metrics := g.digestEvaluator.EvaluateDigest(g.digest, g.articles)

	// Check coverage
	if metrics.CoveragePct < g.config.MinCoverage {
		err := fmt.Errorf("low article coverage: %.0f%% (min: %.0f%%)",
			metrics.CoveragePct*100, g.config.MinCoverage*100)

		if g.IsBlocking() {
			fmt.Printf("   ❌ GATE FAILED (blocking): %v\n", err)
			return err
		}

		fmt.Printf("   ⚠️  GATE WARNING (non-blocking): %v\n", err)
	}

	// Check vagueness
	if metrics.VaguePhrases > g.config.MaxVagueness {
		err := fmt.Errorf("too many vague phrases: %d (max: %d)",
			metrics.VaguePhrases, g.config.MaxVagueness)

		if g.IsBlocking() {
			fmt.Printf("   ❌ GATE FAILED (blocking): %v\n", err)
			return err
		}

		fmt.Printf("   ⚠️  GATE WARNING (non-blocking): %v\n", err)
	}

	// Check specificity
	if metrics.SpecificityScore < g.config.MinSpecificity {
		err := fmt.Errorf("low specificity score: %d (min: %d)",
			metrics.SpecificityScore, g.config.MinSpecificity)

		if g.IsBlocking() {
			fmt.Printf("   ❌ GATE FAILED (blocking): %v\n", err)
			return err
		}

		fmt.Printf("   ⚠️  GATE WARNING (non-blocking): %v\n", err)
	}

	// Print quality report
	fmt.Printf("   📊 Quality Metrics:\n")
	fmt.Printf("      Coverage: %.0f%% (%d/%d articles)\n",
		metrics.CoveragePct*100, metrics.CitationsFound, metrics.ArticleCount)
	fmt.Printf("      Vagueness: %d phrases\n", metrics.VaguePhrases)
	fmt.Printf("      Specificity: %d/100\n", metrics.SpecificityScore)
	fmt.Printf("      Grade: %s\n", metrics.Grade)

	if metrics.Passed {
		fmt.Printf("   ✓ Digest meets quality standards\n")
	} else {
		if g.IsBlocking() {
			return fmt.Errorf("digest failed quality gate: grade %s", metrics.Grade)
		}
		fmt.Printf("   ⚠️  Digest quality below target but continuing (non-blocking)\n")
	}

	return nil
}

// ============================================================================
// Quality Gate Runner
// ============================================================================

// QualityGateRunner executes a series of quality gates
type QualityGateRunner struct {
	gates []QualityGate
}

// NewQualityGateRunner creates a new gate runner
func NewQualityGateRunner() *QualityGateRunner {
	return &QualityGateRunner{
		gates: []QualityGate{},
	}
}

// AddGate adds a quality gate to the runner
func (r *QualityGateRunner) AddGate(gate QualityGate) {
	r.gates = append(r.gates, gate)
}

// RunGates executes all gates in sequence
func (r *QualityGateRunner) RunGates(ctx context.Context) error {
	if len(r.gates) == 0 {
		return nil // No gates configured
	}

	fmt.Printf("\n🚦 Running %d quality gates...\n", len(r.gates))

	passedCount := 0
	warningCount := 0

	for _, gate := range r.gates {
		err := gate.Validate(ctx)
		if err != nil {
			if gate.IsBlocking() {
				// Blocking gate failed, stop pipeline
				fmt.Printf("\n❌ Pipeline stopped at: %s\n", gate.Name())
				return err
			}
			// Non-blocking gate, count as warning
			warningCount++
		} else {
			passedCount++
		}
	}

	fmt.Printf("\n✅ Quality gates complete: %d passed", passedCount)
	if warningCount > 0 {
		fmt.Printf(", %d warnings", warningCount)
	}
	fmt.Println()

	return nil
}
