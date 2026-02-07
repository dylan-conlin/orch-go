# Session Synthesis

**Agent:** og-debug-cross-project-agents-07jan-37bd
**Issue:** orch-go-fpg4x
**Duration:** 2026-01-07T16:38 -> 2026-01-07T16:42
**Outcome:** success

---

## TLDR

Fixed cross-project agents showing wrong `project_dir` by adding project directory tracking to the workspace cache. The cache now rebuilds when the set of project directories changes, ensuring kb-registered projects are always scanned for cross-project agent visibility.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/serve_agents_cache.go` - Added `projectDirs` field to `globalWorkspaceCacheType` to track which directories the cache was built with. Added `projectDirsMatch()` helper function. Updated `getCachedWorkspace()` to rebuild cache if project directories change. Updated `invalidate()` to clear projectDirs.

### Commits
- (pending) - Fix cross-project agents showing wrong project_dir by tracking cache projectDirs

---

## Evidence (What Was Observed)

- **Root cause identified**: The workspace cache stored data but did NOT track which `projectDirs` it was built with. If the cache was built before a new project was registered in `kb projects`, subsequent requests would reuse the stale cache even though `extractUniqueProjectDirs` now included more projects.
- Prior investigation correctly identified that kb projects are needed as an alternative source of project directories (since OpenCode `--attach` uses server's cwd)
- The code at `extractUniqueProjectDirs` correctly adds kb projects to the list
- The code at `buildMultiProjectWorkspaceCache` correctly scans all project workspaces
- The issue was that `getCachedWorkspace` only checked TTL, not whether projectDirs had changed

### Tests Run
```bash
go test ./... 
# PASS: all tests passing

curl -sk https://localhost:3348/api/agents | jq '.[] | select(.beads_id == "pw-49eb")'
# project_dir: /Users/dylanconlin/Documents/work/.../price-watch (CORRECT)
```

---

## Knowledge (What Was Learned)

### Decisions Made
- Added `projectDirsMatch()` function that compares sets (order-independent) rather than slices
- Store `projectDirs` in cache metadata to enable staleness detection based on directory changes
- Clear `projectDirs` in `invalidate()` for explicit cache reset behavior

### Constraints Discovered
- Cache invalidation must consider both TTL AND input parameters
- Agents without workspaces in any scanned project will still show wrong project_dir (expected - no SPAWN_CONTEXT.md to read)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Smoke test verified: `pw-49eb` shows correct `project_dir`
- [x] Ready for `orch complete orch-go-fpg4x`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Other cross-project agents (e.g., `pw-9cnq`) still show wrong project_dir - but these don't have workspaces yet (spawned recently, no SPAWN_CONTEXT.md exists)
- Should there be a fallback mechanism for agents without workspaces?

**What remains unclear:**
- Why some agents have workspaces and others don't (likely timing - recently spawned agents haven't created workspaces yet)

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** Claude
**Workspace:** `.orch/workspace/og-debug-cross-project-agents-07jan-37bd/`
**Prior Investigation:** `.kb/investigations/2026-01-07-inv-cross-project-agents-show-wrong.md`
**Beads:** `bd show orch-go-fpg4x`
