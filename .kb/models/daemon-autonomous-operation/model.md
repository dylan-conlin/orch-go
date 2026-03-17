# Model: Daemon Autonomous Operation

**Domain:** Daemon / Autonomous Spawning / Batch Processing
**Last Updated:** 2026-03-17
**Synthesized From:** 39+ investigations + daemon.md guide (verified Mar 1, 2026) + 36 probes (Feb 9 – Mar 12, 2026) on poll loops, skill inference, capacity management, completion tracking, cross-project operation, dedup pipeline, verification threshold, orphan detection

---

## Summary (30 seconds)

The daemon is an **autonomous agent spawner** that operates in a **poll-spawn-complete cycle**: polls beads for `triage:ready` issues across all kb-registered projects, infers skill via a **4-level priority chain** (label → title → description → type fallback), spawns within capacity limits, monitors for `Phase: Complete`, and auto-completes routine work. The daemon **runs from orch-go** (its orchestration home) but **polls cross-project** via ProjectRegistry. Four safety mechanisms prevent runaway operation: a **PID lock** prevents concurrent daemon processes (L0), a **6-layer spawn dedup pipeline** (L1-L6) prevents duplicate spawns, a **VerificationTracker** pauses spawning after N unverified completions, and an **orphan detector** resets dead agents while preserving spawn cache cooldown.

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
│     - Infer model (from skill type)                          │
│     - Spawn: orch work <beads-id> --model <model>            │
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

**CompletionOnce failure mode:** If `CompletionOnce()` returns an error, the `CompletionFailureTracker` records it. After 3 consecutive failures, the daemon pauses spawning. Error is always logged to stderr (not verbose-only). This prevents orphaned agents accumulating silently when completion processing is broken.

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

**Non-spawnable types:** `epic` and `chore` are not spawnable. Epics are containers for child issues; chores are non-agent maintenance. The daemon skips these with a rejection reason. `question` type is also not spawnable — all question issues are routed to orchestrator queue regardless of labels.

**Model inference by skill:** Deep reasoning skills (investigation, architect, systematic-debugging, codebase-audit, research) → Opus. Implementation skills (feature-impl, issue-creation) → Sonnet. Model is inferred via `InferModelFromSkill()` in `skill_inference.go` and passed as `--model` to `orch work`. When the extraction gate replaces a skill with `feature-impl`, the model is also re-inferred.

**Source:** `pkg/daemon/skill_inference.go`

### Spawn Prerequisites: Fail-Fast Gates

All spawn prerequisites are **hard gates**, not soft warnings (constraint kb-035b64). The Feb 14 duplicate spawn incident (10 spawns in 20 minutes) established this constraint.

| Prerequisite | Behavior on failure | Notes |
|--------------|---------------------|-------|
| Beads status update (L6) | **Fail-fast** — skip spawn, release slot | Primary dedup gate |
| Dependency check | Skip issue (`continue`) | Was warn-and-continue; fixed Feb 17 |
| Epic expansion | Return error to caller | Was warn-and-continue; fixed Feb 17 |
| Extraction gate | Skip issue, Processed=false | Was warn-and-continue; fixed Feb 17 |
| Rollback after spawn failure | Return wrapped error to stderr | Was warn-and-continue; fixed Feb 17 |

**Acceptable warn-and-continue:** Logging failures, status file writes, resume signal checks, event logging — these are monitoring/observability, not spawn correctness.

**Dependency type semantics:** `blocks` edges prevent spawning (blocking dependency). `relates_to` and `parent-child` edges do NOT block spawning. A bug in `GetBlockingDependencies()` previously treated `relates_to` as blocking via a catch-all `default` case; fixed Feb 16 with explicit type switching.

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

### Single-Instance Guard (PID Lock)

The daemon requires exactly one running process. A PID lock at `~/.orch/daemon.pid` enforces this:

- Second invocation fails immediately with `"daemon already running: PID N"`
- Stale PID files (from crashed daemons) detected via `kill(pid, 0)` and cleaned up
- PID lock released on graceful shutdown via `defer`; crash leaves stale file
- PID included in status file for dashboard visibility

