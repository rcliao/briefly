package clustering

import (
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"sort"
	"time"

	"briefly/internal/core"
	"briefly/internal/logger"
)

// KMeansConfig holds configuration for K-means clustering
type KMeansConfig struct {
	MaxIterations int     // Maximum number of iterations
	Tolerance     float64 // Convergence tolerance
	MinK          int     // Minimum number of clusters for auto-detection
	MaxK          int     // Maximum number of clusters for auto-detection
	MinSilhouette float64 // Minimum acceptable silhouette score
	UseOptimalK   bool    // Whether to auto-detect optimal K using silhouette
}

// DefaultKMeansConfig returns sensible defaults for K-means clustering
func DefaultKMeansConfig() KMeansConfig {
	return KMeansConfig{
		MaxIterations: 100,
		Tolerance:     1e-6,
		MinK:          2,
		MaxK:          8, // Changed from 5 to 8 for better flexibility
		MinSilhouette: 0.3,
		UseOptimalK:   true,
	}
}

// KMeansClustererV2 implements enhanced K-means with optimal K selection
type KMeansClustererV2 struct {
	config KMeansConfig
	log    *slog.Logger
}

// NewKMeansClustererV2 creates a new enhanced K-means clusterer
func NewKMeansClustererV2(config KMeansConfig) *KMeansClustererV2 {
	return &KMeansClustererV2{
		config: config,
		log:    logger.Get(),
	}
}

// ClusterWithOptimalK performs K-means clustering with automatic optimal K selection
func (km *KMeansClustererV2) ClusterWithOptimalK(
	articles []core.Article,
) ([]core.TopicCluster, *SilhouetteAnalysis, error) {
	// Extract embeddings and filter articles
	articlesWithEmbeddings, embeddings, err := km.prepareData(articles)
	if err != nil {
		return nil, nil, err
	}

	n := len(articlesWithEmbeddings)

	// Determine K range
	minK := km.config.MinK
	maxK := km.config.MaxK
	if maxK > n {
		maxK = n
	}
	if minK > maxK {
		minK = maxK
	}

	if !km.config.UseOptimalK {
		// Use fixed K (average of min and max)
		fixedK := (minK + maxK) / 2
		return km.clusterWithK(articlesWithEmbeddings, embeddings, fixedK)
	}

	// Find optimal K using silhouette method
	km.log.Info(fmt.Sprintf("🔍 Finding optimal K (testing K=%d to %d)...", minK, maxK))

	bestK := minK
	bestScore := -2.0
	scores := make(map[int]float64)

	// Build distance matrix once (expensive operation)
	distances := DistanceMatrix(embeddings, CosineDistance)

	for k := minK; k <= maxK; k++ {
		// Cluster with this K
		assignments, _, err := km.runKMeans(embeddings, k)
		if err != nil {
			continue
		}

		// Calculate silhouette score
		score := AverageSilhouetteScore(assignments, distances)
		scores[k] = score

		km.log.Info(fmt.Sprintf("   K=%d: silhouette=%.3f", k, score))

		if score > bestScore {
			bestScore = score
			bestK = k
		}
	}

	km.log.Info(fmt.Sprintf("✓ Optimal K selected: %d (silhouette=%.3f)", bestK, bestScore))

	// Check if best score meets minimum threshold
	if bestScore < km.config.MinSilhouette {
		km.log.Warn(fmt.Sprintf("⚠️  Clustering quality below threshold: %.3f < %.3f",
			bestScore, km.config.MinSilhouette))
		km.log.Warn("   Consider using HDBSCAN or adjusting cluster count")
	}

	// Cluster with optimal K
	return km.clusterWithK(articlesWithEmbeddings, embeddings, bestK)
}

// clusterWithK performs K-means clustering with a specific K value
func (km *KMeansClustererV2) clusterWithK(
	articles []core.Article,
	embeddings [][]float64,
	k int,
) ([]core.TopicCluster, *SilhouetteAnalysis, error) {
	// Run K-means
	assignments, centroids, err := km.runKMeans(embeddings, k)
	if err != nil {
		return nil, nil, err
	}

	// Perform silhouette analysis
	analysis := PerformSilhouetteAnalysis(embeddings, assignments)

	// Build clusters
	clusters := km.buildClusters(articles, assignments, centroids, analysis)

	return clusters, analysis, nil
}

