# Probe: Cross-Project Orchestration Friction Audit

**Date:** 2026-03-12
**Model:** daemon-autonomous-operation
**Status:** Complete
**Related Models:** spawn-architecture, beads-integration-architecture

---

## Question

The daemon-autonomous-operation model documents cross-project operation with specific claims: (1) "polls all kb-registered projects via ProjectRegistry", (2) "can SPAWN cross-project but cannot COMPLETE cross-project", (3) "account routing gap: single globally-active account for all spawns", (4) "VerificationTracker seed reads orch-go checkpoints only." The spawn-architecture model claims: (5) "runKBContextQuery sets cmd.Dir=projectDir for both local and global search" (fixed Feb 25-27). The project-group-model decision (Feb 25) accepted the hybrid group design, but groups.yaml doesn't exist yet.

**Specific claims to test:**
- Is claim (2) still accurate? The explore agent found `listCompletedAgentsMultiProject()` exists.
- Is the project group model implemented but unwired (code exists, config missing)?
- What is the actual blast radius of `FindSocketPath("")` calls remaining after prior fixes?
- What is the minimum change set to make cross-project spawning frictionless?

## What I Tested

### Test 1: Completion Processing — Is It Still Single-Project?

**Code inspection:** `pkg/daemon/completion_processing.go`

The model claim says "can SPAWN cross-project but cannot COMPLETE cross-project." Tested by reading actual code paths.

**Observed:** Claim (2) is **OUTDATED/CONTRADICTED.** Multi-project completion IS implemented:

- `ListCompletedAgentsDefault()` (line 103-108) routes to `listCompletedAgentsMultiProject()` when `config.ProjectDirs` is non-empty
- `listCompletedAgentsMultiProject()` (lines 217-241) iterates all configured project dirs, calls single-project completion finder for each, tags results with `agent.ProjectDir`
- `ProcessCompletion()` (lines 361-542) correctly uses `agent.ProjectDir` for beads operations

**But:** The wiring depends on `config.ProjectDirs` being populated. This comes from the daemon loop's ProjectRegistry. If ProjectRegistry resolves all kb-registered projects (which it does), completion processing IS cross-project aware.

**Verdict:** The model's claim "cannot COMPLETE cross-project" was accurate when written but is now stale. Completion processing has been extended to support multi-project, though VerificationTracker seeding remains single-project.

### Test 2: groups.yaml Existence and Wiring

**Tested:**
```bash
ls ~/.orch/groups.yaml  # DOES NOT EXIST
ls ~/.kb/groups.yaml    # DOES NOT EXIST
```

**Code wiring audit:**

| Component | groups.yaml Used? | Fallback When Missing |
|-----------|-------------------|----------------------|
| `pkg/group/group.go` | Full implementation (Load, GroupsForProject, SiblingsOf, ResolveGroupMembers) | N/A |
| `pkg/spawn/kbcontext_filter.go:64` | `group.Load()` called in `resolveProjectAllowlistForDir()` | Falls back to `OrchEcosystemRepos` hardcode |
| Daemon polling | NOT wired | Polls ALL 19 kb-registered projects |
| Account routing | NOT wired | Single global account |
| Dashboard filtering | NOT wired | Shows all projects |

**Verdict:** The group package is fully implemented and tested, and correctly wired into kb context filtering. But the config file doesn't exist, so the fallback to `OrchEcosystemRepos` hardcode is always taken. Creating groups.yaml would immediately enable group-based kb context filtering with zero code changes.

### Test 3: FindSocketPath("") Blast Radius

**Tested:** grep for all `FindSocketPath("")` calls in Go source files.

**Observed:** 40+ calls across the codebase. Categorized by cross-project risk:

**Category A — Safe (single-project-scoped commands):**
These are in command handlers where the user's CWD determines the project. Cross-project callers use explicit `--workdir` which threads projectDir through separate code paths.
- `pkg/orch/spawn_beads.go:99` — spawn's beads setup (has separate `SetupBeadsTrackingForProject()`)
- `pkg/orch/spawn_preflight.go:122` — preflight checks
- `cmd/orch/reconcile.go:218, 237, 399` — reconcile operates on CWD project
- `cmd/orch/handoff.go:407, 441, 485` — handoff operates on CWD project
- `cmd/orch/focus.go:540` — focus operates on CWD project
- `cmd/orch/shared.go:268` — shared helper
- `cmd/orch/review_triage.go:147, 349` — triage operates on CWD project

**Category B — Problematic (reachable from cross-project paths):**
- `pkg/daemon/active_count.go:111` — `GetClosedIssuesBatch()` queries CWD beads for cross-project agent IDs → wrong database
- `pkg/daemon/active_count.go:258` — `BeadsActiveCount()` scoped to CWD only → undercounts cross-project agents
- `pkg/daemon/cleanup.go:173` — cleanup scoped to CWD → misses cross-project stale agents
- `pkg/daemon/artifact_sync_default.go:81` — artifact sync scoped to CWD
- `pkg/discovery/discovery.go:150, 240` — discovery queries CWD beads only → invisible cross-project agents
- `pkg/daemon/issue_adapter.go:34, 95, 125, 204, 284, 316, 530` — these are the **single-project variants** (e.g., `ListReadyIssues` vs `ListReadyIssuesForProject`). The daemon uses the multi-project variants. But other callers (CLI commands) use these and would query wrong beads for cross-project work.

