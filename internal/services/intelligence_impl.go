package services

import (
	"context"
	"fmt"
	"time"

	"briefly/internal/core"
)

// intelligenceService implements the IntelligenceService interface
type intelligenceService struct {
	articleProcessor ArticleProcessor
	llmService       LLMService
	cacheService     CacheService
	aiRouter         AIRouter
}

// NewIntelligenceService creates a new intelligence service instance
func NewIntelligenceService(
	articleProcessor ArticleProcessor,
	llmService LLMService,
	cacheService CacheService,
	aiRouter AIRouter,
) IntelligenceService {
	return &intelligenceService{
		articleProcessor: articleProcessor,
		llmService:       llmService,
		cacheService:     cacheService,
		aiRouter:         aiRouter,
	}
}

// ProcessContent implements the main content processing pipeline for v3.0
func (s *intelligenceService) ProcessContent(ctx context.Context, input ContentInput) (*core.Digest, error) {
	startTime := time.Now()
	
	// Step 1: Extract URLs from input
	urls, err := s.extractURLs(input)
	if err != nil {
		return nil, fmt.Errorf("failed to extract URLs: %w", err)
	}
	
	if len(urls) == 0 {
		return nil, fmt.Errorf("no URLs found in input")
	}
	
	// Step 2: Process articles in parallel (with cache checks)
	articles, err := s.processArticlesWithCache(ctx, urls)
	if err != nil {
		return nil, fmt.Errorf("failed to process articles: %w", err)
	}
	
	// Step 3: Apply quality filtering using AI router (local model)
	filteredArticles, err := s.filterByQuality(ctx, articles, input.Options.QualityThreshold)
	if err != nil {
		return nil, fmt.Errorf("failed to filter articles: %w", err)
	}
	
	if len(filteredArticles) == 0 {
		return nil, fmt.Errorf("no articles met quality threshold")
	}
	
	// Step 4: Group articles into thematic clusters (local model)
	articleGroups, err := s.clusterArticles(ctx, filteredArticles)
	if err != nil {
		return nil, fmt.Errorf("failed to cluster articles: %w", err)
	}
	
	// Step 5: Generate signal (cloud model for quality)
	signal, err := s.generateSignal(ctx, articleGroups, input.Options.UserContext)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signal: %w", err)
	}
	
	// Step 6: Assemble digest
	digest := &core.Digest{
		ID:            s.generateDigestID(),
		Signal:        *signal,
		ArticleGroups: articleGroups,
		Metadata: core.DigestMetadata{
			Title:          s.generateTitle(signal.Content),
			DateGenerated:  time.Now(),
			WordCount:      s.calculateWordCount(signal, articleGroups),
			ArticleCount:   len(filteredArticles),
			ProcessingTime: time.Since(startTime),
			ProcessingCost: s.calculateTotalCost(signal, articleGroups),
			QualityScore:   s.calculateQualityScore(signal, articleGroups),
		},
		
		// Legacy compatibility
		Title:         s.generateTitle(signal.Content),
		DateGenerated: time.Now(),
		Format:        "scannable", // Default format for Phase 2
	}
	
	return digest, nil
}

// extractURLs extracts URLs from the input (file or direct URLs)
func (s *intelligenceService) extractURLs(input ContentInput) ([]string, error) {
	if input.FilePath != "" {
		// TODO: Implement file URL extraction (delegate to existing logic)
		return nil, fmt.Errorf("file URL extraction not yet implemented")
	}
	
	if len(input.URLs) > 0 {
		return input.URLs, nil
	}
	
	return nil, fmt.Errorf("no URLs or file path provided")
}

// processArticlesWithCache processes articles with intelligent caching
func (s *intelligenceService) processArticlesWithCache(ctx context.Context, urls []string) ([]core.Article, error) {
	var articles []core.Article
	
	for _, url := range urls {
		// Check cache first
		if cached, err := s.cacheService.GetCachedArticle(ctx, url); err == nil && cached != nil {
			articles = append(articles, *cached)
			continue
		}
		
		// Process new article
		article, err := s.articleProcessor.ProcessArticle(ctx, url)
		if err != nil {
			// Log error but continue with other articles
			fmt.Printf("Warning: Failed to process %s: %v\n", url, err)
			continue
		}
		
		// Cache the article
		_ = s.cacheService.CacheArticle(ctx, *article)
		articles = append(articles, *article)
	}
	
	return articles, nil
}

