# Verification Criteria - Content Digest System

## Component Verification

### 1. URL Parser

#### Functional Verification
```
GIVEN a markdown file with mixed URL formats
  - [Article Title](https://example.com/article)
  - https://example.com/raw-url
  - - [Bullet Link](https://example.com/bullet)
WHEN parser processes the file
THEN all three URLs are extracted
  AND duplicates are removed
  AND tracking parameters are stripped
```

#### Error Handling
```
GIVEN a markdown file with invalid URLs
  - [Broken](not-a-url)
  - [Malformed](htp://missing-t)
WHEN parser processes the file
THEN invalid URLs are logged
  AND valid URLs are still processed
  AND warning count equals 2
```

#### Performance
- Parse 100 URLs in < 10ms
- Handle 10MB markdown files without memory issues

### 2. Content Fetcher

#### Functional Verification
```
GIVEN a valid HTML URL
WHEN fetcher retrieves content
THEN response includes:
  - Raw HTML content
  - Content-Type header detection
  - Redirect chain if applicable
  AND rate limiting is applied between requests
```

#### Retry Logic
```
GIVEN a URL that fails initially
WHEN fetcher attempts retrieval
THEN retry occurs 3 times with exponential backoff
  - Attempt 1: immediate
  - Attempt 2: after 1 second
  - Attempt 3: after 2 seconds
AND final failure is reported after all retries
```

#### Content Type Detection
```
GIVEN URLs of different types:
  - https://example.com/article.html
  - https://example.com/document.pdf
  - https://youtube.com/watch?v=123
WHEN fetcher processes each
THEN correct content type is identified:
  - HTML for .html
  - PDF for .pdf
  - YOUTUBE for youtube.com
```

### 3. Content Extractors

#### HTML Extractor
```
GIVEN an HTML page with:
  - Navigation menu
  - Main article content
  - Sidebar ads
  - Footer links
WHEN extractor processes HTML
THEN output contains:
  - Only main article text
  - Preserved paragraph breaks
  - Extracted title and metadata
  AND excludes navigation, ads, footer
```

#### PDF Extractor
```
GIVEN a multi-page PDF document
WHEN extractor processes PDF
THEN output contains:
  - All text from all pages
  - Proper reading order
  - Document metadata
  AND text length > 100 characters
```

#### YouTube Extractor
```
GIVEN a YouTube video URL
WHEN extractor attempts extraction
THEN:
  IF transcript exists:
    - Return full transcript text
    - Include video metadata
  ELSE:
    - Return error NO_TRANSCRIPT
    - Log video ID for debugging
```

### 4. Article Summarizer

#### Summary Generation
```
GIVEN article text of 2000 words
WHEN summarizer processes with LLM
THEN output includes:
  - Summary of 150 words (±10%)
  - Exactly 3-5 key points
  - Identified main theme
  - Generated title if missing
```

#### LLM Failure Handling
```
GIVEN LLM API is unavailable
WHEN summarizer attempts generation
THEN:
  - Retry 2 times
  - If still failing, return first 200 words as fallback
  - Mark summary as "fallback" type
```

### 5. Topic Clusterer

#### Clustering Logic
```
GIVEN 10 articles with embeddings
WHEN clusterer processes with threshold 0.7
THEN:
  - Articles with similarity > 0.7 are grouped
  - Each cluster has minimum 2 articles
  - Maximum 5 clusters created
  - Outliers marked as unclustered
```

#### Edge Cases
```
GIVEN only 2 articles
WHEN clusterer processes
THEN:
  IF similarity > 0.7:
    - Single cluster with both articles
  ELSE:
    - No clusters formed
    - Both marked as unclustered
```

### 6. Article Reorderer

#### Ordering Logic
```
GIVEN 3 clusters with articles:
  - Cluster A: 3 articles, avg relevance 0.9
  - Cluster B: 4 articles, avg relevance 0.7
  - Cluster C: 2 articles, avg relevance 0.8
WHEN reorderer processes
THEN cluster order is: A, C, B
  AND within each cluster, articles ordered by centroid distance
```

### 7. Executive Summary Generator

#### Narrative Generation
```
GIVEN top 3 articles from each of 2 clusters
WHEN generator creates narrative
THEN output includes:
  - 200-word cohesive narrative
  - Mentions both cluster themes
  - Identifies cross-cutting themes
  - Maintains story-driven flow
```

#### Fallback Behavior
```
GIVEN LLM API fails
WHEN generator attempts narrative
THEN fallback to:
  - Bullet points of top articles
  - Simple theme listing
  - Basic concatenation of summaries
```

### 8. Markdown Renderer

#### Template Rendering
```
GIVEN complete digest data
WHEN renderer applies LinkedIn template
THEN output includes:
  - Executive summary section
  - Table of contents
  - Clustered articles with proper headers
  - Each article with title, summary, key points
  - Source citations
  - Metadata footer
```

#### Format Validation
```
GIVEN rendered markdown
WHEN validated
THEN:
  - All URLs are properly formatted
  - Headers use correct levels (##, ###)
  - Lists are properly indented
  - No broken markdown syntax
  - LinkedIn 3000 character limit considered
```

