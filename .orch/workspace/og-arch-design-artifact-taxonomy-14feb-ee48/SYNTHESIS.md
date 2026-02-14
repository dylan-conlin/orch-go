# Session Synthesis

**Agent:** og-arch-design-artifact-taxonomy-14feb-ee48
**Issue:** orch-go-d6k
**Duration:** 2026-02-14 → 2026-02-14
**Outcome:** success

---

## TLDR

Designed the artifact taxonomy evolution: probes become the universal evidence-gathering primitive, investigations retire. Decision record proposes expanding probe targets from "model claims only" to "any named claim" (model, decision, code hypothesis, assumption), a merged template (probe leanness + investigation rigor), coexistence migration (.kb/probes/ for new, .kb/investigations/ frozen), and skill rename (investigation → probe).

---

## Delta (What Changed)

### Files Created
- `.kb/decisions/2026-02-14-probe-as-universal-evidence-gathering-primitive.md` — Decision record with taxonomy, template design, migration strategy, and 5-phase implementation plan
- `.kb/investigations/2026-02-14-inv-design-artifact-taxonomy-evolution-probe.md` — Full design investigation with 6 findings, 4 alternatives evaluated, substrate-traced recommendations

### Commits
- (pending — will commit with this synthesis)

---

## Evidence (What Was Observed)

- Investigation skill already has dual-mode detection (probe mode vs investigation mode via SPAWN_CONTEXT markers) — infrastructure for universal probes partially exists
- Models-as-understanding decision (2026-01-12) already says "investigations = probes (temporal, narrow questions)" — concept alignment already exists
- Models README provenance chain says "Investigations (probe findings)" — naming hasn't caught up with reality
- 935 existing investigations in `.kb/investigations/` — mass rename would break thousands of cross-references
- 11 existing model-scoped probes across 4 model domains — small set, can be frozen as legacy
- 7 skills now have probe routing via SPAWN_CONTEXT markers (expanded Feb 13) — pattern is portable
- Probe/spike/scouting terminology disambiguated (Feb 13) — no conflation remaining

---

## Knowledge (What Was Learned)

### Decisions Made
- Probe becomes universal evidence-gathering primitive (strategic, requires Dylan's approval)
- Probe targets expand: model claims + decisions + code hypotheses + assumptions
- Merged template: probe leanness + D.E.K.N. + Prior Work + Structured Uncertainty
- Location: `.kb/probes/` for new, `.kb/investigations/` frozen as archive
- Migration: coexistence, no mass rename
- Skill rename: investigation → probe

### Constraints Discovered
- Template must stay LEAN — if it grows to investigation size, the forcing function weakens
- `kb context` must search both `.kb/probes/` and `.kb/investigations/` for archived knowledge
- The skill rename cascades through: orch-knowledge sources, deployed skills, orchestrator SKILL.md, CLAUDE.md files, spawn prompts, all 7 skills with probe routing

### Key Design Insight
The artifact name IS the thinking tool. "Probe" demands "probe WHAT?" while "investigation" allows "look into X." The forcing function for specificity lives in the NAME, not just the template.

---

## Next (What Should Happen)

**Recommendation:** escalate

### If Escalate
**Question:** Should we accept this taxonomy evolution? This is a strategic, irreversible change to how all evidence-gathering work is conceptualized and routed.

**The core tradeoff:**
1. **Accept:** Probes become universal. Decision tax eliminated. Agents must learn to name targets for novel exploration. 935 old investigations coexist as archive.
2. **Reject:** Keep current dual system. Decision tax remains but is familiar. No migration needed.

**Recommendation:** Accept. The concept already exists in practice — this decision aligns naming with reality. The primary risk (vague targets for novel exploration) is mitigable through self-review checklist.

---

## Unexplored Questions

**Questions that emerged during this session:**
- Should `kb create probe` generate the Target field interactively or from CLI args?
- Should existing `.kb/models/{name}/probes/` directories be deprecated or kept as write destinations?
- How does this affect the `kb reflect` synthesis opportunities? (29 clusters reference "investigation")
- Should the "Leave it Better" mandate from investigation skill transfer to probe skill?

**What remains unclear:**
- Whether agents can produce genuinely good probes against assumptions (untested)
- Whether the merged template will stay lean in practice or bloat over time

---

## Verification Contract

**Spec:** No VERIFICATION_SPEC.yaml needed — this is a design/decision artifact, not implementation.

**Key Outcomes:**
1. Decision record created: `.kb/decisions/2026-02-14-probe-as-universal-evidence-gathering-primitive.md`
2. Design investigation created: `.kb/investigations/2026-02-14-inv-design-artifact-taxonomy-evolution-probe.md`
3. Both include: taxonomy, merged template, migration strategy, 5-phase implementation plan
4. All 8 design questions from spawn context resolved with substrate-traced recommendations

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-design-artifact-taxonomy-14feb-ee48/`
**Investigation:** `.kb/investigations/2026-02-14-inv-design-artifact-taxonomy-evolution-probe.md`
**Decision:** `.kb/decisions/2026-02-14-probe-as-universal-evidence-gathering-primitive.md`
**Beads:** `bd show orch-go-d6k`
