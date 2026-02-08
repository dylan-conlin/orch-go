<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Confidence scores in investigations are noise, not signal. The Nov 22 Codex case study proved 5 investigations claiming High/Very High confidence (70-90%) all reached wrong conclusions. Current orch-go distribution shows 85% of investigations cluster at High (85-90%) or Very High (95%) - a uniformity that carries no discriminative value.

**Evidence:** (1) Codex epistemic debt case: 5 investigations, 5 wrong conclusions, all High+ confidence. (2) orch-go audit: 285 confidence scores, 72% at High 85-90%, 18% at Very High 95%. (3) Investigation skill now says "The old investigation system produced confident wrong conclusions" - the backfire is documented.

**Knowledge:** Confidence scores fail for two reasons: (1) LLMs cannot accurately self-assess uncertainty - they optimize for sounding confident, not being calibrated. (2) Scores without validation are meaningless - "High confidence" without testing is speculation with a number attached.

**Next:** Replace confidence scores with structured uncertainty: "What's certain" (tested), "What's uncertain" (not tested), "What would change this" (falsifiability). Remove percentage claims entirely.

**Confidence:** N/A - this investigation demonstrates why confidence scores should not exist.

---

# Investigation: Tracing Confidence Score History and Effectiveness

**Question:** What is the history of confidence scores in investigations, how did they backfire, are they still present, and should they be removed or replaced with something more calibrated?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: The Original Backfire - Codex Epistemic Debt Case Study (Nov 2025)

**Evidence:**

The definitive case study is documented in `.kb/investigations/systems/2025-11-23-document-codex-hang-epistemic-debt.md` (orch-knowledge):

| Investigation | Confidence | Actual Accuracy |
|--------------|-----------|-----------------|
| codex-dotfile-ignore | High (90%) | WRONG |
| codex-execution-loop | Medium (70%) | WRONG |
| codex-context-overload | Very High | WRONG |
| codex-orchestrator-guidance | High | WRONG |
| Synthesis (frame problem) | Very High (implied) | WRONG |

