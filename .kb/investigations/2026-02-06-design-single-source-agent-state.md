<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Designed a single-source agent state architecture using SQLite at `~/.orch/state.db`, materialized via SSE subscription (Option B), with 6 architectural forks navigated: orch-direct phase reporting, hybrid backfill, periodic token polling, incremental migration with shadow-read verification, global DB location, and beads as audit trail with orch as runtime authority.

**Evidence:** Two completed investigations (field-level audit: 70% of state is immutable after spawn; OpenCode fork audit: SSE infrastructure exists and works) provide complete requirements. Existing Monitor code (`pkg/opencode/monitor.go`) already subscribes to SSE and handles reconnection — materializer extends this proven pattern.

**Knowledge:** The key architectural insight is separating runtime state (SQLite — fast, local, owned by orch) from audit trail (beads — persistent, searchable, agent-facing). Phase reporting moves to `orch phase <id> <phase>` for runtime (direct SQLite write, ~1ms) while agents optionally still write `bd comment` for audit. Registry becomes fully replaceable. Migration is incremental: write to both old + new, shadow-read for validation, then cut over.

**Next:** Create 5 implementation issues for phased rollout: (1) SQLite schema + write path, (2) SSE materializer, (3) `orch phase` command, (4) shadow-read migration for `orch status`, (5) registry removal.

**Authority:** architectural - Cross-component boundaries (orch, OpenCode, beads), multiple valid approaches, establishes new system pattern

---

# Investigation: Single-Source Agent State Architecture

**Question:** How should orch-go materialize agent state into a single-source SQLite database to replace the current 6-system distributed JOIN in `orch status`?

**Started:** 2026-02-06
**Updated:** 2026-02-06
**Owner:** Architect Agent (orch-go-21364)
**Phase:** Complete
**Next Step:** None - 5 implementation issues ready for creation
**Status:** Complete

<!-- Lineage -->
**Related-Models:**
- `.kb/models/agent-lifecycle-state-model.md` - Four-layer state model (this design replaces reconciliation with materialization)
- `.kb/models/dashboard-agent-status.md` - Priority Cascade (SQLite replaces cascade with single query)

**Related-Investigation:**
- `.kb/investigations/2026-02-06-inv-agent-state-field-level-audit.md` - Complete field map (INPUT to this design)
- `.kb/investigations/2026-02-06-inv-opencode-fork-audit-session-lifecycle-integration.md` - Integration options (INPUT to this design)
- `.kb/investigations/2026-02-04-inv-agents-own-declaration-via-bd.md` - Three orthogonal dimensions (informs phase reporting fork)

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `2026-02-06-inv-agent-state-field-level-audit.md` | Foundation — provides field map + schema proposal | Yes — verified against `spawn/session.go`, `serve_agents.go`, `monitor.go` | None |
| `2026-02-06-inv-opencode-fork-audit-session-lifecycle-integration.md` | Foundation — provides integration options A/B/C | Yes — verified SSE infrastructure exists in `pkg/opencode/` | None |
| `2026-02-04-inv-agents-own-declaration-via-bd.md` | Informs fork 1 — three orthogonal dimensions | Verified against code | None |
| Decision: `2026-01-12-registry-is-spawn-cache.md` | Confirms registry is replaceable | Verified — zero production usage of lifecycle methods | None |

---

## Problem Framing

**Design Question:** Given that `orch status` currently performs a distributed JOIN across 6 systems (OpenCode API, beads, tmux, registry, workspace disk, Anthropic API) taking 0.7s-23.6s, how should we design a single-source SQLite materialized view that replaces this with ~1ms local reads?

**Success Criteria:**
1. `orch status` reads from SQLite only (no multi-system queries)
2. State staleness ≤5 seconds for mutable fields (phase, is_processing)
3. Zero data loss during SSE disconnection (backfill on reconnect)
4. Incremental migration (old path works as fallback during transition)
5. Registry can be deleted after migration completes

**Constraints:**
- **Principle: Local-First** — SQLite file, not a service
- **Principle: Graceful Degradation** — Must work if OpenCode is down (show stale data)
- **Principle: Observation Infrastructure** — Every state transition must emit an event
- **Constraint (from kb):** Registry is caching layer, not source of truth — all data exists in primary sources
- **Constraint (from kb):** Post-registry lifecycle uses 4 state sources: OpenCode sessions, tmux windows, beads issues, workspaces
- **Constraint (from investigation 2):** Session status (busy/idle) is IN-MEMORY ONLY in OpenCode — not persisted to disk
- **Constraint (from investigation 1):** 70% of fields are immutable after spawn — set once by `orch spawn`

