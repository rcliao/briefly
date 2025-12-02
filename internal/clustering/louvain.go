package clustering

import (
	"briefly/internal/core"
	"context"
	"fmt"
	"log/slog"
	"time"

	"briefly/internal/logger"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/community"
	"gonum.org/v1/gonum/graph/simple"
)

// LouvainClusterer implements community detection using the Louvain algorithm.
// Key improvement over connected components: uses edge WEIGHTS (similarity scores)
// instead of binary connections, and optimizes modularity Q for better cluster quality.
type LouvainClusterer struct {
	searcher       VectorSearcher // Reuse existing pgvector interface
	resolution     float64        // Controls cluster granularity (1.0 = standard, higher = more clusters)
	minSimilarity  float64        // Minimum similarity for edge creation (lower threshold OK - weights matter)
	maxNeighbors   int            // k for k-NN graph building
	minClusterSize int            // Minimum articles per cluster
	tagAware       bool           // Whether to cluster within tag boundaries
	log            *slog.Logger
}

// NewLouvainClusterer creates a new Louvain clusterer with quality-focused defaults
func NewLouvainClusterer(searcher VectorSearcher) *LouvainClusterer {
	return &LouvainClusterer{
		searcher:       searcher,
		resolution:     1.0,  // Standard resolution (tune 0.5-2.0)
		minSimilarity:  0.3,  // Lower threshold OK - Louvain uses edge weights
		maxNeighbors:   10,   // More neighbors = better community detection
		minClusterSize: 2,    // Minimum 2 articles per cluster
		tagAware:       false,
		log:            logger.Get(),
	}
}

// WithResolution sets the resolution parameter for cluster granularity
// Higher values (>1.0) produce more, smaller clusters
// Lower values (<1.0) produce fewer, larger clusters
func (l *LouvainClusterer) WithResolution(resolution float64) *LouvainClusterer {
	l.resolution = resolution
	return l
}

// WithMinSimilarity sets the minimum similarity threshold for edge creation
func (l *LouvainClusterer) WithMinSimilarity(minSimilarity float64) *LouvainClusterer {
	l.minSimilarity = minSimilarity
	return l
}

// WithMaxNeighbors sets the k for k-NN graph building
func (l *LouvainClusterer) WithMaxNeighbors(maxNeighbors int) *LouvainClusterer {
	l.maxNeighbors = maxNeighbors
	return l
}

// WithMinClusterSize sets the minimum articles per cluster
func (l *LouvainClusterer) WithMinClusterSize(minClusterSize int) *LouvainClusterer {
	l.minClusterSize = minClusterSize
	return l
}

// WithTagAware enables tag-aware clustering
func (l *LouvainClusterer) WithTagAware(enabled bool) *LouvainClusterer {
	l.tagAware = enabled
	return l
}

// ClusterArticles performs Louvain community detection on articles
func (l *LouvainClusterer) ClusterArticles(ctx context.Context, articles []core.Article, embeddings map[string][]float64) ([]core.TopicCluster, error) {
	if len(articles) == 0 {
		return nil, fmt.Errorf("no articles to cluster")
	}

	// Filter articles that have embeddings
	var articlesWithEmbeddings []core.Article
	for _, article := range articles {
		if _, hasEmbedding := embeddings[article.ID]; hasEmbedding {
			articlesWithEmbeddings = append(articlesWithEmbeddings, article)
		}
	}

	if len(articlesWithEmbeddings) == 0 {
		return nil, fmt.Errorf("no articles have embeddings")
	}

	l.log.Info(fmt.Sprintf("   Louvain clustering %d articles (resolution=%.2f, minSim=%.2f, k=%d)",
		len(articlesWithEmbeddings), l.resolution, l.minSimilarity, l.maxNeighbors))

	// If tag-aware mode is enabled, cluster within theme boundaries
	if l.tagAware {
		return l.clusterByTheme(ctx, articlesWithEmbeddings, embeddings)
	}

	// Build weighted graph using pgvector HNSW index
	g, articleIDMap, nodeIDMap := l.buildWeightedGraph(ctx, articlesWithEmbeddings, embeddings)

	// Check if graph has any edges
	edgeCount := g.Edges().Len()
	if edgeCount == 0 {
		l.log.Warn("   No edges in similarity graph - articles may be too dissimilar")
		// Return all articles in one cluster
		return l.createSingleCluster(articlesWithEmbeddings, embeddings), nil
	}

	l.log.Info(fmt.Sprintf("   Built similarity graph: %d nodes, %d edges", len(articlesWithEmbeddings), edgeCount))

	// Run Louvain community detection (optimizes modularity Q)
	reducedGraph := community.Modularize(g, l.resolution, nil)
	communities := reducedGraph.Communities()

	// Calculate modularity score
	q := community.Q(g, communities, l.resolution)
	l.log.Info(fmt.Sprintf("   Louvain result: %d communities, modularity Q=%.3f", len(communities), q))

	// Convert to TopicClusters
	clusters := l.buildClusters(communities, articlesWithEmbeddings, articleIDMap, nodeIDMap, embeddings)

	// Filter and merge small clusters
	clusters = l.filterAndMergeClusters(clusters, articlesWithEmbeddings, embeddings)

	return clusters, nil
}

