# Event Tracking

**Purpose:** Reference for all event types logged to `~/.orch/events.jsonl` for stats aggregation.

**Last verified:** 2026-03-27

---

## Event Types

| Event | Source | Purpose |
| --- | --- | --- |
| `session.spawned` | `orch spawn` | Agent created |
| `session.started` | `orch session start` | Orchestrator session started |
| `session.ended` | `orch session end` | Orchestrator session ended |
| `session.completed` | completion pipeline | Session finished |
| `session.auto_completed` | daemon | Daemon auto-completed session |
| `session.error` | error handler | Session error |
| `session.status` | status poller | Status change (busy/idle) |
| `session.labeled` | `orch session label` | Session labeled |
| `session.send` | `orch send` | Message sent to existing session |
| `agent.completed` | `orch complete` or `bd close` hook | Agent finished work |
| `agent.abandoned` | `orch abandon` | Agent abandoned |
| `agent.abandoned.telemetry` | `orch abandon` | Enriched abandonment data (skill, tokens, duration) |
| `agent.rejected` | `orch reject` | Agent work rejected, issue reopened |
| `agent.reworked` | `orch rework` | Rework spawned |
| `agent.resumed` | `orch resume` | Agent resumed |
| `agent.recovered` | daemon recovery | Stuck agent recovered |
| `agent.crashed_with_artifacts` | GC | Orphaned agent with committed work |
| `agent.force_completed` | GC | GC-initiated completion |
| `agent.force_abandoned` | GC | GC-initiated abandonment |
| `agent.wait.complete` | `orch wait` | Wait completed (phase reached) |
| `agent.wait.timeout` | `orch wait` | Wait timed out |
| `verification.failed` | verify pipeline | Verification gate failed |
| `verification.bypassed` | `--skip-*` flags | Verification gate bypassed |
| `verification.auto_skipped` | skill exemption | Verification auto-skipped |
| `spawn.gate_decision` | spawn gates | Gate evaluation result |
| `spawn.hotspot_bypassed` | spawn pipeline | Hotspot gate bypassed (legacy, gates now advisory) |
| `spawn.triage_bypassed` | `--bypass-triage` | Triage gate bypassed |
| `spawn.verification_bypassed` | spawn pipeline | Verification gate bypassed during spawn |
| `spawn.infrastructure_detected` | spawn pipeline | Infrastructure work detected |
| `spawn.skill_inferred` | `orch work` | Skill inferred for issue |
| `daemon.spawn` | daemon | Daemon spawn decision |
| `daemon.once` | `daemon once` | Single OODA cycle executed |
| `daemon.complete` | daemon | Daemon auto-completion |
| `daemon.completion_error` | daemon | Daemon completion error |
| `daemon.architect_escalation` | daemon | Hotspot routing to architect |
| `daemon.cleanup` | daemon periodic | Agent cleanup (completed/abandoned) |
| `daemon.recovery` | daemon periodic | Stuck agent recovery |
| `daemon.orphan_detection` | daemon periodic | Orphaned agent detection |
| `daemon.phase_timeout` | daemon periodic | Phase timeout detection and escalation |
| `daemon.question_detection` | daemon periodic | Question detection polling errors |
| `daemon.question_detected` | daemon periodic | Question actually found in agent output |
| `daemon.agreement_check` | daemon periodic | Cross-validation of daemon decisions |
| `daemon.beads_health` | daemon periodic | Beads health monitoring with circuit breaker |
| `daemon.artifact_sync` | daemon periodic | Documentation drift detection |
| `daemon.registry_refresh` | daemon periodic | Agent registry refresh |
| `daemon.verification_failed_escalation` | daemon periodic | Failed verification retry |
| `daemon.lightweight_cleanup` | daemon periodic | Lightweight workspace cleanup |
| `daemon.audit_select` | daemon periodic | Random quality audit selection |
| `decision.made` | daemon | Decision with classification tier |
| `accretion.snapshot` | periodic | Directory-level line counts |
| `duplication.detected` | dupdetect | Similar function pairs found |
| `duplication.suppressed` | dupdetect | Allowlist-suppressed pairs (precision tracking) |
| `service.started` | service monitor | Service first started |
| `service.crashed` | service monitor | Service PID changed |
| `service.restarted` | service monitor | Service auto-restarted |
| `exploration.decomposed` | exploration | Question decomposed into subproblems |
| `exploration.judged` | exploration | Judge verdicts on findings |
| `exploration.synthesized` | exploration | Final synthesis produced |
| `exploration.iterated` | exploration | Judge-triggered re-exploration round |
| `focus.set` | `orch focus` | North star goal set |
| `focus.cleared` | `orch focus clear` | North star goal cleared |
| `handoff.created` | `orch handoff` | Session handoff created |
| `debrief.quality` | `orch debrief` | Debrief quality measurement |
| `swarm.start` | `orch swarm` | Swarm session started |
| `swarm.spawn` | `orch swarm` | Swarm agent spawned |
| `swarm.agent.complete` | `orch swarm` | Individual swarm agent completed |
| `swarm.complete` | `orch swarm` | Swarm session completed |
| `swarm.detach` | `orch swarm` | Swarm agent detached |
| `agents.cleaned` | `orch clean` | Completed agents cleaned from registry |
| `account.auto_switched` | spawn pipeline | Account auto-switched on rate limit |
| `review_tier.escalated` | review | Review tier auto-escalated |
| `trigger.outcome` | daemon | Per-detector false positive tracking (issue closed without action) |
| `command.invoked` | measurement commands | Tracks which diagnostic commands are used and by whom (human/orchestrator/worker) |
| `artifact.drift` | artifact sync | Documentation drift detected (files changed since last sync) |
| `gap.gate.bypassed` | spawn pipeline | KB gap gate bypassed via `--skip-gap-gate` |
| `audit.passed` | daemon audit | Daemon quality audit passed |
| `audit.failed` | daemon audit | Daemon quality audit failed/rejected |
| `spawn.bypass` | `orch spawn` | Direct (non-daemon) spawn tracked for human vs daemon rate |
| `spawn.routing_impact` | spawn pipeline | Model routing decision impact tracking |
| `bench.complete` | `orch bench` | Benchmark suite completed (name, passed, failed, pass_rate, verdict) |
| `loop.iteration` | loop controller | Individual iteration of a `--loop` spawn cycle |
| `loop.complete` | loop controller | Final outcome of a `--loop` spawn cycle |
| `session.empty_execution_retry` | review/verify | Session retried after empty-execution classification |
| `spawn.model_route` | daemon | Daemon model routing decision for a spawn (skill→model mapping) |
| `daemon.thin_issue_detected` | daemon orient | Issue flagged as thin (insufficient context for spawning) |
| `kb.context.timeout` | spawn pipeline | KB context gathering timed out during spawn |

**Enrichment fields:** `verification.failed`, `agent.completed`, `verification.bypassed`, and `verification.auto_skipped` events include a `verification_level` field (V0-V3) tracking what "verified" means at completion.

## Beads Close Hook

When issues are closed directly via `bd close` (bypassing `orch complete`), the beads hook at `.beads/hooks/on_close` emits an `agent.completed` event. This closes the tracking gap.

**Hook location:** `.beads/hooks/on_close` (project-specific)

**Manual event emission:**

```bash
# Emit completion event directly (used by hooks)
orch emit agent.completed --beads-id proj-123 --reason "Closed via bd close"
```

**To enable in a project:**

1. Create `.beads/hooks/on_close` (executable)
2. Copy content from orch-go's hook as a template
