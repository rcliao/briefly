package core

import "time"

// Link represents a URL to be processed.
type Link struct {
	ID        string    `json:"id"`         // Unique identifier for the link
	URL       string    `json:"url"`        // The URL string
	DateAdded time.Time `json:"date_added"` // Timestamp when the link was added
	Source    string    `json:"source"`     // Source of the link (e.g., "file", "rss", "deep_research")
}

// ContentType represents the type of content being processed
type ContentType string

const (
	ContentTypeHTML    ContentType = "html"
	ContentTypePDF     ContentType = "pdf"
	ContentTypeYouTube ContentType = "youtube"
)

// Article represents the content fetched and processed from a Link.
type Article struct {
	ID              string      `json:"id"`               // Unique identifier for the article
	LinkID          string      `json:"link_id"`          // Identifier of the source Link
	Title           string      `json:"title"`            // Title of the article
	ContentType     ContentType `json:"content_type"`     // Type of content (html, pdf, youtube)
	FetchedHTML     string      `json:"fetched_html"`     // Raw HTML content fetched
	RawContent      string      `json:"raw_content"`      // Raw content for non-HTML types (PDF text, YouTube transcript)
	CleanedText     string      `json:"cleaned_text"`     // Cleaned and parsed text content
	DateFetched     time.Time   `json:"date_fetched"`     // Timestamp when the article was fetched
	MyTake          string      `json:"my_take"`          // Optional user's note on the article (can be empty)
	Embedding       []float64   `json:"embedding"`        // Vector embedding of the article content
	TopicCluster    string      `json:"topic_cluster"`    // Assigned topic cluster label
	TopicConfidence float64     `json:"topic_confidence"` // Confidence score for topic assignment
	// Content-specific metadata
	Duration  int    `json:"duration,omitempty"`   // Video duration in seconds (YouTube only)
	Channel   string `json:"channel,omitempty"`    // Channel name (YouTube only)
	FileSize  int64  `json:"file_size,omitempty"`  // File size in bytes (PDF only)
	PageCount int    `json:"page_count,omitempty"` // Number of pages (PDF only)
	// v0.4 Insights fields
	SentimentScore  float64  `json:"sentiment_score"`  // Sentiment analysis score (-1.0 to 1.0)
	SentimentLabel  string   `json:"sentiment_label"`  // Sentiment label (positive, negative, neutral)
	SentimentEmoji  string   `json:"sentiment_emoji"`  // Emoji representation of sentiment
	AlertTriggered  bool     `json:"alert_triggered"`  // Whether this article triggered any alerts
	AlertConditions []string `json:"alert_conditions"` // List of alert conditions that matched
	ResearchQueries []string `json:"research_queries"` // Generated research queries for this article
}

// Summary represents a summarized version of one or more articles.
type Summary struct {
	ID              string    `json:"id"`               // Unique identifier for the summary
	ArticleIDs      []string  `json:"article_ids"`      // IDs of the articles this summary is based on
	SummaryText     string    `json:"summary_text"`     // The generated summary text
	ModelUsed       string    `json:"model_used"`       // LLM model used for summarization
	Length          string    `json:"length"`           // Target length/style (e.g., "short", "detailed")
	Instructions    string    `json:"instructions"`     // Instructions/prompt used for summarization
	MyTake          string    `json:"my_take"`          // User's take on the summary itself
	DateGenerated   time.Time `json:"date_generated"`   // Timestamp when the summary was generated
	Embedding       []float64 `json:"embedding"`        // Vector embedding of the summary content
	TopicCluster    string    `json:"topic_cluster"`    // Assigned topic cluster label
	TopicConfidence float64   `json:"topic_confidence"` // Confidence score for topic assignment
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
	OverallSentiment    string   `json:"overall_sentiment"`    // Overall sentiment of the digest (positive, negative, neutral, mixed)
	AlertsSummary       string   `json:"alerts_summary"`       // Summary of alerts triggered in this digest
	TrendsSummary       string   `json:"trends_summary"`       // Summary of trends identified in this digest
	ResearchSuggestions []string `json:"research_suggestions"` // Suggested research queries for follow-up
}

// Prompt represents a generic prompt that can be used for various LLM interactions.
type Prompt struct {
	ID           string    `json:"id"`             // Unique identifier for the prompt
	Text         string    `json:"text"`           // The prompt text
	Category     string    `json:"category"`       // Category of the prompt (e.g., "critique", "questions")
	CreationDate time.Time `json:"creation_date"`  // Timestamp when the prompt was created
	LastUsedDate time.Time `json:"last_used_date"` // Timestamp when the prompt was last used (zero value if never)
	UsageCount   int       `json:"usage_count"`    // How many times this prompt has been used
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
	ID             string    `json:"id"`              // Unique identifier for the feed item
	FeedID         string    `json:"feed_id"`         // ID of the parent feed
	Title          string    `json:"title"`           // Item title
	Link           string    `json:"link"`            // Item URL
	Description    string    `json:"description"`     // Item description/summary
	Published      time.Time `json:"published"`       // Publication date
	GUID           string    `json:"guid"`            // Unique identifier from the feed
	Processed      bool      `json:"processed"`       // Whether the item has been processed
	DateDiscovered time.Time `json:"date_discovered"` // When the item was discovered
}

