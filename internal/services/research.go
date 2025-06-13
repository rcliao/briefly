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

// PerformResearch conducts comprehensive research on a given topic with enhanced v2 features
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

	// Score and rank initial results using research v2 scoring
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

	// Phase 3: Research V2 enhancements - Clustering and Insights
	var clusteringResult *ClusteringResult
	var insights *ActionableInsights

	if depth >= 2 && len(rankedResults) > 0 {
		// Cluster results into meaningful categories
		clusterer := NewResultClusterer(r.llmClient)
		clusteringResult, err = clusterer.ClusterResults(ctx, query, rankedResults)
		if err != nil {
			fmt.Printf("Warning: failed to cluster results: %v\n", err)
		}

		// Generate actionable insights (for depth 3+)
		if depth >= 3 && clusteringResult != nil {
			insightsSynthesizer := NewInsightsSynthesizer(r.llmClient)
			insights, err = insightsSynthesizer.SynthesizeInsights(ctx, query, clusteringResult)
			if err != nil {
				fmt.Printf("Warning: failed to generate insights: %v\n", err)
			}
		}
	}

	// Generate enhanced summary using clustering information
	summary, err := r.generateEnhancedSummary(ctx, query, rankedResults, clusteringResult, insights)
	if err != nil {
		return nil, fmt.Errorf("failed to generate research summary: %w", err)
	}

	// Create enhanced research report
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

Examples of effective queries (use individual keywords, not exact phrases):
- %s overview 2024
- %s current trends
- %s use cases examples
- %s best practices
- %s tutorial guide
- %s features benefits

Focus on using individual keywords and avoid wrapping queries in quotes.
Format: Return only the search queries, one per line.`, baseQueries, topic, topic, topic, topic, topic, topic, topic)
}

// buildCompetitiveAnalysisPrompt creates prompts for competitive analysis
func (r *ResearchServiceImpl) buildCompetitiveAnalysisPrompt(topic string, depth int) string {
	queryCount := 3
	if depth >= 4 {
		queryCount = 4
	}

	return fmt.Sprintf(`Generate %d targeted search queries for competitive analysis of: %s

Create comparison-focused queries covering:
1. Direct competitor comparisons and market positioning
2. Feature gap analysis and capability comparisons  
3. Pricing strategies and value proposition analysis
4. User sentiment, adoption patterns, and community feedback

Enhanced query patterns for competitive intelligence (use keywords, not exact phrases):
- %s vs competitors performance benchmarks
- %s alternatives comparison 2024
- %s market share analysis
- %s user complaints limitations
- %s pricing strategy comparison
- %s competitive advantages disadvantages
- %s adoption rates enterprise
- %s feature matrix comparison
- %s reviews reddit
- %s vs GitHub Copilot

Focus on finding quantitative comparisons, user testimonials, and market analysis reports.
Prioritize recent content (2023-2024) for current competitive landscape.
Use individual keywords and avoid wrapping queries in quotes.

Format: Return only the search queries, one per line.`, queryCount, topic, topic, topic, topic, topic, topic, topic, topic, topic, topic, topic)
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

	return fmt.Sprintf(`Generate %d search queries for technical deep-dive analysis of: %s

Create technical-focused queries covering:
1. Architecture design patterns and implementation details
2. Performance benchmarks, scalability analysis, and optimization
3. API documentation, integration patterns, and developer experience
4. Security architecture, vulnerabilities, and compliance considerations
5. Technical limitations, challenges, and known issues

Advanced technical query patterns (use keywords, not exact phrases):
- %s architecture design patterns
- %s performance benchmarks scalability  
- %s API documentation examples
- %s security vulnerabilities CVE
- %s technical implementation details
- %s scalability limits bottlenecks
- %s integration patterns SDK
- %s code examples github
- %s technical specifications
- %s source code repository
- %s installation setup guide

Focus on finding official documentation, technical papers, code repositories, and detailed implementation guides.
Prioritize authoritative sources: official docs, academic papers, GitHub repos, technical blogs.
Use individual keywords and avoid wrapping queries in quotes.

