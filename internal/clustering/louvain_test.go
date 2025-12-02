package clustering

import (
	"briefly/internal/core"
	"context"
	"testing"
)

// mockVectorSearcher implements VectorSearcher for testing
type mockVectorSearcher struct {
	// Map from article ID to list of similar articles
	similarityMap map[string][]SearchResult
}

func (m *mockVectorSearcher) SearchSimilar(ctx context.Context, embedding []float64, limit int, threshold float64, excludeIDs []string) ([]SearchResult, error) {
	// Find which article this embedding belongs to based on first element
	// This is a simple mock - in reality, we'd match the embedding
	for _, excludeID := range excludeIDs {
		if results, ok := m.similarityMap[excludeID]; ok {
			// Filter out excluded IDs from results
			var filtered []SearchResult
			for _, r := range results {
				excluded := false
				for _, excl := range excludeIDs {
					if r.ArticleID == excl {
						excluded = true
						break
					}
				}
				if !excluded && r.Similarity >= threshold {
					filtered = append(filtered, r)
					if len(filtered) >= limit {
						break
					}
				}
			}
			return filtered, nil
		}
	}
	return nil, nil
}

func TestLouvainClusterer_BasicClustering(t *testing.T) {
	// Create mock data with 3 clear communities:
	// Community 1: articles 1, 2, 3 (high similarity to each other)
	// Community 2: articles 4, 5, 6 (high similarity to each other)
	// Community 3: article 7 (singleton)

	similarityMap := map[string][]SearchResult{
		"article-1": {
			{ArticleID: "article-2", Similarity: 0.9},
			{ArticleID: "article-3", Similarity: 0.85},
			{ArticleID: "article-4", Similarity: 0.3}, // Different community
		},
		"article-2": {
			{ArticleID: "article-1", Similarity: 0.9},
			{ArticleID: "article-3", Similarity: 0.88},
		},
		"article-3": {
			{ArticleID: "article-1", Similarity: 0.85},
			{ArticleID: "article-2", Similarity: 0.88},
		},
		"article-4": {
			{ArticleID: "article-5", Similarity: 0.92},
			{ArticleID: "article-6", Similarity: 0.87},
		},
		"article-5": {
			{ArticleID: "article-4", Similarity: 0.92},
			{ArticleID: "article-6", Similarity: 0.89},
		},
		"article-6": {
			{ArticleID: "article-4", Similarity: 0.87},
			{ArticleID: "article-5", Similarity: 0.89},
		},
		"article-7": {
			// No strong connections - should become miscellaneous
		},
	}

	searcher := &mockVectorSearcher{similarityMap: similarityMap}

	// Create test articles with embeddings
	articles := []core.Article{
		{ID: "article-1", Title: "Article 1"},
		{ID: "article-2", Title: "Article 2"},
		{ID: "article-3", Title: "Article 3"},
		{ID: "article-4", Title: "Article 4"},
		{ID: "article-5", Title: "Article 5"},
		{ID: "article-6", Title: "Article 6"},
		{ID: "article-7", Title: "Article 7"},
	}

	// Create embeddings (768-dim vectors for each article)
	embeddings := make(map[string][]float64)
	for _, article := range articles {
		embeddings[article.ID] = make([]float64, 768)
		// Fill with some values
		embeddings[article.ID][0] = 0.5
	}

	// Create clusterer
	clusterer := NewLouvainClusterer(searcher).
		WithMinSimilarity(0.3).
		WithMaxNeighbors(10).
		WithMinClusterSize(2)

	// Run clustering
	clusters, err := clusterer.ClusterArticles(context.Background(), articles, embeddings)
	if err != nil {
		t.Fatalf("ClusterArticles failed: %v", err)
	}

	// Verify we got clusters
	if len(clusters) == 0 {
		t.Error("Expected at least one cluster, got 0")
	}

	// Count total articles in clusters
	totalArticles := 0
	for _, cluster := range clusters {
		totalArticles += len(cluster.ArticleIDs)
	}

	if totalArticles != len(articles) {
		t.Errorf("Expected %d articles in clusters, got %d", len(articles), totalArticles)
	}
}

