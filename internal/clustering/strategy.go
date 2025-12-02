package clustering

import (
	"fmt"
	"log/slog"
	"math"

	"briefly/internal/core"
	"briefly/internal/logger"
)

// ClusteringStrategy represents which clustering algorithm to use
type ClusteringStrategy string

const (
	StrategyKMeans  ClusteringStrategy = "kmeans"
	StrategyHDBSCAN ClusteringStrategy = "hdbscan"
	StrategyLouvain ClusteringStrategy = "louvain"
	StrategyAuto    ClusteringStrategy = "auto"
)

// StrategySelector intelligently selects the best clustering algorithm
// based on data characteristics
type StrategySelector struct {
	log *slog.Logger
}

// NewStrategySelector creates a new strategy selector
func NewStrategySelector() *StrategySelector {
	return &StrategySelector{
		log: logger.Get(),
	}
}

// SelectStrategy chooses the best clustering algorithm based on data characteristics
func (s *StrategySelector) SelectStrategy(articles []core.Article) ClusteringStrategy {
	// Extract articles with embeddings
	var articlesWithEmbeddings []core.Article
	var embeddings [][]float64

	for _, article := range articles {
		if len(article.Embedding) > 0 {
			articlesWithEmbeddings = append(articlesWithEmbeddings, article)
			embeddings = append(embeddings, article.Embedding)
		}
	}

	n := len(articlesWithEmbeddings)

	s.log.Info(fmt.Sprintf("🤔 Selecting clustering strategy for %d articles...", n))

	// Decision criteria:

	// 1. Small datasets (< 8 articles): Use K-means
	//    - HDBSCAN needs more data to find patterns
	//    - K-means faster and more predictable for small datasets
	if n < 8 {
		s.log.Info("   ✓ Selected K-means: Small dataset (< 8 articles)")
		return StrategyKMeans
	}

	// 2. Large datasets (>= 15 articles): Use HDBSCAN
	//    - HDBSCAN better at finding natural clusters in larger datasets
	//    - Can identify outliers/noise
	//    - Auto-determines optimal number of clusters
	if n >= 15 {
		s.log.Info("   ✓ Selected HDBSCAN: Large dataset (>= 15 articles)")
		return StrategyHDBSCAN
	}

	// 3. Medium datasets (8-14 articles): Analyze diversity
	//    - High diversity → HDBSCAN (can find multiple distinct clusters)
	//    - Low diversity → K-means (more stable with similar articles)

	diversity := s.calculateDatasetDiversity(embeddings)
	s.log.Info(fmt.Sprintf("   Dataset diversity: %.3f", diversity))

	// Diversity threshold: 0.6
	// - Above 0.6 → High diversity, use HDBSCAN
	// - Below 0.6 → Low diversity, use K-means
	if diversity > 0.6 {
		s.log.Info("   ✓ Selected HDBSCAN: High diversity (> 0.6)")
		return StrategyHDBSCAN
	} else {
		s.log.Info("   ✓ Selected K-means: Low diversity (≤ 0.6)")
		return StrategyKMeans
	}
}

// calculateDatasetDiversity measures how diverse/varied the dataset is
// Returns a score from 0 (very similar) to 1 (very diverse)
func (s *StrategySelector) calculateDatasetDiversity(embeddings [][]float64) float64 {
	n := len(embeddings)
	if n <= 1 {
		return 0.0
	}

	// Calculate mean pairwise cosine distance
	// Higher distance = more diverse dataset
	totalDistance := 0.0
	count := 0

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			distance := CosineDistance(embeddings[i], embeddings[j])
			totalDistance += distance
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	avgDistance := totalDistance / float64(count)

	// Normalize: cosine distance ranges from 0 to 2
	// Map to 0-1 range
	diversity := avgDistance / 2.0

	return diversity
}

// AdaptiveClusterer wraps clustering algorithms with adaptive strategy selection
type AdaptiveClusterer struct {
	kmeansConfig  KMeansConfig
	hdbscanConfig HDBSCANConfig
	selector      *StrategySelector
	log           *slog.Logger
}

// NewAdaptiveClusterer creates a new adaptive clusterer
func NewAdaptiveClusterer() *AdaptiveClusterer {
	return &AdaptiveClusterer{
		kmeansConfig:  DefaultKMeansConfig(),
		hdbscanConfig: DefaultHDBSCANConfig(),
		selector:      NewStrategySelector(),
		log:           logger.Get(),
	}
}

// NewAdaptiveClustererWithConfig creates an adaptive clusterer with custom configs
func NewAdaptiveClustererWithConfig(
	kmeansConfig KMeansConfig,
	hdbscanConfig HDBSCANConfig,
) *AdaptiveClusterer {
	return &AdaptiveClusterer{
		kmeansConfig:  kmeansConfig,
		hdbscanConfig: hdbscanConfig,
		selector:      NewStrategySelector(),
		log:           logger.Get(),
	}
}