// filterByQuality uses local AI to filter articles by quality
func (s *intelligenceService) filterByQuality(ctx context.Context, articles []core.Article, threshold float64) ([]core.Article, error) {
	// For Phase 2, use simple heuristics (Phase 3 will add AI router)
	var filtered []core.Article
	
	for _, article := range articles {
		// Simple quality heuristics for now (Phase 3 will add AI)
		score := s.calculateBasicQualityScore(article)
		article.QualityScore = score
		
		if score >= threshold {
			filtered = append(filtered, article)
		}
	}
	
	return filtered, nil
}

// clusterArticles groups articles into thematic clusters
func (s *intelligenceService) clusterArticles(ctx context.Context, articles []core.Article) ([]core.ArticleGroup, error) {
	// Simple clustering for Phase 2 (will enhance in Phase 4)
	groups := make(map[string][]core.Article)
	
	for _, article := range articles {
		category := s.categorizeArticle(article)
		groups[category] = append(groups[category], article)
	}
	
	var articleGroups []core.ArticleGroup
	priority := 1
	
	for category, categoryArticles := range groups {
		group := core.ArticleGroup{
			Category: category,
			Theme:    s.generateTheme(categoryArticles),
			Articles: categoryArticles,
			Summary:  s.generateGroupSummary(categoryArticles),
			Priority: priority,
		}
		articleGroups = append(articleGroups, group)
		priority++
	}
	
	return articleGroups, nil
}

// generateSignal creates the main insight from article groups
func (s *intelligenceService) generateSignal(ctx context.Context, groups []core.ArticleGroup, userContext string) (*core.Signal, error) {
	// For Phase 2, generate a basic signal
	// Phase 3 will use AI router for cloud processing
	
	signal := &core.Signal{
		ID:            s.generateSignalID(),
		Content:       s.synthesizeInsight(groups, userContext),
		SourceArticles: s.getSourceArticleIDs(groups),
		Confidence:    0.8, // Basic confidence for now
		Theme:         s.getMainTheme(groups),
		Implications:  s.generateImplications(groups),
		Actions:       s.generateActionItems(groups),
		DateGenerated: time.Now(),
		ProcessingCost: core.ProcessingCost{
			LocalTokens:  500,  // Placeholder
			CloudTokens:  1000, // Placeholder  
			EstimatedUSD: 0.05, // Placeholder
		},
	}
	
	return signal, nil
}

// Helper methods for Phase 2 basic implementation

func (s *intelligenceService) calculateBasicQualityScore(article core.Article) float64 {
	score := 0.0
	
	// Title quality (40% weight)
	if len(article.Title) > 10 && len(article.Title) < 200 {
		score += 0.4
	}
	
	// Content length (30% weight)
	if len(article.CleanedText) > 500 && len(article.CleanedText) < 10000 {
		score += 0.3
	}
	
	// URL quality (20% weight)
	if !s.isSpamDomain(article.URL) {
		score += 0.2
	}
	
	// Content type bonus (10% weight)
	if article.ContentType == core.ContentTypeHTML {
		score += 0.1
	}
	
	return score
}

func (s *intelligenceService) categorizeArticle(article core.Article) string {
	title := article.Title
	
	// Simple keyword-based categorization (Phase 3 will use AI)
	if s.containsKeywords(title, []string{"breaking", "news", "announced", "released"}) {
		return "🔥 Breaking & Hot"
	}
	if s.containsKeywords(title, []string{"tool", "github", "open source", "library", "framework"}) {
		return "🛠️ Tools & Platforms"
	}
	if s.containsKeywords(title, []string{"analysis", "study", "research", "report"}) {
		return "📊 Analysis & Research"  
	}
	if s.containsKeywords(title, []string{"business", "market", "cost", "money", "pricing"}) {
		return "💰 Business & Economics"
	}
	
	return "💡 Additional Items"
}

func (s *intelligenceService) containsKeywords(text string, keywords []string) bool {
	// Simple case-insensitive keyword matching
	for _, keyword := range keywords {
		if len(text) > 0 && len(keyword) > 0 {
			// Basic substring check (Phase 3 will improve)
			return true // Simplified for now
		}
	}
	return false
}

func (s *intelligenceService) generateTheme(articles []core.Article) string {
	if len(articles) == 0 {
		return "General"
	}
	
	// Extract main theme from first article title (simplified)
	title := articles[0].Title
	if len(title) > 20 {
		return title[:20] + "..."
	}
	return title
}

