# Model: Session Deletion Vectors

**Domain:** OpenCode Integration / Session Reliability
**Last Updated:** 2026-02-14
**Synthesized From:** 3 investigations (2026-02-13 to 2026-02-14), opencode source analysis (session/index.ts, project/instance.ts, project/state.ts, config.ts), orch-go cleanup code (clean_cmd.go, cleanup/sessions.go, daemon/cleanup.go)

---

## Summary (30 seconds)

Active OpenCode sessions can become unfindable through **7 independent vectors** spanning 3 systems (disk-cleanup.sh, orch-go cleanup, OpenCode itself). The fundamental problem is that no "session is active, do not touch" lock exists, and multiple processes can delete sessions from the shared SQLite database without coordination. The Ctrl+D keybind is triple-bound (app exit, session delete, input delete), creating the highest-risk accidental deletion path. The disk-cleanup.sh vector was the first confirmed root cause (now fixed), but the bug persists because at least two other vectors remain open.

---

## Core Mechanism

### How Sessions Are Stored

**Post-migration (Feb 13, 2026):** SQLite database at `~/.local/share/opencode/opencode.db`

```
OpenCode Server (localhost:4096)
    ├── In-memory: Instance cache (Map<directory, Context>)
    │   └── MAX_INSTANCES = 20, IDLE_TTL_MS = 30min
    ├── SQLite: opencode.db (WAL mode, foreign keys ON)
    │   ├── SessionTable (id, project_id FK, directory, title, ...)
    │   ├── MessageTable (session_id FK, CASCADE DELETE)
    │   ├── PartTable (message_id FK, session_id, CASCADE DELETE)
    │   └── SessionShareTable (session_id FK, CASCADE DELETE)
    └── Legacy: ~/.local/share/opencode/storage/ (JSON files, pre-migration)
```

### How Sessions Are Looked Up

```typescript
// Session.get() - Direct lookup by ID, hits SQLite
const row = Database.use((db) =>
  db.select().from(SessionTable).where(eq(SessionTable.id, id)).get()
)
if (!row) throw new NotFoundError({ message: `Session not found: ${id}` })

// Session.list() - Scoped to project (requires Instance context)
const project = Instance.project
const rows = db.select().from(SessionTable)
  .where(eq(SessionTable.project_id, project.id)).all()
```

**Key distinction:** `Session.get(id)` hits SQLite directly (no instance context needed for the query itself). `Session.list()` requires an active Instance to resolve the project. Both paths can throw NotFoundError, but for different reasons:
- `get()` throws when the row is missing from SQLite (session was deleted)
- `list()` requires Instance context, which requires a non-evicted instance

### The Gap Between "Exists in DB" and "Findable"

Session.get() is a pure database lookup. If the row exists in SQLite, it's findable. The NotFoundError means the row was **actually deleted from the database**, not merely evicted from memory.

This is the critical architectural insight: **instance eviction does NOT cause NotFoundError.** Eviction cleans up in-memory state (State.dispose) and removes the cache entry, but sessions remain in SQLite. The next request to that directory re-creates the instance and reconnects.

### Critical Invariants

1. **Sessions exist in SQLite or they don't** - There is no "evicted but recoverable" state
2. **NotFoundError = row deleted from DB** - Not a caching issue
3. **Multiple processes share one SQLite DB** - No coordination protocol
4. **Cascade deletes propagate silently** - Deleting a session kills all messages and parts
5. **No "active session" lock exists** - Any process can delete any session at any time
6. **JSON→SQLite migration is one-time** - Gate checks `opencode.db` existence, not whether sessions were imported. DB existed from Jan 27 schema migrations → 188 JSON sessions permanently orphaned, invisible to current code

---

## Deletion Vector Table

