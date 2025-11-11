package persistence

import (
	"briefly/internal/core"
	"briefly/internal/markdown"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
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
	var lastFetched sql.NullTime
	var lastModified, etag, lastError sql.NullString

	err := row.Scan(
		&feed.ID, &feed.URL, &feed.Title, &feed.Description, &lastFetched,
		&lastModified, &etag, &feed.Active, &feed.ErrorCount,
		&lastError, &feed.DateAdded,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("feed not found")
		}
		return nil, err
	}

	// Handle nullable fields
	if lastFetched.Valid {
		feed.LastFetched = &lastFetched.Time
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

	return &feed, nil
}

func (r *postgresFeedRepo) scanFeedRow(rows *sql.Rows) (*core.Feed, error) {
	var feed core.Feed
	var lastFetched sql.NullTime
	var lastModified, etag, lastError sql.NullString

	err := rows.Scan(
		&feed.ID, &feed.URL, &feed.Title, &feed.Description, &lastFetched,
		&lastModified, &etag, &feed.Active, &feed.ErrorCount,
		&lastError, &feed.DateAdded,
	)
	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if lastFetched.Valid {
		feed.LastFetched = &lastFetched.Time
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
		defer func() {
			_ = tx.Rollback() // Rollback is safe to ignore if commit succeeds
		}()
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
	// Marshal entire digest for legacy content column
	digestJSON, err := json.Marshal(digest)
	if err != nil {
		return fmt.Errorf("failed to marshal digest: %w", err)
	}

	// Marshal JSON fields
	keyMomentsJSON, err := json.Marshal(digest.KeyMoments)
	if err != nil {
		return fmt.Errorf("failed to marshal key_moments: %w", err)
	}

	perspectivesJSON, err := json.Marshal(digest.Perspectives)
	if err != nil {
		return fmt.Errorf("failed to marshal perspectives: %w", err)
	}

	// v3.0: Marshal by_the_numbers
	byTheNumbersJSON, err := json.Marshal(digest.ByTheNumbers)
	if err != nil {
		return fmt.Errorf("failed to marshal by_the_numbers: %w", err)
	}

	// v3.0: Insert into proper columns including new scannable format fields
	query := `
		INSERT INTO digests (
			id, date, content, created_at,
			title, summary, tldr_summary, key_moments, perspectives,
			cluster_id, processed_date, article_count,
			top_developments, by_the_numbers, why_it_matters
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (date)
		DO UPDATE SET
			id = EXCLUDED.id,
			content = EXCLUDED.content,
			title = EXCLUDED.title,
			summary = EXCLUDED.summary,
			tldr_summary = EXCLUDED.tldr_summary,
			key_moments = EXCLUDED.key_moments,
			perspectives = EXCLUDED.perspectives,
			cluster_id = EXCLUDED.cluster_id,
			processed_date = EXCLUDED.processed_date,
			article_count = EXCLUDED.article_count,
			top_developments = EXCLUDED.top_developments,
			by_the_numbers = EXCLUDED.by_the_numbers,
			why_it_matters = EXCLUDED.why_it_matters,
			created_at = EXCLUDED.created_at
	`
	_, err = r.query().ExecContext(ctx, query,
		digest.ID,
		digest.ProcessedDate,
		digestJSON,
		time.Now().UTC(),
		digest.Title,
		digest.Summary,
		digest.TLDRSummary,
		keyMomentsJSON,
		perspectivesJSON,
		digest.ClusterID,
		digest.ProcessedDate,
		digest.ArticleCount,
		pq.Array(digest.TopDevelopments), // $13: v3.0 top_developments
		byTheNumbersJSON,                  // $14: v3.0 by_the_numbers
		digest.WhyItMatters,               // $15: v3.0 why_it_matters
	)
	if err != nil {
		return fmt.Errorf("failed to insert digest: %w", err)
	}

	// Insert article relationships into digest_articles
	for i, article := range digest.Articles {
		insertArticleQuery := `
			INSERT INTO digest_articles (digest_id, article_id, citation_order)
			VALUES ($1, $2, $3)
			ON CONFLICT (digest_id, article_id) DO NOTHING
		`
		_, err = r.query().ExecContext(ctx, insertArticleQuery,
			digest.ID, article.ID, i+1,
		)
		if err != nil {
			return fmt.Errorf("failed to insert digest_article: %w", err)
		}
	}

	// Extract themes from ArticleGroups and insert into digest_themes
	// Need to look up theme IDs from theme names
	themeSet := make(map[string]bool)
	for _, group := range digest.ArticleGroups {
		if group.Theme != "" {
			themeSet[group.Theme] = true
		}
	}

	for themeName := range themeSet {
		// Look up theme ID by name
		var themeID string
		lookupQuery := `SELECT id FROM themes WHERE name = $1 LIMIT 1`
		err = r.query().QueryRowContext(ctx, lookupQuery, themeName).Scan(&themeID)
		if err != nil {
			// Theme not found in database, skip it (or could create it)
			// For now, just log and continue
			continue
		}

		// Insert into digest_themes using theme_id
		insertThemeQuery := `
			INSERT INTO digest_themes (digest_id, theme_id)
			VALUES ($1, $2)
			ON CONFLICT (digest_id, theme_id) DO NOTHING
		`
		_, err = r.query().ExecContext(ctx, insertThemeQuery,
			digest.ID, themeID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert digest_theme: %w", err)
		}
	}

	return nil
}

func (r *postgresDigestRepo) Get(ctx context.Context, id string) (*core.Digest, error) {
	// v3.0: Select from proper columns including new scannable format fields
	query := `
		SELECT
			id, content, title, summary, tldr_summary, key_moments, perspectives,
			cluster_id, processed_date, article_count, created_at, date,
			top_developments, by_the_numbers, why_it_matters
		FROM digests
		WHERE id = $1
	`
	row := r.query().QueryRowContext(ctx, query, id)

	var digest core.Digest
	var contentJSON []byte
	var keyMomentsJSON, perspectivesJSON, byTheNumbersJSON []byte
	var clusterID sql.NullInt64
	var processedDate, createdAt, legacyDate sql.NullTime
	var whyItMatters sql.NullString

	if err := row.Scan(
		&digest.ID,
		&contentJSON,
		&digest.Title,
		&digest.Summary,
		&digest.TLDRSummary,
		&keyMomentsJSON,
		&perspectivesJSON,
		&clusterID,
		&processedDate,
		&digest.ArticleCount,
		&createdAt,
		&legacyDate,
		pq.Array(&digest.TopDevelopments),  // v3.0
		&byTheNumbersJSON,                   // v3.0
		&whyItMatters,                       // v3.0
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("digest not found")
		}
		return nil, err
	}

	// Unmarshal key_moments
	if len(keyMomentsJSON) > 0 && string(keyMomentsJSON) != "null" {
		if err := json.Unmarshal(keyMomentsJSON, &digest.KeyMoments); err != nil {
			return nil, fmt.Errorf("failed to unmarshal key_moments: %w", err)
		}
	}

	// Unmarshal perspectives
	if len(perspectivesJSON) > 0 && string(perspectivesJSON) != "null" {
		if err := json.Unmarshal(perspectivesJSON, &digest.Perspectives); err != nil {
			return nil, fmt.Errorf("failed to unmarshal perspectives: %w", err)
		}
	}

	// v3.0: Unmarshal by_the_numbers
	if len(byTheNumbersJSON) > 0 && string(byTheNumbersJSON) != "null" {
		if err := json.Unmarshal(byTheNumbersJSON, &digest.ByTheNumbers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal by_the_numbers: %w", err)
		}
	}

	// v3.0: Set why_it_matters
	if whyItMatters.Valid {
		digest.WhyItMatters = whyItMatters.String
	}

	// Handle nullable cluster_id
	if clusterID.Valid {
		clusterIDInt := int(clusterID.Int64)
		digest.ClusterID = &clusterIDInt
	}

	// Set dates
	if processedDate.Valid {
		digest.ProcessedDate = processedDate.Time
	}

	// Unmarshal legacy content JSONB for ArticleGroups and Metadata
	if len(contentJSON) > 0 {
		var legacyData struct {
			ArticleGroups []core.ArticleGroup `json:"article_groups"`
			Metadata      core.DigestMetadata `json:"metadata"`
			DigestSummary string              `json:"summary"`
		}
		if err := json.Unmarshal(contentJSON, &legacyData); err == nil {
			digest.ArticleGroups = legacyData.ArticleGroups
			digest.Metadata = legacyData.Metadata
			// Use legacy summary if v2.0 summary is empty
			if digest.DigestSummary == "" {
				digest.DigestSummary = legacyData.DigestSummary
			}
		}
	}

	// Load associated articles from digest_articles relationship
	articlesQuery := `
		SELECT a.id, a.url, a.title, a.content_type, a.publisher, a.cleaned_text,
		       a.date_fetched, da.citation_order
		FROM articles a
		INNER JOIN digest_articles da ON a.id = da.article_id
		WHERE da.digest_id = $1
		ORDER BY da.citation_order ASC
	`
	articleRows, err := r.query().QueryContext(ctx, articlesQuery, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load digest articles: %w", err)
	}
	defer articleRows.Close()

	var articles []core.Article
	for articleRows.Next() {
		var article core.Article
		var citationOrder int
		var publisher sql.NullString // Handle nullable publisher field

		if err := articleRows.Scan(
			&article.ID,
			&article.URL,
			&article.Title,
			&article.ContentType,
			&publisher,
			&article.CleanedText,
			&article.DateFetched,
			&citationOrder,
		); err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}

		// Set publisher if not null
		if publisher.Valid {
			article.Publisher = publisher.String
		}

		articles = append(articles, article)
	}

	// Set articles on digest
	digest.Articles = articles

	// Also populate ArticleGroups if empty (for backward compatibility)
	if len(digest.ArticleGroups) == 0 && len(articles) > 0 {
		digest.ArticleGroups = []core.ArticleGroup{
			{
				Theme:    digest.Title,
				Articles: articles,
				Summary:  digest.TLDRSummary,
			},
		}
	} else if len(digest.ArticleGroups) > 0 {
		// Update existing article groups with loaded articles
		for i := range digest.ArticleGroups {
			digest.ArticleGroups[i].Articles = articles
		}
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

	// v3.0: Select from proper columns including new scannable format fields
	query := `
		SELECT
			d.id, d.content, d.title, d.summary, d.tldr_summary, d.key_moments, d.perspectives,
			d.cluster_id, d.processed_date, d.article_count, d.created_at, d.date,
			d.top_developments, d.by_the_numbers, d.why_it_matters,
			COALESCE(
				(SELECT json_agg(t.name)
				 FROM digest_themes dt
				 JOIN themes t ON dt.theme_id = t.id
				 WHERE dt.digest_id = d.id),
				'[]'::json
			) as themes
		FROM digests d
		ORDER BY d.processed_date DESC, d.date DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.query().QueryContext(ctx, query, limit, opts.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var digests []core.Digest
	for rows.Next() {
		var digest core.Digest
		var contentJSON []byte
		var keyMomentsJSON, perspectivesJSON, byTheNumbersJSON []byte
		var themesJSON []byte
		var clusterID sql.NullInt64
		var processedDate, createdAt, legacyDate sql.NullTime
		var whyItMatters sql.NullString

		if err := rows.Scan(
			&digest.ID,
			&contentJSON,
			&digest.Title,
			&digest.Summary,
			&digest.TLDRSummary,
			&keyMomentsJSON,
			&perspectivesJSON,
			&clusterID,
			&processedDate,
			&digest.ArticleCount,
			&createdAt,
			&legacyDate,
			pq.Array(&digest.TopDevelopments),  // v3.0
			&byTheNumbersJSON,                   // v3.0
			&whyItMatters,                       // v3.0
			&themesJSON,
		); err != nil {
			return nil, err
		}

		// Unmarshal key_moments
		if len(keyMomentsJSON) > 0 && string(keyMomentsJSON) != "null" {
			if err := json.Unmarshal(keyMomentsJSON, &digest.KeyMoments); err != nil {
				return nil, fmt.Errorf("failed to unmarshal key_moments: %w", err)
			}
		}

		// Unmarshal perspectives
		if len(perspectivesJSON) > 0 && string(perspectivesJSON) != "null" {
			if err := json.Unmarshal(perspectivesJSON, &digest.Perspectives); err != nil {
				return nil, fmt.Errorf("failed to unmarshal perspectives: %w", err)
			}
		}

		// v3.0: Unmarshal by_the_numbers
		if len(byTheNumbersJSON) > 0 && string(byTheNumbersJSON) != "null" {
			if err := json.Unmarshal(byTheNumbersJSON, &digest.ByTheNumbers); err != nil {
				return nil, fmt.Errorf("failed to unmarshal by_the_numbers: %w", err)
			}
		}

		// v3.0: Set why_it_matters
		if whyItMatters.Valid {
			digest.WhyItMatters = whyItMatters.String
		}

		// Handle nullable cluster_id
		if clusterID.Valid {
			clusterIDInt := int(clusterID.Int64)
			digest.ClusterID = &clusterIDInt
		}

		// Set dates
		if processedDate.Valid {
			digest.ProcessedDate = processedDate.Time
		}

		// Unmarshal themes from digest_themes table
		var themes []string
		if len(themesJSON) > 0 && string(themesJSON) != "null" && string(themesJSON) != "[]" {
			if err := json.Unmarshal(themesJSON, &themes); err == nil && len(themes) > 0 {
				// Build ArticleGroups from themes for backward compatibility
				digest.ArticleGroups = make([]core.ArticleGroup, len(themes))
				for i, theme := range themes {
					digest.ArticleGroups[i] = core.ArticleGroup{
						Theme:    theme,
						Category: theme,
					}
				}
			}
		}

		// Unmarshal legacy content JSONB for Metadata and fallback ArticleGroups
		if len(contentJSON) > 0 {
			var legacyData struct {
				ArticleGroups []core.ArticleGroup `json:"article_groups"`
				Metadata      core.DigestMetadata `json:"metadata"`
				DigestSummary string              `json:"summary"`
			}
			if err := json.Unmarshal(contentJSON, &legacyData); err == nil {
				// Use legacy ArticleGroups only if we didn't get themes from digest_themes
				if len(digest.ArticleGroups) == 0 && len(legacyData.ArticleGroups) > 0 {
					digest.ArticleGroups = legacyData.ArticleGroups
				}
				digest.Metadata = legacyData.Metadata
				// Use legacy summary if v2.0 summary is empty
				if digest.DigestSummary == "" {
					digest.DigestSummary = legacyData.DigestSummary
				}
			}
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

// StoreWithRelationships stores a digest with article and theme relationships (v2.0)
// This method performs all operations in a transaction for atomicity:
// 1. Insert digest
// 2. Create digest_articles relationships with citation order
// 3. Create digest_themes relationships
// 4. Extract and store citations from summary markdown
func (r *postgresDigestRepo) StoreWithRelationships(ctx context.Context, digest *core.Digest, articleIDs []string, themeIDs []string) error {
	// Start transaction
	var tx *sql.Tx
	var err error

	if r.tx != nil {
		// Already in a transaction, use it
		tx = r.tx
	} else {
		// Start new transaction
		tx, err = r.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer func() {
			if err != nil {
				_ = tx.Rollback()
			}
		}()
	}

	// 1. Insert digest (using v2.0 schema)
	keyMomentsJSON, err := json.Marshal(digest.KeyMoments)
	if err != nil {
		return fmt.Errorf("failed to marshal key_moments: %w", err)
	}

	perspectivesJSON, err := json.Marshal(digest.Perspectives)
	if err != nil {
		return fmt.Errorf("failed to marshal perspectives: %w", err)
	}

	// v3.0: Marshal by_the_numbers
	byTheNumbersJSON, err := json.Marshal(digest.ByTheNumbers)
	if err != nil {
		return fmt.Errorf("failed to marshal by_the_numbers: %w", err)
	}

	// Build legacy content JSONB for backward compatibility
	contentJSON := map[string]interface{}{
		"summary": digest.Summary,
		"title":   digest.Title,
		"my_take": "",
		"metadata": map[string]interface{}{
			"title":          digest.Title,
			"tldr_summary":   digest.TLDRSummary,
			"article_count":  digest.ArticleCount,
			"date_generated": time.Now().UTC(),
		},
	}
	contentJSONBytes, err := json.Marshal(contentJSON)
	if err != nil {
		return fmt.Errorf("failed to marshal content JSON: %w", err)
	}

	query := `
		INSERT INTO digests (
			id, date, content, title, summary, tldr_summary, key_moments, perspectives,
			cluster_id, processed_date, article_count, created_at,
			top_developments, by_the_numbers, why_it_matters
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	// Use processed_date for both date and processed_date (backward compatibility)
	dateValue := digest.ProcessedDate
	if dateValue.IsZero() {
		dateValue = time.Now()
	}

	_, err = tx.ExecContext(ctx, query,
		digest.ID,
		dateValue,                        // Legacy date column
		contentJSONBytes,                 // Legacy content JSONB column
		digest.Title,                     // Legacy title column
		digest.Summary,                   // v2.0 summary field
		digest.TLDRSummary,               // v2.0 tldr
		keyMomentsJSON,                   // v2.0
		perspectivesJSON,                 // v2.0
		digest.ClusterID,                 // v2.0
		digest.ProcessedDate,             // v2.0
		digest.ArticleCount,              // v2.0
		time.Now().UTC(),
		pq.Array(digest.TopDevelopments), // v3.0 top_developments
		byTheNumbersJSON,                  // v3.0 by_the_numbers
		digest.WhyItMatters,               // v3.0 why_it_matters
	)
	if err != nil {
		return fmt.Errorf("failed to insert digest: %w", err)
	}

	// 2. Create digest_articles relationships with citation order
	for i, articleID := range articleIDs {
		citationOrder := i + 1
		query := `
			INSERT INTO digest_articles (digest_id, article_id, citation_order, added_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (digest_id, article_id) DO NOTHING
		`
		_, err = tx.ExecContext(ctx, query, digest.ID, articleID, citationOrder, time.Now().UTC())
		if err != nil {
			return fmt.Errorf("failed to insert digest_article relationship: %w", err)
		}
	}

	// 3. Create digest_themes relationships
	for _, themeID := range themeIDs {
		query := `
			INSERT INTO digest_themes (digest_id, theme_id, added_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (digest_id, theme_id) DO NOTHING
		`
		_, err = tx.ExecContext(ctx, query, digest.ID, themeID, time.Now().UTC())
		if err != nil {
			return fmt.Errorf("failed to insert digest_theme relationship: %w", err)
		}
	}

	// 4. Extract and store citations from summary markdown
	if digest.Summary != "" {
		// Extract citations from the markdown summary
		citationRefs := markdown.ExtractCitations(digest.Summary)

		if len(citationRefs) > 0 {
			// Build article map for citation lookup (URL -> Article)
			articleMap := make(map[string]*core.Article)
			for i := range digest.Articles {
				articleMap[digest.Articles[i].URL] = &digest.Articles[i]
			}

			// Build citation records
			citationRecords := markdown.BuildCitationRecords(digest.ID, citationRefs, articleMap)

			// Insert citations into database
			for _, citation := range citationRecords {
				query := `
					INSERT INTO citations (
						id, article_id, url, title, publisher, published_date,
						accessed_date, created_at, digest_id, citation_number, context
					) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
					ON CONFLICT (id) DO NOTHING
				`
				_, err = tx.ExecContext(ctx, query,
					citation.ID,
					citation.ArticleID,
					citation.URL,
					citation.Title,
					citation.Publisher,
					citation.PublishedDate,
					citation.AccessedDate,
					citation.CreatedAt,
					citation.DigestID,
					citation.CitationNumber,
					citation.Context,
				)
				if err != nil {
					return fmt.Errorf("failed to insert citation: %w", err)
				}
			}
		}
	}

	// Commit transaction if we started it
	if r.tx == nil {
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	return nil
}

// GetWithArticles retrieves a digest with all associated articles loaded (v2.0)
func (r *postgresDigestRepo) GetWithArticles(ctx context.Context, id string) (*core.Digest, error) {
	// TODO: Implement v2.0 digest retrieval with article relationships
	// For now, fall back to basic Get
	return r.Get(ctx, id)
}

// GetWithThemes retrieves a digest with all associated themes loaded (v2.0)
func (r *postgresDigestRepo) GetWithThemes(ctx context.Context, id string) (*core.Digest, error) {
	// TODO: Implement v2.0 digest retrieval with theme relationships
	// For now, fall back to basic Get
	return r.Get(ctx, id)
}

// GetFull retrieves a digest with articles, themes, and citations loaded (v2.0)
func (r *postgresDigestRepo) GetFull(ctx context.Context, id string) (*core.Digest, error) {
	// TODO: Implement v2.0 full digest retrieval with all relationships
	// For now, fall back to basic Get
	return r.Get(ctx, id)
}

// ListRecent retrieves digests processed since a given date (v2.0)
// Used for homepage digest list with time window filtering
func (r *postgresDigestRepo) ListRecent(ctx context.Context, since time.Time, limit int) ([]core.Digest, error) {
	if limit == 0 {
		limit = 50
	}

	query := `
		SELECT
			d.id, d.summary, d.tldr_summary, d.key_moments, d.perspectives,
			d.cluster_id, d.processed_date, d.article_count, d.created_at,
			COALESCE(
				json_agg(
					DISTINCT jsonb_build_object(
						'id', t.id,
						'name', t.name,
						'description', t.description
					)
				) FILTER (WHERE t.id IS NOT NULL),
				'[]'
			) as themes
		FROM digests d
		LEFT JOIN digest_themes dt ON d.id = dt.digest_id
		LEFT JOIN themes t ON dt.theme_id = t.id
		WHERE d.processed_date >= $1
		GROUP BY d.id, d.summary, d.tldr_summary, d.key_moments, d.perspectives,
				 d.cluster_id, d.processed_date, d.article_count, d.created_at
		ORDER BY d.processed_date DESC, d.created_at DESC
		LIMIT $2
	`

	rows, err := r.query().QueryContext(ctx, query, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent digests: %w", err)
	}
	defer rows.Close()

	var digests []core.Digest
	for rows.Next() {
		var d core.Digest
		var keyMomentsJSON, perspectivesJSON, themesJSON []byte

		err := rows.Scan(
			&d.ID, &d.Summary, &d.TLDRSummary, &keyMomentsJSON, &perspectivesJSON,
			&d.ClusterID, &d.ProcessedDate, &d.ArticleCount, &d.DateGenerated,
			&themesJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan digest row: %w", err)
		}

		// Unmarshal JSONB fields
		if len(keyMomentsJSON) > 0 && string(keyMomentsJSON) != "null" {
			if err := json.Unmarshal(keyMomentsJSON, &d.KeyMoments); err != nil {
				return nil, fmt.Errorf("failed to unmarshal key_moments: %w", err)
			}
		}

		if len(perspectivesJSON) > 0 && string(perspectivesJSON) != "null" {
			if err := json.Unmarshal(perspectivesJSON, &d.Perspectives); err != nil {
				return nil, fmt.Errorf("failed to unmarshal perspectives: %w", err)
			}
		}

		if len(themesJSON) > 0 && string(themesJSON) != "[]" {
			if err := json.Unmarshal(themesJSON, &d.Themes); err != nil {
				return nil, fmt.Errorf("failed to unmarshal themes: %w", err)
			}
		}

		digests = append(digests, d)
	}

	return digests, rows.Err()
}

// ListByTheme retrieves digests associated with a specific theme (v2.0)
func (r *postgresDigestRepo) ListByTheme(ctx context.Context, themeID string, since time.Time, limit int) ([]core.Digest, error) {
	// TODO: Implement v2.0 theme-based digest filtering
	// For now, return empty list
	return []core.Digest{}, nil
}

// ListByCluster retrieves digests for a specific HDBSCAN cluster (v2.0)
func (r *postgresDigestRepo) ListByCluster(ctx context.Context, clusterID int, limit int) ([]core.Digest, error) {
	// TODO: Implement v2.0 cluster-based digest filtering
	// For now, return empty list
	return []core.Digest{}, nil
}

// GetByID retrieves a digest by ID (alias for Get, for API consistency)
func (r *postgresDigestRepo) GetByID(ctx context.Context, id string) (*core.Digest, error) {
	return r.Get(ctx, id)
}

// GetDigestArticles retrieves all articles associated with a specific digest
func (r *postgresDigestRepo) GetDigestArticles(ctx context.Context, digestID string) ([]core.Article, error) {
	query := `
		SELECT
			a.id, a.url, a.title, a.content_type, a.cleaned_text, a.raw_content,
			a.topic_cluster, a.cluster_confidence, a.date_fetched, a.embedding
		FROM articles a
		INNER JOIN digest_articles da ON a.id = da.article_id
		WHERE da.digest_id = $1
		ORDER BY da.citation_number ASC`

	rows, err := r.query().QueryContext(ctx, query, digestID)
	if err != nil {
		return nil, fmt.Errorf("query articles for digest failed: %w", err)
	}
	defer rows.Close()

	var articles []core.Article
	for rows.Next() {
		var article core.Article
		var embedding []byte

		err := rows.Scan(
			&article.ID,
			&article.URL,
			&article.Title,
			&article.ContentType,
			&article.CleanedText,
			&article.RawContent,
			&article.TopicCluster,
			&article.ClusterConfidence,
			&article.DateFetched,
			&embedding,
		)
		if err != nil {
			return nil, fmt.Errorf("scan article failed: %w", err)
		}

		// Deserialize embedding if present
		if len(embedding) > 0 {
			if err := json.Unmarshal(embedding, &article.Embedding); err != nil {
				return nil, fmt.Errorf("unmarshal embedding failed: %w", err)
			}
		}

		articles = append(articles, article)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return articles, nil
}

// postgresThemeRepo implements ThemeRepository for PostgreSQL (Phase 0)
type postgresThemeRepo struct {
	db *sql.DB
	tx *sql.Tx
}

func (r *postgresThemeRepo) query() interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
} {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

func (r *postgresThemeRepo) Create(ctx context.Context, theme *core.Theme) error {
	query := `
		INSERT INTO themes (id, name, description, keywords, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	now := time.Now().UTC()
	_, err := r.query().ExecContext(ctx, query,
		theme.ID,
		theme.Name,
		theme.Description,
		pq.Array(theme.Keywords), // PostgreSQL text[] array
		theme.Enabled,
		now,
		now,
	)
	return err
}

func (r *postgresThemeRepo) Get(ctx context.Context, id string) (*core.Theme, error) {
	query := `SELECT id, name, description, keywords, enabled, created_at, updated_at FROM themes WHERE id = $1`
	row := r.query().QueryRowContext(ctx, query, id)
	return r.scanTheme(row)
}

func (r *postgresThemeRepo) GetByName(ctx context.Context, name string) (*core.Theme, error) {
	query := `SELECT id, name, description, keywords, enabled, created_at, updated_at FROM themes WHERE name = $1`
	row := r.query().QueryRowContext(ctx, query, name)
	return r.scanTheme(row)
}

func (r *postgresThemeRepo) List(ctx context.Context, enabledOnly bool) ([]core.Theme, error) {
	var query string
	var rows *sql.Rows
	var err error

	if enabledOnly {
		query = `SELECT id, name, description, keywords, enabled, created_at, updated_at FROM themes WHERE enabled = true ORDER BY name ASC`
		rows, err = r.query().QueryContext(ctx, query)
	} else {
		query = `SELECT id, name, description, keywords, enabled, created_at, updated_at FROM themes ORDER BY name ASC`
		rows, err = r.query().QueryContext(ctx, query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var themes []core.Theme
	for rows.Next() {
		theme, err := r.scanThemeRow(rows)
		if err != nil {
			return nil, err
		}
		themes = append(themes, *theme)
	}
	return themes, rows.Err()
}

func (r *postgresThemeRepo) ListEnabled(ctx context.Context) ([]core.Theme, error) {
	return r.List(ctx, true)
}

func (r *postgresThemeRepo) Update(ctx context.Context, theme *core.Theme) error {
	query := `
		UPDATE themes
		SET name = $2, description = $3, keywords = $4, enabled = $5, updated_at = $6
		WHERE id = $1
	`
	_, err := r.query().ExecContext(ctx, query,
		theme.ID,
		theme.Name,
		theme.Description,
		pq.Array(theme.Keywords),
		theme.Enabled,
		time.Now().UTC(),
	)
	return err
}

func (r *postgresThemeRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM themes WHERE id = $1`
	_, err := r.query().ExecContext(ctx, query, id)
	return err
}

func (r *postgresThemeRepo) scanTheme(row *sql.Row) (*core.Theme, error) {
	var theme core.Theme
	err := row.Scan(
		&theme.ID,
		&theme.Name,
		&theme.Description,
		pq.Array(&theme.Keywords),
		&theme.Enabled,
		&theme.CreatedAt,
		&theme.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("theme not found")
		}
		return nil, err
	}
	return &theme, nil
}

func (r *postgresThemeRepo) scanThemeRow(rows *sql.Rows) (*core.Theme, error) {
	var theme core.Theme
	err := rows.Scan(
		&theme.ID,
		&theme.Name,
		&theme.Description,
		pq.Array(&theme.Keywords),
		&theme.Enabled,
		&theme.CreatedAt,
		&theme.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &theme, nil
}

// postgresManualURLRepo implements ManualURLRepository for PostgreSQL (Phase 0)
type postgresManualURLRepo struct {
	db *sql.DB
	tx *sql.Tx
}

func (r *postgresManualURLRepo) query() interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
} {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

func (r *postgresManualURLRepo) Create(ctx context.Context, manualURL *core.ManualURL) error {
	query := `
		INSERT INTO manual_urls (id, url, submitted_by, status, error_message, processed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.query().ExecContext(ctx, query,
		manualURL.ID,
		manualURL.URL,
		manualURL.SubmittedBy,
		manualURL.Status,
		manualURL.ErrorMessage,
		manualURL.ProcessedAt,
		time.Now().UTC(),
	)
	return err
}

func (r *postgresManualURLRepo) CreateBatch(ctx context.Context, urls []string, submittedBy string) error {
	query := `
		INSERT INTO manual_urls (id, url, submitted_by, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	for _, url := range urls {
		id := fmt.Sprintf("mu_%d", time.Now().UnixNano())
		_, err := r.query().ExecContext(ctx, query,
			id,
			url,
			submittedBy,
			core.ManualURLStatusPending,
			time.Now().UTC(),
		)
		if err != nil {
			return fmt.Errorf("failed to insert URL %s: %w", url, err)
		}
	}
	return nil
}

func (r *postgresManualURLRepo) Get(ctx context.Context, id string) (*core.ManualURL, error) {
	query := `SELECT id, url, submitted_by, status, error_message, processed_at, created_at FROM manual_urls WHERE id = $1`
	row := r.query().QueryRowContext(ctx, query, id)
	return r.scanManualURL(row)
}

func (r *postgresManualURLRepo) List(ctx context.Context, opts ListOptions) ([]core.ManualURL, error) {
	limit := opts.Limit
	if limit == 0 {
		limit = 100
	}
	query := `SELECT id, url, submitted_by, status, error_message, processed_at, created_at FROM manual_urls ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.query().QueryContext(ctx, query, limit, opts.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var manualURLs []core.ManualURL
	for rows.Next() {
		manualURL, err := r.scanManualURLRow(rows)
		if err != nil {
			return nil, err
		}
		manualURLs = append(manualURLs, *manualURL)
	}
	return manualURLs, rows.Err()
}

func (r *postgresManualURLRepo) GetPending(ctx context.Context, limit int) ([]core.ManualURL, error) {
	if limit == 0 {
		limit = 100
	}
	query := `SELECT id, url, submitted_by, status, error_message, processed_at, created_at FROM manual_urls WHERE status = $1 ORDER BY created_at ASC LIMIT $2`
	rows, err := r.query().QueryContext(ctx, query, core.ManualURLStatusPending, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var manualURLs []core.ManualURL
	for rows.Next() {
		manualURL, err := r.scanManualURLRow(rows)
		if err != nil {
			return nil, err
		}
		manualURLs = append(manualURLs, *manualURL)
	}
	return manualURLs, rows.Err()
}

func (r *postgresManualURLRepo) GetByURL(ctx context.Context, url string) (*core.ManualURL, error) {
	query := `SELECT id, url, submitted_by, status, error_message, processed_at, created_at FROM manual_urls WHERE url = $1`
	row := r.query().QueryRowContext(ctx, query, url)
	return r.scanManualURL(row)
}

func (r *postgresManualURLRepo) GetByStatus(ctx context.Context, status string, limit int) ([]core.ManualURL, error) {
	if limit == 0 {
		limit = 100
	}
	query := `SELECT id, url, submitted_by, status, error_message, processed_at, created_at FROM manual_urls WHERE status = $1 ORDER BY created_at DESC LIMIT $2`
	rows, err := r.query().QueryContext(ctx, query, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var manualURLs []core.ManualURL
	for rows.Next() {
		manualURL, err := r.scanManualURLRow(rows)
		if err != nil {
			return nil, err
		}
		manualURLs = append(manualURLs, *manualURL)
	}
	return manualURLs, rows.Err()
}

func (r *postgresManualURLRepo) UpdateStatus(ctx context.Context, id string, status string, errorMessage string) error {
	query := `UPDATE manual_urls SET status = $2, error_message = $3, processed_at = $4 WHERE id = $1`
	var processedAt *time.Time
	if status == core.ManualURLStatusProcessed || status == core.ManualURLStatusFailed {
		now := time.Now().UTC()
		processedAt = &now
	}
	_, err := r.query().ExecContext(ctx, query, id, status, errorMessage, processedAt)
	return err
}

func (r *postgresManualURLRepo) MarkProcessed(ctx context.Context, id string) error {
	return r.UpdateStatus(ctx, id, core.ManualURLStatusProcessed, "")
}

func (r *postgresManualURLRepo) MarkFailed(ctx context.Context, id string, errorMessage string) error {
	return r.UpdateStatus(ctx, id, core.ManualURLStatusFailed, errorMessage)
}

func (r *postgresManualURLRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM manual_urls WHERE id = $1`
	_, err := r.query().ExecContext(ctx, query, id)
	return err
}

func (r *postgresManualURLRepo) scanManualURL(row *sql.Row) (*core.ManualURL, error) {
	var manualURL core.ManualURL
	err := row.Scan(
		&manualURL.ID,
		&manualURL.URL,
		&manualURL.SubmittedBy,
		&manualURL.Status,
		&manualURL.ErrorMessage,
		&manualURL.ProcessedAt,
		&manualURL.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("manual URL not found")
		}
		return nil, err
	}
	return &manualURL, nil
}

func (r *postgresManualURLRepo) scanManualURLRow(rows *sql.Rows) (*core.ManualURL, error) {
	var manualURL core.ManualURL
	err := rows.Scan(
		&manualURL.ID,
		&manualURL.URL,
		&manualURL.SubmittedBy,
		&manualURL.Status,
		&manualURL.ErrorMessage,
		&manualURL.ProcessedAt,
		&manualURL.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &manualURL, nil
}