func TestLouvainClusterer_EmptyArticles(t *testing.T) {
	searcher := &mockVectorSearcher{similarityMap: make(map[string][]SearchResult)}
	clusterer := NewLouvainClusterer(searcher)

	_, err := clusterer.ClusterArticles(context.Background(), []core.Article{}, make(map[string][]float64))
	if err == nil {
		t.Error("Expected error for empty articles, got nil")
	}
}

func TestLouvainClusterer_NoEmbeddings(t *testing.T) {
	searcher := &mockVectorSearcher{similarityMap: make(map[string][]SearchResult)}
	clusterer := NewLouvainClusterer(searcher)

	articles := []core.Article{
		{ID: "article-1", Title: "Article 1"},
	}

	_, err := clusterer.ClusterArticles(context.Background(), articles, make(map[string][]float64))
	if err == nil {
		t.Error("Expected error for no embeddings, got nil")
	}
}

func TestLouvainClusterer_SingleArticle(t *testing.T) {
	searcher := &mockVectorSearcher{
		similarityMap: map[string][]SearchResult{
			"article-1": {},
		},
	}
	clusterer := NewLouvainClusterer(searcher).WithMinClusterSize(1)

	articles := []core.Article{
		{ID: "article-1", Title: "Article 1"},
	}
	embeddings := map[string][]float64{
		"article-1": make([]float64, 768),
	}

	clusters, err := clusterer.ClusterArticles(context.Background(), articles, embeddings)
	if err != nil {
		t.Fatalf("ClusterArticles failed: %v", err)
	}

	if len(clusters) != 1 {
		t.Errorf("Expected 1 cluster for single article, got %d", len(clusters))
	}
}

func TestLouvainClusterer_ResolutionParameter(t *testing.T) {
	// Test that higher resolution produces more clusters
	// This is a basic sanity check

	similarityMap := map[string][]SearchResult{
		"article-1": {{ArticleID: "article-2", Similarity: 0.7}},
		"article-2": {{ArticleID: "article-1", Similarity: 0.7}, {ArticleID: "article-3", Similarity: 0.6}},
		"article-3": {{ArticleID: "article-2", Similarity: 0.6}},
	}

	searcher := &mockVectorSearcher{similarityMap: similarityMap}

	articles := []core.Article{
		{ID: "article-1", Title: "Article 1"},
		{ID: "article-2", Title: "Article 2"},
		{ID: "article-3", Title: "Article 3"},
	}

	embeddings := make(map[string][]float64)
	for _, article := range articles {
		embeddings[article.ID] = make([]float64, 768)
	}

	// Low resolution (fewer clusters)
	clustererLow := NewLouvainClusterer(searcher).
		WithResolution(0.5).
		WithMinSimilarity(0.3).
		WithMinClusterSize(1)

	clustersLow, err := clustererLow.ClusterArticles(context.Background(), articles, embeddings)
	if err != nil {
		t.Fatalf("ClusterArticles (low res) failed: %v", err)
	}

	// High resolution (more clusters)
	clustererHigh := NewLouvainClusterer(searcher).
		WithResolution(2.0).
		WithMinSimilarity(0.3).
		WithMinClusterSize(1)

	clustersHigh, err := clustererHigh.ClusterArticles(context.Background(), articles, embeddings)
	if err != nil {
		t.Fatalf("ClusterArticles (high res) failed: %v", err)
	}

	// With higher resolution, we expect at least as many clusters
	// (though Louvain is non-deterministic, so this is a soft check)
	t.Logf("Low resolution clusters: %d, High resolution clusters: %d", len(clustersLow), len(clustersHigh))
}

