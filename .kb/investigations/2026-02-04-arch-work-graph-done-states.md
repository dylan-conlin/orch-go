# Architect Investigation: Work Graph "Done-ness" State Model

**Date:** 2026-02-04
**Status:** Active
**Type:** architect
**Beads:** orch-go-21250

**Question:** How should Work Graph present "done-ness" states to make completion status clear at a glance, while preserving provenance for what happened and what was produced?

**Delta:** Currently `verify` and `unverified` badges are confusingly named, overlap conceptually, and completed items are interleaved with open items. Users can't tell in <5s what work is actually done vs needs attention.

---

## Summary (D.E.K.N.)

**Delta:** The current state model conflates two different lifecycles: *agent completion* (Phase: Complete → orch complete) and *human verification* (closed → human confirms it works). The `verify` and `unverified` badges both mean "needs verification" but at different points. Renaming and visual separation solves this.

**Evidence:** Code analysis confirms: `verify` badge = "Phase: Complete reported, needs `orch complete`". `unverified` badge = "recently closed, human hasn't confirmed it works". Both use "verify" terminology but different meanings.

**Knowledge:** The completion lifecycle has 4 distinct states: Active Work → Agent Done → Orchestrator Closed → Human Verified. Each transition has different actors and visibility needs. Current UI conflates states 2 and 3.

**Next:** Implement 3 issues: (1) Rename badges for clarity, (2) Add visual delimiter for completed items, (3) Enhance inspection UX for outputs.

---

## Findings

### Finding 1: Two Distinct Lifecycles Being Conflated

**Evidence:** The codebase reveals two separate verification flows:

| Flow | Start | End | Actor | Badge |
|------|-------|-----|-------|-------|
| **Agent Completion** | Agent reports "Phase: Complete" | `orch complete` runs verification gates | Orchestrator | `verify` |
| **Human Verification** | Issue closed (by orchestrator or daemon) | Human marks as verified | Human | `unverified` |

The naming is confusing:
- `verify` = "this needs verification by orchestrator to run gates and close"
- `unverified` = "this was already closed but human hasn't verified it works"

Both use "verify" terminology but at different lifecycle points.

**Source:** 
- `web/src/lib/stores/attention.ts:8-11` - AttentionBadgeType definitions
- `web/src/lib/stores/attention.ts:23-26` - VerificationStatus definitions
- `cmd/orch/serve_attention.go:320-410` - Verification endpoint

**Significance:** Users see "VERIFY" and "UNVERIFIED" badges mixed together and can't quickly understand the difference. The naming conflict causes cognitive overhead.

---

### Finding 2: Current Attention Signals Taxonomy

**Evidence:** The attention system has 10 distinct signals:

| Signal | Meaning | Lifecycle Stage | Actor |
|--------|---------|-----------------|-------|
| `verify` | Agent done, needs `orch complete` | Agent → Orchestrator | Orchestrator |
| `verify_failed` | Auto-completion verification failed | Orchestrator → Rework | Human |
| `unverified` | Closed but not human-checked | Closed → Human Verified | Human |
| `needs_fix` | Human verified as broken | Human Verified → Rework | Orchestrator |
| `recently_closed` | Closed within 24h | Closed → Human Verified | Human |
| `likely_done` | Has commits but no workspace | Unknown → Verify | Orchestrator |
| `decide` | Investigation has recommendation | Knowledge → Action | Orchestrator |
| `escalate` | Question needs human judgment | Blocked → Action | Human |
| `unblocked` | Blocker resolved | Blocked → Ready | Daemon |
| `stuck` | Agent stuck >2h | Active → Intervention | Human |

The post-completion signals (`verify`, `verify_failed`, `unverified`, `needs_fix`, `recently_closed`) all relate to "done-ness" but at different stages.

**Source:** 
- `web/src/lib/stores/attention.ts:5-18` - Full badge type enum
- `pkg/attention/types.go:26-53` - ConcernType definitions

**Significance:** The signal taxonomy is comprehensive but the naming doesn't communicate the lifecycle progression clearly.

---

### Finding 3: UI Currently Interleaves Completed with Open

**Evidence:** The work-graph-tree component renders items in this order:
1. WIP items (running agents, queued issues)
2. Completed issues (pending verification) - filtered to `unverified` or `needs_fix`
3. Main tree (open issues)

