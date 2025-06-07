package deepresearch

import (
	"context"
	"sort"
	"strings"
)

// EmbeddingRanker implements content ranking using embeddings and heuristics
type EmbeddingRanker struct {
	// For MVP, we'll use simple text-based ranking
	// In the future, this could integrate with embedding models
}

// NewEmbeddingRanker creates a new embedding-based ranker
func NewEmbeddingRanker() *EmbeddingRanker {
	return &EmbeddingRanker{}
}

// RankSources ranks sources by relevance to the topic and removes duplicates
func (r *EmbeddingRanker) RankSources(ctx context.Context, sources []Source, topic string) ([]Source, error) {
	if len(sources) == 0 {
		return sources, nil
	}

	// Step 1: Remove duplicates based on URL and content similarity
	dedupedSources := r.deduplicateSources(sources)

	// Step 2: Calculate relevance scores
	for i := range dedupedSources {
		dedupedSources[i].Relevance = r.calculateRelevanceScore(dedupedSources[i], topic)
	}

	// Step 3: Sort by relevance score (highest first)
	sort.Slice(dedupedSources, func(i, j int) bool {
		return dedupedSources[i].Relevance > dedupedSources[j].Relevance
	})

	// Step 4: Apply diversity filtering to ensure source variety
	rankedSources := r.diversifySourceSelection(dedupedSources)

	return rankedSources, nil
}

// deduplicateSources removes duplicate sources based on URL and content similarity
func (r *EmbeddingRanker) deduplicateSources(sources []Source) []Source {
	seen := make(map[string]bool)
	var unique []Source

	for _, source := range sources {
		// Create a key for deduplication
		key := r.createDeduplicationKey(source)
		
		if !seen[key] {
			seen[key] = true
			unique = append(unique, source)
		}
	}

	return unique
}

// createDeduplicationKey creates a key for identifying duplicate sources
func (r *EmbeddingRanker) createDeduplicationKey(source Source) string {
	// Use domain + simplified title for deduplication
	simplified := strings.ToLower(source.Title)
	simplified = strings.ReplaceAll(simplified, " ", "")
	simplified = strings.ReplaceAll(simplified, "-", "")
	simplified = strings.ReplaceAll(simplified, "_", "")
	
	// Keep only first 50 characters to handle slight variations
	if len(simplified) > 50 {
		simplified = simplified[:50]
	}
	
	return source.Domain + ":" + simplified
}

// calculateRelevanceScore calculates a relevance score for a source
func (r *EmbeddingRanker) calculateRelevanceScore(source Source, topic string) float64 {
	score := 0.0

	// Factor 1: Title relevance (weight: 0.4)
	titleScore := r.calculateTextRelevance(source.Title, topic)
	score += titleScore * 0.4

	// Factor 2: Content relevance (weight: 0.3)
	contentScore := r.calculateTextRelevance(source.Content, topic)
	score += contentScore * 0.3

	// Factor 3: Source authority (weight: 0.2)
	authorityScore := r.calculateAuthorityScore(source)
	score += authorityScore * 0.2

	// Factor 4: Recency bonus (weight: 0.1)
	recencyScore := r.calculateRecencyScore(source)
	score += recencyScore * 0.1

	return score
}

// calculateTextRelevance calculates how relevant text is to the topic
func (r *EmbeddingRanker) calculateTextRelevance(text, topic string) float64 {
	if text == "" || topic == "" {
		return 0.0
	}

	textLower := strings.ToLower(text)
	topicLower := strings.ToLower(topic)

	// Extract keywords from topic
	topicWords := r.extractKeywords(topicLower)
	
	matchCount := 0
	totalWords := len(topicWords)

	// Count keyword matches
	for _, word := range topicWords {
		if strings.Contains(textLower, word) {
			matchCount++
		}
	}

	if totalWords == 0 {
		return 0.0
	}

	return float64(matchCount) / float64(totalWords)
}

// extractKeywords extracts meaningful keywords from text
func (r *EmbeddingRanker) extractKeywords(text string) []string {
	// Remove common stop words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "is": true,
		"are": true, "was": true, "were": true, "be": true, "been": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
	}

	words := strings.Fields(text)
	var keywords []string

	for _, word := range words {
		// Clean the word
		word = strings.Trim(word, ".,!?;:")
		word = strings.ToLower(word)

		// Skip if it's a stop word or too short
		if len(word) < 3 || stopWords[word] {
			continue
		}

		keywords = append(keywords, word)
	}

	return keywords
}

// calculateAuthorityScore assigns authority scores based on source type and domain
func (r *EmbeddingRanker) calculateAuthorityScore(source Source) float64 {
	score := 0.5 // Base score

	// Bonus for academic sources
	switch source.Type {
	case "paper":
		score += 0.3
	case "news":
		score += 0.2
	case "repo":
		score += 0.15
	case "blog":
		score += 0.1
	}

	// Domain-specific authority bonuses
	domain := strings.ToLower(source.Domain)
	
	// High authority domains
	highAuthority := []string{
		"arxiv.org", "doi.org", "pubmed.ncbi.nlm.nih.gov",
		"github.com", "stackoverflow.com", "medium.com",
		"nytimes.com", "washingtonpost.com", "reuters.com",
		"bbc.co.uk", "cnn.com", "techcrunch.com",
	}

	for _, auth := range highAuthority {
		if strings.Contains(domain, auth) {
			score += 0.2
			break
		}
	}

	// Penalty for potentially low-quality domains
	lowQuality := []string{"wordpress.com", "blogspot.com", "wix.com"}
	for _, low := range lowQuality {
		if strings.Contains(domain, low) {
			score -= 0.1
			break
		}
	}

	// Ensure score stays within bounds
	if score > 1.0 {
		score = 1.0
	}
	if score < 0.0 {
		score = 0.0
	}

	return score
}

// calculateRecencyScore gives bonus points for recent content
func (r *EmbeddingRanker) calculateRecencyScore(source Source) float64 {
	// For MVP, we'll use a simple time-based decay
	// In practice, this would use actual publication dates from the content
	
	// Since we don't have reliable publication dates from search results,
	// we'll use retrieval time as a proxy and give a small bonus
	return 0.1 // Small constant bonus for now
}

// diversifySourceSelection ensures variety in source types and domains
func (r *EmbeddingRanker) diversifySourceSelection(sources []Source) []Source {
	if len(sources) <= 10 {
		return sources // No need to diversify if we have few sources
	}

	var result []Source
	domainCount := make(map[string]int)
	typeCount := make(map[string]int)
	
	// First pass: take highest-ranking sources while maintaining diversity
	for _, source := range sources {
		// Check diversity constraints
		if domainCount[source.Domain] >= 2 { // Max 2 articles per domain
			continue
		}
		if typeCount[source.Type] >= 5 { // Max 5 articles per type
			continue
		}

		result = append(result, source)
		domainCount[source.Domain]++
		typeCount[source.Type]++

		// Stop when we have enough diverse sources
		if len(result) >= 25 {
			break
		}
	}

	// Second pass: fill remaining slots if needed
	if len(result) < 15 {
		for _, source := range sources {
			// Skip if already included
			included := false
			for _, existing := range result {
				if existing.URL == source.URL {
					included = true
					break
				}
			}
			if included {
				continue
			}

			// Add remaining sources until we reach minimum threshold
			result = append(result, source)
			if len(result) >= 15 {
				break
			}
		}
	}

	return result
}