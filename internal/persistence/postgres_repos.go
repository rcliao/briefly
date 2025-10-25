package persistence

import (
	"briefly/internal/core"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// postgresSummaryRepo implements SummaryRepository for PostgreSQL
type postgresSummaryRepo struct {
	db *sql.DB
	tx *sql.Tx
}

func (r *postgresSummaryRepo) query() interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
} {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

func (r *postgresSummaryRepo) Create(ctx context.Context, summary *core.Summary) error {
	articleIDsJSON, err := json.Marshal(summary.ArticleIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal article IDs: %w", err)
	}

	query := `
		INSERT INTO summaries (id, article_ids, summary_text, model_used, date_created)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = r.query().ExecContext(ctx, query,
		summary.ID, articleIDsJSON, summary.SummaryText, summary.ModelUsed, time.Now().UTC(),
	)
	return err
}

func (r *postgresSummaryRepo) Get(ctx context.Context, id string) (*core.Summary, error) {
	query := `SELECT id, article_ids, summary_text, model_used, date_created FROM summaries WHERE id = $1`
	row := r.query().QueryRowContext(ctx, query, id)
	return r.scanSummary(row)
}

func (r *postgresSummaryRepo) GetByArticleID(ctx context.Context, articleID string) ([]core.Summary, error) {
	query := `SELECT id, article_ids, summary_text, model_used, date_created FROM summaries WHERE article_ids @> $1`
	rows, err := r.query().QueryContext(ctx, query, fmt.Sprintf(`["%s"]`, articleID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []core.Summary
	for rows.Next() {
		summary, err := r.scanSummaryRow(rows)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, *summary)
	}
	return summaries, rows.Err()
}

func (r *postgresSummaryRepo) List(ctx context.Context, opts ListOptions) ([]core.Summary, error) {
	limit := opts.Limit
	if limit == 0 {
		limit = 100
	}
	query := `SELECT id, article_ids, summary_text, model_used, date_created FROM summaries ORDER BY date_created DESC LIMIT $1 OFFSET $2`
	rows, err := r.query().QueryContext(ctx, query, limit, opts.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []core.Summary
	for rows.Next() {
		summary, err := r.scanSummaryRow(rows)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, *summary)
	}
	return summaries, rows.Err()
}

func (r *postgresSummaryRepo) Update(ctx context.Context, summary *core.Summary) error {
	articleIDsJSON, err := json.Marshal(summary.ArticleIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal article IDs: %w", err)
	}

	query := `UPDATE summaries SET article_ids = $2, summary_text = $3, model_used = $4 WHERE id = $1`
	_, err = r.query().ExecContext(ctx, query, summary.ID, articleIDsJSON, summary.SummaryText, summary.ModelUsed)
	return err
}

func (r *postgresSummaryRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM summaries WHERE id = $1`
	_, err := r.query().ExecContext(ctx, query, id)
	return err
}

func (r *postgresSummaryRepo) scanSummary(row *sql.Row) (*core.Summary, error) {
	var summary core.Summary
	var articleIDsJSON []byte
	var dateCreated time.Time

	err := row.Scan(&summary.ID, &articleIDsJSON, &summary.SummaryText, &summary.ModelUsed, &dateCreated)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("summary not found")
		}
		return nil, err
	}

	if err := json.Unmarshal(articleIDsJSON, &summary.ArticleIDs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal article IDs: %w", err)
	}

	return &summary, nil
}

func (r *postgresSummaryRepo) scanSummaryRow(rows *sql.Rows) (*core.Summary, error) {
	var summary core.Summary
	var articleIDsJSON []byte
	var dateCreated time.Time

	err := rows.Scan(&summary.ID, &articleIDsJSON, &summary.SummaryText, &summary.ModelUsed, &dateCreated)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(articleIDsJSON, &summary.ArticleIDs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal article IDs: %w", err)
	}

	return &summary, nil
}

// postgresFeedRepo implements FeedRepository for PostgreSQL
type postgresFeedRepo struct {
	db *sql.DB
	tx *sql.Tx
}

