package clustering

import (
	"math"
)

// SilhouetteScore calculates the silhouette score for a single data point
// Returns a score between -1 and 1:
//   -1: Point likely in wrong cluster
//    0: Point on the border between clusters
//   +1: Point well matched to its cluster
func SilhouetteScore(
	pointIdx int,
	clusterAssignments []int,
	distances [][]float64,
) float64 {
	n := len(clusterAssignments)
	if n == 0 || pointIdx >= n {
		return 0.0
	}

	currentCluster := clusterAssignments[pointIdx]

	// Calculate a(i): mean distance to other points in same cluster
	a := meanIntraClusterDistance(pointIdx, currentCluster, clusterAssignments, distances)

	// Calculate b(i): min mean distance to points in other clusters
	b := minInterClusterDistance(pointIdx, currentCluster, clusterAssignments, distances)

	// Silhouette score
	if a < b {
		return 1.0 - (a / b)
	} else if a > b {
		return (b / a) - 1.0
	}
	return 0.0 // a == b
}

// meanIntraClusterDistance calculates mean distance to other points in same cluster
func meanIntraClusterDistance(
	pointIdx int,
	clusterLabel int,
	clusterAssignments []int,
	distances [][]float64,
) float64 {
	sumDistance := 0.0
	count := 0

	for i, label := range clusterAssignments {
		if i == pointIdx {
			continue // Skip self
		}
		if label == clusterLabel {
			sumDistance += distances[pointIdx][i]
			count++
		}
	}

	if count == 0 {
		return 0.0 // Single point in cluster
	}

	return sumDistance / float64(count)
}

// minInterClusterDistance finds minimum mean distance to points in other clusters
func minInterClusterDistance(
	pointIdx int,
	currentCluster int,
	clusterAssignments []int,
	distances [][]float64,
) float64 {
	// Find all unique cluster labels except current
	clusterLabels := make(map[int]bool)
	for _, label := range clusterAssignments {
		if label != currentCluster {
			clusterLabels[label] = true
		}
	}

	if len(clusterLabels) == 0 {
		return 1.0 // No other clusters
	}

	minDistance := math.MaxFloat64

	// For each other cluster, calculate mean distance
	for otherCluster := range clusterLabels {
		sumDistance := 0.0
		count := 0

		for i, label := range clusterAssignments {
			if label == otherCluster {
				sumDistance += distances[pointIdx][i]
				count++
			}
		}

		if count > 0 {
			meanDistance := sumDistance / float64(count)
			if meanDistance < minDistance {
				minDistance = meanDistance
			}
		}
	}

	if minDistance == math.MaxFloat64 {
		return 1.0
	}

	return minDistance
}

// AverageSilhouetteScore calculates the mean silhouette score across all points
func AverageSilhouetteScore(
	clusterAssignments []int,
	distances [][]float64,
) float64 {
	n := len(clusterAssignments)
	if n == 0 {
		return 0.0
	}

	totalScore := 0.0
	for i := 0; i < n; i++ {
		score := SilhouetteScore(i, clusterAssignments, distances)
		totalScore += score
	}

	return totalScore / float64(n)
}

// ClusterSilhouetteScores calculates per-cluster silhouette scores
// Returns a slice where index corresponds to cluster label
func ClusterSilhouetteScores(
	clusterAssignments []int,
	distances [][]float64,
) map[int]float64 {
	clusterScores := make(map[int][]float64)

	// Group scores by cluster
	for i, label := range clusterAssignments {
		score := SilhouetteScore(i, clusterAssignments, distances)
		clusterScores[label] = append(clusterScores[label], score)
	}

	// Calculate mean for each cluster
	means := make(map[int]float64)
	for label, scores := range clusterScores {
		if len(scores) == 0 {
			means[label] = 0.0
			continue
		}

		sum := 0.0
		for _, score := range scores {
			sum += score
		}
		means[label] = sum / float64(len(scores))
	}

	return means
}

// DistanceMatrix computes pairwise distances between all points
// Uses the provided distance function (e.g., cosine distance, euclidean)
func DistanceMatrix(
	embeddings [][]float64,
	distanceFunc func(a, b []float64) float64,
) [][]float64 {
	n := len(embeddings)
	matrix := make([][]float64, n)

	for i := 0; i < n; i++ {
		matrix[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			if i == j {
				matrix[i][j] = 0.0
			} else {
				matrix[i][j] = distanceFunc(embeddings[i], embeddings[j])
			}
		}
	}

	return matrix
}