**Scope:**
- **In:** Schema design, write-path architecture, migration strategy, 6 decision forks
- **In:** Implementation issues for phased rollout
- **Out:** Actual implementation code, dashboard UI changes, OpenCode fork modifications

---

## Fork Navigation (6 Decisions)

### Fork 1: Phase Reporting — Orch-Direct or Beads-Passthrough?

**Decision:** How do agents report phase transitions (Planning, Implementing, Complete)?

**Options:**
- **A: Orch-direct** — Agent runs `orch phase <id> Implementing` which writes directly to SQLite. Beads never sees phases.
- **B: Beads-passthrough** — Agent still uses `bd comment "Phase: X"`, orch polls beads or receives webhook, materializes.
- **C: Hybrid** — Agent writes to orch for runtime, optionally writes to beads for audit trail.

**Substrate says:**
- **Investigation (agents-own-declaration):** Three orthogonal dimensions — Work Status (agent progress), Verification Status (quality gates), Issue Status (beads lifecycle). Phase is Work Status, owned by agent.
- **Investigation (field-level audit):** Beads comments are most expensive read at ~700ms per issue, O(n) in agent count. Phase parsing requires regex over all comments.
- **Principle: Observation Infrastructure:** Every state transition should emit an event.
- **Principle: Compose Over Monolith:** Small, focused tools that combine.

**SUBSTRATE:**
- Principle: Observation Infrastructure says every transition must emit an event
- Model: Three orthogonal dimensions says phase is Work Status (agent-owned), separate from Issue Status (beads)
- Investigation: Field audit shows beads comments at 700ms/issue is the hot-path bottleneck

**RECOMMENDATION:** Option C (Hybrid) — `orch phase <id> <phase>` writes to SQLite (~1ms) for runtime. Agent's SPAWN_CONTEXT.md continues to include `bd comment` instruction for audit trail. Phase in SQLite is authoritative for dashboard; beads comments become searchable history.

**Why Hybrid over Pure Orch-Direct (A):**
- Beads comments create permanent, searchable audit trail linked to issues
- `bd show <id>` with comments is valuable for debugging agent sessions
- Zero-change for existing agent instructions initially — `bd comment "Phase: X"` keeps working
- `orch phase` is additive, agents adopt it as SPAWN_CONTEXT templates are updated

**Why Hybrid over Pure Beads-Passthrough (B):**
- Eliminates 700ms/issue from hot path of `orch status`
- Phase data is in SQLite within 1ms of agent reporting
- No polling or webhook infrastructure needed for beads
- Beads has no event system (comments require polling)

**Trade-off accepted:** Two write paths for phase (orch phase + bd comment) during transition. Accepted because bd comment is audit, orch phase is runtime — different concerns.

**When this would change:** If beads develops a webhook/event system, beads-passthrough becomes viable for runtime too.

---

### Fork 2: Backfill Strategy on SSE Reconnect

**Decision:** When the SSE connection drops and reconnects, how do we fill the gap in materialized state?

**Options:**
- **A: Full scan** — GET /session API, reconcile all sessions against SQLite.
- **B: Timestamp-based** — Query only sessions updated since `last_event_time`.
- **C: Hybrid** — Full scan on first connect, timestamp-based on reconnect.

**Substrate says:**
- **Investigation (OpenCode fork audit):** Session list is O(n) in disk I/O (sequential file reads). At 89 sessions, this took 5.8s. This is the known P0 performance cliff.
- **Principle: Graceful Degradation:** Core functionality works without optional layers.
- **Investigation (field audit):** 70% of fields are immutable after spawn — only mutable fields need refresh.

**SUBSTRATE:**
- Investigation: Session list at 89 sessions = 5.8s (P0)
- Principle: Graceful Degradation requires working with stale data
- Field audit: Only 3 mutable runtime fields need refresh (is_processing, session_updated_at, window_exists)

**RECOMMENDATION:** Option C (Hybrid) with optimization.

**First connect:**
- Full scan of GET /session API with `?start=<24h_ago>` filter (limits to recent sessions)
- Reconcile against SQLite — INSERT any sessions not in DB, UPDATE mutable fields
- This handles cold-start and daemon restart scenarios

