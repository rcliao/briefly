// Package feeds provides RSS/Atom feed parsing and management functionality
package feeds

import (
	"briefly/internal/core"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RSS represents an RSS feed structure
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

// Atom represents an Atom feed structure
type Atom struct {
	XMLName xml.Name    `xml:"feed"`
	Title   string      `xml:"title"`
	Link    []AtomLink  `xml:"link"`
	Entries []AtomEntry `xml:"entry"`
}

// Channel represents an RSS channel
type Channel struct {
	Title       string    `xml:"title"`
	Description string    `xml:"description"`
	Link        string    `xml:"link"`
	Items       []RSSItem `xml:"item"`
}

// RSSItem represents an RSS item
type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

// AtomLink represents an Atom link element
type AtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

// AtomEntry represents an Atom entry
type AtomEntry struct {
	Title     string     `xml:"title"`
	Link      []AtomLink `xml:"link"`
	Summary   string     `xml:"summary"`
	Published string     `xml:"published"`
	Updated   string     `xml:"updated"`
	ID        string     `xml:"id"`
}

// FeedManager manages RSS/Atom feed operations
type FeedManager struct {
	client *http.Client
}

// NewFeedManager creates a new feed manager
func NewFeedManager() *FeedManager {
	return &FeedManager{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchFeed fetches and parses a feed from the given URL
func (fm *FeedManager) FetchFeed(feedURL string, lastModified, etag string) (*ParsedFeed, error) {
	req, err := http.NewRequest("GET", feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set conditional headers for efficient fetching
	if lastModified != "" {
		req.Header.Set("If-Modified-Since", lastModified)
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	req.Header.Set("User-Agent", "Briefly RSS Reader/1.0")

	resp, err := fm.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Handle not modified response
	if resp.StatusCode == http.StatusNotModified {
		return &ParsedFeed{NotModified: true}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feed returned status %d", resp.StatusCode)
	}

	// Try to parse as RSS first, then Atom
	parsedFeed, err := fm.parseResponse(resp, feedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	// Extract caching headers
	parsedFeed.LastModified = resp.Header.Get("Last-Modified")
	parsedFeed.ETag = resp.Header.Get("ETag")

	return parsedFeed, nil
}

// ParsedFeed represents a parsed feed with metadata
type ParsedFeed struct {
	Feed         core.Feed
	Items        []core.FeedItem
	LastModified string
	ETag         string
	NotModified  bool
}

// parseResponse attempts to parse the HTTP response as either RSS or Atom
func (fm *FeedManager) parseResponse(resp *http.Response, feedURL string) (*ParsedFeed, error) {
	// Read and decode the response
	decoder := xml.NewDecoder(resp.Body)

	// Try RSS first
	var rss RSS
	if err := decoder.Decode(&rss); err == nil && rss.Channel.Title != "" {
		return fm.parseRSS(rss, feedURL), nil
	}

	// Reset and try Atom
	_ = resp.Body.Close()
	resp, err := fm.client.Get(feedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to re-fetch for Atom parsing: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	decoder = xml.NewDecoder(resp.Body)
	var atom Atom
	if err := decoder.Decode(&atom); err == nil && atom.Title != "" {
		return fm.parseAtom(atom, feedURL), nil
	}

	return nil, fmt.Errorf("unable to parse as RSS or Atom feed")
}

// parseRSS converts RSS data to core types
func (fm *FeedManager) parseRSS(rss RSS, feedURL string) *ParsedFeed {
	feed := core.Feed{
		ID:          generateFeedID(feedURL),
		URL:         feedURL,
		Title:       rss.Channel.Title,
		Description: rss.Channel.Description,
		Active:      true,
		DateAdded:   time.Now().UTC(),
	}

	var items []core.FeedItem
	for _, item := range rss.Channel.Items {
		feedItem := core.FeedItem{
			ID:              generateItemID(feed.ID, item.Link),
			FeedID:          feed.ID,
			Title:           item.Title,
			Link:            item.Link,
			Description:     item.Description,
			GUID:            item.GUID,
			Published:       parseRSSDate(item.PubDate),
			DateDiscovered:  time.Now().UTC(),
			Processed:       false,
		}
		items = append(items, feedItem)
	}

	return &ParsedFeed{
		Feed:  feed,
		Items: items,
	}
}

// parseAtom converts Atom data to core types
func (fm *FeedManager) parseAtom(atom Atom, feedURL string) *ParsedFeed {
	feed := core.Feed{
		ID:          generateFeedID(feedURL),
		URL:         feedURL,
		Title:       atom.Title,
		Description: "",
		Active:      true,
		DateAdded:   time.Now().UTC(),
	}

	var items []core.FeedItem
	for _, entry := range atom.Entries {
		// Find the main link
		var link string
		for _, l := range entry.Link {
			if l.Rel == "" || l.Rel == "alternate" {
				link = l.Href
				break
			}
		}

		feedItem := core.FeedItem{
			ID:              generateItemID(feed.ID, link),
			FeedID:          feed.ID,
			Title:           entry.Title,
			Link:            link,
			Description:     entry.Summary,
			GUID:            entry.ID,
			Published:       parseAtomDate(entry.Published),
			DateDiscovered:  time.Now().UTC(),
			Processed:       false,
		}
		items = append(items, feedItem)
	}

	return &ParsedFeed{
		Feed:  feed,
		Items: items,
	}
}

// generateFeedID creates a deterministic ID for a feed based on its URL
func generateFeedID(feedURL string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(feedURL)).String()
}

// generateItemID creates a deterministic ID for a feed item
func generateItemID(feedID, link string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(feedID+link)).String()
}

// parseRSSDate parses RSS date formats
func parseRSSDate(dateStr string) time.Time {
	if dateStr == "" {
		return time.Time{}
	}

	// Common RSS date formats
	formats := []string{
		time.RFC1123,
		time.RFC1123Z,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 MST",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, strings.TrimSpace(dateStr)); err == nil {
			return t.UTC()
		}
	}

	return time.Time{}
}

// parseAtomDate parses Atom date formats
func parseAtomDate(dateStr string) time.Time {
	if dateStr == "" {
		return time.Time{}
	}

	// Atom uses RFC3339
	if t, err := time.Parse(time.RFC3339, strings.TrimSpace(dateStr)); err == nil {
		return t.UTC()
	}

	// Fallback to common formats
	return parseRSSDate(dateStr)
}

// ValidateFeedURL checks if a URL appears to be a valid feed
func (fm *FeedManager) ValidateFeedURL(feedURL string) error {
	// Simple validation - try to fetch and parse
	_, err := fm.FetchFeed(feedURL, "", "")
	if err != nil {
		return fmt.Errorf("invalid feed URL: %w", err)
	}
	return nil
}

// DiscoverFeedURL attempts to discover feed URLs from a website
func (fm *FeedManager) DiscoverFeedURL(websiteURL string) ([]string, error) {
	resp, err := fm.client.Get(websiteURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch website: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// This is a simplified implementation
	// In a full implementation, you would parse HTML and look for:
	// <link rel="alternate" type="application/rss+xml" href="...">
	// <link rel="alternate" type="application/atom+xml" href="...">

	var candidates []string

	// Common feed URL patterns
	baseURL := strings.TrimSuffix(websiteURL, "/")
	candidates = append(candidates,
		baseURL+"/feed",
		baseURL+"/rss",
		baseURL+"/atom.xml",
		baseURL+"/rss.xml",
		baseURL+"/feed.xml",
		baseURL+"/feeds/all.atom.xml",
	)

	var validFeeds []string
	for _, candidate := range candidates {
		if err := fm.ValidateFeedURL(candidate); err == nil {
			validFeeds = append(validFeeds, candidate)
		}
	}

	return validFeeds, nil
}
