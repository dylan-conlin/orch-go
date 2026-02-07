# Decision: Daemon Resilience - Retry Limits, Escalation, Staging, and Model Routing

**Date:** 2026-02-06
**Status:** Decided
**Context:** orch-go-21353 (zombie respawn loop), orch-go-21354 (race condition staging), orch-go-21358 (architect task)
**Authority:** Architectural - affects daemon behavior, spawn workflow, orchestrator UX, and multi-model routing

---

## Background

Two daemon behaviors created friction in multi-agent workflows:

1. **Zombie respawn loop:** Dead sessions get reset to open, daemon respawns, agent dies again. No retry limit, no escalation, no context preservation.
2. **Race condition staging:** Daemon grabs `triage:ready` issues within seconds of labeling. If orchestrator needs to adjust (model selection, spawn context), must abandon + respawn.

## Key Finding: Most Infrastructure Already Exists

Investigation of the current codebase revealed that **retry tracking, escalation, grace period (staging), and context preservation are already implemented**. The spawn context referenced three issues dying twice each, but the fixes for those exact problems have since landed. This decision documents the current state, validates the design, identifies remaining gaps, and establishes the model routing extension.

---

## Decision 1: Retry Tracking via Beads Comments (ALREADY IMPLEMENTED)

**Where retry count is stored:** In beads comments on the issue itself. Each time the daemon detects a dead session, it adds a `DEAD SESSION:` prefixed comment. The retry count is derived by counting these comments via `CountDeadSessionComments()`.

**Why comments, not labels or external state:**

- Survives daemon restarts (comments are persistent in beads)
- Self-documenting audit trail (each comment includes reason, last phase, timestamp)
- No new state to manage (no database, no config file)
- Queryable via `bd comments <id>`

**Source:** `pkg/daemon/dead_session_detection.go:107-122`

**Implementation details:**

- `FindDeadSessions()` finds in_progress issues with no active session and no Phase: Complete
- `CountDeadSessionComments()` counts `DEAD SESSION:` prefix comments
- `MarkSessionAsDead()` adds comment with context and resets status to open
- `GetLastPhaseComment()` extracts the dying agent's last phase for context preservation

**Validated:** This design is correct. Comments as retry tracking avoids external state and creates a permanent audit trail.

---

## Decision 2: Escalation Protocol (ALREADY IMPLEMENTED)

**Escalation threshold:** `MaxDeadSessionRetries` (default: 2). After an issue has died this many times, the daemon escalates instead of respawning.

**Escalation mechanism:**

1. `EscalateDeadSession()` adds a `DEAD SESSION ESCALATED:` comment
2. Adds `needs:human` label to prevent daemon respawning
3. Resets status to `open` (visible in dashboard, but daemon skips it due to label)
4. Comment includes retry count, last phase, and manual retry instructions

**Why `needs:human` label, not a special status:**

- Daemon's label filter (`triage:ready`) naturally skips issues with `needs:human`
- Visible in dashboard as requiring attention
- Orchestrator can clear label + re-add `triage:ready` to retry
- No new status values needed in beads

**Source:** `pkg/daemon/dead_session_detection.go:204-232`

**Detection interval:** Every 10 minutes (`DeadSessionDetectionInterval`), configurable.

**Validated:** This design is correct. The escalation pathway clearly separates "retryable" from "needs attention" without new infrastructure.

---

## Decision 3: Context Preservation Across Retries (ALREADY IMPLEMENTED)

**How prior attempt context is preserved:**

1. `GetLastPhaseComment()` extracts the dying agent's last reported phase
2. This is included in the `DEAD SESSION:` comment on the issue
3. When the daemon respawns the issue, `orch work` fetches issue comments
4. Spawn context generation (`fetchIssueCommentsForSpawn()`) includes non-Phase comments as "ORCHESTRATOR NOTES"
5. The respawned agent sees the dead session comment with last phase info

**Gap identified:** The `POST-COMPLETION-FAILURE` comments (from rework pattern) need more prominent surfacing. This is tracked in orch-go-21240.

**Source:** `pkg/daemon/dead_session_detection.go:127-140`, `cmd/orch/spawn_validation.go:330-361`

**Validated:** The basic mechanism works. Prior investigation (2026-02-04-inv-address-daemon-interaction-reopened-rework.md) recommends enhancing spawn context to surface failure context more prominently - that's a separate enhancement, not a design change.

---

## Decision 4: Staging Mechanism - Grace Period (ALREADY IMPLEMENTED)

**Chosen mechanism:** Grace period. When daemon first sees a `triage:ready` issue, it waits `GracePeriod` (default: 30 seconds) before spawning.

**Why grace period over alternatives:**

| Option                                              | Pros                                   | Cons                                                         | Verdict                         |
| --------------------------------------------------- | -------------------------------------- | ------------------------------------------------------------ | ------------------------------- |
| **Grace period** (chosen)                           | Simple, no label changes, configurable | Fixed delay even when not needed                             | Best balance                    |
| Two-step label (`triage:staging` -> `triage:ready`) | Explicit control                       | Extra label management, orchestrator must remember two steps | Over-engineered                 |
| Hold flag (`bd create --hold`)                      | Per-issue control                      | New flag, daemon must check field                            | Adds complexity                 |
| Model hint label                                    | Solves model routing too               | More complex daemon logic                                    | Handled separately (Decision 5) |

**Implementation:**

- `firstSeen` map tracks when each issue was first observed
- `InGracePeriod()` returns true if issue is within grace window
- `CleanFirstSeen()` prevents unbounded memory growth
- Configurable: `GracePeriod: 30 * time.Second` in `DefaultConfig()`

**Source:** `pkg/daemon/daemon.go:183-186, 808-843`