This addresses **process-level** duplication that issue-level dedup (L1-L6) cannot catch — each daemon instance has its own in-memory tracker.

**Dashboard restart default:** `orch-dashboard restart` does NOT auto-start the daemon (disabled by default since Feb 9, 2026). Opt-in: `ORCH_DASHBOARD_START_DAEMON=1 orch-dashboard restart`.

### Spawn Dedup Pipeline (7-Layer: L0 + L1–L6)

The daemon prevents duplicate spawns via 7 sequential layers. Layer 0 is process-level; L1-L6 operate within a single process.

| Layer | Check | Source | Fail Mode | Nature |
|-------|-------|--------|-----------|--------|
| L0 | PID lock (single process) | `daemon.pid` | Exit with error | Hard gate |
| L1 | SpawnedIssueTracker (ID-based, 6h TTL) | `spawn_tracker.go` | Blocks spawn | Heuristic |
| L2 | Session/Tmux existence check | `session_dedup.go` | Blocks (fail-open if API down) | Heuristic |
| L3 | Title dedup (in-memory, TTL-coupled) | `spawn_tracker.go` | Blocks spawn | Heuristic |
| L4 | Title dedup (beads DB query) | `spawn_tracker.go` | Blocks (fail-open) | Structural-ish |
| L5 | Fresh beads status re-check | `daemon.go` | Blocks (fail-open) | Structural |
| L6 | UpdateStatus("in_progress") | `daemon.go` | **Fail-fast** | Structural (PRIMARY) |

**Key properties:**
- L6 is the only fail-fast layer — if it fails, spawn is aborted (not warn-and-continue)
- L2, L4, L5 are fail-open — they allow spawn if their backing service is unavailable
- L1-L3 survive daemon restarts via disk persistence (`~/.orch/spawn_cache.json`)
- L1 includes thrash detection: warns at 3+ spawn attempts for same issue
- L3/L4 are **content-aware** (title-based), added Feb 16 after 9 duplicate issues with identical titles but different IDs bypassed all ID-based layers

**Known limitation:** Correlated failures — when beads is unavailable, L4, L5, and L6 all degrade simultaneously. No atomic CAS between L5 and L6 (TOCTOU race window). Content-aware dedup (`FindInProgressByTitle`) only checks the local beads database — cross-project dedup is a blind spot.

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

### Extraction Gate (Hotspot Auto-Extraction)

When the daemon picks up an issue targeting a CRITICAL hotspot file (>1500 lines), it creates an extraction issue instead of spawning the original work directly:

1. `CheckExtractionNeeded(issue, d.HotspotChecker)` runs on every issue in `Once()`
2. If extraction needed: `createFunc(extraction.ExtractionTask, issue.ID)` creates extraction issue and blocks parent
3. Daemon spawns the extraction issue instead; parent waits for extraction to close before next poll

**Extraction recursion guard:** Issues whose titles start with `"Extract "` are skipped by `CheckExtractionNeeded()`. Without this guard, extraction issues containing the critical filename in their title triggered recursive extraction chains (titles concatenated 4x, e.g., `xy7n`). Fixed Feb 16.

**Extraction cascade pattern (observed Mar 10-17):** The extraction gate participates in a larger feedback loop: pre-commit gate signals → daemon detects hotspot → spawns extraction architect → extraction creates new smaller files → file count grows but hotspot count drops. In the first week post-gate-wiring (Mar 10-17), this cascade reduced files >800 lines from 12 to 3 (75% reduction). daemon.go went from 1,559 to 197 lines. Growth was redistributed from few large files to many small files (125→172 files, but 75% fewer hotspots). The gate's direct blocking effect is negligible (2 blocks, both bypassed), but its signaling role in triggering this cascade is the primary mechanism of structural improvement.

