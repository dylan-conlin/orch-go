## Summary (D.E.K.N.)

**Delta:** Publish the empirical knowledge system — the investigation/probe/model cycle as the core product, with knowledge physics as the theory and harness engineering as evidence. Path B: we're building a system that compounds understanding, not agent tooling.

**Evidence:** Knowledge physics model (32nd model describing the other 31), harness engineering model (code substrate), 1,166 investigations measured, 85.5%→52% orphan rate with probes, 265 contrastive trials, daemon.go +892 lines (coordination failure), 3 entropy spirals, 1,625 lost commits, compliance vs coordination distinction.

**Knowledge:** The investigation/probe/model cycle is the automated scientific method. The system's unique value is compounding understanding across amnesiac contributors — not agent coordination. Path B chosen 2026-03-09: empirical knowledge system, not AI agent tooling.

**Next:** First hard knowledge gate experiment (Phase 1), coordination failure controlled demo (Phase 1), three strategic investigations (minimal kb substrate, first user profile, human probes), then rewrite publication with Path B framing (Phase 2).

---

# Plan: Knowledge Physics Publication

**Date:** 2026-03-08 (reframed 2026-03-09)
**Status:** Active
**Owner:** Dylan

**Extracted-From:** Harness engineering plan (complete), knowledge-physics model, thread "knowledge-physics-does-knowledge-have", thread "harness-engineering-as-strategic-position"
**Supersedes:** Original harness-publication plan (code-only framing)
**Superseded-By:** N/A

---

## Objective

Publish the empirical knowledge system and the theory behind it. The system automates the scientific method: investigations observe, probes test hypotheses against models, models formalize understanding, and the cycle compounds. The 32nd model the system produced describes the physics of the other 31.

The publication serves two purposes:
1. **Establish the theory** (knowledge physics) with unassailable dual-substrate evidence
2. **Introduce the product** (the investigation/probe/model cycle) as something others can run

Path B (decided 2026-03-09): We are not building agent tooling. We are building an empirical knowledge system that happens to use AI agents as investigators. The ideal domain is any organization where institutional amnesia is expensive — regulated R&D, defense, finance, high-turnover teams.

Success = publication that makes people want to run the system, not just read about the theory. Three strategic questions answered (minimal substrate, first user, human probes). Two experiments complete.

---

## Substrate Consulted

- **Models:** `knowledge-physics` (substrate-independent dynamics, dual evidence), `harness-engineering` (code substrate instance, 5 layers), `system-learning-loop` (proto-knowledge-physics), `skill-content-transfer` (attention primers vs action directives)
- **Decisions:** Three-layer hotspot enforcement (2026-02-26), harness plan phases 1-6 (complete)
- **Threads:** "Knowledge physics — does knowledge have the same accretion/attractor/gate dynamics as code?", "Harness engineering as strategic position"
- **Constraints:** Coordination failure demo needed before claiming "stronger models need more coordination gates." First hard knowledge gate needed to show the physics predict interventions, not just describe patterns.

---

## Phases

### Phase 1: Two Experiments (parallelizable)

**Goal:** Produce the two pieces of evidence that make the publication unassailable.

**Experiment A: First Hard Knowledge Gate** (`orch-go-vfd6v`)
- Implement `kb create investigation --model X` as required flag (or explicit `--orphan` to opt out)
- Measure orphan rate before/after over 2-4 weeks
- If adding structural coupling drops orphan rate (as probes did: 94.7%→52%), we have causal evidence that the physics predict interventions
- This is also the answer to knowledge-physics open question #6

**Experiment B: Coordination Failure Demo** (`orch-go-qrfhe`)
- Same task, same codebase, Haiku vs Opus (or similar capability gap)
- Measure: do more capable agents produce more coordination failures (duplication, cross-cutting reimplementation)?
- Even a small controlled demo makes the "stronger models need more coordination gates" claim empirical rather than logical

**Also continue (from prior plan):**
- Governance health metric (`orch-go-ycdbr`)
- Escape hatch tracking (`orch-go-v892h`)
- 30-day accretion trajectory (`orch-go-1ittt`)

**Exit criteria:** Both experiments complete with documented results; governance metrics reporting.
**Depends on:** Nothing — can start immediately.

### Phase 2: Publication Draft (rewrite)

**Goal:** Rewrite the publication with substrate-independence as the frame.

**Structure (Path B reframe):**
1. **The meta-story** (hook) — One person built a system where AI agents investigate, test hypotheses, and formalize understanding. The knowledge compounds instead of evaporating. The 32nd model describes the physics of the other 31.
2. **The problem** — Institutional amnesia. Organizations re-learn things they already know because no contributor remembers what happened before. This costs billions in regulated industries. AI agents have the same problem at 100x speed.
3. **The system** — Investigation/probe/model cycle as automated scientific method. How it works, what it produces, why knowledge compounds instead of accreting.
4. **The theory** — Knowledge physics. Four conditions → substrate-independent dynamics. Why this applies to any shared mutable substrate with amnesiac contributors.
5. **Evidence: Code substrate** — daemon.go +892, 265 contrastive trials, 3 entropy spirals, hard/soft harness, compliance vs coordination
6. **Evidence: Knowledge substrate** — 1,166 investigations, orphan rate trajectory, model behaviors, first hard gate experiment results
7. **The sharp claim** — Stronger models/contributors need more coordination infrastructure, not less. The system is permanent, not transitional.
8. **Running it yourself** — What's the minimal substrate? How do you start? (Informed by Phase 1 strategic investigations)
9. **Honest gaps** — What we don't know yet

