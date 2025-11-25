package clustering

import (
	"briefly/internal/core"
	"context"
	"fmt"
	"time"
)

// SearchQuery wraps search parameters for semantic clustering
type SearchQuery struct {
	Embedding           []float64
	Limit               int
	SimilarityThreshold float64
	ExcludeIDs          []string
}

// SearchResult wraps search results for semantic clustering
type SearchResult struct {
	ArticleID  string
	Similarity float64
}

// VectorSearcher defines minimal interface for semantic search
type VectorSearcher interface {
	SearchSimilar(ctx context.Context, embedding []float64, limit int, threshold float64, excludeIDs []string) ([]SearchResult, error)
}

// SemanticClusterer implements graph-based semantic clustering using pgvector
// Instead of K-means centroids, it builds a similarity graph and finds connected components
type SemanticClusterer struct {
	searcher            VectorSearcher // Searcher implementation
	similarityThreshold float64        // Minimum similarity to form an edge (default: 0.7)
	maxNeighbors        int            // Maximum neighbors to consider per article (default: 5)
	minClusterSize      int            // Minimum articles per cluster (default: 2)
	tagAware            bool           // Whether to cluster within tag boundaries
}

// NewSemanticClusterer creates a new semantic clusterer with default parameters
func NewSemanticClusterer(searcher VectorSearcher) *SemanticClusterer {
	return &SemanticClusterer{
		searcher:            searcher,
		similarityThreshold: 0.7, // 70% similarity threshold
		maxNeighbors:        5,   // Consider up to 5 nearest neighbors
		minClusterSize:      2,   // At least 2 articles per cluster
		tagAware:            false,
	}
}

// WithSimilarityThreshold sets the minimum similarity for clustering
func (s *SemanticClusterer) WithSimilarityThreshold(threshold float64) *SemanticClusterer {
	s.similarityThreshold = threshold
	return s
}

// WithMaxNeighbors sets the maximum neighbors to consider
func (s *SemanticClusterer) WithMaxNeighbors(max int) *SemanticClusterer {
	s.maxNeighbors = max
	return s
}

// WithMinClusterSize sets the minimum cluster size
func (s *SemanticClusterer) WithMinClusterSize(min int) *SemanticClusterer {
	s.minClusterSize = min
	return s
}

// WithTagAware enables tag-aware clustering
func (s *SemanticClusterer) WithTagAware(enabled bool) *SemanticClusterer {
	s.tagAware = enabled
	return s
}

