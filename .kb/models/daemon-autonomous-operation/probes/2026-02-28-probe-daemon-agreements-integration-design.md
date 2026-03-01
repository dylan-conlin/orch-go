# Probe: Daemon-Agreements Integration Design

**Model:** Daemon Autonomous Operation
**Status:** Complete
**Date:** 2026-02-28

## Question

Can the daemon's periodic task pattern accommodate agreement checking with auto-issue creation, and does this extend the model's claims about the poll-spawn-complete cycle?

## What I Tested

### 1. Periodic task pattern fit

Examined the existing periodic task pattern (knowledge_health, orphan_detection, phase_timeout, question_detection) to verify the daemon can support agreement checking without violating architectural constraints.

**Files examined:**
- `pkg/daemonconfig/config.go` - Config struct pattern (135 lines, Config with Enable/Interval per task)
- `pkg/daemon/daemon.go` - State tracking pattern (878 lines, `lastXxx time.Time` fields)
- `pkg/daemon/knowledge_health.go` - Issue creation pattern (175 lines, dedup via `bd list`)
- `cmd/orch/daemon_periodic.go` - Handler pattern (251 lines, `runPeriodicTasks()`)
- `pkg/spawn/gates/agreements.go` - Existing agreement integration (115 lines, spawn-time warning)

### 2. Dedup pattern analysis

Knowledge health uses label-based dedup: `bd list --status=open -l area:knowledge` + title matching.
Spawn tracker uses in-memory TTL-based dedup for race condition prevention.
Agreement dedup can use label per agreement ID: `agreement:<id>`.

### 3. Agreement YAML schema review

Agreement YAMLs have: `id`, `title`, `description`, `severity`, `failure_mode`, `contract`, `check`, `parties`.
No existing `auto_fix` field — this is a schema extension opportunity.
The `contract` field is plain-language description of what should be true — critical for fix agent context.

### 4. Daemon constraint check

Constraint: `cmd/orch/daemon.go runDaemonLoop must be extracted before adding new daemon subsystems`
- `daemon.go` at 1174 lines (79% of 1500 threshold)
- Adding agreement checking goes to `daemon_periodic.go` (already extracted) and `pkg/daemon/agreement_check.go` (new file)
- Only adds ~3 lines to `daemon.go` struct + snapshot, not to `runDaemonLoop`
- The periodic task orchestrator is already extracted — this adds to it, not to the main loop

**Assessment:** This doesn't violate the extraction constraint because it adds to the already-extracted periodic task system, not to runDaemonLoop itself. However, `pkg/daemon/daemon.go` (878 lines) will grow by ~20 lines for state fields.

## What I Observed

### Periodic task integration is clean and predictable

Every periodic task follows the same 6-part pattern:
1. Config fields (daemonconfig)
2. State tracking (Daemon struct)
3. ShouldRun method
4. RunPeriodic method
5. Handler in daemon_periodic.go
6. Optional snapshot for status file

### Agreement checker already exists for spawn gates

`pkg/spawn/gates/agreements.go` defines `AgreementsChecker func(projectDir string) (*AgreementsResult, error)` — this exact function signature can be reused by the daemon. The `buildAgreementsChecker()` in `cmd/orch/kb.go` creates this function by shelling out to `kb agreements check --json`.

### Knowledge health is the closest pattern for auto-issue creation

`DefaultCreateKnowledgeHealthIssue` shows the dedup-then-create pattern:
1. `bd list --status=open -l area:knowledge` — check for existing
2. Title matching — prevent near-duplicates
3. `bd create --title ... --type task -l triage:review -l area:knowledge` — create with labels

### Cross-project agreements need special handling

Agreement YAMLs can reference other projects in `parties.source.project`, but `kb agreements check` runs in the current project directory. The check verifies the contract locally — if the fix is in another repo, the spawned agent needs to discover and escalate.

## Model Impact

### Extends the model

The daemon model describes a poll-spawn-complete cycle for `triage:ready` issues. Agreement integration extends this with a **detect-create-spawn-verify** sub-cycle that feeds into the existing cycle:

```
Agreement fails → Issue created (triage:ready) → Daemon spawns fix → Agent completes →
Next agreement check cycle verifies fix
```

This is a **signal-to-action loop** layered on top of the existing poll-spawn-complete cycle. The daemon doesn't just process work — it detects work that needs to be done.

### Confirms model claims

- **Periodic task pattern**: Fully extensible, well-factored. Adding a new periodic task is ~100-150 lines of new code across 4 files.
- **Issue creation dedup**: Label-based dedup (`bd list --status=open -l agreement:<id>`) follows the knowledge health pattern.
- **Skill inference**: Auto-created issues use type `task` → daemon infers `investigation` skill. This may not be optimal for simple documentation fixes, but works as a starting point.

### New model insight

Agreement checking introduces a **self-healing property** to the daemon: the system detects its own contract violations and creates work to fix them. This is distinct from the existing model where all work enters via human triage. The daemon becomes both executor and inspector.