**Reconnect after gap:**
- Only refresh mutable fields for sessions already in SQLite
- For each active agent in SQLite: GET /session/{id} (single session, fast)
- If agent count is small (<20), this is fast enough (~50ms × 20 = ~1s)
- If agent count is large, batch via GET /session with title filter

**Key optimization:** Only refresh sessions that are still `is_completed = 0 AND is_abandoned = 0`. Completed/abandoned agents don't need refresh.

**Trade-off accepted:** Full scan on first connect is slow (~5.8s at 89 sessions). Accepted because it only happens once per daemon lifecycle, not on every status query.

**When this would change:** If OpenCode gets `?since=<timestamp>` filter or event log with offset-based replay.

---

### Fork 3: Token Tracking — Event-Driven or Periodic Poll?

**Decision:** How to track token usage (input, output, reasoning, cache_read)?

**Options:**
- **A: Event-driven** — Subscribe to `message.part.updated` SSE events, aggregate tokens.
- **B: Periodic poll** — Poll GET /session/{id}/message on a schedule (e.g., every 30s).
- **C: On-demand** — Only fetch tokens when `orch status` or dashboard requests them.

**Substrate says:**
- **Investigation (OpenCode fork audit):** `message.part.updated` fires on every streaming chunk. High-frequency writes during agent execution. This would generate enormous SSE traffic for marginal benefit.
- **Investigation (field audit):** Token freshness requirement is "seconds" — not real-time. Token data is derived (aggregated from all messages).
- **Principle: Compose Over Monolith:** Keep components simple and focused.

**SUBSTRATE:**
- Investigation: message.part.updated is high-frequency (every streaming chunk)
- Field audit: Token freshness is "seconds" — not real-time
- Principle: Compose Over Monolith says keep the materializer simple

**RECOMMENDATION:** Option B (Periodic poll) with smart scheduling.

**Strategy:**
- Poll token counts every 30 seconds for agents where `is_processing = 1`
- Use GET /session/{id} which includes token summary (no need to fetch all messages)
- Skip polling for idle/completed/abandoned agents
- On `session.status → idle` event, do one final token poll to capture final counts

**Why not event-driven (A):**
- `message.part.updated` fires hundreds of times per agent response
- Would need debouncing/batching logic (complexity)
- Token aggregation from events requires parsing each part — more complex than reading summary

**Why not purely on-demand (C):**
- Dashboard refreshes every 5s — would trigger token queries for every active agent every 5s
- Better to poll proactively and serve from cache

**Trade-off accepted:** Token counts may be up to 30s stale. Accepted because users don't need sub-second token freshness.

**When this would change:** If OpenCode exposes a `session.tokens` event with pre-aggregated counts.

---

### Fork 4: Migration Path — Incremental or Cutover?

**Decision:** How to transition from the current 6-system query to SQLite reads?

**Options:**
- **A: Big bang cutover** — Switch `orch status` to SQLite reads, remove old path.
- **B: Incremental with shadow-read** — Write to SQLite alongside old systems, compare results.
- **C: Feature flag** — `--sqlite` flag to opt into new path.

**Substrate says:**
- **Principle: Verification Bottleneck:** System cannot change faster than human can verify behavior.
- **Principle: Graceful Degradation:** Core functionality works without optional layers.
- **Decision: Registry is spawn-cache:** Registry was demoted but never removed — same pattern applies here.

**SUBSTRATE:**
- Principle: Verification Bottleneck requires human observation before trusting new path
- Principle: Graceful Degradation means old path must remain as fallback
- Decision: Registry demote pattern — document, deprecate, then remove

**RECOMMENDATION:** Option B (Incremental with shadow-read).

**Phase 1: Write path (no read changes)**
- `orch spawn` writes to SQLite + existing systems
- `orch complete` / `orch abandon` update SQLite + existing systems
- SSE materializer updates SQLite
- `orch phase` command created (writes to SQLite only)
- **No read path changes.** `orch status` continues using old 6-system query.

**Phase 2: Shadow-read validation**
- `orch status` queries both old path AND SQLite
- Compares results field-by-field
- Logs discrepancies to `~/.orch/events.jsonl`
- Returns old path results (SQLite is shadow-only)
- Run for 1-2 weeks of real usage to validate

**Phase 3: Cutover**
- `orch status` reads from SQLite as primary
- Old 6-system query becomes fallback (if SQLite returns empty)
- Remove fallback after 1 week of clean operation

**Phase 4: Cleanup**
- Remove registry entirely
- Remove old query path from `orch status`
- Remove shadow comparison logic

