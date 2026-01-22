<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented pre-spawn Phase: Complete check that prevents respawning completed work.

**Evidence:** Added `HasPhaseComplete()` function and integrated into all three spawn paths (`OnceExcluding`, `OnceWithSlot`, `CrossProjectOnceExcluding`). All tests pass.

**Knowledge:** The check queries beads comments (via RPC first, CLI fallback) and looks for "Phase: Complete" case-insensitively.

**Next:** None - implementation complete, ready for review.

**Promote to Decision:** recommend-no (tactical implementation of documented but unimplemented feature)

---

# Investigation: Implement Pre-Spawn Phase Complete Check

**Question:** How to prevent daemon from respawning work that an agent has already completed?

**Started:** 2026-01-22
**Updated:** 2026-01-22
**Owner:** worker
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Phase: Complete check was documented but not implemented

**Evidence:** The `.kb/guides/agent-lifecycle.md:99-106` documents the check, but no code actually implemented it.

**Source:**
- `.kb/investigations/2026-01-22-inv-strategic-audit-daemon-reliability-multiple.md` (Finding 3)
- Code search confirmed no existing Phase: Complete check in daemon spawn flow

**Significance:** This gap allowed agents that reported Phase: Complete but weren't closed (e.g., waiting for orchestrator review) to be respawned when other dedup mechanisms expired.

---

### Finding 2: Three spawn paths need the check

**Evidence:** The daemon has three entry points for spawning:
1. `OnceExcluding()` - main single-project spawn path
2. `OnceWithSlot()` - single project with explicit slot management
3. `CrossProjectOnceExcluding()` - cross-project daemon mode

**Source:** `pkg/daemon/daemon.go`

**Significance:** All three paths needed the check added to ensure comprehensive coverage.

---

### Finding 3: Beads client supports comment retrieval

**Evidence:** The `BeadsClient` interface includes `Comments(id string) ([]Comment, error)` method, implemented in both RPC and CLI clients.

**Source:**
- `pkg/beads/interface.go:27-28`
- `pkg/beads/cli_client.go:173-187`

**Significance:** Enabled implementing the check following existing patterns (RPC first, CLI fallback).

---

## Implementation

### Files Modified

**pkg/daemon/issue_adapter.go:**
- Added `HasPhaseComplete(beadsID string) (bool, error)`
- Added `HasPhaseCompleteForProject(beadsID, projectPath string) (bool, error)` for cross-project mode
- Added `hasPhaseCompleteCLI()` for CLI fallback
- Added `checkCommentsForPhaseComplete()` for the actual string matching (case-insensitive)

**pkg/daemon/daemon.go:**
- Added check in `OnceExcluding()` after session dedup, before pool acquire
- Added check in `OnceWithSlot()` after session dedup, before pool acquire
- Added check in `CrossProjectOnceExcluding()` after session dedup, before pool acquire

**pkg/daemon/issue_adapter_test.go:**
- Added 13 test cases for `checkCommentsForPhaseComplete()`
- Added tests for `HasPhaseComplete()` and `HasPhaseCompleteForProject()` edge cases

### Design Decisions

1. **Check position:** Added after session dedup check, before pool acquire. This ensures we don't acquire a pool slot for work we're going to skip.

2. **Graceful degradation:** If beads query fails (e.g., invalid ID, daemon unavailable), the function returns `false` (not complete) rather than blocking the spawn. This is intentional - we'd rather have a duplicate spawn than completely block the daemon.

3. **Case-insensitive matching:** Uses `strings.ToLower()` to match "Phase: Complete" case-insensitively, handling variations like "phase: complete" or "Phase: COMPLETE".

4. **Project-aware variant:** Cross-project mode uses `HasPhaseCompleteForProject()` to ensure correct beads socket lookup for each project.

---

## Structured Uncertainty

**What's tested:**

- ✅ `checkCommentsForPhaseComplete()` correctly identifies various Phase: Complete formats (13 test cases)
- ✅ Empty beads ID returns false gracefully
- ✅ Invalid beads ID returns false gracefully (doesn't crash)
- ✅ Code compiles and builds successfully

**What's untested:**

- ⚠️ End-to-end respawn prevention in production (would need daemon running against real beads)
- ⚠️ Performance impact of additional beads query per spawn attempt

**What would change this:**

- Finding would be wrong if agents use different Phase: Complete formats than expected
- Implementation might need adjustment if beads comments API changes

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go` - Main daemon spawn logic
- `pkg/daemon/issue_adapter.go` - Beads integration layer
- `pkg/beads/interface.go` - BeadsClient interface
- `pkg/beads/cli_client.go` - CLI client implementation
- `.kb/investigations/2026-01-22-inv-strategic-audit-daemon-reliability-multiple.md` - Source of this task

**Commands Run:**
```bash
# Build verification
go build ./...

# Run Phase Complete tests
go test ./pkg/daemon/... -run "Phase" -v
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-22-inv-strategic-audit-daemon-reliability-multiple.md` - Identified this gap

---

## Investigation History

**2026-01-22 10:00:** Investigation started
- Initial question: How to implement the documented but unimplemented pre-spawn Phase: Complete check
- Context: Spawned from strategic daemon audit findings (orch-go-gr7qq)

**2026-01-22 10:12:** Implementation completed
- Status: Complete
- Key outcome: Added HasPhaseComplete check to all three daemon spawn paths with 13+ unit tests