**Category C — Fixed (have cross-project variants):**
- `pkg/daemon/issue_adapter.go` — ListReadyIssuesForProject, UpdateBeadsStatusForProject, etc. exist
- `pkg/spawn/kbcontext.go` — RunKBContextCheckForDir threads projectDir
- `pkg/daemon/completion_processing.go` — listCompletedAgentsMultiProject exists

**Blast radius summary:** 7 calls in Category B (daemon support code), 7+ in Category A single-project wrappers that CLI commands use without `--workdir`.

### Test 4: Daemon Account Routing

**Code inspection:** Searched daemon code for account switching.

**Observed:** No account routing exists in the daemon. The daemon spawns with whatever account is globally active (`orch account switch <name>` sets this). The `groups.yaml` config has an `account` field per group, and the group package reads it, but no daemon code consumes it.

**Specific gap:** `spawnIssue()` in `pkg/daemon/spawn_execution.go` calls `spawner.SpawnWork(issue.ID, inferredModel, workdir)` — no account parameter. The `orch work` command doesn't accept an `--account` flag either.

### Test 5: --no-track as the Cross-Project Workaround

**Prior decision:** "Cross-project epics use Option A: ad-hoc spawns with --no-track in secondary repos, manual bd close with commit refs."

**Observed operational cost:**
- --no-track generates synthetic beads IDs (`{project}-untracked-{timestamp}`)
- 5 `isUntrackedBeadsID()` guards across codebase to prevent crashes
- Invisible to: `orch status`, `orch clean --orphans`, daemon active count
- Since Claude CLI became default backend (Feb 19), --no-track agents are invisible in BOTH lanes
- The spawn-architecture model explicitly recommends: "Replace --no-track with lightweight tracking"

**Current friction path for cross-project ad-hoc work:**
1. `orch spawn --bypass-triage --no-track --workdir ~/other-project investigation "task"` — 4 flags
2. Agent completes — no visibility in any dashboard
3. Manual `cd ~/other-project && bd close <id>` — or agent reports Phase: Complete to synthetic ID (noop)
4. No completion verification possible

## What I Observed (Summary)

### Gap Map

| Gap | Severity | Fix Effort | Blocks |
|-----|----------|------------|--------|
| groups.yaml doesn't exist | Critical | Zero code (create file) | KB context sibling resolution, daemon scoping, account routing |
| No daemon account routing | High | ~100 LOC | SCS work uses wrong account |
| No daemon --group flag | Medium | ~50 LOC | Daemon polls 19 projects including irrelevant ones |
| FindSocketPath("") in active_count.go | Medium | ~30 LOC | Incorrect concurrency counts for cross-project |
| FindSocketPath("") in cleanup/discovery | Low | ~40 LOC | Cross-project ghost agents not cleaned |
| --no-track as cross-project pattern | High | ~200 LOC | Invisible agents, no lifecycle management |
| VerificationTracker single-project seed | Low | ~30 LOC | Threshold miscounted after restart |

### The "80% Ready" Insight

The codebase is architecturally ready for cross-project:
- `pkg/group/` — fully implemented, tested
- `pkg/identity/` — ProjectRegistry resolves prefix → directory
- Spawn path — `projectDir` threaded through entire pipeline
- Completion — multi-project completion implemented
- KB context — group-based filtering implemented (code level)

What's missing is the **last-mile wiring:**
1. The config file (`groups.yaml`)
2. The daemon consuming groups (account routing, --group flag)
3. Replacing --no-track with lightweight tracking

## Model Impact

### CONTRADICTS (1 claim):
- **"can SPAWN cross-project but cannot COMPLETE cross-project"** — Multi-project completion IS implemented via `listCompletedAgentsMultiProject()`. The completion gap documented in the model has been fixed. The model should update this to: "Cross-project completion is implemented but VerificationTracker seeding is single-project."

### CONFIRMS (3 claims):
- **"account routing gap: single globally-active account"** — Confirmed. No account routing code exists in daemon.
- **"VerificationTracker seed reads orch-go checkpoints only"** — Confirmed. `SeedFromBacklog()` only reads local checkpoint.
- **"polls all kb-registered projects via ProjectRegistry"** — Confirmed. ProjectRegistry queries `kb projects list --json`.

### EXTENDS (2 new claims):
- **New claim:** "The project group model (Decision Feb 25) is fully implemented in code but zero-wired in production — groups.yaml doesn't exist, so all group-based features fall back to hardcoded behavior." Creating the config file is the single highest-leverage change for cross-project friction.
- **New claim:** "Cross-project ad-hoc work relies on --no-track (4-flag ceremony, invisible agents), but the daemon/completion infrastructure is mature enough to support lightweight tracking. The operational cost of --no-track exceeds the implementation cost of replacing it."
