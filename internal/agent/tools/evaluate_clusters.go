package tools

import (
	"briefly/internal/agent"
	"context"
	"fmt"
	"math"

	"google.golang.org/genai"
)

// EvaluateClustersTool evaluates cluster quality using embedding distances.
type EvaluateClustersTool struct{}

// NewEvaluateClustersTool creates a new cluster evaluation tool.
func NewEvaluateClustersTool() *EvaluateClustersTool {
	return &EvaluateClustersTool{}
}

func (t *EvaluateClustersTool) Name() string { return "evaluate_clusters" }

func (t *EvaluateClustersTool) Description() string {
	return "Evaluate the quality of current clusters. Scores coherence (how topically similar articles within a cluster are) and separation (how distinct clusters are from each other). Suggests improvements like merging or splitting."
}

func (t *EvaluateClustersTool) Parameters() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"cluster_ids": {
				Type:        genai.TypeArray,
				Items:       &genai.Schema{Type: genai.TypeString},
				Description: "IDs of clusters to evaluate. Omit to evaluate all.",
			},
		},
	}
}

func (t *EvaluateClustersTool) Execute(ctx context.Context, memory *agent.WorkingMemory, params map[string]any) (map[string]any, error) {
	clusters := memory.GetClusters()
	embeddings := memory.GetEmbeddings()

	if len(clusters) == 0 {
		return nil, fmt.Errorf("no clusters to evaluate")
	}

	evals := make([]agent.ClusterEvaluation, 0, len(clusters))
	evalResults := make([]map[string]any, 0, len(clusters))
	var totalCoherence, totalSeparation float64

	for i, cluster := range clusters {
		coherence := computeCoherence(cluster.ArticleIDs, cluster.Centroid, embeddings)

		// Compute separation: min distance from this centroid to all other centroids
		separation := 1.0
		for j, other := range clusters {
			if i == j || len(cluster.Centroid) == 0 || len(other.Centroid) == 0 {
				continue
			}
			sim := cosineSim(cluster.Centroid, other.Centroid)
			dist := 1.0 - sim
			if dist < separation {
				separation = dist
			}
		}

		sizeAppropriateness := "appropriate"
		suggestedAction := "keep"
		reasoning := ""

		if len(cluster.ArticleIDs) < 2 {
			sizeAppropriateness = "too_small"
			suggestedAction = "dissolve"
			reasoning = "Cluster has fewer than 2 articles"
		} else if len(cluster.ArticleIDs) > 10 {
			sizeAppropriateness = "too_large"
			suggestedAction = "split"
			reasoning = "Cluster has more than 10 articles, may be too broad"
		} else if coherence < 0.3 {
			suggestedAction = "split"
			reasoning = fmt.Sprintf("Low coherence (%.2f) suggests articles are not topically similar", coherence)
		} else if separation < 0.2 {
			suggestedAction = "merge"
			reasoning = fmt.Sprintf("Low separation (%.2f) suggests overlap with another cluster", separation)
		}

		eval := agent.ClusterEvaluation{
			ClusterID:           cluster.ID,
			Label:               cluster.Label,
			CoherenceScore:      coherence,
			SeparationScore:     separation,
			SizeAppropriateness: sizeAppropriateness,
			SuggestedAction:     suggestedAction,
			Reasoning:           reasoning,
		}
		evals = append(evals, eval)
		totalCoherence += coherence
		totalSeparation += separation

		evalResults = append(evalResults, map[string]any{
			"cluster_id":           cluster.ID,
			"label":                cluster.Label,
			"coherence_score":      coherence,
			"separation_score":     separation,
			"size_appropriateness": sizeAppropriateness,
			"suggested_action":     suggestedAction,
			"reasoning":            reasoning,
		})
	}

	memory.SetClusterEvaluations(evals)

	avgCoherence := totalCoherence / float64(len(clusters))
	avgSeparation := totalSeparation / float64(len(clusters))

	overallQuality := "good"
	if avgCoherence < 0.3 || avgSeparation < 0.2 {
		overallQuality = "needs_improvement"
	} else if avgCoherence < 0.5 || avgSeparation < 0.4 {
		overallQuality = "acceptable"
	}

	return map[string]any{
		"evaluations":        evalResults,
		"average_coherence":  avgCoherence,
		"average_separation": avgSeparation,
		"overall_quality":    overallQuality,
	}, nil
}

// computeCoherence computes average cosine similarity of articles to centroid.
func computeCoherence(articleIDs []string, centroid []float64, embeddings map[string][]float64) float64 {
	if len(articleIDs) <= 1 || len(centroid) == 0 {
		return 1.0
	}

	var totalSim float64
	var count int
	for _, id := range articleIDs {
		emb, ok := embeddings[id]
		if !ok {
			continue
		}
		totalSim += cosineSim(emb, centroid)
		count++
	}
	if count == 0 {
		return 0.0
	}
	return totalSim / float64(count)
}

// cosineSim computes cosine similarity between two vectors.
func cosineSim(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
