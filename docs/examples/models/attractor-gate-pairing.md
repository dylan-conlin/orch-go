# Model: Attractor-Gate Pairing

<!-- ABOUT THIS MODEL
    This model demonstrates the cross-cutting synthesis style —
    it consolidates findings that appeared independently across
    multiple domain-specific models. It also shows the epistemic
    tiering structure: Confirmed > Working Hypotheses > Open Frontier.
-->

**Domain:** Multi-Agent Coordination Mechanisms
**Created:** 2026-03-25
**Updated:** 2026-03-25
**Status:** Active
**Source:** Consolidated from coordination experiments (329 trials), knowledge management (1,199 investigations), harness engineering (12 intervention audits), architectural enforcement (31 interventions measured), + 6 external framework validations

## What This Is

A cross-cutting model for how structural attractors and enforcement gates interact as a paired mechanism in multi-agent systems. The gate/attractor distinction appears independently in 7+ domain-specific models. This model consolidates the shared mechanism, defines its empirical boundaries, and identifies the open questions.

**Core claim:** Neither attractors nor gates work alone. Attractors embed coordination at design time (structure IS coordination). Gates enforce boundaries at runtime (require correct decisions). The pairing — attractor provides destination, gate blocks the old path — is the only configuration that has held empirically.

<!-- Epistemic tiering: Organize claims by confidence level.
     This is more honest than mixing confirmed findings with
     hypotheses in a flat list. -->

This document is organized by epistemic tier:

1. **Confirmed** — Observed across multiple domains with quantitative evidence
2. **Working Hypotheses** — Supported by evidence but not independently reproduced
3. **Open Frontier** — The questions that matter most, where the model has no answer yet

---

# Part I: Confirmed Findings

## Claim 1: Attractor + gate together is the only configuration that holds

Neither mechanism works alone. Both together produce durable results.

| Configuration | Domain | Outcome | Evidence |
|---|---|---|---|
| Attractor + gate | Code: large command file | -1,755 lines, held | Extracted package (attractor) + accretion gate |
| Attractor + gate | Knowledge: orphan rate | 94.7% down to 52.0% | Probe directories (attractor) + model-stub gate |
| Attractor only | Code: daemon file | +892 lines post-extraction | Extracted package existed but no gate blocked original path |
| Gate only | Code: large command file | Ignored | Pre-commit warning with no destination package |
| Gate only | Behavioral: orchestrator | 83/87 non-functional | 87 constraints, 100% bypass rate on advisory gates |

**Cross-domain consistency:** This pattern holds in code organization (file growth), knowledge organization (investigation orphans), and agent coordination (merge conflicts). Three independent domains, same result.

## Claim 2: Attractors succeed because coordination is embedded at design time

This is the mechanism distinction, confirmed empirically and across external frameworks.

**Internal evidence:**
- Placement (attractor) = 20/20 SUCCESS; post-hoc verification (gate) = 20/20 CONFLICT
- Agents performed gate checks, reported "no conflict expected," kept conflicting insertion points
- 83/87 behavioral constraints non-functional despite agents reading and understanding them

**External validation (6 frameworks):**

| Framework | Mechanism | Success |
|---|---|---|
| autoresearch | Pure attractor (N=1 structural constraint) | High |
| Anthropic production | Attractor-dominant (task regions + output formats) | High |
| CrewAI | Gate (manager LLM routing) | Low |
| LangGraph | Gate (conditional graph edges) | Low |
| OpenAI Agents SDK | Gate (output-mediated handoffs) | Low |

## Claim 3: Behavioral constraints dilute — but from tension, not count

**Original observation:** 87 behavioral constraints, 83 classified as non-functional (>50% violation rate).

**Qualification:** A controlled experiment (40 trials, N=1 to 20) found NO degradation with orthogonal additive constraints — compliance stayed ~97% regardless of count. The original observation likely reflects **constraint tension** (rules that interfere with each other or the primary task), not constraint count alone.

