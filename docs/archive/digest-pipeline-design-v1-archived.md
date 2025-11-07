# Briefly: GenAI News Aggregator Pipeline Design (v1 - ARCHIVED)

> **⚠️ DEPRECATED:** This design document has been superseded by [digest-pipeline-v2.md](../digest-pipeline-v2.md).
>
> **Date Archived:** 2025-11-06
>
> **Reason for Replacement:**
> - This design was Python-focused, but the actual implementation is in Go
> - Proposed weekly "living digests" concept, but actual requirement is "few digests per day" (like Kagi News)
> - Envisioned single digest per run, but correct architecture is many digests per run (one per cluster)
> - Suggested HDBSCAN + pgvector, but implementation uses K-means (simpler, works well)
> - Did not match the actual Go codebase structure and Phase 0/1 implementation
>
> **This document is preserved for historical reference only.**
>
> See the new design at: [docs/digest-pipeline-v2.md](../digest-pipeline-v2.md)

---

## Overview

**Goal:** Automatically aggregate, cluster, and summarize GenAI news articles into digestible daily updates and weekly newsletters.

**Target Audience:** Busy developers, PMs, designers who need to stay current on GenAI developments.

**Key Features:**
- Daily digest page showing today's updated topics (organic traffic)
- Weekly newsletter with top stories from the week
- Transparent source citations (reduce hallucination)
- Semantic clustering to discover topics automatically

---

## Core Architecture Principles

### 1. Time-Bound Evolving Digests
- **Weekly cycles:** Monday-Sunday boundaries
- **Living digests:** Digests grow as articles arrive throughout the week
- **Fresh starts:** Each Monday begins new clustering cycle (no carryover)

### 2. Deduplication Strategy
- Each article belongs to exactly ONE digest per week
- Daily page shows only digests updated TODAY (new articles added)
- Weekly newsletter pulls stable digests from completed week

### 3. LLM Techniques to Showcase
- **Relevance Classification:** LLM filters GenAI-related articles
- **Embeddings:** Semantic similarity for clustering
- **Vector Database:** pgvector for similarity search
- **RAG-style Summarization:** Digest generation with inline citations
- **Clustering:** HDBSCAN for automatic topic discovery

---

## Data Model

```sql
-- Core Entities
CREATE TABLE article (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  url TEXT UNIQUE NOT NULL,
  title TEXT NOT NULL,
  content TEXT,
  published_at TIMESTAMP NOT NULL,
  processed_at TIMESTAMP DEFAULT NOW(),
  embedding VECTOR(1536),  -- pgvector extension
  week_start_date DATE NOT NULL,  -- which week this belongs to (Monday)
  is_relevant BOOLEAN DEFAULT TRUE,  -- passed GenAI filter
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE digest (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL,  -- < 20 characters, punchy headline
  tldr TEXT NOT NULL,   -- 1 sentence summary
  summary TEXT NOT NULL,  -- 2-3 paragraphs with citations
  key_moments JSONB,  -- array of {quote: string, article_id: uuid}
  week_start_date DATE NOT NULL,  -- which week this digest belongs to
  first_seen_at TIMESTAMP NOT NULL,  -- when digest was created
  last_updated_at TIMESTAMP NOT NULL,  -- when new articles were last added
  is_stable BOOLEAN DEFAULT FALSE,  -- no changes in 48+ hours
  article_count INT DEFAULT 0,
  cluster_id INT,  -- internal tracking (changes each run)
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE digest_article (
  digest_id UUID REFERENCES digest(id) ON DELETE CASCADE,
  article_id UUID REFERENCES article(id) ON DELETE CASCADE,
  added_at TIMESTAMP DEFAULT NOW(),
  relevance_score FLOAT,  -- how central is this article to digest (0-1)
  PRIMARY KEY (digest_id, article_id)
);

-- Indexes for performance
CREATE INDEX idx_article_week ON article(week_start_date, processed_at);
CREATE INDEX idx_article_embedding ON article USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX idx_digest_week ON digest(week_start_date, last_updated_at DESC);
CREATE INDEX idx_digest_stable ON digest(is_stable, week_start_date);
CREATE INDEX idx_digest_article_added ON digest_article(added_at DESC);
```

