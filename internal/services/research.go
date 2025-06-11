package services

import (
	"briefly/internal/core"
	"briefly/internal/llm"
	"briefly/internal/search"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ResearchServiceImpl implements the ResearchService interface
type ResearchServiceImpl struct {
	llmClient    *llm.Client
	searchClient search.Provider
}

// NewResearchService creates a new research service
func NewResearchService(llmClient *llm.Client, searchProvider search.Provider) *ResearchServiceImpl {
	return &ResearchServiceImpl{
		llmClient:    llmClient,
		searchClient: searchProvider,
	}
}

// PerformResearch conducts comprehensive research on a given topic
func (r *ResearchServiceImpl) PerformResearch(ctx context.Context, query string, depth int) (*core.ResearchReport, error) {
	// Generate search queries using LLM
	queries, err := r.generateSearchQueries(ctx, query, depth)
	if err != nil {
		return nil, fmt.Errorf("failed to generate search queries: %w", err)
	}

	// Execute searches for each query
	var allResults []core.ResearchResult
	for _, searchQuery := range queries {
		results, err := r.executeSearch(ctx, searchQuery)
		if err != nil {
			// Log error but continue with other queries
			continue
		}
		allResults = append(allResults, results...)
	}

	// Score and rank results
	rankedResults := r.rankResults(allResults, query)

	// Generate summary using LLM
	summary, err := r.generateSummary(ctx, query, rankedResults)
	if err != nil {
		return nil, fmt.Errorf("failed to generate research summary: %w", err)
	}

	// Create research report
	report := &core.ResearchReport{
		ID:               uuid.New().String(),
		Query:            query,
		Depth:            depth,
		GeneratedQueries: queries,
		Results:          rankedResults,
		Summary:          summary,
		DateGenerated:    time.Now().UTC(),
		TotalResults:     len(rankedResults),
		RelevanceScore:   r.calculateOverallRelevance(rankedResults),
	}

	return report, nil
}

// GenerateResearchQueries generates research queries for a given article
func (r *ResearchServiceImpl) GenerateResearchQueries(ctx context.Context, article core.Article) ([]string, error) {
	prompt := fmt.Sprintf(`Based on this article content, generate 3-5 research queries that would help find related or follow-up information:

Title: %s
Content: %s

Generate specific, targeted search queries that would find:
1. Related developments in this field
2. Different perspectives on this topic
3. Background or foundational information
4. Recent news or updates

Format: Return only the search queries, one per line.`, article.Title, r.truncateContent(article.CleanedText, 1000))

	response, err := r.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		MaxTokens:   300,
		Temperature: 0.7,
		Model:       "gemini-1.5-flash",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate research queries: %w", err)
	}

	queries := strings.Split(strings.TrimSpace(response), "\n")
	var cleanQueries []string
	for _, query := range queries {
		query = strings.TrimSpace(query)
		if query != "" && !strings.HasPrefix(query, "#") {
			cleanQueries = append(cleanQueries, query)
		}
	}

	return cleanQueries, nil
}

// AnalyzeTopics analyzes a collection of articles to identify common topics
func (r *ResearchServiceImpl) AnalyzeTopics(ctx context.Context, articles []core.Article) ([]string, error) {
	var content strings.Builder
	for _, article := range articles {
		content.WriteString(fmt.Sprintf("Title: %s\nContent: %s\n\n",
			article.Title, r.truncateContent(article.CleanedText, 200)))
	}

	prompt := fmt.Sprintf(`Analyze the following articles and identify the top 5-7 main topics or themes:

%s

For each topic, provide:
1. A clear topic name (2-4 words)
2. Brief description

Format: Return as "Topic Name: Description", one per line.`, content.String())

	response, err := r.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		MaxTokens:   400,
		Temperature: 0.5,
		Model:       "gemini-1.5-flash",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to analyze topics: %w", err)
	}

	topics := strings.Split(strings.TrimSpace(response), "\n")
	var cleanTopics []string
	for _, topic := range topics {
		topic = strings.TrimSpace(topic)
		if topic != "" && strings.Contains(topic, ":") {
			cleanTopics = append(cleanTopics, topic)
		}
	}

	return cleanTopics, nil
}

