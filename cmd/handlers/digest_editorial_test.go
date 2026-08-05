package handlers

import (
	"briefly/internal/core"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCleanArticleTitle(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"GitHub - openai/codex-security: OpenAI's Codex Security CLI · GitHub", "openai/codex-security: OpenAI's Codex Security CLI"},
		{"Introducing Claude Opus 5 \\ Anthropic", "Introducing Claude Opus 5"},
		{"The 2026-07-28 Specification | Model Context Protocol Blog", "The 2026-07-28 Specification"},
		{"Plain Title With No Suffix", "Plain Title With No Suffix"},
		{"  Whitespace\n\tEverywhere  ", "Whitespace Everywhere"},
	}
	for _, c := range cases {
		if got := cleanArticleTitle(c.in); got != c.want {
			t.Errorf("cleanArticleTitle(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func testArticles(n int) []core.Article {
	articles := make([]core.Article, n)
	for i := range articles {
		articles[i] = core.Article{
			ID:                   string(rune('a' + i)),
			URL:                  "https://example.com/" + string(rune('a'+i)),
			Title:                "Article " + string(rune('A'+i)),
			EstimatedReadMinutes: i + 1,
		}
	}
	return articles
}

func testSummaries(articles []core.Article) map[string]*core.Summary {
	m := make(map[string]*core.Summary)
	for i := range articles {
		m[articles[i].ID] = &core.Summary{
			ArticleIDs:  []string{articles[i].ID},
			SummaryText: "Summary sentence for " + articles[i].Title + ". More detail follows here.",
		}
	}
	return m
}

func TestNormalizeEditorialDigestInvariants(t *testing.T) {
	articles := testArticles(5)
	summaries := testSummaries(articles)

	// LLM output with problems: duplicate citation, out-of-range citation,
	// missing citations 4 and 5, missing article entries, bad intent.
	d := &editorialDigest{
		Title: "Test Digest",
		Topics: []editorialTopic{
			{Name: "Topic A", Citations: []int{1, 2, 2, 99}},
			{Name: "Topic B", Citations: []int{3}},
		},
		Articles: []editorialArticle{
			{Citation: 1, DisplayTitle: "One", OneLiner: "Takeaway one.", Intent: "banana"},
		},
		MustRead: &editorialMustRead{Citation: 42, Reason: "out of range"},
	}
	normalizeEditorialDigest(d, articles, summaries)

	// Every citation 1..5 appears in exactly one topic
	seen := map[int]int{}
	for _, topic := range d.Topics {
		for _, c := range topic.Citations {
			seen[c]++
		}
	}
	for c := 1; c <= 5; c++ {
		if seen[c] != 1 {
			t.Errorf("citation %d appears %d times in topics, want exactly 1", c, seen[c])
		}
	}
	if seen[99] != 0 {
		t.Errorf("out-of-range citation 99 survived normalization")
	}

	// Every article has an entry with title, one-liner, and valid intent
	if len(d.Articles) != 5 {
		t.Fatalf("got %d article entries, want 5", len(d.Articles))
	}
	for _, a := range d.Articles {
		if a.DisplayTitle == "" {
			t.Errorf("citation %d: empty display title", a.Citation)
		}
		if a.OneLiner == "" {
			t.Errorf("citation %d: empty one-liner", a.Citation)
		}
		if a.Intent != "skim" && a.Intent != "read" && a.Intent != "deep_dive" {
			t.Errorf("citation %d: invalid intent %q", a.Citation, a.Intent)
		}
	}

	// Out-of-range must-read is dropped
	if d.MustRead != nil {
		t.Errorf("out-of-range must_read survived normalization")
	}
}

func TestRenderEditorialDigest(t *testing.T) {
	articles := testArticles(2)
	summaries := testSummaries(articles)
	d := &editorialDigest{
		Title:            "Weekly Digest",
		ExecutiveSummary: "Big week for agents [1] and models [2].",
		MustRead:         &editorialMustRead{Citation: 1, Reason: "Worth your time."},
		Topics: []editorialTopic{
			{Name: "Agents", Emoji: "🤖", Citations: []int{1, 2}},
		},
	}
	normalizeEditorialDigest(d, articles, summaries)

	failed := []failedLink{{URL: "https://blocked.example.com", Reason: "fetch failed: status code 403"}}
	out := renderEditorialDigest(d, articles, failed, time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC))

	for _, want := range []string{
		"🗞️ Weekly Digest",
		"🎯 Must-read:",
		"🤖 AGENTS",
		"📖 1 min",
		"https://example.com/a",           // plain URL on its own line
		"https://blocked.example.com",     // failure footer
		"Big week for agents and models.", // citations stripped from prose
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered digest missing %q\n---\n%s", want, out)
		}
	}

	// Paste-friendly output: no markdown links, no indentation, no rules
	for _, banned := range []string{"](", "---", "\n  ", "**", "# "} {
		if strings.Contains(out, banned) {
			t.Errorf("rendered digest contains banned sequence %q\n---\n%s", banned, out)
		}
	}
}

func TestSelectBannerSubject(t *testing.T) {
	subjects := []string{"first scene", "second scene", "third scene"}

	// Non-interactive stdin (a pipe/file) must default to the first subject
	r, w, _ := os.Pipe()
	_ = w.Close()
	if got := selectBannerSubject(subjects, r); got != "first scene" {
		t.Errorf("non-interactive selection = %q, want first subject", got)
	}
	_ = r.Close()

	// Single subject short-circuits
	if got := selectBannerSubject([]string{"only"}, nil); got != "only" {
		t.Errorf("single-subject selection = %q, want it returned directly", got)
	}
}
