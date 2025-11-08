package narrative

import (
	"briefly/internal/core"
	"briefly/internal/llm"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/generative-ai-go/genai"
)

// LLMClient defines the interface for LLM operations needed by the narrative generator
type LLMClient interface {
	// GenerateText generates text from a prompt with optional structured output
	GenerateText(ctx context.Context, prompt string, options llm.TextGenerationOptions) (string, error)

	// GetGenaiModel returns the underlying genai model for schema definition
	GetGenaiModel() *genai.GenerativeModel
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

// DigestContent contains all generated content for a digest (v2.0 structured format)
type DigestContent struct {
	Title            string             `json:"title"`             // Generated title (25-45 chars ideal, 50 max)
	TLDRSummary      string             `json:"tldr_summary"`      // One-sentence summary (40-80 chars ideal, 100 max)
	KeyMoments       []core.KeyMoment   `json:"key_moments"`       // 3-5 key developments with structured quotes and citations
	Perspectives     []core.Perspective `json:"perspectives"`      // Supporting/opposing viewpoints (optional)
	ExecutiveSummary string             `json:"executive_summary"` // Full narrative summary with [N] citation placeholders
}

// GenerateClusterSummary generates a comprehensive narrative for a single cluster using ALL articles
// This implements hierarchical summarization: cluster summary → executive summary
func (g *Generator) GenerateClusterSummary(ctx context.Context, cluster core.TopicCluster, articles map[string]core.Article, summaries map[string]core.Summary) (*core.ClusterNarrative, error) {
	// Collect all articles in this cluster
	clusterArticles := make([]ArticleSummary, 0, len(cluster.ArticleIDs))

	for _, articleID := range cluster.ArticleIDs {
		article, hasArticle := articles[articleID]
		summary, hasSummary := summaries[articleID]

		if !hasArticle || !hasSummary {
			continue
		}

		clusterArticles = append(clusterArticles, ArticleSummary{
			Title:     article.Title,
			URL:       article.URL,
			Summary:   summary.SummaryText,
			KeyPoints: g.extractKeyPoints(summary),
		})
	}

	if len(clusterArticles) == 0 {
		return nil, fmt.Errorf("no articles found for cluster %s", cluster.Label)
	}

	// Generate cluster narrative using LLM
	prompt := g.buildClusterSummaryPrompt(cluster.Label, clusterArticles)
	schema := g.buildClusterNarrativeSchema()

	response, err := g.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		ResponseSchema: schema,
		Temperature:    0.7,
		MaxTokens:      1500,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate cluster summary: %w", err)
	}

	// Parse JSON response
	narrative, err := g.parseClusterNarrative(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cluster narrative: %w", err)
	}

	return narrative, nil
}

// GenerateDigestContentWithCritique generates digest content with self-critique refinement pass
// This is the NEW recommended entry point that ensures quality through critique
func (g *Generator) GenerateDigestContentWithCritique(ctx context.Context, clusters []core.TopicCluster, articles map[string]core.Article, summaries map[string]core.Summary, config CritiqueConfig) (*DigestContent, error) {
	// Step 1: Generate initial draft
	fmt.Println("   📝 Generating initial digest draft...")
	draftDigest, err := g.GenerateDigestContent(ctx, clusters, articles, summaries)
	if err != nil {
		return nil, fmt.Errorf("draft generation failed: %w", err)
	}

	// Step 2: Determine if critique pass should run
	if !g.ShouldRunCritique(draftDigest, config) {
		fmt.Println("   ℹ️  Skipping critique pass (not required)")
		return draftDigest, nil
	}

	// Step 3: Run self-critique with retry logic
	fmt.Println("   🔍 Running self-critique refinement pass...")

	var finalDigest *DigestContent
	var critiqueResult *CritiqueResult

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			fmt.Printf("   🔄 Retry attempt %d/%d...\n", attempt, config.MaxRetries)
		}

		// Run critique
		critiqueResult, err = g.RefineDigestWithCritique(ctx, draftDigest, clusters, articles, summaries)
		if err != nil {
			fmt.Printf("   ⚠️  Critique failed: %v\n", err)

			// If this is the last retry, return draft
			if attempt == config.MaxRetries {
				fmt.Println("   ⚠️  Max retries reached, using draft digest")
				return draftDigest, nil
			}

			continue // Retry
		}

		// Check if quality improved
		if critiqueResult.QualityImproved {
			fmt.Println("   ✓ Quality improved by critique pass")

			// Print critique summary
			if len(critiqueResult.Critique.ArticlesMissing) > 0 {
				fmt.Printf("      • Fixed missing articles: %v\n", critiqueResult.Critique.ArticlesMissing)
			}
			if len(critiqueResult.Critique.VaguePhrases) > 0 {
				fmt.Printf("      • Fixed vague phrases: %v\n", critiqueResult.Critique.VaguePhrases)
			}
			if critiqueResult.Critique.SpecificityScore > 0 {
				fmt.Printf("      • Specificity score: %d/100\n", critiqueResult.Critique.SpecificityScore)
			}

			finalDigest = critiqueResult.ImprovedDigest
			break
		}

		// Quality not improved, retry if attempts remaining
		if attempt < config.MaxRetries {
			fmt.Println("   ⚠️  Quality not improved, retrying...")
			continue
		}

		// Max retries reached with no improvement
		fmt.Println("   ⚠️  Quality not improved after retries, using draft")
		finalDigest = draftDigest
	}

	if finalDigest == nil {
		// Fallback to draft if something went wrong
		finalDigest = draftDigest
	}

	return finalDigest, nil
}

