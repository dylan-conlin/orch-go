# Decision: Orchestrator Reflection Sessions Use Hybrid Triggers, Lane Scope, and Amnesia-Tax Prioritization

**Date:** 2026-02-07
**Status:** Accepted
**Context:** `kb-reflect` handles worker triage mechanics, but orchestrator-level reflection sessions lack a reliable protocol for when to run, what to include, how to prioritize, and how to measure impact.

## Decision

Adopt an orchestrator reflection protocol with six rules:

1. **Cadence is hybrid:** event-driven triggers with a time-floor safety check.
2. **Scope is lane-based:** each session chooses one primary lane, plus a short hygiene sweep.
3. **Prioritization uses amnesia tax:** rank by rediscovery cost, not raw keyword/investigation counts.
4. **Success is measured by rediscovery reduction:** track repeat investigations and citation/follow-through quality.
5. **Automation surfaces, orchestrator decides:** tooling pre-scores candidates; human judgment determines coherence and promotion.
6. **Artifact placement is split:** detailed procedure lives as a guide; orchestrator skill gets a lightweight trigger/checklist hook.

## Why

- Current full-output reflection views are too broad for high-quality decisions in one pass.
- Existing evidence shows mixed-coherence clusters (some actionable, some keyword noise), so counts alone are weak.
- Prior decisions already establish that synthesis quality comes from orchestrator review and follow-up, not blind automation.
- Principles (`Reflection Before Action`, `Premise Before Solution`) favor process-level discipline over ad hoc cleanup.

## Protocol

### 1) Cadence (When to run)

Run a reflection session when **any** trigger fires:

- `Volume trigger`: >=10 new investigations since last reflection
- `Recurrence trigger`: >=3 investigations in 14 days for the same coherent problem family
- `Milestone trigger`: major feature/epic completion or repeated constraint surfacing in the same area
- `Time-floor trigger`: 14 days elapsed since last reflection (safety backstop)

This replaces aspirational weekly scheduling with concrete conditions.

### 2) Scope per Session (What to include)

Each session has:

- **Primary lane (70-80%):** choose one lane only (synthesis, stale, open, drift/principles, or skill updates)
- **Maintenance lane (20-30%):** quick sweep for urgent stale/open items requiring immediate action

Rotation rule:
- Default to the lane with the highest amnesia-tax score among top candidates.
- If scores are close, rotate lane from previous session to avoid starvation.

### 3) Prioritization (How to rank)

Use an **Amnesia Tax Score (ATS)** per candidate:

`ATS = Recurrence (0-3) + Delay Cost (0-3) + Blast Radius (0-2) + Preventability (0-2) - Noise Penalty (0-2)`

Definitions:
- **Recurrence:** how often this rediscovery appears recently
- **Delay Cost:** cumulative time/context cost from not addressing it
- **Blast Radius:** how many sessions/agents/areas are affected
- **Preventability:** confidence that a guide/decision/constraint can reduce repeats
- **Noise Penalty:** weak coherence signal (e.g., broad keyword collisions)

Select top 3-5 candidates by ATS, then apply orchestrator coherence judgment before action.

### 4) Success Metrics (How to know it works)

Track rolling 30-day metrics:

- **Repeat Investigation Rate (RIR):** repeats per top problem family
- **Rediscovery Latency:** median days between first and repeat investigation for same family
- **Promotion Yield Quality:** percent of promoted artifacts cited within 30 days
- **Reflection Throughput Quality:** durable outcomes per session (target 1-3), not raw output count

Reflection is successful when RIR trends down and citation/follow-through trends up without backlog inflation.

### 5) Automation vs Manual Boundary

**Automate in `kb reflect`:**
- candidate surfacing and lane grouping
- ATS pre-scoring fields and recurrence windows
- stale/open threshold detection
- reflection run telemetry collection

**Keep manual at orchestrator layer:**
- coherence validation (is this a real family or keyword collision?)
- intervention choice (guide vs decision vs constraint vs issue)
- trade-off decisions and sequencing
- final promotion acceptance

### 6) Artifact Placement

Use a split model:

- **Guide:** primary home for step-by-step reflection session procedure
- **Orchestrator skill:** short hook that says when to invoke the reflection guide and which checklist to follow

Rationale: this avoids bloating skill prompt context while keeping procedure reusable and updateable.

## Session Checklist (Operational)

At session start:
- confirm trigger that fired
- pick primary lane
- review top ATS candidates (3-5)

During session:
- decide action per candidate (promote, defer, discard)
- record rationale for defer/discard to improve future scoring

At session end:
- produce 1-3 durable outputs (decision/guide/constraints/issues)
- log metrics snapshot and next-lane recommendation

## Consequences

**Positive:**
- Reflection happens on realistic triggers.
- Sessions become focused and actionable.
- Metric feedback loop supports threshold tuning.
- Strategic synthesis stays with orchestrator judgment.

**Costs/Trade-offs:**
- Requires maintaining simple telemetry and scoring fields.
- Some valid items wait for lane rotation.
- ATS calibration needs iteration over early sessions.

## Implementation Notes

1. Create guide: `orchestrator-reflection-session-protocol.md` (procedural details).
2. Add minimal orchestrator skill hook to invoke guide when trigger fires.
3. Extend `kb reflect` output with ATS input fields and run metadata.
4. Review metrics after first 3-4 sessions and tune thresholds.

## References

- `.kb/investigations/2026-02-07-inv-design-orchestrator-level-reflection-session.md`
- `~/.kb/principles.md:697`
- `~/.kb/principles.md:796`
- `.kb/decisions/2026-01-07-synthesis-is-strategic-orchestrator-work.md`
