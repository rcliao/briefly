package tools

import (
	"briefly/internal/agent"
	"briefly/internal/llm"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/genai"
)

// ReflectTool evaluates the current digest for newsletter quality.
type ReflectTool struct {
	llmClient *llm.Client
}

func NewReflectTool(llmClient *llm.Client) *ReflectTool {
	return &ReflectTool{llmClient: llmClient}
}

func (t *ReflectTool) Name() string { return "reflect" }

func (t *ReflectTool) Description() string {
	return "Evaluate the digest quality as a curated newsletter. Scores editorial voice, topic accuracy, coverage, and scannability. Call after triage to check editorial summaries, or after executive summary to check the overall digest."
}

func (t *ReflectTool) Parameters() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"focus_dimensions": {
				Type:        genai.TypeArray,
				Items:       &genai.Schema{Type: genai.TypeString},
				Description: "Dimensions to focus on. Options: editorial_voice, topic_fit, coverage, scannability, specificity",
			},
		},
	}
}

func (t *ReflectTool) Execute(ctx context.Context, memory *agent.WorkingMemory, params map[string]any) (map[string]any, error) {
	articleIndex := memory.GetArticleIndex()
	triageScores := memory.GetTriageScores()
	digest := memory.GetDigestDraft()

	if len(articleIndex) == 0 {
		return nil, fmt.Errorf("no articles to reflect on")
	}

	// Build the content to evaluate
	var content string

	if digest != nil {
		content += fmt.Sprintf("Title: %s\nTLDR: %s\n\n", digest.Title, digest.TLDRSummary)
	}

	content += "ARTICLE TRIAGE RESULTS:\n"
	for _, entry := range articleIndex {
		score, hasScore := triageScores[entry.ArticleID]
		topicID := entry.TopicID
		editorial := entry.EditorialSummary
		intent := entry.ReaderIntent
		if hasScore {
			if topicID == "" {
				topicID = score.TopicID
			}
			if editorial == "" {
				editorial = score.EditorialSummary
			}
			if intent == "" {
				intent = score.ReaderIntent
			}
		}
		content += fmt.Sprintf("\n[%d] %s\n  Topic: %s | Intent: %s\n  Editorial: %s\n",
			entry.CitationNum, entry.Title, topicID, intent, editorial)
	}

	topicList := agent.TopicListForPrompt()

	trajectory := memory.GetQualityTrajectory()
	previousContext := ""
	if len(trajectory) > 0 {
		previousContext = fmt.Sprintf("\nPREVIOUS SCORES: %v\nBe consistent with scoring.\n", trajectory)
	}

	prompt := fmt.Sprintf(`You are evaluating a curated GenAI newsletter. The goal is a scannable link digest where readers quickly catch up on the week's news by topic.

Evaluate on five dimensions (0.0 to 1.0):

1. **Editorial Voice** (0-1): Do editorial summaries sound like a human recommending links? Score 0.8+ if they use natural language ("Cloudflare figured out how to..."). Score below 0.5 if they sound like LLM output ("This article discusses the implications of...").

2. **Topic Fit** (0-1): Are articles assigned to the right topic categories? Score 0.8+ if groupings make intuitive sense. Score below 0.5 if articles are in wrong categories.

3. **Coverage** (0-1): Are all %d articles included with summaries? Score = articles_with_summaries / total_articles.

4. **Scannability** (0-1): Can a reader scan the full digest in under 2 minutes? Score 0.8+ if summaries are concise (1-2 sentences each). Score below 0.5 if summaries are bloated or repetitive.

5. **Specificity** (0-1): Do summaries mention specific facts (names, numbers, dates)? Score 0.7+ if most summaries name specific entities. Score below 0.5 if generic.

AVAILABLE TOPICS:
%s

Only flag weaknesses if they materially hurt the reader experience. Focus on editorial summaries that sound robotic or articles in the wrong topic.
%s
CONTENT TO EVALUATE:
%s
`, len(articleIndex), topicList, previousContext, content)

	schema := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"editorial_voice": {Type: genai.TypeNumber},
			"topic_fit":       {Type: genai.TypeNumber},
			"coverage":        {Type: genai.TypeNumber},
			"scannability":    {Type: genai.TypeNumber},
			"specificity":     {Type: genai.TypeNumber},
			"weaknesses": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"section":       {Type: genai.TypeString},
						"dimension":     {Type: genai.TypeString},
						"description":   {Type: genai.TypeString},
						"severity":      {Type: genai.TypeString},
						"suggested_fix": {Type: genai.TypeString},
					},
					Required: []string{"section", "dimension", "description", "severity"},
				},
			},
			"strengths": {
				Type:  genai.TypeArray,
				Items: &genai.Schema{Type: genai.TypeString},
			},
		},
		Required: []string{"editorial_voice", "topic_fit", "coverage", "scannability", "specificity", "weaknesses", "strengths"},
	}

	resp, err := t.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		ResponseSchema: schema,
		Temperature:    0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("reflect LLM call failed: %w", err)
	}

	var result struct {
		EditorialVoice float64 `json:"editorial_voice"`
		TopicFit       float64 `json:"topic_fit"`
		Coverage       float64 `json:"coverage"`
		Scannability   float64 `json:"scannability"`
		Specificity    float64 `json:"specificity"`
		Weaknesses     []struct {
			Section      string `json:"section"`
			Dimension    string `json:"dimension"`
			Description  string `json:"description"`
			Severity     string `json:"severity"`
			SuggestedFix string `json:"suggested_fix"`
		} `json:"weaknesses"`
		Strengths []string `json:"strengths"`
	}

	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return nil, fmt.Errorf("failed to parse reflect response: %w", err)
	}

	dimensions := agent.DimensionScores{
		Specificity: result.Specificity,
		Grounding:   result.TopicFit,
		Coherence:   result.Scannability,
		ReaderValue: result.EditorialVoice,
		Coverage:    result.Coverage,
	}
	overallScore := dimensions.WeightedAverage()

	iteration := len(trajectory)
	var improvementDelta float64
	var bestPrevious float64
	for _, score := range trajectory {
		if score > bestPrevious {
			bestPrevious = score
		}
	}
	if len(trajectory) > 0 {
		improvementDelta = overallScore - bestPrevious
	}

	shouldContinue := overallScore < 0.7
	if iteration > 0 {
		if iteration >= 2 && overallScore <= bestPrevious-0.02 {
			shouldContinue = false
		}
		if iteration >= 2 && improvementDelta < 0.03 && improvementDelta >= 0 {
			shouldContinue = false
		}
	}

	weaknesses := make([]agent.Weakness, 0)
	weaknessResults := make([]map[string]any, 0)
	for _, w := range result.Weaknesses {
		if w.Severity == "minor" && !shouldContinue {
			continue
		}
		weaknesses = append(weaknesses, agent.Weakness{
			Section: w.Section, Dimension: w.Dimension,
			Description: w.Description, Severity: w.Severity, SuggestedFix: w.SuggestedFix,
		})
		weaknessResults = append(weaknessResults, map[string]any{
			"section": w.Section, "dimension": w.Dimension,
			"description": w.Description, "severity": w.Severity, "suggested_fix": w.SuggestedFix,
		})
	}

	report := agent.ReflectionReport{
		Iteration: iteration, Timestamp: time.Now(),
		OverallScore: overallScore, Dimensions: dimensions,
		Weaknesses: weaknesses, Strengths: result.Strengths,
		ImprovementDelta: improvementDelta, ShouldContinue: shouldContinue,
	}
	memory.AddReflection(report)

	return map[string]any{
		"iteration":     iteration,
		"overall_score": overallScore,
		"dimension_scores": map[string]any{
			"editorial_voice": result.EditorialVoice,
			"topic_fit":       result.TopicFit,
			"coverage":        result.Coverage,
			"scannability":    result.Scannability,
			"specificity":     result.Specificity,
		},
		"weaknesses":        weaknessResults,
		"strengths":         result.Strengths,
		"improvement_delta": improvementDelta,
		"should_continue":   shouldContinue,
	}, nil
}
