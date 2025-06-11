package store

import (
	"briefly/internal/core"
	"bytes"
	"database/sql"
	"encoding/binary"
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
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return store, nil
}

// initialize creates the necessary tables and runs migrations
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
		metadata TEXT,
		embedding BLOB,
		topic_cluster TEXT,
		topic_confidence REAL
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
		embedding BLOB,
		topic_cluster TEXT,
		topic_confidence REAL,
		FOREIGN KEY (article_url) REFERENCES articles (url)
	);`

	// Create digests table for storing generated digests
	digestsTable := `
	CREATE TABLE IF NOT EXISTS digests (
		id TEXT PRIMARY KEY,
		title TEXT,
		content TEXT,
		digest_summary TEXT,
		my_take TEXT,
		format TEXT,
		article_urls TEXT,
		date_generated DATETIME,
		model_used TEXT
	);`

	// Create RSS feeds table for managing RSS/Atom feeds
	feedsTable := `
	CREATE TABLE IF NOT EXISTS feeds (
		id TEXT PRIMARY KEY,
		url TEXT UNIQUE NOT NULL,
		title TEXT,
		description TEXT,
		last_fetched DATETIME,
		last_modified TEXT,
		etag TEXT,
		active BOOLEAN DEFAULT TRUE,
		error_count INTEGER DEFAULT 0,
		last_error TEXT,
		date_added DATETIME
	);`

	// Create feed items table for tracking discovered links
	feedItemsTable := `
	CREATE TABLE IF NOT EXISTS feed_items (
		id TEXT PRIMARY KEY,
		feed_id TEXT,
		title TEXT,
		link TEXT UNIQUE NOT NULL,
		description TEXT,
		published DATETIME,
		guid TEXT,
		processed BOOLEAN DEFAULT FALSE,
		date_discovered DATETIME,
		FOREIGN KEY (feed_id) REFERENCES feeds (id)
	);`

	tables := []string{articlesTable, summariesTable, digestsTable, feedsTable, feedItemsTable}
	for _, table := range tables {
		if _, err := s.db.Exec(table); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Run migrations
	if err := s.runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// runMigrations handles database schema updates
func (s *Store) runMigrations() error {
	// Check if my_take column exists in digests table, add if not
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('digests') WHERE name='my_take'").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check digests schema: %w", err)
	}

	if count == 0 {
		_, err = s.db.Exec("ALTER TABLE digests ADD COLUMN my_take TEXT DEFAULT ''")
		if err != nil {
			return fmt.Errorf("failed to add my_take column to digests: %w", err)
		}
	}

	// Check if format column exists in digests table, add if not
	err = s.db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('digests') WHERE name='format'").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check digests schema for format: %w", err)
	}

	if count == 0 {
		_, err = s.db.Exec("ALTER TABLE digests ADD COLUMN format TEXT DEFAULT 'standard'")
		if err != nil {
			return fmt.Errorf("failed to add format column to digests: %w", err)
		}
	}

	// Add embedding columns to articles table if they don't exist
	err = s.db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('articles') WHERE name='embedding'").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check articles schema for embedding: %w", err)
	}

	if count == 0 {
		_, err = s.db.Exec("ALTER TABLE articles ADD COLUMN embedding BLOB")
		if err != nil {
			return fmt.Errorf("failed to add embedding column to articles: %w", err)
		}
		_, err = s.db.Exec("ALTER TABLE articles ADD COLUMN topic_cluster TEXT")
		if err != nil {
			return fmt.Errorf("failed to add topic_cluster column to articles: %w", err)
		}
		_, err = s.db.Exec("ALTER TABLE articles ADD COLUMN topic_confidence REAL")
		if err != nil {
			return fmt.Errorf("failed to add topic_confidence column to articles: %w", err)
		}
	}

	// Add embedding columns to summaries table if they don't exist
	err = s.db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('summaries') WHERE name='embedding'").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check summaries schema for embedding: %w", err)
	}

	if count == 0 {
		_, err = s.db.Exec("ALTER TABLE summaries ADD COLUMN embedding BLOB")
		if err != nil {
			return fmt.Errorf("failed to add embedding column to summaries: %w", err)
		}
		_, err = s.db.Exec("ALTER TABLE summaries ADD COLUMN topic_cluster TEXT")
		if err != nil {
			return fmt.Errorf("failed to add topic_cluster column to summaries: %w", err)
		}
		_, err = s.db.Exec("ALTER TABLE summaries ADD COLUMN topic_confidence REAL")
		if err != nil {
			return fmt.Errorf("failed to add topic_confidence column to summaries: %w", err)
		}
	}

	// Add v0.4 insights columns to articles table if they don't exist
	err = s.db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('articles') WHERE name='sentiment_score'").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check articles schema for insights: %w", err)
	}

	if count == 0 {
		_, err = s.db.Exec("ALTER TABLE articles ADD COLUMN sentiment_score REAL DEFAULT 0.0")
		if err != nil {
			return fmt.Errorf("failed to add sentiment_score column to articles: %w", err)
		}
		_, err = s.db.Exec("ALTER TABLE articles ADD COLUMN sentiment_label TEXT DEFAULT 'neutral'")
		if err != nil {
			return fmt.Errorf("failed to add sentiment_label column to articles: %w", err)
		}
		_, err = s.db.Exec("ALTER TABLE articles ADD COLUMN sentiment_emoji TEXT DEFAULT '😐'")
		if err != nil {
			return fmt.Errorf("failed to add sentiment_emoji column to articles: %w", err)
		}
		_, err = s.db.Exec("ALTER TABLE articles ADD COLUMN alert_triggered BOOLEAN DEFAULT FALSE")
		if err != nil {
			return fmt.Errorf("failed to add alert_triggered column to articles: %w", err)
		}
		_, err = s.db.Exec("ALTER TABLE articles ADD COLUMN alert_conditions TEXT DEFAULT ''")
		if err != nil {
			return fmt.Errorf("failed to add alert_conditions column to articles: %w", err)
		}
		_, err = s.db.Exec("ALTER TABLE articles ADD COLUMN research_queries TEXT DEFAULT ''")
		if err != nil {
			return fmt.Errorf("failed to add research_queries column to articles: %w", err)
		}
	}

	// Add v0.4 insights columns to digests table if they don't exist
	err = s.db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('digests') WHERE name='overall_sentiment'").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check digests schema for insights: %w", err)
	}

	if count == 0 {
		_, err = s.db.Exec("ALTER TABLE digests ADD COLUMN overall_sentiment TEXT DEFAULT 'neutral'")
		if err != nil {
			return fmt.Errorf("failed to add overall_sentiment column to digests: %w", err)
		}
		_, err = s.db.Exec("ALTER TABLE digests ADD COLUMN alerts_summary TEXT DEFAULT ''")
		if err != nil {
			return fmt.Errorf("failed to add alerts_summary column to digests: %w", err)
		}
		_, err = s.db.Exec("ALTER TABLE digests ADD COLUMN trends_summary TEXT DEFAULT ''")
		if err != nil {
			return fmt.Errorf("failed to add trends_summary column to digests: %w", err)
		}
		_, err = s.db.Exec("ALTER TABLE digests ADD COLUMN research_suggestions TEXT DEFAULT ''")
		if err != nil {
			return fmt.Errorf("failed to add research_suggestions column to digests: %w", err)
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

	// Serialize embedding
	embeddingData, err := serializeEmbedding(article.Embedding)
	if err != nil {
		return fmt.Errorf("failed to serialize embedding: %w", err)
	}

	// Serialize alert conditions and research queries
	alertConditionsJSON, _ := json.Marshal(article.AlertConditions)
	researchQueriesJSON, _ := json.Marshal(article.ResearchQueries)

	query := `
	INSERT OR REPLACE INTO articles 
	(url, title, content, html_content, my_take, date_fetched, content_hash, metadata, embedding, topic_cluster, topic_confidence,
	 sentiment_score, sentiment_label, sentiment_emoji, alert_triggered, alert_conditions, research_queries)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = s.db.Exec(query,
		article.LinkID, // Use LinkID as URL identifier
		article.Title,
		article.CleanedText,
		article.FetchedHTML,
		article.MyTake,
		article.DateFetched,
		generateContentHash(article.CleanedText),
		string(metadata),
		embeddingData,
		article.TopicCluster,
		article.TopicConfidence,
		article.SentimentScore,
		article.SentimentLabel,
		article.SentimentEmoji,
		article.AlertTriggered,
		string(alertConditionsJSON),
		string(researchQueriesJSON),
	)

	return err
}

