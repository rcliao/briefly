// Package persistence provides database implementations
package persistence

import (
	"briefly/internal/core"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq" // Postgres driver
)

// PostgresDB implements the Database interface for PostgreSQL
type PostgresDB struct {
	db         *sql.DB
	articles   ArticleRepository
	summaries  SummaryRepository
	feeds      FeedRepository
	feedItems  FeedItemRepository
	digests    DigestRepository
	themes     ThemeRepository     // Phase 0
	manualURLs ManualURLRepository // Phase 0
	citations  CitationRepository  // Phase 1
	tags       TagRepository       // Phase 1
}

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB(connectionString string) (*PostgresDB, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	pgDB := &PostgresDB{db: db}
	pgDB.articles = &postgresArticleRepo{db: db}
	pgDB.summaries = &postgresSummaryRepo{db: db}
	pgDB.feeds = &postgresFeedRepo{db: db}
	pgDB.feedItems = &postgresFeedItemRepo{db: db}
	pgDB.digests = &postgresDigestRepo{db: db}
	pgDB.themes = &postgresThemeRepo{db: db}         // Phase 0
	pgDB.manualURLs = &postgresManualURLRepo{db: db} // Phase 0
	pgDB.citations = &postgresCitationRepo{db: db}   // Phase 1
	pgDB.tags = &postgresTagRepo{db: db}             // Phase 1

	return pgDB, nil
}

func (p *PostgresDB) Articles() ArticleRepository     { return p.articles }
func (p *PostgresDB) Summaries() SummaryRepository    { return p.summaries }
func (p *PostgresDB) Feeds() FeedRepository           { return p.feeds }
func (p *PostgresDB) FeedItems() FeedItemRepository   { return p.feedItems }
func (p *PostgresDB) Digests() DigestRepository       { return p.digests }
func (p *PostgresDB) Themes() ThemeRepository         { return p.themes }     // Phase 0
func (p *PostgresDB) ManualURLs() ManualURLRepository { return p.manualURLs } // Phase 0
func (p *PostgresDB) Citations() CitationRepository   { return p.citations }  // Phase 1
func (p *PostgresDB) Tags() TagRepository             { return p.tags }       // Phase 1

func (p *PostgresDB) Close() error {
	return p.db.Close()
}

func (p *PostgresDB) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

func (p *PostgresDB) BeginTx(ctx context.Context) (Transaction, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &postgresTx{
		tx:         tx,
		articles:   &postgresArticleRepo{db: p.db, tx: tx},
		summaries:  &postgresSummaryRepo{db: p.db, tx: tx},
		feeds:      &postgresFeedRepo{db: p.db, tx: tx},
		feedItems:  &postgresFeedItemRepo{db: p.db, tx: tx},
		digests:    &postgresDigestRepo{db: p.db, tx: tx},
		themes:     &postgresThemeRepo{db: p.db, tx: tx},     // Phase 0
		manualURLs: &postgresManualURLRepo{db: p.db, tx: tx}, // Phase 0
		citations:  &postgresCitationRepo{db: p.db, tx: tx},  // Phase 1
		tags:       &postgresTagRepo{db: p.db, tx: tx},       // Phase 1
	}, nil
}

// postgresTx implements Transaction interface
type postgresTx struct {
	tx         *sql.Tx
	articles   ArticleRepository
	summaries  SummaryRepository
	feeds      FeedRepository
	feedItems  FeedItemRepository
	digests    DigestRepository
	themes     ThemeRepository     // Phase 0
	manualURLs ManualURLRepository // Phase 0
	citations  CitationRepository  // Phase 1
	tags       TagRepository       // Phase 1
}

func (t *postgresTx) Commit() error                   { return t.tx.Commit() }
func (t *postgresTx) Rollback() error                 { return t.tx.Rollback() }
func (t *postgresTx) Articles() ArticleRepository     { return t.articles }
func (t *postgresTx) Summaries() SummaryRepository    { return t.summaries }
func (t *postgresTx) Feeds() FeedRepository           { return t.feeds }
func (t *postgresTx) FeedItems() FeedItemRepository   { return t.feedItems }
func (t *postgresTx) Digests() DigestRepository       { return t.digests }
func (t *postgresTx) Themes() ThemeRepository         { return t.themes }     // Phase 0
func (t *postgresTx) ManualURLs() ManualURLRepository { return t.manualURLs } // Phase 0
func (t *postgresTx) Citations() CitationRepository   { return t.citations }  // Phase 1
func (t *postgresTx) Tags() TagRepository             { return t.tags }       // Phase 1

// postgresArticleRepo implements ArticleRepository for PostgreSQL
type postgresArticleRepo struct {
	db *sql.DB
	tx *sql.Tx
}