**Extraction convergence gap:** If a file remains >1500 lines after an extraction closes, the parent issue becomes unblocked and triggers a new extraction. No convergence check exists to detect "extraction was already attempted but file didn't shrink." The content-aware dedup (L3/L4) catches same-title extraction issues, but not extraction issues with slightly different titles.

**Source:** `pkg/daemon/extraction.go`, `CheckExtractionNeeded()`, `DefaultCreateExtractionIssue()`

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

**Cross-project spawn CWD bug (fixed Feb 25):** `runWork()` called `verify.GetIssue(beadsID)` before consulting `--workdir`. All beads calls defaulted to `FindSocketPath("")` which uses `os.Getwd()` — the daemon's CWD, not the target project. Fix: set `beads.DefaultDir` from `--workdir` at the start of `runWork()`. Prior to this fix, 100% of cross-project spawns failed silently.

**Cross-project completion (implemented Mar 2026):** `listCompletedAgentsMultiProject()` iterates all configured project dirs, calls single-project completion for each, tags results with `agent.ProjectDir`. `ProcessCompletion()` uses `agent.ProjectDir` for beads operations. **Residual gap:** VerificationTracker seeding (`SeedFromBacklog()`) only reads orch-go checkpoints — cross-project completions are counted at runtime but lost on daemon restart.

**Account routing gap:** The daemon uses a single globally-active account for all spawns. No per-group account routing exists. Work-account SCS projects use the same personal account as orch-go.

**Project group config gap (as of Mar 12, 2026):** The `pkg/group/` package is fully implemented and `kbcontext_filter.go` correctly consumes it, but `~/.kb/groups.yaml` does not exist. All group-based features fall back to hardcoded `OrchEcosystemRepos`. Creating the config file is the single highest-leverage change for cross-project friction — zero code changes required.

**Source:** `pkg/daemon/project_resolution.go`, `pkg/daemon/daemon.go`, `pkg/daemon/completion_processing.go`

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

### Periodic Task System

Beyond poll-spawn-complete, the daemon runs a set of **periodic tasks** at configurable intervals:

| Task | Default Interval | Purpose |
|------|-----------------|---------|
| kb reflect | 1h | Synthesis of quick entries and open investigations |
| Cleanup | 6h | Close stale tmux windows for completed agents |
| Recovery | 5m | Detect and reset idle agents |
| Orphan detection | configurable | Reset dead agents to `open` |
| Knowledge health | configurable | Create issues for stale kb artifacts |

**Daemon cleanup note:** Even when OpenCode manages session TTL, `RunPeriodicCleanup` must still run to close stale tmux windows. The daemon cleanup runs when due and updates `lastCleanup`.

**Agreement checking (planned):** Agreement checking follows the same periodic task pattern — detect-create-spawn-verify sub-cycle feeding into the existing poll-spawn-complete loop. Agreement failures create `triage:ready` issues which daemon then spawns. This makes the daemon self-healing: it detects its own contract violations.

### Config Construction Divergence

**Warning:** Daemon `Config` is constructed at 4 independent call sites in `cmd/orch/daemon.go`. The compiler does not catch omitted fields (Go zero-values are valid). Known divergences:

| Field | Default | Production (`runDaemonLoop`) |
|-------|---------|------------------------------|
| `RecoveryEnabled` | true | **false** (never set → recovery silently disabled) |
| `MaxSpawnsPerHour` | 20 | **0** (rate limiter never initialized) |
| `VerificationPauseThreshold` | 3 | 0 in `runDaemonPreview` only |

The convenience constructor `New()` uses `DefaultConfig()` correctly, but it's not used in production CLI paths. Any new `Config` field requires updating all 4 construction sites manually.

**Source:** `cmd/orch/daemon.go` (4 construction sites), `pkg/daemon/daemon.go:DefaultConfig()`

### Status File Accuracy

**Known gap:** `orch status` and dashboard API report daemon as "running" based on file existence, not process liveness.

