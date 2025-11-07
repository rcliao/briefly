package narrative

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"briefly/internal/core"
	"briefly/internal/llm"
)

// CritiqueResult contains the critique and improved digest
type CritiqueResult struct {
	Critique        *Critique       `json:"critique"`
	ImprovedDigest  *DigestContent  `json:"improved_digest"`
	QualityImproved bool            `json:"quality_improved"`
}

// Critique contains detailed analysis of digest quality issues
type Critique struct {
	ArticlesMentioned []int    `json:"articles_mentioned"`
	ArticlesMissing   []int    `json:"articles_missing"`
	VaguePhrases      []string `json:"vague_phrases"`
	QuoteAccuracy     string   `json:"quote_accuracy"`
	TLDRQuality       string   `json:"tldr_quality"`
	OverallIssues     []string `json:"overall_issues"`
	SpecificityScore  int      `json:"specificity_score"` // 0-100
}

// RefineDigestWithCritique performs self-critique and refinement on generated digest
// This is the "always-on" self-critique pass for quality assurance
func (g *Generator) RefineDigestWithCritique(
	ctx context.Context,
	draftDigest *DigestContent,
	clusters []core.TopicCluster,
	articles map[string]core.Article,
	summaries map[string]core.Summary,
) (*CritiqueResult, error) {
	// Build critique prompt with draft digest + cluster narratives
	prompt := g.buildCritiquePrompt(draftDigest, clusters, articles, summaries)

	// Generate critique and improved digest using structured output
	schema := g.buildCritiqueSchema()

	response, err := g.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		ResponseSchema: schema,
		Temperature:    0.7,
		MaxTokens:      2000,
	})
	if err != nil {
		return nil, fmt.Errorf("critique generation failed: %w", err)
	}

	// Parse response
	result, err := g.parseCritiqueResult(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse critique result: %w", err)
	}

	return result, nil
}

