# Model: Agent Lifecycle State Model

**Domain:** Agent Lifecycle / State Management
**Last Updated:** 2026-03-12
**Synthesized From:** 17 investigations (Dec 20, 2025 - Jan 6, 2026) into agent state, completion detection, cross-project visibility, and dashboard status display. Updated Feb 2026 after major restructuring (registry elimination, two-lane architecture, single-pass query engine). Updated Feb 25 after phase-based liveness, cross-project querying, and verification gate additions. Updated Feb 27 after V0-V3 verification levels, all-at-once gate failure reporting, architectural choices gate, auto-implementation issue creation, and `pkg/agent/` formal types package. Updated Feb 28 after full LifecycleManager implementation, lifecycle adapter wiring (complete, abandon, orphan GC), model impact advisory, and daemon orphan recovery. Updated late Feb 28 for model drift correction: rework lifecycle path (`orch rework` bypasses LifecycleManager), phase timeout detection, `TrackedIssue`/`WorkspaceInfo` types, completion pipeline extraction (architect/cleanup/hotspot), daemon periodic task expansion, status_cmd.go extraction, and new commands (rework, review orphans, orient).

---

## Summary (30 seconds)

Agent state exists across **four independent layers** (tmux windows, OpenCode in-memory, OpenCode on-disk, beads comments). These layers fall into two distinct categories: **state layers** (beads, workspace files) that represent what work was done, and **infrastructure layers** (OpenCode sessions, tmux windows) that represent transient execution resources. The lifecycle state machine is formally codified in `pkg/agent/` with typed states (7), transitions (6), and validation. Additionally, `orch rework` provides a **completed → active** path that bypasses `LifecycleManager` by directly manipulating beads status (reopen + rework label + new workspace). The dashboard reconciles layers via a **Priority Cascade**: check beads issue status first (highest authority), then Phase comments, then SYNTHESIS.md existence, then session status. Agents are discovered via a **two-lane architecture**: tracked work (beads-first via `queryTrackedAgents`) and untracked sessions (OpenCode session list). **Phase timeout detection** (30-minute threshold in `status_cmd.go`, daemon-driven periodic checks in `pkg/daemon/phase_timeout.go`) flags agents that stop reporting phase updates as unresponsive. Status can appear "wrong" at the dashboard level while being "correct" at each individual layer - this is a measurement artifact from combining multiple sources of truth.

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

**AGENT_MANIFEST.json is the canonical workspace state artifact.** Written by ALL backends (OpenCode headless, OpenCode+tmux, and Claude CLI) via `spawn.WriteAgentManifest()`, it consolidates all spawn-time state (beads_id, tier, spawn_mode, spawn_time, project_dir, skill, model, git_baseline) into a single self-describing JSON file. Individual dotfiles (`.beads_id`, `.tier`, `.spawn_time`, `.spawn_mode`) are legacy duplication — read AGENT_MANIFEST.json first, fall back to dotfiles for backward compatibility. Infrastructure handles (session_id for OpenCode, tmux window target for Claude CLI) are NOT in AGENT_MANIFEST.json — they are mutable/discoverable references, not immutable spawn-time state.

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
- `TrackedIssue` — Beads issue returned by `ListByLabel`, used by `DetectOrphans` to find agents tagged with `orch:agent`.
- `WorkspaceInfo` — Workspace metadata (name, path, beads ID, session ID, spawn mode, spawn time), used by `DetectOrphans` to join workspace data with beads issues.

