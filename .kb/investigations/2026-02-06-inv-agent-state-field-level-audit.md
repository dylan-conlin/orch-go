# Investigation: Agent State Read/Write Field-Level Audit

**Date:** 2026-02-06
**Status:** Complete
**Purpose:** Map every field of agent state that orch reads/writes, across all commands, as the requirements doc for a single-source SQLite schema.

---

## Summary

`orch status` performs a distributed JOIN across 6 systems at query time. This investigation traces every field of agent state to document: (1) what system it comes from, (2) what writes it, (3) freshness requirements, (4) whether it's derived or authoritative, and (5) which commands consume it.

---

## 1. Systems Inventory

| # | System               | Storage                                               | Access Method                                                             | Latency                                                  |
|---|----------------------|-------------------------------------------------------|---------------------------------------------------------------------------|----------------------------------------------------------|
| 1 | **OpenCode API**     | In-memory + disk (`~/.local/share/opencode/storage/`) | HTTP REST (`/session`, `/session/{id}/message`)                           | 5-50ms per call; 5.8s for unfiltered list of 89 sessions |
| 2 | **Beads**            | `.beads/issues.jsonl` per project                     | RPC via Unix socket (`bd.sock`) or CLI fallback (`bd show/comments/list`) | ~700ms per issue (RPC); ~200ms batch list                |
| 3 | **Tmux**             | Runtime (volatile)                                    | Subprocess (`tmux list-sessions/list-windows/display-message`)            | <50ms per call                                           |
| 4 | **Registry**         | `~/.orch/agent-registry.json`                         | Direct file read with flock                                               | <10ms                                                    |
| 5 | **Workspace (disk)** | `.orch/workspace/{name}/` per project                 | Direct file read                                                          | <5ms                                                     |
| 6 | **Anthropic API**    | External                                              | HTTP (`api.anthropic.com`)                                                | ~200ms per account                                       |

---

## 2. Complete Field-Level Map

### 2.1 Core Identity Fields

| Field            | Type   | Source System     | Written By                                                               | Freshness                        | Derived?      | Consumed By                                                  |
|------------------|--------|-------------------|--------------------------------------------------------------------------|----------------------------------|---------------|--------------------------------------------------------------|
| `session_id`     | string | OpenCode API      | `orch spawn` (headless: CreateSession; tmux: FindRecentSessionWithRetry) | stale-ok (immutable after spawn) | Authoritative | status, complete, abandon, resume, send, tail, wait, serve   |
| `beads_id`       | string | Beads             | `orch spawn` (spawn_beads.go: FallbackCreate)                            | stale-ok (immutable after spawn) | Authoritative | status, complete, abandon, review, frontier, serve, question |
| `workspace_name` | string | Disk              | `orch spawn` (creates directory `.orch/workspace/{name}/`)               | stale-ok (immutable after spawn) | Authoritative | status, complete, review, tail, question                     |
| `tmux_window`    | string | Tmux              | `orch spawn` (tmux.CreateWindow)                                         | real-time (window can be killed) | Authoritative | status, complete, abandon, tail                              |
| `mode`           | string | Registry/Manifest | `orch spawn` (Registry.Register, WriteAgentManifest)                     | stale-ok (immutable)             | Authoritative | status, serve                                                |

### 2.2 Session State Fields (from OpenCode API)

