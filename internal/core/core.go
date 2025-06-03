package core

import "time"

// Link represents a URL to be processed.
type Link struct {
	ID        string    `json:"id"`         // Unique identifier for the link
	URL       string    `json:"url"`        // The URL string
	DateAdded time.Time `json:"date_added"` // Timestamp when the link was added
	Source    string    `json:"source"`     // Source of the link (e.g., "file", "rss", "deep_research")
}

// Article represents the content fetched and processed from a Link.
type Article struct {
	ID               string    `json:"id"`               // Unique identifier for the article
	LinkID           string    `json:"link_id"`          // Identifier of the source Link
	Title            string    `json:"title"`            // Title of the article
	FetchedHTML      string    `json:"fetched_html"`     // Raw HTML content fetched
	CleanedText      string    `json:"cleaned_text"`     // Cleaned and parsed text content
	DateFetched      time.Time `json:"date_fetched"`     // Timestamp when the article was fetched
	MyTake           string    `json:"my_take"`          // Optional user's note on the article (can be empty)
	Embedding        []float64 `json:"embedding"`        // Vector embedding of the article content
	TopicCluster     string    `json:"topic_cluster"`    // Assigned topic cluster label
	TopicConfidence  float64   `json:"topic_confidence"` // Confidence score for topic assignment
	// v0.4 Insights fields
	SentimentScore   float64   `json:"sentiment_score"`   // Sentiment analysis score (-1.0 to 1.0)
	SentimentLabel   string    `json:"sentiment_label"`   // Sentiment label (positive, negative, neutral)
	SentimentEmoji   string    `json:"sentiment_emoji"`   // Emoji representation of sentiment
	AlertTriggered   bool      `json:"alert_triggered"`   // Whether this article triggered any alerts
	AlertConditions  []string  `json:"alert_conditions"`  // List of alert conditions that matched
	ResearchQueries  []string  `json:"research_queries"`  // Generated research queries for this article
}

// Summary represents a summarized version of one or more articles.
type Summary struct {
	ID               string    `json:"id"`               // Unique identifier for the summary
	ArticleIDs       []string  `json:"article_ids"`      // IDs of the articles this summary is based on
	SummaryText      string    `json:"summary_text"`     // The generated summary text
	ModelUsed        string    `json:"model_used"`       // LLM model used for summarization
	Length           string    `json:"length"`           // Target length/style (e.g., "short", "detailed")
	Instructions     string    `json:"instructions"`     // Instructions/prompt used for summarization
	MyTake           string    `json:"my_take"`          // User's take on the summary itself
	DateGenerated    time.Time `json:"date_generated"`   // Timestamp when the summary was generated
	Embedding        []float64 `json:"embedding"`        // Vector embedding of the summary content
	TopicCluster     string    `json:"topic_cluster"`    // Assigned topic cluster label
	TopicConfidence  float64   `json:"topic_confidence"` // Confidence score for topic assignment
}

// Digest represents a complete digest with user's take
type Digest struct {
	ID            string    `json:"id"`             // Unique identifier for the digest
	Title         string    `json:"title"`          // Title of the digest
	Content       string    `json:"content"`        // The full digest content
	DigestSummary string    `json:"digest_summary"` // Executive summary of the digest
	MyTake        string    `json:"my_take"`        // User's take on the entire digest
	ArticleURLs   []string  `json:"article_urls"`   // URLs of articles included in this digest
	ModelUsed     string    `json:"model_used"`     // LLM model used for digest generation
	Format        string    `json:"format"`         // Format used (brief, standard, detailed, newsletter)
	DateGenerated time.Time `json:"date_generated"` // Timestamp when the digest was generated
	// v0.4 Insights fields
	OverallSentiment  string    `json:"overall_sentiment"`   // Overall sentiment of the digest (positive, negative, neutral, mixed)
	AlertsSummary     string    `json:"alerts_summary"`      // Summary of alerts triggered in this digest
	TrendsSummary     string    `json:"trends_summary"`      // Summary of trends identified in this digest
	ResearchSuggestions []string `json:"research_suggestions"` // Suggested research queries for follow-up
}

// Prompt represents a generic prompt that can be used for various LLM interactions.
type Prompt struct {
	ID           string    `json:"id"`            // Unique identifier for the prompt
	Text         string    `json:"text"`          // The prompt text
	Category     string    `json:"category"`      // Category of the prompt (e.g., "critique", "questions")
	CreationDate time.Time `json:"creation_date"` // Timestamp when the prompt was created
	LastUsedDate time.Time `json:"last_used_date"`// Timestamp when the prompt was last used (zero value if never)
	UsageCount   int       `json:"usage_count"`   // How many times this prompt has been used
}

// Feed represents an RSS/Atom feed source.
type Feed struct {
	ID           string    `json:"id"`            // Unique identifier for the feed
	URL          string    `json:"url"`           // Feed URL
	Title        string    `json:"title"`         // Feed title
	Description  string    `json:"description"`   // Feed description
	LastFetched  time.Time `json:"last_fetched"`  // Last time the feed was fetched
	LastModified string    `json:"last_modified"` // Last-Modified header from the feed
	ETag         string    `json:"etag"`          // ETag header from the feed
	Active       bool      `json:"active"`        // Whether the feed is active for polling
	ErrorCount   int       `json:"error_count"`   // Number of consecutive errors
	LastError    string    `json:"last_error"`    // Last error encountered
	DateAdded    time.Time `json:"date_added"`    // When the feed was added
}

// FeedItem represents an item discovered in an RSS/Atom feed.
type FeedItem struct {
	ID              string    `json:"id"`               // Unique identifier for the feed item
	FeedID          string    `json:"feed_id"`          // ID of the parent feed
	Title           string    `json:"title"`            // Item title
	Link            string    `json:"link"`             // Item URL
	Description     string    `json:"description"`      // Item description/summary
	Published       time.Time `json:"published"`        // Publication date
	GUID            string    `json:"guid"`             // Unique identifier from the feed
	Processed       bool      `json:"processed"`        // Whether the item has been processed
	DateDiscovered  time.Time `json:"date_discovered"`  // When the item was discovered
}

// TopicCluster represents a cluster of articles with similar topics.
type TopicCluster struct {
	ID          string   `json:"id"`          // Unique identifier for the cluster
	Label       string   `json:"label"`       // Human-readable topic label
	Keywords    []string `json:"keywords"`    // Key terms associated with this topic
	ArticleIDs  []string `json:"article_ids"` // IDs of articles in this cluster
	Centroid    []float64 `json:"centroid"`   // Cluster centroid in embedding space
	CreatedAt   time.Time `json:"created_at"` // When the cluster was created
}

// CacheStats represents statistics about the cache.
type CacheStats struct {
	ArticleCount  int       `json:"article_count"`  // Number of cached articles
	SummaryCount  int       `json:"summary_count"`  // Number of cached summaries
	DigestCount   int       `json:"digest_count"`   // Number of cached digests
	FeedCount     int       `json:"feed_count"`     // Number of RSS feeds
	FeedItemCount int       `json:"feed_item_count"`// Number of feed items
	CacheSize     int64     `json:"cache_size"`     // Total cache size in bytes
	LastUpdated   time.Time `json:"last_updated"`   // Last cache update time
}
