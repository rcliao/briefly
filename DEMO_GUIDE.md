# Briefly Demo Guide

**Quick pitch:** AI-powered news aggregator that turns information overload into concise, LinkedIn-ready digests.

---

## The Problem

**Information overload is killing productivity.**

- Following 10+ tech blogs/newsletters → hundreds of articles per week
- Reading everything = impossible
- Skimming headlines = missing important context
- Traditional RSS readers = still too much content
- Manual curation = too time-consuming

**The real pain:** You want to stay informed without drowning in content.

---

## The Solution

**Briefly** is an AI-powered news aggregator with **hierarchical summarization**:

1. **Subscribe to feeds** (RSS/Atom) or submit individual URLs
2. **Aggregate daily** - fetches articles automatically
3. **AI classifies** - categorizes by theme (AI/ML, DevOps, Security, etc.)
4. **Generate weekly digest** - creates LinkedIn-ready summary

### What Makes It Different?

**Hierarchical Summarization (v3.1 Innovation):**

Traditional approach:
- Summarize top 3-5 articles → loses 80% of content
- Executive summary ignores most articles

Briefly's approach:
1. **Stage 1:** Group articles by topic (K-means clustering)
2. **Stage 2:** Generate comprehensive narrative for *each cluster* from ALL articles
3. **Stage 3:** Synthesize cluster narratives into executive summary

**Result:** Concise digest that's grounded in 100% of your content, not just top articles.

### Demo Workflow

```bash
# 1. Add RSS feeds
briefly feed add https://hnrss.org/newest
briefly feed add https://blog.golang.org/feed.atom

# 2. Aggregate daily (cron job)
briefly aggregate --since 24

# 3. Generate weekly digest
briefly digest generate --since 7

# Output: digests/digest_2025-11-06.md
# - Executive summary (synthesized from all articles)
# - Theme-grouped articles with summaries
# - Citations and relevance scores
# - LinkedIn-ready formatting
```

### Web Interface (Phase 1)

```bash
# Start server
briefly serve

# Visit:
# - http://localhost:8080/digests - Browse all digests
# - http://localhost:8080/digests/{id} - Read digest with markdown rendering
# - http://localhost:8080/submit - Submit individual articles
# - http://localhost:8080/themes - Manage classification themes
```

---

## Technical Highlights

### Architecture

**Database-Driven Pipeline (PostgreSQL)**

```
RSS Feeds + Manual URLs
    ↓
Aggregate → Articles (fetched, cleaned, cached)
    ↓
Classify → Theme Assignment (LLM-based, 0.4 threshold)
    ↓
Store → PostgreSQL (articles, themes, relationships)
    ↓
Generate Digest:
  1. Fetch articles by date/theme
  2. Summarize each article (Gemini API, cached)
  3. Generate embeddings (768-dim vectors)
  4. Cluster by topic (K-means)
  5. Generate cluster narratives (ALL articles)
  6. Synthesize executive summary (from narratives)
  7. Render markdown + save to DB
```

### Tech Stack

- **Language:** Go 1.23+
- **Database:** PostgreSQL (with migrations)
- **LLM:** Google Gemini 2.5 Flash
  - Article summarization
  - Theme classification
  - Embedding generation (768-dim)
  - Executive summary synthesis
- **Web:** Standard library HTTP server + TailwindCSS
- **Caching:** SQLite (article content, summaries)
- **Observability:** LangFuse (LLM tracing) + PostHog (analytics)

### Key Innovations

**1. Hierarchical Summarization**
- No information loss (every article contributes)
- Better credibility (summary reflects all content)
- Maintains conciseness (clusters → synthesis)

**2. Theme-Based Classification**
- 10 default themes (extensible)
- LLM-powered relevance scoring
- 0.4 threshold for quality control

**3. Smart Caching**
- Article content: 24-hour TTL
- Summaries: 7-day TTL, content-hash linked
- Reduces API costs by 60%+ on regeneration

**4. Observability**
- LangFuse: All LLM calls traced (prompts, tokens, latency, costs)
- PostHog: User analytics (digest views, article clicks, theme filters)

### Code Quality