// Cluster performs adaptive clustering - automatically selects best algorithm
func (ac *AdaptiveClusterer) Cluster(
	articles []core.Article,
) ([]core.TopicCluster, *SilhouetteAnalysis, ClusteringStrategy, error) {
	// Select strategy
	strategy := ac.selector.SelectStrategy(articles)

	// Cluster using selected strategy
	var clusters []core.TopicCluster
	var analysis *SilhouetteAnalysis
	var err error

	switch strategy {
	case StrategyKMeans:
		clusterer := NewKMeansClustererV2(ac.kmeansConfig)
		clusters, analysis, err = clusterer.ClusterWithOptimalK(articles)
		if err != nil {
			return nil, nil, strategy, fmt.Errorf("K-means clustering failed: %w", err)
		}

	case StrategyHDBSCAN:
		// For HDBSCAN, we need to use the existing implementation
		// and then add silhouette analysis
		hdbscanClusterer := &HDBSCANClusterer{
			MinClusterSize: ac.hdbscanConfig.MinClusterSize,
			MinSamples:     ac.hdbscanConfig.MinSamples,
		}

		clusters, err = hdbscanClusterer.Cluster(articles, 0) // k is ignored for HDBSCAN
		if err != nil {
			return nil, nil, strategy, fmt.Errorf("HDBSCAN clustering failed: %w", err)
		}

		// Perform silhouette analysis post-clustering
		analysis = ac.performPostClusteringAnalysis(clusters, articles)

	default:
		return nil, nil, strategy, fmt.Errorf("unknown clustering strategy: %s", strategy)
	}

	// Validate cluster quality
	if analysis.OverallScore < ac.kmeansConfig.MinSilhouette {
		ac.log.Warn(fmt.Sprintf("⚠️  Clustering quality below threshold: %.3f < %.3f",
			analysis.OverallScore, ac.kmeansConfig.MinSilhouette))
	}

	ac.log.Info(fmt.Sprintf("✓ Clustering complete: %d clusters, silhouette=%.3f (%s)",
		len(clusters), analysis.OverallScore, analysis.Quality))

	return clusters, analysis, strategy, nil
}

// performPostClusteringAnalysis performs silhouette analysis on existing clusters
func (ac *AdaptiveClusterer) performPostClusteringAnalysis(
	clusters []core.TopicCluster,
	articles []core.Article,
) *SilhouetteAnalysis {
	// Build article ID to embedding map
	articleEmbeddings := make(map[string][]float64)
	for _, article := range articles {
		if len(article.Embedding) > 0 {
			articleEmbeddings[article.ID] = article.Embedding
		}
	}

	// Build embeddings array and assignments array
	var embeddings [][]float64
	var assignments []int
	articleIDToIdx := make(map[string]int)

	idx := 0
	for _, article := range articles {
		if emb, ok := articleEmbeddings[article.ID]; ok {
			embeddings = append(embeddings, emb)
			articleIDToIdx[article.ID] = idx
			idx++
		}
	}

	// Assign cluster labels
	assignments = make([]int, len(embeddings))
	for clusterIdx, cluster := range clusters {
		for _, articleID := range cluster.ArticleIDs {
			if artIdx, ok := articleIDToIdx[articleID]; ok {
				assignments[artIdx] = clusterIdx
			}
		}
	}

	// Perform silhouette analysis
	return PerformSilhouetteAnalysis(embeddings, assignments)
}

// ClusterWithStrategy clusters using a specific strategy (no auto-selection)
func (ac *AdaptiveClusterer) ClusterWithStrategy(
	articles []core.Article,
	strategy ClusteringStrategy,
) ([]core.TopicCluster, *SilhouetteAnalysis, error) {
	ac.log.Info(fmt.Sprintf("🎯 Using %s clustering strategy (manual override)", strategy))

	var clusters []core.TopicCluster
	var analysis *SilhouetteAnalysis
	var err error

	switch strategy {
	case StrategyKMeans:
		clusterer := NewKMeansClustererV2(ac.kmeansConfig)
		clusters, analysis, err = clusterer.ClusterWithOptimalK(articles)

	case StrategyHDBSCAN:
		hdbscanClusterer := &HDBSCANClusterer{
			MinClusterSize: ac.hdbscanConfig.MinClusterSize,
			MinSamples:     ac.hdbscanConfig.MinSamples,
		}
		clusters, err = hdbscanClusterer.Cluster(articles, 0)
		if err == nil {
			analysis = ac.performPostClusteringAnalysis(clusters, articles)
		}

	default:
		return nil, nil, fmt.Errorf("unknown clustering strategy: %s", strategy)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("%s clustering failed: %w", strategy, err)
	}

	return clusters, analysis, nil
}

