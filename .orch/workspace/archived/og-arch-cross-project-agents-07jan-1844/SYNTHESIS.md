# Session Synthesis

**Agent:** og-arch-cross-project-agents-07jan-1844
**Issue:** orch-go-j5h4w
**Duration:** 2026-01-07 18:44 → 2026-01-07 19:45
**Outcome:** success

---

## TLDR

Cross-project agents show wrong project_dir because OpenCode `--attach` mode creates sessions with the server's cwd (orch-go), not the CLI's cwd. The fix is to use `kb projects` as an additional source of project directories for workspace scanning, ensuring all registered projects are indexed regardless of OpenCode session state.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-07-inv-cross-project-agents-show-wrong.md` - Design investigation with root cause analysis and recommendation

### Files Modified
- `.orch/features.json` - Added feat-041 for kb projects integration

### Commits
- None yet (investigation only, implementation pending)

---

## Evidence (What Was Observed)

- All 248 OpenCode sessions have `directory="/Users/dylanconlin/Documents/personal/orch-go"` regardless of `--workdir` spawn flag (verified: `curl -s http://localhost:4096/session | jq '[.[] | .directory] | unique'`)
- `opencode run --attach` mode connects to server which determines session directory from its own cwd, not CLI cwd
- `cmd.Dir = cfg.ProjectDir` (spawn_cmd.go:1433) sets CLI's working directory but this is ignored by the server
- `kb projects list` returns 17 registered projects with full paths - a reliable alternative source
- SPAWN_CONTEXT.md correctly contains PROJECT_DIR but the workspace is never scanned because its parent directory isn't in the session directory list

### Tests Run
```bash
# Verified all sessions point to orch-go
curl -s http://localhost:4096/session | jq '[.[] | .directory] | unique'
# Result: ["/Users/dylanconlin/Documents/personal/orch-go"]

# Verified kb projects provides alternative source
kb projects list
# Result: 17 registered projects including price-watch
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-07-inv-cross-project-agents-show-wrong.md` - Complete design investigation with 3 options evaluated

### Decisions Made
- Use kb projects as additional source of project directories (Option C recommended)
- Reason: kb projects is explicitly user-managed, captures orchestration intent, and already exists

### Constraints Discovered
- OpenCode `--attach` mode is architecturally server-controlled - session directory cannot be overridden from CLI
- Workspace cache relies on knowing directories before scanning - chicken-and-egg problem without external source

### Externalized via `kn`
- None (investigation findings captured in .kb/investigations/)

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** feat-041: Use kb projects as source for multi-project workspace scanning
**Skill:** feature-impl
**Context:**
```
Implementation needed to fix cross-project agent visibility. Root cause: OpenCode --attach mode
uses server's cwd for session.directory. Solution: Add getKBProjects() to extract registered
projects from `kb projects list`, merge into extractUniqueProjectDirs(), add graceful fallback.
See .kb/investigations/2026-01-07-inv-cross-project-agents-show-wrong.md for full design.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Could OpenCode add a `--directory` flag to `run --attach` mode? (upstream change)
- Should kb projects have a Go library to avoid CLI invocation overhead?

**Areas worth exploring further:**
- Optimal caching strategy when scanning 17+ project directories
- Error handling when kb CLI is not in PATH in server context

**What remains unclear:**
- Performance impact of scanning all 17 project workspaces

---

## Session Metadata

**Skill:** architect
**Model:** opus (default)
**Workspace:** `.orch/workspace/og-arch-cross-project-agents-07jan-1844/`
**Investigation:** `.kb/investigations/2026-01-07-inv-cross-project-agents-show-wrong.md`
**Beads:** `bd show orch-go-j5h4w`
