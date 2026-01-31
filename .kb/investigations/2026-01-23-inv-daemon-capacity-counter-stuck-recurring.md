<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Daemon capacity leak recurs because `getClosedIssuesForProject()` treats beads lookup errors as "issue is open", inflating `actualCount` and preventing reconciliation from freeing slots.

**Evidence:** Code analysis shows `client.Show()` errors cause `continue` (line 228) without adding to `closed` map; all 3 prior fixes addressed project resolution but not error handling.

**Knowledge:** Capacity counting that depends on external lookups must handle errors defensively - lookup failure ≠ "issue is open"; the system needs active slot release on completion, not just passive reconciliation.

**Next:** Implement two-part fix: (1) Release slots immediately when daemon auto-completes agents, (2) Make beads lookup errors non-fatal for reconciliation.

**Promote to Decision:** Actioned - capacity fixes implemented

---

# Investigation: Daemon Capacity Counter Stuck Recurring

**Question:** Why does the daemon capacity counter get stuck at max despite 3 prior fixes, and what systemic fix will prevent recurrence?

**Started:** 2026-01-23
**Updated:** 2026-01-23
**Owner:** Architect Agent (orch-go-ngsok)
**Phase:** Complete
**Next Step:** Implement recommended fix
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A (extends prior investigations)
**Superseded-By:** N/A

---

## Findings

### Finding 1: Error handling in getClosedIssuesForProject treats lookup failures as "open"

**Evidence:** In `pkg/daemon/active_count.go:217-234`:
```go
for _, id := range beadsIDs {
    issue, err := client.Show(id)
    if err != nil {
        // If we can't find the issue, assume it's not running
        // (might have been deleted or never existed)
        continue  // ← BUG: doesn't add to closed map
    }
    if strings.EqualFold(issue.Status, "closed") {
        closed[id] = true
    }
}
```

When `client.Show()` returns ANY error (connection timeout, RPC unavailable, wrong directory), the issue is NOT added to `closed` map. In `DefaultActiveCount()`, issues not in `closed` are counted as active.

Same bug exists in CLI fallback (lines 239-248).

**Source:** `pkg/daemon/active_count.go:217-248`

**Significance:** This is the root cause. Any failure in beads lookup inflates `actualCount`, which prevents `Pool.Reconcile()` from freeing slots. Prior fixes improved project resolution but didn't address error handling.

---

### Finding 2: Prior fixes addressed different failure modes but not error handling

**Evidence:** Three prior investigations and fixes:

1. **2025-12-26:** Added `Pool.Reconcile()` mechanism - assumed `DefaultActiveCount()` returns accurate count
2. **2026-01-22 (cross-project):** Fixed `GetClosedIssuesBatch()` to query correct project's beads database - improved project resolution
3. **2026-01-22 (session.Directory):** Fixed `DefaultActiveCount()` to use `session.Directory` for project path - improved project resolution

All three fixes share a common assumption: **if the beads lookup succeeds, it returns accurate status**. None addressed: **what if the beads lookup fails?**

**Source:**
- `.kb/investigations/2025-12-26-inv-daemon-capacity-count-goes-stale.md`
- `.kb/investigations/2026-01-22-inv-daemon-capacity-tracking-stale-after.md`
- `.kb/investigations/2026-01-22-inv-fix-daemon-capacity-counter-getting.md`

**Significance:** This explains the recurring nature of the bug. Each fix improved one failure mode (no reconciliation, wrong project, missing directory), but the error handling gap remained, allowing the bug to recur when any beads lookup fails.

---

### Finding 3: Daemon already processes completions but doesn't release slots directly

**Evidence:** In `cmd/orch/daemon.go:378-422`, the daemon loop includes completion processing:
```go
completionConfig := daemon.CompletionConfig{...}
completionResult, err := d.CompletionOnce(completionConfig)
// ... logs completions but doesn't release pool slots
```

The slot structure already tracks BeadsID:
```go
type Slot struct {
    ID         int
    AcquiredAt time.Time
    BeadsID    string // Optional - for tracking which issue is in this slot
}
```

But when an agent is auto-completed (beads issue closed), the corresponding slot is NOT explicitly released. The code relies on the next reconciliation cycle to discover the closed status and free the slot - but that discovery depends on the beads lookup succeeding.

**Source:**
- `cmd/orch/daemon.go:378-422` - completion processing loop
- `pkg/daemon/pool.go:22-26` - Slot struct with BeadsID