**Implementation status (Feb 28):** `lifecycle_impl.go` is **fully implemented** (662 lines) with all LifecycleManager methods: `BeginSpawn`, `ActivateSpawn`, `Complete`, `Abandon`, `ForceComplete`, `ForceAbandon`, `DetectOrphans`, `CurrentState`. Comprehensive test suite in `lifecycle_manager_test.go` (~1,726 lines). The cmd layer bridges to `pkg/agent` via `lifecycle_adapters.go` which provides concrete implementations of `BeadsClient`, `OpenCodeClient`, `TmuxClient`, `EventLogger`, and `WorkspaceManager` interfaces, wrapping the real packages (`pkg/beads`, `pkg/opencode`, `pkg/tmux`, `pkg/events`, `pkg/spawn`) with compile-time interface compliance checks.

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
orch complete runs (orchestrator verification, wired to LifecycleManager.Complete)
Verification uses V0-V3 levels (read from workspace manifest, default V1):
    V0: phase_complete
    V1: + synthesis, handoff_content, skill_output, phase_gate,
        constraint, decision_patch_limit, architectural_choices
    V2: + test_evidence, git_diff, build, vet, accretion
    V3: + visual_verification, explain_back
    (session_handoff checked inline for orchestrator tier, not in level system)
All gate failures collected and reported at once (no early-return)
Runs knowledge maintenance (kb maintenance check)
Surfaces model impact advisory (cross-references synthesis keywords against .kb/models/)
Closes beads issue (Status: closed) via LifecycleManager.Complete
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
orch abandon (human judgment, wired to LifecycleManager.Abandon)
    ↓
Beads issue remains open (NOT closed)
    ↓
Dashboard shows abandoned (yellow badge)
Session remains in OpenCode
```

**Orphan recovery path (Feb 28):**

```
spawned → working → stalled (no phase reported, no live execution)
    ↓
Daemon orphan detection (periodic scan, 2h threshold)
  OR orch clean --orphans (manual GC)
    ↓
LifecycleManager.DetectOrphans() scans for in_progress + no liveness signals
    ↓
ForceAbandon or ForceComplete based on evidence
```

**Rework path (Feb 27):**

```
completed (beads closed, workspace archived)
    ↓
orch rework <beads-id> "feedback" --bypass-triage
    ↓
Reopens beads issue (closed → open → in_progress)
Adds rework:<N> label
Records REWORK #N comment
Finds prior workspace (archived) for synthesis extraction
Creates NEW workspace with rework context (prior synthesis + feedback)
Spawns fresh agent via orch spawn pipeline
    ↓
active (new agent working on same beads issue)
```

**Note:** Rework does NOT go through `LifecycleManager`. It directly manipulates beads status via `verify.UpdateIssueStatus()` and spawns through the standard `orch.DispatchSpawn()` pipeline. This means the formal state machine (6 transitions) doesn't cover rework — it's an out-of-band transition from a terminal state back to active. The `events.LogAgentReworked()` event captures this for tracking.

**Phase timeout detection (Feb 28):**

```
spawned → working → no phase update for 30+ minutes
    ↓
status_cmd.go flags as "unresponsive" (phaseTimeoutThreshold = 30min)
Daemon periodic scan (pkg/daemon/phase_timeout.go) detects + escalates
    ↓
Dashboard shows timeout warning
Orchestrator investigates (may abandon or send message)
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
9. **Claude-backend agents use phase comments as liveness proxy, NOT tmux window checks** - Tmux liveness check violates Invariant 6 and causes dashboard oscillation (structural, not fixable). Phase comments come from beads (authoritative), require zero additional data fetching, and are already queried. A tmux window existence check was tried (orch-go-1182/1183) and reverted.
10. **Tmux window mutating operations must use stable `@ID`, not `session:index`** - Window indices shift when `renumber-windows` is on or concurrent completions run. All four kill/cleanup code paths in `complete_cleanup.go`, `clean_cmd.go`, `abandon_cmd.go`, `review.go` should use `KillWindowByID(window.ID)`. The `WindowInfo.Target` field is for display/logs only.

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

### Failure Mode 5: Scan Ordering Pre-empts Tmux Discovery (Historical)

**Symptom:** Dashboard shows Claude CLI tmux agent as "completed" with no phase, no window.

