**Product Requirements Document (PRD)**
**Feature:** `briefly deep-research` command
**Owner:** Eric / Briefly Core
**Last updated:** 2025-06-05

---

## 1. Purpose & Background

Weekly AI digests are only as good as the source material. Today the **briefly** workflow still relies on manual link-hunting or one-shot search queries. A “deep research” agent will:

* break a broad topic into sub-questions automatically,
* scout diverse, fresh sources (news, papers, blog posts, repos),
* deduplicate / rank content,
* output a well-cited research brief that can be further refined in the chat-TUI, and
* feed the brief straight into the existing `digest create` flow.

This feature closes the gap between “find content” and “summarise content” while keeping the human reviewer in the loop for final polish.

---

## 2. Goals & Non-Goals

| #   | Goal                                                                                          | Metric / Exit-Criteria                                                                |
| --- | --------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------- |
| G1  | Generate a multi-section research brief (< 2 min median wall-clock) for a user-supplied topic | ✔ Brief returned with ≥ 5 unique sources and at least one peer-reviewed or arXiv item |
| G2  | Provide inline `[n]` citations that resolve to URLs in a `sources` block                      | ✔ 100 % of factual sentences have a citation; broken-link rate < 2 %                  |
| G3  | Allow iterative chat refinement without re-scraping unless explicitly requested               | ✔ Follow-up chat latency < 3 s (uses cached notes)                                    |
| NG1 | Does *not* attempt to evaluate factual correctness beyond source variety                      |                                                                                       |
| NG2 | Does *not* build a full-text search UI (out of scope for MVP)                                 |                                                                                       |

---

## 3. User Stories

* **U1 – Curious Engineer**
  “As a subscriber, I want to run `briefly deep-research "open-source agent frameworks"` so I receive a concise, cited brief I can drop into my weekly newsletter.”
* **U2 – Newsletter Author**
  “After reading the brief, I want to ask follow-up questions (`briefly chat`) and update the brief in place without re-scraping every source.”
* **U3 – Cron Job**
  “As an automation, I want to schedule deep-research presets every Sunday night and commit new briefs to the repo so Monday’s digest builds automatically.”

---

## 4. Functional Requirements

| ID                                               | Requirement                                                                                                                      |
| ------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------- |
| F-1                                              | New CLI verb: `briefly deep-research <topic> [flags]`                                                                            |
| F-2                                              | `--since` (days) and `--max-sources` flags gate recency & size                                                                   |
| F-3                                              | Planner step: LLM decomposes `<topic>` into 3-7 sub-questions (JSON)                                                             |
| F-4                                              | For each sub-question, agent calls Search API (DuckDuckGo, SerpAPI) with recency filter; top-N URLs pulled                       |
| F-5                                              | Fetcher downloads pages, strips boiler-plate (`go-readability`) and stores `(url, sha256, retrieved_at, text)` in SQLite cache   |
| F-6                                              | Embedding ranker (MiniLM-all-v2) scores relevance; keep top-k unique sources overall                                             |
| F-7                                              | RAG prompt synthesises `executive_summary`, `detailed_findings`, `open_questions`, and `sources[]` with inline numeric citations |
| F-8                                              | Output targets:                                                                                                                  |
|   a) stdout Markdown,                            |                                                                                                                                  |
|   b) `research/<slug>.json` raw artefact,        |                                                                                                                                  |
|   c) optional `--html` 👉 `research/<slug>.html` |                                                                                                                                  |
| F-9                                              | `briefly chat <slug>` opens the paper & cached notes in the chat-TUI for iterative Q\&A                                          |
| F-10                                             | `--refresh` forces re-scrape, bypassing cache                                                                                    |
| F-11                                             | Errors (network, parse, 5xx) collected and shown at bottom of brief; non-fatal failures degrade gracefully                       |

---

## 5. Non-Functional Requirements

