# Investigation: OpenCode Fork Audit — Session Lifecycle Hooks & Integration Points

**Date:** 2026-02-06
**Status:** Complete
**Beads:** orch-go-21363

## Summary

OpenCode's session lifecycle is built on a simple but deeply coupled architecture: JSON files on disk + an in-process event bus + SSE broadcast. There is no database, no WAL, no query engine. Every write is a JSON file write via `Bun.write()`, every read is `Bun.file().json()`, and every event goes through the `Bus` pub/sub which also feeds the plugin system and SSE stream.

This investigation maps the complete write path, inventories all hooks, and assesses three integration options for a single source of truth for agent state.

---

## 1. Session Lifecycle — Complete Write Path

### Session Creation

**Code path:** `Session.createNext()` at `session/index.ts:206-247`

```
Session.create(input)
  → Session.createNext({parentID, directory, title, permission})
    → Constructs Info object (id, slug, projectID, directory, parentID, title, time)
    → Storage.write(["session", projectID, id], info)          // disk write
    → Bus.publish(Event.Created, {info})                       // event
    → (optional) auto-share if configured
    → Bus.publish(Event.Updated, {info})                       // second event
    → return info
```

**Storage key:** `~/.local/share/opencode/storage/session/{projectID}/{sessionID}.json`

**Bus events emitted:** `session.created`, `session.updated`

### Session Update

**Code path:** `Session.update()` at `session/index.ts:297-309`

```
Session.update(id, editor, options?)
  → Storage.update(["session", projectID, id], draft => { editor(draft); draft.time.updated = Date.now() })
    → Lock.write(target)
    → Bun.file(target).json()     // read
    → fn(content)                  // mutate in-memory
    → Bun.write(target, JSON.stringify(content, null, 2))  // write back
  → Bus.publish(Event.Updated, {info})
  → return info
```

**Called by:** title changes, share/unshare, session touch, status updates, revert operations, compaction, archive

### Message Write

**Code path:** `Session.updateMessage()` at `session/index.ts:376-382`

```
Session.updateMessage(msg)
  → Storage.write(["message", sessionID, messageID], msg)
  → Bus.publish(MessageV2.Event.Updated, {info: msg})
```

**Storage key:** `~/.local/share/opencode/storage/message/{sessionID}/{messageID}.json`

### Part Write (Tool Results, Text, Reasoning)

**Code path:** `Session.updatePart()` at `session/index.ts:428-437`

```
Session.updatePart(input)
  → Storage.write(["part", messageID, partID], part)
  → Bus.publish(MessageV2.Event.PartUpdated, {part, delta?})
```

**Storage key:** `~/.local/share/opencode/storage/part/{messageID}/{partID}.json`

**Critical:** This is called on every streaming chunk from the LLM. High-frequency writes during agent execution.

### Session Status (In-Memory Only)

**Code path:** `SessionStatus.set()` at `session/status.ts:61-75`

```
SessionStatus.set(sessionID, status)
  → Bus.publish(Event.Status, {sessionID, status})    // always
  → if idle: Bus.publish(Event.Idle, {sessionID})      // deprecated compat
  → if idle: delete state()[sessionID]                  // remove from memory
  → else: state()[sessionID] = status                   // store in memory
```

**NOT persisted to disk.** This is purely in-memory. Status transitions are:
- `busy` → set when LLM stream starts (`processor.ts:59`)
- `idle` → set when prompt loop completes (`prompt.ts:250,258`)
- `busy` → set when prompt begins (`prompt.ts:276`)
- `retry` → set on retryable error (`processor.ts:349`)

### Session Delete

**Code path:** `Session.remove()` at `session/index.ts:353-374`

```
Session.remove(sessionID)
  → Session.get(sessionID)
  → for each child: Session.remove(child.id)           // recursive
  → Session.unshare(sessionID)                          // cleanup share
  → for each message: for each part: Storage.remove     // delete parts
  → for each message: Storage.remove                    // delete messages
  → Storage.remove(["session", projectID, sessionID])   // delete session
  → Bus.publish(Event.Deleted, {info})
```

### Session List (The Known Bottleneck)

**Code path:** `Session.list()` at `session/index.ts:332-339`

```
async function* list() {
  for (const item of await Storage.list(["session", projectID])) {
    const session = await Storage.read<Info>(item).catch(() => undefined)
    if (!session) continue
    yield session
  }
}
```

**Problem:** Sequential `Storage.read()` per session. O(n) in disk I/O. This is the P0 from today's investigation.

