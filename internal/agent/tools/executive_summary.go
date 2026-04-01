package tools

import (
	"briefly/internal/agent"
	"briefly/internal/core"
	"briefly/internal/narrative"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/genai"
)

// GenerateExecutiveSummaryTool wraps the narrative generator for full digest content.
type GenerateExecutiveSummaryTool struct {
	generator *narrative.Generator
}

// NewGenerateExecutiveSummaryTool creates a new executive summary tool.
func NewGenerateExecutiveSummaryTool(generator *narrative.Generator) *GenerateExecutiveSummaryTool {
	return &GenerateExecutiveSummaryTool{generator: generator}
}

func (t *GenerateExecutiveSummaryTool) Name() string { return "generate_executive_summary" }

func (t *GenerateExecutiveSummaryTool) Description() string {
	return "Generate the executive summary for the full digest, synthesizing all cluster narratives into a cohesive overview. Produces title, TLDR, top developments, by-the-numbers, and why-it-matters sections. Citations [N] in the output use the global article index numbers. Call after all cluster narratives are generated."
}

func (t *GenerateExecutiveSummaryTool) Parameters() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"include_must_read": {
				Type:        genai.TypeBoolean,
				Description: "Whether to include a Must-Read highlight (default: true)",
			},
		},
	}
}

func (t *GenerateExecutiveSummaryTool) Execute(ctx context.Context, memory *agent.WorkingMemory, params map[string]any) (map[string]any, error) {
	clusters := memory.GetClusters()
	articles := memory.GetArticles()
	summaries := memory.GetSummaries()
	narratives := memory.GetNarratives()

	// Attach narratives to clusters before passing to generator
	for i, c := range clusters {
		if narr, ok := narratives[c.ID]; ok {
			clusters[i].Narrative = &narr
		}
	}

	// Build citation remap: the generator numbers articles sequentially across clusters
	// [1],[2],[3] for cluster 1, then [4],[5] for cluster 2, etc.
	digestCitationMap := make(map[int]int)
	localNum := 1
	for _, c := range clusters {
		for _, aid := range c.ArticleIDs {
			globalNum := memory.GetCitationNum(aid)
			if globalNum > 0 {
				digestCitationMap[localNum] = globalNum
			}
			localNum++
		}
	}

	// Generate digest content using the existing generator
	digestContent, err := t.generator.GenerateDigestContent(ctx, clusters, articles, summaries)
	if err != nil {
		return nil, fmt.Errorf("executive summary generation failed: %w", err)
	}

	// Remap all citations from generator-local to global
	digestContent.TLDRSummary = remapCitations(digestContent.TLDRSummary, digestCitationMap)
	digestContent.WhyItMatters = remapCitations(digestContent.WhyItMatters, digestCitationMap)
	digestContent.TopDevelopments = remapCitationSlice(digestContent.TopDevelopments, digestCitationMap)
	for i := range digestContent.ByTheNumbers {
		digestContent.ByTheNumbers[i].Context = remapCitations(digestContent.ByTheNumbers[i].Context, digestCitationMap)
	}
	for i := range digestContent.KeyMoments {
		digestContent.KeyMoments[i].Quote = remapCitations(digestContent.KeyMoments[i].Quote, digestCitationMap)
		if num, ok := digestCitationMap[digestContent.KeyMoments[i].CitationNumber]; ok {
			digestContent.KeyMoments[i].CitationNumber = num
		}
	}
	if digestContent.MustRead != nil {
		if num, ok := digestCitationMap[digestContent.MustRead.ArticleNum]; ok {
			digestContent.MustRead.ArticleNum = num
		}
		digestContent.MustRead.WhyMustRead = remapCitations(digestContent.MustRead.WhyMustRead, digestCitationMap)
	}

	// Build the digest draft
	digest := &core.Digest{
		ID:              uuid.NewString(),
		Title:           digestContent.Title,
		TLDRSummary:     digestContent.TLDRSummary,
		TopDevelopments:  digestContent.TopDevelopments,
		WhyItMatters:    digestContent.WhyItMatters,
		KeyMoments:      digestContent.KeyMoments,
		Perspectives:    digestContent.Perspectives,
		ArticleCount:    len(articles),
		ProcessedDate:   time.Now(),
	}

	// Convert ByTheNumbers
	for _, stat := range digestContent.ByTheNumbers {
		digest.ByTheNumbers = append(digest.ByTheNumbers, core.Statistic{
			Stat:    stat.Stat,
			Context: stat.Context,
		})
	}

	// Convert MustRead
	if digestContent.MustRead != nil {
		digest.MustRead = &core.MustReadHighlight{
			ArticleNum:  digestContent.MustRead.ArticleNum,
			Title:       digestContent.MustRead.Title,
			WhyMustRead: digestContent.MustRead.WhyMustRead,
			ReadTime:    digestContent.MustRead.ReadTime,
		}
	}

	// Build article groups from clusters
	for _, c := range clusters {
		var clusterArticles []core.Article
		for _, aid := range c.ArticleIDs {
			if a, ok := articles[aid]; ok {
				clusterArticles = append(clusterArticles, a)
			}
		}
		digest.ArticleGroups = append(digest.ArticleGroups, core.ArticleGroup{
			Theme:            c.Label,
			Articles:         clusterArticles,
			ClusterNarrative: c.Narrative,
		})
	}

	memory.SetDigestDraft(digest)
	memory.SetExecutiveSummary(digestContent.Title + ": " + digestContent.TLDRSummary)

	// Store sections for reflect/revise
	memory.SetDigestSection("title", digestContent.Title)
	memory.SetDigestSection("tldr", digestContent.TLDRSummary)
	memory.SetDigestSection("why_it_matters", digestContent.WhyItMatters)
	if len(digestContent.TopDevelopments) > 0 {
		devText := ""
		for _, d := range digestContent.TopDevelopments {
			devText += "- " + d + "\n"
		}
		memory.SetDigestSection("top_developments", devText)
	}

	// Build response
	result := map[string]any{
		"title":            digestContent.Title,
		"tldr_summary":     digestContent.TLDRSummary,
		"top_developments": digestContent.TopDevelopments,
		"why_it_matters":   digestContent.WhyItMatters,
		"article_count":    len(articles),
		"cluster_count":    len(clusters),
	}

	if digestContent.MustRead != nil {
		result["must_read"] = map[string]any{
			"article_num":       digestContent.MustRead.ArticleNum,
			"title":             digestContent.MustRead.Title,
			"why_must_read":     digestContent.MustRead.WhyMustRead,
			"read_time_minutes": digestContent.MustRead.ReadTime,
		}
	}

	byNumbers := make([]map[string]any, 0)
	for _, s := range digestContent.ByTheNumbers {
		byNumbers = append(byNumbers, map[string]any{
			"stat":    s.Stat,
			"context": s.Context,
		})
	}
	result["by_the_numbers"] = byNumbers

	return result, nil
}
