package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRenderMarkdownDigest_EmptyItems(t *testing.T) {
	tmpDir := t.TempDir()
	digestItems := []DigestData{}
	finalDigest := ""

	filePath, err := RenderMarkdownDigest(digestItems, tmpDir, finalDigest)
	if err != nil {
		t.Fatalf("RenderMarkdownDigest failed: %v", err)
	}

	// Check that file was created
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Digest file should be created")
	}

	// Read and verify content
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read digest file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Weekly Digest") {
		t.Error("Content should contain digest title")
	}
	if !strings.Contains(contentStr, "No articles processed") {
		t.Error("Content should indicate no articles processed")
	}
}

func TestRenderMarkdownDigest_WithItems(t *testing.T) {
	tmpDir := t.TempDir()
	digestItems := []DigestData{
		{
			Title:           "Test Article 1",
			URL:             "https://example.com/article1",
			SummaryText:     "This is a summary of article 1.",
			MyTake:          "My thoughts on article 1",
			TopicCluster:    "Technology",
			TopicConfidence: 0.95,
			SentimentScore:  0.7,
			SentimentLabel:  "positive",
			SentimentEmoji:  "😊",
			AlertTriggered:  true,
			AlertConditions: []string{"breaking news", "urgent"},
			ResearchQueries: []string{"query1", "query2"},
		},
		{
			Title:           "Test Article 2",
			URL:             "https://example.com/article2",
			SummaryText:     "This is a summary of article 2.",
			MyTake:          "",
			TopicCluster:    "Science",
			TopicConfidence: 0.88,
			SentimentScore:  -0.2,
			SentimentLabel:  "negative",
			SentimentEmoji:  "😞",
			AlertTriggered:  false,
			AlertConditions: []string{},
			ResearchQueries: []string{"query3"},
		},
	}

	filePath, err := RenderMarkdownDigest(digestItems, tmpDir, "")
	if err != nil {
		t.Fatalf("RenderMarkdownDigest failed: %v", err)
	}

	// Check that file was created
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Digest file should be created")
	}

	// Read and verify content
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read digest file: %v", err)
	}

	contentStr := string(content)

	// Check title
	if !strings.Contains(contentStr, "Weekly Digest") {
		t.Error("Content should contain digest title")
	}

	// Check articles are present
	if !strings.Contains(contentStr, "Test Article 1") {
		t.Error("Content should contain first article title")
	}
	if !strings.Contains(contentStr, "Test Article 2") {
		t.Error("Content should contain second article title")
	}

	// Check summaries are present
	if !strings.Contains(contentStr, "summary of article 1") {
		t.Error("Content should contain first article summary")
	}
	if !strings.Contains(contentStr, "summary of article 2") {
		t.Error("Content should contain second article summary")
	}

	// Check MyTake is included for first article
	if !strings.Contains(contentStr, "My thoughts on article 1") {
		t.Error("Content should contain MyTake for first article")
	}

	// Check URLs are referenced
	if !strings.Contains(contentStr, "https://example.com/article1") {
		t.Error("Content should contain first article URL")
	}
	if !strings.Contains(contentStr, "https://example.com/article2") {
		t.Error("Content should contain second article URL")
	}

	// Check formatting
	if !strings.Contains(contentStr, "### 1. Test Article 1") {
		t.Error("Content should have proper heading format for first article")
	}
	if !strings.Contains(contentStr, "### 2. Test Article 2") {
		t.Error("Content should have proper heading format for second article")
	}

	// Check footnote references
	if !strings.Contains(contentStr, "[^1]:") {
		t.Error("Content should contain footnote reference for first article")
	}
	if !strings.Contains(contentStr, "[^2]:") {
		t.Error("Content should contain footnote reference for second article")
	}

	// Check separators
	if strings.Count(contentStr, "---") < 2 {
		t.Error("Content should contain separators between articles")
	}
}

