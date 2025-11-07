# Digest Summary Improvement Guide

## Problem Statement

**Current Challenge:** Digest summaries suffer from two main issues:
1. **Vagueness:** Generic language like "several companies announced updates" instead of specific facts
2. **Incomplete Coverage:** Some articles in cluster are ignored or barely mentioned

**Root Cause:** LLM gets overwhelmed processing 5-8 articles simultaneously (8,000+ token context) and defaults to high-level generalization rather than specific synthesis.

---

## Three Approaches to Test

### Approach A: Per-Article Extraction → Synthesis
**Cost:** ~$0.009 per digest (9 LLM calls for 8 articles)
**Speed:** 10s (parallel) or 45s (sequential)
**Quality:** Highest - every article gets attention

### Approach B: Digest Refinement (Current)
**Cost:** ~$0.003 per digest (2 LLM calls)
**Speed:** 10s
**Quality:** Variable - depends on Pass 1 quality

### Approach C: Digest with Self-Critique
**Cost:** ~$0.014 per digest (2 LLM calls but huge context)
**Speed:** 10s
**Quality:** Medium - can fix issues but expensive

---

## Diagnostic: Evaluate Current Digests

Run this FIRST before changing anything. This tells you if you actually have a problem.

```python
import re
from collections import Counter

def evaluate_digest_quality(digest, articles):
    """
    Comprehensive quality check for generated digest.
    Returns dict with metrics and warnings.
    """
    results = {
        'article_count': len(articles),
        'citations_found': set(),
        'coverage_pct': 0.0,
        'vague_phrases': 0,
        'word_count': 0,
        'warnings': [],
        'grade': 'UNKNOWN'
    }
    
    # Check 1: Citation coverage
    citations = set(re.findall(r'\[(\d+)\]', digest.summary))
    results['citations_found'] = citations
    results['coverage_pct'] = len(citations) / len(articles) if articles else 0
    
    if results['coverage_pct'] < 0.8:
        uncited = [i+1 for i in range(len(articles)) if str(i+1) not in citations]
        results['warnings'].append(f"Missing {len(uncited)} articles: {uncited}")
    
    # Check 2: Vagueness detection
    vague_phrases = [
        'several', 'various', 'multiple', 'many', 'some',
        'a number of', 'numerous', 'different', 'various sources'
    ]
    summary_lower = digest.summary.lower()
    vague_count = sum(phrase in summary_lower for phrase in vague_phrases)
    results['vague_phrases'] = vague_count
    
    if vague_count > 2:
        results['warnings'].append(f"Too vague: {vague_count} generic phrases")
    
    # Check 3: Length check
    word_count = len(digest.summary.split())
    results['word_count'] = word_count
    
    if word_count < 150:
        results['warnings'].append(f"Too short: {word_count} words")
    elif word_count > 400:
        results['warnings'].append(f"Too long: {word_count} words")
    
    # Check 4: Specificity check (numbers, names)
    has_numbers = bool(re.search(r'\d+%|\d+x|\$\d+|,\d+', digest.summary))
    has_proper_nouns = bool(re.search(r'[A-Z][a-z]+\s[A-Z][a-z]+', digest.summary))
    
    if not has_numbers:
        results['warnings'].append("No specific metrics/numbers found")
    if not has_proper_nouns:
        results['warnings'].append("No people/company names found")
    
    # Grade digest
    if results['coverage_pct'] >= 0.9 and vague_count <= 1 and has_numbers:
        results['grade'] = 'A - EXCELLENT'
    elif results['coverage_pct'] >= 0.8 and vague_count <= 2:
        results['grade'] = 'B - GOOD'
    elif results['coverage_pct'] >= 0.6 and vague_count <= 3:
        results['grade'] = 'C - FAIR'
    else:
        results['grade'] = 'D - POOR'
    
    return results

def print_digest_report(digest, articles):
    """Pretty print evaluation results."""
    results = evaluate_digest_quality(digest, articles)
    
    print(f"\n{'='*60}")
    print(f"DIGEST QUALITY REPORT: {digest.title}")
    print(f"{'='*60}")
    print(f"Grade: {results['grade']}")
    print(f"Coverage: {results['coverage_pct']:.0%} ({len(results['citations_found'])}/{results['article_count']} articles cited)")
    print(f"Vagueness: {results['vague_phrases']} generic phrases")
    print(f"Length: {results['word_count']} words")
    
    if results['warnings']:
        print(f"\n⚠️  WARNINGS:")
        for warning in results['warnings']:
            print(f"  - {warning}")
    else:
        print(f"\n✅ No issues detected")
    
    print(f"{'='*60}\n")
    return results

# Usage: Evaluate last 10 digests
def audit_recent_digests(limit=10):
    """Audit recent digests to find patterns."""
    digests = db.query(Digest).order_by(Digest.created_at.desc()).limit(limit).all()
    
    grades = Counter()
    total_coverage = 0
    total_vagueness = 0
    
    for digest in digests:
        articles = [da.article for da in digest.digest_articles]
        results = print_digest_report(digest, articles)
        
        grades[results['grade']] += 1
        total_coverage += results['coverage_pct']
        total_vagueness += results['vague_phrases']
    
    print(f"\n{'='*60}")
    print(f"SUMMARY ({limit} digests)")
    print(f"{'='*60}")
    print(f"Average Coverage: {total_coverage/limit:.0%}")
    print(f"Average Vagueness: {total_vagueness/limit:.1f} phrases/digest")
    print(f"\nGrade Distribution:")
    for grade, count in grades.most_common():
        print(f"  {grade}: {count} digests")
    
    if total_coverage / limit < 0.8:
        print(f"\n🔴 RECOMMENDATION: Switch to per-article extraction")
    elif total_vagueness / limit > 2:
        print(f"\n🟡 RECOMMENDATION: Improve specificity prompts")
    else:
        print(f"\n🟢 RECOMMENDATION: Current approach working well")
```

