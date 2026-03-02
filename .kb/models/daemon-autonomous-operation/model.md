# Model: Daemon Autonomous Operation

**Domain:** Daemon / Autonomous Spawning / Batch Processing
**Last Updated:** 2026-03-02
**Synthesized From:** 39+ investigations + daemon.md guide (verified Mar 1, 2026) on poll loops, skill inference, capacity management, completion tracking, cross-project operation, dedup pipeline, verification threshold, orphan detection

---

## Summary (30 seconds)

The daemon is an **autonomous agent spawner** that operates in a **poll-spawn-complete cycle**: polls beads for `triage:ready` issues across all kb-registered projects, infers skill via a **4-level priority chain** (label → title → description → type fallback), spawns within capacity limits, monitors for `Phase: Complete`, and auto-completes routine work. The daemon **runs from orch-go** (its orchestration home) but **polls cross-project** via ProjectRegistry. Three safety mechanisms prevent runaway operation: a **6-layer spawn dedup pipeline** (L1-L6) prevents duplicate spawns, a **VerificationTracker** pauses spawning after N unverified completions, and an **orphan detector** resets dead agents while preserving spawn cache cooldown.

---

## Core Mechanism

### Poll-Spawn-Complete Cycle

The daemon runs continuously via launchd, executing this cycle:

```
┌──────────────────────────────────────────────────────────────┐
│  Daemon Poll Loop (every 60s)                                │
│                                                              │
│  1. Reconcile with OpenCode (free stale slots)               │
│  2. CompletionOnce (auto-complete finished agents)           │
│  3. Check periodic kb reflect (if due)                       │
│  4. Check VerificationTracker (pause if threshold reached)   │
│  5. Poll beads: bd ready --limit 0 (all registered projects) │
│  6. Filter for triage:ready label                            │
│  7. For each ready issue (within capacity):                  │
│     - Run 6-layer dedup pipeline                             │
│     - Infer skill (label → title → description → type)       │
│     - Spawn: orch work <beads-id>                            │
│  8. Sleep 60s, repeat                                        │
│                                                              │
│  Completion Loop (integrated into poll cycle):               │
│  - Poll for Phase: Complete comments                         │
│  - Verify completion (check artifacts)                       │
│  - Auto-close routine work (escalate Block/Failed)           │
│  - Release pool slots                                        │
│  - Record in VerificationTracker                             │
└──────────────────────────────────────────────────────────────┘
```

**Key insight:** Completion is integrated into the poll cycle via `CompletionOnce()`, not a separate loop. Auto-completion respects a 5-tier escalation model: None/Info/Review auto-complete, Block/Failed require human review.

### Skill Inference (4-Level Priority Chain)

Daemon infers skill via `InferSkillFromIssue()` with 4 priority levels:

| Priority | Source | Example | Override? |
|----------|--------|---------|-----------|
| 1 (highest) | `skill:*` label | `skill:architect` → `architect` | Explicit override |
| 2 | Title pattern | `Investigate: X` → `investigation` | Naming convention |
| 3 | Description heuristic | Keywords like "root cause" → `systematic-debugging` | Content analysis |
| 4 (fallback) | Issue type | `bug` → `systematic-debugging` | Default mapping |

**Type-based fallback table (Priority 4):**

| Issue Type | Inferred Skill | Rationale |
|------------|----------------|-----------|
| `task` | `feature-impl` | Generic implementation work |
| `bug` | `systematic-debugging` | Bugs need root cause analysis |
| `feature` | `feature-impl` | Features need implementation |
| `investigation` | `investigation` | Explicit investigation work |

**Non-spawnable types:** `epic` and `chore` are not spawnable. Epics are containers for child issues; chores are non-agent maintenance. The daemon skips these with a rejection reason.

**Model inference by skill:** Deep reasoning skills (investigation, architect, systematic-debugging, codebase-audit, research) → Opus. Implementation skills (feature-impl, issue-creation) → Sonnet.

**Source:** `pkg/daemon/skill_inference.go`

### Capacity Management

**WorkerPool** manages concurrent agent limits:

| Component | Purpose |
|-----------|---------|
| `MaxAgents` | Hard limit (default: 5) from CLI flag or config |
| `active map` | Tracks spawned agents by beads ID |
| `Acquire()` | Blocks if at capacity, returns slot when available |
| `Release()` | Frees slot when agent completes |