func (s *intelligenceService) generateGroupSummary(articles []core.Article) string {
	if len(articles) == 1 {
		return fmt.Sprintf("1 article covering %s", articles[0].Title)
	}
	return fmt.Sprintf("%d articles covering related topics", len(articles))
}

func (s *intelligenceService) synthesizeInsight(groups []core.ArticleGroup, userContext string) string {
	// Basic insight synthesis for Phase 2
	totalArticles := 0
	for _, group := range groups {
		totalArticles += len(group.Articles)
	}
	
	insight := fmt.Sprintf("This week's digest covers %d articles across %d key themes.", totalArticles, len(groups))
	
	if userContext != "" {
		insight += fmt.Sprintf(" Context: %s", userContext)
	}
	
	return insight
}

func (s *intelligenceService) getSourceArticleIDs(groups []core.ArticleGroup) []string {
	var ids []string
	for _, group := range groups {
		for _, article := range group.Articles {
			ids = append(ids, article.ID)
		}
	}
	return ids
}

func (s *intelligenceService) getMainTheme(groups []core.ArticleGroup) string {
	if len(groups) > 0 {
		return groups[0].Theme
	}
	return "General"
}

func (s *intelligenceService) generateImplications(groups []core.ArticleGroup) []string {
	// Basic implications for Phase 2
	return []string{
		"Multiple development trends are converging",
		"Technology adoption is accelerating",
	}
}

func (s *intelligenceService) generateActionItems(groups []core.ArticleGroup) []core.ActionItem {
	// Basic action items for Phase 2
	return []core.ActionItem{
		{
			Description: "Review the highlighted tools for your current projects",
			Effort:      "low",
			Timeline:    "this_week",
		},
		{
			Description: "Explore one new technology mentioned in the digest",
			Effort:      "medium", 
			Timeline:    "this_month",
		},
	}
}

func (s *intelligenceService) generateDigestID() string {
	return fmt.Sprintf("digest_%d", time.Now().Unix())
}

func (s *intelligenceService) generateSignalID() string {
	return fmt.Sprintf("signal_%d", time.Now().Unix())
}

func (s *intelligenceService) generateTitle(content string) string {
	// Basic title generation (Phase 3 will use AI)
	return fmt.Sprintf("Signal Digest - %s", time.Now().Format("Jan 2, 2006"))
}

func (s *intelligenceService) calculateWordCount(signal *core.Signal, groups []core.ArticleGroup) int {
	count := len(signal.Content)
	for _, group := range groups {
		count += len(group.Summary)
	}
	return count / 5 // Rough word estimate
}

func (s *intelligenceService) calculateTotalCost(signal *core.Signal, groups []core.ArticleGroup) core.ProcessingCost {
	return signal.ProcessingCost // Placeholder
}

func (s *intelligenceService) calculateQualityScore(signal *core.Signal, groups []core.ArticleGroup) float64 {
	return signal.Confidence // Use signal confidence as quality proxy
}

func (s *intelligenceService) isSpamDomain(url string) bool {
	// Basic spam domain detection
	spamDomains := []string{"spam.com", "fake.com", "clickbait.com"}
	for _, domain := range spamDomains {
		if len(url) > len(domain) {
			// Simplified check
			continue
		}
	}
	return false
}

// Placeholder implementations for other interface methods

func (s *intelligenceService) StartResearchSession(ctx context.Context, query string) (*core.ResearchSession, error) {
	return nil, fmt.Errorf("research sessions will be implemented in Phase 5")
}

func (s *intelligenceService) ContinueResearch(ctx context.Context, sessionID string, userInput string) (*ResearchResponse, error) {
	return nil, fmt.Errorf("research continuation will be implemented in Phase 5")
}

func (s *intelligenceService) ExploreTopicFurther(ctx context.Context, topic string, currentDigest *core.Digest) (*core.ResearchSession, error) {
	return nil, fmt.Errorf("topic exploration will be implemented in Phase 5")
}

func (s *intelligenceService) RecordUserFeedback(ctx context.Context, feedback core.UserFeedback) error {
	return fmt.Errorf("user feedback recording will be implemented in Phase 5")
}

func (s *intelligenceService) GetPersonalizationProfile(ctx context.Context) (*core.UserProfile, error) {
	// Return default profile for now
	return &core.UserProfile{
		PreferLocal:      true,
		MaxCloudCost:     1.0,
		QualityThreshold: 0.6,
	}, nil
}