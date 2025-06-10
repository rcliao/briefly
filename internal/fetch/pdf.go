package fetch

import (
	"briefly/internal/core"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ledongthuc/pdf"
)

// ProcessPDFContent extracts text content from a PDF file
func ProcessPDFContent(link core.Link) (core.Article, error) {
	var reader io.ReaderAt
	var size int64
	var err error

	// Handle both local files and remote URLs
	if strings.HasPrefix(link.URL, "file://") || !strings.HasPrefix(link.URL, "http") {
		// Local file
		filePath := strings.TrimPrefix(link.URL, "file://")
		if !strings.HasPrefix(link.URL, "file://") {
			filePath = link.URL // Direct file path
		}
		
		file, err := os.Open(filePath)
		if err != nil {
			return core.Article{}, fmt.Errorf("failed to open PDF file %s: %w", filePath, err)
		}
		defer func() {
			if err := file.Close(); err != nil {
				fmt.Printf("Warning: failed to close PDF file: %v\n", err)
			}
		}()

		stat, err := file.Stat()
		if err != nil {
			return core.Article{}, fmt.Errorf("failed to stat PDF file %s: %w", filePath, err)
		}
		
		reader = file
		size = stat.Size()
	} else {
		// Remote URL
		resp, err := http.Get(link.URL)
		if err != nil {
			return core.Article{}, fmt.Errorf("failed to fetch PDF from URL %s: %w", link.URL, err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("Warning: failed to close HTTP response body: %v\n", err)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			return core.Article{}, fmt.Errorf("failed to fetch PDF from URL %s: status code %d", link.URL, resp.StatusCode)
		}

		// Check content type
		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/pdf") {
			return core.Article{}, fmt.Errorf("URL %s does not return a PDF (Content-Type: %s)", link.URL, contentType)
		}

		// Read the entire response into memory
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return core.Article{}, fmt.Errorf("failed to read PDF data from %s: %w", link.URL, err)
		}

		reader = strings.NewReader(string(data))
		size = int64(len(data))
	}

	// Parse PDF
	pdfReader, err := pdf.NewReader(reader, size)
	if err != nil {
		return core.Article{}, fmt.Errorf("failed to create PDF reader for %s: %w", link.URL, err)
	}

	// Extract text from all pages
	var textBuilder strings.Builder
	pageCount := pdfReader.NumPage()
	
	for i := 1; i <= pageCount; i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}

		// Extract text from the page
		pageText, err := page.GetPlainText(nil)
		if err != nil {
			// Log warning but continue with other pages
			fmt.Printf("Warning: failed to extract text from page %d of %s: %v\n", i, link.URL, err)
			continue
		}

		textBuilder.WriteString(pageText)
		textBuilder.WriteString("\n\n") // Add page break
	}

	rawText := textBuilder.String()
	cleanedText := cleanPDFText(rawText)

	// Generate title from first few lines if available
	title := extractPDFTitle(cleanedText, link.URL)

	article := core.Article{
		ID:          uuid.NewString(),
		LinkID:      link.ID,
		Title:       title,
		ContentType: core.ContentTypePDF,
		RawContent:  rawText,
		CleanedText: cleanedText,
		DateFetched: time.Now().UTC(),
		FileSize:    size,
		PageCount:   pageCount,
	}

	return article, nil
}

// cleanPDFText performs basic cleaning of extracted PDF text
func cleanPDFText(rawText string) string {
	// Remove excessive whitespace and line breaks
	lines := strings.Split(rawText, "\n")
	var cleanLines []string
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && len(trimmed) > 2 { // Skip very short lines that are likely noise
			cleanLines = append(cleanLines, trimmed)
		}
	}
	
	cleanText := strings.Join(cleanLines, "\n")
	
	// Replace multiple consecutive newlines with double newlines
	cleanText = strings.ReplaceAll(cleanText, "\n\n\n", "\n\n")
	
	return strings.TrimSpace(cleanText)
}

// extractPDFTitle attempts to extract a title from PDF content
func extractPDFTitle(content string, sourceURL string) string {
	if content == "" {
		return fmt.Sprintf("PDF Document (%s)", sourceURL)
	}
	
	lines := strings.Split(content, "\n")
	
	// Look for the first substantial line as title
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 10 && len(trimmed) < 200 {
			// Check if it looks like a title (not a URL, not all caps unless short)
			if !strings.Contains(trimmed, "http") && 
			   (len(trimmed) < 50 || !isAllUpperCase(trimmed)) {
				return trimmed
			}
		}
	}
	
	// Fallback: use first few words
	words := strings.Fields(content)
	if len(words) > 3 {
		return strings.Join(words[:3], " ") + "..."
	}
	
	return fmt.Sprintf("PDF Document (%s)", sourceURL)
}

// isAllUpperCase checks if a string is all uppercase
func isAllUpperCase(s string) bool {
	return strings.ToUpper(s) == s && strings.ToLower(s) != s
}

// DetectPDFURL checks if a URL points to a PDF
func DetectPDFURL(url string) bool {
	// Check file extension
	if strings.HasSuffix(strings.ToLower(url), ".pdf") {
		return true
	}
	
	// Check for local file paths
	if strings.HasPrefix(url, "file://") && strings.HasSuffix(strings.ToLower(url), ".pdf") {
		return true
	}
	
	// For HTTP URLs without .pdf extension, we'll need to check Content-Type header
	// This will be handled in the HTTP request phase
	
	return false
}