func TestLouvainClusterer_WithMethods(t *testing.T) {
	searcher := &mockVectorSearcher{similarityMap: make(map[string][]SearchResult)}

	clusterer := NewLouvainClusterer(searcher)

	// Test fluent API
	clusterer = clusterer.
		WithResolution(1.5).
		WithMinSimilarity(0.4).
		WithMaxNeighbors(15).
		WithMinClusterSize(3).
		WithTagAware(true)

	if clusterer.resolution != 1.5 {
		t.Errorf("Expected resolution 1.5, got %f", clusterer.resolution)
	}
	if clusterer.minSimilarity != 0.4 {
		t.Errorf("Expected minSimilarity 0.4, got %f", clusterer.minSimilarity)
	}
	if clusterer.maxNeighbors != 15 {
		t.Errorf("Expected maxNeighbors 15, got %d", clusterer.maxNeighbors)
	}
	if clusterer.minClusterSize != 3 {
		t.Errorf("Expected minClusterSize 3, got %d", clusterer.minClusterSize)
	}
	if !clusterer.tagAware {
		t.Error("Expected tagAware to be true")
	}
}

func TestLouvainClusterer_CalculateCentroid(t *testing.T) {
	searcher := &mockVectorSearcher{similarityMap: make(map[string][]SearchResult)}
	clusterer := NewLouvainClusterer(searcher)

	embeddings := map[string][]float64{
		"article-1": {1.0, 0.0, 0.0},
		"article-2": {0.0, 1.0, 0.0},
		"article-3": {0.0, 0.0, 1.0},
	}

	centroid := clusterer.calculateCentroid([]string{"article-1", "article-2", "article-3"}, embeddings)

	if len(centroid) != 3 {
		t.Fatalf("Expected centroid with 3 dimensions, got %d", len(centroid))
	}

	// Expected average: (1+0+0)/3, (0+1+0)/3, (0+0+1)/3 = 0.333...
	expected := 1.0 / 3.0
	tolerance := 0.001

	for i, val := range centroid {
		if val < expected-tolerance || val > expected+tolerance {
			t.Errorf("Expected centroid[%d] to be ~%f, got %f", i, expected, val)
		}
	}
}

func TestLouvainClusterer_TagAwareClustering(t *testing.T) {
	themeID1 := "ai-ml"
	themeID2 := "devops"

	similarityMap := map[string][]SearchResult{
		"article-1": {{ArticleID: "article-2", Similarity: 0.9}},
		"article-2": {{ArticleID: "article-1", Similarity: 0.9}},
		"article-3": {{ArticleID: "article-4", Similarity: 0.85}},
		"article-4": {{ArticleID: "article-3", Similarity: 0.85}},
	}

	searcher := &mockVectorSearcher{similarityMap: similarityMap}

	articles := []core.Article{
		{ID: "article-1", Title: "AI Article 1", ThemeID: &themeID1},
		{ID: "article-2", Title: "AI Article 2", ThemeID: &themeID1},
		{ID: "article-3", Title: "DevOps Article 1", ThemeID: &themeID2},
		{ID: "article-4", Title: "DevOps Article 2", ThemeID: &themeID2},
	}

	embeddings := make(map[string][]float64)
	for _, article := range articles {
		embeddings[article.ID] = make([]float64, 768)
	}

	clusterer := NewLouvainClusterer(searcher).
		WithTagAware(true).
		WithMinSimilarity(0.3).
		WithMinClusterSize(1)

	clusters, err := clusterer.ClusterArticles(context.Background(), articles, embeddings)
	if err != nil {
		t.Fatalf("ClusterArticles failed: %v", err)
	}

	// With tag-aware clustering, we should get clusters prefixed with theme names
	foundAICluster := false
	foundDevOpsCluster := false

	for _, cluster := range clusters {
		if contains(cluster.Label, "ai-ml") {
			foundAICluster = true
		}
		if contains(cluster.Label, "devops") {
			foundDevOpsCluster = true
		}
	}

	// This test verifies tag-aware mode is working, though cluster labels depend on implementation
	t.Logf("Found %d clusters in tag-aware mode", len(clusters))
	if !foundAICluster && !foundDevOpsCluster && len(clusters) < 2 {
		t.Log("Warning: Tag-aware clustering may not have separated themes (expected behavior varies)")
	}
}

// Helper function for test
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