- `handleDaemon()` (serve_system.go): sets `Running = true` if `daemon-status.json` is readable
- `readDaemonStatus()` (status_cmd.go): returns stale struct without PID liveness check
- When daemon dies without graceful shutdown (SIGKILL, crash), stale files persist
- `DetermineStatus()` has staleness logic (2× poll interval) but it's only used by the daemon writer

**Fix:** Check PID liveness (`kill(pid, 0)`) or staleness threshold in the reader path.

---

## Why This Fails

### 1. Capacity Starvation (Two Axes)

**Slot starvation:**
- **What happens:** Pool shows MaxAgents active, but `orch status` shows fewer actual agents running.
- **Root cause:** Spawn failures don't release slots. Agent counted against pool, but spawn fails — slot never released.
- **Fix:** Reconciliation with OpenCode each poll cycle. Orphan detector catches agents where session died after initial spawn.

**Queue poisoning (new axis):**
- **What happens:** A persistently-failing issue retries every 15-second poll cycle indefinitely, consuming CPU/IO without consuming a slot.
- **Root cause:** On spawn failure, `spawnIssue()` rolls status back to `open` and unmarks from spawn tracker. The `skippedThisCycle` map resets every cycle — no persistent memory.
- **Fix:** Per-issue circuit breaker in `SpawnFailureTracker`. After `MaxIssueFailures` (default: 3) consecutive failures for a single issue, that issue is circuit-broken and skipped in `NextIssueExcluding()`. Successful spawn clears the counter.
- **Common trigger:** Cross-project issues where `--workdir` doesn't exist or can't be resolved locally.

### 2. Duplicate Spawns

**What happens:** Same issue spawned multiple times by daemon.

**Root causes:**
- (a) **Spawn latency:** issue hasn't transitioned to `in_progress` by next poll
- (b) **Content duplicates:** different beads IDs with identical titles (L3/L4 address this)
- (c) **Correlated service failure:** beads unavailable degrades L4/L5/L6 simultaneously
- (d) **Fail-open + UpdateBeadsStatus failure:** prior to Feb 14 fix, `UpdateBeadsStatus` failing caused daemon to continue spawning, leaving issue `open` for next poll
- (e) **Manual spawn race:** orchestrator creates issue with `triage:ready`, then immediately spawns manually with `--bypass-triage`. Daemon picks up the `triage:ready` issue during the spawn pipeline's pre-flight checks (before `SetupBeadsTracking` sets `in_progress`). Result: two agents on same issue.
- (f) **Completion-spawn loop cycle:** completion loop adds `daemon:ready-review` but does not remove `triage:ready`. Spawn loop only checked `triage:ready` (not daemon labels), so completed issues re-entered the spawn queue. Combined with stale Phase: Complete comments from reused issues, the completion loop would reprocess the same Phase: Complete 3x (Mar 11, 2026 orlcp incident).

**Fix:** 7-layer dedup pipeline (L0 process lock + L1-L6). See "Spawn Dedup Pipeline" section. For (e): manual spawn with `--bypass-triage` removes `triage:ready`/`triage:approved` labels immediately (before pre-flight checks), closing the race window. For (f): three layers — spawn queue now filters `daemon:ready-review`/`daemon:verification-failed` labels; completion processing removes `triage:ready` after adding `daemon:ready-review`; in-memory `CompletionDedupTracker` prevents same Phase:Complete from being processed twice. Added Mar 11, 2026.

### 3. Skill Inference Mismatch

**What happens:** Daemon spawns wrong skill for issue type.

**Root cause:** Issue type doesn't match actual work needed, or title/description heuristics fire incorrectly.

**Fix:** Use `skill:*` label for explicit override (Priority 1 in inference chain). Or spawn manually with correct skill.

### 4. Extraction Recursion / Unbounded Issue Creation

**What happens:** Extraction issues trigger more extraction checks, creating cascading chains of duplicate extraction issues with progressively concatenated titles.

**Root causes:**
1. `GenerateExtractionTask()` embeds the target file path in the issue title
2. `InferTargetFilesFromIssue()` parses file paths from issue titles
3. Without the guard, `CheckExtractionNeeded()` ran on extraction issues themselves