// ClusterArticles performs graph-based semantic clustering
// Uses pgvector to build a similarity graph, then finds connected components
func (s *SemanticClusterer) ClusterArticles(ctx context.Context, articles []core.Article, embeddings map[string][]float64) ([]core.TopicCluster, error) {
	if len(articles) == 0 {
		return nil, fmt.Errorf("no articles to cluster")
	}

	// If tag-aware mode is enabled, cluster within theme boundaries
	if s.tagAware {
		return s.clusterByTheme(ctx, articles, embeddings)
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

	// Build similarity graph using pgvector
	similarityGraph, err := s.buildSimilarityGraph(ctx, articlesWithEmbeddings, embeddings)
	if err != nil {
		return nil, fmt.Errorf("failed to build similarity graph: %w", err)
	}

	// Find connected components (clusters)
	clusterAssignments := s.findConnectedComponents(similarityGraph, len(articlesWithEmbeddings))

	// Build TopicCluster objects
	clusters := s.buildClusters(articlesWithEmbeddings, clusterAssignments, embeddings)

	// Filter out small clusters and merge singletons
	clusters = s.filterAndMergeClusters(clusters, articlesWithEmbeddings)

	return clusters, nil
}

// buildSimilarityGraph creates a graph where edges represent semantic similarity
// Returns adjacency list: map[articleIndex][]neighborIndex
func (s *SemanticClusterer) buildSimilarityGraph(ctx context.Context, articles []core.Article, embeddings map[string][]float64) (map[int][]int, error) {
	graph := make(map[int][]int)
	articleIndexMap := make(map[string]int)

	// Build index map for quick lookup
	for i, article := range articles {
		articleIndexMap[article.ID] = i
		graph[i] = []int{} // Initialize empty neighbor list
	}

	// For each article, find similar neighbors using pgvector
	totalEdges := 0
	for i, article := range articles {
		embedding, ok := embeddings[article.ID]
		if !ok {
			continue
		}

		// Search for similar articles using pgvector HNSW index
		results, err := s.searcher.SearchSimilar(ctx, embedding, s.maxNeighbors, s.similarityThreshold, []string{article.ID})
		if err != nil {
			// Log warning but continue with other articles
			fmt.Printf("   ⚠️  Failed to search neighbors for article %s: %v\n", article.ID, err)
			continue
		}

		// Debug: log search results
		if len(results) > 0 {
			fmt.Printf("   🔍 Article %d found %d neighbors (threshold: %.2f)\n", i+1, len(results), s.similarityThreshold)
			for _, result := range results {
				fmt.Printf("      • Similarity: %.3f with article %s\n", result.Similarity, result.ArticleID[:8])
			}
		}

		// Add edges to similarity graph
		for _, result := range results {
			neighborIndex, exists := articleIndexMap[result.ArticleID]
			if exists {
				// Add bidirectional edge (undirected graph)
				graph[i] = append(graph[i], neighborIndex)
				graph[neighborIndex] = append(graph[neighborIndex], i)
				totalEdges++
			}
		}
	}

	fmt.Printf("   📊 Built similarity graph: %d nodes, %d edges\n", len(articles), totalEdges)

	return graph, nil
}

// findConnectedComponents uses DFS to find connected components in the similarity graph
// Returns cluster assignments: map[articleIndex]clusterID
func (s *SemanticClusterer) findConnectedComponents(graph map[int][]int, numArticles int) map[int]int {
	visited := make(map[int]bool)
	clusterAssignments := make(map[int]int)
	currentClusterID := 0

	// DFS helper function
	var dfs func(node int, clusterID int)
	dfs = func(node int, clusterID int) {
		visited[node] = true
		clusterAssignments[node] = clusterID

		for _, neighbor := range graph[node] {
			if !visited[neighbor] {
				dfs(neighbor, clusterID)
			}
		}
	}

	// Find all connected components
	for i := 0; i < numArticles; i++ {
		if !visited[i] {
			dfs(i, currentClusterID)
			currentClusterID++
		}
	}

	return clusterAssignments
}

// buildClusters creates TopicCluster objects from cluster assignments
func (s *SemanticClusterer) buildClusters(articles []core.Article, assignments map[int]int, embeddings map[string][]float64) []core.TopicCluster {
	clusterMap := make(map[int]*core.TopicCluster)

	// Group articles by cluster
	for i, article := range articles {
		clusterID := assignments[i]

		if _, exists := clusterMap[clusterID]; !exists {
			clusterMap[clusterID] = &core.TopicCluster{
				ID:         fmt.Sprintf("semantic_cluster_%d", clusterID),
				Label:      fmt.Sprintf("Cluster %d", clusterID+1),
				ArticleIDs: []string{},
				CreatedAt:  time.Now().UTC(),
			}
		}

		clusterMap[clusterID].ArticleIDs = append(clusterMap[clusterID].ArticleIDs, article.ID)
	}

	// Convert map to slice
	var clusters []core.TopicCluster
	for _, cluster := range clusterMap {
		// Calculate cluster centroid (average of all embeddings)
		cluster.Centroid = s.calculateCentroid(cluster.ArticleIDs, embeddings)
		clusters = append(clusters, *cluster)
	}

	return clusters
}

// calculateCentroid computes the average embedding for a cluster
func (s *SemanticClusterer) calculateCentroid(articleIDs []string, embeddings map[string][]float64) []float64 {
	if len(articleIDs) == 0 {
		return nil
	}

	// Get embedding dimension from first article
	firstEmbedding, ok := embeddings[articleIDs[0]]
	if !ok {
		return nil
	}

	embeddingDim := len(firstEmbedding)
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
func (s *SemanticClusterer) filterAndMergeClusters(clusters []core.TopicCluster, articles []core.Article) []core.TopicCluster {
	var validClusters []core.TopicCluster
	var singletons []core.TopicCluster

	// Separate valid clusters from singletons
	for _, cluster := range clusters {
		if len(cluster.ArticleIDs) >= s.minClusterSize {
			validClusters = append(validClusters, cluster)
		} else {
			singletons = append(singletons, cluster)
		}
	}

	// If we have singletons, group them into a "Miscellaneous" cluster
	if len(singletons) > 0 {
		miscCluster := core.TopicCluster{
			ID:         "semantic_cluster_misc",
			Label:      "Miscellaneous Topics",
			ArticleIDs: []string{},
			CreatedAt:  time.Now().UTC(),
		}

		for _, singleton := range singletons {
			miscCluster.ArticleIDs = append(miscCluster.ArticleIDs, singleton.ArticleIDs...)
		}

		validClusters = append(validClusters, miscCluster)
	}

	// If still no clusters, return all articles as one cluster
	if len(validClusters) == 0 {
		allArticles := core.TopicCluster{
			ID:         "semantic_cluster_all",
			Label:      "All Articles",
			ArticleIDs: []string{},
			CreatedAt:  time.Now().UTC(),
		}

		for _, article := range articles {
			allArticles.ArticleIDs = append(allArticles.ArticleIDs, article.ID)
		}

		validClusters = append(validClusters, allArticles)
	}

	return validClusters
}

// clusterByTheme performs tag-aware semantic clustering
// Groups articles by theme, clusters each theme separately, then combines results
func (s *SemanticClusterer) clusterByTheme(ctx context.Context, articles []core.Article, embeddings map[string][]float64) ([]core.TopicCluster, error) {
	fmt.Println("   🏷️  Tag-aware clustering enabled: clustering within theme boundaries")

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

	fmt.Printf("   📊 Grouped into %d themes (+ %d untagged articles)\n", len(themeGroups), len(untaggedArticles))

	// Cluster each theme group separately
	var allClusters []core.TopicCluster
	clusterIndex := 0

	for themeID, themeArticles := range themeGroups {
		if len(themeArticles) == 0 {
			continue
		}

		fmt.Printf("   🎯 Clustering theme '%s': %d articles\n", themeID, len(themeArticles))

		// Filter articles that have embeddings
		var articlesWithEmbeddings []core.Article
		for _, article := range themeArticles {
			if _, hasEmbedding := embeddings[article.ID]; hasEmbedding {
				articlesWithEmbeddings = append(articlesWithEmbeddings, article)
			}
		}

		if len(articlesWithEmbeddings) == 0 {
			fmt.Printf("      ⚠️  No embeddings found for theme '%s', skipping\n", themeID)
			continue
		}

		// Build similarity graph for this theme
		similarityGraph, err := s.buildSimilarityGraph(ctx, articlesWithEmbeddings, embeddings)
		if err != nil {
			fmt.Printf("      ⚠️  Failed to build similarity graph for theme '%s': %v\n", themeID, err)
			continue
		}

		// Find connected components
		clusterAssignments := s.findConnectedComponents(similarityGraph, len(articlesWithEmbeddings))

		// Build clusters with theme-aware naming
		themeClusters := s.buildClusters(articlesWithEmbeddings, clusterAssignments, embeddings)

		// Prefix cluster labels with theme name
		for i := range themeClusters {
			themeClusters[i].ID = fmt.Sprintf("theme_%s_cluster_%d", themeID, clusterIndex)
			themeClusters[i].Label = fmt.Sprintf("%s - %s", themeID, themeClusters[i].Label)
			clusterIndex++
		}

		allClusters = append(allClusters, themeClusters...)
		fmt.Printf("      ✓ Created %d clusters for theme '%s'\n", len(themeClusters), themeID)
	}

	// Handle untagged articles separately
	if len(untaggedArticles) > 0 {
		fmt.Printf("   📝 Clustering %d untagged articles\n", len(untaggedArticles))

		var articlesWithEmbeddings []core.Article
		for _, article := range untaggedArticles {
			if _, hasEmbedding := embeddings[article.ID]; hasEmbedding {
				articlesWithEmbeddings = append(articlesWithEmbeddings, article)
			}
		}

		if len(articlesWithEmbeddings) > 0 {
			similarityGraph, err := s.buildSimilarityGraph(ctx, articlesWithEmbeddings, embeddings)
			if err == nil {
				clusterAssignments := s.findConnectedComponents(similarityGraph, len(articlesWithEmbeddings))
				untaggedClusters := s.buildClusters(articlesWithEmbeddings, clusterAssignments, embeddings)

				for i := range untaggedClusters {
					untaggedClusters[i].ID = fmt.Sprintf("untagged_cluster_%d", clusterIndex)
					untaggedClusters[i].Label = fmt.Sprintf("Untagged - %s", untaggedClusters[i].Label)
					clusterIndex++
				}

				allClusters = append(allClusters, untaggedClusters...)
				fmt.Printf("      ✓ Created %d clusters for untagged articles\n", len(untaggedClusters))
			}
		}
	}

	// Apply filtering and merging
	finalClusters := s.filterAndMergeClusters(allClusters, articles)

	fmt.Printf("   ✓ Tag-aware clustering complete: %d final clusters\n", len(finalClusters))
	return finalClusters, nil
}