**Reconciliation (every poll):** Queries OpenCode for active sessions, frees pool slots for non-existent agents. Prevents drift from spawn failures, crashes, restarts.

**Source:** `pkg/daemon/pool.go`

### Spawn Dedup Pipeline (6-Layer)

The daemon prevents duplicate spawns via 6 sequential layers in `spawnIssue()`. These accumulated over 9 tactical fixes (Jan-Mar 2026), each patching a gap in the previous layer.

| Layer | Check | Source | Fail Mode | Nature |
|-------|-------|--------|-----------|--------|
| L1 | SpawnedIssueTracker (ID-based, 6h TTL) | `spawn_tracker.go` | Blocks spawn | Heuristic |
| L2 | Session/Tmux existence check | `session_dedup.go` | Blocks (fail-open if API down) | Heuristic |
| L3 | Title dedup (in-memory, TTL-coupled) | `spawn_tracker.go` | Blocks spawn | Heuristic |
| L4 | Title dedup (beads DB query) | `spawn_tracker.go` | Blocks (fail-open) | Structural-ish |
| L5 | Fresh beads status re-check | `daemon.go` | Blocks (fail-open) | Structural |
| L6 | UpdateStatus("in_progress") | `daemon.go` | **Fail-fast** | Structural (PRIMARY) |

**Key properties:**
- L6 is the only fail-fast layer — if it fails, spawn is aborted
- L2, L4, L5 are fail-open — they allow spawn if their backing service is unavailable
- L1-L3 survive daemon restarts via disk persistence (`~/.orch/spawn_cache.json`)
- L1 includes thrash detection: warns at 3+ spawn attempts for same issue

**Known limitation:** Correlated failures — when beads is unavailable, L4, L5, and L6 all degrade simultaneously. No atomic CAS between L5 and L6 (TOCTOU race window).

**Structural redesign recommended:** See `.kb/investigations/2026-03-01-inv-structural-review-daemon-dedup-after.md`

### Verification Threshold (VerificationTracker)

Prevents unchecked autonomous operation by pausing the daemon after N unverified completions.

| Parameter | Value | Purpose |
|-----------|-------|---------|
| `VerificationPauseThreshold` | 3 (default) | Max completions before pause |
| Pause trigger | `IsPaused() == true` | Daemon stops spawning new work |
| Resume: human verification | `orch complete` writes `~/.orch/daemon-verification.signal` | Clears pause, resets counter |
| Resume: manual override | `orch daemon resume` writes `~/.orch/daemon-resume.signal` | Clears pause without verification |
| Disable | threshold = 0 | Never pauses |

**How it works:** Each time `CompletionOnce()` labels an agent as `daemon:ready-review`, VerificationTracker increments its counter (deduped by beads ID — same agent counted only once). When counter >= threshold, `IsPaused()` returns true and the daemon skips spawning until a human runs `orch complete` or `orch daemon resume`.

**Cross-project gap:** `SeedFromBacklog()` reads from `verify.ListUnverifiedWork()` which uses the daemon home's checkpoint file only (orch-go). Cross-project completions ARE counted during runtime (any `RecordCompletion(beadsID)` call), but the initial seed on daemon startup only knows about orch-go's unverified work. This means restarting the daemon after cross-project completions resets the counter for non-orch-go work.

**Why this matters:** Without this threshold, the daemon could auto-complete dozens of agents overnight with no human verification. The threshold forces periodic human review, catching verification failures before they compound.

**Source:** `pkg/daemon/verification_tracker.go`, seeding in `cmd/orch/daemon.go`

### Orphan Detection

Detects dead agents (session/tmux gone but issue still `in_progress`) and resets them to `open` for potential respawn.

**Detection logic:**
1. Find all `in_progress` agents (periodic check, configurable interval)
2. Skip agents with `Phase: Complete` (waiting for orchestrator review, not orphaned)
3. Skip agents younger than `OrphanAgeThreshold` (default: 1h)
4. For remaining: check if OpenCode session or tmux window exists
5. If no session/window exists → reset issue status from `in_progress` to `open`

**Spawn cache interaction:** Orphan detector intentionally does NOT clear spawn cache entries after resetting an orphan. The 6h TTL in L1 provides a natural cooldown, preventing thrash loops where an issue is repeatedly spawned and fails. Trade-off: legitimate retries are blocked for the remainder of the TTL.