**Significance:** This reveals the architectural gap: the daemon has two independent mechanisms (completion processing and reconciliation) that don't communicate. Completion processing knows an agent finished but doesn't free the slot; reconciliation can free slots but depends on beads lookups that can fail.

---

### Finding 4: The capacity counting architecture has circular dependency

**Evidence:** The flow for capacity recovery:
```
1. Agent completes → beads issue closed
2. Daemon reconciliation runs
3. DefaultActiveCount() queries OpenCode for sessions
4. GetClosedIssuesBatchWithProjectDirs() checks which are closed via beads
5. If beads lookup fails → session counted as active → actualCount inflated
6. Pool.Reconcile(actualCount) → no slots freed (actualCount == poolCount)
```

The system depends on beads to tell it when agents are done, but:
- Beads daemon can be down
- RPC connection can timeout
- Project directory resolution can fail
- Any of these causes capacity to get stuck

**Source:** Analysis of flow through `pkg/daemon/active_count.go`, `pkg/daemon/pool.go`, `cmd/orch/daemon.go`

**Significance:** This is a fundamental architectural issue. The capacity system depends on an external lookup that can fail, with no fallback mechanism. The fix must either eliminate this dependency or handle failures gracefully.

---

## Synthesis

**Key Insights:**

1. **Error handling is the gap, not project resolution** - The prior fixes progressively improved project resolution (cross-project → session.Directory), but none addressed what happens when the resolved lookup still fails. This is why the bug keeps recurring in different forms.

2. **Passive reconciliation + unreliable lookup = stuck counter** - The current architecture relies on reconciliation to discover completed agents via beads lookups. When lookups fail, the counter can't recover. The fix needs an active mechanism that doesn't depend on successful lookups.

3. **Completion processing is the missing link** - The daemon already knows when agents complete (via `CompletionOnce`). This is the reliable signal that should trigger slot release, not the passive beads lookup during reconciliation.

**Answer to Investigation Question:**

The daemon capacity counter gets stuck despite prior fixes because all fixes addressed project resolution (getting to the right beads database) but none addressed error handling (what happens when the lookup fails). When `client.Show()` or `FallbackShowWithDir()` return errors for any reason, the issue is treated as "open" and counted toward active capacity. This inflates `actualCount` in `DefaultActiveCount()`, which prevents `Pool.Reconcile()` from freeing slots.