// GenerateDigestContent creates title, TL;DR, and executive summary from clustered articles
// Returns structured content with all three components
// NEW: If clusters have narratives (hierarchical summarization), uses those instead of top-3 articles
// NOTE: This is the base generation function. Use GenerateDigestContentWithCritique for quality-assured generation.
func (g *Generator) GenerateDigestContent(ctx context.Context, clusters []core.TopicCluster, articles map[string]core.Article, summaries map[string]core.Summary) (*DigestContent, error) {
	if len(clusters) == 0 {
		return nil, fmt.Errorf("no clusters provided")
	}

	// Check if we have cluster narratives (hierarchical summarization)
	hasNarratives := false
	for _, cluster := range clusters {
		if cluster.Narrative != nil {
			hasNarratives = true
			break
		}
	}

	var prompt string
	schema := g.buildDigestContentSchema()

	if hasNarratives {
		// NEW: Use hierarchical summarization with cluster narratives
		prompt = g.buildNarrativePromptFromClusters(clusters, articles, summaries)
		fmt.Println("   ✓ Using hierarchical summarization (cluster narratives)")
	} else {
		// LEGACY: Fall back to top-3 article approach
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

		prompt = g.buildStructuredNarrativePrompt(clusterInsights)
		fmt.Println("   ⚠️  Using legacy top-3 article summarization")
	}

	// Generate content using LLM with structured output (v2.0)
	response, err := g.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		ResponseSchema: schema,
		Temperature:    0.7,
		MaxTokens:      2000,
	})
	if err != nil {
		// Fallback to simple generation if LLM fails
		// Build fallback from cluster insights (legacy)
		clusterInsights := make([]ClusterInsight, 0, len(clusters))
		for _, cluster := range clusters {
			insight, _ := g.extractClusterInsight(cluster, articles, summaries)
			clusterInsights = append(clusterInsights, insight)
		}
		content := g.generateFallbackContent(clusterInsights)
		return &content, nil
	}

	// Parse JSON response to DigestContent
	content, err := g.parseStructuredDigestContent(response)
	if err != nil {
		// Fallback if parsing fails
		clusterInsights := make([]ClusterInsight, 0, len(clusters))
		for _, cluster := range clusters {
			insight, _ := g.extractClusterInsight(cluster, articles, summaries)
			clusterInsights = append(clusterInsights, insight)
		}
		fallback := g.generateFallbackContent(clusterInsights)
		return &fallback, nil
	}

	return content, nil
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
	Theme        string
	TopArticles  []ArticleSummary
	KeyThemes    []string
	ArticleCount int
}

