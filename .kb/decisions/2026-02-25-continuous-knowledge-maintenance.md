---
stability: foundational
---
# Continuous Knowledge Maintenance via Orchestration Side Effects

**Date:** 2026-02-25
**Status:** Accepted
**Enforcement:** context-only
**Context:** Knowledge system reflection session

## Decision

Knowledge maintenance is a side effect of normal orchestration operations, not a separate activity. The "leave it better" pattern — already proven at the worker level — extends to completion, spawn, and daemon touchpoints.

`kb reflect` becomes an audit/health-check tool, not the primary maintenance mechanism.

## Problem

The knowledge system follows a neglect → accumulate → batch reflect → guilt → neglect cycle:

- Quick entries accumulate (498 decisions, 0% tracked references, no promotion pipeline)
- Investigations pile up (774 pre-entropy-revert investigations archived this session)
- Models go stale without anyone noticing
- Maintenance happens in giant batch sessions that feel like debt repayment

The root cause: knowledge maintenance is decoupled from knowledge consumption. Entries are created during work but only reviewed in dedicated reflect sessions that are easy to skip.

## Solution: Maintenance at Three Touchpoints

### Touchpoint 1: Completion Review (highest value)

When `orch complete` runs, the orchestrator is already reviewing SYNTHESIS.md, diffs, and building understanding. This is the second-highest-context moment after the worker itself.

**Added responsibility:**
- **Promote:** Quick entries validated by the completed work → formal decisions or model updates
- **Supersede:** Quick entries absorbed into deliverables → mark superseded with reason
- **Flag:** Contradictions between completed work and existing models → create follow-up issue

**Cost:** 2-3 minutes added to completion review.

### Touchpoint 2: Spawn Context (pruning)

`kb context` already runs at spawn time. The orchestrator reads the knowledge. Two additions:

**Orchestrator observation:**
- If `kb context` returns 15+ constraints for a domain, that domain needs a model (entries are compensating for missing synthesis)
- Stale entries surfaced during spawn review get flagged

**Worker instruction (extend "leave it better"):**
- Workers already create entries. Also instruct: "if you find constraints or decisions in your spawn context that are no longer accurate, supersede them."
- "Leave it better" expands from creation-only to creation + pruning.

### Touchpoint 3: Daemon Idle Cycles (lightweight hygiene)

The daemon polls `bd ready`, sometimes finds nothing to spawn. Those idle cycles are free.

**Added responsibility:**
- Count active quick entries by domain
- Flag domains where entries accumulate without promotion (e.g., 20+ active entries in same area)
- Create `triage:review` issues for knowledge maintenance when thresholds exceeded

**Cost:** Near zero — runs only during idle polling gaps.

## Design Principle

> Close the feedback loop at the point of action, not in a separate maintenance pass.

**Before (open loop):**
```
worker creates entry → entry sits in JSONL → ... → batch reflect → promote/prune
                                               ↑
                                          this gap grows forever
```

**After (closed loop):**
```
worker creates entry → completion reviews → promoted/superseded
                     → next spawn surfaces → validated or pruned
                     → daemon flags accumulation → model created
```

Every touchpoint does one small maintenance action. No single touchpoint does all the work. The aggregate effect is continuous curation.

## Relationship to Existing Patterns

- **"Leave it better" (worker skill pattern):** This decision extends the pattern from workers to all orchestration levels. Workers capture; completion promotes; spawn prunes; daemon monitors.
- **Entropy spiral revert (Jan 18):** The investigation accumulation was the same anti-pattern — unbounded creation without feedback. This decision applies the lesson to all knowledge artifacts.
- **Pressure Over Compensation principle:** Don't compensate for knowledge staleness by pasting answers. Let the maintenance touchpoints create natural pressure for curation.

## What This Changes

| Before | After |
|--------|-------|
| `kb reflect` is primary maintenance | `kb reflect` is audit/health-check |
| Quick entries accumulate indefinitely | Quick entries promoted or superseded at completion |
| Models go stale silently | Completion flags contradictions |
| Batch reflect sessions feel like debt | Continuous small actions prevent debt |
| Only workers "leave it better" | All orchestration levels maintain knowledge |

## Consequences

- Completion reviews take slightly longer (2-3 min) but produce higher-quality knowledge state
- Quick entry volume should stabilize as promotion/superseding keeps pace with creation
- Daemon creates maintenance issues, keeping orchestrator aware of knowledge health
- `kb reflect` sessions become rare strategic reviews, not routine cleanup

## Implementation

Three phases, independently spawnable:

1. **Completion-time knowledge review** — Add knowledge maintenance step to `orch complete` flow
2. **Spawn-time pruning instructions** — Update worker skills to include pruning, add orchestrator observation at spawn
3. **Daemon idle hygiene** — Add knowledge health check to daemon idle cycles