func (r *postgresFeedRepo) query() interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
} {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

func (r *postgresFeedRepo) Create(ctx context.Context, feed *core.Feed) error {
	query := `
		INSERT INTO feeds (
			id, url, title, description, last_fetched, last_modified, etag,
			active, error_count, last_error, date_added
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.query().ExecContext(ctx, query,
		feed.ID, feed.URL, feed.Title, feed.Description, feed.LastFetched,
		feed.LastModified, feed.ETag, feed.Active, feed.ErrorCount,
		feed.LastError, feed.DateAdded,
	)
	return err
}

func (r *postgresFeedRepo) Get(ctx context.Context, id string) (*core.Feed, error) {
	query := `
		SELECT id, url, title, description, last_fetched, last_modified, etag,
			   active, error_count, last_error, date_added
		FROM feeds WHERE id = $1
	`
	row := r.query().QueryRowContext(ctx, query, id)
	return r.scanFeed(row)
}

func (r *postgresFeedRepo) GetByURL(ctx context.Context, url string) (*core.Feed, error) {
	query := `
		SELECT id, url, title, description, last_fetched, last_modified, etag,
			   active, error_count, last_error, date_added
		FROM feeds WHERE url = $1
	`
	row := r.query().QueryRowContext(ctx, query, url)
	return r.scanFeed(row)
}

func (r *postgresFeedRepo) ListActive(ctx context.Context) ([]core.Feed, error) {
	query := `
		SELECT id, url, title, description, last_fetched, last_modified, etag,
			   active, error_count, last_error, date_added
		FROM feeds WHERE active = true
		ORDER BY title
	`
	rows, err := r.query().QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []core.Feed
	for rows.Next() {
		feed, err := r.scanFeedRow(rows)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, *feed)
	}
	return feeds, rows.Err()
}

func (r *postgresFeedRepo) List(ctx context.Context, opts ListOptions) ([]core.Feed, error) {
	limit := opts.Limit
	if limit == 0 {
		limit = 100
	}
	query := `
		SELECT id, url, title, description, last_fetched, last_modified, etag,
			   active, error_count, last_error, date_added
		FROM feeds ORDER BY title LIMIT $1 OFFSET $2
	`
	rows, err := r.query().QueryContext(ctx, query, limit, opts.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []core.Feed
	for rows.Next() {
		feed, err := r.scanFeedRow(rows)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, *feed)
	}
	return feeds, rows.Err()
}

func (r *postgresFeedRepo) Update(ctx context.Context, feed *core.Feed) error {
	query := `
		UPDATE feeds SET
			url = $2, title = $3, description = $4, last_fetched = $5,
			last_modified = $6, etag = $7, active = $8, error_count = $9,
			last_error = $10
		WHERE id = $1
	`
	_, err := r.query().ExecContext(ctx, query,
		feed.ID, feed.URL, feed.Title, feed.Description, feed.LastFetched,
		feed.LastModified, feed.ETag, feed.Active, feed.ErrorCount, feed.LastError,
	)
	return err
}

func (r *postgresFeedRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM feeds WHERE id = $1`
	_, err := r.query().ExecContext(ctx, query, id)
	return err
}

func (r *postgresFeedRepo) UpdateLastFetched(ctx context.Context, id string, lastModified, etag string) error {
	query := `UPDATE feeds SET last_fetched = $2, last_modified = $3, etag = $4 WHERE id = $1`
	_, err := r.query().ExecContext(ctx, query, id, time.Now().UTC(), lastModified, etag)
	return err
}

func (r *postgresFeedRepo) scanFeed(row *sql.Row) (*core.Feed, error) {
	var feed core.Feed
	err := row.Scan(
		&feed.ID, &feed.URL, &feed.Title, &feed.Description, &feed.LastFetched,
		&feed.LastModified, &feed.ETag, &feed.Active, &feed.ErrorCount,
		&feed.LastError, &feed.DateAdded,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("feed not found")
		}
		return nil, err
	}
	return &feed, nil
}

