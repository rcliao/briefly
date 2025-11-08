package quality

import (
	"fmt"
	"math"

	"briefly/internal/core"
)

// ClusterCoherenceEvaluator evaluates the quality of topic clustering
type ClusterCoherenceEvaluator struct {
	thresholds QualityThresholds
}

// NewClusterCoherenceEvaluator creates a new cluster coherence evaluator
func NewClusterCoherenceEvaluator() *ClusterCoherenceEvaluator {
	return &ClusterCoherenceEvaluator{
		thresholds: DefaultThresholds(),
	}
}

// NewClusterCoherenceEvaluatorWithThresholds creates an evaluator with custom thresholds
func NewClusterCoherenceEvaluatorWithThresholds(thresholds QualityThresholds) *ClusterCoherenceEvaluator {
	return &ClusterCoherenceEvaluator{
		thresholds: thresholds,
	}
}

// EvaluateClusterCoherence performs comprehensive evaluation of clustering quality
func (e *ClusterCoherenceEvaluator) EvaluateClusterCoherence(
	clusters []core.TopicCluster,
	embeddings map[string][]float64,
) *ClusterCoherenceMetrics {
	metrics := &ClusterCoherenceMetrics{
		NumClusters:              len(clusters),
		NumArticles:              0,
		ClusterSilhouettes:       make([]float64, 0, len(clusters)),
		IntraClusterSimilarities: make([]float64, 0, len(clusters)),
		Issues:                   []string{},
	}

	// Count total articles
	for _, cluster := range clusters {
		metrics.NumArticles += len(cluster.ArticleIDs)
	}

	if metrics.NumArticles > 0 {
		metrics.AvgClusterSize = float64(metrics.NumArticles) / float64(metrics.NumClusters)
	}

	// Build mapping of article ID to cluster index
	articleToCluster := make(map[string]int)
	for clusterIdx, cluster := range clusters {
		for _, articleID := range cluster.ArticleIDs {
			articleToCluster[articleID] = clusterIdx
		}
	}

	// Calculate per-cluster metrics
	for clusterIdx, cluster := range clusters {
		// Skip empty clusters
		if len(cluster.ArticleIDs) == 0 {
			metrics.Issues = append(metrics.Issues,
				fmt.Sprintf("Cluster %d (%s) is empty", clusterIdx, cluster.Label))
			continue
		}

		// Calculate intra-cluster similarity (cohesion)
		intraClusterSim := e.calculateIntraClusterSimilarity(cluster, embeddings)
		metrics.IntraClusterSimilarities = append(metrics.IntraClusterSimilarities, intraClusterSim)
		metrics.AvgIntraClusterSimilarity += intraClusterSim

		// Check for poor cohesion
		if intraClusterSim < e.thresholds.MinIntraClusterSim {
			metrics.Issues = append(metrics.Issues,
				fmt.Sprintf("Cluster %d (%s) has low cohesion: %.2f (min: %.2f)",
					clusterIdx, cluster.Label, intraClusterSim, e.thresholds.MinIntraClusterSim))
		}

		// Calculate silhouette score for this cluster
		silhouette := e.calculateClusterSilhouette(cluster, clusters, embeddings, articleToCluster)
		metrics.ClusterSilhouettes = append(metrics.ClusterSilhouettes, silhouette)
		metrics.AvgSilhouette += silhouette

		// Check for poor silhouette
		if silhouette < e.thresholds.MinSilhouetteScore {
			metrics.Issues = append(metrics.Issues,
				fmt.Sprintf("Cluster %d (%s) has low silhouette score: %.2f (min: %.2f)",
					clusterIdx, cluster.Label, silhouette, e.thresholds.MinSilhouetteScore))
		}
	}

	// Average the per-cluster metrics
	if len(clusters) > 0 {
		metrics.AvgIntraClusterSimilarity /= float64(len(clusters))
		metrics.AvgSilhouette /= float64(len(clusters))
	}

	// Calculate inter-cluster distance (separation)
	metrics.AvgInterClusterDistance = e.calculateInterClusterDistance(clusters)

	// Check for poor separation
	if metrics.AvgInterClusterDistance < e.thresholds.MinInterClusterDist {
		metrics.Issues = append(metrics.Issues,
			fmt.Sprintf("Low cluster separation: %.2f (min: %.2f)",
				metrics.AvgInterClusterDistance, e.thresholds.MinInterClusterDist))
	}

	// Assign grade
	metrics.CoherenceGrade = GradeClusterCoherence(metrics, e.thresholds)

	// Overall pass/fail
	metrics.Passed = metrics.AvgSilhouette >= e.thresholds.MinSilhouetteScore &&
		metrics.AvgIntraClusterSimilarity >= e.thresholds.MinIntraClusterSim &&
		len(metrics.Issues) == 0

	return metrics
}

