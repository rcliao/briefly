package tools

import (
	"briefly/internal/agent"
	"briefly/internal/core"
	"briefly/internal/narrative"
	"context"
	"fmt"

	"google.golang.org/genai"
)

// GenerateClusterNarrativeTool wraps the narrative generator for single cluster narratives.
type GenerateClusterNarrativeTool struct {
	generator *narrative.Generator
}

// NewGenerateClusterNarrativeTool creates a new cluster narrative tool.
func NewGenerateClusterNarrativeTool(generator *narrative.Generator) *GenerateClusterNarrativeTool {
	return &GenerateClusterNarrativeTool{generator: generator}
}

func (t *GenerateClusterNarrativeTool) Name() string { return "generate_cluster_narrative" }

func (t *GenerateClusterNarrativeTool) Description() string {
	return "Generate a narrative summary for a single topic cluster. Synthesizes ALL articles in the cluster into a coherent narrative with title, key developments, and statistics. Citations [N] in the output use the global article index numbers. Call once per cluster."
}

func (t *GenerateClusterNarrativeTool) Parameters() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"cluster_id": {
				Type:        genai.TypeString,
				Description: "ID of the cluster to narrate",
			},
		},
		Required: []string{"cluster_id"},
	}
}

func (t *GenerateClusterNarrativeTool) Execute(ctx context.Context, memory *agent.WorkingMemory, params map[string]any) (map[string]any, error) {
	clusterID := extractStringParam(params, "cluster_id", "")
	if clusterID == "" {
		return nil, fmt.Errorf("cluster_id is required")
	}

	clusters := memory.GetClusters()
	articles := memory.GetArticles()
	summaries := memory.GetSummaries()

	// Find the target cluster
	var targetClusterIdx int = -1
	for i, c := range clusters {
		if c.ID == clusterID {
			targetClusterIdx = i
			break
		}
	}
	if targetClusterIdx == -1 {
		return nil, fmt.Errorf("cluster %q not found", clusterID)
	}

	cluster := clusters[targetClusterIdx]

	// Build citation remap: the generator numbers articles [1],[2],[3] based on
	// their position in cluster.ArticleIDs. We need to remap to global citation numbers.
	citationMap := buildClusterCitationMap(cluster.ArticleIDs, memory.GetCitationNum)

	narr, err := t.generator.GenerateClusterSummary(ctx, cluster, articles, summaries)
	if err != nil {
		return nil, fmt.Errorf("narrative generation failed for cluster %s: %w", clusterID, err)
	}

	// Remap all citations from cluster-local to global
	narr.OneLiner = remapCitations(narr.OneLiner, citationMap)
	narr.Summary = remapCitations(narr.Summary, citationMap)
	narr.KeyDevelopments = remapCitationSlice(narr.KeyDevelopments, citationMap)
	narr.ArticleRefs = remapArticleRefs(narr.ArticleRefs, citationMap)
	// Remap key stats contexts
	for i := range narr.KeyStats {
		narr.KeyStats[i].Context = remapCitations(narr.KeyStats[i].Context, citationMap)
	}

	memory.SetNarrative(clusterID, *narr)

	// Also update the cluster in memory with the narrative attached
	updateClusterNarrative(memory, clusterID, narr)

	return map[string]any{
		"cluster_id":       clusterID,
		"title":            narr.Title,
		"one_liner":        narr.OneLiner,
		"key_developments": narr.KeyDevelopments,
		"key_stats":        narr.KeyStats,
		"key_themes":       narr.KeyThemes,
		"article_refs":     narr.ArticleRefs,
		"confidence":       narr.Confidence,
	}, nil
}

// updateClusterNarrative attaches the narrative to its cluster in memory.
func updateClusterNarrative(memory *agent.WorkingMemory, clusterID string, narr *core.ClusterNarrative) {
	clusters := memory.GetClusters()
	for i := range clusters {
		if clusters[i].ID == clusterID {
			clusters[i].Narrative = narr
			break
		}
	}
	memory.SetClusters(clusters)
}
