package persistence

import (
	"briefly/internal/core"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// postgresCitationRepo implements CitationRepository for PostgreSQL
type postgresCitationRepo struct {
	db *sql.DB
	tx *sql.Tx
}

func (r *postgresCitationRepo) query() interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
} {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

func (r *postgresCitationRepo) Create(ctx context.Context, citation *core.Citation) error {
	metadataJSON, err := json.Marshal(citation.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO citations (
			id, article_id, url, title, publisher, author,
			published_date, accessed_date, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err = r.query().ExecContext(ctx, query,
		citation.ID,
		citation.ArticleID,
		citation.URL,
		citation.Title,
		citation.Publisher,
		citation.Author,
		citation.PublishedDate,
		citation.AccessedDate,
		metadataJSON,
		time.Now().UTC(),
	)
	return err
}

func (r *postgresCitationRepo) Get(ctx context.Context, id string) (*core.Citation, error) {
	query := `
		SELECT id, article_id, url, title, publisher, author,
		       published_date, accessed_date, metadata, created_at
		FROM citations
		WHERE id = $1
	`
	row := r.query().QueryRowContext(ctx, query, id)
	return r.scanCitation(row)
}

func (r *postgresCitationRepo) GetByArticleID(ctx context.Context, articleID string) (*core.Citation, error) {
	query := `
		SELECT id, article_id, url, title, publisher, author,
		       published_date, accessed_date, metadata, created_at
		FROM citations
		WHERE article_id = $1
	`
	row := r.query().QueryRowContext(ctx, query, articleID)
	return r.scanCitation(row)
}

func (r *postgresCitationRepo) GetByArticleIDs(ctx context.Context, articleIDs []string) (map[string]*core.Citation, error) {
	if len(articleIDs) == 0 {
		return make(map[string]*core.Citation), nil
	}

	query := `
		SELECT id, article_id, url, title, publisher, author,
		       published_date, accessed_date, metadata, created_at
		FROM citations
		WHERE article_id = ANY($1)
	`

	rows, err := r.query().QueryContext(ctx, query, articleIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	citations := make(map[string]*core.Citation)
	for rows.Next() {
		citation, err := r.scanCitationRow(rows)
		if err != nil {
			return nil, err
		}
		citations[citation.ArticleID] = citation
	}

	return citations, rows.Err()
}

func (r *postgresCitationRepo) List(ctx context.Context, opts ListOptions) ([]core.Citation, error) {
	limit := opts.Limit
	if limit == 0 {
		limit = 100
	}

	query := `
		SELECT id, article_id, url, title, publisher, author,
		       published_date, accessed_date, metadata, created_at
		FROM citations
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.query().QueryContext(ctx, query, limit, opts.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var citations []core.Citation
	for rows.Next() {
		citation, err := r.scanCitationRow(rows)
		if err != nil {
			return nil, err
		}
		citations = append(citations, *citation)
	}

	return citations, rows.Err()
}

func (r *postgresCitationRepo) Update(ctx context.Context, citation *core.Citation) error {
	metadataJSON, err := json.Marshal(citation.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE citations SET
			url = $2,
			title = $3,
			publisher = $4,
			author = $5,
			published_date = $6,
			accessed_date = $7,
			metadata = $8
		WHERE id = $1
	`

	_, err = r.query().ExecContext(ctx, query,
		citation.ID,
		citation.URL,
		citation.Title,
		citation.Publisher,
		citation.Author,
		citation.PublishedDate,
		citation.AccessedDate,
		metadataJSON,
	)
	return err
}

func (r *postgresCitationRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM citations WHERE id = $1`
	_, err := r.query().ExecContext(ctx, query, id)
	return err
}

func (r *postgresCitationRepo) DeleteByArticleID(ctx context.Context, articleID string) error {
	query := `DELETE FROM citations WHERE article_id = $1`
	_, err := r.query().ExecContext(ctx, query, articleID)
	return err
}

func (r *postgresCitationRepo) scanCitation(row *sql.Row) (*core.Citation, error) {
	var citation core.Citation
	var metadataJSON []byte

	err := row.Scan(
		&citation.ID,
		&citation.ArticleID,
		&citation.URL,
		&citation.Title,
		&citation.Publisher,
		&citation.Author,
		&citation.PublishedDate,
		&citation.AccessedDate,
		&metadataJSON,
		&citation.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("citation not found")
		}
		return nil, err
	}

	// Unmarshal metadata if present
	if len(metadataJSON) > 0 && string(metadataJSON) != "null" {
		if err := json.Unmarshal(metadataJSON, &citation.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &citation, nil
}

func (r *postgresCitationRepo) scanCitationRow(rows *sql.Rows) (*core.Citation, error) {
	var citation core.Citation
	var metadataJSON []byte

	err := rows.Scan(
		&citation.ID,
		&citation.ArticleID,
		&citation.URL,
		&citation.Title,
		&citation.Publisher,
		&citation.Author,
		&citation.PublishedDate,
		&citation.AccessedDate,
		&metadataJSON,
		&citation.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Unmarshal metadata if present
	if len(metadataJSON) > 0 && string(metadataJSON) != "null" {
		if err := json.Unmarshal(metadataJSON, &citation.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &citation, nil
}