- **Clean architecture:** Interface-driven design (`internal/pipeline/interfaces.go`)
- **Repository pattern:** Database abstraction (`internal/persistence/`)
- **Migration system:** 7 migrations for schema evolution
- **Comprehensive testing:** Unit tests for parsers, summarizers, templates
- **Graceful degradation:** Article failures don't stop pipeline

### Performance

- **Processing:** ~2-3 minutes for 15-20 articles
- **Concurrency:** Sequential (parallelization TODO)
- **Cache hit rate:** 0-60% depending on freshness
- **API costs:** ~$0.10-0.20 per weekly digest (Gemini Flash)

---

## Demo Flow (5 Minutes)

### 1. Show the Problem (1 min)
"I follow 15 tech blogs. Here's my RSS reader—200 unread articles this week."

### 2. Add Feeds (30 sec)
```bash
briefly feed add https://hnrss.org/newest
briefly feed list
```

### 3. Aggregate Articles (30 sec)
```bash
# Show command + output
briefly aggregate --since 24
```

### 4. Generate Digest (1 min)
```bash
# Generate with live output
briefly digest generate --since 7

# Show generated markdown
cat digests/digest_2025-11-06.md
```

**Point out:**
- Executive summary (concise but comprehensive)
- Theme grouping (AI/ML, DevOps, Security)
- Article summaries with citations
- All articles included (not just top 3)

### 5. Web Viewer (2 min)
```bash
briefly serve
```

Visit `http://localhost:8080/digests`:
- Show digest list (cards with metadata)
- Click digest → show detail page
- Point out:
  - Markdown rendering
  - Theme filtering
  - Article summaries
  - Mobile-responsive design

---

## Questions to Ask Friends

1. **Problem validation:** "Do you struggle with information overload from newsletters/RSS?"
2. **Solution fit:** "Would you use this for your tech reading?"
3. **Features:** "What's missing? What would you want?"
4. **UI/UX:** "Is the web viewer intuitive?"
5. **Pricing:** "Would you pay for this? How much?"

### Potential Pain Points to Surface

- Setup complexity (Go install, PostgreSQL, API keys)
- Feed discovery (how to find good RSS feeds?)
- Customization (theme relevance, summary length)
- Sharing (can I share digests with team?)
- Integration (export to Notion, email, Slack?)

---

## Next Steps (Roadmap)

**Phase 2: API Enhancements** (Next)
- Full REST API (CRUD for articles, digests, feeds)
- Pagination, filtering, search
- API authentication

**Phase 3: Production Deployment**
- Docker containerization
- Deploy to Railway/Fly.io
- Scheduled cron jobs (daily aggregation)
- Multi-user support

**Future Ideas** (Based on Feedback)
- Browser extension (right-click → add to Briefly)
- Email digests (weekly delivery)
- Team collaboration (shared feeds, comments)
- Mobile app (read digests on-the-go)
- Integrations (Notion, Slack, Discord)

---

## Quick Install (For Demo)

```bash
# Prerequisites: Go 1.23+, PostgreSQL

# 1. Clone and build
git clone https://github.com/rcliao/briefly.git
cd briefly
go build -o briefly ./cmd/briefly

# 2. Setup database
createdb briefly
export DATABASE_URL="postgresql://user:pass@localhost:5432/briefly"

# 3. Get Gemini API key
export GEMINI_API_KEY="your-key-here"

# 4. Run migrations (automatic on first command)
./briefly feed list

# 5. Start demo!
```

---

## Feedback Collection

**After demo, ask:**

1. **Value Prop:** "Does this solve a real problem for you?"
2. **Usability:** "Is it easy to understand/use?"
3. **Features:** "What's the #1 feature you'd want next?"
4. **Pricing:** "Would you pay? How much?"
5. **Referral:** "Who else would benefit from this?"

**Track responses in a simple doc:**
- Name
- Current pain (yes/no)
- Would use (yes/maybe/no)
- Requested features
- Willingness to pay

---

## Contact & Links

- **GitHub:** https://github.com/rcliao/briefly
- **Docs:** See `README.md` and `CLAUDE.md` for detailed documentation
- **Issues:** Found a bug? https://github.com/rcliao/briefly/issues

**Built with:** Go, PostgreSQL, Gemini AI, TailwindCSS

**License:** (Add your license here)
