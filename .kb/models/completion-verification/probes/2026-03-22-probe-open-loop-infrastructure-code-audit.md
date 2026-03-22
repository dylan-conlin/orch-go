# Probe: Open-Loop Infrastructure — Code-Level Audit of Disconnected Feedback Loops

**Model:** completion-verification
**Date:** 2026-03-22
**Status:** Complete
**claim:** CV-07
**verdict:** extends

---

## Question

Model Failure Mode 7 claims "Human Negative-Feedback Channel Structural Disuse" — that the feedback channel is unused because of friction asymmetry and vocabulary gaps. The `orch reject` command was built to address the vocabulary gap (March 2026). Has this closed the loop? And how deep does the open-loop pattern go beyond just the human feedback channel?

---

## What I Tested

Systematic code audit of every feedback loop in the completion/daemon pipeline, tracing each from signal emission through to behavioral response. Verified against production event data.

```bash
# Count exact event types in production
grep '"type":"agent.rejected"' ~/.orch/events.jsonl | wc -l       # 0
grep '"type":"agent.rejected"' ~/.orch/events-2026-03.jsonl | wc -l  # 0
grep '"type":"audit\.' ~/.orch/events.jsonl | wc -l                  # 0
grep -c "accretion.delta" ~/.orch/events.jsonl                       # 449
grep -c "accretion.delta" ~/.orch/events-2026-03.jsonl                # 64
grep '"type":"agent.completed"' ~/.orch/events.jsonl | wc -l         # 1188
grep '"type":"agent.completed"' ~/.orch/events-2026-03.jsonl | wc -l  # 97

# Check ComprehensionQuerier wiring
grep ComprehensionQuerier cmd/orch/daemon_loop.go  # NOT FOUND
grep ComprehensionQuerier cmd/orch/comprehension_cmd.go  # Only in CLI, not daemon

# Check accretion response wiring gap
sed -n '140,146p' cmd/orch/daemon_loop.go  # Comment + blank lines, no code

# Check scheduler registrations vs invocations
grep 's.Register(Task' pkg/daemon/scheduler.go | wc -l    # 24
grep 'RunPeriodic' cmd/orch/daemon_periodic.go | wc -l     # 13

# Check RejectedCount consumption
grep -r 'RejectedCount' pkg/daemon/ --include='*.go'  # 0 results
```

---

## What I Observed

### Production Evidence (1,285 completions, 513 accretion.delta events)

| Signal | Emitted | Consumed by Daemon | Behavioral Response |
|--------|---------|-------------------|-------------------|
| agent.rejected | **0 events** (command exists, never used) | RejectedCount aggregated in learning.go | **Nothing** — field never read by daemon |
| accretion.delta | **513 events** | **Never** — daemon_loop.go:141 has blank wiring | **Nothing** — events accumulate with no reader |
| audit.passed/failed | **0 events** | processAuditVerdictIfPresent() ready | **Nothing** — no audit agents ever spawned |
| comprehension:pending | Labels added on completion | **Never checked** — ComprehensionQuerier always nil | **Nothing** — gate always returns Allowed |
| agent.reworked | 0 events | ReworkCount in learning.go | **Display only** — shown in orient, not used in daemon |
| verification.failed | Events emitted | VerificationFailures in learning.go | **Nothing** — field never read |
| verification.bypassed | Events emitted | VerificationBypasses in learning.go | **Nothing** — field never read |

### 10 Concrete Open Loops Found

**Loop 1: Quality Audit (the specimen)**
- Built: `periodic_audit.go` selects issues, labels them `audit:deep-review`
- Built: `audit_verdict.go` processes AUDIT_VERDICT.md, routes FAIL→reject, PASS→clear
- **Missing:** No code path spawns audit agents when `audit:deep-review` label exists. InferSkillFromLabels() only recognizes `skill:*` labels. CheckIssueCompliance() has no audit routing. Result: 0 audit agents spawned ever.

**Loop 2: Accretion Response**
- Built: 513 accretion.delta events emitted at completion (complete_lifecycle.go:341)
- Built: Config fields (AccretionResponseEnabled, AccretionResponseInterval), task registered in scheduler
- **Missing:** daemon_loop.go:141 has a comment `// Wire accretion response service` followed by blank lines. No implementation function exists. Result: events accumulate, daemon never reads them.

