# Model: Attractor-Gate Pairing

**Created:** 2026-03-25
**Updated:** 2026-03-25
**Status:** Active
**Source:** Consolidated from coordination (329 trials), knowledge-accretion (1,199 investigations), harness-engineering (12 intervention audits), architectural-enforcement (31 interventions measured), + 6 external framework validations

## What This Is

A cross-cutting model for how structural attractors and enforcement gates interact as a paired mechanism in multi-agent systems. The gate/attractor distinction appears independently in 7+ domain-specific models in this knowledge base. This model consolidates the shared mechanism, defines its empirical boundaries, and identifies the open questions that no single domain model can answer.

**Core claim:** Neither attractors nor gates work alone. Attractors embed coordination at design time (structure IS coordination). Gates enforce boundaries at runtime (require correct decisions). The pairing — attractor provides destination, gate blocks the old path — is the only configuration that has held empirically.

This document is organized by epistemic tier:

1. **Confirmed** — Observed across multiple domains with quantitative evidence
2. **Working Hypotheses** — Supported by evidence but not independently reproduced or mechanism-tested
3. **Open Frontier** — The questions that matter most, where the model has no answer yet

---

# Part I: Confirmed Findings

*Cross-domain evidence from code coordination, knowledge management, and external framework validation.*

## Claim 1: Attractor + gate together is the only configuration that holds

Neither mechanism works alone. Both together produce durable results.

| Configuration | Domain | Outcome | Evidence |
|---|---|---|---|
| Attractor + gate | Code: spawn_cmd.go | -1,755 lines, held | `pkg/spawn/backends/` (attractor) + accretion gate |
| Attractor + gate | Knowledge: orphan rate | 94.7% → 52.0% | `.kb/models/*/probes/` (attractor) + model-stub gate |
| Attractor only | Code: daemon.go | +892 lines post-extraction | `pkg/daemon/` existed but no gate blocked cmd/ path |
| Gate only | Code: spawn_cmd.go | Ignored | Pre-commit warning with no destination package |
| Gate only | Behavioral: orchestrator | 83/87 non-functional | 87 constraints, 100% bypass rate on advisory gates |

**Cross-domain consistency:** This pattern holds in code organization (file growth), knowledge organization (investigation orphans), and agent coordination (merge conflicts). Three independent domains, same result.