func (r *postgresFeedRepo) scanFeedRow(rows *sql.Rows) (*core.Feed, error) {
	var feed core.Feed
	err := rows.Scan(
		&feed.ID, &feed.URL, &feed.Title, &feed.Description, &feed.LastFetched,
		&feed.LastModified, &feed.ETag, &feed.Active, &feed.ErrorCount,
		&feed.LastError, &feed.DateAdded,
	)
	if err != nil {
		return nil, err
	}
	return &feed, nil
}

// postgresFeedItemRepo implements FeedItemRepository for PostgreSQL
type postgresFeedItemRepo struct {
	db *sql.DB
	tx *sql.Tx
}

func (r *postgresFeedItemRepo) query() interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
} {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

func (r *postgresFeedItemRepo) Create(ctx context.Context, item *core.FeedItem) error {
	query := `
		INSERT INTO feed_items (
			id, feed_id, title, link, description, published, guid, processed, date_discovered
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.query().ExecContext(ctx, query,
		item.ID, item.FeedID, item.Title, item.Link, item.Description,
		item.Published, item.GUID, item.Processed, item.DateDiscovered,
	)
	return err
}

func (r *postgresFeedItemRepo) CreateBatch(ctx context.Context, items []core.FeedItem) error {
	if len(items) == 0 {
		return nil
	}

	// Use a transaction for batch insert
	tx, ok := r.query().(*sql.Tx)
	if !ok {
		var err error
		tx, err = r.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO feed_items (
			id, feed_id, title, link, description, published, guid, processed, date_discovered
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO NOTHING
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, item := range items {
		_, err := stmt.ExecContext(ctx,
			item.ID, item.FeedID, item.Title, item.Link, item.Description,
			item.Published, item.GUID, item.Processed, item.DateDiscovered,
		)
		if err != nil {
			return err
		}
	}

	if _, ok := r.query().(*sql.Tx); !ok {
		return tx.Commit()
	}
	return nil
}

func (r *postgresFeedItemRepo) Get(ctx context.Context, id string) (*core.FeedItem, error) {
	query := `
		SELECT id, feed_id, title, link, description, published, guid, processed, date_discovered
		FROM feed_items WHERE id = $1
	`
	row := r.query().QueryRowContext(ctx, query, id)
	return r.scanFeedItem(row)
}

func (r *postgresFeedItemRepo) GetByFeedID(ctx context.Context, feedID string, limit int) ([]core.FeedItem, error) {
	query := `
		SELECT id, feed_id, title, link, description, published, guid, processed, date_discovered
		FROM feed_items WHERE feed_id = $1
		ORDER BY published DESC
		LIMIT $2
	`
	rows, err := r.query().QueryContext(ctx, query, feedID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []core.FeedItem
	for rows.Next() {
		item, err := r.scanFeedItemRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *postgresFeedItemRepo) GetUnprocessed(ctx context.Context, limit int) ([]core.FeedItem, error) {
	query := `
		SELECT id, feed_id, title, link, description, published, guid, processed, date_discovered
		FROM feed_items WHERE processed = false
		ORDER BY published DESC
		LIMIT $1
	`
	rows, err := r.query().QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []core.FeedItem
	for rows.Next() {
		item, err := r.scanFeedItemRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *postgresFeedItemRepo) List(ctx context.Context, opts ListOptions) ([]core.FeedItem, error) {
	limit := opts.Limit
	if limit == 0 {
		limit = 100
	}
	query := `
		SELECT id, feed_id, title, link, description, published, guid, processed, date_discovered
		FROM feed_items ORDER BY published DESC LIMIT $1 OFFSET $2
	`
	rows, err := r.query().QueryContext(ctx, query, limit, opts.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []core.FeedItem
	for rows.Next() {
		item, err := r.scanFeedItemRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *postgresFeedItemRepo) MarkProcessed(ctx context.Context, id string) error {
	query := `UPDATE feed_items SET processed = true WHERE id = $1`
	_, err := r.query().ExecContext(ctx, query, id)
	return err
}

func (r *postgresFeedItemRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM feed_items WHERE id = $1`
	_, err := r.query().ExecContext(ctx, query, id)
	return err
}

func (r *postgresFeedItemRepo) scanFeedItem(row *sql.Row) (*core.FeedItem, error) {
	var item core.FeedItem
	err := row.Scan(
		&item.ID, &item.FeedID, &item.Title, &item.Link, &item.Description,
		&item.Published, &item.GUID, &item.Processed, &item.DateDiscovered,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("feed item not found")
		}
		return nil, err
	}
	return &item, nil
}

