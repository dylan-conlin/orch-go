# Design: Structural Fix for Consumer-Last Construction Pattern

**Date:** 2026-03-22
**Status:** Complete
**Source:** orch-go-ay620 (architect design from investigation orch-go-pn9au)
**Beads:** orch-go-ay620

---

## Problem Statement

Investigation orch-go-pn9au found 10 open feedback loops in the daemon/completion pipeline where emitters were built but consumers never written. ~2,800 lines of structurally unreachable code. Root cause: each component completed as an independent beads issue; no issue was ever created for the consumer/wiring.

---

## Part A: Triage — Close vs Delete vs Demote

### Verdict Summary

| # | Loop | Verdict | Rationale | LOE |
|---|------|---------|-----------|-----|
| 1 | Quality Audit (spawn gap) | **Close** | Full pipeline built except spawn routing. High value — enables automated quality feedback. | M |
| 2 | Accretion Response | **Delete** | 513 events with no reader. Accretion gates converted to advisory (2026-03-17 decision). Event emission is waste. | S |
| 3 | Reject → Learning | **Close** | RejectedCount write-only. Allocation should use rejection signal alongside SuccessRate. | S |
| 4 | Comprehension Queue | **Close** | Full gate infrastructure built but ComprehensionQuerier never instantiated. One line of wiring missing. | S |
| 5 | Verification Metrics | **Delete** | VerificationFailures/Bypasses never read. No clear behavioral use case — the verification system already blocks at completion time. Keeping write-only counters is misleading. | S |
| 6 | Rework Feedback | **Demote** | ReworkCount shown in orient. Could feed allocation, but rework is rare (0 production events). Keep display, don't invest in daemon wiring until rework is actually used. | - |
| 7-10 | 11 Registered Periodic Tasks | **Delete registrations** | Constants + scheduler registrations with no implementation functions. Pure config noise. Remove registrations and config fields. When/if these features are built, re-add registration alongside implementation. | M |

### Decision Criteria

- **Close:** Infrastructure mostly built, high value, small remaining gap
- **Delete:** No clear behavioral use case, or superseded by other mechanisms
- **Demote:** Keep what exists, don't invest more until production signal justifies it

### Highest-Value Closures (ordered)

