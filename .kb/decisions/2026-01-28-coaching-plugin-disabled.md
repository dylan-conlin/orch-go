---
status: active
blocks:
  - keywords:
      - coaching plugin
      - coaching alerts
      - coaching worker detection
      - coaching.ts
      - orchestrator coaching
    patterns:
      - "**/plugins/coaching*"
---

# Decision: Coaching Plugin Disabled

**Date:** 2026-01-28
**Status:** Active
**Decision:** Disable the coaching plugin entirely

## Context

The coaching plugin was designed to detect orchestrator anti-patterns (action_ratio, analysis_paralysis, circular_pattern) and inject coaching messages. Worker sessions needed to be excluded from coaching.

## The Failure

Worker detection never worked reliably despite:
- **18+ investigations** over 3 weeks
- **13+ fix commits** to coaching.ts
- **4 different detection approaches** (metadata.role, title patterns, tool paths, API lookup)

On Jan 28 alone, 5 investigations produced contradictory conclusions:
- One said "working correctly, no changes needed"
- One said "just rebuild OpenCode"
- One said "metadata.role can't work, revert to title-based"
- One said "migrate all plugins to metadata.role"

Each agent verified their slice looked correct. None verified the actual end-to-end experience.

## The Deeper Problem

This revealed a **verification bottleneck**: agents can grep logs, check code paths, and conclude "fixed" without the bug actually being fixed. The human (Dylan) was the only one who could observe "I'm still getting alerts on workers."

When asked what he was seeing, Dylan said: "I've lost faith that this is actually possible. I've lost trust in what workers say and what orchestrators say."

## Decision

Disable the coaching plugin (`coaching.ts.disabled`) rather than continue the investigation churn.

## Constraints Established

- `kb-4ee4e1`: Failed attempt - coaching plugin worker detection
- `kb-9c8c38`: Constraint - agent "fixed" claims require end-to-end human verification

## Reopening Criteria

To re-enable the coaching plugin, we would need:
1. OpenCode to expose `session.metadata.role` reliably (upstream fix)
2. A single, simple detection mechanism (not 4 layered heuristics)
3. Human-verified end-to-end test before claiming "fixed"

## References

- `.kb/investigations/2026-01-28-inv-orchestrator-coaching-plugin-cannot-reliably.md`
- `.kb/investigations/2026-01-28-inv-coaching-plugin-still-fires-workers.md`
- `.kb/investigations/2026-01-28-inv-verify-coaching-plugin-worker-detection.md`
- `.kb/investigations/2026-01-28-inv-debug-coaching-plugin-still-fires.md`
- `.kb/investigations/2026-01-27-inv-coaching-plugin-worker-detection-keep.md`
