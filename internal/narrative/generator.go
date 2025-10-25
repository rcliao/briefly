package narrative

import (
	"briefly/internal/core"
	"context"
	"fmt"
	"sort"
	"strings"
)

// LLMClient defines the interface for LLM operations needed by the narrative generator
type LLMClient interface {
	// GenerateText generates text from a prompt
	GenerateText(ctx context.Context, prompt string) (string, error)
}

// Generator creates executive summaries from clustered articles
type Generator struct {
	llmClient LLMClient
}

// NewGenerator creates a new narrative generator
func NewGenerator(llmClient LLMClient) *Generator {
	return &Generator{
		llmClient: llmClient,
	}
}

// GenerateExecutiveSummary creates a story-driven narrative from clustered articles
// Takes the top 3 articles from each cluster and synthesizes them into a 200-word narrative
func (g *Generator) GenerateExecutiveSummary(ctx context.Context, clusters []core.TopicCluster, articles map[string]core.Article, summaries map[string]core.Summary) (string, error) {
	if len(clusters) == 0 {
		return "", fmt.Errorf("no clusters provided")
	}

	// Collect top articles from each cluster
	clusterInsights := make([]ClusterInsight, 0, len(clusters))

	for _, cluster := range clusters {
		insight, err := g.extractClusterInsight(cluster, articles, summaries)
		if err != nil {
			// Log warning but continue with other clusters
			continue
		}
		clusterInsights = append(clusterInsights, insight)
	}

	if len(clusterInsights) == 0 {
		return "", fmt.Errorf("no valid cluster insights generated")
	}

	// Generate narrative using LLM
	prompt := g.buildNarrativePrompt(clusterInsights)
	narrative, err := g.llmClient.GenerateText(ctx, prompt)
	if err != nil {
		// Fallback to bullet points if LLM fails
		return g.generateFallbackNarrative(clusterInsights), nil
	}

	return strings.TrimSpace(narrative), nil
}

// ClusterInsight represents the key information from a topic cluster
type ClusterInsight struct {
	Theme         string
	TopArticles   []ArticleSummary
	KeyThemes     []string
	ArticleCount  int
}

// ArticleSummary contains essential article information for narrative generation
type ArticleSummary struct {
	Title      string
	URL        string
	Summary    string
	KeyPoints  []string
}

// extractClusterInsight extracts the top 3 articles and key information from a cluster
func (g *Generator) extractClusterInsight(cluster core.TopicCluster, articles map[string]core.Article, summaries map[string]core.Summary) (ClusterInsight, error) {
	insight := ClusterInsight{
		Theme:        cluster.Label,
		KeyThemes:    cluster.Keywords,
		ArticleCount: len(cluster.ArticleIDs),
	}

	// Collect article summaries for this cluster
	articleSummaries := make([]ArticleSummary, 0, len(cluster.ArticleIDs))

	for _, articleID := range cluster.ArticleIDs {
		article, hasArticle := articles[articleID]
		summary, hasSummary := summaries[articleID]

		if !hasArticle || !hasSummary {
			continue
		}

		// Extract key points from summary
		keyPoints := g.extractKeyPoints(summary)

		articleSummaries = append(articleSummaries, ArticleSummary{
			Title:     article.Title,
			URL:       article.URL,
			Summary:   summary.SummaryText,
			KeyPoints: keyPoints,
		})
	}

	if len(articleSummaries) == 0 {
		return insight, fmt.Errorf("no article summaries found for cluster")
	}

	// Sort by relevance/quality and take top 3
	// For now, just take first 3 (can add scoring later)
	maxArticles := 3
	if len(articleSummaries) < maxArticles {
		maxArticles = len(articleSummaries)
	}

	insight.TopArticles = articleSummaries[:maxArticles]

	return insight, nil
}

// extractKeyPoints extracts key points from a summary
// Looks for bullet points or numbered lists
func (g *Generator) extractKeyPoints(summary core.Summary) []string {
	// Simple heuristic: split by newlines and look for bullet points
	lines := strings.Split(summary.SummaryText, "\n")
	keyPoints := make([]string, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for bullet points or numbered lists
		if strings.HasPrefix(line, "-") ||
		   strings.HasPrefix(line, "•") ||
		   strings.HasPrefix(line, "*") {
			point := strings.TrimSpace(line[1:])
			if point != "" {
				keyPoints = append(keyPoints, point)
			}
		} else if len(line) > 2 && line[0] >= '1' && line[0] <= '9' && (line[1] == '.' || line[1] == ')') {
			point := strings.TrimSpace(line[2:])
			if point != "" {
				keyPoints = append(keyPoints, point)
			}
		}
	}

	// If no bullet points found, return empty (LLM will work with full summary)
	return keyPoints
}