---

## 2. Event Emission — Complete Inventory

### Bus Architecture

```
Bus.publish(event)
  → Notify all type-specific subscribers
  → Notify all wildcard ("*") subscribers
  → GlobalBus.emit("event", {directory, payload})     // Node EventEmitter
```

The SSE `/event` endpoint subscribes via `Bus.subscribeAll()` and forwards all events to connected clients. The plugin system also subscribes via `Bus.subscribeAll()` and calls `hook.event({event})` for each plugin.

### All Session-Related Events

| Event Type | Defined In | When Emitted | Data |
|------------|-----------|--------------|------|
| `session.created` | `session/index.ts:106` | Session creation | `{info: Session.Info}` |
| `session.updated` | `session/index.ts:113` | Any session mutation | `{info: Session.Info}` |
| `session.deleted` | `session/index.ts:118` | Session removal | `{info: Session.Info}` |
| `session.diff` | `session/index.ts:124` | File changes computed | `{sessionID, diff[]}` |
| `session.error` | `session/index.ts:131` | Processing error | `{sessionID?, error}` |
| `session.status` | `session/status.ts:28` | busy/idle/retry transition | `{sessionID, status}` |
| `session.idle` | `session/status.ts:36` | (deprecated) idle transition | `{sessionID}` |
| `message.updated` | `session/message-v2.ts` | Message write | `{info: MessageV2.Info}` |
| `message.removed` | `session/message-v2.ts` | Message deleted | `{sessionID, messageID}` |
| `message.part.updated` | `session/message-v2.ts` | Part write (high freq) | `{part, delta?}` |
| `message.part.removed` | `session/message-v2.ts` | Part deleted | `{sessionID, messageID, partID}` |
| `session.compacted` | `session/compaction.ts` | After compaction | `{sessionID}` |
| `todo.updated` | `session/todo.ts` | Todo list change | `{todos}` |
| `command.executed` | `session/prompt.ts` | Command run | `{command}` |

### Plugin Hook Points

| Hook | When Called | Can Block? | Data Available |
|------|-----------|------------|----------------|
| `event` (catch-all) | Every Bus event | No | Full event payload |
| `tool.execute.before` | Before tool execution | Yes (throw) | `{tool, sessionID, callID}`, `{args}` |
| `tool.execute.after` | After tool execution | No | `{tool, sessionID, callID}`, `{output}` |
| `config` | Plugin initialization | N/A | Mutable config object |
| `experimental.chat.system.transform` | Before LLM call | No | `{system}` array |
| `experimental.session.compacting` | During compaction | No | Context preservation |

### Where to Add Event Emission for Orch

The most natural integration points for orch to subscribe to:

1. **`Bus.subscribeAll()`** — Already exists. The SSE `/event` endpoint and plugin system both use this. A new subscriber could materialize events to an external store.

2. **`GlobalBus.emit("event")`** — The GlobalBus (`bus/global.ts`) is a Node EventEmitter that bridges per-instance Bus events to a global scope. This is already called on every `Bus.publish()`.

3. **Plugin `event` hook** — The existing plugin system calls `hook.event({event})` for every bus event. An orch plugin could materialize state here.

4. **New hook: `session.lifecycle`** — Could be added at `session/index.ts` level to emit a structured lifecycle event combining session + status info. This doesn't exist today.

---

## 3. Storage Format — On-Disk Structure

### Directory Layout

```
~/.local/share/opencode/storage/
├── migration                              # Migration version (integer)
├── project/
│   └── {projectID}.json                   # Project metadata
├── session/
│   └── {projectID}/
│       └── {sessionID}.json               # Session info
├── session_diff/
│   └── {sessionID}.json                   # File diffs per session
├── message/
│   └── {sessionID}/
│       └── {messageID}.json               # Individual messages
├── part/
│   └── {messageID}/
│       └── {partID}.json                  # Message parts (text, tool, reasoning)
└── share/
    └── {sessionID}.json                   # Share metadata
```

### Session JSON Schema

```json
{
  "id": "ses_01jkx...",
  "slug": "random-slug",
  "version": "1.1.52",
  "projectID": "abc123...",
  "directory": "/Users/dylanconlin/Documents/personal/orch-go",
  "parentID": "ses_01jkw...",            // optional, for child sessions
  "title": "New session - 2026-02-06T...",
  "time": {
    "created": 1738800000000,
    "updated": 1738800100000,
    "compacting": null,
    "archived": null
  },
  "permission": [...],                    // optional PermissionNext.Ruleset
  "summary": {                            // optional, computed
    "additions": 42,
    "deletions": 10,
    "files": 3
  },
  "share": {                              // optional
    "url": "https://..."
  }
}
```