From `work-graph-tree.svelte:53-67`:
```javascript
const pendingVerification = completedIssues
    .filter(issue => issue.verificationStatus !== 'verified')
    .sort((a, b) => {
        // needs_fix before unverified
        if (a.verificationStatus === 'needs_fix' && b.verificationStatus !== 'needs_fix') return -1;
        // then by priority
        return a.priority - b.priority;
    });
```

The WIP section separates running/queued work, but completed issues that need human verification are rendered inline without a visual delimiter.

**Source:** `web/src/lib/components/work-graph-tree/work-graph-tree.svelte:53-80`

**Significance:** Users can't quickly scan to see "what's done and needs review" vs "what's still open". The items are deduped from the main tree but there's no visual section break.

---

### Finding 4: Inspection UX for Completed Work is Limited

**Evidence:** When a completed issue is expanded (L1 details), the user sees:
- Description (if present)
- Completion timestamp
- Verification status
- Action hints (press `v` to verify, `x` to mark needs fix)

**Missing from L1:**
- What artifacts were produced (investigations, decisions, knowledge)
- Attempt history (how many spawns, what phases completed)
- Links to commits referencing the issue
- Quick path to screenshots/evidence in beads comments

From `work-graph-tree.svelte:490-525`:
```svelte
<!-- L1: Expanded details for completed issues -->
{#if expandedDetails.has(itemId)}
    <div class="expanded-details ...">
        <!-- Description -->
        <!-- Completion info -->
        <!-- Action hints -->
    </div>
{/if}
```

**Source:** `web/src/lib/components/work-graph-tree/work-graph-tree.svelte:490-525`

**Significance:** Users want to quickly verify "what did this produce?" but current L1 only shows metadata, not outputs. The "fastest path to see outputs" acceptance criterion isn't met.

---

## Recommendation

### 1. State Model: Canonical Post-Work States

**Proposed lifecycle with clear naming:**

```
┌───────────────────────────────────────────────────────────────────────────┐
│                           ISSUE LIFECYCLE                                  │
├───────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│  OPEN         WORKING          COMPLETE         CLOSED        CONFIRMED   │
│   │              │                │               │              │        │
│   │    ┌─────────┼────────────────┼───────────────┼──────────────┼──────┐ │
│   │    │         │                │               │              │      │ │
│   ▼    ▼         ▼                ▼               ▼              ▼      │ │
│ ┌───┐ ┌─────┐ ┌──────────┐ ┌────────────┐ ┌────────────┐ ┌──────────┐  │ │
│ │ ○ │→│ ▶/⏸ │→│ COMPLETE │→│ NEEDS REVIEW│→│  VERIFIED  │ │NEEDS FIX │  │ │
│ └───┘ └─────┘ └──────────┘ └────────────┘ └────────────┘ └──────────┘  │ │
│                    │              │              │             │        │ │
│                    │              │              │             │        │ │
│                    │    orch      │   human      │      ◄──────┘        │ │
│                    └──complete────┘──verifies────┘      (rework loop)   │ │
│                                                                         │ │
└─────────────────────────────────────────────────────────────────────────┘
```

**State definitions:**

| State | Badge | Color | Meaning | Actor to Transition |
|-------|-------|-------|---------|---------------------|
| Open | (none) | gray | Work not started | Agent |
| Working | (phase indicator) | blue | Agent actively working | Agent |
| **COMPLETE** | `COMPLETE` | yellow | Agent done, awaiting `orch complete` | Orchestrator |
| **NEEDS REVIEW** | `NEEDS REVIEW` | orange | Closed, awaiting human verification | Human |
| **VERIFIED** | (hidden) | green | Human confirmed it works | (terminal) |
| **NEEDS FIX** | `NEEDS FIX` | red | Human verified broken, needs rework | Orchestrator |
| **VERIFY FAILED** | `VERIFY FAILED` | red | Auto-completion failed | Orchestrator |