// GetCachedArticle retrieves an article from the cache
func (s *Store) GetCachedArticle(url string, maxAge time.Duration) (*core.Article, error) {
	query := `
	SELECT url, title, content, html_content, my_take, date_fetched, metadata, embedding, topic_cluster, topic_confidence,
	       sentiment_score, sentiment_label, sentiment_emoji, alert_triggered, alert_conditions, research_queries
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
	var sentimentScore sql.NullFloat64
	var sentimentLabel sql.NullString
	var sentimentEmoji sql.NullString
	var alertTriggered sql.NullBool
	var alertConditionsJSON sql.NullString
	var researchQueriesJSON sql.NullString

	err := row.Scan(
		&article.LinkID, // Use LinkID as URL identifier
		&article.Title,
		&article.CleanedText,
		&article.FetchedHTML,
		&article.MyTake,
		&dateFetched,
		&metadata,
		&embeddingData,
		&topicCluster,
		&topicConfidence,
		&sentimentScore,
		&sentimentLabel,
		&sentimentEmoji,
		&alertTriggered,
		&alertConditionsJSON,
		&researchQueriesJSON,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan article: %w", err)
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

	// Handle insights fields
	if sentimentScore.Valid {
		article.SentimentScore = sentimentScore.Float64
	}
	if sentimentLabel.Valid {
		article.SentimentLabel = sentimentLabel.String
	}
	if sentimentEmoji.Valid {
		article.SentimentEmoji = sentimentEmoji.String
	}
	if alertTriggered.Valid {
		article.AlertTriggered = alertTriggered.Bool
	}
	if alertConditionsJSON.Valid && alertConditionsJSON.String != "" {
		_ = json.Unmarshal([]byte(alertConditionsJSON.String), &article.AlertConditions)
	}
	if researchQueriesJSON.Valid && researchQueriesJSON.String != "" {
		_ = json.Unmarshal([]byte(researchQueriesJSON.String), &article.ResearchQueries)
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
func (s *Store) CacheDigest(digestID, title, content, digestSummary string, articleURLs []string, modelUsed string) error {
	urlsJSON, _ := json.Marshal(articleURLs)

	query := `
	INSERT OR REPLACE INTO digests 
	(id, title, content, digest_summary, my_take, format, article_urls, date_generated, model_used)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query,
		digestID,
		title,
		content,
		digestSummary,
		"",         // my_take starts empty
		"standard", // default format
		string(urlsJSON),
		time.Now().UTC(),
		modelUsed,
	)

	return err
}

