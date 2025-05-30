package store

import (
	"briefly/internal/core"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

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
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return store, nil
}

// initialize creates the necessary tables
func (s *Store) initialize() error {
	// Create articles table for caching fetched articles
	articlesTable := `
	CREATE TABLE IF NOT EXISTS articles (
		url TEXT PRIMARY KEY,
		title TEXT,
		content TEXT,
		html_content TEXT,
		my_take TEXT,
		date_fetched DATETIME,
		content_hash TEXT,
		metadata TEXT
	);`

	// Create summaries table for caching LLM summaries
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
		FOREIGN KEY (article_url) REFERENCES articles (url)
	);`

	// Create digests table for storing generated digests
	digestsTable := `
	CREATE TABLE IF NOT EXISTS digests (
		id TEXT PRIMARY KEY,
		title TEXT,
		content TEXT,
		digest_summary TEXT,
		article_urls TEXT,
		date_generated DATETIME,
		model_used TEXT
	);`

	tables := []string{articlesTable, summariesTable, digestsTable}
	for _, table := range tables {
		if _, err := s.db.Exec(table); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// CacheArticle stores an article in the cache
func (s *Store) CacheArticle(article core.Article) error {
	metadata, _ := json.Marshal(map[string]interface{}{
		"link_id": article.LinkID,
	})

	query := `
	INSERT OR REPLACE INTO articles 
	(url, title, content, html_content, my_take, date_fetched, content_hash, metadata)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query,
		article.LinkID, // Use LinkID as URL identifier
		article.Title,
		article.CleanedText,
		article.FetchedHTML,
		article.MyTake,
		article.DateFetched,
		generateContentHash(article.CleanedText),
		string(metadata),
	)

	return err
}

// GetCachedArticle retrieves an article from the cache
func (s *Store) GetCachedArticle(url string, maxAge time.Duration) (*core.Article, error) {
	query := `
	SELECT url, title, content, html_content, my_take, date_fetched, metadata
	FROM articles 
	WHERE url = ? AND date_fetched > ?`

	cutoff := time.Now().UTC().Add(-maxAge)
	row := s.db.QueryRow(query, url, cutoff)

	var article core.Article
	var dateFetched time.Time
	var metadata string

	err := row.Scan(
		&article.LinkID, // Use LinkID as URL identifier
		&article.Title,
		&article.CleanedText,
		&article.FetchedHTML,
		&article.MyTake,
		&dateFetched,
		&metadata,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan article: %w", err)
	}

	article.DateFetched = dateFetched
	return &article, nil
}

// CacheSummary stores a summary in the cache
func (s *Store) CacheSummary(summary core.Summary, articleURL string, contentHash string) error {
	query := `
	INSERT OR REPLACE INTO summaries 
	(id, article_url, summary_text, key_insights, action_items, model_used, date_generated, content_hash)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	// Convert ArticleIDs to JSON for key_insights field (reusing the field for article references)
	articleIDs, _ := json.Marshal(summary.ArticleIDs)
	// Use Instructions as action_items (reusing the field)
	instructions := summary.Instructions

	_, err := s.db.Exec(query,
		summary.ID,
		articleURL,
		summary.SummaryText,
		string(articleIDs), // Store ArticleIDs in key_insights field
		instructions,       // Store Instructions in action_items field
		summary.ModelUsed,
		summary.DateGenerated,
		contentHash,
	)

	return err
}

// GetCachedSummary retrieves a summary from the cache
func (s *Store) GetCachedSummary(articleURL string, contentHash string, maxAge time.Duration) (*core.Summary, error) {
	query := `
	SELECT id, summary_text, key_insights, action_items, model_used, date_generated
	FROM summaries 
	WHERE article_url = ? AND content_hash = ? AND date_generated > ?`

	cutoff := time.Now().UTC().Add(-maxAge)
	row := s.db.QueryRow(query, articleURL, contentHash, cutoff)

	var summary core.Summary
	var articleIDsJSON, instructions string

	err := row.Scan(
		&summary.ID,
		&summary.SummaryText,
		&articleIDsJSON,
		&instructions,
		&summary.ModelUsed,
		&summary.DateGenerated,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan summary: %w", err)
	}

	// Unmarshal JSON fields
	json.Unmarshal([]byte(articleIDsJSON), &summary.ArticleIDs)
	summary.Instructions = instructions

	return &summary, nil
}

// CacheDigest stores a generated digest
func (s *Store) CacheDigest(digestID, title, content, digestSummary string, articleURLs []string, modelUsed string) error {
	urlsJSON, _ := json.Marshal(articleURLs)

	query := `
	INSERT OR REPLACE INTO digests 
	(id, title, content, digest_summary, article_urls, date_generated, model_used)
	VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query,
		digestID,
		title,
		content,
		digestSummary,
		string(urlsJSON),
		time.Now().UTC(),
		modelUsed,
	)

	return err
}

// CacheStats represents cache statistics
type CacheStats struct {
	ArticleCount  int
	SummaryCount  int
	DigestCount   int
	CacheSize     int64
	LastUpdated   time.Time
}

// GetCacheStats returns statistics about the cache
func (s *Store) GetCacheStats() (*CacheStats, error) {
	stats := &CacheStats{}

	// Get counts
	queries := map[string]*int{
		"SELECT COUNT(*) FROM articles":  &stats.ArticleCount,
		"SELECT COUNT(*) FROM summaries": &stats.SummaryCount,
		"SELECT COUNT(*) FROM digests":   &stats.DigestCount,
	}

	for query, target := range queries {
		err := s.db.QueryRow(query).Scan(target)
		if err != nil {
			return nil, fmt.Errorf("failed to get count: %w", err)
		}
	}

	// Get cache size (file size)
	if fileInfo, err := os.Stat(s.path); err == nil {
		stats.CacheSize = fileInfo.Size()
		stats.LastUpdated = fileInfo.ModTime()
	}

	return stats, nil
}

// ClearCache removes all cached data
func (s *Store) ClearCache() error {
	tables := []string{"articles", "summaries", "digests"}
	
	for _, table := range tables {
		_, err := s.db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			return fmt.Errorf("failed to clear %s table: %w", table, err)
		}
	}
	
	// Vacuum to reclaim space
	_, err := s.db.Exec("VACUUM")
	if err != nil {
		return fmt.Errorf("failed to vacuum database: %w", err)
	}
	
	return nil
}

// CleanupOldCache removes old cached items
func (s *Store) CleanupOldCache(articleMaxAge, summaryMaxAge time.Duration) error {
	now := time.Now().UTC()

	// Clean old articles
	_, err := s.db.Exec("DELETE FROM articles WHERE date_fetched < ?", now.Add(-articleMaxAge))
	if err != nil {
		return fmt.Errorf("failed to clean old articles: %w", err)
	}

	// Clean old summaries
	_, err = s.db.Exec("DELETE FROM summaries WHERE date_generated < ?", now.Add(-summaryMaxAge))
	if err != nil {
		return fmt.Errorf("failed to clean old summaries: %w", err)
	}

	return nil
}

// generateContentHash creates a simple hash of content for cache validation
func generateContentHash(content string) string {
	// Simple hash based on content length and first/last chars
	if len(content) == 0 {
		return "empty"
	}
	return fmt.Sprintf("%d-%c-%c", len(content), content[0], content[len(content)-1])
}