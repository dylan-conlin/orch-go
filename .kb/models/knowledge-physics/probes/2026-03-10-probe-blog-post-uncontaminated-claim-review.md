# Probe: Blog Post Uncontaminated Claim Review

**Model:** knowledge-physics
**Date:** 2026-03-10
**Status:** Complete

---

## Question

Do the published blog posts (Harness Engineering, Knowledge Physics, Coordination Demo) contain claims that assume the theory is validated or novel without sufficient evidence? Does language overclaim beyond what the data supports?

---

## What I Tested

Read all three blog posts as an external reader with no prior exposure to the knowledge-physics or harness-engineering model framing. Systematically flagged every claim against three categories:
1. **Validation assumptions** — claims that treat internal testing as proof
2. **Novelty assumptions** — claims of absence from literature without systematic review
3. **Overclaimed language** — language that goes beyond what the evidence demonstrates

Posts reviewed:
- `.kb/publications/harness-engineering-draft.md` (309 lines)
- `.kb/publications/knowledge-physics-draft.md` (253 lines)
- `.kb/publications/coordination-failure-demo-post.md` (221 lines)

---

## What I Observed

### Category 1: Claims Assuming Theory is Validated

| Post | Line | Claim | Problem |
|------|------|-------|---------|
| Knowledge Physics | 3 | "applicable to any substrate where amnesiac agents contribute" | Universality claim from 2 substrates in 1 system operated by 1 person |
| Knowledge Physics | 31 | "kept surviving every attempt to break it" | All "attempts to break it" were by agents operating inside the framework. Closed-loop validation. |
| Knowledge Physics | 39 | "Five conditions predict whether a shared system will degrade" | Never tested predictively — fit to historical data. "Predict" implies forward-looking validation that hasn't occurred. |
| Knowledge Physics | 71 | "these conditions produce accretion in any shared mutable substrate" | "any" is a universality claim from N=2 substrates |
| Knowledge Physics | 89 | "No clean counterexample survived" | Every counterexample was resolved by the same framework that generated the theory. Could be retrofitting — categories broad enough to absorb any evidence. |
| Knowledge Physics | 113 | "Accretion is measurable" → orphan rate drop evidence | Treating internal metrics from one system as validation of a general theory. Simpler explanation available: "adding structure to an unstructured system produces structure." |
| Harness Engineering | 64 | "This makes harness engineering a permanent discipline, not a transitional one" | Requires the premise that coordination worsens with model improvement, which is under-evidenced |
| Harness Engineering | 195 | "tested the framework against a TypeScript codebase and found it language-independent" | N=2 (Go + TypeScript) doesn't establish language-independence |

### Category 2: Claims Assuming Novelty Without Evidence

| Post | Line | Claim | Problem |
|------|------|-------|---------|
| Harness Engineering | 217 | "absent from the published literature" | Novelty claim requires systematic literature review. Post cites 4 references. |
| Harness Engineering | 293 | "The field is converging on 'harness' as a concept but hasn't yet grappled with the coordination problem" | Claims knowledge of the entire field's state. |
| Knowledge Physics | 37 | "The closest comparison is Elinor Ostrom's institutional analysis of commons governance" | Comparing a 3-month, 1-person study to decades of multi-country empirical commons research elevates the work's status by association. |
| Coordination Demo | 128 | "That's a compliance answer to a coordination question" | Dismisses MAST's proposed solution without testing whether "social reasoning" (agents aware of each other's work) would actually help — which is arguably what sequential execution does. |

### Category 3: Overclaimed Language

| Post | Location | Language | Issue |
|------|----------|----------|-------|
| Knowledge Physics | Title | "Knowledge Physics" | Name implies law-like, substrate-independent universality. Honest Gaps section (line 185) says "Not a law of physics" but the name and rhetorical treatment contradict that hedging. |
| Knowledge Physics | 14 | "thermodynamic tendency" (implicit via "accretion" definition) | Physics language for a software/organizational pattern |
| Knowledge Physics | 51-59 | "four dynamics emerge — regardless of what the substrate is made of" | "regardless of substrate" from 2 substrates in 1 system |
| Knowledge Physics | 143-144 | "Greater capability created greater divergence" | Stated as general finding from N=1 complex task trial |
| Knowledge Physics | 167 | "Every one of these cases meets the five conditions. Every one exhibits the predicted dynamics." | Every external case is post-hoc classified, not predicted. Framework explains everything = explains nothing risk. |
| Harness Engineering | 14 | "thermodynamic tendency of multi-agent codebases toward entropy" | Metaphor presented as mechanism. Actual thermodynamics has well-defined entropy measures; code complexity doesn't. |
| Harness Engineering | 15 | "faster, more capable agents accrete more code per session with higher confidence" | Stated as fact. The merge experiment shows equal conflict rates, not worse ones. The daemon.go data doesn't isolate model capability as a variable. |
| Harness Engineering | 297 | "The physics appear to be substrate-independent" | "Physics" for pattern observations across 2 substrates |
| Coordination Demo | 214 | "The physics appear to be substrate-independent" | Same phrase, same problem |
| Harness Engineering | 95-101 | "265 contrastive trials across 7 agent skills" | Number cited but methodology never explained. Reader cannot evaluate what constitutes a "trial" or how the +5/+2-7/inert measurements were derived. |
| Harness Engineering | 103 | "Approximately 83 [constraints] were non-functional" | Very specific number with no explanation of how "non-functional" was measured or what threshold was used |

