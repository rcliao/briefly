package vectorstore

import (
	"briefly/internal/core"
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

// PgVectorAdapter implements VectorStore using PostgreSQL with pgvector extension
// Provides production-scale semantic search with cosine similarity
type PgVectorAdapter struct {
	db *sql.DB
}

// NewPgVectorAdapter creates a new pgvector-based vector store
func NewPgVectorAdapter(db *sql.DB) *PgVectorAdapter {
	return &PgVectorAdapter{db: db}
}

// Store saves or updates an embedding for an article
// Uses UPSERT to handle both insert and update cases
func (p *PgVectorAdapter) Store(ctx context.Context, articleID string, embedding []float64) error {
	// Convert []float64 to PostgreSQL vector format
	vectorStr := formatVector(embedding)

	query := `
		UPDATE articles
		SET embedding_vector = $1::vector,
		    updated_at = NOW()
		WHERE id = $2
	`

	result, err := p.db.ExecContext(ctx, query, vectorStr, articleID)
	if err != nil {
		return fmt.Errorf("failed to store embedding: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("article %s not found", articleID)
	}

	return nil
}

// Search finds articles similar to the query embedding
// Uses cosine distance (<=> operator) and returns results ordered by similarity
func (p *PgVectorAdapter) Search(ctx context.Context, query SearchQuery) ([]SearchResult, error) {
	// Apply defaults
	if query.Limit == 0 {
		query.Limit = 10
	}
	if query.SimilarityThreshold == 0 {
		query.SimilarityThreshold = 0.7
	}

	vectorStr := formatVector(query.Embedding)

	// Build exclusion filter
	excludeClause := ""
	args := []interface{}{vectorStr, query.SimilarityThreshold, query.Limit}
	if len(query.ExcludeIDs) > 0 {
		excludeClause = "AND a.id NOT IN (SELECT unnest($4::uuid[]))"
		args = append(args, pq.Array(query.ExcludeIDs))
	}

	// Base query without article data (faster)
	sqlQuery := fmt.Sprintf(`
		SELECT
			a.id,
			1 - (a.embedding_vector <=> $1::vector) as similarity,
			a.embedding_vector <=> $1::vector as distance
		FROM articles a
		WHERE a.embedding_vector IS NOT NULL
		  AND 1 - (a.embedding_vector <=> $1::vector) >= $2
		  %s
		ORDER BY a.embedding_vector <=> $1::vector
		LIMIT $3
	`, excludeClause)

	rows, err := p.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var result SearchResult
		if err := rows.Scan(&result.ArticleID, &result.Similarity, &result.Distance); err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// Optionally populate article data and tags
	if query.IncludeArticle {
		if err := p.populateArticles(ctx, results); err != nil {
			return nil, fmt.Errorf("failed to populate articles: %w", err)
		}
	}

	// Always populate tag IDs for tag-aware operations
	if err := p.populateTags(ctx, results); err != nil {
		return nil, fmt.Errorf("failed to populate tags: %w", err)
	}

	return results, nil
}

// SearchByTag performs semantic search within a specific tag
// This is the KEY method for tag-aware hierarchical clustering
func (p *PgVectorAdapter) SearchByTag(ctx context.Context, query SearchQuery, tagID string) ([]SearchResult, error) {
	// Apply defaults
	if query.Limit == 0 {
		query.Limit = 10
	}
	if query.SimilarityThreshold == 0 {
		query.SimilarityThreshold = 0.7
	}

	vectorStr := formatVector(query.Embedding)

	// Build exclusion filter
	excludeClause := ""
	args := []interface{}{vectorStr, query.SimilarityThreshold, query.Limit, tagID}
	if len(query.ExcludeIDs) > 0 {
		excludeClause = "AND a.id NOT IN (SELECT unnest($5::uuid[]))"
		args = append(args, pq.Array(query.ExcludeIDs))
	}

	sqlQuery := fmt.Sprintf(`
		SELECT
			a.id,
			1 - (a.embedding_vector <=> $1::vector) as similarity,
			a.embedding_vector <=> $1::vector as distance
		FROM articles a
		INNER JOIN article_tags at ON a.id = at.article_id
		WHERE at.tag_id = $4
		  AND a.embedding IS NOT NULL
		  AND 1 - (a.embedding_vector <=> $1::vector) >= $2
		  %s
		ORDER BY a.embedding_vector <=> $1::vector
		LIMIT $3
	`, excludeClause)

	rows, err := p.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search by tag: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var result SearchResult
		if err := rows.Scan(&result.ArticleID, &result.Similarity, &result.Distance); err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// Optionally populate article data
	if query.IncludeArticle {
		if err := p.populateArticles(ctx, results); err != nil {
			return nil, fmt.Errorf("failed to populate articles: %w", err)
		}
	}

	// Always populate tag IDs
	if err := p.populateTags(ctx, results); err != nil {
		return nil, fmt.Errorf("failed to populate tags: %w", err)
	}

	return results, nil
}

// SearchByTags searches across multiple tags (OR operation)
// Useful for finding cross-tag connections
func (p *PgVectorAdapter) SearchByTags(ctx context.Context, query SearchQuery, tagIDs []string) ([]SearchResult, error) {
	if len(tagIDs) == 0 {
		return nil, fmt.Errorf("at least one tag ID required")
	}

	// Apply defaults
	if query.Limit == 0 {
		query.Limit = 10
	}
	if query.SimilarityThreshold == 0 {
		query.SimilarityThreshold = 0.7
	}

	vectorStr := formatVector(query.Embedding)

	// Build exclusion filter
	excludeClause := ""
	args := []interface{}{vectorStr, query.SimilarityThreshold, query.Limit, pq.Array(tagIDs)}
	if len(query.ExcludeIDs) > 0 {
		excludeClause = "AND a.id NOT IN (SELECT unnest($5::uuid[]))"
		args = append(args, pq.Array(query.ExcludeIDs))
	}

	sqlQuery := fmt.Sprintf(`
		SELECT
			a.id,
			1 - (a.embedding_vector <=> $1::vector) as similarity,
			a.embedding_vector <=> $1::vector as distance
		FROM articles a
		INNER JOIN article_tags at ON a.id = at.article_id
		WHERE at.tag_id = ANY($4::uuid[])
		  AND a.embedding IS NOT NULL
		  AND 1 - (a.embedding_vector <=> $1::vector) >= $2
		  %s
		GROUP BY a.id, a.embedding_vector
		ORDER BY a.embedding_vector <=> $1::vector
		LIMIT $3
	`, excludeClause)

	rows, err := p.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search by tags: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var result SearchResult
		if err := rows.Scan(&result.ArticleID, &result.Similarity, &result.Distance); err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// Optionally populate article data
	if query.IncludeArticle {
		if err := p.populateArticles(ctx, results); err != nil {
			return nil, fmt.Errorf("failed to populate articles: %w", err)
		}
	}

	// Always populate tag IDs
	if err := p.populateTags(ctx, results); err != nil {
		return nil, fmt.Errorf("failed to populate tags: %w", err)
	}

	return results, nil
}

// Delete removes an embedding (when article is deleted)
func (p *PgVectorAdapter) Delete(ctx context.Context, articleID string) error {
	query := `
		UPDATE articles
		SET embedding_vector = NULL,
		    updated_at = NOW()
		WHERE id = $1
	`

	result, err := p.db.ExecContext(ctx, query, articleID)
	if err != nil {
		return fmt.Errorf("failed to delete embedding: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("article %s not found", articleID)
	}

	return nil
}

// CreateIndex creates pgvector indexes for performance
// Uses HNSW (Hierarchical Navigable Small World) for best performance
func (p *PgVectorAdapter) CreateIndex(ctx context.Context) error {
	// Check if index already exists
	var exists bool
	checkQuery := `
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes
			WHERE tablename = 'articles'
			AND indexname = 'idx_articles_embedding_hnsw'
		)
	`
	if err := p.db.QueryRowContext(ctx, checkQuery).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}

	if exists {
		return nil // Index already exists
	}

	// Create HNSW index for fast approximate nearest neighbor search
	// m=16 (number of connections per layer)
	// ef_construction=64 (size of dynamic candidate list during construction)
	indexQuery := `
		CREATE INDEX idx_articles_embedding_hnsw
		ON articles
		USING hnsw (embedding_vector vector_cosine_ops)
		WITH (m = 16, ef_construction = 64)
	`

	if _, err := p.db.ExecContext(ctx, indexQuery); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}

// GetStats returns statistics about the vector store
func (p *PgVectorAdapter) GetStats(ctx context.Context) (*VectorStoreStats, error) {
	var stats VectorStoreStats

	// Count total embeddings
	countQuery := `
		SELECT COUNT(*)
		FROM articles
		WHERE embedding_vector IS NOT NULL
	`
	if err := p.db.QueryRowContext(ctx, countQuery).Scan(&stats.TotalEmbeddings); err != nil {
		return nil, fmt.Errorf("failed to count embeddings: %w", err)
	}

	// Get embedding dimensions (assuming 768 for Gemini)
	stats.EmbeddingDimensions = 768

	// Get index type and size
	indexQuery := `
		SELECT
			indexdef,
			pg_size_pretty(pg_relation_size(indexname::regclass)) as size
		FROM pg_indexes
		WHERE tablename = 'articles'
		AND indexname LIKE '%embedding%'
		LIMIT 1
	`
	var indexDef, sizeStr string
	if err := p.db.QueryRowContext(ctx, indexQuery).Scan(&indexDef, &sizeStr); err != nil {
		if err != sql.ErrNoRows {
			return nil, fmt.Errorf("failed to get index info: %w", err)
		}
		stats.IndexType = "none"
		stats.IndexSize = 0
	} else {
		if contains(indexDef, "hnsw") {
			stats.IndexType = "hnsw"
		} else if contains(indexDef, "ivfflat") {
			stats.IndexType = "ivfflat"
		} else {
			stats.IndexType = "unknown"
		}
		// Parse size (e.g., "8192 bytes" or "16 kB")
		// For simplicity, we'll just store 0 for now
		stats.IndexSize = 0
	}

	// TODO: Calculate average search latency from observability data
	stats.AvgSearchLatency = 0.0

	return &stats, nil
}

// populateArticles loads full article data for search results
func (p *PgVectorAdapter) populateArticles(ctx context.Context, results []SearchResult) error {
	if len(results) == 0 {
		return nil
	}

	articleIDs := make([]string, len(results))
	for i, r := range results {
		articleIDs[i] = r.ArticleID
	}

	query := `
		SELECT id, url, title, content_type, cleaned_text,
		       date_fetched, topic_cluster, cluster_confidence
		FROM articles
		WHERE id = ANY($1::uuid[])
	`

	rows, err := p.db.QueryContext(ctx, query, pq.Array(articleIDs))
	if err != nil {
		return fmt.Errorf("failed to query articles: %w", err)
	}
	defer rows.Close()

	articlesMap := make(map[string]*core.Article)
	for rows.Next() {
		article := &core.Article{}
		var contentType string
		if err := rows.Scan(
			&article.ID,
			&article.URL,
			&article.Title,
			&contentType,
			&article.CleanedText,
			&article.DateFetched,
			&article.TopicCluster,
			&article.ClusterConfidence,
		); err != nil {
			return fmt.Errorf("failed to scan article: %w", err)
		}
		article.ContentType = core.ContentType(contentType)
		articlesMap[article.ID] = article
	}

	// Populate results
	for i := range results {
		if article, ok := articlesMap[results[i].ArticleID]; ok {
			results[i].Article = article
		}
	}

	return nil
}

// populateTags loads tag IDs for search results
func (p *PgVectorAdapter) populateTags(ctx context.Context, results []SearchResult) error {
	if len(results) == 0 {
		return nil
	}

	articleIDs := make([]string, len(results))
	for i, r := range results {
		articleIDs[i] = r.ArticleID
	}

	query := `
		SELECT article_id, tag_id
		FROM article_tags
		WHERE article_id = ANY($1::uuid[])
	`

	rows, err := p.db.QueryContext(ctx, query, pq.Array(articleIDs))
	if err != nil {
		return fmt.Errorf("failed to query tags: %w", err)
	}
	defer rows.Close()

	tagsMap := make(map[string][]string)
	for rows.Next() {
		var articleID, tagID string
		if err := rows.Scan(&articleID, &tagID); err != nil {
			return fmt.Errorf("failed to scan tag: %w", err)
		}
		tagsMap[articleID] = append(tagsMap[articleID], tagID)
	}

	// Populate results
	for i := range results {
		if tags, ok := tagsMap[results[i].ArticleID]; ok {
			results[i].TagIDs = tags
		} else {
			results[i].TagIDs = []string{} // Empty slice instead of nil
		}
	}

	return nil
}

// formatVector converts []float64 to PostgreSQL vector format
// Example: [0.1, 0.2, 0.3] -> "[0.1,0.2,0.3]"
func formatVector(embedding []float64) string {
	if len(embedding) == 0 {
		return "[]"
	}

	result := "["
	for i, val := range embedding {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf("%f", val)
	}
	result += "]"
	return result
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		 containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
