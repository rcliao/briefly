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

// Article represents the content fetched and processed from a Link (v3.0 simplified)
type Article struct {
	// Core identity
	ID              string      `json:"id"`
	URL             string      `json:"url"`              // Direct URL (no LinkID indirection)
	Title           string      `json:"title"`
	ContentType     ContentType `json:"content_type"`     // html, pdf, youtube
	
	// Content
	CleanedText     string    `json:"cleaned_text"`
	RawContent      string    `json:"raw_content,omitempty"` // For non-HTML
	
	// Processing metadata
	DateFetched     time.Time `json:"date_fetched"`
	ProcessingMode  string    `json:"processing_mode"` // local, cloud, hybrid
	
	// Intelligence
	TopicCluster      string  `json:"topic_cluster"`
	ClusterConfidence float64 `json:"cluster_confidence"`
	QualityScore      float64 `json:"quality_score"`    // 0.0-1.0
	SignalStrength    float64 `json:"signal_strength"`  // 0.0-1.0 (replaces RelevanceScore)
	
	// Content-specific metadata (conditional)
	Duration  int    `json:"duration,omitempty"`   // YouTube only
	Channel   string `json:"channel,omitempty"`    // YouTube only
	PageCount int    `json:"page_count,omitempty"` // PDF only
	
	// User interaction
	ExplorationCount int       `json:"exploration_count"` // How often user clicked through
	UserRating       *float64  `json:"user_rating,omitempty"` // 1-5 stars
	Notes            string    `json:"notes,omitempty"`
	
	// Backward compatibility (deprecated in v3.0)
	LinkID           string    `json:"link_id,omitempty"`        // Legacy
	FetchedHTML      string    `json:"fetched_html,omitempty"`   // Legacy
	MyTake           string    `json:"my_take,omitempty"`        // Legacy
	Embedding        []float64 `json:"embedding,omitempty"`      // Legacy (expensive)
	TopicConfidence  float64   `json:"topic_confidence,omitempty"` // Legacy naming
	SentimentScore   float64   `json:"sentiment_score,omitempty"`  // Legacy
	SentimentLabel   string    `json:"sentiment_label,omitempty"`  // Legacy
	SentimentEmoji   string    `json:"sentiment_emoji,omitempty"`  // Legacy
	AlertTriggered   bool      `json:"alert_triggered,omitempty"`  // Legacy
	AlertConditions  []string  `json:"alert_conditions,omitempty"` // Legacy
	ResearchQueries  []string  `json:"research_queries,omitempty"` // Legacy
	RelevanceScore   float64   `json:"relevance_score,omitempty"`  // Legacy
	FileSize         int64     `json:"file_size,omitempty"`        // Legacy
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

// Digest represents a complete digest with user's take (v3.0 simplified)
type Digest struct {
	ID               string         `json:"id"`
	
	// v3.0 new structure
	Signal           Signal         `json:"signal,omitempty"`            // Primary insight
	ArticleGroups    []ArticleGroup `json:"article_groups,omitempty"`    // Clustered articles
	Metadata         DigestMetadata `json:"metadata,omitempty"`
	
	// User interaction
	UserFeedback     *UserFeedback  `json:"user_feedback,omitempty"`
	ExplorationPaths []string       `json:"exploration_paths,omitempty"` // What user clicked
	
	// Legacy fields (maintained for backward compatibility)
	Title         string    `json:"title,omitempty"`          // Legacy title
	Content       string    `json:"content,omitempty"`        // Legacy full digest content
	DigestSummary string    `json:"digest_summary,omitempty"` // Legacy executive summary
	MyTake        string    `json:"my_take,omitempty"`        // Legacy user take
	ArticleURLs   []string  `json:"article_urls,omitempty"`   // Legacy URL list
	ModelUsed     string    `json:"model_used,omitempty"`     // Legacy model info
	Format        string    `json:"format,omitempty"`         // Legacy format
	DateGenerated time.Time `json:"date_generated,omitempty"` // Legacy timestamp
	
	// Legacy insights fields  
	OverallSentiment    string   `json:"overall_sentiment,omitempty"`    // Legacy sentiment
	AlertsSummary       string   `json:"alerts_summary,omitempty"`       // Legacy alerts
	TrendsSummary       string   `json:"trends_summary,omitempty"`       // Legacy trends
	ResearchSuggestions []string `json:"research_suggestions,omitempty"` // Legacy research
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

// ProcessingCost tracks AI usage and costs
type ProcessingCost struct {
	LocalTokens  int     `json:"local_tokens"`
	CloudTokens  int     `json:"cloud_tokens"`
	EstimatedUSD float64 `json:"estimated_usd"`
}

// ActionItem represents a suggested action for the user
type ActionItem struct {
	Description string `json:"description"` // What to do
	Effort      string `json:"effort"`      // low, medium, high
	Timeline    string `json:"timeline"`    // immediate, this_week, this_month
}

// Signal represents synthesized insight from multiple articles (v3.0)
type Signal struct {
	ID              string         `json:"id"`
	Content         string         `json:"content"`          // The synthesized insight (60-80 words)
	SourceArticles  []string       `json:"source_articles"`  // Article IDs
	Confidence      float64        `json:"confidence"`       // 0.0-1.0
	Theme           string         `json:"theme"`            // Primary theme
	Implications    []string       `json:"implications"`     // What it means
	Actions         []ActionItem   `json:"actions"`          // Suggested actions
	RelatedSignals  []string       `json:"related_signals"`  // For connection building
	DateGenerated   time.Time      `json:"date_generated"`
	ProcessingCost  ProcessingCost `json:"processing_cost"`  // Track AI usage
}

// ArticleGroup represents a cluster of related articles (v3.0)
type ArticleGroup struct {
	Category    string    `json:"category"`    // "Breaking", "Tools", "Analysis"
	Theme       string    `json:"theme"`       // "AI Context Scaling", "Cost Optimization"
	Articles    []Article `json:"articles"`
	Summary     string    `json:"summary"`     // Group-level insight (50 words max)
	Priority    int       `json:"priority"`    // 1-5 for ordering
}

// DigestMetadata contains digest processing information (v3.0)
type DigestMetadata struct {
	Title           string         `json:"title"`
	DateGenerated   time.Time      `json:"date_generated"`
	WordCount       int            `json:"word_count"`
	ArticleCount    int            `json:"article_count"`
	ProcessingTime  time.Duration  `json:"processing_time"`
	ProcessingCost  ProcessingCost `json:"processing_cost"`
	QualityScore    float64        `json:"quality_score"`    // Overall digest quality
}

// UserFeedback captures user ratings and comments (v3.0)
type UserFeedback struct {
	Rating          int       `json:"rating"`           // 1-5 stars
	SignalQuality   int       `json:"signal_quality"`   // 1-5 stars
	Completeness    int       `json:"completeness"`     // 1-5 stars
	Actionability   int       `json:"actionability"`    // 1-5 stars
	Comments        string    `json:"comments"`
	DateProvided    time.Time `json:"date_provided"`
}

// UserProfile represents user preferences and context (v3.0)
type UserProfile struct {
	PreferLocal      bool    `json:"prefer_local"`       // Prefer local when possible
	MaxCloudCost     float64 `json:"max_cloud_cost"`     // USD per operation
	QualityThreshold float64 `json:"quality_threshold"`  // Minimum acceptable quality
}

// ResearchSession represents an interactive research session (v3.0)
type ResearchSession struct {
	ID              string             `json:"id"`
	InitialQuery    string             `json:"initial_query"`
	CurrentState    ResearchState      `json:"current_state"`
	ConversationLog []ConversationTurn `json:"conversation_log"`
	DiscoveredItems []ResearchItem     `json:"discovered_items"`
	QueuedForDigest []string           `json:"queued_for_digest"`  // Article URLs
	StartTime       time.Time          `json:"start_time"`
	LastActivity    time.Time          `json:"last_activity"`
	ProcessingCost  ProcessingCost     `json:"processing_cost"`
}

// ResearchState tracks current research session state
type ResearchState struct {
	Phase            string   `json:"phase"`             // "overview", "deep_dive", "exploration"
	CurrentTopic     string   `json:"current_topic"`
	AvailableActions []string `json:"available_actions"` // What user can do next
	Progress         float64  `json:"progress"`          // 0.0-1.0
}

// ConversationTurn represents one exchange in research session
type ConversationTurn struct {
	Timestamp      time.Time `json:"timestamp"`
	UserInput      string    `json:"user_input"`
	SystemAction   string    `json:"system_action"`   // "search", "analyze", "synthesize"
	Response       string    `json:"response"`
	ProcessingMode string    `json:"processing_mode"` // "local", "cloud"
}

// ResearchItem represents discovered content during research
type ResearchItem struct {
	URL             string    `json:"url"`
	Title           string    `json:"title"`
	Summary         string    `json:"summary"`
	Relevance       float64   `json:"relevance"`
	QualityScore    float64   `json:"quality_score"`
	Category        string    `json:"category"`
	UserInterest    *int      `json:"user_interest,omitempty"` // 1-5 if rated
	AddedToQueue    bool      `json:"added_to_queue"`
	DateDiscovered  time.Time `json:"date_discovered"`
}