**Root cause (pre-two-lane architecture):** Completed workspace scan ran before tmux scan. Both found the same agent by beads_id, but workspace scan claimed it first as "completed" without beads enrichment. Tmux scan then skipped the agent as a duplicate.

**Why it happened:**
- Claude CLI agent creates workspace (SPAWN_CONTEXT.md) AND tmux window
- Workspace scan found SPAWN_CONTEXT.md → marked agent "completed"
- Workspace scan optimization: did NOT add beads_id to enrichment queue
- Tmux scan found window → duplicate check matched beads_id → skipped
- Beads enrichment never ran → Phase: Planning comment invisible

**Resolution:** Superseded by two-lane architecture (Feb 2026) and phase-based liveness for claude-backend agents. In the current architecture, phase comments are the authoritative liveness signal for claude-backend agents, not scan ordering.

### Failure Mode 6: QUESTION Deadlock (Structural Gap)

**Symptom:** Agent reports Phase: QUESTION, waits indefinitely — no mechanism delivers the answer.

**Root cause:** Headless agents have no channel to receive answers. Daemon monitors bd comments for phase transitions, but there's no inbox or delivery path from orchestrator to a running headless session.

**Why it happens:**
- Agent discovers ambiguity, reports `bd comment "Phase: QUESTION - ..."` as instructed
- Orchestrator may not be monitoring or may answer too late
- Agent context exhausts while waiting

**Scale (Feb 2026 audit):** 5 confirmed stalls from QUESTION deadlock, all GPT-5.2-codex, all same issue (orch-go-fq5).

**Fix:** Use `orch send <session-id> "answer"` to inject answers into running sessions. For daemon-spawned agents, the daemon should detect Phase: QUESTION and pause/notify the orchestrator rather than continuing to schedule work.

### Failure Mode 7: SYNTHESIS Compliance Gap (Protocol Gap)

**Symptom:** Agent completes work (Phase: Complete comment written) but no SYNTHESIS.md exists. Dashboard shows agent as awaiting-cleanup or stalled rather than completed.

**Root cause:** Agent follows the phase protocol but doesn't write the synthesis artifact. This is a prompt compliance failure, not a lifecycle failure.

**Scale (Feb 2026 audit):** 194 agents (out of 441 manifested archived agents without SYNTHESIS.md) reported Phase: Complete but created no SYNTHESIS.md. This is NOT a stall — the agent finished; it just skipped the artifact.

**Fix:** V1 gate (`GateSynthesis`) in `pkg/verify/check.go` rejects completion without SYNTHESIS.md. This surfaces the gap at `orch complete` time rather than silently accepting it.

### Failure Mode 8: Duplicate Spawn Storm (Daemon Restart Race)

**Symptom:** Same beads issue spawned 5-10x in rapid succession, creating N-1 wasted agent sessions and workspace pollution.

**Root cause:** Daemon in-memory dedup cache doesn't survive restarts. If the daemon restarts while issues are in the `triage:ready` queue, the same issues appear fresh and spawn again.

**Scale (Feb 2026 audit):** 5 agents spawned for orch-go-dr0u within 2 minutes; 10 retries on another slug.

**Fix:** Dedup at the beads layer (check for existing `orch:agent` label + open workspace) rather than relying on in-memory cache. An existing `orch:agent` label on an open issue means it's already been spawned.

### Failure Mode 9: Model Protocol Incompatibility

**Symptom:** Non-Anthropic model agent stalls in Implementing, Planning, or QUESTION phase at dramatically higher rates than Anthropic models.

**Root cause:** GPT-4o and GPT-5.2-codex cannot reliably follow the multi-step worker protocol (phase reporting → implementation → SYNTHESIS → completion). This is a protocol compliance gap, not a capability gap.

**Scale (Feb 2026 audit):** 15 of 19 true stalls (79%) were non-Anthropic models. GPT-4o: 87.5% stall rate. GPT-5.2-codex: 67.5%. Opus: 44.6% (inflated by pre-protocol agents).

