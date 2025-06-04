package sentiment

import (
	"briefly/internal/core"
	"fmt"
	"strings"
	"time"
)

// SentimentScore represents the sentiment analysis result
type SentimentScore struct {
	Overall    float64 `json:"overall"`    // Overall sentiment score (-1.0 to 1.0)
	Positive   float64 `json:"positive"`   // Positive sentiment confidence (0.0 to 1.0)
	Negative   float64 `json:"negative"`   // Negative sentiment confidence (0.0 to 1.0)
	Neutral    float64 `json:"neutral"`    // Neutral sentiment confidence (0.0 to 1.0)
	Confidence float64 `json:"confidence"` // Overall confidence in the analysis (0.0 to 1.0)
}

// SentimentClassification represents the discrete sentiment category
type SentimentClassification string

const (
	SentimentVeryPositive SentimentClassification = "very_positive"
	SentimentPositive     SentimentClassification = "positive"
	SentimentNeutral      SentimentClassification = "neutral"
	SentimentNegative     SentimentClassification = "negative"
	SentimentVeryNegative SentimentClassification = "very_negative"
	SentimentMixed        SentimentClassification = "mixed"
)

// SentimentEmoji maps sentiment classifications to emojis
var SentimentEmoji = map[SentimentClassification]string{
	SentimentVeryPositive: "🚀",
	SentimentPositive:     "😊",
	SentimentNeutral:      "😐",
	SentimentNegative:     "😞",
	SentimentVeryNegative: "😱",
	SentimentMixed:        "🤔",
}

// ArticleSentiment contains sentiment analysis for an article
type ArticleSentiment struct {
	ArticleID      string                  `json:"article_id"`
	Title          string                  `json:"title"`
	URL            string                  `json:"url"`
	Score          SentimentScore          `json:"score"`
	Classification SentimentClassification `json:"classification"`
	Emoji          string                  `json:"emoji"`
	KeyPhrases     []string                `json:"key_phrases"` // Phrases that influenced sentiment
	AnalyzedAt     time.Time               `json:"analyzed_at"`
}

// DigestSentiment contains sentiment analysis for an entire digest
type DigestSentiment struct {
	DigestID          string             `json:"digest_id"`
	OverallScore      SentimentScore     `json:"overall_score"`
	OverallEmoji      string             `json:"overall_emoji"`
	ArticleSentiments []ArticleSentiment `json:"article_sentiments"`
	SentimentSummary  SentimentSummary   `json:"sentiment_summary"`
	AnalyzedAt        time.Time          `json:"analyzed_at"`
}

// SentimentSummary provides aggregate sentiment statistics
type SentimentSummary struct {
	TotalArticles int                             `json:"total_articles"`
	PositiveCount int                             `json:"positive_count"`
	NegativeCount int                             `json:"negative_count"`
	NeutralCount  int                             `json:"neutral_count"`
	MixedCount    int                             `json:"mixed_count"`
	Distribution  map[SentimentClassification]int `json:"distribution"`
	AverageScore  float64                         `json:"average_score"`
	DominantTone  SentimentClassification         `json:"dominant_tone"`
}

// SentimentAnalyzer handles sentiment analysis for articles and digests
type SentimentAnalyzer struct {
	// Configuration options can be added here
}

// NewSentimentAnalyzer creates a new sentiment analyzer
func NewSentimentAnalyzer() *SentimentAnalyzer {
	return &SentimentAnalyzer{}
}

// AnalyzeArticle performs sentiment analysis on a single article
func (sa *SentimentAnalyzer) AnalyzeArticle(article core.Article) (*ArticleSentiment, error) {
	// Combine title and content for analysis
	text := article.Title + " " + article.CleanedText

	// Perform sentiment analysis (simple rule-based approach)
	score := sa.calculateSentimentScore(text)
	classification := sa.classifySentiment(score)
	emoji := SentimentEmoji[classification]
	keyPhrases := sa.extractKeyPhrases(text, classification)

	return &ArticleSentiment{
		ArticleID:      article.ID,
		Title:          article.Title,
		URL:            article.LinkID, // Using LinkID as URL
		Score:          score,
		Classification: classification,
		Emoji:          emoji,
		KeyPhrases:     keyPhrases,
		AnalyzedAt:     time.Now(),
	}, nil
}