---

## RSS Feed Sources

### Company Blogs (High Signal, Low Volume)
```python
RSS_FEEDS = {
    'openai': 'https://openai.com/blog/rss.xml',
    'anthropic': 'https://www.anthropic.com/blog/rss',
    'google_ai': 'https://ai.googleblog.com/feeds/posts/default',
    'meta_ai': 'https://ai.meta.com/blog/rss/',
    'mistral': 'https://mistral.ai/news/rss.xml',
}
```

### Tech News (Moderate Signal, High Volume - Needs Filtering)
```python
TECH_NEWS_FEEDS = {
    'techcrunch_ai': 'https://techcrunch.com/category/artificial-intelligence/feed/',
    'verge_ai': 'https://www.theverge.com/ai-artificial-intelligence/rss/index.xml',
    'venturebeat_ai': 'https://venturebeat.com/category/ai/feed/',
}
```

### Developer/Community (Needs Aggressive Filtering)
```python
COMMUNITY_FEEDS = {
    'simon_willison': 'https://simonwillison.net/atom/everything/',
    'lil_log': 'https://lilianweng.github.io/feed.xml',
}

# Hacker News via API (not RSS)
HACKERNEWS_API = 'https://hn.algolia.com/api/v1/search?tags=story&query=LLM OR GPT OR Claude'
```

**Start with:** OpenAI, Anthropic, TechCrunch AI, Hacker News API, Simon Willison (5 feeds total)

---

## Pipeline Workflow

### Nightly Job (Runs at 2am UTC)

```
┌─────────────────────────────────────────────────────────────┐
│                    NIGHTLY PIPELINE                         │
└─────────────────────────────────────────────────────────────┘
        │
        ├─→ 1. INGEST
        │   └─→ Fetch articles from RSS feeds (last 24 hours)
        │
        ├─→ 2. RELEVANCE FILTER
        │   └─→ LLM classifies if article is GenAI-related
        │
        ├─→ 3. CONTENT EXTRACTION
        │   └─→ Fetch full article content (not just RSS snippet)
        │
        ├─→ 4. EMBED
        │   └─→ Generate embeddings for article content
        │
        ├─→ 5. CLUSTER (ALL ARTICLES FROM CURRENT WEEK)
        │   └─→ HDBSCAN on embeddings → cluster labels
        │
        ├─→ 6. UPDATE/CREATE DIGESTS
        │   └─→ Match clusters to existing digests OR create new
        │
        ├─→ 7. GENERATE SUMMARIES
        │   └─→ LLM summarizes each digest with citations
        │
        └─→ 8. MARK STABLE DIGESTS
            └─→ Flag digests unchanged for 48+ hours
```

### Detailed Step Breakdown

#### Step 1: Ingest Articles
```python
def ingest_articles(since_hours=24):
    """
    Fetch articles from RSS feeds published in last N hours.
    Only fetches metadata: title, URL, published_at
    """
    articles = []
    for feed_name, feed_url in RSS_FEEDS.items():
        feed = feedparser.parse(feed_url)
        for entry in feed.entries:
            published = parse_date(entry.published)
            if published >= datetime.now() - timedelta(hours=since_hours):
                articles.append({
                    'url': entry.link,
                    'title': entry.title,
                    'published_at': published,
                    'source': feed_name
                })
    return articles
```