Format: Return only the search queries, one per line.`, queryCount, topic, topic, topic, topic, topic, topic, topic, topic, topic, topic, topic, topic)
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
			// Remove any remaining quotes that might have been added
			query = strings.Trim(query, "\"'")

			// Break complex queries into keyword-based variants
			expandedQueries := r.expandQueryToKeywords(query)
			cleanQueries = append(cleanQueries, expandedQueries...)
		}
	}

	return cleanQueries
}

// expandQueryToKeywords breaks complex queries into individual keyword combinations for better search results
func (r *ResearchServiceImpl) expandQueryToKeywords(query string) []string {
	// If query is already short and simple, return as-is
	if len(strings.Fields(query)) <= 3 {
		return []string{query}
	}

	// For complex queries, create multiple keyword-based variants
	words := strings.Fields(query)
	var expanded []string

	// Add original query (cleaned)
	expanded = append(expanded, query)

	// Create keyword combinations
	if len(words) >= 4 {
		// Take first 3 words
		if len(words) >= 3 {
			expanded = append(expanded, strings.Join(words[:3], " "))
		}

		// Take key terms (skip common words)
		keyWords := r.extractKeyTerms(words)
		if len(keyWords) >= 2 && len(keyWords) != len(words) {
			expanded = append(expanded, strings.Join(keyWords, " "))
		}

		// For very long queries, create a simplified version
		if len(words) >= 6 {
			// Take every other significant word
			var simplified []string
			for i, word := range words {
				if i == 0 || i%2 == 0 || r.isSignificantTerm(word) {
					simplified = append(simplified, word)
				}
			}
			if len(simplified) >= 2 && len(simplified) < len(words) {
				expanded = append(expanded, strings.Join(simplified, " "))
			}
		}
	}

	// Remove duplicates and return
	return r.removeDuplicateQueries(expanded)
}

// extractKeyTerms filters out common words and keeps significant terms
func (r *ResearchServiceImpl) extractKeyTerms(words []string) []string {
	commonWords := map[string]bool{
		"and": true, "or": true, "the": true, "a": true, "an": true, "in": true, "on": true, "at": true,
		"to": true, "for": true, "of": true, "with": true, "by": true, "from": true, "as": true, "is": true,
		"are": true, "was": true, "were": true, "be": true, "been": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true, "could": true, "should": true,
		"that": true, "this": true, "these": true, "those": true, "vs": true, "comparison": true,
	}

	var keyTerms []string
	for _, word := range words {
		word = strings.ToLower(strings.Trim(word, ".,!?;:"))
		if len(word) > 2 && !commonWords[word] {
			keyTerms = append(keyTerms, word)
		}
	}

	return keyTerms
}

// isSignificantTerm determines if a word is likely to be important for search
func (r *ResearchServiceImpl) isSignificantTerm(word string) bool {
	word = strings.ToLower(word)

	// Technical terms and important keywords
	significantTerms := []string{
		"ai", "api", "cli", "code", "tool", "framework", "library", "sdk", "github", "performance",
		"benchmark", "security", "architecture", "documentation", "integration", "features", "pricing",
		"alternative", "comparison", "review", "tutorial", "guide", "2024", "2023", "trends",
	}

	for _, term := range significantTerms {
		if strings.Contains(word, term) {
			return true
		}
	}

	// Words with numbers or technical patterns
	if len(word) > 3 && (strings.ContainsAny(word, "0123456789") || strings.Contains(word, "-")) {
		return true
	}

	return false
}

// removeDuplicateQueries removes duplicate queries from the list
func (r *ResearchServiceImpl) removeDuplicateQueries(queries []string) []string {
	seen := make(map[string]bool)
	var unique []string

	for _, query := range queries {
		normalized := strings.ToLower(strings.TrimSpace(query))
		if !seen[normalized] && len(normalized) > 0 {
			seen[normalized] = true
			unique = append(unique, query)
		}
	}

	return unique
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

	prompt := fmt.Sprintf(`Based on the initial research results for %s, generate %d refined search queries to fill information gaps and explore deeper insights.

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

