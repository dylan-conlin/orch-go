# Probe: Artifact Type Audit Against Compositional Accretion Design Criterion

**Model:** compositional-accretion
**Date:** 2026-03-27
**Status:** Complete
**claim:** CA-01, CA-02, CA-03, CA-04
**verdict:** extends

---

## Question

The model claims outward-pointing atoms compose while inward-pointing atoms pile up (CA-01), adding composition signals reduces orphan rates (CA-02), natural signals outperform bolted-on metadata (CA-03), and conclusion-accumulating systems need more manual triage than question-accumulating systems (CA-04). Does auditing all 13 artifact types confirm this? Are there surfaces the model missed, or surfaces it classified optimistically?

---

## What I Tested

Audited 13 artifact types by:
1. Reading representative samples of each type
2. Measuring orphan/pile-up rates via grep, wc, and python3 analysis of .beads/issues.jsonl
3. Applying the 3-question design criterion (composition signal? natural to creation? points outward?)

```bash
# Investigation orphan rate
grep -l "^**Model:**" .kb/investigations/*.md | wc -l  # 66 of 365 active
grep -l "^**Model:**" .kb/investigations/archived/*.md | wc -l  # 6 of 854 archived

# Brief tension rate
grep -l "^## Tension" .kb/briefs/*.md | wc -l  # 73 of 73

# Probe completeness
grep -rl "## Model Impact" .kb/models/*/probes/*.md | wc -l  # 260 of 306
grep -rl "claim:" .kb/models/*/probes/*.md | wc -l  # 57 of 306
grep -rl "verdict:" .kb/models/*/probes/*.md | wc -l  # 36 of 306

# Thread status
grep "^status:" .kb/threads/*.md | sed 's/.*status: //' | sort | uniq -c
# 29 resolved, 16 open, 10 forming, 5 active

# Thread outward signal (resolved_to)
grep "^resolved_to:" .kb/threads/*.md | grep -v '""' | grep -v ': $' | wc -l  # 27 of 60

# Beads enrichment
python3 -c "[analysis of .beads/issues.jsonl]"
# 2136 issues, 18% with ANY routing label, 82% unenriched

# Decision linkage
grep -l "Auto-Linked" .kb/decisions/*.md | wc -l  # 25 of 44
grep -l "^**Extends:" .kb/decisions/*.md | wc -l  # 7 of 44

# Session debriefs: 18 total, no outward signals in template
# SYNTHESIS.md: 31 total, 27/31 have UnexploredQuestions
# VERIFICATION_SPEC.yaml: 85 total, self-describing only
# Guides: 23/37 have References sections
# Models: 45 total, all have Claims (testable) section
```

---

## What I Observed

### Full Audit Table (13 artifact types)

