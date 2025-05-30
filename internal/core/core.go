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
	ID          string    `json:"id"`           // Unique identifier for the article
	LinkID      string    `json:"link_id"`      // Identifier of the source Link
	Title       string    `json:"title"`        // Title of the article
	FetchedHTML string    `json:"fetched_html"` // Raw HTML content fetched
	CleanedText string    `json:"cleaned_text"` // Cleaned and parsed text content
	DateFetched time.Time `json:"date_fetched"` // Timestamp when the article was fetched
	MyTake      string    `json:"my_take"`      // Optional user's note on the article (can be empty)
}

// Summary represents a summarized version of one or more articles.
type Summary struct {
	ID            string    `json:"id"`             // Unique identifier for the summary
	ArticleIDs    []string  `json:"article_ids"`    // IDs of the articles this summary is based on
	SummaryText   string    `json:"summary_text"`   // The generated summary text
	ModelUsed     string    `json:"model_used"`     // LLM model used for summarization
	Length        string    `json:"length"`         // Target length/style (e.g., "short", "detailed")
	Instructions  string    `json:"instructions"`   // Instructions/prompt used for summarization
	MyTake        string    `json:"my_take"`        // User's take on the summary itself
	DateGenerated time.Time `json:"date_generated"` // Timestamp when the summary was generated
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