| Field                      | Type       | Source System       | Written By                                                                                           | Freshness            | Derived?                                                  | Consumed By                                              |
|----------------------------|------------|---------------------|------------------------------------------------------------------------------------------------------|----------------------|-----------------------------------------------------------|----------------------------------------------------------|
| `session.title`            | string     | OpenCode API        | `orch spawn` (BuildSpawnCommand --title); `orch claim` (UpdateSessionTitle)                          | stale-ok             | Authoritative                                             | status (extractBeadsIDFromTitle), serve                  |
| `session.directory`        | string     | OpenCode API        | OpenCode (on session create)                                                                         | stale-ok             | Authoritative                                             | status (cross-project resolution), serve                 |
| `session.time.created`     | int64 (ms) | OpenCode API        | OpenCode (on session create)                                                                         | stale-ok (immutable) | Authoritative                                             | status (runtime calc), serve, frontier (stuck detection) |
| `session.time.updated`     | int64 (ms) | OpenCode API        | OpenCode (on any activity)                                                                           | seconds              | Authoritative                                             | status (idle filtering, compact mode), serve             |
| `is_processing`            | bool       | OpenCode API + Tmux | Derived: last assistant message has `finish==""` AND `completed==0`; OR tmux pane has active process | real-time            | **Derived** (from messages endpoint + tmux pane activity) | status, serve                                            |
| `model`                    | string     | OpenCode API        | Derived: most recent assistant message's `modelID` field                                             | seconds              | **Derived**                                               | status, serve                                            |
| `tokens.input_tokens`      | int        | OpenCode API        | Derived: sum of all messages' `tokens.input`                                                         | seconds              | **Derived** (aggregated)                                  | status, serve, complete (transcript export)              |
| `tokens.output_tokens`     | int        | OpenCode API        | Derived: sum of all messages' `tokens.output`                                                        | seconds              | **Derived** (aggregated)                                  | status, serve                                            |
| `tokens.reasoning_tokens`  | int        | OpenCode API        | Derived: sum of all messages' `tokens.reasoning`                                                     | seconds              | **Derived** (aggregated)                                  | status, serve                                            |
| `tokens.cache_read_tokens` | int        | OpenCode API        | Derived: sum of all messages' `tokens.cache.read`                                                    | seconds              | **Derived** (aggregated)                                  | status, serve                                            |
| `tokens.total_tokens`      | int        | OpenCode API        | Derived: `input + output + reasoning`                                                                | seconds              | **Derived** (aggregated)                                  | status (context risk), serve                             |
| `last_activity`            | time.Time  | OpenCode API        | Derived from `session.time.updated`                                                                  | seconds              | **Derived**                                               | status (ghost filtering)                                 |

### 2.3 Beads State Fields

| Field                | Type       | Source System  | Written By                                                                                                                     | Freshness            | Derived?                                                      | Consumed By                                       |
|----------------------|------------|----------------|--------------------------------------------------------------------------------------------------------------------------------|----------------------|---------------------------------------------------------------|---------------------------------------------------|
| `issue.status`       | string     | Beads          | `orch complete` (CloseIssue/CloseIssueForce); `orch review` (batch close); daemon (auto-complete); reconcile (forceCloseIssue) | minutes              | Authoritative                                                 | status (IsCompleted), serve, frontier, complete   |
| `issue.title`        | string     | Beads          | `orch spawn` (FallbackCreate)                                                                                                  | stale-ok (immutable) | Authoritative                                                 | status (Task column), serve, frontier             |
| `issue.issue_type`   | string     | Beads          | `orch spawn` (FallbackCreate)                                                                                                  | stale-ok             | Authoritative                                                 | frontier (skill inference)                        |
| `issue.labels`       | []string   | Beads          | `orch spawn` (FallbackCreate with labels); daemon (FallbackAddLabel/RemoveLabel)                                               | minutes              | Authoritative                                                 | frontier (triage filtering), daemon (ready query) |
| `issue.dependencies` | json       | Beads          | Manual (`bd dep add`) or `orch spawn` with --parent                                                                            | stale-ok             | Authoritative                                                 | frontier (decidability), spawn (blocking check)   |
| `issue.priority`     | int        | Beads          | `orch spawn` (FallbackCreate)                                                                                                  | stale-ok             | Authoritative                                                 | frontier (sort), daemon (ready query)             |
| `issue.created_at`   | string     | Beads          | Beads (on create)                                                                                                              | stale-ok (immutable) | Authoritative                                                 | frontier                                          |
| `issue.closed_at`    | string     | Beads          | Beads (on close)                                                                                                               | minutes              | Authoritative                                                 | frontier (stuck agent filtering)                  |
| `phase`              | string     | Beads comments | Agent (via `bd comment "Phase: X"`); recovered from ACTIVITY.json                                                              | minutes              | **Derived** (parsed from comments via regex `Phase:\s*(\w+)`) | status, serve, complete, review, frontier         |
| `phase_reported_at`  | *time.Time | Beads comments | Derived from comment timestamp                                                                                                 | minutes              | **Derived**                                                   | status (compact mode age filter)                  |
| `phase_summary`      | string     | Beads comments | Agent (text after phase declaration)                                                                                           | minutes              | **Derived**                                                   | serve                                             |

