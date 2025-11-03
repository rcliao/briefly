# Briefly - User Guide (Phase 0 Features)

This guide covers the new features added in **Phase 0** of the Briefly v2.0 roadmap:
- Theme-based article classification
- Manual URL submission
- Observability tracking (LangFuse + PostHog)

For general Briefly usage, see the main README.md.

---

## Table of Contents

1. [Theme System](#theme-system)
2. [Manual URL Submission](#manual-url-submission)
3. [Observability Setup](#observability-setup)
4. [Web Interface](#web-interface)
5. [Common Workflows](#common-workflows)
6. [Troubleshooting](#troubleshooting)

---

## Theme System

The theme system allows you to categorize articles automatically using LLM-based classification. Articles are scored against each theme (0.0-1.0 relevance) and assigned to themes above a 0.4 threshold.

### Default Themes

Briefly ships with 10 pre-configured themes:

1. **AI & Machine Learning** - AI, neural networks, machine learning
2. **Cloud Infrastructure & DevOps** - Cloud computing, containers, CI/CD
3. **Software Engineering & Best Practices** - Code quality, testing, architecture
4. **Web Development & Frontend** - HTML, CSS, JavaScript frameworks
5. **Data Engineering & Analytics** - Data pipelines, analytics, databases
6. **Security & Privacy** - Cybersecurity, privacy, encryption
7. **Programming Languages & Tools** - Language features, tooling
8. **Mobile Development** - iOS, Android, mobile frameworks
9. **Open Source & Community** - OSS projects, community updates
10. **Product & Startup** - Product management, startup news

### CLI Commands

#### List Themes

```bash
# Show all enabled themes
briefly theme list

# Show all themes (including disabled)
briefly theme list --all
```

**Output:**
```
ID: theme-1
Name: AI & Machine Learning
Description: Articles about artificial intelligence and machine learning
Keywords: AI, ML, machine learning, neural networks, deep learning
Enabled: Yes
---
```

#### Add a New Theme

```bash
briefly theme add "Blockchain & Web3" \
  --description "Articles about blockchain, cryptocurrency, and decentralized web" \
  --keywords "blockchain,crypto,web3,defi,nft"
```

**Tips:**
- Use quotes around theme names with spaces
- Keywords are comma-separated (no spaces after commas)
- Keywords help the LLM classifier identify relevant articles

#### Update a Theme

```bash
# Update description
briefly theme update theme-11 \
  --description "Updated description for blockchain"

# Update keywords
briefly theme update theme-11 \
  --keywords "blockchain,ethereum,solidity,smart contracts"

# Update both
briefly theme update theme-11 \
  --description "New description" \
  --keywords "new,keywords"
```

#### Enable/Disable Themes

```bash
# Disable a theme (won't be used for classification)
briefly theme disable theme-11

# Re-enable it later
briefly theme enable theme-11
```

**Use Case:** Temporarily disable themes you're not interested in to reduce classification noise.

#### Remove a Theme

```bash
briefly theme remove theme-11
```

**Warning:** This permanently deletes the theme. Articles already classified under this theme will lose that association.

### How Theme Classification Works

1. **Article Fetched**: When processing articles, each article's content is analyzed
2. **LLM Classification**: Gemini analyzes the article against all enabled themes
3. **Scoring**: Each theme gets a relevance score (0.0 to 1.0)
4. **Threshold Filtering**: Only themes with scores ≥ 0.4 (40%) are assigned
5. **Best Match**: The highest-scoring theme becomes the article's primary category

**Example Classification:**
```
Article: "New GPT-4 Model Achieves 95% Accuracy"

Theme Scores:
- AI & Machine Learning: 0.92 ✓ (assigned)
- Software Engineering: 0.48 ✓ (assigned)
- Cloud Infrastructure: 0.15 (below threshold)
- Product & Startup: 0.33 (below threshold)

Result: Article categorized as "AI & Machine Learning" (highest score)
```

---

## Manual URL Submission

Submit URLs manually for processing instead of waiting for RSS feeds to discover them.

### CLI Commands

#### Submit URLs

```bash
# Single URL
briefly url add https://example.com/article

# Multiple URLs
briefly url add \
  https://example.com/article1 \
  https://example.com/article2 \
  https://example.com/article3
```

**What happens:**
1. URLs are stored with status `pending`
2. Run `briefly aggregate` to process them
3. URLs are converted to feed items and processed like RSS articles
4. Status updates to `processed` or `failed`

#### List Submitted URLs

```bash
# All URLs
briefly url list

# Filter by status
briefly url list --status pending
briefly url list --status processed
briefly url list --status failed
briefly url list --status processing
```

**Output:**
```
ID: url-123abc
URL: https://example.com/article
Status: pending
Submitted By: user@example.com
Created: 2025-10-31 14:30:00
---
```

#### Check URL Status

```bash
briefly url status url-123abc
```

**Statuses:**
- `pending` - Waiting to be processed
- `processing` - Currently being fetched and summarized
- `processed` - Successfully processed ✓
- `failed` - Processing failed (see error message)

#### Retry Failed URLs

```bash
# Retry specific URL
briefly url retry url-123abc

# Retry all failed URLs
briefly url retry --all
```

**Use Case:** URL fetch failed due to temporary network issue or rate limiting.

#### Clear Processed/Failed URLs

```bash
# Remove processed URLs (cleanup)
briefly url clear --processed

# Remove failed URLs
briefly url clear --failed

# Remove both
briefly url clear --processed --failed
```

### Processing Manual URLs

Manual URLs are **not** automatically processed. You must run:

```bash
briefly aggregate
```

This command:
1. Fetches all `pending` manual URLs
2. Converts them to feed items
3. Updates status to `processing`
4. Processes each URL (fetch, summarize, classify)
5. Marks as `processed` or `failed`

**Recommended Workflow:**

```bash
# Morning: Submit interesting URLs you found
briefly url add https://news.ycombinator.com/item?id=12345
briefly url add https://blog.example.com/new-feature

# Later: Process all pending URLs
briefly aggregate

# Check results
briefly url list --status processed
```

---

## Observability Setup

Briefly tracks LLM operations and user interactions using **LangFuse** (LLM tracing) and **PostHog** (analytics).

### LangFuse (LLM Tracing)

Tracks all Gemini API calls: prompts, completions, tokens, latency, and estimated costs.

**Setup:**

1. Sign up at [langfuse.com](https://langfuse.com) or self-host
2. Get your API keys
3. Set environment variables:

```bash
export LANGFUSE_PUBLIC_KEY="pk-lf-..."
export LANGFUSE_SECRET_KEY="sk-lf-..."
export LANGFUSE_HOST="https://cloud.langfuse.com"  # Optional
```

4. Restart Briefly

**Current Status:** Phase 0 uses **local logging mode** (logs to stdout). HTTP API integration coming in Phase 1.

**What's Tracked:**
- Text generation (summarization, theme classification)
- Embedding generation (for clustering)
- Token usage and costs
- Latency per operation

### PostHog (Analytics)

Tracks user behavior and system events for product insights.

**Setup:**

1. Sign up at [posthog.com](https://posthog.com) or self-host
2. Get your project API key
3. Set environment variables:

```bash
export POSTHOG_API_KEY="phc_..."
export POSTHOG_HOST="https://app.posthog.com"  # Optional
```

4. Restart Briefly

**What's Tracked:**

**Backend Events:**
- `digest_generated` - Digest creation completed
- `article_processed` - Article fetched and summarized
- `theme_classification` - Article assigned to theme
- `manual_url_submitted` - User submitted URL
- `llm_call` - LLM API call made (model, tokens, latency)

**Frontend Events (Web UI):**
- `themes_page_viewed` - User opened theme management page
- `submit_page_viewed` - User opened URL submission page
- `urls_submitted` - User submitted URLs via web form

**Dashboard Ideas:**
- Daily article processing volume
- Theme classification distribution
- LLM cost tracking
- User engagement (web UI usage)

---

## Web Interface

Briefly now includes a web interface for managing themes and submitting URLs.

### Starting the Web Server

```bash
# Start server on default port (8080)
briefly serve

# Custom port
briefly serve --port 3000
```

Access at: http://localhost:8080

### Available Pages

#### Theme Management (`/themes`)
- View all themes
- Add new themes (modal form)
- Edit existing themes (click theme card)
- Enable/disable themes (toggle switch)
- Delete themes (confirmation required)

**Features:**
- Real-time updates (no page refresh)
- Responsive design (TailwindCSS)
- PostHog tracking for analytics

#### Manual URL Submission (`/submit`)
- Submit single URL (form)
- Bulk submit (textarea, one URL per line)
- View submission history with statuses
- Status icons: ⏳ pending, ⚙️ processing, ✅ processed, ❌ failed

**Features:**
- Instant feedback on submission
- Status auto-refresh (polling)
- PostHog tracking for submissions

### API Endpoints

All web pages use these REST APIs (can be used directly):

**Themes:**
- `GET /api/themes` - List themes (`?enabled=true` filter)
- `GET /api/themes/{id}` - Get theme details
- `POST /api/themes` - Create theme (JSON body)
- `PUT /api/themes/{id}` - Update theme (JSON body)
- `DELETE /api/themes/{id}` - Delete theme

**Manual URLs:**
- `GET /api/manual-urls` - List URLs (`?status=pending` filter)
- `GET /api/manual-urls/{id}` - Get URL details
- `POST /api/manual-urls/submit` - Submit URLs (JSON array)
- `DELETE /api/manual-urls/{id}` - Delete URL

**Example API Usage:**

```bash
# Create theme via API
curl -X POST http://localhost:8080/api/themes \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Rust Programming",
    "description": "Articles about Rust language",
    "keywords": ["rust", "cargo", "tokio"],
    "enabled": true
  }'

# Submit URL via API
curl -X POST http://localhost:8080/api/manual-urls/submit \
  -H "Content-Type: application/json" \
  -d '{
    "urls": [
      "https://blog.rust-lang.org/2024/11/01/release.html"
    ],
    "submitted_by": "user@example.com"
  }'
```

---

## Common Workflows

### Workflow 1: Curating Tech News for Specific Interests

**Goal:** Create a weekly digest focused only on AI/ML and Cloud/DevOps articles.

```bash
# 1. Disable themes you're not interested in
briefly theme disable theme-3  # Software Engineering
briefly theme disable theme-4  # Web Development
briefly theme disable theme-5  # Data Engineering
briefly theme disable theme-6  # Security
briefly theme disable theme-7  # Programming Languages
briefly theme disable theme-8  # Mobile
briefly theme disable theme-9  # Open Source
briefly theme disable theme-10 # Product & Startup

# 2. Keep only AI/ML and Cloud/DevOps enabled
briefly theme list  # Verify only 2 themes enabled

# 3. Submit URLs from your favorite sources
briefly url add https://news.ycombinator.com/item?id=12345
briefly url add https://aws.amazon.com/blogs/aws/new-feature/

# 4. Process URLs
briefly aggregate

# 5. Generate digest (only AI/ML and Cloud articles will appear)
briefly digest input/weekly-links.md
```

### Workflow 2: Building a Custom Theme for Your Niche

**Goal:** Track articles about Kubernetes specifically.

```bash
# 1. Create specialized theme
briefly theme add "Kubernetes" \
  --description "Articles about Kubernetes, container orchestration, and k8s ecosystem" \
  --keywords "kubernetes,k8s,kubectl,helm,kustomize,istio,service mesh"

# 2. Test classification
briefly url add https://kubernetes.io/blog/2024/11/01/feature/
briefly aggregate

# 3. Check if article was classified correctly
briefly url list --status processed

# 4. If misclassified, refine keywords
briefly theme update <theme-id> \
  --keywords "kubernetes,k8s,container orchestration,pod,deployment,service"

# 5. Retry classification
briefly url retry <url-id>
briefly aggregate
```

### Workflow 3: Weekly Digest with Manual Curation

**Goal:** Mix RSS feeds with manually curated articles.

```bash
# Monday: Start collecting URLs
briefly url add https://techcrunch.com/article-of-the-week

# Tuesday-Friday: Add more as you discover them
briefly url add https://example.com/interesting-post

# Friday afternoon: Process everything
briefly aggregate

# Friday evening: Generate digest
briefly digest input/weekly-links.md

# Saturday: Clean up processed URLs
briefly url clear --processed
```

---

## Troubleshooting

### Theme Classification Not Working

**Symptom:** Articles show as "Uncategorized"

**Possible Causes:**
1. **No themes enabled**
   ```bash
   briefly theme list
   # Enable at least one theme
   briefly theme enable theme-1
   ```

2. **Article content too short/irrelevant**
   - Theme requires ≥0.4 relevance score
   - Short articles may not have enough content to classify

3. **LLM API key not set**
   ```bash
   echo $GEMINI_API_KEY
   # If empty, set it in .env file
   ```

4. **Classification threshold too high**
   - Currently hardcoded at 0.4 (40%)
   - Consider creating more specific themes with better keywords

### Manual URLs Stuck in "Pending"

**Symptom:** URLs remain `pending` even after adding them

**Solution:** You must run aggregation manually:

```bash
briefly aggregate
```

**Why:** Manual URLs are **not** automatically processed. The `aggregate` command fetches and processes pending URLs.

### LangFuse Not Tracking

**Symptom:** No traces appearing in LangFuse dashboard

**Current Status:** Phase 0 uses **local logging only**. LangFuse HTTP API integration is pending.

**Check logs:**
```bash
# Look for "LangFuse" in application logs
briefly digest input/links.md 2>&1 | grep -i langfuse
```

**Workaround:** For now, LangFuse logs operations to stdout. Full dashboard integration coming in Phase 1.

### PostHog Events Not Appearing

**Symptom:** Dashboard empty, no events tracked

**Debug Steps:**

1. **Verify API key is set:**
   ```bash
   echo $POSTHOG_API_KEY
   # Should show phc_...
   ```

2. **Check PostHog is enabled:**
   ```bash
   # Add logging to verify PostHog client creation
   # Check application logs for "PostHog enabled" or similar
   ```

3. **Trigger events manually:**
   ```bash
   # Generate events
   briefly url add https://example.com
   briefly aggregate
   ```

4. **Check PostHog dashboard:**
   - Events may take 1-2 minutes to appear
   - Look in "Live Events" tab first

### Database Connection Errors

**Symptom:** `failed to connect to database`

**Check:**

1. **PostgreSQL running:**
   ```bash
   psql $DATABASE_URL -c "SELECT 1;"
   ```

2. **DATABASE_URL format:**
   ```bash
   # Correct format:
   export DATABASE_URL="postgresql://user:password@localhost:5432/briefly"
   ```

3. **Migrations applied:**
   ```bash
   # Run migrations (if using golang-migrate or similar)
   migrate -path internal/persistence/migrations -database $DATABASE_URL up
   ```

### Web UI Not Loading

**Symptom:** Browser shows "Connection refused" at http://localhost:8080

**Solutions:**

1. **Server not started:**
   ```bash
   briefly serve
   ```

2. **Port already in use:**
   ```bash
   # Try different port
   briefly serve --port 3000
   ```

3. **Firewall blocking:**
   ```bash
   # Check if port is accessible
   curl http://localhost:8080
   ```

---

## Tips and Best Practices

### Theme Design

**Good Theme:**
```
Name: Go Programming
Description: Articles about Go language, libraries, and best practices
Keywords: golang,go,goroutine,channel,concurrency,go modules
```

**Bad Theme:**
```
Name: Programming
Description: Programming stuff
Keywords: code,programming,development
```

**Why:** Specific themes with precise keywords yield better classification accuracy.

### URL Submission

**Do:**
- Submit high-quality, technical articles
- Use descriptive `submitted_by` (e.g., email or username)
- Check status after processing

**Don't:**
- Submit paywalled articles (may fail to fetch)
- Submit duplicate URLs (check `briefly url list` first)
- Submit URLs without content (redirects, sign-in pages)

### Performance

- **Batch URL submissions:** Submit multiple URLs at once instead of one-by-one
- **Clear processed URLs regularly:** Keeps database clean
- **Disable unused themes:** Reduces classification overhead

---

## Next Steps

**Phase 0 Complete! ✓**

**Phase 1 Features (Coming Soon):**
- Enhanced RSS aggregation with theme filtering
- Structured summaries (Key Moments, Perspectives, Why It Matters)
- Citation tracking for source attribution
- Theme-based digest filtering

**Stay Updated:**
- Check `docs/executions/2025-10-31.md` for implementation roadmap
- Follow release notes in README.md

---

## Support & Feedback

Found a bug? Have a feature request?

- **Issues:** https://github.com/rcliao/briefly/issues
- **Discussions:** https://github.com/rcliao/briefly/discussions
- **Email:** [Your contact email]

Happy curating! 📰
