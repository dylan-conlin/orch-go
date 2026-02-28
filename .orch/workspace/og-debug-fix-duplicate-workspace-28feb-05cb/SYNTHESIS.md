# Session Synthesis

**Agent:** og-debug-fix-duplicate-workspace-28feb-05cb
**Issue:** orch-go-qbfv
**Outcome:** success

---

## Plain-Language Summary

When duplicate agent spawns created two workspace directories for the same beads issue ID, `orch complete` (and daemon completion) would pick whichever workspace sorted first alphabetically — often the older, incomplete one. This fix changes `findWorkspaceByBeadsID` and `findWorkspaceForIssue` to collect all matching workspaces and pick the best one: preferring a workspace that has SYNTHESIS.md (indicating completed work), then falling back to the most recently spawned workspace (by `.spawn_time` file). This prevents `orch complete` from verifying against a stale duplicate workspace.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace directory.

---

## TLDR

Fixed duplicate workspace resolution in `orch complete` and daemon completion to prefer workspaces with SYNTHESIS.md over alphabetical ordering when multiple workspaces share the same beads ID.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/shared.go` - `findWorkspaceByBeadsID()` now collects all candidates and ranks by SYNTHESIS.md presence then spawn time, instead of returning first match. Added `workspaceSpawnTime()` helper.
- `pkg/daemon/completion_processing.go` - `findWorkspaceForIssue()` same collect-and-rank fix. Added `pickBestWorkspacePath()` and `readSpawnTime()` helpers.
- `pkg/daemon/issue_adapter_test.go` - Added 6 tests covering duplicate workspace resolution: synthesis preference, newest-when-no-synthesis, synthesis-beats-newer, pickBestWorkspacePath with 3 candidates, and readSpawnTime.

---

## Evidence (What Was Observed)

- Real duplicate workspaces confirmed: `og-feat-add-live-event-27feb-b001` and `og-feat-add-live-event-28feb-b00f` both have beads ID `orch-go-twrn`
- spawn_time values: 1772217838171079000 (27feb) vs 1772300935545981000 (28feb) — old code picked the wrong (older) one alphabetically
- `os.ReadDir` returns entries sorted alphabetically, so `27feb-b001` < `28feb-b00f` — the first match was always the older workspace

### Tests Run
```bash
go test -run "TestFindWorkspaceForIssue|TestPickBestWorkspacePath|TestReadSpawnTime" ./pkg/daemon/ -v
# PASS: 6 tests passing (0.013s)

go test ./pkg/daemon/ -v
# PASS: all daemon tests passing (8.289s), no regressions
```

---

## Architectural Choices

### Collect-and-rank vs early-return
- **What I chose:** Collect all matching candidates, then rank by SYNTHESIS.md > newest spawn time
- **What I rejected:** Short-circuit with weighted directory traversal order
- **Why:** The collect approach is simple, correct, and handles all future ranking criteria changes in one place. Workspace counts are small (<100 active), so scanning all is cheap.
- **Risk accepted:** Slightly more filesystem reads per resolution (checking SYNTHESIS.md and .spawn_time for each candidate), but workspace counts are small.

### Separate helpers per package vs shared utility
- **What I chose:** Duplicate `pickBestWorkspacePath`/`readSpawnTime` in daemon package, `workspaceSpawnTime` in cmd/orch
- **What I rejected:** Extracting a shared `pkg/workspace` utility package
- **Why:** The two functions are small (<30 lines each) and the packages have different candidate representations (struct vs string). Creating a shared package for two small functions is over-engineering.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- `os.ReadDir` returns alphabetically sorted entries — any "return first match" pattern silently becomes "return alphabetically first match", which is rarely the desired behavior when duplicates exist.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (6 new + all existing daemon tests)
- [x] Ready for `orch complete orch-go-qbfv`

---

## Unexplored Questions

- The `serve_agents_cache.go:lookupWorkspace` function (line 698) also does workspace resolution via a cache — it may have the same first-match bug but operates on a different data structure (pre-built cache). Worth checking separately.
- Could add duplicate workspace detection at spawn time to prevent the issue upstream.

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-debug-fix-duplicate-workspace-28feb-05cb/`
**Beads:** `bd show orch-go-qbfv`