// buildWeightedGraph creates a similarity graph with edge weights from pgvector
func (l *LouvainClusterer) buildWeightedGraph(ctx context.Context, articles []core.Article, embeddings map[string][]float64) (*simple.WeightedUndirectedGraph, map[int64]string, map[string]int64) {
	g := simple.NewWeightedUndirectedGraph(0, 0)
	articleIDMap := make(map[int64]string) // nodeID -> articleID
	nodeIDMap := make(map[string]int64)    // articleID -> nodeID

	// Add nodes
	for i, article := range articles {
		nodeID := int64(i)
		articleIDMap[nodeID] = article.ID
		nodeIDMap[article.ID] = nodeID
		g.AddNode(simple.Node(nodeID))
	}

	// Build WEIGHTED edges using pgvector HNSW index
	totalEdges := 0
	for _, article := range articles {
		embedding, ok := embeddings[article.ID]
		if !ok {
			continue
		}
		fromNode := nodeIDMap[article.ID]

		// Use VectorSearcher to find k-nearest neighbors
		results, err := l.searcher.SearchSimilar(ctx, embedding, l.maxNeighbors, l.minSimilarity, []string{article.ID})
		if err != nil {
			l.log.Warn(fmt.Sprintf("   Failed to search neighbors for article %s: %v", article.ID[:8], err))
			continue
		}

		for _, result := range results {
			toNode, exists := nodeIDMap[result.ArticleID]
			if !exists {
				continue
			}

			// KEY DIFFERENCE from connected components:
			// Use similarity as edge WEIGHT - Louvain will prefer stronger connections
			if e := g.WeightedEdge(fromNode, toNode); e == nil {
				g.SetWeightedEdge(simple.WeightedEdge{
					F: simple.Node(fromNode),
					T: simple.Node(toNode),
					W: result.Similarity, // Edge weight = cosine similarity
				})
				totalEdges++
			}
		}
	}

	return g, articleIDMap, nodeIDMap
}

// buildClusters creates TopicCluster objects from Louvain communities
func (l *LouvainClusterer) buildClusters(communities [][]graph.Node, articles []core.Article, articleIDMap map[int64]string, nodeIDMap map[string]int64, embeddings map[string][]float64) []core.TopicCluster {
	var clusters []core.TopicCluster

	for i, comm := range communities {
		cluster := core.TopicCluster{
			ID:         fmt.Sprintf("louvain_cluster_%d", i),
			Label:      fmt.Sprintf("Cluster %d", i+1),
			ArticleIDs: make([]string, 0, len(comm)),
			CreatedAt:  time.Now().UTC(),
		}

		// Convert node IDs to article IDs
		for _, node := range comm {
			if articleID, ok := articleIDMap[node.ID()]; ok {
				cluster.ArticleIDs = append(cluster.ArticleIDs, articleID)
			}
		}

		// Calculate centroid
		cluster.Centroid = l.calculateCentroid(cluster.ArticleIDs, embeddings)

		clusters = append(clusters, cluster)
	}

	return clusters
}

// calculateCentroid computes the average embedding for a cluster
func (l *LouvainClusterer) calculateCentroid(articleIDs []string, embeddings map[string][]float64) []float64 {
	if len(articleIDs) == 0 {
		return nil
	}

	// Get embedding dimension from first article
	var embeddingDim int
	for _, id := range articleIDs {
		if emb, ok := embeddings[id]; ok {
			embeddingDim = len(emb)
			break
		}
	}

	if embeddingDim == 0 {
		return nil
	}

	centroid := make([]float64, embeddingDim)

	// Sum all embeddings
	count := 0
	for _, articleID := range articleIDs {
		embedding, ok := embeddings[articleID]
		if !ok {
			continue
		}

		for i, val := range embedding {
			centroid[i] += val
		}
		count++
	}

	// Average
	if count > 0 {
		for i := range centroid {
			centroid[i] /= float64(count)
		}
	}

	return centroid
}

// filterAndMergeClusters removes small clusters and handles singletons
func (l *LouvainClusterer) filterAndMergeClusters(clusters []core.TopicCluster, articles []core.Article, embeddings map[string][]float64) []core.TopicCluster {
	var validClusters []core.TopicCluster
	var singletonArticleIDs []string

	// Separate valid clusters from singletons
	for _, cluster := range clusters {
		if len(cluster.ArticleIDs) >= l.minClusterSize {
			validClusters = append(validClusters, cluster)
		} else {
			singletonArticleIDs = append(singletonArticleIDs, cluster.ArticleIDs...)
		}
	}

	// If we have singletons, group them into a "Miscellaneous" cluster
	if len(singletonArticleIDs) > 0 {
		miscCluster := core.TopicCluster{
			ID:         "louvain_cluster_misc",
			Label:      "Miscellaneous Topics",
			ArticleIDs: singletonArticleIDs,
			Centroid:   l.calculateCentroid(singletonArticleIDs, embeddings),
			CreatedAt:  time.Now().UTC(),
		}
		validClusters = append(validClusters, miscCluster)
	}

	// If still no clusters, return all articles as one cluster
	if len(validClusters) == 0 {
		return l.createSingleCluster(articles, embeddings)
	}

	return validClusters
}

