package templates

import (
	"briefly/internal/render"
	"math"
	"strings"
	"testing"
)

func TestDigestFormatConstants(t *testing.T) {
	expectedFormats := map[DigestFormat]string{
		FormatBrief:      "brief",
		FormatStandard:   "standard",
		FormatDetailed:   "detailed",
		FormatNewsletter: "newsletter",
		FormatEmail:      "email",
	}

	for format, expectedValue := range expectedFormats {
		if string(format) != expectedValue {
			t.Errorf("Expected %s to be %s, got %s", format, expectedValue, string(format))
		}
	}
}

func TestGetTemplateBrief(t *testing.T) {
	template := GetTemplate(FormatBrief)

	if template.Format != FormatBrief {
		t.Errorf("Expected format to be %s, got %s", FormatBrief, template.Format)
	}
	if template.Title != "Brief Digest" {
		t.Errorf("Expected title to be 'Brief Digest', got %s", template.Title)
	}
	if !template.IncludeSummaries {
		t.Error("Expected IncludeSummaries to be true for brief format")
	}
	if template.IncludeKeyInsights {
		t.Error("Expected IncludeKeyInsights to be false for brief format")
	}
	if template.IncludeIndividualArticles {
		t.Error("Expected IncludeIndividualArticles to be false for brief format")
	}
	if template.MaxSummaryLength != 150 {
		t.Errorf("Expected MaxSummaryLength to be 150, got %d", template.MaxSummaryLength)
	}
}

func TestGetTemplateStandard(t *testing.T) {
	template := GetTemplate(FormatStandard)

	if template.Format != FormatStandard {
		t.Errorf("Expected format to be %s, got %s", FormatStandard, template.Format)
	}
	if template.Title != "Daily Digest" {
		t.Errorf("Expected title to be 'Daily Digest', got %s", template.Title)
	}
	if !template.IncludeSummaries {
		t.Error("Expected IncludeSummaries to be true for standard format")
	}
	if !template.IncludeKeyInsights {
		t.Error("Expected IncludeKeyInsights to be true for standard format")
	}
	if !template.IncludeIndividualArticles {
		t.Error("Expected IncludeIndividualArticles to be true for standard format")
	}
	if !template.IncludeTopicClustering {
		t.Error("Expected IncludeTopicClustering to be true for standard format")
	}
	if template.MaxSummaryLength != 300 {
		t.Errorf("Expected MaxSummaryLength to be 300, got %d", template.MaxSummaryLength)
	}
}

func TestGetTemplateDetailed(t *testing.T) {
	template := GetTemplate(FormatDetailed)

	if template.Format != FormatDetailed {
		t.Errorf("Expected format to be %s, got %s", FormatDetailed, template.Format)
	}
	if template.Title != "Comprehensive Digest" {
		t.Errorf("Expected title to be 'Comprehensive Digest', got %s", template.Title)
	}
	if !template.IncludeActionItems {
		t.Error("Expected IncludeActionItems to be true for detailed format")
	}
	if template.MaxSummaryLength != 0 {
		t.Errorf("Expected MaxSummaryLength to be 0 (no limit), got %d", template.MaxSummaryLength)
	}
}

func TestGetTemplateNewsletter(t *testing.T) {
	template := GetTemplate(FormatNewsletter)

	if template.Format != FormatNewsletter {
		t.Errorf("Expected format to be %s, got %s", FormatNewsletter, template.Format)
	}
	if template.Title != "Weekly Newsletter" {
		t.Errorf("Expected title to be 'Weekly Newsletter', got %s", template.Title)
	}
	if !template.IncludePromptCorner {
		t.Error("Expected IncludePromptCorner to be true for newsletter format")
	}
	if template.IncludeIndividualArticles {
		t.Error("Expected IncludeIndividualArticles to be false for newsletter format")
	}
}