### Structural Observations

**Hedging is present but misplaced.** The Knowledge Physics post has an "Honest Gaps" section (lines 219-229) that acknowledges "One system, one operator" and "Two confirmed substrates." But this section comes after 200+ lines of confident claims. The framing effect: readers absorb the confident claims in the body and skim the caveats at the end. The coordination demo post handles this better — "I think is underappreciated" (line 119) hedges inline.

**The Ostrom comparison cuts both ways.** Ostrom studied hundreds of commons across dozens of countries over decades. This is one system, one operator, three months. The comparison is apt in type (structural conditions predicting outcomes) but not in evidence base. Readers familiar with Ostrom may see the comparison as claiming comparable rigor.

**"The physics" as recurring overclaim.** The phrase "the physics appear to be substrate-independent" appears identically in two of the three posts (Harness Engineering line 297, Coordination Demo line 214). This phrase does three overclaiming things simultaneously: (1) "physics" elevates pattern observations to law status, (2) "substrate-independent" claims universality from N=2, (3) "appear to be" is technically hedged but in practice reads as confident assertion.

**The 265-trial claim is the most concerning methodological gap.** The Harness Engineering post's claim about 265 contrastive trials across 7 skills is the backbone of the "soft instructions fail" argument. But the methodology is never described: What was the control condition? What was measured? How were "contrastive trials" defined? Were these controlled experiments or operational observations retroactively analyzed? The coordination demo post provides full methodology for its N=10 experiment. The harness post does not for its N=265 claim.

**The compliance/coordination distinction may not be novel.** The claim that this distinction is "absent from the published literature" (Harness Engineering line 217) requires a systematic literature review that isn't evident. Conway's Law (1967) addresses structural determinism. Brooks's Law (1975) addresses coordination cost scaling. The distributed systems literature on consistency/availability tradeoffs addresses coordination at the infrastructure level. The distinction between agents following rules correctly but producing bad collective outcomes is well-known in multi-agent systems research, mechanism design, and game theory (e.g., tragedy of the commons, Arrow's impossibility theorem). What may be new is applying it specifically to AI coding agents.

**Simpler explanations exist for most observations.** The knowledge system's orphan rate dropped from 94.7% to 52% after probes were introduced. The theory frames this as "structural attractor reducing accretion." The simpler explanation: "categorizing things reduces uncategorized things." The daemon.go growth is framed as "accretion in a multi-agent system." Simpler: "code grows when you keep adding features to a monolith." The theory may be correct, but the evidence doesn't exclude simpler explanations.

---

## Model Impact

- [x] **Extends** model with: Publication readiness assessment — specific overclaimed language and validation assumptions that need addressing before the publication gate can be satisfied

### Specific Recommendations

1. **Rename or justify "Knowledge Physics."** Either scope the name to acknowledge it's aspirational, or provide evidence commensurate with the claim. "Knowledge Coordination Patterns" or "Accretion Dynamics" would be more proportionate to the evidence.

2. **Move hedging inline.** Don't bury "one system, one operator" in a late section. Include caveats near the claims they qualify. "In our system..." instead of "in any substrate..."

3. **Explain the 265-trial methodology.** Add a section or footnote describing what constitutes a contrastive trial, what was measured, and how controls were defined. Without this, the number is unverifiable.

4. **Weaken universality claims.** Replace "any substrate" → "the substrates we've tested." Replace "regardless of what the substrate is made of" → "in both substrates we've measured." Replace "The physics appear to be substrate-independent" → "We observe similar patterns in both code and knowledge substrates."

5. **Soften the Ostrom comparison.** Acknowledge the evidence gap: "Ostrom studied hundreds of commons over decades; we've studied one system for three months. The comparison is structural, not evidentiary."

6. **Address simpler explanations.** For key findings, explicitly state the simpler explanation and argue why the five-condition framework adds explanatory power beyond it.

7. **Remove or qualify "physics" language.** "Thermodynamic tendency" → "tendency." "Entropy" when used technically should be defined. If it's a metaphor, say so.

8. **The "better models make it worse" claim needs evidence or weakening.** The merge experiment shows equal conflict rates (not worse). The daemon.go evidence doesn't isolate model capability. Either produce evidence of the directional claim or weaken to "model improvement doesn't help coordination."

9. **Do a systematic literature search** before claiming the compliance/coordination distinction is absent from published work. Multi-agent systems, mechanism design, and organizational theory may have addressed this.

---

## Notes

This probe was conducted as an adversarial external review — reading the posts as a reader with no exposure to the internal models would. The posts are well-written and the evidence base (when described) is genuine. The core observation — that coordination failure is distinct from compliance failure — is valuable regardless of whether it's novel. The main risk is that overclaimed language and validation assumptions undermine credibility with readers who would otherwise find the framework useful.

The coordination demo post is the strongest of the three — it has clear methodology, reproducible experiment, and proportionate claims. The knowledge physics post is the most overclaimed — the gap between evidence (1 system, 1 person, 3 months) and claim scope (universal "physics") is the largest. The harness engineering post falls in between — solid operational evidence with some unjustified generalizations.
