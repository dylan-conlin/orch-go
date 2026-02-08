## Summary (D.E.K.N.)

**Delta:** Designed synthesis review workflow that intercepts batch-close operations to surface SYNTHESIS.md recommendations before closing, with three integration points: `orch complete`, `orch review done`, and dashboard.

**Evidence:** Code analysis of `review.go:516-674` shows `runReviewDone()` closes issues without surfacing synthesis recommendations. Constraint exists: "orch complete must verify SYNTHESIS.md exists and is not placeholder before closing."

**Knowledge:** The gap is in batch operations - individual `orch complete` can prompt for recommendations, but `review done` bypasses this. Review state tracking is needed to know which recommendations were acted on vs dismissed.

**Next:** Implement - three-phase approach: (1) Add review state to SYNTHESIS.md parsing, (2) Add recommendation prompts to `review done`, (3) Add dashboard synthesis review section with actionable buttons.

**Confidence:** High (85%) - Design addresses the problem clearly; implementation details need validation during development.

---

# Design Investigation: Synthesis Review Workflow

**Question:** How should orchestrators review SYNTHESIS.md recommendations before batch-closing agents, and where should this review experience live?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** design-session agent
**Phase:** Complete
**Next Step:** Implementation
**Status:** Complete
**Confidence:** High (85%)

---

## Problem Statement

Completed agents have SYNTHESIS.md files with valuable recommendations (follow-up issues, escalations, unexplored questions), but orchestrators using `orch review done <project>` batch-close without extracting this value.

The constraint exists: "orch complete must verify SYNTHESIS.md exists and is not placeholder before closing issue" (from 24h chaos period where 70% of agents completed without synthesis).

The gap: **individual `orch complete <beads-id>` can prompt for recommendations, but batch operations bypass this.**

---

## Findings

### Finding 1: `runReviewDone()` closes without surfacing recommendations

**Evidence:** In `cmd/orch/review.go:516-674`, the batch completion loop:
```go
for _, c := range canComplete {
    // ...
    if err := verify.CloseIssue(c.BeadsID, reason); err != nil {
        // ...
    }
}
```

The synthesis is parsed earlier (`verify.ParseSynthesis`) but only used for display, not for prompting follow-up actions.

**Source:** `cmd/orch/review.go:596-619`

**Significance:** Batch-close is the primary workflow for busy orchestrators. If it doesn't surface recommendations, they're lost.

---

### Finding 2: Synthesis parsing already extracts actionable items

**Evidence:** `verify.ParseSynthesis()` in `pkg/verify/check.go:167-216` extracts:
- `NextActions []string` - Follow-up items
- `Recommendation string` - close, spawn-follow-up, escalate, resume
- `AreasToExplore []string` - Areas worth exploring further
- `Uncertainties []string` - What remains unclear

**Source:** `pkg/verify/check.go:141-165`, `pkg/verify/review.go:87-108`

**Significance:** No new parsing needed. The infrastructure exists to identify recommendations that need orchestrator action.

---

### Finding 3: Dashboard already has issue creation API

**Evidence:** `POST /api/issues` in `cmd/orch/serve.go:1501-1596` creates beads issues from dashboard. Frontend `agent-detail-panel.svelte` displays synthesis with "Create Issue" buttons for next actions.

**Source:** `cmd/orch/serve.go:199` (endpoint), recent investigation `2025-12-26-inv-synthesis-review-view-parse-synthesis.md`

**Significance:** Dashboard integration exists. The gap is making synthesis review a first-class section (vs buried in agent detail).

---

### Finding 4: Decision exists - prompt individually for recommendations

**Evidence:** From kb context: "orch complete prompts for each SYNTHESIS.md recommendation individually - Reason: Better UX than batch y/n - allows selective follow-up issue creation"

**Source:** spawn context prior knowledge

**Significance:** The UX pattern is established. Apply the same pattern to `review done` instead of auto-closing.

---

## Synthesis

**Key Insights:**

1. **Batch operations are the value leak** - Individual `orch complete` can prompt, but orchestrators prefer batch workflows. If batch-close skips recommendations, the value is lost.

2. **Three integration points exist:**
   - `orch complete <beads-id>` - Already can prompt (verify this is implemented)
   - `orch review done <project>` - Currently bypasses prompts (needs fix)
   - Dashboard - Has display but buried in agent detail (needs dedicated section)

3. **Review state tracking needed** - To know what was acted on vs dismissed, we need:
   - Mark synthesis as "reviewed" after orchestrator sees it
   - Track which recommendations became issues vs skipped
   - Show unreviewed synthesis prominently

---

## Implementation Recommendations

### Recommended Approach: Three-Phase Integration