// ArticleSummary contains essential article information for narrative generation
type ArticleSummary struct {
	Title     string
	URL       string
	Summary   string
	KeyPoints []string
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
//
//nolint:unused
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

	theme, err := g.llmClient.GenerateText(ctx, prompt.String(), llm.TextGenerationOptions{})
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
//
//nolint:unused
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
			// Convert old string format to structured format
			oldMoments := g.parseKeyMoments(sectionContent)
			content.KeyMoments = g.convertOldMomentsToStructured(oldMoments)
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

// parseKeyMoments extracts key moments from numbered list format (DEPRECATED: use structured output)
//
//nolint:unused
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

// convertOldMomentsToStructured converts old []string key moments to structured format
//
//nolint:unused
func (g *Generator) convertOldMomentsToStructured(oldMoments []string) []core.KeyMoment {
	structured := make([]core.KeyMoment, 0, len(oldMoments))

	for i, moment := range oldMoments {
		// Try to extract citation number from moment text like "[See #1]"
		citationNum := i + 1 // Default to sequential numbering

		// Extract quote text (remove citation references if present)
		quote := moment
		if idx := strings.Index(moment, "[See #"); idx > 0 {
			quote = strings.TrimSpace(moment[:idx])
		}

		structured = append(structured, core.KeyMoment{
			Quote:          quote,
			CitationNumber: citationNum,
		})
	}

	return structured
}

// ============================================================================
// Hierarchical Summarization Functions (Cluster-level)
// ============================================================================

// buildClusterSummaryPrompt creates a prompt for generating cluster narrative from ALL articles
func (g *Generator) buildClusterSummaryPrompt(clusterLabel string, articles []ArticleSummary) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("Generate a cohesive narrative for the \"%s\" topic cluster.\n\n", clusterLabel))

	prompt.WriteString("**ALL Articles in this cluster:**\n")
	for i, article := range articles {
		prompt.WriteString(fmt.Sprintf("\n[%d] %s\n", i+1, article.Title))
		prompt.WriteString(fmt.Sprintf("    URL: %s\n", article.URL))
		prompt.WriteString(fmt.Sprintf("    Summary: %s\n", article.Summary))
		if len(article.KeyPoints) > 0 {
			prompt.WriteString("    Key Points:\n")
			for _, point := range article.KeyPoints {
				prompt.WriteString(fmt.Sprintf("    - %s\n", point))
			}
		}
	}

	prompt.WriteString("\n**TASK:**\n")
	prompt.WriteString("Synthesize ALL articles above into a cohesive 2-3 paragraph narrative that:\n")
	prompt.WriteString("1. Identifies the common themes and patterns across articles\n")
	prompt.WriteString("2. Shows how the articles relate to each other (complementary, contrasting, building on each other)\n")
	prompt.WriteString("3. Extracts the key insights that matter to technical readers\n")
	prompt.WriteString("4. Maintains accuracy - don't invent information not in the articles\n")
	prompt.WriteString("5. Uses SPECIFIC facts, numbers, names, and dates from the articles\n\n")

	prompt.WriteString("**CRITICAL SPECIFICITY RULES:**\n")
	prompt.WriteString("❌ BANNED VAGUE PHRASES - Never use:\n")
	prompt.WriteString("   \"several\", \"various\", \"multiple\", \"many\", \"some\", \"a few\", \"numerous\"\n")
	prompt.WriteString("   \"recently\", \"soon\", \"significant\", \"substantial\"\n\n")

	prompt.WriteString("✅ REQUIRED SPECIFICITY - Must include:\n")
	prompt.WriteString("   • At least 3 specific numbers/percentages/metrics\n")
	prompt.WriteString("   • At least 5 specific proper nouns (companies, people, products)\n")
	prompt.WriteString("   • Exact dates or specific timeframes when mentioned\n")
	prompt.WriteString("   • Citations for EVERY claim: \"Company X announced Y [1]\" not \"A company announced something\"\n\n")

	prompt.WriteString("✅ EXAMPLES:\n")
	prompt.WriteString("   WRONG: \"Several companies announced updates\"\n")
	prompt.WriteString("   RIGHT: \"Google [1], Meta [2], and Anthropic [3] announced updates\"\n\n")

	prompt.WriteString("   WRONG: \"Performance improved significantly\"\n")
	prompt.WriteString("   RIGHT: \"Inference speed increased 40% from 3.5s to 2.1s [2]\"\n\n")

	prompt.WriteString("**VERIFICATION BEFORE FINALIZING:**\n")
	prompt.WriteString("Before returning, verify:\n")
	prompt.WriteString(fmt.Sprintf("1. All %d articles are cited at least once in article_refs array\n", len(articles)))
	prompt.WriteString("2. No banned vague phrases appear in the summary\n")
	prompt.WriteString("3. At least 3 specific numbers/metrics in summary\n")
	prompt.WriteString("4. At least 5 proper nouns (companies/people/products)\n")
	prompt.WriteString("If any check fails, revise the summary until all pass.\n\n")

	prompt.WriteString("**OUTPUT REQUIREMENTS:**\n")
	prompt.WriteString("- Title: Short, punchy cluster title (5-8 words)\n")
	prompt.WriteString("- Summary: 2-3 paragraph narrative synthesizing all articles (150-250 words)\n")
	prompt.WriteString(fmt.Sprintf("  * MUST include specific facts from ALL %d articles\n", len(articles)))
	prompt.WriteString("  * MUST cite sources: [1], [2], [3] etc.\n")
	prompt.WriteString("  * NO vague/generic phrases\n")
	prompt.WriteString("- Key Themes: 3-5 main themes from the cluster\n")
	prompt.WriteString(fmt.Sprintf("- Article Refs: Citation numbers of ALL %d articles (1-based array)\n", len(articles)))
	prompt.WriteString("- Confidence: How coherent this cluster is (0.0-1.0)\n\n")

	prompt.WriteString("Generate the cluster narrative in JSON format matching the schema.\n")

	return prompt.String()
}

