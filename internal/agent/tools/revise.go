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

// ReviseSectionTool revises a specific section of the digest based on critique from reflection.
type ReviseSectionTool struct {
	llmClient *llm.Client
}

// NewReviseSectionTool creates a new revise tool.
func NewReviseSectionTool(llmClient *llm.Client) *ReviseSectionTool {
	return &ReviseSectionTool{llmClient: llmClient}
}

func (t *ReviseSectionTool) Name() string { return "revise_section" }

func (t *ReviseSectionTool) Description() string {
	return "Revise a specific section of the digest based on critique from reflection. Takes a section identifier and the weakness to address. Produces an improved version of that section only, preserving the rest of the digest."
}

func (t *ReviseSectionTool) Parameters() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"section": {
				Type:        genai.TypeString,
				Description: "Section to revise: title, tldr, top_developments, why_it_matters, executive_summary, or cluster_narrative:<cluster_id>",
			},
			"weakness_description": {
				Type:        genai.TypeString,
				Description: "What is wrong with the current version",
			},
			"revision_instructions": {
				Type:        genai.TypeString,
				Description: "Specific instructions for how to improve",
			},
		},
		Required: []string{"section", "weakness_description"},
	}
}

func (t *ReviseSectionTool) Execute(ctx context.Context, memory *agent.WorkingMemory, params map[string]any) (map[string]any, error) {
	section := extractStringParam(params, "section", "")
	if section == "" {
		return nil, fmt.Errorf("section is required")
	}

	weaknessDesc := extractStringParam(params, "weakness_description", "")
	revisionInstructions := extractStringParam(params, "revision_instructions", "")

	// Get current content
	currentContent, ok := memory.GetDigestSection(section)
	if !ok {
		// Try to extract from digest draft
		digest := memory.GetDigestDraft()
		if digest == nil {
			return nil, fmt.Errorf("no digest draft available")
		}
		switch section {
		case "title":
			currentContent = digest.Title
		case "tldr":
			currentContent = digest.TLDRSummary
		case "why_it_matters":
			currentContent = digest.WhyItMatters
		case "top_developments":
			for _, d := range digest.TopDevelopments {
				currentContent += "- " + d + "\n"
			}
		default:
			return nil, fmt.Errorf("unknown section: %q", section)
		}
	}

	// Use stable article index for grounding
	articleContext := memory.FormatArticleList()

	prompt := fmt.Sprintf(`You are revising a section of a technology news digest.

SECTION: %s
CURRENT CONTENT:
%s

WEAKNESS:
%s

REVISION INSTRUCTIONS:
%s

AVAILABLE ARTICLES (for citations):
%s

Produce an improved version of this section that addresses the weakness.
Keep the same format and approximate length.
Ensure all claims are grounded with [N] citations.
Be specific — use company names, numbers, and concrete details.
`, section, currentContent, weaknessDesc, revisionInstructions, articleContext)

	schema := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"revised_content": {Type: genai.TypeString, Description: "The improved version of the section"},
			"changes_made":    {Type: genai.TypeString, Description: "Brief description of what changed"},
		},
		Required: []string{"revised_content", "changes_made"},
	}

	resp, err := t.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		ResponseSchema: schema,
		Temperature:    0.3,
	})
	if err != nil {
		return nil, fmt.Errorf("revision LLM call failed: %w", err)
	}

	var result struct {
		RevisedContent string `json:"revised_content"`
		ChangesMade    string `json:"changes_made"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return nil, fmt.Errorf("failed to parse revision response: %w", err)
	}

	// Update the section in memory
	memory.SetDigestSection(section, result.RevisedContent)

	// Also update the digest draft
	digest := memory.GetDigestDraft()
	if digest != nil {
		switch section {
		case "title":
			digest.Title = result.RevisedContent
		case "tldr":
			digest.TLDRSummary = result.RevisedContent
		case "why_it_matters":
			digest.WhyItMatters = result.RevisedContent
		}
		memory.SetDigestDraft(digest)
	}

	// Log the revision
	memory.AddRevision(agent.RevisionRecord{
		Iteration:         len(memory.GetReflections()),
		Timestamp:         time.Now(),
		TargetSection:     section,
		WeaknessAddressed: weaknessDesc,
		OriginalContent:   currentContent,
		RevisedContent:    result.RevisedContent,
		ChangesMade:       result.ChangesMade,
	})

	return map[string]any{
		"section":          section,
		"original_content": truncateStr(currentContent, 200),
		"revised_content":  truncateStr(result.RevisedContent, 200),
		"changes_made":     result.ChangesMade,
	}, nil
}