The systemic fix requires two changes:
1. **Primary:** Release slots immediately when daemon auto-completes agents (using the BeadsID already tracked in slots)
2. **Secondary:** Make beads lookup errors non-fatal for reconciliation (don't inflate count on lookup failure)

---

## Structured Uncertainty

**What's tested:**

- ✅ Code analysis confirms error handling treats lookup failure as "open" (verified: read active_count.go:217-248)
- ✅ Prior investigations confirm they focused on project resolution not error handling (verified: read all 3 investigations)
- ✅ Slot struct already has BeadsID field for lookup (verified: read pool.go:22-26)
- ✅ Completion processing loop exists but doesn't release slots (verified: read daemon.go:378-422)

**What's untested:**

- ⚠️ Actual frequency of beads lookup failures in production (no metrics exist)
- ⚠️ Whether explicit slot release during completion would fix the symptom (needs implementation)
- ⚠️ Performance impact of tracking slot-to-beadsID mapping (likely minimal but untested)

**What would change this:**

- Finding would be wrong if beads lookups never fail in production (unlikely given observed symptoms)
- Finding would be wrong if there's another code path that inflates activeCount (possible but not found)
- Finding would be wrong if completion processing doesn't reliably detect completions (would need investigation)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Two-part fix: Active slot release + Defensive reconciliation** - Release slots immediately when daemon processes completions, and make beads lookup errors non-fatal for reconciliation.

**Why this approach:**
- Addresses root cause (error handling) not just symptoms (project resolution)
- Uses existing infrastructure (BeadsID already tracked in slots, completion processing exists)
- Provides two layers of defense: active release + passive reconciliation as fallback
- Simple implementation with clear responsibility boundaries

**Trade-offs accepted:**
- Adds coupling between completion processing and pool management
- Requires slot lookup by BeadsID (new method on WorkerPool)
- Reconciliation may free slots slightly later if completion processing misses something

**Implementation sequence:**
1. Add `Pool.ReleaseByBeadsID(beadsID string) bool` method - finds and releases slot by beads ID
2. Update completion processing in daemon loop - call ReleaseByBeadsID after successful auto-complete
3. Update `getClosedIssuesForProject()` - log beads lookup errors instead of silently ignoring
4. Add metrics/logging for lookup failure rate to detect degradation

### Alternative Approaches Considered

**Option B: Trust session age instead of beads status**
- **Pros:** Eliminates dependency on beads lookups
- **Cons:** May prematurely release slots for long-running agents; loses visibility into actual completion
- **When to use instead:** If beads becomes fundamentally unreliable

**Option C: Only count sessions with confirmed open status**
- **Pros:** Lookup failure → don't count → slots freed naturally
- **Cons:** May allow over-spawning if many lookups fail simultaneously
- **When to use instead:** If false positives (stuck counter) are worse than false negatives (over-spawning)

**Option D: Use OpenCode session status directly**
- **Pros:** Doesn't depend on beads at all
- **Cons:** OpenCode "busy/idle" doesn't directly map to "agent complete"; loses beads integration
- **When to use instead:** If beads is deprecated or fundamentally redesigned

**Rationale for recommendation:** Option A addresses the root cause (error handling) while preserving the existing architecture's benefits (beads integration, accurate capacity tracking). It adds defense in depth without major refactoring.

---

### Implementation Details

**What to implement first:**
1. `Pool.ReleaseByBeadsID(beadsID string) bool` in `pkg/daemon/pool.go`
   - Iterate over slots, find matching BeadsID, release it
   - Return true if found and released, false if not found
2. Update daemon loop completion processing in `cmd/orch/daemon.go`
   - After successful `d.CompletionOnce()`, call `d.Pool.ReleaseByBeadsID(cr.BeadsID)`
   - Log when slot released via completion vs reconciliation (for debugging)

**Things to watch out for:**
- ⚠️ BeadsID might not be set on all slots (e.g., manual spawns) - ReleaseByBeadsID should handle gracefully
- ⚠️ Race condition between completion processing and reconciliation - both may try to release same slot
- ⚠️ CompletionOnce returns multiple results - iterate and release each

**Areas needing further investigation:**
- Metrics for beads lookup failure rate (to quantify the problem)
- Whether completion processing is 100% reliable (may need SSE fallback)
- Long-running sessions that exceed 30-min idle threshold before completion

**Success criteria:**
- ✅ Daemon capacity reflects actual running agents within 2 poll cycles
- ✅ No manual daemon restart required when agents complete
- ✅ Beads lookup errors are logged (visible in daemon output)
- ✅ Test case: spawn 3 agents, all complete, capacity returns to 0/3

---

## References

**Files Examined:**
- `pkg/daemon/active_count.go` - DefaultActiveCount, GetClosedIssuesBatchWithProjectDirs, getClosedIssuesForProject
- `pkg/daemon/daemon.go` - Daemon struct, ReconcileWithOpenCode, Once methods
- `pkg/daemon/pool.go` - WorkerPool, Slot, Reconcile
- `cmd/orch/daemon.go` - daemon run loop, completion processing
- `pkg/beads/client.go` - RPC client, Fallback functions

**Commands Run:**
```bash
# Create investigation file
kb create investigation daemon-capacity-counter-stuck-recurring

# Report investigation path
bd comment orch-go-ngsok "investigation_path: ..."
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-26-inv-daemon-capacity-count-goes-stale.md` - Original reconciliation fix
- **Investigation:** `.kb/investigations/2026-01-22-inv-daemon-capacity-tracking-stale-after.md` - Cross-project fix
- **Investigation:** `.kb/investigations/2026-01-22-inv-fix-daemon-capacity-counter-getting.md` - Session.Directory fix
- **Guide:** `.kb/guides/daemon.md` - Daemon operational guide

---

## Investigation History

**2026-01-23 14:00:** Investigation started
- Initial question: Why does daemon capacity get stuck despite 3 prior fixes?
- Context: User reported 3/3 active when orch status shows 0; requires daemon restart

**2026-01-23 14:30:** Root cause identified
- Error handling in getClosedIssuesForProject treats lookup failures as "open"
- Prior fixes addressed project resolution, not error handling

**2026-01-23 15:00:** Investigation completed
- Status: Complete
- Key outcome: Two-part fix needed - active slot release on completion + defensive reconciliation