**Trade-off accepted:** Longer migration (4 phases vs 1) but much lower risk. Accepted because `orch status` is the most-used command — breaking it breaks the orchestrator.

**When this would change:** If we had comprehensive test coverage for `orch status` output, cutover could be faster.

---

### Fork 5: DB Location — Global or Per-Project?

**Decision:** Where does the SQLite database live?

**Options:**
- **A: Global** — `~/.orch/state.db` (single database for all projects)
- **B: Per-project** — `{project}/.orch/state.db` (one database per project)
- **C: Global with project column** — Global DB with `project_dir` as a queryable column.

**Substrate says:**
- **Investigation (field audit):** `orch status` currently does cross-project discovery by querying OpenCode sessions for unique directories. Multi-project view is a core feature.
- **Principle: Local-First:** Files over databases. Single file is simpler.
- **Model: Cross-project visibility:** Agents spawn across projects. Dashboard must show all.

**SUBSTRATE:**
- Field audit: Cross-project discovery is essential for `orch status`
- Principle: Local-First says single file is simpler
- Model: Cross-project visibility requires multi-project view

**RECOMMENDATION:** Option A (Global) — `~/.orch/state.db`.

**Why global:**
- Single query for cross-project `orch status` (no multi-DB ATTACH/JOIN)
- Single SSE materializer connection writes to one DB
- Dashboard serves one DB file
- Simpler backup/migration (one file)
- `project_dir` column enables project-specific queries: `WHERE project_dir = ?`

**Why not per-project (B):**
- `orch status` (the primary consumer) shows all agents across all projects
- Would require opening N SQLite databases and JOINing across them
- SQLite's ATTACH has a limit of ~10 databases
- Adds complexity for the most common query

**Why not option C separately:**
- C is just A with explicit naming. A global DB inherently has a project column.

**Trade-off accepted:** Global DB is not version-controlled per-project. Accepted because this is runtime state, not project source — it's ephemeral and reconstructable from primary sources.

**When this would change:** If projects need truly isolated state (multi-user, multi-machine). Currently Dylan is sole user.

---

### Fork 6: What Happens to Beads' Role?

**Decision:** How does beads' role change with orch owning runtime agent state?

**Options:**
- **A: Beads exits agent lifecycle entirely** — Only tracks issues (create, close, dependencies, labels). No phase, no lifecycle.
- **B: Beads keeps phase as audit trail** — Comments still written for history, but not read by runtime hot path.
- **C: No change** — Beads remains authoritative for everything, orch just caches.

**Substrate says:**
- **Investigation (agents-own-declaration):** Three orthogonal dimensions. Issue Status (beads) is separate from Work Status (phase). This separation is clean.
- **Decision: Registry is spawn-cache:** Demoting registry's role without removal was the correct strategy. Same applies to beads' lifecycle role.
- **Principle: Observation Infrastructure:** Beads comments are valuable historical records for debugging.
- **Model: Agent lifecycle state model:** "Beads is the source of truth for completion." This changes: SQLite becomes source of truth for runtime, beads for issue tracking.

**SUBSTRATE:**
- Investigation: Three orthogonal dimensions — beads owns Issue Status, not Work Status
- Decision: Registry demote pattern works — demote role, don't remove system
- Principle: Observation Infrastructure values the audit trail

**RECOMMENDATION:** Option B (Beads keeps phase as audit trail).

**New role separation:**

| Concern | Authority | System |
|---------|-----------|--------|
| Agent runtime state (phase, is_processing, tokens) | SQLite (`~/.orch/state.db`) | orch |
| Issue lifecycle (open, in_progress, closed) | Beads (`.beads/issues.jsonl`) | beads |
| Phase audit trail (searchable history) | Beads comments | beads |
| Agent identity (workspace, skill, model) | SQLite + workspace manifest | orch |
| Agent completion verification | orch (reads SQLite + runs gates) | orch |

**What beads KEEPS:**
- Issue CRUD (create, close, update, labels, dependencies, priority)
- Comment history (including Phase comments as audit trail)
- `bd show`, `bd ready`, `bd search` — all keep working
- Issue status (open/closed) — canonical for "is this work done?"

**What beads LOSES (from runtime hot path):**
- Phase as runtime source of truth → SQLite
- Phase parsing from comments → `orch phase` writes directly
- Being queried by `orch status` for phase/comments → SQLite read

**Migration for beads:**
- `orch complete` still closes beads issue (issue lifecycle unchanged)
- `bd comment "Phase: X"` still works for audit (agents keep writing)
- `orch status` no longer reads beads comments (reads SQLite instead)
- Daemon auto-complete still uses beads for issue closure

