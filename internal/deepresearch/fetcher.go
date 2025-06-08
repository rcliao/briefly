package deepresearch

import (
	"context"
	"fmt"
	"time"

	"briefly/internal/core"
	"briefly/internal/fetch"
	"briefly/internal/store"
)

// Fetcher wraps the existing fetch.FetchArticle function
type Fetcher interface {
	FetchArticle(ctx context.Context, link *core.Link) (*core.Article, error)
}

// BasicFetcher implements Fetcher using the existing fetch package
type BasicFetcher struct{}

// NewFetcher creates a new basic fetcher
func NewFetcher() Fetcher {
	return &BasicFetcher{}
}

// FetchArticle fetches an article using the existing fetch package
func (f *BasicFetcher) FetchArticle(ctx context.Context, link *core.Link) (*core.Article, error) {
	article, err := fetch.FetchArticle(*link)
	if err != nil {
		return nil, err
	}

	// Clean the article HTML
	if err := fetch.CleanArticleHTML(&article); err != nil {
		return nil, fmt.Errorf("failed to clean article HTML: %w", err)
	}

	return &article, nil
}

// ResearchContentFetcher implements ContentFetcher for deep research
type ResearchContentFetcher struct {
	baseFetcher Fetcher
	store       *store.Store
}

// NewResearchContentFetcher creates a new research content fetcher
func NewResearchContentFetcher(baseFetcher Fetcher, store *store.Store) *ResearchContentFetcher {
	return &ResearchContentFetcher{
		baseFetcher: baseFetcher,
		store:       store,
	}
}

// FetchContent retrieves and processes content from a URL for research purposes
func (f *ResearchContentFetcher) FetchContent(ctx context.Context, url string, useJS bool) (*core.Article, error) {
	// First check if we have this content cached
	if article, err := f.store.GetArticleByURL(url); err == nil && article != nil {
		// Check if cache is still fresh (24 hours for research content)
		if time.Since(article.DateFetched) < 24*time.Hour {
			return article, nil
		}
	}

	// Create a Link object for the fetcher
	link := &core.Link{
		URL:    url,
		Source: "deep_research",
	}

	// Use the base fetcher to get the content
	article, err := f.baseFetcher.FetchArticle(ctx, link)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch article from %s: %w", url, err)
	}

	// Enhance the article with research-specific processing
	if err := f.enhanceForResearch(article); err != nil {
		// Log error but don't fail - we can still use the basic content
		fmt.Printf("Warning: failed to enhance article for research: %v\n", err)
	}

	// Cache the article
	if err := f.store.SaveArticle(article); err != nil {
		// Log error but don't fail
		fmt.Printf("Warning: failed to cache article: %v\n", err)
	}

	return article, nil
}

// enhanceForResearch adds research-specific enhancements to the article
func (f *ResearchContentFetcher) enhanceForResearch(article *core.Article) error {
	// Clean up text for better research quality
	article.CleanedText = f.cleanTextForResearch(article.CleanedText)

	// Ensure we have a title
	if article.Title == "" {
		article.Title = f.extractTitleFromContent(article.CleanedText)
	}

	return nil
}

// cleanTextForResearch performs additional text cleaning specific to research needs
func (f *ResearchContentFetcher) cleanTextForResearch(text string) string {
	// Remove common boilerplate text that appears in articles
	cleanText := text

	// Remove navigation elements, cookie notices, etc.
	// This is a simplified version - could be enhanced with more sophisticated cleaning

	// Remove very short lines (likely navigation or ads)
	lines := make([]string, 0)
	for _, line := range splitLines(cleanText) {
		if len(line) > 20 { // Keep lines with substantial content
			lines = append(lines, line)
		}
	}

	return joinLines(lines)
}

// extractTitleFromContent attempts to extract a title from the content if none exists
func (f *ResearchContentFetcher) extractTitleFromContent(content string) string {
	lines := splitLines(content)
	for _, line := range lines {
		if len(line) > 10 && len(line) < 200 {
			return line
		}
	}
	return "Untitled Article"
}

// Helper functions for text processing
func splitLines(text string) []string {
	var lines []string
	var current string

	for _, char := range text {
		if char == '\n' {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		lines = append(lines, current)
	}

	return lines
}

func joinLines(lines []string) string {
	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}
