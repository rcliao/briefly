package fetch

import (
	"briefly/internal/core"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadLinksFromFile(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-links.md")

	testContent := `# Test Links

Here are some test links:

- https://example.com/article1
- [Test Article](https://example.com/article2)
- Some text with https://example.com/article3 inline
- Invalid URL: not-a-url
- ftp://example.com/file (should be skipped)
- https://example.com/article1 (duplicate)

## More links
https://example.com/article4
`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	links, err := ReadLinksFromFile(testFile)
	if err != nil {
		t.Fatalf("ReadLinksFromFile failed: %v", err)
	}

	expectedURLs := []string{
		"https://example.com/article1",
		"https://example.com/article2",
		"https://example.com/article3",
		"https://example.com/article4",
	}

	if len(links) != len(expectedURLs) {
		t.Errorf("Expected %d links, got %d", len(expectedURLs), len(links))
	}

	for i, link := range links {
		if i >= len(expectedURLs) {
			break
		}
		if link.URL != expectedURLs[i] {
			t.Errorf("Expected URL %s, got %s", expectedURLs[i], link.URL)
		}
		if link.ID == "" {
			t.Error("Link ID should not be empty")
		}
		if link.Source != "file:"+testFile {
			t.Errorf("Expected source 'file:%s', got '%s'", testFile, link.Source)
		}
		if link.DateAdded.IsZero() {
			t.Error("DateAdded should not be zero")
		}
	}
}

func TestReadLinksFromFile_NonExistentFile(t *testing.T) {
	_, err := ReadLinksFromFile("/nonexistent/file.md")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestReadLinksFromFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.md")

	err := os.WriteFile(testFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	links, err := ReadLinksFromFile(testFile)
	if err != nil {
		t.Fatalf("ReadLinksFromFile failed: %v", err)
	}

	if len(links) != 0 {
		t.Errorf("Expected 0 links from empty file, got %d", len(links))
	}
}

func TestURLRegex(t *testing.T) {
	testCases := []struct {
		input    string
		expected []string
	}{
		{
			input:    "Visit https://example.com for more info",
			expected: []string{"https://example.com"},
		},
		{
			input:    "Check out http://test.org and https://another.site.com/path",
			expected: []string{"http://test.org", "https://another.site.com/path"},
		},
		{
			input:    "No URLs here",
			expected: []string{},
		},
		{
			input:    "ftp://example.com should not match",
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		matches := urlRegex.FindAllString(tc.input, -1)
		if len(matches) != len(tc.expected) {
			t.Errorf("Input: %s\nExpected %d matches, got %d", tc.input, len(tc.expected), len(matches))
			continue
		}
		for i, match := range matches {
			if match != tc.expected[i] {
				t.Errorf("Input: %s\nExpected match %s, got %s", tc.input, tc.expected[i], match)
			}
		}
	}
}

func TestFetchArticle_Success(t *testing.T) {
	testHTML := `<!DOCTYPE html>
<html>
<head>
    <title>Test Article Title</title>
    <meta property="og:title" content="OG Title">
</head>
<body>
    <h1>Main Title</h1>
    <p>This is test content.</p>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(testHTML))
	}))
	defer server.Close()

	link := core.Link{
		ID:  "test-link-id",
		URL: server.URL,
	}

	article, err := FetchArticle(link)
	if err != nil {
		t.Fatalf("FetchArticle failed: %v", err)
	}

	if article.ID == "" {
		t.Error("Article ID should not be empty")
	}
	if article.LinkID != link.ID {
		t.Errorf("Expected LinkID %s, got %s", link.ID, article.LinkID)
	}
	if article.FetchedHTML != testHTML {
		t.Error("FetchedHTML does not match expected content")
	}
	if article.DateFetched.IsZero() {
		t.Error("DateFetched should not be zero")
	}
	if article.Title != "Test Article Title" {
		t.Errorf("Expected title 'Test Article Title', got '%s'", article.Title)
	}
}

func TestFetchArticle_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	link := core.Link{
		ID:  "test-link-id",
		URL: server.URL,
	}

	_, err := FetchArticle(link)
	if err == nil {
		t.Error("Expected error for HTTP 404")
	}
	if !strings.Contains(err.Error(), "status code 404") {
		t.Errorf("Expected error to mention status code 404, got: %v", err)
	}
}

func TestFetchArticle_InvalidURL(t *testing.T) {
	link := core.Link{
		ID:  "test-link-id",
		URL: "invalid-url",
	}

	_, err := FetchArticle(link)
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestExtractTitle(t *testing.T) {
	testCases := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "Title tag",
			html:     `<html><head><title>Test Title</title></head><body></body></html>`,
			expected: "Test Title",
		},
		{
			name:     "OpenGraph title",
			html:     `<html><head><meta property="og:title" content="OG Title"></head><body></body></html>`,
			expected: "OG Title",
		},
		{
			name:     "H1 title",
			html:     `<html><head></head><body><h1>H1 Title</h1></body></html>`,
			expected: "H1 Title",
		},
		{
			name:     "No title",
			html:     `<html><head></head><body><p>No title here</p></body></html>`,
			expected: "",
		},
		{
			name:     "Title with whitespace",
			html:     `<html><head><title>  Spaced Title  </title></head><body></body></html>`,
			expected: "Spaced Title",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractTitle(tc.html, "test-url")
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestParseArticleContent(t *testing.T) {
	testHTML := `<!DOCTYPE html>
<html>
<head>
    <title>Test Article</title>
</head>
<body>
    <nav>Navigation menu</nav>
    <header>Page header</header>
    <script>console.log("script");</script>
    <style>.test { color: red; }</style>
    
    <main>
        <h1>Main Article Title</h1>
        <p>This is the first paragraph of the article.</p>
        <p>This is the second paragraph with more content.</p>
        <h2>Subheading</h2>
        <p>Content under subheading.</p>
        <ul>
            <li>List item 1</li>
            <li>List item 2</li>
        </ul>
    </main>
    
    <aside>Sidebar content</aside>
    <footer>Page footer</footer>
</body>
</html>`

	article := &core.Article{
		ID:          "test-id",
		LinkID:      "test-link-id",
		FetchedHTML: testHTML,
	}

	err := ParseArticleContent(article)
	if err != nil {
		t.Fatalf("ParseArticleContent failed: %v", err)
	}

	if article.CleanedText == "" {
		t.Error("CleanedText should not be empty")
	}

	// Check that main content is extracted and unwanted elements are removed
	if strings.Contains(article.CleanedText, "Navigation menu") {
		t.Error("CleanedText should not contain navigation content")
	}
	if strings.Contains(article.CleanedText, "script") {
		t.Error("CleanedText should not contain script content")
	}
	if strings.Contains(article.CleanedText, "Sidebar content") {
		t.Error("CleanedText should not contain sidebar content")
	}
	if !strings.Contains(article.CleanedText, "Main Article Title") {
		t.Error("CleanedText should contain main article title")
	}
	if !strings.Contains(article.CleanedText, "first paragraph") {
		t.Error("CleanedText should contain article paragraphs")
	}
}

func TestParseArticleContent_NoFetchedHTML(t *testing.T) {
	article := &core.Article{
		ID:     "test-id",
		LinkID: "test-link-id",
	}

	err := ParseArticleContent(article)
	if err == nil {
		t.Error("Expected error when FetchedHTML is empty")
	}
}

func TestParseArticleContent_FallbackTitle(t *testing.T) {
	testHTML := `<html><body><p>This is some content without a proper title tag</p></body></html>`

	article := &core.Article{
		ID:          "test-id",
		LinkID:      "test-link-id",
		FetchedHTML: testHTML,
	}

	err := ParseArticleContent(article)
	if err != nil {
		t.Fatalf("ParseArticleContent failed: %v", err)
	}

	if article.Title == "" {
		t.Error("Title should be generated from content when no title tag exists")
	}
	if !strings.Contains(article.Title, "This is some content") {
		t.Errorf("Expected title to contain content excerpt, got: %s", article.Title)
	}
}

func TestCleanArticleHTML(t *testing.T) {
	testHTML := `<html><body><p>Test content</p></body></html>`

	article := &core.Article{
		ID:          "test-id",
		LinkID:      "test-link-id",
		FetchedHTML: testHTML,
	}

	err := CleanArticleHTML(article)
	if err != nil {
		t.Fatalf("CleanArticleHTML failed: %v", err)
	}

	if article.CleanedText == "" {
		t.Error("CleanedText should not be empty")
	}
}

func TestParseArticleContent_NoMainContent(t *testing.T) {
	// Test fallback behavior when no main content selectors match
	testHTML := `<html><body><div><p>Some general content</p></div></body></html>`

	article := &core.Article{
		ID:          "test-id",
		LinkID:      "test-link-id",
		FetchedHTML: testHTML,
	}

	err := ParseArticleContent(article)
	if err != nil {
		t.Fatalf("ParseArticleContent failed: %v", err)
	}

	if !strings.Contains(article.CleanedText, "Some general content") {
		t.Error("Should fallback to extracting all body text when no main content found")
	}
}

func TestParseArticleContent_InvalidHTML(t *testing.T) {
	article := &core.Article{
		ID:          "test-id",
		LinkID:      "test-link-id",
		FetchedHTML: "invalid html content <",
	}

	// Should not fail even with invalid HTML - goquery is tolerant
	err := ParseArticleContent(article)
	if err != nil {
		t.Fatalf("ParseArticleContent should handle invalid HTML gracefully: %v", err)
	}
}