func TestGetTemplateEmail(t *testing.T) {
	template := GetTemplate(FormatEmail)

	if template.Format != FormatEmail {
		t.Errorf("Expected format to be %s, got %s", FormatEmail, template.Format)
	}
	if template.Title != "Email Digest" {
		t.Errorf("Expected title to be 'Email Digest', got %s", template.Title)
	}
	if !template.IncludeTopicClustering {
		t.Error("Expected IncludeTopicClustering to be true for email format")
	}
}

func TestGetTemplateUnknown(t *testing.T) {
	template := GetTemplate("unknown")

	// Should default to standard
	if template.Format != FormatStandard {
		t.Errorf("Expected unknown format to default to %s, got %s", FormatStandard, template.Format)
	}
}

func TestGroupArticlesByTopic(t *testing.T) {
	digestItems := []render.DigestData{
		{
			Title:           "AI Article 1",
			TopicCluster:    "Technology",
			TopicConfidence: 0.9,
		},
		{
			Title:           "AI Article 2",
			TopicCluster:    "Technology",
			TopicConfidence: 0.8,
		},
		{
			Title:           "Finance Article",
			TopicCluster:    "Finance",
			TopicConfidence: 0.7,
		},
		{
			Title:           "Uncategorized Article",
			TopicCluster:    "",
			TopicConfidence: 0.0,
		},
	}

	groups := GroupArticlesByTopic(digestItems)

	if len(groups) != 3 {
		t.Errorf("Expected 3 topic groups, got %d", len(groups))
	}

	// Check that groups are sorted by average confidence
	if groups[0].AvgConfidence < groups[1].AvgConfidence {
		t.Error("Expected groups to be sorted by average confidence (descending)")
	}

	// Find Technology group
	var techGroup *TopicGroup
	for i := range groups {
		if groups[i].TopicCluster == "Technology" {
			techGroup = &groups[i]
			break
		}
	}

	if techGroup == nil {
		t.Fatal("Expected to find Technology topic group")
	}

	if len(techGroup.Articles) != 2 {
		t.Errorf("Expected Technology group to have 2 articles, got %d", len(techGroup.Articles))
	}

	expectedAvgConfidence := (0.9 + 0.8) / 2.0
	tolerance := 0.000001
	if math.Abs(techGroup.AvgConfidence - expectedAvgConfidence) > tolerance {
		t.Errorf("Expected Technology group average confidence to be %.6f, got %.6f", expectedAvgConfidence, techGroup.AvgConfidence)
	}

	// Find General group (for uncategorized)
	var generalGroup *TopicGroup
	for i := range groups {
		if groups[i].TopicCluster == "General" {
			generalGroup = &groups[i]
			break
		}
	}

	if generalGroup == nil {
		t.Fatal("Expected to find General topic group for uncategorized articles")
	}

	if len(generalGroup.Articles) != 1 {
		t.Errorf("Expected General group to have 1 article, got %d", len(generalGroup.Articles))
	}
}

func TestGroupArticlesByTopicEmpty(t *testing.T) {
	groups := GroupArticlesByTopic([]render.DigestData{})

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups for empty input, got %d", len(groups))
	}
}

func TestGetSentimentEmoji(t *testing.T) {
	tests := []struct {
		sentiment string
		expected  string
	}{
		{"positive", "😊"},
		{"very positive", "😊"},
		{"negative", "😟"},
		{"very negative", "😟"},
		{"neutral", "😐"},
		{"mixed", "🤔"},
		{"unknown", "📄"},
		{"", "📄"},
	}

	for _, test := range tests {
		result := getSentimentEmoji(test.sentiment)
		if result != test.expected {
			t.Errorf("getSentimentEmoji(%q) = %q, expected %q", test.sentiment, result, test.expected)
		}
	}
}

func TestGetAvailableFormats(t *testing.T) {
	formats := GetAvailableFormats()

	expectedFormats := []string{
		"brief",
		"standard",
		"detailed",
		"newsletter",
		"email",
	}

	if len(formats) != len(expectedFormats) {
		t.Errorf("Expected %d formats, got %d", len(expectedFormats), len(formats))
	}

	formatMap := make(map[string]bool)
	for _, format := range formats {
		formatMap[format] = true
	}

	for _, expected := range expectedFormats {
		if !formatMap[expected] {
			t.Errorf("Expected format %s to be in available formats list", expected)
		}
	}
}

