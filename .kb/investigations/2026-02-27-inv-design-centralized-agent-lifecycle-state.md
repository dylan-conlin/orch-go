## Summary (D.E.K.N.)

**Delta:** Designed a centralized agent lifecycle state machine (`pkg/agent/lifecycle.go`) that makes incomplete transitions structurally impossible by bundling ALL side effects (beads, workspace, tmux, OpenCode, events) into atomic transition functions.

**Evidence:** Mapped 4 lifecycle commands (spawn, complete, abandon, clean) across 6 source files. Found 4 bugs caused by independently-implemented partial cleanup: abandon missing label/assignee removal, clean killing active Claude agents, stale daemon status, orphaned in_progress issues.

**Knowledge:** Each transition touches 4-10 side effects across 5 subsystems. The root cause of all 4 bugs is the same: no single package owns the complete set of side effects for a transition, so each command implements a subset independently and misses some.

**Next:** Create implementation issues for phased migration (types → transitions → command migration → GC rework). Recommend architect review before implementation given hotspot overlap.

**Authority:** architectural - Cross-component redesign affecting spawn, complete, abandon, clean, status, and daemon. Establishes new package boundary.

---

# Investigation: Design Centralized Agent Lifecycle State Machine

**Question:** How should we centralize agent lifecycle state transitions to make incomplete cleanup structurally impossible?

**Defect-Class:** state-corruption

**Started:** 2026-02-27
**Updated:** 2026-02-27
**Owner:** og-arch-design-centralized-agent-27feb-bda3
**Phase:** Complete
**Next Step:** None — design ready for implementation issues
**Status:** Complete

**Patches-Decision:** .kb/decisions/2026-02-18-two-lane-agent-discovery.md (extends, does not contradict)

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/decisions/2026-02-18-two-lane-agent-discovery.md | extends | Yes — atomic spawn pattern confirmed | None |
| .kb/models/agent-lifecycle-state-model/model.md | extends | Yes — four-layer model is the foundation | None |
| .kb/guides/agent-lifecycle.md | extends | Yes — documents current flow | None |
| .kb/decisions/2026-02-13-lifecycle-ownership-boundaries.md | confirms | Yes — Own/Accept/Lobby matches | None |

---

## Findings

### Finding 1: Each transition has 4-10 side effects across 5 subsystems

**Evidence:** Full side-effect mapping of all 4 lifecycle commands:

| Transition | Beads Ops | Workspace Ops | OpenCode Ops | Tmux Ops | Event Ops |
|-----------|-----------|---------------|--------------|----------|-----------|
| **Spawn** | create/tag issue, status→in_progress, set assignee | create dir, write SPAWN_CONTEXT, MANIFEST, dotfiles | create session, send prompt | create session/window, send keys | session.spawned |
| **Complete** | close issue, remove orch:agent, remove triage:ready, signal daemon | archive to archived/, export activity | delete session | close window (deferred) | agent.completed, accretion.delta, verification.bypassed |
| **Abandon** | status→open | write FAILURE_REPORT | delete session, export transcript | kill window | agent.abandoned |
| **Clean** | (reads only) | archive stale workspaces | (reads only) | kill stale windows | agents.cleaned |

**Source:** `cmd/orch/spawn_cmd.go`, `cmd/orch/complete_cmd.go`, `cmd/orch/abandon_cmd.go`, `cmd/orch/clean_cmd.go`, `pkg/spawn/atomic.go`, `pkg/verify/beads_api.go`

**Significance:** The complexity is not in any individual side effect — it's in ensuring ALL of them run for each transition. When each command implements its own subset, gaps emerge.

---

### Finding 2: Four bugs share the same root cause — no lifecycle authority

**Evidence:**

