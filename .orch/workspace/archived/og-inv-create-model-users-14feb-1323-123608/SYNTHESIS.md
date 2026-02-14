# Session Synthesis

**Agent:** og-inv-create-model-users-14feb-1323
**Issue:** (ad-hoc, no beads tracking)
**Duration:** 2026-02-14
**Outcome:** success

---

## TLDR

Created model documenting 7 independent session deletion vectors across 3 systems (disk-cleanup.sh, orch-go, OpenCode). Key finding: instance eviction does NOT cause NotFoundError (sessions persist in SQLite after eviction). The two highest-risk open vectors are the `orch clean --sessions` gap for untracked sessions and the Ctrl+D triple-keybind conflict.

---

## Delta (What Changed)

### Files Created
- `.kb/models/session-deletion-vectors.md` - Model documenting all 7 vectors with risk assessment, code paths, and fix recommendations

### Files Modified
- None

---

## Evidence (What Was Observed)

- `Session.get()` at `session/index.ts:322-326` does a direct SQLite lookup by ID - no instance context needed. NotFoundError means the row is truly gone from the database.
- `disposeCurrent()` at `instance.ts:86-104` calls `State.dispose()` + deletes cache entry + metadata. NO session deletion from database. Instance eviction is memory-only.
- `State.dispose()` at `state.ts:31-69` clears `recordsByKey` entries and calls dispose callbacks. Pure in-memory cleanup.
- Ctrl+D is bound to THREE actions simultaneously: `app_exit` (config.ts:771), `session_delete` (config.ts:784), `input_delete` (config.ts:878).
- Session list dialog at `dialog-session-list.tsx:52` shows deletion confirmation as red-highlighted title text, not a modal.
- `cleanUntrackedDiskSessions()` at `clean_cmd.go:464-485` has a 5-minute recency threshold. Sessions idle >5min without workspace tracking skip the `IsSessionProcessing()` check entirely.
- Daemon cleanup at `daemon.go:99-102` uses 7-day threshold with `PreserveOrchestrator: true` - safe for active sessions.
- SQLite schema has CASCADE DELETE on all foreign keys: MessageTable → SessionTable, PartTable → MessageTable, SessionShareTable → SessionTable.

---

## Knowledge (What Was Learned)

### Key Architectural Insight
The NotFoundError is a **database deletion problem**, not a memory/caching problem. Instance eviction (the least-investigated hypothesis) was ruled out as a vector. This refocuses the investigation on the two confirmed open vectors.

### Constraints Discovered
- TUI sessions have zero orch-go workspace protection because workspaces are an orch-go concept
- SQLite WAL mode allows concurrent reads but no coordination protocol for deletions
- Session cleanup cannot be removed (unbounded growth observed: 627+ sessions)

---

## Next (What Should Happen)

**Recommendation:** close

### Immediate Fixes (2 issues to create)
1. **Fix orch clean gap** - Call `IsSessionProcessing()` for ALL untracked sessions, not just recently active ones. Cost: one API call per untracked session. File: `cmd/orch/clean_cmd.go:464-485`
2. **Fix Ctrl+D keybind conflict** - Rebind `session_delete` from `ctrl+d` to a non-conflicting key in Dylan's opencode fork. File: `opencode/src/config/config.ts:784`

### Probes Recommended
- **SQLite migration probe** - Verify that pre-migration JSON sessions were correctly imported to SQLite. Check if any sessions were lost during the Feb 13 migration.

---

## Unexplored Questions

- **Does the Feb 14 commit `b02075844` (session listing changes) create new confusion?** - Removed directory filtering from session listings, showing ALL project sessions. Could cause users to accidentally delete sessions from other directories.
- **What happens to an in-flight agent when its session is deleted?** - Does the agent crash immediately, or does it continue writing to a now-deleted session?
- **Can disk-cleanup.sh in aggressive mode (`--aggressive`) affect the SQLite DB file?** - The aggressive mode deletes caches broadly; need to verify it doesn't touch `~/.local/share/opencode/opencode.db`.

---

## Session Metadata

**Skill:** investigation
**Model:** opus
**Workspace:** `.orch/workspace/og-inv-create-model-users-14feb-1323/`
**Deliverable:** `.kb/models/session-deletion-vectors.md`