**Trade-off accepted:** Beads comments may lag behind SQLite for phase. Accepted because beads is now audit, not runtime.

**When this would change:** If beads develops real-time event infrastructure (webhooks, SSE).

---

## Final Schema

Based on investigation 1's proposed schema, refined by fork navigation:

```sql
-- ~/.orch/state.db

CREATE TABLE agents (
    -- Core identity (set at spawn, immutable)
    workspace_name  TEXT PRIMARY KEY,
    beads_id        TEXT UNIQUE,
    session_id      TEXT,
    tmux_window     TEXT,
    mode            TEXT NOT NULL DEFAULT 'opencode',  -- 'opencode' | 'claude'
    skill           TEXT,
    model           TEXT,
    tier            TEXT,                               -- 'light' | 'full' | 'orchestrator'
    project_dir     TEXT NOT NULL,
    project_name    TEXT,
    spawn_time      INTEGER NOT NULL,                   -- unix ms
    git_baseline    TEXT,
    issue_title     TEXT,
    issue_type      TEXT,
    issue_priority  INTEGER,

    -- Mutable lifecycle state (event-driven writes)
    phase           TEXT,                               -- 'Planning' | 'Implementing' | 'Complete' | etc.
    phase_summary   TEXT,                               -- text after phase declaration
    phase_reported_at INTEGER,                          -- unix ms
    is_processing   INTEGER DEFAULT 0,                  -- boolean (from SSE session.status)
    session_updated_at INTEGER,                         -- unix ms (from SSE session.updated)
    is_completed    INTEGER DEFAULT 0,                  -- boolean (set by orch complete)
    is_abandoned    INTEGER DEFAULT 0,                  -- boolean (set by orch abandon)
    completed_at    INTEGER,                            -- unix ms
    abandoned_at    INTEGER,                            -- unix ms

    -- Token aggregates (updated by periodic poll during processing)
    tokens_input    INTEGER DEFAULT 0,
    tokens_output   INTEGER DEFAULT 0,
    tokens_reasoning INTEGER DEFAULT 0,
    tokens_cache_read INTEGER DEFAULT 0,
    tokens_total    INTEGER DEFAULT 0,

    -- Timestamps
    created_at      INTEGER NOT NULL,                   -- unix ms (row creation)
    updated_at      INTEGER NOT NULL                    -- unix ms (last row update)
);

-- Indexes for common query patterns
CREATE INDEX idx_agents_beads_id ON agents(beads_id);
CREATE INDEX idx_agents_session_id ON agents(session_id);
CREATE INDEX idx_agents_project ON agents(project_name);
CREATE INDEX idx_agents_active ON agents(is_completed, is_abandoned);  -- for orch status (active agents)
CREATE INDEX idx_agents_phase ON agents(phase);
```

**Derived fields (computed at read time, NOT stored):**
- `is_phantom` = `beads issue open AND session_id IS NULL AND tmux_window IS NULL`
- `is_untracked` = `beads_id IS NULL OR beads_id LIKE '%-untracked-%'`
- `runtime` = `now() - spawn_time`
- `context_risk` = `tokens_total > threshold`
- `swarm.active` = `COUNT(*) WHERE is_completed = 0 AND is_abandoned = 0 AND is_phantom = 0`

---

## Write-Path Diagram

