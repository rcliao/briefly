package handlers

import (
	"briefly/internal/core"
	"briefly/internal/llm"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// editorialArticle is the per-article editorial output from the single triage call.
type editorialArticle struct {
	Citation     int    `json:"citation"`
	DisplayTitle string `json:"display_title"`
	OneLiner     string `json:"one_liner"`
	Intent       string `json:"intent"` // skim | read | deep_dive
}

// editorialTopic is a topic group naming which citations belong to it.
type editorialTopic struct {
	Name      string `json:"name"`
	Emoji     string `json:"emoji"`
	Citations []int  `json:"citations"`
}

// editorialMustRead is the week's top pick.
type editorialMustRead struct {
	Citation int    `json:"citation"`
	Reason   string `json:"reason"`
}

// editorialDigest is the full structured output of the editorial call.
type editorialDigest struct {
	Title            string             `json:"title"`
	ExecutiveSummary string             `json:"executive_summary"`
	MustRead         *editorialMustRead `json:"must_read"`
	Topics           []editorialTopic   `json:"topics"`
	Articles         []editorialArticle `json:"articles"`
}

// failedLink records a curated URL that never made it into the digest.
type failedLink struct {
	URL    string
	Reason string
}

// siteSuffixRegex matches trailing site-name boilerplate like " · GitHub",
// " | Anthropic", " — Google DeepMind" when the suffix is short (≤4 words).
var siteSuffixRegex = regexp.MustCompile(`\s*[\|·—–\\-]\s*[^|·—–\\-]{1,40}$`)

// cleanArticleTitle strips common site boilerplate from raw <title> text.
// The editorial LLM writes better display titles; this is the fallback and
// keeps the prompt input tidy.
func cleanArticleTitle(title string) string {
	t := strings.TrimSpace(title)
	t = strings.Join(strings.Fields(t), " ") // collapse whitespace/newlines

	// "GitHub - owner/repo: description" → "owner/repo: description"
	t = strings.TrimPrefix(t, "GitHub - ")

	// Repeatedly strip short trailing "· Site" / "| Site" segments, but never
	// strip a title down to almost nothing.
	for i := 0; i < 2; i++ {
		stripped := siteSuffixRegex.ReplaceAllString(t, "")
		if stripped == t || len(stripped) < 20 {
			break
		}
		suffix := strings.TrimSpace(t[len(stripped):])
		// Only strip when the suffix looks like a site name (short, no sentence punctuation)
		if len(strings.Fields(suffix)) > 5 || strings.ContainsAny(suffix, ".!?") {
			break
		}
		t = strings.TrimSpace(stripped)
	}
	return t
}

// buildEditorialPrompt assembles the single editorial/triage prompt.
// Citation numbers are 1-based positions in the articles slice.
func buildEditorialPrompt(articles []core.Article, summaries map[string]*core.Summary) string {
	var b strings.Builder

	b.WriteString(`You are the editor of a weekly GenAI/engineering digest read by busy senior tech professionals. They spend under 5 minutes on it: they scan, then click only what matters to them.

Below are this week's articles, numbered [1..N]. Produce the complete editorial structure as JSON.

RULES:
- "title": digest headline, ≤60 chars, specific (name the biggest concrete development, no hype words like "revolutionary").
- "executive_summary": 2-3 sentences on the week's throughline, citing articles as [N]. Concrete nouns and numbers over vague claims.
- "must_read": the single highest-value article for this audience (prefer practical/technical depth over announcements) and one sentence on why it earns their time.
- "topics": 2-5 groups by SUBJECT MATTER (e.g. "Agentic Engineering", "Model Releases", "Security"). Every citation appears in exactly one topic. Order topics by importance. Pick a fitting emoji per topic.
- "articles": for EVERY citation:
  - "display_title": cleaned title, ≤80 chars, strip site names/boilerplate.
  - "one_liner": ONE sentence with the specific takeaway — what happened or what the reader can do with it, not "This article discusses...". Example: "Cloudflare sandboxes agents 100x faster by reusing V8 isolates instead of containers."
  - "intent": "skim" (awareness is enough), "read" (worth the full read for builders), or "deep_dive" (long/specialist material).

Respond with ONLY the JSON object, no markdown fences, matching:
{"title": "...", "executive_summary": "...", "must_read": {"citation": 1, "reason": "..."}, "topics": [{"name": "...", "emoji": "...", "citations": [1]}], "articles": [{"citation": 1, "display_title": "...", "one_liner": "...", "intent": "read"}]}

ARTICLES:

`)

	for i, article := range articles {
		summaryText := ""
		if s, ok := summaries[article.ID]; ok {
			summaryText = s.SummaryText
		}
		if len(summaryText) > 700 {
			summaryText = summaryText[:700] + "..."
		}
		b.WriteString(fmt.Sprintf("[%d] %s\nURL: %s\nRead time: %d min\nSummary: %s\n\n",
			i+1, cleanArticleTitle(article.Title), article.URL, article.EstimatedReadMinutes, summaryText))
	}

	return b.String()
}

// generateEditorialDigest runs the single editorial LLM call and validates the result.
func generateEditorialDigest(ctx context.Context, llmClient *llm.Client, articles []core.Article, summaries map[string]*core.Summary) (*editorialDigest, error) {
	prompt := buildEditorialPrompt(articles, summaries)

	raw, err := llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		Temperature: 0.3,
		MaxTokens:   8192,
	})
	if err != nil {
		return nil, fmt.Errorf("editorial call failed: %w", err)
	}

	// Strip markdown fences if the model added them anyway
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var digest editorialDigest
	if err := json.Unmarshal([]byte(raw), &digest); err != nil {
		return nil, fmt.Errorf("editorial response is not valid JSON: %w", err)
	}

	normalizeEditorialDigest(&digest, articles, summaries)
	return &digest, nil
}

