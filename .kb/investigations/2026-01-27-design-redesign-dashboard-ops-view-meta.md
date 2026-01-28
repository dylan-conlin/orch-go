## Summary (D.E.K.N.)

**Delta:** Current dashboard Ops view surfaces operational concerns but doesn't support meta-orchestrator strategic decision-making. Redesigning as "Decision Center" with action-oriented categories (Absorb Knowledge, Give Approvals, Answer Questions, Handle Failures) that manifest the 5-tier escalation model in user-facing UI.

**Evidence:** Analysis of escalation.go (5-tier model exists but not surfaced), NeedsAttention component (operationally-oriented grouping), existing dashboard patterns (QuestionsSection, FrontierSection for action-oriented sections).

**Knowledge:** Escalation levels are internal plumbing - users think in actions ("what do I need to do?"), not technical levels. The mapping: knowledge-producing skills → "Absorb Knowledge", visual verification → "Give Approvals", strategic questions → "Answer Questions", failures → "Handle Failures".

**Next:** Implement as new feature with phased approach: 1) API endpoint /api/decisions, 2) decisions.ts store, 3) decision-center.svelte component, 4) integrate into +page.svelte replacing NeedsAttention.

**Promote to Decision:** recommend-yes - This establishes the pattern for decision-oriented dashboard design, maps escalation model to UX, and creates architectural constraint that new agent completion flows should integrate with decision queue.

---

# Investigation: Redesign Dashboard Ops View for Meta-Orchestrator Decision-Making

**Question:** How should the dashboard Ops view be restructured to support meta-orchestrator strategic decision-making by surfacing escalation-level-based decision points rather than operational status categories?

**Started:** 2026-01-27
**Updated:** 2026-01-27
**Owner:** Architect session
**Phase:** Complete
**Next Step:** None (investigation complete)
**Status:** Complete

---

## Findings

### Finding 1: The 5-tier escalation model exists in code but doesn't manifest in UI

**Evidence:** `pkg/verify/escalation.go` defines EscalationNone through EscalationFailed with clear criteria for each level. Knowledge-producing skills (investigation, architect, research, design-session, codebase-audit, issue-creation) are explicitly categorized. The model includes:
- EscalationNone - Auto-complete silently
- EscalationInfo - Auto-complete, log for optional review  
- EscalationReview - Queue for mandatory review (knowledge work)
- EscalationBlock - Surface immediately (visual approval needed)
- EscalationFailed - Failure state requiring intervention

**Source:** `pkg/verify/escalation.go:13-76`, specifically the `knowledgeProducingSkills` map and `DetermineEscalation()` function.

**Significance:** The backend has a sophisticated model for categorizing what needs human attention, but the dashboard doesn't use this model for organizing information. The NeedsAttention component groups by operational state (dead, stalled, errors) rather than decision type.

---

### Finding 2: Current NeedsAttention component is operationally-oriented, not decision-oriented

**Evidence:** `needs-attention.svelte` groups items by:
- Dead Agents (no activity 3+ min)
- Awaiting Cleanup (completed but not closed)
- Stalled Agents (same phase 15+ min)
- Escalated Agents (failed resume attempts)
- Errors (recent error events)
- Blocked Issues (from beads)
- Pending Reviews (SYNTHESIS.md items)

This answers "what's wrong?" but not "what decision do I need to make?"

**Source:** `web/src/lib/components/needs-attention/needs-attention.svelte:111-370`

**Significance:** The meta-orchestrator has to mentally translate operational states into decisions. "Dead agent" → decide to abandon or investigate. "Pending review" → absorb knowledge or dismiss. This mental translation is cognitive overhead.

---

### Finding 3: Existing dashboard patterns support action-oriented sections

**Evidence:** QuestionsSection and FrontierSection already follow an action-oriented pattern:
- Questions: "What needs answering?" with open/investigating/answered categories
- Frontier: "What's decidable?" with ready/blocked/active/stuck categories

Both use the pattern: count badge + category badges + preview in collapsed state, full list in expanded state.

**Source:** `web/src/lib/components/questions-section/questions-section.svelte`, `web/src/lib/components/frontier-section/frontier-section.svelte`

**Significance:** The dashboard already has patterns for action-oriented sections. The Decision Center can follow the same pattern, maintaining visual consistency.

---

### Finding 4: Dashboard panel additions follow established API → Store → Page pattern

