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
	db               *sql.DB
	articles         ArticleRepository
	summaries        SummaryRepository
	feeds            FeedRepository
	feedItems        FeedItemRepository
	digests          DigestRepository
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

	return pgDB, nil
}

func (p *PostgresDB) Articles() ArticleRepository   { return p.articles }
func (p *PostgresDB) Summaries() SummaryRepository  { return p.summaries }
func (p *PostgresDB) Feeds() FeedRepository         { return p.feeds }
func (p *PostgresDB) FeedItems() FeedItemRepository { return p.feedItems }
func (p *PostgresDB) Digests() DigestRepository     { return p.digests }

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
		tx:        tx,
		articles:  &postgresArticleRepo{db: p.db, tx: tx},
		summaries: &postgresSummaryRepo{db: p.db, tx: tx},
		feeds:     &postgresFeedRepo{db: p.db, tx: tx},
		feedItems: &postgresFeedItemRepo{db: p.db, tx: tx},
		digests:   &postgresDigestRepo{db: p.db, tx: tx},
	}, nil
}

// postgresTx implements Transaction interface
type postgresTx struct {
	tx        *sql.Tx
	articles  ArticleRepository
	summaries SummaryRepository
	feeds     FeedRepository
	feedItems FeedItemRepository
	digests   DigestRepository
}

func (t *postgresTx) Commit() error                      { return t.tx.Commit() }
func (t *postgresTx) Rollback() error                    { return t.tx.Rollback() }
func (t *postgresTx) Articles() ArticleRepository        { return t.articles }
func (t *postgresTx) Summaries() SummaryRepository       { return t.summaries }
func (t *postgresTx) Feeds() FeedRepository              { return t.feeds }
func (t *postgresTx) FeedItems() FeedItemRepository      { return t.feedItems }
func (t *postgresTx) Digests() DigestRepository          { return t.digests }

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
			topic_cluster, cluster_confidence, embedding, date_fetched, date_added
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err = r.query().ExecContext(ctx, query,
		article.ID, article.URL, article.Title, article.ContentType,
		article.CleanedText, article.RawContent, article.TopicCluster,
		article.ClusterConfidence, embeddingJSON, article.DateFetched, time.Now().UTC(),
	)
	return err
}

func (r *postgresArticleRepo) Get(ctx context.Context, id string) (*core.Article, error) {
	query := `
		SELECT id, url, title, content_type, cleaned_text, raw_content,
			   topic_cluster, cluster_confidence, embedding, date_fetched, date_added
		FROM articles WHERE id = $1
	`
	row := r.query().QueryRowContext(ctx, query, id)
	return r.scanArticle(row)
}

func (r *postgresArticleRepo) GetByURL(ctx context.Context, url string) (*core.Article, error) {
	query := `
		SELECT id, url, title, content_type, cleaned_text, raw_content,
			   topic_cluster, cluster_confidence, embedding, date_fetched, date_added
		FROM articles WHERE url = $1
	`
	row := r.query().QueryRowContext(ctx, query, url)
	return r.scanArticle(row)
}

func (r *postgresArticleRepo) List(ctx context.Context, opts ListOptions) ([]core.Article, error) {
	query := `
		SELECT id, url, title, content_type, cleaned_text, raw_content,
			   topic_cluster, cluster_confidence, embedding, date_fetched, date_added
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