func (r *postgresFeedItemRepo) scanFeedItemRow(rows *sql.Rows) (*core.FeedItem, error) {
	var item core.FeedItem
	err := rows.Scan(
		&item.ID, &item.FeedID, &item.Title, &item.Link, &item.Description,
		&item.Published, &item.GUID, &item.Processed, &item.DateDiscovered,
	)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// postgresDigestRepo implements DigestRepository for PostgreSQL
type postgresDigestRepo struct {
	db *sql.DB
	tx *sql.Tx
}

func (r *postgresDigestRepo) query() interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
} {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

func (r *postgresDigestRepo) Create(ctx context.Context, digest *core.Digest) error {
	digestJSON, err := json.Marshal(digest)
	if err != nil {
		return fmt.Errorf("failed to marshal digest: %w", err)
	}

	query := `INSERT INTO digests (id, date, content, created_at) VALUES ($1, $2, $3, $4)`
	_, err = r.query().ExecContext(ctx, query,
		digest.Metadata.Title, digest.Metadata.DateGenerated, digestJSON, time.Now().UTC(),
	)
	return err
}

func (r *postgresDigestRepo) Get(ctx context.Context, id string) (*core.Digest, error) {
	query := `SELECT content FROM digests WHERE id = $1`
	row := r.query().QueryRowContext(ctx, query, id)

	var digestJSON []byte
	if err := row.Scan(&digestJSON); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("digest not found")
		}
		return nil, err
	}

	var digest core.Digest
	if err := json.Unmarshal(digestJSON, &digest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal digest: %w", err)
	}

	return &digest, nil
}

func (r *postgresDigestRepo) GetByDate(ctx context.Context, date time.Time) (*core.Digest, error) {
	query := `SELECT content FROM digests WHERE date = $1`
	row := r.query().QueryRowContext(ctx, query, date)

	var digestJSON []byte
	if err := row.Scan(&digestJSON); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("digest not found")
		}
		return nil, err
	}

	var digest core.Digest
	if err := json.Unmarshal(digestJSON, &digest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal digest: %w", err)
	}

	return &digest, nil
}

func (r *postgresDigestRepo) List(ctx context.Context, opts ListOptions) ([]core.Digest, error) {
	limit := opts.Limit
	if limit == 0 {
		limit = 50
	}
	query := `SELECT content FROM digests ORDER BY date DESC LIMIT $1 OFFSET $2`
	rows, err := r.query().QueryContext(ctx, query, limit, opts.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var digests []core.Digest
	for rows.Next() {
		var digestJSON []byte
		if err := rows.Scan(&digestJSON); err != nil {
			return nil, err
		}

		var digest core.Digest
		if err := json.Unmarshal(digestJSON, &digest); err != nil {
			return nil, fmt.Errorf("failed to unmarshal digest: %w", err)
		}
		digests = append(digests, digest)
	}
	return digests, rows.Err()
}

func (r *postgresDigestRepo) Update(ctx context.Context, digest *core.Digest) error {
	digestJSON, err := json.Marshal(digest)
	if err != nil {
		return fmt.Errorf("failed to marshal digest: %w", err)
	}

	query := `UPDATE digests SET content = $2, date = $3 WHERE id = $1`
	_, err = r.query().ExecContext(ctx, query, digest.Metadata.Title, digestJSON, digest.Metadata.DateGenerated)
	return err
}

func (r *postgresDigestRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM digests WHERE id = $1`
	_, err := r.query().ExecContext(ctx, query, id)
	return err
}

func (r *postgresDigestRepo) GetLatest(ctx context.Context, limit int) ([]core.Digest, error) {
	return r.List(ctx, ListOptions{Limit: limit})
}
