# Session Synthesis

**Agent:** og-work-user-describes-symptoms-27dec
**Issue:** orch-go-5pgu
**Duration:** 2025-12-27 ~08:00 → ~09:15
**Outcome:** success

---

## TLDR

User symptom reports contain embedded hypotheses that orchestrators should parse using Evidence Hierarchy: trust observations (primary evidence), verify technical framings (secondary evidence). Recommended update to orchestrator skill Bug Triage section with symptom parsing guidance.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-27-inv-user-describes-symptoms-technical-terms.md` - Investigation documenting the user symptom → hypothesis gap and how to address it

### Files Modified
- None (recommendations only - implementation is for orchestrator to decide)

### Commits
- Pending: Investigation file commit

---

## Evidence (What Was Observed)

- Evidence Hierarchy principle (`~/.kb/principles.md:129-141`) already distinguishes primary vs secondary evidence - user observations fit primary, user hypotheses fit secondary
- Premise Before Solution principle (`~/.kb/principles.md:311-341`) applies to symptom interpretation - "fix hydration" assumes hydration is the problem, same as "how do we X" assumes X is correct
- Issue-creation skill (`~/.opencode/skills/issue-creation/SKILL.md:80-96`) has hypothesis testing phase, but applies to agent-formed hypotheses, not user-provided framings
- Orchestrator skill Bug Triage section lacks guidance on parsing user statements to separate observations from hypotheses
- Related investigation orch-go-khf5 "Investigate question-asking process - validate premises" addresses the strategic question version of this pattern

### Context Checked
```bash
kb context "premise before solution"
# Found related investigations

kb context "evidence hierarchy"  
# Found 13 related investigations

bd list | grep -i "premise\|symptom"
# Found orch-go-khf5 related issue
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-27-inv-user-describes-symptoms-technical-terms.md` - Full analysis of user symptom → hypothesis gap

### Decisions Made
- Decision 1: Apply existing Evidence Hierarchy principle rather than create new principle - the framework exists, just needs application guidance
- Decision 2: Fix at orchestrator level rather than worker skill level - workers receive already-parsed requests, orchestrators interpret raw user statements

### Constraints Discovered
- User statements contain two layers (observation + hypothesis) that look like single facts
- "Technical term" in symptom description is often user hypothesis, not observed reality
- Orchestrator skill Bug Triage section assumes orchestrators can distinguish "symptom only" from "cause embedded in symptom"

### Externalized via `kn`
- Recommend: `kn constrain "Separate user observations from hypotheses before spawning" --reason "Evidence Hierarchy: observations are primary evidence, hypotheses are secondary claims to verify"`

---

## Next (What Should Happen)

**Recommendation:** close + spawn-follow-up

### If Close
- [x] Investigation file complete with D.E.K.N. summary
- [x] All sections filled with evidence and recommendations
- [x] Clear actionable path forward
- [x] Ready for orchestrator review

### Spawn Follow-up

**Issue:** Update orchestrator skill Bug Triage with symptom parsing guidance
**Skill:** feature-impl
**Context:**
```
Add "Symptom Parsing" subsection to orchestrator skill Bug Triage section. 
Content is in .kb/investigations/2025-12-27-inv-user-describes-symptoms-technical-terms.md 
under "Implementation Details > What to implement first". 
Also record constraint via kn constrain.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How often does this pattern actually occur? (need more examples to validate frequency)
- What's the cost of verifying correct user diagnoses? (false positive rate of the parsing heuristic)
- Should workers explicitly report "user hypothesis validated/invalidated" as part of their completion?

**Areas worth exploring further:**
- Whether orch-go-khf5 (Premise Before Solution for strategic questions) and this investigation should be merged into unified "premise validation" guidance
- Whether the "red flag signals" reliably identify hypotheses (only pattern-matched from single example)

**What remains unclear:**
- How to handle users who actually know the cause (don't want to question their expertise unnecessarily)
- Whether this is common enough to warrant skill updates (single incident reported)

---

## Session Metadata

**Skill:** design-session
**Model:** claude-opus-4
**Workspace:** `.orch/workspace/og-work-user-describes-symptoms-27dec/`
**Investigation:** `.kb/investigations/2025-12-27-inv-user-describes-symptoms-technical-terms.md`
**Beads:** `bd show orch-go-5pgu`