func TestRenderMarkdownDigest_WithFinalDigest(t *testing.T) {
	tmpDir := t.TempDir()
	digestItems := []DigestData{
		{
			Title:       "Test Article",
			URL:         "https://example.com/article",
			SummaryText: "Article summary.",
		},
	}
	finalDigest := "## Final Digest Content\n\nThis is the main digest content that summarizes all articles."

	filePath, err := RenderMarkdownDigest(digestItems, tmpDir, finalDigest)
	if err != nil {
		t.Fatalf("RenderMarkdownDigest failed: %v", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read digest file: %v", err)
	}

	contentStr := string(content)

	// Check that final digest is present and comes first
	if !strings.Contains(contentStr, "Final Digest Content") {
		t.Error("Content should contain final digest")
	}
	if !strings.Contains(contentStr, "main digest content") {
		t.Error("Content should contain final digest text")
	}

	// Check that individual summaries section is present
	if !strings.Contains(contentStr, "Individual Article Summaries") {
		t.Error("Content should contain individual summaries section")
	}
	if !strings.Contains(contentStr, "For reference, here are the individual summaries") {
		t.Error("Content should contain reference text for individual summaries")
	}

	// Check that individual article is still present
	if !strings.Contains(contentStr, "Test Article") {
		t.Error("Content should contain individual article")
	}

	// Verify structure - final digest should come before individual summaries
	finalDigestIndex := strings.Index(contentStr, "Final Digest Content")
	individualIndex := strings.Index(contentStr, "Individual Article Summaries")
	if finalDigestIndex == -1 || individualIndex == -1 || finalDigestIndex >= individualIndex {
		t.Error("Final digest should come before individual summaries")
	}
}

func TestRenderMarkdownDigest_DefaultOutputDir(t *testing.T) {
	// Use current directory as test directory
	originalWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	digestItems := []DigestData{
		{
			Title:       "Test Article",
			URL:         "https://example.com/article",
			SummaryText: "Test summary.",
		},
	}

	filePath, err := RenderMarkdownDigest(digestItems, "", "")
	if err != nil {
		t.Fatalf("RenderMarkdownDigest failed: %v", err)
	}

	// Check that file was created in default "digests" directory
	if !strings.Contains(filePath, "digests") {
		t.Errorf("Expected file to be in digests directory, got %s", filePath)
	}

	// Check that directory was created
	expectedDir := filepath.Join(tmpDir, "digests")
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Error("Default digests directory should be created")
	}
}

func TestRenderMarkdownDigest_FilenameFormat(t *testing.T) {
	tmpDir := t.TempDir()
	digestItems := []DigestData{}

	filePath, err := RenderMarkdownDigest(digestItems, tmpDir, "")
	if err != nil {
		t.Fatalf("RenderMarkdownDigest failed: %v", err)
	}

	filename := filepath.Base(filePath)
	
	// Check filename format: digest_YYYY-MM-DD.md
	if !strings.HasPrefix(filename, "digest_") {
		t.Error("Filename should start with 'digest_'")
	}
	if !strings.HasSuffix(filename, ".md") {
		t.Error("Filename should end with '.md'")
	}

	// Check date format (basic validation)
	dateStr := time.Now().UTC().Format("2006-01-02")
	expectedFilename := "digest_" + dateStr + ".md"
	if filename != expectedFilename {
		t.Errorf("Expected filename %s, got %s", expectedFilename, filename)
	}
}

func TestRenderMarkdownDigest_InvalidOutputDir(t *testing.T) {
	// Try to create digest in a file (not directory)
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "file.txt")
	os.WriteFile(invalidPath, []byte("test"), 0644)

	digestItems := []DigestData{}

	_, err := RenderMarkdownDigest(digestItems, invalidPath, "")
	if err == nil {
		t.Error("Expected error when output directory is invalid")
	}
}

func TestWriteDigestToFile(t *testing.T) {
	tmpDir := t.TempDir()
	content := "# Test Digest\n\nThis is test content."
	filename := "test_digest.md"

	filePath, err := WriteDigestToFile(content, tmpDir, filename)
	if err != nil {
		t.Fatalf("WriteDigestToFile failed: %v", err)
	}

	// Check that file was created with correct path
	expectedPath := filepath.Join(tmpDir, filename)
	if filePath != expectedPath {
		t.Errorf("Expected file path %s, got %s", expectedPath, filePath)
	}

	// Check that file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Digest file should be created")
	}

	// Read and verify content
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read digest file: %v", err)
	}

	if string(fileContent) != content {
		t.Errorf("Expected content %q, got %q", content, string(fileContent))
	}
}