### 2.4 Tmux State Fields

| Field                  | Type   | Source System | Written By                                                                                | Freshness | Derived?      | Consumed By                                                       |
|------------------------|--------|---------------|-------------------------------------------------------------------------------------------|-----------|---------------|-------------------------------------------------------------------|
| `window_exists`        | bool   | Tmux          | `orch spawn` (CreateWindow creates); `orch complete`/`orch abandon` (KillWindow destroys) | real-time | Authoritative | status (agent discovery Phase 1), serve                           |
| `window_name`          | string | Tmux          | `orch spawn` (BuildWindowName with emoji + workspace + beadsID)                           | stale-ok  | Authoritative | status (extractBeadsIDFromWindowName, extractSkillFromWindowName) |
| `window_target`        | string | Tmux          | `orch spawn` (CreateWindow returns `session:index`)                                       | stale-ok  | Authoritative | status, complete, abandon, tail                                   |
| `pane_current_command` | string | Tmux          | Tmux runtime                                                                              | real-time | Authoritative | status (IsPaneProcessRunning → IsProcessing fallback)             |
| `pane_active`          | bool   | Tmux          | Derived: `current_command` not in shell list (bash/zsh/sh/fish)                           | real-time | **Derived**   | status (IsProcessing override)                                    |

### 2.5 Registry Fields

| Field             | Type       | Source System | Written By                                                 | Freshness            | Derived?                              | Consumed By                         |
|-------------------|------------|---------------|------------------------------------------------------------|----------------------|---------------------------------------|-------------------------------------|
| `reg.id`          | string     | Registry      | `orch spawn` (Register)                                    | stale-ok (immutable) | Authoritative (spawn metadata)        | status (Phase 1.5 discovery)        |
| `reg.beads_id`    | string     | Registry      | `orch spawn` (Register)                                    | stale-ok             | Copy of beads_id                      | status, abandon                     |
| `reg.mode`        | string     | Registry      | `orch spawn` (Register)                                    | stale-ok             | Copy                                  | status                              |
| `reg.session_id`  | string     | Registry      | `orch spawn` (Register)                                    | stale-ok             | Copy                                  | abandon (session lookup)            |
| `reg.tmux_window` | string     | Registry      | `orch spawn` (Register)                                    | stale-ok             | Copy                                  | status (verified against live tmux) |
| `reg.status`      | AgentState | Registry      | `orch spawn` (active); deprecated: Abandon/Complete/Remove | stale-ok             | Authoritative for registry layer only | status (ListActive filter)          |
| `reg.project_dir` | string     | Registry      | `orch spawn` (Register)                                    | stale-ok             | Copy                                  | status (cross-project)              |
| `reg.skill`       | string     | Registry      | `orch spawn` (Register)                                    | stale-ok             | Copy                                  | status                              |
| `reg.model`       | string     | Registry      | `orch spawn` (Register)                                    | stale-ok             | Copy                                  | status                              |
| `reg.spawned_at`  | string     | Registry      | `orch spawn` (Register)                                    | stale-ok (immutable) | Copy                                  | —                                   |
| `reg.updated_at`  | string     | Registry      | `orch spawn` (Register); merge logic                       | stale-ok             | Used for conflict resolution          | —                                   |

### 2.6 Workspace (Disk) Fields

