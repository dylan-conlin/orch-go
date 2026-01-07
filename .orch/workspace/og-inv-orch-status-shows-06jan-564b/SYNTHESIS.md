# Session Synthesis

**Agent:** og-inv-orch-status-shows-06jan-564b
**Issue:** orch-go-ij1pl
**Duration:** 2026-01-06 21:30 → 2026-01-06 22:15
**Outcome:** success

---

## TLDR

Investigated why `orch status --all` shows completed agents after `orch complete` runs. Root cause: `orch complete` does not delete the OpenCode session, causing stale agents to appear in status for up to 30 minutes. Fix is to add session deletion (copy pattern from `orch abandon`).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-06-inv-orch-status-shows-completed-agents.md` - Full investigation with findings, recommendations, and implementation guidance

### Files Modified
- None (investigation only)

### Commits
- `4089b26c` - investigation: orch-status-shows-completed-agents - checkpoint

---

## Evidence (What Was Observed)

- `complete_cmd.go` does NOT contain any `DeleteSession` calls (verified via `rg -n "DeleteSession"`)
- `abandon_cmd.go:165-174` correctly deletes OpenCode sessions (reference implementation)
- 136 OpenCode sessions exist with orch-go beads IDs (`curl localhost:4096/session | jq ...`)
- `orch-go-wrrks` beads issue is closed but session persisted until recent cleanup
- 30-minute idle window in status_cmd.go filters sessions for matching (`maxIdleTime = 30 * time.Minute`)

### Tests Run
```bash
# Count persisted sessions
curl -s localhost:4096/session | jq '[.[] | select(.title | test("orch-go-"))] | length'
# Result: 136

# Search for DeleteSession in complete_cmd.go
rg -n "DeleteSession" cmd/orch/complete_cmd.go
# Result: No matches

# Search for DeleteSession in abandon_cmd.go
rg -n "DeleteSession" cmd/orch/abandon_cmd.go
# Result: Line 169: if err := client.DeleteSession(sessionID); err != nil {
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-06-inv-orch-status-shows-completed-agents.md` - Investigation with root cause analysis and fix recommendation

### Decisions Made
- Fix approach: Add session deletion to `complete_cmd.go` (copy pattern from `abandon_cmd.go`)
- Reason: Direct root cause fix, proven pattern, minimal code change

### Constraints Discovered
- OpenCode sessions persist to disk independently of agent lifecycle
- `orch complete` cleans tmux but not OpenCode (asymmetry in cleanup)
- 30-minute idle window means stale state can persist for a while

### Externalized via `kn`
- None required (investigation captures the knowledge, fix is straightforward)

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Add OpenCode session deletion to orch complete
**Skill:** feature-impl
**Context:**
```
orch complete does not delete OpenCode sessions after closing beads issue.
Copy session deletion logic from abandon_cmd.go:165-174 to complete_cmd.go after tmux cleanup (around line 537).
Investigation: .kb/investigations/2026-01-06-inv-orch-status-shows-completed-agents.md
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Why was session deletion omitted from `complete` originally? (Intentional or oversight?)
- Should there be a session "archive" option for post-mortem analysis?

**Areas worth exploring further:**
- Could `orch clean --stale` be integrated into `orch complete` for automatic cleanup?

**What remains unclear:**
- Whether any workflows rely on session persistence after completion (unlikely but unverified)

---

## Session Metadata

**Skill:** investigation
**Model:** Claude Opus
**Workspace:** `.orch/workspace/og-inv-orch-status-shows-06jan-564b/`
**Investigation:** `.kb/investigations/2026-01-06-inv-orch-status-shows-completed-agents.md`
**Beads:** `bd show orch-go-ij1pl`
