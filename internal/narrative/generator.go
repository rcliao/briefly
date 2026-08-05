// Package narrative generates the Slack-format digest content.
// (The hierarchical cluster-narrative machinery this package once held was
// removed with the database pipeline; only the Slack slice is live.)
package narrative

import (
	"briefly/internal/core"
	"briefly/internal/llm"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"google.golang.org/genai"
)

func cleanJSONResponse(response string) string {
	cleaned := strings.TrimSpace(response)

	// Remove markdown code blocks (```json ... ``` or ``` ... ```)
	if strings.HasPrefix(cleaned, "```json") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
		cleaned = strings.TrimSuffix(cleaned, "```")
		cleaned = strings.TrimSpace(cleaned)
	} else if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```")
		cleaned = strings.TrimSuffix(cleaned, "```")
		cleaned = strings.TrimSpace(cleaned)
	}

	return cleaned
}

// LLMClient defines the interface for LLM operations needed by the narrative generator
type LLMClient interface {
	// GenerateText generates text from a prompt with optional structured output
	GenerateText(ctx context.Context, prompt string, options llm.TextGenerationOptions) (string, error)
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

// Statistic represents a key metric or data point for the "By the Numbers" section
type SlackDigestContent struct {
	WeekRange     string       `json:"week_range"`     // e.g., "Jan 6-10"
	Big3          []Big3Item   `json:"big_3"`          // Top 3 impactful articles
	AlsoOnRadar   []RadarItem  `json:"also_on_radar"`  // Secondary articles
	ThreadContent []ThreadItem `json:"thread_content"` // Expanded thread details
}

// Big3Item represents one of the "Big 3" featured articles with editorial voice
type Big3Item struct {
	Headline   string `json:"headline"`    // Punchy 3-5 word headline
	Editorial  string `json:"editorial"`   // 1-2 opinionated sentences
	ArticleNum int    `json:"article_num"` // Citation number for link (1-based)
}

// RadarItem represents a secondary article mention
type RadarItem struct {
	Title      string `json:"title"`       // Short title
	ArticleNum int    `json:"article_num"` // Citation number (1-based)
}

// ThreadItem represents expanded thread content for Slack replies
type ThreadItem struct {
	Title       string `json:"title"`       // Article title
	Explanation string `json:"explanation"` // 2-3 sentence explanation
	ArticleNum  int    `json:"article_num"` // Position in thread (1-based)
}

// GenerateClusterSummary generates a comprehensive narrative for a single cluster using ALL articles
// This implements hierarchical summarization: cluster summary → executive summary
func (g *Generator) GenerateSlackDigest(ctx context.Context, clusters []core.TopicCluster, articles map[string]core.Article, summaries map[string]core.Summary) (*SlackDigestContent, error) {
	if len(clusters) == 0 {
		return nil, fmt.Errorf("no clusters provided")
	}

	prompt := g.buildSlackDigestPrompt(clusters, articles, summaries)
	schema := g.buildSlackDigestSchema()

	response, err := g.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		ResponseSchema: schema,
		Temperature:    0.8,   // Slightly higher for editorial voice creativity
		MaxTokens:      16384, // Increased for generating content for all articles
	})
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	return g.parseSlackDigestContent(response)
}

