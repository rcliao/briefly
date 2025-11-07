package clustering

import (
	"briefly/internal/core"
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/humilityai/hdbscan"
)

// HDBSCANConfig holds configuration for HDBSCAN clustering
type HDBSCANConfig struct {
	MinClusterSize int // Minimum number of articles to form a cluster
	MinSamples     int // Minimum samples in neighborhood for core point
}

// DefaultHDBSCANConfig returns sensible defaults for HDBSCAN clustering
func DefaultHDBSCANConfig() HDBSCANConfig {
	return HDBSCANConfig{
		MinClusterSize: 3, // Minimum 3 articles per topic (allows smaller, more focused clusters)
		MinSamples:     1, // Low threshold - let HDBSCAN find natural groupings
	}
}

// HDBSCANClusterer implements HDBSCAN clustering for articles based on embeddings
type HDBSCANClusterer struct {
	MinClusterSize int // Minimum number of articles to form a cluster
	MinSamples     int // Minimum samples in neighborhood for core point
}

// NewHDBSCANClusterer creates a new HDBSCAN clusterer with default parameters
func NewHDBSCANClusterer() *HDBSCANClusterer {
	config := DefaultHDBSCANConfig()
	return &HDBSCANClusterer{
		MinClusterSize: config.MinClusterSize,
		MinSamples:     config.MinSamples,
	}
}

// cosineDistance computes cosine distance between two vectors
// For high-dimensional embeddings (768 dims), cosine distance works much better than Euclidean
// Cosine distance = 1 - cosine similarity
// Cosine similarity = dot(A, B) / (||A|| * ||B||)
func cosineDistance(x1, x2 []float64) float64 {
	if len(x1) != len(x2) {
		return 1.0 // Maximum distance for mismatched dimensions
	}

	// Calculate dot product and magnitudes
	var dotProduct, mag1, mag2 float64
	for i := range x1 {
		dotProduct += x1[i] * x2[i]
		mag1 += x1[i] * x1[i]
		mag2 += x2[i] * x2[i]
	}

	// Handle zero vectors
	if mag1 == 0 || mag2 == 0 {
		return 1.0
	}

	// Cosine similarity = dot / (||A|| * ||B||)
	similarity := dotProduct / (math.Sqrt(mag1) * math.Sqrt(mag2))

	// Clamp to [-1, 1] to handle floating point errors
	if similarity > 1.0 {
		similarity = 1.0
	} else if similarity < -1.0 {
		similarity = -1.0
	}

	// Convert similarity to distance: distance = 1 - similarity
	// Range: [0, 2] where 0 = identical, 1 = orthogonal, 2 = opposite
	return 1.0 - similarity
}

// Cluster performs HDBSCAN clustering on articles using their embeddings
// Note: The 'k' parameter is IGNORED for HDBSCAN (included for interface compatibility)
// HDBSCAN automatically discovers the optimal number of clusters
func (h *HDBSCANClusterer) Cluster(articles []core.Article, k int) ([]core.TopicCluster, error) {
	if len(articles) == 0 {
		return nil, fmt.Errorf("no articles to cluster")
	}

	// Filter articles that have embeddings
	var articlesWithEmbeddings []core.Article
	for _, article := range articles {
		if len(article.Embedding) > 0 {
			articlesWithEmbeddings = append(articlesWithEmbeddings, article)
		}
	}

	if len(articlesWithEmbeddings) == 0 {
		return nil, fmt.Errorf("no articles have embeddings")
	}

	// If we have fewer than MinClusterSize articles, just put them all in one cluster
	if len(articlesWithEmbeddings) < h.MinClusterSize {
		return []core.TopicCluster{
			{
				ID:         "cluster_0",
				Label:      "All Articles",
				ArticleIDs: getArticleIDs(articlesWithEmbeddings),
				Centroid:   calculateCentroid(articlesWithEmbeddings),
				CreatedAt:  time.Now().UTC(),
			},
		}, nil
	}

	// Convert article embeddings to format expected by HDBSCAN library
	// Format: [][]float64 where each inner slice is one article's embedding
	dataPoints := make([][]float64, len(articlesWithEmbeddings))
	for i, article := range articlesWithEmbeddings {
		dataPoints[i] = article.Embedding
	}

	// Create HDBSCAN clusterer
	clustering, err := hdbscan.NewClustering(dataPoints, h.MinClusterSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create HDBSCAN clusterer: %w", err)
	}

	// Configure: mark outliers, log progress
	clustering = clustering.OutlierDetection().Verbose()

	// Run HDBSCAN clustering with COSINE DISTANCE (critical for high-dimensional embeddings!)
	// Euclidean distance fails on 768-dim embeddings due to curse of dimensionality
	err = clustering.Run(cosineDistance, hdbscan.VarianceScore, true)
	if err != nil {
		return nil, fmt.Errorf("HDBSCAN clustering failed: %w", err)
	}

	// Convert HDBSCAN clusters to TopicCluster format
	clusters, noiseCount := h.clusteringToTopicClusters(articlesWithEmbeddings, clustering)

	// Log clustering results
	fmt.Printf("\n🔍 HDBSCAN Clustering Results:\n")
	fmt.Printf("   • Total articles: %d\n", len(articlesWithEmbeddings))
	fmt.Printf("   • Clusters found: %d\n", len(clusters))
	fmt.Printf("   • Noise articles: %d\n", noiseCount)
	for i, cluster := range clusters {
		fmt.Printf("   • Cluster %d: %d articles (%s)\n", i, len(cluster.ArticleIDs), cluster.Label)
	}
	fmt.Println()

	return clusters, nil
}

