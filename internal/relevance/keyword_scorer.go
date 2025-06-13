package relevance

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
)

// KeywordScorer implements fast keyword-based relevance scoring
type KeywordScorer struct {
	// Configuration
	caseSensitive bool
	stemming      bool
	stopWords     map[string]bool
}

// NewKeywordScorer creates a new keyword-based scorer
func NewKeywordScorer() *KeywordScorer {
	return &KeywordScorer{
		caseSensitive: false,
		stemming:      false, // Simple implementation without stemming for now
		stopWords:     getCommonStopWords(),
	}
}

// Score calculates relevance score for a single piece of content
func (ks *KeywordScorer) Score(ctx context.Context, content Scorable, criteria Criteria) (Score, error) {
	scores, err := ks.ScoreBatch(ctx, []Scorable{content}, criteria)
	if err != nil {
		return Score{}, err
	}
	if len(scores) == 0 {
		return Score{}, fmt.Errorf("no scores returned")
	}
	return scores[0], nil
}

// ScoreBatch calculates relevance scores for multiple pieces of content
func (ks *KeywordScorer) ScoreBatch(ctx context.Context, contents []Scorable, criteria Criteria) ([]Score, error) {
	if len(contents) == 0 {
		return []Score{}, nil
	}

	// Extract and prepare keywords from query
	queryKeywords := ks.extractKeywords(criteria.Query)
	allKeywords := append(queryKeywords, criteria.Keywords...)

	// Remove duplicates and stop words
	keywords := ks.cleanKeywords(allKeywords)

	if len(keywords) == 0 {
		// No keywords to score against, return neutral scores
		scores := make([]Score, len(contents))
		for i := range scores {
			scores[i] = Score{
				Value:      0.5, // Neutral score
				Confidence: 0.1, // Low confidence
				Factors:    map[string]float64{},
				Reasoning:  "No keywords to score against",
				Metadata:   map[string]interface{}{"method": "keyword", "keywords_count": 0},
			}
		}
		return scores, nil
	}

	scores := make([]Score, len(contents))
	for i, content := range contents {
		score, err := ks.scoreContent(content, keywords, criteria.Weights)
		if err != nil {
			return nil, fmt.Errorf("failed to score content %d: %w", i, err)
		}
		scores[i] = score
	}

	return scores, nil
}

// scoreContent calculates relevance score for a single piece of content
func (ks *KeywordScorer) scoreContent(content Scorable, keywords []string, weights ScoringWeights) (Score, error) {
	title := ks.normalizeText(content.GetTitle())
	contentText := ks.normalizeText(content.GetContent())

	// Calculate individual factor scores
	factors := make(map[string]float64)

	// 1. Content relevance score
	factors["content_relevance"] = ks.calculateTextRelevance(contentText, keywords)

	// 2. Title relevance score
	factors["title_relevance"] = ks.calculateTextRelevance(title, keywords)

	// 3. Authority score (basic domain-based scoring)
	factors["authority"] = ks.calculateAuthorityScore(content.GetURL())

	// 4. Recency score (if timestamp available in metadata)
	factors["recency"] = ks.calculateRecencyScore(content.GetMetadata())

	// 5. Quality score (basic content quality indicators)
	factors["quality"] = ks.calculateQualityScore(content)

	// Calculate weighted overall score
	overallScore := 0.0
	overallScore += factors["content_relevance"] * weights.ContentRelevance
	overallScore += factors["title_relevance"] * weights.TitleRelevance
	overallScore += factors["authority"] * weights.Authority
	overallScore += factors["recency"] * weights.Recency
	overallScore += factors["quality"] * weights.Quality

	// Ensure score is in [0, 1] range
	overallScore = math.Max(0.0, math.Min(1.0, overallScore))

	// Calculate confidence based on keyword matches and content length
	confidence := ks.calculateConfidence(factors, len(contentText), len(keywords))

	// Generate reasoning
	reasoning := ks.generateReasoning(factors, keywords, weights)

	return Score{
		Value:      overallScore,
		Confidence: confidence,
		Factors:    factors,
		Reasoning:  reasoning,
		Metadata: map[string]interface{}{
			"method":         "keyword",
			"keywords":       keywords,
			"keyword_count":  len(keywords),
			"content_length": len(contentText),
		},
	}, nil
}