```
SPAWN TIME:
orch spawn
  ├── INSERT INTO agents (all identity fields) ... -- SQLite write (~1ms)
  ├── CreateSession(OpenCode API)                   -- existing
  ├── FallbackCreate(beads)                         -- existing
  ├── CreateWindow(tmux)                            -- existing
  ├── WriteAgentManifest(workspace)                 -- existing
  └── Register(registry)                            -- DEPRECATED (keep during migration)

RUNTIME (SSE MATERIALIZER — runs in daemon/serve):
OpenCode SSE → orch materializer
  ├── session.status: busy
  │     └── UPDATE agents SET is_processing = 1, updated_at = ? WHERE session_id = ?
  ├── session.status: idle
  │     └── UPDATE agents SET is_processing = 0, updated_at = ? WHERE session_id = ?
  │     └── (trigger final token poll)
  ├── session.updated
  │     └── UPDATE agents SET session_updated_at = ?, updated_at = ? WHERE session_id = ?
  └── session.deleted
        └── (no action — we track deletion via orch complete/abandon, not OpenCode)

RUNTIME (PHASE REPORTING — agent-initiated):
orch phase <beads_id> <phase> [summary]
  └── UPDATE agents SET phase = ?, phase_summary = ?, phase_reported_at = ?, updated_at = ? WHERE beads_id = ?

RUNTIME (TOKEN POLLING — periodic for processing agents):
Every 30s for agents WHERE is_processing = 1:
  ├── GET /session/{id} → extract token summary
  └── UPDATE agents SET tokens_input = ?, tokens_output = ?, ... WHERE session_id = ?

COMPLETION:
orch complete <beads_id>
  ├── UPDATE agents SET is_completed = 1, completed_at = ?, updated_at = ? WHERE beads_id = ?
  ├── CloseIssue(beads)                             -- existing
  ├── DeleteSession(OpenCode)                       -- existing
  └── KillWindow(tmux)                              -- existing

ABANDONMENT:
orch abandon <beads_id>
  ├── UPDATE agents SET is_abandoned = 1, abandoned_at = ?, updated_at = ? WHERE beads_id = ?
  ├── DeleteSession(OpenCode)                       -- existing
  └── KillWindow(tmux)                              -- existing

BACKFILL (on SSE reconnect):
First connect: GET /session?start=24h_ago → reconcile against SQLite
Reconnect: For each active agent in SQLite → GET /session/{id} → update mutable fields
```

---

## Migration Plan

### Phase 1: Write Path + Schema (LOE: 2-3 days)

**Goal:** SQLite database exists and is written to, but not read from.

**Tasks:**
1. Create `pkg/state/db.go` — SQLite schema creation, CRUD operations
2. Create `pkg/state/materializer.go` — SSE event handler that writes to SQLite
3. Modify `orch spawn` — INSERT into SQLite after existing writes
4. Modify `orch complete` — UPDATE SQLite after existing operations
5. Modify `orch abandon` — UPDATE SQLite after existing operations
6. Create `orch phase` command — direct SQLite write
7. Wire materializer into `orch serve` and daemon

**Acceptance criteria:**
- `sqlite3 ~/.orch/state.db "SELECT * FROM agents"` shows agent rows
- Spawning an agent creates a row
- Completing/abandoning updates the row
- SSE events update `is_processing`

### Phase 2: Shadow-Read Validation (LOE: 1-2 days)

**Goal:** Validate SQLite matches old multi-source query.

**Tasks:**
1. Add shadow-read to `orch status` — query both paths, compare
2. Log discrepancies to `~/.orch/events.jsonl` with field-level diffs
3. Run for 1-2 weeks of real usage

**Acceptance criteria:**
- Zero discrepancies for identity fields (immutable)
- Mutable field discrepancies are within expected staleness bounds (≤30s)
- No data loss scenarios discovered

### Phase 3: Read Cutover (LOE: 1 day)

**Goal:** `orch status` reads from SQLite as primary source.

**Tasks:**
1. Switch `orch status` to SQLite-first reads
2. Keep old path as fallback (if SQLite returns empty set)
3. Update `orch serve` agent list endpoint to read from SQLite

**Acceptance criteria:**
- `orch status` completes in <100ms (vs current 0.7s-23.6s)
- Dashboard loads agent list from SQLite via API
- Fallback to old path works if DB is empty/missing

### Phase 4: Cleanup (LOE: 1-2 days)

**Goal:** Remove deprecated systems.

**Tasks:**
1. Remove registry (`pkg/registry/`) entirely
2. Remove old multi-source query path from `orch status`
3. Remove shadow comparison logic
4. Update `orch clean` to handle SQLite rows
5. Update CLAUDE.md, guides, models to reflect new architecture

**Acceptance criteria:**
- No imports of `pkg/registry` remain
- `orch status` only reads from SQLite
- All tests pass without registry

---

## Structured Uncertainty

**What's tested:**

- ✅ SSE subscription infrastructure exists and works (verified: `pkg/opencode/monitor.go` + test suite)
- ✅ AGENT_MANIFEST.json already captures most immutable fields (verified: `pkg/spawn/session.go:186-215`)
- ✅ Monitor handles reconnection with automatic retry (verified: `pkg/opencode/monitor.go:79-123`)
- ✅ 70% of agent state is immutable after spawn (verified: field-level audit investigation)
- ✅ Beads comments are O(n) at ~700ms/issue (verified: field-level audit investigation)

**What's untested:**

