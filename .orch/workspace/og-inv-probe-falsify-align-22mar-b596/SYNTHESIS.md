# Session Synthesis

**Agent:** og-inv-probe-falsify-align-22mar-b596
**Issue:** orch-go-ekkut
**Duration:** 2026-03-22
**Outcome:** success

---

## Plain-Language Summary

The coordination model claimed Align is the "substrate" or "meta-primitive" that the other three primitives (Route, Sequence, Throttle) operate on — meaning they can't function without it. I tried to falsify this by finding cases where Align is broken but the others still work. I found 5 such cases from MAST academic data, McEntire's experiment, and orch-go production failures. In every case, Route/Sequence/Throttle still *ran* (messages got delivered, steps stayed ordered, velocity was controlled) but produced wrong outputs because the agents' model of "correct" was wrong. McEntire's hierarchical architecture achieved 64% success with only partial Align, proving the relationship is proportional (a multiplier) not binary (a substrate). The model has been updated to replace "meta-primitive" language with "validity condition" / "multiplier" — more precise without losing the core insight that Align is asymmetrically important.

---

## TLDR

Attempted to falsify "Align is the substrate for Route/Sequence/Throttle." Found 5 counterexamples where others hold mechanically without Align, but 0 where they produce coordination value without Align. Verdict: Align is a multiplier/validity condition, not a substrate. Model updated.

---

## Delta (What Changed)

### Files Created
- `.kb/models/coordination/probes/2026-03-22-probe-falsify-align-as-substrate.md` - Full probe with 15 classified failure cases
- `.kb/investigations/2026-03-22-inv-probe-falsify-align-substrate-find.md` - Investigation coordination artifact

### Files Modified
- `.kb/models/coordination/model.md` - Replaced "meta-primitive"/"substrate" language with "highest-leverage primitive and validity condition." Added probe to evidence table. Updated Align decomposition open question with 80-trial evidence.

---

## Evidence (What Was Observed)

- **5 Case 1 examples** (Align broken, others hold mechanically): MAST FM-1.1, McEntire hierarchical 36% failures, launchd post-mortem, orch-go competing instructions, stale knowledge cascades
- **4 Case 2 examples** (co-occurrence): System spiral, McEntire pipeline, agent ignores architect, 80-trial messaging condition
- **6 Case 3 examples** (others broken, Align intact): 80-trial no-coord, MAST FM-1.3/FM-3.1, cross-project skill injection, duplicate spawn race, user-as-message-bus
- **Key finding:** McEntire hierarchical at 64% success with partial Align proves proportional relationship
- **Key finding:** 80-trial messaging shows Align internal decomposition — task alignment defeats coordination alignment in 18/20 trials

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification details.

Key outcomes:
- 15 cases classified into 3 categories (5 + 4 + 6)
- At least 5 cases in Case 1 (falsification candidates) ✅
- Model updated with refined language ✅
- "Substrate" assessed — multiplier/validity condition is more precise ✅

---

## Architectural Choices

### Multiplier vs Substrate terminology
- **What I chose:** "Validity condition" and "multiplier" as replacement terms
- **What I rejected:** Keeping "substrate" or "meta-primitive"
- **Why:** 5 counterexamples show mechanical independence; McEntire shows proportional (not binary) degradation
- **Risk accepted:** Less dramatic framing may reduce rhetorical impact for publication

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/coordination/probes/2026-03-22-probe-falsify-align-as-substrate.md` - Falsification probe with 15 classified cases

### Decisions Made
- Replace "meta-primitive"/"substrate" with "validity condition"/"multiplier" because 5 empirical counterexamples show mechanical independence

### Constraints Discovered
- The mechanical/functional distinction is critical: primitives can mechanically operate independently but only produce value when Align holds
- Align has at least 3 sub-components (task, state, coordination) that can conflict with each other

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Probe file created and merged into model
- [x] Investigation file filled with D.E.K.N.
- [x] Model updated with refined language
- [x] Ready for `orch complete orch-go-ekkut`

---

## Unexplored Questions

- Does Align formally decompose into 3 sub-primitives (task, state, coordination), and would that make it a 6-primitive framework?
- Is there a case where broken Align causes Route/Sequence/Throttle to mechanically fail (not just produce wrong outputs)?
- Can perfect Align compensate for missing Route? (Would weaken the multiplier model)

---

## Friction

No friction — smooth session

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-inv-probe-falsify-align-22mar-b596/`
**Investigation:** `.kb/investigations/2026-03-22-inv-probe-falsify-align-substrate-find.md`
**Beads:** `bd show orch-go-ekkut`
