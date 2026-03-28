# Model: OpenCode Session Lifecycle

**Domain:** OpenCode Integration / Session Management
**Last Updated:** 2026-03-28
**Synthesized From:** 24 investigations (2025-12-19 to 2026-01-08) into OpenCode HTTP API, session persistence, SSE monitoring, and plugin system

---

## Summary (30 seconds)

OpenCode sessions persist across server restarts via disk storage at `~/.local/share/opencode/storage/`. Sessions are queried differently based on whether you need in-memory (running) or disk (historical) data. Completion detection relies on SSE `session.status` events transitioning from `busy` to `idle`, NOT session existence. The system supports three spawn modes (headless/tmux/inline) with different trade-offs for automation vs visibility.

---

## Core Mechanism

### How Sessions Work

**Storage architecture (post-Feb 13, 2026 SQLite migration):**

```
OpenCode Server (localhost:4096)
    ├── In-memory: Instance cache (Map<directory, Context>), MAX_INSTANCES=20, IDLE_TTL=30min
    └── SQLite: ~/.local/share/opencode/opencode.db (WAL mode, foreign keys ON)
             └── SessionTable, MessageTable, PartTable (all historical sessions)
    [Legacy: ~/.local/share/opencode/storage/ — JSON files, pre-Feb 13 migration]
```

**Note:** This model predates the Feb 13, 2026 SQLite migration. The JSON disk storage description below is superseded. See the `session-deletion-vectors` model for the current SQLite architecture. Key difference: `x-opencode-directory` header filtering behavior may have changed with SQLite migration.

**Key insight (pre-migration):** Two types of queries, controlled by HTTP headers:

| Query Type | Header | Returns |
|------------|--------|---------|
| In-memory | None | Currently running sessions only |
| Disk | `x-opencode-directory: /path/to/project` | All historical sessions for that directory |

### Session Lifecycle

**State transitions:**

```
spawn (orch spawn creates session)
    ↓
session created (POST /session)   ← NOT /api/sessions — orch-go client has used /session since initial commit
    ↓
busy (agent working, SSE: session.status "busy")
    ↓                              ↘
idle (agent finished)          retry (transient failure, auto-retry)
    ↓                              ↓
session persists on disk       retries then → idle or error
```

**Status types:** `busy` (working), `idle` (finished), `retry` (transient failure, has attempt count and next retry time).

**Critical behaviors:**
- Sessions persist on disk by default (survives server restarts)
- Sessions can be created with a `TimeTTL` (seconds) for automatic expiration — 0 means no expiration
- Sessions accept messages even after `idle` state
- `DELETE /session/{id}` deletes individual sessions; `orch clean --sessions` for bulk cleanup

### Spawn Modes

**Three execution modes:**

| Mode | Command | Implementation | Use Case |
|------|---------|----------------|----------|
| **Headless** (default) | `orch spawn` | HTTP API only, returns immediately | Automation, daemon, batch work |
| **Tmux** | `orch spawn --tmux` | HTTP API + tmux window | Interactive monitoring, visual debugging |
| **Inline** | `orch spawn --inline` | Blocking TUI in current terminal | Quick tasks, debugging |

**Why headless is default:**
- No TUI overhead (faster)
- Returns immediately (non-blocking)
- Enables daemon automation
- Supports high concurrency (5+ agents)
- Session still accessible via `orch status`, `orch send`

### Completion Detection

**Two mechanisms available:**

**1. SSE (event-driven, used by Monitor):**
```
SSE Stream: /event
──────────────────────────────────────────
session.status { status: "busy" }    ← Agent working
message.part.updated                 ← Content streaming
session.status { status: "idle" }    ← Agent finished
```

**2. HTTP Polling (used by WaitForSessionIdle):**
```
GET /session/status?ids=<session-id>
→ Returns: { "<session-id>": { "type": "busy"|"idle"|"retry", ... } }
Polls every 500ms until busy→idle transition.
```

**Key insight:** Completion is `busy` → `idle` transition, NOT session disappearance.

**Why this matters:**
- Session existence ≠ agent still working
- Both SSE and polling are supported — polling via `GET /session/status` is simpler
- Monitor uses SSE for multi-session watching; `WaitForSessionIdle` uses polling for single-session blocking

### Critical Invariants

