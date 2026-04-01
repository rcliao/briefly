# Agentic Digest Generation: Verification Criteria

## Table of Contents
- [Component-Level Verification](#component-level-verification)
- [Integration Verification](#integration-verification)
- [Quality and Reflection Verification](#quality-and-reflection-verification)
- [Data Verification](#data-verification)
- [Performance Verification](#performance-verification)
- [Fallback and Resilience Verification](#fallback-and-resilience-verification)
- [Security Verification](#security-verification)

---

## Component-Level Verification

### Agent Orchestrator (`internal/agent/orchestrator.go`)

**Functional:**
- [ ] Creates a valid Gemini conversation with system prompt and tool declarations
- [ ] Sends initial user message with input file path and configuration
- [ ] Correctly parses Gemini function-call responses into tool invocations
- [ ] Executes tool calls and returns results to Gemini in the correct format
- [ ] Continues conversation loop until Gemini produces text without tool calls
- [ ] Respects max_iterations configuration for the reflect/revise loop
- [ ] Stops iterating when quality threshold is met
- [ ] Stops iterating when improvement delta falls below 0.05 (diminishing returns)
- [ ] Falls back to linear pipeline when agent initialization fails
- [ ] Returns AgentDigestResult with complete metadata

**Non-Functional:**
- [ ] Completes within 10-minute timeout for typical corpus (5-20 articles)
- [ ] Handles Gemini API rate limits with exponential backoff
- [ ] Logs every tool call with timing information
- [ ] Does not leak API keys in logs or error messages

### Tool Registry (`internal/agent/registry.go`)

**Functional:**
- [ ] Registers all 11 tools with correct names and schemas
- [ ] Converts tool definitions to `genai.FunctionDeclaration` format correctly
- [ ] Routes tool calls to the correct implementation by name
- [ ] Returns structured error for unknown tool names
- [ ] Validates required parameters before execution
- [ ] Accepts optional parameters with correct defaults

**Non-Functional:**
- [ ] Registration is O(1) lookup by tool name
- [ ] Thread-safe for concurrent reads (tools are registered once at startup)

### Working Memory (`internal/agent/memory.go`)

**Functional:**
- [ ] Initializes with empty state for all fields
- [ ] Stores and retrieves articles by ID
- [ ] Stores and retrieves summaries by article ID
- [ ] Stores and retrieves triage scores by article ID
- [ ] Stores and retrieves embeddings by article ID
- [ ] Stores and retrieves clusters as an ordered list
- [ ] Stores and retrieves narratives by cluster ID
- [ ] Maintains ordered list of ReflectionReports
- [ ] Maintains ordered list of ToolCallRecords
- [ ] Computes quality trajectory from reflection history
- [ ] Snapshot method returns deep copy (mutations do not affect original)

**Non-Functional:**
- [ ] All operations are O(1) or O(n) where n is the collection size
- [ ] Memory usage scales linearly with corpus size

### Tool: `fetch_articles`

**Functional:**
- [ ] Parses URLs from markdown file using existing URLParser
- [ ] Fetches each URL using existing ContentFetcher
- [ ] Uses cache when use_cache=true and cache hit exists
- [ ] Stores fetched articles in Working Memory
- [ ] Returns correct counts for total, successful, failed, cache_hits
- [ ] Returns failed URLs in the response
- [ ] Handles mixed content types (HTML, PDF, YouTube)
- [ ] Continues fetching remaining URLs when individual fetches fail

**Edge Cases:**
- [ ] Empty file returns NO_URLS_FOUND error
- [ ] File with only invalid URLs returns ALL_FETCHES_FAILED error
- [ ] File not found returns FILE_NOT_FOUND error
- [ ] Duplicate URLs are deduplicated

### Tool: `summarize_batch`

**Functional:**
- [ ] Summarizes all articles without summaries when article_ids is omitted
- [ ] Summarizes only specified articles when article_ids is provided
- [ ] Skips articles that already have summaries in Working Memory
- [ ] Uses cache for previously summarized articles
- [ ] Stores generated summaries in Working Memory
- [ ] Returns correct counts for generated, cache_hits, failed

**Edge Cases:**
- [ ] Empty article list (all already summarized) returns 0 generated
- [ ] Article with empty CleanedText is skipped with warning

### Tool: `triage_articles`

**Functional:**
- [ ] Scores all articles when article_ids is omitted
- [ ] Produces relevance_score, quality_score, signal_strength for each article (all 0-1)
- [ ] Assigns recommended_action: "include", "deprioritize", or "exclude"
- [ ] Provides reasoning for each score
- [ ] Stores triage scores in Working Memory
- [ ] Returns correct category counts

**Edge Cases:**
- [ ] Single article is scored without error
- [ ] Article with very short content still receives scores

### Tool: `generate_embeddings`

**Functional:**
- [ ] Generates 768-dimensional embeddings for article summaries
- [ ] Skips articles that already have embeddings
- [ ] Stores embeddings in Working Memory
- [ ] Returns correct count of newly generated embeddings

**Edge Cases:**
- [ ] Article without summary is skipped
- [ ] All articles already have embeddings: returns 0 generated

### Tool: `cluster_articles`

**Functional:**
- [ ] Clusters articles using K-means with existing TopicClusterer
- [ ] Auto-detects cluster count when num_clusters=0
- [ ] Respects min_cluster_size parameter
- [ ] Generates human-readable cluster labels
- [ ] Stores clusters in Working Memory
- [ ] Returns cluster details with article IDs and titles

**Edge Cases:**
- [ ] Fewer articles than requested clusters: reduces cluster count automatically
- [ ] All articles identical: produces single cluster
- [ ] Articles without embeddings are excluded from clustering

### Tool: `evaluate_clusters`

**Functional:**
- [ ] Computes coherence score per cluster using embedding distances
- [ ] Computes separation score between cluster centroids
- [ ] Identifies size appropriateness (too_small, appropriate, too_large)
- [ ] Suggests actions (keep, merge, split, dissolve) based on scores
- [ ] Stores evaluations in Working Memory

**Edge Cases:**
- [ ] Single cluster: separation score is N/A, coherence is still computed
- [ ] Cluster with single article: coherence is 1.0

### Tool: `generate_cluster_narrative`

**Functional:**
- [ ] Generates narrative for specified cluster using ALL articles in that cluster
- [ ] Produces title (5-8 words), one_liner, key_developments, key_stats
- [ ] Includes correct article citation references [N]
- [ ] Respects emphasis_article_ids for prioritization
- [ ] Stores narrative in Working Memory associated with cluster

**Edge Cases:**
- [ ] Cluster with single article: narrative is essentially a summary of that article
- [ ] Invalid cluster_id returns error

### Tool: `generate_executive_summary`

**Functional:**
- [ ] Synthesizes all cluster narratives into cohesive executive summary
- [ ] Produces title, TLDR, top developments, by_the_numbers, why_it_matters
- [ ] Includes must_read highlight when include_must_read=true
- [ ] All citations [N] reference valid article numbers
- [ ] Stores executive summary and full digest draft in Working Memory

**Edge Cases:**
- [ ] No clusters (small corpus path): generates from article summaries directly
- [ ] Single cluster: executive summary is based on that one cluster narrative

### Tool: `reflect`

**Functional:**
- [ ] Evaluates current digest draft on all 5 dimensions
- [ ] Scores each dimension 0-1
- [ ] Computes overall_score as weighted average
- [ ] Identifies specific weaknesses with section, dimension, severity, and suggested fix
- [ ] Computes improvement_delta from previous iteration
- [ ] Sets should_continue based on delta and current scores
- [ ] Appends ReflectionReport to Working Memory

**Scoring Accuracy:**
- [ ] Specificity: detects vague statements like "significant progress" without specifics
- [ ] Grounding: detects claims without [N] citations
- [ ] Coherence: detects non-sequiturs and abrupt topic changes
- [ ] Reader Value: detects purely descriptive content lacking insight
- [ ] Coverage: detects articles not referenced in the digest

**Edge Cases:**
- [ ] First iteration: improvement_delta is 0
- [ ] Perfect scores: should_continue is false

### Tool: `revise_section`

**Functional:**
- [ ] Revises only the specified section
- [ ] Preserves all other sections unchanged
- [ ] Returns both original and revised content
- [ ] Logs revision in Working Memory
- [ ] Addresses the specific weakness described

**Edge Cases:**
- [ ] Invalid section identifier returns error
- [ ] Section that is already high quality: revision makes minimal changes

### Tool: `render_digest`

**Functional:**
- [ ] Renders current digest draft as markdown using existing MarkdownRenderer
- [ ] Writes file to specified output_path
- [ ] Returns file path, word count, article count, cluster count
- [ ] Includes all sections: title, TLDR, must-read, top developments, clusters, references

**Edge Cases:**
- [ ] Output directory does not exist: creates it
- [ ] Empty digest: returns error

---

## Integration Verification

### End-to-End: Happy Path (Medium Corpus, 10-15 articles)

1. [ ] User runs `briefly digest from-file input/weekly.md`
2. [ ] Agent fetches all articles (some cache hits)
3. [ ] Agent summarizes uncached articles
4. [ ] Agent triages articles for relevance
5. [ ] Agent generates embeddings
6. [ ] Agent clusters articles (3-5 clusters)
7. [ ] Agent evaluates clusters (acceptable quality)
8. [ ] Agent generates narrative per cluster
9. [ ] Agent generates executive summary
10. [ ] Agent reflects: overall score above threshold on first try
11. [ ] Agent renders markdown
12. [ ] Output file exists with correct structure
13. [ ] All articles appear in references section
14. [ ] All citations [N] are valid
15. [ ] Total time under 5 minutes

### End-to-End: Small Corpus (2-4 articles)

1. [ ] Agent fetches all articles
2. [ ] Agent summarizes articles
3. [ ] Agent triages articles
4. [ ] Agent skips embedding/clustering (corpus too small)
5. [ ] Agent generates executive summary directly from summaries
6. [ ] Agent reflects and potentially revises
7. [ ] Agent renders markdown without cluster groupings
8. [ ] Output is coherent and well-structured despite no clustering

### End-to-End: Reflect/Revise Loop

1. [ ] Agent generates initial digest
2. [ ] Reflection identifies low grounding score (missing citations)
3. [ ] Agent calls revise_section on executive_summary
4. [ ] Agent re-reflects: grounding score improved
5. [ ] If still below threshold, agent revises again
6. [ ] Quality trajectory shows improvement across iterations
7. [ ] Agent stops when threshold met or max iterations reached
8. [ ] Final digest quality is strictly better than or equal to initial draft

### End-to-End: Diminishing Returns Early Stop

1. [ ] Agent generates initial digest (iteration 0)
2. [ ] Reflection scores 0.72 (below 0.75 threshold)
3. [ ] Agent revises, re-reflects: score 0.74 (delta: +0.02)
4. [ ] Delta < 0.05: agent stops early (should_continue=false)
5. [ ] Agent renders with best-effort output
6. [ ] AgentMetadata.early_stop_reason = "diminishing_returns"

### End-to-End: Fallback to Linear Pipeline

1. [ ] Agent initialization fails (Gemini tool-use not supported for model)
2. [ ] System logs warning about fallback
3. [ ] Linear pipeline executes as before
4. [ ] Output is produced with status="fallback"
5. [ ] AgentMetadata.fallback_reason documents the cause

---

## Quality and Reflection Verification

### Reflection Accuracy

- [ ] Given a digest with no citations: grounding score is below 0.3
- [ ] Given a digest citing all articles: coverage score is above 0.9
- [ ] Given a digest with vague statements: specificity score is below 0.5
- [ ] Given a digest with good insights: reader_value score is above 0.7
- [ ] Given a digest with logical flow: coherence score is above 0.7

### Revision Effectiveness

- [ ] Revising for grounding increases citation count
- [ ] Revising for specificity replaces vague phrases with concrete data
- [ ] Revising for coverage adds references to previously uncited articles
- [ ] Revision does not introduce new factual errors
- [ ] Revision preserves content that was already high quality

### Iteration Behavior

- [ ] Quality scores are monotonically non-decreasing across iterations (with tolerance of 0.02 for measurement noise)
- [ ] Total LLM calls for reflection are at most max_iterations * 2 (reflect + revise per iteration)
- [ ] Diminishing returns detection correctly identifies plateaus

---

## Data Verification

### Working Memory Consistency

- [ ] After fetch: all returned article IDs exist in memory
- [ ] After summarize: every summarized article has a corresponding summary in memory
- [ ] After cluster: every article in a cluster exists in the articles map
- [ ] After narrative: every cluster with a narrative has the narrative stored
- [ ] After executive summary: digest draft contains all cluster narratives
- [ ] Reflection history length equals the number of reflect calls

### Citation Integrity

- [ ] Every [N] in executive summary maps to a valid article in the corpus
- [ ] Citation numbers are 1-indexed and contiguous
- [ ] No duplicate citation numbers
- [ ] Every article appears in the references section

### Tool Call Log

- [ ] Sequence numbers are contiguous starting from 1
- [ ] Every tool call has non-zero duration
- [ ] No overlapping timestamps (tools execute sequentially)
- [ ] Total tool call count matches sum of tool_call_breakdown

---

## Performance Verification

### Timing Budgets

| Corpus Size | Total Time Budget | Reflect/Revise Budget |
|---|---|---|
| 2-4 articles | Under 2 minutes | 1 iteration max |
| 5-15 articles | Under 5 minutes | 2 iterations max |
| 15-30 articles | Under 8 minutes | 3 iterations max |
| 30+ articles | Under 12 minutes | 3 iterations max |

### LLM Call Efficiency

- [ ] No redundant LLM calls (articles summarized at most once)
- [ ] Embeddings generated at most once per article
- [ ] Reflection calls are at most max_iterations
- [ ] Revision calls are at most max_iterations (one revise per reflect)
- [ ] Total Gemini API calls for orchestration are under 50 per session

### Cache Effectiveness

- [ ] Second run on same input file achieves over 80% cache hit rate for fetch and summarize
- [ ] Cached articles produce identical results to fresh fetches
- [ ] Cache key includes content hash to invalidate stale entries

---

## Fallback and Resilience Verification

### Graceful Degradation

- [ ] Single article fetch failure does not abort the session
- [ ] Single summary failure does not abort the session
- [ ] Embedding failure for one article excludes it from clustering but includes it in output
- [ ] Clustering failure falls back to single-cluster mode (all articles in one group)
- [ ] Narrative generation failure for one cluster proceeds with remaining clusters
- [ ] Reflection failure proceeds to render without quality loop
- [ ] Revision failure proceeds to render with current quality

### Fallback Triggers

- [ ] `--no-agent` flag bypasses agent entirely and uses linear pipeline
- [ ] Gemini API connection failure triggers fallback to linear pipeline
- [ ] Agent timeout (10 min) triggers fallback with partial results
- [ ] Tool-use parsing failure (invalid function call format) retries once, then falls back

### Error Reporting

- [ ] All errors include context (which tool, which article, which iteration)
- [ ] Partial results are returned when possible (some articles processed even if others failed)
- [ ] AgentMetadata.warnings captures all non-fatal issues
- [ ] Processing stats are accurate even on partial failure

---

## Security Verification

- [ ] API keys are never included in tool call parameters or results
- [ ] API keys are never logged
- [ ] Working memory is not serialized to disk (in-memory only)
- [ ] Tool parameters are validated before execution (no path traversal in file operations)
- [ ] LLM responses are sanitized before being used as file paths
- [ ] System prompt does not leak internal implementation details to end users

---

## Observability Verification

- [ ] Every tool call is logged with: name, duration, status, brief result summary
- [ ] Quality trajectory is logged at session end
- [ ] Agent strategy decisions are logged (e.g., "skipping clustering: only 3 articles")
- [ ] Reflection reports are included in final metadata for post-hoc analysis
- [ ] LangFuse tracing wraps orchestration LLM calls (if configured)
- [ ] PostHog tracks: digest generation events, agent strategy, iteration counts (if configured)
