package tools

import (
	"briefly/internal/agent"
	"briefly/internal/clustering"
	"briefly/internal/core"
	"context"
	"fmt"

	"google.golang.org/genai"
)

// ClusterArticlesTool wraps the existing K-means clusterer.
type ClusterArticlesTool struct{}

// NewClusterArticlesTool creates a new cluster tool.
func NewClusterArticlesTool() *ClusterArticlesTool {
	return &ClusterArticlesTool{}
}

func (t *ClusterArticlesTool) Name() string { return "cluster_articles" }

func (t *ClusterArticlesTool) Description() string {
	return "Group articles by topic similarity using K-means clustering on embeddings. Requires embeddings to be generated first. Produces topic clusters with labels."
}

func (t *ClusterArticlesTool) Parameters() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"num_clusters": {
				Type:        genai.TypeInteger,
				Description: "Number of clusters. 0 = auto-detect based on corpus size.",
			},
		},
	}
}

func (t *ClusterArticlesTool) Execute(ctx context.Context, memory *agent.WorkingMemory, params map[string]any) (map[string]any, error) {
	articles := memory.GetArticles()
	embeddings := memory.GetEmbeddings()
	numClusters := extractIntParam(params, "num_clusters", 0)

	if len(embeddings) < 2 {
		return nil, fmt.Errorf("need at least 2 articles with embeddings to cluster, have %d", len(embeddings))
	}

	// Auto-detect cluster count
	if numClusters == 0 {
		numClusters = (len(embeddings) + 4) / 5
		if numClusters < 2 {
			numClusters = 2
		}
		if numClusters > 8 {
			numClusters = 8
		}
	}

	// Build article list with embeddings set on the article objects
	// The KMeansClusterer.Cluster() reads from article.Embedding
	var articleList []core.Article
	for id, emb := range embeddings {
		if a, ok := articles[id]; ok {
			a.Embedding = emb
			articleList = append(articleList, a)
		}
	}

	clusterer := clustering.NewKMeansClusterer()
	clusters, err := clusterer.Cluster(articleList, numClusters)
	if err != nil {
		return nil, fmt.Errorf("clustering failed: %w", err)
	}

	memory.SetClusters(clusters)

	// Build response
	clusterList := make([]map[string]any, 0, len(clusters))
	for _, c := range clusters {
		titles := make([]string, 0)
		for _, aid := range c.ArticleIDs {
			if a, ok := articles[aid]; ok {
				titles = append(titles, a.Title)
			}
		}
		clusterList = append(clusterList, map[string]any{
			"cluster_id":     c.ID,
			"label":          c.Label,
			"article_count":  len(c.ArticleIDs),
			"article_ids":    c.ArticleIDs,
			"article_titles": titles,
		})
	}

	return map[string]any{
		"clusters":             clusterList,
		"total_clusters":       len(clusters),
		"unclustered_articles": len(articles) - len(embeddings),
	}, nil
}