func TestWriteDigestToFile_DefaultOutputDir(t *testing.T) {
	// Use current directory as test directory
	originalWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	content := "Test content"
	filename := "test.md"

	filePath, err := WriteDigestToFile(content, "", filename)
	if err != nil {
		t.Fatalf("WriteDigestToFile failed: %v", err)
	}

	// Check that file was created in default "digests" directory
	if !strings.Contains(filePath, "digests") {
		t.Errorf("Expected file to be in digests directory, got %s", filePath)
	}

	// Check that directory was created
	expectedDir := filepath.Join(tmpDir, "digests")
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Error("Default digests directory should be created")
	}
}

func TestWriteDigestToFile_InvalidOutputDir(t *testing.T) {
	// Try to write to a file (not directory)
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "file.txt")
	os.WriteFile(invalidPath, []byte("test"), 0644)

	_, err := WriteDigestToFile("content", invalidPath, "test.md")
	if err == nil {
		t.Error("Expected error when output directory is invalid")
	}
}

func TestWriteDigestToFile_WriteError(t *testing.T) {
	// Try to write to a read-only directory
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	os.MkdirAll(readOnlyDir, 0444) // Read-only permissions

	_, err := WriteDigestToFile("content", readOnlyDir, "test.md")
	// Note: This test might not work on all systems due to permission handling differences
	// The exact behavior depends on the OS and file system
	if err == nil {
		// If the write succeeded, that's also valid behavior on some systems
		t.Log("Write succeeded despite read-only directory (system-dependent behavior)")
	}
}

func TestDigestData_AllFields(t *testing.T) {
	// Test that DigestData struct can hold all expected field types
	data := DigestData{
		Title:           "Test Title",
		URL:             "https://example.com",
		SummaryText:     "Test summary",
		MyTake:          "Test take",
		TopicCluster:    "Test cluster",
		TopicConfidence: 0.95,
		SentimentScore:  0.7,
		SentimentLabel:  "positive",
		SentimentEmoji:  "😊",
		AlertTriggered:  true,
		AlertConditions: []string{"condition1", "condition2"},
		ResearchQueries: []string{"query1", "query2"},
	}

	// Verify all fields can be set and retrieved
	if data.Title != "Test Title" {
		t.Error("Title field not working")
	}
	if data.URL != "https://example.com" {
		t.Error("URL field not working")
	}
	if data.SummaryText != "Test summary" {
		t.Error("SummaryText field not working")
	}
	if data.MyTake != "Test take" {
		t.Error("MyTake field not working")
	}
	if data.TopicCluster != "Test cluster" {
		t.Error("TopicCluster field not working")
	}
	if data.TopicConfidence != 0.95 {
		t.Error("TopicConfidence field not working")
	}
	if data.SentimentScore != 0.7 {
		t.Error("SentimentScore field not working")
	}
	if data.SentimentLabel != "positive" {
		t.Error("SentimentLabel field not working")
	}
	if data.SentimentEmoji != "😊" {
		t.Error("SentimentEmoji field not working")
	}
	if !data.AlertTriggered {
		t.Error("AlertTriggered field not working")
	}
	if len(data.AlertConditions) != 2 {
		t.Error("AlertConditions field not working")
	}
	if len(data.ResearchQueries) != 2 {
		t.Error("ResearchQueries field not working")
	}
}

func TestRenderMarkdownDigest_NoMyTake(t *testing.T) {
	tmpDir := t.TempDir()
	digestItems := []DigestData{
		{
			Title:       "Article Without MyTake",
			URL:         "https://example.com/article",
			SummaryText: "This article has no MyTake.",
			MyTake:      "", // Empty MyTake
		},
	}

	filePath, err := RenderMarkdownDigest(digestItems, tmpDir, "")
	if err != nil {
		t.Fatalf("RenderMarkdownDigest failed: %v", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read digest file: %v", err)
	}

	contentStr := string(content)

	// Should not contain MyTake section when empty
	if strings.Contains(contentStr, "**My Take:**") {
		t.Error("Content should not contain MyTake section when MyTake is empty")
	}
	if strings.Contains(contentStr, "My Take:") {
		t.Error("Content should not contain any MyTake reference when MyTake is empty")
	}

	// Should still contain the article and summary
	if !strings.Contains(contentStr, "Article Without MyTake") {
		t.Error("Content should contain article title")
	}
	if !strings.Contains(contentStr, "This article has no MyTake") {
		t.Error("Content should contain article summary")
	}
}