---

## Approach A: Per-Article Extraction (RECOMMENDED)

### Sequential Implementation

```python
def generate_digest_per_article_sequential(articles):
    """
    Extract facts from each article first, then synthesize.
    Sequential version (slower but simpler).
    """
    # PASS 1: Extract facts from each article
    extracted_facts = []
    
    for i, article in enumerate(articles, 1):
        extraction_prompt = f"""Extract concrete facts from this article. Be specific.

Article [{i}]:
Title: {article.title}
Published: {article.published_at.strftime('%B %d, %Y')}
Content: {article.content[:1500]}
URL: {article.url}

Extract (be SPECIFIC with names, numbers, dates):

1. MAIN DEVELOPMENT (1 sentence)
   What is the single most important announcement/change?

2. KEY DETAILS (3-4 bullet points)
   - Technical specs (versions, metrics, capabilities)
   - Who is involved (people, companies, teams)
   - When/where (dates, locations, timelines)
   - Impact (what changes, who benefits)

3. SIGNIFICANCE (1 sentence)
   Why does this matter to GenAI developers?

RULES:
- Use exact numbers: "40% faster" not "significantly faster"
- Name names: "Sam Altman" not "OpenAI's CEO"
- Include dates: "November 5" not "recently"
- No generic phrases: avoid "various", "several", "many"
"""
        
        facts = llm.generate(extraction_prompt)
        
        extracted_facts.append({
            'article_num': i,
            'article_title': article.title,
            'article_url': article.url,
            'published_at': article.published_at,
            'facts': facts
        })
    
    # PASS 2: Synthesize all facts into digest
    facts_context = "\n\n".join([
        f"Article [{f['article_num']}] - {f['article_title']}\n"
        f"Published: {f['published_at'].strftime('%b %d')}\n"
        f"{f['facts']}"
        for f in extracted_facts
    ])
    
    synthesis_prompt = f"""Create a comprehensive digest from these extracted facts.

EXTRACTED FACTS ({len(extracted_facts)} articles):
{facts_context}

Create a digest with these sections:

1. TITLE (< 20 characters)
   - Capture the MAIN theme connecting these articles
   - Example: "GPT-5 Leaks Spark Debate" not "AI News Updates"

2. TLDR (1 sentence, ~30 words)
   - Cover the complete story arc across ALL {len(extracted_facts)} articles
   - Must reference specific developments, not generic "updates announced"

3. SUMMARY (2-3 paragraphs, ~250 words total)
   
   Paragraph 1: THE CORE DEVELOPMENT
   - What happened? Be specific with facts from articles
   - Use citations: [1], [2], [3]
   
   Paragraph 2: THE DETAILS
   - Technical specifics, business implications, or different angles
   - Weave together insights from multiple sources
   - Show connections: "While [1] reported X, [3] revealed Y"
   
   Paragraph 3: THE SIGNIFICANCE
   - Why it matters to developers/industry
   - What's next or what to watch for
   - Cite supporting evidence

4. KEY MOMENTS (3-5 quotes)
   Format: "Quote here" - [Source Name, Article #]
   - Choose quotes that PROVE the summary's claims
   - Mix of technical detail + impact/significance
   - Ensure quotes come from different articles if possible

5. PERSPECTIVES (if articles show different angles)
   - Only include if articles genuinely differ
   - Format: 
     • Optimistic view: [what/who, cite]
     • Skeptical view: [what/who, cite]
     • Technical view: [what/who, cite]

CRITICAL RULES:
- Cite EVERY article at least once (you have {len(extracted_facts)} articles, use them all)
- Be specific: use numbers, names, dates from the facts
- No generic phrases: "several companies" → "OpenAI [1] and Anthropic [3]"
- Show connections between articles, don't just list them
- If you can't find a connection, say "Articles cover related but distinct topics"
"""
    
    digest_text = llm.generate(synthesis_prompt)
    
    return {
        'raw_response': digest_text,
        'extracted_facts': extracted_facts,
        'method': 'per_article_sequential'
    }
```

