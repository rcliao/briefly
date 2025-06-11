package fetch

import (
	"briefly/internal/core"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// YouTubeTranscriptResponse represents the structure of YouTube transcript data
type YouTubeTranscriptResponse struct {
	Events []struct {
		Text  string  `json:"text"`
		Start float64 `json:"start"`
		Dur   float64 `json:"dur"`
	} `json:"events"`
}

// YouTubeVideoInfo represents basic video information
type YouTubeVideoInfo struct {
	Title    string `json:"title"`
	Channel  string `json:"channel"`
	Duration int    `json:"duration"`
}

// ProcessYouTubeContent extracts transcript content from a YouTube video
func ProcessYouTubeContent(link core.Link) (core.Article, error) {
	videoID, err := extractYouTubeVideoID(link.URL)
	if err != nil {
		return core.Article{}, fmt.Errorf("failed to extract video ID from %s: %w", link.URL, err)
	}

	// Get video info
	videoInfo, err := getYouTubeVideoInfo(videoID)
	if err != nil {
		return core.Article{}, fmt.Errorf("failed to get video info for %s: %w", videoID, err)
	}

	// Get transcript
	transcript, err := getYouTubeTranscript(videoID)
	if err != nil {
		return core.Article{}, fmt.Errorf("failed to get transcript for video %s: %w", videoID, err)
	}

	cleanedText := cleanYouTubeTranscript(transcript)

	article := core.Article{
		ID:          uuid.NewString(),
		LinkID:      link.ID,
		Title:       videoInfo.Title,
		ContentType: core.ContentTypeYouTube,
		RawContent:  transcript,
		CleanedText: cleanedText,
		DateFetched: time.Now().UTC(),
		Duration:    videoInfo.Duration,
		Channel:     videoInfo.Channel,
	}

	return article, nil
}

// extractYouTubeVideoID extracts the video ID from various YouTube URL formats
func extractYouTubeVideoID(youtubeURL string) (string, error) {
	// Regular expressions for different YouTube URL formats
	patterns := []string{
		`(?:youtube\.com/watch\?v=|youtu\.be/|youtube\.com/embed/)([a-zA-Z0-9_-]{11})`,
		`youtube\.com/watch\?.*v=([a-zA-Z0-9_-]{11})`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(youtubeURL)
		if len(matches) > 1 {
			return matches[1], nil
		}
	}

	return "", fmt.Errorf("could not extract video ID from URL: %s", youtubeURL)
}

// getYouTubeVideoInfo fetches basic video information
func getYouTubeVideoInfo(videoID string) (*YouTubeVideoInfo, error) {
	// Use YouTube's oEmbed API for basic info (no API key required)
	oembedURL := fmt.Sprintf("https://www.youtube.com/oembed?url=https://www.youtube.com/watch?v=%s&format=json", videoID)

	resp, err := http.Get(oembedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch video info: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Warning: failed to close HTTP response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("YouTube oEmbed API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var oembed struct {
		Title      string `json:"title"`
		AuthorName string `json:"author_name"`
		Height     int    `json:"height"`
		Width      int    `json:"width"`
	}

	if err := json.Unmarshal(body, &oembed); err != nil {
		return nil, fmt.Errorf("failed to parse oEmbed response: %w", err)
	}

	return &YouTubeVideoInfo{
		Title:    oembed.Title,
		Channel:  oembed.AuthorName,
		Duration: 0, // Duration not available from oEmbed, would need YouTube Data API
	}, nil
}

// getYouTubeTranscript attempts to fetch transcript using various methods
func getYouTubeTranscript(videoID string) (string, error) {
	// Method 1: Try to get auto-generated captions from YouTube's internal API
	transcript, err := getTranscriptFromYouTubeAPI(videoID)
	if err == nil && transcript != "" {
		return transcript, nil
	}

	// Method 2: Try alternative approach (this is a simplified version)
	// In a production environment, you might want to use a more robust solution
	// or integrate with the YouTube Data API v3

	return "", fmt.Errorf("no transcript available for video %s", videoID)
}

// getTranscriptFromYouTubeAPI attempts to get transcript from YouTube's internal API
func getTranscriptFromYouTubeAPI(videoID string) (string, error) {
	// This is a simplified approach. In practice, you would need to:
	// 1. First get the video page to extract transcript track URLs
	// 2. Parse the available transcript tracks
	// 3. Fetch the transcript data

	// For now, return an error to indicate transcripts are not available
	// This can be enhanced with proper YouTube Data API integration
	return "", fmt.Errorf("transcript fetching not implemented - requires YouTube Data API key")
}

// cleanYouTubeTranscript cleans and formats the transcript text
func cleanYouTubeTranscript(transcript string) string {
	if transcript == "" {
		return ""
	}

	// Remove timestamps and format transcript for better readability
	lines := strings.Split(transcript, "\n")
	var cleanLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			// Remove timestamp patterns like [00:00:00]
			timestampRegex := regexp.MustCompile(`\[\d{2}:\d{2}:\d{2}\]`)
			cleaned := timestampRegex.ReplaceAllString(trimmed, "")
			cleaned = strings.TrimSpace(cleaned)

			if cleaned != "" {
				cleanLines = append(cleanLines, cleaned)
			}
		}
	}

	// Join lines and clean up spacing
	result := strings.Join(cleanLines, " ")

	// Replace multiple spaces with single spaces
	spaceRegex := regexp.MustCompile(`\s+`)
	result = spaceRegex.ReplaceAllString(result, " ")

	return strings.TrimSpace(result)
}

// DetectYouTubeURL checks if a URL is a YouTube video URL
func DetectYouTubeURL(urlStr string) bool {
	patterns := []string{
		`youtube\.com/watch\?.*v=`,
		`youtu\.be/`,
		`youtube\.com/embed/`,
		`m\.youtube\.com/watch\?.*v=`,
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, urlStr)
		if matched {
			return true
		}
	}

	return false
}

// CreateMockTranscript creates a mock transcript for testing (when API is not available)
func CreateMockTranscript(videoInfo *YouTubeVideoInfo) string {
	return fmt.Sprintf("This is a video titled '%s' by %s. Transcript is not available without YouTube Data API access. The video content cannot be processed for summarization.",
		videoInfo.Title, videoInfo.Channel)
}