| Bug | Command | Missing Side Effect | Impact |
|-----|---------|-------------------|--------|
| Ghost agents after abandon | `orch abandon` | Does NOT remove `orch:agent` label or clear assignee | Abandoned agents appear active in dashboard/status |
| Clean kills active Claude agents | `orch clean` | Checks OpenCode sessions for liveness but Claude-mode agents have no OpenCode session | Active agents get garbage collected |
| Stale daemon status | `orch status` / `orch serve` | Reads daemon-status.json without PID liveness check | Dashboard shows daemon "running" when it's dead |
| Orphaned in_progress issues | `pkg/daemon` | Sets in_progress on spawn but no recovery when agents die silently | Issues stuck in in_progress forever |

**Source:** `cmd/orch/abandon_cmd.go:283-291` (missing label removal), `cmd/orch/clean_cmd.go:478-480` (OpenCode-only liveness), `cmd/orch/serve_system.go:417` (stale file read), `pkg/daemon/daemon.go` (no orphan recovery)

**Significance:** All four bugs would be prevented if transitions were centralized — the abandon transition would always remove labels, clean would always check phase-based liveness (not just OpenCode), etc.

---

### Finding 3: Atomic spawn pattern proves the concept

**Evidence:** `pkg/spawn/atomic.go` already implements atomic transitions for spawning:
- Phase 1: tag beads + write workspace (with rollback on failure)
- Phase 2: write session_id (best-effort after session creation)

This pattern works. The 238-dead-agents bug (orch-go-1074) was caused by partial state before this pattern existed.

**Source:** `pkg/spawn/atomic.go:24-53` (AtomicSpawnPhase1 with rollback), `pkg/spawn/atomic.go:62-83` (AtomicSpawnPhase2 best-effort)

**Significance:** Extending this pattern to complete, abandon, and GC transitions would close the same class of bugs that atomic spawn already prevents for the spawn path.

---

## Synthesis

**Key Insights:**

1. **The problem is organizational, not technical** — Every individual side effect already has working code (verify.CloseIssue, verify.RemoveOrchAgentLabel, workspace archival, etc.). The bugs exist because no single location enforces that ALL side effects run together.

2. **Transitions, not state storage** — This design adds transition *logic* (a state machine), not a state *store*. It's compatible with the "No Local Agent State" constraint because the lifecycle manager reads from and writes to authoritative sources (beads, OpenCode, tmux, workspace) without caching anything.

3. **Two abstraction layers needed** — (a) A `SideEffectRunner` that encapsulates individual operations with error handling, and (b) a `LifecycleManager` that composes side effects into atomic transitions. Commands become thin callers of the lifecycle manager.

**Answer to Investigation Question:**

Centralize transitions into `pkg/agent/lifecycle.go` with a `LifecycleManager` struct that exposes one method per transition (Spawn, Complete, Abandon, ForceComplete). Each method runs ALL required side effects in order, with explicit error handling for each. The manager does NOT store state — it's a coordinator that reads from and writes to the four authoritative layers (beads, workspace, OpenCode, tmux). Commands (spawn_cmd, complete_cmd, abandon_cmd, clean_cmd) become thin wrappers that call the appropriate lifecycle method.

---

## Design: State Machine

### States

```
                        ┌──────────────────────────────────────────────┐
                        │                                              │
  ┌──────────┐    ┌─────┴────┐    ┌────────────────┐    ┌───────────┐ │
  │ Spawning │───►│  Active   │───►│ Phase Complete │───►│Completing │─┘
  └──────────┘    └──────────┘    └────────────────┘    └───────────┘
       │               │                                      │
       │               │                                      ▼
       │               │                               ┌───────────┐
       │               ├──────────────────────────────►│ Completed │
       │               │                               └───────────┘
       │               │                                      ▲
       │               ▼                                      │
       │         ┌───────────┐                                │
       │         │ Abandoned │     ┌──────────┐               │
       │         └───────────┘     │ Orphaned │───────────────┘
       │                           └──────────┘  (GC path)
       │                                ▲
       │                                │
       └─ (spawn failure = rollback,    │
           no state change)        (detected by
                                   orch clean)
```

**State definitions:**

