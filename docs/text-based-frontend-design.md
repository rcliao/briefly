# Briefly Text-Based Frontend Design

**Version:** 2.0 (Major Refactor)
**Date:** November 5, 2025
**Status:** Design Phase - Awaiting Review
**Breaking Changes:** Yes - Complete UI overhaul

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Design Philosophy](#2-design-philosophy)
3. [Visual Design System](#3-visual-design-system)
4. [Information Architecture](#4-information-architecture)
5. [Database Schema Analysis](#5-database-schema-analysis)
6. [Component Specifications](#6-component-specifications)
7. [Keyboard Navigation System](#7-keyboard-navigation-system)
8. [Backend Changes](#8-backend-changes)
9. [Frontend Changes](#9-frontend-changes)
10. [Migration Plan](#10-migration-plan)
11. [Accessibility](#11-accessibility)
12. [Appendices](#12-appendices)

---

## 1. Executive Summary

### 1.1 Vision

Transform Briefly from a **card-based, mouse-centric** interface to a **text-list, keyboard-first** interface inspired by Hacker News. Focus on typography, minimal color, and keyboard efficiency for power users.

### 1.2 Core Principles

**Signal Over Noise:**
- Remove visual clutter (badges, cards, shadows, gradients)
- Use whitespace and typography to create hierarchy
- Minimal accent colors (Nord palette)

**Keyboard-First:**
- Vim-style navigation (j/k/l/h)
- Mouse-optional: All actions keyboard-accessible, mouse/touch fully supported
- Clear focus indicators

**Text-Focused:**
- Numbered lists instead of cards
- Multi-line entries: title + short summary (text-constrained for focus)
- Markdown rendered server-side (no client JS dependency)

**Performance:**
- Server-side rendering → Faster initial paint
- Minimal CSS/JS → Smaller bundle
- Semantic HTML → Better SEO

### 1.3 Breaking Changes Acknowledgment

This is a **major refactor** with intentional breaking changes:

| Current | New | Breaking? |
|---------|-----|-----------|
| daisyUI card components | Plain HTML text lists | ✅ Yes |
| Client-side markdown (marked.js) | Server-side HTML | ✅ Yes |
| Mouse-only navigation | Keyboard-first (mouse-optional) | ✅ Yes |
| Theme chip buttons | Plain text links | ✅ Yes |
| Unlimited digest list | Top 5 most recent | ✅ Yes |
| HTMX expand/collapse | Direct navigation to detail page | ✅ Yes |
| Go template engine | **Preserved** | ❌ No |

**Justification:** App is not in production use, allowing for bold redesign aligned with power-user audience.

---

## 2. Design Philosophy

### 2.1 Hacker News Inspiration

**What We're Adopting:**

```
Hacker News Layout:
┌────────────────────────────────────────┐
│ Hacker News    new · past · favorites  │ ← Minimal header
├────────────────────────────────────────┤
│ 1. Lorem ipsum dolor sit amet          │ ← Numbered list
│    31 points by alfa 2 min ago | 61 c  │   Metadata inline
│                                        │
│ 2. Aliquam mauris massa, rhoncus nec   │
│    173 points by beta 1h ago | 1143 c  │
│                                        │
│ 3. Show HN: consectetur adipiscing     │ ← Special prefixes
│    1331 points by gamma 3h | 145 c     │
└────────────────────────────────────────┘
```

**Key Patterns:**
- Numbered ordered list (`<ol>`)
- Title as primary element (largest text)
- Metadata on second line (smaller, gray text)
- Domain in parentheses
- Minimal spacing between items

### 2.2 Minimalism with Purpose

**Typography Hierarchy:**
```
Level 1: Digest title (16px, bold, high contrast)
Level 2: Digest summary (13px, normal, medium contrast) - Homepage preview
Level 3: Article title (14px, normal, high contrast)
Level 4: Metadata (12px, normal, low contrast)
Level 5: Action links (12px, underline, accent color)
```

**Color Purpose:**
```
Background:     Nord Polar Night #2E3440 (dark base)
Primary Text:   Nord Snow Storm #ECEFF4 (white)
Secondary Text: Nord Polar Night #4C566A (gray)
Accent:         Nord Frost #88C0D0 (blue)
Hover/Focus:    Nord Frost #81A1C1 (darker blue)
Danger:         Nord Aurora #BF616A (red, for errors)
Success:        Nord Aurora #A3BE8C (green, for status)
```

**Spacing System:**
```
Extra tight:  0.25rem (4px)  - Between inline elements
Tight:        0.5rem  (8px)  - List item padding
Normal:       1rem    (16px) - Section spacing
Loose:        2rem    (32px) - Major section breaks
```

### 2.3 Departure from Current Design

| Aspect | Current (Cards) | New (Text) |
|--------|-----------------|------------|
| **Visual Weight** | Heavy (shadows, borders, badges) | Light (text, subtle borders) |
| **Interaction** | Click buttons | Press keys (j/k/l/h) |
| **Density** | Low (large cards, lots of padding) | High (compact lines, minimal padding) |
| **Color Use** | Many colors (badges, backgrounds) | ~3 colors (text, accent, background) |
| **Focus Indicator** | None | Left border + background highlight |

---

## 3. Visual Design System

### 3.1 Nord Color Palette

**Official Palette:** [nordtheme.com](https://www.nordtheme.com/)

#### Polar Night (Backgrounds)
```css
--nord0:  #2E3440;  /* Base background */
--nord1:  #3B4252;  /* Lighter background (hover states) */
--nord2:  #434C5E;  /* Selection background */
--nord3:  #4C566A;  /* Comments, secondary text */
```

#### Snow Storm (Foregrounds)
```css
--nord4:  #D8DEE9;  /* Darker white (secondary text) */
--nord5:  #E5E9F0;  /* Off-white (body text) */
--nord6:  #ECEFF4;  /* Pure white (headings) */
```

#### Frost (Accents - Blue/Cyan)
```css
--nord7:  #8FBCBB;  /* Teal (less common) */
--nord8:  #88C0D0;  /* Light blue (links, primary accent) */
--nord9:  #81A1C1;  /* Medium blue (hover state) */
--nord10: #5E81AC;  /* Dark blue (active state) */
```

#### Aurora (Status Colors)
```css
--nord11: #BF616A;  /* Red (errors, danger) */
--nord12: #D08770;  /* Orange (warnings) */
--nord13: #EBCB8B;  /* Yellow (highlights) */
--nord14: #A3BE8C;  /* Green (success) */
--nord15: #B48EAD;  /* Purple (special) */
```

### 3.2 Typography

#### Font Stack
```css
font-family:
  -apple-system,
  BlinkMacSystemFont,
  'Segoe UI',
  'Roboto',
  'Oxygen',
  'Ubuntu',
  'Helvetica Neue',
  sans-serif;
```

**Why System Fonts:**
- Zero network requests
- Native OS rendering (best for readability)
- Familiar to users
- Perfect for text-heavy interfaces

#### Type Scale (Responsive)

**Base Font Size Strategy:**
```css
/* Set base font-size on <html> for responsive scaling */
html {
  font-size: 16px;  /* Desktop baseline */
}

/* Mobile: smaller base */
@media (max-width: 600px) {
  html { font-size: 14px; }
}

/* Tablet: medium base */
@media (min-width: 601px) and (max-width: 1024px) {
  html { font-size: 15px; }
}

/* All sizing now scales with base using rem/em */
```

**Type Scale (using rem):**
```css
/* Digest Title (h1) */
.digest-title {
  font-size: 1rem;        /* Scales: 14px mobile, 15px tablet, 16px desktop */
  line-height: 1.5;
  font-weight: 600;
  color: var(--nord6);
}

/* Digest Summary (homepage preview) */
.digest-summary {
  font-size: 0.8125rem;   /* ~13px desktop, 11px mobile */
  line-height: 1.5;
  font-weight: 400;
  color: var(--nord4);
}

/* Article Title (h2) */
.article-title {
  font-size: 0.875rem;    /* ~14px desktop, 12px mobile */
  line-height: 1.4;
  font-weight: 400;
  color: var(--nord6);
}

/* Metadata (small) */
.metadata {
  font-size: 0.75rem;     /* ~12px desktop, 10.5px mobile */
  line-height: 1.5;
  font-weight: 400;
  color: var(--nord4);
}

/* Action Links */
.action-link {
  font-size: 0.75rem;     /* ~12px desktop */
  font-weight: 400;
  color: var(--nord8);
  text-decoration: underline;
}

/* Body Text (rendered markdown) */
.prose {
  font-size: 0.875rem;    /* ~14px desktop, 12px mobile */
  line-height: 1.6;
  color: var(--nord5);
}
```

### 3.3 Spacing & Layout

#### Vertical Rhythm (using rem)
```css
/* List Items */
.digest-item {
  padding: 0.5rem 0;          /* 8px desktop, 7px mobile */
  border-bottom: 1px solid var(--nord1);
}

/* Digest Detail Sections */
.section {
  margin-bottom: 2rem;        /* 32px desktop, 28px mobile */
}

/* Section Content Padding */
.section-content {
  padding: 1rem 0 1rem 1.5rem;  /* Left indent for hierarchy */
  margin-bottom: 1rem;
}
```

#### Horizontal Spacing (using rem)
```css
/* Inline Metadata Separators */
.metadata-item::after {
  content: " · ";
  color: var(--nord3);
  margin: 0 0.25rem;          /* 4px */
}

/* Theme Filter Links */
.theme-link {
  margin-right: 0.75rem;      /* 12px */
}

/* Container Padding */
.container {
  padding: 1rem;              /* Scales with base font */
}
```

### 3.4 Interactive States

#### Focus Indicator (Keyboard Navigation)
```css
.digest-item.selected {
  background-color: var(--nord1);
  border-left: 3px solid var(--nord8);
  padding-left: 5px;  /* Compensate for border */
}
```

#### Hover State (Mouse)
```css
.digest-item:hover {
  background-color: rgba(59, 66, 82, 0.5);  /* --nord1 at 50% */
}
```

#### Link States
```css
a {
  color: var(--nord8);
  text-decoration: none;
}

a:hover {
  color: var(--nord9);
  text-decoration: underline;
}

a:active {
  color: var(--nord10);
}

a:visited {
  color: var(--nord7);  /* Slightly different shade */
}
```

---

## 4. Information Architecture

### 4.1 Site Structure

```
briefly.com/
├── /                      # Homepage (top 5 digests)
├── /digests/{id}          # Individual digest detail
├── /about                 # About page (NEW)
└── /contact               # Contact page (NEW)
```

### 4.2 Navigation Hierarchy

```
┌─────────────────────────────────────────────────────────────┐
│ Briefly                              Home · About · Contact │ ← Header
├─────────────────────────────────────────────────────────────┤
│ All · AI/ML · Cloud · Security · ... (12 more)             │ ← Theme filter
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ 1. Weekly Tech Digest - Nov 3, 2025                        │ ← Digest #1
│    AI acceleration meets economic reality, kernel...       │   (summary preview)
│    16 articles · Nov 3                                     │   (metadata)
│                                                             │
│ 2. Weekly Tech Digest - Oct 27, 2025                       │ ← Digest #2
│    Cloud infrastructure advances as security...            │
│    12 articles · Oct 27                                    │
│                                                             │
│ ... (3 more)                                               │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│ About · GitHub                                             │ ← Footer
└─────────────────────────────────────────────────────────────┘
```

### 4.3 Homepage → Digest Detail Flow

**Homepage (/):**
```
GET /?theme=all
    ↓
Show top 5 most recent digests (title + summary preview + metadata)
    ↓
User presses 'Enter' or 'l' or clicks title
    ↓
Navigate to /digests/{id}
    ↓
Full page digest detail with all sections
```

**Direct Link (/digests/{id}):**
```
GET /digests/3dcfd638-1ee6-4558-93dd-d5d3e0bdf642
    ↓
Full page digest detail (shareable URL)
    ↓
Server-rendered, no JavaScript required
```

### 4.4 Digest Detail Structure

**Required Sections (in order):**

1. **Title** - Digest name (e.g., "Weekly Tech Digest - Nov 3, 2025")
2. **Metadata** - Date, article count, theme count
3. **Summary** - Executive summary (2-3 paragraphs, markdown rendered with inline citations [1], [2], [3] linking to source articles for credibility)
4. **Key Moments** - Bullet list of highlights
5. **Perspectives** - Different viewpoints or themes
6. **Articles** - Numbered list of source articles (matches citation numbers)

**Example Layout:**
```
Weekly Tech Digest - Nov 3, 2025
16 articles · 3 themes · Nov 3, 2025

Summary
────────────────────────────────────────
This week's digest reveals critical fault lines across
AI deployment and core system security [1]. AI acceleration
is democratizing via novel tooling [2], while kernel backdoors
pose severe security risks [3]...

Key Moments
────────────────────────────────────────
• GPU acceleration meets economic reality
• Kernel backdoors pose severe security risk
• R interface unlocks Apple Silicon for ML

Perspectives
────────────────────────────────────────
AI Economics: The industry faces an unsustainable AI
investment bubble...

Security: Actionable security intelligence focuses on
stealthy persistence...

Articles
────────────────────────────────────────
1. R interface to Apple's MLX library (hughjonesd.github.io) · 23 min
2. Big Tech Needs $2T in AI Revenue by 2030 (bloomberg.com) · 1h
3. Linux kernel CVE-2025-0062 (lwn.net) · 3h
...
```

---

## 5. Database Schema Analysis

### 5.1 Current `core.Digest` Structure

```go
// From internal/core/core.go
type Digest struct {
    ID             string
    ArticleGroups  []ArticleGroup    // Articles grouped by theme
    Summaries      []Summary         // Article summaries
    DigestSummary  string            // Executive summary (MARKDOWN)
    Metadata       DigestMetadata
    UserFeedback   *UserFeedback
}

type DigestMetadata struct {
    Title          string            // ✅ Used for digest title
    DateGenerated  time.Time         // ✅ Used for date
    WordCount      int
    ArticleCount   int               // ✅ Used for metadata
    ProcessingTime time.Duration
    ProcessingCost ProcessingCost
    QualityScore   float64
}

type ArticleGroup struct {
    Theme    string                 // ✅ Used for perspectives
    Summary  string                 // ✅ Potential for perspectives
    Articles []Article
}
```

### 5.2 Field Mapping to New Structure

| New Section | Source | Current Field | Notes |
|-------------|--------|---------------|-------|
| **Title** | `DigestMetadata.Title` | ✅ Exists | Use as-is |
| **Metadata** | `DigestMetadata` | ✅ Exists | Extract date, article count, theme count |
| **Summary** | `Digest.DigestSummary` | ✅ Exists | Markdown → HTML (SSR) |
| **Key Moments** | ❌ Missing | N/A | **EXTRACT** from Summary or ArticleGroups |
| **Perspectives** | `ArticleGroup.Summary` | ⚠️ Partial | **USE** group summaries as perspectives |
| **Articles** | `ArticleGroup.Articles` | ✅ Exists | Flatten across all groups |

### 5.3 Missing Fields: Key Moments

**Option 1: Extract from Existing Data (Recommended)**

Parse the `DigestSummary` markdown and extract lines starting with `**` or `-`:

```go
// Example extraction logic
func extractKeyMoments(summary string) []string {
    lines := strings.Split(summary, "\n")
    var moments []string

    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        // Match "**Highlight**" or "- Bullet point"
        if strings.HasPrefix(trimmed, "**") || strings.HasPrefix(trimmed, "- ") {
            moments = append(moments, trimmed)
        }
    }

    return moments
}
```

**Option 2: Add to Digest Generation (Future)**

Modify the digest generation pipeline to explicitly create a `KeyMoments []string` field:

```go
type Digest struct {
    // ... existing fields
    KeyMoments  []string  // NEW: Extracted highlights
}
```

**Decision for V1:** Use **Option 1** (extract from existing data). Avoids backend changes.

### 5.4 Perspectives Structure

**Current:** `ArticleGroup.Summary` is a string per theme.

**Proposed:** Restructure as named perspectives:

```go
type Perspective struct {
    Title   string  // e.g., "AI Economics", "Security Threats"
    Content string  // Group summary (markdown)
}

// In template:
{{ range .Perspectives }}
<div class="perspective">
    <h3>{{ .Title }}</h3>
    <p>{{ .Content }}</p>
</div>
{{ end }}
```

**Mapping:**
```
ArticleGroup.Theme = "AI & Machine Learning"
ArticleGroup.Summary = "This week's AI news..."
    ↓
Perspective.Title = "AI & Machine Learning"
Perspective.Content = "This week's AI news..."
```

**Implementation:** Transform `ArticleGroups` into `Perspectives` in the handler.

### 5.5 Schema Change Recommendations

**For V1 (No DB Changes):**
- ✅ Extract Key Moments from `DigestSummary` at render time
- ✅ Map `ArticleGroups` to Perspectives at render time

**For V2 (Optional Future Enhancement):**
- Add `Digest.KeyMoments []string` field
- Add `Digest.Perspectives []Perspective` field
- Update digest generation to populate these explicitly

---

## 6. Component Specifications

### 6.1 Page: Homepage (`pages/home.html`)

**Template Structure:**
```html
{{ define "content" }}
<div class="container">
    <!-- Theme Filter -->
    {{ template "partials/theme-filter.html" . }}

    <!-- Digest List (Top 5) -->
    <ol class="digest-list">
        {{ range .Digests }}
            {{ template "partials/digest-item.html" . }}
        {{ end }}
    </ol>

    <!-- Keyboard Shortcut Hint -->
    <div class="keyboard-hint">
        j/k: navigate · l/Enter: expand · h: collapse · o: open link
    </div>
</div>
{{ end }}
```

**Input Data:**
```go
type HomePageData struct {
    Themes      []Theme            // For theme filter
    Digests     []DigestListItem   // Top 5 most recent
    ActiveTheme string             // Currently selected theme
}

type DigestListItem struct {
    ID           string
    Title        string
    ArticleCount int
    DateGenerated time.Time
    Collapsed    bool   // UI state (always true on load)
}
```

**CSS Classes:**
```css
.container {
    max-width: 800px;
    margin: 0 auto;
    padding: 16px;
}

.digest-list {
    list-style: decimal;
    padding-left: 24px;
    margin: 16px 0;
}

.keyboard-hint {
    font-size: 12px;
    color: var(--nord4);
    text-align: center;
    margin-top: 32px;
    padding: 8px;
    border-top: 1px solid var(--nord1);
}
```

---

### 6.2 Partial: Theme Filter (`partials/theme-filter.html`)

**Current (Chips):**
```html
<!-- OLD: daisyUI button chips -->
<button class="btn btn-sm btn-primary">🤖 AI/ML</button>
```

**New (Text Links):**
```html
<nav class="theme-filter">
    <a href="/?theme=all"
       class="{{ if not .ActiveTheme }}active{{ end }}"
       hx-get="/?theme=all"
       hx-target=".digest-list"
       hx-push-url="true">
        All
    </a>
    {{ range .Themes }}
    <a href="/?theme={{ .ID }}"
       class="{{ if eq .ID $.ActiveTheme }}active{{ end }}"
       hx-get="/?theme={{ .ID }}"
       hx-target=".digest-list"
       hx-push-url="true">
        {{ .Name }}
    </a>
    {{ end }}
</nav>
```

**CSS:**
```css
.theme-filter {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    padding: 12px 0;
    border-bottom: 1px solid var(--nord1);
    margin-bottom: 16px;
}

.theme-filter a {
    color: var(--nord4);
    text-decoration: none;
    font-size: 13px;
}

.theme-filter a:hover {
    color: var(--nord6);
}

.theme-filter a.active {
    color: var(--nord8);
    font-weight: 600;
}
```

---

### 6.3 Partial: Digest Item (`partials/digest-item.html`)

**Homepage List Item (Multi-line):**
```html
<li class="digest-item" data-digest-id="{{ .ID }}" tabindex="0">
    <!-- Title -->
    <div class="digest-title">
        <a href="/digests/{{ .ID }}">{{ .Title }}</a>
    </div>

    <!-- Summary Preview (truncated to ~100 chars) -->
    <div class="digest-summary">
        {{ truncate .SummaryPreview 100 }}...
    </div>

    <!-- Metadata -->
    <div class="digest-meta">
        <span class="article-count">{{ .ArticleCount }} articles</span>
        <span class="date">{{ formatDate .DateGenerated }}</span>
    </div>
</li>
```

**Input Data:**
```go
type DigestListItem struct {
    ID             string
    Title          string
    SummaryPreview string     // First ~100 chars of digest summary
    ArticleCount   int
    DateGenerated  time.Time
}
```

**CSS (using rem/em):**
```css
.digest-item {
    padding: 0.75rem 0;              /* ~12px desktop */
    border-bottom: 1px solid var(--nord1);
    transition: background-color 0.2s;
    cursor: pointer;
}

.digest-item:hover {
    background-color: rgba(59, 66, 82, 0.5);  /* --nord1 at 50% */
}

.digest-item.selected {
    background-color: var(--nord1);
    border-left: 3px solid var(--nord8);
    padding-left: 0.3125rem;        /* 5px compensation for border */
}

.digest-title a {
    font-size: 1rem;                /* 16px desktop, scales responsive */
    font-weight: 600;
    color: var(--nord6);
    text-decoration: none;
}

.digest-title a:hover {
    color: var(--nord8);
}

.digest-summary {
    font-size: 0.8125rem;           /* ~13px desktop */
    line-height: 1.5;
    color: var(--nord4);
    margin-top: 0.25rem;            /* 4px */
}

.digest-meta {
    font-size: 0.75rem;             /* ~12px desktop */
    color: var(--nord4);
    margin-top: 0.25rem;            /* 4px */
}

.digest-meta span::after {
    content: " · ";
    margin: 0 0.25rem;              /* 4px */
}

.digest-meta span:last-of-type::after {
    content: "";
}

/* Digest Detail Page Sections */
.section {
    margin-bottom: 1.5rem;          /* ~24px desktop */
}

.section-title {
    font-size: 0.875rem;            /* ~14px desktop */
    font-weight: 600;
    color: var(--nord6);
    margin-bottom: 0.5rem;          /* 8px */
    border-bottom: 1px solid var(--nord2);
    padding-bottom: 0.25rem;        /* 4px */
}

/* Prose (Rendered Markdown) */
.prose {
    font-size: 0.875rem;                /* ~14px desktop */
    line-height: 1.6;
    color: var(--nord5);
}

.prose p {
    margin-bottom: 0.75rem;             /* 12px */
}

.prose strong {
    color: var(--nord6);
    font-weight: 600;
}

.prose em {
    color: var(--nord4);
    font-style: italic;
}

.prose ul,
.prose ol {
    margin-left: 1.25rem;               /* 20px */
    margin-bottom: 0.75rem;             /* 12px */
}

.prose li {
    margin-bottom: 0.25rem;             /* 4px */
}

.prose code {
    background-color: var(--nord1);
    padding: 0.125rem 0.25rem;          /* 2px 4px */
    border-radius: 0.1875rem;           /* 3px */
    font-family: 'Monaco', 'Courier New', monospace;
    font-size: 0.75rem;                 /* 12px */
}

/* Citation Links */
.prose a[href^="#article-"] {
    color: var(--nord8);
    text-decoration: none;
    font-weight: 600;
    padding: 0 0.125rem;
}

.prose a[href^="#article-"]:hover {
    background-color: var(--nord1);
    text-decoration: underline;
}

/* Key Moments */
.key-moments {
    list-style: disc;
    padding-left: 20px;
}

.key-moments li {
    margin-bottom: 8px;
    color: var(--nord5);
}

/* Perspectives */
.perspective {
    margin-bottom: 16px;
}

.perspective h4 {
    font-size: 13px;
    font-weight: 600;
    color: var(--nord8);
    margin-bottom: 4px;
}
```

---

### 6.4 Partial: Article Item (`partials/article-item.html`)

**Structure:**
```html
<li class="article-item" data-article-id="{{ .ID }}">
    <div class="article-title">
        <a href="{{ .URL }}"
           target="_blank"
           rel="noopener noreferrer">
            {{ .Title }}
        </a>
        <span class="article-domain">({{ extractDomain .URL }})</span>
    </div>
    <div class="article-meta">
        <span class="article-date">{{ formatTimeAgo .DatePublished }}</span>
        {{ if eq .ContentType "pdf" }}
        <span class="content-badge">[PDF]</span>
        {{ else if eq .ContentType "youtube" }}
        <span class="content-badge">[Video]</span>
        {{ end }}
    </div>
</li>
```

**CSS:**
```css
.article-list {
    list-style: decimal;
    padding-left: 24px;
}

.article-item {
    margin-bottom: 12px;
}

.article-title {
    font-size: 14px;
}

.article-title a {
    color: var(--nord6);
    text-decoration: none;
}

.article-title a:hover {
    color: var(--nord8);
    text-decoration: underline;
}

.article-title a:visited {
    color: var(--nord7);  /* Visited link color */
}

.article-domain {
    font-size: 12px;
    color: var(--nord4);
    margin-left: 4px;
}

.article-meta {
    font-size: 11px;
    color: var(--nord4);
    margin-top: 2px;
}

.content-badge {
    background-color: var(--nord2);
    padding: 2px 6px;
    border-radius: 3px;
    font-size: 10px;
    color: var(--nord5);
    margin-left: 8px;
}
```

---

### 6.5 Page: Digest Detail (`pages/digest-detail.html`)

**Full Page (No HTMX, Direct Link):**

```html
{{ define "content" }}
<div class="container">
    <!-- Breadcrumb -->
    <nav class="breadcrumb">
        <a href="/">← Back to Home</a>
    </nav>

    <!-- Digest Header -->
    <header class="digest-header">
        <h1>{{ .Title }}</h1>
        <div class="digest-meta">
            <span>{{ .ArticleCount }} articles</span>
            <span>{{ len .Perspectives }} themes</span>
            <span>{{ formatDate .DateGenerated }}</span>
        </div>
    </header>

    <!-- Same sections as expanded partial -->
    <section class="section">
        <h2 class="section-title">Summary</h2>
        <div class="prose">{{ .SummaryHTML }}</div>
    </section>

    <section class="section">
        <h2 class="section-title">Key Moments</h2>
        <ul class="key-moments">
            {{ range .KeyMoments }}
            <li>{{ . }}</li>
            {{ end }}
        </ul>
    </section>

    <section class="section">
        <h2 class="section-title">Perspectives</h2>
        {{ range .Perspectives }}
        <div class="perspective">
            <h3>{{ .Title }}</h3>
            <div class="prose">{{ .ContentHTML }}</div>
        </div>
        {{ end }}
    </section>

    <section class="section">
        <h2 class="section-title">Articles ({{ len .Articles }})</h2>
        <ol class="article-list">
            {{ range .Articles }}
                {{ template "partials/article-item.html" . }}
            {{ end }}
        </ol>
    </section>
</div>
{{ end }}
```

**CSS:**
```css
.breadcrumb {
    margin-bottom: 16px;
}

.breadcrumb a {
    color: var(--nord8);
    font-size: 13px;
}

.digest-header {
    margin-bottom: 32px;
    border-bottom: 2px solid var(--nord2);
    padding-bottom: 16px;
}

.digest-header h1 {
    font-size: 24px;
    font-weight: 700;
    color: var(--nord6);
    margin-bottom: 8px;
}

.digest-header .digest-meta {
    font-size: 13px;
    color: var(--nord4);
}
```

---

## 7. Keyboard Navigation System

### 7.1 Keyboard Shortcuts

**Homepage Navigation:**

| Key | Action | Description |
|-----|--------|-------------|
| `j` | Next | Move selection down (next digest) |
| `k` | Previous | Move selection up (previous digest) |
| `l` / `Enter` | View | Navigate to digest detail page |
| `o` | Open | Navigate to digest detail page (same as Enter) |
| `gg` | Top | Jump to top of page |
| `G` | Bottom | Jump to bottom of page |
| `/` | Search | Focus search input (future feature) |
| `?` | Help | **Show keyboard shortcuts modal** |
| `Esc` | Close | Close keyboard shortcuts modal |

**Digest Detail Page Navigation:**

| Key | Action | Description |
|-----|--------|-------------|
| `s` | Summary | Jump to Summary section |
| `k` | Key Moments | Jump to Key Moments section |
| `p` | Perspectives | Jump to Perspectives section |
| `a` | Articles | Jump to Articles section |
| `y` | Copy | Copy summary markdown to clipboard |
| `j` / `k` | Navigate | Move between articles (when in Articles section) |
| `o` / `Enter` | Open | Open selected article in new tab |
| `h` | Back | Return to homepage |
| `gg` | Top | Jump to top of page |
| `G` | Bottom | Jump to bottom of page |
| `?` | Help | **Show keyboard shortcuts modal** |
| `Esc` | Close | Close modal or clear selection |

### 7.2 Focus Management

**Selection State:**
```css
.digest-item.selected {
    background-color: var(--nord1);
    border-left: 3px solid var(--nord8);
    padding-left: 5px;  /* Compensate for border width */
}

.article-item.selected {
    background-color: var(--nord2);
    border-left: 3px solid var(--nord9);
    padding-left: 5px;
}
```

**Homepage Navigation Behavior:**
```
[Digest 1]  ← Selected (j/k moves here)
  Weekly Tech Digest - Nov 3, 2025
  AI acceleration meets economic reality...
  16 articles · Nov 3

[Digest 2]
  Weekly Tech Digest - Oct 27, 2025
  Cloud infrastructure advances...
  12 articles · Oct 27

[Digest 3]
[Digest 4]
[Digest 5]
```

**Digest Detail Page Navigation:**
```
[Summary Section]     ← Press 's' to jump here
[Key Moments]         ← Press 'k' to jump here
[Perspectives]        ← Press 'p' to jump here
[Articles Section]    ← Press 'a' to jump here
  [Article 1]  ← j/k navigates within articles
  [Article 2]
  [Article 3]
```

**Navigation Flow:**
- Homepage: `j/k` → Move between digests, `Enter/l` → Navigate to detail page
- Detail page: `s/k/p/a` → Jump to sections, `j/k` → Navigate articles, `o/Enter` → Open article
- `h` on detail page → Return to homepage
- `y` on detail page → Copy summary markdown to clipboard
- `?` on any page → Show keyboard shortcuts modal
- `Esc` → Close modal or clear selection

### 7.3 Keyboard Shortcuts Modal

**Purpose:** Help users discover keyboard shortcuts with `?` keybind.

**Modal Structure:**

```html
<!-- Keyboard Shortcuts Modal (hidden by default) -->
<div id="keyboard-help-modal" class="modal" style="display: none;" role="dialog" aria-labelledby="modal-title" aria-modal="true">
    <div class="modal-overlay" onclick="closeKeyboardModal()"></div>
    <div class="modal-content">
        <div class="modal-header">
            <h2 id="modal-title">Keyboard Shortcuts</h2>
            <button class="modal-close" onclick="closeKeyboardModal()" aria-label="Close modal">×</button>
        </div>
        <div class="modal-body">
            <!-- Homepage Shortcuts -->
            <section class="shortcut-section">
                <h3>Homepage</h3>
                <dl class="shortcut-list">
                    <div class="shortcut-item">
                        <dt><kbd>j</kbd> / <kbd>k</kbd></dt>
                        <dd>Navigate up/down</dd>
                    </div>
                    <div class="shortcut-item">
                        <dt><kbd>Enter</kbd> / <kbd>l</kbd></dt>
                        <dd>View digest detail</dd>
                    </div>
                    <div class="shortcut-item">
                        <dt><kbd>g</kbd><kbd>g</kbd></dt>
                        <dd>Jump to top</dd>
                    </div>
                    <div class="shortcut-item">
                        <dt><kbd>G</kbd></dt>
                        <dd>Jump to bottom</dd>
                    </div>
                </dl>
            </section>

            <!-- Digest Detail Shortcuts -->
            <section class="shortcut-section">
                <h3>Digest Detail</h3>
                <dl class="shortcut-list">
                    <div class="shortcut-item">
                        <dt><kbd>s</kbd></dt>
                        <dd>Jump to Summary</dd>
                    </div>
                    <div class="shortcut-item">
                        <dt><kbd>k</kbd></dt>
                        <dd>Jump to Key Moments</dd>
                    </div>
                    <div class="shortcut-item">
                        <dt><kbd>p</kbd></dt>
                        <dd>Jump to Perspectives</dd>
                    </div>
                    <div class="shortcut-item">
                        <dt><kbd>a</kbd></dt>
                        <dd>Jump to Articles</dd>
                    </div>
                    <div class="shortcut-item">
                        <dt><kbd>y</kbd></dt>
                        <dd>Copy summary</dd>
                    </div>
                    <div class="shortcut-item">
                        <dt><kbd>o</kbd> / <kbd>Enter</kbd></dt>
                        <dd>Open article</dd>
                    </div>
                    <div class="shortcut-item">
                        <dt><kbd>h</kbd></dt>
                        <dd>Back to home</dd>
                    </div>
                </dl>
            </section>

            <!-- General Shortcuts -->
            <section class="shortcut-section">
                <h3>General</h3>
                <dl class="shortcut-list">
                    <div class="shortcut-item">
                        <dt><kbd>?</kbd></dt>
                        <dd>Show this help</dd>
                    </div>
                    <div class="shortcut-item">
                        <dt><kbd>Esc</kbd></dt>
                        <dd>Close modal</dd>
                    </div>
                </dl>
            </section>
        </div>
    </div>
</div>
```

**Modal CSS:**

```css
/* Modal Overlay */
.modal {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    z-index: 1000;
    display: flex;
    align-items: center;
    justify-content: center;
}

.modal-overlay {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background-color: rgba(46, 52, 64, 0.8);  /* --nord0 at 80% */
    backdrop-filter: blur(4px);
}

/* Modal Content */
.modal-content {
    position: relative;
    background-color: var(--nord1);
    border: 1px solid var(--nord2);
    border-radius: 0.5rem;           /* 8px */
    max-width: 48rem;                /* 768px */
    max-height: 90vh;
    width: 90%;
    overflow-y: auto;
    box-shadow: 0 1rem 3rem rgba(0, 0, 0, 0.5);
    z-index: 1001;
}

/* Modal Header */
.modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1.5rem;
    border-bottom: 1px solid var(--nord2);
}

.modal-header h2 {
    font-size: 1.25rem;              /* 20px */
    font-weight: 600;
    color: var(--nord6);
    margin: 0;
}

.modal-close {
    background: none;
    border: none;
    color: var(--nord4);
    font-size: 2rem;                 /* 32px */
    line-height: 1;
    cursor: pointer;
    padding: 0;
    width: 2rem;
    height: 2rem;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 0.25rem;          /* 4px */
    transition: background-color 0.2s, color 0.2s;
}

.modal-close:hover {
    background-color: var(--nord2);
    color: var(--nord6);
}

/* Modal Body */
.modal-body {
    padding: 1.5rem;
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(15rem, 1fr));
    gap: 2rem;
}

/* Shortcut Sections */
.shortcut-section {
    min-width: 0;                    /* Prevent grid blowout */
}

.shortcut-section h3 {
    font-size: 0.875rem;             /* 14px */
    font-weight: 600;
    color: var(--nord8);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-bottom: 0.75rem;          /* 12px */
}

/* Shortcut List */
.shortcut-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;                     /* 8px */
}

.shortcut-item {
    display: flex;
    align-items: center;
    gap: 1rem;                       /* 16px */
}

.shortcut-item dt {
    display: flex;
    gap: 0.25rem;                    /* 4px */
    min-width: 6rem;                 /* 96px - fixed width for alignment */
}

.shortcut-item dd {
    color: var(--nord5);
    font-size: 0.875rem;             /* 14px */
    margin: 0;
}

/* Keyboard Key Styling */
kbd {
    display: inline-block;
    padding: 0.25rem 0.5rem;         /* 4px 8px */
    font-family: 'Monaco', 'Courier New', monospace;
    font-size: 0.75rem;              /* 12px */
    font-weight: 600;
    color: var(--nord6);
    background-color: var(--nord2);
    border: 1px solid var(--nord3);
    border-radius: 0.25rem;          /* 4px */
    box-shadow: 0 1px 0 var(--nord3);
    white-space: nowrap;
}

/* Responsive */
@media (max-width: 600px) {
    .modal-content {
        width: 95%;
        max-height: 85vh;
    }

    .modal-body {
        grid-template-columns: 1fr;
        gap: 1.5rem;
        padding: 1rem;
    }

    .modal-header {
        padding: 1rem;
    }

    .shortcut-item {
        flex-direction: column;
        align-items: flex-start;
        gap: 0.25rem;
    }

    .shortcut-item dt {
        min-width: auto;
    }
}
```

**Footer Keyboard Hints:**

Add subtle keyboard shortcut hints to the footer to help with discoverability.

```html
<footer>
    <div class="footer-content">
        <nav class="footer-links">
            <a href="/about">About</a> ·
            <a href="https://github.com/your-org/briefly" target="_blank">GitHub</a>
        </nav>
        <div class="keyboard-hints">
            <span class="hint"><kbd>j</kbd>/<kbd>k</kbd> navigate</span>
            <span class="hint"><kbd>?</kbd> shortcuts</span>
        </div>
    </div>
</footer>
```

**Footer CSS:**

```css
footer {
    margin-top: 3rem;                /* 48px */
    padding: 1rem 0;                 /* 16px 0 */
    border-top: 1px solid var(--nord1);
}

.footer-content {
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: 1rem;
}

.footer-links {
    font-size: 0.75rem;              /* 12px */
    color: var(--nord4);
    text-align: center;
}

.keyboard-hints {
    display: flex;
    gap: 1rem;                       /* 16px */
    font-size: 0.75rem;              /* 12px */
    color: var(--nord3);             /* Greyed out */
}

.keyboard-hints .hint {
    display: flex;
    align-items: center;
    gap: 0.25rem;                    /* 4px */
}

.keyboard-hints kbd {
    font-size: 0.625rem;             /* 10px - smaller in footer */
    padding: 0.125rem 0.375rem;      /* 2px 6px */
    background-color: var(--nord1);
    border-color: var(--nord2);
}

@media (max-width: 600px) {
    .footer-content {
        flex-direction: column;
        text-align: center;
    }

    .keyboard-hints {
        justify-content: center;
    }
}
```

### 7.4 JavaScript Implementation

**File:** `web/static/js/keyboard-nav.js`

```javascript
// Keyboard Navigation for Briefly
(function() {
    'use strict';

    let selectedElement = null;
    let navigationLevel = 'digest';  // 'digest' or 'article'

    // Initialize on page load
    document.addEventListener('DOMContentLoaded', function() {
        selectFirstDigest();
        attachKeyboardListeners();
    });

    // Select first digest on load
    function selectFirstDigest() {
        const firstDigest = document.querySelector('.digest-item');
        if (firstDigest) {
            selectElement(firstDigest, 'digest');
        }
    }

    // Attach keyboard event listeners
    function attachKeyboardListeners() {
        document.addEventListener('keydown', function(e) {
            // Ignore if user is typing in input
            if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') {
                return;
            }

            // Prevent default for navigation keys
            const navKeys = ['j', 'k', 'l', 'h', 'o', 'g', 'G', '/', '?'];
            if (navKeys.includes(e.key)) {
                e.preventDefault();
            }

            handleKeyPress(e);
        });
    }

    // Main key handler
    function handleKeyPress(e) {
        // Check if modal is open
        const modal = document.getElementById('keyboard-help-modal');
        const isModalOpen = modal && modal.style.display === 'flex';

        // Handle Escape key
        if (e.key === 'Escape') {
            if (isModalOpen) {
                closeKeyboardModal();
            } else if (selectedElement) {
                // Clear selection if no modal
                selectedElement.classList.remove('selected');
                selectedElement = null;
            }
            return;
        }

        // Don't handle other keys if modal is open
        if (isModalOpen) {
            return;
        }

        switch(e.key) {
            case 'j':
                navigateNext();
                break;
            case 'k':
                navigatePrevious();
                break;
            case 'l':
            case 'Enter':
                expandSelected();
                break;
            case 'h':
                collapseSelected();
                break;
            case 'o':
                openSelectedLink();
                break;
            case 'g':
                // Check for 'gg' (two g's quickly)
                if (e.timeStamp - window.lastGPress < 500) {
                    jumpToTop();
                }
                window.lastGPress = e.timeStamp;
                break;
            case 'G':
                jumpToBottom();
                break;
            case '/':
                focusSearch();
                break;
            case '?':
                showKeyboardHelp();
                break;
        }
    }

    // Navigate to next item
    function navigateNext() {
        if (!selectedElement) {
            selectFirstDigest();
            return;
        }

        let nextElement;

        if (navigationLevel === 'digest') {
            nextElement = selectedElement.nextElementSibling;
            if (nextElement && nextElement.classList.contains('digest-item')) {
                selectElement(nextElement, 'digest');
            }
        } else if (navigationLevel === 'article') {
            nextElement = selectedElement.nextElementSibling;
            if (nextElement && nextElement.classList.contains('article-item')) {
                selectElement(nextElement, 'article');
            } else {
                // No more articles, go to next digest
                const currentDigest = selectedElement.closest('.digest-item');
                const nextDigest = currentDigest?.nextElementSibling;
                if (nextDigest) {
                    selectElement(nextDigest, 'digest');
                }
            }
        }
    }

    // Navigate to previous item
    function navigatePrevious() {
        if (!selectedElement) return;

        let prevElement;

        if (navigationLevel === 'digest') {
            prevElement = selectedElement.previousElementSibling;
            if (prevElement && prevElement.classList.contains('digest-item')) {
                selectElement(prevElement, 'digest');
            }
        } else if (navigationLevel === 'article') {
            prevElement = selectedElement.previousElementSibling;
            if (prevElement && prevElement.classList.contains('article-item')) {
                selectElement(prevElement, 'article');
            } else {
                // At first article, go back to digest level
                const currentDigest = selectedElement.closest('.digest-item');
                if (currentDigest) {
                    selectElement(currentDigest, 'digest');
                }
            }
        }
    }

    // Expand selected digest
    function expandSelected() {
        if (!selectedElement || navigationLevel !== 'digest') return;

        const expandLink = selectedElement.querySelector('.expand-link');
        if (expandLink) {
            expandLink.click();  // Trigger HTMX request

            // After HTMX swap, focus first article
            document.body.addEventListener('htmx:afterSwap', function selectFirstArticle() {
                const firstArticle = selectedElement.querySelector('.article-item');
                if (firstArticle) {
                    selectElement(firstArticle, 'article');
                }
                document.body.removeEventListener('htmx:afterSwap', selectFirstArticle);
            });
        }
    }

    // Collapse selected digest
    function collapseSelected() {
        let digestElement;

        if (navigationLevel === 'digest') {
            digestElement = selectedElement;
        } else if (navigationLevel === 'article') {
            digestElement = selectedElement.closest('.digest-item');
        }

        if (digestElement) {
            const collapseLink = digestElement.querySelector('.collapse-link');
            if (collapseLink) {
                collapseLink.click();  // Trigger HTMX collapse
                selectElement(digestElement, 'digest');
            }
        }
    }

    // Open article link in new tab
    function openSelectedLink() {
        if (navigationLevel === 'article' && selectedElement) {
            const link = selectedElement.querySelector('.article-title a');
            if (link) {
                window.open(link.href, '_blank');
            }
        }
    }

    // Jump to top of page
    function jumpToTop() {
        window.scrollTo({ top: 0, behavior: 'smooth' });
        selectFirstDigest();
    }

    // Jump to bottom of page
    function jumpToBottom() {
        window.scrollTo({ top: document.body.scrollHeight, behavior: 'smooth' });
        const lastDigest = document.querySelector('.digest-item:last-child');
        if (lastDigest) {
            selectElement(lastDigest, 'digest');
        }
    }

    // Focus search input (future)
    function focusSearch() {
        const searchInput = document.querySelector('input[type="search"]');
        if (searchInput) {
            searchInput.focus();
        }
    }

    // Show keyboard help modal
    function showKeyboardHelp() {
        const modal = document.getElementById('keyboard-help-modal');
        if (modal) {
            modal.style.display = 'flex';
            // Trap focus within modal
            modal.querySelector('.modal-close').focus();
        }
    }

    // Close keyboard help modal (exposed globally for onclick handlers)
    window.closeKeyboardModal = function() {
        const modal = document.getElementById('keyboard-help-modal');
        if (modal) {
            modal.style.display = 'none';
        }
    };

    // Select an element (add .selected class)
    function selectElement(element, level) {
        // Remove previous selection
        if (selectedElement) {
            selectedElement.classList.remove('selected');
        }

        // Add new selection
        selectedElement = element;
        selectedElement.classList.add('selected');
        navigationLevel = level;

        // Scroll into view if needed
        element.scrollIntoView({
            behavior: 'smooth',
            block: 'nearest'
        });
    }

})();
```

### 7.4 Accessibility Considerations

**Screen Reader Support:**
```html
<!-- Add ARIA labels -->
<li class="digest-item"
    data-digest-id="{{ .ID }}"
    tabindex="0"
    role="article"
    aria-label="Digest: {{ .Title }}">
```

**Focus Trap Prevention:**
```javascript
// Allow Escape key to reset focus
case 'Escape':
    if (selectedElement) {
        selectedElement.classList.remove('selected');
        selectedElement = null;
    }
    break;
```

**Skip to Content:**
```html
<!-- Add to header -->
<a href="#main-content" class="skip-link">
    Skip to content
</a>
```

---

## 8. Backend Changes

### 8.1 Server-Side Markdown Rendering

**Dependency:** `github.com/gomarkdown/markdown`

```bash
go get github.com/gomarkdown/markdown
```

#### 8.1.1 Update `handleExpandDigest()`

**File:** `internal/server/digest_handlers_v2.go`

**Current:**
```go
viewData := struct {
    ID            string
    DigestSummary string  // Raw markdown
    ArticleGroups []EnrichedGroup
}{
    ID:            digest.ID,
    DigestSummary: digest.DigestSummary,  // ❌ Markdown text
    ArticleGroups: enrichedGroups,
}
```

**New:**
```go
import (
    "github.com/gomarkdown/markdown"
    "github.com/gomarkdown/markdown/html"
    "github.com/gomarkdown/markdown/parser"
)

func (s *Server) handleExpandDigest(w http.ResponseWriter, r *http.Request) {
    // ... existing digest fetch code ...

    // Configure markdown parser
    extensions := parser.CommonExtensions | parser.AutoHeadingIDs
    mdParser := parser.NewWithExtensions(extensions)

    htmlFlags := html.CommonFlags | html.HrefTargetBlank
    renderer := html.NewRenderer(html.RendererOptions{Flags: htmlFlags})

    // Convert digest summary to HTML
    summaryHTML := markdown.ToHTML(
        []byte(digest.DigestSummary),
        mdParser,
        renderer,
    )

    // Extract key moments from summary
    keyMoments := extractKeyMoments(digest.DigestSummary)

    // Build perspectives from article groups
    perspectives := make([]Perspective, 0, len(digest.ArticleGroups))
    for _, group := range digest.ArticleGroups {
        contentHTML := markdown.ToHTML(
            []byte(group.Summary),
            mdParser,
            renderer,
        )
        perspectives = append(perspectives, Perspective{
            Title:       group.Theme,
            ContentHTML: template.HTML(contentHTML),
        })
    }

    // Flatten all articles across groups
    allArticles := make([]ArticleWithSummary, 0)
    for _, group := range digest.ArticleGroups {
        for _, article := range group.Articles {
            // Find summary for this article
            var summary string
            for _, s := range digest.Summaries {
                for _, aid := range s.ArticleIDs {
                    if aid == article.ID {
                        summary = s.SummaryText
                        break
                    }
                }
            }

            allArticles = append(allArticles, ArticleWithSummary{
                ID:            article.ID,
                URL:           article.URL,
                Title:         article.Title,
                ContentType:   string(article.ContentType),
                DatePublished: article.DatePublished,
            })
        }
    }

    viewData := struct {
        ID            string
        SummaryHTML   template.HTML
        KeyMoments    []string
        Perspectives  []Perspective
        Articles      []ArticleWithSummary
    }{
        ID:           digest.ID,
        SummaryHTML:  template.HTML(summaryHTML),  // ✅ Pre-rendered HTML
        KeyMoments:   keyMoments,
        Perspectives: perspectives,
        Articles:     allArticles,
    }

    s.renderer.Render(w, "partials/digest-expanded.html", viewData)
}

// Helper: Extract key moments from markdown
func extractKeyMoments(markdown string) []string {
    lines := strings.Split(markdown, "\n")
    var moments []string

    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        // Match "**Highlight**" or "- Bullet" or "• Bullet"
        if strings.HasPrefix(trimmed, "**") && strings.HasSuffix(trimmed, "**") {
            // Extract text between **
            moment := strings.Trim(trimmed, "**")
            moments = append(moments, moment)
        } else if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "• ") {
            moment := strings.TrimPrefix(strings.TrimPrefix(trimmed, "- "), "• ")
            moments = append(moments, moment)
        }
    }

    // Limit to top 5 key moments
    if len(moments) > 5 {
        moments = moments[:5]
    }

    return moments
}

type Perspective struct {
    Title       string
    ContentHTML template.HTML
}
```

#### 8.1.2 Implement `handleDigestDetailPage()`

**File:** `internal/server/web_pages.go` (or create if doesn't exist)

```go
func (s *Server) handleDigestDetailPage(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    digestID := chi.URLParam(r, "id")

    // Fetch digest
    digest, err := s.db.Digests().Get(ctx, digestID)
    if err != nil {
        http.Error(w, "Digest not found", http.StatusNotFound)
        return
    }

    // Reuse same logic as handleExpandDigest
    // Convert markdown to HTML
    extensions := parser.CommonExtensions | parser.AutoHeadingIDs
    mdParser := parser.NewWithExtensions(extensions)
    htmlFlags := html.CommonFlags | html.HrefTargetBlank
    renderer := html.NewRenderer(html.RendererOptions{Flags: htmlFlags})

    summaryHTML := markdown.ToHTML(
        []byte(digest.DigestSummary),
        mdParser,
        renderer,
    )

    keyMoments := extractKeyMoments(digest.DigestSummary)

    perspectives := make([]Perspective, 0, len(digest.ArticleGroups))
    for _, group := range digest.ArticleGroups {
        contentHTML := markdown.ToHTML(
            []byte(group.Summary),
            mdParser,
            renderer,
        )
        perspectives = append(perspectives, Perspective{
            Title:       group.Theme,
            ContentHTML: template.HTML(contentHTML),
        })
    }

    allArticles := make([]ArticleWithSummary, 0)
    for _, group := range digest.ArticleGroups {
        for _, article := range group.Articles {
            allArticles = append(allArticles, ArticleWithSummary{
                ID:            article.ID,
                URL:           article.URL,
                Title:         article.Title,
                ContentType:   string(article.ContentType),
                DatePublished: article.DatePublished,
            })
        }
    }

    viewData := struct {
        Title         string
        ArticleCount  int
        DateGenerated time.Time
        SummaryHTML   template.HTML
        KeyMoments    []string
        Perspectives  []Perspective
        Articles      []ArticleWithSummary
    }{
        Title:         digest.Metadata.Title,
        ArticleCount:  digest.Metadata.ArticleCount,
        DateGenerated: digest.Metadata.DateGenerated,
        SummaryHTML:   template.HTML(summaryHTML),
        KeyMoments:    keyMoments,
        Perspectives:  perspectives,
        Articles:      allArticles,
    }

    s.renderer.Render(w, "pages/digest-detail.html", viewData)
}
```

### 8.2 Update `handleHomePage()`

**File:** `internal/server/home_handler.go`

**Current:** Passes all digests, no limit

**New:** Limit to top 5 most recent

```go
func (s *Server) handleHomePage(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    themeID := r.URL.Query().Get("theme")
    if themeID == "all" {
        themeID = ""
    }

    // Check if HTMX request
    if isHTMXRequest(r) {
        s.handleHomePagePartial(w, r, ctx, themeID)
        return
    }

    data, err := s.getHomePageData(ctx, themeID)
    if err != nil {
        slog.Error("Failed to get homepage data", "error", err)
        http.Error(w, "Failed to load homepage", http.StatusInternalServerError)
        return
    }

    s.renderer.Render(w, "layouts/base.html", data)
}

func (s *Server) getHomePageData(ctx context.Context, activeThemeID string) (*HomePageData, error) {
    // Get enabled themes
    themes, err := s.db.Themes().ListEnabled(ctx)
    if err != nil {
        return nil, err
    }

    // Get top 5 most recent digests (filtered by theme if specified)
    digests, err := s.getDigestsForTheme(ctx, activeThemeID, 5)  // ✅ Limit to 5
    if err != nil {
        return nil, err
    }

    return &HomePageData{
        Themes:      themes,
        Digests:     digests,
        ActiveTheme: activeThemeID,
    }, nil
}

func (s *Server) getDigestsForTheme(ctx context.Context, themeID string, limit int) ([]DigestSummaryView, error) {
    var digests []core.Digest
    var err error

    if themeID == "" {
        // Get all digests (most recent first)
        digests, err = s.db.Digests().List(ctx, persistence.ListOptions{
            Limit:  limit,  // ✅ Top 5
            Offset: 0,
        })
    } else {
        // Get digests for theme
        allDigests, err := s.db.Digests().List(ctx, persistence.ListOptions{
            Limit:  100,
            Offset: 0,
        })
        if err != nil {
            return nil, err
        }

        // Filter and limit
        for _, d := range allDigests {
            hasTheme := false
            for _, group := range d.ArticleGroups {
                if group.Theme == themeID || containsThemeByID(group, themeID) {
                    hasTheme = true
                    break
                }
            }
            if hasTheme {
                digests = append(digests, d)
                if len(digests) >= limit {  // ✅ Stop at limit
                    break
                }
            }
        }
    }

    if err != nil {
        return nil, err
    }

    // Convert to view models
    result := make([]DigestSummaryView, 0, len(digests))
    for _, d := range digests {
        result = append(result, DigestSummaryView{
            ID:            d.ID,
            Title:         d.Metadata.Title,
            ArticleCount:  d.Metadata.ArticleCount,
            DateGenerated: d.Metadata.DateGenerated,
        })
    }

    return result, nil
}
```

### 8.3 Citation Links Implementation (Future Enhancement)

**Goal:** Add inline citation links [1], [2], [3] in digest summaries linking to source articles for credibility.

**Backend Changes Required:**

1. **Modify Digest Generation Pipeline** - Track which articles support which summary claims
2. **Update `core.Digest` structure:**
```go
type Digest struct {
    // ... existing fields
    Citations map[string][]int  // Summary claim ID → Article indices
}
```

3. **Render Citations in Summary:**
```go
func renderSummaryWithCitations(summary string, citations map[string][]int) string {
    // Replace citation placeholders with links
    // Input:  "AI acceleration is democratizing [cite:1,2]"
    // Output: "AI acceleration is democratizing [1] [2]"
    // Where [1] and [2] are <a href="#article-1"> links
}
```

**Template Changes:**
```html
<div class="prose">
    {{ .SummaryHTMLWithCitations }}
    <!-- Renders: AI acceleration [<a href="#article-1">1</a>]... -->
</div>

<!-- Articles section uses id="article-N" for anchor targets -->
<li id="article-{{ .Index }}" class="article-item">...</li>
```

**CSS** (already added in section 6.3):
```css
.prose a[href^="#article-"] {
    color: var(--nord8);
    text-decoration: none;
    font-weight: 600;
}
```

**Note:** This is a significant backend enhancement requiring LLM prompt changes to generate citations during digest summarization. Consider implementing in Phase 2 after core UI is complete.

---

## 9. Frontend Changes

### 9.1 Remove Dependencies

#### Update `layouts/base.html`

**Current:**
```html
<!-- daisyUI -->
<link href="https://cdn.jsdelivr.net/npm/daisyui@4.4.24/dist/full.min.css" rel="stylesheet" />
<!-- Tailwind -->
<script src="https://cdn.tailwindcss.com"></script>
<!-- Marked.js -->
<script src="https://cdn.jsdelivr.net/npm/marked@11.1.0/marked.min.js"></script>
<!-- HTMX -->
<script src="https://unpkg.com/htmx.org@1.9.10"></script>
```

**New:**
```html
<!-- Minimal Custom CSS (Nord-based) -->
<link rel="stylesheet" href="/static/css/nord-minimal.css">
<!-- HTMX (keep for partial updates) -->
<script src="https://unpkg.com/htmx.org@1.9.10"></script>
<!-- Keyboard Navigation -->
<script src="/static/js/keyboard-nav.js"></script>
```

### 9.2 New CSS File

**File:** `web/static/css/nord-minimal.css`

```css
/* ========================================
   Briefly - Nord Minimal Theme
   ======================================== */

/* CSS Variables - Nord Palette */
:root {
    /* Polar Night */
    --nord0:  #2E3440;
    --nord1:  #3B4252;
    --nord2:  #434C5E;
    --nord3:  #4C566A;

    /* Snow Storm */
    --nord4:  #D8DEE9;
    --nord5:  #E5E9F0;
    --nord6:  #ECEFF4;

    /* Frost */
    --nord7:  #8FBCBB;
    --nord8:  #88C0D0;
    --nord9:  #81A1C1;
    --nord10: #5E81AC;

    /* Aurora */
    --nord11: #BF616A;  /* Red */
    --nord12: #D08770;  /* Orange */
    --nord13: #EBCB8B;  /* Yellow */
    --nord14: #A3BE8C;  /* Green */
    --nord15: #B48EAD;  /* Purple */
}

/* Reset & Base */
* {
    box-sizing: border-box;
    margin: 0;
    padding: 0;
}

html {
    font-size: 16px;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto',
                 'Oxygen', 'Ubuntu', 'Helvetica Neue', sans-serif;
    background-color: var(--nord0);
    color: var(--nord5);
    line-height: 1.6;
}

/* Container */
.container {
    max-width: 800px;
    margin: 0 auto;
    padding: 16px;
}

/* Typography */
h1 {
    font-size: 24px;
    font-weight: 700;
    color: var(--nord6);
    margin-bottom: 8px;
}

h2 {
    font-size: 18px;
    font-weight: 600;
    color: var(--nord6);
    margin-bottom: 12px;
}

h3 {
    font-size: 14px;
    font-weight: 600;
    color: var(--nord6);
    margin-bottom: 8px;
}

p {
    margin-bottom: 12px;
}

/* Links */
a {
    color: var(--nord8);
    text-decoration: none;
    transition: color 0.2s;
}

a:hover {
    color: var(--nord9);
    text-decoration: underline;
}

a:active {
    color: var(--nord10);
}

a:visited {
    color: var(--nord7);
}

/* Header */
header.site-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px;
    border-bottom: 1px solid var(--nord1);
    margin-bottom: 16px;
}

header.site-header .logo {
    font-size: 18px;
    font-weight: 700;
    color: var(--nord6);
}

header.site-header nav a {
    margin-left: 16px;
    font-size: 14px;
}

/* Theme Filter */
.theme-filter {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    padding: 12px 0;
    border-bottom: 1px solid var(--nord1);
    margin-bottom: 16px;
}

.theme-filter a {
    color: var(--nord4);
    font-size: 13px;
}

.theme-filter a:hover {
    color: var(--nord6);
}

.theme-filter a.active {
    color: var(--nord8);
    font-weight: 600;
}

/* Digest List */
.digest-list {
    list-style: decimal;
    padding-left: 24px;
    margin: 16px 0;
}

.digest-item {
    padding: 12px 0;
    border-bottom: 1px solid var(--nord1);
    transition: background-color 0.2s, border-left 0.2s;
}

.digest-item:hover {
    background-color: rgba(59, 66, 82, 0.5);
}

.digest-item.selected {
    background-color: var(--nord1);
    border-left: 3px solid var(--nord8);
    padding-left: 5px;
}

.digest-title a {
    font-size: 16px;
    font-weight: 600;
    color: var(--nord6);
}

.digest-meta {
    font-size: 12px;
    color: var(--nord4);
    margin-top: 4px;
}

.digest-meta span::after {
    content: " · ";
    margin: 0 4px;
}

.digest-meta span:last-of-type::after {
    content: "";
}

/* Expanded Digest */
.digest-expanded {
    margin-top: 16px;
    padding-left: 24px;
    border-left: 2px solid var(--nord2);
}

.section {
    margin-bottom: 24px;
}

.section-title {
    font-size: 14px;
    font-weight: 600;
    color: var(--nord6);
    margin-bottom: 8px;
    border-bottom: 1px solid var(--nord2);
    padding-bottom: 4px;
}

/* Prose (Rendered Markdown) */
.prose {
    font-size: 14px;
    line-height: 1.6;
    color: var(--nord5);
}

.prose p {
    margin-bottom: 12px;
}

.prose strong {
    color: var(--nord6);
    font-weight: 600;
}

.prose em {
    color: var(--nord4);
    font-style: italic;
}

.prose ul,
.prose ol {
    margin-left: 20px;
    margin-bottom: 12px;
}

.prose li {
    margin-bottom: 4px;
}

.prose a {
    color: var(--nord8);
    text-decoration: underline;
}

.prose code {
    background-color: var(--nord1);
    padding: 2px 4px;
    border-radius: 3px;
    font-family: 'Monaco', 'Courier New', monospace;
    font-size: 12px;
    color: var(--nord13);
}

.prose pre {
    background-color: var(--nord1);
    padding: 12px;
    border-radius: 4px;
    overflow-x: auto;
    margin-bottom: 12px;
}

.prose pre code {
    background: none;
    padding: 0;
}

/* Key Moments */
.key-moments {
    list-style: disc;
    padding-left: 20px;
}

.key-moments li {
    margin-bottom: 8px;
    color: var(--nord5);
}

/* Perspectives */
.perspective {
    margin-bottom: 16px;
}

.perspective h4 {
    font-size: 13px;
    font-weight: 600;
    color: var(--nord8);
    margin-bottom: 4px;
}

/* Article List */
.article-list {
    list-style: decimal;
    padding-left: 24px;
}

.article-item {
    margin-bottom: 12px;
    transition: background-color 0.2s, border-left 0.2s;
}

.article-item:hover {
    background-color: rgba(67, 76, 94, 0.3);
}

.article-item.selected {
    background-color: var(--nord2);
    border-left: 3px solid var(--nord9);
    padding-left: 5px;
}

.article-title {
    font-size: 14px;
}

.article-title a {
    color: var(--nord6);
}

.article-title a:visited {
    color: var(--nord7);
}

.article-domain {
    font-size: 12px;
    color: var(--nord4);
    margin-left: 4px;
}

.article-meta {
    font-size: 11px;
    color: var(--nord4);
    margin-top: 2px;
}

.content-badge {
    background-color: var(--nord2);
    padding: 2px 6px;
    border-radius: 3px;
    font-size: 10px;
    color: var(--nord5);
    margin-left: 8px;
}

/* Keyboard Hint */
.keyboard-hint {
    font-size: 12px;
    color: var(--nord4);
    text-align: center;
    margin-top: 32px;
    padding: 8px;
    border-top: 1px solid var(--nord1);
}

/* Footer */
footer {
    margin-top: 48px;
    padding: 16px 0;
    border-top: 1px solid var(--nord1);
    text-align: center;
    font-size: 12px;
    color: var(--nord4);
}

/* Breadcrumb */
.breadcrumb {
    margin-bottom: 16px;
}

.breadcrumb a {
    font-size: 13px;
}

/* Skip Link (Accessibility) */
.skip-link {
    position: absolute;
    top: -40px;
    left: 0;
    background: var(--nord8);
    color: var(--nord0);
    padding: 8px;
    z-index: 100;
}

.skip-link:focus {
    top: 0;
}

/* Responsive Adjustments */
@media (max-width: 600px) {
    .container {
        padding: 12px;
    }

    .digest-title a {
        font-size: 14px;
    }

    .digest-meta {
        font-size: 11px;
    }

    .theme-filter {
        gap: 8px;
    }

    .digest-expanded {
        padding-left: 12px;
    }
}

/* ========================================
   Keyboard Shortcuts Modal
   See Section 7.3 for complete modal CSS
   ======================================== */

/* ========================================
   Footer with Keyboard Hints
   See Section 7.3 for complete footer CSS
   ======================================== */
```

**Note:** The complete `nord-minimal.css` file should include:
1. All styles shown above (~400 lines)
2. Modal styles from Section 7.3 (~150 lines)
3. Footer with keyboard hints from Section 7.3 (~50 lines)

**Total:** ~600 lines of CSS

### 9.3 JavaScript File

**File:** `web/static/js/keyboard-nav.js`

(Already included in Section 7.4 above)

---

## 10. Migration Plan

### 10.1 Phase Breakdown

**Total Effort:** ~35 hours (4-5 days)

| Phase | Tasks | Hours | Priority |
|-------|-------|-------|----------|
| **Phase 1: Design Review** | This document | 0 | P0 |
| **Phase 2: Backend SSR** | Add markdown lib, update handlers | 5 | P0 |
| **Phase 3: CSS Foundation** | Create nord-minimal.css | 5 | P0 |
| **Phase 4: Template Rewrites** | Rewrite all 8 templates | 12 | P0 |
| **Phase 5: Keyboard Nav** | Implement keyboard-nav.js | 8 | P1 |
| **Phase 6: Testing** | Cross-browser, mobile, a11y | 5 | P1 |

### 10.2 Phase 2: Backend SSR (5 hours)

**Tasks:**
1. Add `github.com/gomarkdown/markdown` to `go.mod`
2. Create `extractKeyMoments()` helper function
3. Update `handleExpandDigest()` with SSR markdown
4. Implement `handleDigestDetailPage()` (new)
5. Update `handleHomePage()` to limit to 5 digests
6. Add new view model types (Perspective, etc.)
7. Test with existing templates

**Validation:**
```bash
# Build should succeed
go build -o briefly ./cmd/briefly

# Test server starts
./briefly serve

# Test existing homepage (should still work)
curl http://localhost:8080/

# Test digest detail (new endpoint)
curl http://localhost:8080/digests/3dcfd638-1ee6-4558-93dd-d5d3e0bdf642
```

### 10.3 Phase 3: CSS Foundation (5 hours)

**Tasks:**
1. Create `web/static/css/nord-minimal.css`
2. Add Nord color variables
3. Define typography styles
4. Add digest/article item styles
5. Add focus indicators for keyboard nav
6. Test responsive breakpoints
7. Update `base.html` to remove daisyUI, load new CSS

**Validation:**
```bash
# Serve static files (ensure route exists)
curl http://localhost:8080/static/css/nord-minimal.css

# Check visual regression manually
open http://localhost:8080/
```

### 10.4 Phase 4: Template Rewrites (12 hours)

**Order of Execution:**

1. **Update `layouts/base.html`** (1 hour)
   - Remove daisyUI CDN
   - Remove marked.js
   - Add nord-minimal.css
   - Add keyboard-nav.js placeholder

2. **Rewrite `partials/theme-filter.html`** (1 hour)
   - Replace chips with text links
   - Keep HTMX attributes

3. **Rewrite `partials/digest-list.html`** (1 hour)
   - Change to `<ol>` list
   - Update loop structure

4. **Rewrite `partials/digest-item.html`** (2 hours)
   - Collapse to single-line entry
   - Add expand/collapse links
   - Add data attributes for keyboard nav

5. **Rewrite `partials/digest-expanded.html`** (2 hours)
   - Add section structure (Summary, Key Moments, etc.)
   - Use `{{ .SummaryHTML }}` instead of markdown text
   - Add perspectives section

6. **Rewrite `partials/article-item.html`** (1 hour)
   - Compact single-line format
   - Add domain extraction
   - Add data attributes

7. **Create `pages/digest-detail.html`** (2 hours)
   - Full page layout
   - Breadcrumb navigation
   - Reuse expanded structure

8. **Update `pages/home.html`** (1 hour)
   - Remove client-side markdown JS
   - Clean up stats header
   - Add keyboard hint

9. **Update `partials/header.html`** (1 hour)
   - Simplify to text-only nav
   - Remove theme toggle (keep dark mode)

**Validation:**
```bash
# Test each template incrementally
go build && ./briefly serve

# Check homepage renders
curl http://localhost:8080/ | grep "digest-list"

# Check digest detail
curl http://localhost:8080/digests/{id} | grep "section-title"

# Check HTMX expand
curl http://localhost:8080/api/digests/{id}/expand | grep "digest-expanded"
```

### 10.5 Phase 5: Keyboard Navigation (8 hours)

**Tasks:**
1. Create `web/static/js/keyboard-nav.js`
2. Implement selection state management
3. Implement j/k navigation (next/previous)
4. Implement l/Enter expansion (trigger HTMX)
5. Implement h collapse
6. Implement o open link
7. Implement gg/G jump to top/bottom
8. Add focus indicators (CSS classes)
9. Test accessibility (screen readers)

**Validation:**
```bash
# Manual testing
open http://localhost:8080/

# Test each shortcut:
# - j/k: Move selection (visual feedback?)
# - l/Enter: Expand digest (HTMX fires?)
# - h: Collapse
# - o: Open article link in new tab
# - gg: Jump to top
```

### 10.6 Phase 6: Testing & Polish (5 hours)

**Cross-Browser Testing:**
- Chrome (latest)
- Firefox (latest)
- Safari (latest)
- Mobile Safari (iOS)
- Mobile Chrome (Android)

**Accessibility Testing:**
- VoiceOver (macOS)
- NVDA (Windows)
- Keyboard-only navigation
- Color contrast (WCAG AA)

**Performance Testing:**
- Lighthouse score
- Page load time
- TTFB (time to first byte)
- SSR markdown render time

**Edge Cases:**
- Empty digest list (no digests)
- Digest with 0 articles
- Very long article titles
- Markdown with code blocks, images
- Browser back button after HTMX navigation

---

## 11. Accessibility

### 11.1 WCAG 2.1 Compliance

**Level AA Requirements:**

| Criterion | Implementation | Status |
|-----------|----------------|--------|
| **1.4.3 Contrast** | Nord colors meet 4.5:1 for text | ✅ |
| **2.1.1 Keyboard** | All interactions keyboard-accessible | ✅ |
| **2.4.7 Focus Visible** | Clear focus indicators (left border + bg) | ✅ |
| **3.2.3 Consistent Navigation** | Header/footer same across pages | ✅ |
| **4.1.2 Name, Role, Value** | Semantic HTML, ARIA labels | ✅ |

### 11.2 Semantic HTML

```html
<!-- Good semantic structure -->
<main id="main-content">
    <nav aria-label="Theme filter">...</nav>
    <ol aria-label="Digest list">
        <li role="article">...</li>
    </ol>
</main>
```

### 11.3 ARIA Labels

```html
<li class="digest-item"
    role="article"
    aria-label="Weekly Tech Digest - Nov 3, 2025. 16 articles."
    tabindex="0">
```

### 11.4 Screen Reader Announcements

```javascript
// Announce selection change
function selectElement(element, level) {
    selectedElement = element;

    // Announce to screen readers
    const announcement = document.createElement('div');
    announcement.setAttribute('role', 'status');
    announcement.setAttribute('aria-live', 'polite');
    announcement.className = 'sr-only';
    announcement.textContent = `Selected ${element.querySelector('.digest-title').textContent}`;
    document.body.appendChild(announcement);

    setTimeout(() => document.body.removeChild(announcement), 1000);
}
```

### 11.5 Skip Links

```html
<!-- Add to top of base.html -->
<a href="#main-content" class="skip-link">
    Skip to main content
</a>
```

```css
.skip-link {
    position: absolute;
    top: -40px;
    left: 0;
    background: var(--nord8);
    color: var(--nord0);
    padding: 8px;
    z-index: 100;
}

.skip-link:focus {
    top: 0;
}

.sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border-width: 0;
}
```

---

## 12. Appendices

### Appendix A: Nord Color Reference

**Visual Palette:**

```
┌─────────────────┬─────────────────┬─────────────────┬─────────────────┐
│ nord0           │ nord1           │ nord2           │ nord3           │
│ #2E3440         │ #3B4252         │ #434C5E         │ #4C566A         │
│ Background      │ Hover BG        │ Selection BG    │ Secondary Text  │
└─────────────────┴─────────────────┴─────────────────┴─────────────────┘

┌─────────────────┬─────────────────┬─────────────────┐
│ nord4           │ nord5           │ nord6           │
│ #D8DEE9         │ #E5E9F0         │ #ECEFF4         │
│ Secondary Text  │ Body Text       │ Headings        │
└─────────────────┴─────────────────┴─────────────────┘

┌─────────────────┬─────────────────┬─────────────────┬─────────────────┐
│ nord7           │ nord8           │ nord9           │ nord10          │
│ #8FBCBB         │ #88C0D0         │ #81A1C1         │ #5E81AC         │
│ Teal/Visited    │ Cyan/Links      │ Blue/Hover      │ Dark Blue       │
└─────────────────┴─────────────────┴─────────────────┴─────────────────┘

┌─────────────────┬─────────────────┬─────────────────┬─────────────────┬─────────────────┐
│ nord11          │ nord12          │ nord13          │ nord14          │ nord15          │
│ #BF616A         │ #D08770         │ #EBCB8B         │ #A3BE8C         │ #B48EAD         │
│ Red/Error       │ Orange/Warning  │ Yellow/Highlight│ Green/Success   │ Purple/Special  │
└─────────────────┴─────────────────┴─────────────────┴─────────────────┴─────────────────┘
```

### Appendix B: Keyboard Shortcuts

**Homepage:**

| Key | Action | Description |
|-----|--------|-------------|
| `j` | Next | Move selection down |
| `k` | Previous | Move selection up |
| `l` / `Enter` | View | Navigate to digest detail page |
| `o` | Open | Same as Enter (navigate to digest) |
| `gg` | Jump Top | Scroll to top of page |
| `G` | Jump Bottom | Scroll to bottom |
| `/` | Search | Focus search input (future) |
| `?` | Help | **Show keyboard shortcuts modal** |
| `Esc` | Close | Close modal or clear selection |

**Digest Detail Page:**

| Key | Action | Description |
|-----|--------|-------------|
| `s` | Summary | Jump to Summary section |
| `k` | Key Moments | Jump to Key Moments section |
| `p` | Perspectives | Jump to Perspectives section |
| `a` | Articles | Jump to Articles section |
| `y` | Copy | Copy summary markdown to clipboard |
| `j` / `k` | Navigate | Move between articles (in Articles section) |
| `o` / `Enter` | Open | Open selected article in new tab |
| `h` | Back | Return to homepage |
| `gg` | Jump Top | Scroll to top of page |
| `G` | Jump Bottom | Scroll to bottom |
| `?` | Help | **Show keyboard shortcuts modal** |
| `Esc` | Close | Close modal or clear selection |

### Appendix C: File Changes Summary

**Modified Files (9):**
1. `web/templates/layouts/base.html` - Remove daisyUI/marked.js, add nord-minimal.css + keyboard-nav.js, add keyboard shortcuts modal
2. `web/templates/pages/home.html` - Add digest summary preview, remove client-side rendering
3. `web/templates/partials/header.html` - Simplify navigation
4. `web/templates/partials/footer.html` - Remove API link, add keyboard hints (j/k navigate, ? shortcuts)
5. `web/templates/partials/theme-tabs.html` → `theme-filter.html` - Plain text links (no HTMX)
6. `web/templates/partials/digest-list.html` - `<ol>` structure
7. `web/templates/partials/digest-card.html` → `digest-item.html` - Multi-line with summary preview, direct navigation (no expand)
8. `web/templates/partials/article-card.html` → `article-item.html` - Compact format with article navigation
9. `internal/server/home_handler.go` - Limit to top 5, add summary preview extraction

**New Files (4):**
10. `web/templates/pages/digest-detail.html` - Full digest page (replaces expand/collapse)
11. `web/static/css/nord-minimal.css` - Complete Nord theme CSS with modal & footer hints (~500 lines, rem/em units)
12. `web/static/js/keyboard-nav.js` - Keyboard navigation with section jumping + modal toggle
13. `internal/server/digest_detail_handler.go` - Digest detail page handler with SSR markdown

**Deleted Files (2):**
14. `web/static/css/custom.css` - Replaced by nord-minimal.css
15. `web/templates/partials/digest-expanded.html` - Replaced by digest-detail.html page

### Appendix D: Template Helper Functions

**Required Template Functions:**

```go
// In internal/server/templates.go (or create if doesn't exist)

// formatTimeAgo returns "2h ago", "3d ago", etc.
func formatTimeAgo(t time.Time) string {
    duration := time.Since(t)

    hours := int(duration.Hours())
    if hours < 1 {
        return fmt.Sprintf("%dm ago", int(duration.Minutes()))
    } else if hours < 24 {
        return fmt.Sprintf("%dh ago", hours)
    } else {
        days := hours / 24
        if days < 7 {
            return fmt.Sprintf("%dd ago", days)
        } else if days < 30 {
            weeks := days / 7
            return fmt.Sprintf("%dw ago", weeks)
        } else {
            months := days / 30
            return fmt.Sprintf("%dmo ago", months)
        }
    }
}

// extractDomain returns "example.com" from "https://example.com/path"
func extractDomain(rawURL string) string {
    u, err := url.Parse(rawURL)
    if err != nil {
        return rawURL
    }

    domain := u.Host
    // Remove "www." prefix
    domain = strings.TrimPrefix(domain, "www.")
    return domain
}

// formatDate returns "Nov 3, 2025"
func formatDate(t time.Time) string {
    if t.IsZero() {
        return ""
    }
    return t.Format("Jan 2, 2006")
}

// Add to funcMap in TemplateRenderer
funcMap := template.FuncMap{
    "formatTimeAgo":  formatTimeAgo,
    "extractDomain":  extractDomain,
    "formatDate":     formatDate,
}
```

### Appendix E: Mobile Considerations

**Touch Targets:**
- Minimum 44×44px (iOS guideline)
- Digest items: 48px height minimum
- Link padding: 8px vertical

**Responsive Breakpoints:**
```css
/* Small phones */
@media (max-width: 375px) {
    .container { padding: 8px; }
    .digest-title a { font-size: 14px; }
}

/* Medium phones */
@media (max-width: 600px) {
    .container { padding: 12px; }
    .theme-filter { gap: 8px; }
}

/* Tablets */
@media (min-width: 768px) {
    .container { padding: 24px; }
}
```

**Touch Gestures (Optional Future Enhancement):**
- Swipe left on digest → Collapse
- Swipe right on digest → Expand
- Swipe up/down → Scroll (native)

---

## 13. Conclusion

This design document outlines a comprehensive refactor from a **card-based, mouse-centric UI** to a **text-list, keyboard-first interface** inspired by Hacker News. The new design prioritizes:

- **Minimalism**: Nord color palette, typography-driven hierarchy
- **Performance**: Server-side markdown rendering, minimal CSS/JS
- **Accessibility**: Keyboard navigation, semantic HTML, WCAG 2.1 AA compliance
- **Focus**: Top 5 most recent digests, signal over noise

**Key Architectural Decisions:**
1. **Preserve HTMX**: Partial updates still work, just triggered by keyboard
2. **SSR Markdown**: Use `gomarkdown/markdown` instead of client-side `marked.js`
3. **No DB Schema Changes**: Extract Key Moments/Perspectives from existing data
4. **Nord Theme**: Official palette for consistent, minimal color use
5. **Vim-Style Keyboard**: j/k/l/h navigation for power users

**Estimated Effort:** 35 hours (~5 days of focused development)

---

## Implementation Summary

**Status:** ✅ **COMPLETE** (November 5, 2025)

**Implementation Phases Completed:**

### Phase 1: Backend Foundation ✅
- Added `gomarkdown/markdown` dependency for server-side rendering
- Created `internal/server/markdown_helpers.go` with SSR functions:
  - `renderMarkdown()` - Converts markdown to safe HTML
  - `extractKeyMoments()` - Extracts bullet points from summary
  - `truncateSummary()` - Creates preview text (~100 chars)
- Updated `internal/server/templates.go` with `formatTimeAgo` helper
- Rewrote `handleDigestDetailPage()` in `web_pages.go` for full SSR
- Updated `home_handler.go` to limit 5 digests and add preview field

### Phase 2: Nord Minimal CSS ✅
- Created `web/static/css/nord-minimal.css` (624 lines)
- Implemented full Nord color palette (16 official colors)
- Responsive typography: 16px (desktop) → 15px (tablet) → 14px (mobile)
- Text-focused component styles (digest-item, article-item, theme-filter)
- Keyboard navigation focus states (blue outline, grey background)
- Modal system for keyboard shortcuts

### Phase 3: Keyboard Navigation ✅
- Created `web/static/js/keyboard-nav.js` (381 lines)
- Homepage navigation: j/k (up/down), Enter/l (open), gg/G (top/bottom)
- Digest detail navigation: s/k/p/a (sections), y (copy), h (home), o (open)
- Keyboard shortcuts modal: ? (open), Esc (close)
- Focus management with visual indicators
- Clipboard API integration for summary copying

### Phase 4: Template Refactoring ✅
**Layouts:**
- `layouts/base.html` - Removed daisyUI/Tailwind, added Nord CSS + keyboard modal

**Pages:**
- `pages/home.html` - Simplified to theme filter + digest list
- `pages/digest-detail.html` - NEW: Full digest view with sections

**Partials:**
- `partials/header.html` - Simple text-based navigation
- `partials/footer.html` - Added keyboard hints (j/k, ?)
- `partials/theme-filter.html` - NEW: Text-based theme links
- `partials/digest-list.html` - Semantic `<ol>` wrapper
- `partials/digest-item.html` - NEW: Multi-line digest entry
- `partials/article-item.html` - NEW: Simple article link

**Deleted Old Templates:**
- `digest-card.html` (replaced by digest-item.html)
- `theme-tabs.html` (replaced by theme-filter.html)
- `article-card.html` (replaced by article-item.html)
- `digest-expanded.html` (replaced by digest-detail page)

### Phase 5: Testing Documentation ✅
- Created `docs/frontend-testing-checklist.md` with comprehensive test plan
- Manual testing checklist for homepage, digest detail, keyboard nav
- Responsive design tests (desktop/tablet/mobile)
- Cross-browser compatibility tests (Chrome/Firefox/Safari)
- Accessibility and performance checks

### Phase 6: Cleanup & Documentation ✅
- Build verification passed: `go build -o briefly ./cmd/briefly`
- All unit tests passing: `go test ./...`
- Documentation updated with implementation notes

**Files Created:** 7 new files
**Files Modified:** 11 existing files
**Files Deleted:** 4 obsolete templates
**Lines of Code:** ~1,500 new lines (CSS + JS + templates)

**Implementation Time:** ~6 hours (actual) vs 35 hours (estimated)
- Faster due to well-structured existing codebase
- Clear design document prevented scope creep
- No major architectural blockers

**Testing Status:**
- ✅ Compilation successful
- ✅ Unit tests passing
- ⏳ Manual browser testing required (see checklist)
- ⏳ Cross-browser verification pending

**Known Limitations:**
- Citation links [1], [2], [3] designed but not implemented (future)
- Banner images not implemented (existing limitation)
- PostHog analytics events need user verification

**Next Steps for User:**
1. Run server: `./briefly server`
2. Test homepage: `http://localhost:8080/`
3. Follow testing checklist: `docs/frontend-testing-checklist.md`
4. Report any issues or refinements needed
5. Consider deploying to production if tests pass

---

**Document Maintainer:** Development Team
**Last Updated:** November 5, 2025
**Status:** ✅ **IMPLEMENTED & READY FOR TESTING**
