# Briefly Documentation

Welcome to Briefly's documentation. This directory contains all design documents, implementation plans, and research materials organized for clarity and longevity.

## 📁 Documentation Structure

```
docs/
├── README.md                           # This file - documentation index
│
├── PRODUCT.md                          # 📋 Product vision, goals, metrics
├── ARCHITECTURE.md                     # 🏗️ System architecture & technical design
│
├── digest-pipeline-v2.md               # ⭐ Current pipeline design (many-digests architecture)
├── migration-plan.md                   # 📋 Migration guide (single → many digests)
│
├── inspirations/                       # 💡 Product research & analysis
│   ├── README.md                       # Index of inspiration sources
│   └── KAGI_NEWS.md                    # Kagi News analysis
│
├── executions/                         # ⚡ Implementation plans (dated)
│   ├── README.md                       # Execution tracking guide
│   └── 2025-10-31.md                   # Current implementation plan
│
└── archive/                            # 📦 Historical design documents
    ├── digest-pipeline-design-v1-archived.md   # OLD: Python-focused design (deprecated)
    ├── DESIGN_NEWS_DIGEST_WEBSITE_V2.1.md
    ├── DESIGN_NEWS_DIGEST_WEBSITE_V2.md
    └── ...
```

---

## 📋 Core Documents

### [PRODUCT.md](PRODUCT.md)
**Purpose:** Product vision, goals, and success metrics
**Status:** Living document (versioned)
**Update frequency:** When product direction changes

**Contains:**
- What we're building (executive summary)
- Vision & goals (product, technical, learning)
- Core capabilities and features
- User flows (admin and public)
- Success metrics (technical, product, learning)
- Portfolio talking points for job search

**When to read:**
- Understanding the "what" and "why" of Briefly
- Writing job applications or portfolio descriptions
- Making product decisions (features, scope, priorities)
- Stakeholder communication

**When to update:**
- Adding major features
- Changing target metrics
- Pivoting product direction
- Quarterly review of goals

---

### [ARCHITECTURE.md](ARCHITECTURE.md)
**Purpose:** System architecture and technical design
**Status:** Living document (always up-to-date)
**Update frequency:** As implementation changes