// generateSearchQueries creates search queries using LLM based on the topic and depth
func (r *ResearchServiceImpl) generateSearchQueries(ctx context.Context, topic string, depth int) ([]string, error) {
	var prompt string
	switch depth {
	case 1:
		prompt = fmt.Sprintf(`Generate 3 simple search queries for basic research on: %s

Create short search queries (3-6 words each) covering:
1. Basic overview
2. Recent developments 2024
3. Key examples

Examples: "AI testing", "machine learning validation", "automated testing tools"

Format: Return only the search queries, one per line.`, topic)
	case 2:
		prompt = fmt.Sprintf(`Generate 5 simple search queries for moderate research on: %s

Create short search queries (3-7 words each) covering:
1. Overview and definition
2. Current tools and trends
3. Case studies examples
4. Common challenges
5. Best practices

Examples: "AI testing frameworks", "ML model validation tools", "testing challenges AI"

Format: Return only the search queries, one per line.`, topic)
	default: // depth 3+
		prompt = fmt.Sprintf(`Generate 7 simple search queries for comprehensive research on: %s

Create short, effective search queries (3-8 words each) covering:
1. Basic overview and definition
2. Current tools and frameworks
3. Recent developments 2024
4. Industry case studies
5. Common challenges
6. Future trends
7. Best practices

Examples of good queries:
- "AI testing tools 2024"
- "machine learning model validation"
- "automated testing frameworks"

Format: Return only the search queries, one per line. Keep each query short and simple.`, topic)
	}

	response, err := r.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		MaxTokens:   500,
		Temperature: 0.7,
		Model:       "gemini-1.5-flash",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate search queries: %w", err)
	}

	queries := strings.Split(strings.TrimSpace(response), "\n")
	var cleanQueries []string
	for _, query := range queries {
		query = strings.TrimSpace(query)
		if query != "" && !strings.HasPrefix(query, "#") {
			cleanQueries = append(cleanQueries, query)
		}
	}

	return cleanQueries, nil
}

// executeSearch performs a search using the configured search provider
func (r *ResearchServiceImpl) executeSearch(ctx context.Context, query string) ([]core.ResearchResult, error) {
	config := search.Config{
		MaxResults: 10,
		Language:   "en",
	}

	results, err := r.searchClient.Search(ctx, query, config)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	var researchResults []core.ResearchResult
	for _, result := range results {
		researchResult := core.ResearchResult{
			ID:        uuid.New().String(),
			Title:     result.Title,
			URL:       result.URL,
			Snippet:   result.Snippet,
			Source:    result.Source,
			Relevance: 0.5, // Will be calculated later
			DateFound: time.Now().UTC(),
			Keywords:  r.extractKeywords(result.Snippet),
		}
		researchResults = append(researchResults, researchResult)
	}

	return researchResults, nil
}

// rankResults scores and ranks research results by relevance
func (r *ResearchServiceImpl) rankResults(results []core.ResearchResult, originalQuery string) []core.ResearchResult {
	queryWords := strings.Fields(strings.ToLower(originalQuery))

	for i := range results {
		score := r.calculateRelevanceScore(results[i], queryWords)
		results[i].Relevance = score
	}

	// Sort by relevance (simple bubble sort for small datasets)
	for i := 0; i < len(results)-1; i++ {
		for j := 0; j < len(results)-i-1; j++ {
			if results[j].Relevance < results[j+1].Relevance {
				results[j], results[j+1] = results[j+1], results[j]
			}
		}
	}

	return results
}