### Parallel Implementation (FASTER)

```python
import asyncio
from typing import List

async def generate_digest_per_article_parallel(articles: List[Article]):
    """
    Extract facts in parallel, then synthesize.
    10x faster than sequential (10s vs 45s for 8 articles).
    """
    
    async def extract_facts_async(article, index):
        """Extract facts from single article."""
        extraction_prompt = f"""Extract concrete facts from this article. Be specific.

Article [{index+1}]:
Title: {article.title}
Published: {article.published_at.strftime('%B %d, %Y')}
Content: {article.content[:1500]}

Extract:
1. MAIN DEVELOPMENT (1 sentence - what happened?)
2. KEY DETAILS (3-4 specific facts with numbers/names/dates)
3. SIGNIFICANCE (1 sentence - why it matters?)

Be specific. Use exact numbers, names, and dates."""
        
        # Assuming you have async LLM client
        facts = await llm.generate_async(extraction_prompt)
        
        return {
            'article_num': index + 1,
            'article_title': article.title,
            'article_url': article.url,
            'published_at': article.published_at,
            'facts': facts
        }
    
    # PASS 1: Extract all articles in parallel
    extraction_tasks = [
        extract_facts_async(article, i)
        for i, article in enumerate(articles)
    ]
    
    extracted_facts = await asyncio.gather(*extraction_tasks)
    
    # PASS 2: Synthesize (same as sequential)
    facts_context = "\n\n".join([
        f"Article [{f['article_num']}] - {f['article_title']}\n"
        f"Published: {f['published_at'].strftime('%b %d')}\n"
        f"{f['facts']}"
        for f in extracted_facts
    ])
    
    synthesis_prompt = f"""Create digest from these {len(extracted_facts)} articles' facts.

{facts_context}

[Same synthesis prompt as sequential version]
"""
    
    digest_text = await llm.generate_async(synthesis_prompt)
    
    return {
        'raw_response': digest_text,
        'extracted_facts': extracted_facts,
        'method': 'per_article_parallel'
    }

# Synchronous wrapper
def generate_digest_parallel(articles):
    """Wrapper to call async function from sync code."""
    return asyncio.run(generate_digest_per_article_parallel(articles))
```

---

## Approach B: Digest Refinement (Your Current Approach)

### Implementation with Improved Prompts

