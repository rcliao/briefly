# Execution Plans - Historical Implementation Tracking

This directory contains dated execution plans that capture the implementation strategy at specific points in time.

## Purpose

Execution plans are **short-lived documents** that serve as:
- Detailed task breakdowns for upcoming work
- Sprint planning references
- Historical record of how plans evolved
- Retrospective analysis of what was planned vs what was built

Unlike PRODUCT.md and ARCHITECTURE.md (which are living documents), execution plans are snapshots in time and should be archived when complete or superseded.

## Naming Convention

**Format:** `YYYY-MM-DD.md`

**Examples:**
- `2025-10-31.md` - Initial v2.0 implementation plan
- `2025-11-15.md` - Revised plan after Phase 0 completion
- `2026-01-10.md` - Q1 2026 feature roadmap

## Structure

Each execution document should include:

1. **Metadata**
   - Version/iteration number
   - Date created
   - Status (Planning, In Progress, Completed, Superseded)
   - Author/team

2. **Overview**
   - Context: What led to this plan?
   - Goals: What are we trying to achieve?
   - Scope: What's included/excluded?

3. **Phases/Sprints**
   - Detailed task breakdowns with checkboxes
   - Time estimates
   - Dependencies
   - Acceptance criteria

4. **Timeline**
   - Week-by-week or sprint-by-sprint plan
   - Critical path identification
   - Milestones and deliverables

5. **Risks & Mitigations**
   - Technical risks
   - Scope risks
   - Resource constraints

6. **Success Criteria**
   - Definition of done for each phase
   - MVP requirements
   - Launch criteria

## Lifecycle

### When to Create a New Execution Plan

- **Starting a new major version** (v2.0, v3.0)
- **Major pivot or feature addition** (new product direction)
- **Quarterly planning** (Q1, Q2, Q3, Q4 roadmaps)
- **Post-mortem revisions** (after completing/abandoning previous plan)

### When to Update Existing Plan

- **Weekly/daily progress** - Check off completed tasks
- **Minor scope changes** - Add/remove tasks within same phase
- **Risk updates** - New risks discovered or mitigated
- **Timeline adjustments** - Slip dates or accelerate schedule

### When to Archive/Supersede

- **Plan completed** - All phases done, mark as "Completed"
- **Major pivot** - New plan invalidates old approach, mark as "Superseded"
- **Abandoned** - Direction changed, mark as "Abandoned" with reason

## Active vs Historical

**Active Plan:** The most recent execution plan currently being worked on.

To identify the active plan:
1. Check the latest dated file
2. Look for status "In Progress" in the header
3. Reference from main README or ARCHITECTURE.md

**Historical Plans:** Older plans that are completed, superseded, or abandoned.

Keep historical plans for:
- Learning what approaches worked/didn't work
- Understanding why decisions were made
- Retrospectives and process improvement
- Onboarding new team members

## Example Timeline

```
2025-10-31.md  [In Progress]   Initial v2.0 plan (10 weeks, 7 phases)
2025-12-15.md  [Planned]       Phase 3-7 revised after Phase 0-2 learnings
2026-02-01.md  [Planned]       Q1 2026 post-launch roadmap
```

## Template for New Execution Plans

When creating a new plan, use this structure:

```markdown
# Briefly - Implementation Plan

**Date:** YYYY-MM-DD
**Version:** X.X
**Status:** [Planning|In Progress|Completed|Superseded|Abandoned]
**Author:** [Your Name]

## Context

Why are we creating this plan? What changed since the last plan?

## Overview

Brief summary of what we're building and the approach.

## Timeline Summary

| Phase | Duration | Key Deliverables | Status |
|-------|----------|------------------|--------|
| Phase 1 | Week 1 | ... | Pending |

## Phases

### Phase 1: [Name] (Duration)

**Goal:** What we're trying to achieve

**Tasks:**
- [ ] Task 1
- [ ] Task 2

**Acceptance Criteria:**
- Specific measurable outcomes

**Deliverables:**
- What's shipped at the end

---

## Dependencies & Risks

[Document what could go wrong and mitigations]

## Success Criteria

[How do we know we're done?]
```

## Tips

1. **Be specific with tasks** - "Implement theme classifier" not "Work on themes"
2. **Include acceptance criteria** - Know when a task is truly done
3. **Estimate conservatively** - Double your initial estimates
4. **Track actuals vs estimates** - Learn from variance over time
5. **Update regularly** - Check off tasks daily/weekly
6. **Document blockers** - Note why tasks are delayed
7. **Celebrate wins** - Mark completed phases prominently

## Relationship to Other Docs

- **PRODUCT.md** - Defines WHAT and WHY (vision, goals)
- **ARCHITECTURE.md** - Defines HOW (technical design)
- **executions/** - Defines WHEN and WHO (timeline, tasks)

Execution plans bridge product vision and technical architecture with concrete implementation steps.