**Validated:** 30s grace period is sufficient for typical orchestrator corrections (relabeling, model selection). If orchestrator needs more time, they can remove `triage:ready` temporarily.

---

## Decision 5: Model Routing via Labels (NEW - TO IMPLEMENT)

**Problem:** Daemon currently spawns all agents with the default model. When orchestrator wants a specific model (e.g., for GPT integration work), they must spawn manually with `--model` flag, bypassing the daemon.

**Design:** Add `model:*` label support, mirroring the existing `skill:*` and `needs:*` label patterns.

**How it works:**

1. Orchestrator labels issue: `bd label <id> model:sonnet` (or `model:opus`, `model:flash`, etc.)
2. `orch work` reads the label via `InferModelFromLabels()` (new function, same pattern as `InferSkillFromLabels`)
3. Passes model to `runSpawnWithSkillInternal()` via `spawnModel` flag
4. If no `model:*` label, uses default model (current behavior)

**Supported model labels:**

| Label          | Maps To | Backend                           |
| -------------- | ------- | --------------------------------- |
| `model:opus`   | opus    | claude (implies --backend claude) |
| `model:sonnet` | sonnet  | opencode (default)                |
| `model:haiku`  | haiku   | opencode                          |
| `model:flash`  | flash   | opencode                          |
| `model:pro`    | pro     | opencode                          |

**Implementation location:** `pkg/daemon/skill_inference.go` (add `InferModelFromLabels`), `cmd/orch/work_cmd.go` (read and pass model)

**Why labels, not a field:**

- Consistent with existing `skill:*` and `needs:*` patterns
- Labels are visible in dashboard, searchable, filterable
- No beads schema changes needed
- Daemon already reads labels for skill inference

**Backend implications:** `model:opus` requires Claude CLI backend (Opus blocked via API). The `orch work` command should detect this and set `--backend claude` automatically, similar to how `--opus` flag works in spawn.

**Interaction with GPT integration (orch-go-21350):** When GPT models are integrated, this same pattern extends naturally: `model:gpt-5.3-codex` label routes to GPT backend. The label abstraction decouples model selection from backend mechanics.

**Trade-offs accepted:**

- One more label convention to remember (mitigated: follows established pattern)
- Opus routing forces Claude backend (acceptable: `--opus` flag already does this)

---

## Decision 6: Failure Pattern Detection (NEW - DEFERRED)

**The question:** Should daemon track failure reasons to detect patterns (all dying at same phase -> systemic issue)?

**Decision:** Defer. The current system provides adequate signals:

1. `DEAD SESSION:` comments include last phase - orchestrator can spot patterns
2. `needs:human` escalation surfaces recurring failures
3. Dashboard shows issue history and comments
4. Event logging (`events.jsonl`) captures recovery attempts

**When to revisit:** If we see 5+ issues dying at the same phase in a session, indicating a systemic issue (e.g., broken tool, model degradation, infrastructure problem). At that point, add daemon-level pattern detection that:

- Aggregates failure phases across recent dead sessions
- Alerts when N issues fail at the same phase within a time window
- Suggests systemic investigation

**Why defer:** Current escalation handles individual issue failures well. Pattern detection across issues is a different problem class (monitoring/alerting) that requires more infrastructure than current friction justifies.

---

## Summary of Decisions

| Area                 | Status                  | Key Design                                  |
| -------------------- | ----------------------- | ------------------------------------------- |
| Retry tracking       | **Already implemented** | Beads comments as source of truth           |
| Escalation threshold | **Already implemented** | 2 failures -> `needs:human` label           |
| Context preservation | **Already implemented** | Last phase comment in DEAD SESSION comments |
| Staging mechanism    | **Already implemented** | 30s grace period                            |
| Model routing        | **New - to implement**  | `model:*` labels read by `orch work`        |
| Pattern detection    | **Deferred**            | Current escalation sufficient for now       |

---

## Implementation Plan for Model Routing

**Priority:** Medium - enables multi-model workflow without manual spawning

**Steps:**

1. Add `InferModelFromLabels()` to `pkg/daemon/skill_inference.go`
2. Add `inferModelFromBeadsIssue()` to `cmd/orch/work_cmd.go` (parallel to `inferSkillFromBeadsIssue`)
3. Wire model into `runSpawnWithSkillInternal()` call
4. Handle `model:opus` -> Claude backend auto-selection
5. Add tests for label parsing and backend routing
6. Update daemon model/guide with model routing

**Estimated effort:** 1-2 hours implementation + tests

---

## References

**Source code examined:**

- `pkg/daemon/daemon.go` - Main daemon with Config, grace period, dead session detection
- `pkg/daemon/dead_session_detection.go` - Retry tracking, escalation, context preservation
- `pkg/daemon/pool.go` - Worker pool and reconciliation
- `pkg/daemon/skill_inference.go` - Skill/MCP inference from labels
- `pkg/daemon/recovery.go` - Stuck agent recovery, server recovery
- `pkg/daemon/issue_adapter.go` - SpawnWork, SpawnWorkForProject
- `cmd/orch/work_cmd.go` - orch work command, skill/MCP inference
- `cmd/orch/spawn_validation.go` - Comment fetching for spawn context

**Related investigations:**

- `.kb/investigations/2026-02-04-inv-address-daemon-interaction-reopened-rework.md` - Daemon/rework interaction
- `.kb/models/daemon-autonomous-operation.md` - Daemon architecture model
- `.kb/models/spawn-architecture.md` - Spawn architecture model

**Related issues:**

- orch-go-21353 - Zombie respawn loop (addressed by existing dead session detection)
- orch-go-21354 - Race condition staging (addressed by existing grace period)
- orch-go-21240 - Spawn context POST-COMPLETION-FAILURE surfacing (enhancement)
- orch-go-21350 - GPT integration (model routing enables this)
