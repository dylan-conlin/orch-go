<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Compliance vs coordination failure (§8) is one instance of a broader pattern — "compositional correctness gap" — where individually valid components compose into non-functional wholes, observable at three abstraction scales: operation→assembly (DFM), geometry→function (LED gates), agent→system (harness).

**Evidence:** LED gate probe (2026-03-20): 4-layer gate stack passes cut-channel LED routing that produces disconnected channels (valid manifold, non-functional enclosure). SendCutSend sheet metal DFM: individual operations (cuts, bends, hardware) validate independently but composed assembly can interfere. daemon.go +892: 30 correct commits compose into structural degradation.

**Knowledge:** The compositional correctness gap is a cross-domain failure mode class where every gate operates at the component level but no gate checks the composition property. This is not unique to agent coordination — it's a property of any system where validation occurs at a different abstraction level than function.

**Next:** Model updated. No further implementation needed.

**Authority:** implementation - Extends existing model section with new evidence and named concept, no architectural changes.

---

# Investigation: Extend Harness Engineering Model — Compositional Correctness Gap

**Question:** Can compliance vs coordination failure (§8 of harness-engineering model) be generalized as instances of a broader "compositional correctness gap" pattern, with LED gate stack and sheet metal DFM as cross-domain evidence?

**Started:** 2026-03-20
**Updated:** 2026-03-20
**Owner:** agent (led-3th)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Model:** harness-engineering

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/global/models/signal-to-design-loop/probes/2026-03-20-probe-domain-gate-coverage-gap-physical-cad.md` | extends | yes | - |
| `.kb/models/harness-engineering/model.md` §8 (Compliance vs Coordination) | extends | yes | - |
| `.kb/decisions/2026-01-14-verification-bottleneck-principle.md` | confirms | yes | - |

---

## Findings

### Finding 1: LED gate stack — geometry validates, function doesn't

**Evidence:** The LED magnetic letters probe (2026-03-20) tested two LED channel routing strategies across 26 letters with a 4-layer gate stack (parameter validation, geometry check, printability, intent alignment). The cut-channel approach (intersection of zigzag pattern with inner letter profile) produces:
- Layer 1: PASS — all parameters within range
- Layer 2: PASS — CGAL manifold, under polygon budget
- Layer 3: would PASS — valid solid geometry
- Layer 4: UNCERTAIN — depends on spec wording

But channels are disconnected for every non-rectangular letter. The LED strip has no continuous path. A completely non-functional design passes all geometric gates.

**Source:** `.kb/global/models/signal-to-design-loop/probes/2026-03-20-probe-domain-gate-coverage-gap-physical-cad.md` — ~150 OpenSCAD renders, 6 letter shapes tested

**Significance:** Gates validate at the geometry abstraction level. Function exists at the connectivity/routing abstraction level. No gate bridges the gap.

---

### Finding 2: Sheet metal DFM — operations validate independently, assembly interferes

**Evidence:** SendCutSend sheet metal DFM (Design for Manufacturability): individual manufacturing operations — cuts, bends, hardware insertions — each have well-defined validation rules (minimum bend radius, minimum hole-to-edge distance, minimum tab width). Each operation passes its own DFM check independently. But composed assembly can produce interference: a bend line crossing a hardware location, cut patterns that weaken a fold region, hardware placements that collide after bending.

**Source:** Dylan's direct domain knowledge from SendCutSend manufacturing operations. The SCS Fin bot analysis (2026-03-12 debrief) identified bend/DFM as the #1 knowledge gap (92 conversations).

**Significance:** This is the same structure as the LED gate failure: per-operation validation passes, but the composition of operations into a physical assembly reveals failures invisible to any single-operation gate.

---

### Finding 3: daemon.go — agents validate individually, system degrades

**Evidence:** daemon.go grew +892 lines in 60 days from 30 individually-correct commits. Each commit:
- Compiled (build gate: PASS)
- Was locally rational
- Passed review
- Added a real capability

But the aggregate: 6 cross-cutting concerns reimplemented across 4-9 files (~2,100 lines of duplicated infrastructure). No gate checked whether the composition of 30 agents' work was coherent.

**Source:** `.kb/investigations/2026-03-07-inv-analyze-accretion-pattern-orch-go.md`, model.md §8

**Significance:** The existing compliance vs coordination framing describes THIS case but doesn't recognize it as an instance of a broader pattern.

---

## Synthesis

**Key Insights:**

1. **Same structure, three scales.** All three cases follow identical structure: (1) individual components pass all applicable validation, (2) composed whole fails because no validation operates at the composition level, (3) the failure is invisible to every existing gate.

2. **Compositional correctness gap as named concept.** This failure mode class deserves a name because it clarifies the design target: gates must cover not just component properties but composition properties. The term "coordination failure" in §8 is accurate for agent systems but doesn't capture the cross-domain pattern (DFM, CAD, hardware design all exhibit the same structure without "coordination" being the right word).

3. **Abstraction scale progression.** The three cases sit at different abstraction scales, but the mechanism is identical:

| Scale | Components | Composition | Gap |
|-------|-----------|-------------|-----|
| Operation → Assembly (DFM) | Cuts, bends, hardware | Physical assembly | Inter-operation interference |
| Geometry → Function (LED) | Parameters, manifold, polygons | Functional design | Connectivity/routing |
| Agent → System (harness) | Individual agent commits | Codebase structure | Cross-agent coherence |

**Answer to Investigation Question:**

Yes. Compliance vs coordination failure is one instance of a compositional correctness gap pattern that manifests at multiple abstraction scales. The generalization is useful because it:
- Explains why the LED gate stack failure and the DFM failure have the same structure as daemon.go coordination failure
- Names the design target for gate systems: compose-level validation
- Predicts where similar gaps will appear in other domains: wherever validation gates operate at a different abstraction level than the functional requirement

---

## Structured Uncertainty

**What's tested:**

- ✅ LED gate stack passes geometry that is functionally broken (~150 renders, 6 letter shapes)
- ✅ daemon.go +892 from 30 correct commits (verified in model primary evidence)
- ✅ The three cases share identical structure (component passes, composition fails)

**What's untested:**

- ⚠️ DFM evidence is from domain knowledge, not from a specific instrumented experiment
- ⚠️ Whether "compositional correctness gap" predicts failures in domains beyond these three
- ⚠️ Whether compose-level gates are tractable to build (LED case suggests LLM vision might work; DFM has commercial tools)

**What would change this:**

- Finding a domain where component-level validation is sufficient (no compositional gap exists)
- Finding that the three cases have fundamentally different mechanisms despite surface similarity
- Evidence that stronger models close the gap without architectural intervention

---

## Implementation Recommendations

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Extend §8 of harness-engineering model with compositional correctness gap concept | implementation | Adds named concept and cross-domain evidence to existing section, no architectural change |

### Recommended Approach

**Extend §8 directly** — add compositional correctness gap as the generalization of compliance vs coordination, with three-scale evidence table and cross-domain examples.

---

## References

**Files Examined:**
- `.kb/models/harness-engineering/model.md` — Target model, §8 (Two Failure Modes)
- `.kb/global/models/signal-to-design-loop/probes/2026-03-20-probe-domain-gate-coverage-gap-physical-cad.md` — LED gate coverage gap evidence
- `.harness/openscad/SELF-MEASUREMENT-REPORT.md` — OpenSCAD harness context
- `.kb/models/entropy-spiral/model.md` — "Local correctness != global correctness" constraint
- `.kb/decisions/2026-01-14-verification-bottleneck-principle.md` — "All fixes were real. The failure was compositional."

**Related Artifacts:**
- **Investigation:** `.kb/global/models/signal-to-design-loop/probes/2026-03-20-probe-domain-gate-coverage-gap-physical-cad.md` — Primary LED evidence
- **Model:** `.kb/models/harness-engineering/model.md` — Target for extension

---

## Investigation History

**2026-03-20:** Investigation started
- Initial question: Can §8 (compliance vs coordination) be generalized as compositional correctness gap?
- Context: LED gate coverage gap probe + DFM domain knowledge converged on same pattern

**2026-03-20:** All three evidence sources reviewed, pattern confirmed, model updated
- Status: Complete
- Key outcome: §8 extended with compositional correctness gap concept and three-scale evidence
