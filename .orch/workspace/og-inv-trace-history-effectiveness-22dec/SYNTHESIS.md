# Session Synthesis

**Agent:** og-inv-trace-history-effectiveness-22dec
**Issue:** orch-go-1jmk
**Duration:** 2025-12-22
**Outcome:** success

---

## TLDR

Traced the history of confidence scores in investigations, found documented backfire (5 investigations with High/Very High confidence all wrong in Nov 2025), and recommended replacing percentage-based scores with structured uncertainty (tested/untested enumeration).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-22-inv-trace-confidence-score-effectiveness.md` - Full investigation documenting the history, backfire, current state, and recommendation

### Files Modified
- None

### Commits
- Pending (investigation file needs commit)

---

## Evidence (What Was Observed)

- **Codex epistemic debt case study** (orch-knowledge, Nov 2025): 5 investigations claimed High/Very High confidence (70-95%), all reached wrong conclusions about Codex hang root cause. Actual cause was simple file size limit, not the complex "frame problems" hypothesized.
- **Investigation skill acknowledgment**: Current SKILL.md explicitly states "The old investigation system produced confident wrong conclusions"
- **Current distribution** (orch-go): 72% of confidence scores at High 85-90%, 18% at Very High 95%, only 2% at Medium - indicating no discriminative value
- **Templates INDEX.md**: States "confidence calibration is meaningless" and deprecates complex templates that optimized for artifact quality over truth-finding

### Tests Run
```bash
# Count confidence score distribution across orch-go investigations
grep -r "Confidence:" .kb/investigations/*.md 2>/dev/null | \
  grep -oE "(High|Medium|Low|Very High|Very Low).*\([0-9]+%\)" | \
  sort | uniq -c | sort -rn

# Result: Heavy clustering at 85-95%, no variation = no signal
#  103 High (85%)
#  101 High (90%)
#   51 Very High (95%)
#   13 High (95%)
#    5 Very High (98%)
#    2 Medium (70%)
#    2 Medium (65%)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-22-inv-trace-confidence-score-effectiveness.md` - Complete analysis with documented backfire and replacement recommendation

### Decisions Made
- Recommend removing percentage-based confidence scores because:
  1. Documented 5/5 failure rate at High+ confidence (Codex case)
  2. LLMs cannot self-calibrate - optimize for sounding confident, not being accurate
  3. Current 90% clustering at 85%+ means no discriminative information

### Constraints Discovered
- LLM confidence scores are fundamentally uncalibratable - this is a known limitation, not fixable with better templates
- The useful part of confidence assessment is structured enumeration (what's tested vs untested), not the percentage

### Externalized via `kn`
- None yet (orchestrator should decide if this warrants a constraint/decision)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with D.E.K.N. summary)
- [x] Tests passing (audit test documented, pattern verified)
- [x] Investigation file has complete status
- [ ] Ready for `orch complete orch-go-1jmk`

### Follow-up Work (Optional)
If orchestrator agrees with recommendation:
1. Update investigation template in orch-knowledge to remove `**Confidence:**` line
2. Rename "What's certain" → "What's tested" (requires evidence)
3. Rename "What's uncertain" → "What's untested" (explicit gaps)
4. Add "What would change this" section (falsifiability criteria)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should the D.E.K.N. summary format change if confidence is removed? (Currently has Confidence line)
- What's the enforcement mechanism for "tested vs untested"? Currently honor-system.
- Are there cases where percentage confidence IS calibrated (outside LLM self-assessment)?

**Areas worth exploring further:**
- Price-watch investigations use structured uncertainty well - could be a template source
- Research on LLM confidence calibration may have newer findings

**What remains unclear:**
- Why confidence scores weren't removed after the Nov 2025 case study documented the failure
- Whether stakeholders (Dylan) prefer having some form of uncertainty indicator

---

## Session Metadata

**Skill:** investigation
**Model:** opus
**Workspace:** `.orch/workspace/og-inv-trace-history-effectiveness-22dec/`
**Investigation:** `.kb/investigations/2025-12-22-inv-trace-confidence-score-effectiveness.md`
**Beads:** `bd show orch-go-1jmk`