// CacheDigestWithFormat stores a generated digest with format
func (s *Store) CacheDigestWithFormat(digestID, title, content, digestSummary, format string, articleURLs []string, modelUsed string) error {
	urlsJSON, _ := json.Marshal(articleURLs)

	query := `
	INSERT OR REPLACE INTO digests 
	(id, title, content, digest_summary, my_take, format, article_urls, date_generated, model_used,
	 overall_sentiment, alerts_summary, trends_summary, research_suggestions)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query,
		digestID,
		title,
		content,
		digestSummary,
		"", // my_take starts empty
		format,
		string(urlsJSON),
		time.Now().UTC(),
		modelUsed,
		"neutral", // default overall_sentiment
		"",        // empty alerts_summary
		"",        // empty trends_summary
		"[]",      // empty research_suggestions array
	)

	return err
}

// GetCachedDigest retrieves a digest from the cache
func (s *Store) GetCachedDigest(digestID string) (*core.Digest, error) {
	query := `
	SELECT id, title, content, digest_summary, my_take, format, article_urls, date_generated, model_used,
	       overall_sentiment, alerts_summary, trends_summary, research_suggestions
	FROM digests 
	WHERE id = ?`

	var digest core.Digest
	var urlsJSON string
	var overallSentiment sql.NullString
	var alertsSummary sql.NullString
	var trendsSummary sql.NullString
	var researchSuggestionsJSON sql.NullString

	err := s.db.QueryRow(query, digestID).Scan(
		&digest.ID,
		&digest.Title,
		&digest.Content,
		&digest.DigestSummary,
		&digest.MyTake,
		&digest.Format,
		&urlsJSON,
		&digest.DateGenerated,
		&digest.ModelUsed,
		&overallSentiment,
		&alertsSummary,
		&trendsSummary,
		&researchSuggestionsJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Digest not found
		}
		return nil, fmt.Errorf("failed to get cached digest: %w", err)
	}

	// Parse article URLs
	_ = json.Unmarshal([]byte(urlsJSON), &digest.ArticleURLs)

	// Handle insights fields
	if overallSentiment.Valid {
		digest.OverallSentiment = overallSentiment.String
	}
	if alertsSummary.Valid {
		digest.AlertsSummary = alertsSummary.String
	}
	if trendsSummary.Valid {
		digest.TrendsSummary = trendsSummary.String
	}
	if researchSuggestionsJSON.Valid && researchSuggestionsJSON.String != "" {
		_ = json.Unmarshal([]byte(researchSuggestionsJSON.String), &digest.ResearchSuggestions)
	}

	return &digest, nil
}

// UpdateDigestMyTake updates the my_take field for a digest
func (s *Store) UpdateDigestMyTake(digestID, myTake string) error {
	query := `UPDATE digests SET my_take = ? WHERE id = ?`
	_, err := s.db.Exec(query, myTake, digestID)
	return err
}

// GetLatestDigests retrieves the most recent digests
func (s *Store) GetLatestDigests(limit int) ([]core.Digest, error) {
	query := `
	SELECT id, title, content, digest_summary, my_take, format, article_urls, date_generated, model_used,
	       overall_sentiment, alerts_summary, trends_summary, research_suggestions
	FROM digests 
	ORDER BY date_generated DESC
	LIMIT ?`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query digests: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var digests []core.Digest
	for rows.Next() {
		var digest core.Digest
		var urlsJSON string
		var overallSentiment sql.NullString
		var alertsSummary sql.NullString
		var trendsSummary sql.NullString
		var researchSuggestionsJSON sql.NullString

		err := rows.Scan(
			&digest.ID,
			&digest.Title,
			&digest.Content,
			&digest.DigestSummary,
			&digest.MyTake,
			&digest.Format,
			&urlsJSON,
			&digest.DateGenerated,
			&digest.ModelUsed,
			&overallSentiment,
			&alertsSummary,
			&trendsSummary,
			&researchSuggestionsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan digest row: %w", err)
		}

		// Parse article URLs
		_ = json.Unmarshal([]byte(urlsJSON), &digest.ArticleURLs)

		// Handle insights fields
		if overallSentiment.Valid {
			digest.OverallSentiment = overallSentiment.String
		}
		if alertsSummary.Valid {
			digest.AlertsSummary = alertsSummary.String
		}
		if trendsSummary.Valid {
			digest.TrendsSummary = trendsSummary.String
		}
		if researchSuggestionsJSON.Valid && researchSuggestionsJSON.String != "" {
			_ = json.Unmarshal([]byte(researchSuggestionsJSON.String), &digest.ResearchSuggestions)
		}

		digests = append(digests, digest)
	}

	return digests, nil
}

// FindDigestByPartialID finds a digest by partial ID match
func (s *Store) FindDigestByPartialID(partialID string) (*core.Digest, error) {
	// First try exact match
	digest, err := s.GetCachedDigest(partialID)
	if err != nil {
		return nil, err
	}
	if digest != nil {
		return digest, nil
	}

	// Try partial match if not found
	query := `
	SELECT id, title, content, digest_summary, my_take, format, article_urls, date_generated, model_used,
	       overall_sentiment, alerts_summary, trends_summary, research_suggestions
	FROM digests 
	WHERE id LIKE ? || '%'
	ORDER BY date_generated DESC
	LIMIT 1`

	var foundDigest core.Digest
	var urlsJSON string
	var overallSentiment sql.NullString
	var alertsSummary sql.NullString
	var trendsSummary sql.NullString
	var researchSuggestionsJSON sql.NullString

	err = s.db.QueryRow(query, partialID).Scan(
		&foundDigest.ID,
		&foundDigest.Title,
		&foundDigest.Content,
		&foundDigest.DigestSummary,
		&foundDigest.MyTake,
		&foundDigest.Format,
		&urlsJSON,
		&foundDigest.DateGenerated,
		&foundDigest.ModelUsed,
		&overallSentiment,
		&alertsSummary,
		&trendsSummary,
		&researchSuggestionsJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Digest not found
		}
		return nil, fmt.Errorf("failed to find digest by partial ID: %w", err)
	}

	// Parse article URLs
	_ = json.Unmarshal([]byte(urlsJSON), &foundDigest.ArticleURLs)

	// Handle insights fields
	if overallSentiment.Valid {
		foundDigest.OverallSentiment = overallSentiment.String
	}
	if alertsSummary.Valid {
		foundDigest.AlertsSummary = alertsSummary.String
	}
	if trendsSummary.Valid {
		foundDigest.TrendsSummary = trendsSummary.String
	}
	if researchSuggestionsJSON.Valid && researchSuggestionsJSON.String != "" {
		_ = json.Unmarshal([]byte(researchSuggestionsJSON.String), &foundDigest.ResearchSuggestions)
	}

	return &foundDigest, nil
}

// AddFeed adds a new RSS/Atom feed to the database
func (s *Store) AddFeed(feed core.Feed) error {
	query := `
	INSERT INTO feeds 
	(id, url, title, description, last_fetched, last_modified, etag, active, error_count, last_error, date_added)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query,
		feed.ID,
		feed.URL,
		feed.Title,
		feed.Description,
		feed.LastFetched,
		feed.LastModified,
		feed.ETag,
		feed.Active,
		feed.ErrorCount,
		feed.LastError,
		feed.DateAdded,
	)

	return err
}

