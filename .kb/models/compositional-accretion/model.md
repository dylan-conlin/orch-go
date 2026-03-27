# Model: Compositional Accretion

**Domain:** Knowledge system design — conditions under which accumulation composes vs. piles up
**Last Updated:** 2026-03-27
**Validation Status:** EXTERNALLY VALIDATED — pattern confirmed independently across 5 domains (Luhmann, Popper, Kuhn, Lakatos, boundary objects) and quantitatively confirmed across 13 artifact types within orch-go. Core claims CA-01, CA-03, CA-04 hold. CA-02 extends: signal adoption rate mediates composition (opt-in signals plateau at 15-25%).
**Synthesized From:**
- Session conversation 2026-03-27 — cross-instance pattern recognition across 7 accretion surfaces
- `.kb/investigations/2026-03-26-design-brief-composition-layer.md` — tension sections as clustering key
- `.kb/models/knowledge-accretion/model.md` — parent model (degradation from correct contributions)
- `.kb/global/models/signal-to-design-loop.md` — five-stage loop where clustering is Stage 3
- `.kb/models/kb-reflect-cluster-hygiene/model.md` — defect-class metadata as structured clustering key

---

## Summary (30 seconds)

Accretion composes when each atom carries a signal about how it relates to other atoms — not just what it is. Atoms that point outward (at gaps, unresolved questions, incompleteness) self-organize into clusters. Atoms that point inward (at conclusions, summaries, resolved findings) pile up and demand manual triage. The principle: **knowledge composes at the edges of what it doesn't know.**

---

## Core Mechanism

### The Distinction: Outward-Pointing vs. Inward-Pointing Atoms

An **outward-pointing atom** carries information about its compositional role — what it doesn't know, what it relates to, where it's incomplete. These atoms compose because their edges recognize each other: two unresolved questions about the same gap cluster naturally.

An **inward-pointing atom** describes what it is and what it concluded. These atoms are self-contained. They don't reach for anything. They accumulate as independent objects requiring manual triage to organize.

### The Composition Signal

The specific feature that makes an atom outward-pointing varies by accretion surface, but the function is the same: it encodes **how this atom relates**, not just **what this atom is**.

