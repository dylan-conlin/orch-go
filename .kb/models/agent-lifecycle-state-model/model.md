# Model: Agent Lifecycle State Model

**Domain:** Agent Lifecycle / State Management
**Last Updated:** 2026-02-27
**Synthesized From:** 17 investigations (Dec 20, 2025 - Jan 6, 2026) into agent state, completion detection, cross-project visibility, and dashboard status display. Updated Feb 2026 after major restructuring (registry elimination, two-lane architecture, single-pass query engine). Updated Feb 25 after phase-based liveness, cross-project querying, and verification gate additions. Updated Feb 27 after V0-V3 verification levels, all-at-once gate failure reporting, architectural choices gate, auto-implementation issue creation, and `pkg/agent/` formal types package.

---

## Summary (30 seconds)

Agent state exists across **four independent layers** (tmux windows, OpenCode in-memory, OpenCode on-disk, beads comments). These layers fall into two distinct categories: **state layers** (beads, workspace files) that represent what work was done, and **infrastructure layers** (OpenCode sessions, tmux windows) that represent transient execution resources. The lifecycle state machine is now formally codified in `pkg/agent/` with typed states (7), transitions (6), and validation. The dashboard reconciles these via a **Priority Cascade**: check beads issue status first (highest authority), then Phase comments, then SYNTHESIS.md existence, then session status. Agents are discovered via a **two-lane architecture**: tracked work (beads-first via `queryTrackedAgents`) and untracked sessions (OpenCode session list). Status can appear "wrong" at the dashboard level while being "correct" at each individual layer - this is a measurement artifact from combining multiple sources of truth.

---

## Core Mechanism

### Four-Layer State Model

Agent state is distributed across four independent systems:

| Layer                  | Category           | Storage                            | Lifecycle           | What It Knows                   | Authority Level        |
| ---------------------- | ------------------ | ---------------------------------- | ------------------- | ------------------------------- | ---------------------- |
| **Beads comments**     | **State**          | `.beads/issues.jsonl`              | Persistent          | Phase transitions, metadata     | Highest (canonical)    |
| **Workspace files**    | **State**          | `.orch/workspace/`                 | Persistent          | SPAWN_CONTEXT, SYNTHESIS, .tier | High (artifact record) |
| **OpenCode on-disk**   | **Infrastructure** | `~/.local/share/opencode/storage/` | Persistent (no TTL) | Full message history            | Medium (historical)    |
| **OpenCode in-memory** | **Infrastructure** | Server process                     | Until restart       | Session ID, current status      | Medium (operational)   |
| **Tmux windows**       | **Infrastructure** | Runtime (volatile)                 | Until window closed | Agent visible, window ID        | Low (UI only)          |

**Key insight:** A registry (`~/.orch/registry.json`) was historically a fifth layer attempting to cache all four, which caused drift. It was **eliminated entirely** in Feb 2026 (see two-lane ADR). The solution is to query authoritative sources directly via a single-pass query engine with explicit reason codes for any missing data. Architecture lint tests structurally prevent registry recreation.

### Formal State Machine (`pkg/agent/`)

The lifecycle state machine is codified in `pkg/agent/` (added Feb 27, 2026). This package provides typed states, transitions, and validation — replacing the previously implicit state model spread across `determineAgentStatus()` and `complete_cmd.go`.

**States (7):**

| State | Type | Description |
|-------|------|-------------|
| `spawning` | Transient | During `orch spawn` execution |
| `active` | Normal | Agent is working |
| `phase_complete` | Normal | Agent declared done, awaiting `orch complete` |
| `completing` | Transient | During `orch complete` execution |
| `completed` | Terminal | Beads issue closed |
| `abandoned` | Terminal | Beads reset to open, respawnable |
| `orphaned` | Detected | GC finds in_progress with no live execution |

**Transitions (6):**

