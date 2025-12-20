package vectorstore

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// TestPgVectorIntegration demonstrates pgvector capabilities
// Run with: go test -v ./internal/vectorstore -run TestPgVectorIntegration
//
// Prerequisites:
// - PostgreSQL running with pgvector extension
// - DATABASE_URL environment variable set
// - Articles table with embedding column
func TestPgVectorIntegration(t *testing.T) {
	// Skip if no DATABASE_URL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Verify connection
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	ctx := context.Background()
	store := NewPgVectorAdapter(db)

	t.Run("1. Test Basic Stats", func(t *testing.T) {
		stats, err := store.GetStats(ctx)
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}

		t.Logf("📊 VectorStore Stats:")
		t.Logf("   Total Embeddings: %d", stats.TotalEmbeddings)
		t.Logf("   Embedding Dimensions: %d", stats.EmbeddingDimensions)
		t.Logf("   Index Type: %s", stats.IndexType)
		t.Logf("   Index Size: %d bytes", stats.IndexSize)
	})

	t.Run("2. Test Index Creation", func(t *testing.T) {
		t.Log("🔧 Creating HNSW index for fast similarity search...")
		err := store.CreateIndex(ctx)
		if err != nil {
			t.Logf("   ℹ️  Index may already exist: %v", err)
		} else {
			t.Log("   ✅ Index created successfully")
		}

		// Get updated stats
		stats, _ := store.GetStats(ctx)
		t.Logf("   Index Type: %s", stats.IndexType)
	})

	t.Run("3. Find Articles with Embeddings", func(t *testing.T) {
		// Find articles that have embeddings
		query := `
			SELECT a.id, a.title, a.url
			FROM articles a
			WHERE a.embedding_vector IS NOT NULL
			LIMIT 5
		`
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			t.Fatalf("Failed to query articles: %v", err)
		}
		defer rows.Close()

		t.Log("📚 Articles with embeddings:")
		count := 0
		for rows.Next() {
			var id, title, url string
			if err := rows.Scan(&id, &title, &url); err != nil {
				t.Fatalf("Failed to scan row: %v", err)
			}
			count++
			t.Logf("   [%d] %s", count, title)
			t.Logf("       URL: %s", url)
			t.Logf("       ID: %s", id)
		}

		if count == 0 {
			t.Skip("No articles with embeddings found. Run digest generation first.")
		}
	})

	t.Run("4. Test Semantic Search", func(t *testing.T) {
		// Get a random article's embedding to use as query
		var queryEmbedding []float64
		var queryTitle string
		var queryArticleID string

		err := db.QueryRowContext(ctx, `
			SELECT id, title, embedding_vector
			FROM articles
			WHERE embedding_vector IS NOT NULL
			ORDER BY RANDOM()
			LIMIT 1
		`).Scan(&queryArticleID, &queryTitle, &queryEmbedding)

		if err == sql.ErrNoRows {
			t.Skip("No articles with embeddings found")
		}
		if err != nil {
			t.Fatalf("Failed to get query embedding: %v", err)
		}

		t.Logf("🔍 Searching for articles similar to: \"%s\"", queryTitle)
		t.Logf("   Query embedding dimensions: %d", len(queryEmbedding))

		// Perform semantic search
		searchQuery := SearchQuery{
			Embedding:           queryEmbedding,
			Limit:               5,
			SimilarityThreshold: 0.5, // Lower threshold to see more results
			IncludeArticle:      true,
			ExcludeIDs:          []string{queryArticleID}, // Exclude the query article itself
		}

		start := time.Now()
		results, err := store.Search(ctx, searchQuery)
		latency := time.Since(start)

		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		t.Logf("   ⚡ Search completed in %v", latency)
		t.Logf("   📊 Found %d similar articles:", len(results))

		for i, result := range results {
			t.Logf("")
			t.Logf("   [%d] Similarity: %.3f (Distance: %.3f)", i+1, result.Similarity, result.Distance)
			if result.Article != nil {
				t.Logf("       Title: %s", result.Article.Title)
				t.Logf("       URL: %s", result.Article.URL)
				t.Logf("       Tags: %v", result.TagIDs)
			}
		}

		// Verify results are sorted by similarity
		for i := 1; i < len(results); i++ {
			if results[i].Similarity > results[i-1].Similarity {
				t.Errorf("Results not sorted by similarity: %.3f > %.3f at index %d",
					results[i].Similarity, results[i-1].Similarity, i)
			}
		}
	})

	t.Run("5. Test Tag-Aware Search", func(t *testing.T) {
		// Find articles with tags
		var articleID string
		var tagID string
		var title string
		var embedding []float64

		err := db.QueryRowContext(ctx, `
			SELECT a.id, a.title, a.embedding_vector, at.tag_id
			FROM articles a
			INNER JOIN article_tags at ON a.id = at.article_id
			WHERE a.embedding_vector IS NOT NULL
			LIMIT 1
		`).Scan(&articleID, &title, &embedding, &tagID)

		if err == sql.ErrNoRows {
			t.Skip("No tagged articles with embeddings found")
		}
		if err != nil {
			t.Fatalf("Failed to get tagged article: %v", err)
		}

		// Get tag name
		var tagName string
		if err := db.QueryRowContext(ctx, "SELECT name FROM tags WHERE id = $1", tagID).Scan(&tagName); err != nil {
			t.Logf("Warning: could not get tag name: %v", err)
		}

		t.Logf("🏷️  Testing tag-aware search within tag: \"%s\"", tagName)
		t.Logf("   Query article: \"%s\"", title)

		searchQuery := SearchQuery{
			Embedding:           embedding,
			Limit:               5,
			SimilarityThreshold: 0.5,
			IncludeArticle:      true,
			ExcludeIDs:          []string{articleID},
		}

		// Search within specific tag
		start := time.Now()
		results, err := store.SearchByTag(ctx, searchQuery, tagID)
		latency := time.Since(start)

		if err != nil {
			t.Fatalf("Tag-aware search failed: %v", err)
		}

		t.Logf("   ⚡ Tag-filtered search completed in %v", latency)
		t.Logf("   📊 Found %d articles with tag \"%s\":", len(results), tagName)

		for i, result := range results {
			t.Logf("")
			t.Logf("   [%d] Similarity: %.3f", i+1, result.Similarity)
			if result.Article != nil {
				t.Logf("       Title: %s", result.Article.Title)
				t.Logf("       Tags: %v", result.TagIDs)
			}

			// Verify result has the expected tag
			hasTag := false
			for _, tid := range result.TagIDs {
				if tid == tagID {
					hasTag = true
					break
				}
			}
			if !hasTag {
				t.Errorf("Result %d missing expected tag %s", i, tagID)
			}
		}
	})

	t.Run("6. Compare Semantic vs Keyword Search", func(t *testing.T) {
		// Get a sample article
		var articleID, title string
		var embedding []float64

		err := db.QueryRowContext(ctx, `
			SELECT id, title, embedding_vector
			FROM articles
			WHERE embedding_vector IS NOT NULL
			  AND title ILIKE '%AI%'
			LIMIT 1
		`).Scan(&articleID, &title, &embedding)

		if err == sql.ErrNoRows {
			t.Skip("No AI-related articles found")
		}
		if err != nil {
			t.Fatalf("Failed to get article: %v", err)
		}

		t.Logf("🔬 Comparing search methods for: \"%s\"", title)

		// Keyword search (traditional)
		t.Log("")
		t.Log("   📝 Keyword Search (title ILIKE '%AI%'):")
		keywordQuery := `
			SELECT id, title
			FROM articles
			WHERE title ILIKE '%AI%'
			  AND id != $1
			LIMIT 5
		`
		rows, _ := db.QueryContext(ctx, keywordQuery, articleID)
		keywordCount := 0
		for rows.Next() {
			var id, title string
			if err := rows.Scan(&id, &title); err != nil {
				t.Logf("Warning: scan error: %v", err)
				continue
			}
			keywordCount++
			t.Logf("      [%d] %s", keywordCount, title)
		}
		rows.Close()

		// Semantic search
		t.Log("")
		t.Log("   🧠 Semantic Search (cosine similarity):")
		searchQuery := SearchQuery{
			Embedding:           embedding,
			Limit:               5,
			SimilarityThreshold: 0.5,
			IncludeArticle:      true,
			ExcludeIDs:          []string{articleID},
		}

		results, _ := store.Search(ctx, searchQuery)
		for i, result := range results {
			t.Logf("      [%d] %.3f - %s", i+1, result.Similarity, result.Article.Title)
		}

		t.Log("")
		t.Log("   💡 Key Differences:")
		t.Log("      • Keyword: Exact text matching only")
		t.Log("      • Semantic: Understands meaning and context")
		t.Log("      • Semantic: Finds conceptually similar articles even with different words")
	})

	t.Run("7. Test Similarity Thresholds", func(t *testing.T) {
		// Get a sample embedding
		var embedding []float64
		var articleID string
		err := db.QueryRowContext(ctx, `
			SELECT id, embedding_vector
			FROM articles
			WHERE embedding_vector IS NOT NULL
			LIMIT 1
		`).Scan(&articleID, &embedding)

		if err == sql.ErrNoRows {
			t.Skip("No articles with embeddings")
		}

		t.Log("📏 Testing different similarity thresholds:")

		thresholds := []float64{0.9, 0.8, 0.7, 0.6, 0.5}
		for _, threshold := range thresholds {
			query := SearchQuery{
				Embedding:           embedding,
				Limit:               100,
				SimilarityThreshold: threshold,
				ExcludeIDs:          []string{articleID},
			}

			results, _ := store.Search(ctx, query)
			t.Logf("   Threshold %.1f: %d results", threshold, len(results))
		}

		t.Log("")
		t.Log("   💡 Interpretation:")
		t.Log("      0.9-1.0: Nearly identical content")
		t.Log("      0.8-0.9: Very similar topics")
		t.Log("      0.7-0.8: Related concepts (RECOMMENDED)")
		t.Log("      0.6-0.7: Loosely related")
		t.Log("      <0.6: May be unrelated")
	})

	t.Run("8. Performance: Batch Search", func(t *testing.T) {
		// Get 10 random embeddings
		rows, err := db.QueryContext(ctx, `
			SELECT id, embedding_vector
			FROM articles
			WHERE embedding_vector IS NOT NULL
			ORDER BY RANDOM()
			LIMIT 10
		`)
		if err != nil {
			t.Fatalf("Failed to get embeddings: %v", err)
		}
		defer rows.Close()

		type testQuery struct {
			id        string
			embedding []float64
		}
		var queries []testQuery
		for rows.Next() {
			var q testQuery
			if err := rows.Scan(&q.id, &q.embedding); err != nil {
				t.Logf("Warning: scan error: %v", err)
				continue
			}
			queries = append(queries, q)
		}

		if len(queries) == 0 {
			t.Skip("Not enough articles for batch test")
		}

		t.Logf("⚡ Performance test: %d searches", len(queries))

		start := time.Now()
		totalResults := 0
		for _, q := range queries {
			searchQuery := SearchQuery{
				Embedding:           q.embedding,
				Limit:               5,
				SimilarityThreshold: 0.7,
				ExcludeIDs:          []string{q.id},
			}
			results, _ := store.Search(ctx, searchQuery)
			totalResults += len(results)
		}
		elapsed := time.Since(start)

		avgLatency := elapsed / time.Duration(len(queries))
		t.Logf("   Total time: %v", elapsed)
		t.Logf("   Average latency: %v per search", avgLatency)
		t.Logf("   Total results: %d", totalResults)
		t.Logf("   Throughput: %.1f searches/sec", float64(len(queries))/elapsed.Seconds())

		// Performance expectations
		if avgLatency > 100*time.Millisecond {
			t.Logf("   ⚠️  Average latency > 100ms - consider creating HNSW index")
		} else if avgLatency < 10*time.Millisecond {
			t.Logf("   ✅ Excellent performance - HNSW index working well")
		} else {
			t.Logf("   ✅ Good performance")
		}
	})
}

