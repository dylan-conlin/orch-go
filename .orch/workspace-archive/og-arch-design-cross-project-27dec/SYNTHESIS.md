# Session Synthesis

**Agent:** og-arch-design-cross-project-27dec
**Issue:** orch-go-y8lp
**Duration:** 2025-12-27T08:00 → 2025-12-27T09:30
**Outcome:** success

---

## TLDR

Designed cross-project completion UX for `orch complete` when agents are spawned in different repos. Recommended auto-detecting project from existing workspace metadata (PROJECT_DIR in SPAWN_CONTEXT.md) with `--workdir` flag as fallback, matching the existing `orch abandon --workdir` pattern.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-27-inv-design-cross-project-completion-ux.md` - Full architect investigation with 5 findings, synthesis, and implementation recommendations

### Files Modified
- `.orch/features.json` - Added feat-020 for cross-project completion implementation

### Commits
- (Investigation and features.json to be committed)

---

## Evidence (What Was Observed)

- SPAWN_CONTEXT.md template contains `PROJECT_DIR: {{.ProjectDir}}` at line 76 of `pkg/spawn/context.go`
- `extractProjectDirFromWorkspace()` already exists in `cmd/orch/review.go:258-279` for extracting this metadata
- `orch abandon` already has `--workdir` flag pattern in `cmd/orch/main.go:654-685`
- Beads socket is per-project at `.beads/bd.sock`, discovered via `beads.FindSocketPath()` in `pkg/beads/client.go:78-106`
- Current `runComplete()` provides helpful error but requires manual `cd` to target project

### Analysis Performed
```bash
# Examined existing code patterns
grep -n "workdir\|WorkDir\|project.*dir" pkg/**/*.go cmd/**/*.go
# Found 97 matches showing established patterns for cross-project operations
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-27-inv-design-cross-project-completion-ux.md` - Architect investigation with design decision

### Decisions Made
- **Use auto-detect from workspace metadata with `--workdir` fallback** because: workspace already stores PROJECT_DIR, pattern matches existing `orch abandon --workdir`, zero friction for happy path, explicit fallback for edge cases

### Constraints Discovered
- Beads issues are per-repo by design (socket at `.beads/bd.sock`) - this is intentional for data isolation, not a bug to fix
- Workspace metadata must exist to auto-detect - cleaned up workspaces require `--workdir` override

### Externalized via `kn`
- (No new kn entries - knowledge externalized to investigation file)

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** feat-020 already created in features.json
**Skill:** feature-impl
**Context:**
```
Implement cross-project completion by adding --workdir flag to orch complete and modifying 
runComplete() to extract PROJECT_DIR from SPAWN_CONTEXT.md. Set beads.DefaultDir before any 
beads operations. Pattern after existing orch abandon --workdir implementation.
```

### File Targets for Implementation
1. `cmd/orch/main.go` - Add `completeWorkdir` flag (like `abandonWorkdir`)
2. `cmd/orch/main.go` - Modify `runComplete()` to call `extractProjectDirFromWorkspace()` and set `beads.DefaultDir`
3. Update error message in line 2965 to suggest `--workdir` option

### Acceptance Criteria
- ✅ `orch complete glass-xyz` from orch-go auto-detects and completes successfully
- ✅ `orch complete unknown-xyz --workdir ~/glass` works as explicit override
- ✅ Error message when issue not found suggests `--workdir` option
- ✅ Existing single-project completion still works unchanged

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should `orch review` also support `--workdir`? (Current implementation uses `extractProjectDirFromWorkspace` which may already work)
- What happens if workspace is in project A but user provides `--workdir` pointing to project B? (Edge case - explicit flag should probably win)

**Areas worth exploring further:**
- Cross-project beads query syntax (`bd show glass:glass-xyz`) as a beads-level solution
- Workspace cleanup timing relative to completion

**What remains unclear:**
- Performance of reading SPAWN_CONTEXT.md for every complete (likely negligible but untested)

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-design-cross-project-27dec/`
**Investigation:** `.kb/investigations/2025-12-27-inv-design-cross-project-completion-ux.md`
**Beads:** `bd show orch-go-y8lp`