**Revised claim:** "Constraints that create competing gradients degrade; orthogonal constraints do not, regardless of count."

**Intervention effectiveness hierarchy (empirical):**
Structural attractors > signaling mechanisms > blocking gates > advisory gates > metrics-only

Only 4 of 31 measured interventions had measurable impact, and all 4 involved attractors or hard gates.

---

# Part II: Working Hypotheses

## Hypothesis 1: Extraction without gates is a pump

Large files shrank via extraction, then re-accreted within weeks. Proposed mechanism: the entry point (command definition, main function) acts as a competing attractor that pulls new code back toward the original file. Extraction without redirecting this gravity source is thermodynamically unstable.

**Status:** Observed in 2 cases. Not yet experimentally isolated from confounding factors.

## Hypothesis 2: Modification tasks are immune to coordination failure

40/40 SUCCESS across all 4 coordination conditions for modification tasks. Agents anchor to their target functions and produce non-overlapping diffs by construction.

**Boundary condition:** When work is structurally self-coordinating, neither attractors nor gates add value. Communication overhead is pure cost (+35% agent time, zero benefit).

## Hypothesis 3: Automated attractor discovery from collision data is feasible

1 collision is sufficient to generate effective placement constraints. Algorithm: parse conflict diffs, extract hunk locations, map to function boundaries, distance-maximize alternative points. 7/7 SUCCESS after 1 learning collision.

**Implication:** Attractors can be discovered rather than designed, turning the system into a closed loop: fail, learn, constrain, prevent.

---

# Part III: Open Frontier

<!-- Open Frontier names the questions the model can't answer yet.
     This is where future probes should target. A model that claims
     to have no open questions is probably overclaiming. -->

## Q1: What is the mechanism behind constraint dilution?

Three candidate mechanisms with distinguishing predictions:

| Mechanism | Failure shape | Pair-specific? | Removal restores? |
|---|---|---|---|
| Resource competition | Gradual degradation | No | Removing any one helps all |
| Interference | Sudden failure at specific pairs | Yes | Removing one restores its partner |
| Threshold collapse | Sharp cutoff at critical N | No | Removing one below threshold restores all |

## Q2: Does the model predict something untested?

All current evidence is retrodictive — the model explains observations we already had. No prediction has been generated and then tested.

**Candidate predictions to test:**
- Adding a second attractor should create basin-boundary accumulation
- Two good design patterns in the same file should interfere predictably
- Attractor effectiveness should degrade with distance from entry point

## Q3: Where does the model break?

Zero disconfirmation attempts. Every probe confirmed. This means either the model is very general or we haven't pushed hard enough.

**Candidate boundaries:**
- **Non-code domains:** Does the pairing predict anything beyond code and knowledge organization?
- **Human teams:** Derived from AI agent coordination. Does it apply to human engineers?
- **Emergent attractors:** All tested attractors are designer-created. What about spontaneous ones?
- **Cross-cutting concerns:** Features that genuinely belong in two well-designed packages
- **Adversarial conditions:** Tasks that *require* violating the attractor's structure

---

# Probe Log

<!-- Probes are listed chronologically with one-line findings.
     This is the model's evidence chain — readers can trace
     how confidence in each claim was built over time. -->

| Date | Probe | Result |
|---|---|---|
| 2026-03-22 | External framework validation | Confirmed: attractor-dominant frameworks succeed, gate-only fail |
| 2026-03-22 | Automated attractor discovery | Confirmed: 1 collision sufficient for constraint generation |
| 2026-03-22 | Attractor decay resilience | Confirmed: 9/9 success with stale anchors |
| 2026-03-23 | Agent scaling (N=4, N=6) | Found boundary: deterministic conflict when agents > regions |
| 2026-03-23 | Modification tasks | Found boundary: self-coordinating tasks don't need attractors |
| 2026-03-26 | Constraint scaling (orthogonal) | **NULL** — flat curve, no degradation N=1 to 20 with additive constraints |