// GetFeeds retrieves all feeds, optionally filtering by active status
func (s *Store) GetFeeds(activeOnly bool) ([]core.Feed, error) {
	query := `
	SELECT id, url, title, description, last_fetched, last_modified, etag, active, error_count, last_error, date_added
	FROM feeds`

	if activeOnly {
		query += " WHERE active = TRUE"
	}

	query += " ORDER BY date_added DESC"

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query feeds: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var feeds []core.Feed
	for rows.Next() {
		var feed core.Feed
		var lastFetched, dateAdded sql.NullTime
		var title, description, lastModified, etag, lastError sql.NullString

		err := rows.Scan(
			&feed.ID,
			&feed.URL,
			&title,
			&description,
			&lastFetched,
			&lastModified,
			&etag,
			&feed.Active,
			&feed.ErrorCount,
			&lastError,
			&dateAdded,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feed row: %w", err)
		}

		// Handle nullable fields
		if title.Valid {
			feed.Title = title.String
		}
		if description.Valid {
			feed.Description = description.String
		}
		if lastFetched.Valid {
			feed.LastFetched = lastFetched.Time
		}
		if lastModified.Valid {
			feed.LastModified = lastModified.String
		}
		if etag.Valid {
			feed.ETag = etag.String
		}
		if lastError.Valid {
			feed.LastError = lastError.String
		}
		if dateAdded.Valid {
			feed.DateAdded = dateAdded.Time
		}

		feeds = append(feeds, feed)
	}

	return feeds, nil
}