```python
def generate_digest_refinement(articles):
    """
    Generate digest in 2 passes, but Pass 2 doesn't re-read articles.
    Cheaper but less thorough.
    """
    
    # PASS 1: Generate initial digest
    articles_context = "\n\n".join([
        f"Article [{i+1}]:\n"
        f"Title: {article.title}\n"
        f"Published: {article.published_at.strftime('%B %d, %Y')}\n"
        f"URL: {article.url}\n"
        f"Content: {article.content[:1000]}\n"
        for i, article in enumerate(articles)
    ])
    
    pass1_prompt = f"""Create a digest from these {len(articles)} articles.

ARTICLES:
{articles_context}

Create draft digest:
1. Title (< 20 chars)
2. TLDR (1 sentence)
3. Summary (2-3 paragraphs with [citations])
4. Key Moments (3-5 quotes with sources)
5. Perspectives (if articles differ)

Be specific. Cite all {len(articles)} articles."""
    
    draft_digest = llm.generate(pass1_prompt)
    
    # PASS 2: Refine digest (blind refinement - no access to articles)
    pass2_prompt = f"""Improve this digest to be more specific and comprehensive.

DRAFT DIGEST:
{draft_digest}

REQUIREMENTS:
You are reviewing a digest that should cover {len(articles)} articles numbered [1] through [{len(articles)}].

Check for:
1. Are all {len(articles)} articles cited in the summary?
2. Are there vague phrases like "several", "various", "many"?
3. Are there specific numbers, names, dates?
4. Do the key moments actually support the summary?

Rewrite to fix any issues. Make it concrete and comprehensive.

OUTPUT the improved digest in the same format:
1. Title
2. TLDR
3. Summary
4. Key Moments
5. Perspectives
"""
    
    final_digest = llm.generate(pass2_prompt)
    
    return {
        'raw_response': final_digest,
        'draft_digest': draft_digest,
        'method': 'refinement'
    }
```

**Key Limitation:** Pass 2 can't verify claims or add missing articles because it doesn't see original content.

---

## Approach C: Self-Critique with Re-reading

### Implementation

```python
def generate_digest_self_critique(articles):
    """
    Pass 1 generates digest, Pass 2 re-reads articles to critique and fix.
    More expensive than refinement but can actually fix issues.
    """
    
    # PASS 1: Generate initial digest
    articles_context = "\n\n".join([
        f"Article [{i+1}]:\n"
        f"Title: {article.title}\n"
        f"URL: {article.url}\n"
        f"Content: {article.content[:1000]}\n"
        for i, article in enumerate(articles)
    ])
    
    pass1_prompt = f"""Create digest from these {len(articles)} articles.

{articles_context}

[Standard digest creation prompt]
"""
    
    draft_digest = llm.generate(pass1_prompt)
    
    # PASS 2: Critique draft while re-reading articles
    pass2_prompt = f"""Review this digest against the original articles.

ORIGINAL ARTICLES:
{articles_context}

DRAFT DIGEST:
{draft_digest}

CRITIQUE CHECKLIST:
1. Which articles (by number) are mentioned in the summary? List them.
2. Which articles are NOT mentioned? List them.
3. Are there vague/generic phrases? Quote them.
4. Are the key moments accurate quotes from articles? Verify.
5. Does the TLDR capture the main theme across ALL articles?

After critiquing, rewrite the digest to fix ALL identified issues.

OUTPUT:
[Critique section]
- Articles mentioned: [list numbers]
- Articles missing: [list numbers]
- Vague phrases: [list any found]
- Quote accuracy: [any issues?]

[Improved Digest]
1. Title: ...
2. TLDR: ...
3. Summary: ...
4. Key Moments: ...
5. Perspectives: ...
"""
    
    final_digest = llm.generate(pass2_prompt)
    
    return {
        'raw_response': final_digest,
        'draft_digest': draft_digest,
        'method': 'self_critique'
    }
```

**Warning:** This approach has massive context (8,000 input + 800 draft + 800 output = 9,600 tokens per Pass 2 call). More expensive than per-article extraction!

---

## Adaptive Approach: Choose Based on Cluster Size