| State | Beads Status | orch:agent Label | Workspace | OpenCode Session | Description |
|-------|-------------|-----------------|-----------|-----------------|-------------|
| **Spawning** | in_progress | present | being created | being created | Transient — during `orch spawn` execution |
| **Active** | in_progress | present | exists | exists (or Claude CLI running) | Agent is working |
| **PhaseComplete** | in_progress | present | exists + SYNTHESIS.md | may be dead | Agent declared done, awaiting `orch complete` |
| **Completing** | in_progress→closed | being removed | being archived | being deleted | Transient — during `orch complete` execution |
| **Completed** | closed | absent | in archived/ | deleted | Terminal state |
| **Abandoned** | open | absent | exists + FAILURE_REPORT | deleted | Terminal (respawnable) |
| **Orphaned** | in_progress | present | exists | dead + no recent phase | Detected by GC, needs forced completion |

### Transitions with ALL Side Effects

#### T1: Spawn (→ Active)

**Trigger:** `orch spawn` or daemon auto-spawn

**Preconditions:**
- Issue exists and is not closed
- No active agent for this issue (dedup check)
- Concurrency limit not exceeded

**Side Effects (ordered):**

| # | Subsystem | Operation | Rollback on Failure |
|---|-----------|-----------|-------------------|
| 1 | Beads | Add `orch:agent` label | Remove label |
| 2 | Beads | Update status → `in_progress` | (no rollback — status was already being set) |
| 3 | Beads | Set assignee → workspace name | (no rollback) |
| 4 | Workspace | Create directory + SPAWN_CONTEXT.md + AGENT_MANIFEST.json | Remove directory |
| 5 | OpenCode/Claude | Create session or launch CLI | Delete session |
| 6 | Workspace | Write session_id to manifest (Phase 2, best-effort) | — |
| 7 | Tmux | Create window (if tmux/claude backend) | Kill window |
| 8 | Events | Log `session.spawned` | — |

**Current code:** `pkg/spawn/atomic.go` (Phase 1+2), `cmd/orch/spawn_cmd.go` (orchestration), backend-specific files

---

#### T2: Complete (PhaseComplete → Completed)

**Trigger:** `orch complete <id>` (orchestrator-initiated)

**Preconditions:**
- Agent reported Phase: Complete (or `--force`)
- Verification gates pass (or `--skip-*` / `--force`)

**Side Effects (ordered):**

| # | Subsystem | Operation | Critical? |
|---|-----------|-----------|-----------|
| 1 | Verify | Run verification gates (delegated to pkg/verify/) | Yes — blocks if gates fail |
| 2 | Beads | Create follow-up issues for discovered work | No |
| 3 | Beads | Close issue (status → closed) | Yes |
| 4 | Beads | Remove `orch:agent` label | Yes — prevents ghost agents |
| 5 | Beads | Remove `triage:ready` label | No |
| 6 | Daemon | Write verification signal | No |
| 7 | OpenCode | Export session activity to ACTIVITY.json | No |
| 8 | OpenCode | Delete session | Yes — prevents phantom agents |
| 9 | Workspace | Archive to `.orch/workspace/archived/` | No |
| 10 | Tmux | Close window (deferred) | No |
| 11 | Events | Log `agent.completed` | No |
| 12 | Events | Log `accretion.delta` | No |
| 13 | Build | Auto-rebuild if Go files changed | No |
| 14 | Cache | Invalidate serve cache | No |

**Current code:** `cmd/orch/complete_cmd.go` (pipeline orchestrator), `cmd/orch/complete_pipeline.go` (phases)

**Key constraint:** Verification gates (step 1) stay in `pkg/verify/` — the lifecycle manager calls into verify, it doesn't absorb it.

---

#### T3: Abandon (Active → Abandoned)

**Trigger:** `orch abandon <id>` (human-initiated)

**Side Effects (ordered, WITH BUG FIXES):**