**Evidence:** From SPAWN_CONTEXT prior decisions: "Dashboard panel additions follow pattern: API endpoint in serve.go → Svelte store → page.svelte integration"

The daemon, beads, questions, frontier sections all follow this pattern:
1. `/api/{resource}` endpoint in serve.go
2. `{resource}.ts` store in web/src/lib/stores/
3. Integration in +page.svelte with collapsible section

**Source:** `.kb/decisions/dashboard-panel-additions-pattern`, existing stores in `web/src/lib/stores/`

**Significance:** The Decision Center should follow the same pattern: `/api/decisions` → `decisions.ts` → `decision-center.svelte` component.

---

### Finding 5: Existing synthesis card component can be reused

**Evidence:** `synthesis-card.svelte` already renders:
- TLDR (truncated to 120 chars)
- Outcome badge (success/partial/blocked/failed)
- Recommendation with icon
- Delta summary
- Next actions (first 2 items)

This component is designed for quick knowledge absorption.

**Source:** `web/src/lib/components/synthesis-card/synthesis-card.svelte`

**Significance:** The "Absorb Knowledge" section can reuse this component for inline knowledge presentation, avoiding duplication.

---

## Synthesis

**Key Insights:**

1. **Escalation levels are internal, actions are external** - The 5-tier model is technical plumbing. Users think in terms of what they need to DO, not what technical state something is in. The mapping: EscalationReview → "Absorb Knowledge", EscalationBlock → "Give Approvals", EscalationFailed → "Handle Failures".

2. **Replace, don't add** - Rather than adding a new section above NeedsAttention, transform NeedsAttention into Decision Center. This avoids duplicating concerns and maintains dashboard density constraints.

3. **Backend aggregation is necessary** - Decision queue computation requires workspace file analysis (SYNTHESIS.md, web/ changes), escalation logic application, and questions integration. This is better done server-side.

**Answer to Investigation Question:**

The dashboard Ops view should be restructured as a "Decision Center" with four action-oriented sections:

1. **Absorb Knowledge** - Knowledge-producing skill completions (investigation, architect, research) that need human synthesis. Shows TLDR + recommendation inline.

2. **Give Approvals** - Items requiring visual verification before closure. Shows which web/ files changed, with Approve/Reject actions.

3. **Answer Questions** - Strategic questions from the questions store. Shows blocking impact and age.

4. **Handle Failures** - Failed verifications, escalated agents, dead agents requiring decision. Shows error count with Investigate/Abandon actions.

This directly manifests the 5-tier escalation model in user-facing terms while maintaining the established dashboard patterns (collapsible sections, badges, progressive disclosure).

---

## Structured Uncertainty

**What's tested:**

- ✅ Escalation model exists and categorizes completions correctly (verified: read escalation.go)
- ✅ NeedsAttention currently groups operationally (verified: read component)
- ✅ Dashboard patterns for action-oriented sections exist (verified: read Questions/Frontier sections)
- ✅ Synthesis card component exists and can be reused (verified: read component)

**What's untested:**

- ⚠️ Performance of backend decision aggregation (not benchmarked)
- ⚠️ UX of action-oriented grouping vs operational grouping (not user tested)
- ⚠️ Edge cases when same agent appears in multiple categories (not designed)

**What would change this:**

- If users prefer operational grouping over action grouping (unlikely given meta-orchestrator role)
- If backend aggregation causes unacceptable latency (would need client-side alternative)
- If visual verification workflow changes significantly (would need UI adjustment)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**"Decision Center" with Backend Aggregation** - Transform NeedsAttention into action-oriented Decision Center using new `/api/decisions` endpoint.

**Why this approach:**
- Directly manifests the 5-tier escalation model in user-facing UI
- Follows established dashboard patterns (API → Store → Page)
- Backend aggregation handles complexity of multi-source data
- Action-oriented grouping matches meta-orchestrator mental model

**Trade-offs accepted:**
- More backend complexity (new endpoint, aggregation logic)
- Loses pure operational view (operational concerns now framed as decisions)
- Why that's acceptable: Meta-orchestrator's role IS making decisions; operational view without decision framing is incomplete

