// Package citations provides citation tracking and metadata extraction for articles (Phase 1)
package citations

import (
	"briefly/internal/core"
	"briefly/internal/persistence"
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Tracker handles citation extraction and storage
type Tracker struct {
	db persistence.Database
}

// NewTracker creates a new citation tracker
func NewTracker(db persistence.Database) *Tracker {
	return &Tracker{
		db: db,
	}
}

// TrackArticle creates a citation record for an article
// Extracts metadata from the article and stores it as a citation
func (t *Tracker) TrackArticle(ctx context.Context, article *core.Article) (*core.Citation, error) {
	if article == nil {
		return nil, fmt.Errorf("article cannot be nil")
	}

	// Check if citation already exists
	existing, err := t.db.Citations().GetByArticleID(ctx, article.ID)
	if err == nil && existing != nil {
		return existing, nil // Already tracked
	}

	// Extract publisher from URL
	publisher := extractPublisher(article.URL)

	// Create citation
	citation := &core.Citation{
		ID:            uuid.NewString(),
		ArticleID:     article.ID,
		URL:           article.URL,
		Title:         article.Title,
		Publisher:     publisher,
		Author:        "",  // Could be extracted from article metadata if available
		PublishedDate: nil, // Could be extracted from article metadata if available
		AccessedDate:  article.DateFetched,
		Metadata:      make(map[string]interface{}),
		CreatedAt:     time.Now().UTC(),
	}

	// Add content type to metadata
	citation.Metadata["content_type"] = string(article.ContentType)

	// Store in database
	if err := t.db.Citations().Create(ctx, citation); err != nil {
		return nil, fmt.Errorf("failed to create citation: %w", err)
	}

	return citation, nil
}

// TrackBatch creates citation records for multiple articles
func (t *Tracker) TrackBatch(ctx context.Context, articles []core.Article) (map[string]*core.Citation, error) {
	citations := make(map[string]*core.Citation)
	errors := []error{}

	for _, article := range articles {
		citation, err := t.TrackArticle(ctx, &article)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to track article %s: %w", article.ID, err))
			continue
		}
		citations[article.ID] = citation
	}

	if len(errors) > 0 {
		return citations, fmt.Errorf("encountered %d errors during batch tracking", len(errors))
	}

	return citations, nil
}

// GetCitationsForArticles retrieves citations for multiple articles
func (t *Tracker) GetCitationsForArticles(ctx context.Context, articleIDs []string) (map[string]*core.Citation, error) {
	return t.db.Citations().GetByArticleIDs(ctx, articleIDs)
}

// GetCitation retrieves a citation by article ID
func (t *Tracker) GetCitation(ctx context.Context, articleID string) (*core.Citation, error) {
	return t.db.Citations().GetByArticleID(ctx, articleID)
}

// UpdateCitation updates an existing citation
func (t *Tracker) UpdateCitation(ctx context.Context, citation *core.Citation) error {
	return t.db.Citations().Update(ctx, citation)
}

// DeleteCitation removes a citation
func (t *Tracker) DeleteCitation(ctx context.Context, articleID string) error {
	return t.db.Citations().DeleteByArticleID(ctx, articleID)
}

// extractPublisher extracts the publisher/domain name from a URL
func extractPublisher(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	host := parsedURL.Hostname()

	// Remove www. prefix if present
	host = strings.TrimPrefix(host, "www.")

	// Extract base domain (e.g., "example.com" from "blog.example.com")
	parts := strings.Split(host, ".")
	if len(parts) >= 2 {
		// Return last two parts (domain.tld)
		return strings.Join(parts[len(parts)-2:], ".")
	}

	return host
}

// EnrichWithMetadata adds additional metadata to a citation
func (t *Tracker) EnrichWithMetadata(citation *core.Citation, metadata map[string]interface{}) {
	if citation.Metadata == nil {
		citation.Metadata = make(map[string]interface{})
	}

	for key, value := range metadata {
		citation.Metadata[key] = value
	}
}

// FormatCitation formats a citation in a standard citation format
func FormatCitation(citation *core.Citation, style string) string {
	switch style {
	case "apa":
		return formatAPA(citation)
	case "mla":
		return formatMLA(citation)
	case "chicago":
		return formatChicago(citation)
	default:
		return formatSimple(citation)
	}
}

// formatSimple creates a simple citation string
func formatSimple(c *core.Citation) string {
	parts := []string{}

	if c.Author != "" {
		parts = append(parts, c.Author)
	}

	if c.Title != "" {
		parts = append(parts, fmt.Sprintf("\"%s\"", c.Title))
	}

	if c.Publisher != "" {
		parts = append(parts, c.Publisher)
	}

	if c.PublishedDate != nil {
		parts = append(parts, c.PublishedDate.Format("2006-01-02"))
	}

	parts = append(parts, c.URL)

	if !c.AccessedDate.IsZero() {
		parts = append(parts, fmt.Sprintf("(accessed %s)", c.AccessedDate.Format("2006-01-02")))
	}

	return strings.Join(parts, ". ")
}

// formatAPA creates an APA-style citation
func formatAPA(c *core.Citation) string {
	parts := []string{}

	// Author (Year). Title. Publisher. URL
	if c.Author != "" {
		year := ""
		if c.PublishedDate != nil {
			year = fmt.Sprintf(" (%d)", c.PublishedDate.Year())
		}
		parts = append(parts, c.Author+year)
	}

	if c.Title != "" {
		parts = append(parts, c.Title)
	}

	if c.Publisher != "" {
		parts = append(parts, c.Publisher)
	}

	parts = append(parts, c.URL)

	return strings.Join(parts, ". ")
}

// formatMLA creates an MLA-style citation
func formatMLA(c *core.Citation) string {
	parts := []string{}

	// Author. "Title." Publisher, Date. URL.
	if c.Author != "" {
		parts = append(parts, c.Author)
	}

	if c.Title != "" {
		parts = append(parts, fmt.Sprintf("\"%s.\"", c.Title))
	}

	publisherDate := []string{}
	if c.Publisher != "" {
		publisherDate = append(publisherDate, c.Publisher)
	}
	if c.PublishedDate != nil {
		publisherDate = append(publisherDate, c.PublishedDate.Format("2 Jan. 2006"))
	}
	if len(publisherDate) > 0 {
		parts = append(parts, strings.Join(publisherDate, ", "))
	}

	parts = append(parts, c.URL)

	return strings.Join(parts, ". ") + "."
}

// formatChicago creates a Chicago-style citation
func formatChicago(c *core.Citation) string {
	parts := []string{}

	// Author. "Title." Publisher. Date. URL.
	if c.Author != "" {
		parts = append(parts, c.Author)
	}

	if c.Title != "" {
		parts = append(parts, fmt.Sprintf("\"%s.\"", c.Title))
	}

	if c.Publisher != "" {
		parts = append(parts, c.Publisher)
	}

	if c.PublishedDate != nil {
		parts = append(parts, c.PublishedDate.Format("January 2, 2006"))
	}

	parts = append(parts, c.URL)

	return strings.Join(parts, ". ") + "."
}
