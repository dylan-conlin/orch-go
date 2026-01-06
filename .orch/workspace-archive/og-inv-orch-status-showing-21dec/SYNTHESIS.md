# Session Synthesis

**Agent:** og-inv-orch-status-showing-21dec
**Issue:** orch-go-26lo
**Duration:** 2025-12-21 → 2025-12-21
**Outcome:** success

---

## TLDR

`orch status` was showing 27 "active" agents when only 18 were actually running. Fixed by changing the OpenCode API call to return only in-memory sessions (not historical disk sessions) and using tmux windows as the primary source of truth.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/main.go` - Rewrote `runStatus()` function (lines 1513-1593)
  - Changed `ListSessions(projectDir)` to `ListSessions("")` to get in-memory sessions only
  - Made tmux windows the primary source of truth for active agents
  - Added 30-minute idle filter for OpenCode sessions to catch cached-but-completed agents
  - Added deduplication logic to avoid showing same agent from both sources

### Files Created
- `.kb/investigations/2025-12-21-inv-orch-status-showing-stale-sessions.md` - Investigation file

### Commits
- (pending commit)

---

## Evidence (What Was Observed)

- Without `x-opencode-directory` header: API returns 4 in-memory sessions
- With `x-opencode-directory` header: API returns 289 historical disk sessions
- All sessions have `status: null` - OpenCode doesn't persist status
- OpenCode keeps sessions in memory for 6+ hours after agents exit
- After fix: `orch status` shows 18 agents (matches actual tmux windows)

### Tests Run
```bash
# Verify API behavior
curl -s http://127.0.0.1:4096/session | jq 'length'
# Result: 4

curl -s -H "x-opencode-directory: $(pwd)" http://127.0.0.1:4096/session | jq 'length'
# Result: 289

# Run test suite
go test ./... 
# PASS: all tests passing

# Verify fix
./build/orch status
# Active: 18 (down from 27)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-21-inv-orch-status-showing-stale-sessions.md` - Root cause analysis and fix

### Decisions Made
- Use tmux windows as primary source of truth for "active" agents because they definitively indicate a running process
- Use 30-minute idle threshold for OpenCode sessions because in-memory sessions can linger after completion

### Constraints Discovered
- OpenCode `x-opencode-directory` header changes API behavior dramatically: without it, returns in-memory only; with it, returns ALL historical sessions
- Session `status` field is always `null` in API - status only available via SSE during active execution

### Externalized via `kn`
- `kn constrain "OpenCode x-opencode-directory header returns ALL disk sessions, not just matching ones" --reason "API behavior is counterintuitive - without header returns in-memory only"` - Created kn-95029d

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-26lo`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Could SSE monitoring be integrated into status command to show real-time session status?
- Should headless agents have a different display in status (they don't have tmux windows)?

**Areas worth exploring further:**
- Alternative liveness detection for headless agents (heartbeat via API?)

**What remains unclear:**
- Whether 30-minute idle threshold is optimal - might need tuning based on usage patterns

---

## Session Metadata

**Skill:** investigation
**Model:** Claude
**Workspace:** `.orch/workspace/og-inv-orch-status-showing-21dec/`
**Investigation:** `.kb/investigations/2025-12-21-inv-orch-status-showing-stale-sessions.md`
**Beads:** `bd show orch-go-26lo`
