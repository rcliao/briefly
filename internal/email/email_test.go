package email

import (
	"briefly/internal/render"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGetDefaultEmailTemplate(t *testing.T) {
	tmpl := GetDefaultEmailTemplate()

	if tmpl.Name != "default" {
		t.Errorf("Expected name 'default', got '%s'", tmpl.Name)
	}
	if !tmpl.IncludeCSS {
		t.Error("Default template should include CSS")
	}
	if tmpl.HeaderColor == "" {
		t.Error("HeaderColor should not be empty")
	}
	if tmpl.FontFamily == "" {
		t.Error("FontFamily should not be empty")
	}
	if !tmpl.ShowTopicClusters {
		t.Error("Default template should show topic clusters")
	}
	if !tmpl.ShowInsights {
		t.Error("Default template should show insights")
	}
}

func TestGetNewsletterEmailTemplate(t *testing.T) {
	tmpl := GetNewsletterEmailTemplate()

	if tmpl.Name != "newsletter" {
		t.Errorf("Expected name 'newsletter', got '%s'", tmpl.Name)
	}
	if !strings.Contains(tmpl.Subject, "Newsletter") {
		t.Error("Newsletter template subject should contain 'Newsletter'")
	}
	if tmpl.HeaderColor == "" {
		t.Error("HeaderColor should not be empty")
	}
	if !tmpl.ShowTopicClusters {
		t.Error("Newsletter template should show topic clusters")
	}
	if !tmpl.ShowInsights {
		t.Error("Newsletter template should show insights")
	}
}

func TestGetMinimalEmailTemplate(t *testing.T) {
	tmpl := GetMinimalEmailTemplate()

	if tmpl.Name != "minimal" {
		t.Errorf("Expected name 'minimal', got '%s'", tmpl.Name)
	}
	if tmpl.ShowTopicClusters {
		t.Error("Minimal template should not show topic clusters")
	}
	if tmpl.ShowInsights {
		t.Error("Minimal template should not show insights")
	}
	if tmpl.HeaderColor == "" {
		t.Error("HeaderColor should not be empty")
	}
}

func TestGetEmailCSS(t *testing.T) {
	tmpl := GetDefaultEmailTemplate()
	css := getEmailCSS(tmpl)

	if css == "" {
		t.Error("CSS should not be empty")
	}

	// Check that template values are included
	if !strings.Contains(css, tmpl.HeaderColor) {
		t.Error("CSS should contain header color")
	}
	if !strings.Contains(css, tmpl.FontFamily) {
		t.Error("CSS should contain font family")
	}
	if !strings.Contains(css, tmpl.MaxWidth) {
		t.Error("CSS should contain max width")
	}

	// Check for responsive media queries
	if !strings.Contains(css, "@media only screen") {
		t.Error("CSS should contain responsive media queries")
	}

	// Check for email-specific styles
	if !strings.Contains(css, "-webkit-text-size-adjust") {
		t.Error("CSS should contain email client compatibility styles")
	}
}

func TestGroupArticlesByTopic(t *testing.T) {
	digestItems := []render.DigestData{
		{
			Title:           "Tech Article 1",
			TopicCluster:    "Technology",
			TopicConfidence: 0.9,
		},
		{
			Title:           "Tech Article 2",
			TopicCluster:    "Technology",
			TopicConfidence: 0.8,
		},
		{
			Title:           "Science Article",
			TopicCluster:    "Science",
			TopicConfidence: 0.95,
		},
		{
			Title:           "General Article",
			TopicCluster:    "", // Empty topic cluster
			TopicConfidence: 0.5,
		},
	}

	groups := groupArticlesByTopic(digestItems)

	if len(groups) != 3 {
		t.Errorf("Expected 3 topic groups, got %d", len(groups))
	}

	// Find groups by topic
	var techGroup, scienceGroup, generalGroup *TopicGroup
	for _, group := range groups {
		switch group.TopicCluster {
		case "Technology":
			techGroup = &group
		case "Science":
			scienceGroup = &group
		case "General":
			generalGroup = &group
		}
	}

	// Check Technology group
	if techGroup == nil {
		t.Error("Technology group should exist")
	} else {
		if len(techGroup.Articles) != 2 {
			t.Errorf("Technology group should have 2 articles, got %d", len(techGroup.Articles))
		}
		expectedAvg := (0.9 + 0.8) / 2
		if math.Abs(techGroup.AvgConfidence - expectedAvg) > 0.000001 {
			t.Errorf("Technology group average confidence should be approximately %.6f, got %.6f", expectedAvg, techGroup.AvgConfidence)
		}
	}

	// Check Science group
	if scienceGroup == nil {
		t.Error("Science group should exist")
	} else {
		if len(scienceGroup.Articles) != 1 {
			t.Errorf("Science group should have 1 article, got %d", len(scienceGroup.Articles))
		}
		if scienceGroup.AvgConfidence != 0.95 {
			t.Errorf("Science group confidence should be 0.95, got %f", scienceGroup.AvgConfidence)
		}
	}

	// Check General group (for empty topic cluster)
	if generalGroup == nil {
		t.Error("General group should exist for empty topic cluster")
	} else {
		if len(generalGroup.Articles) != 1 {
			t.Errorf("General group should have 1 article, got %d", len(generalGroup.Articles))
		}
	}
}

func TestGroupArticlesByTopic_EmptySlice(t *testing.T) {
	groups := groupArticlesByTopic([]render.DigestData{})
	if len(groups) != 0 {
		t.Errorf("Expected 0 groups for empty slice, got %d", len(groups))
	}
}

func TestRenderHTMLEmail_Basic(t *testing.T) {
	emailData := EmailData{
		Title:            "Test Digest",
		Date:             "January 1, 2024",
		Introduction:     "This is a test digest.",
		ExecutiveSummary: "Test executive summary.",
		DigestItems: []render.DigestData{
			{
				Title:          "Test Article",
				URL:            "https://example.com/article",
				SummaryText:    "Test article summary.",
				MyTake:         "My test take.",
				SentimentEmoji: "😊",
			},
		},
		Conclusion: "Test conclusion.",
	}

	tmpl := GetDefaultEmailTemplate()
	html, err := RenderHTMLEmail(emailData, tmpl)
	if err != nil {
		t.Fatalf("RenderHTMLEmail failed: %v", err)
	}

	if html == "" {
		t.Error("HTML output should not be empty")
	}

	// Check basic HTML structure
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("HTML should contain DOCTYPE declaration")
	}
	if !strings.Contains(html, "<html") && !strings.Contains(html, "</html>") {
		t.Error("HTML should contain html tags")
	}
	if !strings.Contains(html, "<head>") && !strings.Contains(html, "</head>") {
		t.Error("HTML should contain head section")
	}
	if !strings.Contains(html, "<body>") && !strings.Contains(html, "</body>") {
		t.Error("HTML should contain body section")
	}

	// Check content
	if !strings.Contains(html, "Test Digest") {
		t.Error("HTML should contain digest title")
	}
	if !strings.Contains(html, "January 1, 2024") {
		t.Error("HTML should contain date")
	}
	if !strings.Contains(html, "This is a test digest") {
		t.Error("HTML should contain introduction")
	}
	if !strings.Contains(html, "Test executive summary") {
		t.Error("HTML should contain executive summary")
	}
	if !strings.Contains(html, "Test Article") {
		t.Error("HTML should contain article title")
	}
	if !strings.Contains(html, "Test article summary") {
		t.Error("HTML should contain article summary")
	}
	if !strings.Contains(html, "My test take") {
		t.Error("HTML should contain my take")
	}
	if !strings.Contains(html, "Test conclusion") {
		t.Error("HTML should contain conclusion")
	}

	// Check emoji
	if !strings.Contains(html, "😊") {
		t.Error("HTML should contain sentiment emoji")
	}

	// Check CSS inclusion
	if !strings.Contains(html, "<style") {
		t.Error("HTML should contain CSS styles")
	}
}