// calculateIntraClusterSimilarity calculates average cosine similarity within a cluster (cohesion)
func (e *ClusterCoherenceEvaluator) calculateIntraClusterSimilarity(
	cluster core.TopicCluster,
	embeddings map[string][]float64,
) float64 {
	if len(cluster.ArticleIDs) <= 1 {
		return 1.0 // Perfect cohesion for single-article clusters
	}

	// Collect embeddings for all articles in cluster
	clusterEmbeddings := make([][]float64, 0, len(cluster.ArticleIDs))
	for _, articleID := range cluster.ArticleIDs {
		if emb, ok := embeddings[articleID]; ok {
			clusterEmbeddings = append(clusterEmbeddings, emb)
		}
	}

	if len(clusterEmbeddings) <= 1 {
		return 1.0 // No valid embeddings or single article
	}

	// Calculate average pairwise similarity
	totalSimilarity := 0.0
	count := 0
	for i := 0; i < len(clusterEmbeddings); i++ {
		for j := i + 1; j < len(clusterEmbeddings); j++ {
			similarity := cosineSimilarity(clusterEmbeddings[i], clusterEmbeddings[j])
			totalSimilarity += similarity
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	return totalSimilarity / float64(count)
}

// calculateClusterSilhouette calculates silhouette score for a cluster
// Silhouette score ranges from -1 (wrong cluster) to +1 (perfect cluster)
func (e *ClusterCoherenceEvaluator) calculateClusterSilhouette(
	cluster core.TopicCluster,
	allClusters []core.TopicCluster,
	embeddings map[string][]float64,
	articleToCluster map[string]int,
) float64 {
	if len(cluster.ArticleIDs) == 0 {
		return 0.0
	}

	totalSilhouette := 0.0
	validArticles := 0

	for _, articleID := range cluster.ArticleIDs {
		emb, ok := embeddings[articleID]
		if !ok {
			continue
		}

		// Calculate a(i): average distance to other points in same cluster
		a := e.calculateAverageDistanceToCluster(emb, cluster, embeddings, articleID)

		// Calculate b(i): min average distance to points in other clusters
		b := e.calculateMinDistanceToOtherClusters(emb, allClusters, embeddings, articleToCluster[articleID])

		// Silhouette score for this article
		silhouette := 0.0
		if a < b {
			silhouette = 1.0 - (a / b)
		} else if a > b {
			silhouette = (b / a) - 1.0
		}
		// If a == b, silhouette = 0

		totalSilhouette += silhouette
		validArticles++
	}

	if validArticles == 0 {
		return 0.0
	}

	return totalSilhouette / float64(validArticles)
}

// calculateAverageDistanceToCluster calculates average distance to all points in cluster (excluding self)
func (e *ClusterCoherenceEvaluator) calculateAverageDistanceToCluster(
	embedding []float64,
	cluster core.TopicCluster,
	embeddings map[string][]float64,
	excludeArticleID string,
) float64 {
	totalDistance := 0.0
	count := 0

	for _, articleID := range cluster.ArticleIDs {
		if articleID == excludeArticleID {
			continue // Skip self
		}

		if otherEmb, ok := embeddings[articleID]; ok {
			distance := cosineDistance(embedding, otherEmb)
			totalDistance += distance
			count++
		}
	}

	if count == 0 {
		return 0.0 // Single article in cluster
	}

	return totalDistance / float64(count)
}

// calculateMinDistanceToOtherClusters finds minimum average distance to other clusters
func (e *ClusterCoherenceEvaluator) calculateMinDistanceToOtherClusters(
	embedding []float64,
	allClusters []core.TopicCluster,
	embeddings map[string][]float64,
	currentClusterIdx int,
) float64 {
	minDistance := math.MaxFloat64

	for clusterIdx, cluster := range allClusters {
		if clusterIdx == currentClusterIdx {
			continue // Skip own cluster
		}

		// Calculate average distance to this cluster
		totalDistance := 0.0
		count := 0

		for _, articleID := range cluster.ArticleIDs {
			if otherEmb, ok := embeddings[articleID]; ok {
				distance := cosineDistance(embedding, otherEmb)
				totalDistance += distance
				count++
			}
		}

		if count > 0 {
			avgDistance := totalDistance / float64(count)
			if avgDistance < minDistance {
				minDistance = avgDistance
			}
		}
	}

	if minDistance == math.MaxFloat64 {
		return 1.0 // No other clusters
	}

	return minDistance
}

// calculateInterClusterDistance calculates average distance between cluster centroids (separation)
func (e *ClusterCoherenceEvaluator) calculateInterClusterDistance(clusters []core.TopicCluster) float64 {
	if len(clusters) <= 1 {
		return 1.0 // Perfect separation for single cluster
	}

	totalDistance := 0.0
	count := 0

	for i := 0; i < len(clusters); i++ {
		for j := i + 1; j < len(clusters); j++ {
			if len(clusters[i].Centroid) > 0 && len(clusters[j].Centroid) > 0 {
				distance := cosineDistance(clusters[i].Centroid, clusters[j].Centroid)
				totalDistance += distance
				count++
			}
		}
	}

	if count == 0 {
		return 0.0
	}

	return totalDistance / float64(count)
}

// PrintCoherenceReport prints a formatted coherence report
func (e *ClusterCoherenceEvaluator) PrintCoherenceReport(metrics *ClusterCoherenceMetrics) {
	fmt.Println("============================================================")
	fmt.Println("CLUSTER COHERENCE QUALITY REPORT")
	fmt.Println("============================================================")
	fmt.Printf("Grade: %s\n", metrics.CoherenceGrade)
	fmt.Printf("Clusters: %d\n", metrics.NumClusters)
	fmt.Printf("Articles: %d (avg %.1f per cluster)\n", metrics.NumArticles, metrics.AvgClusterSize)
	fmt.Printf("Avg Silhouette Score: %.3f\n", metrics.AvgSilhouette)
	fmt.Printf("Avg Intra-Cluster Similarity: %.3f\n", metrics.AvgIntraClusterSimilarity)
	fmt.Printf("Avg Inter-Cluster Distance: %.3f\n", metrics.AvgInterClusterDistance)

	if len(metrics.ClusterSilhouettes) > 0 {
		fmt.Println("\nPer-Cluster Silhouette Scores:")
		for i, score := range metrics.ClusterSilhouettes {
			fmt.Printf("  Cluster %d: %.3f", i+1, score)
			if score < e.thresholds.MinSilhouetteScore {
				fmt.Printf(" ⚠️  (below threshold)")
			}
			fmt.Println()
		}
	}

	if len(metrics.IntraClusterSimilarities) > 0 {
		fmt.Println("\nPer-Cluster Cohesion:")
		for i, sim := range metrics.IntraClusterSimilarities {
			fmt.Printf("  Cluster %d: %.3f", i+1, sim)
			if sim < e.thresholds.MinIntraClusterSim {
				fmt.Printf(" ⚠️  (below threshold)")
			}
			fmt.Println()
		}
	}

	if len(metrics.Issues) > 0 {
		fmt.Println("\n⚠️  ISSUES:")
		for _, issue := range metrics.Issues {
			fmt.Printf("  - %s\n", issue)
		}
	} else {
		fmt.Println("\n✅ No clustering issues detected")
	}

	fmt.Println("============================================================")
	fmt.Println()
}

// cosineSimilarity calculates cosine similarity between two vectors (range: -1 to 1)
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	dotProduct := 0.0
	magA := 0.0
	magB := 0.0

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}

	magA = math.Sqrt(magA)
	magB = math.Sqrt(magB)

	if magA == 0.0 || magB == 0.0 {
		return 0.0
	}

	return dotProduct / (magA * magB)
}

// cosineDistance calculates cosine distance between two vectors (range: 0 to 2)
// Distance = 1 - similarity
func cosineDistance(a, b []float64) float64 {
	return 1.0 - cosineSimilarity(a, b)
}
