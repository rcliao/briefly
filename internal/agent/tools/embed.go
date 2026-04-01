package tools

import (
	"briefly/internal/agent"
	"briefly/internal/llm"
	"context"
	"fmt"

	"google.golang.org/genai"
)

// GenerateEmbeddingsTool wraps the LLM client's embedding generation.
type GenerateEmbeddingsTool struct {
	llmClient *llm.Client
}

// NewGenerateEmbeddingsTool creates a new embeddings tool.
func NewGenerateEmbeddingsTool(llmClient *llm.Client) *GenerateEmbeddingsTool {
	return &GenerateEmbeddingsTool{llmClient: llmClient}
}

func (t *GenerateEmbeddingsTool) Name() string { return "generate_embeddings" }

func (t *GenerateEmbeddingsTool) Description() string {
	return "Generate 768-dimensional embedding vectors for article summaries. Required before clustering. Only generates for articles that do not have embeddings yet."
}

func (t *GenerateEmbeddingsTool) Parameters() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"article_ids": {
				Type:        genai.TypeArray,
				Items:       &genai.Schema{Type: genai.TypeString},
				Description: "IDs of articles to embed. Omit to embed all articles with summaries.",
			},
		},
	}
}

func (t *GenerateEmbeddingsTool) Execute(ctx context.Context, memory *agent.WorkingMemory, params map[string]any) (map[string]any, error) {
	summaries := memory.GetSummaries()
	existingEmbeddings := memory.GetEmbeddings()
	targetIDs := extractStringSliceParam(params, "article_ids")

	var toEmbed []string
	if len(targetIDs) > 0 {
		for _, id := range targetIDs {
			if _, hasSummary := summaries[id]; hasSummary {
				if _, hasEmbed := existingEmbeddings[id]; !hasEmbed {
					toEmbed = append(toEmbed, id)
				}
			}
		}
	} else {
		for id := range summaries {
			if _, hasEmbed := existingEmbeddings[id]; !hasEmbed {
				toEmbed = append(toEmbed, id)
			}
		}
	}

	var generated int
	alreadyEmbedded := len(existingEmbeddings)

	for _, id := range toEmbed {
		summary := summaries[id]
		text := summary.SummaryText
		if text == "" {
			continue
		}

		// GenerateEmbedding takes only text (no context)
		embedding, err := t.llmClient.GenerateEmbedding(text)
		if err != nil {
			return nil, fmt.Errorf("embedding generation failed for article %s: %w", id, err)
		}

		memory.AddEmbedding(id, embedding)
		generated++
	}

	return map[string]any{
		"embeddings_generated": generated,
		"dimensions":           768,
		"already_embedded":     alreadyEmbedded,
	}, nil
}
