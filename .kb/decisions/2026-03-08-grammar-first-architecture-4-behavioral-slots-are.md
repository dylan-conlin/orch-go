# Decision: Grammar-First Architecture for Behavioral Skill Slots

**Date:** 2026-03-04
**Status:** Accepted
**Enforcement:** convention

## Context

Skill content transfer experiments revealed a U-curve: behavioral constraints without calibration context produce worse outcomes than no constraints at all. This led to a "grammar-first" approach where behavioral items are structured as paired slots rather than standalone mandates.

## Decision

**Chosen:** 4 behavioral slots, each paired with calibration knowledge:

1. **Delegation** — What the worker can decide vs must escalate (paired with: examples of each)
2. **Undefined-behavior-handler** — What to do when instructions don't cover the situation (paired with: common edge cases)
3. **Filter + act-by-default** (combined) — Default behaviors that apply unless overridden (paired with: when to override)
4. **Pressure-over-compensation** — Inject friction rather than compensate for failures (paired with: what friction looks like)

**Rationale:** U-curve data shows behavioral constraints without calibration context create a danger zone worse than bare (no constraints). Matched-pair design addresses this by ensuring every behavioral mandate has adjacent calibration knowledge.

## Consequences

- Positive: Prevents the U-curve where partial behavioral guidance degrades performance
- Positive: Constrains skill authors to a finite set of behavioral slots, preventing unbounded growth
- Risk: Requires discipline to pair every behavioral item with calibration context

## Source

**Promoted from:** quick entry kb-2a6199 (decision)
**Original date:** 2026-03-04


## Auto-Linked Investigations

- .kb/investigations/2026-03-01-investigation-orchestrator-skill-behavioral-testing-baseline.md
- .kb/investigations/2026-03-04-design-grammar-first-skill-architecture.md
- .kb/investigations/2026-03-01-inv-formal-grammar-theory-llm-constraint-systems.md