// AnalyzeDigest performs sentiment analysis on multiple articles in a digest
func (sa *SentimentAnalyzer) AnalyzeDigest(articles []core.Article, digestID string) (*DigestSentiment, error) {
	var articleSentiments []ArticleSentiment
	var totalScore float64
	distribution := make(map[SentimentClassification]int)

	// Analyze each article
	for _, article := range articles {
		sentiment, err := sa.AnalyzeArticle(article)
		if err != nil {
			continue // Skip articles with analysis errors
		}

		articleSentiments = append(articleSentiments, *sentiment)
		totalScore += sentiment.Score.Overall
		distribution[sentiment.Classification]++
	}

	if len(articleSentiments) == 0 {
		return nil, fmt.Errorf("no articles could be analyzed for sentiment")
	}

	// Calculate overall sentiment
	averageScore := totalScore / float64(len(articleSentiments))
	overallScore := SentimentScore{
		Overall:    averageScore,
		Confidence: 0.8, // Default confidence for aggregated scores
	}

	// Determine dominant tone
	dominantTone := sa.getDominantTone(distribution)
	overallEmoji := SentimentEmoji[dominantTone]

	// Create sentiment summary
	summary := SentimentSummary{
		TotalArticles: len(articleSentiments),
		Distribution:  distribution,
		AverageScore:  averageScore,
		DominantTone:  dominantTone,
	}

	// Count by broad categories
	for classification, count := range distribution {
		switch classification {
		case SentimentVeryPositive, SentimentPositive:
			summary.PositiveCount += count
		case SentimentVeryNegative, SentimentNegative:
			summary.NegativeCount += count
		case SentimentNeutral:
			summary.NeutralCount += count
		case SentimentMixed:
			summary.MixedCount += count
		}
	}

	return &DigestSentiment{
		DigestID:          digestID,
		OverallScore:      overallScore,
		OverallEmoji:      overallEmoji,
		ArticleSentiments: articleSentiments,
		SentimentSummary:  summary,
		AnalyzedAt:        time.Now(),
	}, nil
}

// calculateSentimentScore performs rule-based sentiment analysis
func (sa *SentimentAnalyzer) calculateSentimentScore(text string) SentimentScore {
	text = strings.ToLower(text)

	// Define sentiment keywords and their weights
	positiveKeywords := map[string]float64{
		"excellent": 1.0, "amazing": 0.9, "outstanding": 0.9, "fantastic": 0.8,
		"great": 0.7, "good": 0.6, "positive": 0.6, "success": 0.7, "win": 0.6,
		"improvement": 0.5, "growth": 0.6, "innovation": 0.7, "breakthrough": 0.8,
		"efficient": 0.6, "effective": 0.6, "beneficial": 0.6, "advantage": 0.5,
		"profit": 0.6, "revenue": 0.5, "gain": 0.5, "achievement": 0.7,
		"opportunity": 0.5, "advance": 0.6, "progress": 0.6, "upgrade": 0.5,
		"optimize": 0.5, "enhance": 0.5, "boost": 0.6, "increase": 0.4,
		"launch": 0.4, "release": 0.3, "new": 0.3, "fresh": 0.4,
	}

	negativeKeywords := map[string]float64{
		"terrible": -1.0, "awful": -0.9, "horrible": -0.9, "disaster": -0.8,
		"bad": -0.6, "poor": -0.6, "negative": -0.6, "failure": -0.7, "lose": -0.6,
		"problem": -0.5, "issue": -0.4, "concern": -0.4, "risk": -0.5, "threat": -0.6,
		"decline": -0.6, "decrease": -0.5, "drop": -0.5, "fall": -0.4, "loss": -0.6,
		"error": -0.5, "bug": -0.4, "fault": -0.5, "flaw": -0.5, "weakness": -0.4,
		"crisis": -0.8, "emergency": -0.7, "alert": -0.6, "warning": -0.5,
		"breach": -0.7, "hack": -0.7, "attack": -0.6, "vulnerability": -0.6,
		"outage": -0.6, "downtime": -0.5, "shutdown": -0.5, "closure": -0.6,
	}

	neutralKeywords := map[string]float64{
		"update": 0.0, "change": 0.0, "announce": 0.0, "report": 0.0,
		"analysis": 0.0, "study": 0.0, "research": 0.0, "data": 0.0,
		"information": 0.0, "details": 0.0, "facts": 0.0, "statistics": 0.0,
	}

	var positiveScore, negativeScore, neutralScore float64
	var totalWords int

	words := strings.Fields(text)
	totalWords = len(words)

	// Calculate scores based on keyword matches
	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, ".,!?;:")

		if weight, exists := positiveKeywords[word]; exists {
			positiveScore += weight
		}
		if weight, exists := negativeKeywords[word]; exists {
			negativeScore += -weight // Convert to positive number for negative sentiment
		}
		if _, exists := neutralKeywords[word]; exists {
			neutralScore += 0.3
		}
	}

	// Normalize scores
	if totalWords > 0 {
		positiveScore = positiveScore / float64(totalWords) * 100
		negativeScore = negativeScore / float64(totalWords) * 100
		neutralScore = neutralScore / float64(totalWords) * 100
	}

	// Calculate overall sentiment (-1.0 to 1.0)
	overall := (positiveScore - negativeScore) / (positiveScore + negativeScore + 1.0)

	// Ensure scores are in valid ranges
	if positiveScore > 1.0 {
		positiveScore = 1.0
	}
	if negativeScore > 1.0 {
		negativeScore = 1.0
	}
	if neutralScore > 1.0 {
		neutralScore = 1.0
	}

	// Calculate confidence based on the strength of sentiment signals
	confidence := (positiveScore + negativeScore) / 2.0
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.3 {
		confidence = 0.3 // Minimum confidence
	}

	return SentimentScore{
		Overall:    overall,
		Positive:   positiveScore,
		Negative:   negativeScore,
		Neutral:    neutralScore,
		Confidence: confidence,
	}
}

