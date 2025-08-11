package categorization

import (
	"briefly/internal/core"
	"briefly/internal/llm"
	"briefly/internal/render"
	"context"
	"fmt"
	"strings"
)

// Category represents a content category with emoji and priority
type Category struct {
	ID          string
	Name        string
	Emoji       string
	Priority    int
	Description string
}

// Categories defines the enhanced categories for scannable format
var Categories = map[string]Category{
	"breaking": {
		ID:          "breaking",
		Name:        "Breaking & Hot",
		Emoji:       "🔥",
		Priority:    1,
		Description: "Urgent, trending, and breaking news that requires immediate attention",
	},
	"products": {
		ID:          "products",
		Name:        "Product Updates",
		Emoji:       "🚀",
		Priority:    2,
		Description: "Product launches, releases, major updates, and announcements",
	},
	"aiml": {
		ID:          "aiml",
		Name:        "AI & Machine Learning",
		Emoji:       "🤖",
		Priority:    3,
		Description: "AI models, ML research, LLMs, machine learning techniques and breakthroughs",
	},
	"security": {
		ID:          "security",
		Name:        "Security & Privacy",
		Emoji:       "🔒",
		Priority:    4,
		Description: "Cybersecurity, privacy, vulnerabilities, and security best practices",
	},
	"devtools": {
		ID:          "devtools",
		Name:        "Dev Tools & Techniques",
		Emoji:       "🛠️",
		Priority:    5,
		Description: "Engineering insights, development tools, tutorials, and techniques",
	},
	"infrastructure": {
		ID:          "infrastructure",
		Name:        "Infrastructure & Cloud",
		Emoji:       "☁️",
		Priority:    6,
		Description: "Cloud platforms, DevOps, infrastructure, scaling, and system architecture",
	},
	"research": {
		ID:          "research",
		Name:        "Research & Analysis",
		Emoji:       "📊",
		Priority:    7,
		Description: "Studies, benchmarks, deep dives, and analytical content",
	},
	"inspiration": {
		ID:          "inspiration",
		Name:        "Ideas & Inspiration",
		Emoji:       "💡",
		Priority:    8,
		Description: "Interesting implementations, case studies, and creative projects",
	},
	"monitoring": {
		ID:          "monitoring",
		Name:        "Worth Monitoring",
		Emoji:       "🔍",
		Priority:    9,
		Description: "Emerging trends, experimental projects, and topics worth exploring",
	},
}

// CategoryResult holds categorization results with confidence
type CategoryResult struct {
	Category   Category
	Confidence float64
	Reasoning  string
	Source     string // "rule-based" or "llm-based"
}

// CategorizedItem represents an article with its category assignment
type CategorizedItem struct {
	DigestItem render.DigestData
	Article    core.Article
	Category   CategoryResult
}

// CategorizationService handles article categorization using multiple strategies
type CategorizationService interface {
	CategorizeArticles(ctx context.Context, digestItems []render.DigestData, articles []core.Article) ([]CategorizedItem, error)
	CategorizeArticle(ctx context.Context, digestItem render.DigestData, article core.Article) (CategoryResult, error)
}

// Service implements the CategorizationService interface
type Service struct {
	llmClient LLMClient
}

// LLMClient interface for LLM-based categorization
type LLMClient interface {
	CategorizeArticle(ctx context.Context, article core.Article, categories map[string]llm.Category) (llm.CategoryResult, error)
}

// NewService creates a new categorization service
func NewService(llmClient LLMClient) *Service {
	return &Service{
		llmClient: llmClient,
	}
}