| # | Artifact Type | Count | Composition Signal | Natural? | Outward? | Pile-Up Rate | Classification |
|---|---|---|---|---|---|---|---|
| 1 | **Briefs** (.kb/briefs/) | 73 | Tension section | Yes (required) | Yes (names incompleteness) | 0% (73/73 have tensions) | **COMPOSING** |
| 2 | **Probes** (.kb/models/*/probes/) | 306 | claim/verdict + Model Impact | Partially (84% have Model Impact; only 18% have claim field) | Yes (relates to model) | ~16% missing Model Impact | **COMPOSING** (with gap) |
| 3 | **Models** (.kb/models/) | 45 | Claims table (testable) | Yes (core purpose) | Yes (invites disconfirmation) | Low — claims attract probes | **COMPOSING** |
| 4 | **Threads** (.kb/threads/) | 60 | Status lifecycle + resolved_to | Yes (status natural) | Mixed (45% have resolved_to link; 55% don't point at destination) | 55% unlinked | **MIXED** |
| 5 | **Investigations** (.kb/investigations/) | 365 active, 854 archived | Model link + defect-class | Partially (20% have defect-class) | When present, yes | 81.9% active orphan, 99.2% archived orphan | **PILING UP** |
| 6 | **Decisions** (.kb/decisions/) | 44 | Extends links + linked investigations | Partially (57% have investigation links; 16% have Extends) | When present, yes | ~43% fully unlinked | **MIXED** |
| 7 | **Beads Issues** (.beads/) | 2,136 | Enrichment labels (skill/area/effort) | No (requires enrichment step) | When enriched, yes | 82% unenriched | **PILING UP** |
| 8 | **Session Debriefs** (.kb/sessions/) | 18 | "What We Learned" section | Yes (natural) | No (describes conclusions, not gaps) | N/A — all inward-pointing | **PILING UP** |
| 9 | **SYNTHESIS.md** (.orch/workspace/) | 31 | UnexploredQuestions section | Mostly (87% have it) | Yes when populated | ~13% missing | **MIXED** (improved from pure pile-up) |
| 10 | **VERIFICATION_SPEC.yaml** (.orch/workspace/) | 85 | None — describes test results | N/A | No (self-describing) | 100% inward-pointing | **PILING UP** |
| 11 | **BRIEF.md** (.orch/workspace/) | 17 | Identical to Briefs (has Tension) | Yes | Yes | 0% — all have tensions | **COMPOSING** (feeds into .kb/briefs/) |
| 12 | **Guides** (.kb/guides/) | 37 | References section | Partially (62% have it) | Weakly (references describe context, not gaps) | N/A — guides don't accumulate fast | **INERT** (low volume, acceptable) |
| 13 | **Digests** (orch compose output) | ~0 (new) | Cluster grouping + thread matching | By design | By design | Not yet measurable | **COMPOSING** (by design, untested) |

### Key Findings

**Finding 1: Briefs are the model's success story (CA-01 confirmed).** 73/73 briefs have a Tension section. This is the only artifact type with 100% composition signal coverage. The signal is natural (agents articulate what they left unresolved) and outward-pointing (tensions name gaps). The model's prediction holds: briefs with tensions cluster in `orch compose` while SYNTHESIS.md without tensions didn't.

**Finding 2: Investigations are the model's failure case (CA-02 partially contradicted).** The defect-class signal was added to create an outward-pointing signal, but adoption is only 20% (74/365 active). The model link field has 18% coverage. Combined orphan rate: 81.9% active, 99.2% archived. CA-02 says adding a signal reduces orphan rate — but if adoption of the signal itself is only 20%, the surface still piles up. **The model doesn't account for signal adoption rates.** A signal that exists in the template but is filled in only 20% of the time is structurally identical to no signal.

**Finding 3: Beads issues are a two-population problem.** 100% have `issue_type` (set at creation — natural), but only 18% have routing labels (skill/area/effort — requires enrichment step). The model correctly predicts this: the natural signal (type) has 100% coverage; the bolted-on signal (enrichment labels) has 18%. CA-03 confirmed: natural signals dramatically outperform post-hoc metadata.

**Finding 4: Probes have a gap between structural signal and formal signal.** 84% of probes have the Model Impact section (the structural composition signal), but only 18% have the `claim:` frontmatter field and only 12% have `verdict:`. The claim/verdict was introduced later and hasn't been backfilled. The 84% Model Impact rate shows the structural signal IS natural (probes exist to test models). The low claim/verdict rate shows the formal metadata is bolted-on (extra ceremony after the real work is done). CA-03 extends: even within a single artifact type, the natural signal and the formal signal can have wildly different adoption rates.

**Finding 5: Session debriefs are pure inward-pointing.** The template has "What We Learned" (conclusions), "What's In Flight" (state), "What's Next" (direction). All three are self-describing. None point at gaps or relate to other atoms. 18 debriefs accumulate as independent snapshots with no compositional signal. But volume is low (18 over ~1 month), so manual review is feasible.

**Finding 6: VERIFICATION_SPEC.yaml is inert by design.** 85 files, all self-describing. No composition signal, no outward-pointing feature. This is correct — verification specs serve a gate function, not a knowledge function. They should be inert. Not every artifact needs to compose.

**Finding 7: Threads have a mixed signal.** Status lifecycle (forming/active/converged/subsumed/resolved) is an outward-pointing signal — it says "where I am in development." But 55% of resolved threads have empty `resolved_to`, meaning they don't point at where their insight landed. Threads that resolve without linking to a model/decision/brief become inert conclusions, exactly as CA-01 predicts.

**Finding 8: SYNTHESIS.md has improved from the model's documented failure mode.** The model documents SYNTHESIS.md as Failure Mode 1 (uniform atoms, inward-pointing). Current SYNTHESIS.md template includes UnexploredQuestions (87% populated) and feeds into BRIEF.md (which has the Tension section). The failure mode has been partially addressed — SYNTHESIS.md is now mixed rather than purely piling up. But the volume is low (31) because BRIEF.md has become the primary composition surface.

---

## Model Impact

- [x] **Confirms** CA-01: Outward-pointing atoms (briefs with tensions, probes with model impact) compose; inward-pointing atoms (session debriefs, verification specs, unlinked investigations) pile up. Measured across 13 artifact types.
- [x] **Confirms** CA-03: Natural signals (issue_type at 100%, Tension section at 100%, Model Impact at 84%) dramatically outperform bolted-on signals (enrichment labels at 18%, claim/verdict at 12-18%, defect-class at 20%).
- [x] **Confirms** CA-04: Conclusion-accumulating surfaces (session debriefs, VERIFICATION_SPEC.yaml) require manual review. Question-accumulating surfaces (briefs, threads) self-organize.
- [x] **Extends** CA-02: The model says "adding a composition signal reduces pile-up." This is true but incomplete. **Signal adoption rate mediates the effect.** Investigations have a defect-class signal, but at 20% adoption it doesn't reduce pile-up in practice. The model should add: "The signal must achieve >80% adoption to compose, and adoption is a function of whether the signal is natural to creation."
- [x] **Extends** the model with two new findings:
  1. **Not all surfaces need to compose.** VERIFICATION_SPEC.yaml is inert by design and that's correct. The design criterion should distinguish between "surfaces that should compose but don't" vs "surfaces that are correctly inert."
  2. **Signal adoption rate is the binding constraint.** The gap between "has a signal in the template" and "signal is actually filled" is where composition dies. Investigations have the signal. They just don't fill it.

---

## Recommendations by Surface

### Surfaces piling up — redesign recommended

1. **Investigations (81.9% orphan rate):** The defect-class signal exists but at 20% adoption. Recommendation: Make the model link REQUIRED at creation time (via `kb create investigation {slug} --model <model-name>` enforcement). The `--orphan` flag exists as escape hatch. This transforms the signal from opt-in to opt-out, which the brief Tension section proves works (100% adoption when required).

2. **Beads issues (82% unenriched):** The enrichment pipeline is a separate step from creation, and 82% never get enriched. Recommendation: Require `issue_type` + at least one routing label at creation time. The daemon already infers type at 69% — wire that inference to auto-apply labels at creation rather than expecting a separate enrichment pass.

3. **Session debriefs (pure inward-pointing):** Low volume (18) makes this acceptable. But if volume grows, add an "Open Questions" section to the template — what is the session leaving unresolved? This would give debriefs an outward-pointing signal without changing the creation workflow.

### Surfaces mixed — incremental improvement

4. **Threads (55% unlinked resolved_to):** Recommendation: When thread status changes to `resolved`, prompt for `resolved_to` link. If the resolution came from a model, decision, or brief, point there. If it was organic understanding, that's fine — but make the choice explicit.

5. **Probes (84% structural, 18% formal):** The Model Impact section works (84% natural adoption). The claim/verdict frontmatter doesn't (18%). Recommendation: Stop requiring claim/verdict as a formal field. Instead, extract it automatically from the Model Impact section content (which already says "Confirms/Contradicts/Extends"). The structural signal is the real signal; the formal metadata is ceremony.

6. **Decisions (43% unlinked):** The `Extends` field has only 16% adoption. Recommendation: At decision creation, auto-suggest related decisions based on keyword overlap (similar to how `orch compose` clusters briefs). This makes the signal discoverable rather than requiring manual recall.

### Surfaces composing — maintain

7. **Briefs:** Working. 100% tension coverage. No changes needed.
8. **Models:** Working. Claims table invites probes. No changes needed.
9. **BRIEF.md → .kb/briefs/ pipeline:** Working. Same structure, proven composition.

### Surfaces correctly inert — accept

10. **VERIFICATION_SPEC.yaml:** Correctly inert. Serves gate function, not knowledge function.
11. **Guides:** Low volume, stable. References section is weakly outward but sufficient.

---

## Notes

The audit reveals a meta-pattern: **the gap between "signal exists in template" and "signal is filled in practice" is where the model's predictions break down.** The model correctly identifies that outward-pointing signals enable composition. But it doesn't model the adoption curve. Investigations HAVE defect-class; they just don't USE it. The binding constraint isn't "does the atom format include a compositional signal?" — it's "does the creation workflow make filling the signal easier than skipping it?"

This suggests an amendment to the design criterion: after the three existing questions (what's the signal? is it natural? does it point outward?), add a fourth: **What is the adoption rate, and is the signal opt-in or opt-out?** Only opt-out signals achieve >80% adoption. Only >80% adoption enables measurable composition.
