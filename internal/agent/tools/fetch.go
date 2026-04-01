package tools

import (
	"briefly/internal/agent"
	"briefly/internal/core"
	"briefly/internal/fetch"
	"briefly/internal/store"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/genai"
)

// FetchArticlesTool wraps the existing fetch pipeline to retrieve articles from URLs in a markdown file.
type FetchArticlesTool struct {
	cache *store.Store
}

// NewFetchArticlesTool creates a new fetch articles tool.
func NewFetchArticlesTool(cache *store.Store) *FetchArticlesTool {
	return &FetchArticlesTool{cache: cache}
}

func (t *FetchArticlesTool) Name() string { return "fetch_articles" }

func (t *FetchArticlesTool) Description() string {
	return "Fetch and parse articles from a markdown file containing URLs. Returns fetched articles with their content. Uses cache when available. Call this first to load the article corpus."
}

func (t *FetchArticlesTool) Parameters() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"input_file": {
				Type:        genai.TypeString,
				Description: "Path to markdown file with URLs",
			},
			"use_cache": {
				Type:        genai.TypeBoolean,
				Description: "Whether to use cached content (default: true)",
			},
		},
		Required: []string{"input_file"},
	}
}

func (t *FetchArticlesTool) Execute(ctx context.Context, memory *agent.WorkingMemory, params map[string]any) (map[string]any, error) {
	inputFile := extractStringParam(params, "input_file", "")
	if inputFile == "" {
		return nil, fmt.Errorf("input_file is required")
	}
	useCache := extractBoolParam(params, "use_cache", true)

	// Parse URLs from the file
	links, err := fetch.ReadLinksFromFile(inputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse input file: %w", err)
	}
	if len(links) == 0 {
		return nil, fmt.Errorf("no URLs found in %s", inputFile)
	}

	totalURLs := len(links)
	var successful, failed, cacheHits int
	var failedURLs []string
	articles := make(map[string]core.Article)
	articleList := make([]map[string]any, 0)

	cacheTTL := 7 * 24 * time.Hour
	processor := fetch.NewContentProcessor()

	for _, link := range links {
		// Try cache first
		if useCache && t.cache != nil {
			cachedArticle, err := t.cache.GetCachedArticle(link.URL, cacheTTL)
			if err == nil && cachedArticle != nil {
				articles[cachedArticle.ID] = *cachedArticle
				cacheHits++
				successful++
				articleList = append(articleList, map[string]any{
					"id":              cachedArticle.ID,
					"url":             cachedArticle.URL,
					"title":           cachedArticle.Title,
					"content_type":    string(cachedArticle.ContentType),
					"content_preview": truncateStr(cachedArticle.CleanedText, 200),
					"word_count":      len(cachedArticle.CleanedText) / 5,
				})
				continue
			}
		}

		// Fetch fresh using ContentProcessor
		article, fetchErr := processor.ProcessArticle(ctx, link.URL)
		if fetchErr != nil {
			failed++
			failedURLs = append(failedURLs, link.URL)
			continue
		}

		if article.ID == "" {
			article.ID = uuid.NewString()
		}
		if article.URL == "" {
			article.URL = link.URL
		}

		// Cache the fetched article
		if useCache && t.cache != nil {
			_ = t.cache.CacheArticle(*article)
		}

		articles[article.ID] = *article
		successful++
		articleList = append(articleList, map[string]any{
			"id":              article.ID,
			"url":             article.URL,
			"title":           article.Title,
			"content_type":    string(article.ContentType),
			"content_preview": truncateStr(article.CleanedText, 200),
			"word_count":      len(article.CleanedText) / 5,
		})
	}

	if successful == 0 {
		return nil, fmt.Errorf("all %d URLs failed to fetch", totalURLs)
	}

	memory.SetArticles(articles)
	memory.BuildArticleIndex() // Build stable citation index [1], [2], ... for all tools

	return map[string]any{
		"articles":    articleList,
		"total_urls":  totalURLs,
		"successful":  successful,
		"failed":      failed,
		"cache_hits":  cacheHits,
		"failed_urls": failedURLs,
	}, nil
}