// calculateRelevanceScore calculates a relevance score for a research result
func (r *ResearchServiceImpl) calculateRelevanceScore(result core.ResearchResult, queryWords []string) float64 {
	var score float64
	titleLower := strings.ToLower(result.Title)
	snippetLower := strings.ToLower(result.Snippet)

	// Score based on title matches (weighted higher)
	for _, word := range queryWords {
		if strings.Contains(titleLower, word) {
			score += 0.3
		}
	}

	// Score based on snippet matches
	for _, word := range queryWords {
		if strings.Contains(snippetLower, word) {
			score += 0.1
		}
	}

	// Bonus for quality domains
	if r.isQualityDomain(result.URL) {
		score += 0.2
	}

	// Normalize to 0-1 range
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// isQualityDomain checks if a URL is from a quality domain
func (r *ResearchServiceImpl) isQualityDomain(url string) bool {
	qualityDomains := []string{
		"github.com", "arxiv.org", "doi.org", "pubmed.ncbi.nlm.nih.gov",
		"stackoverflow.com", "medium.com", "nature.com", "ieee.org",
		"acm.org", "nytimes.com", "bbc.com", "reuters.com",
	}

	urlLower := strings.ToLower(url)
	for _, domain := range qualityDomains {
		if strings.Contains(urlLower, domain) {
			return true
		}
	}
	return false
}

// generateSummary creates a comprehensive summary of research results
func (r *ResearchServiceImpl) generateSummary(ctx context.Context, originalQuery string, results []core.ResearchResult) (string, error) {
	if len(results) == 0 {
		return "No research results found for the given query.", nil
	}

	var content strings.Builder
	content.WriteString("Research Results:\n\n")

	// Include top 10 results in summary generation
	maxResults := 10
	if len(results) < maxResults {
		maxResults = len(results)
	}

	for i := 0; i < maxResults; i++ {
		result := results[i]
		content.WriteString(fmt.Sprintf("%d. %s\n   %s\n   URL: %s\n\n",
			i+1, result.Title, result.Snippet, result.URL))
	}

	prompt := fmt.Sprintf(`Based on the research results below about "%s", create a comprehensive research summary:

%s

Create a summary that includes:
1. Overview of the topic based on findings
2. Key insights and trends identified
3. Notable resources or sources found
4. Recommended follow-up actions or areas for deeper research

Format as a well-structured markdown summary (400-600 words).`, originalQuery, content.String())

	response, err := r.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		MaxTokens:   800,
		Temperature: 0.6,
		Model:       "gemini-1.5-flash",
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	return strings.TrimSpace(response), nil
}

// calculateOverallRelevance calculates the overall relevance score for a set of results
func (r *ResearchServiceImpl) calculateOverallRelevance(results []core.ResearchResult) float64 {
	if len(results) == 0 {
		return 0.0
	}

	var total float64
	for _, result := range results {
		total += result.Relevance
	}

	return total / float64(len(results))
}

// extractKeywords extracts keywords from text
func (r *ResearchServiceImpl) extractKeywords(text string) []string {
	words := strings.Fields(strings.ToLower(text))
	keywordMap := make(map[string]bool)

	// Simple keyword extraction - filter out common words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "have": true, "has": true, "had": true, "do": true,
		"does": true, "did": true, "will": true, "would": true, "could": true, "should": true,
	}

	for _, word := range words {
		word = strings.Trim(word, ".,!?;:\"'()[]{}") // Remove punctuation
		if len(word) > 3 && !stopWords[word] {
			keywordMap[word] = true
		}
	}

	var keywords []string
	for keyword := range keywordMap {
		keywords = append(keywords, keyword)
	}

	// Limit to top 5 keywords
	if len(keywords) > 5 {
		keywords = keywords[:5]
	}

	return keywords
}

// truncateContent truncates content to a specified length
func (r *ResearchServiceImpl) truncateContent(content string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}

	truncated := content[:maxLength]
	// Try to end at a word boundary
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > maxLength-50 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}
