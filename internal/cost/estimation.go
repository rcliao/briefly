package cost

import (
	"briefly/internal/core"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

// GeminiPricing represents the current pricing for Gemini models
type GeminiPricing struct {
	Model                   string
	InputCostPer1MTokens    float64 // Cost per 1M input tokens in USD
	OutputCostPer1MTokens   float64 // Cost per 1M output tokens in USD
	EstimatedOutputTokens   int     // Estimated output tokens per request
	MaxRequestsPerMinute    int     // Rate limiting
}

// PricingTable contains current Gemini pricing as of 2025
var PricingTable = map[string]GeminiPricing{
	"gemini-1.5-flash-latest": {
		Model:                   "gemini-1.5-flash-latest",
		InputCostPer1MTokens:    0.075,  // $0.075 per 1M tokens
		OutputCostPer1MTokens:   0.30,   // $0.30 per 1M tokens
		EstimatedOutputTokens:   200,    // Typical summary length
		MaxRequestsPerMinute:    1000,
	},
	"gemini-1.5-pro-latest": {
		Model:                   "gemini-1.5-pro-latest", 
		InputCostPer1MTokens:    3.50,   // $3.50 per 1M tokens
		OutputCostPer1MTokens:   10.50,  // $10.50 per 1M tokens
		EstimatedOutputTokens:   250,    // Slightly longer for pro model
		MaxRequestsPerMinute:    360,
	},
	"gemini-1.5-flash": {
		Model:                   "gemini-1.5-flash",
		InputCostPer1MTokens:    0.075,
		OutputCostPer1MTokens:   0.30,
		EstimatedOutputTokens:   200,
		MaxRequestsPerMinute:    1000,
	},
	"gemini-1.5-pro": {
		Model:                   "gemini-1.5-pro",
		InputCostPer1MTokens:    3.50,
		OutputCostPer1MTokens:   10.50,
		EstimatedOutputTokens:   250,
		MaxRequestsPerMinute:    360,
	},
}

// EstimateTokenCount provides a rough estimation of token count for text
// This is a simplified approximation: typically 1 token ≈ 0.75 words ≈ 4 characters
func EstimateTokenCount(text string) int {
	// Remove excessive whitespace and normalize
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\n", " ")
	
	// Count characters (more accurate than word count for mixed content)
	charCount := utf8.RuneCountInString(text)
	
	// Rough estimation: 1 token ≈ 4 characters for English text
	// Add some buffer for special tokens, formatting, etc.
	tokenCount := int(math.Ceil(float64(charCount) / 3.5))
	
	return tokenCount
}

// ArticleCostEstimate represents the cost estimation for processing a single article
type ArticleCostEstimate struct {
	URL                string
	Title              string
	EstimatedInputTokens  int
	EstimatedOutputTokens int
	InputCost          float64
	OutputCost         float64
	TotalCost          float64
	Error              error
}

// DigestCostEstimate represents the total cost estimation for a digest operation
type DigestCostEstimate struct {
	Model                  string
	Articles               []ArticleCostEstimate
	FinalDigestInputTokens int
	FinalDigestOutputTokens int
	FinalDigestCost        float64
	TotalInputTokens       int
	TotalOutputTokens      int
	TotalCost              float64
	ProcessingTimeMinutes  float64
	RateLimitWarning       string
}

// EstimateDigestCost estimates the cost of processing a digest with the given links
func EstimateDigestCost(links []core.Link, modelName string) (*DigestCostEstimate, error) {
	pricing, exists := PricingTable[modelName]
	if !exists {
		// Default to Flash pricing if model not found
		pricing = PricingTable["gemini-1.5-flash-latest"]
	}
	
	estimate := &DigestCostEstimate{
		Model:    modelName,
		Articles: make([]ArticleCostEstimate, 0, len(links)),
	}
	
	// Estimate cost for each article
	for _, link := range links {
		articleEst := estimateArticleCost(link, pricing)
		estimate.Articles = append(estimate.Articles, articleEst)
		
		if articleEst.Error == nil {
			estimate.TotalInputTokens += articleEst.EstimatedInputTokens
			estimate.TotalOutputTokens += articleEst.EstimatedOutputTokens
			estimate.TotalCost += articleEst.TotalCost
		}
	}
	
	// Estimate final digest generation cost
	// Assume combining all summaries into one prompt
	summaryTokens := estimate.TotalOutputTokens // All individual summaries
	finalDigestPromptTokens := summaryTokens + 200 // Add prompt overhead
	finalDigestOutputTokens := 400 // Longer summary for final digest
	
	estimate.FinalDigestInputTokens = finalDigestPromptTokens
	estimate.FinalDigestOutputTokens = finalDigestOutputTokens
	
	finalDigestInputCost := float64(finalDigestPromptTokens) * pricing.InputCostPer1MTokens / 1000000
	finalDigestOutputCost := float64(finalDigestOutputTokens) * pricing.OutputCostPer1MTokens / 1000000
	estimate.FinalDigestCost = finalDigestInputCost + finalDigestOutputCost
	
	estimate.TotalInputTokens += finalDigestPromptTokens
	estimate.TotalOutputTokens += finalDigestOutputTokens
	estimate.TotalCost += estimate.FinalDigestCost
	
	// Estimate processing time (sequential processing + rate limits)
	totalRequests := len(links) + 1 // Articles + final digest
	estimate.ProcessingTimeMinutes = float64(totalRequests) * 2 / 60 // Assume 2 seconds per request
	
	// Check rate limits
	requestsPerMinute := float64(totalRequests) / math.Max(estimate.ProcessingTimeMinutes, 1)
	if requestsPerMinute > float64(pricing.MaxRequestsPerMinute) {
		estimate.RateLimitWarning = fmt.Sprintf(
			"Warning: Estimated %d requests may exceed rate limit of %d/min for %s",
			totalRequests, pricing.MaxRequestsPerMinute, modelName,
		)
	}
	
	return estimate, nil
}

// estimateArticleCost estimates the cost for processing a single article
func estimateArticleCost(link core.Link, pricing GeminiPricing) ArticleCostEstimate {
	// For cost estimation, we'll use some heuristics based on typical article lengths
	// Real implementation would fetch the article, but that defeats the purpose of dry-run
	
	// Estimate article length based on URL patterns and common article sizes
	estimatedContentLength := estimateArticleLength(link.URL)
	
	// Add prompt overhead (template + instructions)
	promptOverhead := 150 // tokens for prompt template
	inputTokens := EstimateTokenCount(estimatedContentLength) + promptOverhead
	outputTokens := pricing.EstimatedOutputTokens
	
	inputCost := float64(inputTokens) * pricing.InputCostPer1MTokens / 1000000
	outputCost := float64(outputTokens) * pricing.OutputCostPer1MTokens / 1000000
	totalCost := inputCost + outputCost
	
	return ArticleCostEstimate{
		URL:                   link.URL,
		Title:                link.URL, // Use URL as title for estimation since we don't have link text
		EstimatedInputTokens:  inputTokens,
		EstimatedOutputTokens: outputTokens,
		InputCost:            inputCost,
		OutputCost:           outputCost,
		TotalCost:            totalCost,
	}
}

// estimateArticleLength provides a rough estimate of article content length
func estimateArticleLength(url string) string {
	// This is a heuristic-based estimation
	// In reality, different sites have different article lengths
	
	// Check for known patterns
	urlLower := strings.ToLower(url)
	
	switch {
	case strings.Contains(urlLower, "twitter.com") || strings.Contains(urlLower, "x.com"):
		return strings.Repeat("word ", 50) // Short tweets/posts
	case strings.Contains(urlLower, "github.com"):
		return strings.Repeat("word ", 300) // Medium README or issue
	case strings.Contains(urlLower, "news.ycombinator.com"):
		return strings.Repeat("word ", 100) // Short discussion
	case strings.Contains(urlLower, "medium.com") || strings.Contains(urlLower, "substack.com"):
		return strings.Repeat("word ", 1200) // Long-form articles
	case strings.Contains(urlLower, "blog") || strings.Contains(urlLower, "post"):
		return strings.Repeat("word ", 800) // Typical blog post
	case strings.Contains(urlLower, "arxiv.org"):
		return strings.Repeat("word ", 2000) // Research papers (abstracts)
	case strings.Contains(urlLower, "documentation") || strings.Contains(urlLower, "docs"):
		return strings.Repeat("word ", 600) // Documentation pages
	default:
		return strings.Repeat("word ", 700) // Default article length
	}
}

// FormatEstimate formats the cost estimate for display
func (e *DigestCostEstimate) FormatEstimate() string {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("Cost Estimation for %s\n", e.Model))
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")
	
	// Summary
	sb.WriteString(fmt.Sprintf("📊 Summary:\n"))
	sb.WriteString(fmt.Sprintf("   Articles to process: %d\n", len(e.Articles)))
	sb.WriteString(fmt.Sprintf("   Total estimated cost: $%.6f\n", e.TotalCost))
	sb.WriteString(fmt.Sprintf("   Estimated processing time: %.1f minutes\n", e.ProcessingTimeMinutes))
	
	if e.RateLimitWarning != "" {
		sb.WriteString(fmt.Sprintf("   ⚠️  %s\n", e.RateLimitWarning))
	}
	sb.WriteString("\n")
	
	// Breakdown
	sb.WriteString(fmt.Sprintf("💰 Cost Breakdown:\n"))
	sb.WriteString(fmt.Sprintf("   Input tokens: %d (~$%.6f)\n", 
		e.TotalInputTokens, float64(e.TotalInputTokens)*PricingTable[e.Model].InputCostPer1MTokens/1000000))
	sb.WriteString(fmt.Sprintf("   Output tokens: %d (~$%.6f)\n", 
		e.TotalOutputTokens, float64(e.TotalOutputTokens)*PricingTable[e.Model].OutputCostPer1MTokens/1000000))
	sb.WriteString(fmt.Sprintf("   Final digest: $%.6f\n", e.FinalDigestCost))
	sb.WriteString("\n")
	
	// Per-article breakdown (show first 5)
	if len(e.Articles) > 0 {
		sb.WriteString(fmt.Sprintf("📝 Per-Article Estimates (showing first 5):\n"))
		for i, article := range e.Articles {
			if i >= 5 {
				sb.WriteString(fmt.Sprintf("   ... and %d more articles\n", len(e.Articles)-5))
				break
			}
			if article.Error != nil {
				sb.WriteString(fmt.Sprintf("   %d. ERROR: %s\n", i+1, article.Error))
			} else {
				sb.WriteString(fmt.Sprintf("   %d. $%.6f - %s\n", i+1, article.TotalCost, article.URL))
			}
		}
	}
	
	return sb.String()
}
