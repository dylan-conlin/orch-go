# Model: Daemon Autonomous Operation

**Domain:** Daemon / Autonomous Spawning / Batch Processing
**Last Updated:** 2026-02-09
**Synthesized From:** 39 investigations + daemon.md guide (synthesized from 33 investigations, Dec 2025 - Jan 2026) + 2 probes (Feb 8-9, 2026) on poll loops, skill inference, capacity management, completion tracking, cross-project operation

**Recent Probes:**

- `probes/2026-02-08-processed-cache-mark-after-success.md` — **Extends** dedup invariants. `ProcessedIssueCache` must be written only after confirmed spawn success, not during evaluation. Rejected/failed spawns stay retryable. Transient race protection uses `SpawnedIssues` (in-memory). Confidence: High.
- `probes/2026-02-09-skill-inference-mapping-verification.md` — **Contradicts (then updates)** stale skill inference claims. Confirms epics are non-spawnable errors, unknown types error (no daemon fallback), `investigation`/`question` mappings are valid, and `skill:*` labels are highest-priority override. Confidence: High.

**Significant Changes (Feb 7-8, 2026):**

- **Polish Mode:** Daemon self-improves when queue empty (audits, consolidates investigations, cleans stale decisions)
- **Auto-detect completion:** Agents with commits + idle sessions auto-detected as complete (see agent-lifecycle probe)
- **Phase-aware idle nudge:** Recovery detects idle agents at late phases and nudges them
- **Model behavior profiles:** Per-model profiles for strict-complete vs needs-nudge agents

---

## Summary (30 seconds)

The daemon is an **autonomous agent spawner** that operates in a **poll-spawn-complete cycle**: polls beads for `triage:ready` issues, infers skill, spawns within capacity limits, monitors for `Phase: Complete`, verifies and closes. The daemon operates **independently of orchestrators** - orchestrators triage (label issues ready), daemon spawns (batch processing), orchestrators synthesize (review completed work). Skill inference uses a **priority chain**: `skill:*` label → title pattern → issue type (`bug`→`systematic-debugging`, `feature`/`task`→`feature-impl`, `investigation`/`question`→`investigation`, `epic`/unknown→error skipped). Capacity management uses **WorkerPool** with reconciliation against OpenCode to free stale slots.

---

## Core Mechanism

### Poll-Spawn-Complete Cycle

The daemon runs continuously via launchd, executing this cycle:

```
┌──────────────────────────────────────────────────────┐
│  Daemon Poll Loop (every 60s)                        │
│                                                      │
│  1. Reconcile with OpenCode                          │
│     - Query active sessions via API                  │
│     - Free pool slots for non-existent sessions      │
│                                                      │
│  2. Check periodic kb reflect (if due)               │
│     - Surface synthesis opportunities               │
│                                                      │
│  3. Poll beads: bd ready --limit 0                   │
│     - Get all ready issues (no blockers)            │
│                                                      │
│  4. Filter for triage:ready label                    │
│     - Skip if wrong type, status, or has deps       │
│                                                      │
│  5. For each ready issue (within capacity):          │
│     - Infer skill from issue type                    │
│     - Acquire slot from WorkerPool                   │
│     - Spawn: orch work <beads-id>                    │
│                                                      │
│  6. Sleep 60s, repeat                                │
└──────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────┐
│  Completion Loop (every 60s, parallel)               │
│                                                      │
│  1. Poll for Phase: Complete comments                │
│     - Query beads for recent phase transitions       │
│                                                      │
│  2. For each completed agent:                        │
│     - Verify completion (check artifacts)            │
│     - Close beads issue with reason                  │
│     - Release pool slot                              │
│                                                      │
│  3. Sleep 60s, repeat                                │
└──────────────────────────────────────────────────────┘
```

**Key insight:** Two independent loops. Poll loop manages spawning (forward flow), completion loop manages closing (backward flow). Decoupled to prevent completion delays from blocking spawns.

### Skill Inference

Daemon infers skill using a **priority chain**: `skill:*` label → title pattern → issue type.

| Issue Type      | Inferred Skill        | Notes                                     |
| --------------- | --------------------- | ----------------------------------------- |
| `bug`           | systematic-debugging  | Bugs need root cause analysis             |
| `feature`       | feature-impl          | Features need implementation              |
| `task`          | feature-impl          | Tasks are implementation work             |
| `investigation` | investigation         | Explicit investigation requests           |
| `question`      | investigation         | Questions need understanding              |
| `epic`          | error (non-spawnable) | Epics are not directly spawnable          |
| (unknown)       | error (skipped)       | Daemon skips issues with inference errors |

**Override priority:**

1. `skill:*` label (highest) — e.g., `skill:architect` forces architect skill
2. Title pattern — e.g., "Synthesize ... investigations" maps to `kb-reflect`
3. Issue type (lowest) — table above

**No silent fallback:** Unknown types produce errors and are skipped, not defaulted.

**Why type as base, with label override:**

- Type is required field on creation (always available)
- Labels enable orchestrator to override inference when needed
- Title patterns catch automated issue types (synthesis, etc.)

