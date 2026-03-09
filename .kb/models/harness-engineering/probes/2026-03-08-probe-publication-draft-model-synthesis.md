# Probe: Publication Draft — Model Synthesis and Evidence Traceability

**Model:** harness-engineering
**Date:** 2026-03-08
**Status:** Complete

---

## Question

Can the harness engineering model's claims be synthesized into a practitioner-facing publication with every evidence claim traceable to source data? Does the synthesis process reveal any claims that lack sufficient evidence or contradict the data?

Testing whether the model's 7 invariants, 7 failure modes, 5 layers, and cross-language portability claims hold up under the pressure of writing for an external audience.

---

## What I Tested

1. **Evidence traceability:** For each claim in the publication draft, verified the source data exists in probes, investigations, or codebase measurements
2. **Internal consistency:** Cross-referenced the model's claims against the two Mar 8 probes (30-day accretion trajectory, cross-language portability)
3. **Honest gaps:** Identified claims that sound confident in the model but have insufficient evidence for external publication

### Sources synthesized:
- `model.md` — harness engineering model (429 lines, 7 invariants, 7 failure modes)
- `probes/2026-03-08-probe-30-day-accretion-trajectory-gate-effectiveness.md` — 12-week line count data
- `probes/2026-03-08-probe-cross-language-harness-portability.md` — Go→TypeScript validation
- `.kb/guides/minimum-viable-harness.md` — 3-tier MVH checklist
- `.kb/investigations/2026-03-08-inv-collect-cross-language-harness-evidence.md` — cross-language findings
- `.kb/investigations/2026-03-08-inv-define-minimum-viable-harness-agent.md` — MVH derivation
- `.kb/threads/2026-03-08-harness-engineering-as-strategic-position.md` — strategic framing
- `.kb/threads/2026-03-08-open-questions-harness-as-governance.md` — open questions
- `.kb/plans/2026-03-08-harness-publication.md` — publication plan (Phase 3)

---

## What I Observed

### 1. Model correction confirmed: spawn_cmd.go shrinkage was -1,755, not -840

The model previously stated "spawn_cmd.go shrank -840 lines after pkg/spawn/backends/ was created." The Mar 8 accretion probe measured the actual value: 2,432→677, which is -1,755 lines. The model was updated to reflect this (line 74). Publication draft uses the corrected figure. The attractor was nearly twice as effective as originally claimed for initial extraction — but re-accretion (+483 in 3 weeks) confirmed the pump dynamic regardless.

### 2. Three claims lack sufficient evidence for external publication

| Claim | Model Confidence | Evidence Available | Gap |
|-------|-----------------|-------------------|-----|
| "Stronger models need more coordination gates" | High (invariant #7) | Theoretical + daemon.go anecdote | No controlled experiment comparing model capability vs accretion rate |
| "265 contrastive trials, 7 skills" | High (quantified) | Referenced in model, actual trial data in skill-content-transfer model | Data exists but needs traceability to specific trial records |
| "3 entropy spirals, 1,625 lost commits" | High (referenced) | Referenced in entropy-spiral model | Individual spiral post-mortems not read in this session |

Publication draft handles these by stating them as observations with evidence references, not as experimentally controlled findings.

### 3. The compliance vs coordination distinction is the strongest new claim

This framing — that compliance failures resolve with model improvement while coordination failures worsen — is the most novel claim in the publication. It's supported by:
- daemon.go evidence (30 correct commits → 892-line growth = coordination)
- Theoretical argument (faster agents accrete more per session)
- Google analogy (architecture review despite top engineers)

But it lacks: controlled experiment showing faster agents produce more accretion, or evidence from systems with different model capability levels.

The publication draft presents this as a framework insight with supporting evidence, not as a proven law.

### 4. Gate effectiveness honesty strengthens rather than weakens the publication

The model and probes are remarkably honest about what isn't working: gates haven't bent the curve, completion gate exempts pre-existing bloat, pre-commit gate was dead code, fix:feat ratio spike was transient. Including this in the publication strengthens credibility — the audience will have their own accretion problems and will be skeptical of "we solved it" claims.

### 5. Cross-language portability framing is correct but needs caveats

The probe found 5/8 patterns translate directly. The publication draft correctly uses "the framework is language-independent, the gates are not." But the TypeScript testing was dry-run only — `orch harness init --dry-run`, not actual agent operation under governance for 30 days. This limitation is acknowledged in the Open Questions section.

---

## Model Impact

- [x] **Confirms** the model's core structure: hard/soft taxonomy, 5 invariants, 5 layers, attractors+gates framework, and cross-language portability all survived synthesis into publication format without contradiction.

- [x] **Confirms** the model's honest self-assessment: the "Why This Fails" section (7 failure modes) and the accretion probe's finding that gates haven't bent the curve both translate directly into the publication's "What's Not Working" section.

- [x] **Extends** model with: Publication-readiness assessment. The model's strongest claims for external audiences are: (1) accretion as coordination failure, (2) hard/soft taxonomy, (3) attractor+gate pairing. The weakest (needing more evidence): (1) "stronger models need more coordination gates" lacks controlled experiment, (2) soft harness budget curve shape unknown, (3) cross-language only validated via dry-run.

- [x] **Extends** model with: The compliance vs coordination distinction should be elevated to a top-level model concept, not just a section in "Core Mechanism." It's the strongest differentiating claim for publication.

---

## Notes

### Publication draft structure decisions

1. **Led with the problem, not the framework.** Practitioners care about their pain first, your taxonomy second.
2. **Included all negative evidence.** Gates haven't bent the curve, completion gate exempts pre-existing bloat, pre-commit gate was dead code, fix:feat spike was transient. This is the honest practitioner voice.
3. **Ended with minimum viable harness.** Readers want to act, not just understand. The "Getting Started" section gives them a concrete Day 1 checklist.
4. **Used "we" throughout.** This is experience report, not academic paper.
5. **Omitted model relationship diagram.** Internal model dependency structure (harness engineering → architectural enforcement → entropy spiral → etc.) is useful for maintainers, not for the external audience.

### Deliverable

Publication draft created at `.kb/publications/harness-engineering-draft.md` (~2,200 words). Structure: Problem → Evidence → Framework → What's Working/Not → Open Questions → Getting Started.
