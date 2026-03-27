# Session Synthesis

**Agent:** og-inv-audit-artifact-types-27mar-ddcf
**Issue:** orch-go-y85zx
**Duration:** 2026-03-27
**Outcome:** success

---

## Plain-Language Summary

We audited all 13 artifact types in orch-go against the compositional accretion model's design criterion: does the atom point outward, is the signal natural, and does it compose? The headline finding is that the model's core claims hold (briefs compose because tensions are required; investigations pile up because their signals are optional), but the model was missing a critical variable: **signal adoption rate**. A composition signal that exists in a template but is filled only 20% of the time doesn't compose — it just looks like it should. The model now has a 4th design criterion question: "Is the signal opt-in or opt-out?" Only opt-out signals achieve the >80% adoption needed for measurable composition.

---

## TLDR

Audited 13 artifact types. 4 compose (briefs, models, probes, comprehension queue), 3 mixed (threads, decisions, SYNTHESIS.md), 3 pile up (investigations, beads issues, session debriefs), 3 correctly inert (VERIFICATION_SPEC, guides, digests untested). The binding constraint isn't whether a signal exists — it's whether agents actually fill it. Opt-out signals hit 100%; opt-in signals plateau at 15-25%.

---

## Delta (What Changed)

### Files Created
- `.kb/models/compositional-accretion/probes/2026-03-27-probe-artifact-type-audit-design-criterion.md` — Full audit probe with measurements across 13 artifact types
- `.orch/workspace/og-inv-audit-artifact-types-27mar-ddcf/SYNTHESIS.md` — This file
- `.orch/workspace/og-inv-audit-artifact-types-27mar-ddcf/BRIEF.md` — Comprehension brief
- `.orch/workspace/og-inv-audit-artifact-types-27mar-ddcf/VERIFICATION_SPEC.yaml` — Verification spec

### Files Modified
- `.kb/models/compositional-accretion/model.md` — Merged audit findings: expanded table to 13 surfaces, added Failure Mode 4, added adoption constraint, added 4th design criterion question, added CA-06

---

## Evidence (What Was Observed)

Key measurements:
- **Briefs:** 73/73 have Tension section (100% adoption, opt-out signal) → **COMPOSING**
- **Probes:** 260/306 have Model Impact (84%) but only 57/306 have claim field (18%) → structural signal works, formal metadata doesn't
- **Investigations:** 66/365 active have model link (18%), 74/365 have defect-class (20%) → 81.9% active orphan rate
- **Beads issues:** 392/2136 have any routing label (18%), 100% have issue_type → natural signal at 100%, bolted-on at 18%
- **Threads:** 27/60 have non-empty resolved_to (45%), all 60 have status field
- **Decisions:** 25/44 have linked investigations (57%), 7/44 have Extends (16%)
- **Session debriefs:** 18 total, zero outward-pointing signals in template
- **SYNTHESIS.md:** 27/31 have UnexploredQuestions (87%, improved from original failure mode)
- **VERIFICATION_SPEC.yaml:** 85 total, all self-describing, correctly inert

### Tests Run
```bash
# Investigation orphan rate
grep -l "^**Model:**" .kb/investigations/*.md | wc -l  # 66 of 365
grep -l "^**Model:**" .kb/investigations/archived/*.md | wc -l  # 6 of 854

# Brief tension rate
grep -l "^## Tension" .kb/briefs/*.md | wc -l  # 73 of 73

# Beads enrichment
python3 analysis of .beads/issues.jsonl  # 18% with routing labels

# Thread statuses
grep "^status:" .kb/threads/*.md  # 29 resolved, 16 open, 10 forming, 5 active
```

---

## Architectural Choices

No architectural choices — task was within existing patterns (model probe against existing artifacts).

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/compositional-accretion/probes/2026-03-27-probe-artifact-type-audit-design-criterion.md` — Full audit of all 13 artifact types

### Constraints Discovered
- **Signal adoption rate is the binding constraint.** The gap between "signal exists in template" and "signal is filled in practice" is where composition dies. Only opt-out signals achieve >80% adoption.
- **Not all surfaces need to compose.** VERIFICATION_SPEC.yaml is correctly inert — it serves a gate function, not a knowledge function.

### Key Insight
The model was predicting outcomes from signal design alone. The audit shows the actual composition function is: `composition = outward_signal × adoption_rate`. If adoption is <25%, the signal is equivalent to absent regardless of how well-designed it is.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Probe file created with all 4 required sections
- [x] Model updated with merged findings
- [x] Ready for `orch complete orch-go-y85zx`

---

## Unexplored Questions

- **Would making investigation --model required (opt-out) actually increase adoption?** The brief Tension section proves opt-out works (100%), but investigations are created differently. The `--orphan` escape hatch might get overused.
- **Is 80% the right threshold for "enough adoption to compose"?** The data shows a gap between 15-25% (opt-in) and 84-100% (opt-out), but the threshold between "composes" and "doesn't compose" hasn't been measured directly.
- **Should session debriefs get an "Open Questions" section?** Volume is low (18), but if daily orchestrator sessions become the norm, this surface would grow fast.

---

## Friction

Friction: tooling: The `status` variable is read-only in zsh, causing one measurement script to fail. Workaround: renamed loop variable.

Friction: tooling: Model file was reverted between edit operations (likely by another concurrent agent or linter). Required re-reading and re-applying changes.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-inv-audit-artifact-types-27mar-ddcf/`
**Probe:** `.kb/models/compositional-accretion/probes/2026-03-27-probe-artifact-type-audit-design-criterion.md`
**Beads:** `bd show orch-go-y85zx`
