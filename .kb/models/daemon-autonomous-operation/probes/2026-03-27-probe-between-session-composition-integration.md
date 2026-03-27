# Probe: Between-Session Composition Integration Points

**Date:** 2026-03-27
**Status:** Complete
**Model:** daemon-autonomous-operation
**Issue:** orch-go-betfg

## Question

Does the daemon-autonomous-operation model's architecture support periodic composition as a new periodic task? Specifically: can composition integrate through the existing `PeriodicScheduler` pattern without violating the model's OODA structure or its "no local agent state" constraint?

## What I Tested

### 1. Periodic Task Registration Pattern

Read `pkg/daemon/scheduler.go` and `pkg/daemonconfig/config.go` to verify the extension pattern for adding new periodic tasks.

**Observed:** The pattern is mechanical:
1. Add `TaskCompose = "compose"` constant in `scheduler.go` (line 6-20, alongside 13 existing tasks)
2. Add `ComposeEnabled bool`, `ComposeInterval time.Duration`, `ComposeThreshold int` to `Config` struct in `daemonconfig/config.go`
3. Add defaults in `DefaultConfig()`
4. Register in `NewSchedulerFromConfig()` (line 112-127)
5. Implement `RunPeriodicCompose()` on `*Daemon`
6. Wire into `runPeriodicTasks()` in `cmd/orch/daemon_periodic.go`

**Pattern confirmed for:** 13 existing tasks follow this exact pattern. No special wiring needed.

### 2. OODA Loop Independence

Read `pkg/daemon/ooda.go` to verify that periodic tasks and the OODA spawn loop don't conflict.

**Observed:** The OODA loop (Sense → Orient → Decide → Act) is invoked via `OnceExcluding()` in the daemon main loop (`cmd/orch/daemon.go:63-121`). Periodic tasks run in `runPeriodicTasks()` which executes BEFORE the OODA loop each cycle, within `BeginCycle()/EndCycle()` boundaries. Composition would run in the periodic block, completely independent of spawn logic.

**No conflict:** Composition reads `.kb/briefs/` and writes `.kb/digests/`. It doesn't touch beads, OpenCode sessions, or the spawn pipeline. The OODA loop reads from beads. They're fully independent.

### 3. No Local Agent State Constraint

Checked whether daemon periodic compose would violate the "no local agent state" constraint from CLAUDE.md.

**Observed:** Composition reads from `.kb/briefs/` (file system artifacts, not agent state) and writes to `.kb/digests/` (knowledge artifacts). The daemon only needs to track `lastComposeTime` via the existing `PeriodicScheduler` (which already tracks last-run times for all tasks). The brief count check reads file system timestamps, not a registry or projection.

**Constraint satisfied:** No local agent state needed. Brief count is derived from file timestamps, not maintained in memory.

### 4. Comprehension Queue Interaction

Read `pkg/daemon/comprehension_queue.go` to verify that skipping `AddComprehensionUnread()` for maintenance items is safe.

**Observed:** `AddComprehensionUnread(beadsID)` is called from the completion pipeline. The comprehension queue has exactly two consumers:
1. `CheckComprehensionThrottle()` — gates spawning (daemon.go)
2. `DrainComprehensionUnread()` — manual reset (daemon resume)

Neither consumer iterates briefs or depends on 1:1 correspondence between completions and comprehension labels. Skipping the label for maintenance items means fewer items in the count, which relaxes the throttle — the intended behavior.

**Safe to skip:** No downstream consumer assumes every completion has a comprehension label.

### 5. Brief Frontmatter Extension

Read `cmd/orch/complete_brief.go` to verify that adding `category:` to brief YAML frontmatter is compatible with existing consumers.

**Observed:** Brief frontmatter currently has: `beads_id`, `quality_signals` (nested), `signal_count`, `signal_total`. Consumers:
- `ParseBriefSignals()` in `comprehension_queue.go` — parses `quality_signals:` block, ignores unknown fields
- `compose.ParseBrief()` in `pkg/compose/brief.go` — extracts `# Brief:` header and `## Frame/Resolution/Tension` sections, ignores frontmatter entirely
- `orient.ScanRecentBriefs()` — reads file, extracts first sentence, checks tension presence

**No consumer will break:** Adding `category: maintenance` to frontmatter is invisible to all existing consumers. They either parse specific known fields or ignore frontmatter entirely.

## What I Observed

The daemon-autonomous-operation model's architecture is well-suited for composition integration:

1. **PeriodicScheduler** is the correct extension point — composition follows the exact same pattern as the 13 existing tasks
2. **OODA independence** — composition runs in the periodic block, never touches the spawn pipeline
3. **No state management** — brief count derived from file timestamps, digest output is a file artifact
4. **Comprehension queue** safely supports conditional labeling — no consumer assumes 1:1 completion:label
5. **Brief frontmatter** is safely extensible — no existing parser will break on new `category:` field

## Model Impact

**Confirms:**
- The model's claim that the daemon architecture supports new periodic tasks through the PeriodicScheduler pattern
- The claim that OODA phases and periodic tasks are independent execution paths
- The "no local agent state" constraint is compatible with file-system-based composition

**Extends:**
- The model should note that the comprehension queue supports conditional labeling (not all completions must enter the queue). This is a new capability: the queue can be selectively populated based on work classification, enabling different treatment for maintenance vs knowledge-producing work.
- The periodic task system has a natural capacity for "method-core" behaviors (per the daemon behaviors investigation) — composition is the first method-core periodic task, joining 13 substrate/bridge tasks.

**No contradictions found.**