**Source:** `pkg/daemon/orphan_detector.go`

### Cross-Project Operation

The daemon **runs from orch-go** (its orchestration home) but **polls all kb-registered projects** via ProjectRegistry.

**Project discovery mechanism:**
1. `ProjectRegistry` queries `kb projects list --json` at startup
2. For each project, reads `.beads/config.yaml` to extract issue prefix (e.g., `orch-go`, `pw`)
3. `ListReadyIssuesMultiProject()` iterates all registered projects, queries each project's beads DB
4. `Resolve(issueID)` maps beads ID prefix to project directory for `--workdir` routing

**Cross-project scoping:**
- `~/.orch/groups.yaml` defines project groups with account routing
- `--cross-project` flag enables multi-project polling
- `--group` flag scopes to a specific project group

**What is scoped to daemon home (orch-go):**
- Daemon process runs from orch-go working directory
- VerificationTracker seed reads orch-go checkpoints only (see gap above)
- `~/.orch/` config and state files

**What spans all projects:**
- Issue polling (via ProjectRegistry)
- Spawn execution (via `--workdir` flag)
- Runtime VerificationTracker counting (any beads ID)
- Capacity pool (shared across all projects)

**Source:** `pkg/daemon/project_resolution.go`, `pkg/daemon/daemon.go`

### Triage Workflow

**Labels control spawn readiness:**

| Label | Meaning | Who Sets | Daemon Action |
|-------|---------|----------|---------------|
| `triage:ready` | Confident spawn | Orchestrator or issue-creation agent | Spawns immediately |
| `triage:review` | Needs review | issue-creation agent | Skips, waits for orchestrator review |
| (no label) | Default triage | N/A | Skips |

**The flow:** User reports symptom → Issue created with `triage:review` → Orchestrator reviews, relabels `triage:ready` → Daemon auto-spawns on next poll.

**Why this pattern:** Separates judgment (orchestrator) from execution (daemon). Daemon handles batch/overnight work, orchestrator stays available for triage and synthesis.

**Triage routing success rate:** Daemon-routed (`triage:ready`) agents succeed 9.4x more often than direct spawns. This validates the triage-first workflow.

---

## Why This Fails

### 1. Capacity Starvation

**What happens:** Pool shows MaxAgents active, but `orch status` shows fewer actual agents running.

**Root cause:** Spawn failures don't release slots. Agent counted against pool, but spawn fails — slot never released.

**Fix:** Reconciliation with OpenCode each poll cycle. Orphan detector catches agents where session died after initial spawn.

### 2. Duplicate Spawns

**What happens:** Same issue spawned multiple times by daemon.

**Root causes:** (a) Spawn latency: issue hasn't transitioned to `in_progress` by next poll. (b) Content duplicates: different beads IDs with identical titles. (c) Correlated service failure: beads unavailable degrades L4/L5/L6 simultaneously.

**Fix:** 6-layer dedup pipeline. See "Spawn Dedup Pipeline" section.

### 3. Skill Inference Mismatch

**What happens:** Daemon spawns wrong skill for issue type.

**Root cause:** Issue type doesn't match actual work needed, or title/description heuristics fire incorrectly.

**Fix:** Use `skill:*` label for explicit override (Priority 1 in inference chain). Or spawn manually with correct skill.

### 4. Verification Threshold Cross-Project Gap

**What happens:** After daemon restart, unverified cross-project completions are not counted in the threshold. Daemon spawns new work despite unreviewed agents from other projects.

**Root cause:** `SeedFromBacklog()` only reads orch-go checkpoint file. No cross-project checkpoint aggregation.

**Impact:** Low risk in practice (most work is orch-go), but a correctness gap.

### 5. Orphan Thrash Prevention Over-Blocks

**What happens:** Legitimate retry is blocked for up to 6 hours after orphan detection because spawn cache entry is preserved.

**Root cause:** Intentional design to prevent thrash loops, but no mechanism to distinguish "flaky failure" from "permanent failure".

**Workaround:** Manual spawn bypasses spawn cache: `orch spawn SKILL "task" --issue <id>`

---

## Constraints

### Why Poll Instead of Event-Driven?