// buildCritiquePrompt creates the self-critique prompt
func (g *Generator) buildCritiquePrompt(
	draftDigest *DigestContent,
	clusters []core.TopicCluster,
	articles map[string]core.Article,
	summaries map[string]core.Summary,
) string {
	var prompt strings.Builder

	prompt.WriteString("Review and improve this digest by critiquing against the source material.\n\n")

	// SECTION 1: Original cluster narratives
	prompt.WriteString("**ORIGINAL CLUSTER NARRATIVES:**\n\n")
	for i, cluster := range clusters {
		if cluster.Narrative != nil {
			prompt.WriteString(fmt.Sprintf("## Cluster %d: %s\n", i+1, cluster.Narrative.Title))
			prompt.WriteString(fmt.Sprintf("**Articles:** %d\n", len(cluster.ArticleIDs)))
			prompt.WriteString(fmt.Sprintf("**Key Themes:** %s\n", strings.Join(cluster.Narrative.KeyThemes, ", ")))
			prompt.WriteString(fmt.Sprintf("**Summary:**\n%s\n\n", cluster.Narrative.Summary))
			prompt.WriteString("---\n\n")
		}
	}

	// SECTION 2: Article reference list
	prompt.WriteString("**ALL ARTICLES (for verification):**\n")
	articleNum := 1
	for _, cluster := range clusters {
		for _, articleID := range cluster.ArticleIDs {
			if article, found := articles[articleID]; found {
				prompt.WriteString(fmt.Sprintf("[%d] %s\n", articleNum, article.Title))
				prompt.WriteString(fmt.Sprintf("    URL: %s\n", article.URL))

				// Include summary excerpt for context
				if summary, found := summaries[articleID]; found {
					excerpt := summary.SummaryText
					if len(excerpt) > 150 {
						excerpt = excerpt[:150] + "..."
					}
					prompt.WriteString(fmt.Sprintf("    Summary: %s\n", excerpt))
				}
				prompt.WriteString("\n")
				articleNum++
			}
		}
	}

	totalArticles := articleNum - 1

	// SECTION 3: Draft digest to critique
	prompt.WriteString("**DRAFT DIGEST TO CRITIQUE:**\n\n")
	prompt.WriteString(fmt.Sprintf("**Title:** %s (%d chars)\n", draftDigest.Title, len(draftDigest.Title)))
	prompt.WriteString(fmt.Sprintf("**TLDR:** %s (%d chars)\n\n", draftDigest.TLDRSummary, len(draftDigest.TLDRSummary)))
	prompt.WriteString(fmt.Sprintf("**Executive Summary:**\n%s\n\n", draftDigest.ExecutiveSummary))

	if len(draftDigest.KeyMoments) > 0 {
		prompt.WriteString("**Key Moments:**\n")
		for i, km := range draftDigest.KeyMoments {
			prompt.WriteString(fmt.Sprintf("%d. \"%s\" [%d]\n", i+1, km.Quote, km.CitationNumber))
		}
		prompt.WriteString("\n")
	}

	// SECTION 4: Critique instructions
	prompt.WriteString("**CRITIQUE CHECKLIST:**\n\n")

	prompt.WriteString("1. **Title Quality:**\n")
	prompt.WriteString("   - Length: ≤ 40 characters (count every character)\n")
	prompt.WriteString("   - Voice: Active voice with strong action verb\n")
	prompt.WriteString("   - BANNED verbs: \"updates\", \"announces\", \"releases\", \"changes\"\n")
	prompt.WriteString("   - REQUIRED: Power verb (\"cuts\", \"hits\", \"beats\", \"breaks\", \"surges\", \"doubles\")\n")
	prompt.WriteString("   - REQUIRED: Specific actor (company/tech) + quantified result\n")
	prompt.WriteString("   - Example GOOD: \"Voice AI Hits 1-Second Latency\"\n")
	prompt.WriteString("   - Example BAD: \"New AI Updates Released\"\n\n")

	prompt.WriteString("2. **TLDR Structure:**\n")
	prompt.WriteString("   - Length: ≤ 75 characters (count every character)\n")
	prompt.WriteString("   - REQUIRED STRUCTURE: [Subject] + [Action Verb] + [Object] + [Impact]\n")
	prompt.WriteString("   - Subject: Specific company/technology (e.g., \"Perplexity\", \"Voice AI\")\n")
	prompt.WriteString("   - Action Verb: Strong active verb (e.g., \"achieves\", \"cuts\", \"hits\")\n")
	prompt.WriteString("   - Object: What changed (e.g., \"latency\", \"throughput\")\n")
	prompt.WriteString("   - Impact: MUST include specific number (e.g., \"to 1 second\", \"by 60%\", \"400 Gbps\")\n")
	prompt.WriteString("   - REQUIRED: At least one specific number/percentage in TLDR\n\n")

	prompt.WriteString("3. **Article Coverage:**\n")
	prompt.WriteString(fmt.Sprintf("   - Identify which articles ([1-%d]) are mentioned in executive summary\n", totalArticles))
	prompt.WriteString("   - List any articles that are NOT mentioned\n")
	prompt.WriteString(fmt.Sprintf("   - REQUIRED: All %d articles should be cited\n\n", totalArticles))

	prompt.WriteString("4. **Vagueness Detection:**\n")
	prompt.WriteString("   - Find any vague/generic phrases: \"several\", \"various\", \"many\", \"some\", \"numerous\"\n")
	prompt.WriteString("   - Find any vague qualifiers: \"recently\", \"significant\", \"substantial\"\n")
	prompt.WriteString("   - Quote each vague phrase found\n")
	prompt.WriteString("   - REQUIRED: Zero vague phrases\n\n")

	prompt.WriteString("5. **Specificity Check:**\n")
	prompt.WriteString("   - Count specific numbers/percentages/metrics in executive summary\n")
	prompt.WriteString("   - Count proper nouns (companies, people, products)\n")
	prompt.WriteString("   - REQUIRED: At least 3 numbers, at least 5 proper nouns\n\n")

	prompt.WriteString("6. **Executive Summary Connections:**\n")
	prompt.WriteString("   - Check for transition phrases between clusters:\n")
	prompt.WriteString("   - \"Building on...\", \"In contrast to...\", \"Meanwhile...\", \"Supporting this...\"\n")
	prompt.WriteString("   - REQUIRED: At least one explicit connection phrase\n\n")

	prompt.WriteString("7. **Key Moments Quality:**\n")
	prompt.WriteString("   - Check each key moment has specific numbers/metrics\n")
	prompt.WriteString("   - REQUIRED: Every key moment must include quantified data\n")
	prompt.WriteString("   - Example GOOD: \"Perplexity hit 400 Gbps throughput\"\n")
	prompt.WriteString("   - Example BAD: \"Performance improved significantly\"\n\n")

	prompt.WriteString("8. **Citation Accuracy:**\n")
	prompt.WriteString("   - Verify citation numbers match article list\n")
	prompt.WriteString("   - Check that cited articles actually support the claims\n\n")

	// SECTION 5: Improvement task
	prompt.WriteString("**IMPROVEMENT TASK:**\n\n")
	prompt.WriteString("After critiquing, generate an IMPROVED version of the digest that fixes ALL identified issues:\n\n")

	prompt.WriteString("✅ MUST FIX:\n")
	prompt.WriteString("- Title: Use power verb (\"cuts\"/\"hits\"/\"beats\"), include specific actor + quantified result\n")
	prompt.WriteString("- TLDR: Follow [Subject]+[Verb]+[Object]+[Impact] structure with specific number\n")
	prompt.WriteString(fmt.Sprintf("- Coverage: Cite all %d articles at least once\n", totalArticles))
	prompt.WriteString("- Connections: Add transition phrases between clusters (\"Building on...\", \"Meanwhile...\")\n")
	prompt.WriteString("- Vagueness: Replace ALL vague phrases with specific facts\n")
	prompt.WriteString("- Key Moments: Ensure every moment includes quantified data/metrics\n")
	prompt.WriteString("- Add specific numbers, names, dates where missing\n")
	prompt.WriteString("- Ensure title ≤ 40 chars, TLDR ≤ 75 chars\n")
	prompt.WriteString("- Maintain 150-200 word executive summary length\n\n")

	prompt.WriteString("✅ PRESERVE:\n")
	prompt.WriteString("- Overall narrative structure and flow\n")
	prompt.WriteString("- Key moments (but improve specificity if needed)\n")
	prompt.WriteString("- Perspectives (if present)\n\n")

	prompt.WriteString("Return both the critique AND the improved digest in JSON format.\n")

	return prompt.String()
}