#### Step 2: Relevance Filter
```python
def is_genai_relevant(article):
    """
    Use Gemini Flash to classify if article is GenAI-related.
    Cheap pre-filter: check keywords first, then LLM confirm.
    """
    # Quick keyword pre-filter (free)
    genai_keywords = ['gpt', 'claude', 'llm', 'chatbot', 'anthropic', 
                      'openai', 'gemini', 'copilot', 'mistral', 'ai model']
    title_lower = article.title.lower()
    
    if not any(kw in title_lower for kw in genai_keywords):
        # No obvious keywords, likely not GenAI
        return False
    
    # LLM confirmation (costs ~$0.0001 per article)
    prompt = f"""Is this article about Generative AI (LLMs, image generation, AI products)?
    
Title: {article.title}

Answer with only: YES or NO"""
    
    response = gemini_flash.generate(prompt)
    return 'YES' in response.upper()
```

#### Step 3: Content Extraction
```python
def extract_content(article_url):
    """
    Fetch full article content (not just RSS snippet).
    Use newspaper3k or BeautifulSoup.
    """
    from newspaper import Article
    
    article = Article(article_url)
    article.download()
    article.parse()
    
    return {
        'content': article.text,
        'top_image': article.top_image,
        'authors': article.authors
    }
```

#### Step 4: Generate Embeddings
```python
def generate_embedding(article):
    """
    Generate embedding from article content.
    Use first 2-3 paragraphs (~500 words) to avoid context limits.
    """
    # Truncate content to first 500 words
    words = article.content.split()[:500]
    text_to_embed = ' '.join(words)
    
    # Use Gemini Embedding API (free tier)
    embedding = gemini.embed(text_to_embed)
    
    return embedding  # returns 1536-dim vector
```

#### Step 5: Cluster Articles
```python
def cluster_weekly_articles(week_start_date):
    """
    Cluster ALL articles from current week (not just today).
    Uses HDBSCAN for density-based clustering.
    """
    # Fetch all articles from current week
    articles = db.query(Article).filter(
        Article.week_start_date == week_start_date,
        Article.is_relevant == True
    ).all()
    
    if len(articles) < 3:
        # Not enough articles to cluster
        return {}
    
    # Extract embeddings
    embeddings = np.array([a.embedding for a in articles])
    
    # Run HDBSCAN
    import hdbscan
    clusterer = hdbscan.HDBSCAN(
        min_cluster_size=3,  # minimum 3 articles per cluster
        metric='cosine',
        cluster_selection_method='eom'
    )
    cluster_labels = clusterer.fit_predict(embeddings)
    
    # Group articles by cluster
    clusters = {}
    for article, label in zip(articles, cluster_labels):
        if label == -1:  # outlier
            continue
        if label not in clusters:
            clusters[label] = []
        clusters[label].append(article)
    
    return clusters  # {0: [article1, article2], 1: [article3, ...]}
```

#### Step 6: Match Clusters to Digests
```python
def find_digest_for_cluster(cluster_articles, week_start_date):
    """
    Find existing digest that best matches this cluster.
    Uses article overlap (Jaccard similarity).
    """
    existing_digests = db.query(Digest).filter(
        Digest.week_start_date == week_start_date
    ).all()
    
    cluster_article_ids = {a.id for a in cluster_articles}
    
    best_match = None
    best_similarity = 0
    
    for digest in existing_digests:
        digest_article_ids = {da.article_id for da in digest.digest_articles}
        
        # Jaccard similarity
        intersection = len(cluster_article_ids & digest_article_ids)
        union = len(cluster_article_ids | digest_article_ids)
        similarity = intersection / union if union > 0 else 0
        
        if similarity > 0.5 and similarity > best_similarity:
            best_match = digest
            best_similarity = similarity
    
    return best_match
```