- ⚠️ Go SQLite library performance under WAL mode with concurrent daemon reads (not benchmarked)
- ⚠️ SSE reconnection gap duration in production (typically 5s, could be longer)
- ⚠️ Token polling at 30s interval provides acceptable freshness (assumption, not validated)
- ⚠️ Shadow-read comparison will catch all discrepancy types (unknown unknowns)

**What would change this:**

- If Go SQLite performance is poor under concurrent access → consider BadgerDB or BoltDB
- If SSE gaps regularly exceed 30s → implement event log in OpenCode for replay
- If agents can't run `orch phase` (e.g., no orch binary in PATH) → fall back to beads-passthrough

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| SQLite as single source | architectural | Cross-component (orch, OpenCode, beads), establishes new pattern |
| SSE materialization (Option B) | architectural | Extends existing infrastructure, no OpenCode fork changes |
| `orch phase` command | architectural | New command, changes agent-orch contract |
| Incremental migration | implementation | How to roll out is tactical, pattern is established |
| Global DB location | implementation | Single-user system, straightforward choice |
| Beads as audit trail | architectural | Changes role boundaries between systems |

### Recommended Approach ⭐

**SSE-Materialized SQLite with `orch phase` Direct Write** — Orch subscribes to OpenCode's SSE stream and materializes session events into `~/.orch/state.db`. Phase reporting moves from beads-passthrough to `orch phase` direct write. Beads retains audit trail role.

**Why this approach:**
- Zero OpenCode fork changes required
- Extends proven SSE/Monitor infrastructure
- Orch owns all writes to its own DB (no dual-write divergence problem)
- Eliminates the 0.7s-23.6s distributed JOIN → ~1ms local read
- Incremental migration ensures no breakage

**Trade-offs accepted:**
- Two phase reporting paths during transition (orch phase + bd comment)
- Token counts up to 30s stale during processing
- Full scan on first connect is slow (~5.8s) but happens once per daemon lifecycle

**Implementation sequence:**
1. Schema + write path — foundational, enables everything else
2. SSE materializer — provides real-time mutable state
3. `orch phase` command — eliminates beads from hot path
4. Shadow-read validation — proves correctness before cutover
5. Read cutover + registry removal — final cleanup

### Alternative Approaches Considered

**Option A: OpenCode Plugin Writes to SQLite**
- **Pros:** Write-time materialization, captures events before orch's SSE subscriber starts
- **Cons:** Dual-write problem (OpenCode JSON + plugin SQLite), plugin runs in OpenCode's process, Go ↔ Bun SQLite concurrency risk
- **When to use instead:** If startup race (events before SSE connects) becomes a real problem in production

**Option C2: Shared SQLite Backend for OpenCode**
- **Pros:** True single source of truth, eliminates Session.list() P0
- **Cons:** Major fork divergence from upstream OpenCode, 1-2 week LOE, breaks external JSON readers
- **When to use instead:** If we decide to deeply integrate storage layers (strategic decision)

**Rationale for recommendation:** Option B has the best risk/reward ratio. It requires no OpenCode changes, extends existing infrastructure, and can be built incrementally. The SSE gap problem is manageable with backfill-on-reconnect.

---

### Implementation Details

**What to implement first:**
- `pkg/state/db.go` — SQLite schema creation using `modernc.org/sqlite` (pure Go, no CGO)
- INSERT path in `orch spawn` — this populates the DB for all new agents
- This alone provides value: `sqlite3 ~/.orch/state.db "SELECT workspace_name, phase, skill FROM agents WHERE is_completed = 0"`

**Things to watch out for:**
- ⚠️ SQLite WAL mode required for concurrent reads (daemon + serve + status)
- ⚠️ `modernc.org/sqlite` is pure Go (no CGO) — slower than `mattn/go-sqlite3` but simpler to build
- ⚠️ Session ID to workspace_name mapping requires workspace scan on first connect (manifest files)
- ⚠️ Beads issue closure must still happen even though SQLite tracks completion (beads is issue tracker, not just lifecycle)

**Areas needing further investigation:**
- SQLite library choice: `modernc.org/sqlite` vs `mattn/go-sqlite3` — benchmark for WAL concurrent reads
- Daemon integration: Should materializer run inside daemon, `orch serve`, or both?
- Token polling implementation: Need to verify GET /session/{id} includes token summary

**Success criteria:**
- ✅ `orch status` returns in <100ms (currently 0.7s-23.6s)
- ✅ Phase updates visible in <5s (currently requires beads comment poll at ~700ms/issue)
- ✅ Zero false positives in shadow-read comparison for identity fields
- ✅ `pkg/registry/` fully removed and no imports remain

