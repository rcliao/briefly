# Kagi News - Inspiration & Analysis

**Version:** 2.1
**Date:** 2025-10-31
**Category:** News Aggregation & Digest

This document captures comprehensive research and analysis of Kagi News, which serves as the primary inspiration for Briefly's weekly digest approach, particularly around time-bounded consumption, structured summaries, and signal-over-noise philosophy.

---

## Table of Contents

1. [Kagi News Analysis](#kagi-news-analysis)
2. [Key Features We're Adopting](#key-features-were-adopting)
3. [What Briefly Does Better](#what-briefly-does-better)
4. [Technical Stack Lessons](#technical-stack-lessons)

---

## Kagi News Analysis

### What is Kagi News?

**Kagi News** is a privacy-focused, AI-powered news digest that publishes **once daily at noon UTC**, synthesizing thousands of community-curated RSS feeds into a ~5-minute comprehensive briefing.

**Key Philosophy:** "Signal over noise" - time-bounded consumption vs. endless scrolling addiction.

**Website:** https://kagi.com/news

### Core Philosophy

**Time-Bounded Consumption:**
- Publishes once daily at fixed time (noon UTC)
- Creates natural consumption endpoint
- ~5-minute reading time estimate
- No infinite scroll or addiction patterns
- Prevents news anxiety and doomscrolling

**Signal Over Noise:**
- Synthesizes thousands of sources into key developments
- AI-powered summarization focuses on substance
- Filters out duplicate coverage and low-quality content
- Surfaces underreported stories from quality sources

**Privacy-First:**
- No tracking, no ads, no surveillance
- RSS requests proxied to prevent publisher tracking
- Free to use without account creation
- No user profiling or data collection

---

## Key Features We're Adopting

### 1. Structured Article Sections

**Kagi's Approach:**

Each story is presented with organized sections:
- **Summary**: AI-generated overview
- **Highlights**: Key points from the article
- **Key Quotes**: Direct quotations from sources
- **Timeline**: Chronological context of events
- **Context**: Background information
- **Impact**: Potential implications

**For Briefly:**

We'll implement similar structure with added developer focus:
- **Summary** - AI-generated overview (2-3 sentences)
- **Key Moments** - Important content highlights (includes notable quotes from sources)
- **Perspectives** - Multiple viewpoints on topics (when applicable)
- **Why It Matters** - Significance and implications for tech/GenAI field
- **Context** - Background information for understanding

**Implementation:**
- Structured output APIs (Gemini `response_schema`)
- No JSON parsing errors or truncation
- Type-safe responses with automatic validation
- Stored in `summaries` table with dedicated fields

### 2. Transparent Source Citations

**Kagi's Approach:**
- Hover-activated citation links
- Clear source metadata (publisher, date, author)
- Multiple perspectives synthesized with source attribution
- Inline citations like [1], [2] in cluster summaries

**For Briefly:**
- Implement `citations` table with rich metadata:
  - Source URL, title, publisher, author
  - Published date, accessed date
  - Citation type (primary, supporting, quote)
  - Quote text if applicable
- Cluster summaries with inline citations
- Transparent source tracking for all content

### 3. Time-Bounded Publishing

**Kagi's Approach:**
- Once-daily at fixed time (noon UTC)
- Creates natural consumption endpoint
- ~5-minute reading time estimate
- No real-time updates or constant refresh

**For Briefly:**
- Weekly digest with fixed schedule (e.g., "Every Monday 9am PST")
- Configurable via `DIGEST_CRON` environment variable:
  - Daily: `0 9 * * *` (every day at 9am)
  - Weekly: `0 9 * * 1` (every Monday at 9am)
  - Custom: Any valid cron expression
- Infrastructure-level OR in-app scheduler options
- Manual CLI commands for testing

### 4. Customizable Digest Structure

**Kagi's Approach:**

Users can:
- Enable/disable categories
- Reorder categories by preference
- Adjust number of stories per category
- Filter by keywords

**For Briefly:**
- Theme-based customization (GenAI, Gaming, Technology, All)
- Cluster ordering by relevance/recency
- Keyword filtering within digests
- Enable/disable specific clusters
- Admin-configured theme definitions

### 5. Community Source Curation

**Kagi's Approach:**
- [Open-source RSS feed list on GitHub](https://github.com/kagisearch/kite-public)
- Community can submit PRs to add sources
- Quality standards enforced:
  - 25+ feeds per category
  - Daily update frequency
  - Trustworthy sources
- Transparent source management

**For Briefly:**
- Consider GitHub repo for source suggestions (future phase)
- Admin-controlled source list (Phase 1)
- Manual URL submission via CLI/web (Phase 1)
- Community contribution path (Phase 3)

### 6. Privacy-First Architecture

**Kagi's Approach:**
- No tracking, no ads, no surveillance
- RSS requests proxied to prevent publisher tracking
- Free to use without account
- No user profiling

**For Briefly:**
- Minimal tracking (PostHog for product analytics only)
- No ads, no monetization
- Optional analytics opt-out
- Transparent data usage policy
- Focus on content quality over engagement metrics

---

## What Briefly Does Better

### 1. Developer/Tech Focus

**Kagi:** General news across all topics (politics, sports, tech, science, etc.)

**Briefly:** Specifically tech/GenAI content with:
- Code examples and technical depth
- Architecture insights and system design patterns
- API/SDK release tracking
- Research paper summaries
- Developer tool comparisons

### 2. Deeper Technical Analysis

**Beyond surface-level summaries:**
- Implementation details and architecture explanations
- Code snippets and API examples
- Performance benchmarks and comparisons
- Migration guides and breaking changes
- Integration patterns and best practices

### 3. Historical Tracking

**Week-over-week analysis:**
- Trend identification across digests
- Topic evolution tracking
- Technology adoption patterns
- Industry shift detection
- Comparative analysis with past weeks

### 4. Email Distribution

**Push vs Pull:**

**Kagi:** Pull-only (visit website to read)

**Briefly:**
- Push digest to subscribers via email (future phase)
- Weekly delivery to inbox
- HTML email with full digest content
- Click-through tracking for engagement

### 5. API Access

**Programmatic consumption:**
- REST API for digest data
- JSON export of articles/summaries
- Webhook support for new digests
- Integration with other tools (Notion, Slack, etc.)

### 6. Customizable Themes

**User-configured topic filtering:**
- GenAI: LLMs, ML frameworks, AI tools
- Gaming: Game engines, graphics, esports tech
- Technology: Web dev, mobile, infrastructure
- Custom themes via admin dashboard

### 7. Manual URL Submission

**Direct content addition:**
- CLI command: `briefly add-url https://example.com`
- Web form for bulk URL submission
- Manual URL tracking and processing
- Override auto-discovery with specific articles

---

## Technical Stack Lessons

### Kagi's Stack

**Frontend:**
- Svelte + TypeScript + Vite
- Static deployment for performance
- Minimal JavaScript for fast load times
- Progressive enhancement approach

**Backend:**
- Crystal language (unique choice for performance)
- PostgreSQL with raw SQL
- Flow-Based Programming (FBP) architecture
- Microservices for processing pipeline

**AI/LLM:**
- AI synthesis and summarization (provider not disclosed)
- Kagi Translate for 24 language support
- Custom prompt engineering for quality
- Caching layer to reduce costs

**Infrastructure:**
- Static site generation for main content
- Edge caching for global distribution
- RSS proxy layer for privacy
- Scalable batch processing

### Briefly's Stack Decisions

**Backend:**
- **Go** (not Crystal) - Demonstrate Go expertise for job market
- Clean architecture with interfaces
- Repository pattern for data access
- Dependency injection via builder pattern

**Database:**
- **PostgreSQL** with **pgvector** extension
- Vector similarity search for clustering
- JSON fields for flexible data (timeline, perspectives)
- Proper indexing for performance

**Frontend:**
- **HTMX + TailwindCSS** (simpler than Svelte, still modern)
- Server-side rendering with templates
- Minimal JavaScript for progressive enhancement
- Fast page loads with Alpine.js for interactivity

**AI/LLM:**
- **Gemini API** for summarization (cost-effective)
- **Structured output APIs** (response_schema) for reliability
- **LangFuse** integration for observability
- **LLM-as-judge** for evaluation

**Infrastructure:**
- Docker containerization
- Railway/Fly.io deployment
- Background job processing with CRON
- Static asset serving via CDN (future)

### Why Not Replicate Kagi Exactly?

1. **Language Choice:** Go over Crystal
   - Go has better job market demand
   - Larger ecosystem and community
   - Demonstrates industry-standard skills
   - Better tooling for production systems

2. **Scope Focus:** Tech/GenAI over general news
   - Narrower domain = higher quality
   - Targeted audience engagement
   - Expertise demonstration in specific field
   - Personal use case alignment

3. **Feature Additions:** Beyond Kagi's model
   - Multi-agent architecture (Kagi likely has this, but we'll document it)
   - LangFuse observability (show LLMOps skills)
   - Evaluation framework (demonstrate prompt engineering)
   - Theme system (show product thinking)
   - RAG implementation (prove ML engineering skills)

4. **Complexity Management:** Phased approach
   - Start with MVP (RSS + manual URLs)
   - Add search in Phase 2
   - Community features in Phase 3
   - Gradual scaling vs big-bang launch

---

## Other Products Considered

### Google News

**Pros:**
- Real-time updates
- Massive scale (thousands of sources)
- Personalization via ML

**Cons:**
- Infinite scroll addiction
- Clickbait amplification
- Privacy concerns
- No synthesis/summarization

**Lesson for Briefly:** Avoid real-time updates, focus on weekly synthesis

### Hacker News

**Pros:**
- Community-curated content
- Tech/startup focus
- Comment discussions

**Cons:**
- Manual submission only
- No summarization
- Noise in comments
- Bias toward YC ecosystem

**Lesson for Briefly:** Combine automated discovery with manual curation

### Techmeme

**Pros:**
- Automated clustering
- Breaking tech news
- Source diversity

**Cons:**
- No summarization
- Real-time focus (anxiety-inducing)
- Link aggregation only

**Lesson for Briefly:** Use clustering + synthesized summaries

### Morning Brew / TLDR Newsletter

**Pros:**
- Email delivery
- Curated summaries
- Consistent format

**Cons:**
- Fully manual curation (doesn't scale)
- No website archive
- Limited technical depth

**Lesson for Briefly:** Automate curation while maintaining quality

---

## Key Takeaways

### Adopt from Kagi:
1. ✅ Time-bounded publishing (weekly schedule)
2. ✅ Structured article sections (Summary, Key Moments, Perspectives, etc.)
3. ✅ Transparent citations with metadata
4. ✅ Privacy-first approach (minimal tracking)
5. ✅ Signal over noise philosophy

### Improve on Kagi:
1. ✅ Tech/GenAI specialization (narrower focus)
2. ✅ Multi-agent architecture (demonstrate advanced skills)
3. ✅ LangFuse observability (show LLMOps)
4. ✅ Evaluation framework (prove prompt engineering)
5. ✅ Theme customization (product thinking)

### Avoid from Others:
1. ❌ Real-time updates (Google News) - creates anxiety
2. ❌ Infinite scroll (most platforms) - addiction pattern
3. ❌ Clickbait amplification (social media) - quality degradation
4. ❌ Manual curation only (newsletters) - doesn't scale

---

## Version History

**v2.1 (2025-10-31):**
- Extracted from main design document
- Added structured output API details
- Clarified stack decisions with reasoning

**v2.0 (2025-10-31):**
- Initial Kagi News research incorporated
- Comparative analysis added
