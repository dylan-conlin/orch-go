# Session Synthesis

**Agent:** og-inv-investigate-question-asking-25dec
**Issue:** orch-go-khf5
**Duration:** 2025-12-25 ~12:00 → ~13:00
**Outcome:** success

---

## TLDR

Investigated why an epic (orch-go-erdw) was created before validating its premise. Found that "how do we X" questions skip premise validation that "should we X" questions require. Recommendation: SHOULD → HOW → EXECUTE sequence for strategic questions.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-25-inv-investigate-question-asking-process-strategic.md` - Investigation answering the question sequence question

### Files Modified
- None

### Commits
- Investigation file (to be committed)

---

## Evidence (What Was Observed)

- Epic orch-go-erdw was created from "how do we evolve skills" question without testing premise
- Architect review later found the premise was wrong (current architecture is correct)
- Constraint kn-c12998 was created: "Ask 'should we' before 'how do we'"
- Design-session skill has no premise validation phase in its workflow
- Reflection Before Action principle applies at pattern level but not explicitly at question level

### Timeline Evidence
```
1. "How do we evolve skills to X?" → Design-session spawned
2. Epic orch-go-erdw created with 5 children
3. Architect spawned (later) → Found premise wrong
4. Epic paused/blocked
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-25-inv-investigate-question-asking-process-strategic.md` - Full investigation with SHOULD-HOW-EXECUTE framework

### Decisions Made
- Strategic questions require SHOULD-HOW-EXECUTE sequence
- "How" questions presuppose premise validity that should be tested first
- Design-session skill needs premise validation phase for strategic direction questions

### Constraints Discovered
- Words like "evolve to", "migrate to", "fix the" signal direction assumptions that need validation
- Reflection Before Action applies at multiple levels (pattern, question, execution)

### Externalized via `kn`
- Existing: kn-c12998 "Ask 'should we' before 'how do we' for strategic direction changes"
- Adding: see below

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created with DEKN summary)
- [x] Tests passing (N/A - investigation, not code)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-khf5`

### Recommendations for Orchestrator

1. **Add premise validation to design-session skill**
   - Insert decision point in Phase 1: "Does this assume strategic direction change?"
   - If yes, reframe to "should we" before proceeding to design

2. **Add question sequencing to orchestrator skill**
   - Under "Design Triage" or similar section
   - Guidance: When receiving "how do we X" questions involving strategic direction, first ask "should we X"

3. **Consider principle promotion**
   - SHOULD-HOW-EXECUTE is an application of Reflection Before Action
   - May be worth adding to principles.md if pattern recurs
   - Recommend waiting for 2-3 more instances to confirm universality

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- What heuristics reliably distinguish "tactical how" from "strategic how" questions?
- Should premise validation be synchronous (same session) or spawn a quick architect?
- How long should premise validation take before escalating?

**Areas worth exploring further:**
- Historical investigations for other premise-failure patterns
- Whether this applies to other skill types beyond design-session

**What remains unclear:**
- Whether SHOULD-HOW-EXECUTE is universal or context-specific
- Exact trigger words/patterns for premise validation

---

## Session Metadata

**Skill:** investigation
**Model:** opus
**Workspace:** `.orch/workspace/og-inv-investigate-question-asking-25dec/`
**Investigation:** `.kb/investigations/2025-12-25-inv-investigate-question-asking-process-strategic.md`
**Beads:** `bd show orch-go-khf5`