#### Step 7: Generate Digest Summary
```python
def generate_digest_summary(articles):
    """
    Use LLM to generate digest with inline citations.
    RAG-style: articles are context, LLM synthesizes.
    """
    # Format articles with citation numbers
    articles_context = ""
    for i, article in enumerate(articles, 1):
        articles_context += f"\n[{i}] {article.title}\n"
        articles_context += f"URL: {article.url}\n"
        articles_context += f"Content: {article.content[:500]}...\n"
    
    prompt = f"""Summarize these {len(articles)} articles into a digest for developers.

Articles:
{articles_context}

Create:
1. Title (< 20 characters, punchy headline like "OpenAI Launches GPT-5")
2. TLDR (1 sentence capturing the main development)
3. Summary (2-3 paragraphs explaining the story with inline citations like [1][2])
4. Key Moments (3-5 important quotes from articles that support the summary)

Format Key Moments as JSON array:
[
  {{"quote": "actual quote from article", "article_number": 1}},
  ...
]

Use citation format: [1], [2], etc. corresponding to article numbers above.
Be specific and technical (our audience are developers).
"""
    
    response = gemini_pro.generate(prompt)
    
    # Parse response into structured data
    # (assumes LLM returns structured format)
    return {
        'title': parse_title(response),
        'tldr': parse_tldr(response),
        'summary': parse_summary(response),
        'key_moments': parse_key_moments(response)
    }
```

#### Step 8: Mark Stable Digests
```python
def mark_stable_digests():
    """
    Mark digests as stable if no updates in 48+ hours.
    Stable digests are candidates for weekly newsletter.
    """
    threshold = datetime.now() - timedelta(hours=48)
    
    db.query(Digest).filter(
        Digest.last_updated_at < threshold,
        Digest.week_start_date == get_current_week_start()
    ).update({'is_stable': True})
```

---

## Daily vs Weekly Logic

### Daily Page Query
```sql
-- Show digests updated TODAY
SELECT 
  d.id,
  d.title,
  d.tldr,
  d.summary,
  d.article_count,
  COUNT(da.article_id) FILTER (WHERE da.added_at >= NOW() - INTERVAL '24 hours') as new_articles_today
FROM digest d
JOIN digest_article da ON d.id = da.digest_id
WHERE d.week_start_date = date_trunc('week', NOW())  -- current week
GROUP BY d.id, d.title, d.tldr, d.summary, d.article_count
HAVING COUNT(da.article_id) FILTER (WHERE da.added_at >= NOW() - INTERVAL '24 hours') > 0
ORDER BY d.last_updated_at DESC;
```

### Weekly Newsletter Query
```sql
-- Get top digests from completed week (run on Friday evening)
SELECT 
  d.id,
  d.title,
  d.tldr,
  d.summary,
  d.article_count
FROM digest d
WHERE d.week_start_date = date_trunc('week', NOW())
  AND d.is_stable = TRUE
  AND d.article_count >= 3
ORDER BY d.article_count DESC, d.first_seen_at ASC
LIMIT 7;
```

---

## Edge Case Handling

### Slow News Day (< 3 articles)
```python
if len(articles_today) < 3:
    # Don't create standalone digest
    # Add to "Other Notable Updates" list (no digest, just links)
    # Wait for tomorrow to cluster with more articles
    pass
```

### Breaking News (> 12 articles in one cluster)
```python
if len(cluster_articles) > 12:
    # Run sub-clustering to split into 2-3 related digests
    # Example: "OpenAI Announces GPT-5"
    #   → "Launch Details"
    #   → "Technical Analysis"
    #   → "Market Reactions"
    sub_clusters = run_sub_clustering(cluster_articles, n_clusters=3)
```

### Article Published Friday, Processed Monday
```python
# Article belongs to PREVIOUS week (based on published_at)
article.week_start_date = get_monday_of_week(article.published_at)

# Joins previous week's digests (won't create new digest in current week)
```

### No New Articles on Sunday
```
Daily Page UI:
  "No new articles today. Check back tomorrow!"
  
  [Show last 3 days of activity]
  Nov 9: 2 digests updated
  Nov 8: 4 digests updated
  Nov 7: 3 digests updated
```

---

## Implementation Milestones

