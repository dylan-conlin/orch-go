# Model: OpenCode Fork

**Domain:** Dylan's OpenCode fork at `~/Documents/personal/opencode`
**Created:** 2026-02-13
**Last Updated:** 2026-03-06
**Source:** Investigation `2026-02-13-inv-build-model-opencode-fork.md` + fork's own `2026-02-11-inv-opencode-fork-resource-audit-investigate.md`

---

## Summary

Dylan owns a fork of [sst/opencode](https://github.com/sst/opencode) at `~/Documents/personal/opencode`. The fork is 16 custom commits ahead of upstream (linear forward — all upstream changes included). Custom changes focus on **memory management** (LRU/TTL instance eviction preventing 8.4GB unbounded growth), **SSE cleanup** (idempotent teardown preventing leaked connections), **OAuth stealth mode** (Claude Max access), **ORCH_WORKER header forwarding** (worker session detection), **session metadata** (extensible key-value storage), and **session TTL** (auto-expiry with periodic cleanup). The fork is a TypeScript monorepo (Bun runtime, Hono HTTP framework) with sessions stored in SQLite database (~/.local/share/opencode/opencode.db) using Drizzle ORM with migration-based schema management. Session status (idle/busy/retry) is tracked in-memory only, lost on server restart. A `GET /session/status` endpoint already exists for querying session state.

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
Sessions are stored in SQLite database at `~/.local/share/opencode/opencode.db`:

**Database:**
- SQLite with WAL mode enabled
- Located at `~/.local/share/opencode/opencode.db`
- Managed via Drizzle ORM (bun-sqlite driver)

**Schema Management:**
- Schema definitions in `packages/opencode/src/*.sql.ts` files (e.g., `session.sql.ts`, `message.sql.ts`)
- Migration files generated via `bun drizzle-kit generate` in `packages/opencode/migration/`
- Migrations applied at server startup via `drizzle-orm/bun-sqlite/migrator`
- **CRITICAL:** Schema changes in `*.sql.ts` MUST have corresponding migrations or server crashes on startup (columns missing from actual database)
- Pre-commit hook enforces migration generation for schema changes

**Tables:**
- `session` — session metadata (id, slug, projectID, directory, title, metadata, time_ttl, etc.)
- `message` — messages per session
- `part` — message parts (text, tool calls, etc.)
- `todo` — task tracking
- `permission` — access control rules
- `project` — project metadata
- `control_account` — account management

**Key Features:**
- **Indexed queries** — no more linear scans for session lists
- **Project scoping** — sessions grouped by `projectID` (git root commit hash)
- **Migration-based evolution** — schema changes require explicit migrations

### Instance vs Session (Critical Distinction)
- **Instance** = execution context for a project directory (spawns LSP servers, MCP clients, file watchers)
- **Session** = conversation record (messages, parts, metadata) — persists in SQLite database
- Instance has TTL/eviction (30min idle); Session can optionally have TTL (configurable per-session)
- Sessions without TTL persist forever until explicitly deleted
- Multiple sessions can share one Instance (same project directory)

### Instance.state Async Init Semantics
`Instance.state(asyncFn)` returns a lazy singleton factory. Key behaviors:
- Calling the factory triggers init; the returned Promise is cached (keyed by directory + init function reference)
- **Rejected Promises are cached with no retry** — a transient failure permanently disables that state for the directory until Instance eviction (30min TTL, >20 LRU) or server restart
- If the factory result is not awaited at the call site, initialization runs in background and code proceeds before state is ready (fire-and-forget risk)
- **Fix applied (2026-02-18):** Cache is now cleared on rejection, enabling retry on next call (validated empirically: `attempts=1, samePromise=true` before fix; `attempts=2, samePromise=false` after fix)

### Provider and Auth Scoping
- **Auth** (`~/.local/share/opencode/auth.json`) is **GLOBAL** — same OAuth tokens used for all project directories
- **Provider.state** is scoped per-directory via `Instance.state`, but because all inputs (auth, models.dev cache, server process.env) are global, provider initialization is deterministically identical across projects
- **`.env` files are NOT loaded** — `Env.state` copies `process.env` at instance creation time; project-level `.env` has zero effect on provider initialization
- **CodexAuthPlugin** is in `INTERNAL_PLUGINS` array, loaded unconditionally for every directory — Codex either works for all projects or none

### MCP Watcher Initialization (Race Condition — Fixed)
A fire-and-forget race existed in MCP hot-reload: `InstanceBootstrap` called `MCP.watchConfig()` without awaiting, and `watchConfig()` itself didn't await `configWatcherState()`. Result: the file watcher wasn't ready until after `readProjectMcpConfig()` completed (10-100ms I/O), so config changes during that window were missed. Tests passed accidentally because `MCP.status()` provided implicit delay. **Fix (2026-02-20):** `watchConfig()` made async and awaited in bootstrap. All 1116 tests pass after fix.

---

## Fork Divergence

### Remote Setup
```
fork     → git@github.com:user/opencode.git  (Dylan's GitHub fork)
origin   → https://github.com/sst/opencode            (upstream, stale Jan 27)
upstream → https://github.com/sst/opencode.git         (upstream, fetched Feb 5)
```

### Custom Commits (upstream/dev..HEAD, 16 commits)

| Date | Commit | Category | Description |
|------|--------|----------|-------------|
| Feb 14 | f3c3865b8 | feat(session) | Add TTL with periodic cleanup and SSE events |
| Feb 14 | 36f084ca5 | feat | Add optional metadata field to Session.Info schema |
| Feb 14 | 3ea245f6f | fix | Rebind session_delete from ctrl+d to <leader>d |
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
- Sync method: `git reset --hard upstream/dev` + cherry-pick custom commits
- upstream/dev IS ancestor of local dev — pure linear forward advancement
- Fork now 16 custom commits ahead (3 new commits added Feb 14: TTL, metadata, keybind fix)
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
- **`metadata?: Record<string, string>`** — extensible metadata bag (added Feb 14, 2026)
- **`time_ttl?: number`** — session TTL/auto-expiry in milliseconds (added Feb 14, 2026)
- **No state field** — idle/busy tracked separately in memory
- **`time.archived`** is a manual flag, not auto-expiry
- Metadata enables storing arbitrary key-value pairs (e.g., beads_id, workspace_path)
- TTL enables automatic cleanup of expired sessions via periodic job

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
1. **Create** → Row inserted into SQLite `session` table
2. **Prompt** → Status set to "busy", messages/parts appended incrementally
3. **Complete** → Status set to "idle" (deleted from status map), Instance stays cached
4. **Archive** → Manual `time.archived` flag via PATCH, hides from active list
5. **TTL Expiry** → Periodic cleanup job deletes sessions where `Date.now() - time.updated > time_ttl` (if set)
6. **Delete** → Recursive deletion of session + all messages + all parts + unshare

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

**Status:** ✅ **IMPLEMENTED** (Feb 14, 2026, commit f3c3865b8)

**Implementation:**
1. ✅ Added `time_ttl?: number` field to `Session.Info` schema (Drizzle schema + Zod validation)
2. ✅ Added periodic cleanup job running every 5 minutes
3. ✅ Cleanup job queries sessions from SQLite where `Date.now() - time.updated > time_ttl`
4. ✅ TTL configurable per-session (set at creation) via `POST /session` input
5. ✅ SSE event `session.deleted` emitted on auto-deletion
6. ✅ Respects active prompts — won't delete sessions with `status = "busy"`

**Database Migration:**
- Required Drizzle migration to add `time_ttl` column to `session` table
- Migration generated via `bun drizzle-kit generate`
- Applied automatically at server startup

### Feature 2: Session Metadata API

**Status:** ✅ **IMPLEMENTED** (Feb 14, 2026, commit 36f084ca5)

**Implementation:**
1. ✅ Added `metadata?: Record<string, string>` field to `Session.Info` schema (Drizzle schema + Zod validation)
2. ✅ Accepts `metadata` in `POST /session` (create) and `PATCH /session/:id` (update)
3. ✅ `Session.createNext()` includes metadata in initial database insert
4. ✅ `Session.update()` PATCH handler merges metadata
5. ✅ Metadata persists to SQLite (JSON column in `session` table)

**Database Migration:**
- Required Drizzle migration to add `metadata` column (JSON type) to `session` table
- Migration generated via `bun drizzle-kit generate`
- Applied automatically at server startup

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
- **16 custom commits is manageable** — any more and rebasing becomes painful

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
- `GET /session/status` returns liveness status only (`idle | busy | retry`) — does NOT include session metadata

### Rejected Promise Caching (Now Fixed)
- Prior to the fix, `State.create` cached rejected Promises with no retry — a transient failure (e.g., auth.json being written during `orch account switch`, or `ModelsDev.get()` timeout) would permanently disable a provider for a directory until Instance eviction (30min TTL, >20 LRU) or server restart
- This explained "all agents at time T fail, agents at time T+42min work" — the Instance was evicted (TTL) clearing the rejected Promise cache
- **Fix applied 2026-02-18:** Cache is cleared on rejection, enabling retry on next call
- If encountering provider failures, check if `Provider.state` is returning cached rejection before assuming auth/network issue

### Schema Changes Require Migrations
- Modifying `*.sql.ts` schema files without running `bun drizzle-kit generate` causes runtime crashes
- Error: columns missing from actual database when server tries to query/insert
- Pre-commit hook enforces migration generation: checks for schema changes without corresponding migrations
- Migration workflow:
  1. Edit schema in `packages/opencode/src/*.sql.ts`
  2. Run `bun drizzle-kit generate` to create migration file in `migration/`
  3. Server auto-applies migrations at startup via `drizzle-orm/bun-sqlite/migrator`
- **Never skip migrations** — SQLite won't auto-sync schema changes like some ORMs

### PATCH Endpoint Extensibility
- Accepts `title`, `time.archived`, and `metadata` (as of Feb 14, 2026)
- Metadata support added via Zod validator update in session.ts
- Future field additions require updating both route validator AND update function body
- Must ensure backward compatibility (optional fields only)

---

## What This Enables (for orch-go lifecycle)

| orch-go Problem | OpenCode Feature | Status | Lines Eliminated |
|-----------------|-----------------|--------|------------------|
| Ghost/phantom/orphan cleanup | Session TTL auto-expiry | ✅ IMPLEMENTED | ~1,200 (cleanup logic) |
| Workspace .session_id cross-reference | Session metadata API | ✅ IMPLEMENTED | ~800 (file-based cross-ref) |
| SSE-only status polling | GET /session/status (exists!) | Available (integration needed) | ~1,400 (SSE status parsing) |
| **Total** | | | **~3,400 lines** |

### Implementation Status (as of Feb 14, 2026)
1. ✅ **Session metadata API** — COMPLETE (commit 36f084ca5)
2. ✅ **Session TTL** — COMPLETE (commit f3c3865b8)
3. ⏳ **GET /session/status integration** — Available in OpenCode, orch-go client integration pending

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

---

## References

**Primary Evidence (Verify These):**
- `~/Documents/personal/opencode/packages/opencode/src/instance.ts` - Instance LRU/TTL eviction implementation (350 lines, custom)
- `~/Documents/personal/opencode/packages/opencode/src/server.ts` - SSE cleanup with idempotent teardown
- `~/Documents/personal/opencode/packages/opencode/src/session.ts` - Session storage and API implementation (now using Drizzle ORM)
- `~/Documents/personal/opencode/packages/opencode/src/session.sql.ts` - Session table schema definition (Drizzle)
- `~/Documents/personal/opencode/packages/opencode/migration/` - Drizzle migration files
- `~/Documents/personal/opencode/packages/sdk/js/src/v2/client.ts` - ORCH_WORKER header forwarding
- `~/Documents/personal/opencode/.git/` - Git history showing 16 custom commits ahead of upstream
- `~/.local/share/opencode/opencode.db` - SQLite database (WAL mode)
- `~/Documents/personal/opencode/packages/opencode/src/project/state.ts` - State.create caching logic (rejected Promise fix applied)
- `~/Documents/personal/opencode/packages/opencode/src/mcp/index.ts` - MCP watcher initialization (race condition fix applied)
- `~/Documents/personal/opencode/packages/opencode/src/project/bootstrap.ts` - InstanceBootstrap sequence (MCP.watchConfig now awaited)

### Merged Probes

| Probe | Date | Verdict | Summary |
|-------|------|---------|---------|
| `probes/2026-02-18-probe-headless-codex-provider-init.md` | 2026-02-18 | EXTENDS | Auth is global not per-project; no .env loading; CodexAuthPlugin is internal plugin unconditionally loaded; rejected Promise caching explains intermittent provider failures |
| `probes/2026-02-18-probe-repro-headless-codex-spawn.md` | 2026-02-18 | NO IMPACT | Blocked by concurrency limit (9/5 agents), headless Codex path not exercised |
| `probes/2026-02-18-probe-state-create-rejected-promise-cache.md` | 2026-02-18 | CONFIRMS + EXTENDS | Empirically confirmed rejected Promise caching (`attempts=1, samePromise=true`); fix validated enabling retry after rejection |
| `probes/2026-02-18-probe-verify-session-metadata-api.md` | 2026-02-18 | CONFIRMS | Metadata persists via `Session.setMetadata`; TTL cleanup guards busy sessions; `GET /session/status` returns liveness only, not metadata |
| `probes/2026-02-20-probe-mcp-hot-reload-production-failure.md` | 2026-02-20 | EXTENDS | Documented MCP watcher fire-and-forget race condition (double unawaited async init); fix applied and verified with regression test |

## Auto-Linked Investigations

- .kb/investigations/archived/2025-12-25-inv-opencode-crashes-no-user-message.md
