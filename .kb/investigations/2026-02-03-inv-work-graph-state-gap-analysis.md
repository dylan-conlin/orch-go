## Summary (D.E.K.N.)

**Delta:** Issues with completed agents (Phase: Complete, status: awaiting-cleanup) don't appear in WIP section or any "needs review" surface. WIP filters to `active || idle` only, attention system defines `verify` badge but doesn't populate it from agent state.

**Evidence:** Traced code path: serve_agents.go status calculation returns "awaiting-cleanup" for Phase:Complete+dead session (line 1095-1099), wip.ts filters to `status === 'active' || status === 'idle'` (line 79), attention.ts defines `verify` badge type (line 16) but no collector surfaces it from agents.

**Knowledge:** This is an attention system gap, not a WIP design issue. The "awaiting-cleanup" state IS an attention signal. Solution: Add AgentAttentionCollector that surfaces awaiting-cleanup agents as `verify` attention items. This fits the unified attention model design.

**Next:** Decision needed on approach - recommend Option A (AgentAttentionCollector + attention badges) over Option B (expand WIP) or Option C (new Needs Review section).

**Authority:** architectural - Cross-component (agent API + attention system + Work Graph UI), affects how orchestrators see completed work, multiple valid approaches.

---

# Investigation: Work Graph State Gap Analysis

**Question:** How should in_progress issues with completed agents be surfaced? Options: 1) Needs Review section, 2) Expand WIP to show completed, 3) Status-based issue view.

**Started:** 2026-02-03
**Updated:** 2026-02-03
**Owner:** Architect Worker
**Phase:** Complete
**Next Step:** Decision from orchestrator
**Status:** Complete

---

## Problem Statement

When an agent reaches `Phase: Complete` and then the session dies (context exhaustion, server restart, etc.), the associated beads issue becomes invisible:

1. **Not in WIP** - WIP filters to `status === 'active' || status === 'idle'`, but status becomes `awaiting-cleanup`
2. **Not in any review section** - No "Needs Review" section exists
3. **Lost in tree** - Just another open issue among potentially hundreds

This creates a gap in the orchestrator's attention model: completed work that needs `orch complete` has no dedicated visibility surface.

---

## Findings

### Finding 1: Status Calculation Is Correct

**Evidence:** serve_agents.go:1095-1099 correctly calculates:
```go
// Priority 2: Phase: Complete reported AND session dead → awaiting-cleanup
if phaseComplete && !sessionAlive {
    return "awaiting-cleanup"
}
```

**Significance:** The backend correctly identifies this state. The gap is in surfacing it.

### Finding 2: WIP Intentionally Filters to Active Agents

**Evidence:** wip.ts:79
```typescript
const running = agentList.filter(a => 
    a.status === 'active' || a.status === 'idle'
);
```

**Significance:** WIP semantically means "work being done now" - including completed-but-not-verified would mix different concerns.

### Finding 3: Attention System Has Verify Badge But No Source

**Evidence:** attention.ts:16 defines:
```typescript
export type AttentionBadgeType =
    | 'verify'         // Phase: Complete, needs orch complete
    | 'decide'         // Investigation has recommendation needing decision
    ...
```

But serve_attention.go collectors are: beads (ready issues), git (likely-done), recently-closed. No collector surfaces awaiting-cleanup agents.

**Significance:** The conceptual model is right (verify is an attention signal), implementation is incomplete.

### Finding 4: Agent Lifecycle Model Supports This Gap

**Evidence:** Agent Lifecycle State Model (`.kb/models/agent-lifecycle-state-model.md`):
> "Beads is the source of truth for completion. OpenCode sessions persist to disk indefinitely. Session existence means nothing about whether the agent is done."

And: Four-layer model shows beads comments (Phase) as highest authority.

**Significance:** The model supports surfacing Phase:Complete as an attention signal independent of session state.

---

## Options Analysis

### Option A: AgentAttentionCollector (RECOMMENDED)

Add a new collector to the attention system that surfaces `awaiting-cleanup` agents as `verify` attention items.

**How it works:**
1. New `AgentAttentionCollector` queries /api/agents for status=awaiting-cleanup
2. Emits attention items with signal=`verify`, concern=`completed-work`
3. Frontend already has `verify` badge styling (VERIFY - blue)
4. Items appear in attention signals, badges show on tree nodes

**Pros:**
- Uses existing attention infrastructure
- Fits conceptual model (awaiting-cleanup IS an attention signal)
- Consistent with "unified attention model" design
- No new UI sections needed

**Cons:**
- Requires new collector implementation
- Attention items are transient (not persisted)

**Implementation scope:** ~100 lines Go (collector), minor frontend wire-up

### Option B: Expand WIP to Include Completed

Keep `awaiting-cleanup` agents in WIP section with visual distinction.

**How it works:**
1. Change WIP filter: `status === 'active' || status === 'idle' || status === 'awaiting-cleanup'`
2. Show different styling for completed agents (checkmark, "Ready for review" text)
3. Group: Running | Waiting for Review | Queued