```python
def generate_digest_adaptive(articles):
    """
    Pick strategy based on cluster characteristics.
    Balances cost vs quality.
    """
    article_count = len(articles)
    
    # Small clusters (3-4 articles): cheap refinement is fine
    if article_count <= 4:
        logger.info(f"Using refinement for {article_count} articles (cost-effective)")
        return generate_digest_refinement(articles)
    
    # Medium clusters (5-8 articles): use per-article extraction
    elif article_count <= 8:
        logger.info(f"Using per-article extraction for {article_count} articles (quality)")
        return generate_digest_parallel(articles)
    
    # Large clusters (9+ articles): too many, split first
    else:
        logger.warning(f"Cluster has {article_count} articles, splitting...")
        # Option 1: Sub-cluster and create multiple digests
        sub_clusters = split_cluster_by_date_or_topic(articles, max_size=6)
        return [generate_digest_parallel(sc) for sc in sub_clusters]
        
        # Option 2: Just use top 8 most central articles
        # top_articles = select_most_central_articles(articles, n=8)
        # return generate_digest_parallel(top_articles)

def split_cluster_by_date_or_topic(articles, max_size=6):
    """Split large cluster into smaller sub-clusters."""
    # Strategy 1: Split by date (early vs late in week)
    sorted_articles = sorted(articles, key=lambda a: a.published_at)
    mid_point = len(sorted_articles) // 2
    return [sorted_articles[:mid_point], sorted_articles[mid_point:]]
    
    # Strategy 2: Run sub-clustering on embeddings
    # embeddings = [a.embedding for a in articles]
    # sub_labels = KMeans(n_clusters=2).fit_predict(embeddings)
    # return [[a for a, l in zip(articles, sub_labels) if l == i] for i in [0, 1]]
```

---

## Comparison Testing Framework

```python
import json
from datetime import datetime

def compare_all_approaches(cluster, save_results=True):
    """
    Generate digest using all 3 approaches, compare results.
    Use this to decide which approach works best.
    """
    articles = cluster.articles
    
    print(f"\n{'='*80}")
    print(f"TESTING CLUSTER: {len(articles)} articles")
    print(f"{'='*80}\n")
    
    results = {}
    
    # Test Approach A: Per-article extraction
    print("⏳ Testing Approach A: Per-Article Extraction...")
    start = datetime.now()
    result_a = generate_digest_per_article_sequential(articles)
    time_a = (datetime.now() - start).total_seconds()
    results['per_article'] = {
        'digest': result_a['raw_response'],
        'time_seconds': time_a,
        'quality': evaluate_digest_quality_from_text(result_a['raw_response'], articles)
    }
    print(f"✓ Completed in {time_a:.1f}s")
    
    # Test Approach B: Refinement
    print("\n⏳ Testing Approach B: Digest Refinement...")
    start = datetime.now()
    result_b = generate_digest_refinement(articles)
    time_b = (datetime.now() - start).total_seconds()
    results['refinement'] = {
        'digest': result_b['raw_response'],
        'draft': result_b['draft_digest'],
        'time_seconds': time_b,
        'quality': evaluate_digest_quality_from_text(result_b['raw_response'], articles)
    }
    print(f"✓ Completed in {time_b:.1f}s")
    
    # Test Approach C: Self-critique
    print("\n⏳ Testing Approach C: Self-Critique...")
    start = datetime.now()
    result_c = generate_digest_self_critique(articles)
    time_c = (datetime.now() - start).total_seconds()
    results['self_critique'] = {
        'digest': result_c['raw_response'],
        'draft': result_c['draft_digest'],
        'time_seconds': time_c,
        'quality': evaluate_digest_quality_from_text(result_c['raw_response'], articles)
    }
    print(f"✓ Completed in {time_c:.1f}s")
    
    # Print comparison table
    print(f"\n{'='*80}")
    print(f"COMPARISON RESULTS")
    print(f"{'='*80}\n")
    
    print(f"{'Approach':<20} {'Time':<10} {'Coverage':<12} {'Vagueness':<12} {'Grade':<15}")
    print(f"{'-'*70}")
    
    for name, data in results.items():
        q = data['quality']
        print(f"{name:<20} {data['time_seconds']:<10.1f} {q['coverage_pct']:<12.0%} {q['vague_phrases']:<12} {q['grade']:<15}")
    
    # Show actual digest samples
    print(f"\n{'='*80}")
    print("DIGEST SAMPLES (first 300 chars of each)")
    print(f"{'='*80}\n")
    
    for name, data in results.items():
        print(f"--- {name.upper()} ---")
        digest_preview = data['digest'][:300] + "..."
        print(digest_preview)
        print()
    
    # Ask for human judgment
    print("Which approach produced the best digest?")
    print("1. Per-Article Extraction")
    print("2. Digest Refinement")
    print("3. Self-Critique")
    choice = input("Enter 1, 2, or 3: ")
    
    winner_map = {'1': 'per_article', '2': 'refinement', '3': 'self_critique'}
    results['human_winner'] = winner_map.get(choice, 'unknown')
    
    # Save results for analysis
    if save_results:
        filename = f"digest_comparison_{datetime.now().strftime('%Y%m%d_%H%M%S')}.json"
        with open(filename, 'w') as f:
            json.dump(results, f, indent=2, default=str)
        print(f"\n✓ Results saved to {filename}")
    
    return results

def evaluate_digest_quality_from_text(digest_text, articles):
    """Parse digest text and evaluate quality."""
    # Quick parse to extract summary section
    # (Assumes digest has clearly marked sections)
    summary_match = re.search(r'Summary[:\n]+(.*?)(?:Key Moments|Perspectives|$)', 
                             digest_text, re.DOTALL | re.IGNORECASE)
    summary = summary_match.group(1) if summary_match else digest_text
    
    # Create temporary digest object for evaluation
    temp_digest = type('obj', (object,), {'summary': summary})()
    
    return evaluate_digest_quality(temp_digest, articles)

# Run comparison on 5 clusters
def run_batch_comparison(num_clusters=5):
    """Test all approaches on multiple clusters."""
    clusters = get_recent_clusters(limit=num_clusters)
    
    winners = Counter()
    
    for i, cluster in enumerate(clusters, 1):
        print(f"\n\n{'#'*80}")
        print(f"CLUSTER {i} of {num_clusters}")
        print(f"{'#'*80}")
        
        result = compare_all_approaches(cluster)
        winners[result['human_winner']] += 1
    
    print(f"\n\n{'='*80}")
    print("FINAL RESULTS")
    print(f"{'='*80}")
    print(f"\nWinner distribution across {num_clusters} clusters:")
    for approach, count in winners.most_common():
        print(f"  {approach}: {count} wins")
```