// CategorizeArticles categorizes multiple articles using hybrid approach
func (s *Service) CategorizeArticles(ctx context.Context, digestItems []render.DigestData, articles []core.Article) ([]CategorizedItem, error) {
	if len(digestItems) != len(articles) {
		return nil, fmt.Errorf("digest items and articles length mismatch: %d vs %d", len(digestItems), len(articles))
	}

	var categorizedItems []CategorizedItem

	for i, digestItem := range digestItems {
		article := articles[i]

		// First try rule-based categorization for speed and reliability
		ruleResult := s.categorizeArticleRuleBased(digestItem, article)

		// If rule-based confidence is high enough, use it
		var finalResult CategoryResult
		if ruleResult.Confidence >= 0.8 {
			finalResult = ruleResult
		} else if s.llmClient != nil {
			// Convert our categories to LLM format
			llmCategories := make(map[string]llm.Category)
			for id, cat := range Categories {
				llmCategories[id] = llm.Category{
					ID:          cat.ID,
					Name:        cat.Name,
					Emoji:       cat.Emoji,
					Priority:    cat.Priority,
					Description: cat.Description,
				}
			}

			// Fall back to LLM-based categorization for ambiguous cases
			llmResult, err := s.llmClient.CategorizeArticle(ctx, article, llmCategories)
			if err == nil && llmResult.Confidence > ruleResult.Confidence {
				// Convert LLM result back to our format
				finalResult = CategoryResult{
					Category: Category{
						ID:          llmResult.Category.ID,
						Name:        llmResult.Category.Name,
						Emoji:       llmResult.Category.Emoji,
						Priority:    llmResult.Category.Priority,
						Description: llmResult.Category.Description,
					},
					Confidence: llmResult.Confidence,
					Reasoning:  llmResult.Reasoning,
					Source:     llmResult.Source,
				}
			} else {
				finalResult = ruleResult
			}
		} else {
			finalResult = ruleResult
		}

		categorizedItems = append(categorizedItems, CategorizedItem{
			DigestItem: digestItem,
			Article:    article,
			Category:   finalResult,
		})
	}

	return categorizedItems, nil
}

// CategorizeArticle categorizes a single article
func (s *Service) CategorizeArticle(ctx context.Context, digestItem render.DigestData, article core.Article) (CategoryResult, error) {
	// Try rule-based first
	ruleResult := s.categorizeArticleRuleBased(digestItem, article)

	// If confidence is high enough, return rule-based result
	if ruleResult.Confidence >= 0.8 {
		return ruleResult, nil
	}

	// Otherwise try LLM-based if available
	if s.llmClient != nil {
		// Convert our categories to LLM format
		llmCategories := make(map[string]llm.Category)
		for id, cat := range Categories {
			llmCategories[id] = llm.Category{
				ID:          cat.ID,
				Name:        cat.Name,
				Emoji:       cat.Emoji,
				Priority:    cat.Priority,
				Description: cat.Description,
			}
		}

		llmResult, err := s.llmClient.CategorizeArticle(ctx, article, llmCategories)
		if err == nil && llmResult.Confidence > ruleResult.Confidence {
			// Convert back to our format
			return CategoryResult{
				Category: Category{
					ID:          llmResult.Category.ID,
					Name:        llmResult.Category.Name,
					Emoji:       llmResult.Category.Emoji,
					Priority:    llmResult.Category.Priority,
					Description: llmResult.Category.Description,
				},
				Confidence: llmResult.Confidence,
				Reasoning:  llmResult.Reasoning,
				Source:     llmResult.Source,
			}, nil
		}
	}

	return ruleResult, nil
}