func TestRenderHTMLEmail_WithTopicClusters(t *testing.T) {
	emailData := EmailData{
		Title: "Test Digest",
		Date:  "January 1, 2024",
		DigestItems: []render.DigestData{
			{
				Title:           "Tech Article",
				TopicCluster:    "Technology",
				TopicConfidence: 0.9,
				SummaryText:     "Tech summary.",
			},
			{
				Title:           "Science Article",
				TopicCluster:    "Science",
				TopicConfidence: 0.8,
				SummaryText:     "Science summary.",
			},
		},
	}

	tmpl := GetDefaultEmailTemplate()
	tmpl.ShowTopicClusters = true

	html, err := RenderHTMLEmail(emailData, tmpl)
	if err != nil {
		t.Fatalf("RenderHTMLEmail failed: %v", err)
	}

	// Check that topic groups are rendered
	if !strings.Contains(html, "Technology") {
		t.Error("HTML should contain Technology topic group")
	}
	if !strings.Contains(html, "Science") {
		t.Error("HTML should contain Science topic group")
	}

	// Check that articles are under their topic groups
	if !strings.Contains(html, "Tech Article") {
		t.Error("HTML should contain tech article under Technology group")
	}
	if !strings.Contains(html, "Science Article") {
		t.Error("HTML should contain science article under Science group")
	}
}