// calculateRelevanceScore calculates a relevance score for a research result using research v2 scoring
func (r *ResearchServiceImpl) calculateRelevanceScore(result core.ResearchResult, queryWords []string) float64 {
	// Use research v2 scoring weights
	weights := r.getResearchV2Weights()

	var score float64
	titleLower := strings.ToLower(result.Title)
	snippetLower := strings.ToLower(result.Snippet)

	// Content relevance (30%)
	contentScore := r.calculateContentRelevance(titleLower, snippetLower, queryWords)
	score += contentScore * weights.ContentRelevance

	// Title relevance (15%)
	titleScore := r.calculateTitleRelevance(titleLower, queryWords)
	score += titleScore * weights.TitleRelevance

	// Authority score (20%)
	authorityScore := r.calculateAuthorityScore(result.URL)
	score += authorityScore * weights.Authority

	// Recency score (15%) - boost recent content
	recencyScore := r.calculateRecencyScore(result.DateFound)
	score += recencyScore * weights.Recency

	// Quality score (20%) - technical depth + competitive value
	qualityScore := r.calculateQualityScore(titleLower, snippetLower)
	score += qualityScore * weights.Quality

	// Normalize to 0-1 range
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// getResearchV2Weights returns research v2 scoring weights
func (r *ResearchServiceImpl) getResearchV2Weights() struct {
	ContentRelevance float64
	TitleRelevance   float64
	Authority        float64
	Recency          float64
	Quality          float64
} {
	return struct {
		ContentRelevance float64
		TitleRelevance   float64
		Authority        float64
		Recency          float64
		Quality          float64
	}{
		ContentRelevance: 0.30,
		TitleRelevance:   0.15,
		Authority:        0.20,
		Recency:          0.15,
		Quality:          0.20,
	}
}

// calculateContentRelevance calculates content relevance score
func (r *ResearchServiceImpl) calculateContentRelevance(title, snippet string, queryWords []string) float64 {
	var matches int
	totalWords := len(queryWords)

	if totalWords == 0 {
		return 0.0
	}

	for _, word := range queryWords {
		if strings.Contains(title, word) || strings.Contains(snippet, word) {
			matches++
		}
	}

	return float64(matches) / float64(totalWords)
}

// calculateTitleRelevance calculates title-specific relevance
func (r *ResearchServiceImpl) calculateTitleRelevance(title string, queryWords []string) float64 {
	var matches int
	totalWords := len(queryWords)

	if totalWords == 0 {
		return 0.0
	}

	for _, word := range queryWords {
		if strings.Contains(title, word) {
			matches++
		}
	}

	// Bonus for exact phrase matches in title
	titleWords := strings.Fields(title)
	if len(titleWords) > 0 && len(queryWords) > 1 {
		queryPhrase := strings.Join(queryWords, " ")
		if strings.Contains(title, queryPhrase) {
			matches += len(queryWords) // Bonus for phrase match
		}
	}

	score := float64(matches) / float64(totalWords)
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// calculateAuthorityScore calculates source authority score with tier-based weighting
func (r *ResearchServiceImpl) calculateAuthorityScore(url string) float64 {
	// Tier 1 (Weight: 1.0): Official documentation, academic papers, technical specifications
	tier1Domains := []string{
		"github.com", "docs.github.com", "arxiv.org", "doi.org", "pubmed.ncbi.nlm.nih.gov",
		"ieee.org", "acm.org", "openai.com", "anthropic.com", "google.ai",
		"microsoft.com", "azure.microsoft.com", "aws.amazon.com", "cloud.google.com",
		"tensorflow.org", "pytorch.org", "huggingface.co", "papers.nips.cc",
		"kubernetes.io", "docker.com", "golang.org", "python.org", "nodejs.org",
	}

	// Tier 2 (Weight: 0.8): Established tech publications, industry reports
	tier2Domains := []string{
		"stackoverflow.com", "medium.com", "dev.to", "hackernews.ycombinator.com",
		"techcrunch.com", "wired.com", "arstechnica.com", "theverge.com",
		"venturebeat.com", "zdnet.com", "infoworld.com", "computerworld.com",
		"towardsdatascience.com", "kdnuggets.com", "machinelearningmastery.com",
	}

	// Tier 3 (Weight: 0.6): Developer blogs, conference presentations
	tier3Domains := []string{
		"blog.", "blogs.", ".blog", ".dev", "engineering.", "tech.",
		"conference", "summit", "meetup", "presentation", "slides",
	}

	// Tier 4 (Weight: 0.4): Community forums, social media discussions
	tier4Domains := []string{
		"reddit.com", "discord.com", "slack.com", "twitter.com", "x.com",
		"facebook.com", "linkedin.com", "youtube.com", "forum",
	}

	urlLower := strings.ToLower(url)

	// Check tier 1 domains
	for _, domain := range tier1Domains {
		if strings.Contains(urlLower, domain) {
			return 1.0
		}
	}

	// Check tier 2 domains
	for _, domain := range tier2Domains {
		if strings.Contains(urlLower, domain) {
			return 0.8
		}
	}

	// Check tier 3 domains
	for _, domain := range tier3Domains {
		if strings.Contains(urlLower, domain) {
			return 0.6
		}
	}

	// Check tier 4 domains
	for _, domain := range tier4Domains {
		if strings.Contains(urlLower, domain) {
			return 0.4
		}
	}

	// Default score for unknown domains
	return 0.5
}

// calculateRecencyScore calculates recency score with 6-month boost
func (r *ResearchServiceImpl) calculateRecencyScore(dateFound time.Time) float64 {
	now := time.Now()
	daysSince := int(now.Sub(dateFound).Hours() / 24)

	// Boost content from last 6 months (180 days)
	if daysSince <= 180 {
		return 1.0 - (float64(daysSince)/180.0)*0.5 // Score: 1.0 to 0.5
	}

	// Older content gets lower score
	if daysSince <= 365 {
		return 0.5 - (float64(daysSince-180)/185.0)*0.3 // Score: 0.5 to 0.2
	}

	// Very old content gets minimum score
	return 0.2
}

// calculateQualityScore calculates quality score based on technical depth and competitive value
func (r *ResearchServiceImpl) calculateQualityScore(title, snippet string) float64 {
	score := 0.0
	text := title + " " + snippet
	textLower := strings.ToLower(text)

	// Technical depth indicators
	technicalTerms := []string{
		"api", "architecture", "implementation", "performance", "benchmark",
		"scalability", "algorithm", "framework", "library", "sdk", "code",
		"technical", "engineering", "development", "deployment", "infrastructure",
		"security", "optimization", "integration", "documentation", "specification",
	}

	// Competitive value indicators
	competitiveTerms := []string{
		"vs", "versus", "comparison", "compare", "alternative", "competitor",
		"market", "analysis", "review", "evaluation", "benchmark", "pros", "cons",
		"advantages", "disadvantages", "features", "pricing", "cost", "adoption",
	}

	// Score technical depth
	technicalMatches := 0
	for _, term := range technicalTerms {
		if strings.Contains(textLower, term) {
			technicalMatches++
		}
	}
	score += float64(technicalMatches) / float64(len(technicalTerms)) * 0.6

	// Score competitive value
	competitiveMatches := 0
	for _, term := range competitiveTerms {
		if strings.Contains(textLower, term) {
			competitiveMatches++
		}
	}
	score += float64(competitiveMatches) / float64(len(competitiveTerms)) * 0.4

	// Bonus for code examples or detailed technical content
	if strings.Contains(textLower, "github") || strings.Contains(textLower, "example") ||
		strings.Contains(textLower, "tutorial") || strings.Contains(textLower, "guide") {
		score += 0.1
	}

	// Normalize to 0-1 range
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// generateEnhancedSummary creates a comprehensive summary with clustering and insights
func (r *ResearchServiceImpl) generateEnhancedSummary(ctx context.Context, originalQuery string, results []core.ResearchResult, clusteringResult *ClusteringResult, insights *ActionableInsights) (string, error) {
	if len(results) == 0 {
		return "No research results found for the given query.", nil
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("Research Query: %s\n", originalQuery))
	content.WriteString(fmt.Sprintf("Total Results: %d\n", len(results)))

	if clusteringResult != nil {
		content.WriteString(fmt.Sprintf("Results Organized into %d Categories\n", len(clusteringResult.Categories)))
		content.WriteString(fmt.Sprintf("Overall Quality Score: %.2f\n", clusteringResult.OverallQuality))

		content.WriteString("\nCategorized Research Findings:\n")
		for _, category := range clusteringResult.Categories {
			if len(category.Results) > 0 {
				content.WriteString(fmt.Sprintf("\n%s (%d results, quality: %.2f):\n",
					category.Name, len(category.Results), category.Quality))

				// Include top 2 results from each category
				maxPerCategory := 2
				if len(category.Results) < maxPerCategory {
					maxPerCategory = len(category.Results)
				}

				for i := 0; i < maxPerCategory; i++ {
					result := category.Results[i]
					content.WriteString(fmt.Sprintf("- %s\n", result.Title))
				}
			}
		}

		if len(clusteringResult.CoverageGaps) > 0 {
			content.WriteString(fmt.Sprintf("\nCoverage Gaps: %s\n", strings.Join(clusteringResult.CoverageGaps, ", ")))
		}
	} else {
		// Fallback to top results if no clustering
		content.WriteString("\nTop Research Results:\n")
		maxResults := 8
		if len(results) < maxResults {
			maxResults = len(results)
		}

		for i := 0; i < maxResults; i++ {
			result := results[i]
			content.WriteString(fmt.Sprintf("%d. %s\n   %s\n", i+1, result.Title, result.Snippet))
		}
	}

	var promptBuilder strings.Builder
	promptBuilder.WriteString(fmt.Sprintf(`Based on the comprehensive research about %s with the following findings:

%s

Create an enhanced research summary that includes:

1. **Executive Overview**: High-level summary of what this technology/topic is and why it matters

2. **Key Findings**: Most important insights discovered across all research categories

3. **Research Highlights**: Notable patterns, trends, or standout information found

4. **Information Quality Assessment**: Confidence level in findings and data gaps identified

`, originalQuery, content.String()))

	if insights != nil {
		promptBuilder.WriteString(fmt.Sprintf(`
5. **Strategic Insights**: Based on actionable insights analysis:
   - Competitive positioning and market context
   - Technical feasibility and implementation considerations  
   - Strategic recommendations for decision-making

Confidence Level: %.2f
`, insights.ConfidenceLevel))
	} else {
		promptBuilder.WriteString("5. **Next Steps**: Recommended areas for follow-up research or evaluation\n")
	}

	promptBuilder.WriteString("\nFormat as a well-structured markdown summary (500-800 words) suitable for both technical and business stakeholders.")

	response, err := r.llmClient.GenerateText(ctx, promptBuilder.String(), llm.TextGenerationOptions{
		MaxTokens:   1200,
		Temperature: 0.6,
		Model:       "gemini-1.5-flash",
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate enhanced summary: %w", err)
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