### 9. Cache Manager

#### Cache Hit
```
GIVEN a URL previously processed
WHEN cache checked
THEN:
  - Return cached content if TTL not expired
  - Increment hit counter
  - Update last accessed timestamp
```

#### Cache Miss
```
GIVEN a new URL
WHEN content processed and stored
THEN:
  - Content stored with URL as key
  - TTL set to 7 days
  - Content hash calculated and stored
  - Indexes updated
```

#### Cache Eviction
```
GIVEN cache approaching size limit
WHEN new content needs storage
THEN:
  - Evict oldest expired entries first
  - Then evict least recently used
  - Maintain 20% free space buffer
```

### 10. Banner Generator

#### Image Generation
```
GIVEN digest with themes ["AI", "Security", "Cloud"]
WHEN banner generator called
THEN:
  - Image prompt generated from themes
  - API called with 1200x627 dimensions
  - Image saved locally
  - Path returned for embedding
```

## Integration Tests

### End-to-End Digest Generation
```
GIVEN markdown file with 10 URLs
WHEN complete digest pipeline runs
THEN:
  1. All 10 URLs parsed correctly
  2. Content fetched (with cache checks)
  3. Text extracted from each
  4. Summaries generated for all
  5. Embeddings created
  6. Articles clustered (2-3 clusters expected)
  7. Clusters reordered by relevance
  8. Executive summary generated
  9. Markdown rendered
  10. Optional banner created
  AND total time < 30 seconds
```

### Cache-First Workflow
```
GIVEN 5 URLs already in cache
  AND 5 new URLs
WHEN digest pipeline runs
THEN:
  - 5 cache hits recorded
  - Only 5 new fetches performed
  - Total time significantly reduced (< 15 seconds)
```

### Graceful Degradation
```
GIVEN 10 URLs where 3 fail to fetch
WHEN digest pipeline runs
THEN:
  - 7 articles processed successfully
  - 3 failures logged with reasons
  - Digest generated with 7 articles
  - User notified of partial success
```

### Quick Read Workflow
```
GIVEN a single URL
WHEN quick summary requested
THEN:
  1. Cache checked first
  2. If miss, content fetched
  3. Text extracted
  4. Summary generated
  5. Markdown formatted
  6. Result returned in < 5 seconds
```

## Performance Benchmarks

### Component Latencies
| Component | Operation | Target | Measurement Method |
|-----------|-----------|--------|-------------------|
| URL Parser | Parse 100 URLs | < 10ms | Time from input to output |
| Content Fetcher | Single URL | < 2s | Including retries |
| HTML Extractor | 50KB HTML | < 100ms | Text extraction only |
| Summarizer | 2000 words | < 3s | LLM API call |
| Clusterer | 50 articles | < 1s | Complete clustering |
| Renderer | Full digest | < 10ms | Template application |

### System Throughput
- Concurrent article processing: 5 articles simultaneously
- Cache operations: 1000 ops/sec
- Full digest (10 articles): < 30 seconds
- Quick read: < 5 seconds

## Data Validation

### Article Content
```
VERIFY article.content:
  - Length between 100 and 10000 chars
  - Contains readable text (not binary)
  - Encoding is UTF-8
```

### Summary Quality
```
VERIFY summary:
  - Word count within 150 ±10%
  - Key points count 3-5
  - No duplicate key points
  - Theme identified and relevant
```

### Embedding Vectors
```
VERIFY embedding:
  - Dimension exactly 768
  - All values between -1 and 1
  - Non-zero vector
  - Normalized for similarity
```

### Cluster Validity
```
VERIFY cluster:
  - Minimum 2 articles
  - Average similarity > 0.5
  - Centroid properly calculated
  - Theme describes member articles
```

## Error Recovery

### Network Failures
```
WHEN network request fails
THEN:
  - Log error with context
  - Retry with backoff
  - After max retries, mark as failed
  - Continue with other articles
  - Include failure in final report
```

### LLM API Failures
```
WHEN LLM API unavailable
THEN:
  - Try alternate endpoint if configured
  - Fall back to basic extraction
  - Use cached summaries if available
  - Notify user of degraded output
```

### Cache Corruption
```
WHEN cache read fails
THEN:
  - Log corruption details
  - Treat as cache miss
  - Regenerate content
  - Attempt to repair cache entry
  - If persistent, clear affected entries
```

## User Experience Validation

### Progress Feedback
```
VERIFY during processing:
  - Progress shown for each article
  - Current step clearly indicated
  - Estimated time remaining shown
  - Failures immediately reported
```

### Output Quality
```
VERIFY final digest:
  - Executive summary is coherent
  - Articles properly grouped
  - No duplicate content
  - All links functional
  - Formatting clean for LinkedIn
```

### Error Messages
```
VERIFY error reporting:
  - Clear description of what failed
  - Actionable steps for user
  - No technical stack traces
  - Partial success clearly indicated
```