# Model: OpenCode Fork

**Domain:** Dylan's OpenCode fork at `~/Documents/personal/opencode`
**Created:** 2026-02-13
**Source:** Investigation `2026-02-13-inv-build-model-opencode-fork.md` + fork's own `2026-02-11-inv-opencode-fork-resource-audit-investigate.md`

---

## Summary

Dylan owns a fork of [sst/opencode](https://github.com/sst/opencode) at `~/Documents/personal/opencode`. The fork is 13 custom commits ahead of upstream (linear forward — all upstream changes included). Custom changes focus on **memory management** (LRU/TTL instance eviction preventing 8.4GB unbounded growth), **SSE cleanup** (idempotent teardown preventing leaked connections), **OAuth stealth mode** (Claude Max access), and **ORCH_WORKER header forwarding** (worker session detection). The fork is a TypeScript monorepo (Bun runtime, Hono HTTP framework) with sessions stored as JSON files on disk — no database. Session status (idle/busy/retry) is tracked in-memory only, lost on server restart. A `GET /session/status` endpoint already exists for querying session state.

---

## Architecture

### Tech Stack
- **Language:** TypeScript (ES modules, strict Zod validation)
- **Runtime:** Bun 1.3.5+
- **HTTP Framework:** Hono 4.10.7 with OpenAPI descriptions
- **AI SDK:** Vercel AI SDK 5.0 with providers for Anthropic, Google, OpenAI, Bedrock, etc.
- **Frontend:** Solid.js 1.9.10 + TailwindCSS 4.1 + Kobalte UI
- **Build:** Turbo monorepo, Vite for frontend
- **IDs:** ULID-based descending identifiers (newer sessions = lower IDs)

### Monorepo Structure
```
packages/
├── opencode/     # Core server + CLI + TUI (the main package)
├── sdk/          # JavaScript SDK (client library for API)
├── app/          # Web UI (Solid.js SPA)
├── web/          # Marketing/docs site
├── desktop/      # Desktop app wrapper
├── plugin/       # Plugin system
├── ui/           # Shared UI components
├── util/         # Shared utilities
└── ...           # enterprise, containers, slack, etc.
```

### Session Storage
Sessions are JSON files on disk at `~/.local/share/opencode/storage/`:
```
session/{projectID}/{sessionID}.json    # Session metadata
message/{sessionID}/{messageID}.json    # Messages per session
part/{messageID}/{partID}.json          # Message parts (text, tool calls, etc.)
session_diff/{sessionID}.json           # File diffs (migrated out of session object)
```

- **No database** — pure filesystem with file locking (read/write locks via `Lock`)
- **Project scoping** — sessions grouped by `projectID` (git root commit hash)
- **Migration system** — numbered migrations for storage format changes
- **Linear scan for list** — no indexing, scans all session files

### Instance vs Session (Critical Distinction)
- **Instance** = execution context for a project directory (spawns LSP servers, MCP clients, file watchers)
- **Session** = conversation record (messages, parts, metadata) — persists as JSON files
- Instance has TTL/eviction; Session does NOT — sessions persist forever until explicitly deleted
- Multiple sessions can share one Instance (same project directory)

---

## Fork Divergence

### Remote Setup
```
fork     → git@github.com:dylan-conlin/opencode.git  (Dylan's GitHub fork)
origin   → https://github.com/sst/opencode            (upstream, stale Jan 27)
upstream → https://github.com/sst/opencode.git         (upstream, fetched Feb 5)
```

### Custom Commits (upstream/dev..HEAD, 13 commits)

| Date | Commit | Category | Description |
|------|--------|----------|-------------|
| Feb 11 | c08ed9c | investigation | Complete OpenCode fork resource audit |
| Feb 11 | fac8fc8 | investigation | Initial checkpoint for fork resource audit |
| Feb 9 | 5032a89 | fix(config) | Skip .test.ts/.spec.ts from plugin directories |
| Feb 9 | 26d2312 | fix(server) | Catch unhandled AbortError from prompt_async cancellation |
| Feb 7 | e9e4834 | fix | **Instance eviction (LRU/TTL) + SSE cleanup** — prevents 8.4GB growth |
| Feb 7 | c9c2796 | fix(server) | Bound instance cache with LRU eviction, fix SSE teardown on dispose |
| Feb 7 | 85b0bc8 | feat(server) | Log periodic memory usage for leak diagnosis |
| Feb 7 | 0d841ae | fix(tui) | Clean up error rendering for DOMException and NotFoundError |
| Jan 29 | 87e9212 | fix(sdk) | Make SSE reconnection fix permanent in build process |
| Jan 29 | 2a2c3bf | investigation | Verify OAuth priority fix was lost from dev branch |
| Jan 28 | 2e851f3 | fix | Forward ORCH_WORKER env var as x-opencode-env-ORCH_WORKER header |
| Jan 28 | f321338 | test | Tests for session.metadata.role from ORCH_WORKER header |
| Jan 28 | 2137108 | feat(stealth) | **Full pi-ai stealth mode** parity for Claude Max OAuth |

### Sync Strategy
- Last upstream fetch: Feb 5, 2026
- Sync method: `git reset --moving to upstream/dev` + cherry-pick custom commits
- upstream/dev IS ancestor of local dev — pure linear forward advancement
- **510 commits ahead of origin/dev** (stale origin, not meaningfully different from upstream)

### Critical Custom Changes

**1. Instance LRU/TTL Eviction (instance.ts)**
Upstream: 95-line file, bare `Map<string, Promise<Context>>()` — zero eviction, unbounded growth.
Fork: 350-line file with `MAX_INSTANCES=20`, `IDLE_TTL_MS=30min`, LRU eviction, TTL eviction, active-instance protection, eviction counters. Prevents the 8.4GB catastrophe that was killing macOS.

**2. SSE Cleanup (server.ts)**
Added idempotent SSE teardown with `cleaned` flag and dual cleanup paths (client abort + server dispose). Prevents leaked SSE connections from accumulating memory.

**3. OAuth Stealth Mode**
Implements pi-ai stealth mode parity for Claude Max OAuth. Enables Opus access via Max subscription.

**4. ORCH_WORKER Header (sdk/client.ts)**
Forwards `ORCH_WORKER` env var as `x-opencode-env-ORCH_WORKER` HTTP header. Enables server to detect worker sessions. **Note:** This change is in `packages/sdk/js/src/v2/client.ts` and the commit message references `session.metadata.role` being set on the server side (session.ts:207-211), but the corresponding server-side code was not found in the current codebase — may have been lost during an upstream rebase.

---

## Session Management

### Session Schema (Zod-validated)
```typescript
Session.Info = {
  id: string                    // Descending ULID (newer = lower)
  slug: string                  // URL-safe slug
  projectID: string             // Git root commit hash
  directory: string             // Working directory path
  parentID?: string             // For forked sessions
  title: string                 // User-visible title
  version: string               // OpenCode version
  time: {
    created: number             // Timestamp (ms)
    updated: number             // Timestamp (ms)
    compacting?: number         // If compaction in progress
    archived?: number           // Archive timestamp (manual flag)
  }
  summary?: { additions, deletions, files }
  share?: { url: string }       // Shareable link
  permission?: PermissionRuleset
  revert?: { messageID, partID?, snapshot?, diff? }
}
```

**Key observations:**
- **No `metadata` bag** — schema is fixed fields only
- **No TTL/expiry field** — sessions persist forever
- **No state field** — idle/busy tracked separately in memory
- **`time.archived`** is a manual flag, not auto-expiry

### Session Status (In-Memory Only)
```typescript
SessionStatus.Info =
  | { type: "idle" }
  | { type: "retry", attempt: number, message: string, next: number }
  | { type: "busy" }
```
- Stored in `Record<string, Info>` keyed by sessionID
- Created via `Instance.state()` — tied to Instance lifecycle, **lost on server restart**
- Idle sessions are deleted from the map (absence = idle)
- Published via `session.status` Bus event → SSE stream

### Session Lifecycle
1. **Create** → JSON file written to `session/{projectID}/{sessionID}.json`
2. **Prompt** → Status set to "busy", messages/parts appended incrementally
3. **Complete** → Status set to "idle" (deleted from status map), Instance stays cached
4. **Archive** → Manual `time.archived` flag via PATCH, hides from active list
5. **Delete** → Recursive deletion of session + all messages + all parts + unshare

---

## API Surface (Session-Relevant Endpoints)

### REST API (Hono, OpenAPI-documented)

| Method | Endpoint | OperationId | Notes |
|--------|----------|-------------|-------|
| GET | `/session` | session.list | Filters: directory, roots, start, search, limit |
| GET | `/session/status` | session.status | **Returns all session statuses (idle/busy/retry)** |
| GET | `/session/:id` | session.get | Single session metadata |
| POST | `/session` | session.create | Input: parentID?, title?, permission? |
| PATCH | `/session/:id` | session.update | **Only title and time.archived** |
| DELETE | `/session/:id` | session.delete | Recursive delete (session + messages + parts) |
| POST | `/session/:id/message` | session.prompt | Streaming response |
| POST | `/session/:id/prompt_async` | session.prompt_async | Fire-and-forget (204) |
| POST | `/session/:id/abort` | session.abort | Cancel active prompt |
| GET | `/session/:id/message` | session.messages | All messages with parts |
| GET | `/event` | global.event | **SSE stream (all bus events)** |
| GET | `/health` | global.health | Server health + version |
| POST | `/dispose` | global.dispose | Dispose all instances |

### SSE Events (via GET /event)
```
session.created, session.updated, session.deleted
session.status (idle/busy/retry changes)
session.diff, session.error
message.updated, message.removed
part.updated, part.removed
server.heartbeat (every 30s)
server.instance.disposed
```

---

## Feasibility: Three Proposed Features

### Feature 1: Session TTL with Configurable Expiry

**Current state:** Sessions persist forever. Instance has TTL (30min); sessions do not. Archive is manual.

**What needs to change:**
1. Add `ttl?: number` field to `Session.Info.time` schema (Zod)
2. Add periodic cleanup job (new — nothing like this exists today)
3. Cleanup job scans sessions, deletes those where `Date.now() - time.updated > ttl`
4. Make TTL configurable — either per-session (set at creation) or global via env var
5. Update `POST /session` create input to accept TTL
6. SSE event on auto-deletion so orch-go can react

**Effort:** MODERATE (2-4 hours)
- Storage layer is simple (JSON files), no schema migration needed
- Periodic job is new pattern (no scheduler for sessions exists today)
- Instance eviction pattern in `instance.ts` is a good template
- Needs careful handling: don't delete sessions with active prompts

**Risk:** LOW
- Schema addition is backward-compatible (optional field)
- Cleanup job is independent, doesn't interfere with existing code paths

### Feature 2: Session Metadata API

**Current state:** `Session.Info` has fixed Zod schema. No extensible metadata bag. PATCH only supports `title` and `time.archived`. The ORCH_WORKER header forwarding was an attempt at metadata but the server-side code appears lost.

**What needs to change:**
1. Add `metadata?: Record<string, string>` field to `Session.Info` schema
2. Accept `metadata` in `POST /session` (create) and `PATCH /session/:id` (update)
3. Update `Session.createNext()` to include metadata in initial write
4. Update `Session.update()` PATCH handler to merge metadata
5. Metadata persists to disk (just another field in the JSON file)

**Effort:** LOW (1-2 hours)
- Adding an optional field to a Zod schema is trivial
- Storage is JSON — no migration needed, old sessions just lack the field
- PATCH endpoint already has the pattern for partial updates
- SDK client needs update to pass metadata in requests

**Risk:** VERY LOW
- Backward-compatible — old sessions without metadata still valid
- No performance impact (tiny additional JSON field)

**orch-go usage:** Store `beads_id`, `workspace_path`, `tier`, `spawn_mode` with session at creation. Query via `GET /session/:id` — eliminates workspace `.session_id` cross-reference files entirely.

### Feature 3: Session State HTTP Endpoint

**Current state:** `GET /session/status` ALREADY EXISTS. Returns `Record<string, SessionStatus>` where status is `idle | busy | retry`. This is exactly what orch-go needs.

**What needs to change:**
1. **Nothing on the OpenCode side** — endpoint already works
2. On orch-go side: use `GET /session/status` instead of SSE-only polling
3. Optionally: add per-session `GET /session/:id/status` endpoint for targeted queries

**Effort:** NEAR-ZERO for OpenCode (endpoint exists), LOW for orch-go integration

**Risk:** VERY LOW
- Read-only endpoint, no side effects
- Already has OpenAPI documentation

**Caveat:** Status is in-memory only — lost on server restart. All sessions appear "idle" after restart. This is acceptable for orch-go since it can treat "no status info" as "check if session exists" fallback.

---

## Constraints and Gotchas

### Sync with Upstream
- Fork must be periodically rebased onto upstream/dev
- Custom changes can be lost during rebase (ORCH_WORKER server-side code was lost)
- Strategy: keep custom commits minimal and well-isolated
- **13 custom commits is manageable** — any more and rebasing becomes painful

### Instance vs Session Confusion
- Instance eviction (TTL/LRU) does NOT delete sessions
- Session deletion does NOT dispose instances
- These are independent lifecycles with different storage (memory vs disk)
- orch-go currently conflates "session exists" with "agent is running" — need to check status, not just session existence

### Status is Ephemeral
- `SessionStatus` lives in-memory only via `Instance.state()`
- Server restart = all status lost = all sessions appear idle
- No persistence of busy/idle transitions
- SSE `session.status` events are the only way to track transitions in real-time

### Storage is Linear Scan
- `Session.list()` iterates all session JSON files (no index)
- `Session.children()` iterates all sessions to find matching `parentID`
- Works fine for <1000 sessions but could become slow with very high session counts
- No query optimization possible without adding an index layer

### PATCH Endpoint is Limited
- Only accepts `title` and `time.archived` today
- Adding metadata support requires updating the Zod validator in `session.ts:264-274`
- Must update both the route validator AND the update function body

---

## What This Enables (for orch-go lifecycle)

| orch-go Problem | OpenCode Feature | Lines Eliminated |
|-----------------|-----------------|------------------|
| Ghost/phantom/orphan cleanup | Session TTL auto-expiry | ~1,200 (cleanup logic) |
| Workspace .session_id cross-reference | Session metadata API | ~800 (file-based cross-ref) |
| SSE-only status polling | GET /session/status (exists!) | ~1,400 (SSE status parsing) |
| **Total** | | **~3,400 lines** |

### Implementation Priority
1. **Session metadata API** — lowest effort, highest immediate value (eliminates cross-ref files)
2. **GET /session/status integration** — zero OpenCode work, orch-go client change only
3. **Session TTL** — most complex but largest cleanup elimination

---

## What This Model Enables / Constrains

**Enables:**
- Planning concrete implementation of the three features in the fork
- Understanding what orch-go changes depend on what OpenCode changes
- Estimating effort accurately (hours, not days)

**Constrains:**
- Must keep custom commits minimal (rebase cost)
- Must not break upstream compatibility (fork may re-sync)
- Session metadata schema must be generic (not orch-go-specific field names in OpenCode)
- TTL cleanup must respect active prompts (don't delete mid-conversation)
