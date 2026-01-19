# Session Synthesis

**Agent:** og-arch-analyze-question-vs-19jan-9457
**Issue:** orch-go-2yzjl
**Duration:** 2026-01-19 10:40 → 2026-01-19 11:10
**Outcome:** success

---

## TLDR

Analyzed whether the Question/Gate distinction in the decidability graph is crisp or fuzzy. **Finding:** The boundary is crisp but dynamic - Questions have unknown option spaces while Gates have known options requiring commitment. The same issue can transition from Question to Gate as understanding develops. Provided sharpened criteria and lifecycle test for classification.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-19-inv-analyze-question-vs-gate-distinction.md` - Full investigation with findings, sharpened criteria, and lifecycle model

### Files Modified
- None (analysis-only session)

### Commits
- (pending) Investigation creation and SYNTHESIS.md

---

## Evidence (What Was Observed)

- The decidability-graph.md model explicitly states Questions can "reveal they were actually Gates" (line 45)
- Question subtypes (factual/judgment/framing) already encode authority escalation paths (lines 105-113)
- The three concrete examples all fit the lifecycle transition model:
  - "Should we adopt event sourcing?" starts as Question, becomes Gate when options known and commitment required
  - "How should we encode subtypes?" depends on reversibility of choice
  - "Is our caching strategy correct?" may never become Gate if answer is affirmative

### Tests Run
```bash
# Conceptual test: Applied lifecycle model to 3 examples
# Result: All examples fit the model - no counterexamples found
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-19-inv-analyze-question-vs-gate-distinction.md` - Complete analysis with lifecycle model

### Decisions Made
- The boundary is **crisp, not fuzzy** - fuzziness comes from lifecycle transitions, not definitional overlap
- The key differentiator is **option space knowability**: Questions explore unknown options; Gates have known options requiring commitment

### Constraints Discovered
- "Reversibility" is context-dependent - what's reversible early becomes irreversible late
- Question subtypes (factual/judgment/framing) are orthogonal to Question→Gate transition
- Some Questions never become Gates (resolved without commitment)

### Sharpened Criteria

| Property | Question | Gate |
|----------|----------|------|
| **Option space** | Unknown or unclear | Known (options enumerable) |
| **Resolution shape** | Open (might fracture, collapse, reframe) | Binary (choose or defer) |
| **What's needed** | Understanding | Commitment |
| **Reversibility** | N/A (not a commitment yet) | Low (irreversible or costly) |

### The Lifecycle Test
```
Ask: Can you enumerate the concrete options?
     ├─ NO → It's a Question (keep exploring)
     └─ YES → Ask: Does choosing require irreversible commitment?
              ├─ NO → It's a Question (options known, but choice is reversible)
              └─ YES → It's a Gate (accumulate options, await authority)
```

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation with criteria and lifecycle model)
- [x] Tests passing (conceptual - examples fit model)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-2yzjl`

### Follow-up Recommendations

1. **Update decidability-graph.md** - Add "Lifecycle Model" section making Question→Gate transition explicit
2. **Document in beads question guide** - Add transition criteria for when Questions become Gates
3. **Consider `bd convert`** - Command to transition Question→Gate type in beads

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Can beads track Question→Gate transitions mechanically? (bd update --type?)
- Should dashboard show "potential Gate" status for Questions with enumerable options?
- How should daemon handle Questions approaching Gate transition?

**What remains unclear:**
- Whether the "enumerable options" heuristic holds across ALL domain types (only tested 3 examples)
- How to mechanically detect when a Question should transition to Gate

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-analyze-question-vs-19jan-9457/`
**Investigation:** `.kb/investigations/2026-01-19-inv-analyze-question-vs-gate-distinction.md`
**Beads:** `bd show orch-go-2yzjl`
