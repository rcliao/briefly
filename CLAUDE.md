# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What Briefly Is

A CLI that turns a hand-curated markdown file of URLs into a scannable weekly digest for busy tech professionals, generated with Gemini. One user, one workflow: collect links during the week, run one command, paste the result into LinkedIn/Slack/email.

**Version:** 4.0.0-editorial
**History:** v3 had a database-driven aggregation pipeline (Postgres, RSS feeds, themes, web server, k-means clustering, agentic mode). All of it was removed in August 2026 — see "Removed" below. Do not reintroduce that machinery; the editorial pipeline replaced it deliberately.

## Commands

```bash
# The weekly workflow
briefly digest from-file input/2026-08-03.md      # → digests/digest_YYYY-MM-DD.md
briefly digest from-file input/weekly.md --format slack
briefly digest from-file input/weekly.md --no-cache

# Quick single-article summary
briefly read https://example.com/article
briefly read --raw https://example.com/article

# Cache management (SQLite in .briefly-cache/)
briefly cache stats
briefly cache clear --confirm
```

**Input format:** a markdown file with one URL per line (bare URLs or markdown links).

**Required env:** `GEMINI_API_KEY` (env or `.env`). Optional: `BRIEFLY_LOG_LEVEL` (debug|info|warn|error, default warn — logs go to stderr as JSON, stdout stays clean for output).

## Development

```bash
go build -o briefly ./cmd/briefly
make test     # go test -race ./...
make lint     # golangci-lint (pinned config in .golangci.yml, same as CI)
make fmt      # gofmt -w .
```

CI (.github/workflows/test.yml) runs test/build/lint on the Go version from go.mod. Lint must stay at zero findings.

## Architecture

### The pipeline (5 steps, `cmd/handlers/digest_from_file.go`)

1. **Parse** — extract URLs from the markdown file (`internal/parser`)
2. **Fetch** — parallel (4 workers), browser UA + one retry, 24h SQLite cache; cached entries are re-extracted from stored HTML so parser fixes apply retroactively; failures are collected, never silently dropped (`internal/fetch`, `internal/store`)
3. **Summarize** — parallel per-article summaries, 12k-char content window (`internal/summarize`)
4. **Editorial pass** — ONE LLM call (`cmd/handlers/digest_editorial.go`) returns title, executive summary, must-read pick, 2-5 topic groups, and per-article display title + one-liner takeaway + intent tag. `normalizeEditorialDigest` enforces invariants (every article in exactly one topic, entries backfilled from summaries) with a deterministic fallback — a bad LLM response degrades to a plainer digest, never a broken one.
5. **Render** — paste-friendly plain text: bare URLs on their own line, no markdown links/bold/headers/rules, `EMOJI UPPERCASE` topic headers, blank lines between title/description/URL, failed links listed in a footer. Read times are computed from cleaned word count (~200 wpm), never LLM-estimated.

The Slack format (`--format slack`) feeds editorial topics into `internal/narrative.GenerateSlackDigest` and renders chunked Slack mrkdwn.

### Design principles (learned the hard way)

- **Judgment in the model, structure in the code.** The LLM names topics and writes takeaways; the renderer and normalization code guarantee the format. An agentic reflect/revise loop was tried and removed — it iterated on prose while scannability is structural.
- **Fail loudly, degrade gracefully.** Fetch failures appear in the digest footer and CLI output. LLM failures fall back to deterministic structure.
- **The digest is pasted into places that don't render markdown.** Never add markdown links, bold, or horizontal rules to the renderer.

### Package map

```
cmd/briefly/          Entry point
cmd/handlers/         root, digest (from-file), read, cache + editorial pass & renderer
internal/core/        Data structures (Article, Summary, Link, ...)
internal/parser/      URL extraction from markdown
internal/fetch/       HTML/PDF/YouTube fetching + content extraction + read time
internal/summarize/   Article summarization prompts + summarizer
internal/narrative/   Slack digest generation (+ legacy digest content types)
internal/llm/         Gemini client (google.golang.org/genai)
internal/store/       SQLite cache (articles, summaries)
internal/config/      Viper config (.briefly.yaml + env)
internal/logger/      slog JSON to stderr, BRIEFLY_LOG_LEVEL
```

Tests: `cmd/handlers` (editorial normalization/rendering), `parser`, `summarize`, `core`, `llm`, `fetch`, `store`.

## Removed (August 2026) — do not resurrect without discussion

The entire database/server half (~20k lines across two cleanup waves): Postgres persistence + migrations, web server + UI, RSS aggregation (`aggregate`, `classify`, `feed`, `theme`, `url`, `serve`, `migrate`, `search`, `quality` commands), k-means/Louvain/HDBSCAN clustering, embeddings + pgvector, themes/tags classification, citations tracking, quality metrics, PostHog/LangFuse observability, the agentic digest mode, and Docker/postgres infra. `DATABASE_URL` is no longer used anywhere.

Rationale: the weekly curated workflow never touched any of it, and it dominated the maintenance surface (untested god functions, stubbed interfaces, three inconsistent taxonomies). If aggregation is ever wanted again, build it as a separate feed-suggestion tool that appends to the input file, not as a parallel digest pipeline.

## Known gaps / future work

- Summaries are not cached (re-runs re-summarize; ~30s of the ~45s runtime). Cache on article content hash in `internal/store`.
- `internal/llm` methods mostly create `context.Background()` internally instead of accepting ctx — LLM calls aren't cancellable.
- `internal/narrative` still carries legacy digest-content types beyond the Slack path; slimming it is safe once Slack format is confirmed stable.
- No sentinel errors / `errors.Is` usage; some tests string-match error messages.
