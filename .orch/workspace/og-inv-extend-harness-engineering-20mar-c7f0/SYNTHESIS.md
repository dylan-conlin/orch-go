# Session Synthesis

**Agent:** og-inv-extend-harness-engineering-20mar-c7f0
**Issue:** led-3th
**Duration:** 2026-03-20T11:04 → 2026-03-20T11:15
**Outcome:** success

---

## Plain-Language Summary

Extended the harness engineering model's section on compliance vs coordination failure (previously §8) to show it as one instance of a broader cross-domain pattern called the "compositional correctness gap." This is where individually valid components compose into non-functional wholes because every validation gate checks component-level properties while failure only appears at the composition level. Added evidence from two new domains (LED gate stack producing valid-but-broken enclosures, and SendCutSend sheet metal DFM where operations pass individually but interfere when assembled) alongside the existing daemon.go evidence. The three cases sit at different abstraction scales (operation→assembly, geometry→function, agent→system) but share identical structure.

---

## TLDR

Extended §8 of the harness engineering model with the "compositional correctness gap" concept — a named failure mode class where individually valid components compose into non-functional wholes, supported by three-scale cross-domain evidence (DFM, LED gates, agent coordination).

---

## Delta (What Changed)

### Files Modified
- `.kb/models/harness-engineering/model.md` — §8 restructured and extended with compositional correctness gap generalization, three-scale evidence table, cross-domain evidence narratives, evolution entry, probe reference, Synthesized From entry, investigation reference

### Files Created
- `.kb/investigations/2026-03-20-inv-extend-harness-engineering-model-kb.md` — Investigation documenting the synthesis

### Commits
- (pending)

---

## Evidence (What Was Observed)

- LED gate stack probe (2026-03-20): ~150 OpenSCAD renders, 6 letter shapes — cut-channel LED routing passes all 4 gate layers but produces disconnected channels for non-rectangular letters
- SendCutSend DFM: domain knowledge from manufacturing operations — per-operation DFM validation passes, assembly reveals inter-operation interference
- daemon.go: 30 correct commits producing +892 lines and 6 duplicated concerns (existing model evidence)
- All three cases share identical structure: component validates, composition fails, no gate bridges the gap
- Entropy spiral model already had "Local correctness != global correctness" as a constraint (line 83)
- Verification bottleneck decision already noted "All fixes were real. The failure was compositional" (2026-01-14)

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for exact commands and expectations.

Key outcomes:
- §8 renamed to "The Compositional Correctness Gap" with compliance vs coordination as a subsection
- Three-scale evidence table present (operation→assembly, geometry→function, agent→system)
- Cross-domain evidence narratives for LED gates and DFM
- Named concept "compositional correctness gap" defined
- Summary updated to mention the concept
- Evolution entry added for 2026-03-20
- Probe entry added with investigation reference

---

## Architectural Choices

No architectural choices — task was within existing patterns. Extended an existing model section with new evidence and a named concept.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-20-inv-extend-harness-engineering-model-kb.md` — Synthesis of compositional correctness gap

### Decisions Made
- Decision: Frame compositional correctness gap as a generalization OF compliance vs coordination, not a replacement — the agent-specific framing remains valid, the new concept adds cross-domain applicability

### Constraints Discovered
- DFM evidence is from domain knowledge, not instrumented experiment — weaker than the LED probe evidence

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Model updated with compositional correctness gap
- [x] Investigation file complete with D.E.K.N. summary
- [x] Ready for `orch complete led-3th`

---

## Unexplored Questions

- Could a Layer 5 gate (LLM vision on rendered preview) close the compositional correctness gap for CAD?
- Does the compositional correctness gap predict failures in other domains (database schemas, API surfaces, CI pipelines)?
- Can composition-level gates be generated from component-level gate specifications?

---

## Friction

No friction — smooth session

---

## Session Metadata

**Skill:** investigation
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-extend-harness-engineering-20mar-c7f0/`
**Investigation:** `.kb/investigations/2026-03-20-inv-extend-harness-engineering-model-kb.md`
**Beads:** `bd show led-3th`
