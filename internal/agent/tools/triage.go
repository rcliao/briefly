package tools

import (
	"briefly/internal/agent"
	"briefly/internal/llm"
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/genai"
)

// TriageArticlesTool scores articles and assigns topic categories, reader intent, and editorial summaries.
type TriageArticlesTool struct {
	llmClient *llm.Client
}

func NewTriageArticlesTool(llmClient *llm.Client) *TriageArticlesTool {
	return &TriageArticlesTool{llmClient: llmClient}
}

func (t *TriageArticlesTool) Name() string { return "triage_articles" }

func (t *TriageArticlesTool) Description() string {
	return "Score each article for relevance, assign a topic category, reader intent (skim/read/deep_dive), and write a 1-2 sentence editorial summary. This is the primary classification step — topic assignments determine how articles are grouped in the final digest."
}

func (t *TriageArticlesTool) Parameters() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"article_ids": {
				Type:        genai.TypeArray,
				Items:       &genai.Schema{Type: genai.TypeString},
				Description: "IDs of articles to triage. Omit to triage all articles.",
			},
		},
	}
}

func (t *TriageArticlesTool) Execute(ctx context.Context, memory *agent.WorkingMemory, params map[string]any) (map[string]any, error) {
	articles := memory.GetArticles()
	summaries := memory.GetSummaries()
	articleIndex := memory.GetArticleIndex()
	targetIDs := extractStringSliceParam(params, "article_ids")

	type articleInfo struct {
		ID          string
		CitationNum int
		Title       string
		Summary     string
		URL         string
	}
	var articleInfos []articleInfo

	if len(targetIDs) > 0 {
		for _, id := range targetIDs {
			if a, ok := articles[id]; ok {
				summaryText := ""
				if s, ok := summaries[id]; ok {
					summaryText = s.SummaryText
				}
				articleInfos = append(articleInfos, articleInfo{
					ID: id, CitationNum: memory.GetCitationNum(id),
					Title: a.Title, Summary: truncateStr(summaryText, 300), URL: a.URL,
				})
			}
		}
	} else {
		for _, entry := range articleIndex {
			summaryText := ""
			if s, ok := summaries[entry.ArticleID]; ok {
				summaryText = s.SummaryText
			}
			articleInfos = append(articleInfos, articleInfo{
				ID: entry.ArticleID, CitationNum: entry.CitationNum,
				Title: entry.Title, Summary: truncateStr(summaryText, 300), URL: entry.URL,
			})
		}
	}

	if len(articleInfos) == 0 {
		return map[string]any{"scores": []any{}, "include_count": 0, "deprioritize_count": 0, "exclude_count": 0}, nil
	}

	topicList := agent.TopicListForPrompt()

	prompt := fmt.Sprintf(`You are curating a weekly GenAI newsletter for senior software engineers.

For each article, provide:
1. **topic_id**: Which topic category best fits this article? Pick ONE from:
%s
2. **reader_intent**: How should the reader engage?
   - "skim": Quick awareness — headline + 1 sentence is enough
   - "read": Worth 5-10 minutes — practical, actionable content
   - "deep_dive": Bookmark for later — research paper or deep technical analysis
3. **editorial_summary**: Write 1-2 sentences as if you're recommending this link to a colleague. Be specific about WHAT makes it interesting. Use a natural, human voice — not "This article discusses..." but "Cloudflare figured out how to sandbox agents 100x faster by..."
4. **signal_strength**: 0.0-1.0, how newsworthy/important is this?
5. **recommended_action**: "include" or "exclude"
6. **estimated_read_minutes**: How long would it take an engineer to read the main content? Estimate based on the article's depth and content type:
   - Landing pages / product announcements: 1-2 min
   - Blog posts / tutorials: 5-15 min
   - Long-form technical posts: 10-20 min
   - Research papers / deep dives: 20-30 min
   Do NOT count navigation, comments, or sidebars. Only the article's main content.

Articles:
`, topicList)

	for _, info := range articleInfos {
		prompt += fmt.Sprintf("\n[%d] ID: %s\nTitle: %s\nURL: %s\nSummary: %s\n", info.CitationNum, info.ID, info.Title, info.URL, info.Summary)
	}

	schema := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"scores": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"article_id":              {Type: genai.TypeString},
						"topic_id":                {Type: genai.TypeString},
						"reader_intent":           {Type: genai.TypeString},
						"editorial_summary":       {Type: genai.TypeString},
						"signal_strength":         {Type: genai.TypeNumber},
						"recommended_action":      {Type: genai.TypeString},
						"estimated_read_minutes":  {Type: genai.TypeInteger},
					},
					Required: []string{"article_id", "topic_id", "reader_intent", "editorial_summary", "signal_strength", "recommended_action", "estimated_read_minutes"},
				},
			},
		},
		Required: []string{"scores"},
	}

	resp, err := t.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		ResponseSchema: schema,
		Temperature:    0.3,
		MaxTokens:      8192,
	})
	if err != nil {
		return nil, fmt.Errorf("triage LLM call failed: %w", err)
	}

	var result struct {
		Scores []struct {
			ArticleID            string  `json:"article_id"`
			TopicID              string  `json:"topic_id"`
			ReaderIntent         string  `json:"reader_intent"`
			EditorialSummary     string  `json:"editorial_summary"`
			SignalStrength       float64 `json:"signal_strength"`
			RecommendedAction    string  `json:"recommended_action"`
			EstimatedReadMinutes int     `json:"estimated_read_minutes"`
		} `json:"scores"`
	}

	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return nil, fmt.Errorf("failed to parse triage response: %w", err)
	}

	scores := make(map[string]agent.TriageScore)
	var includeCount, excludeCount int
	scoreList := make([]map[string]any, 0, len(result.Scores))

	for _, s := range result.Scores {
		title := ""
		if a, ok := articles[s.ArticleID]; ok {
			title = a.Title
		}

		intent := s.ReaderIntent
		if intent != "skim" && intent != "read" && intent != "deep_dive" {
			intent = "read"
		}

		scores[s.ArticleID] = agent.TriageScore{
			ArticleID:         s.ArticleID,
			Title:             title,
			SignalStrength:    s.SignalStrength,
			RecommendedAction: s.RecommendedAction,
			ReaderIntent:      intent,
			TopicID:           s.TopicID,
			EditorialSummary:  s.EditorialSummary,
		}

		// Update article index with triage results
		memory.SetReaderIntent(s.ArticleID, intent)
		memory.SetArticleTopic(s.ArticleID, s.TopicID)
		memory.SetEditorialSummary(s.ArticleID, s.EditorialSummary)
		if s.EstimatedReadMinutes > 0 {
			memory.SetReadMinutes(s.ArticleID, s.EstimatedReadMinutes)
		}

		switch s.RecommendedAction {
		case "include":
			includeCount++
		case "exclude":
			excludeCount++
		}
		scoreList = append(scoreList, map[string]any{
			"article_id":             s.ArticleID,
			"title":                  title,
			"topic_id":               s.TopicID,
			"reader_intent":          intent,
			"editorial_summary":      s.EditorialSummary,
			"signal_strength":        s.SignalStrength,
			"recommended_action":     s.RecommendedAction,
			"estimated_read_minutes": s.EstimatedReadMinutes,
		})
	}

	memory.SetTriageScores(scores)

	return map[string]any{
		"scores":        scoreList,
		"include_count": includeCount,
		"exclude_count": excludeCount,
	}, nil
}