// clusteringToTopicClusters converts HDBSCAN Clustering result to TopicCluster structs
// Returns: (clusters, noiseCount)
func (h *HDBSCANClusterer) clusteringToTopicClusters(articles []core.Article, clustering *hdbscan.Clustering) ([]core.TopicCluster, int) {
	// Use reflection to access the Clusters field
	// Structure: clustering.Clusters is a slice of *cluster
	// Each cluster has: Centroid []float64, Points []int, Outliers []Outlier
	clusterData := extractClusterData(clustering)

	// Build map of point index -> cluster ID for assigned points
	pointToCluster := make(map[int]int)
	for clusterID, cluster := range clusterData {
		for _, pointIdx := range cluster.Points {
			pointToCluster[pointIdx] = clusterID
		}
	}

	// Count outliers (noise points)
	noiseCount := len(articles) - len(pointToCluster)

	// Build map of cluster ID -> article indices
	clusterMap := make(map[int][]int)
	for i := 0; i < len(articles); i++ {
		if clusterID, found := pointToCluster[i]; found {
			clusterMap[clusterID] = append(clusterMap[clusterID], i)
		}
		// Points not in pointToCluster are noise/outliers - skip them
	}

	// Build TopicCluster structs
	clusters := make([]core.TopicCluster, 0, len(clusterMap))

	for clusterID, articleIndices := range clusterMap {
		// Get articles in this cluster
		var clusterArticles []core.Article
		var articleIDs []string

		for _, idx := range articleIndices {
			clusterArticles = append(clusterArticles, articles[idx])
			articleIDs = append(articleIDs, articles[idx].ID)
		}

		// Use HDBSCAN's centroid if available, otherwise calculate
		var centroid []float64
		if clusterID < len(clusterData) && len(clusterData[clusterID].Centroid) > 0 {
			centroid = clusterData[clusterID].Centroid
		} else {
			centroid = calculateCentroid(clusterArticles)
		}

		// Generate label from article titles
		label := h.generateTopicLabel(clusterArticles)

		// Extract keywords
		keywords := extractKeywordsFromArticles(clusterArticles)

		cluster := core.TopicCluster{
			ID:         fmt.Sprintf("hdbscan_cluster_%d", clusterID),
			Label:      label,
			ArticleIDs: articleIDs,
			Centroid:   centroid,
			Keywords:   keywords,
			CreatedAt:  time.Now().UTC(),
		}

		clusters = append(clusters, cluster)
	}

	return clusters, noiseCount
}

// generateTopicLabel creates a human-readable label for a topic cluster
func (h *HDBSCANClusterer) generateTopicLabel(articles []core.Article) string {
	if len(articles) == 0 {
		return "Empty Cluster"
	}

	// Simple approach: find common words in titles
	wordCounts := make(map[string]int)
	for _, article := range articles {
		words := extractWords(article.Title)
		for _, word := range words {
			if len(word) > 3 { // Filter out short words
				wordCounts[word]++
			}
		}
	}

	// Find most common word
	var mostCommonWord string
	maxCount := 0
	for word, count := range wordCounts {
		if count > maxCount {
			maxCount = count
			mostCommonWord = word
		}
	}

	if mostCommonWord != "" && maxCount > 1 {
		return fmt.Sprintf("%s & Related", mostCommonWord)
	}

	// Fallback: use first article title (truncated)
	firstTitle := articles[0].Title
	if len(firstTitle) > 40 {
		firstTitle = firstTitle[:37] + "..."
	}
	return firstTitle
}