**Exit criteria:** Draft complete, evidence claims traceable, reviewed by Dylan.
**Depends on:** Phase 1 (experiment results needed for sections 4 and 5).
**Issues:** `orch-go-ap2jw`

### Phase 3: Standalone KB System

**Goal:** Extract the investigation/probe/model cycle into something others can run.
**Strategic investigations (inform design):**
- Minimal kb substrate — what's needed without orch? (`orch-go-hrgor`)
- First external user profile — who, what do they need? (`orch-go-j2ziz`)
- Can humans run probes without AI agents? (`orch-go-5j2cq`)
**Deliverables:**
- Standalone kb system (CLI or library) that runs the investigation/probe/model cycle
- Works without orch, beads, or the full agent stack
- First external user running it
**Exit criteria:** Someone outside Dylan's system produces their first model from investigations.
**Depends on:** Phase 2 (publication creates demand), strategic investigations (inform scope).
**Issues:** `orch-go-hrgor`, `orch-go-j2ziz`, `orch-go-5j2cq`

### Phase 4: Cross-Substrate Validation

**Goal:** Demonstrate the system working on a non-code, non-knowledge substrate. Regulated R&D, documentation, or database schemas.
**Deliverables:**
- Run kb system on a new domain for 4+ weeks
- Measure: do the same dynamics emerge? (accretion, attractors, gates needed)
- Document substrate-specific adaptations
**Exit criteria:** Third substrate confirms or refutes generalization.
**Depends on:** Phase 3 (standalone system exists).
**Issues:** `orch-go-xi1tk` (repurposed from cross-language to cross-substrate)

---

## Decision Points

### Decision 1: Publication format

**Context:** Blog post vs paper vs both.

**Options:**
- **A: Long-form blog post** — Accessible, shareable. Fast. Wide reach.
- **B: Technical paper** — Formal, durable. Academic reach.
- **C: Blog post first, paper later** — Blog captures position, paper follows with more data.

**Recommendation:** C — the meta-story and compliance/coordination distinction are blog-native. The substrate generalization and empirical measurements are paper-native. Do both, blog first.

**Status:** Open

### Decision 2: Portable tooling scope

**Status:** Deferred until Phase 2 clarifies the framework's final shape.

---

## Structured Uncertainty

**What's tested:**
- Code substrate dynamics: accretion, attractors, gates (daemon.go, spawn_cmd.go, 265 trials)
- Knowledge substrate dynamics: orphan rate, model behaviors, gate deficit (1,166 investigations)
- Probe system as structural coupling fix (94.7%→52% orphan rate)
- Soft instructions dilute under pressure (265 contrastive trials)
- System-learning-loop is proto-knowledge-physics (mapping confirmed)

**What's untested:**
- Whether adding hard knowledge gates reduces orphan rate further (Experiment A)
- Whether stronger models produce more coordination failures (Experiment B)
- Whether 5-layer harness bends 30-day code trajectory
- Cross-language portability in practice (dry-run only so far)
- Soft harness budget curve shape

**What would change this plan:**
- Experiment A shows no orphan rate improvement → weaken "physics predict interventions" claim
- Experiment B shows no capability→coordination correlation → reframe as logical argument, not empirical
- Someone publishes substrate-independence framing first → accelerate Phase 2
- 30-day code trajectory shows no improvement → honest negative evidence (strengthens credibility per publication probe)

---

## Success Criteria

**Phase 1: Experiments + Strategic Questions**
- [ ] First hard knowledge gate shipped and measured (`orch-go-vfd6v`)
- [ ] Coordination failure controlled demo complete (`orch-go-qrfhe`)
- [ ] Governance health metric reporting (`orch-go-ycdbr`)
- [ ] Escape hatch tracking live (`orch-go-v892h`)
- [ ] 30-day accretion trajectory measured (`orch-go-1ittt`)
- [ ] Minimal kb substrate defined (`orch-go-hrgor`)
- [ ] First user profile identified (`orch-go-j2ziz`)
- [ ] Human probes feasibility answered (`orch-go-5j2cq`)

**Phase 2: Publication**
- [ ] Publication draft rewritten with Path B framing (`orch-go-ap2jw`)

**Phase 3: Standalone System**
- [ ] KB system works without orch stack
- [ ] First external user produces a model

**Phase 4: Cross-Substrate**
- [ ] Third substrate validates or refutes generalization (`orch-go-xi1tk`)