**Fix:** Restrict protocol-heavy skills (architect, investigation) to Anthropic models. The diagnostic classifier (`pkg/daemon/diagnostic.go`) detects this pattern and recommends respawning with an Anthropic model.

### Failure Mode 10: Silent Failure

**Symptom:** Agent has a manifest but never reported any phase. Beads issue has 0 comments from the agent.

**Root cause:** Agent crashed on startup (API error, context exhaustion from large spawn context), or was spawned with a pre-phase-reporting skill version.

**Scale (Feb 2026 audit):** 228 agents. Predominantly Sonnet 4.5 from Feb 14-17 (pre-phase-reporting skill versions).

**Fix:** Phase timeout detection (`pkg/daemon/phase_timeout.go`) catches agents that have sessions but haven't reported. The diagnostic classifier flags agents with no phase and no session as silent failures after 30 minutes.

### Failure Mode 11: Prior Art Confusion

**Symptom:** Agent discovers that prior agents already completed overlapping work mid-task, gets confused about remaining scope, and stalls in Exploration.

**Root cause:** Spawn context doesn't include information about prior agent completions for the same beads issue or similar work.

**Scale (Feb 2026 audit):** 1 confirmed instance (orch-go-nn43), but the pattern is systematic — daemon spawns without checking prior completions.

**Fix:** Prior art check at spawn time — check if prior agents completed overlapping work and inject a "Prior Completions" section into SPAWN_CONTEXT.md. The diagnostic classifier flags agents stalled in Exploration with known prior agents.

### Failure Mode 12: Concurrency Ceiling Stall

**Symptom:** Agent reports Phase: BLOCKED because it needs to spawn sub-tasks but hits concurrency limits. Waits indefinitely for slot availability.

**Root cause:** No escalation path from BLOCKED to orchestrator attention. Agent correctly surfaces the constraint but the system has no automated response.

**Scale (Feb 2026 audit):** 1 confirmed instance (orch-go-1054).

**Fix:** The diagnostic classifier detects BLOCKED phase and recommends waiting for slot or escalating to orchestrator.

### Diagnostic Classifier (`pkg/daemon/diagnostic.go`)

A unified classifier maps agents to failure modes 1-12. Given a `DiagnosticAgent` (populated from beads, workspace, and session data), `ClassifyFailureMode()` returns a `DiagnosticResult` with the failure mode, severity, and recommended actions. `RunDiagnostics()` produces an aggregate report across all agents, grouped by failure mode.

The classifier runs in priority order: QUESTION deadlock → concurrency ceiling → silent failure → prior art confusion → model incompatibility → generic phase stall → synthesis compliance gap.

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

### Why Non-Anthropic Models Have High Stall Rates

**Constraint:** Non-Anthropic models have structurally higher failure-to-complete rates due to protocol instruction-following fidelity, not capability.

**Observed stall rates (Feb 2026 audit, archived agents):**

| Model | Stall Rate | Notes |
|-------|-----------|-------|
| Opus | 44.6% | Inflated by pre-protocol era |
| Sonnet | 68.3% | Inflated by pre-protocol era (133 pre-phase-reporting agents) |
| GPT-5.2-codex | 67.5% | Protocol compliance failures dominate |
| GPT-4o | 87.5% | Highest failure rate |

**Root cause:** Worker-base skill protocol (phase reporting, SYNTHESIS creation, bd comment discipline) is a multi-step protocol that requires consistent instruction-following across many tool calls. Non-Anthropic models fail to maintain protocol fidelity at significantly higher rates.