// calculateCentroid computes the average embedding for a set of articles
func calculateCentroid(articles []core.Article) []float64 {
	if len(articles) == 0 {
		return nil
	}

	embeddingDim := len(articles[0].Embedding)
	centroid := make([]float64, embeddingDim)

	// Sum all embeddings
	for _, article := range articles {
		for i, val := range article.Embedding {
			centroid[i] += val
		}
	}

	// Average
	for i := range centroid {
		centroid[i] /= float64(len(articles))
	}

	return centroid
}

// extractKeywordsFromArticles extracts key terms from a set of articles
func extractKeywordsFromArticles(articles []core.Article) []string {
	wordCounts := make(map[string]int)

	for _, article := range articles {
		// Extract words from title and first part of content
		words := extractWords(article.Title)
		if len(article.CleanedText) > 200 {
			words = append(words, extractWords(article.CleanedText[:200])...)
		} else {
			words = append(words, extractWords(article.CleanedText)...)
		}

		for _, word := range words {
			if len(word) > 3 {
				wordCounts[word]++
			}
		}
	}

	// Sort words by frequency
	type wordFreq struct {
		word  string
		count int
	}
	var sortedWords []wordFreq
	for word, count := range wordCounts {
		sortedWords = append(sortedWords, wordFreq{word, count})
	}

	// Sort by count descending
	for i := 0; i < len(sortedWords)-1; i++ {
		for j := i + 1; j < len(sortedWords); j++ {
			if sortedWords[j].count > sortedWords[i].count {
				sortedWords[i], sortedWords[j] = sortedWords[j], sortedWords[i]
			}
		}
	}

	// Return top 5 keywords
	var keywords []string
	for i, wf := range sortedWords {
		if i >= 5 {
			break
		}
		keywords = append(keywords, wf.word)
	}

	return keywords
}

// getArticleIDs extracts article IDs from a list of articles
func getArticleIDs(articles []core.Article) []string {
	ids := make([]string, len(articles))
	for i, article := range articles {
		ids[i] = article.ID
	}
	return ids
}

// ClusterData holds extracted cluster information from HDBSCAN
type ClusterData struct {
	Centroid []float64
	Points   []int
}

// ExtractClusterDataPublic is a public wrapper for extractClusterData (for testing)
func ExtractClusterDataPublic(clustering *hdbscan.Clustering) []ClusterData {
	return extractClusterData(clustering)
}

// extractClusterData uses reflection to extract cluster assignments from HDBSCAN Clustering
// Returns a slice of ClusterData, one for each cluster
func extractClusterData(clustering *hdbscan.Clustering) []ClusterData {
	// Use reflection to access the Clusters field
	v := reflect.ValueOf(clustering).Elem()
	clustersField := v.FieldByName("Clusters")

	if !clustersField.IsValid() {
		fmt.Println("Warning: Could not access Clusters field")
		return []ClusterData{}
	}

	// Clusters is a slice of *cluster
	numClusters := clustersField.Len()
	result := make([]ClusterData, numClusters)

	for i := 0; i < numClusters; i++ {
		clusterPtr := clustersField.Index(i)

		// Dereference pointer to get cluster struct
		if clusterPtr.Kind() == reflect.Ptr {
			clusterPtr = clusterPtr.Elem()
		}

		// Extract Centroid field ([]float64)
		centroidField := clusterPtr.FieldByName("Centroid")
		if centroidField.IsValid() && centroidField.Kind() == reflect.Slice {
			centroid := make([]float64, centroidField.Len())
			for j := 0; j < centroidField.Len(); j++ {
				centroid[j] = centroidField.Index(j).Float()
			}
			result[i].Centroid = centroid
		}

		// Extract Points field ([]int)
		pointsField := clusterPtr.FieldByName("Points")
		if pointsField.IsValid() && pointsField.Kind() == reflect.Slice {
			points := make([]int, pointsField.Len())
			for j := 0; j < pointsField.Len(); j++ {
				points[j] = int(pointsField.Index(j).Int())
			}
			result[i].Points = points
		}
	}

	return result
}