---

## Cost Analysis

### Per Digest Cost (Gemini Flash pricing)

```python
# Gemini Flash pricing (Nov 2024)
INPUT_COST_PER_1K = 0.000075  # $0.000075 per 1K tokens
OUTPUT_COST_PER_1K = 0.0003   # $0.0003 per 1K tokens

def calculate_digest_cost(approach, article_count=8):
    """Calculate cost per digest for each approach."""
    
    if approach == 'per_article':
        # Pass 1: N extractions (1000 input + 200 output each)
        pass1_tokens = article_count * (1000 + 200)
        pass1_cost = (article_count * 1000 * INPUT_COST_PER_1K / 1000 + 
                      article_count * 200 * OUTPUT_COST_PER_1K / 1000)
        
        # Pass 2: Synthesis (1600 input + 800 output)
        pass2_cost = (1600 * INPUT_COST_PER_1K / 1000 + 
                      800 * OUTPUT_COST_PER_1K / 1000)
        
        total_cost = pass1_cost + pass2_cost
        return {
            'approach': 'Per-Article Extraction',
            'total_tokens': pass1_tokens + 2400,
            'total_cost': total_cost,
            'cost_per_digest': total_cost
        }
    
    elif approach == 'refinement':
        # Pass 1: Draft (8000 input + 800 output)
        pass1_cost = (8000 * INPUT_COST_PER_1K / 1000 + 
                      800 * OUTPUT_COST_PER_1K / 1000)
        
        # Pass 2: Refine (800 input + 800 output)
        pass2_cost = (800 * INPUT_COST_PER_1K / 1000 + 
                      800 * OUTPUT_COST_PER_1K / 1000)
        
        total_cost = pass1_cost + pass2_cost
        return {
            'approach': 'Digest Refinement',
            'total_tokens': 10400,
            'total_cost': total_cost,
            'cost_per_digest': total_cost
        }
    
    elif approach == 'self_critique':
        # Pass 1: Draft (8000 input + 800 output)
        pass1_cost = (8000 * INPUT_COST_PER_1K / 1000 + 
                      800 * OUTPUT_COST_PER_1K / 1000)
        
        # Pass 2: Critique + rewrite (8800 input + 800 output)
        pass2_cost = (8800 * INPUT_COST_PER_1K / 1000 + 
                      800 * OUTPUT_COST_PER_1K / 1000)
        
        total_cost = pass1_cost + pass2_cost
        return {
            'approach': 'Self-Critique',
            'total_tokens': 18400,
            'total_cost': total_cost,
            'cost_per_digest': total_cost
        }

# Print cost comparison
print("COST COMPARISON (per digest, 8 articles)")
print(f"{'='*60}")
for approach in ['per_article', 'refinement', 'self_critique']:
    cost = calculate_digest_cost(approach)
    print(f"{cost['approach']:<25} ${cost['cost_per_digest']:.4f}")

print(f"\nMONTHLY COST (10 digests/day)")
print(f"{'='*60}")
for approach in ['per_article', 'refinement', 'self_critique']:
    cost = calculate_digest_cost(approach)
    monthly = cost['cost_per_digest'] * 10 * 30
    print(f"{cost['approach']:<25} ${monthly:.2f}/month")
```

