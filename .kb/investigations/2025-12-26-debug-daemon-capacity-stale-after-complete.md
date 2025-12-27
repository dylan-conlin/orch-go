<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `DefaultActiveCount()` counted all OpenCode sessions updated in last 30 min, regardless of whether their beads issue was closed; completed agents still counted toward capacity.

**Evidence:** Test showed 7 recent sessions, but only 2 with open issues; daemon stuck at 3/3 while orch status showed 0 active agents. After fix, reconciliation correctly freed slots.

**Knowledge:** OpenCode sessions persist after agent completion; checking beads issue status is required to know if an agent is actually running.

**Next:** Fix implemented and verified - `DefaultActiveCount()` now queries beads to exclude sessions with closed issues.

---

# Investigation: Daemon Capacity Stale After Complete

**Question:** Why does daemon capacity show 3/3 active when orch status shows 0 active agents, even after `orch complete` closes all agents?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Supersedes:** .kb/investigations/2025-12-26-inv-daemon-capacity-count-goes-stale.md (partial fix that missed closed issue checking)

---

## Findings

### Finding 1: DefaultActiveCount() didn't check beads issue status

**Evidence:** 
- Original `DefaultActiveCount()` in pkg/daemon/daemon.go:418-474 only filtered by:
  1. Session update time (30 min recency)
  2. Untracked agents (beads ID contains "-untracked-")
- Did NOT check if the beads issue was closed
- When agents complete via `orch complete`, beads issue is closed but OpenCode session persists

**Source:** pkg/daemon/daemon.go:418-474 (before fix)

**Significance:** Root cause of stale capacity. Completed agents continued to count toward daemon capacity because their sessions were "recently updated" even though work was done.

---

### Finding 2: OpenCode sessions persist after agent completion

**Evidence:**
- `curl http://127.0.0.1:4096/session | jq 'length'` returned 61+ sessions
- Many sessions were from days ago (Dec 20, 22, etc.)
- Closed agents showed recent "updated" timestamps from their final output or `/exit` command
- Test showed 7 "recent" sessions but only 2 with open beads issues

**Source:** OpenCode /session API, manual testing

**Significance:** The 30-min recency filter wasn't sufficient. OpenCode updates session timestamps when agents produce output, even final output before completion.

---

### Finding 3: Pool.Reconcile() works correctly but needed accurate actual count

**Evidence:**
- `Pool.Reconcile(actualCount)` at pkg/daemon/pool.go:224-253
- Returns 0 if `actualCount >= p.activeCount`
- Was receiving inflated counts (7 instead of 2) because DefaultActiveCount included closed agents
- After fix, reconciliation correctly freed slots: pool went from 3 → 2 when 1 agent completed

**Source:** pkg/daemon/pool.go:224-253, daemon logs showing "Spawned 1 this cycle"

**Significance:** The reconciliation mechanism was sound; only the input was wrong.

---

## Synthesis

**Key Insights:**

1. **Session lifetime ≠ agent lifetime** - OpenCode sessions persist indefinitely; beads issue status is the authoritative source of whether an agent is running.

2. **Prior fixes were incremental** - Previous investigations (orch-go-59m3, orch-go-s2j7) added recency filtering and untracked filtering, but missed the fundamental issue that completed agents still have recently-updated sessions.

3. **Batch lookup for performance** - Added `getClosedIssuesBatch()` to check multiple beads issues in one RPC call, avoiding N+1 query pattern.

**Answer to Investigation Question:**

The daemon capacity stayed at 3/3 because `DefaultActiveCount()` counted all OpenCode sessions updated in the last 30 minutes, regardless of whether their beads issues were closed. When agents complete via `orch complete`, their beads issues are closed but their OpenCode sessions persist with recent timestamps. The fix adds a beads status check to `DefaultActiveCount()` - sessions with closed beads issues are no longer counted toward capacity.

---

## Structured Uncertainty

**What's tested:**

- ✅ Fix correctly excludes closed issues (verified: closed 4 daemon-spawned agents, count dropped from 7 → 2)
- ✅ Daemon reconciliation frees slots after completions (verified: saw "Spawned 1 this cycle" after slots freed)
- ✅ Beads RPC client works for batch status lookup (verified: test script successfully queried issue status)

