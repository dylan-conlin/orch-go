# Probe: Constraint Scaling Curve (Null Result)

**Date:** 2026-03-26
**Model:** attractor-gate
**Claim tested:** Claim 3 — "Behavioral constraints dilute at 10+ co-resident rules"
**Result:** CONTRADICTS (at this experiment's constraint type)

## Design

Phase 1 of the mechanism discrimination experiment. 40 trials across N=1,2,4,6,8,10,14,20 co-resident constraints, 5 trials per N, Haiku model. Task: implement FormatBytes function in Go. 20 grep-detectable constraints covering error handling, function style, documentation, testing, and type system patterns.

## Results

| N | Mean compliance | StdDev | Min | Max |
|---|---|---|---|---|
| 1 | 1.000 | 0.000 | 1.000 | 1.000 |
| 2 | 0.900 | 0.200 | 0.500 | 1.000 |
| 4 | 1.000 | 0.000 | 1.000 | 1.000 |
| 6 | 0.933 | 0.134 | 0.666 | 1.000 |
| 8 | 1.000 | 0.000 | 1.000 | 1.000 |
| 10 | 0.920 | 0.040 | 0.900 | 1.000 |
| 14 | 0.986 | 0.029 | 0.928 | 1.000 |
| 20 | 0.970 | 0.040 | 0.900 | 1.000 |

**Curve shape:** FLAT. No degradation from N=1 to N=20.

## Measurement incident

First run showed apparent degradation (0.90→0.70 from N=2→N=4). Root cause: 5 of 20 constraint detectors used `grep -P` (Perl regex), which silently fails on macOS BSD grep. Those 5 constraints scored as 0% compliance across all N values, creating a fake degradation curve proportional to how many broken detectors appeared in each random sample. After fixing detectors to `grep -E`, the curve went flat.

## Interpretation

The null result does NOT mean "constraints don't degrade." It means **this experiment measured satisfiable boilerplate accretion, not constraint competition.**

The 20 constraints were largely orthogonal additive requests: "add named returns," "add a const block," "add an example test." They don't force tradeoffs in path selection, planning budget, or implementation commitments. An agent can satisfy all 20 because each is a local addition that doesn't affect the others.

The prior observational claim (83/87 orchestrator constraints non-functional) may have been measuring a different phenomenon: constraints that **interfere** with each other ("never implement" + "verify agent output by reading code") or that conflict with the primary task. The experiment's constraints didn't interfere.

## What this means for the model

- **Claim 3 needs qualification:** Constraints dilute at 10+ only when they create tension — not when they're orthogonal additions. The number alone is not the variable.
- **The mechanism question is reframed:** Not "resource competition vs interference vs threshold collapse" but "what types of constraint tension produce degradation?" Three candidate tension types:
  1. **Resource tension** — constraints competing for the same budget (lines, complexity, tokens)
  2. **Design tension** — constraints implying incompatible architectural choices
  3. **Priority tension** — constraints that can't all be satisfied under finite context
- **Energy landscape framing (from Codex conversation):** Orthogonal constraints don't reshape the energy landscape — they just add decoration to whatever path the agent was already taking. Interfering constraints create competing gradients. The experiment needs to measure gradient competition, not checklist compliance.

## Next experiment needed

Redesigned constraint pool with genuine tension between constraints, not orthogonal additions. The dependent variable should be which constraint the agent sacrifices when it can't satisfy all of them, not just compliance rate.

## Raw data

- Results: `experiments/coordination-demo/redesign/results/mechanism-p1-20260325-230114/`
- Broken-detector run (for comparison): `experiments/coordination-demo/redesign/results/mechanism-p1-20260325-212247/`
- Scripts: `experiments/coordination-demo/redesign/run-mechanism.sh`, `score-mechanism.sh`