### Week 1: Manual Pipeline Validation
**Goal:** Prove clustering + summarization works before automating

- [ ] Fetch 50 articles from 3 RSS feeds (OpenAI, TechCrunch AI, Hacker News)
- [ ] Manually label which are GenAI-relevant (build ground truth dataset)
- [ ] Test LLM relevance filter (Gemini Flash) → measure precision/recall
- [ ] Embed relevant articles → cluster with HDBSCAN
- [ ] Manually inspect clusters (should get 3-5 coherent topics)
- [ ] Generate 1-2 digests by copy/pasting articles into LLM prompt

**Success Criteria:** 3-5 coherent clusters that make sense semantically

### Week 2: Automate Core Pipeline
**Goal:** Wake up to fresh articles every morning

- [ ] Build ingestion script (RSS → PostgreSQL)
- [ ] Add relevance filter (LLM classification)
- [ ] Generate embeddings → store in pgvector
- [ ] Run nightly clustering job (cron)
- [ ] Generate 1-2 digests automatically (LLM API call)

**Success Criteria:** 5-10 articles ingested nightly, 1-2 digests auto-generated

### Week 3: Basic UI + Daily Page
**Goal:** Ship to 5 people for feedback

- [ ] Build homepage: list of digests with TLDR
- [ ] Build digest detail page: summary + linked articles with citations
- [ ] Deploy to Vercel/Netlify
- [ ] Test on mobile + desktop

**Success Criteria:** 5 people can read digests and understand the content

### Week 4: Weekly Newsletter
**Goal:** Send first weekly newsletter

- [ ] Add "mark as stable" logic for digests
- [ ] Build email template (top 5 digests from week)
- [ ] Integrate with email service (SendGrid/Mailchimp)
- [ ] Test send to your newsletter list (150+ subscribers)

**Success Criteria:** >30% open rate, >10% click-through to digest pages

---

## Technology Stack

### Backend
- **Language:** Python 3.11+
- **Web Framework:** FastAPI (lightweight, async)
- **Database:** PostgreSQL 15+ with pgvector extension
- **ORM:** SQLAlchemy 2.0
- **Task Queue:** APScheduler (for nightly jobs)

### AI/ML
- **LLM API:** Google Gemini (free tier for relevance + summarization)
- **Embeddings:** Gemini Embedding API (free tier, 1536 dimensions)
- **Clustering:** HDBSCAN (scikit-learn-extra)
- **Vector Search:** pgvector (PostgreSQL extension)

### Frontend
- **Framework:** Next.js 14 (App Router)
- **Styling:** Tailwind CSS
- **Deployment:** Vercel

### Infrastructure
- **Database Hosting:** Supabase (free tier with pgvector)
- **Backend Hosting:** Railway/Render (free tier)
- **Email:** SendGrid (free tier: 100 emails/day)

---

## Key Configuration

```python
# config.py

# Database
DATABASE_URL = "postgresql://user:pass@host:5432/briefly"

# LLM APIs
GEMINI_API_KEY = "your-api-key"
GEMINI_MODEL_FLASH = "gemini-1.5-flash"  # for relevance filter
GEMINI_MODEL_PRO = "gemini-1.5-pro"  # for summarization
GEMINI_EMBEDDING_MODEL = "text-embedding-004"

# Clustering
MIN_CLUSTER_SIZE = 3  # minimum articles per digest
MAX_CLUSTER_SIZE = 12  # split if larger
CLUSTER_SIMILARITY_THRESHOLD = 0.5  # for digest matching

# Pipeline
INGEST_HOURS_LOOKBACK = 24  # fetch articles from last N hours
STABILITY_HOURS = 48  # mark stable if no updates for N hours
WEEK_START_DAY = 0  # 0 = Monday

# Newsletter
NEWSLETTER_SEND_DAY = 4  # 4 = Friday
NEWSLETTER_SEND_HOUR = 18  # 6pm UTC
NEWSLETTER_MAX_DIGESTS = 7
```