| # | Subsystem | Operation | Bug Fix? |
|---|-----------|-----------|----------|
| 1 | Tmux | Kill window (if exists) | — |
| 2 | OpenCode | Export session transcript to SESSION_LOG.md | — |
| 3 | OpenCode | Delete session | — |
| 4 | Workspace | Write FAILURE_REPORT.md | — |
| 5 | Beads | Reset status → open | — |
| 6 | **Beads** | **Remove `orch:agent` label** | **YES — fixes ghost agent bug** |
| 7 | **Beads** | **Clear assignee** | **YES — fixes stale assignment** |
| 8 | Events | Log `agent.abandoned` | — |

**Current code:** `cmd/orch/abandon_cmd.go` — steps 6 and 7 are MISSING

---

#### T4: ForceComplete (Orphaned → Completed)

**Trigger:** `orch clean` GC detection

**Preconditions (orphan detection):**
- Beads status = `in_progress` AND `orch:agent` label present
- No OpenCode session alive (for opencode-mode agents)
- No phase comment in recent window (>2h for all agents)
- No active process in tmux window (for claude-mode agents)

**Side Effects (ordered):**

| # | Subsystem | Operation |
|---|-----------|-----------|
| 1 | Beads | Close issue with reason "orphaned — agent died without completing" |
| 2 | Beads | Remove `orch:agent` label |
| 3 | Beads | Remove `triage:ready` label |
| 4 | Workspace | Archive to `.orch/workspace/archived/` |
| 5 | Tmux | Close window (if exists) |
| 6 | Events | Log `agent.completed` with `orphaned: true` flag |

---

#### T5: ForceAbandon (Orphaned → Abandoned)

**Trigger:** `orch clean` GC detection when issue should be respawned

**Preconditions:** Same as T4, but issue has `triage:ready` label (suggesting retry)

**Side Effects (ordered):**

| # | Subsystem | Operation |
|---|-----------|-----------|
| 1 | Beads | Reset status → open |
| 2 | Beads | Remove `orch:agent` label |
| 3 | Beads | Clear assignee |
| 4 | Tmux | Close window (if exists) |
| 5 | Events | Log `agent.abandoned` with `orphaned: true` flag |

---

## Design: pkg/agent/lifecycle.go Interface

### Key Design Decisions

**Fork 1: Package location**
- **Recommendation:** `pkg/agent/` (new package)
- **SUBSTRATE:** "No Local Agent State" constraint prohibits state *storage*, not transition *logic*. The lifecycle manager is a coordinator, not a cache.
- **Trade-off:** New package adds import but centralizes lifecycle authority

**Fork 2: Interface vs concrete type**
- **Recommendation:** Concrete struct `LifecycleManager` (not interface)
- **SUBSTRATE:** Go convention — define interfaces at the consumer, not the provider. Testing uses dependency injection through the client interfaces.
- **Trade-off:** Less flexible but simpler; interfaces can be extracted later if needed

**Fork 3: How does verification integrate?**
- **Recommendation:** Verification stays in `pkg/verify/`. The Complete transition calls verify as a precondition, then runs cleanup side effects.
- **SUBSTRATE:** Principle: "Evolve by distinction" — verification (should we close?) is distinct from lifecycle (how do we close?). Merging them conflates concerns.
- **Trade-off:** Two packages coordinate, but each has clear responsibility

**Fork 4: How does clean/GC integrate?**
- **Recommendation:** Clean detects orphaned state, then calls `lm.ForceComplete()` or `lm.ForceAbandon()`. Clean becomes pure detection + delegation.
- **SUBSTRATE:** Principle: "Coherence over patches" — clean currently duplicates partial cleanup logic from complete/abandon. Delegating to lifecycle transitions eliminates duplication.
- **Trade-off:** Clean loses independent cleanup logic, but gains completeness guarantees

**Fork 5: How does daemon orphan recovery work?**
- **Recommendation:** Daemon periodically calls `lm.DetectOrphans()` which returns issues that are `in_progress` with no active execution. Daemon then either respawns or calls `lm.ForceAbandon()`.
- **SUBSTRATE:** Constraint: "Daemon sets issues to in_progress when spawning but has no recovery when agents die." This directly fixes the gap.
- **Trade-off:** Adds periodic scan to daemon, but prevents stuck in_progress issues