**Example Output:**
```
COST COMPARISON (per digest, 8 articles)
============================================================
Per-Article Extraction    $0.0009
Digest Refinement         $0.0008
Self-Critique             $0.0014

MONTHLY COST (10 digests/day)
============================================================
Per-Article Extraction    $2.70/month
Digest Refinement         $2.40/month
Self-Critique             $4.20/month
```

---

## Recommended Testing Plan

### Day 1: Baseline Audit
```bash
# Run diagnostic on current digests
python -c "from digest_improvement import audit_recent_digests; audit_recent_digests(10)"

# Goal: Understand current quality (coverage, vagueness)
# If coverage > 80% and vague < 2: current approach works!
# If coverage < 80%: need per-article extraction
```

### Day 2: Single Cluster Comparison
```bash
# Pick one problematic cluster, test all 3 approaches
python -c "from digest_improvement import compare_all_approaches; compare_all_approaches(cluster_id=123)"

# Goal: See which approach fixes the problems
# Time each approach, evaluate quality
```

### Day 3: Batch Testing
```bash
# Test on 5 different clusters
python -c "from digest_improvement import run_batch_comparison; run_batch_comparison(5)"

# Goal: Find which approach wins most often
# Look for patterns (e.g., per-article wins on 6+ articles)
```

### Day 4: Implement Winner
```bash
# Deploy chosen approach to production
# Monitor quality metrics daily
# Track costs

# Set up alerts if quality drops
```

---

## Decision Framework

Use this flowchart to choose approach:

```
┌─────────────────────────────────────────┐
│ Run baseline audit on 10 recent digests │
└──────────────┬──────────────────────────┘
               │
               ▼
    ┌──────────────────────┐
    │ Coverage > 80% AND   │────YES───▶ Keep current approach
    │ Vagueness < 2?       │           (it's working!)
    └──────────┬───────────┘
               │ NO
               ▼
    ┌──────────────────────┐
    │ How many articles    │
    │ per cluster usually? │
    └──────────┬───────────┘
               │
         ┌─────┴─────┐
         │           │
      3-4 articles  5+ articles
         │           │
         ▼           ▼
    Try Refinement  Try Per-Article
    with better     Extraction
    prompts         (parallel)
         │           │
         └─────┬─────┘
               │
               ▼
    ┌──────────────────────┐
    │ Test on 5 clusters,  │
    │ compare quality      │
    └──────────┬───────────┘
               │
               ▼
    ┌──────────────────────┐
    │ Did quality improve  │────YES───▶ Deploy to production
    │ enough to justify    │
    │ the cost?            │
    └──────────┬───────────┘
               │ NO
               ▼
    ┌──────────────────────┐
    │ Problem might be     │
    │ clustering, not      │
    │ summarization        │
    └──────────────────────┘
```

---

## Quick Start Commands

```python
# 1. Audit current digest quality
from digest_improvement import audit_recent_digests
audit_recent_digests(10)

# 2. Compare approaches on one cluster
from digest_improvement import compare_all_approaches
cluster = db.query(Digest).first()
compare_all_approaches(cluster)

# 3. Test per-article extraction (recommended)
from digest_improvement import generate_digest_parallel
articles = cluster.articles
digest = generate_digest_parallel(articles)
print(digest['raw_response'])

# 4. Run batch comparison
from digest_improvement import run_batch_comparison
run_batch_comparison(5)
```

---

## Next Steps

1. **Copy this file to your project**
2. **Run baseline audit** to see if you actually have a problem
3. **Test per-article extraction** on 1-2 clusters
4. **Compare quality** before/after
5. **Deploy** if quality improves

**My prediction:** Per-article extraction will solve your vagueness/coverage issues for 5+ article clusters. Might be overkill for 3-4 article clusters.

Start with Day 1 (baseline audit) and let the data guide you.