func TestRenderHTMLEmail_WithInsights(t *testing.T) {
	emailData := EmailData{
		Title:               "Test Digest",
		Date:                "January 1, 2024",
		OverallSentiment:    "Generally positive sentiment across articles",
		AlertsSummary:       "2 alerts triggered for breaking news",
		TrendsSummary:       "AI topics trending upward",
		ResearchSuggestions: []string{"Research query 1", "Research query 2"},
		DigestItems:         []render.DigestData{},
	}

	tmpl := GetDefaultEmailTemplate()
	tmpl.ShowInsights = true

	html, err := RenderHTMLEmail(emailData, tmpl)
	if err != nil {
		t.Fatalf("RenderHTMLEmail failed: %v", err)
	}

	// Check insights section
	if !strings.Contains(html, "AI-Powered Insights") {
		t.Error("HTML should contain insights section header")
	}
	if !strings.Contains(html, "Generally positive sentiment") {
		t.Error("HTML should contain sentiment analysis")
	}
	if !strings.Contains(html, "2 alerts triggered") {
		t.Error("HTML should contain alerts summary")
	}
	if !strings.Contains(html, "AI topics trending") {
		t.Error("HTML should contain trends summary")
	}
	if !strings.Contains(html, "Research query 1") {
		t.Error("HTML should contain research suggestions")
	}
}

func TestRenderHTMLEmail_MinimalTemplate(t *testing.T) {
	emailData := EmailData{
		Title: "Test Digest",
		Date:  "January 1, 2024",
		DigestItems: []render.DigestData{
			{
				Title:           "Test Article",
				TopicCluster:    "Technology",
				SummaryText:     "Test summary.",
				AlertTriggered:  true,
			},
		},
		OverallSentiment: "Positive",
	}

	tmpl := GetMinimalEmailTemplate()

	html, err := RenderHTMLEmail(emailData, tmpl)
	if err != nil {
		t.Fatalf("RenderHTMLEmail failed: %v", err)
	}

	// Minimal template should not show topic clusters
	if strings.Contains(html, "Technology") {
		t.Error("Minimal template should not show topic clusters")
	}

	// Minimal template should not show insights
	if strings.Contains(html, "AI-Powered Insights") {
		t.Error("Minimal template should not show insights section")
	}

	// Should still show basic article content
	if !strings.Contains(html, "Test Article") {
		t.Error("HTML should contain article title")
	}
}

func TestGenerateSubject(t *testing.T) {
	tmpl := GetDefaultEmailTemplate()
	title := "Weekly Tech Digest"
	date := "January 1, 2024"

	subject, err := GenerateSubject(tmpl, title, date)
	if err != nil {
		t.Fatalf("GenerateSubject failed: %v", err)
	}

	if subject == "" {
		t.Error("Subject should not be empty")
	}

	// Should contain the date
	if !strings.Contains(subject, date) {
		t.Error("Subject should contain the date")
	}

	// Should match the template pattern
	if !strings.Contains(subject, "Your Briefly Digest") {
		t.Error("Subject should match default template pattern")
	}
}

func TestGenerateSubject_InvalidTemplate(t *testing.T) {
	tmpl := &EmailTemplate{
		Subject: "{{.InvalidField}}", // Invalid template
	}

	_, err := GenerateSubject(tmpl, "title", "date")
	if err == nil {
		t.Error("Expected error for invalid subject template")
	}
}

func TestWriteHTMLEmail(t *testing.T) {
	tmpDir := t.TempDir()
	content := "<html><body>Test email content</body></html>"
	filename := "test_email.md" // Should be converted to .html

	filePath, err := WriteHTMLEmail(content, tmpDir, filename)
	if err != nil {
		t.Fatalf("WriteHTMLEmail failed: %v", err)
	}

	// Check file extension was converted
	if !strings.HasSuffix(filePath, ".html") {
		t.Error("File should have .html extension")
	}

	// Check file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("HTML email file should be created")
	}

	// Read and verify content
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read HTML email file: %v", err)
	}

	if string(fileContent) != content {
		t.Errorf("Expected content %q, got %q", content, string(fileContent))
	}
}