// buildSlackDigestPrompt creates the prompt for Slack digest generation with editorial voice
func (g *Generator) buildSlackDigestPrompt(clusters []core.TopicCluster, articles map[string]core.Article, summaries map[string]core.Summary) string {
	var prompt strings.Builder

	prompt.WriteString("You are a senior tech editor writing a weekly digest for busy engineering leaders.\n")
	prompt.WriteString("Your voice is PUNCHY, OPINIONATED, and SPECIFIC - not neutral or academic.\n\n")

	// Build article reference list with citation numbers
	prompt.WriteString("**Articles for reference (use these citation numbers):**\n\n")
	articleNum := 1
	for _, cluster := range clusters {
		for _, articleID := range cluster.ArticleIDs {
			if article, found := articles[articleID]; found {
				summary := summaries[articleID]
				fmt.Fprintf(&prompt, "[%d] %s\n", articleNum, article.Title)
				fmt.Fprintf(&prompt, "    URL: %s\n", article.URL)
				if summary.SummaryText != "" {
					fmt.Fprintf(&prompt, "    Summary: %s\n\n", truncateText(summary.SummaryText, 200))
				} else {
					prompt.WriteString("\n")
				}
				articleNum++
			}
		}
	}

	prompt.WriteString("\n**YOUR TASK:**\n")
	prompt.WriteString("Generate a Slack-optimized digest with the following sections:\n\n")

	prompt.WriteString("**1. WEEK RANGE** (e.g., \"Jan 6-10\", \"Dec 30-Jan 3\")\n")
	prompt.WriteString("   - Format: \"Mon DD-DD\" for same month, or \"Mon DD-Mon DD\" for cross-month\n\n")

	prompt.WriteString("**2. BIG 3** - The three most impactful articles this week\n")
	prompt.WriteString("   SELECTION CRITERIA (in priority order):\n")
	prompt.WriteString("   - Practical tools/frameworks engineers can use immediately\n")
	prompt.WriteString("   - Major announcements affecting engineering workflows\n")
	prompt.WriteString("   - Technical breakthroughs with measurable impact\n\n")

	prompt.WriteString("   For each Big 3 item:\n")
	prompt.WriteString("   - headline: Punchy 3-5 word headline (e.g., \"Claude 4.5 drops\", \"DeepSeek surprises everyone\")\n")
	prompt.WriteString("     * Use STRONG ACTIVE VERBS: \"drops\", \"ships\", \"breaks\", \"surprises\", \"crushes\"\n")
	prompt.WriteString("     * NO weak verbs: \"announces\", \"releases\", \"updates\"\n")
	prompt.WriteString("   - editorial: 1-2 sentences with OPINION and SPECIFICS\n")
	prompt.WriteString("     * Include SPECIFIC metrics when available (\"98.5%% accuracy\", \"$5.5M training cost\")\n")
	prompt.WriteString("     * Be opinionated: \"If real, changes everything\" not \"This may have implications\"\n")
	prompt.WriteString("     * End with a take: \"First impressions: noticeably better at code architecture\"\n")
	prompt.WriteString("   - article_num: Citation number [1-N] of the source article\n\n")

	prompt.WriteString("   EXAMPLES OF GOOD BIG 3:\n")
	prompt.WriteString("   ✓ {\"headline\": \"Claude 4.5 drops\", \"editorial\": \"Anthropic's new flagship. Extended thinking mode is the headline feature—lets model reason longer on hard problems. First impressions: noticeably better at code architecture.\", \"article_num\": 1}\n")
	prompt.WriteString("   ✓ {\"headline\": \"DeepSeek V3 surprises everyone\", \"editorial\": \"Open-weights 685B MoE matching GPT-4 on benchmarks. Training cost allegedly $5.5M. If real, changes the economics conversation.\", \"article_num\": 3}\n")
	prompt.WriteString("   ✗ {\"headline\": \"New AI model released\", \"editorial\": \"A new model has been announced with improved capabilities.\"} - TOO GENERIC!\n\n")

	fmt.Fprintf(&prompt, "**3. ALSO ON RADAR** - ALL remaining articles (%d total minus Big 3 = %d items)\n", articleNum-1, articleNum-1-3)
	prompt.WriteString("   - Include EVERY article that didn't make Big 3\n")
	prompt.WriteString("   - title: Short, descriptive title (e.g., \"OpenAI o3 ARC-AGI results\")\n")
	prompt.WriteString("   - article_num: Citation number of source article\n\n")

	prompt.WriteString("**4. THREAD CONTENT** - Expanded details for EVERY \"Also on radar\" item\n")
	prompt.WriteString("   CRITICAL: You MUST provide thread content for EVERY item in also_on_radar\n")
	prompt.WriteString("   For each item in also_on_radar, provide:\n")
	prompt.WriteString("   - title: Same as radar item title\n")
	prompt.WriteString("   - explanation: 2-3 sentence explanation of \"why it matters\" for engineers\n")
	prompt.WriteString("     * Be specific: name products, metrics, companies\n")
	prompt.WriteString("     * Focus on practical impact: \"You can now...\" or \"This means...\"\n")
	prompt.WriteString("   - article_num: Same citation number as radar item\n\n")

	prompt.WriteString("**CRITICAL RULES:**\n")
	prompt.WriteString("1. Big 3 MUST be different articles (no duplicates)\n")
	prompt.WriteString("2. Also on radar MUST NOT include any Big 3 articles\n")
	prompt.WriteString("3. All article_num values must be valid citation numbers from the list above\n")
	prompt.WriteString("4. Editorial voice: Be opinionated, punchy, specific - NOT neutral/academic\n")
	prompt.WriteString("5. Thread content must match also_on_radar items exactly\n\n")

	prompt.WriteString("Generate the Slack digest content in JSON format.\n")

	return prompt.String()
}