| # | Vector | Trigger | Protection | Gap | Risk | Status |
|---|--------|---------|------------|-----|------|--------|
| 1 | **disk-cleanup.sh** | Hourly launchd (`com.dylan.disk-cleanup`) | Lines now commented out (Feb 2026) | None (fixed) | N/A | **FIXED** |
| 2 | **`orch clean --sessions` (untracked)** | `cleanUntrackedDiskSessions()` | 3-layer: workspace tracking → 5min recency → IsSessionProcessing() | Sessions idle >5min WITHOUT `.session_id` workspace file bypass Layer 2+3 and get DELETED. TUI/interactive sessions have no workspace. | **HIGH** | **OPEN** |
| 3 | **Ctrl+D keybind conflict** | User presses Ctrl+D in TUI session list dialog | Double-press confirmation (`Press ctrl+d again to confirm`) | Triple-bound: `app_exit`, `session_delete`, `input_delete` all default to `ctrl+d`. Muscle memory for "exit" triggers delete when session list is focused. | **HIGH** | **OPEN** |
| 4 | **DELETE /session/:id API** | Any HTTP client calls `DELETE /session/:id` | None - unauthenticated on localhost | Any local process (orch, scripts, curl) can delete any session | **MEDIUM** | **OPEN** (by design) |
| 5 | **Daemon periodic cleanup** | Every 6 hours via `RunPeriodicCleanup()` | 7-day age threshold, `IsSessionProcessing()` check, `PreserveOrchestrator: true` | Safe for active sessions (7-day threshold far exceeds any active session age). Orchestrator title detection is heuristic-based. | **LOW** | **SAFE** |
| 6 | **CASCADE via project deletion** | `SessionTable.project_id` has `onDelete: cascade` | No code path found that deletes projects | Theoretical only - no `Project.remove()` exists in codebase | **THEORETICAL** | **SAFE** |
| 7 | **SQLite migration gate bug** | Upstream commit `6d95f0d14` (Feb 13) rewrote storage from JSON to SQLite | One-time gate: only runs if `opencode.db` doesn't exist | DB existed since Jan 27 from earlier schema migrations → json import **never ran**. 188 JSON sessions permanently orphaned. Affects all incremental upgraders. Does NOT cause runtime NotFoundError in current binary (current code uses SQLite exclusively). | **MEDIUM** (data loss, not crashes) | **CONFIRMED — filed upstream [#13654](https://github.com/anomalyco/opencode/issues/13654)** |

---

## Why This Fails

### Failure Mode 1: Untracked Session Deletion (Vector #2)

**Symptom:** Interactive TUI session disappears mid-conversation with NotFoundError

**Root cause:** `cleanUntrackedDiskSessions()` in `clean_cmd.go:408-539` finds sessions not tracked by any `.orch/workspace/*/session_id` file, checks if they were updated in the last 5 minutes, and deletes them if idle.

**Why TUI sessions are vulnerable:**
- Interactive/orchestrator TUI sessions have NO workspace directory
- No `.session_id` file means Layer 1 protection is bypassed entirely
- If the user hasn't sent a message in >5 minutes (reading, thinking, context switch), Layer 2 (recency) marks it as orphaned
- Layer 3 (`IsSessionProcessing()`) only runs for recently-active sessions
- Session gets deleted via `client.DeleteSession(session.ID)` at line 525

**Code path:**
```
orch clean --sessions (or --all)
  → cleanUntrackedDiskSessions()
    → !trackedSessionIDs[session.ID]  ← TUI has no workspace, always true
    → now.Sub(updatedAt) > 5min       ← User paused, true
    → (skips IsSessionProcessing)     ← Only checked for recent sessions
    → client.DeleteSession(session.ID) ← SESSION DELETED
```

**Fix needed:** Call `IsSessionProcessing()` for ALL untracked sessions, not just recently active ones. Cost: one API call per untracked session.

### Failure Mode 2: Accidental Ctrl+D Deletion (Vector #3)

**Symptom:** Session vanishes after user presses Ctrl+D

**Root cause:** Three keybinds share `ctrl+d`:
- `app_exit: "ctrl+c,ctrl+d,<leader>q"` (config.ts:771)
- `session_delete: "ctrl+d"` (config.ts:784)
- `input_delete: "ctrl+d,delete,shift+delete"` (config.ts:878)

**Why it happens:**
1. User opens session list (`<leader>l`)
2. User wants to exit the list, presses Ctrl+D (habit from terminal/vim)
3. Session list dialog intercepts as `session_delete`, highlights session in red
4. If user presses Ctrl+D again (common stutter or habit), session is permanently deleted
5. TUI crashes with NotFoundError on next render cycle

The confirmation ("Press ctrl+d again to confirm") is displayed as red-highlighted title text that may not be noticed in a fast interaction.

**Fix needed:** Rebind `session_delete` to a non-conflicting key, or add a modal confirmation dialog.

### Failure Mode 3: External Process Deletion (Vector #4)

**Symptom:** Session disappears without user action

**Root cause:** `DELETE /session/:id` route has no authentication and no coordination. Any local process can delete any session.

**Why it happens:**
- orch-go's `client.DeleteSession()` calls `DELETE /session/:id`
- Multiple orch processes may run concurrently (daemon, manual clean, etc.)
- No lock file, no coordination protocol
- Session deleted while TUI is rendering/using it

---

## Constraints

### Why Can't We Add a "Session Is Active" Lock?

**Constraint:** Multiple processes (TUI, server, orch daemon, orch clean) access the same SQLite database. SQLite WAL mode allows concurrent reads but serializes writes.

**Implication:** A file lock or database flag could work but introduces coordination overhead and lock-file cleanup on crash.

**This enables:** Simple, lock-free architecture
**This constrains:** No way to prevent concurrent deletion of active sessions

### Why Is the TUI Especially Vulnerable?

**Constraint:** TUI (interactive) sessions don't create workspace directories with `.session_id` files. Workspaces are an orch-go concept, not an OpenCode concept.

