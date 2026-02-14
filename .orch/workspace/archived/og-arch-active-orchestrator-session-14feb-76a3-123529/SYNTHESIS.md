# Session Synthesis

**Agent:** og-arch-active-orchestrator-session-14feb-76a3
**Issue:** orch-go-f3g
**Duration:** 2026-02-14 09:56:44 → 2026-02-14 10:16:00 (approx 20 minutes)
**Outcome:** success

---

## TLDR

Investigated why active orchestrator sessions get deleted mid-conversation in OpenCode. Root cause: sessions are explicitly deleted via API calls (not automatic cleanup), likely triggered by recent upstream commits (SQLite migration Feb 13 + session listing changes Feb 14) that may cause session list confusion, combined with TUI keyboard shortcut (Ctrl+D) for deletion. Recommend reporting bug upstream with reproduction steps.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-14-inv-active-orchestrator-session-deleted-while.md` - Complete investigation documenting session deletion mechanisms, recent upstream changes, and root cause hypothesis

### Files Modified
- (None - investigation only, no code changes)

### Commits
- (Pending - will commit investigation file before exit)

---

## Evidence (What Was Observed)

- Session ID `ses_3a4b0aaf2ffe6kzgksyu5RyRz1` not found in database (confirmed deleted)
- `Session.remove()` at session/index.ts:569-589 is the ONLY code path that deletes sessions
- Exposed via DELETE API endpoint at server/routes/session.ts:236
- No automatic cleanup code found in codebase (grep search confirmed)
- Sessions cascade delete when projects deleted, but no code deletes projects
- Foreign keys enabled: `PRAGMA foreign_keys = ON` (db.ts:76)
- 4 processes share database: opencode server + 3 agent bun processes
- Recent upstream commits:
  - `6d95f0d14` (Feb 13): Massive SQLite migration from JSON storage
  - `b02075844` (Feb 14): Changed session listing to show ALL project sessions regardless of directory
- TUI uses Ctrl+D for both `app_exit` and `session_delete`, creating potential for user error
- Session list dialog confirms deletion with double Ctrl+D press (dialog-session-list.tsx:89)

### Tests Run
```bash
# Searched for session in database
sqlite3 ~/.local/share/opencode/opencode.db "SELECT id FROM session WHERE id LIKE '%3a4b0aaf%';"
# Result: empty (session was deleted)

# Counted total sessions
sqlite3 ~/.local/share/opencode/opencode.db "SELECT COUNT(*) FROM session;"
# Result: 184 sessions

# Checked database locks
lsof ~/.local/share/opencode/opencode.db
# Result: 4 processes with database open (server + 3 agents)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-14-inv-active-orchestrator-session-deleted-while.md` - Documents session deletion mechanisms and root cause hypothesis

### Decisions Made
- Decision: Report bug upstream rather than implement workarounds in orch-go
  - Rationale: Bug exists in OpenCode (external dependency), not orch-go; upstream maintainers have better context on recent SQLite migration and session listing changes; workarounds would be band-aids, not root cause fixes

### Constraints Discovered
- OpenCode model claim "sessions are never deleted by OpenCode" is misleading - sessions ARE deleted, but only via explicit API calls, not automatically
- Recent upstream commits created high-risk window: SQLite migration + session listing changes within 24 hours could interact in unexpected ways
- Multiple deletion vectors exist: TUI dialog, direct API, potential race conditions from shared database

### Externalized via `kb`
- Updated investigation file with detailed findings on session deletion mechanisms
- Documented discrepancy between model claim and actual behavior

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created and filled)
- [ ] Tests passing (N/A - investigation only, no code changes)
- [ ] Investigation file has `**Phase:** Complete` (✓ updated)
- [ ] Ready for `orch complete orch-go-f3g`

**Follow-up work needed (via separate issues):**

1. **Report upstream bug to OpenCode maintainers**
   - Session ID: `ses_3a4b0aaf2ffe6kzgksyu5RyRz1`
   - Suspect commits: `b02075844` (session listing), `6d95f0d14` (SQLite migration)
   - Hypothesis: Session list shows stale/incorrect data after recent changes, leading to wrong-session deletion

2. **Consider temporary mitigation in Dylan's OpenCode fork**
   - Add enhanced deletion confirmation showing full session details (ID, directory, last updated)
   - Prevents accidental deletion of active sessions that appear inactive in list

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Could there be a race condition where multiple processes delete the same session simultaneously?
- Does the SQLite migration handle session state correctly during server restart?
- Are there edge cases in the new "show all sessions" logic that could mix up sessions from different worktrees?

**Areas worth exploring further:**
- Test session deletion behavior with multiple TUI windows open in different directories
- Review SQLite migration code (commit 6d95f0d14) for potential bugs in session handling
- Investigate if session list refresh/caching could show stale data after recent changes

**What remains unclear:**
- Exact user action that triggered deletion (was Dylan in session list dialog when it happened?)
- Whether this is a one-time occurrence or a reproducible bug
- If other users have reported similar session deletion issues after Feb 13 SQLite migration

---

## Session Metadata

**Skill:** architect
**Model:** claude-sonnet-4-5-20250929
**Workspace:** `.orch/workspace/og-arch-active-orchestrator-session-14feb-76a3/`
**Investigation:** `.kb/investigations/2026-02-14-inv-active-orchestrator-session-deleted-while.md`
**Beads:** `bd show orch-go-f3g`