**Implication:** For protocol-heavy skills (architect, investigation, feature-impl), use Anthropic models. Non-Anthropic models are appropriate for simpler, bounded tasks where stall recovery cost is low.

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
| Route rework through LifecycleManager | Currently bypassed intentionally — rework is a terminal→active transition not in the 6-transition model; formalizing would add complexity for a low-frequency operation | State machine completeness |
| Add Claude Code plan mode to feature-impl Planning phase | Plan mode creates a "dark period" (bash blocked → no bd comments possible), hangs headless/daemon-spawned agents at approval gate, and defaults to clearing SPAWN_CONTEXT. Feature-impl's Investigation/Design phases are superior: observable, headless-compatible, produce durable artifacts. | Continuous observability assumption |
| Kill tmux windows by session:index (unstable) | Window indices shift when `renumber-windows` is enabled or concurrent completions race. TOCTOU bug can kill wrong window. Use `KillWindowByID(window.ID)` for all mutating operations. | #10: Stable @ID for tmux mutations |
| Rely on CLI fallback path for closed-issue filtering | The CLI fallback (`bd list -l orch:agent`) includes closed issues without status filtering. If the RPC path is unavailable, `/api/agents` may surface completed work as "tracked". | #2: Beads issue closed = canonical completion |

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
- Verification suite expanded to 16 gates across V0-V3 (vet Feb 25, architectural_choices Feb 27, session_handoff checked inline for orchestrator tier)

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

**Feb 27-28, 2026: Full LifecycleManager Wiring & Orphan GC**

- `lifecycle_impl.go` completed (662 lines) — all LifecycleManager methods fully implemented:
  - `BeginSpawn`/`ActivateSpawn` (two-phase spawn with rollback)
  - `Complete` (coordinates beads close, session export, workspace archival, event logging)
  - `Abandon` (beads status reset, tmux cleanup, event logging)
  - `ForceComplete`/`ForceAbandon` (GC-initiated transitions for orphaned agents)
  - `DetectOrphans` (scans for in_progress issues with no live execution, 2h threshold)
  - `CurrentState` (derives state from authoritative sources)
- `lifecycle_adapters.go` added — bridges real packages to `pkg/agent` interfaces:
  - `beadsAdapter` wraps `pkg/beads.CLIClient` → `agent.BeadsClient`
  - `openCodeAdapter` wraps `pkg/opencode.Client` → `agent.OpenCodeClient`
  - `tmuxAdapter` wraps `pkg/tmux` → `agent.TmuxClient`
  - `eventLoggerAdapter` wraps `pkg/events` → `agent.EventLogger`
  - `workspaceAdapter` wraps `pkg/spawn` → `agent.WorkspaceManager`
  - Compile-time interface compliance checks via `var _ Interface = (*Impl)(nil)`
- `orch complete` wired to `LifecycleManager.Complete()` (replaces direct beads/session/event calls)
- `orch abandon` wired to `LifecycleManager.Abandon()` (replaces direct beads manipulation)
- `orch clean --orphans` wired to `LifecycleManager.DetectOrphans()` + `ForceAbandon`/`ForceComplete`
- Daemon orphan recovery: periodic scan for stuck in_progress issues + auto-recovery
- Daemon stop/restart subcommands added
- Model impact advisory added to orch complete (cross-references synthesis keywords against .kb/models/)
- `orch orient` command added for session start orientation
- Skip flags require `--skip-reason` (minimum 10 characters); `--force` deprecated
- Cross-project ghost agent fix: auto-resolve abandon for agents from other projects
- Duplicate workspace resolution: prefer SYNTHESIS.md + newest spawn time
- Comprehensive test suite: `lifecycle_manager_test.go` (~1,726 lines)

**Late Feb 2026: Rework Lifecycle, Phase Timeout Detection, Completion Pipeline Extraction**

- `orch rework` command added — reopens closed beads issues with rework context, spawns new agent with prior synthesis and feedback
  - Rework lifecycle bypasses `LifecycleManager` (directly manipulates beads status)
  - Tracks rework iteration count via `rework:<N>` labels and `REWORK #N` comments
  - `events.LogAgentReworked()` event for tracking
