# Session Synthesis

**Agent:** og-inv-daemon-logs-spawn-25mar-f768
**Issue:** orch-go-iwp5d
**Outcome:** success

---

## Plain-Language Summary

The daemon was logging "Spawned" and marking issues as in_progress based solely on `orch work` exiting with code 0, without checking if a workspace was actually created. This meant silent failures in workspace setup produced "phantom agents" — issues stuck in_progress with no actual agent running. The fix adds a post-spawn workspace verification step: after `SpawnWork` returns success, it checks that a workspace directory exists for the beads ID. If not, it rolls back the issue to open status and reports the failure.

## Verification Contract

See `VERIFICATION_SPEC.yaml` — 10 new tests covering phantom spawn detection, normal spawn, backward compatibility, pool slot release, and workspace existence checking.

---

## TLDR

Added post-spawn workspace verification to the daemon's `spawnIssue` pipeline. When `orch work` exits 0 but no workspace directory is created, the daemon now detects the phantom spawn, rolls back the issue to open, unmarks the spawn tracker, and releases the pool slot — instead of logging "Spawned" and creating a phantom agent.

---

## Delta (What Changed)

### Files Created
- `pkg/daemon/workspace_verify.go` - `workspaceExistsForIssue()` function that scans `.orch/workspace/` for a workspace matching a beads ID via SPAWN_CONTEXT.md content
- `pkg/daemon/workspace_verify_test.go` - 6 unit tests for workspace existence checking

### Files Modified
- `pkg/daemon/interfaces.go` - Added `WorkspaceVerifier` interface and `defaultWorkspaceVerifier` implementation
- `pkg/daemon/daemon.go` - Added `WorkspaceVerifier` field to `Daemon` struct, wired default in both init sites
- `pkg/daemon/spawn_execution.go` - Added post-spawn workspace verification block after `SpawnWork` success
- `pkg/daemon/mock_test.go` - Added `mockWorkspaceVerifier` for test mocking
- `pkg/daemon/spawn_failure_test.go` - Added 4 tests for phantom spawn detection

---

## Evidence (What Was Observed)

- `spawnIssue` returned `Processed: true` after `SpawnWork` nil error with no workspace check (spawn_execution.go:120-199)
- Workspace names use random 4-char hex suffixes, not beads IDs (spawn/config.go:417) — verification must check SPAWN_CONTEXT.md content
- Existing spawn failure rollback (line 156) uses `UpdateBeadsStatusForProject` directly, bypassing mock. New phantom rollback uses resolved `statusUpdater` interface for consistency and testability.

### Tests Run
```bash
go test ./pkg/daemon/ -run "TestWorkspaceExists|TestSpawnIssue_Phantom|TestSpawnIssue_WorkspaceExists|TestSpawnIssue_NoWorkspaceVerifier" -v
# 10 tests, all PASS (1.1s)

go test ./pkg/daemon/ -count=1
# PASS (19.8s) — full daemon suite, no regressions

go build ./...
# No errors
```

---

## Architectural Choices

### WorkspaceVerifier as interface (not inline check)
- **What I chose:** New `WorkspaceVerifier` interface + `defaultWorkspaceVerifier` implementation
- **What I rejected:** Inline filesystem check or importing `pkg/state.FindWorkspaceByBeadsID`
- **Why:** Follows existing daemon interface pattern (Spawner, IssueUpdater, etc.), is testable via mock, avoids new cross-package dependency
- **Risk accepted:** Slight duplication with `state.FindWorkspaceByBeadsID` — both scan workspace dirs

### Rollback via statusUpdater interface (not raw function)
- **What I chose:** `statusUpdater.UpdateStatus(issue.ID, "open")` for phantom spawn rollback
- **What I rejected:** `UpdateBeadsStatusForProject(issue.ID, "open", statusProjectDir)` (used by existing spawn failure rollback)
- **Why:** Uses same resolved updater as initial in_progress update, works with mocks in tests
- **Risk accepted:** Minor inconsistency with pre-existing spawn failure rollback pattern

---

## Knowledge (What Was Learned)

### Decisions Made
- Workspace verification checks SPAWN_CONTEXT.md content, not directory names (beads ID not in dir name)
- WorkspaceVerifier is nil-safe: when nil, check is skipped (backward compatible)

### Constraints Discovered
- `StatusUpdater` interface is only used for the initial in_progress update; rollback uses raw `UpdateBeadsStatusForProject` — pre-existing inconsistency

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (10 new, full suite green)
- [x] Investigation file has Phase: Complete
- [x] Ready for `orch complete orch-go-iwp5d`

---

## Unexplored Questions

- The original led-totem-toppers-916 incident involved a `bd show` failure during dependency check. Whether the phantom spawn was caused by that specific failure or a separate workspace creation failure is unclear from logs alone.
- The pre-existing spawn failure rollback (line 156) using raw `UpdateBeadsStatusForProject` instead of the interface could cause test brittleness.

---

## Friction

No friction — smooth session

---

## Session Metadata

**Skill:** investigation
**Workspace:** `.orch/workspace/og-inv-daemon-logs-spawn-25mar-f768/`
**Investigation:** `.kb/investigations/2026-03-25-inv-daemon-phantom-spawn-verification.md`
**Beads:** `bd show orch-go-iwp5d`
