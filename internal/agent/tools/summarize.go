package tools

import (
	"briefly/internal/agent"
	"briefly/internal/core"
	"briefly/internal/store"
	"briefly/internal/summarize"
	"context"
	"crypto/md5"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/genai"
)

// SummarizeBatchTool wraps the existing summarizer to generate summaries for articles.
type SummarizeBatchTool struct {
	summarizer *summarize.Summarizer
	cache      *store.Store
}

// NewSummarizeBatchTool creates a new summarize batch tool.
func NewSummarizeBatchTool(summarizer *summarize.Summarizer, cache *store.Store) *SummarizeBatchTool {
	return &SummarizeBatchTool{
		summarizer: summarizer,
		cache:      cache,
	}
}

func (t *SummarizeBatchTool) Name() string { return "summarize_batch" }

func (t *SummarizeBatchTool) Description() string {
	return "Generate summaries for articles that do not yet have summaries. Uses the existing summarization pipeline with cache support. Call after fetch_articles to prepare articles for analysis."
}

func (t *SummarizeBatchTool) Parameters() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"article_ids": {
				Type:        genai.TypeArray,
				Items:       &genai.Schema{Type: genai.TypeString},
				Description: "IDs of articles to summarize. Omit to summarize all unsummarized articles.",
			},
		},
	}
}

func (t *SummarizeBatchTool) Execute(ctx context.Context, memory *agent.WorkingMemory, params map[string]any) (map[string]any, error) {
	articles := memory.GetArticles()
	existingSummaries := memory.GetSummaries()
	targetIDs := extractStringSliceParam(params, "article_ids")

	// Determine which articles to summarize
	var toSummarize []core.Article
	if len(targetIDs) > 0 {
		for _, id := range targetIDs {
			if a, ok := articles[id]; ok {
				if _, hasSummary := existingSummaries[id]; !hasSummary {
					toSummarize = append(toSummarize, a)
				}
			}
		}
	} else {
		for id, a := range articles {
			if _, hasSummary := existingSummaries[id]; !hasSummary {
				toSummarize = append(toSummarize, a)
			}
		}
	}

	var generated, cacheHits, failures int
	summaryList := make([]map[string]any, 0)
	cacheTTL := 7 * 24 * time.Hour

	for _, article := range toSummarize {
		// Try cache first
		if t.cache != nil {
			contentHash := fmt.Sprintf("%x", md5.Sum([]byte(article.CleanedText)))
			cachedSummary, err := t.cache.GetCachedSummary(article.URL, contentHash, cacheTTL)
			if err == nil && cachedSummary != nil {
				memory.AddSummary(article.ID, *cachedSummary)
				cacheHits++
				summaryList = append(summaryList, map[string]any{
					"article_id":      article.ID,
					"title":           article.Title,
					"summary_preview": truncateStr(cachedSummary.SummaryText, 150),
				})
				continue
			}
		}

		// Generate new summary
		summary, err := t.summarizer.SummarizeArticle(ctx, &article)
		if err != nil {
			failures++
			continue
		}

		if summary.ID == "" {
			summary.ID = uuid.NewString()
		}
		summary.ArticleIDs = []string{article.ID}

		// Cache the summary
		if t.cache != nil {
			contentHash := fmt.Sprintf("%x", md5.Sum([]byte(article.CleanedText)))
			_ = t.cache.CacheSummary(*summary, article.URL, contentHash)
		}

		memory.AddSummary(article.ID, *summary)
		generated++
		summaryList = append(summaryList, map[string]any{
			"article_id":      article.ID,
			"title":           article.Title,
			"summary_preview": truncateStr(summary.SummaryText, 150),
		})
	}

	return map[string]any{
		"summaries_generated":     generated,
		"cache_hits":              cacheHits,
		"failed":                  failures,
		"articles_with_summaries": summaryList,
	}, nil
}