func (r *postgresArticleRepo) query() interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
} {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

func (r *postgresArticleRepo) Create(ctx context.Context, article *core.Article) error {
	embeddingJSON, err := json.Marshal(article.Embedding)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding: %w", err)
	}

	query := `
		INSERT INTO articles (
			id, url, title, content_type, cleaned_text, raw_content,
			topic_cluster, cluster_confidence, embedding, embedding_vector, date_fetched, date_added,
			theme_id, theme_relevance_score
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, CAST($10 AS VECTOR(768)), $11, $12, $13, $14)
		ON CONFLICT (url) DO UPDATE SET
			title = EXCLUDED.title,
			content_type = EXCLUDED.content_type,
			cleaned_text = EXCLUDED.cleaned_text,
			embedding = EXCLUDED.embedding,
			embedding_vector = CAST(EXCLUDED.embedding_vector AS TEXT)::VECTOR(768),
			theme_id = EXCLUDED.theme_id,
			theme_relevance_score = EXCLUDED.theme_relevance_score,
			date_fetched = EXCLUDED.date_fetched
	`

	// Convert embedding to VECTOR format for pgvector
	// Format: '[1.0,2.0,3.0,...]' - pgvector can parse this format
	var embeddingVector interface{}
	if len(article.Embedding) == 768 {
		// Build comma-separated string for VECTOR type
		embeddingStr := "["
		for i, val := range article.Embedding {
			if i > 0 {
				embeddingStr += ","
			}
			embeddingStr += fmt.Sprintf("%f", val)
		}
		embeddingStr += "]"
		embeddingVector = embeddingStr
	}

	_, err = r.query().ExecContext(ctx, query,
		article.ID, article.URL, article.Title, article.ContentType,
		article.CleanedText, article.RawContent, article.TopicCluster,
		article.ClusterConfidence, embeddingJSON, embeddingVector, article.DateFetched, time.Now().UTC(),
		article.ThemeID, article.ThemeRelevanceScore,
	)

	if err != nil {
		return fmt.Errorf("failed to insert/update article: %w", err)
	}
	return nil
}

func (r *postgresArticleRepo) Get(ctx context.Context, id string) (*core.Article, error) {
	query := `
		SELECT id, url, title, content_type, cleaned_text, raw_content,
			   topic_cluster, cluster_confidence, embedding, date_fetched, date_added,
			   theme_id, theme_relevance_score
		FROM articles WHERE id = $1
	`
	row := r.query().QueryRowContext(ctx, query, id)
	return r.scanArticle(row)
}

func (r *postgresArticleRepo) GetByURL(ctx context.Context, url string) (*core.Article, error) {
	query := `
		SELECT id, url, title, content_type, cleaned_text, raw_content,
			   topic_cluster, cluster_confidence, embedding, date_fetched, date_added,
			   theme_id, theme_relevance_score
		FROM articles WHERE url = $1
	`
	row := r.query().QueryRowContext(ctx, query, url)
	return r.scanArticle(row)
}

func (r *postgresArticleRepo) List(ctx context.Context, opts ListOptions) ([]core.Article, error) {
	query := `
		SELECT id, url, title, content_type, cleaned_text, raw_content,
			   topic_cluster, cluster_confidence, embedding, date_fetched, date_added,
			   theme_id, theme_relevance_score
		FROM articles
		ORDER BY date_added DESC
		LIMIT $1 OFFSET $2
	`
	limit := opts.Limit
	if limit == 0 {
		limit = 100 // Default limit
	}
	rows, err := r.query().QueryContext(ctx, query, limit, opts.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []core.Article
	for rows.Next() {
		article, err := r.scanArticleRow(rows)
		if err != nil {
			return nil, err
		}
		articles = append(articles, *article)
	}
	return articles, rows.Err()
}

func (r *postgresArticleRepo) Update(ctx context.Context, article *core.Article) error {
	embeddingJSON, err := json.Marshal(article.Embedding)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding: %w", err)
	}

	query := `
		UPDATE articles SET
			url = $2, title = $3, content_type = $4, cleaned_text = $5,
			raw_content = $6, topic_cluster = $7, cluster_confidence = $8,
			embedding = $9, date_fetched = $10
		WHERE id = $1
	`
	_, err = r.query().ExecContext(ctx, query,
		article.ID, article.URL, article.Title, article.ContentType,
		article.CleanedText, article.RawContent, article.TopicCluster,
		article.ClusterConfidence, embeddingJSON, article.DateFetched,
	)
	return err
}