// CompareStrategies runs both K-means and HDBSCAN and returns results for comparison
func (ac *AdaptiveClusterer) CompareStrategies(
	articles []core.Article,
) (*StrategyComparison, error) {
	comparison := &StrategyComparison{
		NumArticles: len(articles),
	}

	// Run K-means
	ac.log.Info("Running K-means for comparison...")
	kmeansClusterer := NewKMeansClustererV2(ac.kmeansConfig)
	kmeansClusters, kmeansAnalysis, err := kmeansClusterer.ClusterWithOptimalK(articles)
	if err != nil {
		comparison.KMeansError = err.Error()
	} else {
		comparison.KMeansClusters = len(kmeansClusters)
		comparison.KMeansSilhouette = kmeansAnalysis.OverallScore
		comparison.KMeansQuality = kmeansAnalysis.Quality
	}

	// Run HDBSCAN
	ac.log.Info("Running HDBSCAN for comparison...")
	hdbscanClusterer := &HDBSCANClusterer{
		MinClusterSize: ac.hdbscanConfig.MinClusterSize,
		MinSamples:     ac.hdbscanConfig.MinSamples,
	}
	hdbscanClusters, err := hdbscanClusterer.Cluster(articles, 0)
	if err != nil {
		comparison.HDBSCANError = err.Error()
	} else {
		hdbscanAnalysis := ac.performPostClusteringAnalysis(hdbscanClusters, articles)
		comparison.HDBSCANClusters = len(hdbscanClusters)
		comparison.HDBSCANSilhouette = hdbscanAnalysis.OverallScore
		comparison.HDBSCANQuality = hdbscanAnalysis.Quality
	}

	// Determine winner
	if comparison.KMeansError == "" && comparison.HDBSCANError == "" {
		if comparison.KMeansSilhouette > comparison.HDBSCANSilhouette {
			comparison.Winner = "K-means"
			comparison.WinnerMargin = comparison.KMeansSilhouette - comparison.HDBSCANSilhouette
		} else {
			comparison.Winner = "HDBSCAN"
			comparison.WinnerMargin = comparison.HDBSCANSilhouette - comparison.KMeansSilhouette
		}
	} else if comparison.KMeansError == "" {
		comparison.Winner = "K-means"
	} else if comparison.HDBSCANError == "" {
		comparison.Winner = "HDBSCAN"
	}

	return comparison, nil
}

// StrategyComparison holds results from comparing both strategies
type StrategyComparison struct {
	NumArticles int

	// K-means results
	KMeansClusters   int
	KMeansSilhouette float64
	KMeansQuality    string
	KMeansError      string

	// HDBSCAN results
	HDBSCANClusters   int
	HDBSCANSilhouette float64
	HDBSCANQuality    string
	HDBSCANError      string

	// Comparison
	Winner       string  // "K-means", "HDBSCAN", or ""
	WinnerMargin float64 // Silhouette score difference
}

// PrintComparison prints a formatted comparison report
func (c *StrategyComparison) PrintComparison() {
	fmt.Println("\n═══════════════════════════════════════════════════════════════════")
	fmt.Printf("CLUSTERING STRATEGY COMPARISON (%d articles)\n", c.NumArticles)
	fmt.Println("═══════════════════════════════════════════════════════════════════")

	// K-means results
	fmt.Println("K-MEANS:")
	if c.KMeansError != "" {
		fmt.Printf("  ❌ Error: %s\n", c.KMeansError)
	} else {
		fmt.Printf("  Clusters: %d\n", c.KMeansClusters)
		fmt.Printf("  Silhouette: %.3f\n", c.KMeansSilhouette)
		fmt.Printf("  Quality: %s\n", c.KMeansQuality)
	}

	fmt.Println()

	// HDBSCAN results
	fmt.Println("HDBSCAN:")
	if c.HDBSCANError != "" {
		fmt.Printf("  ❌ Error: %s\n", c.HDBSCANError)
	} else {
		fmt.Printf("  Clusters: %d\n", c.HDBSCANClusters)
		fmt.Printf("  Silhouette: %.3f\n", c.HDBSCANSilhouette)
		fmt.Printf("  Quality: %s\n", c.HDBSCANQuality)
	}

	fmt.Println()

	// Winner
	if c.Winner != "" {
		icon := "🏆"
		if c.WinnerMargin < 0.05 {
			icon = "🤝" // Close race
		}
		fmt.Printf("%s WINNER: %s (margin: %.3f)\n", icon, c.Winner, math.Abs(c.WinnerMargin))
	}

	fmt.Println("═══════════════════════════════════════════════════════════════════")
}
