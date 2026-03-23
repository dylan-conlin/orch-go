# Session Synthesis

**Agent:** og-inv-probe-overclaimed-coordination-22mar-794d
**Issue:** orch-go-dotqm
**Duration:** 2026-03-22
**Outcome:** success

---

## Plain-Language Summary

Three coordination model claims were graded "overclaimed" by an epistemic audit. I probed each by analyzing existing experimental data (139 trials) and mapping against established coordination theory from organizational science and distributed systems. All three claims SCOPE (narrow) rather than confirm or contradict: the experimental findings are real but the language implies broader applicability than the evidence supports. Claim 4 (task complexity) applies to additive tasks with shared gravitational insertion points, not all task types. Claim 6 (messaging) fails due to a specific false model of git merge mechanics, not because communication "fundamentally" can't coordinate. Claim 9 (four primitives) covers multi-agent SE well but misses decomposition, recovery, and meta-coordination found in broader coordination theory. The model has been updated with scoped language and three new open questions for future experimental work.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification details.

**Key outcomes:**
1. Three probe files created in `.kb/models/coordination/probes/`
2. Parent model (`model.md`) updated with scoped language for all three claims
3. Evidence table extended with three new entries
4. Open questions section updated with new questions from probes
5. Boundaries section expanded with newly identified out-of-scope areas

---

## Delta (What Changed)

### Files Created
- `.kb/models/coordination/probes/2026-03-22-probe-claim4-task-type-scope.md` — Claim 4 probe: task structure vs complexity
- `.kb/models/coordination/probes/2026-03-22-probe-claim6-messaging-scope.md` — Claim 6 probe: false merge model mechanism
- `.kb/models/coordination/probes/2026-03-22-probe-claim9-primitives-generality.md` — Claim 9 probe: taxonomy mapping
- `.kb/investigations/2026-03-22-inv-probe-overclaimed-coordination-model-claims.md` — Coordination investigation

### Files Modified
- `.kb/models/coordination/model.md` — Scoped Claims 4, 6, 9; updated evidence table, open questions, boundaries

---

## Knowledge (What Was Learned)

### Key Insights

1. **"Task complexity" is the wrong variable** — Both tested task types share the same gravitational-convergence structure. The right variable is task structure (additive vs modification vs cross-file), not implementation difficulty.

2. **Messaging fails due to false merge models** — Agents communicate effectively but have wrong understanding of git merge mechanics. This is a specific, addressable failure mechanism, not a fundamental limitation of communication.

3. **Four primitives are domain-scoped** — They map well to established coordination theory (Malone & Crowston, Mintzberg) at the requirements level but miss decomposition, recovery, and meta-coordination primitives found in broader theory.

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete
- [x] Three probes written with all required sections
- [x] Probes merged into parent model
- [x] Investigation file has Status: Complete
- [x] SYNTHESIS.md created

---

## Unexplored Questions

- Would a modification-task experiment (agents refactoring different existing functions) show 0% conflict rate with messaging? This would directly confirm the task-structure distinction.
- Would git-merge-aware messaging ("additions at the same hunk position produce CONFLICT") change agent behavior? Cheapest untested intervention.
- Should decomposition and recovery be added as coordination primitives?

---

## Friction

No friction — smooth session. Analytical probes worked well without needing to run new experiments.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-inv-probe-overclaimed-coordination-22mar-794d/`
**Investigation:** `.kb/investigations/2026-03-22-inv-probe-overclaimed-coordination-model-claims.md`
**Beads:** `bd show orch-go-dotqm`