### File Characteristics

- **Format:** Pretty-printed JSON (`JSON.stringify(content, null, 2)`)
- **Locking:** Read/write file locks via `Lock.read()`/`Lock.write()` (in `util/lock.ts`)
- **No indexing:** No secondary indexes. All queries are full directory scans + sequential file reads.
- **No transactions:** Each file write is independent. No atomicity across files.
- **Migration system:** Sequential numbered migrations stored in `storage/migration` file.

### Could OpenCode Write to SQLite?

**Yes, technically.** The `Storage` namespace (`storage/storage.ts:12-227`) is a clean abstraction with 5 methods: `read`, `write`, `update`, `remove`, `list`. Replacing the implementation with SQLite would require:

1. **Replace `storage.ts`** — Swap `Bun.file().json()` / `Bun.write()` with SQLite queries. All 5 methods would change.

2. **Schema design:**
   ```sql
   CREATE TABLE storage (
     key TEXT PRIMARY KEY,     -- e.g., "session/projectID/sessionID"
     value TEXT NOT NULL,      -- JSON blob
     updated_at INTEGER
   );
   CREATE INDEX idx_prefix ON storage(key);
   ```

3. **What would break:**
   - Migration system (reads file at `storage/migration`) — trivial to adapt
   - `Storage.list()` uses `Bun.Glob` — would become SQL `WHERE key LIKE ?`
   - File locks (`Lock.read/write`) — replaced by SQLite's native locking
   - Any external tools reading JSON files directly (e.g., `orch clean --sessions`)

4. **What would improve:**
   - `Session.list()` becomes a single query instead of N file reads
   - Filtering (`?start=`, `?search=`, `?roots=`) becomes SQL WHERE clauses
   - Atomic multi-file operations become transactions
   - Index on `key` enables O(log n) lookups

**LOE:** ~2-3 days for a drop-in SQLite backend. ~1 week including testing, migration from existing JSON files, and edge cases.

**Risk:** Medium. Bun has built-in `bun:sqlite` which is fast. The main risk is subtle differences in concurrent access patterns (file locks vs SQLite WAL mode) and ensuring the migration path from existing JSON storage doesn't lose data.

---

## 4. Three Integration Options

### Option A: OpenCode Writes to Orch's SQLite Directly

**Mechanism:** A plugin or hook in OpenCode that, on every lifecycle event, writes a summary row to orch's SQLite database.

**Concrete code path:**

```
Plugin event hook (plugin/index.ts:128-135)
  → hook.event({event})
  → if event.type in ["session.created", "session.updated", "session.status", "session.deleted"]
    → SQLite write to ~/.orch/state.db
```

**Implementation:**

1. Create an OpenCode plugin at `~/.config/opencode/plugin/orch-state-sync.ts`
2. Plugin receives all bus events via `event` hook
3. On relevant events, write to orch's SQLite:
   ```typescript
   event: async ({event}) => {
     if (event.type === "session.created") {
       db.run("INSERT INTO sessions ...", event.properties.info)
     }
     if (event.type === "session.status") {
       db.run("UPDATE sessions SET status = ? WHERE id = ?", ...)
     }
   }
   ```

**Pros:**
- Write-time materialization (no query-time joins)
- Plugin system already exists and is proven
- Events already flow through this path
- No OpenCode core changes needed
- `orch status` becomes a simple `SELECT` instead of 6 system joins

**Cons:**
- Plugin runs in OpenCode's process — if OpenCode crashes, writes stop
- Dual-write problem: OpenCode writes JSON + plugin writes SQLite. If either fails, they diverge.
- SQLite from Bun (OpenCode) and Go (orch) requires WAL mode and careful concurrency
- Plugin can't capture status transitions that happen before plugin loads (startup race)
- Plugin duplication issue — global + project plugin may both fire

**LOE:** ~3-5 days
- Day 1: SQLite schema design + plugin skeleton
- Day 2-3: Event filtering, dedup, error handling
- Day 4-5: Testing concurrent access from Go + Bun, migration from current approach

**Risk:** Medium. Main risk is the dual-write divergence and Go ↔ Bun SQLite concurrency.

---

### Option B: OpenCode Emits Events, Orch Subscribes and Materializes

