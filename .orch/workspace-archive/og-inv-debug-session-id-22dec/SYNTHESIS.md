# Session Synthesis

**Agent:** og-inv-debug-session-id-22dec
**Issue:** orch-go-untracked-1766445116 (note: beads issue not found)
**Duration:** 2025-12-22 15:12 → 2025-12-22 15:35
**Outcome:** success

---

## TLDR

Investigated why tmux-spawned agents don't have session IDs written to workspaces. Root cause: `FindRecentSessionWithRetry` runs before prompt is sent, but OpenCode only creates sessions after receiving a message. Only 8 of 233 workspaces (3.4%) have session IDs.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-22-inv-debug-session-id-write.md` - Full investigation with findings, root cause, and fix recommendation

### Files Modified
- None (investigation only)

### Commits
- Pending (investigation file not yet committed)

---

## Evidence (What Was Observed)

- Only 8 of 233 workspaces have `.session_id` files (verified with `find`)
- All 4 recent tmux spawn events show `session_id: ""` in `~/.orch/events.jsonl`
- Session `ses_4b7aa4810ffeCXd4...` was created at 1766445135855, but spawn event was at 1766445135000 (855ms earlier)
- Code order in `runSpawnTmux`: WaitForOpenCodeReady → FindRecentSession → SendPrompt → WriteSessionID
- Session lookup happens BEFORE prompt is sent, but session only exists AFTER

### Tests Run
```bash
# Count session ID files
find .orch/workspace -name ".session_id" | wc -l
# Result: 8

# Count total workspaces
ls -d .orch/workspace/*/ | wc -l
# Result: 233

# Check session creation time
curl -s http://127.0.0.1:4096/session/ses_4b7aa4810ffeCXd4jaiJRsMkL6 | jq '.time.created'
# Result: 1766445135855

# Check spawn event timestamp from events.jsonl
# Result: 1766445135 (session created 855ms AFTER event logged)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-22-inv-debug-session-id-write.md` - Full investigation with D.E.K.N. summary

### Decisions Made
- Decision 1: Fix by moving session lookup to after prompt is sent - this matches actual session lifecycle

### Constraints Discovered
- OpenCode attach mode creates sessions only after first message received (not when TUI starts)
- 30-second window for session matching can cause wrong session ID in concurrent spawns

### Externalized via `kn`
- None yet (recommend recording after fix is implemented)

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Fix session ID capture timing in runSpawnTmux
**Skill:** feature-impl (or systematic-debugging)
**Context:**
```
Session ID lookup runs before prompt is sent, but OpenCode only creates sessions after receiving a message.
Fix: Move FindRecentSessionWithRetry to after SendEnter, add ~2s delay.
See: .kb/investigations/2025-12-22-inv-debug-session-id-write.md
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Why do some workspaces have the SAME session ID? (suggests race condition in concurrent spawns)
- How long does OpenCode take to register a session after receiving a message?
- Should session title matching be added to avoid wrong session ID?

**Areas worth exploring further:**
- Whether the fix should also include title matching for robustness
- Whether inline mode has similar issues (currently parses stdout for session ID)

**What remains unclear:**
- Exact timing of when OpenCode registers session (after first byte? after Enter? after response?)
- Whether there are edge cases where session still won't be found after the fix

---

## Session Metadata

**Skill:** investigation
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-debug-session-id-22dec/`
**Investigation:** `.kb/investigations/2025-12-22-inv-debug-session-id-write.md`
**Beads:** `orch-go-untracked-1766445116` (not found in beads)