* **Performance:** 90-percentile runtime < 2 min with default `max-sources=25`.
* **Cost:** ≤ \$0.10 per run assuming Gemini 1.5-Pro planning & synthesis; embedding via open-source model.
* **Observability:** Structured logs for each agent step, Prometheus counters (`scrape_errors_total`, `llm_tokens_total`).
* **Security & Compliance:** Honor `robots.txt`; redact cookies / PII before storing raw pages.
* **Extensibility:** Planner & ranker interfaces defined as Go interfaces so future models or heuristics can swap in.

---

## 6. Technical Approach (MVP)

```mermaid
flowchart TD
    A[CLI Input] --> B[Planner (LLM)]
    B --> C{Sub-questions}
    C --> D[Search API]
    D --> E[Fetcher + Cleaner]
    E --> F[Cache]
    F --> G[Embedding Ranker]
    G --> H[Synthesis (LLM RAG)]
    H --> I[Markdown / JSON / HTML]
    I --> J[TUI Chat Loop]
```

* **Planner Prompt** stored in `prompts/planner.tmpl`.
* **Search Adapter** initially DuckDuckGo HTML scrape; feature-flag SerpAPI key if set.
* **Fetcher** uses `chromedp` headless browser when `--javascript` flag passed.
* **Embedding** via `sentence-transformers/all-MiniLM-L6-v2` served from local ONNX.
* **Synthesis Prompt** in `prompts/synth.tmpl`, max 5 k tokens.

---

## 7. CLI & Config Spec

```bash
briefly deep-research "topic string" \
  --since 21d \
  --max-sources 30 \
  --html \
  --model gemini-1.5-pro \
  --refresh
```

* Global `briefly.yml` gains a `deep_research` section:

  ```yaml
  search:
    provider: duckduckgo      # or serpapi
    serpapi_key: ENV:SERP_KEY
  embedding_model: all-MiniLM-L6-v2
  llm_model: gemini-1.5-pro
  cache_db: ~/.briefly/cache.db
  ```

---

## 8. Success Metrics & Analytics

| Metric                   | Target                                 | Collection             |
| ------------------------ | -------------------------------------- | ---------------------- |
| Mean generation time     | < 120 s                                | CLI timer              |
| Avg. citations per brief | ≥ 8                                    | Count in JSON          |
| Broken citation ratio    | < 2 %                                  | Nightly link-check job |
| Follow-up chat adoption  | ≥ 40 % of briefs receive ≥ 1 chat turn | TUI telemetry          |

---

## 9. Risks & Mitigations

| Risk                             | Impact            | Mitigation                                                      |
| -------------------------------- | ----------------- | --------------------------------------------------------------- |
| Search engine blocks scraping    | Brief fails       | Respect rate limits; random back-off; allow SerpAPI             |
| LLM cost spikes                  | Budget blowout    | Default to smaller model (`gemini-pro`) with `--model` override |
| Context overflow on large topics | Generation errors | Ranker hard-caps tokens; spill extra sources to appendix        |

---

## 10. Milestones

| Date   | Milestone                                           |
| ------ | --------------------------------------------------- |
| Jun 13 | Core planner + fetcher prototype returns JSON notes |
| Jun 20 | End-to-end brief in Markdown with citations         |
| Jun 27 | TUI chat integration & caching                      |
| Jul 04 | Beta release to internal users                      |
| Jul 11 | GA + scheduled presets                              |

---

## 11. Open Questions

1. Should we support PDF ingestion in MVP or hold for v2?
  - E: v2
2. Is `chromedp` + headless Chrome sufficient for paywalls, or integrate boilerpipe-style heuristics?
  - E: headless chrome is okay starting point. we can consider integrate with Browserbase at v2
3. Licensing—confirm MiniLM weights are okay for commercial newsletter use.
  - E: I'm not running commercial newsletter. should be okay
4. Where to surface “source diversity” warnings if all hits come from e.g. a single corporate blog?
  - Serve as a terminal alert if all researches hits the same corporate blog