| Field                 | Type | Source System | Written By                                          | Freshness                        | Derived?                           | Consumed By                                                                                        |
|-----------------------|------|---------------|-----------------------------------------------------|----------------------------------|------------------------------------|----------------------------------------------------------------------------------------------------|
| `SPAWN_CONTEXT.md`    | file | Disk          | `orch spawn` (writeSpawnContext)                    | stale-ok (immutable after spawn) | Authoritative                      | complete (VerifyCompletionFull), question, workspace lookup                                        |
| `AGENT_MANIFEST.json` | file | Disk          | `orch spawn` (WriteAgentManifest); `orch claim`     | stale-ok (immutable)             | Authoritative (canonical identity) | status (readAgentManifest for skill, projectDir, mode, model), complete (git diff baseline), serve |
| `.tier`               | file | Disk          | `orch spawn` (writes "light"/"full"/"orchestrator") | stale-ok (immutable)             | Authoritative                      | complete (ReadTierFromWorkspace)                                                                   |
| `.session_id`         | file | Disk          | `orch spawn` (writes OpenCode session ID)           | stale-ok                         | Copy                               | resume, tail, complete, abandon                                                                    |
| `.beads_id`           | file | Disk          | `orch spawn` (writes beads issue ID)                | stale-ok                         | Copy                               | complete, tail                                                                                     |
| `.spawn_time`         | file | Disk          | `orch spawn` (writes nanosecond timestamp)          | stale-ok (immutable)             | Copy                               | —                                                                                                  |
| `.process_id`         | file | Disk          | `orch spawn` (WriteProcessID)                       | stale-ok                         | Authoritative (for cleanup)        | complete, abandon (process termination)                                                            |
| `SYNTHESIS.md`        | file | Disk          | Agent (during work)                                 | minutes                          | Authoritative                      | complete (VerifySynthesis, ValidateHandoffContent), review                                         |
| `ACTIVITY.json`       | file | Disk          | Agent activity tracking                             | seconds                          | Authoritative                      | complete (Phase: Complete recovery)                                                                |

### 2.7 Derived/Computed Fields (not stored)

| Field              | Type   | Derived From                                                                       | Freshness | Consumed By             |
|--------------------|--------|------------------------------------------------------------------------------------|-----------|-------------------------|
| `is_phantom`       | bool   | beads issue open + no session + no tmux window                                     | real-time | status, serve           |
| `is_completed`     | bool   | `issue.status == "closed"`                                                         | minutes   | status, serve           |
| `is_untracked`     | bool   | session has no beads ID in title OR beads ID matches `project-untracked-*` pattern | real-time | status, serve           |
| `runtime`          | string | `now - session.time.created`                                                       | real-time | status, serve           |
| `context_risk`     | object | `tokens.total_tokens` > threshold + `project_dir` git status + `is_processing`     | real-time | status, serve           |
| `source`           | string | Which discovery phase found it (T=tmux, O=opencode, B=beads, W=workspace)          | real-time | status                  |
| `project`          | string | Extracted from beads ID prefix (e.g., `orch-go-xxxx` → `orch-go`)                  | stale-ok  | status, serve           |
| `skill`            | string | Extracted from workspace name OR manifest OR window name                           | stale-ok  | status, serve, frontier |
| `swarm.active`     | int    | Count of non-phantom, non-completed agents                                         | real-time | status, serve           |
| `swarm.processing` | int    | Count of agents with `is_processing=true`                                          | real-time | status, serve           |

### 2.8 Infrastructure/Account Fields

| Field                    | Type    | Source System                | Written By                          | Freshness | Derived?      | Consumed By   |
|--------------------------|---------|------------------------------|-------------------------------------|-----------|---------------|---------------|
| `account.used_percent`   | float64 | Anthropic API                | Anthropic (external)                | minutes   | Authoritative | status, serve |
| `account.reset_time`     | string  | Anthropic API                | Anthropic (external)                | minutes   | Authoritative | status, serve |
| `daemon.status`          | string  | `~/.orch/daemon-status.json` | daemon (writes periodically)        | seconds   | Authoritative | status        |
| `daemon.last_poll`       | string  | daemon-status.json           | daemon                              | seconds   | Authoritative | status        |
| `daemon.capacity`        | struct  | daemon-status.json           | daemon                              | seconds   | Authoritative | status, serve |
| `infra.opencode_running` | bool    | TCP connect test             | Derived (connect to localhost:4096) | real-time | **Derived**   | status        |
| `infra.beads_running`    | bool    | TCP/socket connect test      | Derived (connect to bd.sock)        | real-time | **Derived**   | status        |

---

## 3. Command → Field Access Matrix

### 3.1 Read Operations

