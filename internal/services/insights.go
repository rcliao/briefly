package services

import (
	"briefly/internal/core"
	"briefly/internal/llm"
	"context"
	"fmt"
	"strings"
	"time"
)

// InsightsSynthesizer generates actionable insights from research results
type InsightsSynthesizer struct {
	llmClient *llm.Client
}

// NewInsightsSynthesizer creates a new insights synthesizer
func NewInsightsSynthesizer(llmClient *llm.Client) *InsightsSynthesizer {
	return &InsightsSynthesizer{
		llmClient: llmClient,
	}
}

// CompetitiveIntelligence contains competitive analysis insights
type CompetitiveIntelligence struct {
	MarketPosition  string   `json:"market_position"`  // Relative strengths/weaknesses vs competitors
	FeatureGaps     []string `json:"feature_gaps"`     // Missing capabilities compared to leaders
	PricingAnalysis string   `json:"pricing_analysis"` // Cost comparison and value proposition
	UserSentiment   string   `json:"user_sentiment"`   // Community feedback and adoption patterns
	KeyCompetitors  []string `json:"key_competitors"`  // Main competing solutions identified
	CompetitiveEdge []string `json:"competitive_edge"` // Unique advantages or differentiators
}

// TechnicalAssessment contains technical analysis insights
type TechnicalAssessment struct {
	ArchitectureOverview  string            `json:"architecture_overview"`  // System design and technical approach
	PerformanceBenchmarks string            `json:"performance_benchmarks"` // Quantitative performance data
	IntegrationComplexity string            `json:"integration_complexity"` // Ease of adoption and implementation
	SecurityPosture       string            `json:"security_posture"`       // Security practices and vulnerabilities
	TechnicalLimitations  []string          `json:"technical_limitations"`  // Known technical constraints
	ScalabilityFactors    map[string]string `json:"scalability_factors"`    // Scalability characteristics
}

// StrategicRecommendations contains actionable strategic insights
type StrategicRecommendations struct {
	AdoptionReadiness      string            `json:"adoption_readiness"`      // Technical requirements and prerequisites
	RiskAssessment         []string          `json:"risk_assessment"`         // Potential challenges and mitigation strategies
	ImplementationTimeline map[string]string `json:"implementation_timeline"` // Suggested evaluation and deployment phases
	SuccessMetrics         []string          `json:"success_metrics"`         // KPIs for measuring adoption success
	NextSteps              []string          `json:"next_steps"`              // Immediate action items
	AlternativeOptions     []string          `json:"alternative_options"`     // Backup or alternative solutions
}

// ActionableInsights contains comprehensive research insights
type ActionableInsights struct {
	CompetitiveIntelligence  CompetitiveIntelligence  `json:"competitive_intelligence"`
	TechnicalAssessment      TechnicalAssessment      `json:"technical_assessment"`
	StrategicRecommendations StrategicRecommendations `json:"strategic_recommendations"`
	ExecutiveSummary         string                   `json:"executive_summary"` // High-level summary for decision makers
	ConfidenceLevel          float64                  `json:"confidence_level"`  // Overall confidence in insights (0-1)
	DataGaps                 []string                 `json:"data_gaps"`         // Areas needing more research
	GeneratedAt              time.Time                `json:"generated_at"`
}

// SynthesizeInsights generates comprehensive actionable insights from clustered research results
func (is *InsightsSynthesizer) SynthesizeInsights(ctx context.Context, query string, clusteringResult *ClusteringResult) (*ActionableInsights, error) {
	if len(clusteringResult.Categories) == 0 {
		return &ActionableInsights{
			ExecutiveSummary: "Insufficient research data to generate actionable insights.",
			ConfidenceLevel:  0.0,
			DataGaps:         []string{"No research results available"},
			GeneratedAt:      time.Now(),
		}, nil
	}

	// Generate competitive intelligence
	competitive, err := is.generateCompetitiveIntelligence(ctx, query, clusteringResult)
	if err != nil {
		return nil, fmt.Errorf("failed to generate competitive intelligence: %w", err)
	}

	// Generate technical assessment
	technical, err := is.generateTechnicalAssessment(ctx, query, clusteringResult)
	if err != nil {
		return nil, fmt.Errorf("failed to generate technical assessment: %w", err)
	}

	// Generate strategic recommendations
	strategic, err := is.generateStrategicRecommendations(ctx, query, clusteringResult, competitive, technical)
	if err != nil {
		return nil, fmt.Errorf("failed to generate strategic recommendations: %w", err)
	}

	// Generate executive summary
	execSummary, err := is.generateExecutiveSummary(ctx, query, competitive, technical, strategic)
	if err != nil {
		return nil, fmt.Errorf("failed to generate executive summary: %w", err)
	}

	// Calculate confidence level based on data quality and coverage
	confidence := is.calculateConfidenceLevel(clusteringResult)

	// Compile data gaps
	dataGaps := is.identifyDataGaps(clusteringResult)

	return &ActionableInsights{
		CompetitiveIntelligence:  competitive,
		TechnicalAssessment:      technical,
		StrategicRecommendations: strategic,
		ExecutiveSummary:         execSummary,
		ConfidenceLevel:          confidence,
		DataGaps:                 dataGaps,
		GeneratedAt:              time.Now(),
	}, nil
}