**Mechanism:** Orch-go subscribes to OpenCode's SSE stream and materializes events into its own SQLite.

**Concrete code path:**

```
OpenCode Bus.publish(event)
  → GlobalBus.emit("event", {directory, payload})
  → SSE /event endpoint: streamSSE(Bus.subscribeAll(...))
  → orch-go SSE client (pkg/opencode/sse.go) receives event
  → orch-go materializer writes to ~/.orch/state.db
```

**Implementation:**

1. orch-go already has SSE parsing (`pkg/opencode/sse.go:159 lines`)
2. orch-go already has a monitor (`pkg/opencode/monitor.go:221 lines`) that subscribes to SSE
3. Extend the monitor or create a new `materializer` service:
   ```go
   func (m *Materializer) HandleEvent(event SSEEvent) {
     switch event.Type {
     case "session.created":
       m.db.Exec("INSERT INTO sessions ...", event.Properties)
     case "session.status":
       m.db.Exec("UPDATE sessions SET status = ? ...", ...)
     }
   }
   ```

**Pros:**
- Orch owns all writes to its own DB (no dual-write problem)
- SSE infrastructure already exists and works
- Go's SQLite access is well-tested (mattn/go-sqlite3 or modernc.org/sqlite)
- Natural extension of existing monitor pattern
- Decoupled: OpenCode doesn't need to know about orch's storage

**Cons:**
- Requires persistent SSE connection (already exists for monitor)
- Events are lost during SSE disconnection (reconnection gap)
- Still depends on OpenCode being up for events
- Event serialization/deserialization overhead (JSON over SSE → Go struct → SQLite)
- Can't retroactively materialize — if SSE subscriber starts after events, they're gone
- High-frequency events (message.part.updated) generate significant SSE traffic

**LOE:** ~3-4 days
- Day 1: SQLite schema + materializer skeleton
- Day 2: Event filtering (which events to materialize, which to drop)
- Day 3-4: Reconnection handling, backfill from API on reconnect, testing

**Risk:** Low-medium. This is the most natural extension of the current architecture. Main risk is SSE gaps.

---

### Option C: Orch Bypasses OpenCode's Storage, Owns Session State

**Mechanism:** Orch becomes the session state authority. OpenCode's disk storage is treated as a cache/history, not source of truth.

**Concrete code path:**

Two sub-options:

**C1: Orch creates sessions via API and tracks them externally**
```
orch spawn
  → POST /session (OpenCode creates session + writes JSON)
  → orch captures session ID from response
  → orch writes to ~/.orch/state.db with session ID + metadata
  → SSE subscription for status updates
  → orch DB becomes source of truth for lifecycle
  → OpenCode JSON storage becomes message history only
```

**C2: Replace OpenCode's Storage with shared SQLite**
```
Replace storage/storage.ts to use SQLite at ~/.orch/state.db
  → All OpenCode reads/writes go through shared SQLite
  → Orch reads same SQLite directly
  → Single source of truth
```

**C1 Implementation (External ownership):**

1. On `orch spawn`, capture session ID from API response
2. Write session metadata to orch's SQLite immediately
3. Subscribe to SSE for status transitions
4. `orch status` queries orch's SQLite only (no OpenCode API calls)
5. OpenCode's JSON files become message archive only

**C2 Implementation (Shared SQLite):**

1. Replace `storage/storage.ts` with SQLite backend (see Section 3 analysis)
2. Point both OpenCode and orch-go at same SQLite file
3. Use WAL mode for concurrent readers
4. orch-go reads session state directly from SQLite