| Accretion Surface | Count | Composition Signal | Adoption | Classification |
|---|---|---|---|---|
| Briefs (.kb/briefs/) | 73 | Tension section (required) | 100% | **COMPOSING** |
| Models (.kb/models/) | 45 | Claims table (testable) | ~100% | **COMPOSING** |
| Probes (.kb/models/*/probes/) | 306 | Model Impact section | 84% | **COMPOSING** (gap: claim/verdict at 18%) |
| Comprehension queue | — | Unread/processed state | 100% | **COMPOSING** |
| Threads (.kb/threads/) | 60 | Status lifecycle + resolved_to | 100% / 45% | **MIXED** |
| Decisions (.kb/decisions/) | 44 | Extends + linked investigations | 57% / 16% | **MIXED** |
| SYNTHESIS.md (.orch/workspace/) | 31 | UnexploredQuestions section | 87% | **MIXED** (improved) |
| Investigations (.kb/investigations/) | 1,219 | Model link + defect-class | 18% / 20% | **PILING UP** |
| Beads issues (.beads/) | 2,136 | Enrichment labels | 18% any routing | **PILING UP** |
| Session debriefs (.kb/sessions/) | 18 | None — conclusions only | N/A | **PILING UP** (low vol) |
| VERIFICATION_SPEC.yaml | 85 | None | N/A | **INERT** (correctly so) |
| Guides (.kb/guides/) | 37 | References section | 62% | **INERT** (low vol) |
| Digests (orch compose) | ~0 | Cluster + thread matching | By design | **COMPOSING** (untested) |

### Why Outward Beats Inward

Conclusions are inert. They don't reach for anything. Two conclusions about different topics sit next to each other without interacting.

Unresolved questions are connective. Two questions about the same gap cluster naturally because gaps recognize each other in a way that answers don't. The gap is the shared surface — and composition happens at surfaces.

---

## Why This Fails

### Failure Mode 1: Uniform Atoms (SYNTHESIS.md, Jan-Feb 2026)

Every worker produced a SYNTHESIS.md regardless of whether it learned anything. A typo fix generated the same artifact type as a major architecture investigation. Every synthesis demanded individual attention. The volume-to-signal ratio was terrible — not because there was too much volume, but because the atoms were inward-pointing (conclusions) with no compositional signal.

**Root cause:** The atom format had no structural feature that separated "I composed knowledge" from "I completed a task." Every atom looked the same.

### Failure Mode 2: Unenriched Backlog (35 orphan issues, ongoing)

Issues created without type, skill, area, or effort labels accumulate as undifferentiated work items. The daemon falls back to the coarsest routing (69% type-based inference). Each issue demands manual triage to route.

**Root cause:** The atom (issue) has no outward-pointing signal until enrichment adds one. The enrichment protocol is the mechanism that transforms an inward-pointing atom ("here's a thing") into an outward-pointing one ("here's a thing that needs *this kind* of attention").

### Failure Mode 3: Archived Investigations (85.5% orphan rate)

Investigations completed and archived without connecting to a model or decision. The finding is correct but has no compositional signal — it doesn't point at what it relates to.

**Root cause:** The investigation atom points inward (findings, conclusions) but not outward (which model does this extend? which decision does this inform?). The defect-class tag was added specifically to create this outward-pointing signal.

### Failure Mode 4: Low Signal Adoption (Investigations + Issues, 2026-03-27 audit)

Investigations have a defect-class field and a model link field. Both are outward-pointing signals. But defect-class adoption is 20% and model link adoption is 18%. At these rates, the surface behaves identically to one with no signal — 81.9% active orphan rate. Similarly, beads issues have enrichment labels, but only 18% ever get enriched.

**Root cause:** The signal is opt-in, not opt-out. Agents can complete the atom without filling the compositional signal. Only opt-out signals (like Brief Tension, which is required) achieve >80% adoption.

---

## Constraints

### Why can't we just add metadata to everything?

**Constraint:** The composition signal must be natural to the atom's creation, not a bureaucratic add-on. If agents must fill out a "clustering metadata form," they'll produce garbage metadata to satisfy the gate.

**Implication:** The best composition signals are ones the atom produces as part of its primary work. Tension sections work because agents naturally articulate what they don't know. Defect-class works because investigators naturally categorize the problem. Probe outcomes work because the probe's purpose is to test the model.

**This enables:** Composition that self-organizes without manual triage.
**This constrains:** We can't bolt composition onto surfaces where the atom has no natural outward-pointing feature. We'd need to redesign the atom format.

### Why does signal adoption vary so dramatically?

**Constraint:** A composition signal achieves high adoption only when filling it is opt-out (harder to skip than to fill). Opt-in signals plateau at 15-25% adoption regardless of template availability.

**Evidence (2026-03-27 audit):**
- Opt-out signals: Brief Tension (100%), issue_type (100%), probe Model Impact (84%)
- Opt-in signals: defect-class (20%), enrichment labels (18%), claim/verdict (18%), decision Extends (16%)

**Implication:** Designing a composition signal is necessary but not sufficient. The signal must be wired into the creation workflow such that skipping it is the exceptional path.

**This enables:** Prediction of which new signals will succeed — only those embedded in the creation workflow.
**This constrains:** Post-hoc enrichment pipelines will never achieve high adoption. The signal must be set at creation time.

### Why don't conclusions compose?

**Constraint:** A conclusion ("X is true") is a point in knowledge space. Two points don't interact unless something relates them. An unresolved question ("is X true?") is an edge — it connects to anything that might answer it.

**Implication:** Systems that accumulate conclusions need external composition mechanisms (summaries, indexes, search). Systems that accumulate questions get composition for free — the questions cluster on shared gaps.

**This enables:** Design criterion for new accretion surfaces: require the atom to name its incompleteness.
**This constrains:** "Done" is inert. Completion should not be the final state of an atom if we want it to compose. The most generative state is "partially resolved with named remaining questions."

---

## Design Criterion (The Constraint)

**Every accretion surface in the system should be evaluated by:**

> Does this artifact point outward or inward? Does it name what it doesn't know?

When designing a new artifact type or modifying an existing one, apply this test:

1. **What is the atom's composition signal?** If none exists, accumulation will pile up.
2. **Is the signal natural to creation?** If it requires extra work, it'll be gamed or skipped.
3. **Does the signal point outward?** If it only describes itself, it won't cluster with others.
4. **What is the signal's adoption rate, and is it opt-in or opt-out?** Only opt-out signals achieve >80% adoption. A signal at 20% adoption is structurally equivalent to no signal.

If any answer is "no," the surface will accumulate but not compose. Either redesign the atom or accept that this surface requires manual triage.

**Not all surfaces need to compose.** Artifacts that serve gate functions (VERIFICATION_SPEC.yaml) or reference functions (guides) are correctly inert. The criterion applies to knowledge-accumulating surfaces, not to every artifact in the system.

---

## External Validation

The inward/outward distinction appears independently across multiple knowledge traditions. The model's novel contribution is not the observation — it's the general design criterion that can be applied to any accretion surface.

### What Others Call It

| Tradition | Their Vocabulary | Model Equivalent |
|---|---|---|
| Luhmann (Zettelkasten) | "Communication via surprise," relational incompleteness | Outward-pointing atoms compose via shared surfaces |
| Popper (Falsificationism) | "Conjectures and refutations," growth through error | Knowledge grows at gaps (refutations), not confirmations |
| Kuhn (Paradigm Shifts) | "Anomalies," crisis as engine of revolution | Gaps accumulate past threshold → composition into new paradigm |
| Lakatos (Research Programmes) | "Positive heuristic," ocean of anomalies | Known gaps drive forward progress |
| Star & Griesemer (Boundary Objects) | "Weakly structured in common use" | Incompleteness enables cross-community composition |
| Matuschak (Evergreen Notes) | "Transient" vs "evergreen," "accrete" | Inward-pointing = transient, outward-pointing = evergreen |
| Milo (Active Ideation) | "Collector's Fallacy," under-thinking | Inward-pointing atoms cause pile-up |
| KG/Ontology Literature | Open World vs Closed World Assumption | CWA = can't represent gaps; OWA = gaps exist but aren't compositional |

### What the Model Adds

None of these traditions name the inward/outward distinction as a **general design test**: "Does this atom point outward? Is the signal natural to creation? Will it cluster without manual triage?" The model converts an observed phenomenon into an actionable design criterion for building knowledge systems.

---

## Relationship to Parent Models

**Knowledge Accretion** describes the *problem* — shared artifacts degrade from correct contributions when coordination is absent. This model describes one *solution mechanism* — how to design atoms that self-organize instead of requiring external coordination.

**Signal-to-Design Loop** describes the *process* — five stages from signal capture through design response. This model explains why Stage 3 (clustering) works for some signals and not others: signals with outward-pointing composition signals cluster; signals without them don't.

**Self-Disconfirming Knowledge** thread describes a *consequence* — if atoms point at what they don't know, the system naturally pressures its own models. Outward-pointing atoms create disconfirmation pressure; inward-pointing atoms create confirmation bias.

---

## Evolution

**2026-03-27:** Model created from cross-instance pattern recognition. Seven instances identified in a single conversation when asking "why would brief composition work when SYNTHESIS.md didn't?"

**2026-03-27:** CA-05 externally validated via research probe. The inward/outward distinction appears independently in Luhmann (1981), Popper (1934/1963), Kuhn (1962), Lakatos (1970), Star & Griesemer (1989), and the PKM community (2020s). The model's novel contribution is the general design criterion — nobody else names it as a test applicable to any accretion surface. Three extensions identified: (1) the vocabulary gap is the contribution, (2) bibliometrics as testable prediction, (3) CWA/OWA maps to inward/outward.

**2026-03-27 (audit):** Full audit of 13 artifact types confirmed CA-01, CA-03, CA-04. Extended CA-02 with signal adoption rate finding: opt-in signals plateau at 15-25% adoption regardless of template availability. Added Failure Mode 4 (low signal adoption), new constraint (adoption varies by opt-in/opt-out), 4th design criterion question. Expanded table from 7 to 13 surfaces with measured adoption rates and classifications. Added CA-06.

---

## Claims (Testable)

| ID | Claim | How to Verify |
|----|-------|---------------|
| CA-01 | Atoms with outward-pointing signals (tensions, defect-class, probe outcomes) cluster more effectively than atoms without | Compare clustering quality of briefs-with-tensions vs briefs-without-tensions in `orch compose` output |
| CA-02 | Adding a composition signal to an existing surface reduces its orphan/pile-up rate — **but only if adoption exceeds ~80%** | Measure issue orphan rate before vs after enrichment protocol enforcement. Also measure defect-class adoption rate vs investigation orphan rate |
| CA-03 | Composition signals that are natural to atom creation produce higher-quality clustering than bolted-on metadata | Compare defect-class usage (natural) vs hypothetical mandatory-tag usage (bureaucratic) |
| CA-04 | Systems that accumulate conclusions require more manual triage than systems that accumulate questions | Compare triage load on SYNTHESIS.md archive vs tension-harvested briefs |
| CA-05 | The principle holds outside orch-go | **Confirmed qualitatively.** Luhmann's Zettelkasten, Popper's falsificationism, Kuhn's anomaly-driven paradigm shifts, Lakatos's positive heuristic, boundary objects, and PKM practitioner literature all exhibit the same pattern. Quantitative measurement of composition rates remains untested externally. See probe: `probes/2026-03-27-probe-external-validation-luhmann-pkm-epistemology.md` |
| CA-06 | Only opt-out signals achieve >80% adoption; opt-in signals plateau at 15-25% | Measure adoption rates when investigation model-link is changed from opt-in to opt-out (--model required, --orphan escape hatch) |

---

## References

**Investigations:**
- `.kb/investigations/2026-03-26-design-brief-composition-layer.md` — Tension sections as clustering key, digest artifact design
- `.kb/investigations/2026-03-26-inv-design-enrichment-pipeline-daemon-spawned.md` — Enrichment as composition signal for issues

**Related Models:**
- `.kb/models/knowledge-accretion/model.md` — Parent model (the problem this helps solve)
- `.kb/global/models/signal-to-design-loop.md` — The process framework (this explains why Stage 3 works selectively)
- `.kb/models/kb-reflect-cluster-hygiene/model.md` — First successful instance (defect-class as clustering key)

**Probes:**
- `probes/2026-03-27-probe-external-validation-luhmann-pkm-epistemology.md` — CA-05 confirmed qualitatively across 5 domains; model's novel contribution is the general design criterion
- `probes/2026-03-27-probe-artifact-type-audit-design-criterion.md` — CA-01, CA-03, CA-04 confirmed across 13 artifact types. CA-02 extended: signal adoption rate mediates composition. Added CA-06 (opt-in vs opt-out adoption threshold).

**Threads:**
- `.kb/threads/2026-03-27-knowledge-composes-edges-doesn-t.md` — The originating insight
- `.kb/threads/2026-03-26-every-spawn-composes-knowledge-task.md` — Related: asks *whether* spawns compose; this model answers *how*
- `.kb/threads/2026-03-26-self-disconfirming-knowledge-system-that.md` — Consequence: outward-pointing atoms create disconfirmation pressure

**External Parallels (from probe):**
- Luhmann, "Kommunikation mit Zettelkästen" (1981) — notes compose via relational incompleteness
- Popper, *Conjectures and Refutations* (1963) — knowledge grows through refutation (gaps), not confirmation
- Kuhn, *Structure of Scientific Revolutions* (1962) — paradigm shifts happen at anomalies (gaps)
- Lakatos, *Methodology of Scientific Research Programmes* (1970) — "positive heuristic" is a set of known gaps driving progress
- Star & Griesemer, "Boundary Objects" (1989) — weak structuring enables cross-community composition
- Matuschak, "Knowledge work should accrete" — transient vs evergreen = inward vs outward
- Milo, "Collector's Fallacy" critique — accumulation without composition as failure mode