// generateCompetitiveIntelligence creates competitive analysis insights
func (is *InsightsSynthesizer) generateCompetitiveIntelligence(ctx context.Context, query string, clusteringResult *ClusteringResult) (CompetitiveIntelligence, error) {
	// Find competitive analysis category
	var competitiveResults []core.ResearchResult
	for _, category := range clusteringResult.Categories {
		if strings.Contains(strings.ToLower(category.Name), "competitive") ||
			strings.Contains(strings.ToLower(category.Name), "comparison") {
			competitiveResults = category.Results
			break
		}
	}

	// Build context from competitive results
	var contextBuilder strings.Builder
	contextBuilder.WriteString("Competitive research findings:\n")
	for i, result := range competitiveResults {
		if i >= 8 { // Limit to top 8 results
			break
		}
		contextBuilder.WriteString(fmt.Sprintf("- %s: %s\n", result.Title, result.Snippet))
	}

	if len(competitiveResults) == 0 {
		contextBuilder.WriteString("(Limited competitive analysis data available)")
	}

	prompt := fmt.Sprintf(`Based on research about "%s" and the competitive findings below, generate competitive intelligence insights:

%s

Analyze the competitive landscape and provide:

1. Market Position: Where does this solution stand relative to competitors? What are its key strengths and weaknesses?

2. Feature Gaps: What capabilities are competitors offering that this solution lacks? What are the most significant gaps?

3. Pricing Analysis: How does pricing compare to alternatives? What's the value proposition?

4. User Sentiment: What do users think about this vs alternatives? Any adoption patterns or preferences?

5. Key Competitors: List 3-5 main competing solutions mentioned in the research.

6. Competitive Edge: What unique advantages or differentiators does this solution have?

Format your response as structured insights, not as a list. Focus on actionable intelligence for decision-making.`, query, contextBuilder.String())

	response, err := is.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		MaxTokens:   1000,
		Temperature: 0.6,
		Model:       "gemini-1.5-flash",
	})
	if err != nil {
		return CompetitiveIntelligence{}, fmt.Errorf("failed to generate competitive intelligence: %w", err)
	}

	// Parse structured response (simplified implementation)
	return is.parseCompetitiveIntelligence(response, competitiveResults), nil
}

// generateTechnicalAssessment creates technical analysis insights
func (is *InsightsSynthesizer) generateTechnicalAssessment(ctx context.Context, query string, clusteringResult *ClusteringResult) (TechnicalAssessment, error) {
	// Find technical details category
	var technicalResults []core.ResearchResult
	for _, category := range clusteringResult.Categories {
		if strings.Contains(strings.ToLower(category.Name), "technical") ||
			strings.Contains(strings.ToLower(category.Name), "architecture") {
			technicalResults = category.Results
			break
		}
	}

	// Build context from technical results
	var contextBuilder strings.Builder
	contextBuilder.WriteString("Technical research findings:\n")
	for i, result := range technicalResults {
		if i >= 8 { // Limit to top 8 results
			break
		}
		contextBuilder.WriteString(fmt.Sprintf("- %s: %s\n", result.Title, result.Snippet))
	}

	if len(technicalResults) == 0 {
		contextBuilder.WriteString("(Limited technical analysis data available)")
	}

	prompt := fmt.Sprintf(`Based on research about "%s" and the technical findings below, generate technical assessment insights:

%s

Analyze the technical aspects and provide:

1. Architecture Overview: How is this solution designed? What's the overall technical approach?

2. Performance Benchmarks: What quantitative performance data is available? How does it perform under load?

3. Integration Complexity: How easy is it to adopt and integrate? What's the developer experience like?

4. Security Posture: What security practices, vulnerabilities, or compliance considerations are mentioned?

5. Technical Limitations: What are the known technical constraints, bottlenecks, or challenges?

6. Scalability Factors: How well does it scale? What are the scaling characteristics and limits?

Focus on concrete technical information that would help evaluate implementation feasibility.`, query, contextBuilder.String())

	response, err := is.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		MaxTokens:   1000,
		Temperature: 0.6,
		Model:       "gemini-1.5-flash",
	})
	if err != nil {
		return TechnicalAssessment{}, fmt.Errorf("failed to generate technical assessment: %w", err)
	}

	// Parse structured response (simplified implementation)
	return is.parseTechnicalAssessment(response, technicalResults), nil
}