- `orch review orphans` command — surfaces closed architect designs with no follow-up implementation issues
- `orch orient` command — session start orientation with decision context per ready issue
- Phase timeout detection:
  - `status_cmd.go` (1,398 lines) extracted with compact mode (age-based filtering), `--project` filtering, `phaseTimeoutThreshold = 30min`
  - `pkg/daemon/phase_timeout.go` — daemon periodic phase timeout detection and escalation
  - Agents without phase updates for 30+ minutes flagged as unresponsive
- Completion pipeline expanded:
  - `complete_architect.go` — auto-create implementation issue on architect completion
  - `complete_cleanup.go` — deferred cleanup step extraction
  - `complete_hotspot.go` — hotspot detection during completion (warns on accretion to large files)
- Daemon periodic tasks expanded (`daemon_periodic.go` extracted from `daemon.go`):
  - Model drift reflection (`pkg/daemon/model_drift_reflection.go`)
  - Knowledge health monitoring (`pkg/daemon/knowledge_health.go`)
  - Orphan lifecycle management (`pkg/daemon/orphan_lifecycle.go`, `pkg/daemon/orphan_detector.go`)
  - Phase timeout detection (`pkg/daemon/phase_timeout.go`)
  - `periodicTasksResult` struct aggregates snapshots (KnowledgeHealth, PhaseTimeout)
- Serve layer expansion:
  - `serve_agents_activity.go` — activity data serving
  - `serve_agents_events.go` — SSE event streaming for dashboard
  - `serve_agents_gap.go` — gap analysis for stale models
  - `serve_agents_types.go` — shared types for serve layer
  - `serve_agents_cache_handler.go` — cache invalidation handler

**2026-03-06: Probe Merge (13 probes)**

- AGENT_MANIFEST.json named explicitly as canonical backend-agnostic workspace state artifact
- Invariant 9 added: phase comments as liveness proxy for claude-backend agents (not tmux windows)
- Invariant 10 added: tmux window mutating operations must use stable @ID
- New Failure Modes 5-8 added: scan ordering pre-emption (historical), QUESTION deadlock, SYNTHESIS compliance gap, duplicate spawn storm
- Pressure Points expanded: plan mode incompatibility, tmux index instability, CLI fallback closed-issue filter bug
- Non-Anthropic model stall rates documented (67-87% vs 44.6% for Opus)
- Stall rate clarification: true stall rate is 4.3%, not 56.6% headline number
- CLI fallback path bug: `bd list -l orch:agent` surfaces closed issues when RPC unavailable

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

**Merged Probes (2026-03-06):**