func (r *postgresArticleRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM articles WHERE id = $1`
	_, err := r.query().ExecContext(ctx, query, id)
	return err
}

func (r *postgresArticleRepo) UpdateClusterAssignment(ctx context.Context, articleID string, clusterLabel string, confidence float64) error {
	query := `
		UPDATE articles
		SET topic_cluster = $2, cluster_confidence = $3
		WHERE id = $1
	`
	_, err := r.query().ExecContext(ctx, query, articleID, clusterLabel, confidence)
	if err != nil {
		return fmt.Errorf("failed to update cluster assignment for article %s: %w", articleID, err)
	}
	return nil
}

func (r *postgresArticleRepo) GetRecent(ctx context.Context, since time.Time, limit int) ([]core.Article, error) {
	query := `
		SELECT id, url, title, content_type, cleaned_text, raw_content,
			   topic_cluster, cluster_confidence, embedding, date_fetched, date_added
		FROM articles
		WHERE date_fetched >= $1
		ORDER BY date_fetched DESC
		LIMIT $2
	`
	rows, err := r.query().QueryContext(ctx, query, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []core.Article
	for rows.Next() {
		article, err := r.scanArticleRow(rows)
		if err != nil {
			return nil, err
		}
		articles = append(articles, *article)
	}
	return articles, rows.Err()
}

func (r *postgresArticleRepo) GetByCluster(ctx context.Context, clusterLabel string, limit int) ([]core.Article, error) {
	query := `
		SELECT id, url, title, content_type, cleaned_text, raw_content,
			   topic_cluster, cluster_confidence, embedding, date_fetched, date_added
		FROM articles
		WHERE topic_cluster = $1
		ORDER BY cluster_confidence DESC, date_fetched DESC
		LIMIT $2
	`
	rows, err := r.query().QueryContext(ctx, query, clusterLabel, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []core.Article
	for rows.Next() {
		article, err := r.scanArticleRow(rows)
		if err != nil {
			return nil, err
		}
		articles = append(articles, *article)
	}
	return articles, rows.Err()
}

func (r *postgresArticleRepo) scanArticle(row *sql.Row) (*core.Article, error) {
	var article core.Article
	var embeddingJSON []byte
	var dateAdded time.Time

	err := row.Scan(
		&article.ID, &article.URL, &article.Title, &article.ContentType,
		&article.CleanedText, &article.RawContent, &article.TopicCluster,
		&article.ClusterConfidence, &embeddingJSON, &article.DateFetched, &dateAdded,
		&article.ThemeID, &article.ThemeRelevanceScore,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("article not found")
		}
		return nil, err
	}

	if len(embeddingJSON) > 0 {
		if err := json.Unmarshal(embeddingJSON, &article.Embedding); err != nil {
			return nil, fmt.Errorf("failed to unmarshal embedding: %w", err)
		}
	}

	return &article, nil
}

func (r *postgresArticleRepo) scanArticleRow(rows *sql.Rows) (*core.Article, error) {
	var article core.Article
	var embeddingJSON []byte
	var dateAdded time.Time

	err := rows.Scan(
		&article.ID, &article.URL, &article.Title, &article.ContentType,
		&article.CleanedText, &article.RawContent, &article.TopicCluster,
		&article.ClusterConfidence, &embeddingJSON, &article.DateFetched, &dateAdded,
		&article.ThemeID, &article.ThemeRelevanceScore,
	)
	if err != nil {
		return nil, err
	}

	if len(embeddingJSON) > 0 {
		if err := json.Unmarshal(embeddingJSON, &article.Embedding); err != nil {
			return nil, fmt.Errorf("failed to unmarshal embedding: %w", err)
		}
	}

	return &article, nil
}

func (r *postgresArticleRepo) UpdateEmbedding(ctx context.Context, articleID string, embedding []float64) error {
	// Validate embedding dimensions
	if len(embedding) != 768 {
		return fmt.Errorf("invalid embedding dimensions: expected 768, got %d", len(embedding))
	}

	// Marshal to JSONB format
	embeddingJSON, err := json.Marshal(embedding)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding to JSON: %w", err)
	}

	// Convert to VECTOR format for pgvector: '[1.0,2.0,3.0,...]'
	embeddingStr := "["
	for i, val := range embedding {
		if i > 0 {
			embeddingStr += ","
		}
		embeddingStr += fmt.Sprintf("%f", val)
	}
	embeddingStr += "]"

	// Update both embedding (JSONB) and embedding_vector (VECTOR) columns
	query := `
		UPDATE articles
		SET embedding = $2,
		    embedding_vector = CAST($3 AS VECTOR(768))
		WHERE id = $1
	`

	result, err := r.query().ExecContext(ctx, query, articleID, embeddingJSON, embeddingStr)
	if err != nil {
		return fmt.Errorf("failed to update embedding for article %s: %w", articleID, err)
	}

	// Check if article was found
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("article %s not found", articleID)
	}

	return nil
}