// buildCritiqueSchema defines the JSON schema for critique output
func (g *Generator) buildCritiqueSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"critique": {
				Type:        genai.TypeObject,
				Description: "Detailed critique of the draft digest",
				Properties: map[string]*genai.Schema{
					"articles_mentioned": {
						Type:        genai.TypeArray,
						Description: "List of article numbers found in executive summary",
						Items: &genai.Schema{
							Type: genai.TypeInteger,
						},
					},
					"articles_missing": {
						Type:        genai.TypeArray,
						Description: "List of article numbers NOT mentioned",
						Items: &genai.Schema{
							Type: genai.TypeInteger,
						},
					},
					"vague_phrases": {
						Type:        genai.TypeArray,
						Description: "List of vague/generic phrases found",
						Items: &genai.Schema{
							Type: genai.TypeString,
						},
					},
					"quote_accuracy": {
						Type:        genai.TypeString,
						Description: "Assessment of citation accuracy",
					},
					"tldr_quality": {
						Type:        genai.TypeString,
						Description: "Assessment of TLDR quality and length",
					},
					"overall_issues": {
						Type:        genai.TypeArray,
						Description: "List of all identified issues",
						Items: &genai.Schema{
							Type: genai.TypeString,
						},
					},
					"specificity_score": {
						Type:        genai.TypeInteger,
						Description: "Specificity score 0-100 (numbers + proper nouns)",
					},
				},
			},
			"improved_digest": {
				Type:        genai.TypeObject,
				Description: "Improved version fixing all issues",
				Properties: map[string]*genai.Schema{
					"title": {
						Type:        genai.TypeString,
						Description: "Improved title (≤40 chars)",
					},
					"tldr_summary": {
						Type:        genai.TypeString,
						Description: "Improved TLDR (≤75 chars)",
					},
					"executive_summary": {
						Type:        genai.TypeString,
						Description: "Improved executive summary (150-200 words, all articles cited)",
					},
					"key_moments": {
						Type:        genai.TypeArray,
						Description: "Improved key moments",
						Items: &genai.Schema{
							Type: genai.TypeObject,
							Properties: map[string]*genai.Schema{
								"quote": {
									Type:        genai.TypeString,
									Description: "Specific quote or development",
								},
								"citation_number": {
									Type:        genai.TypeInteger,
									Description: "Article citation number",
								},
							},
						},
					},
					"perspectives": {
						Type:        genai.TypeArray,
						Description: "Perspectives (if any)",
						Items: &genai.Schema{
							Type: genai.TypeObject,
							Properties: map[string]*genai.Schema{
								"type": {
									Type:        genai.TypeString,
									Description: "supporting or opposing",
								},
								"summary": {
									Type:        genai.TypeString,
									Description: "Perspective summary",
								},
								"citation_numbers": {
									Type: genai.TypeArray,
									Items: &genai.Schema{
										Type: genai.TypeInteger,
									},
								},
							},
						},
					},
				},
			},
			"quality_improved": {
				Type:        genai.TypeBoolean,
				Description: "Whether quality was successfully improved",
			},
		},
		Required: []string{"critique", "improved_digest", "quality_improved"},
	}
}

// parseCritiqueResult parses the JSON response into CritiqueResult
func (g *Generator) parseCritiqueResult(jsonResponse string) (*CritiqueResult, error) {
	var result CritiqueResult

	err := json.Unmarshal([]byte(jsonResponse), &result)
	if err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}

	// Validate required fields
	if result.Critique == nil {
		return nil, fmt.Errorf("missing critique in response")
	}
	if result.ImprovedDigest == nil {
		return nil, fmt.Errorf("missing improved_digest in response")
	}

	return &result, nil
}

// ShouldRunCritique determines if critique pass should run
// For "always-on" mode, this always returns true
// Can be extended with conditional logic if needed
func (g *Generator) ShouldRunCritique(draftDigest *DigestContent, config CritiqueConfig) bool {
	if config.AlwaysRun {
		return true
	}

	// Optional: Add conditional logic based on quality indicators
	// For example, run critique if:
	// - Title or TLDR too long
	// - Too few citations
	// - etc.

	return false
}

// CritiqueConfig holds configuration for self-critique pass
type CritiqueConfig struct {
	AlwaysRun       bool    // Run critique on every digest
	MinQualityGrade string  // Run critique if quality below this grade (A/B/C/D)
	MaxRetries      int     // Maximum retry attempts
}

// DefaultCritiqueConfig returns default configuration
func DefaultCritiqueConfig() CritiqueConfig {
	return CritiqueConfig{
		AlwaysRun:       true,  // Always run for maximum quality
		MinQualityGrade: "B",   // Critique if below B grade
		MaxRetries:      1,     // One refinement pass
	}
}