| Probe | Date | Verdict | Summary |
|-------|------|---------|---------|
| `2026-02-14-backend-agnostic-session-contract.md` | 2026-02-14 | CONFIRMS + EXTENDS | State/infrastructure distinction holds across backends; AGENT_MANIFEST.json is canonical backend-agnostic contract; individual dotfiles are legacy duplication |
| `2026-02-17-dashboard-blind-to-tmux-agents.md` | 2026-02-17 | CONFIRMS + EXTENDS | Scan ordering pre-empted tmux discovery in old architecture (historical); Priority Cascade never ran for tmux agents; adds Failure Mode 5 |
| `2026-02-18-session-status-empty-phantoms.md` | 2026-02-18 | CONTRADICTS assumption | `/session/status` is NOT always empty; phantom accumulation comes from other lifecycle gaps, not empty status map |
| `2026-02-18-cross-project-visibility-cache-context.md` | 2026-02-18 | EXTENDS | Cross-project project discovery has registry fallback (`~/.kb/projects.json`) when `kb` CLI is unavailable |
| `2026-02-18-filter-unspawned-issues.md` | 2026-02-18 | CONFIRMS | Filtering by agent evidence (workspace/session/daemon labels) reduces dashboard noise dramatically (283→25 agents) |
| `2026-02-19-agents-api-closed-issues-filter.md` | 2026-02-19 | EXTENDS | CLI fallback path (`bd list -l orch:agent`) surfaces closed issues without status filtering; `/api/agents` may include completed work when RPC unavailable |
| `2026-02-20-model-drift-major-restructuring.md` | 2026-02-20 | CONTRADICTS stale content | Registry references were stale (registry fully eliminated); confirmed all core principles (4-layer, beads canonical, Priority Cascade) still hold after major restructuring |
| `2026-02-20-tradeoff-visibility-gap-analysis.md` | 2026-02-20 | EXTENDS | Architectural tradeoffs made by workers are invisible to orchestrator until damage occurs; "Pressure Points" section format addresses this |
| `2026-02-24-probe-claude-spawn-dashboard-visibility-gap.md` | 2026-02-24 | EXTENDS | Claude-backend agents have no session_id → query engine marks them dead; tmux liveness (Option A) was the proposed fix and was immediately superseded by probe below |
| `2026-02-24-probe-tmux-liveness-two-lane-violation.md` | 2026-02-24 | CONFIRMS + EXTENDS | Tmux liveness check (orch-go-1182) violated Invariant 6; phase comments are the correct authoritative liveness proxy for claude-backend agents; adds Invariant 9 |
| `2026-02-24-probe-claude-code-plan-mode-feature-impl-alignment.md` | 2026-02-24 | EXTENDS | Claude Code plan mode is incompatible with orchestrated headless agents (dark period, daemon incompatibility, context clearing); feature-impl's Investigation/Design phases are superior |
| `2026-02-24-probe-orch-complete-kills-wrong-tmux-window.md` | 2026-02-24 | EXTENDS | All 4 tmux kill paths use unstable `session:index` instead of stable `@ID`; TOCTOU race can kill wrong window; adds Invariant 10 |
| `2026-02-28-probe-stalled-agent-failure-pattern-audit.md` | 2026-02-28 | EXTENDS | Audit of 1655 archived workspaces reveals 5 new failure modes (QUESTION deadlock, prior art confusion, concurrency ceiling stall, duplicate spawn storm, SYNTHESIS compliance gap); true stall rate is 4.3%, not 56.6%; non-Anthropic models have 67-87% stall rates vs 44.6% for Opus |
| (diagnostic classifier implementation) | 2026-03-12 | EXTENDS | `pkg/daemon/diagnostic.go` implements unified classifier mapping agents to all 12 failure modes with severity levels and recommended actions. Adds Failure Modes 9-12: model protocol incompatibility, silent failure, prior art confusion, concurrency ceiling stall |

**Primary Evidence (Verify These):**

