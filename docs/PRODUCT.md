# Briefly - Product Vision

**Version:** 2.1
**Date:** 2025-10-31
**Status:** Planning

---

## Table of Contents

1. [What We're Building](#what-were-building)
2. [Vision & Goals](#vision--goals)
3. [Core Capabilities](#core-capabilities)
4. [User Flows](#user-flows)
5. [Success Metrics](#success-metrics)
6. [Why This Matters](#why-this-matters)

---

## What We're Building

Transform Briefly from a CLI-based digest generator into **an autonomous, weekly-updating news digest website** that demonstrates advanced GenAI engineering patterns suitable for Series A/B startup environments.

**Personal Use Case:** A tool to catch up on latest GenAI news efficiently, given how fast the field changes (new product releases every week).

**Inspired by:** Kagi News's "signal over noise" philosophy - time-bounded, high-quality synthesis vs. endless scrolling.

---

## Vision & Goals

### Product Vision

**"A self-updating weekly news digest website that automatically discovers, processes, and publishes curated GenAI/tech content, powered by a multi-agent pipeline with LangFuse observability and theme-based personalization."**

**Inspired by Kagi News:** Time-bounded consumption (weekly), high-quality synthesis, transparent citations, no addiction patterns.

### Technical Goals

1. **Autonomous Operation**: System runs weekly without manual intervention
2. **Scalable Processing**: Handle 100+ articles per week across multiple sources
3. **Theme-Based Filtering**: LLM classifies articles into configurable themes with relevance scoring
4. **Observable Performance**: LangFuse tracks every LLM call, cost, latency, prompt effectiveness
5. **User Analytics**: PostHog tracks public usage (X users/day, engagement metrics)
6. **Evaluation-Driven**: CLI-based prompt testing with golden datasets
7. **Production-Ready**: Clean architecture, error handling, monitoring, deployment

### Learning Goals (GenAI Job Preparation)

1. **Multi-Agent Systems**: Design and implement Manager-Worker orchestration
2. **LLMOps with LangFuse**: Production observability for LLM applications
3. **RAG Patterns**: Implement pgvector + context retrieval
4. **Evaluation Engineering**: Build CLI eval framework with LLM judges
5. **Product Analytics**: Integrate PostHog to demonstrate user growth metrics
6. **Production Patterns**: Scalable, maintainable GenAI application

---

## Core Capabilities

### 1. Multi-Source Content Aggregation

- **RSS Feeds** (Phase 1 - implemented separately)
- **Web Search** (Phase 2 - implemented separately)
- **Manual URL Submission** (CLI + Web admin)

### 2. Theme-Based Filtering

- Admin-configured themes (GenAI, Gaming, Technology, etc.)
- LLM-powered relevance filtering
- Customizable digest structure per theme

### 3. Multi-Agent Processing

- Manager-Worker architecture for parallel article processing
- LangFuse observability for all LLM calls

### 4. Kagi-Inspired Structured Summaries

- **Summary**: AI-generated overview
- **Key Moments**: Important content highlights (includes notable quotes)
- **Perspectives**: Multiple viewpoints on topics
- **Why It Matters**: Significance and implications
- **Context**: Background information

### 5. RAG-Enhanced Metadata

- Vector search (pgvector) for context-aware classification
- Historical article context for consistent summarization

### 6. LLM Evaluation Framework

- CLI-based evaluation commands
- LLM-as-judge for prompt optimization
- Golden dataset management

### 7. Configurable Automation

- Daily or Weekly schedule (configurable via DIGEST_CRON env var)
- Examples: Daily `0 9 * * *` or Weekly `0 9 * * 1`
- Manual CLI commands for testing pipeline steps
- Infrastructure-level OR in-app scheduler options

### 8. Web Interface with Analytics

- Public digest viewing with PostHog analytics
- Admin dashboard for source/theme management
- Citation tracking and transparent source attribution

---

## User Flows

### Admin Flow (You)

1. **Configure Sources:**
   - Add/edit RSS feeds via web admin
   - Configure search terms/queries
   - Manually submit URLs (CLI or web form)

2. **Configure Themes:**
   - Create themes (GenAI, Gaming, Technology)
   - Set theme keywords for LLM classification
   - Define relevance criteria per theme

3. **Monitor Performance:**
   - View LangFuse dashboard (LLM costs, latency, token usage)
   - Review PostHog analytics (users, page views, engagement)
   - Check weekly digest generation status

4. **Optimize Prompts:**
   - Run CLI eval commands against golden datasets
   - Compare prompt variants
   - Deploy improved prompts

### Public User Flow

1. **Visit Briefly website**
2. **Select theme** (GenAI, Gaming, All Tech)
3. **Read weekly digest** (published every Monday 9am)
   - Browse clusters (auto-generated topics)
   - Read structured summaries (Summary, Key Moments, Perspectives, Why It Matters)
   - Click through to original sources (transparent citations)
4. **Browse past digests** by date
5. **Search articles** by keyword, date, topic
6. **Customize view** (enable/disable clusters, filter keywords)

---

## Success Metrics

### Technical Metrics

**System Performance:**
- Weekly digest generated successfully by 9am Monday (100% uptime target)
- Processing time for 50 articles: <20 minutes
- API response time p95: <500ms
- Database query time p95: <100ms

**LLM Efficiency (LangFuse):**
- Average cost per digest: <$3
- Token efficiency: >80% of prompts under optimal length
- LLM call success rate: >99%
- Average summarization quality score: >4.0/5.0 (from evals)

**RAG Performance:**
- Vector similarity search latency: <100ms
- RAG context relevance score: >0.8
- Classification consistency improvement: +15% vs baseline

### Product Metrics (PostHog)

**User Engagement:**
- Daily Active Users (DAU): Track growth over time
- Weekly Active Users (WAU): Target X users/week
- Average session duration: Target >3 minutes
- Page views per session: Target >5
- Digest completion rate: % of users who read to bottom

**Content Metrics:**
- Articles aggregated per week: 50-100
- Unique sources covered: 20+
- Topic cluster count per digest: 3-5
- Most popular themes (GenAI, Gaming, etc.)
- Most clicked articles (top 10)

**Retention:**
- 7-day retention: Target >30%
- 30-day retention: Target >15%
- Returning users: % of sessions from returning vs new

### Learning Metrics (GenAI Job Preparation)

**Capabilities Demonstrated:**
- ✅ Multi-source content aggregation (RSS, search, manual)
- ✅ Theme-based filtering and classification
- ✅ LangFuse integration for production LLM observability
- ✅ PostHog analytics showing real user engagement
- ✅ RAG implementation with pgvector
- ✅ CLI-based evaluation framework
- ✅ Kagi-inspired structured summaries
- ✅ Clean Go architecture (interfaces, repositories, DI)
- ✅ Production deployment (Docker, Railway/Fly.io)

**Portfolio Talking Points:**
1. "Built weekly GenAI news digest with multi-source aggregation (RSS, search, manual), processing 50+ articles with LangFuse observability"
2. "Implemented theme-based content filtering with LLM classification, achieving X% relevance improvement"
3. "Integrated PostHog analytics tracking X users/day with Y% weekly retention across themed digests"
4. "Deployed RAG-enhanced article processing with pgvector, improving classification consistency by Z%"
5. "Created CLI-based evaluation framework with LLM-as-judge, optimizing prompts to 4.3/5.0 quality score"

---

## Why This Matters

### For GenAI Roles

**Demonstrates mastery of:**
- **Multi-Agent Systems**: Manager-Worker orchestration with isolated LLM contexts
- **LLMOps**: LangFuse integration for production-grade observability
- **RAG Patterns**: pgvector + context retrieval for improved outputs
- **Evaluation Engineering**: CLI-based eval framework with LLM judges
- **Product Analytics**: PostHog integration showing user engagement metrics
- **Clean Architecture**: Interface-driven design, repository pattern, dependency injection

### For Series A/B Startups

**Shows understanding of:**
- Production-grade LLM applications with observability
- Scalable multi-agent architectures
- Data-driven prompt optimization
- User analytics and growth metrics
- Clean, maintainable Go codebases
- DevOps and deployment patterns

---

## Version History

**v2.1 (2025-10-31):**
- Removed Key Quotes (merged into Key Moments)
- Added structured output APIs (Gemini response_schema)
- Added Go concurrency design (goroutines/channels)
- Added cluster-level summaries with citations
- Added daily/weekly CRON configuration
- Added single article CLI command (`briefly summarize`)
- Clarified summary type relationships (1:1 with type discrimination)

**v2.0 (2025-10-31):**
- Incorporated Kagi News analysis
- LangFuse/PostHog integration
- Theme system
- Manual URLs

**v1.0:**
- CLI-based digest generator
- Manual URL input
- Basic summarization
