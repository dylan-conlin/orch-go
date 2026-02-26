<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented session-level dedup by checking OpenCode sessions for matching beads IDs before spawn, plus extended TTL from 5min to 6h as backup.

**Evidence:** Tests pass, integration test confirms HasExistingSessionForBeadsID('orch-go-nqgjr') returns true for existing duplicate sessions.

**Knowledge:** Primary dedup via OpenCode session query (checks title for `[beads-id]` pattern); backup via 6-hour TTL on SpawnedIssueTracker. Fail-open design if API unavailable.

**Next:** Deploy and monitor - duplicate spawns for same beads ID should no longer occur.

**Promote to Decision:** recommend-no (tactical fix, but pattern of session-based dedup could become standard)

---

# Investigation: Implement Session Level Dedup Prevent

**Question:** How to prevent daemon from spawning duplicate agents for the same beads issue?

**Started:** 2026-01-15
**Updated:** 2026-01-15
**Owner:** og-feat-implement-session-level agent
**Phase:** Complete
**Next Step:** None - implementation complete, ready for review
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Root cause is TTL expiration while agent still running

**Evidence:** SpawnedIssueTracker had 5-minute TTL, but agents work for hours. After TTL expires, the tracker no longer blocks duplicate spawns. Multiple sessions for orch-go-nqgjr (19+ duplicates) confirm this in production.

**Source:**
- `pkg/daemon/spawn_tracker.go:37` (5-minute TTL)
- `curl http://localhost:4096/session | jq` showed 4+ sessions with same beads ID

**Significance:** The TTL was designed for short race windows, not hours-long agent work. This confirms the fix needs both immediate dedup (session check) and longer backup TTL.

---

### Finding 2: Session titles contain beads ID in `[brackets]` format

**Evidence:** Session titles follow pattern: `"workspace-name [beads-id]"`. Function `extractBeadsIDFromSessionTitle` already exists in `active_count.go` to parse this format.

**Source:**
- `pkg/daemon/active_count.go:143-151` (extractBeadsIDFromSessionTitle)
- `cmd/orch/spawn_cmd.go:1247-1255` (formatSessionTitle)

**Significance:** Can query OpenCode sessions and extract beads IDs to check for existing sessions before spawn.

---

### Finding 3: Implementation complete with two-layer protection

**Evidence:**
1. Created `pkg/daemon/session_dedup.go` with `HasExistingSessionForBeadsID()` function
2. Integrated check into `daemon.OnceExcluding()` at line 740 and `daemon.OnceWithSlot()` at line 845
3. Extended TTL from 5 minutes to 6 hours in `spawn_tracker.go:40`

Tests pass and integration test confirms:
```
HasExistingSessionForBeadsID('orch-go-nqgjr') = true (correctly finds duplicates)
HasExistingSessionForBeadsID('nonexistent-id-xyz') = false (correctly rejects)
```

**Source:**
- `pkg/daemon/session_dedup.go` (new file)
- `pkg/daemon/daemon.go:735-750` (dedup check in OnceExcluding)
- `pkg/daemon/spawn_tracker.go:40` (6-hour TTL)

**Significance:** Two-layer protection: primary (session query) catches running agents, backup (6h TTL) catches edge cases when API unavailable.

---

## Synthesis

**Key Insights:**

1. **OpenCode sessions are the source of truth** - Checking existing sessions is more reliable than tracking spawn timestamps because it reflects actual running agents.

2. **Fail-open design is correct** - If OpenCode API is unavailable, spawn should proceed rather than blocking all work. The 6-hour TTL provides backup protection.

3. **Session title convention enables dedup** - The `[beads-id]` suffix in session titles makes extraction reliable.

**Answer to Investigation Question:**

Daemon duplicate spawns are prevented by:
1. Primary: Query OpenCode sessions for matching beads ID before spawn (6-hour age limit)
2. Backup: Extended SpawnedIssueTracker TTL from 5 minutes to 6 hours

---

## Structured Uncertainty

**What's tested:**

- ✅ HasExistingSessionForBeadsID correctly identifies existing sessions (verified: integration test with real API)
- ✅ Function returns false for non-existent beads IDs (verified: test with 'nonexistent-id-xyz')
- ✅ 6-hour max age correctly filters old sessions (verified: unit test)
- ✅ Fail-open behavior on API error (verified: TestHasExistingSession_ServerError)

**What's untested:**

- ⚠️ Daemon behavior over extended time with new code (needs production monitoring)
- ⚠️ Race conditions with concurrent daemon instances
- ⚠️ Performance impact of additional API call per spawn attempt

**What would change this:**

- Finding would be wrong if session titles don't consistently contain beads IDs
- Solution would need adjustment if OpenCode API frequently unavailable

---

## Implementation Recommendations

**Purpose:** Document the implemented solution.

### Recommended Approach ⭐ (IMPLEMENTED)

**Session-level dedup + extended TTL** - Check OpenCode for existing sessions with same beads ID before spawn, extend TTL as backup.

**Why this approach:**
- Directly addresses root cause: no dedup check before spawn
- Uses source of truth (OpenCode sessions) rather than cached status
- Fail-open design prevents API issues from blocking work
- Extended TTL provides backup when primary check fails

**Trade-offs accepted:**
- Additional API call per spawn attempt (latency)
- Depends on session title format convention

**Implementation sequence:**
1. Created `pkg/daemon/session_dedup.go` with SessionDedupChecker
2. Integrated check into daemon.Once() and OnceWithSlot()
3. Extended TTL from 5 minutes to 6 hours

---

## References

**Files Modified:**
- `pkg/daemon/session_dedup.go` - New file: session dedup checker
- `pkg/daemon/session_dedup_test.go` - New file: tests
- `pkg/daemon/daemon.go:735-750` - Integration into OnceExcluding
- `pkg/daemon/daemon.go:840-855` - Integration into OnceWithSlot
- `pkg/daemon/spawn_tracker.go:37-41` - Extended TTL

**Commands Run:**
```bash
# Verify existing duplicates
curl -s http://localhost:4096/session | jq -r '.[] | select(.title | contains("[")) | {title: .title, created: (.time.created / 1000 | todate)}'

# Run tests
go test ./pkg/daemon/... -run "SessionDedup|HasExistingSession" -v

# Integration test
go run /tmp/test_dedup.go
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-15-inv-investigate-daemon-duplicate-spawn-issue.md` - Root cause analysis
- **Issue:** orch-go-2nruy - This implementation ticket

---

## Investigation History

**2026-01-15 21:55:** Investigation started
- Initial question: How to prevent daemon duplicate spawns?
- Context: orch-go-nqgjr had 19+ duplicate sessions, daemon spawning after 5-min TTL

**2026-01-15 22:00:** Implementation started
- Created session_dedup.go with HasExistingSessionForBeadsID
- Integrated into daemon.Once() and OnceWithSlot()

**2026-01-15 22:05:** Tests written and passing
- Unit tests for HasExistingSession
- Integration test confirms detection of real duplicates

**2026-01-15 22:10:** Investigation completed
- Status: Complete
- Key outcome: Two-layer dedup protection implemented (session query + 6h TTL)