1. **Sessions persist across restarts** - SQLite storage at `~/.local/share/opencode/opencode.db`
2. **Directory filtering is required for disk queries** - Without `x-opencode-directory` header, only get in-memory sessions
3. **Completion detection supports both SSE and polling** - SSE via `/event`, polling via `GET /session/status`
4. **Sessions can have TTL** - `TimeTTL` field on creation enables automatic expiration; 0 = no expiration (default)
5. **Session directory is explicitly set at creation** - `CreateSession` accepts `directory` parameter and sets `x-opencode-directory` header (cross-project bug fixed)
6. **Server can become unresponsive without crashing** - The Bun process stays alive (same PID) but the event loop blocks, making the HTTP server stop responding. PID-based crash detection (overmind `--auto-restart`, ServiceMonitor) cannot detect this. Requires HTTP liveness probing. (Added 2026-03-28)

---

## Why This Fails

### Failure Mode 1: Cross-Project Sessions Show Wrong Directory (FIXED)

**Symptom:** `orch spawn --workdir /other/project` creates session with orchestrator's directory

**Root cause (was):** `spawn_cmd.go` didn't pass `--workdir` value to OpenCode session creation

**Fix (confirmed 2026-03-21):** `CreateSession` now accepts explicit `directory` parameter and sets `x-opencode-directory` header. `SendMessageInDirectory` also accepts directory for message routing. The `x-opencode-env-ORCH_WORKER` header signals worker sessions to the plugin system.

### Failure Mode 2: Session Accumulation

**Symptom:** 627 sessions accumulated over 3 weeks, slowing queries

**Root cause:** OpenCode never deletes sessions, no automatic cleanup

**Why it happens:**
- Sessions persist indefinitely by design
- No TTL or expiration mechanism
- Dashboard queries all sessions (slow with 600+)

**Fix (Jan 6):** `orch clean --sessions --days N` command to delete old sessions

### Failure Mode 4: Plugin Context Injection Mechanism

**Symptom:** Assuming plugins inject context via system prompt mutation

**Root cause:** Plugin context injection uses `client.session.prompt({ sessionID, prompt, noReply: true })` — a new message with parts inserted into the session, not `experimental.chat.system.transform`. `tool.execute.after` fires for gpt-5.2-codex worker sessions (confirmed via event-test.jsonl `tool.executed.bash` entries).

**Why it matters:**
- Coaching plugin also writes metrics to `~/.orch/coaching-metrics.jsonl`
- Plugin logging to console requires `ORCH_PLUGIN_DEBUG=1`

### Failure Mode 3 (Original): Deprecated session.idle Event

**Symptom:** Plugin code using `session.idle` event fails to detect completion

**Root cause:** OpenCode changed event structure - `session.idle` is deprecated

**Why it happens:**
- Old event: `session.idle` (simple)
- New event: `session.status` with `status.type === "idle"` (structured)
- Breaking change, no migration guide

**Fix (Jan 8):** Updated skills and plugins to use `session.status` event

### Failure Mode 5: Server Unresponsiveness Without Crash (Added 2026-03-28)

**Symptom:** OpenCode on port 4096 stops responding to HTTP requests during long sessions (20+ hours). No crash reports, no OOM kills. Process stays alive with unchanged PID.

**Root cause:** The SSE subscription mechanism in `Bus.publish()` awaits ALL accumulated subscribers. Dead or slow SSE subscribers block the event loop, making the HTTP server unresponsive.

**Why supervision didn't detect it:**
- Overmind `--auto-restart` only triggers on process death
- ServiceMonitor only checked PID changes (`last.PID != current.PID`)
- Zero `service.crashed` events in event log despite repeated unresponsiveness

**Fix (2026-03-28):** Added HTTP liveness probing to `pkg/service/monitor.go`. 3 consecutive health check failures (30s) with unchanged PID triggers force-restart via overmind. Logs `service.unresponsive` events.

**Remaining (OpenCode fork):** SSE subscription accumulation needs defensive measures: subscriber timeout, max listener enforcement, writeSSE error handling.

---

## Constraints

### Session State is Queryable via HTTP (constraint removed)

**Previous constraint (now resolved):** OpenCode HTTP API didn't expose session state.

**Current state:** `GET /session/status` returns status (`busy`/`idle`/`retry`) for all or specific sessions. `WaitForSessionIdle` polls this endpoint every 500ms. SSE via `/event` remains available for event-driven monitoring.

**This enables:** Both poll-based and event-driven completion detection
**This constrains:** Status is in-memory only — sessions not in the status map are considered idle

### Why Can't We Filter Sessions Without Headers?

**Constraint:** Default `/api/sessions` query returns in-memory sessions only

**Implication:** Must explicitly request disk storage via `x-opencode-directory` header

**Workaround:** Always include directory header for comprehensive queries

**This enables:** Fast queries when you only need running sessions
**This constrains:** Can't get "all sessions across all projects" in one query

### Why Do Sessions Persist by Default?

**Constraint:** Sessions persist unless given a TTL or explicitly deleted

**Implication:** Without TTL, session count grows without bound

