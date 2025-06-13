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
	var allQueries []string
	var allResults []core.ResearchResult

	// Phase 1: Initial query generation
	initialQueries, err := r.generateSearchQueries(ctx, query, depth)
	if err != nil {
		return nil, fmt.Errorf("failed to generate initial queries: %w", err)
	}
	allQueries = append(allQueries, initialQueries...)

	// Execute initial searches
	for _, searchQuery := range initialQueries {
		results, err := r.executeSearch(ctx, searchQuery)
		if err != nil {
			// Log error but continue with other queries
			continue
		}
		allResults = append(allResults, results...)
	}

	// Score and rank initial results
	rankedResults := r.rankResults(allResults, query)

	// Phase 2: Iterative refinement (for depth 3+)
	if depth >= 3 && len(rankedResults) > 0 {
		refinedQueries, err := r.generateRefinedQueries(ctx, query, rankedResults, depth)
		if err != nil {
			// Log error but continue
			fmt.Printf("Warning: failed to generate refined queries: %v\n", err)
		} else {
			allQueries = append(allQueries, refinedQueries...)

			// Execute refined searches
			for _, searchQuery := range refinedQueries {
				results, err := r.executeSearch(ctx, searchQuery)
				if err != nil {
					continue
				}
				allResults = append(allResults, results...)
			}

			// Re-rank all results
			rankedResults = r.rankResults(allResults, query)
		}
	}

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
		GeneratedQueries: allQueries,
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

// ResearchIntent defines the type of research being conducted
type ResearchIntent string

const (
	IntentGeneral     ResearchIntent = "general"
	IntentCompetitive ResearchIntent = "competitive"
	IntentTechnical   ResearchIntent = "technical"
)

// generateSearchQueries creates search queries using LLM based on the topic and depth
func (r *ResearchServiceImpl) generateSearchQueries(ctx context.Context, topic string, depth int) ([]string, error) {
	// Generate queries for different research intents
	var allQueries []string

	// General research queries (always included)
	generalQueries, err := r.generateQueriesForIntent(ctx, topic, depth, IntentGeneral)
	if err != nil {
		return nil, fmt.Errorf("failed to generate general queries: %w", err)
	}
	allQueries = append(allQueries, generalQueries...)

	// Competitive analysis queries (depth 2+)
	if depth >= 2 {
		competitiveQueries, err := r.generateQueriesForIntent(ctx, topic, depth, IntentCompetitive)
		if err != nil {
			// Log error but continue
			fmt.Printf("Warning: failed to generate competitive queries: %v\n", err)
		} else {
			allQueries = append(allQueries, competitiveQueries...)
		}
	}

	// Technical deep-dive queries (depth 3+)
	if depth >= 3 {
		technicalQueries, err := r.generateQueriesForIntent(ctx, topic, depth, IntentTechnical)
		if err != nil {
			// Log error but continue
			fmt.Printf("Warning: failed to generate technical queries: %v\n", err)
		} else {
			allQueries = append(allQueries, technicalQueries...)
		}
	}

	return allQueries, nil
}

// generateQueriesForIntent generates queries for a specific research intent
func (r *ResearchServiceImpl) generateQueriesForIntent(ctx context.Context, topic string, depth int, intent ResearchIntent) ([]string, error) {
	var prompt string

	switch intent {
	case IntentGeneral:
		prompt = r.buildGeneralResearchPrompt(topic, depth)
	case IntentCompetitive:
		prompt = r.buildCompetitiveAnalysisPrompt(topic, depth)
	case IntentTechnical:
		prompt = r.buildTechnicalDeepDivePrompt(topic, depth)
	}

	response, err := r.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		MaxTokens:   600,
		Temperature: 0.7,
		Model:       "gemini-1.5-flash",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate %s queries: %w", intent, err)
	}

	return r.parseQueryResponse(response), nil
}

// buildGeneralResearchPrompt creates prompts for general research
func (r *ResearchServiceImpl) buildGeneralResearchPrompt(topic string, depth int) string {
	baseQueries := 3
	if depth >= 2 {
		baseQueries = 4
	}
	if depth >= 3 {
		baseQueries = 5
	}

	return fmt.Sprintf(`Generate %d search queries for general research on: %s

Create targeted search queries covering:
1. Basic overview and definition
2. Current state and trends 2024
3. Key examples and use cases
4. Recent developments and news
5. Best practices and guidelines

Examples of effective queries:
- "%s overview 2024"
- "%s current trends"
- "%s use cases examples"
- "%s best practices"

Format: Return only the search queries, one per line.`, baseQueries, topic, topic, topic, topic, topic)
}

// buildCompetitiveAnalysisPrompt creates prompts for competitive analysis
func (r *ResearchServiceImpl) buildCompetitiveAnalysisPrompt(topic string, depth int) string {
	queryCount := 3
	if depth >= 4 {
		queryCount = 4
	}

	return fmt.Sprintf(`Generate %d search queries for competitive analysis of: %s

Create comparison-focused queries covering:
1. Market positioning and competitors
2. Feature comparisons and gaps
3. Pricing and value proposition
4. User sentiment and feedback

Query patterns to use:
- "%s vs [competitor]"
- "%s alternatives comparison"
- "%s market share analysis"
- "%s user reviews complaints"
- "%s pricing comparison"
- "%s limitations disadvantages"

Format: Return only the search queries, one per line.`, queryCount, topic, topic, topic, topic, topic, topic, topic)
}

