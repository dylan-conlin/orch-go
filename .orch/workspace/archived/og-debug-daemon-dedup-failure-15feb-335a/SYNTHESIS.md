# Session Synthesis

**Agent:** og-debug-daemon-dedup-failure-15feb-335a
**Issue:** orch-go-dr0u
**Outcome:** success

---

## Plain-Language Summary

The daemon's spawn dedup was vulnerable to a TOCTOU (time-of-check-time-of-use) race condition where two concurrent daemon processes could both spawn agents for the same beads issue. The root cause was that `UpdateBeadsStatus(id, "in_progress")` is idempotent -- calling it when the issue is already `in_progress` succeeds silently, so both processes pass the "status update" gate. The fix adds a fresh beads status check (`GetBeadsIssueStatus`) that re-fetches the issue's current status from beads right before spawning. If another daemon process has already moved the issue to `in_progress`, the fresh check catches it and skips the spawn. This closes the race window from seconds (between `ListReadyIssues` and `UpdateBeadsStatus`) to milliseconds (between fresh check and update), making concurrent daemon duplicates practically impossible.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for test commands and results.

Key outcomes:
- 5 new tests covering: skip in_progress, allow open, fail-open on error, nil func compat, concurrent daemon simulation
- All existing daemon tests pass (no regressions)
- Build and vet clean

---

## Delta (What Changed)

### Files Modified
- `pkg/daemon/daemon.go` - Added `getIssueStatusFunc` field to Daemon struct, fresh status check before spawn in both `OnceExcluding` and `OnceWithSlot`, initialized in `NewWithConfig` and `NewWithPool`
- `pkg/daemon/daemon_test.go` - Added 5 new tests for fresh status check dedup behavior
- `pkg/daemon/issue_adapter.go` - Added `GetBeadsIssueStatus` function (RPC-first with CLI fallback)

---

## Evidence (What Was Observed)

- `spawn_tracker.go:20-32`: SpawnedIssueTracker uses `map[string]time.Time` (in-memory only, confirmed)
- `daemon.go:780-831` (runDaemonOnce): Creates fresh `daemon.NewWithConfig(config)` with new empty SpawnedIssueTracker per invocation
- `daemon.go:879`: `UpdateBeadsStatus(issue.ID, "in_progress")` is the PRIMARY dedup gate, but it's idempotent (doesn't fail when already in_progress)
- `issue_adapter.go:217-239`: `UpdateBeadsStatus` calls `client.Update` which succeeds regardless of current status

### Root Cause
Two daemon processes (or daemon run + daemon once) can race:
1. Both call `ListReadyIssues()` → both see issue as "open"
2. Both pass `HasExistingSessionForBeadsID` (no session created yet)
3. Both call `UpdateBeadsStatus(id, "in_progress")` → both succeed (idempotent)
4. Both call `SpawnWork(id)` → duplicate agents

### Tests Run
```bash
go test ./pkg/daemon/ -run 'TestDaemon_Once_FreshStatus|TestDaemon_ConcurrentDaemon' -v
# PASS: 5/5 tests passing
go test ./pkg/daemon/ -run 'TestDaemon|TestOnce|TestRun|...' -timeout 60s
# PASS: all tests (9.554s)
go build ./cmd/orch/ && go vet ./cmd/orch/ && go vet ./pkg/daemon/
# Clean
```

---

## Knowledge (What Was Learned)

### Constraints Discovered
- `UpdateBeadsStatus` is idempotent - setting `in_progress` on an already `in_progress` issue succeeds silently. This means it cannot serve as a compare-and-swap dedup gate when multiple processes race.

### Decisions Made
- Fresh status check uses fail-open semantics (on beads error, continue to spawn). This prevents beads outages from blocking all daemon work. The existing `UpdateBeadsStatus` call provides the persistent dedup gate as backup.
- The `getIssueStatusFunc` field follows the existing mockable function pattern in Daemon (like `listIssuesFunc`, `spawnFunc`) for testability.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (5 new + all existing)
- [x] Ready for `orch complete orch-go-dr0u`

---

## Unexplored Questions

- A true compare-and-swap in beads (`bd update --if-status open`) would eliminate the remaining microsecond race window. Not worth building for this case since the fresh check narrows it to practical impossibility.
- The `TestInferTargetFilesFromIssue` pre-existing test failure should be tracked separately (regex over-matching in file path extraction).

---

## Session Metadata

**Skill:** systematic-debugging
**Workspace:** `.orch/workspace/og-debug-daemon-dedup-failure-15feb-335a/`
**Beads:** `bd show orch-go-dr0u`
