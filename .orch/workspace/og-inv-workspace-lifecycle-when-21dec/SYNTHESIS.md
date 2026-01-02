# Session Synthesis

**Agent:** og-inv-workspace-lifecycle-when-21dec
**Issue:** orch-go-4kwt.1
**Duration:** 2025-12-21 14:01 → 2025-12-21 14:20
**Outcome:** success

---

## TLDR

Investigated workspace lifecycle in orch-go. Workspaces are created at spawn time and persist indefinitely - there is no automatic cleanup. The "clean" command only updates registry status, never removes workspace directories from the filesystem.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-21-inv-workspace-lifecycle-when-workspaces-created.md` - Complete investigation documenting workspace lifecycle

### Files Modified
- None

### Commits
- (to be committed with investigation file)

---

## Evidence (What Was Observed)

- `pkg/spawn/context.go:218-220` - Workspace created via `os.MkdirAll(workspacePath, 0755)`
- `pkg/registry/registry.go:497-512` - `Remove()` only sets status to "deleted", no filesystem operation
- `cmd/orch/main.go:1803-1966` - `runClean()` calls `reg.Remove()` but no `RemoveAll`
- `rg "RemoveAll|remove.*workspace" --type go` - No matches found
- `ls .orch/workspace/ | wc -l` - 150 workspaces accumulated

### Tests Run
```bash
# Verified workspace count
ls .orch/workspace/ | wc -l  # → 150

# Searched for cleanup code
rg "RemoveAll|remove.*workspace" --type go  # → No matches

# Verified my own workspace creation
ls .orch/workspace/og-inv-workspace-lifecycle-when-21dec/  # → SPAWN_CONTEXT.md
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-21-inv-workspace-lifecycle-when-workspaces-created.md` - Complete lifecycle documentation

### Decisions Made
- This behavior appears intentional - workspaces are valuable for post-mortems and synthesis

### Constraints Discovered
- Workspaces accumulate indefinitely - manual cleanup is required if disk space is a concern
- Registry "deleted" state is a soft delete - files remain on disk

### Externalized via `kn`
- None required - straightforward documentation of existing behavior

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Investigation file created with full lifecycle documentation
- [x] Investigation file has complete D.E.K.N. summary
- [x] Ready for `orch complete orch-go-4kwt.1`

---

## Session Metadata

**Skill:** investigation
**Model:** opus
**Workspace:** `.orch/workspace/og-inv-workspace-lifecycle-when-21dec/`
**Investigation:** `.kb/investigations/2025-12-21-inv-workspace-lifecycle-when-workspaces-created.md`
**Beads:** `bd show orch-go-4kwt.1`