---

## Concrete Implementation Issues

### Issue 1: SQLite Schema + Write Path
**Type:** feature | **Priority:** P1 | **LOE:** 2-3 days
- Create `pkg/state/` package with DB schema, migrations, CRUD
- Modify `orch spawn` to INSERT, `orch complete`/`orch abandon` to UPDATE
- Use `modernc.org/sqlite` with WAL mode
- Tests for all write operations

### Issue 2: SSE Materializer Service
**Type:** feature | **Priority:** P1 | **LOE:** 2 days
- Extend Monitor pattern to write to SQLite
- Handle session.status, session.updated events
- Implement backfill-on-reconnect (first connect: full scan; reconnect: active agents only)
- Wire into daemon and `orch serve`

### Issue 3: `orch phase` Command
**Type:** feature | **Priority:** P2 | **LOE:** 1 day
- `orch phase <beads_id> <phase> [summary]` → direct SQLite write
- Update SPAWN_CONTEXT.md template to include `orch phase` instruction
- Keep `bd comment "Phase: X"` as audit trail instruction

### Issue 4: Shadow-Read Migration for `orch status`
**Type:** feature | **Priority:** P2 | **LOE:** 1-2 days
- Add shadow-read to `orch status`: query SQLite alongside old path, compare, log discrepancies
- After validation period: switch to SQLite-primary with old-path fallback
- Update `orch serve` agent list endpoint

### Issue 5: Registry Removal + Cleanup
**Type:** task | **Priority:** P3 | **LOE:** 1-2 days
- Remove `pkg/registry/` entirely
- Remove old multi-source query path
- Remove shadow comparison logic
- Update guides, models, CLAUDE.md

---

## References

**Files Examined:**
- `pkg/opencode/monitor.go` — Existing SSE Monitor with reconnection (proven pattern to extend)
- `pkg/opencode/sse.go` — SSE client, event parsing, session.status handling
- `pkg/spawn/session.go:186-215` — AgentManifest struct (current immutable field schema)
- `cmd/orch/serve_agents.go` — Current status calculation (~1400 lines)
- `cmd/orch/status_agents.go:193-213` — AgentManifest reading in status command

**External Documentation:**
- `modernc.org/sqlite` — Pure Go SQLite driver (no CGO dependency)
- SQLite WAL mode — Concurrent reader support for Go processes

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-12-registry-is-spawn-cache.md` — Establishes registry as replaceable
- **Investigation:** `.kb/investigations/2026-02-06-inv-agent-state-field-level-audit.md` — Complete field map
- **Investigation:** `.kb/investigations/2026-02-06-inv-opencode-fork-audit-session-lifecycle-integration.md` — Integration options
- **Investigation:** `.kb/investigations/2026-02-04-inv-agents-own-declaration-via-bd.md` — Three orthogonal dimensions
- **Guide:** `.kb/guides/agent-lifecycle.md` — Current lifecycle documentation (needs update after migration)

---

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This decision resolves the P0 `orch status` performance issue (5.8s session list)
- This decision establishes SQLite as the agent state authority
- Future spawns might try to use registry or beads for state that should come from SQLite

**Suggested blocks keywords:**
- "agent state", "sqlite", "state database"
- "orch status performance", "session list slow"
- "registry replacement", "materialized view"

---

## Investigation History

**2026-02-06 13:05:** Investigation started
- Context: Spawned from orch-go-21364 to design single-source agent state architecture
- Inputs: Two completed investigations (field-level audit + OpenCode fork audit)

**2026-02-06 13:15:** Substrate consultation complete
- Read principles.md, agent-lifecycle-state-model, registry-is-spawn-cache decision
- Verified primary sources: monitor.go, sse.go, session.go (AgentManifest)

**2026-02-06 13:30:** All 6 forks navigated with substrate trace
- Fork 1: Hybrid phase reporting (orch-direct + beads audit)
- Fork 2: Hybrid backfill (full scan first, targeted on reconnect)
- Fork 3: Periodic token polling (30s for processing agents)
- Fork 4: Incremental migration with shadow-read
- Fork 5: Global DB at ~/.orch/state.db
- Fork 6: Beads keeps audit trail, loses runtime authority

**2026-02-06 13:45:** Investigation completed
- Key outcome: Complete architecture design with schema, write-path diagram, 4-phase migration plan, and 5 implementation issues