### Type Definitions

```go
package agent

import (
    "time"
)

// State represents an agent's lifecycle state.
type State string

const (
    StateSpawning      State = "spawning"       // Transient: during orch spawn
    StateActive        State = "active"          // Agent is working
    StatePhaseComplete State = "phase_complete"  // Agent declared done
    StateCompleting    State = "completing"      // Transient: during orch complete
    StateCompleted     State = "completed"       // Terminal: beads closed
    StateAbandoned     State = "abandoned"       // Terminal: beads reset to open
    StateOrphaned      State = "orphaned"        // Detected by GC
)

// Transition represents a state change with its side effects.
type Transition string

const (
    TransitionSpawn         Transition = "spawn"          // → Active
    TransitionComplete      Transition = "complete"       // → Completed
    TransitionAbandon       Transition = "abandon"        // → Abandoned
    TransitionForceComplete Transition = "force_complete"  // Orphaned → Completed
    TransitionForceAbandon  Transition = "force_abandon"   // Orphaned → Abandoned
)

// AgentRef identifies an agent for lifecycle operations.
// This is NOT stored state — it's a query handle.
type AgentRef struct {
    BeadsID       string
    WorkspaceName string
    WorkspacePath string
    SessionID     string // May be empty for Claude-mode agents
    ProjectDir    string // For cross-project agents
    SpawnMode     string // "opencode", "claude", "tmux"
}

// TransitionResult captures what happened during a transition.
type TransitionResult struct {
    Transition  Transition
    Agent       AgentRef
    Effects     []EffectResult
    Warnings    []string
    Success     bool
}

// EffectResult tracks individual side effect execution.
type EffectResult struct {
    Subsystem string // "beads", "opencode", "tmux", "workspace", "events"
    Operation string // "close_issue", "remove_label", "archive_workspace"
    Success   bool
    Error     error
    Critical  bool   // If true and failed, transition fails
    Duration  time.Duration
}

// OrphanDetectionResult from periodic GC scan.
type OrphanDetectionResult struct {
    Orphans []OrphanedAgent
    Scanned int
    Elapsed time.Duration
}

// OrphanedAgent represents an agent detected as orphaned.
type OrphanedAgent struct {
    Agent       AgentRef
    Reason      string    // "no_session_no_phase", "session_dead_no_phase", etc.
    LastPhase   string    // Last known phase (may be empty)
    StaleFor    time.Duration // How long since last activity
    ShouldRetry bool      // Based on triage:ready label
}
```

### LifecycleManager

```go
// LifecycleManager coordinates state transitions across all four layers.
// It does NOT store agent state — it reads from and writes to authoritative sources.
//
// Architectural constraint: This is a coordinator, not a cache.
// After any method returns, the manager holds no agent state.
// All state lives in beads, workspace files, OpenCode, and tmux.
type LifecycleManager struct {
    beads    BeadsClient
    opencode OpenCodeClient
    tmux     TmuxClient
    events   EventLogger
}

// Client interfaces (for dependency injection / testing)

type BeadsClient interface {
    AddLabel(beadsID, label string) error
    RemoveLabel(beadsID, label string) error
    UpdateStatus(beadsID, status string) error
    UpdateAssignee(beadsID, assignee string) error
    CloseIssue(beadsID, reason string) error
    GetIssue(beadsID string) (*Issue, error)
    GetComments(beadsID string) ([]Comment, error)
}

type OpenCodeClient interface {
    SessionExists(sessionID string) (bool, error)
    DeleteSession(sessionID string) error
    ExportActivity(sessionID, outputPath string) error
}

type TmuxClient interface {
    WindowExists(name string) (bool, error)
    KillWindow(name string) error
    GetPaneCommand(name string) (string, error)
}

type EventLogger interface {
    Log(eventType string, data map[string]interface{}) error
}
```

### Transition Methods