**Source:** `pkg/daemon/skill_inference.go`, `pkg/daemon/daemon_spawn.go:240`
**Verified by:** `probes/2026-02-09-skill-inference-mapping-verification.md`

### Capacity Management

**WorkerPool** manages concurrent agent limits:

| Component    | Purpose                                            |
| ------------ | -------------------------------------------------- |
| `MaxAgents`  | Hard limit (default: 5) from `~/.orch/config.yaml` |
| `active map` | Tracks spawned agents by beads ID                  |
| `Acquire()`  | Blocks if at capacity, returns slot when available |
| `Release()`  | Frees slot when agent completes                    |

**Reconciliation (every poll):**

```go
// Query OpenCode for active sessions
sessions := opencode.ListSessions()

// Free stale slots (spawned but session gone)
for beadsID := range pool.active {
    if !sessions.Contains(beadsID) {
        pool.Release(beadsID)
    }
}
```

**Why reconciliation:** Prevents drift. Agents can fail to start, OpenCode server can restart, sessions can crash. Reconciliation ensures pool matches reality.

**Source:** `pkg/daemon/pool.go`

### Cross-Project Operation

Daemon operates across **multiple project directories**:

```yaml
# ~/.orch/config.yaml
projects:
  - path: ~/Documents/personal/orch-go
    name: orch-go
  - path: ~/orch-knowledge
    name: orch-knowledge
```

**Poll behavior:**