func TestDigestTemplateFields(t *testing.T) {
	template := DigestTemplate{
		Format:                    FormatStandard,
		Title:                     "Test Digest",
		IncludeSummaries:          true,
		IncludeKeyInsights:        true,
		IncludeActionItems:        false,
		IncludeSourceLinks:        true,
		IncludePromptCorner:       false,
		IncludeIndividualArticles: true,
		IncludeTopicClustering:    true,
		MaxSummaryLength:          200,
		IntroductionText:          "Welcome to the digest",
		ConclusionText:            "Thank you for reading",
		SectionSeparator:          "\n---\n",
	}

	if template.Format != FormatStandard {
		t.Errorf("Expected Format to be %s, got %s", FormatStandard, template.Format)
	}
	if template.Title != "Test Digest" {
		t.Errorf("Expected Title to be 'Test Digest', got %s", template.Title)
	}
	if !template.IncludeSummaries {
		t.Error("Expected IncludeSummaries to be true")
	}
	if !template.IncludeKeyInsights {
		t.Error("Expected IncludeKeyInsights to be true")
	}
	if template.IncludeActionItems {
		t.Error("Expected IncludeActionItems to be false")
	}
	if !template.IncludeSourceLinks {
		t.Error("Expected IncludeSourceLinks to be true")
	}
	if template.IncludePromptCorner {
		t.Error("Expected IncludePromptCorner to be false")
	}
	if !template.IncludeIndividualArticles {
		t.Error("Expected IncludeIndividualArticles to be true")
	}
	if !template.IncludeTopicClustering {
		t.Error("Expected IncludeTopicClustering to be true")
	}
	if template.MaxSummaryLength != 200 {
		t.Errorf("Expected MaxSummaryLength to be 200, got %d", template.MaxSummaryLength)
	}
}

func TestTopicGroupCreation(t *testing.T) {
	articles := []render.DigestData{
		{Title: "Article 1", TopicConfidence: 0.8},
		{Title: "Article 2", TopicConfidence: 0.9},
	}

	group := TopicGroup{
		TopicCluster:  "Technology",
		Articles:      articles,
		AvgConfidence: 0.85,
	}

	if group.TopicCluster != "Technology" {
		t.Errorf("Expected TopicCluster to be 'Technology', got %s", group.TopicCluster)
	}
	if len(group.Articles) != 2 {
		t.Errorf("Expected 2 articles, got %d", len(group.Articles))
	}
	if group.AvgConfidence != 0.85 {
		t.Errorf("Expected AvgConfidence to be 0.85, got %f", group.AvgConfidence)
	}
}

func TestRenderInsightsSectionEmpty(t *testing.T) {
	template := GetTemplate(FormatBrief)
	digestItems := []render.DigestData{}
	
	result := renderInsightsSection(digestItems, template, "", "", "", []string{})
	
	if result != "" {
		t.Errorf("Expected empty insights section for brief format, got: %s", result)
	}
}

func TestRenderInsightsSectionWithData(t *testing.T) {
	template := GetTemplate(FormatDetailed)
	digestItems := []render.DigestData{
		{SentimentLabel: "positive"},
		{SentimentLabel: "negative"},
		{SentimentLabel: "positive"},
	}
	
	result := renderInsightsSection(digestItems, template, "Overall positive sentiment", "No alerts", "Trending topics", []string{"AI research", "Tech trends"})
	
	if !strings.Contains(result, "## 🧠 AI-Powered Insights") {
		t.Error("Expected insights section header")
	}
	if !strings.Contains(result, "📊 Sentiment Analysis") {
		t.Error("Expected sentiment analysis section")
	}
	if !strings.Contains(result, "Overall positive sentiment") {
		t.Error("Expected overall sentiment text")
	}
	if !strings.Contains(result, "🔍 Research Suggestions") {
		t.Error("Expected research suggestions section")
	}
	if !strings.Contains(result, "AI research") {
		t.Error("Expected research suggestion content")
	}
}