| Command      | OpenCode Sessions                   | OpenCode Messages               | Beads Issues                                  | Beads Comments            | Tmux                                                      | Registry                | Workspace Disk                                                      | Anthropic API      |
|--------------|:-----------------------------------:|:-------------------------------:|:---------------------------------------------:|:-------------------------:|:---------------------------------------------------------:|:-----------------------:|:-------------------------------------------------------------------:|:------------------:|
| **status**   | ✅ ListSessionsWithOpts             | ✅ GetSessionEnrichment         | ✅ GetIssuesBatch                             | ✅ GetCommentsBatch       | ✅ ListWorkersSessions, ListWindows, IsPaneProcessRunning | ✅ ListActive           | ✅ readAgentManifest, findWorkspaceByBeadsID                        | ✅ getAccountUsage |
| **serve**    | ✅ ListSessions, GetSession         | ✅ GetMessages, GetLastActivity | ✅ GetIssuesBatch                             | ✅ GetCommentsBatch       | ✅ ListWindows                                            | ✅ ListActive           | ✅ readAgentManifest                                                | ✅                 |
| **complete** | ✅ GetSession, GetSessionTokens     | ✅ ExportSessionTranscript      | ✅ FallbackShow                               | ✅ (via VerifyCompletion) | ✅ FindWindowByBeadsID                                    | —                       | ✅ findWorkspaceByBeadsID, ReadTierFromWorkspace, ReadAgentManifest | —                  |
| **spawn**    | ✅ CreateSession, FindRecentSession | —                               | ✅ FallbackCreate                             | —                         | ✅ EnsureWorkersSession, CreateWindow                     | ✅ Register, Save       | ✅ WriteAgentManifest, SPAWN_CONTEXT.md                             | —                  |
| **abandon**  | ✅ GetSession                       | —                               | ✅ FallbackShow                               | ✅ (phase check)          | ✅ FindWindowByBeadsID, KillWindow                        | ✅ Find                 | ✅ findWorkspaceByBeadsID                                           | —                  |
| **frontier** | ✅ ListSessions                     | —                               | ✅ FallbackReady, FallbackShow                | ✅                        | —                                                         | —                       | —                                                                   | —                  |
| **review**   | —                                   | —                               | ✅ (batch)                                    | ✅ GetCommentsBatch       | —                                                         | —                       | ✅ findWorkspaceByBeadsID, SYNTHESIS.md                             | —                  |
| **resume**   | ✅ GetSession                       | —                               | —                                             | —                         | ✅ FindWindowByWorkspaceName                              | —                       | ✅ .session_id                                                      | —                  |
| **send**     | ✅ SendMessageAsync                 | —                               | —                                             | —                         | —                                                         | —                       | ✅ .session_id                                                      | —                  |
| **tail**     | ✅ GetMessages                      | ✅                              | —                                             | —                         | ✅ GetPaneContent                                         | —                       | ✅ .session_id, findWorkspaceByBeadsID                              | —                  |
| **wait**     | ✅ IsSessionProcessing              | ✅                              | ✅ FallbackShow                               | ✅                        | —                                                         | —                       | ✅ .session_id, findWorkspaceByBeadsID                              | —                  |
| **question** | —                                   | —                               | —                                             | ✅ FallbackAddComment     | —                                                         | —                       | ✅ findWorkspaceByBeadsID                                           | —                  |
| **clean**    | ✅ ListDiskSessions, DeleteSession  | —                               | —                                             | —                         | ✅ KillWindow                                             | ✅ ListCleanable, Purge | ✅ workspace scan                                                   | —                  |
| **claim**    | ✅ UpdateSessionTitle               | —                               | ✅ FallbackCreate                             | —                         | —                                                         | —                       | ✅ WriteAgentManifest                                               | —                  |
| **daemon**   | ✅ ListSessions                     | ✅ GetSessionEnrichment         | ✅ FallbackReady, FallbackShow, FallbackClose | ✅                        | ✅                                                        | ✅                      | ✅ findWorkspaceByBeadsID                                           | —                  |

### 3.2 Write Operations

