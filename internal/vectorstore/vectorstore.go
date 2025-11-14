package vectorstore

import (
	"briefly/internal/core"
	"context"
)

// VectorStore provides semantic search operations for article embeddings
// Using pgvector for production-scale similarity search with cosine distance
type VectorStore interface {
	// Store saves or updates an embedding for an article
	// Returns error if article doesn't exist or embedding is invalid
	Store(ctx context.Context, articleID string, embedding []float64) error

	// Search finds articles similar to the query embedding
	// Uses cosine similarity (1 - cosine distance) for ranking
	// Returns results ordered by similarity (highest first)
	Search(ctx context.Context, query SearchQuery) ([]SearchResult, error)

	// SearchByTag performs semantic search within a specific tag
	// Combines tag filtering with cosine similarity for hierarchical search
	// This is the key method for tag-aware clustering
	SearchByTag(ctx context.Context, query SearchQuery, tagID string) ([]SearchResult, error)

	// SearchByTags searches across multiple tags (OR operation)
	// Useful for finding cross-tag connections
	SearchByTags(ctx context.Context, query SearchQuery, tagIDs []string) ([]SearchResult, error)

	// Delete removes an embedding (when article is deleted)
	Delete(ctx context.Context, articleID string) error

	// CreateIndex creates pgvector indexes for performance
	// Should be called after bulk inserts
	CreateIndex(ctx context.Context) error

	// GetStats returns statistics about the vector store
	GetStats(ctx context.Context) (*VectorStoreStats, error)
}

// SearchQuery configures semantic search parameters
type SearchQuery struct {
	// Embedding is the query vector (768-dim for Gemini)
	Embedding []float64

	// Limit is the maximum number of results to return (default: 10)
	Limit int

	// SimilarityThreshold is the minimum cosine similarity (0.0-1.0, default: 0.7)
	// Higher values = more strict matching
	SimilarityThreshold float64

	// IncludeArticle populates the Article field in results (default: false)
	// Set to true when you need full article data, not just IDs
	IncludeArticle bool

	// ExcludeIDs filters out specific articles (useful for "more like this" queries)
	ExcludeIDs []string
}

// SearchResult contains a similar article and its similarity score
type SearchResult struct {
	// ArticleID is the unique identifier
	ArticleID string

	// Similarity is the cosine similarity (0.0-1.0, higher = more similar)
	Similarity float64

	// Article is the full article data (only populated if IncludeArticle=true)
	Article *core.Article

	// TagIDs are the tags assigned to this article
	TagIDs []string

	// Distance is the raw cosine distance (lower = more similar)
	// Similarity = 1 - Distance
	Distance float64
}

// VectorStoreStats provides metrics about the vector store
type VectorStoreStats struct {
	// TotalEmbeddings is the count of stored embeddings
	TotalEmbeddings int64

	// EmbeddingDimensions is the vector size (should be 768 for Gemini)
	EmbeddingDimensions int

	// IndexType describes the pgvector index (e.g., "ivfflat", "hnsw")
	IndexType string

	// IndexSize is the disk space used by indexes
	IndexSize int64

	// AvgSearchLatency is the average search query time in milliseconds
	AvgSearchLatency float64
}

// DefaultSearchQuery returns sensible defaults
func DefaultSearchQuery(embedding []float64) SearchQuery {
	return SearchQuery{
		Embedding:           embedding,
		Limit:               10,
		SimilarityThreshold: 0.7,
		IncludeArticle:      false,
		ExcludeIDs:          []string{},
	}
}