// createSingleCluster creates a single cluster containing all articles
func (l *LouvainClusterer) createSingleCluster(articles []core.Article, embeddings map[string][]float64) []core.TopicCluster {
	articleIDs := make([]string, len(articles))
	for i, article := range articles {
		articleIDs[i] = article.ID
	}

	return []core.TopicCluster{
		{
			ID:         "louvain_cluster_all",
			Label:      "All Articles",
			ArticleIDs: articleIDs,
			Centroid:   l.calculateCentroid(articleIDs, embeddings),
			CreatedAt:  time.Now().UTC(),
		},
	}
}

// clusterByTheme performs tag-aware Louvain clustering
// Groups articles by theme, clusters each theme separately, then combines results
func (l *LouvainClusterer) clusterByTheme(ctx context.Context, articles []core.Article, embeddings map[string][]float64) ([]core.TopicCluster, error) {
	l.log.Info("   Tag-aware Louvain clustering enabled")

	// Group articles by ThemeID
	themeGroups := make(map[string][]core.Article)
	var untaggedArticles []core.Article

	for _, article := range articles {
		if article.ThemeID != nil && *article.ThemeID != "" {
			themeGroups[*article.ThemeID] = append(themeGroups[*article.ThemeID], article)
		} else {
			untaggedArticles = append(untaggedArticles, article)
		}
	}

	l.log.Info(fmt.Sprintf("   Grouped into %d themes (+ %d untagged articles)", len(themeGroups), len(untaggedArticles)))

	// Cluster each theme group separately
	var allClusters []core.TopicCluster
	clusterIndex := 0

	for themeID, themeArticles := range themeGroups {
		if len(themeArticles) == 0 {
			continue
		}

		// Filter articles that have embeddings
		var articlesWithEmbeddings []core.Article
		for _, article := range themeArticles {
			if _, hasEmbedding := embeddings[article.ID]; hasEmbedding {
				articlesWithEmbeddings = append(articlesWithEmbeddings, article)
			}
		}

		if len(articlesWithEmbeddings) == 0 {
			continue
		}

		l.log.Info(fmt.Sprintf("   Clustering theme '%s': %d articles", themeID, len(articlesWithEmbeddings)))

		// Build similarity graph for this theme
		g, articleIDMap, nodeIDMap := l.buildWeightedGraph(ctx, articlesWithEmbeddings, embeddings)

		var themeClusters []core.TopicCluster
		if g.Edges().Len() == 0 {
			// No edges - put all theme articles in one cluster
			themeClusters = l.createSingleCluster(articlesWithEmbeddings, embeddings)
		} else {
			// Run Louvain
			reducedGraph := community.Modularize(g, l.resolution, nil)
			communities := reducedGraph.Communities()
			themeClusters = l.buildClusters(communities, articlesWithEmbeddings, articleIDMap, nodeIDMap, embeddings)
		}

		// Prefix cluster labels with theme name
		for i := range themeClusters {
			themeClusters[i].ID = fmt.Sprintf("theme_%s_louvain_%d", themeID, clusterIndex)
			themeClusters[i].Label = fmt.Sprintf("%s - %s", themeID, themeClusters[i].Label)
			clusterIndex++
		}

		allClusters = append(allClusters, themeClusters...)
	}

	// Handle untagged articles separately
	if len(untaggedArticles) > 0 {
		l.log.Info(fmt.Sprintf("   Clustering %d untagged articles", len(untaggedArticles)))

		var articlesWithEmbeddings []core.Article
		for _, article := range untaggedArticles {
			if _, hasEmbedding := embeddings[article.ID]; hasEmbedding {
				articlesWithEmbeddings = append(articlesWithEmbeddings, article)
			}
		}

		if len(articlesWithEmbeddings) > 0 {
			g, articleIDMap, nodeIDMap := l.buildWeightedGraph(ctx, articlesWithEmbeddings, embeddings)

			var untaggedClusters []core.TopicCluster
			if g.Edges().Len() == 0 {
				untaggedClusters = l.createSingleCluster(articlesWithEmbeddings, embeddings)
			} else {
				reducedGraph := community.Modularize(g, l.resolution, nil)
				communities := reducedGraph.Communities()
				untaggedClusters = l.buildClusters(communities, articlesWithEmbeddings, articleIDMap, nodeIDMap, embeddings)
			}

			for i := range untaggedClusters {
				untaggedClusters[i].ID = fmt.Sprintf("untagged_louvain_%d", clusterIndex)
				untaggedClusters[i].Label = fmt.Sprintf("Untagged - %s", untaggedClusters[i].Label)
				clusterIndex++
			}

			allClusters = append(allClusters, untaggedClusters...)
		}
	}

	// Apply filtering and merging
	finalClusters := l.filterAndMergeClusters(allClusters, articles, embeddings)

	l.log.Info(fmt.Sprintf("   Tag-aware Louvain complete: %d final clusters", len(finalClusters)))
	return finalClusters, nil
}