// buildTechnicalDeepDivePrompt creates prompts for technical deep-dive research
func (r *ResearchServiceImpl) buildTechnicalDeepDivePrompt(topic string, depth int) string {
	queryCount := 3
	if depth >= 4 {
		queryCount = 4
	}
	if depth >= 5 {
		queryCount = 5
	}

	return fmt.Sprintf(`Generate %d search queries for technical analysis of: %s

Create technical-focused queries covering:
1. Architecture and implementation details
2. Performance benchmarks and scalability
3. Integration examples and API documentation
4. Security considerations and vulnerabilities
5. Technical limitations and challenges

Query patterns to use:
- "%s architecture design"
- "%s performance benchmarks"
- "%s API documentation"
- "%s security vulnerabilities"
- "%s technical implementation"
- "%s scalability limits"
- "%s integration examples"

Format: Return only the search queries, one per line.`, queryCount, topic, topic, topic, topic, topic, topic, topic, topic)
}

// parseQueryResponse parses the LLM response and extracts clean queries
func (r *ResearchServiceImpl) parseQueryResponse(response string) []string {
	queries := strings.Split(strings.TrimSpace(response), "\n")
	var cleanQueries []string

	for _, query := range queries {
		query = strings.TrimSpace(query)
		// Remove numbering, bullets, and other formatting
		query = strings.TrimLeft(query, "0123456789.- ")

		if query != "" && !strings.HasPrefix(query, "#") && len(query) > 5 {
			cleanQueries = append(cleanQueries, query)
		}
	}

	return cleanQueries
}

// generateRefinedQueries creates refined queries based on initial results
func (r *ResearchServiceImpl) generateRefinedQueries(ctx context.Context, originalQuery string, results []core.ResearchResult, depth int) ([]string, error) {
	// Analyze top results to identify key themes and gaps
	topResults := results
	if len(results) > 10 {
		topResults = results[:10] // Use top 10 results for analysis
	}

	// Filter high-relevance results (>0.6) for context learning
	var highRelevanceResults []core.ResearchResult
	for _, result := range topResults {
		if result.Relevance > 0.6 {
			highRelevanceResults = append(highRelevanceResults, result)
		}
	}

	if len(highRelevanceResults) == 0 {
		// If no high-relevance results, use top 5 results
		if len(topResults) > 5 {
			highRelevanceResults = topResults[:5]
		} else {
			highRelevanceResults = topResults
		}
	}

	// Build context from high-relevance results
	var contextBuilder strings.Builder
	contextBuilder.WriteString("High-relevance findings:\n")
	for i, result := range highRelevanceResults {
		contextBuilder.WriteString(fmt.Sprintf("%d. %s\n   %s\n", i+1, result.Title, result.Snippet))
	}

	// Generate refined queries based on gaps and opportunities
	queryCount := 2
	if depth >= 4 {
		queryCount = 3
	}
	if depth >= 5 {
		queryCount = 4
	}

	prompt := fmt.Sprintf(`Based on the initial research results for "%s", generate %d refined search queries to fill information gaps and explore deeper insights.

%s

Analyze the above findings and create targeted queries that:
1. Address any obvious information gaps
2. Explore specific aspects mentioned but not fully covered
3. Find more recent or authoritative sources
4. Investigate related topics that emerged from the results

Query patterns for refinement:
- Focus on specific technical terms or concepts found in results
- Target recent developments (2024) if results seem outdated
- Look for authoritative sources (academic, official documentation)
- Explore alternatives or competing approaches mentioned

Format: Return only the refined search queries, one per line.`, originalQuery, queryCount, contextBuilder.String())

	response, err := r.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		MaxTokens:   400,
		Temperature: 0.6, // Lower temperature for more focused refinement
		Model:       "gemini-1.5-flash",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate refined queries: %w", err)
	}

	return r.parseQueryResponse(response), nil
}

// QueryContext holds context information for query generation
type QueryContext struct {
	OriginalQuery     string
	PreviousQueries   []string
	HighScoreKeywords []string
	IdentifiedGaps    []string
	RelevanceThemes   map[string]float64
}

// buildQueryContext analyzes results to build context for future queries
func (r *ResearchServiceImpl) buildQueryContext(originalQuery string, queries []string, results []core.ResearchResult) *QueryContext {
	context := &QueryContext{
		OriginalQuery:     originalQuery,
		PreviousQueries:   queries,
		HighScoreKeywords: make([]string, 0),
		IdentifiedGaps:    make([]string, 0),
		RelevanceThemes:   make(map[string]float64),
	}

	// Extract keywords from high-relevance results
	keywordFreq := make(map[string]int)
	relevanceSum := make(map[string]float64)

	for _, result := range results {
		if result.Relevance > 0.7 { // High relevance threshold
			for _, keyword := range result.Keywords {
				keywordFreq[keyword]++
				relevanceSum[keyword] += result.Relevance
			}
		}
	}

	// Build high-score keywords list
	for keyword, freq := range keywordFreq {
		if freq >= 2 { // Appeared in at least 2 high-relevance results
			context.HighScoreKeywords = append(context.HighScoreKeywords, keyword)
			context.RelevanceThemes[keyword] = relevanceSum[keyword] / float64(freq)
		}
	}

	return context
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