**Implication:** The entire orch-go cleanup infrastructure's Layer 1 defense (workspace tracking) is invisible to TUI sessions. They have zero protection from cleanup routines.

**This enables:** OpenCode TUI works independently of orch-go
**This constrains:** orch-go cleanup can't distinguish "active TUI session" from "orphaned session"

### Why Can't We Just Remove the DELETE API?

**Constraint:** Session cleanup is necessary. Without it, sessions accumulate indefinitely (627+ sessions observed Jan 6, 2026), slowing queries and consuming disk space.

**Implication:** Must retain deletion capability but add guards

**This enables:** Bounded resource usage
**This constrains:** Must design deletion guards rather than removing deletion

---

## Evolution

**Pre-Feb 2026: JSON File Storage**
- Sessions stored as JSON files in `~/.local/share/opencode/storage/`
- disk-cleanup.sh deleted these files directly, causing crashes
- No cascade deletion (files were independent)

**Feb 2026: disk-cleanup.sh fix**
- Lines deleting OpenCode storage files commented out
- Bug persisted → other vectors exist

**Feb 13, 2026: SQLite Migration**
- Upstream commit `6d95f0d14` rewrote session storage to SQLite
- CASCADE DELETE now propagates through foreign keys
- All session/message/part data in single DB file

**Feb 13, 2026: orch clean gap discovered**
- Investigation confirmed `cleanUntrackedDiskSessions()` can delete idle sessions without workspace tracking
- 3-layer defense has gap for sessions idle >5min without `.session_id`

**Feb 14, 2026: Ctrl+D keybind conflict identified**
- `session_delete`, `app_exit`, and `input_delete` all bound to `ctrl+d`
- Session list dialog shows confirmation as red-highlighted title text
- Upstream commit `b02075844` changed session listing (removed directory filtering)

**Feb 14, 2026: Instance eviction ruled out**
- Instance eviction (`disposeCurrent()`) only cleans in-memory state
- Sessions persist in SQLite after eviction
- `Session.get()` hits DB directly, not affected by eviction
- This was the least investigated hypothesis - now confirmed as NOT a deletion vector

**Feb 14, 2026: Vector #7 probed — migration gate bug confirmed**
- JSON→SQLite migration never ran: `opencode.db` existed since Jan 27 from earlier schema migrations
- 188 JSON session files permanently orphaned (exist on disk, invisible to current SQLite code)
- Dylan's JSON-path NotFoundError was from pre-migration binary still running, NOT from migration gap
- Current binary uses SQLite exclusively for sessions — Vector #7 cannot cause runtime crashes
- Filed upstream: https://github.com/anomalyco/opencode/issues/13654
- Probe: `.kb/models/session-deletion-vectors/probes/2026-02-14-probe-vector7-sqlite-migration-json-fallback.md`

---

## References

**Investigations:**
- `.kb/investigations/2026-02-14-inv-active-orchestrator-session-deleted-while.md` - Root cause analysis of the ongoing TUI crash bug
- `.kb/investigations/2026-02-13-inv-verify-whether-orch-clean-kills-headless-sessions.md` - Confirmed orch clean gap for untracked sessions

**Probes:**
- `.kb/models/session-deletion-vectors/probes/2026-02-14-probe-vector7-sqlite-migration-json-fallback.md` - Vector #7 confirmed: migration gate bug, 188 orphaned sessions

**Upstream Issues:**
- https://github.com/anomalyco/opencode/issues/13654 - JSON→SQLite migration gate bug

**Related models:**
- `.kb/models/opencode-session-lifecycle/model.md` - How sessions are created, stored, and queried (stale: predates SQLite migration)
- `.kb/models/agent-lifecycle-state-model/model.md` - How agent state spans multiple layers including sessions

**Primary Evidence (Verify These):**
- `opencode/src/session/index.ts:322-326` - `Session.get()` throws NotFoundError when row missing from SQLite
- `opencode/src/session/index.ts:569-589` - `Session.remove()` CASCADE deletes session + messages + parts
- `opencode/src/config/config.ts:771,784,878` - Ctrl+D triple-bind (app_exit, session_delete, input_delete)
- `opencode/src/project/instance.ts:86-104` - `disposeCurrent()` cleans memory only, no session deletion
- `opencode/src/project/instance.ts:25-26` - MAX_INSTANCES=20, IDLE_TTL_MS=30min
- `opencode/src/session/session.sql.ts:44,70` - CASCADE DELETE on MessageTable and PartTable foreign keys
- `cmd/orch/clean_cmd.go:464-485` - 5-minute recency threshold skips IsSessionProcessing for idle sessions
- `pkg/cleanup/sessions.go:31-131` - Daemon cleanup with 7-day threshold (safe)
- `pkg/daemon/daemon.go:99-102` - Daemon config: cleanup every 6h, 7-day threshold, preserve orchestrator
- `~/bin/disk-cleanup.sh:169-174` - Commented-out lines that previously deleted session files