**Source probes:** coordination/probes/* (329 trials), knowledge-accretion/probes/2026-03-22-probe-validate-gate-attractor-external-frameworks.md, knowledge-accretion/probes/2026-03-20-probe-intervention-effectiveness-audit.md

## Claim 2: Attractors succeed because coordination is embedded at design time; gates fail because they require correct runtime decisions

This is the mechanism distinction, confirmed empirically and across external frameworks.

**Internal evidence:**
- Placement (attractor) = 20/20 SUCCESS; post-hoc verification (gate) = 20/20 CONFLICT (coordination model, same-file experiments)
- Agents performed gate checks, reported "no conflict expected," kept conflicting insertion points (Claim 5, coordination model)
- 83/87 behavioral constraints non-functional despite agents reading and understanding them (knowledge-accretion)

**External validation (6 frameworks):**

| Framework | Mechanism | Success |
|---|---|---|
| autoresearch | Pure attractor (N=1 structural constraint) | High |
| Anthropic production | Attractor-dominant (task regions + output formats) | High |
| CrewAI | Gate (manager LLM routing) | Low |
| LangGraph | Gate (conditional graph edges) | Low |
| OpenAI Agents SDK | Gate (output-mediated handoffs) | Low |

**McEntire degradation gradient:** Success drops monotonically with gate/attractor ratio: single agent (100%) → hierarchical (64%) → swarm (32%) → pipeline (0%).

**Source:** knowledge-accretion/probes/2026-03-22-probe-validate-gate-attractor-external-frameworks.md

## Claim 3: Behavioral constraints dilute at 10+ co-resident rules — QUALIFIED

**Original claim:** Measured in orchestrator skill: 87 behavioral constraints, 83 classified as non-functional (>50% violation rate). Constraints compete with system prompt at 17:1 signal disadvantage.

**Qualification (2026-03-26):** A controlled experiment (40 trials, N=1→20, Haiku) found NO degradation with orthogonal additive constraints — compliance stayed ~97% from N=1 through N=20. The original observation may reflect **constraint tension** (rules that interfere with each other or the primary task), not constraint count. The number alone is not the variable; the degree of tension between constraints is.

Claim needs redesign: "constraints dilute at 10+" → "constraints that create competing gradients degrade; orthogonal constraints do not, regardless of count." Investigation orch-go-bptuy created to test with tension-based constraints.

**Intervention effectiveness hierarchy (empirical):**
Structural attractors > signaling mechanisms > blocking gates > advisory gates > metrics-only

Only 4 of 31 measured interventions had measurable impact, and all 4 involved attractors or hard gates — never advisory constraints alone.

**Source:** knowledge-accretion/model.md (intervention audit), harness-engineering/model.md (constraint dilution measurement), attractor-gate/probes/2026-03-26-probe-constraint-scaling-null-result.md

## Claim 4: Conflicts under competing attractors are deterministic, not stochastic

When agents compete for shared insertion points, each pair is 5/5 SUCCESS or 0/5 CONFLICT — never 3/5. Conflict structure is fully determined at design time by region assignment + import compatibility.

Two failure mechanisms at N>2:
1. **Same-region gravitational convergence** — unavoidable when agents > regions (pigeonhole)
2. **Import-block conflicts** — agents in different regions still conflict if they modify shared imports incompatibly. Invisible at N=2.

**Source:** coordination/probes/2026-03-23-probe-agent-scaling-limited-insertion-points.md (130 agent invocations, 210 pairwise merge checks)

## Claim 5: Attractors tolerate stale anchors; region separation is the load-bearing property

Tested with ORIGINAL (stale) placement prompts after three mutation types: function renames, file reorganization, competing insertion points. 9/9 SUCCESS.

Agents compensate via semantic adaptation (find equivalent location) and anchor redundancy (multi-anchor placement survives losing one). The coordination value comes from **region separation**, not specific anchor names.

**Source:** coordination/probes/2026-03-22-probe-attractor-decay-degradation-curve.md

---

# Part II: Working Hypotheses

*Supported by evidence but needing further testing or independent reproduction.*

## Hypothesis 1: Extraction without gates is a pump

spawn_cmd.go shrank -1,755 lines via extraction to `pkg/spawn/backends/`, then re-accreted +483 lines in 3 weeks (~160 lines/week). daemon.go showed the same pattern: extraction to `pkg/daemon/` followed by +892 lines growth in cmd/.

**Proposed mechanism:** Feature gravity — the Cobra command definition (or equivalent entry point) acts as a competing attractor that pulls new code back toward the original file. Extraction without redirecting this gravity source is thermodynamically unstable.

**Status:** Observed in 2 cases (spawn, daemon). Not yet experimentally isolated from confounding factors (team velocity, feature load, file complexity).

## Hypothesis 2: Modification tasks are immune to coordination failure

40/40 SUCCESS across all 4 coordination conditions (including no-coord) for modification tasks. Agents anchor to their target functions and produce non-overlapping diffs by construction.

**Boundary condition:** This is a genuine boundary of attractor-gate pairing's applicability. When work is structurally self-coordinating, neither attractors nor gates add value. Communication overhead is pure cost (+35% agent time, zero benefit).

**Source:** coordination/probes/2026-03-23-probe-modification-task-experiment.md

## Hypothesis 3: Automated attractor discovery from collision data is feasible

1 collision is sufficient to generate effective placement constraints. Algorithm: parse conflict diffs → extract hunk locations → map to function boundaries → distance-maximize alternative points. 7/7 SUCCESS after 1 learning collision.

**Implication:** Attractors can be discovered rather than designed, turning the system into a closed loop: fail → learn insertion point conflict → generate constraint → prevent future conflict.

**Source:** coordination/probes/2026-03-22-probe-automated-attractor-discovery.md

## Hypothesis 4: The intervention effectiveness hierarchy is general

Structural attractors > signaling > blocking gates > advisory gates > metrics-only. Measured in orch-go's own governance system across 31 interventions. Unknown whether this ranking holds in other codebases, team sizes, or domains.

**Sub-hypothesis (2026-03-27):** Within "blocking gates," structurally-informed gates (those that count against a structural property like phase count) are more effective than boolean gates (those that check "at least one"). The architect handoff gate was boolean ("does any issue exist?") and failed silently for multi-phase designs — same failure mode as advisory gates. After upgrading to count issues against detected phases, the gate catches the gap. This suggests: **structurally-informed gates > boolean gates > advisory gates**.

**Source:** attractor-gate/probes/2026-03-27-probe-multi-phase-handoff-gate-coverage.md

---

# Part III: Open Frontier

*The questions that matter most. These are where the model's value will be won or lost.*

## Q1: What is the mechanism? (Resource competition vs. interference vs. threshold collapse)

We know constraints dilute at 10+. We know extraction without attractors pumps. We don't know WHY.

**Three candidate mechanisms with distinguishing predictions:**

| Mechanism | Failure shape | Pair-specific? | Removal restores? |
|---|---|---|---|
| Resource competition (finite context carved up) | Gradual degradation | No — all constraints degrade together | Removing any one helps all others |
| Interference (constraints contradict at margin) | Sudden failure at specific pairs | Yes — specific pairs fail while others hold | Removing one restores its partner only |
| Threshold collapse (phase transition) | Sharp cutoff at critical N | No — binary flip | Removing one below threshold restores all |

**Existing evidence leans toward interference/threshold:** Conflicts are 5/5 or 0/5 (never gradual), import-block conflicts are pair-specific and invisible at N=2. But this comes from agent-scaling experiments, not constraint-scaling.

**Needed experiment:** Vary co-resident constraints 1→20 on identical task. Measure degradation shape + pair-specificity. Investigation issued: **orch-go-sispn**.

## Q2: Does the model predict something untested?

All current evidence is retrodictive — the model explains observations we already had. No prediction has been generated and then tested.

**Candidate predictions to test:**
- Adding a second well-designed attractor to a system with one working attractor should create basin-boundary accumulation (code that fits neither cleanly piles up at the boundary)
- Two good design patterns in the same file should interfere in a specific, predictable way — not just dilute each other
- Attractor effectiveness should degrade with distance (semantic or structural) from the entry point

## Q3: Where does the model break?

Zero disconfirmation attempts. Every probe confirmed. This means either the model is very general or we haven't pushed hard enough.

**Candidate boundaries:**
- **Non-code domains:** Does attractor-gate pairing predict anything about knowledge organization beyond what we've measured? (Orphan rate is one data point)
- **Human teams:** Derived from AI agent coordination. Does it apply to how human engineers organize code? If not, that's a real boundary.
- **Emergent attractors:** All tested attractors are designer-created. What about attractors agents create spontaneously? (Thread: artifact-attractors)
- **Cross-cutting concerns:** A feature that genuinely belongs in two well-designed packages. Does the model predict which package wins, or does it break down?
- **Adversarial conditions:** Tasks that *require* violating the attractor's structure.

---

# Cross-Model References

This model consolidates findings that appear independently in:

| Model | What it contributes | What it borrows |
|---|---|---|
| **coordination** | Empirical trials (329), mechanism distinction, scaling behavior | — |
| **knowledge-accretion** | Orphan rate data, intervention hierarchy, external validation | Gate/attractor distinction from coordination |
| **harness-engineering** | Hard/soft harness framework, constraint dilution measurement | — |
| **architectural-enforcement** | Gate taxonomy (4 layers), bypass rate data | — |
| **completion-verification** | 14-gate verification architecture | Gate classification |
| **domain-harness-architecture** | Cross-project enforcement layering | Gate/attractor pairing |
| **system-learning-loop** | Reframes learning as attractor formation (RecurrenceThreshold=3) | Knowledge-accretion dynamics |

---

# Probe Log

| Date | Probe | Result | Source |
|---|---|---|---|
| 2026-03-22 | External framework validation | Confirmed: attractor-dominant frameworks succeed, gate-only fail | knowledge-accretion/probes/ |
| 2026-03-22 | Automated attractor discovery | Confirmed: 1 collision sufficient for constraint generation | coordination/probes/ |
| 2026-03-22 | Attractor decay resilience | Confirmed: 9/9 success with stale anchors | coordination/probes/ |
| 2026-03-23 | Agent scaling (N=4, N=6) | Found boundary: deterministic conflict when agents > regions | coordination/probes/ |
| 2026-03-23 | Modification tasks | Found boundary: self-coordinating tasks don't need attractors | coordination/probes/ |
| 2026-03-23 | Merge-educated messaging | Found ceiling: communication reaches ~30%, below placement 100% | coordination/probes/ |
| 2026-03-26 | Constraint scaling (orthogonal) | **NULL** — flat curve, no degradation N=1→20 with additive constraints. Measurement artifact in first run (BSD grep -P). | attractor-gate/probes/ |
| 2026-03-26 | Tension-based constraint redesign | **COMPLETE** — 12 tension pairs (HARD/MEDIUM/EASY), 48/48 detector tests pass, ready for experiment run | attractor-gate/investigations/ |
| 2026-03-27 | Multi-phase handoff gate coverage | Confirmed: boolean gate failed for multi-phase designs, structurally-informed gate catches the gap. Extends Hypothesis 4 with gate sub-hierarchy. | attractor-gate/probes/ |

## Auto-Linked Investigations

- .kb/investigations/2026-03-27-inv-fix-architect-handoff-gate-designs.md
- .kb/investigations/archived/misc-bug-fixes/2026-01-08-inv-bug-git-diff-gate-parses.md
- .kb/investigations/archived/2026-01-14-inv-implement-targeted-skip-gate-flags.md
- .kb/investigations/archived/epic-management-deprecated/2026-01-07-inv-epic-readiness-gate-understanding-section.md
- .kb/investigations/2026-03-22-inv-probe-validate-gate-attractor-mechanism.md
- .kb/investigations/archived/2026-01-04-inv-test-liveness-gate-fix-report.md
- .kb/investigations/archived/2026-01-17-inv-design-800-line-bloat-gate.md
- .kb/investigations/archived/2026-01-09-inv-add-test-first-gate-investigation.md
- .kb/investigations/archived/2026-01-13-inv-opus-gate-latest-status-still.md
- .kb/investigations/archived/2026-01-15-inv-verify-test-first-gate-implementation.md
- .kb/investigations/archived/2026-01-07-inv-gate-kb-reflect-surface-consolidation.md
- .kb/investigations/2026-03-11-design-artifact-sync-mechanism.md
- .kb/investigations/archived/2025-12-25-inv-enhance-orch-handoff-scaffold-gate.md
- .kb/investigations/2026-03-25-inv-design-discriminating-experiment-gate-attractor.md
- .kb/investigations/archived/2026-01-15-inv-verify-test-first-gate-already-exists.md
- .kb/investigations/archived/misc-bug-fixes/2026-01-08-inv-bug-test-evidence-gate-triggers.md