- `pkg/agent/types.go` - Formal lifecycle states (7), transitions (6), `AgentRef`, `TransitionEvent`, `EffectResult`, `TrackedIssue`, `WorkspaceInfo`, orphan detection types
- `pkg/agent/lifecycle.go` - `LifecycleManager` interface (coordinator, not cache), client interfaces (`BeadsClient`, `OpenCodeClient`, `TmuxClient`, `EventLogger`, `WorkspaceManager`)
- `pkg/agent/spawn.go` - `SpawnInput` (lifecycle spawn parameters), `SpawnHandle` (two-phase spawn with rollback)
- `pkg/agent/lifecycle_impl.go` - **Full** `LifecycleManager` implementation (662 lines): BeginSpawn, ActivateSpawn, Complete, Abandon, ForceComplete, ForceAbandon, DetectOrphans, CurrentState
- `pkg/agent/lifecycle_manager_test.go` - Comprehensive test suite (~1,726 lines) covering all transitions, effect tracking, rollback, orphan detection
- `pkg/agent/filters.go` - `IsActiveForConcurrency()`, `IsVisibleByDefault()` with threshold-based filtering
- `cmd/orch/lifecycle_adapters.go` - Bridges real packages (beads, opencode, tmux, events, spawn) to `pkg/agent` interfaces with compile-time compliance checks
- `cmd/orch/lifecycle_adapters_test.go` - Tests for lifecycle adapter interface compliance
- `cmd/orch/serve_agents_status.go` - Priority Cascade implementation (`determineAgentStatus()`), stall tracking, completion backlog detection
- `cmd/orch/query_tracked.go` - Single-pass query engine (`queryTrackedAgents()`), phase-based liveness for claude-backend agents, cross-project beads querying
- `cmd/orch/serve_agents_handlers.go` - Dashboard API handlers
- `cmd/orch/serve_agents_discovery.go` - Workspace and investigation discovery
- `cmd/orch/serve_agents_cache.go` - TTL-based beads cache (prevents excessive bd process spawning from dashboard polls)
- `cmd/orch/serve_agents_activity.go` - Activity data serving for dashboard
- `cmd/orch/serve_agents_events.go` - SSE event streaming for dashboard
- `cmd/orch/serve_agents_gap.go` - Gap analysis for stale models
- `cmd/orch/serve_agents_types.go` - Shared types for serve layer
- `cmd/orch/serve_agents_cache_handler.go` - Cache invalidation handler
- `cmd/orch/status_cmd.go` - CLI status command (1,398 lines): compact mode, `--project` filtering, `phaseTimeoutThreshold = 30min`, `--json` output
- `pkg/verify/check.go` - Completion verification (16 gates across V0-V3, including `GateArchitecturalChoices`)
- `pkg/verify/level.go` - V0-V3 gate level definitions and `ShouldRunGate()` query function
- `cmd/orch/complete_cmd.go` - Completion pipeline orchestrator (wired to LifecycleManager.Complete, knowledge maintenance, model impact advisory, deferred tmux cleanup)
- `cmd/orch/complete_architect.go` - Auto-create implementation issue on architect completion
- `cmd/orch/complete_cleanup.go` - Deferred cleanup step extraction
- `cmd/orch/complete_hotspot.go` - Hotspot detection during completion (warns on accretion to large files)
- `cmd/orch/complete_model_impact.go` - Model impact advisory (cross-references synthesis keywords against .kb/models/)
- `cmd/orch/abandon_cmd.go` - Abandon command (wired to LifecycleManager.Abandon)
- `cmd/orch/clean_cmd.go` - Clean command with `--orphans` flag (wired to LifecycleManager.DetectOrphans + ForceAbandon/ForceComplete)
- `cmd/orch/rework_cmd.go` - Rework command: reopens closed issue, adds rework context, spawns new agent (bypasses LifecycleManager)
- `cmd/orch/orient_cmd.go` - Session start orientation with decision context per ready issue
- `cmd/orch/review_orphans.go` - Surface closed architect designs with no follow-up implementation issues
- `cmd/orch/daemon.go` - Daemon with orphan recovery, stop/restart subcommands, phase timeout detection
- `cmd/orch/daemon_periodic.go` - Extracted periodic task scheduler (reflection, model drift, knowledge health, cleanup, recovery, orphan detection, phase timeout)
- `pkg/daemon/phase_timeout.go` - Phase timeout detection and escalation logic
- `pkg/daemon/orphan_detector.go` - Orphan detection: finds agents with no liveness signals
- `pkg/daemon/orphan_lifecycle.go` - Orphan lifecycle management (auto-recovery decisions)
- `pkg/daemon/model_drift_reflection.go` - Periodic model staleness detection and spawn event emission
- `pkg/daemon/knowledge_health.go` - Knowledge base health monitoring
- `cmd/orch/architecture_lint_test.go` - Structural guardrails preventing registry recreation (forbidden packages/files)
- `cmd/orch/contract_two_lane_test.go` - 12-scenario acceptance matrix enforcing two-lane architecture contract
- `.beads/issues.jsonl` - Canonical completion source
