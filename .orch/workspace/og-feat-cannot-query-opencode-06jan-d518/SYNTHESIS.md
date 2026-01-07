# Session Synthesis

**Agent:** og-feat-cannot-query-opencode-06jan-d518
**Issue:** orch-go-6g2mf
**Duration:** 2026-01-06 17:32 → 2026-01-06 18:10
**Outcome:** success

---

## TLDR

Investigated why OpenCode sessions from other projects can't be queried. Found that servers share session storage (not project-scoped) but cross-project spawns incorrectly set session directory to orchestrator's project instead of target project.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-06-inv-cannot-query-opencode-sessions-other.md` - Full investigation with findings

### Files Modified
- None (investigation only)

### Commits
- None yet (investigation file ready for commit)

---

## Evidence (What Was Observed)

- Two OpenCode servers (ports 4096 and 55450) return identical 167 sessions
- pw-* sessions (price-watch work) show `directory: /Users/dylanconlin/Documents/personal/orch-go` instead of price-watch path
- Session storage is centralized, not per-server or per-project
- `x-opencode-directory` header filters results but doesn't isolate storage
- Cross-project workspaces are created in orchestrator's `.orch/workspace/` (by design) but sessions have wrong directory metadata

### Tests Run
```bash
# Verify both servers return same sessions
curl -s http://localhost:4096/session | jq length   # 167
curl -s http://localhost:55450/session | jq length  # 167

# Check session directories are wrong
curl -s http://localhost:4096/session | jq '.[] | select(.title | contains("pw-")) | .directory'
# All return "/Users/dylanconlin/Documents/personal/orch-go" instead of price-watch
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-06-inv-cannot-query-opencode-sessions-other.md` - Complete investigation with root cause analysis

### Decisions Made
- Root cause is spawn configuration, not OpenCode architecture
- Fix should be in `orch spawn --workdir` directory propagation

### Constraints Discovered
- OpenCode sessions share central storage - directory header is for filtering, not isolation
- Session directory is set at creation time and determines queryability
- Workspace location and session directory are independent concerns

### Externalized via `kn`
- `kn tried "hypothesis: servers are project-scoped" --failed "Both servers return identical session lists - storage is centralized"` - Key finding that changed approach

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Fix cross-project spawn directory metadata
**Skill:** feature-impl
**Context:**
```
orch spawn --workdir creates sessions with orchestrator's directory instead of target 
directory. Need to trace directory propagation in spawn_cmd.go runSpawnHeadless() and 
ensure CreateSession/opencode run gets correct --dir parameter. See investigation at
.kb/investigations/2026-01-06-inv-cannot-query-opencode-sessions-other.md for root cause.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How does OpenCode TUI determine directory vs CLI spawn?
- Is there a session update/migration API to fix existing sessions?
- Should `orch sessions` command be added for cross-project visibility?

**Areas worth exploring further:**
- The relationship between `cmd.Dir` and session directory registration
- Whether `opencode run --dir` explicitly is supported

**What remains unclear:**
- Why the second OpenCode server on port 55450 exists (started Dec 28, older)
- Whether this affects only headless spawns or all spawn modes

---

## Session Metadata

**Skill:** feature-impl (investigation phase)
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-feat-cannot-query-opencode-06jan-d518/`
**Investigation:** `.kb/investigations/2026-01-06-inv-cannot-query-opencode-sessions-other.md`
**Beads:** `bd show orch-go-6g2mf`