// buildClusterNarrativeSchema defines the Gemini JSON schema for cluster narrative
func (g *Generator) buildClusterNarrativeSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"title": {
				Type:        genai.TypeString,
				Description: "Short, punchy cluster title (5-8 words)",
			},
			"summary": {
				Type:        genai.TypeString,
				Description: "2-3 paragraph narrative synthesizing all articles (150-250 words)",
			},
			"key_themes": {
				Type:        genai.TypeArray,
				Description: "3-5 main themes from the cluster",
				Items: &genai.Schema{
					Type: genai.TypeString,
				},
			},
			"article_refs": {
				Type:        genai.TypeArray,
				Description: "Citation numbers of articles included (1-based)",
				Items: &genai.Schema{
					Type: genai.TypeInteger,
				},
			},
			"confidence": {
				Type:        genai.TypeNumber,
				Description: "Cluster coherence confidence (0.0-1.0)",
			},
		},
		Required: []string{"title", "summary", "key_themes", "article_refs", "confidence"},
	}
}

// parseClusterNarrative parses JSON response into ClusterNarrative
func (g *Generator) parseClusterNarrative(jsonResponse string) (*core.ClusterNarrative, error) {
	var response struct {
		Title       string   `json:"title"`
		Summary     string   `json:"summary"`
		KeyThemes   []string `json:"key_themes"`
		ArticleRefs []int    `json:"article_refs"`
		Confidence  float64  `json:"confidence"`
	}

	err := json.Unmarshal([]byte(jsonResponse), &response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &core.ClusterNarrative{
		Title:       response.Title,
		Summary:     response.Summary,
		KeyThemes:   response.KeyThemes,
		ArticleRefs: response.ArticleRefs,
		Confidence:  response.Confidence,
	}, nil
}

// ============================================================================
// v2.0 Structured Output Functions
// ============================================================================

// buildNarrativePromptFromClusters creates prompt using cluster narratives (hierarchical summarization)
// This is the NEW approach that synthesizes from cluster-level summaries
func (g *Generator) buildNarrativePromptFromClusters(clusters []core.TopicCluster, articles map[string]core.Article, summaries map[string]core.Summary) string {
	var prompt strings.Builder

	prompt.WriteString("Generate structured content for a technical digest using hierarchical summarization.\n\n")

	// Build cluster narrative list
	prompt.WriteString("**Cluster Narratives (synthesized from all articles in each cluster):**\n\n")

	for i, cluster := range clusters {
		if cluster.Narrative == nil {
			continue
		}

		prompt.WriteString(fmt.Sprintf("## Cluster %d: %s\n", i+1, cluster.Narrative.Title))
		prompt.WriteString(fmt.Sprintf("**Theme:** %s\n", cluster.Label))
		prompt.WriteString(fmt.Sprintf("**Key Themes:** %s\n", strings.Join(cluster.Narrative.KeyThemes, ", ")))
		prompt.WriteString(fmt.Sprintf("**Articles Covered:** %d\n\n", len(cluster.ArticleIDs)))
		prompt.WriteString("**Cluster Summary:**\n")
		prompt.WriteString(cluster.Narrative.Summary)
		prompt.WriteString("\n\n---\n\n")
	}

	// Add article reference list for citations
	prompt.WriteString("**All Articles (for citation references):**\n")
	articleNum := 1
	for _, cluster := range clusters {
		for _, articleID := range cluster.ArticleIDs {
			if article, found := articles[articleID]; found {
				prompt.WriteString(fmt.Sprintf("[%d] %s\n", articleNum, article.Title))
				prompt.WriteString(fmt.Sprintf("    URL: %s\n\n", article.URL))
				articleNum++
			}
		}
	}

	prompt.WriteString("\n**REQUIREMENTS:**\n\n")

	prompt.WriteString("**Title (20-40 characters STRICT MAXIMUM):**\n")
	prompt.WriteString("- MUST use active voice with strong action verbs\n")
	prompt.WriteString("- REQUIRED: Include specific actor (company/tech/concept)\n")
	prompt.WriteString("- BANNED generic verbs: \"updates\", \"changes\", \"announces\", \"releases\"\n")
	prompt.WriteString("- PREFERRED power verbs: \"cuts\", \"hits\", \"beats\", \"breaks\", \"surges\", \"doubles\", \"slashes\"\n")
	prompt.WriteString("- CRITICAL: ABSOLUTE MAXIMUM 40 characters\n")
	prompt.WriteString("- Examples:\n")
	prompt.WriteString("  ✓ \"Voice AI Hits 1-Second Latency\" (32 chars) - specific metric\n")
	prompt.WriteString("  ✓ \"GPT-5 Cuts Inference Cost 60%\" (31 chars) - quantified impact\n")
	prompt.WriteString("  ✗ \"New AI Model Updates Released\" (31 chars) - generic, passive\n\n")

	prompt.WriteString("**TLDR Summary (40-75 characters STRICT MAXIMUM):**\n")
	prompt.WriteString("- REQUIRED STRUCTURE: [Subject] + [Action Verb] + [Object] + [Quantified Impact]\n")
	prompt.WriteString("- Subject: Specific company/technology (e.g., \"OpenAI\", \"Voice AI\")\n")
	prompt.WriteString("- Action Verb: Strong active verb (e.g., \"cuts\", \"achieves\", \"beats\")\n")
	prompt.WriteString("- Object: What was changed (e.g., \"inference speed\", \"latency\")\n")
	prompt.WriteString("- Impact: Numeric result (e.g., \"by 40%\", \"to 1 second\", \"95% accuracy\")\n")
	prompt.WriteString("- CRITICAL: ABSOLUTE MAXIMUM 75 characters\n")
	prompt.WriteString("- Examples:\n")
	prompt.WriteString("  ✓ \"Perplexity hits 400 Gbps for distributed AI inference\" (56 chars)\n")
	prompt.WriteString("  ✓ \"Voice AI achieves 1-second latency with open models\" (54 chars)\n")
	prompt.WriteString("  ✗ \"AI technology sees improvements in performance\" (49 chars) - vague, no metrics\n\n")

	prompt.WriteString("**Executive Summary (2-3 paragraphs):**\n")
	prompt.WriteString("- REQUIRED: Explicitly connect clusters with transition phrases:\n")
	prompt.WriteString("  \"Building on...\", \"In contrast to...\", \"Supporting this trend...\", \"Meanwhile...\"\n")
	prompt.WriteString("- Paragraph 1: Main narrative thread connecting 2-3 largest clusters\n")
	prompt.WriteString("- Paragraph 2: Secondary trends and how they relate to the main thread\n")
	prompt.WriteString("- Paragraph 3 (optional): Implications or contrasts (e.g., technical vs creative domains)\n")
	prompt.WriteString("- Include citations using [1], [2], [3] format (CRITICAL: use numbers only)\n")
	prompt.WriteString("- Focus on 'why it matters' not just 'what happened'\n")
	prompt.WriteString("- Write for developers, PMs, and technical leaders\n")
	prompt.WriteString("- 150-200 words total\n\n")

	prompt.WriteString("**CRITICAL SPECIFICITY RULES:**\n")
	prompt.WriteString("❌ BANNED VAGUE PHRASES - Never use:\n")
	prompt.WriteString("   \"several\", \"various\", \"multiple\", \"many\", \"some\", \"numerous\"\n")
	prompt.WriteString("   \"recently\", \"significant\", \"substantial\"\n\n")

	prompt.WriteString("✅ REQUIRED SPECIFICITY:\n")
	prompt.WriteString(fmt.Sprintf("   • Cite EVERY article at least once (you have %d articles total)\n", articleNum-1))
	prompt.WriteString("   • Include at least 5 specific facts with citations\n")
	prompt.WriteString("   • Use exact numbers/percentages from cluster narratives\n")
	prompt.WriteString("   • Name specific companies, products, people\n\n")

	prompt.WriteString("**SELF-VALIDATION CHECKLIST (verify before finalizing):**\n")
	prompt.WriteString("Before returning JSON, verify ALL these conditions:\n")
	prompt.WriteString("1. ✓ Title ≤ 40 characters AND uses active power verb (not \"updates\", \"announces\")\n")
	prompt.WriteString("2. ✓ Title includes specific actor (company/tech) + quantified result\n")
	prompt.WriteString("3. ✓ TLDR ≤ 75 characters AND follows [Subject]+[Verb]+[Object]+[Impact] structure\n")
	prompt.WriteString("4. ✓ TLDR contains at least one specific number/percentage\n")
	prompt.WriteString(fmt.Sprintf("5. ✓ All %d articles cited at least once in executive summary\n", articleNum-1))
	prompt.WriteString("6. ✓ Executive summary has clear cluster connections (\"Building on...\", \"Meanwhile...\")\n")
	prompt.WriteString("7. ✓ No banned vague phrases in executive summary or key moments\n")
	prompt.WriteString("8. ✓ At least 5 specific facts with citations [1][2][3]\n")
	prompt.WriteString("9. ✓ At least 3 specific numbers/percentages/metrics\n")
	prompt.WriteString("10. ✓ At least 5 proper nouns (companies/people/products)\n")
	prompt.WriteString("11. ✓ Key moments have diversity (at least one from each major cluster)\n\n")

	prompt.WriteString("**IF ANY CHECK FAILS:** Revise the content until ALL checks pass.\n")
	prompt.WriteString("Do NOT return the JSON until all 11 validation checks are satisfied.\n\n")

	prompt.WriteString("**Key Moments (3-5 structured quotes):**\n")
	prompt.WriteString("- REQUIRED: At least one quote from each major cluster (clusters with 3+ articles)\n")
	prompt.WriteString("- DIVERSITY REQUIREMENT: No more than 2 quotes from the same cluster\n")
	prompt.WriteString("- PRIORITY ORDER:\n")
	prompt.WriteString("  1. Quantified achievements (e.g., \"400 Gbps throughput\", \"98.5% accuracy\")\n")
	prompt.WriteString("  2. Major product launches or acquisitions\n")
	prompt.WriteString("  3. Funding announcements with specific amounts\n")
	prompt.WriteString("  4. Technical breakthroughs with measurable impacts\n")
	prompt.WriteString("- Each must have:\n")
	prompt.WriteString("  - quote: Important insight or development (1-2 sentences, MUST include specific numbers)\n")
	prompt.WriteString("  - citation_number: Reference to article [1-N]\n")
	prompt.WriteString("- Example: {\"quote\": \"Perplexity's TransferEngine hit 400 Gbps for distributed inference\", \"citation_number\": 1}\n\n")

	prompt.WriteString("**Perspectives (optional, 0-3 viewpoints):**\n")
	prompt.WriteString("- Identify supporting or opposing viewpoints if present\n")
	prompt.WriteString("- Each must have:\n")
	prompt.WriteString("  - type: \"supporting\" or \"opposing\"\n")
	prompt.WriteString("  - summary: Summary of this perspective (1-2 sentences)\n")
	prompt.WriteString("  - citation_numbers: Array of article numbers [1,2,3]\n\n")

	prompt.WriteString("Generate the digest content in JSON format matching the schema.\n")

	return prompt.String()
}

// buildStructuredNarrativePrompt creates a prompt for generating structured digest content with JSON schema
func (g *Generator) buildStructuredNarrativePrompt(insights []ClusterInsight) string {
	var prompt strings.Builder

	prompt.WriteString("Generate structured content for a technical digest following v2.0 architecture.\n\n")

	// Build article reference list with citation numbers
	prompt.WriteString("**Articles for reference:**\n")
	articleNum := 1
	for _, insight := range insights {
		for _, article := range insight.TopArticles {
			prompt.WriteString(fmt.Sprintf("[%d] %s\n", articleNum, article.Title))
			prompt.WriteString(fmt.Sprintf("    URL: %s\n", article.URL))
			prompt.WriteString(fmt.Sprintf("    Summary: %s\n\n", truncateText(article.Summary, 200)))
			articleNum++
		}
	}

	prompt.WriteString("\n**REQUIREMENTS:**\n\n")

	prompt.WriteString("**Title (20-40 characters STRICT MAXIMUM):**\n")
	prompt.WriteString("- MUST use active voice with strong action verbs\n")
	prompt.WriteString("- REQUIRED: Include specific actor (company/tech/concept)\n")
	prompt.WriteString("- BANNED generic verbs: \"updates\", \"changes\", \"announces\", \"releases\"\n")
	prompt.WriteString("- PREFERRED power verbs: \"cuts\", \"hits\", \"beats\", \"breaks\", \"surges\", \"doubles\", \"slashes\"\n")
	prompt.WriteString("- CRITICAL: ABSOLUTE MAXIMUM 40 characters (database hard limit: 50 chars)\n")
	prompt.WriteString("- Examples:\n")
	prompt.WriteString("  ✓ \"Voice AI Hits 1-Second Latency\" (32 chars) - specific metric\n")
	prompt.WriteString("  ✓ \"GPT-5 Cuts Inference Cost 60%\" (31 chars) - quantified impact\n")
	prompt.WriteString("  ✗ \"New AI Model Updates Released\" (31 chars) - generic, passive\n\n")

	prompt.WriteString("**TLDR Summary (40-75 characters STRICT MAXIMUM):**\n")
	prompt.WriteString("- REQUIRED STRUCTURE: [Subject] + [Action Verb] + [Object] + [Quantified Impact]\n")
	prompt.WriteString("- Subject: Specific company/technology (e.g., \"OpenAI\", \"Voice AI\")\n")
	prompt.WriteString("- Action Verb: Strong active verb (e.g., \"cuts\", \"achieves\", \"beats\")\n")
	prompt.WriteString("- Object: What was changed (e.g., \"inference speed\", \"latency\")\n")
	prompt.WriteString("- Impact: Numeric result (e.g., \"by 40%\", \"to 1 second\", \"95% accuracy\")\n")
	prompt.WriteString("- CRITICAL: ABSOLUTE MAXIMUM 75 characters (database hard limit: 100 chars)\n")
	prompt.WriteString("- Examples:\n")
	prompt.WriteString("  ✓ \"Perplexity hits 400 Gbps for distributed AI inference\" (56 chars)\n")
	prompt.WriteString("  ✓ \"Voice AI achieves 1-second latency with open models\" (54 chars)\n")
	prompt.WriteString("  ✗ \"AI technology sees improvements in performance\" (49 chars) - vague, no metrics\n\n")

	prompt.WriteString("**Executive Summary (2-3 paragraphs):**\n")
	prompt.WriteString("- Tell the story of this week's developments\n")
	prompt.WriteString("- Include citations using [1], [2], [3] format (CRITICAL: use numbers only)\n")
	prompt.WriteString("- Focus on 'why it matters' not just 'what happened'\n")
	prompt.WriteString("- Write for developers, PMs, and technical leaders\n")
	prompt.WriteString("- 150-200 words total\n\n")

	prompt.WriteString("**Key Moments (3-5 structured quotes):**\n")
	prompt.WriteString("- PRIORITY ORDER:\n")
	prompt.WriteString("  1. Quantified achievements (e.g., \"400 Gbps throughput\", \"98.5% accuracy\")\n")
	prompt.WriteString("  2. Major product launches or acquisitions\n")
	prompt.WriteString("  3. Funding announcements with specific amounts\n")
	prompt.WriteString("  4. Technical breakthroughs with measurable impacts\n")
	prompt.WriteString("- Each must have:\n")
	prompt.WriteString("  - quote: Exact quote from article (1-2 sentences, MUST include specific numbers)\n")
	prompt.WriteString("  - citation_number: Reference to article [1-N]\n")
	prompt.WriteString("- Examples:\n")
	prompt.WriteString("  ✓ {\"quote\": \"GPT-5 achieves 95% on MMLU benchmarks\", \"citation_number\": 1}\n")
	prompt.WriteString("  ✓ {\"quote\": \"Early testing shows 40% cost reduction\", \"citation_number\": 3}\n")
	prompt.WriteString("  ✗ {\"quote\": \"Performance improved significantly\", \"citation_number\": 2} - no numbers\n\n")

	prompt.WriteString("**Perspectives (optional, 0-3 viewpoints):**\n")
	prompt.WriteString("- Identify supporting or opposing viewpoints if present\n")
	prompt.WriteString("- Each must have:\n")
	prompt.WriteString("  - type: \"supporting\" or \"opposing\"\n")
	prompt.WriteString("  - summary: Summary of this perspective (1-2 sentences)\n")
	prompt.WriteString("  - citation_numbers: Array of article numbers [1,2,3]\n")
	prompt.WriteString("- Only include if there are clear different perspectives\n\n")

	prompt.WriteString("Generate the digest content in JSON format matching the schema.\n")

	return prompt.String()
}

// buildDigestContentSchema defines the Gemini JSON schema for digest content
func (g *Generator) buildDigestContentSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"title": {
				Type:        genai.TypeString,
				Description: "Catchy headline (25-45 chars MAXIMUM - hard limit 50)",
			},
			"tldr_summary": {
				Type:        genai.TypeString,
				Description: "One-sentence summary (40-80 chars MAXIMUM - hard limit 100)",
			},
			"executive_summary": {
				Type:        genai.TypeString,
				Description: "2-3 paragraph story with [1][2][3] citations (150-200 words)",
			},
			"key_moments": {
				Type:        genai.TypeArray,
				Description: "3-5 important quotes with citations",
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"quote": {
							Type:        genai.TypeString,
							Description: "Exact quote from article (1-2 sentences)",
						},
						"citation_number": {
							Type:        genai.TypeInteger,
							Description: "Reference to article [1-N]",
						},
					},
					Required: []string{"quote", "citation_number"},
				},
			},
			"perspectives": {
				Type:        genai.TypeArray,
				Description: "Supporting/opposing viewpoints (optional)",
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"type": {
							Type:        genai.TypeString,
							Description: "supporting or opposing",
						},
						"summary": {
							Type:        genai.TypeString,
							Description: "Summary of this perspective (1-2 sentences)",
						},
						"citation_numbers": {
							Type:        genai.TypeArray,
							Description: "Array of article numbers",
							Items: &genai.Schema{
								Type: genai.TypeInteger,
							},
						},
					},
					Required: []string{"type", "summary", "citation_numbers"},
				},
			},
		},
		Required: []string{"title", "tldr_summary", "executive_summary", "key_moments"},
	}
}