```go
// Complete performs all side effects for the Complete transition.
// Precondition: verification gates have already passed (caller's responsibility).
// The lifecycle manager owns cleanup, not verification.
func (lm *LifecycleManager) Complete(agent AgentRef, reason string) (*TransitionResult, error)

// Abandon performs all side effects for the Abandon transition.
// Fixes known bugs: removes orch:agent label, clears assignee.
func (lm *LifecycleManager) Abandon(agent AgentRef, reason string) (*TransitionResult, error)

// ForceComplete performs GC-initiated completion for orphaned agents.
func (lm *LifecycleManager) ForceComplete(agent AgentRef, reason string) (*TransitionResult, error)

// ForceAbandon performs GC-initiated abandonment for orphaned agents that should retry.
func (lm *LifecycleManager) ForceAbandon(agent AgentRef) (*TransitionResult, error)

// DetectOrphans scans for agents in Active state with no live execution.
func (lm *LifecycleManager) DetectOrphans(projectDirs []string, threshold time.Duration) (*OrphanDetectionResult, error)

// ResolveAgent builds an AgentRef from a beads ID by querying authoritative sources.
func (lm *LifecycleManager) ResolveAgent(beadsID string) (*AgentRef, error)
```

### Transition Implementation Pattern

Each transition method follows this pattern:

```go
func (lm *LifecycleManager) Complete(agent AgentRef, reason string) (*TransitionResult, error) {
    result := &TransitionResult{
        Transition: TransitionComplete,
        Agent:      agent,
    }

    // Run effects in order. Critical effects fail the transition.
    // Non-critical effects log warnings but continue.

    // Step 1: Close beads issue (CRITICAL)
    result.addEffect(lm.closeBeadsIssue(agent, reason))
    if result.hasCriticalFailure() {
        return result, fmt.Errorf("critical: beads close failed")
    }

    // Step 2: Remove orch:agent label (CRITICAL — prevents ghost agents)
    result.addEffect(lm.removeOrchAgentLabel(agent))

    // Step 3: Remove triage:ready label (non-critical)
    result.addEffect(lm.removeTriageReadyLabel(agent))

    // Step 4: Signal daemon (non-critical)
    result.addEffect(lm.signalDaemon())

    // Step 5: Export session activity (non-critical)
    if agent.SessionID != "" {
        result.addEffect(lm.exportSessionActivity(agent))
    }

    // Step 6: Delete OpenCode session (non-critical)
    if agent.SessionID != "" {
        result.addEffect(lm.deleteSession(agent))
    }

    // Step 7: Archive workspace (non-critical)
    result.addEffect(lm.archiveWorkspace(agent))

    // Step 8: Close tmux window (non-critical, deferred)
    result.addEffect(lm.closeTmuxWindow(agent))

    // Step 9: Log event (non-critical)
    result.addEffect(lm.logEvent(EventTypeAgentCompleted, agent, reason))

    result.Success = !result.hasCriticalFailure()
    return result, nil
}
```

### Daemon Status Liveness Fix

The daemon status bug (reading stale file without PID check) is separate from the lifecycle state machine but should be fixed alongside it:

```go
// ReadStatusFileWithLiveness reads daemon status and validates PID is alive.
// Returns status with IsStale=true if PID is dead.
func ReadStatusFileWithLiveness(path string) (*DaemonStatus, error) {
    status, err := ReadStatusFile(path)
    if err != nil {
        return nil, err
    }
    if status.PID > 0 {
        // Check if PID is actually running
        process, err := os.FindProcess(status.PID)
        if err != nil || !isProcessAlive(process) {
            status.Status = "stale"
            status.IsStale = true
        }
    }
    return status, nil
}
```

---

## Migration Plan

### Phase 1: Define Types (1 issue, ~1h)

Create `pkg/agent/` package with:
- `types.go` — State, Transition, AgentRef, TransitionResult, EffectResult, OrphanedAgent
- `interfaces.go` — BeadsClient, OpenCodeClient, TmuxClient, EventLogger

**No behavior changes.** Just type definitions.

### Phase 2: Implement Abandon Transition (1 issue, ~2h)

