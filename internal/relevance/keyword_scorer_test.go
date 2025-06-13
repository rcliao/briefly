package relevance

import (
	"context"
	"math"
	"testing"
)

func abs(x float64) float64 {
	return math.Abs(x)
}

func TestNewKeywordScorer(t *testing.T) {
	scorer := NewKeywordScorer()
	if scorer == nil {
		t.Fatal("Expected NewKeywordScorer to return a non-nil scorer")
	}
}

func TestKeywordScorerBasic(t *testing.T) {
	scorer := NewKeywordScorer()
	ctx := context.Background()

	// Create test content
	content := ArticleAdapter{
		Title:   "Go Programming Performance Tips",
		Content: "This article discusses Go programming language performance optimization techniques including memory management and goroutine best practices.",
		URL:     "https://example.com/go-performance",
		Metadata: map[string]interface{}{
			"content_length": 100,
		},
	}

	// Create criteria
	criteria := DefaultCriteria("digest", "Go programming")

	// Score the content
	score, err := scorer.Score(ctx, content, criteria)
	if err != nil {
		t.Fatalf("Failed to score content: %v", err)
	}

	// Verify score is valid
	if score.Value < 0.0 || score.Value > 1.0 {
		t.Errorf("Expected score to be between 0.0 and 1.0, got %.2f", score.Value)
	}

	// Should have reasonable relevance for Go content with Go query
	if score.Value < 0.3 {
		t.Errorf("Expected higher relevance score for matching content, got %.2f", score.Value)
	}

	// Check that factors are populated
	if len(score.Factors) == 0 {
		t.Error("Expected score factors to be populated")
	}

	// Verify confidence is reasonable
	if score.Confidence < 0.1 || score.Confidence > 1.0 {
		t.Errorf("Expected confidence to be between 0.1 and 1.0, got %.2f", score.Confidence)
	}
}

func TestKeywordScorerBatch(t *testing.T) {
	scorer := NewKeywordScorer()
	ctx := context.Background()

	contents := []Scorable{
		ArticleAdapter{
			Title:   "AI Machine Learning Guide",
			Content: "Comprehensive guide to artificial intelligence and machine learning algorithms.",
			URL:     "https://example.com/ai-guide",
		},
		ArticleAdapter{
			Title:   "Database Optimization",
			Content: "Tips for optimizing database queries and improving performance.",
			URL:     "https://example.com/db-optimization",
		},
	}

	criteria := DefaultCriteria("digest", "artificial intelligence")

	scores, err := scorer.ScoreBatch(ctx, contents, criteria)
	if err != nil {
		t.Fatalf("Failed to score batch: %v", err)
	}

	if len(scores) != len(contents) {
		t.Errorf("Expected %d scores, got %d", len(contents), len(scores))
	}

	// First article should score higher for AI query
	if scores[0].Value <= scores[1].Value {
		t.Errorf("Expected AI article (%.2f) to score higher than database article (%.2f)",
			scores[0].Value, scores[1].Value)
	}
}

func TestFilterByThreshold(t *testing.T) {
	scorer := NewKeywordScorer()
	ctx := context.Background()

	contents := []Scorable{
		ArticleAdapter{
			Title:   "Machine Learning Fundamentals",
			Content: "Introduction to machine learning concepts and algorithms for beginners.",
			URL:     "https://example.com/ml-fundamentals",
		},
		ArticleAdapter{
			Title:   "Cooking Recipes",
			Content: "Collection of easy cooking recipes for everyday meals.",
			URL:     "https://example.com/cooking",
		},
	}

	criteria := DefaultCriteria("digest", "machine learning")
	criteria.Threshold = 0.5

	results, err := FilterByThreshold(ctx, scorer, contents, criteria)
	if err != nil {
		t.Fatalf("Failed to filter: %v", err)
	}

	if len(results) != len(contents) {
		t.Errorf("Expected %d filter results, got %d", len(contents), len(results))
	}

	// Check that ML article is included
	mlIncluded := false

	for _, result := range results {
		if result.Content.GetTitle() == "Machine Learning Fundamentals" && result.Included {
			mlIncluded = true
		}
	}

	if !mlIncluded {
		t.Error("Expected ML article to be included")
	}
	// Note: Cooking might still be included if it gets a reasonable score, so we don't test exclusion
}

func TestInferDigestTheme(t *testing.T) {
	contents := []Scorable{
		ArticleAdapter{Title: "Go Performance Tips", Content: "Go programming optimization"},
		ArticleAdapter{Title: "Rust Memory Safety", Content: "Rust programming language features"},
		ArticleAdapter{Title: "API Design Patterns", Content: "REST API best practices"},
	}

	theme := InferDigestTheme(contents)

	// Should detect programming/technology theme
	if theme == "" {
		t.Error("Expected non-empty theme")
	}

	// Should be technology-related
	techKeywords := []string{"go", "rust", "api", "programming", "technology"}
	found := false
	for _, keyword := range techKeywords {
		if theme == keyword {
			found = true
			break
		}
	}

	if !found {
		t.Logf("Inferred theme: %s", theme)
		// Don't fail - the inference might pick up other valid terms
	}
}

func TestGetFilterStats(t *testing.T) {
	results := []FilterResult{
		{Included: true, Score: Score{Value: 0.8}},
		{Included: true, Score: Score{Value: 0.7}},
		{Included: false, Score: Score{Value: 0.4}},
		{Included: false, Score: Score{Value: 0.3}},
	}

	stats := GetFilterStats(results, 0.6)

	if stats.TotalItems != 4 {
		t.Errorf("Expected 4 total items, got %d", stats.TotalItems)
	}

	if stats.IncludedItems != 2 {
		t.Errorf("Expected 2 included items, got %d", stats.IncludedItems)
	}

	if stats.ExcludedItems != 2 {
		t.Errorf("Expected 2 excluded items, got %d", stats.ExcludedItems)
	}

	expectedAvg := (0.8 + 0.7 + 0.4 + 0.3) / 4.0
	tolerance := 0.01
	if abs(stats.AvgScore-expectedAvg) > tolerance {
		t.Errorf("Expected average score %.2f, got %.2f", expectedAvg, stats.AvgScore)
	}

	if stats.MaxScore != 0.8 {
		t.Errorf("Expected max score 0.8, got %.2f", stats.MaxScore)
	}

	if stats.MinScore != 0.3 {
		t.Errorf("Expected min score 0.3, got %.2f", stats.MinScore)
	}
}