// generateStrategicRecommendations creates strategic insights and recommendations
func (is *InsightsSynthesizer) generateStrategicRecommendations(ctx context.Context, query string, clusteringResult *ClusteringResult, competitive CompetitiveIntelligence, technical TechnicalAssessment) (StrategicRecommendations, error) {
	// Combine insights from multiple categories
	var allResults []core.ResearchResult
	for _, category := range clusteringResult.Categories {
		allResults = append(allResults, category.Results...)
	}

	// Build comprehensive context
	var contextBuilder strings.Builder
	contextBuilder.WriteString(fmt.Sprintf("Research Query: %s\n", query))
	contextBuilder.WriteString(fmt.Sprintf("Total Results Analyzed: %d\n", len(allResults)))
	contextBuilder.WriteString(fmt.Sprintf("Overall Quality Score: %.2f\n", clusteringResult.OverallQuality))
	contextBuilder.WriteString(fmt.Sprintf("Coverage Gaps: %s\n", strings.Join(clusteringResult.CoverageGaps, ", ")))

	prompt := fmt.Sprintf(`Based on comprehensive research about "%s" with the following context:

%s

Generate strategic recommendations for decision-making:

1. Adoption Readiness: What technical requirements and prerequisites are needed before implementation?

2. Risk Assessment: What are the top 3-5 potential challenges and how can they be mitigated?

3. Implementation Timeline: What are the suggested phases for evaluation and deployment? 

4. Success Metrics: What KPIs should be tracked to measure adoption success?

5. Next Steps: What are the immediate action items to move forward?

6. Alternative Options: What backup or alternative solutions should be considered?

Provide actionable, practical recommendations for technology decision-makers.`, query, contextBuilder.String())

	response, err := is.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		MaxTokens:   1000,
		Temperature: 0.6,
		Model:       "gemini-1.5-flash",
	})
	if err != nil {
		return StrategicRecommendations{}, fmt.Errorf("failed to generate strategic recommendations: %w", err)
	}

	// Parse structured response (simplified implementation)
	return is.parseStrategicRecommendations(response), nil
}

// generateExecutiveSummary creates a high-level summary for decision makers
func (is *InsightsSynthesizer) generateExecutiveSummary(ctx context.Context, query string, competitive CompetitiveIntelligence, technical TechnicalAssessment, strategic StrategicRecommendations) (string, error) {
	prompt := fmt.Sprintf(`Create a concise executive summary for technology decision-makers about "%s".

Key points to address:
- What is this solution and why is it relevant?
- How does it compare to alternatives in the market?
- What are the main technical considerations?
- What's the recommended approach for evaluation/adoption?
- What are the key risks and benefits?

Keep it to 3-4 paragraphs, focusing on business impact and decision-making insights.
Write for executives who need the key points without technical details.`, query)

	response, err := is.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		MaxTokens:   500,
		Temperature: 0.5,
		Model:       "gemini-1.5-flash",
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate executive summary: %w", err)
	}

	return strings.TrimSpace(response), nil
}

// calculateConfidenceLevel determines confidence in insights based on data quality
func (is *InsightsSynthesizer) calculateConfidenceLevel(clusteringResult *ClusteringResult) float64 {
	if clusteringResult.TotalCategorized == 0 {
		return 0.0
	}

	// Base confidence on overall quality
	confidence := clusteringResult.OverallQuality

	// Boost confidence for good coverage (fewer gaps)
	gapPenalty := float64(len(clusteringResult.CoverageGaps)) * 0.1
	confidence -= gapPenalty

	// Boost confidence for sufficient data volume
	if clusteringResult.TotalCategorized >= 10 {
		confidence += 0.1
	}
	if clusteringResult.TotalCategorized >= 20 {
		confidence += 0.1
	}

	// Penalize for many uncategorized results
	if clusteringResult.UncategorizedCount > clusteringResult.TotalCategorized/2 {
		confidence -= 0.15
	}

	// Ensure confidence stays in 0-1 range
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	}

	return confidence
}

// identifyDataGaps finds areas needing more research
func (is *InsightsSynthesizer) identifyDataGaps(clusteringResult *ClusteringResult) []string {
	gaps := clusteringResult.CoverageGaps

	// Add specific insight gaps
	competitiveFound := false
	technicalFound := false
	useCaseFound := false

	for _, category := range clusteringResult.Categories {
		name := strings.ToLower(category.Name)
		if strings.Contains(name, "competitive") && len(category.Results) > 2 {
			competitiveFound = true
		}
		if strings.Contains(name, "technical") && len(category.Results) > 2 {
			technicalFound = true
		}
		if strings.Contains(name, "use") && len(category.Results) > 2 {
			useCaseFound = true
		}
	}

	if !competitiveFound {
		gaps = append(gaps, "Insufficient competitive analysis data")
	}
	if !technicalFound {
		gaps = append(gaps, "Limited technical implementation details")
	}
	if !useCaseFound {
		gaps = append(gaps, "Missing real-world use case examples")
	}

	return gaps
}