**Fix:** Issues with titles starting with `"Extract "` are skipped by `CheckExtractionNeeded()` (added Feb 16).

**Remaining gap:** If parent file remains >1500 lines after an extraction closes, parent issue unblocks and triggers another extraction. No convergence condition exists.

### 5. Model Incompatibility Stall

**What happens:** Daemon-spawned agents using non-Claude models (e.g., GPT-5.2-codex) stall without completing. The spawn counts as successful (session exists) but produces no useful output.

**Failure modes observed with GPT-5.2-codex:**
1. Hallucinated constraints: agent invented "orchestrator policy forbids reading code files" (non-existent restriction)
2. Excessive token consumption: 145K tokens on exploration, hit context window, session terminated mid-exploration
3. Silent session termination: no error message, no completion report

**Root cause:** 63-76KB SPAWN_CONTEXT files consume ~40-50K GPT tokens (GPT tokenizes ~2x more than Claude). Combined with tool definitions and CLAUDE.md, the initial context can consume 60-80% of GPT-5.2's context window before the agent starts working.

**Config resolution bug:** `runWork()` injects user config `default_model` as CLI-flag priority, bypassing project config's `opencode.model` override. The project config `opencode.model: flash` is never applied when `default_model: codex` is set in user config.

**Fix:** Use Anthropic models (opus/sonnet) for daemon spawns. Remove `default_model` from user config or add per-daemon config (`daemon.model`).

### 6. Verification Threshold Cross-Project Gap

**What happens:** After daemon restart, unverified cross-project completions are not counted in the threshold. Daemon spawns new work despite unreviewed agents from other projects.

**Root cause:** `SeedFromBacklog()` only reads orch-go checkpoint file. No cross-project checkpoint aggregation.

**Impact:** Low risk in practice (most work is orch-go), but a correctness gap.

### 7. Orphan Thrash Prevention Over-Blocks

**What happens:** Legitimate retry is blocked for up to 6 hours after orphan detection because spawn cache entry is preserved.

**Root cause:** Intentional design to prevent thrash loops, but no mechanism to distinguish "flaky failure" from "permanent failure".

**Workaround:** Manual spawn bypasses spawn cache: `orch spawn SKILL "task" --issue <id>`

### 8. Function Field Nil Dereference

**What happens:** New methods added to `Daemon` call function fields directly instead of through resolve methods, causing nil pointer panics in production.

**Root cause:** `listIssuesFunc` and similar injectable fields are nil in production (only set in tests). New code that copies the field name without using the resolver (`resolveListIssuesFunc()`) will panic. Example: `preview.go` called `d.listIssuesFunc()` directly instead of `d.resolveListIssuesFunc()()`.