**Key changes:**
- Rename `verify` → `COMPLETE` (what it means: agent is complete)
- Rename `unverified` → `NEEDS REVIEW` (clearer action required)
- Keep `needs_fix` and `verify_failed` as-is (already clear)
- Hide `VERIFIED` from main view (truly done = don't show)

---

### 2. Source of Truth

**Recommendation: Derive from multiple sources, present unified view**

| State | Source | Detection Method |
|-------|--------|------------------|
| COMPLETE | beads comments | "Phase: Complete" in comments |
| NEEDS REVIEW | beads + JSONL | status=closed + not in verifications.jsonl as verified |
| VERIFIED | JSONL | verifications.jsonl entry with status=verified |
| NEEDS FIX | JSONL | verifications.jsonl entry with status=needs_fix |
| VERIFY FAILED | JSONL | verify-failed.jsonl entries |

**No changes to source of truth required.** Current architecture is correct; only presentation needs updating.

---

### 3. Presentation: Visual Separation

**Recommendation: Add "Recently Completed" section with visual delimiter**

```
┌─────────────────────────────────────────────────────────────────┐
│ 🔵 WIP (2)                                                      │
├─────────────────────────────────────────────────────────────────┤
│ ▶ orch-go-xyz  Implementing feature...       [Planning] 5m      │
│ ○ orch-go-abc  Queued for spawn                        queued   │
├─────────────────────────────────────────────────────────────────┤
│ 📋 NEEDS ATTENTION (3)                                          │
├─────────────────────────────────────────────────────────────────┤
│ ✓ orch-go-123  Fixed the bug            [COMPLETE]    2h        │
│ ○ orch-go-456  Investigation done       [NEEDS REVIEW] 6h       │
│ ✗ orch-go-789  UI change broken         [NEEDS FIX]    12h      │
├─────────────────────────────────────────────────────────────────┤
│ 📂 OPEN WORK                                                    │
├─────────────────────────────────────────────────────────────────┤
│ ○ P0 orch-go-abc  Epic: New Dashboard                           │
│   ○ P1 orch-go-def  Add sidebar component                       │
│   ○ P1 orch-go-ghi  Wire up routing                             │
└─────────────────────────────────────────────────────────────────┘
```

**Key changes:**
1. Rename implicit "completed issues" → explicit "NEEDS ATTENTION" section
2. Add section headers with counts
3. Group by state within section (COMPLETE → NEEDS REVIEW → NEEDS FIX)
4. Hide VERIFIED items entirely (they're done-done)

---

### 4. Disambiguation: Eliminate verify/unverified Overlap

**Current:**
- `verify` badge → Agent reported Phase: Complete
- `unverified` status → Recently closed, not human-verified

**Proposed:**
- `COMPLETE` badge → Agent reported Phase: Complete (replaces `verify`)
- `NEEDS REVIEW` badge → Closed, awaiting human verification (replaces `unverified`)

**Terminology alignment:**
| Old | New | Rationale |
|-----|-----|-----------|
| `verify` | `COMPLETE` | Describes the state, not the action needed |
| `unverified` | `NEEDS REVIEW` | Action-oriented, clear what human should do |
| `needs_fix` | `NEEDS FIX` | Keep as-is (already clear) |
| `verify_failed` | `VERIFY FAILED` | Keep as-is (already clear) |

---

### 5. Inspection UX: Fastest Path to Outputs

**Recommendation: Enhance L1 details with deliverables and artifacts**

For `COMPLETE` items (agent done, awaiting orch complete):
```
┌──────────────────────────────────────────────────────────────┐
│ ✓ orch-go-123  Fixed the login bug                [COMPLETE] │
├──────────────────────────────────────────────────────────────┤
│ 📦 Deliverables:                                             │
│    ✓ Investigation file                                      │
│    ✓ Tests added (12 new)                                    │
│    ✓ SYNTHESIS.md                                            │
│                                                              │
│ 📸 Evidence:                                                 │
│    • Screenshot attached (1)                                 │
│    • Test output: "15 passed, 0 failed"                      │
│                                                              │
│ ⌨️ Actions: [c]omplete  [i]nspect  [r]eopen                  │
└──────────────────────────────────────────────────────────────┘
```

For `NEEDS REVIEW` items (closed, awaiting human verification):
```
┌──────────────────────────────────────────────────────────────┐
│ ○ orch-go-456  New dashboard layout           [NEEDS REVIEW] │
├──────────────────────────────────────────────────────────────┤
│ 📦 Produced:                                                 │
│    • Investigation: 2026-02-04-inv-dashboard.md              │
│    • Decision: Progressive disclosure (accepted)             │
│                                                              │
│ 📊 Attempt History:                                          │
│    Spawn 1: investigation → complete ✓                       │
│    Spawn 2: implementation → complete ✓                      │
│                                                              │
│ ⌨️ Actions: [v]erify  [x]needs fix  [o]pen details           │
└──────────────────────────────────────────────────────────────┘
```

---

## Implementation Issues

### Issue 1: Rename verify/unverified badges for clarity

**Problem:** Current `verify` and `unverified` badges use confusing terminology that doesn't communicate the lifecycle stage.

**Solution:** 
- Rename `verify` → `COMPLETE` (AttentionBadgeType and ATTENTION_BADGE_CONFIG)
- Rename `unverified` → `NEEDS REVIEW` (VerificationStatus and display)
- Update frontend components to use new labels
- Update backend signal naming for consistency

**Evidence:** User report: "verify/unverified mixed with open issues, unclear how to interpret"

**Files affected:**
- `web/src/lib/stores/attention.ts` - Badge type definitions
- `web/src/lib/components/work-graph-tree/work-graph-tree.svelte` - Badge rendering
- `cmd/orch/serve_attention.go` - API response mapping (optional, for consistency)

**Acceptance criteria:**
- User sees `COMPLETE` badge for "Phase: Complete" issues
- User sees `NEEDS REVIEW` badge for closed-but-unverified issues
- No more confusion between "verify" terminology variants

---

### Issue 2: Add visual delimiter for "Needs Attention" section

**Problem:** Completed items are inline with open items, making it hard to scan for "what needs my attention".

**Solution:**
- Add explicit section header between WIP and completed items
- Group completed items by state (COMPLETE → NEEDS REVIEW → NEEDS FIX)
- Show counts in section header (e.g., "NEEDS ATTENTION (3)")
- Hide VERIFIED items from the list entirely

**Evidence:** Acceptance criteria: "Completed items not visually interleaved with open items without strong delimiter"

**Files affected:**
- `web/src/lib/components/work-graph-tree/work-graph-tree.svelte` - Section rendering
- `web/src/routes/work-graph/+page.svelte` - Section data structure

**Acceptance criteria:**
- User can tell in <5s what needs attention vs what's open
- Section header shows count of items
- VERIFIED items don't appear in the list

---

### Issue 3: Enhance L1 inspection with deliverables and artifacts

**Problem:** Expanded details for completed items only show metadata, not what was produced.

**Solution:**
- Add "Deliverables" section showing SYNTHESIS.md, investigation files, etc.
- Add "Evidence" section showing screenshots and test output from comments
- Add "Attempt History" for closed items showing spawn attempts
- Add keyboard shortcuts for common actions (complete, inspect, reopen)

**Evidence:** Acceptance criteria: "Clear affordance to inspect what the completed item produced"

**Files affected:**
- `web/src/lib/components/work-graph-tree/work-graph-tree.svelte` - L1 details
- `web/src/lib/stores/deliverables.ts` - May need to expose more data
- New API endpoint for artifact lookup (optional enhancement)

**Acceptance criteria:**
- User can see what artifacts were produced in <3 clicks
- Evidence (screenshots, test output) visible in L1 details
- Attempt history visible for closed issues

---

## References

**Prior Investigations:**
- `.kb/investigations/2026-02-02-design-work-graph-unified-attention-model.md` - Unified attention model
- `.kb/investigations/2026-02-02-inv-deep-review-work-graph-read.md` - Deep review of Work Graph
- `.kb/investigations/2026-02-02-inv-work-graph-verification-gap-issues.md` - Verification gap analysis

**Models:**
- `.kb/models/completion-verification.md` - Three-layer verification architecture
- `.kb/models/completion-lifecycle.md` - Agent completion lifecycle

**Source Code:**
- `web/src/lib/stores/attention.ts` - Attention store with badge types
- `web/src/lib/components/work-graph-tree/work-graph-tree.svelte` - Tree component
- `cmd/orch/serve_attention.go` - Attention API
- `pkg/attention/types.go` - Attention type definitions

---

## Investigation History

**2026-02-04 11:00:** Investigation started
- Read SPAWN_CONTEXT.md, understood the problem
- Reviewed all related investigations and models

**2026-02-04 11:30:** Code analysis complete
- Mapped full attention signal taxonomy (10 signals)
- Identified verify/unverified naming confusion
- Documented current UI presentation

**2026-02-04 12:00:** Recommendations drafted
- 5-state lifecycle model proposed
- Terminology clarification defined
- Visual presentation designed
- 3 implementation issues created

**2026-02-04 12:30:** Investigation complete

---

## Follow-up Issues Created

| Issue | Title | Priority |
|-------|-------|----------|
| orch-go-21254 | Rename verify/unverified badges for clarity | P2 |
| orch-go-21252 | Add "Recently Completed" section with visual separation | P2 |
| orch-go-21255 | Enhance L1 inspection with deliverables and artifacts | P2 |

All issues linked as children of orch-go-21250 (this investigation).
