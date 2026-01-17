<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `checkConcurrencyLimit()` in spawn_cmd.go did not check if beads issues were closed, causing sessions with closed issues to count toward the concurrency limit.

**Evidence:** Build passed, spawn now works correctly - counts 4 active agents instead of 95 idle ones. Test spawn succeeded with `--max-agents 5`.

**Knowledge:** Concurrency counting must check beads issue status (not just Phase: Complete comments) because issues can be closed via `bd close` without the agent reporting Phase: Complete.

**Next:** Close this issue - fix implemented and verified.

**Promote to Decision:** recommend-no (tactical bug fix, not architectural change)

---

# Investigation: Fix Logic Pkg Registry Spawn

**Question:** Why does `orch spawn` block with "concurrency limit reached" even when 0 agents are in running state?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: `DefaultActiveCount()` checks closed issues, `checkConcurrencyLimit()` does not

**Evidence:**
- `pkg/daemon/active_count.go:81` - Calls `GetClosedIssuesBatch()` to exclude sessions with closed beads issues
- `cmd/orch/spawn_cmd.go:420-482` (before fix) - Only checked `verify.IsPhaseComplete()` for Phase: Complete comments

**Source:**
- `pkg/daemon/active_count.go:14-94` - DefaultActiveCount function
- `cmd/orch/spawn_cmd.go:420-482` - checkConcurrencyLimit function

**Significance:** This discrepancy meant that sessions whose beads issues were closed (but without a "Phase: Complete" comment) would still count toward the concurrency limit. With many legacy sessions, this caused the limit to be reached even with 0 truly running agents.

---

### Finding 2: Two different paths to closing agents

**Evidence:**
1. Agent reports "Phase: Complete" via `bd comment` -> `IsPhaseComplete()` returns true -> excluded from count
2. Issue closed via `bd close` or `orch complete` -> beads status = "closed" -> NOT checked in spawn concurrency

**Source:**
- `pkg/verify/beads_api.go:172-183` - IsPhaseComplete checks for comment
- `pkg/daemon/active_count.go:119-122` - Checks issue.Status == "closed"

**Significance:** The "Phase: Complete" path works for active agents, but many legacy/completed sessions have closed issues without this comment, causing false positives in concurrency counting.

---

### Finding 3: Fix verified with actual spawns

**Evidence:**
- Before fix: "95 idle agents counted as active" (from issue description)
- After fix: Only 4 agents counted as active
- Test spawn with `--max-agents 5` succeeded

**Source:** Test run output showing spawn succeeded after fix

**Significance:** Confirms the fix correctly filters out sessions with closed beads issues from the concurrency count.

---

## Synthesis

**Key Insights:**

1. **Consistency required** - Both `DefaultActiveCount()` (daemon) and `checkConcurrencyLimit()` (spawn) must use the same logic for counting active agents, including checking if beads issues are closed.

2. **Two completion signals** - Agents can be "done" in two ways: Phase: Complete comment OR closed beads issue. The concurrency check must respect both.

3. **Batch processing for efficiency** - Using `GetClosedIssuesBatch()` avoids N+1 queries when checking many sessions.

**Answer to Investigation Question:**

The spawn blocked because `checkConcurrencyLimit()` was counting sessions with closed beads issues as active. These sessions had lingering OpenCode sessions but their corresponding beads issues were already closed. The fix adds a batch check for closed issues using the existing `GetClosedIssuesBatch()` function, matching the behavior of `DefaultActiveCount()`.

---

## Structured Uncertainty

**What's tested:**

- ✅ Code compiles (verified: `go build ./...` succeeded)
- ✅ Daemon tests pass (verified: `go test ./pkg/daemon/...` - all passing)
- ✅ Spawn works after fix (verified: test spawn succeeded with `--max-agents 5`)
- ✅ Active count reduced from 95 to 4 (verified: concurrency check output)

**What's untested:**

- ⚠️ Edge case: beads RPC client unavailable (code falls back to CLI, not tested)
- ⚠️ Edge case: beads issue deleted entirely (assumed not running, not tested)

**What would change this:**

- If OpenCode session timestamps don't update correctly, idle threshold would be wrong
- If beads issue status check has high latency, spawn could be slow

---

## Implementation Recommendations

### Recommended Approach ⭐

**Export and reuse GetClosedIssuesBatch** - Rename the existing function in daemon package and import it in spawn_cmd.go.

**Why this approach:**
- Reuses proven code from DefaultActiveCount
- Single source of truth for closed issue detection
- Maintains batch efficiency

**Trade-offs accepted:**
- Slight code coupling between daemon and spawn_cmd
- Acceptable because this is the same logical operation

**Implementation sequence:**
1. Export `GetClosedIssuesBatch` in `pkg/daemon/active_count.go` (done)
2. Import daemon package in `cmd/orch/spawn_cmd.go` (done)
3. Add closed issue check in `checkConcurrencyLimit` loop (done)

---

## References

**Files Examined:**
- `pkg/daemon/active_count.go` - DefaultActiveCount implementation
- `cmd/orch/spawn_cmd.go` - checkConcurrencyLimit implementation
- `pkg/agent/filters.go` - IsActiveForConcurrency logic
- `pkg/verify/beads_api.go` - IsPhaseComplete implementation

**Commands Run:**
```bash
# Build verification
go build ./...

# Test verification
go test ./pkg/daemon/... -v

# Smoke test
./build/orch spawn --bypass-triage --max-agents 5 --no-track investigation "verify fix"
```

**Related Artifacts:**
- **Decision:** None - tactical bug fix

---

## Investigation History

**2026-01-17 23:00:** Investigation started
- Initial question: Why does spawn block with 95 idle agents?
- Context: Health check identified 95 idle agents counting as active

**2026-01-17 23:01:** Root cause identified
- Found discrepancy between DefaultActiveCount and checkConcurrencyLimit
- DefaultActiveCount checks closed issues, checkConcurrencyLimit doesn't

**2026-01-17 23:02:** Fix implemented
- Exported GetClosedIssuesBatch from daemon package
- Added closed issue check to checkConcurrencyLimit

**2026-01-17 23:03:** Investigation completed
- Status: Complete
- Key outcome: Spawn now correctly counts only truly active agents
