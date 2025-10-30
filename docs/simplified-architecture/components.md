# Component Specifications - Content Digest System

## Core Components

### 1. URL Parser
**Responsibility**: Extract and validate URLs from markdown files
**Capabilities**:
- Parse markdown links in format `[text](url)`
- Parse raw URLs in bullet lists
- Normalize URLs (remove tracking parameters)
- Deduplicate URLs

**Limitations**:
- Does not fetch content
- Does not validate URL accessibility
- Does not parse non-markdown formats

### 2. Content Fetcher
**Responsibility**: Retrieve raw content from URLs
**Capabilities**:
- HTTP/HTTPS requests with proper headers
- Handle redirects (max 3)
- Detect content type from headers/URL
- Apply rate limiting between requests

**Limitations**:
- Does not parse content
- Does not handle authentication
- Does not execute JavaScript
- Maximum 10MB per document

### 3. Content Extractors

#### HTML Extractor
**Responsibility**: Extract readable text from HTML
**Capabilities**:
- Identify main article content
- Remove navigation, ads, footers
- Preserve paragraph structure
- Extract metadata (title, author, date)

**Limitations**:
- Does not render JavaScript
- Does not extract images
- Does not preserve formatting

#### PDF Extractor
**Responsibility**: Extract text from PDF documents
**Capabilities**:
- Extract all text pages
- Preserve reading order
- Handle multi-column layouts
- Extract document metadata

**Limitations**:
- Does not extract images
- Does not handle encrypted PDFs
- Does not preserve tables/charts

#### YouTube Extractor
**Responsibility**: Extract video transcripts
**Capabilities**:
- Fetch auto-generated transcripts
- Fetch manual captions if available
- Extract video metadata (title, duration, channel)
- Handle multiple languages (prefer English)

**Limitations**:
- Requires transcript availability
- Does not transcribe audio
- Does not extract visual content

### 4. Article Summarizer
**Responsibility**: Generate concise summaries using LLM
**Capabilities**:
- Create 150-word summary
- Extract 3-5 key points
- Identify main theme
- Generate descriptive title if missing

**Limitations**:
- Does not fact-check
- Does not add external context
- Depends on LLM availability

### 5. Embedding Generator
**Responsibility**: Create vector representations of text
**Capabilities**:
- Generate 768-dimensional embeddings
- Batch process multiple texts
- Normalize vectors for similarity

**Limitations**:
- Fixed embedding model
- Requires LLM API access
- Maximum 8000 tokens per text

### 6. Topic Clusterer
**Responsibility**: Group similar articles
**Capabilities**:
- Calculate cosine similarity between embeddings
- Apply hierarchical clustering
- Identify cluster themes
- Handle 2-100 articles

**Limitations**:
- Minimum 2 articles per cluster
- Maximum 5 clusters total
- Does not handle outliers well

### 7. Article Reorderer
**Responsibility**: Organize articles for optimal reading
**Capabilities**:
- Order clusters by importance
- Order within clusters by relevance
- Preserve chronological hints
- Apply user preferences

**Limitations**:
- Deterministic ordering only
- No dynamic reordering
- No personalization

### 8. Executive Summary Generator
**Responsibility**: Create narrative from top articles
**Capabilities**:
- Synthesize multi-cluster insights
- Generate story-driven narrative
- Identify cross-cutting themes
- Maintain 200-word limit

**Limitations**:
- Requires minimum 3 articles
- Depends on LLM availability
- No fact verification

### 9. Markdown Renderer
**Responsibility**: Format final digest output
**Capabilities**:
- Apply consistent markdown template
- Format for LinkedIn compatibility
- Include all metadata
- Generate table of contents

**Limitations**:
- Markdown output only
- No rich media embedding
- Fixed template structure

### 10. Cache Manager
**Responsibility**: Store and retrieve processed content
**Capabilities**:
- Key-value storage for articles
- TTL-based expiration (7 days)
- Content hash validation
- Concurrent read access

**Limitations**:
- Local storage only
- No distributed caching
- Simple eviction policy

### 11. Banner Generator (Optional)
**Responsibility**: Create social media banner images
**Capabilities**:
- Generate from content themes
- Optimize for LinkedIn (1200x627)
- Apply consistent styling
- Include text overlay

**Limitations**:
- Requires image API access
- Fixed dimensions only
- No custom branding

## Component Interaction Matrix

| Component | Depends On | Provides To | Data Format |
|-----------|------------|-------------|-------------|
| URL Parser | None | Content Fetcher | URL[] |
| Content Fetcher | URL Parser | Extractors | RawContent |
| HTML Extractor | Content Fetcher | Summarizer | CleanText |
| PDF Extractor | Content Fetcher | Summarizer | CleanText |
| YouTube Extractor | Content Fetcher | Summarizer | CleanText |
| Article Summarizer | Extractors | Embedding Generator, Clusterer | Summary |
| Embedding Generator | Summarizer | Topic Clusterer | Vector[] |
| Topic Clusterer | Embedding Generator | Article Reorderer | Cluster[] |
| Article Reorderer | Topic Clusterer | Executive Summary Gen | OrderedArticle[] |
| Executive Summary Generator | Article Reorderer | Markdown Renderer | Narrative |
| Markdown Renderer | All above | Output | Markdown |
| Cache Manager | None | All | Various |
| Banner Generator | Executive Summary | Output | ImagePath |

## Component Deployment

### Stateless Components (Can scale horizontally)
- URL Parser
- Content Fetcher
- All Extractors
- Article Summarizer
- Embedding Generator
- Markdown Renderer
- Banner Generator

### Stateful Components (Require coordination)
- Cache Manager (shared state)
- Topic Clusterer (needs all articles)
- Article Reorderer (needs all clusters)
- Executive Summary Generator (needs ordered articles)

## Error Handling Strategy

### Graceful Degradation
- Failed article fetch: Skip article, continue with others
- Failed summarization: Use first 200 words as fallback
- Failed clustering: Treat as single cluster
- Failed narrative generation: Use bullet points
- Failed banner generation: Proceed without banner

### Critical Failures (Stop processing)
- No valid URLs found
- All articles failed to fetch
- LLM API completely unavailable
- Cache corruption detected

## Performance Requirements

| Component | Latency Target | Throughput Target |
|-----------|---------------|-------------------|
| URL Parser | < 10ms | 1000 URLs/sec |
| Content Fetcher | < 2s per URL | 5 concurrent |
| HTML Extractor | < 100ms | 50 docs/sec |
| PDF Extractor | < 500ms | 10 docs/sec |
| YouTube Extractor | < 1s | 5 videos/sec |
| Article Summarizer | < 3s | 3 concurrent |
| Embedding Generator | < 500ms | Batch of 10 |
| Topic Clusterer | < 1s for 50 articles | N/A |
| Article Reorderer | < 50ms | N/A |
| Executive Summary | < 3s | N/A |
| Markdown Renderer | < 10ms | 100 docs/sec |
| Cache Manager | < 5ms read, < 20ms write | 1000 ops/sec |
| Banner Generator | < 5s | 1 concurrent |