**Pattern:** Always use `d.resolve*Func()` (not `d.*Func`) for injectable function fields. The resolve method handles nil fallback.

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
**This constrains:** Cross-project restart gap (see "Why This Fails" #6). Threshold of 3 may pause too aggressively during batch operations.
**Workaround:** `orch daemon resume` for manual override; threshold 0 disables entirely.

### Why Spawn Prerequisites Are Hard Gates?

**Constraint:** Any spawn prerequisite failure (beads status update, dependency check, extraction gate) must abort the spawn — never warn and continue.
**This enables:** Prevents duplicate spawns and surfaces real infrastructure problems (beads unavailable) instead of hiding them.
**This constrains:** Spawn failure becomes a visible event that surfaces in logs/health metrics instead of silently proceeding.
**Historical note:** The Feb 14, 2026 incident (orch-go-w50 spawned 10 times in 20 minutes) was caused by violating this constraint.

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
Evolved from simple ID tracker to 7-layer pipeline. Each layer patched a gap: L0 (PID lock), L2 (session existence), L3/L4 (content-aware title dedup), L5/L6 (structural gates). Disk persistence for restart survival.

### Phase 7: Safety Mechanisms (Feb-Mar 2026)
VerificationTracker (pause after N unverified completions), orphan detector (reset dead agents), orphan-spawn-cache interaction (prevent thrash loops), per-issue circuit breaker (queue poisoning prevention), CompletionFailureTracker (pause spawning when completion processing is broken), fail-fast prerequisite gates (all spawn prerequisites converted from warn-and-continue).

### Phase 8: Extraction and Model Inference (Feb 2026)
Hotspot auto-extraction gate, extraction recursion guard, model inference from skill type, content-aware dedup for duplicate-title issues.

---

## References

**Guide:**
- `.kb/guides/daemon.md` - Procedural guide (commands, configuration, troubleshooting). Verified Mar 1, 2026.

**Decisions:**
- `.kb/decisions/2026-01-16-single-daemon-orchestration-home.md` - Original single-daemon decision (superseded — code now implements cross-project polling)
- `.kb/decisions/2026-02-25-project-group-model.md` - Project groups via `~/.orch/groups.yaml`

**Models:**
- `.kb/models/spawn-architecture/model.md` - How `orch work` spawns agents
- `.kb/models/beads-integration-architecture/model.md` - How daemon polls beads
- `.kb/models/completion-verification/model.md` - How completion loop verifies agents

**Source code:**
- `pkg/daemon/daemon.go` - Main poll loop, spawnIssue() with 7-layer dedup
- `pkg/daemon/pool.go` - WorkerPool capacity management
- `pkg/daemon/skill_inference.go` - 4-level skill inference chain, model inference
- `pkg/daemon/spawn_tracker.go` - L1/L3 dedup, disk persistence, thrash detection, per-issue circuit breaker
- `pkg/daemon/session_dedup.go` - L2 session/tmux existence checking
- `pkg/daemon/verification_tracker.go` - Verification pause threshold
- `pkg/daemon/orphan_detector.go` - Dead agent detection and reset
- `pkg/daemon/project_resolution.go` - ProjectRegistry for cross-project polling
- `pkg/daemon/completion_processing.go` - Beads-polling completion detection
- `pkg/daemon/completion_failure_tracker.go` - Completion processing health tracking
- `pkg/daemon/extraction.go` - Hotspot auto-extraction gate
- `pkg/daemon/reflect.go` - kb reflect integration
- `pkg/daemon/status.go` - Status file management
- `pkg/daemon/periodic.go` - Extracted periodic task runners
- `pkg/daemon/preview.go` - Extracted preview/rejection logic
- `pkg/daemon/capacity.go` - Extracted pool/rate-limit convenience methods
- `cmd/orch/daemon.go` - CLI commands (run, preview, resume)
- `cmd/orch/daemon_periodic.go` - Periodic task handlers

---

## Merged Probes

All 34 probes merged as of 2026-03-06. Listed chronologically with 1-line summaries:

| Probe | Date | Verdict | Summary |
|-------|------|---------|---------|
| `2026-02-09-dashboard-restart-daemon-autostart-default-disabled` | Feb 9 | Extends | Dashboard restart auto-started daemon by default; fixed to opt-in via `ORCH_DASHBOARD_START_DAEMON=1` |
| `2026-02-14-control-plane-heuristic-calibration` | Feb 14 | Extends | Single-day commit count is wrong circuit breaker signal; entropy spiral required sustained velocity + zero human interaction + degrading quality (multi-dimensional) |
| `2026-02-14-daemon-duplicate-spawn-ttl-fragility` | Feb 14 | Extends | SpawnedIssueTracker 6h TTL was primary dedup layer; fix made beads status update the primary (persistent) gate |
| `2026-02-14-daemon-duplicate-spawn-feb14-incident` | Feb 14 | Contradicts | Prior fix was effective only when `UpdateBeadsStatus` succeeds; when it fails, daemon was continuing with spawn (warn-and-continue) → fixed to fail-fast |
| `2026-02-15-daemon-config-construction-divergence-audit` | Feb 15 | Contradicts + Extends | Recovery never runs in production (RecoveryEnabled=false in runDaemonLoop); MaxSpawnsPerHour rate limiter never initialized; 4 divergent construction sites |
| `2026-02-15-daemon-warn-continue-anti-pattern-audit` | Feb 15 | Contradicts | Feb 14 incident was NOT isolated; 5 additional warn-and-continue patterns in spawn prerequisites; established kb-035b64 constraint |
| `2026-02-16-daemon-dedup-fundamentally-broken-content-aware-fix` | Feb 16 | Extends | All dedup layers were ID-keyed only; 5 issues with identical titles all spawned; added two-layer content-aware (title-based) dedup |
| `2026-02-16-daemon-relates-to-links-blocking` | Feb 16 | Extends | `relates_to` dependency type incorrectly treated as blocking via catch-all `default` case; fixed to explicit type switching |
| `2026-02-16-duplicate-extraction-provenance-trace` | Feb 16 | Contradicts + Extends | Duplicate spawns not just spawn latency — extraction logic without convergence + recursive self-triggering were dominant mechanisms |
| `2026-02-16-extraction-recursion-fix` | Feb 16 | Extends | Extraction issues triggered recursive extraction chains; fixed with title-prefix guard (`strings.HasPrefix(title, "Extract ")`) |
| `2026-02-16-test-suite-health-new-failures` | Feb 16 | Extends | Extraction file path inference had over-aggressive adjacent-word heuristic producing nonsensical paths; removed; synthesis gate wasn't using `IsKnowledgeProducingSkill()` |
| `2026-02-17-daemon-completion-fail-fast-fix` | Feb 17 | Extends | CompletionOnce errors were only logged in verbose mode; added `CompletionFailureTracker` that pauses spawning after 3 consecutive failures |
| `2026-02-17-daemon-dependency-check-fail-fast-fix` | Feb 17 | Confirms + Extends | Dependency check converted from warn-and-continue to fail-fast (skip issue on error) |
| `2026-02-17-daemon-epic-expansion-fail-fast` | Feb 17 | Confirms | `expandTriageReadyEpics` warn-and-continue silently dropped epic children; converted to error return |
| `2026-02-17-daemon-rollback-fail-fast-fix` | Feb 17 | Extends | Rollback-after-spawn-failure was warn-and-continue; now returns wrapped error to stderr immediately; rollback failure tracked in SpawnFailureTracker |
| `2026-02-17-daemon-test-fail-fast-fix` | Feb 17 | Confirms | Fail-fast changes required injecting `updateBeadsStatusFunc` field for testability; 50+ tests updated |
| `2026-02-17-extraction-gate-fail-fast-fix` | Feb 17 | Confirms | Extraction gate was last remaining warn-and-continue spawn prerequisite; now fail-fast (Processed=false, no spawn) |
| `2026-02-18-probe-config-spawn-override-audit` | Feb 18 | Extends | Backend resolution silently ignores malformed user config; project config `spawn_mode` can override user backend even when unset |
| `2026-02-18-probe-daemon-cleanup-after-pkg-cleanup-deletion` | Feb 18 | Extends | Daemon periodic cleanup must still run to close stale tmux windows even when OpenCode handles session TTL |
| `2026-02-18-probe-project-config-spawn-mode-explicitness` | Feb 18 | Extends | Backend resolution now treats project `spawn_mode` as explicit only when YAML key is present; user backend not overridden by project defaults |
| `2026-02-19-probe-config-surface-area-extraction` | Feb 19 | Extends | Adding 1 config boolean requires 10-12 file touches; three duplicate Config structs exist; agent spiraling caused by large file exploration costs |
| `2026-02-19-probe-daemon-spawned-agents-stall-gpt52-codex` | Feb 19 | Extends | GPT-5.2-codex agents stall with hallucinated constraints, context exhaustion; config bug injects `default_model` as CLI priority bypassing project override |
| `2026-02-19-probe-daemon-go-extraction-completeness` | Feb 19 | Extends | All P0/P1 extractions from Jan 4 plan complete, but daemon.go grew from new features (accretion gravity); 200-300 line target unreachable via extraction alone |
| `2026-02-20-probe-daemon-model-inference-from-skill` | Feb 20 | Extends | Model inference added alongside skill inference; opus for deep-reasoning skills, sonnet for implementation; flows through entire spawn path |
| `2026-02-20-probe-daemon-once-dedup-shared-spawn-path` | Feb 20 | Confirms | OnceExcluding and OnceWithSlot share dedup/status-update logic after refactor; status update failure blocks spawn and releases slot |
| `2026-02-24-probe-preview-nil-pointer-listissuesfunc` | Feb 24 | Extends | `Preview()` called `d.listIssuesFunc()` directly (nil in production) instead of `d.resolveListIssuesFunc()()` — nil dereference bug |
| `2026-02-24-probe-daemon-single-instance-pid-lock` | Feb 24 | Extends | No single-instance guard allowed concurrent daemon processes; added PID lock at `~/.orch/daemon.pid` |
| `2026-02-25-probe-cross-project-spawn-beads-defaultdir` | Feb 25 | Extends | `runWork()` used daemon CWD for beads lookups before consulting `--workdir`; 100% cross-project spawn failure rate; fixed by setting `beads.DefaultDir` at start of `runWork()` |
| `2026-02-25-probe-cross-repo-orchestration-consequences` | Feb 25 | Extends | Daemon can SPAWN cross-project but cannot COMPLETE cross-project; parent issue orphaning is a critical workflow gap; account routing is single-account |
| `2026-02-25-probe-project-group-model-design` | Feb 25 | Contradicts + Extends | Decision 2026-01-16 (no cross-project polling) is stale — code already implements it; kb context uses hardcoded `OrchEcosystemRepos` blocking SCS sibling context; account routing is missing |
| `2026-02-27-probe-phantom-agent-spawns-daemon-status-stale-file` | Feb 27 | Extends | Stale `daemon-status.json` reports daemon as "running" after death; both API and CLI check file existence not PID liveness |
| `2026-02-28-probe-daemon-agreements-integration-design` | Feb 28 | Extends | Agreement checking fits cleanly into periodic task pattern; creates self-healing detect-create-spawn-verify sub-cycle on top of poll-spawn-complete |
| `2026-03-01-probe-decidability-graph-coherence` | Mar 1 | Extends | `question` type is explicitly non-spawnable; subtype labels (factual/judgment/framing) are unread by daemon; daemon integration is acknowledged optional future work |
| `2026-03-01-probe-cross-repo-queue-poisoning-circuit-breaker` | Mar 1 | Extends | Queue poisoning (persistently-failing issues retrying every poll cycle) is distinct from capacity starvation; fixed with per-issue circuit breaker in `SpawnFailureTracker` |
| `2026-03-11-probe-completion-spawn-loop-label-asymmetry` | Mar 11 | Extends | Completion and spawn loops had asymmetric label awareness: completion loop filtered `daemon:ready-review` but spawn loop didn't, causing completed issues to re-enter spawn queue. Fixed with 3 layers: spawn queue label filter, triage label cleanup on completion, in-memory CompletionDedupTracker |
| `2026-03-12-probe-cross-project-orchestration-friction-audit` | Mar 12 | Contradicts + Extends | "Cannot COMPLETE cross-project" claim is stale — listCompletedAgentsMultiProject() implemented. Project group model (pkg/group/) fully implemented but groups.yaml doesn't exist — all group features fall back to hardcode. Creating config file is highest-leverage zero-code change. |
| `../harness-engineering/probes/2026-03-17-probe-pre-commit-accretion-gate-2-week-effectiveness.md` | Mar 17 | Extends | Extraction cascade pattern documented: gate signals → daemon detects hotspot → spawns extraction → hotspot count 12→3 (75% reduction). Gate's direct blocking negligible (2 blocks, both bypassed); value is in triggering extraction cascades via daemon. |