// Simplified parsing functions (in a real implementation, these would be more sophisticated)

func (is *InsightsSynthesizer) parseCompetitiveIntelligence(response string, results []core.ResearchResult) CompetitiveIntelligence {
	// Extract competitors mentioned in results
	var competitors []string
	competitorKeywords := []string{"vs", "alternative", "competitor", "compared to"}

	for _, result := range results {
		text := strings.ToLower(result.Title + " " + result.Snippet)
		for _, keyword := range competitorKeywords {
			if strings.Contains(text, keyword) {
				// Simple extraction - would be more sophisticated in practice
				words := strings.Fields(text)
				for i, word := range words {
					if word == keyword && i+1 < len(words) {
						competitors = append(competitors, words[i+1])
					}
				}
			}
		}
	}

	// Remove duplicates and limit
	competitors = is.uniqueStrings(competitors)
	if len(competitors) > 5 {
		competitors = competitors[:5]
	}

	return CompetitiveIntelligence{
		MarketPosition:  is.extractSection(response, "market position", "Market Position"),
		FeatureGaps:     is.extractList(response, "feature gaps", "gaps"),
		PricingAnalysis: is.extractSection(response, "pricing", "Pricing Analysis"),
		UserSentiment:   is.extractSection(response, "user sentiment", "sentiment"),
		KeyCompetitors:  competitors,
		CompetitiveEdge: is.extractList(response, "competitive edge", "advantages"),
	}
}

func (is *InsightsSynthesizer) parseTechnicalAssessment(response string, results []core.ResearchResult) TechnicalAssessment {
	return TechnicalAssessment{
		ArchitectureOverview:  is.extractSection(response, "architecture", "Architecture Overview"),
		PerformanceBenchmarks: is.extractSection(response, "performance", "benchmarks"),
		IntegrationComplexity: is.extractSection(response, "integration", "Integration Complexity"),
		SecurityPosture:       is.extractSection(response, "security", "Security Posture"),
		TechnicalLimitations:  is.extractList(response, "limitations", "constraints"),
		ScalabilityFactors: map[string]string{
			"assessment": is.extractSection(response, "scalability", "Scalability Factors"),
		},
	}
}

func (is *InsightsSynthesizer) parseStrategicRecommendations(response string) StrategicRecommendations {
	return StrategicRecommendations{
		AdoptionReadiness: is.extractSection(response, "adoption readiness", "requirements"),
		RiskAssessment:    is.extractList(response, "risk", "challenges"),
		ImplementationTimeline: map[string]string{
			"timeline": is.extractSection(response, "timeline", "Implementation Timeline"),
		},
		SuccessMetrics:     is.extractList(response, "success metrics", "kpis"),
		NextSteps:          is.extractList(response, "next steps", "action items"),
		AlternativeOptions: is.extractList(response, "alternative", "backup"),
	}
}

// Helper functions for parsing responses

func (is *InsightsSynthesizer) extractSection(text, keyword, fallback string) string {
	text = strings.ToLower(text)
	keyword = strings.ToLower(keyword)

	// Find the section containing the keyword
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.Contains(line, keyword) {
			// Return the next few lines as the section content
			var section strings.Builder
			for j := i; j < len(lines) && j < i+3; j++ {
				if strings.TrimSpace(lines[j]) != "" {
					section.WriteString(strings.TrimSpace(lines[j]))
					section.WriteString(" ")
				}
			}
			content := strings.TrimSpace(section.String())
			if len(content) > 10 {
				return content
			}
		}
	}

	return fmt.Sprintf("No specific %s information found in research results.", fallback)
}

func (is *InsightsSynthesizer) extractList(text, keyword, fallback string) []string {
	// Simple list extraction - would be more sophisticated in practice
	section := is.extractSection(text, keyword, fallback)
	items := strings.Split(section, ",")

	var cleanItems []string
	for _, item := range items {
		clean := strings.TrimSpace(item)
		if len(clean) > 3 && len(clean) < 100 {
			cleanItems = append(cleanItems, clean)
		}
	}

	if len(cleanItems) == 0 {
		return []string{fmt.Sprintf("No specific %s identified in research results", fallback)}
	}

	// Limit to top 5 items
	if len(cleanItems) > 5 {
		cleanItems = cleanItems[:5]
	}

	return cleanItems
}

func (is *InsightsSynthesizer) uniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !seen[item] && len(item) > 2 {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