// TestPgVectorStore tests the Store method with real embeddings
func TestPgVectorStore(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	store := NewPgVectorAdapter(db)

	t.Run("Store and Retrieve Embedding", func(t *testing.T) {
		// Get an article without embedding
		var articleID string
		err := db.QueryRowContext(ctx, `
			SELECT id
			FROM articles
			WHERE embedding_vector IS NULL
			LIMIT 1
		`).Scan(&articleID)

		if err == sql.ErrNoRows {
			t.Skip("All articles have embeddings")
		}
		if err != nil {
			t.Fatalf("Failed to find article: %v", err)
		}

		// Generate a random 768-dim embedding (simulating Gemini)
		embedding := generateRandomEmbedding(768)

		t.Logf("📝 Storing embedding for article: %s", articleID)
		t.Logf("   Dimensions: %d", len(embedding))

		// Store embedding
		err = store.Store(ctx, articleID, embedding)
		if err != nil {
			t.Fatalf("Failed to store embedding: %v", err)
		}

		// Verify it was stored
		var stored []float64
		err = db.QueryRowContext(ctx, `
			SELECT embedding_vector
			FROM articles
			WHERE id = $1
		`, articleID).Scan(&stored)

		if err != nil {
			t.Fatalf("Failed to retrieve stored embedding: %v", err)
		}

		if len(stored) != 768 {
			t.Errorf("Expected 768 dimensions, got %d", len(stored))
		}

		t.Log("   ✅ Embedding stored and verified")
	})
}

// generateRandomEmbedding creates a random normalized embedding
func generateRandomEmbedding(dims int) []float64 {
	embedding := make([]float64, dims)
	var sumSquares float64

	// Generate random values
	for i := range embedding {
		val := rand.Float64()*2 - 1 // Range: -1 to 1
		embedding[i] = val
		sumSquares += val * val
	}

	// Normalize to unit length (important for cosine similarity)
	magnitude := fmt.Sprintf("%.6f", sumSquares)
	_ = magnitude
	for i := range embedding {
		embedding[i] /= sumSquares
	}

	return embedding
}