// buildNarrativePrompt constructs the prompt for executive summary generation
// Uses domain storytelling and structured narrative format
func (g *Generator) buildNarrativePrompt(insights []ClusterInsight) string {
	var prompt strings.Builder

	prompt.WriteString("Generate an executive summary for a weekly tech digest newsletter using domain storytelling principles.\n\n")

	// Build article reference list with numbers
	prompt.WriteString("**Articles for reference:**\n")
	articleNum := 1
	for _, insight := range insights {
		for _, article := range insight.TopArticles {
			prompt.WriteString(fmt.Sprintf("[%d] %s\n", articleNum, article.Title))
			prompt.WriteString(fmt.Sprintf("    Summary: %s\n\n", truncateText(article.Summary, 150)))
			articleNum++
		}
	}

	prompt.WriteString("\n**REQUIRED STRUCTURE:**\n\n")

	prompt.WriteString("1. **Executive Summary (2-3 sentences max)**\n")
	prompt.WriteString("   - State the main pattern/trend immediately\n")
	prompt.WriteString("   - Include the recommendation or key insight upfront\n")
	prompt.WriteString("   - Example: 'AI agents are shifting from assistants to autonomous teammates this week. Three major platforms launched features enabling agents to operate independently, while research reveals critical reliability challenges.'\n\n")

	prompt.WriteString("2. **Key Developments (as workflow/narrative sequence)**\n")
	prompt.WriteString("   - Use domain storytelling format: [Actor] [verb] [System/Data]\n")
	prompt.WriteString("   - Show the workflow/progression of events\n")
	prompt.WriteString("   - Each numbered point should tell part of the story\n")
	prompt.WriteString("   - Include cross-references: [See #X below]\n")
	prompt.WriteString("   - Format example:\n")
	prompt.WriteString("     1. **Anthropic → launches** Skills system enabling developers to customize Claude for domain-specific tasks [See #3]\n")
	prompt.WriteString("     2. **Practitioners → adopt** YOLO mode running 3-8 concurrent agents with minimal oversight [See #1]\n")
	prompt.WriteString("     3. **Researchers → discover** brain rot phenomenon degrading LLM cognition from low-quality data [See #8]\n\n")

	prompt.WriteString("3. **Bottom Line (1 sentence)**\n")
	prompt.WriteString("   - Synthesize the implications\n")
	prompt.WriteString("   - State what matters for the audience\n")
	prompt.WriteString("   - Example: 'As agents gain autonomy, reliability and data quality become the critical bottlenecks for production deployment.'\n\n")

	prompt.WriteString("**NARRATIVE PRINCIPLES:**\n")
	prompt.WriteString("- Tell a story with a clear arc (setup → developments → implications)\n")
	prompt.WriteString("- Use active voice with clear actors and actions\n")
	prompt.WriteString("- Focus on 'why it matters' not 'what happened'\n")
	prompt.WriteString("- Show connections and workflow between developments\n")
	prompt.WriteString("- Keep total length under 150 words\n")
	prompt.WriteString("- Write for software engineers, PMs, and technical leaders\n\n")

	prompt.WriteString("**Example output:**\n")
	prompt.WriteString("**AI development tools reached a turning point this week with three simultaneous breakthroughs in agent autonomy. The shift: from AI-as-helper to AI-as-autonomous-developer.**\n\n")
	prompt.WriteString("1. **Anthropic → releases** Claude Code web platform where developers assign tasks and agents work independently across repositories [See #1]\n")
	prompt.WriteString("2. **Claude → gains** persistent memory for teams, eliminating context re-explanation and enabling true project continuity [See #2]\n")
	prompt.WriteString("3. **Practitioners → discover** optimal workflows running 8+ agents simultaneously with atomic git commits and blast-radius management [See #5]\n\n")
	prompt.WriteString("**Bottom line:** Agent autonomy is production-ready, but success requires new workflows built around parallel execution and granular task isolation.\n\n")

	prompt.WriteString("Now generate the executive summary following this exact structure:")

	return prompt.String()
}