1. For each project directory:
   - `cd` into project
   - Run `bd ready` (reads that project's `.beads/`)
   - Filter for `triage:ready` issues
   - Spawn with `--workdir` flag pointing to project

**Key insight:** Daemon is project-agnostic. It doesn't "live" in one repo. It polls all configured projects and spawns agents in the appropriate directory.

**Source:** `pkg/daemon/daemon.go:processProjects()`

### Triage Workflow

**Labels control spawn readiness:**

| Label           | Meaning         | Who Sets                             | Daemon Action                        |
| --------------- | --------------- | ------------------------------------ | ------------------------------------ |
| `triage:ready`  | Confident spawn | Orchestrator or issue-creation agent | Spawns immediately                   |
| `triage:review` | Needs review    | issue-creation agent                 | Skips, waits for orchestrator review |
| (no label)      | Default triage  | N/A                                  | Skips                                |

**The flow:**

```
User reports symptom
    ↓
orch spawn issue-creation "symptom"  (or orchestrator creates issue directly)
    ↓
Issue created with triage:review (default for uncertainty)
    ↓
Orchestrator reviews, relabels triage:ready
    ↓
Daemon auto-spawns on next poll
```

**Why this pattern:** Separates judgment (orchestrator) from execution (daemon). Daemon handles batch/overnight work, orchestrator stays available for triage and synthesis.

---

## Why This Fails

### 1. Capacity Starvation

**What happens:** Pool shows MaxAgents active, but `orch status` shows fewer actual agents running.

**Root cause:** Spawn failures don't release slots. Agent spawned, counted against pool, but spawn fails (bad skill name, missing context, etc.) - slot never released.

**Why detection is hard:** Pool only knows about attempts, not outcomes. No feedback loop from spawn failure to pool.

**Fix:** Reconciliation with OpenCode. Query actual sessions, release slots for non-existent agents.

**Prevention:** Spawn tracking with retry limits (`pkg/daemon/spawn_tracker.go`).

### 2. Duplicate Spawns

**What happens:** Same issue spawned multiple times by daemon on consecutive polls.

**Root cause:** Spawn latency. Issue labeled `triage:ready` at poll N, daemon spawns, but spawn hasn't transitioned issue to `in_progress` by poll N+1. Daemon sees same issue still ready, spawns again.

**Why detection is hard:** Race condition between poll interval (60s) and spawn transition time (variable).

**Fix:** Spawn deduplication via tracking. Track spawned beads IDs in memory, skip on subsequent polls until status confirms transition.

**Source:** `pkg/daemon/spawn_tracker.go`

### 3. Skill Inference Mismatch

**What happens:** Daemon spawns wrong skill for issue type.

**Root cause:** Issue metadata doesn't reflect actual work, or explicit override is missing. Example: a complex recurring bug needs architecture exploration, but default `bug` inference spawns `systematic-debugging` unless `skill:architect` is set.

**Why detection is hard:** Inference is deterministic and appears valid from metadata alone. The mismatch is semantic (intent vs metadata), not a runtime error.

**Fix:** Apply explicit override before marking `triage:ready` (for example `skill:architect`), or adjust issue type/title to match intended workflow.

**Prevention:** Better issue creation prompts, triage checklist for override labels, and validation that issue metadata reflects intended execution path.

---

## Constraints

### Why Poll Instead of Event-Driven?

**Constraint:** Daemon polls beads every 60s instead of subscribing to beads events.

**Implication:** Up to 60s latency between "issue ready" and "daemon spawns".

**Workaround:** Manual spawn for urgent work: `orch spawn SKILL "task" --issue <id>`

**This enables:** Simple, reliable batch processing without beads architecture changes
**This constrains:** Cannot react to issues in real-time (up to 60s latency)

---

### Why Type-Based Inference with Label/Title Overrides?

**Constraint:** Daemon uses deterministic priority: `skill:*` label → title pattern → issue type.

**Implication:** Type remains the baseline for most issues, but explicit labels and known title patterns can override default behavior.

**Workaround:** For edge cases, set explicit `skill:*` labels (for example `skill:architect`) before `triage:ready`.

**This enables:** Predictable default routing with low-friction orchestrator overrides
**This constrains:** Unknown and non-spawnable types still error and are skipped (no daemon fallback)

**Source:** `pkg/daemon/skill_inference.go`, `pkg/daemon/daemon_spawn.go`

---

### Why MaxAgents Hard Limit?

**Constraint:** WorkerPool enforces hard limit on concurrent agents (default: 5).

**Implication:** Daemon blocks spawning even if more ready issues exist and system has capacity.

**Workaround:** Increase MaxAgents in config, or spawn manually outside daemon.

**This enables:** Controlled resource usage, forces prioritization
**This constrains:** Cannot spawn more than limit even when system has capacity

---

### Why 60s Poll Interval?

**Constraint:** Both poll loop and completion loop run every 60s.

**Implication:** Up to 60s between state changes (issue ready → spawned, agent complete → closed).

**Workaround:** Manual intervention for time-sensitive work.

**This enables:** Balanced responsiveness vs API load for batch work
**This constrains:** Cannot achieve sub-minute response times for state changes

---

## Evolution

### Phase 1: Poll and Spawn (Dec 2025)

**What existed:** Basic poll loop, skill inference from type, single-project operation.

**Gap:** No completion tracking, no capacity limits, no cross-project support.

**Trigger:** Wanted overnight batch processing, but daemon didn't know when agents finished.

### Phase 2: Completion Loop (Late Dec 2025)

**What changed:** Separate completion polling loop, monitors for `Phase: Complete` comments, verifies and closes issues.

**Investigations:** 8 investigations on completion detection, verification integration, pool slot release.

**Key insight:** Completion must be parallel to spawning. Blocking spawn loop on completion creates dependency chains (spawn → wait → complete → spawn).

### Phase 3: Capacity Management (Dec 28-30, 2025)

**What changed:** WorkerPool with MaxAgents limit, reconciliation with OpenCode, spawn tracking for dedup.

**Investigations:** 12 investigations on capacity starvation, duplicate spawns, pool drift.

**Key insight:** Pool needs ground truth. In-memory tracking drifts from reality (spawn failures, crashes, restarts). Reconciliation with OpenCode session list keeps pool accurate.

### Phase 4: Cross-Project Operation (Jan 2-6, 2026)

**What changed:** Multi-project polling, `--workdir` flag propagation, per-project spawn tracking.

**Investigations:** 7 investigations on cross-project issues, beads scope, workspace creation.

**Key insight:** Daemon is the orchestration home (lives in orch-go), but spawns across projects. Issues created in current project, agents execute in target project.

### Phase 5: kb reflect Integration (Jan 2026)

**What changed:** Periodic `kb reflect` execution to surface synthesis opportunities.

**Investigations:** 4 investigations on synthesis timing, reflection frequency, integration points.

**Key insight:** Daemon has the complete view (all projects, all ready work). Ideal position to identify when synthesis is due.

---

## References

**Guide:**

- `.kb/guides/daemon.md` - Procedural guide (commands, configuration, troubleshooting)

**Investigations:**

- Daemon.md references 33 investigations from Dec 2025 - Jan 2026
- Additional 6+ investigations on cross-project operation, kb reflect integration

**Decisions:**

- `.kb/decisions/2026-01-21-cross-project-daemon-architecture.md` - Single daemon polls all kb-registered projects.
- `.kb/decisions/2026-01-17-five-tier-completion-escalation-model.md` - Completion loop escalation semantics.
- `.kb/decisions/2026-01-14-two-tier-cleanup-pattern.md` - Event cleanup + periodic cleanup for daemon-managed resources.
- `.kb/decisions/2026-01-21-strategic-first-gate-advisory-only.md` - Spawn hotspot signal is advisory, not blocking.

**Models:**

- `.kb/models/spawn-architecture.md` - How `orch work` spawns agents
- `.kb/models/beads-integration-architecture.md` - How daemon polls beads
- `.kb/models/completion-verification.md` - How completion loop verifies agents

**Source code:**

- `pkg/daemon/daemon.go` - Main poll loop, Next() and Once() methods
- `pkg/daemon/pool.go` - WorkerPool capacity management
- `pkg/daemon/skill_inference.go` - Issue type → skill mapping
- `pkg/daemon/completion.go` - SSE-based completion tracking (legacy)
- `pkg/daemon/completion_processing.go` - Beads-polling completion detection
- `pkg/daemon/spawn_tracker.go` - Deduplication tracking
- `cmd/orch/daemon.go` - CLI commands (run, preview)