**Mitigations:**
- `TimeTTL` on `CreateSession` enables automatic expiration
- `DELETE /session/{id}` for individual deletion
- `orch clean --sessions --days N` for bulk cleanup

**This enables:** Session history survives crashes/restarts
**This constrains:** Requires either TTL or periodic cleanup for long-running systems

---

## Evolution

**Dec 19-21, 2025: Initial HTTP Integration**
- POC: Can we spawn agents via HTTP API?
- Session creation, message sending via REST
- SSE monitoring for real-time updates

**Dec 22-26, 2025: Spawn Mode Evolution**
- Tmux mode for visual monitoring
- Headless mode added
- Flipped default from tmux to headless (Dec 22)

**Jan 6, 2026: Session Management**
- Session accumulation discovered (627 sessions)
- Cleanup mechanism added (`--sessions` flag)
- Cross-project session directory bug identified

**Jan 6-8, 2026: Plugin System**
- Plugin capabilities explored (gates, context injection, observation)
- Event reliability tested
- `session.idle` deprecation handled

**Feb 13, 2026: SQLite Migration (major architecture change)**
- OpenCode migrated from JSON file storage to SQLite (`opencode.db`)
- This model's storage section is largely superseded; see `session-deletion-vectors` model

**Feb 18, 2026: API prefix history confirmed**
- orch-go client has ALWAYS used `/session` (not `/api/sessions`) since initial commit `26f9acba`
- OpenCode server routes have no committed `/api` prefix in git history (uncommitted local change adds `/api` mount alongside existing direct routes)
- No SPA proxy stripping of `/api` in Vite configs

**Feb 20, 2026: Plugin context injection mechanism confirmed**
- Context injection uses `client.session.prompt` with `noReply: true` (message insertion, not system prompt mutation)
- `tool.execute.after` confirmed firing for gpt-5.2-codex sessions

---

## References

**Key Investigations:**
- `2025-12-19-inv-opencode-poc-can-spawn-agents.md` - Initial HTTP API proof of concept
- `2025-12-22-inv-flip-default-spawn-mode-headless.md` - Default mode change
- `2026-01-06-inv-implement-opencode-session-cleanup-mechanism.md` - Session cleanup
- `2026-01-06-inv-cannot-query-opencode-sessions-other.md` - Cross-project directory bug
- `2026-01-08-inv-test-opencode-plugin-event-reliability.md` - Event structure changes
- ...and 19 others

**Decisions Informed by This Model:**
- Headless spawn as default (enables daemon automation)
- Plugin system as principle mechanization bridge
- Session persistence across restarts (reliability)

**Related Models:**
- `.kb/models/spawn-architecture/model.md` - How spawn creates and configures sessions
- `.kb/models/dashboard-agent-status.md` - How dashboard interprets session state

**Related Guides:**
- `.kb/guides/opencode.md` - How to use OpenCode integration (procedural)
- `.kb/guides/opencode-plugins.md` - How to write plugins (procedural)

### Merged Probes

| Probe | Date | Key Finding |
|-------|------|-------------|
| `2026-02-18-probe-api-prefix-history.md` | 2026-02-18 | orch-go always used `/session` (never `/api/sessions`); no committed `/api` prefix in OpenCode server history; no SPA proxy stripping — contradicts model's "POST /api/sessions" claim |
| `2026-02-20-probe-coaching-plugin-injection.md` | 2026-02-20 | Plugin injection uses `client.session.prompt` with `noReply: true` (message insertion, not system prompt mutation); `tool.execute.after` fires for gpt-5.2-codex sessions |
| `2026-03-21-probe-knowledge-decay-verification.md` | 2026-03-21 | Two stale constraints corrected: `GET /session/status` enables polling (not SSE-only); `TimeTTL` enables session expiration; `retry` status type documented; cross-project directory bug confirmed fixed |
| `2026-03-28-probe-opencode-server-unresponsiveness-liveness-gap.md` | 2026-03-28 | Server becomes unresponsive without crashing; PID-based supervision blind to event loop blocking; added HTTP liveness probing to ServiceMonitor |

**Primary Evidence (Verify These):**
- `pkg/opencode/client.go` - HTTP REST client (~728 lines)
- `pkg/opencode/sse.go` - SSE stream parsing (~159 lines)
- `pkg/opencode/monitor.go` - Completion detection (~221 lines)
- `cmd/orch/spawn_cmd.go` - Session creation (cross-project bug location)

## Auto-Linked Investigations

- .kb/investigations/archived/2025-12-20-design-explore-tradeoffs-orch-opencode-integration.md
- .kb/investigations/archived/2025-12-19-inv-client-opencode-session-management.md
