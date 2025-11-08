package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMarkdownContent(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name: "markdown links",
			content: `# Weekly Links
- [Article 1](https://example.com/article1)
- [Article 2](https://example.com/article2)`,
			expected: []string{
				"https://example.com/article1",
				"https://example.com/article2",
			},
		},
		{
			name: "raw URLs in bullet list",
			content: `# Weekly Links
- https://example.com/article1
- https://example.com/article2`,
			expected: []string{
				"https://example.com/article1",
				"https://example.com/article2",
			},
		},
		{
			name: "mixed markdown and raw URLs",
			content: `# Weekly Links
- [Article 1](https://example.com/article1)
- https://example.com/article2
Check this out: https://example.com/article3`,
			expected: []string{
				"https://example.com/article1",
				"https://example.com/article2",
				"https://example.com/article3",
			},
		},
		{
			name: "duplicate URLs",
			content: `# Weekly Links
- [Article 1](https://example.com/article1)
- [Same Article](https://example.com/article1)
- https://example.com/article1`,
			expected: []string{
				"https://example.com/article1",
			},
		},
		{
			name: "URLs with tracking parameters (should be normalized)",
			content: `# Weekly Links
- https://example.com/article?utm_source=twitter&utm_campaign=promo
- https://example.com/article?fbclid=123456`,
			expected: []string{
				"https://example.com/article",
			},
		},
		{
			name:     "empty content",
			content:  "",
			expected: []string{},
		},
		{
			name: "no URLs",
			content: `# Weekly Links
Just some text without any links.`,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls := parser.ParseMarkdownContent(tt.content)

			if len(urls) != len(tt.expected) {
				t.Errorf("Expected %d URLs, got %d: %v", len(tt.expected), len(urls), urls)
				return
			}

			for i, expected := range tt.expected {
				if urls[i] != expected {
					t.Errorf("Expected URL[%d] = %s, got %s", i, expected, urls[i])
				}
			}
		})
	}
}

func TestNormalizeURL(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "remove utm parameters",
			input:    "https://example.com/article?utm_source=twitter&utm_campaign=promo",
			expected: "https://example.com/article",
		},
		{
			name:     "remove fbclid",
			input:    "https://example.com/article?fbclid=123456",
			expected: "https://example.com/article",
		},
		{
			name:     "keep query parameters that aren't tracking",
			input:    "https://example.com/search?q=golang&page=2",
			expected: "https://example.com/search?page=2&q=golang", // Note: query params may be reordered
		},
		{
			name:     "remove fragment",
			input:    "https://example.com/article#section-1",
			expected: "https://example.com/article",
		},
		{
			name:     "remove trailing slash",
			input:    "https://example.com/article/",
			expected: "https://example.com/article",
		},
		{
			name:     "keep root trailing slash",
			input:    "https://example.com/",
			expected: "https://example.com/",
		},
		{
			name:     "already normalized",
			input:    "https://example.com/article",
			expected: "https://example.com/article",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.NormalizeURL(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name      string
		url       string
		wantError bool
	}{
		{
			name:      "valid http URL",
			url:       "http://example.com/article",
			wantError: false,
		},
		{
			name:      "valid https URL",
			url:       "https://example.com/article",
			wantError: false,
		},
		{
			name:      "empty URL",
			url:       "",
			wantError: true,
		},
		{
			name:      "invalid scheme",
			url:       "ftp://example.com/file",
			wantError: true,
		},
		{
			name:      "missing host",
			url:       "https://",
			wantError: true,
		},
		{
			name:      "malformed URL",
			url:       "not a url",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.ValidateURL(tt.url)
			hasError := err != nil
			if hasError != tt.wantError {
				t.Errorf("Expected error=%v, got error=%v (%v)", tt.wantError, hasError, err)
			}
		})
	}
}

func TestDeduplicateURLs(t *testing.T) {
	parser := NewParser()

	input := []string{
		"https://example.com/article1",
		"https://example.com/article2",
		"https://example.com/article1",                    // Duplicate
		"https://example.com/article1?utm_source=twitter", // Duplicate after normalization
		"https://example.com/article3",
	}

	result := parser.DeduplicateURLs(input)

	expected := []string{
		"https://example.com/article1",
		"https://example.com/article2",
		"https://example.com/article3",
	}

	if len(result) != len(expected) {
		t.Errorf("Expected %d URLs, got %d", len(expected), len(result))
		return
	}

	for i, expectedURL := range expected {
		if result[i] != expectedURL {
			t.Errorf("Expected URL[%d] = %s, got %s", i, expectedURL, result[i])
		}
	}
}

func TestParseMarkdownFile(t *testing.T) {
	parser := NewParser()

	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test-links.md")

	content := `# Test Links
- [Article 1](https://example.com/article1)
- https://example.com/article2
- [Article 3](https://example.com/article3)
`

	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	links, err := parser.ParseMarkdownFile(testFile)
	if err != nil {
		t.Fatalf("ParseMarkdownFile failed: %v", err)
	}

	expectedURLs := []string{
		"https://example.com/article1",
		"https://example.com/article2",
		"https://example.com/article3",
	}

	if len(links) != len(expectedURLs) {
		t.Errorf("Expected %d links, got %d", len(expectedURLs), len(links))
		return
	}

	for i, expectedURL := range expectedURLs {
		if links[i].URL != expectedURL {
			t.Errorf("Expected link[%d].URL = %s, got %s", i, expectedURL, links[i].URL)
		}

		// Verify link metadata is populated
		if links[i].ID == "" {
			t.Errorf("Link[%d] missing ID", i)
		}

		if links[i].Source != "file:"+testFile {
			t.Errorf("Expected source 'file:%s', got '%s'", testFile, links[i].Source)
		}
	}
}

func TestParseFile(t *testing.T) {
	// Test the convenience function
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test-links.md")

	content := `# Test Links
- https://example.com/article1
`

	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	links, err := ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(links) != 1 {
		t.Errorf("Expected 1 link, got %d", len(links))
	}

	if links[0].URL != "https://example.com/article1" {
		t.Errorf("Expected URL 'https://example.com/article1', got '%s'", links[0].URL)
	}
}
