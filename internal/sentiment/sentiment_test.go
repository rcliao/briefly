package sentiment

import (
	"briefly/internal/core"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewSentimentAnalyzer(t *testing.T) {
	analyzer := NewSentimentAnalyzer()
	if analyzer == nil {
		t.Error("NewSentimentAnalyzer should return a non-nil analyzer")
	}
}

func TestSentimentClassificationConstants(t *testing.T) {
	// Test that all sentiment classifications have corresponding emojis
	expectedClassifications := []SentimentClassification{
		SentimentVeryPositive,
		SentimentPositive,
		SentimentNeutral,
		SentimentNegative,
		SentimentVeryNegative,
		SentimentMixed,
	}

	for _, classification := range expectedClassifications {
		if emoji, exists := SentimentEmoji[classification]; !exists || emoji == "" {
			t.Errorf("Classification %s should have a non-empty emoji", classification)
		}
	}
}

func TestCalculateSentimentScore_Positive(t *testing.T) {
	analyzer := NewSentimentAnalyzer()
	text := "This is an excellent breakthrough with amazing results and outstanding performance"

	score := analyzer.calculateSentimentScore(text)

	if score.Overall <= 0 {
		t.Error("Text with positive keywords should have positive overall score")
	}
	if score.Positive <= 0 {
		t.Error("Text with positive keywords should have positive sentiment score")
	}
	if score.Confidence <= 0 {
		t.Error("Confidence should be greater than 0")
	}
}

func TestCalculateSentimentScore_Negative(t *testing.T) {
	analyzer := NewSentimentAnalyzer()
	text := "This is a terrible disaster with awful problems and horrible failures"

	score := analyzer.calculateSentimentScore(text)

	if score.Overall >= 0 {
		t.Error("Text with negative keywords should have negative overall score")
	}
	if score.Negative <= 0 {
		t.Error("Text with negative keywords should have negative sentiment score")
	}
	if score.Confidence <= 0 {
		t.Error("Confidence should be greater than 0")
	}
}

func TestCalculateSentimentScore_Neutral(t *testing.T) {
	analyzer := NewSentimentAnalyzer()
	text := "This is a report about data analysis and research findings"

	score := analyzer.calculateSentimentScore(text)

	// Neutral text should have scores close to zero
	if score.Overall > 0.3 || score.Overall < -0.3 {
		t.Errorf("Neutral text should have overall score near zero, got %f", score.Overall)
	}
	if score.Confidence <= 0 {
		t.Error("Confidence should be greater than 0")
	}
}

func TestCalculateSentimentScore_Mixed(t *testing.T) {
	analyzer := NewSentimentAnalyzer()
	text := "This excellent breakthrough has some terrible problems and outstanding achievements despite awful failures"

	score := analyzer.calculateSentimentScore(text)

	if score.Positive <= 0 {
		t.Error("Mixed text should have positive component")
	}
	if score.Negative <= 0 {
		t.Error("Mixed text should have negative component")
	}
}

func TestClassifySentiment(t *testing.T) {
	analyzer := NewSentimentAnalyzer()

	testCases := []struct {
		score    SentimentScore
		expected SentimentClassification
	}{
		{
			score:    SentimentScore{Overall: 0.8, Positive: 0.8, Negative: 0.1},
			expected: SentimentVeryPositive,
		},
		{
			score:    SentimentScore{Overall: 0.5, Positive: 0.6, Negative: 0.1},
			expected: SentimentPositive,
		},
		{
			score:    SentimentScore{Overall: 0.0, Positive: 0.2, Negative: 0.2},
			expected: SentimentNeutral,
		},
		{
			score:    SentimentScore{Overall: -0.5, Positive: 0.1, Negative: 0.6},
			expected: SentimentNegative,
		},
		{
			score:    SentimentScore{Overall: -0.8, Positive: 0.1, Negative: 0.8},
			expected: SentimentVeryNegative,
		},
		{
			score:    SentimentScore{Overall: 0.1, Positive: 0.4, Negative: 0.4},
			expected: SentimentMixed,
		},
	}

	for _, tc := range testCases {
		result := analyzer.classifySentiment(tc.score)
		if result != tc.expected {
			t.Errorf("Expected classification %s, got %s for score %+v", tc.expected, result, tc.score)
		}
	}
}

func TestExtractKeyPhrases(t *testing.T) {
	analyzer := NewSentimentAnalyzer()

	testCases := []struct {
		text           string
		classification SentimentClassification
		shouldContain  string
	}{
		{
			text:           "This is a breakthrough innovation with excellent results",
			classification: SentimentVeryPositive,
			shouldContain:  "breakthrough",
		},
		{
			text:           "Major problem with security breach and system failure",
			classification: SentimentVeryNegative,
			shouldContain:  "security breach",
		},
		{
			text:           "Mixed results with some concerns but improvements needed",
			classification: SentimentMixed,
			shouldContain:  "mixed results",
		},
	}

	for _, tc := range testCases {
		phrases := analyzer.extractKeyPhrases(tc.text, tc.classification)

		found := false
		for _, phrase := range phrases {
			if strings.Contains(phrase, tc.shouldContain) {
				found = true
				break
			}
		}

		if tc.shouldContain != "" && !found {
			t.Errorf("Expected to find phrase containing '%s' in %v", tc.shouldContain, phrases)
		}

		// Should not return more than 3 phrases
		if len(phrases) > 3 {
			t.Errorf("Should not return more than 3 key phrases, got %d", len(phrases))
		}
	}
}

func TestAnalyzeArticle(t *testing.T) {
	analyzer := NewSentimentAnalyzer()

	article := core.Article{
		ID:          uuid.NewString(),
		LinkID:      "https://example.com/article",
		Title:       "Amazing Breakthrough in Technology",
		CleanedText: "This excellent innovation represents outstanding progress in the field",
	}

	sentiment, err := analyzer.AnalyzeArticle(article)
	if err != nil {
		t.Fatalf("AnalyzeArticle failed: %v", err)
	}

	if sentiment == nil {
		t.Fatal("Expected sentiment result, got nil")
	}

	if sentiment.ArticleID != article.ID {
		t.Errorf("Expected ArticleID %s, got %s", article.ID, sentiment.ArticleID)
	}
	if sentiment.Title != article.Title {
		t.Errorf("Expected title %s, got %s", article.Title, sentiment.Title)
	}
	if sentiment.URL != article.LinkID {
		t.Errorf("Expected URL %s, got %s", article.LinkID, sentiment.URL)
	}

	// Should be positive sentiment
	if sentiment.Score.Overall <= 0 {
		t.Error("Expected positive sentiment for positive article")
	}
	if sentiment.Classification != SentimentPositive && sentiment.Classification != SentimentVeryPositive {
		t.Errorf("Expected positive classification, got %s", sentiment.Classification)
	}
	if sentiment.Emoji == "" {
		t.Error("Emoji should not be empty")
	}
	if sentiment.AnalyzedAt.IsZero() {
		t.Error("AnalyzedAt should be set")
	}
}

func TestAnalyzeDigest_Success(t *testing.T) {
	analyzer := NewSentimentAnalyzer()

	articles := []core.Article{
		{
			ID:          uuid.NewString(),
			LinkID:      "https://example.com/article1",
			Title:       "Excellent breakthrough",
			CleanedText: "Amazing innovation with outstanding results",
		},
		{
			ID:          uuid.NewString(),
			LinkID:      "https://example.com/article2",
			Title:       "Terrible disaster",
			CleanedText: "Awful problems with horrible failures",
		},
		{
			ID:          uuid.NewString(),
			LinkID:      "https://example.com/article3",
			Title:       "Research update",
			CleanedText: "Analysis of data and research findings",
		},
	}

	digestID := uuid.NewString()
	digestSentiment, err := analyzer.AnalyzeDigest(articles, digestID)
	if err != nil {
		t.Fatalf("AnalyzeDigest failed: %v", err)
	}

	if digestSentiment == nil {
		t.Fatal("Expected digest sentiment result, got nil")
	}

	if digestSentiment.DigestID != digestID {
		t.Errorf("Expected DigestID %s, got %s", digestID, digestSentiment.DigestID)
	}

	// Should have analyzed all articles
	if len(digestSentiment.ArticleSentiments) != 3 {
		t.Errorf("Expected 3 article sentiments, got %d", len(digestSentiment.ArticleSentiments))
	}

	// Check summary
	summary := digestSentiment.SentimentSummary
	if summary.TotalArticles != 3 {
		t.Errorf("Expected 3 total articles, got %d", summary.TotalArticles)
	}
	if summary.PositiveCount < 1 {
		t.Error("Should have at least 1 positive article")
	}
	if summary.NegativeCount < 1 {
		t.Error("Should have at least 1 negative article")
	}
	if summary.Distribution == nil {
		t.Error("Distribution should not be nil")
	}
	if summary.DominantTone == "" {
		t.Error("DominantTone should not be empty")
	}

	// Overall sentiment should be set
	if digestSentiment.OverallScore.Confidence <= 0 {
		t.Error("Overall confidence should be greater than 0")
	}
	if digestSentiment.OverallEmoji == "" {
		t.Error("Overall emoji should not be empty")
	}
	if digestSentiment.AnalyzedAt.IsZero() {
		t.Error("AnalyzedAt should be set")
	}
}

func TestAnalyzeDigest_EmptyArticles(t *testing.T) {
	analyzer := NewSentimentAnalyzer()

	articles := []core.Article{}
	digestID := uuid.NewString()

	_, err := analyzer.AnalyzeDigest(articles, digestID)
	if err == nil {
		t.Error("Expected error for empty articles slice")
	}
	if !strings.Contains(err.Error(), "no articles could be analyzed") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestGetDominantTone(t *testing.T) {
	analyzer := NewSentimentAnalyzer()

	testCases := []struct {
		distribution map[SentimentClassification]int
		expected     SentimentClassification
	}{
		{
			distribution: map[SentimentClassification]int{
				SentimentPositive: 5,
				SentimentNegative: 2,
				SentimentNeutral:  1,
			},
			expected: SentimentPositive,
		},
		{
			distribution: map[SentimentClassification]int{
				SentimentNegative: 4,
				SentimentPositive: 2,
				SentimentNeutral:  1,
			},
			expected: SentimentNegative,
		},
		{
			distribution: map[SentimentClassification]int{},
			expected:     SentimentNeutral, // Default when empty
		},
	}

	for _, tc := range testCases {
		result := analyzer.getDominantTone(tc.distribution)
		if result != tc.expected {
			t.Errorf("Expected dominant tone %s, got %s for distribution %v", tc.expected, result, tc.distribution)
		}
	}
}

func TestFormatSentimentSummary(t *testing.T) {
	analyzer := NewSentimentAnalyzer()

	digestSentiment := &DigestSentiment{
		OverallScore: SentimentScore{
			Overall: 0.3,
		},
		OverallEmoji: "😊",
		SentimentSummary: SentimentSummary{
			TotalArticles: 5,
			PositiveCount: 3,
			NegativeCount: 1,
			NeutralCount:  1,
			MixedCount:    0,
			DominantTone:  SentimentPositive,
		},
	}

	formatted := analyzer.FormatSentimentSummary(digestSentiment)

	if formatted == "" {
		t.Error("Formatted summary should not be empty")
	}
	if !strings.Contains(formatted, "Sentiment Analysis") {
		t.Error("Should contain sentiment analysis header")
	}
	if !strings.Contains(formatted, "😊") {
		t.Error("Should contain overall emoji")
	}
	if !strings.Contains(formatted, "Positive: 3") {
		t.Error("Should contain positive count")
	}
	if !strings.Contains(formatted, "Negative: 1") {
		t.Error("Should contain negative count")
	}
	if !strings.Contains(formatted, "Neutral: 1") {
		t.Error("Should contain neutral count")
	}
	if !strings.Contains(formatted, "0.30") {
		t.Error("Should contain overall score")
	}
}

func TestFormatArticleSentiments(t *testing.T) {
	analyzer := NewSentimentAnalyzer()

	articleSentiments := []ArticleSentiment{
		{
			Title:      "Positive Article",
			Emoji:      "😊",
			KeyPhrases: []string{"excellent", "breakthrough"},
		},
		{
			Title:      "Negative Article",
			Emoji:      "😞",
			KeyPhrases: []string{"problem", "failure"},
		},
		{
			Title:      "Neutral Article",
			Emoji:      "😐",
			KeyPhrases: []string{},
		},
	}

	formatted := analyzer.FormatArticleSentiments(articleSentiments)

	if formatted == "" {
		t.Error("Formatted article sentiments should not be empty")
	}
	if !strings.Contains(formatted, "Article Sentiments") {
		t.Error("Should contain article sentiments header")
	}
	if !strings.Contains(formatted, "Positive Article") {
		t.Error("Should contain first article title")
	}
	if !strings.Contains(formatted, "😊") {
		t.Error("Should contain first article emoji")
	}
	if !strings.Contains(formatted, "excellent, breakthrough") {
		t.Error("Should contain key phrases for first article")
	}
	if !strings.Contains(formatted, "Negative Article") {
		t.Error("Should contain second article title")
	}
	if !strings.Contains(formatted, "Neutral Article") {
		t.Error("Should contain third article title")
	}
}

func TestSentimentScore_Ranges(t *testing.T) {
	analyzer := NewSentimentAnalyzer()

	// Test with extreme positive text
	positiveText := strings.Repeat("excellent amazing outstanding fantastic ", 20)
	score := analyzer.calculateSentimentScore(positiveText)

	// Scores should be within valid ranges
	if score.Overall < -1.0 || score.Overall > 1.0 {
		t.Errorf("Overall score %f should be between -1.0 and 1.0", score.Overall)
	}
	if score.Positive < 0.0 || score.Positive > 1.0 {
		t.Errorf("Positive score %f should be between 0.0 and 1.0", score.Positive)
	}
	if score.Negative < 0.0 || score.Negative > 1.0 {
		t.Errorf("Negative score %f should be between 0.0 and 1.0", score.Negative)
	}
	if score.Neutral < 0.0 || score.Neutral > 1.0 {
		t.Errorf("Neutral score %f should be between 0.0 and 1.0", score.Neutral)
	}
	if score.Confidence < 0.0 || score.Confidence > 1.0 {
		t.Errorf("Confidence %f should be between 0.0 and 1.0", score.Confidence)
	}
}

func TestAnalyzeArticle_EmptyContent(t *testing.T) {
	analyzer := NewSentimentAnalyzer()

	article := core.Article{
		ID:          uuid.NewString(),
		LinkID:      "https://example.com/article",
		Title:       "",
		CleanedText: "",
	}

	sentiment, err := analyzer.AnalyzeArticle(article)
	if err != nil {
		t.Fatalf("AnalyzeArticle should handle empty content: %v", err)
	}

	if sentiment == nil {
		t.Fatal("Expected sentiment result for empty article")
	}

	// Should have minimum confidence
	if sentiment.Score.Confidence < 0.3 {
		t.Error("Should have minimum confidence even for empty content")
	}
}

func TestSentimentStructures(t *testing.T) {
	// Test that all sentiment structures can be created and have expected fields

	score := SentimentScore{
		Overall:    0.5,
		Positive:   0.7,
		Negative:   0.2,
		Neutral:    0.1,
		Confidence: 0.8,
	}

	if score.Overall != 0.5 {
		t.Error("SentimentScore Overall field not working")
	}

	articleSentiment := ArticleSentiment{
		ArticleID:      "test-id",
		Title:          "Test Title",
		URL:            "https://example.com",
		Score:          score,
		Classification: SentimentPositive,
		Emoji:          "😊",
		KeyPhrases:     []string{"phrase1", "phrase2"},
		AnalyzedAt:     time.Now(),
	}

	if articleSentiment.Classification != SentimentPositive {
		t.Error("ArticleSentiment Classification field not working")
	}

	summary := SentimentSummary{
		TotalArticles: 5,
		PositiveCount: 3,
		NegativeCount: 1,
		NeutralCount:  1,
		MixedCount:    0,
		Distribution:  make(map[SentimentClassification]int),
		AverageScore:  0.3,
		DominantTone:  SentimentPositive,
	}

	if summary.TotalArticles != 5 {
		t.Error("SentimentSummary TotalArticles field not working")
	}

	digestSentiment := DigestSentiment{
		DigestID:          "digest-id",
		OverallScore:      score,
		OverallEmoji:      "😊",
		ArticleSentiments: []ArticleSentiment{articleSentiment},
		SentimentSummary:  summary,
		AnalyzedAt:        time.Now(),
	}

	if len(digestSentiment.ArticleSentiments) != 1 {
		t.Error("DigestSentiment ArticleSentiments field not working")
	}
}

func TestSentimentAnalyzer_Integration(t *testing.T) {
	// Integration test that tests the full sentiment analysis pipeline
	analyzer := NewSentimentAnalyzer()

	articles := []core.Article{
		{
			ID:          uuid.NewString(),
			LinkID:      "https://example.com/positive",
			Title:       "Revolutionary AI Breakthrough Achieves Outstanding Results",
			CleanedText: "This excellent innovation represents a major breakthrough in artificial intelligence, with amazing performance improvements and outstanding efficiency gains.",
		},
		{
			ID:          uuid.NewString(),
			LinkID:      "https://example.com/negative",
			Title:       "Major Security Breach Causes Widespread System Failures",
			CleanedText: "A terrible security vulnerability led to catastrophic system failures, causing major problems for users and significant data loss.",
		},
		{
			ID:          uuid.NewString(),
			LinkID:      "https://example.com/neutral",
			Title:       "Quarterly Financial Report Released",
			CleanedText: "The company released its quarterly financial report showing revenue data and market analysis for the period.",
		},
	}

	// Analyze individual articles
	for _, article := range articles {
		sentiment, err := analyzer.AnalyzeArticle(article)
		if err != nil {
			t.Fatalf("Failed to analyze article %s: %v", article.Title, err)
		}

		if sentiment.ArticleID != article.ID {
			t.Error("Article sentiment should have correct article ID")
		}
		if sentiment.AnalyzedAt.IsZero() {
			t.Error("Article sentiment should have analyzed timestamp")
		}
	}

	// Analyze digest
	digestSentiment, err := analyzer.AnalyzeDigest(articles, "test-digest")
	if err != nil {
		t.Fatalf("Failed to analyze digest: %v", err)
	}

	// Verify digest analysis
	if len(digestSentiment.ArticleSentiments) != 3 {
		t.Error("Digest should contain all article sentiments")
	}
	if digestSentiment.SentimentSummary.TotalArticles != 3 {
		t.Error("Summary should show correct total articles")
	}
	if digestSentiment.SentimentSummary.PositiveCount < 1 {
		t.Error("Should detect at least one positive article")
	}
	if digestSentiment.SentimentSummary.NegativeCount < 1 {
		t.Error("Should detect at least one negative article")
	}

	// Format outputs
	summaryFormatted := analyzer.FormatSentimentSummary(digestSentiment)
	if summaryFormatted == "" {
		t.Error("Formatted summary should not be empty")
	}

	articlesFormatted := analyzer.FormatArticleSentiments(digestSentiment.ArticleSentiments)
	if articlesFormatted == "" {
		t.Error("Formatted articles should not be empty")
	}
}