**Implementation sequence:**
1. **Phase 1: API endpoint** - Create `/api/decisions` in serve.go that aggregates agent escalation data, questions, and failure states
2. **Phase 2: Frontend store** - Create `decisions.ts` store with fetch and derived state
3. **Phase 3: Component** - Create `decision-center.svelte` using existing patterns (CollapsibleSection, synthesis-card)
4. **Phase 4: Integration** - Replace NeedsAttention with DecisionCenter in +page.svelte

### Alternative Approaches Considered

**Option B: Client-side Aggregation**
- **Pros:** No new API endpoint, simpler backend
- **Cons:** Requires all data sources available client-side, escalation logic duplicated in TypeScript
- **When to use instead:** If all data sources were already available with consistent format

**Option C: Add Section Above NeedsAttention**
- **Pros:** Preserves existing operational view
- **Cons:** Duplicates concerns, adds vertical space, violates 666px constraint
- **When to use instead:** If there's a strong need for both decision and operational views

**Option D: Mode Toggle (Decision vs Operational)**
- **Pros:** Clean separation of concerns
- **Cons:** Already have Operational/Historical mode toggle, adding another toggle is confusing
- **When to use instead:** If the two views have completely different audiences

**Rationale for recommendation:** Option A (backend aggregation, replace NeedsAttention) best serves the meta-orchestrator use case while maintaining existing patterns and constraints.

---

### Implementation Details

**What to implement first:**
- `/api/decisions` endpoint with escalation-level detection
- Basic Decision Center component with all four categories
- Integration with existing agents and questions stores

**Things to watch out for:**
- ⚠️ Edge case: Agent could be knowledge-producing AND have visual verification needed (show in both?)
- ⚠️ Performance: Workspace file scanning for SYNTHESIS.md could be slow with many agents
- ⚠️ Cache invalidation: When to refresh decision queue (on SSE events? on poll?)

**Areas needing further investigation:**
- How to handle agents that span multiple categories
- Whether to add quick actions (Approve, Abandon) directly in dashboard or keep in detail panel
- Caching strategy for decision queue

**Success criteria:**
- ✅ Meta-orchestrator can see all decision points at a glance
- ✅ Items grouped by action type, not operational state
- ✅ 666px width constraint maintained
- ✅ Escalation levels reflected (knowledge work surfaced for review)

---

## File Targets

**Backend (Go):**
- `cmd/orch/serve_decisions.go` (new) - `/api/decisions` endpoint
- `pkg/verify/escalation.go` - May need helper exports for decision categorization

**Frontend (Svelte):**
- `web/src/lib/stores/decisions.ts` (new) - Decision queue store
- `web/src/lib/components/decision-center/decision-center.svelte` (new) - Main component
- `web/src/routes/+page.svelte` - Replace NeedsAttention with DecisionCenter

---

## References

**Files Examined:**
- `pkg/verify/escalation.go` - 5-tier escalation model definition
- `web/src/lib/components/needs-attention/needs-attention.svelte` - Current operational view
- `web/src/lib/components/questions-section/questions-section.svelte` - Action-oriented section pattern
- `web/src/lib/components/frontier-section/frontier-section.svelte` - Action-oriented section pattern
- `web/src/lib/components/synthesis-card/synthesis-card.svelte` - Knowledge display component
- `web/src/routes/+page.svelte` - Dashboard main page structure
- `.kb/models/dashboard-architecture.md` - Dashboard architecture model

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-09-dashboard-reliability-architecture.md` - Dashboard as tier-0 infrastructure
- **Model:** `.kb/models/dashboard-architecture.md` - Two-mode design, SSE patterns
- **Model:** `.kb/models/dashboard-agent-status.md` - Agent status calculation

---

## Investigation History

**2026-01-27 09:00:** Investigation started
- Initial question: How to restructure Ops view for meta-orchestrator decision-making?
- Context: Task spawned from orchestrator to design strategic UI redesign

**2026-01-27 09:30:** Phase 1 (Problem Framing) completed
- Identified gap between 5-tier escalation model and UI representation
- Defined success criteria around decision-oriented grouping

**2026-01-27 10:00:** Phase 2 (Exploration) completed
- Identified 5 decision forks
- Consulted substrate (principles, existing patterns, constraints)

**2026-01-27 10:30:** Phase 3 (Synthesis) completed  
- Navigated all forks with recommendations
- Designed "Decision Center" structure with 4 categories

**2026-01-27 11:00:** Investigation completed
- Status: Complete
- Key outcome: Recommended "Decision Center" design that manifests escalation model in action-oriented UI
