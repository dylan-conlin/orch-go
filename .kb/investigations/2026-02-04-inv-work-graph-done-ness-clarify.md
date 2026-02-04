# Investigation: Work Graph "Done-ness" State Clarification

**Question:** How should Work Graph present recently-completed work and disambiguate "verify" vs "unverified" states?

**Started:** 2026-02-04
**Updated:** 2026-02-04
**Owner:** Investigation Worker
**Phase:** Complete
**Status:** Complete

---

## Summary (D.E.K.N.)

**Delta:** Current terminology conflates two different concepts: "verify" means "pending closure" (about ticket lifecycle) while "unverified" means "not quality-checked" (about deliverable quality). This creates confusion when displayed together.

**Evidence:** Analyzed attention.ts (lines 8-22 for badge types, lines 25-28 for verification status), recently_closed_collector.go (recently closed signal), verify_failed_collector.go (failure handling), and work-graph-tree.svelte (display logic mixing WIP, completed, and tree nodes).

**Knowledge:** The state model needs two independent dimensions: Closure State (open → pending_close → closed) and Quality State (unverified → verified/needs_rework). Mixing these into one badge creates confusion.

**Next:** 3 implementation issues: (1) Rename "verify" badge to "pending_close", (2) Add "Recently Completed" section with visual separator, (3) Add inspection affordances for completed work outputs.

**Authority:** architectural - UI/UX design decision affecting multiple components

---

## Findings

### Finding 1: Current State Model Is Conflated

**Evidence:** The current attention.ts defines:

```typescript
// Attention badge types for active work
export type AttentionBadgeType =
  | 'verify'         // Phase: Complete, needs orch complete
  | 'recently_closed' // Recently closed, needs verification
  | 'verify_failed'  // Verification failed during auto-completion
  // ... other badges

// Verification status for completed issues
export type VerificationStatus =
  | 'unverified'  // Completed but not human-verified
  | 'verified'    // Human verified as correct
  | 'needs_fix';  // Verified incorrect, needs rework
```

**Source:** web/src/lib/stores/attention.ts:8-28

**Significance:** Two dimensions are mixed:
1. **Closure lifecycle** (verify, recently_closed, verify_failed) - about whether the ticket is closed
2. **Quality verification** (unverified, verified, needs_fix) - about whether the deliverable is correct

The badge "verify" sounds like it's about quality verification, but it actually means "pending closure". This naming collision causes user confusion.

---

### Finding 2: Display Mixes Open and Completed Work

**Evidence:** work-graph-tree.svelte rebuilds flattenedNodes as:
```typescript
// Order: WIP items first, then pending verification, then main tree
flattenedNodes = [...wipItems, ...pendingVerification, ...dedupedTreeNodes];
```

Where pendingVerification includes issues with verificationStatus !== 'verified'.

**Source:** web/src/lib/components/work-graph-tree/work-graph-tree.svelte:60-82

**Significance:** Completed-but-unverified issues are interleaved with running agents and open issues. There's no visual delimiter saying "these are done, these are in progress, these need action."

---

### Finding 3: Source of Truth for Each State

**Evidence:** Based on collectors and stores:

| State | Source | Signal |
|-------|--------|--------|
| Agent claims complete | beads comments ("Phase: Complete") | verify badge from agent_collector |
| Issue closed | beads status field | recently_closed from recently_closed_collector |
| Auto-completion failed | verify-failed.jsonl | verify_failed from verify_failed_collector |
| Human quality review | attention store API | verificationStatus (local state only currently) |

**Source:** 
- pkg/attention/agent_collector.go (for verify signal)
- pkg/attention/recently_closed_collector.go (for recently_closed)
- pkg/attention/verify_failed_collector.go (for verify_failed)
- web/src/lib/stores/attention.ts:markVerified() (local quality tracking)

**Significance:** Quality verification (verified/unverified) is tracked in frontend state but not persisted to beads. This means verification status is lost on page refresh unless the API stores it.

---

### Finding 4: User Cannot Quickly Inspect Completed Work Outputs

**Evidence:** The tree component shows issues but doesn't expose:
- What artifacts the completed work produced
- What commits were made
- What the agent's final summary said

The ATTENTION_BADGE_CONFIG shows labels like "VERIFY", "UNVERIFIED", "NEEDS FIX" but clicking them doesn't reveal the completion evidence.

**Source:** web/src/lib/stores/attention.ts:52-64 (badge config), web/src/lib/components/work-graph-tree/work-graph-tree.svelte

**Significance:** Users see "this is done" but cannot quickly answer "what did it produce?" The inspection UX is missing.

---

## State Model Recommendation

### Two Independent Dimensions

**Closure State** (ticket lifecycle):
| State | Meaning | Badge Display |
|-------|---------|---------------|
| open | Work not done | (no badge) |
| pending_close | Agent claims complete, needs orch complete | PENDING CLOSE |
| closed | Ticket closed | (moved to completed section) |
| verify_failed | Auto-completion failed | VERIFY FAILED |

**Quality State** (deliverable quality - only applies to closed work):
| State | Meaning | Badge Display |
|-------|---------|---------------|
| unreviewed | Closed but human hasn't checked | UNREVIEWED |
| approved | Human verified correct | APPROVED |
| needs_rework | Human found problem | NEEDS REWORK |