**Pros:**
- Single location for all "work in flight"
- Less cognitive overhead (one place to look)

**Cons:**
- Mixes different concerns (active work vs completed work)
- WIP gets cluttered with old completed agents
- "Work in Progress" label becomes misleading

**Implementation scope:** ~50 lines TypeScript changes

### Option C: New "Needs Review" Section

Add dedicated section below WIP for completed agents awaiting review.

**How it works:**
1. New `NeedsReviewSection` component (similar to WIPSection)
2. Query /api/agents for status=awaiting-cleanup
3. Show agent details with "Run orch complete" CTA

**Pros:**
- Clear visual separation
- Dedicated UX for review workflow

**Cons:**
- Yet another section (WIP + Needs Review + Tree)
- More UI to maintain
- Duplicates some WIPSection logic

**Implementation scope:** ~150 lines Svelte + store

### Option D: Status-Based Issue View

Organize tree by issue status rather than hierarchy.

**How it works:**
1. Add "Status View" toggle (alongside Tree/Phase views)
2. Group issues: Ready | In Progress (has agent) | Awaiting Review | Done
3. Show agent state within each group

**Pros:**
- Issue-centric rather than agent-centric
- Comprehensive view of all work states

**Cons:**
- Big paradigm shift
- Loses hierarchy visibility
- More complex implementation

**Implementation scope:** ~300+ lines (significant UI rework)

---

## Recommendation

**Recommended: Option A (AgentAttentionCollector)**

**Substrate reasoning:**

1. **Unified Attention Model Design** (`.kb/investigations/2026-02-02-design-work-graph-unified-attention-model.md`): 
   > "Work Graph should *compute* attention priority by synthesizing... Issue open, workspace exists, Phase: Complete → 'Ready for orch complete'"
   
   This explicitly calls out Phase:Complete as an attention signal.

2. **Dashboard Agent Status Model** (`.kb/models/dashboard-agent-status.md`):
   > "Status can be 'wrong' at the dashboard level while being 'correct' at each individual check"
   
   Attention system reconciles across sources - correct pattern.

3. **Session Amnesia Principle** (`~/.kb/principles.md`):
   > "State must externalize to files... Resumption must be explicit"
   
   Attention signals are ephemeral but computed from durable state (beads comments + agent status).

**Trade-offs accepted:**
- Attention items are transient (acceptable - computed from durable sources)
- Requires backend changes (acceptable - follows established pattern)

**What this rejects:**
- Expanding WIP (mixes concerns, violates semantic clarity)
- New section (increases cognitive load without conceptual benefit)
- Status view (scope creep, separate concern from this gap)

---

## Implementation Sequence (If Approved)

### Phase 1: Backend AgentAttentionCollector

1. Add `pkg/attention/agent_collector.go`:
   ```go
   type AgentCollector struct {
       client *http.Client
       apiURL string
   }
   
   func (c *AgentCollector) Collect(role string) ([]AttentionItem, error) {
       // Query /api/agents?status=awaiting-cleanup
       // For each agent, emit AttentionItem with:
       //   Signal: "verify"
       //   Subject: agent.beads_id
       //   Summary: "Phase: Complete - {task}"
       //   ActionHint: "orch complete {beads_id}"
   }
   ```

2. Register in serve_attention.go handleAttention()

3. Test: Create agent, Phase:Complete, kill session, verify attention API returns verify signal

### Phase 2: Frontend Wire-up

1. Verify attention.ts mapSignalToBadge handles "verify" → `verify` badge
2. Verify work-graph tree attaches attention badges (already implemented)
3. Test: Load Work Graph, verify VERIFY badge appears on completed-agent issues

### Phase 3: (Optional) WIP Enhancement

If orchestrator wants completed agents in WIP too:
1. Add "Awaiting Review" subsection within WIP
2. Query attention signals for verify type
3. Display below running agents

---

## Success Criteria

- [ ] `/api/attention` returns verify signals for awaiting-cleanup agents
- [ ] Work Graph tree shows VERIFY badge on issues with completed agents
- [ ] Clicking badge shows "Run orch complete" action hint
- [ ] Badge disappears when `orch complete` runs (issue closes)

---

## References

**Primary Sources:**
- `cmd/orch/serve_agents.go:1095-1099` - Status calculation
- `web/src/lib/stores/wip.ts:79` - WIP filter
- `web/src/lib/stores/attention.ts:16` - AttentionBadgeType definition
- `cmd/orch/serve_attention.go:153-181` - Collector registration

**Models Consulted:**
- `.kb/models/dashboard-agent-status.md` - Priority Cascade
- `.kb/models/agent-lifecycle-state-model.md` - Four-layer model
- `.kb/investigations/2026-02-02-design-work-graph-unified-attention-model.md` - Unified attention concept

**Principles Applied:**
- Session Amnesia - state externalizes to files
- Provenance - signals computed from durable sources
