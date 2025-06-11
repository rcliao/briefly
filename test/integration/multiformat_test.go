package integration

import (
	"briefly/internal/core"
	"briefly/internal/fetch"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMultiFormatContentProcessing(t *testing.T) {
	t.Run("ContentTypeDetection", func(t *testing.T) {
		tests := []struct {
			url      string
			expected core.ContentType
		}{
			{"https://example.com/article.html", core.ContentTypeHTML},
			{"https://example.com/document.pdf", core.ContentTypePDF},
			{"https://youtube.com/watch?v=abc123", core.ContentTypeYouTube},
			{"https://youtu.be/abc123", core.ContentTypeYouTube},
			{"file://./test.pdf", core.ContentTypePDF},
			{"./local-doc.pdf", core.ContentTypePDF},
		}

		for _, test := range tests {
			// We can't actually process these URLs without real content,
			// but we can test URL detection patterns
			if strings.Contains(test.url, "youtube.com") || strings.Contains(test.url, "youtu.be") {
				if !fetch.DetectYouTubeURL(test.url) {
					t.Errorf("Expected %s to be detected as YouTube URL", test.url)
				}
			}
			if strings.HasSuffix(test.url, ".pdf") {
				if !fetch.DetectPDFURL(test.url) {
					t.Errorf("Expected %s to be detected as PDF URL", test.url)
				}
			}
		}
	})

	t.Run("ContentTypeLabels", func(t *testing.T) {
		tests := []struct {
			contentType core.ContentType
			label       string
			icon        string
		}{
			{core.ContentTypeHTML, "Web Article", "🌐"},
			{core.ContentTypePDF, "PDF Document", "📄"},
			{core.ContentTypeYouTube, "YouTube Video", "🎥"},
		}

		for _, test := range tests {
			label := fetch.GetContentTypeLabel(test.contentType)
			icon := fetch.GetContentTypeIcon(test.contentType)

			if label != test.label {
				t.Errorf("Expected label %s, got %s", test.label, label)
			}
			if icon != test.icon {
				t.Errorf("Expected icon %s, got %s", test.icon, icon)
			}
		}
	})

	t.Run("ProcessMixedContentInput", func(t *testing.T) {
		// Create a temporary input file with mixed content types
		tempDir := t.TempDir()
		inputFile := filepath.Join(tempDir, "mixed-input.md")

		content := `# Mixed Content Test

## Web Articles
- https://example.com/article1.html
- https://example.com/article2

## Research Papers  
- ./test-papers/research.pdf
- file://./documents/whitepaper.pdf

## Videos
- https://youtube.com/watch?v=test123
- https://youtu.be/demo456
`

		err := os.WriteFile(inputFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write test input file: %v", err)
		}

		// Test link extraction with mixed content
		links, err := fetch.ReadLinksFromFile(inputFile)
		if err != nil {
			t.Fatalf("Failed to read links from file: %v", err)
		}

		// Should find all different content types
		expectedCount := 6
		if len(links) != expectedCount {
			t.Errorf("Expected %d links, got %d", expectedCount, len(links))
		}

		// Verify different URL types are detected
		foundTypes := make(map[string]bool)
		for _, link := range links {
			if strings.Contains(link.URL, "youtube") || strings.Contains(link.URL, "youtu.be") {
				foundTypes["youtube"] = true
			} else if strings.HasSuffix(link.URL, ".pdf") || strings.Contains(link.URL, ".pdf") {
				foundTypes["pdf"] = true
			} else {
				foundTypes["html"] = true
			}
		}

		expectedTypes := []string{"youtube", "pdf", "html"}
		for _, expectedType := range expectedTypes {
			if !foundTypes[expectedType] {
				t.Errorf("Expected to find %s content type", expectedType)
			}
		}
	})
}

func TestPDFContentExtraction(t *testing.T) {
	// Test basic PDF processing functionality
	t.Run("PDFTextCleaning", func(t *testing.T) {
		// Test that we have the basic building blocks
		if fetch.GetContentTypeLabel(core.ContentTypePDF) != "PDF Document" {
			t.Error("PDF content type not properly configured")
		}

		if fetch.GetContentTypeIcon(core.ContentTypePDF) != "📄" {
			t.Error("PDF icon not properly configured")
		}
	})
}

func TestYouTubeContentProcessing(t *testing.T) {
	t.Run("YouTubeURLExtraction", func(t *testing.T) {
		tests := []struct {
			url         string
			shouldMatch bool
		}{
			{"https://youtube.com/watch?v=abc123def", true},
			{"https://www.youtube.com/watch?v=abc123def", true},
			{"https://youtu.be/abc123def", true},
			{"https://youtube.com/watch?v=abc123def&t=123s", true},
			{"https://example.com/not-youtube", false},
			{"https://vimeo.com/123456", false},
		}

		for _, test := range tests {
			result := fetch.DetectYouTubeURL(test.url)
			if result != test.shouldMatch {
				t.Errorf("URL %s: expected %v, got %v", test.url, test.shouldMatch, result)
			}
		}
	})

	t.Run("YouTubeMetadata", func(t *testing.T) {
		// Test that YouTube content type is properly configured
		if fetch.GetContentTypeLabel(core.ContentTypeYouTube) != "YouTube Video" {
			t.Error("YouTube content type not properly configured")
		}

		if fetch.GetContentTypeIcon(core.ContentTypeYouTube) != "🎥" {
			t.Error("YouTube icon not properly configured")
		}
	})
}