// runKMeans executes the K-means algorithm
// Returns assignments (cluster labels) and centroids
func (km *KMeansClustererV2) runKMeans(
	embeddings [][]float64,
	k int,
) ([]int, [][]float64, error) {
	if len(embeddings) == 0 {
		return nil, nil, fmt.Errorf("no embeddings provided")
	}
	if k <= 0 || k > len(embeddings) {
		return nil, nil, fmt.Errorf("invalid k: %d (must be 1-%d)", k, len(embeddings))
	}

	embeddingDim := len(embeddings[0])

	// Initialize centroids using K-means++
	centroids := km.initializeCentroidsKMeansPP(embeddings, k, embeddingDim)

	var assignments []int
	converged := false

	for iteration := 0; iteration < km.config.MaxIterations && !converged; iteration++ {
		// Assignment step: assign each point to nearest centroid
		newAssignments := make([]int, len(embeddings))
		for i, embedding := range embeddings {
			newAssignments[i] = km.findNearestCentroid(embedding, centroids)
		}

		// Check convergence
		if iteration > 0 {
			converged = true
			for i := range assignments {
				if assignments[i] != newAssignments[i] {
					converged = false
					break
				}
			}
		}

		assignments = newAssignments

		if !converged {
			// Update step: recalculate centroids
			centroids = km.updateCentroids(embeddings, assignments, k, embeddingDim)
		}
	}

	return assignments, centroids, nil
}

// initializeCentroidsKMeansPP uses K-means++ initialization for better cluster quality
func (km *KMeansClustererV2) initializeCentroidsKMeansPP(
	embeddings [][]float64,
	k int,
	embeddingDim int,
) [][]float64 {
	centroids := make([][]float64, k)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Step 1: Choose first centroid randomly
	firstIndex := rng.Intn(len(embeddings))
	centroids[0] = make([]float64, embeddingDim)
	copy(centroids[0], embeddings[firstIndex])

	// Step 2: Choose remaining centroids using weighted probability
	for i := 1; i < k; i++ {
		distances := make([]float64, len(embeddings))
		totalDistance := 0.0

		// Calculate squared distance from each point to nearest existing centroid
		for j, embedding := range embeddings {
			minDist := math.Inf(1)
			for c := 0; c < i; c++ {
				dist := CosineDistance(embedding, centroids[c])
				if dist < minDist {
					minDist = dist
				}
			}
			distances[j] = minDist * minDist // Square the distance
			totalDistance += distances[j]
		}

		// Choose next centroid with probability proportional to squared distance
		if totalDistance == 0 {
			// Fallback to random
			randomIndex := rng.Intn(len(embeddings))
			centroids[i] = make([]float64, embeddingDim)
			copy(centroids[i], embeddings[randomIndex])
			continue
		}

		target := rng.Float64() * totalDistance
		cumulative := 0.0
		selectedIndex := 0

		for j, dist := range distances {
			cumulative += dist
			if cumulative >= target {
				selectedIndex = j
				break
			}
		}

		centroids[i] = make([]float64, embeddingDim)
		copy(centroids[i], embeddings[selectedIndex])
	}

	return centroids
}

// findNearestCentroid finds the index of the nearest centroid using cosine distance
func (km *KMeansClustererV2) findNearestCentroid(
	embedding []float64,
	centroids [][]float64,
) int {
	minDistance := math.Inf(1)
	nearestIndex := 0

	for i, centroid := range centroids {
		distance := CosineDistance(embedding, centroid)
		if distance < minDistance {
			minDistance = distance
			nearestIndex = i
		}
	}

	return nearestIndex
}