Start with abandon because it's the simplest transition and has the most obvious bug fix.

1. Implement `LifecycleManager.Abandon()` in `pkg/agent/lifecycle.go`
2. Wire it into `cmd/orch/abandon_cmd.go`
3. Fixes bugs: adds orch:agent label removal + assignee clearing

**Why abandon first:** Smallest transition (8 side effects), clearest bug to fix, lowest risk.

### Phase 3: Implement Complete Transition (1 issue, ~3h)

1. Implement `LifecycleManager.Complete()` in `pkg/agent/lifecycle.go`
2. Wire it into `cmd/orch/complete_cmd.go` (replace cleanup phases)
3. Verification gates remain in `pkg/verify/` — lifecycle manager called AFTER verification passes

**Key constraint:** Complete pipeline has 7 phases. Only the cleanup phases (4-7) move to the lifecycle manager. Phases 1-3 (identification, discovery, knowledge) remain in the pipeline.

### Phase 4: Implement Orphan Detection + GC Transitions (1 issue, ~3h)

1. Implement `DetectOrphans()` in `pkg/agent/lifecycle.go`
2. Implement `ForceComplete()` and `ForceAbandon()`
3. Refactor `cmd/orch/clean_cmd.go` to use orphan detection + lifecycle transitions
4. Clean becomes: detect orphans → prompt user → call ForceComplete/ForceAbandon

**Fixes:** Clean no longer kills active Claude agents (orphan detection checks phase-based liveness for all backends).

### Phase 5: Daemon Orphan Recovery + Status Liveness (1 issue, ~2h)

1. Add periodic orphan scan to daemon loop
2. Daemon calls `ForceAbandon()` for orphaned issues (allows respawn)
3. Fix daemon status reading with PID liveness check

**Fixes:** Orphaned in_progress issues get recovered. Stale daemon status detected.

### Phase 6: Spawn Integration (1 issue, ~2h)

Move spawn side effects from `pkg/spawn/atomic.go` into lifecycle manager. Spawn is last because it already works well with atomic.go — the pattern just gets formalized.

### Total: 6 issues, ~13h estimated

---

## Structured Uncertainty

**What's tested:**

- ✅ Side effect inventory for all 4 transitions (verified by reading all source files)
- ✅ Bug root causes confirmed (abandon missing label removal at line 283-291, clean using OpenCode-only liveness at line 478)
- ✅ Atomic spawn pattern works (in production since Feb 2026)
- ✅ Design compatible with "No Local Agent State" constraint (lifecycle manager is coordinator, not cache)

**What's untested:**

- ⚠️ Performance impact of additional beads operations in abandon (2 extra RPC calls per abandon)
- ⚠️ Orphan detection threshold (2h proposed, but optimal value unknown)
- ⚠️ Whether daemon periodic orphan scan creates contention with daemon polling loop
- ⚠️ Impact on complete pipeline when cleanup phases are extracted

**What would change this:**

- If beads RPC becomes unreliable, the "all side effects must succeed" model needs partial-success handling (currently non-critical effects tolerate failure)
- If a new spawn backend is added (e.g., native Claude SDK), new side effects would need to be added to the transition definitions
- If the "No Local Agent State" constraint is relaxed, the lifecycle manager could cache agent refs for faster multi-operation transitions

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Create pkg/agent/ package with lifecycle transitions | architectural | New package boundary affecting 6+ existing files across cmd/ and pkg/ |
| Fix abandon missing label/assignee removal | implementation | Bug fix within existing patterns |
| Refactor clean to use lifecycle transitions | architectural | Changes GC behavior, affects how orphans are detected/handled |
| Add daemon orphan recovery | architectural | New daemon behavior, affects auto-spawn lifecycle |

### Recommended Approach: Phased Migration with Abandon-First

**Why this approach:**
- Abandon-first validates the pattern with the simplest, most bug-ridden transition
- Each phase delivers standalone value (bug fix, cleaner code, new capability)
- No big-bang rewrite — commands migrate incrementally
- Verification gates stay in pkg/verify/ (separation of concerns preserved)