**What's untested:**

- ⚠️ Behavior when beads daemon is unavailable (falls back to CLI, but not tested under load)
- ⚠️ Performance with many concurrent sessions (unlikely to be an issue at daemon's 60s poll interval)
- ⚠️ Edge case: session exists but beads issue was deleted (would not count, which is correct)

**What would change this:**

- Finding would be wrong if beads issue status isn't authoritative for agent running state
- Fix might need adjustment if OpenCode changes session persistence behavior

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐ (Implemented)

**Query beads status for each session's beads ID and exclude closed issues from count**

**Why this approach:**
- Uses authoritative source (beads) for agent state
- Minimal code change - adds filtering step to existing function
- Self-healing - stale counts correct on next poll cycle

**Trade-offs accepted:**
- Adds beads RPC/CLI dependency to DefaultActiveCount
- Adds latency for batch beads lookup (mitigated by RPC daemon)

**Implementation sequence:**
1. Extract beads IDs from session titles (already done)
2. Batch fetch issue status via beads client
3. Exclude sessions with closed issues from count

### Alternative Approaches Considered

**Option B: Track daemon-spawned session IDs explicitly**
- **Pros:** More precise - only count sessions daemon actually spawned
- **Cons:** Requires changes to spawn path, more invasive
- **When to use instead:** If beads check becomes performance bottleneck

**Option C: Have `orch complete` notify daemon**
- **Pros:** Real-time slot release without polling
- **Cons:** Requires IPC mechanism, more complex architecture
- **When to use instead:** If near-real-time capacity updates become important

**Rationale for recommendation:** Option A is simplest, uses existing infrastructure, and provides correctness at acceptable latency.

---

### Implementation Details

**What was implemented:**

1. `getClosedIssuesBatch(beadsIDs []string) map[string]bool` in pkg/daemon/daemon.go
   - Uses beads RPC client (falls back to CLI)
   - Returns map of closed beads IDs

2. Modified `DefaultActiveCount()` to:
   - Collect beads IDs from recent sessions
   - Query beads for closed status
   - Exclude closed issues from count

3. Tests in pkg/daemon/daemon_test.go:
   - `TestGetClosedIssuesBatch_EmptyInput`
   - `TestGetClosedIssuesBatch_Integration` (integration test)

**Success criteria:**
- ✅ Daemon frees slots after agents complete via `orch complete`
- ✅ `DefaultActiveCount()` returns correct count matching orch status
- ✅ All daemon tests pass

---

## References

**Files Modified:**
- `pkg/daemon/daemon.go:418-550` - Modified DefaultActiveCount, added getClosedIssuesBatch
- `pkg/daemon/daemon_test.go` - Added tests for new function

**Commands Run:**
```bash
# Test active count calculation
go run /tmp/test_active_fixed.go

# Run daemon tests
go test ./pkg/daemon/... -v -count=1

# Install and restart daemon
make install && launchctl kickstart -k gui/$(id -u)/com.orch.daemon
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2025-12-26-inv-daemon-capacity-count-goes-stale.md - Added reconciliation (partial fix)
- **Investigation:** .kb/investigations/2025-12-26-inv-daemon-capacity-count-stuck-while.md - Added recency filter (partial fix)
- **Investigation:** .kb/investigations/2025-12-26-inv-daemon-capacity-count-stale-after.md - Added untracked filter (partial fix)

---

## Investigation History

**2025-12-26 16:47:** Investigation started
- Initial question: Why daemon shows 3/3 active when orch status shows 0?
- Context: Prior fixes didn't fully solve the issue

**2025-12-26 16:52:** Root cause identified
- DefaultActiveCount counts all recent sessions regardless of beads issue status
- OpenCode sessions persist after agent completion

**2025-12-26 17:00:** Fix implemented and verified
- Added beads status check to DefaultActiveCount
- Tested: closing daemon agents correctly freed slots
- Daemon spawned new work after reconciliation

**2025-12-26 17:01:** Investigation completed
- Status: Complete
- Key outcome: DefaultActiveCount now excludes sessions with closed beads issues
