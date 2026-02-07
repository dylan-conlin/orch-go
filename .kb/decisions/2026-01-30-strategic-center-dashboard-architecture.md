# Decision: Strategic Center Dashboard Architecture

**Date:** 2026-01-30
**Status:** Accepted
**Context:** Synthesized from investigations 2026-01-27-design-redesign-dashboard-ops-view-meta.md and 2026-01-28-inv-design-unified-strategic-center-dashboard.md

## Summary

Transform NeedsAttention into "Strategic Center" with 5 action-oriented categories that manifest the 5-tier escalation model in user-facing UI. Categories: (1) Absorb Knowledge, (2) Give Approvals, (3) Answer Questions, (4) Handle Failures, (5) Tend Knowledge. Backend aggregation via `/api/decisions` endpoint. Replaces operational grouping with decision-oriented grouping.

## The Problem

Current dashboard groups items operationally ("what's wrong?"):
- Dead Agents (no activity 3+ min)
- Awaiting Cleanup (completed but not closed)
- Stalled Agents (same phase 15+ min)
- Errors (recent error events)
- Blocked Issues (from beads)

Gap: Meta-orchestrator has to mentally translate operational states into decisions.
- "Dead agent" → decide to abandon or investigate
- "Pending review" → absorb knowledge or dismiss
- "Blocked issue" → answer question or provide data

Backend has sophisticated 5-tier escalation model (`pkg/verify/escalation.go`) but dashboard doesn't use it for organizing information.

## The Decision

### Transform NeedsAttention into Strategic Center

**Replace, don't add** - Transform NeedsAttention component into Strategic Center to avoid duplicating concerns and maintain 666px width constraint.

**Five action-oriented categories:**

1. **Absorb Knowledge** - Knowledge-producing skill completions (investigation, architect, research)
   - Shows TLDR + recommendation inline
   - Reuses synthesis-card component
   - Maps to: `EscalationReview` level

2. **Give Approvals** - Items requiring visual verification
   - Shows which web/ files changed
   - Approve/Reject actions
   - Maps to: `EscalationBlock` level

3. **Answer Questions** - Strategic questions from questions store
   - Shows blocking impact and age
   - Maps to: question entities with blocking dependencies

4. **Handle Failures** - Failed verifications, escalated agents, dead agents
   - Shows error count
   - Investigate/Abandon actions
   - Maps to: `EscalationFailed` level

5. **Tend Knowledge** (NEW) - Knowledge hygiene signals
   - Synthesis opportunities (3+ investigations on same topic)
   - Pending promotions (kb quick entries worth promoting)
   - Stale decisions (no citations >7 days)
   - Investigation promotions (marked recommend-yes)
   - Purple border (matches Frontier/Questions for strategic content)

### Backend Architecture

**New endpoints:**
- `/api/decisions` - Aggregates agent escalation data, questions, failure states
- `/api/kb-health` - Calls `kb reflect --format json --limit 5` per type (5-minute cache TTL)

**Frontend:**
- `web/src/lib/stores/decisions.ts` - Decision queue store
- `web/src/lib/stores/kb-health.ts` - Knowledge health store
- `web/src/lib/components/decision-center/decision-center.svelte` - Main component
- Replace NeedsAttention with DecisionCenter in `+page.svelte`

## Why This Design

### Principle: Surfacing Over Browsing

From principles: Users shouldn't have to hunt for what needs attention. The Strategic Center surfaces decision points at a glance.

### Principle: Action-Oriented Over State-Oriented

Existing Frontier and Questions sections already follow this pattern:
- Questions: "What needs answering?" (not "what is in answered state?")
- Frontier: "What's decidable?" (not "what has status=ready?")

Strategic Center extends the pattern: group by "what do I DO with this?" not "what IS this?"

### Key Insight: Knowledge Hygiene IS a Decision Type

"Should I consolidate these 12 dashboard investigations into a guide?" is a decision just like "Should I close this agent?"

Knowledge surfaces belong IN the Strategic Center (not separate) because they represent decisions the meta-orchestrator must make.

### Backend Already Exists via kb reflect

Synthesis detection, promotion candidates, stale decision identification are all implemented in kb-cli. The dashboard integration is API exposure + UI, not algorithm development.

