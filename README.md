# Briefly

Turn a hand-curated list of URLs into a scannable weekly digest, generated with Gemini.

Built for one workflow: you collect interesting links during the week, run one command, and paste the result into LinkedIn, Slack, or email. The output is plain text that survives paste targets that don't render markdown.

## Example output

```
🗞️ MCP Spec Update and 2.8T Kimi K3 Model Release

9 links this week · August 5, 2026

The Model Context Protocol (MCP) reaches a major milestone with header-based
routing, while the open-weight landscape shifts with the 2.8T parameter
Kimi K3 architecture. ...

🎯 Must-read: Anatomy of a Frontier Lab Agent Intrusion — a rare, highly
technical post-mortem of an autonomous agent attack. 📖 34 min

https://huggingface.co/blog/agent-intrusion-technical-timeline

🛠️ AGENTIC INFRASTRUCTURE & PROTOCOLS

MCP Specification 2026-07-28 · 📖 11 min

The updated protocol introduces multi-round-trip requests and header-based
routing to scale agentic workflows.

https://blog.modelcontextprotocol.io/posts/2026-07-28
...
```

Every article gets a cleaned title, an honest read-time estimate, a one-sentence editorial takeaway, and its URL in plain sight. Links that fail to fetch are listed in a footer — never silently dropped.

## Install

```bash
go install ./cmd/briefly
# or
go build -o briefly ./cmd/briefly
```

Requires a Gemini API key ([get one here](https://aistudio.google.com/apikey)):

```bash
export GEMINI_API_KEY="your-key"   # or put it in .env
```

## Usage

```bash
# 1. Collect links during the week — one URL per line
echo "https://example.com/article" >> input/weekly.md

# 2. Generate the digest (~45s for 10 links)
briefly digest from-file input/weekly.md
# → digests/2026-08-05/digest.md (+ banner.jpg, banner.prompt.txt)

# Slack-optimized format
briefly digest from-file input/weekly.md --format slack

# Quick summary of a single article
briefly read https://example.com/article

# Cache management (SQLite, .briefly-cache/)
briefly cache stats
briefly cache clear --confirm

# Regenerate the banner image for a digest (runs automatically during
# digest generation; pick 1 of 3 suggested prompts, or bring your own)
briefly banner digests/2026-08-05/digest.md
```

## How it works

Five steps: parse URLs → fetch in parallel (browser UA, retry, 24h cache) → summarize each article → **one editorial LLM pass** that picks the title, executive summary, must-read, topic groups, and per-article takeaways → render plain text.

The design principle throughout: *judgment in the model, structure in the code*. The LLM makes editorial decisions; deterministic code guarantees the format, enforces invariants (every article appears in exactly one topic), and degrades gracefully when the LLM misbehaves.

Read-time estimates come from actual cleaned word counts, never from the LLM.

## Configuration

Optional `.briefly.yaml`:

```yaml
ai:
  gemini:
    model: "gemini-3.6-flash"
cache:
  directory: ".briefly-cache"
banner:
  enabled: true
  model: "gemini-3.1-flash-image"
  style: "pixel art style, 16-bit, limited color palette, abstract, minimal clean composition, no text or lettering"
  aspect_ratio: "16:9"
```

`BRIEFLY_LOG_LEVEL` (debug|info|warn|error, default warn) controls JSON logging to stderr; stdout carries only command output.

## Development

```bash
make test   # go test -race ./...
make lint   # golangci-lint, pinned config, zero-findings policy
make fmt
```

## History

v3 was a full news-aggregation platform (Postgres, RSS feeds, theme classification, embeddings + clustering, a web UI, an agentic LLM loop). In August 2026 all of it was removed (~35k lines) in favor of the single curated-digest pipeline — the platform half was never part of the weekly workflow it existed to serve. See `CLAUDE.md` for the architecture notes and the reasoning.