// UpdateFeed updates feed metadata after fetching
func (s *Store) UpdateFeed(feedID string, title, description, lastModified, etag string, lastFetched time.Time) error {
	query := `
	UPDATE feeds 
	SET title = ?, description = ?, last_modified = ?, etag = ?, last_fetched = ?
	WHERE id = ?`

	_, err := s.db.Exec(query, title, description, lastModified, etag, lastFetched, feedID)
	return err
}

// UpdateFeedError updates feed error information
func (s *Store) UpdateFeedError(feedID string, errorMsg string) error {
	query := `
	UPDATE feeds 
	SET error_count = error_count + 1, last_error = ?
	WHERE id = ?`

	_, err := s.db.Exec(query, errorMsg, feedID)
	return err
}

// SetFeedActive enables or disables a feed
func (s *Store) SetFeedActive(feedID string, active bool) error {
	query := `UPDATE feeds SET active = ? WHERE id = ?`
	_, err := s.db.Exec(query, active, feedID)
	return err
}

// DeleteFeed removes a feed and all its items
func (s *Store) DeleteFeed(feedID string) error {
	// Delete feed items first
	_, err := s.db.Exec("DELETE FROM feed_items WHERE feed_id = ?", feedID)
	if err != nil {
		return fmt.Errorf("failed to delete feed items: %w", err)
	}

	// Delete the feed
	_, err = s.db.Exec("DELETE FROM feeds WHERE id = ?", feedID)
	if err != nil {
		return fmt.Errorf("failed to delete feed: %w", err)
	}

	return nil
}