// buildSlackDigestSchema defines the Gemini JSON schema for Slack digest content
func (g *Generator) buildSlackDigestSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"week_range": {
				Type:        genai.TypeString,
				Description: "Date range for the digest (e.g., 'Jan 6-10')",
			},
			"big_3": {
				Type:        genai.TypeArray,
				Description: "Top 3 most impactful articles with editorial summaries",
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"headline": {
							Type:        genai.TypeString,
							Description: "Punchy 3-5 word headline with strong active verbs",
						},
						"editorial": {
							Type:        genai.TypeString,
							Description: "1-2 opinionated sentences with specific metrics",
						},
						"article_num": {
							Type:        genai.TypeInteger,
							Description: "Citation number of source article (1-based)",
						},
					},
					Required: []string{"headline", "editorial", "article_num"},
				},
			},
			"also_on_radar": {
				Type:        genai.TypeArray,
				Description: "3-5 additional notable articles (short titles)",
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"title": {
							Type:        genai.TypeString,
							Description: "Short descriptive title",
						},
						"article_num": {
							Type:        genai.TypeInteger,
							Description: "Citation number of source article (1-based)",
						},
					},
					Required: []string{"title", "article_num"},
				},
			},
			"thread_content": {
				Type:        genai.TypeArray,
				Description: "Expanded content for each also_on_radar item",
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"title": {
							Type:        genai.TypeString,
							Description: "Article title (same as radar item)",
						},
						"explanation": {
							Type:        genai.TypeString,
							Description: "2-3 sentence explanation of why it matters",
						},
						"article_num": {
							Type:        genai.TypeInteger,
							Description: "Citation number (same as radar item)",
						},
					},
					Required: []string{"title", "explanation", "article_num"},
				},
			},
		},
		Required: []string{"week_range", "big_3", "also_on_radar", "thread_content"},
	}
}

// parseSlackDigestContent parses JSON response into SlackDigestContent
func (g *Generator) parseSlackDigestContent(jsonResponse string) (*SlackDigestContent, error) {
	cleaned := cleanJSONResponse(jsonResponse)

	if len(cleaned) == 0 {
		return nil, fmt.Errorf("empty JSON response")
	}

	var response struct {
		WeekRange string `json:"week_range"`
		Big3      []struct {
			Headline   string `json:"headline"`
			Editorial  string `json:"editorial"`
			ArticleNum int    `json:"article_num"`
		} `json:"big_3"`
		AlsoOnRadar []struct {
			Title      string `json:"title"`
			ArticleNum int    `json:"article_num"`
		} `json:"also_on_radar"`
		ThreadContent []struct {
			Title       string `json:"title"`
			Explanation string `json:"explanation"`
			ArticleNum  int    `json:"article_num"`
		} `json:"thread_content"`
	}

	err := json.Unmarshal([]byte(cleaned), &response)
	if err != nil {
		preview := cleaned
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		log.Printf("[ERROR] parseSlackDigestContent: JSON parse failed: %v\nResponse preview: %s", err, preview)
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Convert to SlackDigestContent
	content := &SlackDigestContent{
		WeekRange:     response.WeekRange,
		Big3:          make([]Big3Item, 0, len(response.Big3)),
		AlsoOnRadar:   make([]RadarItem, 0, len(response.AlsoOnRadar)),
		ThreadContent: make([]ThreadItem, 0, len(response.ThreadContent)),
	}

	for _, item := range response.Big3 {
		content.Big3 = append(content.Big3, Big3Item{
			Headline:   item.Headline,
			Editorial:  item.Editorial,
			ArticleNum: item.ArticleNum,
		})
	}

	for _, item := range response.AlsoOnRadar {
		content.AlsoOnRadar = append(content.AlsoOnRadar, RadarItem{
			Title:      item.Title,
			ArticleNum: item.ArticleNum,
		})
	}

	for _, item := range response.ThreadContent {
		content.ThreadContent = append(content.ThreadContent, ThreadItem{
			Title:       item.Title,
			Explanation: item.Explanation,
			ArticleNum:  item.ArticleNum,
		})
	}

	return content, nil
}

// truncateText truncates text to maxLength characters, appending "..." when cut.
func truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	for maxLength > 0 && (text[maxLength]&0xC0) == 0x80 {
		maxLength--
	}
	return text[:maxLength] + "..."
}