Reflection types provided by `kb reflect`:
- `synthesis`: Investigations needing consolidation (3+ on same topic)
- `promote`: kb quick entries worth promoting to decisions
- `stale`: Decisions with no citations >7 days
- `investigation-promotion`: Investigations marked recommend-yes awaiting decision creation

### Escalation Levels are Internal, Actions are External

The 5-tier model is technical plumbing. Users think in terms of what they need to DO, not what technical state something is in.

Mapping:
- `EscalationReview` → "Absorb Knowledge"
- `EscalationBlock` → "Give Approvals"
- `EscalationFailed` → "Handle Failures"
- Questions → "Answer Questions"
- kb reflect signals → "Tend Knowledge"

## Trade-offs

**Accepted:**
- More backend complexity (new endpoints, aggregation logic)
- Loses pure operational view (acceptable: meta-orchestrator's role IS making decisions)
- 5 categories may feel crowded (acceptable: each represents distinct action type)
- Requires kb CLI available for API calls (graceful degradation if missing)

**Rejected:**
- Add section above NeedsAttention: Violates 666px constraint, duplicates concerns
- Client-side aggregation: Requires all data sources available, duplicates escalation logic
- Mode toggle (Decision vs Operational): Already have Operational/Historical toggle, another is confusing
- Separate Knowledge State section: Knowledge hygiene IS a decision type, shouldn't be separate

## Constraints

1. **666px minimum width** - Dashboard must be fully usable at half MacBook Pro screen. No horizontal scrolling.
2. **Follow existing patterns** - Use CollapsibleSection, badges, progressive disclosure like Frontier/Questions sections
3. **Backend aggregation required** - Decision queue computation requires workspace file analysis, escalation logic, questions integration
4. **kb quick entry count: 743 entries** - Substantial knowledge captured (478 decision, 170 constraint, 67 attempt, 28 question)
5. **Graceful degradation** - If kb CLI unavailable, "Tend Knowledge" shows empty state, other categories still work

## Implementation Notes

**Phase 1: API endpoint (Day 1)**
- Create `/api/decisions` in `cmd/orch/serve_decisions.go`
- Aggregate agent escalation data, questions, failure states
- Create `/api/kb-health` in `cmd/orch/serve_kb_health.go`
- Call `kb reflect --format json --limit 5` per type
- 5-minute cache TTL (knowledge changes slowly)

**Phase 2: Frontend store (Day 2)**
- Create `decisions.ts` store with fetch and derived state
- Create `kb-health.ts` store
- Wire into dashboard polling

**Phase 3: Component (Day 3)**
- Create `decision-center.svelte` using existing patterns
- 5 sections: Absorb Knowledge, Give Approvals, Answer Questions, Handle Failures, Tend Knowledge
- Reuse synthesis-card for knowledge display
- Purple border for "Tend Knowledge" section

**Phase 4: Integration (Day 4)**
- Replace NeedsAttention with DecisionCenter in `+page.svelte`
- Test all 5 categories with real data
- Verify 666px width constraint maintained

**Success Criteria:**
- Meta-orchestrator can see all decision points at a glance
- Items grouped by action type, not operational state
- 666px width constraint maintained
- Escalation levels reflected (knowledge work surfaced for review)
- Knowledge hygiene signals visible without running `kb reflect` manually

## References

**Investigations:**
- `.kb/investigations/2026-01-27-design-redesign-dashboard-ops-view-meta.md` - Decision Center design
- `.kb/investigations/2026-01-28-inv-design-unified-strategic-center-dashboard.md` - Knowledge State integration

**Files:**
- `pkg/verify/escalation.go:13-76` - 5-tier escalation model definition
- `web/src/lib/components/needs-attention/needs-attention.svelte` - Current operational view
- `web/src/lib/components/questions-section/questions-section.svelte` - Action-oriented pattern
- `web/src/lib/components/synthesis-card/synthesis-card.svelte` - Knowledge display component

**Models:**
- `.kb/models/dashboard-architecture.md` - Dashboard architecture patterns

**Decisions:**
- `.kb/decisions/2026-01-09-dashboard-reliability-architecture.md` - Dashboard as tier-0 infrastructure

**Principles:**
- Surfacing Over Browsing - `~/.kb/principles.md`
- Session Amnesia - `~/.kb/principles.md`
