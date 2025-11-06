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

// DigestContent contains all generated content for a digest
type DigestContent struct {
	Title            string   // Generated title (e.g., "GPU Economics and Kernel Security")
	TLDRSummary      string   // One-line summary for homepage preview
	KeyMoments       []string // 3-5 key developments/highlights
	ExecutiveSummary string   // Full narrative summary
}

// GenerateDigestContent creates title, TL;DR, and executive summary from clustered articles
// Returns structured content with all three components
func (g *Generator) GenerateDigestContent(ctx context.Context, clusters []core.TopicCluster, articles map[string]core.Article, summaries map[string]core.Summary) (*DigestContent, error) {
	if len(clusters) == 0 {
		return nil, fmt.Errorf("no clusters provided")
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
		return nil, fmt.Errorf("no valid cluster insights generated")
	}

	// Generate content using LLM
	prompt := g.buildNarrativePrompt(clusterInsights)
	response, err := g.llmClient.GenerateText(ctx, prompt)
	if err != nil {
		// Fallback to simple generation if LLM fails
		content := g.generateFallbackContent(clusterInsights)
		return &content, nil
	}

	// Parse structured response
	content := g.parseDigestContent(response, clusterInsights)
	return &content, nil
}

// GenerateExecutiveSummary creates a story-driven narrative from clustered articles
// DEPRECATED: Use GenerateDigestContent instead
// Maintained for backward compatibility
func (g *Generator) GenerateExecutiveSummary(ctx context.Context, clusters []core.TopicCluster, articles map[string]core.Article, summaries map[string]core.Summary) (string, error) {
	content, err := g.GenerateDigestContent(ctx, clusters, articles, summaries)
	if err != nil {
		return "", err
	}
	return content.ExecutiveSummary, nil
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

// buildNarrativePrompt constructs the prompt for digest content generation
// Generates Title, TL;DR, and Executive Summary
func (g *Generator) buildNarrativePrompt(insights []ClusterInsight) string {
	var prompt strings.Builder

	prompt.WriteString("Generate complete content for a weekly tech digest newsletter using domain storytelling principles.\n\n")

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

	prompt.WriteString("\n**REQUIRED OUTPUT FORMAT:**\n\n")

	prompt.WriteString("===TITLE===\n")
	prompt.WriteString("[Generate a catchy 3-6 word title capturing the week's main themes]\n")
	prompt.WriteString("Example: \"GPU Economics and Kernel Security\"\n")
	prompt.WriteString("Example: \"AI Agents Gain Autonomy\"\n")
	prompt.WriteString("Example: \"Platform Wars and Developer Tools\"\n\n")

	prompt.WriteString("===TLDR===\n")
	prompt.WriteString("[Generate ONE sentence (max 150 chars) capturing the week's key insight]\n")
	prompt.WriteString("Example: \"AI infrastructure costs collide with security threats as kernel backdoors expose fundamental platform vulnerabilities\"\n")
	prompt.WriteString("Example: \"Agent autonomy reaches production readiness while reliability challenges emerge as the new bottleneck\"\n\n")

	prompt.WriteString("===KEY_MOMENTS===\n")
	prompt.WriteString("[Generate 3-5 key developments as a numbered list. Each moment should be actor-verb-object format]\n")
	prompt.WriteString("Format: \"1. **Actor → verb** description [See #X]\"\n")
	prompt.WriteString("Example:\n")
	prompt.WriteString("1. **Anthropic → releases** Claude Code web platform enabling autonomous development [See #1]\n")
	prompt.WriteString("2. **Claude → gains** persistent team memory eliminating context re-explanation [See #2]\n")
	prompt.WriteString("3. **Practitioners → discover** optimal workflows running 8+ agents with atomic commits [See #5]\n\n")

	prompt.WriteString("===SUMMARY===\n")
	prompt.WriteString("[Generate a cohesive executive summary (150-200 words) that tells the story of this week's developments]\n\n")
	prompt.WriteString("Structure:\n")
	prompt.WriteString("1. Opening: State the main pattern/trend (2-3 sentences)\n")
	prompt.WriteString("2. Key developments: Describe 3-5 important developments as a flowing narrative\n")
	prompt.WriteString("3. Synthesis: What this means for the audience\n\n")
	prompt.WriteString("Style:\n")
	prompt.WriteString("- Use domain storytelling format where relevant: Actor → verb → System/Data\n")
	prompt.WriteString("- Connect ideas with transitions, not bullet points\n")
	prompt.WriteString("- Include article references [See #X] inline where relevant\n")
	prompt.WriteString("- Focus on 'why it matters' not just 'what happened'\n\n")

	prompt.WriteString("**NARRATIVE PRINCIPLES:**\n")
	prompt.WriteString("- Tell a story with a clear arc (setup → developments → implications)\n")
	prompt.WriteString("- Use active voice with clear actors and actions\n")
	prompt.WriteString("- Focus on 'why it matters' not 'what happened'\n")
	prompt.WriteString("- Show connections and workflow between developments\n")
	prompt.WriteString("- Keep total length under 150 words\n")
	prompt.WriteString("- Write for software engineers, PMs, and technical leaders\n\n")

	prompt.WriteString("**Example output:**\n\n")
	prompt.WriteString("===TITLE===\n")
	prompt.WriteString("AI Agents Gain Autonomy\n\n")
	prompt.WriteString("===TLDR===\n")
	prompt.WriteString("Agent autonomy reaches production readiness while reliability challenges emerge as the new bottleneck\n\n")
	prompt.WriteString("===KEY_MOMENTS===\n")
	prompt.WriteString("1. **Anthropic → releases** Claude Code web platform enabling autonomous development [See #1]\n")
	prompt.WriteString("2. **Claude → gains** persistent team memory eliminating context re-explanation [See #2]\n")
	prompt.WriteString("3. **Practitioners → discover** optimal workflows running 8+ agents with atomic commits [See #5]\n\n")
	prompt.WriteString("===SUMMARY===\n")
	prompt.WriteString("AI development tools reached a turning point this week with three simultaneous breakthroughs in agent autonomy. The shift: from AI-as-helper to AI-as-autonomous-developer.\n\n")
	prompt.WriteString("Anthropic released Claude Code web platform where developers assign tasks and agents work independently across repositories [See #1]. Meanwhile, Claude gained persistent memory for teams, eliminating context re-explanation and enabling true project continuity [See #2]. Practitioners are discovering optimal workflows running 8+ agents simultaneously with atomic git commits and blast-radius management [See #5].\n\n")
	prompt.WriteString("Agent autonomy is production-ready, but success requires new workflows built around parallel execution and granular task isolation. The challenge shifts from capability to workflow design.\n\n")

	prompt.WriteString("Now generate the digest content following this EXACT structure with four sections (TITLE, TLDR, KEY_MOMENTS, SUMMARY):")

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

// parseDigestContent extracts title, TL;DR, and summary from LLM response
func (g *Generator) parseDigestContent(response string, insights []ClusterInsight) DigestContent {
	content := DigestContent{}

	// Parse structured response with ===MARKERS===
	parts := strings.Split(response, "===")

	for i := 0; i < len(parts)-1; i++ {
		sectionName := strings.TrimSpace(parts[i])
		if i+1 >= len(parts) {
			continue
		}
		sectionContent := parts[i+1]

		// Remove next marker if present
		if idx := strings.Index(sectionContent, "==="); idx > 0 {
			sectionContent = sectionContent[:idx]
		}
		sectionContent = strings.TrimSpace(sectionContent)

		switch sectionName {
		case "TITLE":
			content.Title = sectionContent
		case "TLDR":
			content.TLDRSummary = sectionContent
		case "KEY_MOMENTS":
			content.KeyMoments = g.parseKeyMoments(sectionContent)
		case "SUMMARY":
			content.ExecutiveSummary = sectionContent
		}
	}

	// Fallback: if parsing failed, treat entire response as summary
	if content.ExecutiveSummary == "" {
		content.ExecutiveSummary = strings.TrimSpace(response)
	}

	// Generate fallbacks if any field is missing
	if content.Title == "" {
		content.Title = g.generateFallbackTitle(insights)
	}
	if content.TLDRSummary == "" {
		content.TLDRSummary = g.generateFallbackTLDR(insights)
	}
	if len(content.KeyMoments) == 0 {
		content.KeyMoments = g.generateFallbackKeyMoments(insights)
	}

	return content
}

// parseKeyMoments extracts key moments from numbered list format
func (g *Generator) parseKeyMoments(content string) []string {
	lines := strings.Split(content, "\n")
	moments := make([]string, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if line starts with number followed by dot
		if len(line) > 2 && line[0] >= '1' && line[0] <= '9' && line[1] == '.' {
			// Extract the content after "N. "
			moment := strings.TrimSpace(line[2:])
			if moment != "" {
				moments = append(moments, moment)
			}
		}
	}

	return moments
}

// generateFallbackContent creates simple content when LLM fails
func (g *Generator) generateFallbackContent(insights []ClusterInsight) DigestContent {
	return DigestContent{
		Title:            g.generateFallbackTitle(insights),
		TLDRSummary:      g.generateFallbackTLDR(insights),
		KeyMoments:       g.generateFallbackKeyMoments(insights),
		ExecutiveSummary: g.generateFallbackNarrative(insights),
	}
}

// generateFallbackTitle creates a simple title from cluster themes
func (g *Generator) generateFallbackTitle(insights []ClusterInsight) string {
	if len(insights) == 0 {
		return "Weekly Tech Digest"
	}

	// Take first 2-3 cluster themes
	themes := make([]string, 0, 3)
	for i, insight := range insights {
		if i >= 3 {
			break
		}
		themes = append(themes, insight.Theme)
	}

	return strings.Join(themes, " & ")
}

// generateFallbackTLDR creates a simple one-line summary
func (g *Generator) generateFallbackTLDR(insights []ClusterInsight) string {
	if len(insights) == 0 {
		return "This week's tech news digest"
	}

	themes := make([]string, 0, len(insights))
	for _, insight := range insights {
		themes = append(themes, strings.ToLower(insight.Theme))
	}

	return fmt.Sprintf("This week covers %s across %d key topics",
		joinWithAnd(themes), len(insights))
}

// generateFallbackKeyMoments creates simple key moments from cluster insights
func (g *Generator) generateFallbackKeyMoments(insights []ClusterInsight) []string {
	moments := make([]string, 0)

	// Take top article from each cluster (up to 5 moments)
	for i, insight := range insights {
		if i >= 5 {
			break
		}
		if len(insight.TopArticles) > 0 {
			article := insight.TopArticles[0]
			moment := fmt.Sprintf("**%s** %s", insight.Theme, article.Title)
			moments = append(moments, moment)
		}
	}

	// If no moments generated, provide generic fallback
	if len(moments) == 0 {
		moments = append(moments, "Multiple tech developments this week")
	}

	return moments
}