```
spawning → active         (spawn)
active → phase_complete   (phase_complete)
phase_complete → completed (complete, via completing)
active → abandoned        (abandon)
orphaned → completed      (force_complete)
orphaned → abandoned      (force_abandon)
```

**Key types:**

- `AgentRef` — Query handle assembled from authoritative sources (NOT stored state). Contains BeadsID, WorkspaceName, SessionID, ProjectDir, SpawnMode.
- `TransitionEvent` — Records a state change with its side effects (`EffectResult` per subsystem: beads, opencode, tmux, workspace, events). Tracks critical vs non-critical failures.
- `SpawnInput` — Lifecycle-relevant spawn parameters, decoupled from content generation (`spawn.Config`). Validates required fields and converts to `AgentRef`.
- `SpawnHandle` — Two-phase spawn pattern: `BeginSpawn()` returns a handle with rollback capability; caller creates session/window between phases; `ActivateSpawn()` finalizes the Spawning → Active transition. Accumulates `EffectResult`s across both phases into a single `TransitionEvent`.
- `LifecycleManager` interface — Coordinates transitions across all four layers. Does NOT store agent state (Invariant #7). Methods: `BeginSpawn`, `ActivateSpawn`, `Complete`, `Abandon`, `ForceComplete`, `ForceAbandon`, `DetectOrphans`, `CurrentState`.
- `OrphanDetectionResult` / `OrphanedAgent` — GC scan results with reason codes and retry recommendations.

**Implementation status (Feb 27):** `lifecycle_impl.go` exists with partial implementation (compilation error: missing `ActivateSpawn` method). Types, interfaces, tests, and filters are complete and tested.

**Filter functions** (`filters.go`):

- `IsActiveForConcurrency()` — 1-hour threshold; running always counts, Phase: Complete never counts
- `IsVisibleByDefault()` — 4-hour threshold; running and Phase: Complete always visible

**Architectural constraint:** `LifecycleManager` is a coordinator, not a cache. After any method returns, the manager holds no agent state. This enforces Invariant #7 at the type level.

### Source of Truth by Concern

Different questions have different authoritative sources:

| Question                | Source                           | NOT this                 |
| ----------------------- | -------------------------------- | ------------------------ |
| Is agent complete?      | Beads issue `status = closed`    | OpenCode session exists  |
| What phase is agent in? | Beads comments (`Phase: X`)      | Dashboard shows "active" |
| Did agent finish?       | `Phase: Complete` comment exists | Session went idle        |
| Is agent processing?    | SSE `session.status = busy`      | Session exists           |
| Is agent visible?       | Tmux window exists               | Session exists           |

**Beads is the source of truth for completion.** OpenCode sessions persist to disk indefinitely. Session existence means nothing about whether the agent is done. Only beads matters.

### State vs Infrastructure: Why This Distinction Matters

The four-layer model (above) conflates two fundamentally different concerns. Recognizing the difference clarifies what orch should _own_ versus what it should merely _use_.

**State layers** (beads comments, workspace files) represent _what work was done_ and _what phase it's in_. They are persistent, orch-controlled, and survive infrastructure restarts. Orch owns their lifecycle entirely.

**Infrastructure layers** (OpenCode sessions, tmux windows) represent _execution resources_. They are transient, externally-controlled (by OpenCode server and tmux respectively), and have no inherent connection to work completion. Orch uses them but doesn't control their lifecycle.

**The reconciliation burden comes from treating infrastructure as state.** When orch tries to derive agent status from session existence or tmux window presence, it must constantly reconcile infrastructure reality against state truth. This is the root cause of phantom agents (tmux window exists but session exited), ghost sessions (OpenCode session persists after work completed), and orphan infrastructure (resources with no corresponding state).

**Ownership model (Own / Accept / Lobby):**

| Bucket     | What                                                                        | Implication                                                |
| ---------- | --------------------------------------------------------------------------- | ---------------------------------------------------------- |
| **Own**    | State layers (beads, workspaces), verification gates, skill integration     | Orch's domain — design, maintain, evolve                   |
| **Accept** | Infrastructure constraints (sessions persist, no metadata API, SSE-only)    | Work within them — periodic cleanup, workspace-as-metadata |
| **Lobby**  | Missing infrastructure features (session TTL, metadata API, state endpoint) | File upstream — would eliminate reconciliation burden      |

See: `.kb/decisions/2026-02-13-lifecycle-ownership-boundaries.md` for the full decision and implementation plan.

### Two-Lane Discovery Architecture

Tracked work and untracked sessions are queried via separate paths (Feb 2026):

| Lane | Query Path | What's Visible | Source of Truth |
|------|------------|----------------|-----------------|
| **Tracked work** | `orch status`, dashboard `/api/agents` | Agents with beads_id | Beads issues |
| **Untracked sessions** | `orch sessions`, `/api/sessions` | Orchestrator sessions, ad-hoc, `--no-track` | OpenCode session list |

The single-pass query engine (`queryTrackedAgents()` in `cmd/orch/query_tracked.go`) replaces ad-hoc multi-source reconciliation. Every missing field has an explicit reason code (`MissingBinding`, `MissingSession`, `SessionDead`, `MissingPhase`).

**Cross-project querying (Feb 25):** `queryTrackedAgents()` accepts `projectDirs []string` and queries beads across all known project directories (not just local). This ensures cross-project agents (e.g., toolshed issues tracked from orch-go) are visible in status and work-graph. Each directory is queried independently with graceful degradation on failure.

**Phase-based liveness for claude-backend agents (Feb 24-25):** Claude CLI agents have no OpenCode session, so the query engine uses a third liveness strategy: phase comments as heartbeat. When `SpawnMode == "claude"` and `SessionID == ""`:
- Phase: Complete reported → status "completed"
- Any phase reported → status "active" (reason: `phase_reported`)
- Recently spawned (<5 min) → status "active" (reason: `recently_spawned`)
- No phase, not recent → status "dead" (reason: `no_phase_reported`)

This replaced a prior tmux liveness check (orch-go-1182→1183→1185) that violated Invariant 6 by using tmux windows as state. See probe: `probes/2026-02-24-probe-tmux-liveness-two-lane-violation.md`.

See: `.kb/decisions/2026-02-18-two-lane-agent-discovery.md`

### State Transitions

**Normal lifecycle:**

```
spawned (orch spawn)
    ↓
AGENT_MANIFEST.json written to workspace (binding: beads_id ↔ session_id ↔ project_dir)
OpenCode session created
Beads issue created/tagged (Status: open)
Tmux window created (if --tmux)
    ↓
working (agent executes task)
    ↓
Phase transitions reported via bd comment
"Phase: Planning" → "Phase: Implementing" → "Phase: Complete"
    ↓
Phase: Complete reached (agent declares done)
SYNTHESIS.md written (if full tier)
Git commits created
    ↓
orch complete runs (orchestrator verification)
Verification uses V0-V3 levels (read from workspace manifest, default V1):
    V0: phase_complete
    V1: + synthesis, session_handoff, handoff_content, skill_output, phase_gate,
        constraint, decision_patch_limit, architectural_choices
    V2: + test_evidence, git_diff, build, vet, accretion
    V3: + visual_verification, explain_back
All gate failures collected and reported at once (no early-return)
Runs knowledge maintenance (kb maintenance check)
Closes beads issue (Status: closed)
Auto-creates implementation issue for architect completions (triage:ready)
Exports session activity to workspace (ACTIVITY.json)
Defers tmux window cleanup (prevents phantom windows)
    ↓
completed (dashboard shows blue badge)
Session may remain in OpenCode storage
Tmux window may remain open
```

**Awaiting-cleanup path:**

```
spawned → working → Phase: Complete reported
    ↓
Agent session dies (context exhaustion, crash)
    ↓
determineAgentStatus: Phase: Complete + session dead → "awaiting-cleanup"
    ↓
Dashboard shows awaiting-cleanup (needs orch complete)
```

**Abandoned path:**

```
spawned → running
    ↓
orch abandon (human judgment)
    ↓
Beads issue remains open (NOT closed)
    ↓
Dashboard shows abandoned (yellow badge)
Session remains in OpenCode
```

### Critical Invariants

1. **Phase: Complete is agent's declaration** - Only agent can reach this, not orchestrator
2. **Beads issue closed = canonical completion** - All status queries defer to beads
3. **Session existence ≠ agent still working** - Sessions persist indefinitely
4. **Status checks don't mutate state** - `determineAgentStatus()` is a pure function, no side effects
5. **Multiple sources must be reconciled** - No single source has complete truth; query engine joins with reason codes
6. **Tmux windows are UI layer only** - Not authoritative for state
7. **No persistent lifecycle caches** - Only in-memory, process-local caches with short TTLs allowed. Disk-backed state (registry, sessions.json, state.db) is structurally prohibited by architecture lint tests
8. **Silent failures must be visible** - Every missing field gets an explicit reason code, never empty metadata

---

## Why This Fails

### Failure Mode 1: Dashboard Shows "Active" When Agent is Done

**Symptom:** Dashboard shows agent as active, but `bd show <id>` says status=closed

**Root cause:** Dashboard caching or SSE lag - hasn't received beads update yet

**Why it happens:**

- Agent reaches Phase: Complete
- `orch complete` closes beads issue
- Beads issue status = closed
- Dashboard hasn't refreshed or polled beads yet
- Dashboard still shows cached "active" state

**Fix:** Refresh dashboard browser tab (forces beads query)

**NOT the fix:** Deleting OpenCode session (treats symptom, not cause)

### Failure Mode 2: Completed Agents Showing Wrong Status

**Symptom:** Agent completed work but dashboard shows unexpected status

**Root cause:** Completion signals exist but session is dead, creating ambiguity

**How the Priority Cascade handles this (current):**

- If beads issue closed → "completed" (Priority 1, regardless of session state)
- If Phase: Complete + session dead → "awaiting-cleanup" (Priority 2, needs orch complete)
- If Phase: Complete + session alive → "completed" (Priority 3)
- If SYNTHESIS.md exists + session dead → "awaiting-cleanup" (Priority 4)
- If SYNTHESIS.md exists + session alive → "completed" (Priority 5)

**Fix (Jan 8, refined Feb 2026):** Priority Cascade puts beads/Phase check before session existence check. The `awaiting-cleanup` status (added Feb 2026) distinguishes completed-but-orphaned agents from truly dead agents.

### Failure Mode 3: Agent Went Idle But Not Complete

**Symptom:** Session status is "idle" but no `Phase: Complete` comment

**Root cause:** Agent ran out of context, crashed, or didn't follow completion protocol

**Why it happens:**

- Session exhausts context (150k tokens)
- Agent stops responding
- SSE event: `session.status = idle`
- No `Phase: Complete` was ever written
- Dashboard shows "idle" or "waiting"

**This is expected behavior.** Session idle ≠ work complete. Only agents that explicitly run `bd comment <id> "Phase: Complete"` are considered done.

**Fix:** Check workspace for what agent accomplished, then either:

- `orch complete <id> --force` if work is done
- `orch abandon <id>` if work is incomplete

### Failure Mode 4: Cross-Project Agents Not Visible

**Symptom:** Agent spawned with `--workdir /other/project` doesn't appear in dashboard

**Root cause:** Dashboard only scans current project's `.orch/workspace/` directory

**Why it happens:**

- Workspace created in `/other/project/.orch/workspace/`
- Dashboard running from `orch-go` only sees `orch-go/.orch/workspace/`
- Cross-project discovery requires querying OpenCode sessions for unique directories

**Fix (Jan 6, updated Feb 25):** `queryTrackedAgents()` accepts `projectDirs []string` and queries beads + workspace manifests across all known project directories. No cache — direct queries with graceful degradation per directory.

---

## Constraints

### Why Four Layers Instead of Single Source of Truth?

**Constraint:** Each layer serves a distinct purpose with different lifecycle requirements

**Implication:** State must be reconciled by combining sources, not stored in one place

**Breakdown (state layers — orch owns):**

- **Beads** - Work tracking (survives everything, multi-session)
- **Workspace files** - Artifact record (SPAWN_CONTEXT, SYNTHESIS, tier metadata)

**Breakdown (infrastructure layers — orch uses):**

- **OpenCode disk** - Message history (debugging, resume)
- **OpenCode memory** - Real-time processing state (fast queries)
- **Tmux** - Visual monitoring (orchestrator needs to SEE work)

**This enables:** Each layer optimized for its purpose
**This constrains:** Must reconcile at query time (eventual consistency). The reconciliation burden is heaviest when infrastructure layers are treated as authoritative — they should be consulted only as fallback, after state layers.

### Why Can't We Infer Completion from Session State?

**Constraint:** Sessions go idle for many reasons (paused, waiting, crashed, context exhausted, completed)

**Implication:** Only explicit `Phase: Complete` signal is reliable

**Workaround:** Agents must follow completion protocol

**This enables:** Agents can pause/wait without being marked complete
**This constrains:** Agents that crash without reporting phase look "incomplete"

### Why Registry Was Eliminated (Historical)

**Background:** A registry (`~/.orch/registry.json`) attempted to cache all four layers, but updates were async and incomplete. This caused 6 weeks of drift bugs (Dec 21 - Feb 18, 2026).

**Root cause of drift:**

- Beads issues closed via `bd close` (not `orch complete`) → registry not updated
- OpenCode sessions persist → registry shows "dead" when session exists
- Tmux windows close → registry still shows "running"

**Resolution (Feb 18, 2026):** Registry eliminated entirely. Replaced by single-pass query engine that queries beads → workspace manifests → OpenCode directly. Architecture lint tests (`architecture_lint_test.go`) structurally prevent recreation of `pkg/registry/`, `pkg/cache/`, `registry.json`, `sessions.json`, or `state.db`.

See: `.kb/decisions/2026-02-18-two-lane-agent-discovery.md`

---

## Pressure Points

Common feature requests or operational changes that would violate this model's invariants. Use these to evaluate whether a proposed change is architecturally safe.

| If Asked To... | Architectural Risk | Invariant at Risk |
|----------------|-------------------|-------------------|
| Cache agent state locally (registry, projection DB, state.db) | Drift from authoritative sources; recreates the 6-week registry cycle (Dec 21 - Feb 18) | #7: No persistent lifecycle caches |
| Infer completion from session idle/dead | False positives; session idle has many causes (paused, waiting, crashed, context exhausted) | #3: Session existence ≠ agent still working |
| Add a fifth state layer | Reconciliation complexity increases quadratically; every new layer drifts independently | #5: Multiple sources must be reconciled |
| Use tmux window presence for agent status | Tmux is UI-only; windows can close independently of work completion | #6: Tmux windows are UI layer only |
| Mutate state during status checks | Side effects in queries cause unpredictable state transitions and ordering bugs | #4: Status checks don't mutate state |
| Skip reason codes for missing data | Silent failures accumulate; operators can't distinguish "not found" from "not checked" | #8: Silent failures must be visible |
| Let workers close their own beads issues | Bypasses verification gates; breaks the orchestrator-reviews-worker hierarchy | #2: Beads issue closed = canonical completion |

---

## Evolution

**Dec 20-21, 2025: Initial Implementation**

- Basic agent tracking via registry
- Tmux windows as primary UI
- OpenCode sessions for execution

**Dec 22-26, 2025: State Reconciliation Issues**

- "Dead" agents that actually completed
- "Active" agents when beads said closed
- Registry drift discovered

**Jan 4-6, 2026: Four-Layer Model**

- Investigation `2026-01-04-design-dashboard-agent-status-model.md` proposed Priority Cascade
- Beads established as canonical source for completion
- Registry demoted to metadata only

**Jan 6, 2026: Cross-Project Visibility**

- Multi-project workspace discovery
- Directory extraction from OpenCode sessions
- Beads queries routed to correct project

**Jan 12, 2026: Model Synthesis**

- 17 investigations synthesized into this model
- Four-layer architecture formalized
- Constraints made explicit

**Feb 13, 2026: State vs Infrastructure Distinction**

- Four-layer table reframed with Category column (State vs Infrastructure)
- Workspace files added as explicit state layer
- New section explaining why conflating state and infrastructure creates reconciliation burden
- Three-bucket ownership model (Own/Accept/Lobby) referenced
- Decision: `.kb/decisions/2026-02-13-lifecycle-ownership-boundaries.md`

**Feb 18, 2026: Major Restructuring — Registry Elimination & Two-Lane Architecture**

- Registry (`~/.orch/registry.json`, `pkg/session/registry.go`) eliminated entirely (529+ lines removed)
- Single-pass query engine (`queryTrackedAgents()`) built in `cmd/orch/query_tracked.go`
- Two-lane architecture: tracked work (beads-first) vs untracked sessions (OpenCode-first)
- `serve_agents.go` (~1700 lines) extracted into 8+ smaller files (`serve_agents_*.go`)
- Architecture lint tests added to prevent registry recreation
- `AGENT_MANIFEST.json` replaces registry entries as workspace-local binding
- Decision: `.kb/decisions/2026-02-18-two-lane-agent-discovery.md`
- Priority Cascade expanded with `awaiting-cleanup` status
- Verification suite expanded to 14 gates (git diff, accretion, build, visual, test evidence, etc.) → 16 gates (vet Feb 25, architectural_choices Feb 27)

**Feb 20-25, 2026: Liveness Rethink & Gate Expansion**

- Phase-based liveness replaces tmux liveness for claude-backend agents (orch-go-1182→1183→1185)
  - Tmux liveness check violated Invariant 6 (tmux as state) and caused dashboard oscillation
  - Phase comments now serve as heartbeat: any reported phase = active, no phase + not recent = dead
  - See probe: `probes/2026-02-24-probe-tmux-liveness-two-lane-violation.md`
- Cross-project querying: `queryTrackedAgents()` now queries beads across all known project directories (orch-go-1231)
- `GateVet` added to completion verification (orch-go-1248)
- `GateArchitecturalChoices` added at V1 (16 gates total)
- Knowledge maintenance step added to `orch complete` flow (orch-go-1243)
- Deferred tmux window cleanup prevents phantom windows during completion
- Completion backlog detection and stall tracking added to dashboard monitoring
- TTL-based beads cache (`serve_agents_cache.go`) prevents excessive bd process spawning

**Feb 27, 2026: Formal Types Package & Gate Expansion**

- `pkg/agent/` created with typed lifecycle states, transitions, and validation (`types.go`, `lifecycle.go`, `filters.go`)
- 7 states formalized: spawning, active, phase_complete, completing, completed, abandoned, orphaned
- 6 transitions with `ValidateTransition()` function enforcing from→to rules
- `LifecycleManager` interface defines coordinator pattern (no stored state, enforcing Invariant #7)
- `AgentRef` as query handle (not stored state), `TransitionEvent` with per-effect tracking
- `OrphanDetectionResult` types for GC-initiated lifecycle operations
- `GateArchitecturalChoices` added at V1 (16 gates total across V0-V3)
- Auto-implementation issue creation for architect completions
- Contract tests (`contract_two_lane_test.go`) enforce 12-scenario acceptance matrix
- Verify package cleanup: `git_commits.go`, `synthesis_content.go`, `stale_bug.go` deleted (logic consolidated)
- Deleted packages: `pkg/registry/`, `pkg/servers/`, `pkg/experiment/`, `pkg/usage/`

---

## References

**Key Investigations:**

- `2026-01-04-design-dashboard-agent-status-model.md` - Priority Cascade design
- `2026-01-06-inv-cross-project-agent-visibility.md` - Multi-project discovery
- `2025-12-26-inv-registry-drift-analysis.md` - Why registry caching failed
- `2025-12-22-inv-completion-detection-race-condition.md` - Session idle ≠ complete
- `2026-02-18-design-agent-observability-rethink.md` - Beads-first design leading to two-lane
- ...and 13 others

**Decisions Informed by This Model:**

- Beads as canonical source of truth (completion)
- Priority Cascade for status calculation
- Four-layer architecture (no single source)
- Two-lane agent discovery (`.kb/decisions/2026-02-18-two-lane-agent-discovery.md`)
- Registry elimination — structurally prevented from returning
- State vs infrastructure distinction (`.kb/decisions/2026-02-13-lifecycle-ownership-boundaries.md`)

**Related Models:**

- `.kb/models/archived/dashboard-agent-status.md` - How Priority Cascade calculates status (archived — superseded by this model + dashboard-architecture)
- `.kb/models/opencode-session-lifecycle/model.md` - How OpenCode sessions work
- `.kb/models/spawn-architecture/model.md` - How agents are created
- `.kb/models/completion-verification/model.md` - How verification gates work

**Related Guides:**

- `.kb/guides/agent-lifecycle.md` - How to use agent lifecycle commands (procedural)
- `.kb/guides/completion.md` - How to complete agents (procedural)
- `.kb/guides/status.md` - How to use orch status (procedural)

**Primary Evidence (Verify These):**

- `pkg/agent/types.go` - Formal lifecycle states (7), transitions (6), `AgentRef`, `TransitionEvent`, `EffectResult`, orphan detection types
- `pkg/agent/lifecycle.go` - `LifecycleManager` interface (coordinator, not cache), client interfaces (`BeadsClient`, `OpenCodeClient`, `TmuxClient`, `EventLogger`, `WorkspaceManager`)
- `pkg/agent/spawn.go` - `SpawnInput` (lifecycle spawn parameters), `SpawnHandle` (two-phase spawn with rollback)
- `pkg/agent/lifecycle_impl.go` - Partial `LifecycleManager` implementation (WIP: missing `ActivateSpawn`)
- `pkg/agent/filters.go` - `IsActiveForConcurrency()`, `IsVisibleByDefault()` with threshold-based filtering
- `cmd/orch/serve_agents_status.go` - Priority Cascade implementation (`determineAgentStatus()`), stall tracking, completion backlog detection
- `cmd/orch/query_tracked.go` - Single-pass query engine (`queryTrackedAgents()`), phase-based liveness for claude-backend agents, cross-project beads querying
- `cmd/orch/serve_agents_handlers.go` - Dashboard API handlers
- `cmd/orch/serve_agents_discovery.go` - Workspace and investigation discovery
- `cmd/orch/serve_agents_cache.go` - TTL-based beads cache (prevents excessive bd process spawning from dashboard polls)
- `pkg/verify/check.go` - Completion verification (16 gates across V0-V3, including `GateArchitecturalChoices`)
- `pkg/verify/level.go` - V0-V3 gate level definitions and `ShouldRunGate()` query function
- `cmd/orch/complete_cmd.go` - Completion pipeline orchestrator (knowledge maintenance step, deferred tmux cleanup)
- `cmd/orch/architecture_lint_test.go` - Structural guardrails preventing registry recreation (forbidden packages/files)
- `cmd/orch/contract_two_lane_test.go` - 12-scenario acceptance matrix enforcing two-lane architecture contract
- `.beads/issues.jsonl` - Canonical completion source