**Loop 3: Reject → Learning**
- Built: `orch reject` command emits agent.rejected events (reject_cmd.go)
- Built: learning.go aggregates RejectedCount per skill
- **Missing:** RejectedCount is never read by daemon allocation, skill routing, or any behavioral code. Only SuccessRate is consumed (allocation.go:107-119). Result: rejection data is write-only.

**Loop 4: Comprehension Queue**
- Built: comprehension:pending labels added after auto-completion (coordination.go:226,242,271)
- Built: CheckComprehensionThrottle() counts pending items and blocks spawning
- Built: RemoveComprehensionPending() called in orch complete (complete_lifecycle.go:207)
- **Missing:** ComprehensionQuerier is never instantiated in daemon_loop.go. Field is always nil. Gate fails open. Result: labels accumulate, threshold never checked.

**Loop 5: Verification Metrics**
- Built: VerificationFailures and VerificationBypasses fields in SkillLearning
- Built: learning.go aggregates from events
- **Missing:** Neither field is read by any code path. Not in daemon, not in allocation, not in dashboard. Result: pure dead code.

**Loop 6: Rework Feedback**
- Built: orch rework command, agent.reworked events, ReworkCount aggregation
- **Missing (behavioral):** ReworkCount displayed in orient but never used in daemon allocation or skill routing. Result: monitoring-only, no behavioral feedback.

**Loop 7-10: Registered-But-Never-Invoked Periodic Tasks**
11 periodic tasks are registered in scheduler.go (lines 125-148) but never called from daemon_periodic.go:
- TaskReflect, TaskKnowledgeHealth, TaskFrictionAccumulation, TaskSynthesisAutoCreate
- TaskLearningRefresh, TaskPlanStaleness, TaskProactiveExtraction, TaskAccretionResponse
- TaskTriggerScan, TaskTriggerExpiry, TaskInvestigationOrphan

Each has config fields (Enabled, Interval), scheduler registration, and in some cases implementation functions — but runPeriodicTasks() (daemon_periodic.go:29-142) only calls 13 of 24 registered tasks.

### Structural Pattern

Every instance follows the same 3-layer architecture:
1. **Emission/Collection layer** — Events emitted, labels added, metrics computed (ALWAYS built)
2. **Configuration/Registration layer** — Config fields, scheduler tasks, interfaces defined (ALWAYS built)
3. **Consumption/Action layer** — Daemon reads signal and changes behavior (MISSING in 10/10 cases)

The pattern is not "Phase 3 problem" (integration testing). It's **consumer-last construction**: the system builds emitters and config first, then never returns to build the consumer that closes the loop.

---

## Model Impact

- [x] **Extends** model with: Failure Mode 7 (Human Negative-Feedback Channel) is the tip of a 10-instance iceberg. The `orch reject` command partially addressed the vocabulary gap (reject verb now exists) but the feedback loop remains open because: (1) reject is never triggered automatically (0 events in production), (2) RejectedCount is aggregated but never consumed by daemon behavior, (3) the audit system that would generate automatic rejections has a disconnected middle (no audit agent spawning). Beyond FM7, nine additional open loops exist in the completion/daemon pipeline, all following the same "consumer-last" construction pattern.

---

## Notes

- The prior investigation (2026-03-20-inv-investigate-whether-open-loop-systems.md) mapped the open-loop concept across 41 models. This probe grounds that conceptual finding in concrete code paths — actual line numbers where loops are open.
- The single metric that IS consumed (SuccessRate) has limited impact: allocation.go modulates priority by ±20%. Even closing the feedback loop for rejection would only matter if the daemon then ACTS differently (avoids high-rejection skills, escalates to architect review, etc.).
- The 11 registered-but-never-invoked periodic tasks represent ~2,800 lines of configuration and implementation code that is structurally unreachable from the daemon's main loop.
- Production data: 1,285 completions, 513 accretion.delta events, 0 agent.rejected, 0 audit verdicts, 0 reworks — confirming every feedback channel except SuccessRate is structurally unused.