**Pros:**
- Eliminates the "distributed JOIN at query time" problem entirely
- Single source of truth (orch's DB)
- `orch status` becomes O(1) instead of O(n) API calls
- No event gap problem (state is persistent, not event-sourced)

**Cons:**
- **C1:** Requires orch to be running for state to be accurate. If orch is down, OpenCode still works but state diverges.
- **C1:** Doesn't capture events that happen outside orch's awareness (TUI sessions, manual API calls)
- **C2:** Major fork divergence from upstream OpenCode. Every upstream merge becomes painful.
- **C2:** SQLite concurrent write from two processes (Bun + Go) requires careful WAL configuration
- **C2:** Breaking change for any tool reading OpenCode's JSON files

**LOE:**
- **C1:** ~2-3 days (orch already captures session ID; extend with SQLite)
- **C2:** ~1-2 weeks (storage backend swap + migration + testing + upstream merge strategy)

**Risk:**
- **C1:** Low. Minimal OpenCode changes.
- **C2:** High. Major fork divergence.

---

## 5. Runtime Needs vs Bookkeeping Separation

### What Orch Genuinely Needs at Runtime

| Data | Source Today | Needed For | Frequency |
|------|------------|------------|-----------|
| Session ID | API response | Sending messages, aborting | Once at spawn |
| Session status (busy/idle) | SSE events | Completion detection, `orch wait` | Per state change |
| Session title | API response | Display in `orch status` | Once (+ title update) |
| Session parentID | API response | Hierarchy display | Once at spawn |
| Session directory | API response | Cross-project filtering | Once at spawn |
| Message count/cost | GET /message | Token tracking, cost display | Per status query |

### What's Only Needed for Bookkeeping

| Data | Source Today | Needed For | Could Be Deferred? |
|------|------------|------------|-------------------|
| Full message history | GET /message (all parts) | Resume, debugging, audit | Yes — query on demand |
| File diffs | session.diff event | Dashboard display | Yes — query on demand |
| Part-level streaming | SSE message.part.updated | TUI display, progress | Only for dashboard |
| Compaction state | session.compacted event | Context management | OpenCode handles internally |
| Permission state | Session.Info.permission | Agent capability display | Query on demand |
| Todo state | /session/{id}/todo | Display | Query on demand |

### Can They Be Separated?

**Yes.** The separation is clean:

**Runtime layer (orch must know immediately):**
- Session exists (id, title, directory, parentID) — captured at spawn time
- Session status (busy/idle/retry) — from SSE `session.status` events
- Session error state — from SSE `session.error` events

**Bookkeeping layer (orch queries on demand):**
- Message history — GET /session/{id}/message
- Token usage — computed from messages
- File diffs — GET /session/{id}/diff
- Todos — GET /session/{id}/todo

**The critical insight:** Orch only needs ~6 fields at runtime. Everything else can be queried lazily from OpenCode's API. The current `orch status` problem isn't that we need more data — it's that we re-derive the runtime fields from scratch on every query instead of materializing them at write time.

---

## Recommendation

**Option B (SSE materialization)** is the recommended path for immediate value. It:

1. Requires zero OpenCode fork changes
2. Extends the existing SSE/monitor infrastructure
3. Gives orch sole ownership of its state DB
4. Makes `orch status` a simple SELECT
5. Can be built incrementally (start with session.created + session.status, expand later)

**Option A (plugin)** is a good secondary approach if we want to capture events that happen before orch's SSE subscriber connects (startup race). The plugin starts with OpenCode.

**Option C1** is essentially what we do today but with persistent storage. It's the least change.

**Option C2** should only be pursued if we decide to deeply integrate the storage layers — which creates long-term fork maintenance burden.

**Recommended first step:** Build a minimal materializer in orch-go that:
1. Subscribes to SSE (already exists)
2. On `session.created`: INSERT into SQLite
3. On `session.status`: UPDATE in SQLite
4. On `session.deleted`: DELETE from SQLite
5. On reconnect: backfill from GET /session API

This replaces the "distributed JOIN" in `orch status` with a local read, immediately fixing the 23s→<1s performance issue at scale.

---

## References

- `packages/opencode/src/session/index.ts` — Session CRUD + Bus events
- `packages/opencode/src/storage/storage.ts` — JSON file storage abstraction
- `packages/opencode/src/session/status.ts` — In-memory status + Bus events
- `packages/opencode/src/session/processor.ts` — LLM stream processing, status transitions
- `packages/opencode/src/session/prompt.ts` — Prompt loop, status transitions
- `packages/opencode/src/bus/index.ts` — Event bus (pub/sub + GlobalBus bridge)
- `packages/opencode/src/bus/global.ts` — GlobalBus (Node EventEmitter)
- `packages/opencode/src/server/server.ts:496-531` — SSE endpoint
- `packages/opencode/src/server/routes/session.ts` — HTTP API routes
- `packages/opencode/src/plugin/index.ts` — Plugin loader + event broadcasting
- `packages/opencode/src/tool/task.ts` — Task tool (child session creation)
- `.kb/investigations/2026-02-06-inv-opencode-session-list-performance-cliff.md` — Today's P0
- `.kb/models/opencode-session-lifecycle.md` — Prior model (updated by this investigation)
- `.kb/models/agent-lifecycle-state-model.md` — Four-layer state model
