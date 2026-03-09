# Decision: Five Design Principles For Automation Legibility

**Date:** 2026-03-01
**Status:** Accepted

## Context

As the orchestration system grew more autonomous (daemon, spawn gates, completion verification), the question arose: how should automation make itself legible to the human operator? Literature review of 40 years of human-automation interaction research yielded five principles.

## Decision

**Chosen:** Five design principles for automation legibility:

1. **Pace-layered transparency** — Different system layers (agent lifecycle, daemon decisions, knowledge evolution) operate at different timescales and need different visibility mechanisms
2. **Support all 3 SA levels** (Endsley 1995) — Perception (what's happening), Comprehension (what it means), Projection (what's next)
3. **Maintain joint cognitive system** (Hollnagel-Woods 2005) — Human and automation form a team; neither is just "monitoring" the other
4. **Honest legibility / avoid Scott trap** (Scott 1998) — Making things legible for management can destroy the richness that makes them work; legibility must preserve fidelity
5. **Design for failure not success** (Bainbridge 1983) — Automation that works perfectly makes humans unable to intervene when it fails

**Rationale:** Distilled from literature review: Bainbridge 1983, Endsley 1995, Parasuraman-Sheridan-Wickens 2000, Hollnagel-Woods 2005, SAT/DSAT Chen 2014/2018, ISA-101, Scott 1998.

## Consequences

- Positive: Provides principled framework for dashboard, spawn visibility, and orchestrator UX decisions
- Positive: Prevents naive "show everything" or "hide everything" approaches
- Risk: Principles are high-level — need per-feature interpretation when applying

## Source

**Promoted from:** quick entry kb-644210 (decision)
**Original date:** 2026-03-01

