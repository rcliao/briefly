package tools

import (
	"briefly/internal/agent"
	"context"
	"fmt"

	"google.golang.org/genai"
)

// GetArticleIndexTool returns the stable article citation index.
// The agent and all prompts should use these [N] numbers for citations.
type GetArticleIndexTool struct{}

func NewGetArticleIndexTool() *GetArticleIndexTool {
	return &GetArticleIndexTool{}
}

func (t *GetArticleIndexTool) Name() string { return "get_article_index" }

func (t *GetArticleIndexTool) Description() string {
	return "Returns the stable article citation index mapping [N] numbers to articles. Use these citation numbers in ALL generated content. Call this if you need to verify which article corresponds to which citation number."
}

func (t *GetArticleIndexTool) Parameters() *genai.Schema {
	return &genai.Schema{
		Type:       genai.TypeObject,
		Properties: map[string]*genai.Schema{},
	}
}

func (t *GetArticleIndexTool) Execute(ctx context.Context, memory *agent.WorkingMemory, params map[string]any) (map[string]any, error) {
	index := memory.GetArticleIndex()
	if len(index) == 0 {
		return nil, fmt.Errorf("article index not yet built — call fetch_articles first")
	}

	entries := make([]map[string]any, 0, len(index))
	for _, entry := range index {
		e := map[string]any{
			"citation_num":  entry.CitationNum,
			"article_id":    entry.ArticleID,
			"title":         entry.Title,
			"url":           entry.URL,
		}
		if entry.ReaderIntent != "" {
			e["reader_intent"] = entry.ReaderIntent
		}
		if entry.ReadMinutes > 0 {
			e["read_minutes"] = entry.ReadMinutes
		}
		entries = append(entries, e)
	}

	return map[string]any{
		"article_index": entries,
		"total_articles": len(index),
		"usage_note":    "Use [N] citation numbers from this index in ALL generated content. These numbers are stable across all tools.",
	}, nil
}