// updateCentroids recalculates centroids based on current assignments
func (km *KMeansClustererV2) updateCentroids(
	embeddings [][]float64,
	assignments []int,
	k int,
	embeddingDim int,
) [][]float64 {
	centroids := make([][]float64, k)
	counts := make([]int, k)

	// Initialize centroids
	for i := range centroids {
		centroids[i] = make([]float64, embeddingDim)
	}

	// Sum embeddings for each cluster
	for i, embedding := range embeddings {
		clusterID := assignments[i]
		counts[clusterID]++
		for j := range embedding {
			centroids[clusterID][j] += embedding[j]
		}
	}

	// Average the sums
	for i := range centroids {
		if counts[i] > 0 {
			for j := range centroids[i] {
				centroids[i][j] /= float64(counts[i])
			}
		}
	}

	return centroids
}

// buildClusters creates TopicCluster objects from clustering results
func (km *KMeansClustererV2) buildClusters(
	articles []core.Article,
	assignments []int,
	centroids [][]float64,
	analysis *SilhouetteAnalysis,
) []core.TopicCluster {
	k := len(centroids)
	clusters := make([]core.TopicCluster, k)

	// Initialize clusters
	for i := range clusters {
		clusters[i] = core.TopicCluster{
			ID:         fmt.Sprintf("cluster_%d", i),
			Label:      fmt.Sprintf("Topic %d", i+1),
			ArticleIDs: []string{},
			Centroid:   centroids[i],
			CreatedAt:  time.Now().UTC(),
		}
	}

	// Assign articles to clusters
	for i, article := range articles {
		clusterID := assignments[i]
		clusters[clusterID].ArticleIDs = append(clusters[clusterID].ArticleIDs, article.ID)
	}

	// Generate labels and keywords
	for i := range clusters {
		clusters[i].Label = km.generateTopicLabel(articles, assignments, i)
		clusters[i].Keywords = km.extractKeywords(articles, assignments, i)
	}

	return clusters
}

// generateTopicLabel creates a human-readable label for a cluster
func (km *KMeansClustererV2) generateTopicLabel(
	articles []core.Article,
	assignments []int,
	clusterID int,
) string {
	var clusterArticles []core.Article
	for i, assignment := range assignments {
		if assignment == clusterID {
			clusterArticles = append(clusterArticles, articles[i])
		}
	}

	if len(clusterArticles) == 0 {
		return fmt.Sprintf("Empty Cluster %d", clusterID+1)
	}

	// Extract common words from titles
	wordCounts := make(map[string]int)
	for _, article := range clusterArticles {
		words := extractWords(article.Title)
		for _, word := range words {
			if len(word) > 3 { // Filter short words
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

	if mostCommonWord != "" {
		return fmt.Sprintf("%s & Related", mostCommonWord)
	}

	return fmt.Sprintf("Topic %d", clusterID+1)
}

// extractKeywords extracts key terms from articles in a cluster
func (km *KMeansClustererV2) extractKeywords(
	articles []core.Article,
	assignments []int,
	clusterID int,
) []string {
	wordCounts := make(map[string]int)

	for i, assignment := range assignments {
		if assignment == clusterID {
			// Extract from title and content preview
			words := extractWords(articles[i].Title)
			contentPreview := articles[i].CleanedText
			if len(contentPreview) > 200 {
				contentPreview = contentPreview[:200]
			}
			words = append(words, extractWords(contentPreview)...)

			for _, word := range words {
				if len(word) > 3 {
					wordCounts[word]++
				}
			}
		}
	}

	// Sort by frequency
	type wordFreq struct {
		word  string
		count int
	}
	var sortedWords []wordFreq
	for word, count := range wordCounts {
		sortedWords = append(sortedWords, wordFreq{word, count})
	}

	sort.Slice(sortedWords, func(i, j int) bool {
		return sortedWords[i].count > sortedWords[j].count
	})

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

// prepareData extracts embeddings and filters articles
func (km *KMeansClustererV2) prepareData(
	articles []core.Article,
) ([]core.Article, [][]float64, error) {
	var filtered []core.Article
	var embeddings [][]float64

	for _, article := range articles {
		if len(article.Embedding) > 0 {
			filtered = append(filtered, article)
			embeddings = append(embeddings, article.Embedding)
		}
	}

	if len(filtered) == 0 {
		return nil, nil, fmt.Errorf("no articles with embeddings found")
	}

	return filtered, embeddings, nil
}