**Constraint:** Daemon polls beads every 60s instead of subscribing to beads events.
**This enables:** Simple, reliable batch processing without beads architecture changes.
**This constrains:** Up to 60s latency between "issue ready" and "daemon spawns".
**Workaround:** Manual spawn for urgent work: `orch spawn SKILL "task" --issue <id>`

### Why Skill Inference Has 4 Levels?

**Constraint:** Multi-level inference (label → title → description → type) adds complexity but covers common override cases.
**This enables:** Explicit override via `skill:*` label without changing issue type. Title conventions for consistent routing.
**This constrains:** Heuristic layers (title/description) can fire incorrectly. Label is the only reliable override.

### Why MaxAgents Hard Limit?

**Constraint:** WorkerPool enforces hard limit on concurrent agents (default: 5).
**This enables:** Controlled resource usage, prevents TPM throttling (observed at >60% session usage).
**This constrains:** Cannot spawn more than limit even when system has capacity.
**Workaround:** CLI override: `orch daemon run --max-agents N`

### Why VerificationTracker Pauses All Projects?

**Constraint:** Threshold counter includes completions from all projects, but seed only covers orch-go.
**This enables:** Simple pause mechanism that forces human review of autonomous work.
**This constrains:** Cross-project restart gap (see "Why This Fails" #4). Threshold of 3 may pause too aggressively during batch operations.
**Workaround:** `orch daemon resume` for manual override; threshold 0 disables entirely.

---

## Evolution

### Phase 1: Poll and Spawn (Dec 2025)
Basic poll loop, skill inference from type only, single-project operation.

### Phase 2: Completion Loop (Late Dec 2025)
Separate completion polling, `Phase: Complete` detection, issue auto-close.

### Phase 3: Capacity Management (Dec 28-30, 2025)
WorkerPool with MaxAgents, OpenCode reconciliation, simple spawn tracker (L1).

### Phase 4: Cross-Project Operation (Jan 2-6, 2026)
ProjectRegistry via `kb projects list`, `--workdir` flag routing, per-project beads polling.

### Phase 5: kb reflect Integration (Jan 2026)
Periodic `kb reflect` execution, two-tier reflection automation (synthesis + open types).

### Phase 6: Multi-Layer Dedup (Jan-Mar 2026)
Evolved from simple ID tracker to 6-layer pipeline. Each layer patched a gap: L2 (session existence), L3/L4 (title dedup), L5/L6 (structural gates). Disk persistence for restart survival.

### Phase 7: Safety Mechanisms (Feb-Mar 2026)
VerificationTracker (pause after N unverified completions), orphan detector (reset dead agents), orphan-spawn-cache interaction (prevent thrash loops).

---

## References

**Guide:**
- `.kb/guides/daemon.md` - Procedural guide (commands, configuration, troubleshooting). Verified Mar 1, 2026.

**Decisions:**
- `.kb/decisions/2026-01-16-single-daemon-orchestration-home.md` - Original single-daemon decision (superseded)
- `.kb/decisions/2026-02-25-project-group-model.md` - Project groups via `~/.orch/groups.yaml`

**Models:**
- `.kb/models/spawn-architecture/model.md` - How `orch work` spawns agents
- `.kb/models/beads-integration-architecture/model.md` - How daemon polls beads
- `.kb/models/completion-verification/model.md` - How completion loop verifies agents

**Source code:**
- `pkg/daemon/daemon.go` - Main poll loop, spawnIssue() with 6-layer dedup
- `pkg/daemon/pool.go` - WorkerPool capacity management
- `pkg/daemon/skill_inference.go` - 4-level skill inference chain
- `pkg/daemon/spawn_tracker.go` - L1/L3 dedup, disk persistence, thrash detection
- `pkg/daemon/session_dedup.go` - L2 session/tmux existence checking
- `pkg/daemon/verification_tracker.go` - Verification pause threshold
- `pkg/daemon/orphan_detector.go` - Dead agent detection and reset
- `pkg/daemon/project_resolution.go` - ProjectRegistry for cross-project polling
- `pkg/daemon/completion_processing.go` - Beads-polling completion detection
- `pkg/daemon/reflect.go` - kb reflect integration
- `pkg/daemon/status.go` - Status file management
- `cmd/orch/daemon.go` - CLI commands (run, preview, resume)
