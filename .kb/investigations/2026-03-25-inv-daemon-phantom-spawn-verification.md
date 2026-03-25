## Summary (D.E.K.N.)

**Delta:** Daemon logged "Spawned" and marked issues in_progress before verifying workspace creation, allowing phantom agents when orch work exits 0 without creating a workspace.

**Evidence:** Code trace of `spawnIssue` → `SpawnWork` chain confirmed no post-spawn workspace verification existed; 10 new tests verify the fix catches phantom spawns and rolls back correctly.

**Knowledge:** The `SpawnWork` exit code alone is insufficient to confirm a successful spawn — workspace existence is the ground truth for whether an agent is actually running.

**Next:** Fix implemented and tested. Ready for orch complete.

**Authority:** implementation - Bug fix within existing spawn pipeline patterns, no architectural changes.

---

# Investigation: Daemon Phantom Spawn Verification

**Question:** Why does the daemon log "Spawned" and mark issues in_progress when no workspace or tmux window is created?

**Started:** 2026-03-25
**Updated:** 2026-03-25
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation | - | - | - |

---

## Findings

### Finding 1: spawnIssue trusts SpawnWork exit code without verification

**Evidence:** `pkg/daemon/spawn_execution.go` — after `spawner.SpawnWork()` returns nil (line 120), `spawnIssue` immediately returns `Processed: true` (line 193). No check for workspace directory existence.

**Source:** `pkg/daemon/spawn_execution.go:120-199`, `pkg/daemon/issue_adapter.go:433-454`

**Significance:** This is the root cause. `SpawnWork` uses `exec.Command("orch", args...).CombinedOutput()` — it only checks the exit code. If `orch work` exits 0 without creating a workspace (e.g., cross-project path resolution failure, workspace setup error that doesn't propagate to exit code), the daemon has no way to know.

---

### Finding 2: Workspace names don't contain beads IDs

**Evidence:** `pkg/spawn/config.go:417` — workspace names use format `{project-prefix}-{skill-prefix}-{task-slug}-{date}-{4-char-hex}`. The hex suffix is random, not the beads ID.

**Source:** `pkg/spawn/config.go:417`, existing workspaces in `.orch/workspace/`

**Significance:** Workspace verification must check `SPAWN_CONTEXT.md` file content for the beads ID, not the directory name. This informed the design of `workspaceExistsForIssue()`.

---

### Finding 3: Existing rollback paths bypass StatusUpdater interface

**Evidence:** The spawn failure rollback at `spawn_execution.go:156` calls `UpdateBeadsStatusForProject()` directly instead of using the resolved `statusUpdater`. This is inconsistent with the initial status update at line 87 which uses the interface.

**Source:** `pkg/daemon/spawn_execution.go:156` vs `spawn_execution.go:87`

**Significance:** For the phantom spawn rollback, I used `statusUpdater.UpdateStatus()` (the interface) rather than the raw function. This is both correct (uses the same resolved updater as the initial update) and testable (respects mocks in tests). The pre-existing inconsistency in the spawn failure rollback is a minor tech debt item.

---

## Synthesis

**Answer to Investigation Question:**

The daemon logged "Spawned" before verifying workspace creation because `spawnIssue` treated a nil error from `SpawnWork` as proof of success. The fix adds a `WorkspaceVerifier` check after `SpawnWork` returns — if no workspace exists for the beads ID, it rolls back the issue to open status, unmarks the spawn tracker, and releases the pool slot.

---

## Structured Uncertainty

**What's tested:**

- ✅ Phantom spawn detected when workspace doesn't exist (verified: TestSpawnIssue_PhantomSpawn_RollsBackOnMissingWorkspace)
- ✅ Normal spawn proceeds when workspace exists (verified: TestSpawnIssue_WorkspaceExists_ProcessesNormally)
- ✅ Backward compatible when WorkspaceVerifier is nil (verified: TestSpawnIssue_NoWorkspaceVerifier_SkipsCheck)
- ✅ Pool slot released on phantom spawn (verified: TestSpawnIssue_PhantomSpawn_ReleasesPoolSlot)
- ✅ workspaceExistsForIssue finds workspaces by SPAWN_CONTEXT.md content (verified: 6 unit tests)

**What's untested:**

- ⚠️ The original led-totem-toppers-916 failure mode (cross-project spawn with bd show failure) — would need to reproduce the exact JSONL sync timing issue
- ⚠️ Whether cross-project spawns with `--workdir` correctly resolve `verifyDir`

**What would change this:**

- If workspace creation is async (created after `orch work` returns), the check would false-positive. Current code confirms workspace is created synchronously in step 13 of `runSpawnWithSkillInternal`.

---

## References

**Files Examined:**
- `pkg/daemon/spawn_execution.go` - spawnIssue pipeline, rollback logic
- `pkg/daemon/interfaces.go` - Spawner/IssueUpdater interfaces
- `pkg/daemon/issue_adapter.go:433-454` - SpawnWork implementation
- `cmd/orch/work_cmd.go` - orch work command
- `cmd/orch/spawn_cmd.go:221-609` - runSpawnWithSkillInternal pipeline
- `pkg/daemon/daemon.go` - Daemon struct, initialization
- `pkg/state/reconcile.go:179-224` - FindWorkspaceByBeadsID (reference)
- `pkg/spawn/config.go:417` - workspace name generation