### Rename Mapping

| Current Term | New Term | Why |
|--------------|----------|-----|
| verify | pending_close | Makes clear this is about ticket closure, not quality |
| unverified | unreviewed | Avoids collision with "verify" terminology |
| verified | approved | Clearer human action |
| needs_fix | needs_rework | More precise about what happens next |

---

## Presentation Recommendation

### Progressive Disclosure Architecture

```
Work Graph
├── Active Work (WIP Section - existing)
│   ├── Running agents
│   └── Queued issues
│
├── Needs Action (NEW SECTION)
│   ├── PENDING CLOSE - ready for orch complete
│   ├── VERIFY FAILED - auto-complete failed  
│   └── NEEDS REWORK - quality issue found
│
├── Open Issues (existing tree)
│   └── Hierarchical tree of open work
│
└── Recently Completed (NEW SECTION - collapsed by default)
    ├── Last 24h completions
    ├── Grouped by: UNREVIEWED | APPROVED
    └── Expandable for inspection
```

### Visual Separation

1. **Hard delimiter** between Active/Needs Action and Open Issues
2. **Collapsed section** for Recently Completed (doesn't clutter main view)
3. **Count badges** on section headers ("Needs Action (3)", "Recently Completed (7)")
4. **Color coding** consistent with concern type:
   - Active work: blue
   - Needs action: yellow/orange
   - Completed-unreviewed: gray
   - Completed-approved: green
   - Needs rework: red

---

## Inspection UX Recommendation

### Quick Inspection Affordances

For completed issues, provide one-click access to:

1. **Completion Summary** - Agent's "Phase: Complete - ..." message
2. **Artifacts Produced** - Links to investigation files, decisions, code changes
3. **Commits** - Git commits referencing this issue
4. **Attempt History** - Previous attempts if any

### Implementation via IssueSidePanel Enhancement

The existing IssueSidePanel component could be enhanced with a "Completion" tab that shows:
- Phase: Complete comment text
- List of commits mentioning issue ID
- List of artifacts created during spawn (from workspace)
- Visual verification screenshot if captured

---

## Acceptance Criteria Validation

| Criterion | How Addressed |
|-----------|---------------|
| User can tell in <5s what is: open, in-progress, blocked, completed+needs verification, completed+verified, completed+verification-failed | Distinct sections with clear headers and badges |
| Completed items not visually interleaved with open items without strong delimiter | "Recently Completed" is a separate collapsible section |
| Clear affordance to inspect completed item outputs | IssueSidePanel "Completion" tab with artifacts/commits |

---

## Implementation Issues

### Issue 1: Rename "verify" badge terminology

**Problem:** "verify" badge collides with "verified/unverified" quality states, causing user confusion.

**Solution:** Rename throughout codebase:
- verify -> pending_close 
- unverified -> unreviewed
- verified -> approved
- Add comment clarifying closure vs quality dimensions

**Files:**
- web/src/lib/stores/attention.ts
- web/src/lib/components/ui/badge/index.ts  
- pkg/attention/agent_collector.go (signal name)
- Tests

**Scope:** Small (renaming + documentation)

---

### Issue 2: Add "Recently Completed" section with visual separation

**Problem:** Completed-but-unreviewed issues are interleaved with open/active work, making it hard to see what's done.

**Solution:** 
- Create RecentlyCompletedSection component
- Collapsed by default with count badge
- Clear header: "Recently Completed (7)"
- Group by review status (unreviewed | approved)
- Hard visual delimiter from open issues tree

**Files:**
- web/src/lib/components/recently-completed-section/ (new)
- web/src/routes/work-graph/+page.svelte (integrate section)
- web/src/lib/stores/attention.ts (ensure data structure supports grouping)

**Scope:** Medium (new component + integration)

---

### Issue 3: Add completion inspection affordances to IssueSidePanel

**Problem:** User can see an issue is "done" but cannot quickly see what it produced.

**Solution:**
- Add "Completion" tab to IssueSidePanel (existing component)
- Show: Phase: Complete message, commits, artifacts, screenshots
- Make commits clickable (link to GitHub/git)
- Make artifacts clickable (open in new tab or inline)

**Files:**
- web/src/lib/components/issue-side-panel/issue-side-panel.svelte
- May need new API endpoint: /api/completion-details/:beadsId
- pkg/beads/completion_details.go (gather completion info)

**Scope:** Medium-Large (new API + UI enhancement)

---

## References

**Files Examined:**
- web/src/lib/stores/attention.ts - Badge types and verification status
- web/src/lib/components/work-graph-tree/work-graph-tree.svelte - Display logic
- pkg/attention/recently_closed_collector.go - Recently closed signal
- pkg/attention/verify_failed_collector.go - Verification failure handling
- .kb/models/completion-verification.md - Verification architecture
- .kb/investigations/2026-02-02-design-work-graph-unified-attention-model.md - Attention model

**Related Issues:**
- orch-go-21233 (verify_failed attention)
- orch-go-21185 (VERIFY/UNBLOCKED/STUCK signals)
- orch-go-21121.4 (AttemptHistory component)
- orch-go-21244 (surface POST-COMPLETION-FAILURE + REWORK badge)