// calculateTextRelevance calculates how relevant text is to keywords
func (ks *KeywordScorer) calculateTextRelevance(text string, keywords []string) float64 {
	if len(text) == 0 || len(keywords) == 0 {
		return 0.0
	}

	text = strings.ToLower(text)
	totalMatches := 0
	uniqueMatches := 0

	for _, keyword := range keywords {
		keyword = strings.ToLower(keyword)
		matches := strings.Count(text, keyword)
		if matches > 0 {
			uniqueMatches++
			totalMatches += matches
		}
	}

	if uniqueMatches == 0 {
		return 0.0
	}

	// Score based on percentage of keywords matched and frequency
	keywordCoverage := float64(uniqueMatches) / float64(len(keywords))

	// Bonus for multiple matches of same keyword (but with diminishing returns)
	frequency := math.Log(float64(totalMatches)+1) / math.Log(float64(len(keywords)*3)+1)

	// Combine coverage and frequency
	relevance := (keywordCoverage * 0.7) + (frequency * 0.3)

	return math.Min(1.0, relevance)
}

// calculateAuthorityScore provides basic domain authority scoring
func (ks *KeywordScorer) calculateAuthorityScore(url string) float64 {
	if len(url) == 0 {
		return 0.5 // Neutral score for missing URL
	}

	url = strings.ToLower(url)

	// High authority domains
	highAuthority := []string{
		"github.com", "stackoverflow.com", "arxiv.org", "ieee.org",
		"acm.org", "nature.com", "science.org", "mit.edu", "stanford.edu",
		"google.com", "microsoft.com", "openai.com", "anthropic.com",
	}

	// Medium authority domains
	mediumAuthority := []string{
		"medium.com", "dev.to", "hashnode.com", "substack.com",
		"techcrunch.com", "arstechnica.com", "wired.com", "theverge.com",
	}

	for _, domain := range highAuthority {
		if strings.Contains(url, domain) {
			return 0.9
		}
	}

	for _, domain := range mediumAuthority {
		if strings.Contains(url, domain) {
			return 0.7
		}
	}

	// Default authority for unknown domains
	return 0.5
}

// calculateRecencyScore calculates content freshness score
func (ks *KeywordScorer) calculateRecencyScore(metadata map[string]interface{}) float64 {
	if metadata == nil {
		return 0.5 // Neutral score if no metadata
	}

	// Look for timestamp in various formats
	timeFields := []string{"published", "date", "timestamp", "created_at", "date_published"}

	for _, field := range timeFields {
		if val, exists := metadata[field]; exists {
			if timestamp, ok := val.(time.Time); ok {
				return ks.timeToRecencyScore(timestamp)
			}
			if timeStr, ok := val.(string); ok {
				if timestamp, err := time.Parse(time.RFC3339, timeStr); err == nil {
					return ks.timeToRecencyScore(timestamp)
				}
			}
		}
	}

	return 0.5 // Neutral score if no valid timestamp found
}

// timeToRecencyScore converts timestamp to recency score (1.0 = very recent, 0.0 = very old)
func (ks *KeywordScorer) timeToRecencyScore(timestamp time.Time) float64 {
	now := time.Now()
	age := now.Sub(timestamp)

	// Score based on age: 1.0 for today, decreasing over time
	daysSincePublication := age.Hours() / 24

	if daysSincePublication <= 1 {
		return 1.0
	} else if daysSincePublication <= 7 {
		return 0.8
	} else if daysSincePublication <= 30 {
		return 0.6
	} else if daysSincePublication <= 90 {
		return 0.4
	} else if daysSincePublication <= 365 {
		return 0.2
	} else {
		return 0.1
	}
}

