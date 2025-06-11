package fetch

import (
	"briefly/internal/core"
	"context"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

// ContentProcessor implements the ArticleProcessor interface with multi-format support
type ContentProcessor struct{}

// NewContentProcessor creates a new ContentProcessor
func NewContentProcessor() *ContentProcessor {
	return &ContentProcessor{}
}

// ProcessArticle processes a single article from a URL, detecting content type automatically
func (cp *ContentProcessor) ProcessArticle(ctx context.Context, urlStr string) (*core.Article, error) {
	// Create a basic link structure
	link := core.Link{
		URL: urlStr,
	}

	// Detect content type
	contentType, err := cp.detectContentType(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to detect content type for %s: %w", urlStr, err)
	}

	// Process based on content type
	var article core.Article
	switch contentType {
	case core.ContentTypePDF:
		article, err = ProcessPDFContent(link)
	case core.ContentTypeYouTube:
		article, err = ProcessYouTubeContent(link)
	case core.ContentTypeHTML:
		fallthrough
	default:
		article, err = FetchArticle(link)
		if err == nil {
			article.ContentType = core.ContentTypeHTML
			err = ParseArticleContent(&article)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to process %s content from %s: %w", contentType, urlStr, err)
	}

	return &article, nil
}

// ProcessArticles processes multiple articles concurrently
func (cp *ContentProcessor) ProcessArticles(ctx context.Context, urls []string) ([]core.Article, error) {
	articles := make([]core.Article, 0, len(urls))

	for _, urlStr := range urls {
		select {
		case <-ctx.Done():
			return articles, ctx.Err()
		default:
		}

		article, err := cp.ProcessArticle(ctx, urlStr)
		if err != nil {
			// Log error but continue with other articles
			fmt.Printf("Warning: failed to process %s: %v\n", urlStr, err)
			continue
		}

		articles = append(articles, *article)
	}

	return articles, nil
}

// CleanAndExtractContent performs additional cleaning on an already fetched article
func (cp *ContentProcessor) CleanAndExtractContent(ctx context.Context, article *core.Article) error {
	switch article.ContentType {
	case core.ContentTypePDF:
		// PDF content is already cleaned during extraction
		return nil
	case core.ContentTypeYouTube:
		// YouTube transcript is already cleaned during extraction
		return nil
	case core.ContentTypeHTML:
		return ParseArticleContent(article)
	default:
		return fmt.Errorf("unknown content type: %s", article.ContentType)
	}
}

// detectContentType determines the content type of a URL
func (cp *ContentProcessor) detectContentType(urlStr string) (core.ContentType, error) {
	// Check for YouTube URLs first
	if DetectYouTubeURL(urlStr) {
		return core.ContentTypeYouTube, nil
	}

	// Check for PDF URLs by extension
	if DetectPDFURL(urlStr) {
		return core.ContentTypePDF, nil
	}

	// For local files, check file extension
	if strings.HasPrefix(urlStr, "file://") || !strings.Contains(urlStr, "://") {
		ext := strings.ToLower(filepath.Ext(urlStr))
		switch ext {
		case ".pdf":
			return core.ContentTypePDF, nil
		case ".html", ".htm":
			return core.ContentTypeHTML, nil
		}
	}

	// For HTTP URLs, check Content-Type header
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return core.ContentTypeHTML, nil // Default to HTML
	}

	if parsedURL.Scheme == "http" || parsedURL.Scheme == "https" {
		contentType, err := cp.getContentTypeFromHTTP(urlStr)
		if err != nil {
			// If we can't determine from HTTP, default to HTML
			return core.ContentTypeHTML, nil
		}
		return contentType, nil
	}

	// Default to HTML for unknown cases
	return core.ContentTypeHTML, nil
}

// getContentTypeFromHTTP makes a HEAD request to determine content type
func (cp *ContentProcessor) getContentTypeFromHTTP(urlStr string) (core.ContentType, error) {
	resp, err := http.Head(urlStr)
	if err != nil {
		return core.ContentTypeHTML, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Warning: failed to close HTTP response body: %v\n", err)
		}
	}()

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		return core.ContentTypeHTML, nil
	}

	// Parse the media type
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return core.ContentTypeHTML, nil
	}

	switch mediaType {
	case "application/pdf":
		return core.ContentTypePDF, nil
	case "text/html":
		return core.ContentTypeHTML, nil
	default:
		// Default to HTML for unknown types
		return core.ContentTypeHTML, nil
	}
}

// ProcessLinksFromFile reads links from a file and processes them with content type detection
func ProcessLinksFromFile(filePath string) ([]core.Article, error) {
	links, err := ReadLinksFromFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read links from file: %w", err)
	}

	processor := NewContentProcessor()
	var articles []core.Article

	for _, link := range links {
		article, err := processor.ProcessArticle(context.Background(), link.URL)
		if err != nil {
			fmt.Printf("Warning: failed to process %s: %v\n", link.URL, err)
			continue
		}

		// Update LinkID to match the original link
		article.LinkID = link.ID
		articles = append(articles, *article)
	}

	return articles, nil
}

// GetContentTypeLabel returns a human-readable label for content type
func GetContentTypeLabel(contentType core.ContentType) string {
	switch contentType {
	case core.ContentTypePDF:
		return "PDF Document"
	case core.ContentTypeYouTube:
		return "YouTube Video"
	case core.ContentTypeHTML:
		return "Web Article"
	default:
		return "Unknown Content"
	}
}

// GetContentTypeIcon returns an icon/emoji for content type
func GetContentTypeIcon(contentType core.ContentType) string {
	switch contentType {
	case core.ContentTypePDF:
		return "📄"
	case core.ContentTypeYouTube:
		return "🎥"
	case core.ContentTypeHTML:
		return "🌐"
	default:
		return "📝"
	}
}