// generateFallbackNarrative creates a simple narrative when LLM fails
func (g *Generator) generateFallbackNarrative(insights []ClusterInsight) string {
	var narrative strings.Builder

	narrative.WriteString("This week's digest covers ")

	themes := make([]string, 0, len(insights))
	for _, insight := range insights {
		themes = append(themes, strings.ToLower(insight.Theme))
	}

	narrative.WriteString(joinWithAnd(themes))
	narrative.WriteString(". ")

	// Add one sentence per cluster
	for _, insight := range insights {
		if len(insight.TopArticles) > 0 {
			narrative.WriteString(fmt.Sprintf("Key developments in %s include ", insight.Theme))

			titles := make([]string, 0, len(insight.TopArticles))
			for _, article := range insight.TopArticles {
				// Extract first sentence of summary as key point
				firstSentence := extractFirstSentence(article.Summary)
				if firstSentence != "" {
					titles = append(titles, strings.ToLower(firstSentence))
				}
			}

			if len(titles) > 0 {
				narrative.WriteString(joinWithAnd(titles))
				narrative.WriteString(". ")
			}
		}
	}

	return narrative.String()
}

// IdentifyClusterTheme generates a descriptive theme for a cluster
func (g *Generator) IdentifyClusterTheme(ctx context.Context, cluster core.TopicCluster, articles []core.Article) (string, error) {
	if cluster.Label != "" && cluster.Label != "unknown" {
		return cluster.Label, nil
	}

	// Build prompt from article titles
	var prompt strings.Builder
	prompt.WriteString("Identify a concise theme (2-4 words) for these related articles:\n\n")

	for i, articleID := range cluster.ArticleIDs {
		for _, article := range articles {
			if article.ID == articleID {
				prompt.WriteString(fmt.Sprintf("%d. %s\n", i+1, article.Title))
				break
			}
		}
	}

	prompt.WriteString("\nTheme:")

	theme, err := g.llmClient.GenerateText(ctx, prompt.String())
	if err != nil {
		// Fallback to generic theme
		return fmt.Sprintf("Topic Cluster %d", 1), nil
	}

	return strings.TrimSpace(theme), nil
}

// SelectTopArticles selects the top N articles from a cluster based on quality/relevance
func (g *Generator) SelectTopArticles(cluster core.TopicCluster, articles []core.Article, n int) []core.Article {
	if n <= 0 || len(cluster.ArticleIDs) == 0 {
		return []core.Article{}
	}

	// Collect articles from cluster
	clusterArticles := make([]core.Article, 0, len(cluster.ArticleIDs))
	for _, articleID := range cluster.ArticleIDs {
		for _, article := range articles {
			if article.ID == articleID {
				clusterArticles = append(clusterArticles, article)
				break
			}
		}
	}

	// Sort by signal strength (or quality score if available)
	sort.Slice(clusterArticles, func(i, j int) bool {
		scoreI := clusterArticles[i].SignalStrength
		if scoreI == 0 {
			scoreI = clusterArticles[i].QualityScore
		}

		scoreJ := clusterArticles[j].SignalStrength
		if scoreJ == 0 {
			scoreJ = clusterArticles[j].QualityScore
		}

		return scoreI > scoreJ
	})

	// Take top N
	if len(clusterArticles) > n {
		clusterArticles = clusterArticles[:n]
	}

	return clusterArticles
}

// Helper functions

func truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}

	truncated := text[:maxLength]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}

func extractFirstSentence(text string) string {
	// Find first sentence (ending with . ! or ?)
	for i, char := range text {
		if char == '.' || char == '!' || char == '?' {
			if i+1 < len(text) {
				return strings.TrimSpace(text[:i+1])
			}
			return strings.TrimSpace(text)
		}
	}

	// If no sentence ending found, truncate at reasonable length
	if len(text) > 100 {
		return truncateText(text, 100)
	}

	return text
}

func joinWithAnd(items []string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}
	if len(items) == 2 {
		return items[0] + " and " + items[1]
	}

	// Join all but last with commas, then add "and" before last
	allButLast := strings.Join(items[:len(items)-1], ", ")
	return allButLast + ", and " + items[len(items)-1]
}