| Command       | OpenCode                                 | Beads                                                                  | Tmux                   | Registry              | Workspace Disk                                                                                 |
|---------------|------------------------------------------|------------------------------------------------------------------------|------------------------|-----------------------|------------------------------------------------------------------------------------------------|
| **spawn**     | CreateSession, SendPrompt                | FallbackCreate (issue)                                                 | CreateWindow, SendKeys | Register + Save       | SPAWN_CONTEXT.md, AGENT_MANIFEST.json, .tier, .session_id, .beads_id, .spawn_time, .process_id |
| **complete**  | DeleteSession                            | CloseIssue/CloseIssueForce; FallbackAddComment                         | KillWindow             | — (deprecated)        | Archive workspace to `archived/`                                                               |
| **abandon**   | DeleteSession                            | — (issue stays open)                                                   | KillWindow             | — (deprecated)        | —                                                                                              |
| **send**      | SendMessageAsync                         | —                                                                      | —                      | —                     | —                                                                                              |
| **resume**    | SendMessageAsync                         | —                                                                      | —                      | —                     | —                                                                                              |
| **question**  | —                                        | FallbackAddComment                                                     | —                      | —                     | —                                                                                              |
| **claim**     | UpdateSessionTitle                       | FallbackCreate (new issue)                                             | —                      | —                     | AGENT_MANIFEST.json                                                                            |
| **review**    | —                                        | CloseIssueForce (batch)                                                | —                      | —                     | —                                                                                              |
| **clean**     | DeleteSession                            | —                                                                      | KillWindow             | Purge + SaveSkipMerge | Remove workspace directories                                                                   |
| **daemon**    | CreateSession, SendPrompt, DeleteSession | FallbackClose, FallbackAddComment, FallbackRemoveLabel, FallbackUpdate | CreateWindow           | Register + Save       | SPAWN_CONTEXT.md, AGENT_MANIFEST.json, etc.                                                    |
| **reconcile** | —                                        | FallbackUpdate, forceCloseIssue                                        | —                      | —                     | —                                                                                              |

---

## 4. Freshness Requirements by Concern

| Concern                          | Required Freshness | Current Source                                  | Notes                                           |
|----------------------------------|--------------------|-------------------------------------------------|-------------------------------------------------|
| "Is agent processing right now?" | Real-time (<1s)    | OpenCode messages endpoint + tmux pane activity | Most expensive query (full message fetch)       |
| "Is agent done?"                 | Minutes            | Beads issue.status + Phase comments             | Authoritative but slow (~700ms per issue)       |
| "What model is agent using?"     | Seconds            | OpenCode messages (last assistant modelID)      | Could be cached at spawn time                   |
| "How many tokens used?"          | Seconds            | OpenCode messages (aggregated)                  | Expensive: scans all messages                   |
| "Is agent visible?"              | Real-time          | Tmux window exists                              | Fast (<50ms)                                    |
| "What skill/task?"               | Stale-ok           | Workspace manifest + beads issue title          | Immutable after spawn — perfect cache candidate |
| "Account usage?"                 | Minutes            | Anthropic API                                   | External, not cacheable locally                 |
| "What phase?"                    | Minutes            | Beads comments (parsed)                         | Must query per-issue                            |

---

## 5. SQLite Schema Implications

### 5.1 Fields writable at spawn time (immutable after)

These fields are set once by `orch spawn` and never change. They are the strongest candidates for a local materialized view because they never need refresh:

- `workspace_name`, `beads_id`, `session_id`, `tmux_window`
- `mode`, `skill`, `model`, `tier`
- `project_dir`, `project_name`
- `spawn_time`, `git_baseline`
- `issue_title`, `issue_type`, `issue_priority`

### 5.2 Fields that change during lifecycle

These need event-driven writes or periodic polling:

| Field                | Write Trigger                      | Proposed Writer                       |
|----------------------|------------------------------------|---------------------------------------|
| `phase`              | Agent runs `bd comment "Phase: X"` | Beads webhook / SSE listener          |
| `phase_reported_at`  | Same as phase                      | Same                                  |
| `is_processing`      | Message activity                   | OpenCode SSE event stream             |
| `tokens.*`           | Message activity                   | OpenCode SSE event stream (aggregate) |
| `session_updated_at` | Any session activity               | OpenCode SSE event stream             |
| `is_completed`       | `orch complete` runs               | `orch complete` writes directly       |
| `is_abandoned`       | `orch abandon` runs                | `orch abandon` writes directly        |
| `window_exists`      | Window created/killed              | Tmux hooks or polling                 |
| `account_usage`      | Anthropic rate window              | Periodic polling (every 60s)          |