---

## SQL Helper Functions

```sql
-- Get Monday of current week
CREATE FUNCTION get_current_week_start() RETURNS DATE AS $$
  SELECT date_trunc('week', NOW())::DATE;
$$ LANGUAGE SQL;

-- Get Monday of specific date's week
CREATE FUNCTION get_week_start(input_date DATE) RETURNS DATE AS $$
  SELECT date_trunc('week', input_date)::DATE;
$$ LANGUAGE SQL;
```

---

## Monitoring & Observability

### Key Metrics to Track
- **Articles ingested per day** (expect 10-30)
- **Relevance filter precision** (% of filtered articles actually relevant)
- **Clustering quality** (silhouette score, manual inspection)
- **Digest generation cost** (LLM API calls * cost per call)
- **Daily page views** (organic traffic)
- **Weekly newsletter open rate** (target: >30%)

### Logging Strategy
```python
# Log every pipeline step
logger.info(f"Ingested {len(articles)} articles")
logger.info(f"Filtered to {len(relevant)} relevant articles")
logger.info(f"Generated {len(clusters)} clusters")
logger.info(f"Created {new_digests} new digests, updated {updated_digests}")
```

---

## Testing Strategy

### Unit Tests
- RSS feed parsing
- Relevance classification (with ground truth dataset)
- Embedding generation
- Cluster-to-digest matching logic

### Integration Tests
- Full pipeline run (end-to-end)
- Database operations (insert, update, query)
- LLM API calls (mock responses)

### Manual QA
- Read generated digests for coherence
- Check citation links work
- Verify no duplicate articles across digests
- Confirm weekly newsletter only shows stable digests

---

## Future Enhancements (Post-MVP)

1. **Multi-theme support:** Add "AI Coding Tools", "LLM Research", etc.
2. **User personalization:** Let users filter by preferred topics
3. **Related articles:** Use vector search to show "similar articles"
4. **Sentiment analysis:** Tag digests as "positive/negative/neutral"
5. **Trending detection:** Highlight breaking news with velocity tracking
6. **Social sharing:** Auto-post digests to Twitter/LinkedIn
7. **API for developers:** Let others query digests programmatically

---

## Questions to Answer During Implementation

1. **How to handle duplicate URLs?** (Some RSS feeds cross-post)
   - Solution: UNIQUE constraint on article.url, INSERT ON CONFLICT DO NOTHING

2. **What if clustering produces weird results?** (e.g., all articles in one cluster)
   - Solution: Add max_cluster_size check, run sub-clustering
   - Fallback: Manual curation for first few weeks

3. **How to handle API rate limits?**
   - Solution: Exponential backoff, queue articles for processing
   - Gemini free tier: 60 requests/minute

4. **Should digests ever be deleted?**
   - Solution: Soft delete (archive old digests after 3 months)
   - Keep for SEO and historical reference

5. **How to measure digest quality?**
   - Solution: Track user engagement (time on page, click-through)
   - A/B test different summarization prompts

---

## Next Steps

1. **Set up local environment:**
   ```bash
   # Install PostgreSQL + pgvector
   brew install postgresql@15
   createdb briefly
   psql briefly -c "CREATE EXTENSION vector;"
   
   # Install Python dependencies
   pip install fastapi sqlalchemy psycopg2-binary pgvector hdbscan feedparser newspaper3k google-generativeai
   ```

2. **Create database schema:**
   - Run SQL from "Data Model" section
   - Add sample data for testing

3. **Build ingestion script:**
   - Fetch from 1-2 RSS feeds
   - Test relevance filter on 20 articles
   - Measure accuracy

4. **Test clustering:**
   - Embed 30-50 articles
   - Run HDBSCAN
   - Manually inspect clusters

**Let's start with Week 1 milestones. Ready to code?**