// AddFeedItem adds a new feed item
func (s *Store) AddFeedItem(item core.FeedItem) error {
	query := `
	INSERT OR IGNORE INTO feed_items 
	(id, feed_id, title, link, description, published, guid, processed, date_discovered)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query,
		item.ID,
		item.FeedID,
		item.Title,
		item.Link,
		item.Description,
		item.Published,
		item.GUID,
		item.Processed,
		item.DateDiscovered,
	)

	return err
}

// GetUnprocessedFeedItems retrieves feed items that haven't been processed yet
func (s *Store) GetUnprocessedFeedItems(limit int) ([]core.FeedItem, error) {
	query := `
	SELECT id, feed_id, title, link, description, published, guid, processed, date_discovered
	FROM feed_items 
	WHERE processed = FALSE
	ORDER BY published DESC, date_discovered DESC
	LIMIT ?`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query feed items: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []core.FeedItem
	for rows.Next() {
		var item core.FeedItem
		var published sql.NullTime
		var title, description, guid sql.NullString

		err := rows.Scan(
			&item.ID,
			&item.FeedID,
			&title,
			&item.Link,
			&description,
			&published,
			&guid,
			&item.Processed,
			&item.DateDiscovered,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feed item row: %w", err)
		}

		// Handle nullable fields
		if title.Valid {
			item.Title = title.String
		}
		if description.Valid {
			item.Description = description.String
		}
		if published.Valid {
			item.Published = published.Time
		}
		if guid.Valid {
			item.GUID = guid.String
		}

		items = append(items, item)
	}

	return items, nil
}

// MarkFeedItemProcessed marks a feed item as processed
func (s *Store) MarkFeedItemProcessed(itemID string) error {
	query := `UPDATE feed_items SET processed = TRUE WHERE id = ?`
	_, err := s.db.Exec(query, itemID)
	return err
}

// GetFeedStats returns statistics about feed items
func (s *Store) GetFeedStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Count feeds
	var feedCount, activeFeedCount int
	err := s.db.QueryRow("SELECT COUNT(*) FROM feeds").Scan(&feedCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count feeds: %w", err)
	}
	err = s.db.QueryRow("SELECT COUNT(*) FROM feeds WHERE active = TRUE").Scan(&activeFeedCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count active feeds: %w", err)
	}

	// Count feed items
	var itemCount, processedCount int
	err = s.db.QueryRow("SELECT COUNT(*) FROM feed_items").Scan(&itemCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count feed items: %w", err)
	}
	err = s.db.QueryRow("SELECT COUNT(*) FROM feed_items WHERE processed = TRUE").Scan(&processedCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count processed items: %w", err)
	}

	stats["total_feeds"] = feedCount
	stats["active_feeds"] = activeFeedCount
	stats["total_items"] = itemCount
	stats["processed_items"] = processedCount
	stats["pending_items"] = itemCount - processedCount

	return stats, nil
}

// CacheStats represents cache statistics
type CacheStats struct {
	ArticleCount       int
	SummaryCount       int
	DigestCount        int
	CacheSize          int64
	LastUpdated        time.Time
	FeedCount          int
	ActiveFeedCount    int
	FeedItemCount      int
	ProcessedItemCount int
	TopicClusters      map[string]int
}

// GetCacheStats returns statistics about the cache
func (s *Store) GetCacheStats() (*CacheStats, error) {
	stats := &CacheStats{
		TopicClusters: make(map[string]int),
	}

	// Get counts
	queries := map[string]*int{
		"SELECT COUNT(*) FROM articles":   &stats.ArticleCount,
		"SELECT COUNT(*) FROM summaries":  &stats.SummaryCount,
		"SELECT COUNT(*) FROM digests":    &stats.DigestCount,
		"SELECT COUNT(*) FROM feeds":      &stats.FeedCount,
		"SELECT COUNT(*) FROM feed_items": &stats.FeedItemCount,
	}

	for query, target := range queries {
		err := s.db.QueryRow(query).Scan(target)
		if err != nil {
			return nil, fmt.Errorf("failed to get count: %w", err)
		}
	}

	// Get active feed count
	err := s.db.QueryRow("SELECT COUNT(*) FROM feeds WHERE active = TRUE").Scan(&stats.ActiveFeedCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get active feed count: %w", err)
	}

	// Get processed item count
	err = s.db.QueryRow("SELECT COUNT(*) FROM feed_items WHERE processed = TRUE").Scan(&stats.ProcessedItemCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get processed item count: %w", err)
	}

	// Get topic cluster distribution
	rows, err := s.db.Query(`
		SELECT topic_cluster, COUNT(*) as count 
		FROM (
			SELECT topic_cluster FROM articles WHERE topic_cluster IS NOT NULL AND topic_cluster != ''
			UNION ALL
			SELECT topic_cluster FROM summaries WHERE topic_cluster IS NOT NULL AND topic_cluster != ''
		) 
		GROUP BY topic_cluster
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get topic clusters: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var cluster string
		var count int
		if err := rows.Scan(&cluster, &count); err != nil {
			return nil, fmt.Errorf("failed to scan topic cluster: %w", err)
		}
		stats.TopicClusters[cluster] = count
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
	tables := []string{"articles", "summaries", "digests", "feed_items", "feeds"}

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
func (s *Store) GetRecentArticles(days int) ([]core.Article, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -days)

	query := `
	SELECT url, title, content, html_content, my_take, date_fetched, metadata, embedding, topic_cluster, topic_confidence,
	       sentiment_score, sentiment_label, sentiment_emoji, alert_triggered, alert_conditions, research_queries
	FROM articles 
	WHERE date_fetched > ?
	ORDER BY date_fetched DESC`

	rows, err := s.db.Query(query, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent articles: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var articles []core.Article
	for rows.Next() {
		var article core.Article
		var dateFetched time.Time
		var metadata string
		var embeddingData []byte
		var topicCluster sql.NullString
		var topicConfidence sql.NullFloat64
		var sentimentScore sql.NullFloat64
		var sentimentLabel sql.NullString
		var sentimentEmoji sql.NullString
		var alertTriggered sql.NullBool
		var alertConditionsJSON sql.NullString
		var researchQueriesJSON sql.NullString

		err := rows.Scan(
			&article.LinkID, // Use LinkID as URL identifier
			&article.Title,
			&article.CleanedText,
			&article.FetchedHTML,
			&article.MyTake,
			&dateFetched,
			&metadata,
			&embeddingData,
			&topicCluster,
			&topicConfidence,
			&sentimentScore,
			&sentimentLabel,
			&sentimentEmoji,
			&alertTriggered,
			&alertConditionsJSON,
			&researchQueriesJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article row: %w", err)
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

		// Handle insights fields
		if sentimentScore.Valid {
			article.SentimentScore = sentimentScore.Float64
		}
		if sentimentLabel.Valid {
			article.SentimentLabel = sentimentLabel.String
		}
		if sentimentEmoji.Valid {
			article.SentimentEmoji = sentimentEmoji.String
		}
		if alertTriggered.Valid {
			article.AlertTriggered = alertTriggered.Bool
		}
		if alertConditionsJSON.Valid && alertConditionsJSON.String != "" {
			_ = json.Unmarshal([]byte(alertConditionsJSON.String), &article.AlertConditions)
		}
		if researchQueriesJSON.Valid && researchQueriesJSON.String != "" {
			_ = json.Unmarshal([]byte(researchQueriesJSON.String), &article.ResearchQueries)
		}

		article.DateFetched = dateFetched
		articles = append(articles, article)
	}

	return articles, nil
}

// GetArticleByURL retrieves an article by its URL
func (s *Store) GetArticleByURL(url string) (*core.Article, error) {
	query := `
	SELECT url, title, content, html_content, my_take, date_fetched, metadata, embedding, topic_cluster, topic_confidence,
	       sentiment_score, sentiment_label, sentiment_emoji, alert_triggered, alert_conditions, research_queries
	FROM articles 
	WHERE url = ?`

	row := s.db.QueryRow(query, url)

	var article core.Article
	var dateFetched time.Time
	var metadata string
	var embeddingData []byte
	var topicCluster sql.NullString
	var topicConfidence sql.NullFloat64
	var sentimentScore sql.NullFloat64
	var sentimentLabel sql.NullString
	var sentimentEmoji sql.NullString
	var alertTriggered sql.NullBool
	var alertConditionsJSON sql.NullString
	var researchQueriesJSON sql.NullString

	err := row.Scan(
		&article.LinkID, // Use LinkID as URL identifier
		&article.Title,
		&article.CleanedText,
		&article.FetchedHTML,
		&article.MyTake,
		&dateFetched,
		&metadata,
		&embeddingData,
		&topicCluster,
		&topicConfidence,
		&sentimentScore,
		&sentimentLabel,
		&sentimentEmoji,
		&alertTriggered,
		&alertConditionsJSON,
		&researchQueriesJSON,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Article not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan article: %w", err)
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

	// Handle insights fields
	if sentimentScore.Valid {
		article.SentimentScore = sentimentScore.Float64
	}
	if sentimentLabel.Valid {
		article.SentimentLabel = sentimentLabel.String
	}
	if sentimentEmoji.Valid {
		article.SentimentEmoji = sentimentEmoji.String
	}
	if alertTriggered.Valid {
		article.AlertTriggered = alertTriggered.Bool
	}
	if alertConditionsJSON.Valid && alertConditionsJSON.String != "" {
		_ = json.Unmarshal([]byte(alertConditionsJSON.String), &article.AlertConditions)
	}
	if researchQueriesJSON.Valid && researchQueriesJSON.String != "" {
		_ = json.Unmarshal([]byte(researchQueriesJSON.String), &article.ResearchQueries)
	}

	article.DateFetched = dateFetched
	return &article, nil
}

// SaveArticle saves an article to the database
func (s *Store) SaveArticle(article *core.Article) error {
	return s.CacheArticle(*article)
}

// GetArticlesByDateRange retrieves articles within a specific date range
func (s *Store) GetArticlesByDateRange(startDate, endDate time.Time) ([]core.Article, error) {
	query := `
	SELECT url, title, content, html_content, my_take, date_fetched, metadata, embedding, topic_cluster, topic_confidence,
	       sentiment_score, sentiment_label, sentiment_emoji, alert_triggered, alert_conditions, research_queries
	FROM articles 
	WHERE date_fetched >= ? AND date_fetched <= ?
	ORDER BY date_fetched DESC`

	rows, err := s.db.Query(query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query articles by date range: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var articles []core.Article
	for rows.Next() {
		var article core.Article
		var dateFetched time.Time
		var metadata string
		var embeddingData []byte
		var topicCluster sql.NullString
		var topicConfidence sql.NullFloat64
		var sentimentScore sql.NullFloat64
		var sentimentLabel sql.NullString
		var sentimentEmoji sql.NullString
		var alertTriggered sql.NullBool
		var alertConditionsJSON sql.NullString
		var researchQueriesJSON sql.NullString

		err := rows.Scan(
			&article.LinkID, // Use LinkID as URL identifier
			&article.Title,
			&article.CleanedText,
			&article.FetchedHTML,
			&article.MyTake,
			&dateFetched,
			&metadata,
			&embeddingData,
			&topicCluster,
			&topicConfidence,
			&sentimentScore,
			&sentimentLabel,
			&sentimentEmoji,
			&alertTriggered,
			&alertConditionsJSON,
			&researchQueriesJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article row: %w", err)
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

		// Handle insights fields
		if sentimentScore.Valid {
			article.SentimentScore = sentimentScore.Float64
		}
		if sentimentLabel.Valid {
			article.SentimentLabel = sentimentLabel.String
		}
		if sentimentEmoji.Valid {
			article.SentimentEmoji = sentimentEmoji.String
		}
		if alertTriggered.Valid {
			article.AlertTriggered = alertTriggered.Bool
		}
		if alertConditionsJSON.Valid && alertConditionsJSON.String != "" {
			_ = json.Unmarshal([]byte(alertConditionsJSON.String), &article.AlertConditions)
		}
		if researchQueriesJSON.Valid && researchQueriesJSON.String != "" {
			_ = json.Unmarshal([]byte(researchQueriesJSON.String), &article.ResearchQueries)
		}

		article.DateFetched = dateFetched
		articles = append(articles, article)
	}

	return articles, nil
}

// FeedItemExists checks if a feed item exists in the database
func (s *Store) FeedItemExists(itemID string) (bool, error) {
	query := `SELECT COUNT(*) FROM feed_items WHERE id = ?`
	var count int
	err := s.db.QueryRow(query, itemID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check feed item existence: %w", err)
	}
	return count > 0, nil
}

// SaveFeedItem saves a feed item to the database (alias for AddFeedItem)
func (s *Store) SaveFeedItem(item core.FeedItem) error {
	return s.AddFeedItem(item)
}

// GetRecentFeedItems retrieves feed items published since the given time
func (s *Store) GetRecentFeedItems(since time.Time) ([]core.FeedItem, error) {
	query := `
	SELECT id, feed_id, title, link, description, published, guid, processed, date_discovered
	FROM feed_items 
	WHERE published >= ? OR (published IS NULL AND date_discovered >= ?)
	ORDER BY COALESCE(published, date_discovered) DESC`

	rows, err := s.db.Query(query, since, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent feed items: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []core.FeedItem
	for rows.Next() {
		var item core.FeedItem
		var published sql.NullTime
		var title, description, guid sql.NullString

		err := rows.Scan(
			&item.ID,
			&item.FeedID,
			&title,
			&item.Link,
			&description,
			&published,
			&guid,
			&item.Processed,
			&item.DateDiscovered,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan recent feed item row: %w", err)
		}

		// Handle nullable fields
		if title.Valid {
			item.Title = title.String
		}
		if description.Valid {
			item.Description = description.String
		}
		if published.Valid {
			item.Published = published.Time
		}
		if guid.Valid {
			item.GUID = guid.String
		}

		items = append(items, item)
	}

	return items, nil
}