// TopicCluster represents a cluster of articles with similar topics.
type TopicCluster struct {
	ID         string    `json:"id"`          // Unique identifier for the cluster
	Label      string    `json:"label"`       // Human-readable topic label
	Keywords   []string  `json:"keywords"`    // Key terms associated with this topic
	ArticleIDs []string  `json:"article_ids"` // IDs of articles in this cluster
	Centroid   []float64 `json:"centroid"`    // Cluster centroid in embedding space
	CreatedAt  time.Time `json:"created_at"`  // When the cluster was created
}

// CacheStats represents statistics about the cache.
type CacheStats struct {
	ArticleCount  int       `json:"article_count"`   // Number of cached articles
	SummaryCount  int       `json:"summary_count"`   // Number of cached summaries
	DigestCount   int       `json:"digest_count"`    // Number of cached digests
	FeedCount     int       `json:"feed_count"`      // Number of RSS feeds
	FeedItemCount int       `json:"feed_item_count"` // Number of feed items
	CacheSize     int64     `json:"cache_size"`      // Total cache size in bytes
	LastUpdated   time.Time `json:"last_updated"`    // Last cache update time
}

// ResearchReport represents the results of a research operation
type ResearchReport struct {
	ID               string           `json:"id"`                // Unique identifier for the research report
	Query            string           `json:"query"`             // Original research query
	Depth            int              `json:"depth"`             // Research depth level
	GeneratedQueries []string         `json:"generated_queries"` // AI-generated search queries
	Results          []ResearchResult `json:"results"`           // Research results
	Summary          string           `json:"summary"`           // Summary of findings
	DateGenerated    time.Time        `json:"date_generated"`    // When the research was conducted
	TotalResults     int              `json:"total_results"`     // Total number of results found
	RelevanceScore   float64          `json:"relevance_score"`   // Overall relevance score
}

// ResearchResult represents a single research result
type ResearchResult struct {
	ID        string    `json:"id"`         // Unique identifier
	Title     string    `json:"title"`      // Result title
	URL       string    `json:"url"`        // Result URL
	Snippet   string    `json:"snippet"`    // Result snippet/description
	Source    string    `json:"source"`     // Source (Google, DuckDuckGo, etc.)
	Relevance float64   `json:"relevance"`  // Relevance score (0-1)
	DateFound time.Time `json:"date_found"` // When this result was found
	Keywords  []string  `json:"keywords"`   // Extracted keywords
}

// FeedAnalysisReport represents analysis of RSS feed content
type FeedAnalysisReport struct {
	ID               string     `json:"id"`                // Unique identifier for the report
	DateGenerated    time.Time  `json:"date_generated"`    // When the analysis was performed
	FeedsAnalyzed    int        `json:"feeds_analyzed"`    // Number of feeds analyzed
	ItemsAnalyzed    int        `json:"items_analyzed"`    // Number of feed items analyzed
	TopTopics        []string   `json:"top_topics"`        // Most common topics
	TrendingKeywords []string   `json:"trending_keywords"` // Trending keywords
	RecommendedItems []FeedItem `json:"recommended_items"` // Recommended items for digest
	QualityScore     float64    `json:"quality_score"`     // Overall quality score
	Summary          string     `json:"summary"`           // Summary of analysis
}

// BannerImage represents a generated banner image for a digest
type BannerImage struct {
	ID          string    `json:"id"`           // Unique identifier for the banner
	DigestID    string    `json:"digest_id"`    // Associated digest ID
	ImageURL    string    `json:"image_url"`    // URL/path to generated image
	PromptUsed  string    `json:"prompt_used"`  // DALL-E prompt used for generation
	Style       string    `json:"style"`        // Banner style (minimalist, tech, etc.)
	Themes      []string  `json:"themes"`       // Identified content themes
	GeneratedAt time.Time `json:"generated_at"` // When the banner was generated
	FileSize    int64     `json:"file_size"`    // File size in bytes
	Width       int       `json:"width"`        // Image width in pixels
	Height      int       `json:"height"`       // Image height in pixels
	Format      string    `json:"format"`       // Image format (JPEG, PNG)
	AltText     string    `json:"alt_text"`     // Accessibility alt text
}

// ContentTheme represents a thematic analysis of digest content
type ContentTheme struct {
	Theme       string   `json:"theme"`       // Primary theme (AI, Security, Development, etc.)
	Keywords    []string `json:"keywords"`    // Key terms associated with this theme
	Articles    []string `json:"articles"`    // Article IDs contributing to this theme
	Confidence  float64  `json:"confidence"`  // Confidence score (0-1)
	Category    string   `json:"category"`    // Visual category (🔧 Dev, 📚 Research, 💡 Insight)
	Description string   `json:"description"` // Theme description for prompt generation
}