// parseStructuredDigestContent parses JSON response from LLM into DigestContent
func (g *Generator) parseStructuredDigestContent(jsonResponse string) (*DigestContent, error) {
	// Define a temporary struct matching the JSON schema
	var response struct {
		Title            string `json:"title"`
		TLDRSummary      string `json:"tldr_summary"`
		ExecutiveSummary string `json:"executive_summary"`
		KeyMoments       []struct {
			Quote          string `json:"quote"`
			CitationNumber int    `json:"citation_number"`
		} `json:"key_moments"`
		Perspectives []struct {
			Type            string `json:"type"`
			Summary         string `json:"summary"`
			CitationNumbers []int  `json:"citation_numbers"`
		} `json:"perspectives"`
	}

	// Parse JSON
	err := json.Unmarshal([]byte(jsonResponse), &response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Convert to DigestContent
	content := &DigestContent{
		Title:            response.Title,
		TLDRSummary:      response.TLDRSummary,
		ExecutiveSummary: response.ExecutiveSummary,
		KeyMoments:       make([]core.KeyMoment, 0, len(response.KeyMoments)),
		Perspectives:     make([]core.Perspective, 0, len(response.Perspectives)),
	}

	// Convert key moments
	for _, km := range response.KeyMoments {
		content.KeyMoments = append(content.KeyMoments, core.KeyMoment{
			Quote:          km.Quote,
			CitationNumber: km.CitationNumber,
		})
	}

	// Convert perspectives
	for _, p := range response.Perspectives {
		content.Perspectives = append(content.Perspectives, core.Perspective{
			Type:            p.Type,
			Summary:         p.Summary,
			CitationNumbers: p.CitationNumbers,
		})
	}

	return content, nil
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

// generateFallbackKeyMoments creates simple structured key moments from cluster insights (v2.0)
func (g *Generator) generateFallbackKeyMoments(insights []ClusterInsight) []core.KeyMoment {
	moments := make([]core.KeyMoment, 0)

	// Take top article from each cluster (up to 5 moments)
	citationNum := 1
	for i, insight := range insights {
		if i >= 5 {
			break
		}
		if len(insight.TopArticles) > 0 {
			article := insight.TopArticles[0]
			// Extract first sentence from summary as quote
			quote := extractFirstSentence(article.Summary)
			if quote == "" {
				quote = article.Title
			}

			moments = append(moments, core.KeyMoment{
				Quote:          quote,
				CitationNumber: citationNum,
			})
			citationNum++
		}
	}

	// If no moments generated, provide generic fallback
	if len(moments) == 0 {
		moments = append(moments, core.KeyMoment{
			Quote:          "Multiple tech developments this week",
			CitationNumber: 1,
		})
	}

	return moments
}