// CosineDistance calculates cosine distance between two vectors
// Distance = 1 - cosine_similarity
func CosineDistance(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 1.0 // Maximum distance for incompatible vectors
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
		return 1.0 // Zero vector = maximum distance
	}

	similarity := dotProduct / (magA * magB)
	return 1.0 - similarity
}

// EuclideanDistance calculates Euclidean distance between two vectors
func EuclideanDistance(a, b []float64) float64 {
	if len(a) != len(b) {
		return math.MaxFloat64
	}

	sumSquares := 0.0
	for i := 0; i < len(a); i++ {
		diff := a[i] - b[i]
		sumSquares += diff * diff
	}

	return math.Sqrt(sumSquares)
}

// SilhouetteAnalysis provides comprehensive silhouette analysis
type SilhouetteAnalysis struct {
	OverallScore      float64            // Average across all points
	ClusterScores     map[int]float64    // Per-cluster average scores
	PointScores       []float64          // Individual point scores
	ClusterAssignments []int             // Cluster labels for each point
	NumClusters       int                // Total number of clusters
	NumPoints         int                // Total number of points
	Quality           string             // Interpretation: Excellent/Good/Fair/Poor
}

// PerformSilhouetteAnalysis performs comprehensive silhouette analysis
func PerformSilhouetteAnalysis(
	embeddings [][]float64,
	clusterAssignments []int,
) *SilhouetteAnalysis {
	// Build distance matrix
	distances := DistanceMatrix(embeddings, CosineDistance)

	// Calculate overall and per-cluster scores
	overallScore := AverageSilhouetteScore(clusterAssignments, distances)
	clusterScores := ClusterSilhouetteScores(clusterAssignments, distances)

	// Calculate individual point scores
	pointScores := make([]float64, len(clusterAssignments))
	for i := range clusterAssignments {
		pointScores[i] = SilhouetteScore(i, clusterAssignments, distances)
	}

	// Count unique clusters
	clusterSet := make(map[int]bool)
	for _, label := range clusterAssignments {
		clusterSet[label] = true
	}

	// Determine quality interpretation
	quality := interpretSilhouetteScore(overallScore)

	return &SilhouetteAnalysis{
		OverallScore:       overallScore,
		ClusterScores:      clusterScores,
		PointScores:        pointScores,
		ClusterAssignments: clusterAssignments,
		NumClusters:        len(clusterSet),
		NumPoints:          len(clusterAssignments),
		Quality:            quality,
	}
}

// interpretSilhouetteScore provides human-readable interpretation
func interpretSilhouetteScore(score float64) string {
	if score >= 0.71 {
		return "Excellent - Strong cluster structure"
	} else if score >= 0.51 {
		return "Good - Reasonable cluster structure"
	} else if score >= 0.26 {
		return "Fair - Weak cluster structure"
	} else if score >= 0.0 {
		return "Poor - No substantial cluster structure"
	} else {
		return "Very Poor - Artificial/forced clustering"
	}
}

// FindOptimalK finds optimal number of clusters using silhouette method
// Tests K from minK to maxK and returns the K with highest average silhouette score
func FindOptimalK(
	embeddings [][]float64,
	minK int,
	maxK int,
	clusterFunc func(embeddings [][]float64, k int) []int,
) (optimalK int, scores map[int]float64) {
	if minK < 2 {
		minK = 2
	}
	if maxK > len(embeddings) {
		maxK = len(embeddings)
	}
	if maxK < minK {
		return minK, nil
	}

	// Build distance matrix once
	distances := DistanceMatrix(embeddings, CosineDistance)

	scores = make(map[int]float64)
	bestK := minK
	bestScore := -2.0 // Minimum possible silhouette score

	for k := minK; k <= maxK; k++ {
		// Cluster with this K
		assignments := clusterFunc(embeddings, k)

		// Calculate silhouette score
		score := AverageSilhouetteScore(assignments, distances)
		scores[k] = score

		if score > bestScore {
			bestScore = score
			bestK = k
		}
	}

	return bestK, scores
}
