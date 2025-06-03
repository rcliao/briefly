package render

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DigestData combines information needed for rendering a single item in the digest.
// This can be expanded as more complex data is needed for rendering.
type DigestData struct {
	Title           string  // Article Title
	URL             string  // Original Article URL
	SummaryText     string  // Summary text from core.Summary
	MyTake          string  // User's take from core.Article or core.Summary (optional)
	TopicCluster    string  // Assigned topic cluster label
	TopicConfidence float64 // Confidence score for topic assignment
	// v0.4 Insights fields
	SentimentScore   float64  // Sentiment analysis score (-1.0 to 1.0)
	SentimentLabel   string   // Sentiment label (positive, negative, neutral) 
	SentimentEmoji   string   // Emoji representation of sentiment
	AlertTriggered   bool     // Whether this article triggered any alerts
	AlertConditions  []string // List of alert conditions that matched
	ResearchQueries  []string // Generated research queries for this article
}

// RenderMarkdownDigest creates a markdown file with the given summaries.
// It will require a list of DigestData which includes article titles and URLs along with summaries.
// If finalDigest is provided, it will be used as the main content with individual summaries as appendix.
// If finalDigest is empty, individual summaries will be used as the main content.
func RenderMarkdownDigest(digestItems []DigestData, outputDir string, finalDigest string) (string, error) {
	dateStr := time.Now().UTC().Format("2006-01-02")
	filename := fmt.Sprintf("digest_%s.md", dateStr)

	if outputDir == "" {
		outputDir = "digests" // Default output directory
	}

	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	filePath := filepath.Join(outputDir, filename)

	var markdownContent strings.Builder

	// Digest Title
	markdownContent.WriteString(fmt.Sprintf("# Weekly Digest - %s\n\n", dateStr))

	if len(digestItems) == 0 {
		markdownContent.WriteString("No articles processed for this digest.\n")
	} else {
		if finalDigest != "" {
			// Use final digest as main content
			markdownContent.WriteString(finalDigest)
			markdownContent.WriteString("\n\n---\n\n")
			markdownContent.WriteString("## Individual Article Summaries\n\n")
			markdownContent.WriteString("*For reference, here are the individual summaries that were used to create the digest above.*\n\n")
		}
		
		// Add individual summaries (either as main content or as appendix)
		for i, item := range digestItems {
			markdownContent.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, item.Title))
			markdownContent.WriteString(item.SummaryText + "\n\n")
			if item.MyTake != "" {
				markdownContent.WriteString(fmt.Sprintf("**My Take:** %s\n\n", item.MyTake))
			}
			markdownContent.WriteString(fmt.Sprintf("[^%d]: %s\n\n", i+1, item.URL))
			markdownContent.WriteString("---\n\n")
		}
	}

	err = os.WriteFile(filePath, []byte(markdownContent.String()), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write digest file %s: %w", filePath, err)
	}

	return filePath, nil
}

// WriteDigestToFile writes the provided content to a file in the specified directory
// This function is used by the template system to save rendered digests
func WriteDigestToFile(content, outputDir, filename string) (string, error) {
	if outputDir == "" {
		outputDir = "digests" // Default output directory
	}

	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	filePath := filepath.Join(outputDir, filename)

	err = os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write digest file %s: %w", filePath, err)
	}

	return filePath, nil
}
