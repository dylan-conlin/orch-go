# Session Synthesis

**Agent:** og-debug-web-ui-shows-23dec
**Issue:** orch-go-yhag
**Duration:** 2025-12-23T09:46 → 2025-12-23T09:52 (~6 minutes)
**Outcome:** success

---

## TLDR

Fixed web UI showing 0 agents despite 415+ active OpenCode sessions. Root cause: serve process ran with CWD=/, causing directory filtering to exclude all sessions. Solution: removed directory filter from /api/agents endpoint.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/serve.go` - Removed directory filtering from handleAgents, changed ListSessions("") to get all sessions

### Commits
- `427b708` - fix: remove directory filtering from /api/agents endpoint

---

## Evidence (What Was Observed)

- OpenCode API had 415 sessions total (cmd/orch/serve.go:154)
- /api/agents endpoint returned `null` before fix
- serve process CWD was `/` (verified via lsof)
- With directory="/", only 12 sessions returned
- With directory="", all 415 sessions returned
- After fix, /api/agents returns 14 agents (sessions updated in last 6 hours)

### Tests Run
```bash
# Before fix
curl http://127.0.0.1:3333/api/agents
# Result: null

# After fix
curl http://127.0.0.1:3333/api/agents | jq '. | length'
# Result: 14 (sessions updated in last 6 hours, filtered by displayThreshold)
```

---

## Knowledge (What Was Learned)

### Decisions Made
- Decision: Remove directory filtering from /api/agents - The serve process CWD is arbitrary (often `/`), so using os.Getwd() for directory filtering is unreliable. Dashboard should show all sessions across all projects.
- Decision: Initialize agents as empty slice not nil - Go's `var agents []T` creates nil slice which JSON encodes as `null`. Using `agents := []T{}` ensures empty array `[]` is returned.

### Constraints Discovered
- Constraint: serve process CWD may not match project directory - When serve is started as background process, CWD may be `/` or arbitrary location, making os.Getwd() unreliable for filtering.

### Externalized via `kn`
- None required for this straightforward bug fix

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (API returns agents)
- [x] Smoke test performed (screenshot taken, API verified)
- [x] Ready for `orch complete orch-go-yhag`

---

## Unexplored Questions

**What remains unclear:**
- Should dashboard show ALL sessions or only those for current project? Current fix shows all sessions, which seems correct for a global dashboard, but this design choice wasn't explicitly validated.
- Is the 6-hour displayThreshold the right value? Could be configurable in future.

*(These are minor design questions not critical to the fix)*

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4
**Workspace:** `.orch/workspace/og-debug-web-ui-shows-23dec/`
**Beads:** `bd show orch-go-yhag`
