# GenAI-Focused RSS Feed Recommendations

## Official Company Blogs

### OpenAI / ChatGPT
- **OpenAI Blog**: `https://openai.com/blog/rss.xml`
  - Product announcements, research updates, GPT releases
  - High signal, official source

### Anthropic / Claude
- **⚠️ No Official RSS Feed Available**
  - Anthropic does not currently provide an RSS feed for their news/blog
  - **Workarounds:**
    - Follow [@AnthropicAI on Twitter](https://twitter.com/AnthropicAI) and use Twitter RSS converters
    - Use change monitoring tools (Visualping, Distill.io) on `https://www.anthropic.com/news`
    - Manually submit URLs using `briefly url add` when announcements drop
    - Check HackerNews - Claude announcements often hit front page

### Google / Gemini
- **Google AI Blog**: `https://blog.google/technology/ai/rss/`
  - Gemini updates, Google AI research, DeepMind news
- **Google Developers Blog**: `https://developers.googleblog.com/feeds/posts/default`
  - API updates, developer tools, SDKs

### Meta AI
- **Meta AI Blog**: `https://ai.meta.com/blog/rss/`
  - Llama releases, Meta AI research

### Microsoft / Azure AI
- **Azure AI Blog**: `https://devblogs.microsoft.com/azure-ai/feed/`
  - Azure OpenAI Service, Copilot updates

## Tech News Sites (AI Sections)

### General Tech with AI Focus
- **TechCrunch AI**: `https://techcrunch.com/category/artificial-intelligence/feed/`
  - Startup news, funding, product launches
- **VentureBeat AI**: `https://venturebeat.com/category/ai/feed/`
  - Enterprise AI, business applications
- **The Verge AI**: `https://www.theverge.com/ai-artificial-intelligence/rss/index.xml`
  - Consumer AI products, reviews

### Developer-Focused
- **HackerNews AI/ML**: `https://hn.algolia.com/api/v1/search?query=AI+OR+LLM+OR+GPT&tags=story&hitsPerPage=20`
  - Note: This is API, not RSS. For RSS, use:
  - **HackerNews Front Page**: `https://news.ycombinator.com/rss`
  - Filter: Use keywords in digest pipeline
- **Dev.to AI Tag**: `https://dev.to/feed/tag/ai`
  - Developer tutorials, use cases, experiments

## Research & Deep Technical

### arXiv (Research Papers)
- **arXiv AI**: `http://export.arxiv.org/rss/cs.AI`
  - Artificial Intelligence papers
- **arXiv CL (Computation and Language)**: `http://export.arxiv.org/rss/cs.CL`
  - NLP and LLM research
- **arXiv LG (Machine Learning)**: `http://export.arxiv.org/rss/cs.LG`
  - Machine learning papers

### Papers Aggregators
- **Hugging Face Blog**: `https://huggingface.co/blog/feed.xml`
  - Model releases, tutorials, research highlights
- **Papers With Code**: Check their blog/updates section

## Community & Aggregators

### Medium
- **Towards Data Science**: `https://towardsdatascience.com/feed`
  - AI/ML tutorials, case studies
- **Medium AI Tag**: `https://medium.com/feed/tag/artificial-intelligence`
  - Mixed quality, broad coverage

### Reddit (via RSS)
- **r/MachineLearning**: `https://www.reddit.com/r/MachineLearning/.rss`
  - Research discussions, paper releases
- **r/LocalLLaMA**: `https://www.reddit.com/r/LocalLLaMA/.rss`
  - Open source LLMs, local deployment
- **r/OpenAI**: `https://www.reddit.com/r/OpenAI/.rss`
  - ChatGPT news, community discussions

## Newsletters (with RSS)

### Specialized AI Newsletters
- **The Batch (DeepLearning.AI)**: Check if RSS available
- **Import AI by Jack Clark**: `https://jack-clark.net/feed/`
  - Weekly AI news curation
- **TLDR AI**: `https://tldr.tech/ai/feed`
  - Daily AI news digest

## Startup/Product Hunt

- **Product Hunt AI Tools**: `https://www.producthunt.com/topics/artificial-intelligence.rss`
  - New AI products and tools
- **Y Combinator Companies (AI-tagged)**: Check YC RSS with AI filter

## Recommended Starting Set (10-15 feeds)

For initial testing, I recommend starting with these **high-signal feeds**:

```bash
# Add these feeds to your database:

# 1. Official Sources (Highest Signal)
briefly feed add https://openai.com/blog/rss.xml
# Note: Anthropic has no RSS - use manual submission or HackerNews
briefly feed add https://blog.google/technology/ai/rss/
briefly feed add https://ai.meta.com/blog/rss/

# 2. Tech News (Curated)
briefly feed add https://techcrunch.com/category/artificial-intelligence/feed/
briefly feed add https://venturebeat.com/category/ai/feed/

# 3. Developer Community
briefly feed add https://news.ycombinator.com/rss
briefly feed add https://dev.to/feed/tag/ai
briefly feed add https://huggingface.co/blog/feed.xml

# 4. Research (Select One)
briefly feed add http://export.arxiv.org/rss/cs.CL  # NLP/LLM focus

# 5. Community Discussions
briefly feed add https://www.reddit.com/r/MachineLearning/.rss
briefly feed add https://www.reddit.com/r/LocalLLaMA/.rss
```

## Feed Quality Tiers

### Tier 1: Essential (Add First)
- OpenAI Blog
- Google AI Blog
- TechCrunch AI
- HackerNews (catches Claude announcements + community signal)
- Hugging Face Blog

### Tier 2: High Value (Add Next)
- VentureBeat AI
- Meta AI Blog
- arXiv cs.CL (NLP/LLM research papers)
- Dev.to AI
- Azure AI Blog

### Tier 3: Supplementary (Add Later)
- Reddit communities
- Medium aggregators
- Product Hunt
- General tech news

## Notes

1. **HackerNews**: No native AI-filtered RSS, but high GenAI signal in front page
2. **arXiv**: High volume, may want to filter by specific authors or keywords
3. **Reddit**: Can be noisy, but good for early signals and community trends
4. **Medium**: Variable quality, but good for use cases and tutorials

## Testing Strategy

1. Start with **5 Tier 1 feeds**
2. Run digest generation
3. Evaluate article quality and relevance
4. Add Tier 2 feeds incrementally
5. Tune theme classification thresholds based on results

## Feed Management Commands

```bash
# List all feeds
briefly feed list

# Add a feed
briefly feed add <url>

# Test a specific feed
briefly aggregate --feed-id <id>

# Generate digest from feeds
briefly digest --from-feeds
```