// categorizeArticleRuleBased performs rule-based categorization
func (s *Service) categorizeArticleRuleBased(digestItem render.DigestData, article core.Article) CategoryResult {
	title := strings.ToLower(digestItem.Title)
	summary := strings.ToLower(digestItem.SummaryText)
	content := strings.ToLower(article.CleanedText)

	// Breaking & Hot - high urgency keywords
	if containsAnyKeywords(title, []string{"breaking", "urgent", "alert", "critical", "emergency", "now", "today"}) ||
		containsAnyKeywords(summary, []string{"breaking", "urgent", "just announced", "happening now"}) {
		return CategoryResult{
			Category:   Categories["breaking"],
			Confidence: 0.9,
			Reasoning:  "Contains urgent/breaking news keywords",
			Source:     "rule-based",
		}
	}

	// Product Updates - launches, releases, versions
	if containsAnyKeywords(title, []string{"launch", "release", "announce", "unveil", "version", "update", "v2", "beta", "ga"}) ||
		containsAnyKeywords(summary, []string{"launched", "released", "announced", "new version", "updated", "available now"}) {
		return CategoryResult{
			Category:   Categories["products"],
			Confidence: 0.85,
			Reasoning:  "Contains product launch/release keywords",
			Source:     "rule-based",
		}
	}

	// AI & Machine Learning - AI models, LLMs, ML research
	if containsAnyKeywords(title, []string{"ai", "artificial intelligence", "machine learning", "ml", "llm", "gpt", "neural", "model", "claude", "gemini", "chatgpt"}) ||
		containsAnyKeywords(summary, []string{"artificial intelligence", "machine learning", "neural network", "deep learning", "llm", "language model", "ai model"}) ||
		containsAnyKeywords(content, []string{"machine learning", "artificial intelligence", "neural", "transformer", "llm"}) {
		return CategoryResult{
			Category:   Categories["aiml"],
			Confidence: 0.9,
			Reasoning:  "Contains AI/ML keywords",
			Source:     "rule-based",
		}
	}

	// Security & Privacy - cybersecurity, vulnerabilities, privacy
	if containsAnyKeywords(title, []string{"security", "vulnerability", "breach", "privacy", "encryption", "cyber", "hack", "exploit", "auth", "ssl", "tls"}) ||
		containsAnyKeywords(summary, []string{"security", "vulnerability", "privacy", "encryption", "cybersecurity", "attack", "breach", "exploit"}) ||
		containsAnyKeywords(content, []string{"security", "vulnerability", "privacy", "encryption", "cybersecurity"}) {
		return CategoryResult{
			Category:   Categories["security"],
			Confidence: 0.85,
			Reasoning:  "Contains security/privacy keywords",
			Source:     "rule-based",
		}
	}

	// Dev Tools & Techniques - engineering, tools, tutorials
	if containsAnyKeywords(title, []string{"guide", "tutorial", "how to", "engineering", "development", "coding", "programming", "tool", "framework", "library"}) ||
		containsAnyKeywords(summary, []string{"technical", "engineering", "development", "tutorial", "guide", "tool", "framework"}) ||
		containsAnyKeywords(content, []string{"implementation", "code", "programming", "development", "technical"}) {
		return CategoryResult{
			Category:   Categories["devtools"],
			Confidence: 0.8,
			Reasoning:  "Contains development/engineering keywords",
			Source:     "rule-based",
		}
	}

	// Infrastructure & Cloud - cloud platforms, DevOps, scaling
	if containsAnyKeywords(title, []string{"cloud", "aws", "azure", "gcp", "kubernetes", "docker", "devops", "infrastructure", "scaling", "deployment"}) ||
		containsAnyKeywords(summary, []string{"cloud", "infrastructure", "scaling", "deployment", "devops", "kubernetes", "container"}) ||
		containsAnyKeywords(content, []string{"cloud", "infrastructure", "scaling", "deployment", "devops"}) {
		return CategoryResult{
			Category:   Categories["infrastructure"],
			Confidence: 0.85,
			Reasoning:  "Contains infrastructure/cloud keywords",
			Source:     "rule-based",
		}
	}

	// Research & Analysis - studies, benchmarks, analysis
	if containsAnyKeywords(title, []string{"study", "research", "analysis", "benchmark", "survey", "report", "findings", "data"}) ||
		containsAnyKeywords(summary, []string{"research", "study", "analysis", "benchmark", "findings", "data", "report"}) {
		return CategoryResult{
			Category:   Categories["research"],
			Confidence: 0.8,
			Reasoning:  "Contains research/analysis keywords",
			Source:     "rule-based",
		}
	}

	// Ideas & Inspiration - implementations, case studies, projects
	if containsAnyKeywords(title, []string{"built", "building", "implementation", "project", "case study", "story", "journey"}) ||
		containsAnyKeywords(summary, []string{"implementation", "built", "developed", "created", "project", "case study"}) {
		return CategoryResult{
			Category:   Categories["inspiration"],
			Confidence: 0.75,
			Reasoning:  "Contains implementation/project keywords",
			Source:     "rule-based",
		}
	}

	// Default to Worth Monitoring with moderate confidence
	return CategoryResult{
		Category:   Categories["monitoring"],
		Confidence: 0.6,
		Reasoning:  "Default category - no strong category indicators found",
		Source:     "rule-based",
	}
}

// containsAnyKeywords checks if text contains any of the given keywords
func containsAnyKeywords(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

// SortCategorizedItems sorts categorized items by category priority and relevance
func SortCategorizedItems(items []CategorizedItem) []CategorizedItem {
	// Create a copy to avoid modifying the original slice
	sorted := make([]CategorizedItem, len(items))
	copy(sorted, items)

	// Sort by category priority first, then by relevance score within category
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			// Primary sort: category priority
			if sorted[i].Category.Category.Priority > sorted[j].Category.Category.Priority {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			} else if sorted[i].Category.Category.Priority == sorted[j].Category.Category.Priority {
				// Secondary sort: article relevance score (higher is better)
				if sorted[i].Article.RelevanceScore < sorted[j].Article.RelevanceScore {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}
	}

	return sorted
}

// GroupByCategory groups categorized items by their assigned categories
func GroupByCategory(items []CategorizedItem) map[string][]CategorizedItem {
	grouped := make(map[string][]CategorizedItem)

	for _, item := range items {
		categoryID := item.Category.Category.ID
		grouped[categoryID] = append(grouped[categoryID], item)
	}

	return grouped
}