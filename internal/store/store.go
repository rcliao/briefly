package store

import (
	"briefly/internal/core"
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	_ "github.com/mattn/go-sqlite3"
)

// Store represents the SQLite-based caching store
type Store struct {
	db   *sql.DB
	path string
}

// NewStore creates a new store instance with SQLite database
func NewStore(dataDir string) (*Store, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "briefly.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &Store{
		db:   db,
		path: dbPath,
	}

	if err := store.initialize(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return store, nil
}

// initialize creates the necessary tables and runs migrations
// initialize creates the necessary tables and runs migrations
func (s *Store) initialize() error {
	articlesTable := `
	CREATE TABLE IF NOT EXISTS articles (
		url TEXT PRIMARY KEY,
		title TEXT,
		content TEXT,
		html_content TEXT,
		my_take TEXT,
		date_fetched DATETIME,
		content_hash TEXT,
		metadata TEXT,
		embedding BLOB,
		topic_cluster TEXT,
		topic_confidence REAL
	);`

	summariesTable := `
	CREATE TABLE IF NOT EXISTS summaries (
		id TEXT PRIMARY KEY,
		article_url TEXT,
		summary_text TEXT,
		key_insights TEXT,
		action_items TEXT,
		model_used TEXT,
		date_generated DATETIME,
		content_hash TEXT,
		embedding BLOB,
		topic_cluster TEXT,
		topic_confidence REAL,
		FOREIGN KEY (article_url) REFERENCES articles (url)
	);`

	for _, table := range []string{articlesTable, summariesTable} {
		if _, err := s.db.Exec(table); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying database connection for advanced operations

func (s *Store) CacheArticle(article core.Article) error {
	// The url column is the primary key. Older code wrote article.LinkID here,
	// which is empty in the digest path — every article collapsed into one row
	// with an empty URL and lookups by real URL never hit.
	url := article.URL
	if url == "" {
		url = article.LinkID // legacy callers that only set LinkID
	}
	if url == "" {
		return fmt.Errorf("article has neither URL nor LinkID; refusing to cache")
	}

	metadata, _ := json.Marshal(map[string]any{
		"id":      article.ID,
		"link_id": article.LinkID,
	})

	// Serialize embedding
	embeddingData, err := serializeEmbedding(article.Embedding)
	if err != nil {
		return fmt.Errorf("failed to serialize embedding: %w", err)
	}

	query := `
	INSERT OR REPLACE INTO articles
	(url, title, content, html_content, my_take, date_fetched, content_hash, metadata, embedding, topic_cluster, topic_confidence)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = s.db.Exec(query,
		url,
		article.Title,
		article.CleanedText,
		article.FetchedHTML,
		article.MyTake,
		article.DateFetched,
		ContentHash(article.CleanedText),
		string(metadata),
		embeddingData,
		article.TopicCluster,
		article.TopicConfidence,
	)

	return err
}

// GetCachedArticle retrieves an article from the cache
func (s *Store) GetCachedArticle(url string, maxAge time.Duration) (*core.Article, error) {
	query := `
	SELECT url, title, content, html_content, my_take, date_fetched, metadata, embedding, topic_cluster, topic_confidence
	FROM articles
	WHERE url = ? AND date_fetched > ?`

	cutoff := time.Now().UTC().Add(-maxAge)
	row := s.db.QueryRow(query, url, cutoff)

	var article core.Article
	var dateFetched time.Time
	var metadata string
	var embeddingData []byte
	var topicCluster sql.NullString
	var topicConfidence sql.NullFloat64

	err := row.Scan(
		&article.URL,
		&article.Title,
		&article.CleanedText,
		&article.FetchedHTML,
		&article.MyTake,
		&dateFetched,
		&metadata,
		&embeddingData,
		&topicCluster,
		&topicConfidence,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan article: %w", err)
	}

	// Restore identity fields from metadata; callers key maps on article.ID,
	// so a cached article must never come back with an empty one.
	var meta struct {
		ID     string `json:"id"`
		LinkID string `json:"link_id"`
	}
	_ = json.Unmarshal([]byte(metadata), &meta)
	article.ID = meta.ID
	article.LinkID = meta.LinkID
	if article.ID == "" {
		article.ID = uuid.NewString()
	}

	// Deserialize embedding
	if embeddingData != nil {
		article.Embedding, err = deserializeEmbedding(embeddingData)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize embedding: %w", err)
		}
	}

	// Handle nullable fields
	if topicCluster.Valid {
		article.TopicCluster = topicCluster.String
	}
	if topicConfidence.Valid {
		article.TopicConfidence = topicConfidence.Float64
	}

	article.DateFetched = dateFetched
	return &article, nil
}

// CacheSummary stores a summary in the cache
func (s *Store) CacheSummary(summary core.Summary, articleURL string, contentHash string) error {
	// Serialize embedding
	embeddingData, err := serializeEmbedding(summary.Embedding)
	if err != nil {
		return fmt.Errorf("failed to serialize embedding: %w", err)
	}

	query := `
	INSERT OR REPLACE INTO summaries 
	(id, article_url, summary_text, key_insights, action_items, model_used, date_generated, content_hash, embedding, topic_cluster, topic_confidence)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	// Convert ArticleIDs to JSON for key_insights field (reusing the field for article references)
	articleIDs, _ := json.Marshal(summary.ArticleIDs)
	// Use Instructions as action_items (reusing the field)
	instructions := summary.Instructions

	_, err = s.db.Exec(query,
		summary.ID,
		articleURL,
		summary.SummaryText,
		string(articleIDs), // Store ArticleIDs in key_insights field
		instructions,       // Store Instructions in action_items field
		summary.ModelUsed,
		summary.DateGenerated,
		contentHash,
		embeddingData,
		summary.TopicCluster,
		summary.TopicConfidence,
	)

	return err
}

// GetCachedSummary retrieves a summary from the cache
func (s *Store) GetCachedSummary(articleURL string, contentHash string, maxAge time.Duration) (*core.Summary, error) {
	query := `
	SELECT id, summary_text, key_insights, action_items, model_used, date_generated, embedding, topic_cluster, topic_confidence
	FROM summaries 
	WHERE article_url = ? AND content_hash = ? AND date_generated > ?`

	cutoff := time.Now().UTC().Add(-maxAge)
	row := s.db.QueryRow(query, articleURL, contentHash, cutoff)

	var summary core.Summary
	var articleIDsJSON, instructions string
	var embeddingData []byte
	var topicCluster sql.NullString
	var topicConfidence sql.NullFloat64

	err := row.Scan(
		&summary.ID,
		&summary.SummaryText,
		&articleIDsJSON,
		&instructions,
		&summary.ModelUsed,
		&summary.DateGenerated,
		&embeddingData,
		&topicCluster,
		&topicConfidence,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan summary: %w", err)
	}

	// Deserialize embedding
	if embeddingData != nil {
		summary.Embedding, err = deserializeEmbedding(embeddingData)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize embedding: %w", err)
		}
	}

	// Handle nullable fields
	if topicCluster.Valid {
		summary.TopicCluster = topicCluster.String
	}
	if topicConfidence.Valid {
		summary.TopicConfidence = topicConfidence.Float64
	}

	// Unmarshal JSON fields
	_ = json.Unmarshal([]byte(articleIDsJSON), &summary.ArticleIDs)
	summary.Instructions = instructions

	return &summary, nil
}

// CacheDigest stores a generated digest
// CacheStats provides statistics about the cache
type CacheStats struct {
	ArticleCount int
	SummaryCount int
	CacheSize    int64
	LastUpdated  time.Time
}

// GetCacheStats returns statistics about cached articles and summaries
func (s *Store) GetCacheStats() (*CacheStats, error) {
	stats := &CacheStats{}

	if err := s.db.QueryRow("SELECT COUNT(*) FROM articles").Scan(&stats.ArticleCount); err != nil {
		return nil, fmt.Errorf("failed to count articles: %w", err)
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM summaries").Scan(&stats.SummaryCount); err != nil {
		return nil, fmt.Errorf("failed to count summaries: %w", err)
	}

	var lastUpdated sql.NullTime
	_ = s.db.QueryRow("SELECT MAX(date_fetched) FROM articles").Scan(&lastUpdated)
	if lastUpdated.Valid {
		stats.LastUpdated = lastUpdated.Time
	}

	if info, err := os.Stat(s.path); err == nil {
		stats.CacheSize = info.Size()
	}

	return stats, nil
}

// ClearCache removes all cached articles and summaries
func (s *Store) ClearCache() error {
	for _, table := range []string{"summaries", "articles"} {
		if _, err := s.db.Exec("DELETE FROM " + table); err != nil {
			return fmt.Errorf("failed to clear %s: %w", table, err)
		}
	}
	return nil
}

func ContentHash(content string) string {
	if len(content) == 0 {
		return "empty"
	}
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:8])
}

// serializeEmbedding converts a float64 slice to bytes for database storage
func serializeEmbedding(embedding []float64) ([]byte, error) {
	if embedding == nil {
		return nil, nil
	}

	buf := new(bytes.Buffer)
	for _, val := range embedding {
		if err := binary.Write(buf, binary.LittleEndian, val); err != nil {
			return nil, fmt.Errorf("failed to serialize embedding: %w", err)
		}
	}
	return buf.Bytes(), nil
}

// deserializeEmbedding converts bytes back to a float64 slice
func deserializeEmbedding(data []byte) ([]float64, error) {
	if data == nil {
		return nil, nil
	}

	buf := bytes.NewReader(data)
	var embedding []float64

	for buf.Len() > 0 {
		var val float64
		if err := binary.Read(buf, binary.LittleEndian, &val); err != nil {
			return nil, fmt.Errorf("failed to deserialize embedding: %w", err)
		}
		embedding = append(embedding, val)
	}

	return embedding, nil
}

// GetRecentArticles retrieves articles from the specified number of days ago
func (s *Store) SaveArticle(article *core.Article) error {
	return s.CacheArticle(*article)
}

// GetArticlesByDateRange retrieves articles within a specific date range