**Phase 1: Enhance `orch review done` with recommendation prompts**

Add synthesis review loop before closing each issue:

```go
// In runReviewDone(), before closing
for _, c := range canComplete {
    if c.Synthesis != nil && len(c.Synthesis.NextActions) > 0 {
        fmt.Printf("\n[%s] Has %d recommendations:\n", c.WorkspaceID, len(c.Synthesis.NextActions))
        for i, action := range c.Synthesis.NextActions {
            fmt.Printf("  %d. %s\n", i+1, action)
        }
        fmt.Print("Create follow-up issues? [y/n/skip-all]: ")
        // Handle response: y=create issues, n=skip this agent, skip-all=close remaining without prompts
    }
    // Then close issue
}
```

**Why:** Directly addresses the batch-close value leak. Minimal implementation effort.

**Phase 2: Add review state tracking**

Extend `Synthesis` struct to track review state:

```go
type Synthesis struct {
    // ... existing fields
    ReviewedAt   time.Time `json:"-"` // When orchestrator reviewed
    ActedOn      []int     `json:"-"` // Indices of recommendations that became issues
    Dismissed    []int     `json:"-"` // Indices explicitly skipped
}
```

Store review state in workspace (e.g., `.orch/workspace/{name}/.review-state.json`).

**Why:** Enables dashboard to show "unreviewed" badges and prevents re-prompting.

**Phase 3: Dashboard synthesis review section**

Add dedicated section in dashboard (not buried in agent detail):

```
## Pending Synthesis Reviews (3)

[og-feat-implement-auth-25dec] - 2 recommendations
  └─ "Add rate limiting to auth endpoints" [Create Issue] [Dismiss]
  └─ "Consider session-based auth for mobile" [Create Issue] [Dismiss]

[og-debug-memory-leak-25dec] - 1 escalation
  └─ ESCALATE: "Memory growth exceeds expectations - need architectural decision"
     Options: 1. Add aggressive GC 2. Refactor to streaming
     [Create Decision Issue] [Dismiss]
```

**Why:** Makes synthesis review a first-class activity, not an afterthought.

---

### Alternative Approaches Considered

**Option B: Auto-create issues from all recommendations**
- **Pros:** No orchestrator interaction needed
- **Cons:** Creates noise; not all recommendations warrant issues
- **When to use:** Never - defeats the purpose of orchestrator curation

**Option C: Require synthesis review before batch-close**
- **Pros:** Guarantees value extraction
- **Cons:** Too friction-heavy; orchestrators will bypass
- **When to use:** For high-value work (epics, architectural changes)

**Rationale for recommendation:** Phase 1 is minimal friction (prompts during batch-close). Phase 2-3 add value without blocking workflows.

---

### Where Review Lives

| Location | Role | When Used |
|----------|------|-----------|
| `orch complete <beads-id>` | Single-agent completion | When completing one agent directly |
| `orch review done <project>` | Batch completion with prompts | When closing multiple agents at once |
| Dashboard "Pending Reviews" | Visual review with issue creation | When orchestrator is in dashboard mode |

All three should share the same review state to avoid duplicate prompts.

---

### Implementation Details

**What to implement first:**
1. Add recommendation prompts to `runReviewDone()` - highest impact, existing code
2. Add `--no-prompt` flag for fully automated batch-close (escape hatch)
3. Test with real synthesis files to validate UX

**Things to watch out for:**
- Long recommendation lists (>5 items) - consider pagination or summary
- Recommendations that are already issues - check beads for duplicates
- Empty synthesis recommendations - skip prompt entirely

**Areas needing further investigation:**
- Should escalations block batch-close entirely?
- Should dashboard show synthesis from other projects?
- How to handle stale recommendations (agent completed days ago)?

**Success criteria:**
- Orchestrator sees recommendations before any batch-close operation
- At least one integration point (CLI or dashboard) is implemented
- Review state persists across sessions

---

## References

**Files Examined:**
- `cmd/orch/review.go` - Batch completion workflow
- `pkg/verify/check.go` - Synthesis parsing
- `pkg/verify/review.go` - Agent review formatting
- `cmd/orch/serve.go` - Dashboard API endpoints
- `.orch/templates/SYNTHESIS.md` - Synthesis template structure

**Related Decisions:**
- "orch complete prompts for each SYNTHESIS.md recommendation individually"
- "Dashboard synthesis review shows synthesis inline with actionable issue creation"

---

## Investigation History

**2025-12-26:** Investigation started
- Initial question: Design synthesis review workflow for completed agents
- Context: Spawned from design-session skill to address value extraction gap

**2025-12-26:** Design complete
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Three-phase implementation plan with clear integration points
