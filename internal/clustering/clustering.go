package clustering

import (
	"briefly/internal/core"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"
)

// ClusteringAlgorithm defines the interface for clustering algorithms
type ClusteringAlgorithm interface {
	Cluster(articles []core.Article, k int) ([]core.TopicCluster, error)
}

// KMeansClusterer implements K-means clustering for articles based on embeddings
type KMeansClusterer struct {
	MaxIterations int
	Tolerance     float64
}

// NewKMeansClusterer creates a new K-means clusterer with default parameters
func NewKMeansClusterer() *KMeansClusterer {
	return &KMeansClusterer{
		MaxIterations: 100,
		Tolerance:     1e-6,
	}
}

// Cluster performs K-means clustering on articles using their embeddings
func (k *KMeansClusterer) Cluster(articles []core.Article, numClusters int) ([]core.TopicCluster, error) {
	if len(articles) == 0 {
		return nil, fmt.Errorf("no articles to cluster")
	}

	if numClusters <= 0 {
		return nil, fmt.Errorf("number of clusters must be positive")
	}

	if numClusters > len(articles) {
		numClusters = len(articles)
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

	// If we have fewer articles with embeddings than clusters, reduce cluster count
	if numClusters > len(articlesWithEmbeddings) {
		numClusters = len(articlesWithEmbeddings)
	}

	embeddingDim := len(articlesWithEmbeddings[0].Embedding)

	// Initialize centroids randomly
	centroids := k.initializeCentroids(articlesWithEmbeddings, numClusters, embeddingDim)

	var assignments []int
	converged := false

	for iteration := 0; iteration < k.MaxIterations && !converged; iteration++ {
		// Assign each article to the nearest centroid
		newAssignments := make([]int, len(articlesWithEmbeddings))
		for i, article := range articlesWithEmbeddings {
			newAssignments[i] = k.findNearestCentroid(article.Embedding, centroids)
		}

		// Check for convergence
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
			// Update centroids
			centroids = k.updateCentroids(articlesWithEmbeddings, assignments, numClusters, embeddingDim)
		}
	}

	// Create topic clusters
	clusters := make([]core.TopicCluster, numClusters)
	for i := range clusters {
		clusters[i] = core.TopicCluster{
			ID:         fmt.Sprintf("cluster_%d", i),
			Label:      fmt.Sprintf("Topic %d", i+1),
			ArticleIDs: []string{},
			Centroid:   centroids[i],
			CreatedAt:  time.Now().UTC(),
		}
	}

	// Assign articles to clusters and calculate confidence scores
	for i, article := range articlesWithEmbeddings {
		clusterID := assignments[i]
		clusters[clusterID].ArticleIDs = append(clusters[clusterID].ArticleIDs, article.ID)

		// Calculate confidence as inverse distance to centroid
		distance := euclideanDistance(article.Embedding, centroids[clusterID])
		confidence := 1.0 / (1.0 + distance) // Normalize to 0-1 range

		// Update the article with cluster assignment (this would need to be persisted)
		article.TopicCluster = clusters[clusterID].ID
		article.TopicConfidence = confidence
	}

	// Generate topic labels based on article titles
	for i := range clusters {
		clusters[i].Label = k.generateTopicLabel(articlesWithEmbeddings, assignments, i)
		clusters[i].Keywords = k.extractKeywords(articlesWithEmbeddings, assignments, i)
	}

	return clusters, nil
}

// initializeCentroids randomly initializes cluster centroids
func (k *KMeansClusterer) initializeCentroids(articles []core.Article, numClusters, embeddingDim int) [][]float64 {
	centroids := make([][]float64, numClusters)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Use random articles as initial centroids
	for i := 0; i < numClusters; i++ {
		randomIndex := rng.Intn(len(articles))
		centroids[i] = make([]float64, embeddingDim)
		copy(centroids[i], articles[randomIndex].Embedding)
	}

	return centroids
}

// findNearestCentroid finds the index of the nearest centroid to the given embedding
func (k *KMeansClusterer) findNearestCentroid(embedding []float64, centroids [][]float64) int {
	minDistance := math.Inf(1)
	nearestIndex := 0

	for i, centroid := range centroids {
		distance := euclideanDistance(embedding, centroid)
		if distance < minDistance {
			minDistance = distance
			nearestIndex = i
		}
	}

	return nearestIndex
}

