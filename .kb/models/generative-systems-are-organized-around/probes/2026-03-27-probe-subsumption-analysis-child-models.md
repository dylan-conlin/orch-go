# Probe: Subsumption Analysis — Does Named Incompleteness Subsume Three Child Models?

**Model:** generative-systems-are-organized-around
**Date:** 2026-03-27
**Status:** Complete
**claim:** NI-04
**verdict:** confirms (with qualification)

---

## Question

NI-04 claims named incompleteness subsumes compositional-accretion, attractor-gate, and knowledge-accretion as instances. "Subsumes" is a strong claim. The precise test: does NI make a prediction that each child model also makes, for the same reason? And do the child models have independent predictions that NI doesn't cover?

---

## What I Tested

Read all three parent models in full (compositional-accretion model.md, attractor-gate model.md, knowledge-accretion model.md + claims.yaml) and systematically checked whether NI's core prediction — "named gaps compose because gaps are specific coordinates in possibility space; unnamed completeness doesn't compose because conclusions are generic" — generates the same core prediction in each child model.

For each child model, mapped:
1. The child model's core prediction
2. NI's prediction in the same domain
3. Whether the reason is the same (information-theoretic, not just analogical)
4. Which child predictions NI doesn't cover (independent predictions)

---

## What I Observed

### Compositional Accretion

**CA core prediction:** Atoms with outward-pointing signals (tensions, defect-class, probe outcomes) compose; atoms with inward-pointing signals (conclusions, summaries) pile up.

**NI's prediction in this domain:** Atoms that carry named incompleteness (gaps, questions, what-they-don't-know) compose; atoms without named incompleteness don't.

**Same reason?** YES. "Outward-pointing" IS "carries named incompleteness." The composition signal IS the named gap. A brief's tension section names what the brief doesn't resolve. A probe's model-impact section names what changed about the model's understanding. A defect-class tag names which category of unsolved problem the investigation addresses. All of these are named gaps — specific coordinates in possibility space.

**CA-06 (opt-in/opt-out adoption):** Initially appears independent, but reduces to: how effectively does the system ensure atoms NAME their incompleteness? Opt-in = naming is optional, so 75-85% of atoms don't name their gaps (15-25% adoption). Opt-out = atoms must name gaps by default, achieving >80% adoption. The adoption rate IS the rate at which incompleteness gets named. NI covers this.

**Independent CA predictions NI doesn't cover:**
- The specific 4-question design criterion (operational how-to)
- The 13-surface audit with measured adoption rates and classifications (empirical data)
- The specific composition signal vocabulary per accretion surface (domain-specific instances)

### Attractor-Gate

**AG core prediction:** Attractors embed coordination at design time; gates enforce at runtime. Only the pair holds.

**NI's prediction in this domain:** An attractor IS a named gap — `pkg/spawn/backends/` says "spawn code should go here, but the new feature doesn't exist yet." A gate alone is a closed assertion — "this file must not exceed N lines." Attractors work because they preserve named incompleteness (the destination is unfilled). Gates fail alone because they close paths without naming alternatives — they say "not here" without saying "where."

**Same reason?** PARTIALLY. NI explains WHY attractors work better than gates (attractors are named gaps; gates are unnamed closures). But the attractor-gate model's deeper mechanism — design-time embedding vs runtime decision — adds a dimension NI doesn't fully explain. NI says the gap must be named; AG says the gap must be named AT THE RIGHT TIME (design time, not runtime). The timing dimension is independent.

**Independent AG predictions NI doesn't cover:**
- Claim 4: Conflicts under competing attractors are deterministic, not stochastic — this is geometry (pigeonhole), not naming
- Claim 5: Attractors tolerate stale anchors — resilience via region separation, not gap naming
- Hypothesis 2: Modification tasks are immune — structural self-coordination, a boundary condition
- Hypothesis 3: Automated attractor discovery — mechanism for creating new named gaps from collision data
- The constraint scaling experiments (orthogonal vs tensioned constraints)

### Knowledge Accretion

**KA core prediction:** Shared artifacts degrade from correct contributions when agents lack coordination (five conditions).

**NI's prediction in this domain:** Degradation occurs when atoms don't carry named incompleteness. Without named gaps, contributions have no compositional signal, so locally correct work piles up instead of composing. Condition 5 ("no structural coordination mechanism") is equivalent to "no mechanism for naming and preserving gaps."

**Same reason?** YES for the core claim. The five conditions describe the context where named incompleteness is absent. NI provides the WHY behind KA-01: contributions degrade because they're complete-unto-themselves (inward-pointing, no named gaps), so they can't compose.

**Independent KA predictions NI doesn't cover:**
- KA-09: Creation/removal asymmetry (thermodynamic ratchet, not about naming)
- KA-10: Anti-accretion mechanisms themselves accrete (meta-coordination, NI doesn't predict recursive application)
- KA-14: Digital substrates lack environmental coordination (substrate properties, not gap properties)
- The detailed substrate taxonomy (10+ substrates catalogued with specific signatures)
- The intervention effectiveness hierarchy (structural > signaling > gates > advisory > metrics)
- The entropy metrics framework (6+ concrete metrics)
- KA-13: Measurement-improvement bias (epistemology of metrics, not gap dynamics)

---

## Model Impact

- [x] **Confirms** NI-04 with qualification: Named incompleteness subsumes the CORE PREDICTION of each child model for the same information-theoretic reason (gaps are specific coordinates → convergence; conclusions are generic → pile-up). The subsumption is at the level of "why" — the unifying reason.

- [x] **Extends** with a precise subsumption definition: NI is a **generative principle** that produces the same core prediction across domains. The child models are **domain instances** that add:
  - **Operational mechanics** (CA: design criteria, adoption rates; AG: timing of embedding; KA: intervention hierarchy)
  - **Boundary conditions** (AG: modification tasks immune; AG: deterministic conflicts; KA: creation/removal asymmetry)
  - **Measurement frameworks** (CA: 13-surface audit; KA: 6+ entropy metrics; AG: 329-trial coordination data)

  The relationship is: NI provides the WHY. Child models provide the HOW, WHAT, and WHEN. NI does not replace the child models — it unifies them.

- [ ] **Does not contradict** any child model claim. No case found where NI predicts the opposite of what a child model predicts.