// classifySentiment converts a sentiment score to a classification
func (sa *SentimentAnalyzer) classifySentiment(score SentimentScore) SentimentClassification {
	overall := score.Overall

	// Check for mixed sentiment (high positive and negative scores)
	if score.Positive > 0.3 && score.Negative > 0.3 {
		return SentimentMixed
	}

	// Classify based on overall score
	if overall >= 0.7 {
		return SentimentVeryPositive
	} else if overall >= 0.2 {
		return SentimentPositive
	} else if overall <= -0.7 {
		return SentimentVeryNegative
	} else if overall <= -0.2 {
		return SentimentNegative
	} else {
		return SentimentNeutral
	}
}

// extractKeyPhrases identifies phrases that contributed to the sentiment
func (sa *SentimentAnalyzer) extractKeyPhrases(text string, classification SentimentClassification) []string {
	// This is a simplified implementation - in production, you'd want more sophisticated phrase extraction
	phrases := []string{}
	lowercaseText := strings.ToLower(text)

	// Define key phrases for different sentiment categories
	var targetPhrases []string

	switch classification {
	case SentimentVeryPositive, SentimentPositive:
		targetPhrases = []string{
			"breakthrough", "innovation", "success", "achievement", "excellent",
			"outstanding", "remarkable", "impressive", "significant progress",
			"major improvement", "game-changing", "revolutionary",
		}
	case SentimentVeryNegative, SentimentNegative:
		targetPhrases = []string{
			"major problem", "significant issue", "serious concern", "critical failure",
			"security breach", "data leak", "major outage", "system failure",
			"widespread issues", "catastrophic", "devastating",
		}
	case SentimentMixed:
		targetPhrases = []string{
			"mixed results", "some concerns", "partially successful", "ongoing issues",
			"improvements needed", "work in progress", "challenges remain",
		}
	}

	// Find matching phrases
	for _, phrase := range targetPhrases {
		if strings.Contains(lowercaseText, phrase) {
			phrases = append(phrases, phrase)
		}
	}

	// Limit to top 3 phrases
	if len(phrases) > 3 {
		phrases = phrases[:3]
	}

	return phrases
}

// getDominantTone determines the most common sentiment classification
func (sa *SentimentAnalyzer) getDominantTone(distribution map[SentimentClassification]int) SentimentClassification {
	maxCount := 0
	var dominantTone SentimentClassification = SentimentNeutral

	for classification, count := range distribution {
		if count > maxCount {
			maxCount = count
			dominantTone = classification
		}
	}

	return dominantTone
}

// FormatSentimentSummary creates a human-readable sentiment summary
func (sa *SentimentAnalyzer) FormatSentimentSummary(digestSentiment *DigestSentiment) string {
	var builder strings.Builder

	builder.WriteString("## 📊 Sentiment Analysis\n\n")
	builder.WriteString(fmt.Sprintf("**Overall Sentiment:** %s %s (Score: %.2f)\n\n",
		digestSentiment.OverallEmoji,
		strings.Title(strings.ReplaceAll(string(digestSentiment.SentimentSummary.DominantTone), "_", " ")),
		digestSentiment.OverallScore.Overall))

	summary := digestSentiment.SentimentSummary
	builder.WriteString("**Article Breakdown:**\n")
	builder.WriteString(fmt.Sprintf("- 😊 Positive: %d articles\n", summary.PositiveCount))
	builder.WriteString(fmt.Sprintf("- 😞 Negative: %d articles\n", summary.NegativeCount))
	builder.WriteString(fmt.Sprintf("- 😐 Neutral: %d articles\n", summary.NeutralCount))
	if summary.MixedCount > 0 {
		builder.WriteString(fmt.Sprintf("- 🤔 Mixed: %d articles\n", summary.MixedCount))
	}

	builder.WriteString("\n")

	return builder.String()
}

// FormatArticleSentiments creates a formatted list of article sentiments
func (sa *SentimentAnalyzer) FormatArticleSentiments(articleSentiments []ArticleSentiment) string {
	var builder strings.Builder

	builder.WriteString("### Article Sentiments\n\n")

	for _, sentiment := range articleSentiments {
		builder.WriteString(fmt.Sprintf("**%s** %s\n", sentiment.Emoji, sentiment.Title))
		if len(sentiment.KeyPhrases) > 0 {
			builder.WriteString(fmt.Sprintf("*Key phrases: %s*\n", strings.Join(sentiment.KeyPhrases, ", ")))
		}
		builder.WriteString("\n")
	}

	return builder.String()
}