1. **Comprehension Queue (#4)** — Smallest gap (nil field → instantiation), highest risk (gate fails open, meaning comprehension throttle is structurally bypassed)
2. **Quality Audit (#1)** — Full pipeline ready, spawn routing is the missing piece
3. **Reject → Learning (#3)** — Small code change: allocation.go should blend RejectedCount into scoring

---

## Part B: Structural Prevention

### Why Consumer-Last Happens

The beads issue workflow naturally decomposes work into components. An agent gets "build event emission for X" and completes it. Another gets "add config for X" and completes it. Nobody gets "wire X into the daemon loop" because the decomposition stops at the emitter boundary.

This is a **decomposition-boundary problem**: the issue creator (orchestrator or daemon) thinks in terms of features ("add quality audit"), but the implementation decomposes into layers (emit → configure → consume). Each layer feels complete on its own. The consumer layer never gets an issue because the emitter agent reports "Phase: Complete" and the orchestrator closes the feature.

### Design: End-to-End Wiring Check (lint test)

Add a **structural lint test** that detects open loops at build time. This catches the pattern where it happens — in CI, before the gap can age.

**Mechanism:** A Go test that:

1. **Scans scheduler registrations** — every `s.Register(TaskFoo, ...)` must have a corresponding `RunPeriodicFoo()` call in `daemon_periodic.go`
2. **Scans event types** — every `Log{EventType}()` call must have at least one `ComputeLearning` consumer or daemon reader
3. **Scans daemon struct fields** — interface fields (like `ComprehensionQuerier`) that are never assigned in `daemonSetup()` are flagged

**Implementation:** Single test file `pkg/daemon/wiring_test.go` using AST scanning or simpler grep-based checks. This is the same pattern as the existing `_lint_test.go` files but for end-to-end wiring rather than code style.

**Why a test, not a hook:**
- Tests run in CI and locally via `make test`
- No governance-path issues (new test file, not editing `_lint_test.go`)
- Developers see the failure immediately when adding an emitter without a consumer
- Cheaper to maintain than a pre-commit hook (which has a history of bypass — see accretion gates decision)

### Design: Issue Template Guard (process)

When the orchestrator creates implementation issues for event-emitting features, the spawn context should include:

```
WIRING_REQUIREMENT: This feature emits [event type]. Consumer must be wired in daemon before marking complete.
Consumer location: [daemon_loop.go | daemon_periodic.go | allocation.go]
```

This is a **spawn context annotation**, not a gate. It makes the consumer requirement visible to the implementing agent. Combined with the lint test, this creates two layers: the agent knows about the requirement (annotation), and CI catches it if missed (test).

### Why NOT a completion gate

The accretion gates decision (2026-03-17) showed that blocking gates get bypassed 100% of the time when they create friction on otherwise-valid completions. A lint test is better because:
- It fails the build, which agents already fix
- It doesn't require `--skip` flags or human judgment
- It's deterministic (no false positives if well-written)

---

## Part C: Implementation Issues

### Issue 1: Wire ComprehensionQuerier in daemon setup
**Priority:** High (gate failing open = silent bypass)
**Scope:** Add `d.ComprehensionQuerier = &BeadsComprehensionQuerier{...}` in `daemonSetup()` (~5 lines)
**Skill:** feature-impl

### Issue 2: Delete accretion.delta emission and dead config
**Priority:** Medium (cleanup, removes ~200 lines of dead path)
**Scope:** Remove `accretion.delta` event emission from `complete_lifecycle.go`, remove `AccretionResponse*` config fields and scheduler registration
**Skill:** feature-impl

### Issue 3: Delete verification metrics dead fields
**Priority:** Medium (cleanup)
**Scope:** Remove `VerificationFailures`, `VerificationBypasses` from `SkillLearning`, remove aggregation code in `learning.go`
**Skill:** feature-impl

### Issue 4: Delete 11 orphaned scheduler registrations
**Priority:** Medium (cleanup, removes ~22 references + config fields)
**Scope:** Remove `TaskReflect`, `TaskKnowledgeHealth`, `TaskFrictionAccumulation`, `TaskSynthesisAutoCreate`, `TaskLearningRefresh`, `TaskPlanStaleness`, `TaskProactiveExtraction`, `TaskAccretionResponse`, `TaskTriggerScan`, `TaskTriggerExpiry`, `TaskInvestigationOrphan` registrations from `scheduler.go` and associated config fields
**Skill:** feature-impl

### Issue 5: Close reject → allocation feedback loop
**Priority:** Medium (behavioral improvement)
**Scope:** In `allocation.go`, blend `RejectedCount` into skill scoring alongside `SuccessRate`. E.g., `adjustedRate = successRate * (1 - rejectionPenalty)`
**Skill:** feature-impl

### Issue 6: Wire audit agent spawning
**Priority:** Low (full pipeline exists, but 0 audit agents ever spawned — validate the design works before investing)
**Scope:** Add `audit:deep-review` label recognition in spawn routing so audit-labeled issues get audit agents spawned
**Skill:** feature-impl (needs investigation phase first)

### Issue 7: Add wiring lint test
**Priority:** High (prevention)
**Scope:** Create `pkg/daemon/wiring_test.go` that checks scheduler registrations have matching RunPeriodic calls, and daemon struct interface fields are assigned in setup
**Skill:** feature-impl (mode: tdd)

---

## Migration Status

```
MIGRATION_STATUS:
  designed: triage of 10 loops, structural prevention via lint test + spawn annotation, 7 implementation issues
  implemented: none
  deployed: none
  remaining: 7 implementation issues (see Part C)
```