// calculateQualityScore provides basic content quality assessment
func (ks *KeywordScorer) calculateQualityScore(content Scorable) float64 {
	title := content.GetTitle()
	contentText := content.GetContent()

	score := 0.5 // Start with neutral

	// Length indicators
	if len(contentText) > 1000 {
		score += 0.2 // Bonus for substantial content
	}
	if len(contentText) < 100 {
		score -= 0.3 // Penalty for very short content
	}

	// Title quality
	if len(title) > 10 && len(title) < 100 {
		score += 0.1 // Good title length
	}

	// Sentence structure (basic check)
	sentences := strings.Split(contentText, ".")
	if len(sentences) > 3 {
		score += 0.1 // Multiple sentences indicate structure
	}

	// Penalize excessive caps or poor formatting
	if strings.ToUpper(title) == title && len(title) > 10 {
		score -= 0.2 // All caps title
	}

	return math.Max(0.0, math.Min(1.0, score))
}

// calculateConfidence determines confidence in the relevance score
func (ks *KeywordScorer) calculateConfidence(factors map[string]float64, contentLength, keywordCount int) float64 {
	confidence := 0.5 // Base confidence

	// Higher confidence with more content to analyze
	if contentLength > 500 {
		confidence += 0.2
	} else if contentLength < 100 {
		confidence -= 0.2
	}

	// Higher confidence with more keywords
	if keywordCount > 3 {
		confidence += 0.2
	} else if keywordCount < 2 {
		confidence -= 0.2
	}

	// Higher confidence if multiple factors agree
	highFactors := 0
	for _, score := range factors {
		if score > 0.6 {
			highFactors++
		}
	}
	if highFactors > 1 {
		confidence += 0.1
	}

	return math.Max(0.1, math.Min(1.0, confidence))
}

// generateReasoning creates human-readable explanation of the score
func (ks *KeywordScorer) generateReasoning(factors map[string]float64, keywords []string, weights ScoringWeights) string {
	var reasons []string

	if factors["content_relevance"] > 0.6 {
		reasons = append(reasons, "Strong keyword matches in content")
	} else if factors["content_relevance"] < 0.3 {
		reasons = append(reasons, "Weak keyword matches in content")
	}

	if factors["title_relevance"] > 0.6 {
		reasons = append(reasons, "Relevant title")
	}

	if factors["authority"] > 0.7 {
		reasons = append(reasons, "High authority source")
	}

	if factors["recency"] > 0.8 {
		reasons = append(reasons, "Recent content")
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "Mixed relevance indicators")
	}

	return strings.Join(reasons, "; ")
}

// extractKeywords extracts keywords from query text
func (ks *KeywordScorer) extractKeywords(query string) []string {
	if len(query) == 0 {
		return []string{}
	}

	// Simple word extraction with basic punctuation removal
	reg := regexp.MustCompile(`[^\w\s]`)
	cleaned := reg.ReplaceAllString(query, " ")
	words := strings.Fields(cleaned)

	var keywords []string
	for _, word := range words {
		word = strings.ToLower(strings.TrimSpace(word))
		if len(word) > 2 && !ks.stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// cleanKeywords removes duplicates and stop words
func (ks *KeywordScorer) cleanKeywords(keywords []string) []string {
	seen := make(map[string]bool)
	var clean []string

	for _, keyword := range keywords {
		keyword = strings.ToLower(strings.TrimSpace(keyword))
		if len(keyword) > 2 && !ks.stopWords[keyword] && !seen[keyword] {
			seen[keyword] = true
			clean = append(clean, keyword)
		}
	}

	return clean
}

// normalizeText normalizes text for comparison
func (ks *KeywordScorer) normalizeText(text string) string {
	if !ks.caseSensitive {
		text = strings.ToLower(text)
	}

	// Basic normalization: remove extra whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	return text
}

// getCommonStopWords returns a set of common English stop words
func getCommonStopWords() map[string]bool {
	stopWords := []string{
		"a", "an", "and", "are", "as", "at", "be", "by", "for", "from",
		"has", "he", "in", "is", "it", "its", "of", "on", "that", "the",
		"to", "was", "were", "will", "with", "the", "this", "but", "they",
		"have", "had", "what", "said", "each", "which", "she", "do", "how",
		"their", "if", "up", "out", "many", "then", "them", "these", "so",
		"some", "her", "would", "make", "like", "into", "him", "time", "two",
	}

	stopWordsMap := make(map[string]bool)
	for _, word := range stopWords {
		stopWordsMap[word] = true
	}

	return stopWordsMap
}