// updateCentroids recalculates centroids based on current assignments
func (k *KMeansClusterer) updateCentroids(articles []core.Article, assignments []int, numClusters, embeddingDim int) [][]float64 {
	centroids := make([][]float64, numClusters)
	counts := make([]int, numClusters)

	// Initialize centroids
	for i := range centroids {
		centroids[i] = make([]float64, embeddingDim)
	}

	// Sum embeddings for each cluster
	for i, article := range articles {
		clusterID := assignments[i]
		counts[clusterID]++
		for j := range article.Embedding {
			centroids[clusterID][j] += article.Embedding[j]
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

// generateTopicLabel creates a human-readable label for a topic cluster
func (k *KMeansClusterer) generateTopicLabel(articles []core.Article, assignments []int, clusterID int) string {
	var clusterArticles []core.Article
	for i, assignment := range assignments {
		if assignment == clusterID {
			clusterArticles = append(clusterArticles, articles[i])
		}
	}

	if len(clusterArticles) == 0 {
		return fmt.Sprintf("Empty Cluster %d", clusterID+1)
	}

	// Simple approach: find common words in titles
	wordCounts := make(map[string]int)
	for _, article := range clusterArticles {
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

	if mostCommonWord != "" {
		return fmt.Sprintf("%s & Related", mostCommonWord)
	}

	return fmt.Sprintf("Topic %d", clusterID+1)
}

// extractKeywords extracts key terms from articles in a cluster
func (k *KMeansClusterer) extractKeywords(articles []core.Article, assignments []int, clusterID int) []string {
	wordCounts := make(map[string]int)

	for i, assignment := range assignments {
		if assignment == clusterID {
			// Extract words from title and first part of content
			words := extractWords(articles[i].Title)
			if len(articles[i].CleanedText) > 200 {
				words = append(words, extractWords(articles[i].CleanedText[:200])...)
			} else {
				words = append(words, extractWords(articles[i].CleanedText)...)
			}

			for _, word := range words {
				if len(word) > 3 {
					wordCounts[word]++
				}
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

// euclideanDistance calculates the Euclidean distance between two vectors
func euclideanDistance(a, b []float64) float64 {
	if len(a) != len(b) {
		return math.Inf(1)
	}

	var sum float64
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return math.Sqrt(sum)
}

// extractWords extracts words from text for keyword analysis
func extractWords(text string) []string {
	// Simple word extraction - in a real implementation, you might want
	// to use a proper tokenizer and remove stop words
	words := []string{}
	word := ""

	for _, char := range text {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
			word += string(char)
		} else {
			if len(word) > 0 {
				words = append(words, word)
				word = ""
			}
		}
	}

	if len(word) > 0 {
		words = append(words, word)
	}

	return words
}

// AutoDetectOptimalClusters uses the elbow method to suggest an optimal number of clusters
func AutoDetectOptimalClusters(articles []core.Article, maxClusters int) (int, error) {
	if len(articles) < 2 {
		return 1, nil
	}

	if maxClusters > len(articles) {
		maxClusters = len(articles)
	}

	clusterer := NewKMeansClusterer()
	var wcss []float64 // Within-cluster sum of squares

	for k := 1; k <= maxClusters; k++ {
		clusters, err := clusterer.Cluster(articles, k)
		if err != nil {
			continue
		}

		// Calculate WCSS for this k
		totalWCSS := 0.0
		for _, cluster := range clusters {
			for _, articleID := range cluster.ArticleIDs {
				// Find the article
				for _, article := range articles {
					if article.ID == articleID && len(article.Embedding) > 0 {
						distance := euclideanDistance(article.Embedding, cluster.Centroid)
						totalWCSS += distance * distance
						break
					}
				}
			}
		}

		wcss = append(wcss, totalWCSS)
	}

	// Find elbow using simple heuristic
	if len(wcss) < 3 {
		return len(wcss), nil
	}

	// Calculate rate of change
	optimalK := 1
	maxImprovement := 0.0

	for i := 1; i < len(wcss)-1; i++ {
		improvement := wcss[i-1] - wcss[i]
		diminishingReturn := wcss[i] - wcss[i+1]

		if improvement > maxImprovement && improvement > 2*diminishingReturn {
			maxImprovement = improvement
			optimalK = i + 1
		}
	}

	return optimalK, nil
}
