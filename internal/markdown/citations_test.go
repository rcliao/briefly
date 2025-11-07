package markdown

import (
	"briefly/internal/core"
	"testing"
	"time"
)

func TestExtractCitations(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     int // number of citations expected
	}{
		{
			name:     "double bracket format",
			markdown: "Recent research [[1]](https://example.com/article1) shows improvements.",
			want:     1,
		},
		{
			name:     "single bracket format",
			markdown: "According to [2](https://example.com/article2), this is true.",
			want:     1,
		},
		{
			name:     "multiple citations",
			markdown: "Evidence from [[1]](url1) and [[2]](url2) suggests [[3]](url3) is correct.",
			want:     3,
		},
		{
			name:     "mixed formats",
			markdown: "Study [[1]](url1) and [2](url2) agree.",
			want:     2,
		},
		{
			name:     "no citations",
			markdown: "This text has no citations at all.",
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			citations := ExtractCitations(tt.markdown)
			if len(citations) != tt.want {
				t.Errorf("ExtractCitations() = %d citations, want %d", len(citations), tt.want)
			}
		})
	}
}

func TestExtractCitationsDetails(t *testing.T) {
	markdown := "Recent AI developments [[1]](https://openai.com/gpt5) and [[2]](https://anthropic.com/claude) show progress."
	citations := ExtractCitations(markdown)

	if len(citations) != 2 {
		t.Fatalf("Expected 2 citations, got %d", len(citations))
	}

	// Check first citation
	if citations[0].Number != 1 {
		t.Errorf("Citation 1 number = %d, want 1", citations[0].Number)
	}
	if citations[0].URL != "https://openai.com/gpt5" {
		t.Errorf("Citation 1 URL = %s, want https://openai.com/gpt5", citations[0].URL)
	}

	// Check second citation
	if citations[1].Number != 2 {
		t.Errorf("Citation 2 number = %d, want 2", citations[1].Number)
	}
	if citations[1].URL != "https://anthropic.com/claude" {
		t.Errorf("Citation 2 URL = %s, want https://anthropic.com/claude", citations[1].URL)
	}
}

func TestInjectCitationURLs(t *testing.T) {
	articles := []core.Article{
		{ID: "1", URL: "https://example.com/article1", Title: "Article 1"},
		{ID: "2", URL: "https://example.com/article2", Title: "Article 2"},
	}

	markdown := "According to [[1]] and [[2]], this is true."
	result := InjectCitationURLs(markdown, articles)

	expected := "According to [[1]](https://example.com/article1) and [[2]](https://example.com/article2), this is true."
	if result != expected {
		t.Errorf("InjectCitationURLs() = %s, want %s", result, expected)
	}
}

func TestBuildCitationRecords(t *testing.T) {
	articles := []core.Article{
		{
			ID:            "article-1",
			URL:           "https://example.com/1",
			Title:         "Test Article 1",
			Publisher:     "example.com",
			DatePublished: time.Now(),
		},
		{
			ID:        "article-2",
			URL:       "https://example.com/2",
			Title:     "Test Article 2",
			Publisher: "example.com",
		},
	}

	articleMap := make(map[string]*core.Article)
	for i := range articles {
		articleMap[articles[i].URL] = &articles[i]
	}

	citationRefs := []CitationReference{
		{Number: 1, URL: "https://example.com/1", Context: "Some context"},
		{Number: 2, URL: "https://example.com/2", Context: "More context"},
	}

	digestID := "digest-123"
	records := BuildCitationRecords(digestID, citationRefs, articleMap)

	if len(records) != 2 {
		t.Fatalf("Expected 2 citation records, got %d", len(records))
	}

	// Check first record
	if records[0].ArticleID != "article-1" {
		t.Errorf("Citation 1 ArticleID = %s, want article-1", records[0].ArticleID)
	}
	if *records[0].DigestID != digestID {
		t.Errorf("Citation 1 DigestID = %s, want %s", *records[0].DigestID, digestID)
	}
	if *records[0].CitationNumber != 1 {
		t.Errorf("Citation 1 Number = %d, want 1", *records[0].CitationNumber)
	}
	if records[0].Context != "Some context" {
		t.Errorf("Citation 1 Context = %s, want 'Some context'", records[0].Context)
	}
}

func TestValidateCitations(t *testing.T) {
	articles := []core.Article{
		{URL: "https://example.com/article1"},
		{URL: "https://example.com/article2"},
	}

	tests := []struct {
		name     string
		markdown string
		wantWarn int // number of warnings expected
	}{
		{
			name:     "all citations valid",
			markdown: "Text with [[1]](https://example.com/article1) and [[2]](https://example.com/article2)",
			wantWarn: 0,
		},
		{
			name:     "one invalid citation",
			markdown: "Text with [[1]](https://example.com/article1) and [[2]](https://unknown.com)",
			wantWarn: 1,
		},
		{
			name:     "all invalid citations",
			markdown: "Text with [[1]](https://unknown1.com) and [[2]](https://unknown2.com)",
			wantWarn: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := ValidateCitations(tt.markdown, articles)
			if len(warnings) != tt.wantWarn {
				t.Errorf("ValidateCitations() = %d warnings, want %d", len(warnings), tt.wantWarn)
			}
		})
	}
}

func TestCountCitations(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     int
	}{
		{"no citations", "Plain text", 0},
		{"one citation", "Text [[1]](url)", 1},
		{"multiple citations", "Text [[1]](url1) and [[2]](url2) and [[3]](url3)", 3},
		{"mixed formats", "Text [[1]](url1) and [2](url2)", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := CountCitations(tt.markdown)
			if count != tt.want {
				t.Errorf("CountCitations() = %d, want %d", count, tt.want)
			}
		})
	}
}

func TestParseCitationNumbers(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []int
	}{
		{
			name: "single bracket",
			text: "Evidence from [1] and [2]",
			want: []int{1, 2},
		},
		{
			name: "double bracket",
			text: "Evidence from [[1]] and [[2]]",
			want: []int{1, 2},
		},
		{
			name: "mixed",
			text: "Evidence from [[1]], [2], and [[3]]",
			want: []int{1, 2, 3},
		},
		{
			name: "duplicates removed",
			text: "Evidence from [[1]], [[1]], and [[2]]",
			want: []int{1, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			numbers := ParseCitationNumbers(tt.text)
			if len(numbers) != len(tt.want) {
				t.Errorf("ParseCitationNumbers() = %v, want %v", numbers, tt.want)
				return
			}
			for i, num := range numbers {
				if num != tt.want[i] {
					t.Errorf("ParseCitationNumbers()[%d] = %d, want %d", i, num, tt.want[i])
				}
			}
		})
	}
}

func TestExtractContext(t *testing.T) {
	text := "This is a longer piece of text with multiple sentences. Recent AI developments [[1]](url) show significant progress. Additional research continues to validate these findings."
	citation := "[[1]](url)"

	context := extractContext(text, citation)

	// Context should include surrounding text
	if context == "" {
		t.Error("extractContext() returned empty string")
	}

	// Should contain the citation reference area
	if len(context) < 50 {
		t.Errorf("extractContext() = %d chars, expected longer context", len(context))
	}

	t.Logf("Context: %s", context)
}

func TestFormatCitationNumber(t *testing.T) {
	tests := []struct {
		num  int
		want string
	}{
		{1, "[1]"},
		{10, "[10]"},
		{99, "[99]"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			result := FormatCitationNumber(tt.num)
			if result != tt.want {
				t.Errorf("FormatCitationNumber(%d) = %s, want %s", tt.num, result, tt.want)
			}
		})
	}
}