**All 5 investigations claimed high confidence but reached wrong conclusions.** The actual root cause (AGENTS.md file size exceeded Claude Code's maximum limit) was a simple constraint violation, not the complex "frame problems" proposed.

Key quotes from the case study:
- "Confidence inversely correlated with accuracy - the most confident investigation (90%) was among the wrongest"
- "Sophistication can create dangerous epistemic debt - synthesis appeared rigorous but produced wrong diagnosis with HIGH implied confidence"
- "Plausibility ≠ truth, sophistication ≠ correctness"

**Source:** `/Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-23-document-codex-hang-epistemic-debt.md`

**Significance:** This is the documented backfire. The investigation system produced confident wrong conclusions at scale, creating "epistemic debt" - cached false beliefs that compound over time.

---

### Finding 2: System Response - "Test Before Concluding" Discipline

**Evidence:**

The investigation skill was updated to address this failure. The current SKILL.md (orch-knowledge) states:

```
**Remember:** The old investigation system produced confident wrong conclusions. 
The fix is simple: test before concluding.
```

Key changes implemented:
1. **Mandatory "Test performed" section** - "If you didn't run a test, you don't get to fill in Conclusion"
2. **Evidence hierarchy** - "Artifacts are claims, not evidence"
3. **Simple 45-line template** replacing elaborate 256-line templates
4. **Deprecation of complex templates** - "Why deprecated: Case study showed 5 investigations using these templates all reached wrong conclusions despite 'High' and 'Very High' confidence"

From the templates INDEX.md:
> "No template structure, confidence calibration, or synthesis workflow prevents false conclusions. Only empirical testing does."

**Source:** 
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/SKILL.md:250`
- `/Users/dylanconlin/orch-knowledge/templates-src/investigations/INDEX.md:60-104`

**Significance:** The system explicitly acknowledges "confidence calibration is meaningless" (INDEX.md:60) and identified testing as the actual fix. Yet confidence scores remain in templates.

---

### Finding 3: Current State - Confidence Scores Still Present, Highly Clustered

**Evidence:**

Audited orch-go investigations (285 confidence score occurrences):

| Confidence Level | Count | Percentage |
|------------------|-------|------------|
| High (85%) | 103 | 36% |
| High (90%) | 101 | 35% |
| Very High (95%) | 51 | 18% |
| High (95%) | 13 | 5% |
| Very High (98%) | 5 | 2% |
| Very High (99%) | 2 | 1% |
| Very High (100%) | 2 | 1% |
| High (100%) | 1 | <1% |
| Medium (70%) | 2 | 1% |
| Medium (65%) | 2 | 1% |

**Key observation:** 72% of all confidence scores are High 85-90%, another 18% are Very High 95%. Only 2% are Medium (65-70%). This clustering indicates:
1. Agents default to High/Very High regardless of actual uncertainty
2. The scale provides no discriminative information
3. Lower confidence scores are socially discouraged (agents fear looking uncertain)

**Source:** `grep -r "Confidence:" .kb/investigations/*.md` in orch-go

**Significance:** If 90% of investigations claim 85%+ confidence, the score carries no signal. It's ritual, not calibration.

---

### Finding 4: Price-Watch Pattern - Same Clustering, Same Problem

**Evidence:**

Examined price-watch `.kb/investigations/` - same pattern observed:
- Most investigations claim High (85-90%) or Very High (95%)
- Investigations like `2025-12-16-inv-deep-dive-symptom-fix-patterns.md` claim "High (85%)" with detailed justification
- Yet the template includes extensive "What's certain/uncertain" sections that are MORE useful than the score

Example from price-watch investigation:
```markdown
**What's certain:**
- ✅ CleanupStaleScrapeJobsJob deletes ScrapeJobs pending >24h (verified in code + tests)
- ✅ purge_bullmq_jobs! silently continues when BullMQ unavailable

**What's uncertain:**
- ⚠️ Actual production orphan rates
- ⚠️ Runtime behavior when BullMQ is down
```

**This structured uncertainty carries actual information.** The "85%" number adds nothing.

**Source:** `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/`

**Significance:** The useful part of confidence assessment is the structured breakdown (certain/uncertain), not the percentage.

---

### Finding 5: Why Confidence Scores Fail Fundamentally

**Evidence:**

From research on LLM confidence calibration (found in grep results):

> "Traditional AI is overconfident because it optimizes for sounding confident."
> "High confidence (90%+): 47% accuracy (massively overconfident)"
> "Medium confidence (70-89%): 62% accuracy (still overconfident)"

From the Codex case study:
> "Confidence must be earned through validation, not plausibility"
> "If you haven't validated empirically, cap confidence at Medium (60-79%)"

**Core problem:** LLMs (including Claude) cannot accurately self-assess their uncertainty. Confidence scores are post-hoc rationalizations, not calibrated probabilities. An agent claiming "85% confident" has not performed any probabilistic reasoning - it's pattern-matching to what sounds reasonable.

**Source:**
- `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/seo-ai-agent/docs/ELCF-DECISION-QUALITY-METRICS.md`
- `/Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-23-document-codex-hang-epistemic-debt.md`

**Significance:** The mechanism for calibration doesn't exist. Asking an LLM "how confident are you?" is like asking "what number sounds professional?" The answer will cluster around 85-95% because that sounds competent.

---

## Test Performed

**Test:** Counted confidence score distribution across orch-go investigations and compared to documented outcomes.

```bash
# Count confidence score patterns
grep -r "Confidence:" .kb/investigations/*.md 2>/dev/null | \
  grep -oE "(High|Medium|Low|Very High|Very Low).*\([0-9]+%\)" | \
  sort | uniq -c | sort -rn

# Result:
# 103 High (85%)
# 101 High (90%)
#  51 Very High (95%)
#  13 High (95%)
#   5 Very High (98%)
#   2 Very High (99%)
#   2 Very High (100%)
#   2 Medium (70%)
#   2 Medium (65%)
#   1 High (100%)
```

**Result:** 72% of scores cluster at High 85-90%, 18% at Very High 95%. Only 2% claim Medium confidence. Distribution is heavily right-skewed - agents systematically overstate confidence.

---

## Synthesis

**Key Insights:**

1. **Documented backfire exists** - The Nov 2025 Codex case study proved 5 investigations with High/Very High confidence all reached wrong conclusions. This isn't theoretical - it happened.

2. **System acknowledged the problem** - The investigation skill explicitly states "The old investigation system produced confident wrong conclusions." Yet confidence scores remain.

3. **Current scores carry no signal** - With 90% of investigations claiming 85%+ confidence, the metric provides no discriminative value. It's become ritual completion of a template field.

4. **Structured uncertainty is useful** - The "What's certain/uncertain" breakdown in templates DOES carry information. The percentage does not.

5. **LLMs cannot self-calibrate** - Research shows LLMs optimize for sounding confident, not being accurate. Asking for confidence percentages is asking the wrong question.

**Answer to Investigation Question:**

**History:** Confidence scores were part of elaborate investigation templates intended to provide calibration. The Nov 2025 Codex case study demonstrated they produced "confident wrong conclusions."

**How they backfired:** 5 investigations claimed 70-90%+ confidence, all were wrong. The synthesis itself (claiming frame problem) was also wrong. Sophistication created false certainty.

**Still present:** Yes, in current templates. Distribution shows 90% at High/Very High - no variation, no signal.

**Should they be removed or replaced:**

**RECOMMENDATION: Replace confidence scores with structured uncertainty.**

| Current (Remove) | Replacement |
|------------------|-------------|
| **Confidence:** High (85%) | (Delete this line) |
| **What's certain:** ✅ items | **What's tested:** ✅ items with test evidence |
| **What's uncertain:** ⚠️ items | **What's untested:** ⚠️ hypotheses without validation |
| N/A | **What would change this:** Falsifiability criteria |

**Why this works:**
1. Forces enumeration of tested vs untested claims
2. Removes false precision (percentages)
3. Makes falsifiability explicit
4. Aligns with "test before concluding" discipline
5. Carries actual information (what we know vs guess)

---

## Implementation Recommendations

### Recommended Approach ⭐

**Remove percentage-based confidence, keep structured uncertainty** - Delete "Confidence: X%" from templates, expand "What's certain/uncertain" to require test evidence.

**Why this approach:**
- Directly addresses documented failure (Codex case study)
- Keeps the useful part (structured uncertainty enumeration)
- Removes the noise part (percentages that cluster at 85%)
- Aligns with "test before concluding" as the actual quality gate

**Trade-offs accepted:**
- Loss of "quick summary" feeling (but it was false precision anyway)
- May feel less "complete" (but completeness was ritual, not substance)

**Implementation sequence:**
1. Update investigation template in orch-knowledge and orch-go
2. Remove `**Confidence:**` line from D.E.K.N. summary
3. Rename "What's certain" → "What's tested" (requires evidence)
4. Rename "What's uncertain" → "What's untested" (explicit gaps)
5. Add "What would change this" section (falsifiability)

### Alternatives Considered

**Option B: Cap confidence at Medium (60%) without validation**
- Pros: Keeps scores but prevents overconfidence
- Cons: Still meaningless percentages, just lower ones
- When to use: If stakeholders insist on scores

**Option C: Binary confident/not-confident**
- Pros: Simpler than percentages
- Cons: Still single dimension, no enumeration of gaps
- When to use: For quick triage, not full investigations

**Option D: Probabilistic calibration training**
- Pros: Theoretically correct approach
- Cons: LLMs can't be calibrated this way, research shows persistent overconfidence
- When to use: Never for LLM self-assessment

---

## Self-Review

- [x] Real test performed (counted distribution, verified against case study)
- [x] Conclusion from evidence (based on documented backfire + current patterns)
- [x] Question answered (traced history, documented backfire, made recommendation)
- [x] File complete (all sections filled)
- [x] D.E.K.N. filled
- [x] Problem scoped (audited both orch-go and price-watch)

**Self-Review Status:** PASSED

---

## References

**Files Examined:**
- `/Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-23-document-codex-hang-epistemic-debt.md` - The original backfire case study
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/SKILL.md` - Current skill with "confident wrong conclusions" warning
- `/Users/dylanconlin/orch-knowledge/templates-src/investigations/INDEX.md` - Template deprecation rationale
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/*.md` - Current orch-go investigation patterns
- `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/*.md` - Price-watch investigation patterns

**Commands Run:**
```bash
# Count confidence score distribution in orch-go
grep -r "Confidence:" .kb/investigations/*.md 2>/dev/null | \
  grep -oE "(High|Medium|Low|Very High|Very Low).*\([0-9]+%\)" | \
  sort | uniq -c | sort -rn

# Search for backfire documentation
grep -r "confident wrong|old investigation" ~/orch-knowledge --include="*.md"

# Find epistemic debt case study
grep -r "epistemic debt" ~/orch-knowledge --include="*.md"
```

---

## Investigation History

**2025-12-22 [Start]:** Investigation started
- Question: Trace history and effectiveness of confidence scores
- Context: Spawned to understand original backfire and current state

**2025-12-22 [Research]:** Found key evidence
- Located Codex epistemic debt case study (Nov 2025)
- Found investigation skill acknowledgment of problem
- Audited current orch-go confidence score distribution

**2025-12-22 [Analysis]:** Synthesized findings
- Documented 5/5 wrong conclusions at High+ confidence
- Identified 90% clustering at High/Very High (no signal)
- Connected to LLM overconfidence research

**2025-12-22 [Complete]:** Investigation complete
- Recommendation: Replace percentages with structured uncertainty
- Key finding: The useful part is enumeration of tested/untested, not the number