func TestWriteHTMLEmail_AlreadyHTMLExtension(t *testing.T) {
	tmpDir := t.TempDir()
	content := "<html><body>Test</body></html>"
	filename := "test_email.html" // Already has .html extension

	filePath, err := WriteHTMLEmail(content, tmpDir, filename)
	if err != nil {
		t.Fatalf("WriteHTMLEmail failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, filename)
	if filePath != expectedPath {
		t.Errorf("Expected file path %s, got %s", expectedPath, filePath)
	}
}

func TestConvertDigestToEmail(t *testing.T) {
	digestItems := []render.DigestData{
		{
			Title:       "Test Article",
			SummaryText: "Test summary",
		},
	}

	title := "Test Digest Title"
	intro := "Test introduction"
	execSummary := "Test executive summary"
	conclusion := "Test conclusion"
	sentiment := "Positive"
	alerts := "No alerts"
	trends := "Upward trends"
	research := []string{"Query 1", "Query 2"}

	emailData := ConvertDigestToEmail(
		digestItems, title, intro, execSummary, conclusion,
		sentiment, alerts, trends, research,
	)

	if emailData.Title != title {
		t.Errorf("Expected title %s, got %s", title, emailData.Title)
	}
	if emailData.Introduction != intro {
		t.Errorf("Expected introduction %s, got %s", intro, emailData.Introduction)
	}
	if emailData.ExecutiveSummary != execSummary {
		t.Errorf("Expected executive summary %s, got %s", execSummary, emailData.ExecutiveSummary)
	}
	if emailData.Conclusion != conclusion {
		t.Errorf("Expected conclusion %s, got %s", conclusion, emailData.Conclusion)
	}
	if emailData.OverallSentiment != sentiment {
		t.Errorf("Expected sentiment %s, got %s", sentiment, emailData.OverallSentiment)
	}
	if emailData.AlertsSummary != alerts {
		t.Errorf("Expected alerts %s, got %s", alerts, emailData.AlertsSummary)
	}
	if emailData.TrendsSummary != trends {
		t.Errorf("Expected trends %s, got %s", trends, emailData.TrendsSummary)
	}
	if len(emailData.ResearchSuggestions) != 2 {
		t.Errorf("Expected 2 research suggestions, got %d", len(emailData.ResearchSuggestions))
	}
	if len(emailData.DigestItems) != 1 {
		t.Errorf("Expected 1 digest item, got %d", len(emailData.DigestItems))
	}

	// Check date format
	if emailData.Date == "" {
		t.Error("Date should not be empty")
	}
	
	// Date should be in a reasonable format
	_, err := time.Parse("January 2, 2006", emailData.Date)
	if err != nil {
		t.Errorf("Date format should be parseable: %v", err)
	}
}

func TestEmailData_Structure(t *testing.T) {
	// Test that EmailData can hold all expected fields
	data := EmailData{
		Title:               "Test",
		Date:                "January 1, 2024",
		Introduction:        "Intro",
		ExecutiveSummary:    "Summary",
		DigestItems:         []render.DigestData{},
		TopicGroups:         []TopicGroup{},
		OverallSentiment:    "Positive",
		AlertsSummary:       "Alerts",
		TrendsSummary:       "Trends",
		ResearchSuggestions: []string{"Research"},
		Conclusion:          "Conclusion",
	}

	// Verify all fields can be set
	if data.Title != "Test" {
		t.Error("Title field not working")
	}
	if len(data.DigestItems) != 0 {
		t.Error("DigestItems field not working")
	}
	if len(data.TopicGroups) != 0 {
		t.Error("TopicGroups field not working")
	}
	if len(data.ResearchSuggestions) != 1 {
		t.Error("ResearchSuggestions field not working")
	}
}

func TestTopicGroup_Structure(t *testing.T) {
	// Test TopicGroup structure
	group := TopicGroup{
		TopicCluster:  "Technology",
		Articles:      []render.DigestData{},
		AvgConfidence: 0.95,
	}

	if group.TopicCluster != "Technology" {
		t.Error("TopicCluster field not working")
	}
	if len(group.Articles) != 0 {
		t.Error("Articles field not working")
	}
	if group.AvgConfidence != 0.95 {
		t.Error("AvgConfidence field not working")
	}
}

func TestEmailTemplate_Structure(t *testing.T) {
	// Test EmailTemplate structure
	tmpl := EmailTemplate{
		Name:              "test",
		Subject:           "Test Subject",
		IncludeCSS:        true,
		HeaderColor:       "#000000",
		BackgroundColor:   "#ffffff",
		TextColor:         "#333333",
		LinkColor:         "#0066cc",
		BorderColor:       "#dddddd",
		MaxWidth:          "600px",
		FontFamily:        "Arial",
		ShowTopicClusters: true,
		ShowInsights:      true,
	}

	// Verify all fields can be set
	if tmpl.Name != "test" {
		t.Error("Name field not working")
	}
	if !tmpl.IncludeCSS {
		t.Error("IncludeCSS field not working")
	}
	if !tmpl.ShowTopicClusters {
		t.Error("ShowTopicClusters field not working")
	}
	if !tmpl.ShowInsights {
		t.Error("ShowInsights field not working")
	}
}