### 5.3 Fields that are purely derived (compute at read time, don't store)

- `is_phantom` (= beads open + no session + no window)
- `is_untracked` (= no beads ID in title)
- `runtime` (= now - spawn_time)
- `context_risk` (= token thresholds)
- `source` (= discovery method)
- `swarm.*` counts (= aggregations)

### 5.4 Proposed Minimal Schema

```sql
CREATE TABLE agents (
    -- Core identity (set at spawn, immutable)
    workspace_name  TEXT PRIMARY KEY,
    beads_id        TEXT UNIQUE,
    session_id      TEXT,
    tmux_window     TEXT,
    mode            TEXT NOT NULL,     -- 'opencode' | 'claude' | 'docker'
    skill           TEXT,
    model           TEXT,
    tier            TEXT,              -- 'light' | 'full' | 'orchestrator'
    project_dir     TEXT,
    project_name    TEXT,
    spawn_time      INTEGER NOT NULL,  -- unix ms
    git_baseline    TEXT,
    issue_title     TEXT,
    issue_type      TEXT,
    issue_priority  INTEGER,

    -- Mutable lifecycle state (event-driven writes)
    phase           TEXT,              -- 'Planning', 'Implementing', 'Complete', etc.
    phase_reported_at INTEGER,         -- unix ms
    is_processing   INTEGER DEFAULT 0, -- boolean
    session_updated_at INTEGER,        -- unix ms
    is_completed    INTEGER DEFAULT 0, -- boolean (set by orch complete)
    is_abandoned    INTEGER DEFAULT 0, -- boolean (set by orch abandon)
    completed_at    INTEGER,           -- unix ms
    abandoned_at    INTEGER,           -- unix ms

    -- Token aggregates (updated on message events)
    tokens_input    INTEGER DEFAULT 0,
    tokens_output   INTEGER DEFAULT 0,
    tokens_reasoning INTEGER DEFAULT 0,
    tokens_cache_read INTEGER DEFAULT 0,
    tokens_total    INTEGER DEFAULT 0,

    -- Timestamps
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL
);

CREATE INDEX idx_agents_beads_id ON agents(beads_id);
CREATE INDEX idx_agents_session_id ON agents(session_id);
CREATE INDEX idx_agents_project ON agents(project_name);
CREATE INDEX idx_agents_phase ON agents(phase);
```

### 5.5 Migration Path

1. **Write path**: `orch spawn` writes to SQLite + existing systems. `orch complete`/`abandon` update SQLite.
2. **Event listener**: SSE stream from OpenCode → update `is_processing`, `tokens_*`, `session_updated_at`.
3. **Beads listener**: Poll or webhook → update `phase`, `phase_reported_at`, `is_completed`.
4. **Read path**: `orch status` becomes `SELECT ... FROM agents WHERE ...` (~1ms) instead of 6-system distributed JOIN (~730ms-23.6s).

---

## 6. Key Observations

1. **70% of status fields are immutable after spawn.** The most expensive queries (session enrichment, beads comments) re-derive data that was known at spawn time. A write-time cache eliminates the majority of read-time cost.

2. **Only 3 fields need real-time freshness:** `is_processing`, `window_exists`, and `session_updated_at`. Everything else can tolerate seconds to minutes of staleness.

3. **Registry is redundant.** Every field in `registry.Agent` is a subset of what's in the proposed SQLite schema. The registry can be deleted once SQLite is the source.

4. **Beads comments are the most expensive read.** At ~700ms per issue, they're O(n) in agent count. Caching `phase` on write (when agent runs `bd comment`) would eliminate this entirely.

5. **The workspace AGENT_MANIFEST.json already has most immutable fields.** The proposed schema extends this with mutable lifecycle state.

---

## References

- `.kb/investigations/2026-02-06-inv-opencode-session-list-performance-cliff.md` — P0 incident that motivated this audit
- `.kb/models/agent-lifecycle-state-model.md` — Four-layer state model
- `.kb/guides/status.md` — Status command architecture
- `.kb/decisions/2026-01-12-registry-is-spawn-cache.md` — Registry design decision