**Trade-offs accepted:**
- Temporary code duplication during migration (two paths for same transition)
- pkg/agent/ adds one more package to maintain
- 6 issues create coordination overhead

**Implementation sequence:**
1. Types first (foundation, no behavior change)
2. Abandon (simplest, most obvious bug fix)
3. Complete (most complex, biggest impact)
4. GC/Orphan detection (fixes clean bugs)
5. Daemon recovery (fixes orphan bug)
6. Spawn (formalize existing pattern)

### Things to watch out for:

- ⚠️ Complete pipeline has interactive prompts (explain-back gate, discovered work). These stay in the pipeline, not the lifecycle manager.
- ⚠️ Auto-rebuild and cache invalidation are complete-specific post-effects. They might stay in complete_cmd.go rather than moving to the lifecycle manager.
- ⚠️ Cross-project agents need correct beads directory resolution — the lifecycle manager must accept a projectDir parameter for beads operations.
- ⚠️ The "No Local Agent State" lint tests in `architecture_lint_test.go` must not flag `pkg/agent/` — it's transition logic, not state storage. May need lint rule update.

### Success criteria:

- ✅ `orch abandon` removes orch:agent label and clears assignee (fix ghost agent bug)
- ✅ `orch clean` does not kill active Claude-mode agents
- ✅ Daemon status shows "stale" when PID is dead
- ✅ Orphaned in_progress issues get recovered within 1 daemon cycle
- ✅ All existing tests continue to pass after each migration phase
- ✅ TransitionResult captures all side effects with success/failure status

---

## References

**Files Examined:**
- `cmd/orch/spawn_cmd.go` — Spawn command orchestration
- `cmd/orch/complete_cmd.go` — Complete pipeline orchestrator
- `cmd/orch/complete_pipeline.go` — Complete pipeline phases
- `cmd/orch/abandon_cmd.go` — Abandon command (bug: missing label/assignee)
- `cmd/orch/clean_cmd.go` — Clean/GC command (bug: kills Claude agents)
- `cmd/orch/status_cmd.go` — Status display
- `cmd/orch/serve_system.go` — Daemon status API (bug: stale file)
- `pkg/spawn/atomic.go` — Atomic spawn pattern
- `pkg/verify/beads_api.go` — Beads RPC/CLI operations
- `pkg/events/logger.go` — Event logging
- `pkg/daemon/daemon.go` — Daemon auto-spawn (bug: no orphan recovery)
- `.kb/decisions/2026-02-18-two-lane-agent-discovery.md` — Two-lane architecture
- `.kb/models/agent-lifecycle-state-model/model.md` — Four-layer state model
- `.kb/guides/agent-lifecycle.md` — Agent lifecycle guide

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-02-18-two-lane-agent-discovery.md` — Extends this with transition authority
- **Model:** `.kb/models/agent-lifecycle-state-model/model.md` — Four-layer foundation
- **Guide:** `.kb/guides/agent-lifecycle.md` — Will need updating post-implementation

---

## Investigation History

**2026-02-27 15:00:** Investigation started
- Initial question: How to centralize agent lifecycle to prevent incomplete transitions?
- Context: 4 bugs discovered in one session, all caused by independent partial cleanup

**2026-02-27 15:30:** Side effect mapping complete
- Mapped all transitions across spawn, complete, abandon, clean
- Confirmed 4-10 side effects per transition across 5 subsystems

**2026-02-27 16:00:** Substrate consultation complete
- Two-lane ADR, lifecycle model, principles all consulted
- No conflicts found — design extends existing architecture

**2026-02-27 16:30:** Design complete
- State machine with 7 states, 5 transitions
- pkg/agent/lifecycle.go interface designed
- 6-phase migration plan with abandon-first approach
- All forks navigated with substrate traces

**2026-02-27 17:00:** Investigation completed
- Status: Complete
- Key outcome: Centralized lifecycle state machine design ready for implementation