**Contains:**
- Current state analysis (what exists, what's missing)
- Architecture overview (system diagram, tech stack)
- Component design (detailed implementation)
- Data model (PostgreSQL schema with pgvector)
- API design (REST endpoints + CLI commands)
- Multi-agent architecture (Go concurrency patterns)
- Observability (LangFuse integration)
- Analytics (PostHog integration)
- RAG implementation (pgvector + retrieval)
- Technical decisions (ADRs with rationale)
- Deployment architecture

**When to read:**
- Onboarding new engineers
- Understanding how the system works
- Making technical decisions (library choices, patterns)
- Debugging or troubleshooting

**When to update:**
- After implementing major components
- When architecture patterns change
- Adding new integrations
- Significant refactoring

---

### [digest-pipeline-v2.md](digest-pipeline-v2.md) ⭐
**Purpose:** Detailed pipeline design for "many digests per run" architecture
**Status:** Current design (2025-11-06)
**Update frequency:** As pipeline evolves

**Contains:**
- Problem statement (GenAI news overload, credibility, brevity)
- Design principles (many digests, two-dimensional organization, citations)
- 8-step pipeline implementation with Go code examples
- PostgreSQL data model (articles, digests, themes, relationships)
- Query patterns (daily/weekly digest generation)
- Frontend display strategy (Kagi News-style digest list)
- Current implementation issues and fixes
- Technology stack (Go/PostgreSQL/Gemini/K-means)
- Implementation roadmap (5 phases)

**When to read:**
- Understanding digest generation pipeline
- Implementing new pipeline features
- Debugging clustering or summarization
- Designing database queries
- Planning frontend pages

**When to update:**
- Pipeline architecture changes
- New pipeline steps added
- Data model modifications
- Query pattern updates

---

### [migration-plan.md](migration-plan.md)
**Purpose:** Step-by-step migration from single-digest to many-digests architecture
**Status:** Ready to execute (2025-11-06)
**Timeline:** 7-10 days

**Contains:**
- Pre-migration checklist
- 6 phases with detailed tasks
  - Phase 1: Database schema migration (2 days)
  - Phase 2: Repository layer updates (1 day)
  - Phase 3: Pipeline refactor (2 days)
  - Phase 4: Handler consolidation (1 day)
  - Phase 5: Frontend implementation (2 days)
  - Phase 6: Testing & validation (2 days)
- Full code examples for each migration step
- Testing checklist
- Rollback procedures

**When to read:**
- Before starting migration implementation
- Understanding breaking changes
- Planning migration timeline
- Troubleshooting migration issues

**When to update:**
- After completing migration (mark phases done)
- If migration approach changes
- When new issues discovered during migration

---

## 💡 Research & Inspiration

### [inspirations/](inspirations/)
**Purpose:** Comparative analysis of products that inspire Briefly
**Status:** Growing collection
**Update frequency:** When researching new products

**Current research:**
- [Kagi News](inspirations/KAGI_NEWS.md) - News aggregation & digest

**Future candidates:**
- Techmeme (clustering approach)
- Morning Brew (newsletter format)
- Hacker News (community curation)
- Perplexity (AI search with citations)

**When to read:**
- Understanding design decisions ("Why did we choose X?")
- Researching new features
- Comparing approaches

**When to add:**
- Researching a new competitive product
- Evaluating feature inspiration
- Documenting lessons from other systems

See [inspirations/README.md](inspirations/README.md) for template and guidelines.

---

## ⚡ Implementation Planning

### [executions/](executions/)
**Purpose:** Dated implementation plans and task tracking
**Status:** Active plan + historical archive
**Update frequency:** Daily/weekly during execution

**Active plan:**
- [2025-10-31.md](executions/2025-10-31.md) - Initial v2.0 implementation (10 weeks, 7 phases)

**Structure:**
- Phase-by-phase task breakdowns
- Timeline and milestones
- Dependencies and risks
- Success criteria

**When to read:**
- Daily standup / sprint planning
- Understanding current priorities
- Tracking progress against plan

**When to create new:**
- Starting new major version
- Major pivot or direction change
- Quarterly planning cycles

**When to update:**
- Check off completed tasks (daily/weekly)
- Add/remove tasks within phases
- Adjust timelines
- Document blockers

See [executions/README.md](executions/README.md) for lifecycle and template.

---

## 📊 Document Relationships

```
                  ┌─────────────┐
                  │ PRODUCT.md  │ ◄── WHAT & WHY
                  │ (Vision)    │     Goals, metrics, features
                  └──────┬──────┘
                         │
                         │ informs
                         │
                  ┌──────▼───────┐
                  │ARCHITECTURE  │ ◄── HOW
                  │    .md       │     Technical design
                  └──────┬───────┘
                         │
                         │ implements
                         │
                  ┌──────▼───────┐
                  │ executions/  │ ◄── WHEN & WHO
                  │ YYYY-MM-DD   │     Timeline, tasks
                  └──────────────┘
                         ▲
                         │ learns from
                         │
                  ┌──────┴───────┐
                  │inspirations/ │ ◄── WHY THIS WAY
                  │ [Products]   │     Design decisions
                  └──────────────┘
```

**Flow:**
1. **PRODUCT.md** defines what we're building and why
2. **inspirations/** research informs product decisions
3. **ARCHITECTURE.md** describes how to build it technically
4. **executions/** plans when and in what order to build

---

## 🔄 Document Lifecycle

| Document | Type | Lifecycle | Versioning |
|----------|------|-----------|------------|
| PRODUCT.md | Living | Updated when product changes | Semantic (v1.0, v2.0, v2.1) |
| ARCHITECTURE.md | Living | Always reflects current state | Dated updates (2025-10-31) |
| inspirations/*.md | Reference | Added when researching | Dated snapshots |
| executions/*.md | Snapshot | Created per planning cycle, archived when done | YYYY-MM-DD filename |

**Living Documents:**
- Continuously updated
- Single source of truth
- Track history via git commits
- Version number in header

**Snapshot Documents:**
- Point-in-time captures
- Archived when complete/superseded
- Filename includes date
- Multiple versions coexist

---

## 🎯 Quick Reference

**I want to understand...**

- **What Briefly does** → Read [PRODUCT.md](PRODUCT.md)
- **How Briefly works** → Read [ARCHITECTURE.md](ARCHITECTURE.md)
- **How digest pipeline works** → Read [digest-pipeline-v2.md](digest-pipeline-v2.md)
- **How to migrate to many-digests** → Read [migration-plan.md](migration-plan.md)
- **Why we chose X feature** → Check [inspirations/](inspirations/)
- **What we're building next** → See latest [executions/](executions/)
- **How a component is implemented** → Search [ARCHITECTURE.md](ARCHITECTURE.md)
- **Success metrics** → See "Success Metrics" in [PRODUCT.md](PRODUCT.md)
- **What changed in v2.0** → Check "Version History" in each doc
- **Why the old design was replaced** → See [archive/digest-pipeline-design-v1-archived.md](archive/digest-pipeline-design-v1-archived.md)

---

## ✍️ Contributing to Docs

### When to Update PRODUCT.md
- Adding major features or capabilities
- Changing success metrics or goals
- Pivoting product direction
- Quarterly OKR review

### When to Update ARCHITECTURE.md
- After implementing components (keep it current)
- Changing tech stack or patterns
- Adding new integrations
- Major refactoring

### When to Create Inspiration Docs
- Researching competitive products
- Evaluating feature ideas from other tools
- Documenting design decisions

### When to Create Execution Plans
- Starting new development phase
- Quarterly planning
- After completing previous plan

### Style Guide
- Use Markdown with GitHub-flavored syntax
- Include table of contents for docs >500 lines
- Add code examples with language tags
- Use diagrams (ASCII or Mermaid) where helpful
- Keep headers consistent (Title Case)
- Include "Version" and "Date" metadata

---

## 📦 Archive

### Pipeline Design Archives

**[archive/digest-pipeline-design-v1-archived.md](archive/digest-pipeline-design-v1-archived.md)**
- **Status:** DEPRECATED (2025-11-06)
- **Reason:** Python-focused, weekly living digests, single digest per run, HDBSCAN + pgvector
- **Replaced by:** [digest-pipeline-v2.md](digest-pipeline-v2.md)
- **Preserved for:** Historical reference

### Old Unified Design Documents

Old unified design documents are kept in the root `docs/` folder for reference:

- `DESIGN_NEWS_DIGEST_WEBSITE_V2.1.md` - Complete v2.1 unified doc (before split)
- `DESIGN_NEWS_DIGEST_WEBSITE_V2.md` - v2.0 with user comments
- `DESIGN_NEWS_DIGEST_WEBSITE_V2_CLEAN.md` - v2.0 published version
- `DESIGN_V2_EDITORIAL_CHANGES.md` - Editorial tracking document
- `DESIGN_NEWS_DIGEST_WEBSITE.md` - Original v1.0 design

These are historical artifacts and should not be updated. Refer to the organized structure above for current documentation.

---

## 🔗 External Resources

- **Main README:** [../README.md](../README.md) - User-facing documentation
- **Code Documentation:** Run `godoc -http=:6060` for package docs
- **Architecture Decision Records:** See "Technical Decisions" in ARCHITECTURE.md
- **API Documentation:** (To be added - OpenAPI spec)

---

**Last Updated:** 2025-11-06
**Documentation Version:** 2.2 (added digest-pipeline-v2.md and migration-plan.md)