// normalizeEditorialDigest enforces the invariants the renderer relies on:
// every citation 1..N appears in exactly one topic and has an article entry.
func normalizeEditorialDigest(d *editorialDigest, articles []core.Article, summaries map[string]*core.Summary) {
	n := len(articles)
	valid := func(c int) bool { return c >= 1 && c <= n }

	// Deduplicate citations across topics, drop out-of-range ones
	assigned := make(map[int]bool)
	topics := d.Topics[:0]
	for _, topic := range d.Topics {
		var kept []int
		for _, c := range topic.Citations {
			if valid(c) && !assigned[c] {
				assigned[c] = true
				kept = append(kept, c)
			}
		}
		if len(kept) > 0 {
			topic.Citations = kept
			topics = append(topics, topic)
		}
	}
	d.Topics = topics

	// Any unassigned article lands in a catch-all topic
	var orphans []int
	for c := 1; c <= n; c++ {
		if !assigned[c] {
			orphans = append(orphans, c)
		}
	}
	if len(orphans) > 0 {
		d.Topics = append(d.Topics, editorialTopic{Name: "Also Notable", Emoji: "📌", Citations: orphans})
	}

	// Ensure every citation has an article entry; fill gaps from raw data
	entries := make(map[int]editorialArticle)
	for _, a := range d.Articles {
		if valid(a.Citation) {
			entries[a.Citation] = a
		}
	}
	for c := 1; c <= n; c++ {
		entry, ok := entries[c]
		if !ok {
			entry = editorialArticle{Citation: c}
		}
		article := articles[c-1]
		if strings.TrimSpace(entry.DisplayTitle) == "" {
			entry.DisplayTitle = cleanArticleTitle(article.Title)
		}
		if strings.TrimSpace(entry.OneLiner) == "" {
			if s, ok := summaries[article.ID]; ok {
				entry.OneLiner = firstSentence(s.SummaryText)
			}
		}
		switch entry.Intent {
		case "skim", "read", "deep_dive":
		default:
			entry.Intent = "read"
		}
		entries[c] = entry
	}
	d.Articles = make([]editorialArticle, 0, n)
	for c := 1; c <= n; c++ {
		d.Articles = append(d.Articles, entries[c])
	}
	sort.Slice(d.Articles, func(i, j int) bool { return d.Articles[i].Citation < d.Articles[j].Citation })

	if d.MustRead != nil && !valid(d.MustRead.Citation) {
		d.MustRead = nil
	}
	if strings.TrimSpace(d.Title) == "" {
		d.Title = fmt.Sprintf("This Week in GenAI — %d Links", n)
	}
}

