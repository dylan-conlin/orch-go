# Session Synthesis

**Agent:** og-inv-probe-validate-gate-22mar-e099
**Issue:** orch-go-ywpze
**Duration:** 2026-03-22
**Outcome:** success

---

## Plain-Language Summary

The intervention effectiveness hierarchy discovered in orch-go (structural attractors work, blocking gates get bypassed) is not unique to orch-go. Re-analyzing 6 external multi-agent frameworks through a gate/attractor lens shows a perfect correlation: every framework that uses gate-based coordination (runtime checking by LLMs) fails, and every framework that uses attractor-based coordination (structure determines work assignment at design time) succeeds. This validates the hierarchy as a general principle of multi-agent coordination, not an orch-go quirk. The practical implication: when building coordination, embed it in system structure rather than in runtime checks.

---

## TLDR

Tested whether the gate/attractor effectiveness hierarchy is orch-go-specific by classifying 6 external multi-agent frameworks by mechanism type. Found 6/6 correlation: gate-based coordination fails (CrewAI, LangGraph, OpenAI Agents SDK, Claude Agent SDK), attractor-based coordination works (Anthropic production, autoresearch). McEntire's controlled experiment confirms monotonic degradation from pure-attractor (100%) to maximum-gate (0%).

---

## Delta (What Changed)

### Files Created
- `.kb/models/knowledge-accretion/probes/2026-03-22-probe-validate-gate-attractor-external-frameworks.md` — Probe with full mechanism classification of 6 frameworks
- `.kb/investigations/2026-03-22-inv-probe-validate-gate-attractor-mechanism.md` — Investigation file with DEKN summary
- `.orch/workspace/og-inv-probe-validate-gate-22mar-e099/SYNTHESIS.md` — This file
- `.orch/workspace/og-inv-probe-validate-gate-22mar-e099/VERIFICATION_SPEC.yaml` — Verification spec

### Files Modified
- `.kb/models/knowledge-accretion/model.md` — Added external validation paragraph to Section 3a (intervention effectiveness), added probe reference to evidence list

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for:
- Probe file exists with all 4 required sections
- Model updated with external validation
- Investigation file complete with DEKN

---

## Evidence (What Was Observed)

- **6/6 correlation:** Every gate-based framework fails, every attractor-based framework works (classified from documented behavior in external framework investigation)
- **McEntire gradient:** 100% (pure attractor) → 64% (mixed) → 32% (pure gate) → 0% (maximum gates) — success drops monotonically as gate/attractor ratio increases
- **Anthropic mixed strategy:** Route+Align as attractors (task definitions, output formats), Throttle+Sequence as gates (scaling rules, dependency ordering) — the heavy-load primitives use attractors, lighter primitives use gates
- **Same mechanism at both scales:** orch-go (directory structure > pre-commit hooks) and external frameworks (structural task definitions > LLM routing decisions) show the same pattern

---

## Architectural Choices

No architectural choices — task was analysis/probe within existing patterns.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/knowledge-accretion/probes/2026-03-22-probe-validate-gate-attractor-external-frameworks.md` — External validation of effectiveness hierarchy

### Constraints Discovered
- Selection bias caveat: the 6 frameworks examined were pre-selected as known successes/failures. A systematic survey might find exceptions.
- The gate/attractor distinction is a special case of structural-vs-runtime coordination — the more fundamental principle.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (probe, investigation, model update, synthesis)
- [x] Probe has all 4 required sections (Question, What I Tested, What I Observed, Model Impact)
- [x] Investigation file has Status: Complete
- [x] Model updated with probe findings
- [x] Ready for `orch complete orch-go-ywpze`

---

## Unexplored Questions

- Whether attractor-based coordination can be retrofitted onto existing gate-heavy frameworks (e.g., could CrewAI add structural routing?)
- Whether the structural/runtime distinction applies to non-LLM multi-agent systems (robotics, distributed computing)
- Whether Align can be decomposed into sub-attractors (it covers 50% of MAST failures — likely multiple mechanisms)
- Whether orch-go's coordination model should add a "mechanism type" column to the four-primitives table

---

## Friction

No friction — smooth session. Evidence was already collected, analysis was the primary task.

---

## Session Metadata

**Skill:** investigation (probe mode)
**Model:** knowledge-accretion
**Workspace:** `.orch/workspace/og-inv-probe-validate-gate-22mar-e099/`
**Investigation:** `.kb/investigations/2026-03-22-inv-probe-validate-gate-attractor-mechanism.md`
**Beads:** `bd show orch-go-ywpze`