// citationRegex matches [N] citation markers in editorial prose.
var citationRegex = regexp.MustCompile(`\s*\[(\d+)\]`)

// stripCitations removes [N] markers from prose. The editorial prompt asks
// for citations because they keep the summary grounded, but the rendered
// digest is pasted into places that don't render markdown, so bare [N]
// markers with no numbered list are just noise.
func stripCitations(text string) string {
	return strings.TrimSpace(citationRegex.ReplaceAllString(text, ""))
}

// firstSentence returns the first sentence of text, capped at 200 chars.
func firstSentence(text string) string {
	text = strings.TrimSpace(text)
	if idx := strings.IndexAny(text, ".!?"); idx > 20 && idx < len(text)-1 {
		text = text[:idx+1]
	}
	if len(text) > 200 {
		text = text[:200] + "..."
	}
	return text
}

// intentTag renders the reader-intent as an inline tag, not a section.
func intentTag(intent string) string {
	switch intent {
	case "skim":
		return "⚡ skim"
	case "deep_dive":
		return "🔬 deep dive"
	default:
		return "" // "read" is the default expectation; no tag needed
	}
}

// renderEditorialDigest renders the final digest. The output is optimized for
// pasting into LinkedIn/Slack, which don't render markdown: URLs are shown as
// plain text on their own line, nothing is indented, and there are no
// horizontal rules. Blank lines and emoji do the visual separation.
func renderEditorialDigest(d *editorialDigest, articles []core.Article, failed []failedLink, generatedAt time.Time) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("🗞️ %s\n\n", d.Title))
	b.WriteString(fmt.Sprintf("%d links this week · %s\n\n", len(articles), generatedAt.Format("January 2, 2006")))

	if summary := stripCitations(d.ExecutiveSummary); summary != "" {
		b.WriteString(summary)
		b.WriteString("\n\n")
	}

	if d.MustRead != nil {
		article := articles[d.MustRead.Citation-1]
		entry := d.Articles[d.MustRead.Citation-1]
		b.WriteString(fmt.Sprintf("🎯 Must-read: %s — %s 📖 %d min\n\n%s\n\n",
			entry.DisplayTitle, d.MustRead.Reason, article.EstimatedReadMinutes, article.URL))
	}

	for _, topic := range d.Topics {
		emoji := topic.Emoji
		if emoji == "" {
			emoji = "📌"
		}
		b.WriteString(fmt.Sprintf("%s %s\n\n", emoji, strings.ToUpper(topic.Name)))

		for _, c := range topic.Citations {
			article := articles[c-1]
			entry := d.Articles[c-1]

			line := fmt.Sprintf("%s · 📖 %d min", entry.DisplayTitle, article.EstimatedReadMinutes)
			if tag := intentTag(entry.Intent); tag != "" {
				line += " · " + tag
			}
			b.WriteString(line + "\n\n")
			if entry.OneLiner != "" {
				b.WriteString(entry.OneLiner + "\n\n")
			}
			b.WriteString(article.URL + "\n\n")
		}
	}

	if len(failed) > 0 {
		b.WriteString(fmt.Sprintf("⚠️ %d link(s) could not be included:\n", len(failed)))
		for _, f := range failed {
			b.WriteString(fmt.Sprintf("%s — %s\n", f.URL, f.Reason))
		}
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("Generated by Briefly on %s\n", generatedAt.Format("2006-01-02")))

	return b.String()
}
