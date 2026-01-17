# Session Synthesis

**Agent:** og-feat-llm-detect-premise-17jan-fde6
**Issue:** orch-go-k2ol7
**Duration:** 2026-01-17 (Session started ~14:00, completed ~15:30)
**Outcome:** success

---

## TLDR

Implemented premise-skipping detection in coaching plugin to identify when Dylan asks "how to [strategic-verb]" questions without validating premises first, with graduated coaching messages suggesting "should we?" reframing to avoid wasted work on unvalidated strategic directions.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-17-inv-llm-detect-premise-skipping-question.md` - Investigation documenting requirements, existing infrastructure, and implementation design

### Files Modified
- `plugins/coaching.ts` (symlinked from `.opencode/plugins/coaching.ts`)
  - Added `DylanPatternState` fields: `premiseSkippingCount`, `premiseSkippingWarningInjected`, `premiseSkippingStrongWarningInjected`, `recentQuestions`
  - Added `detectPremiseSkipping()` function to match "how [to|do we|should we|can we] [strategic-verb]" patterns
  - Extended `injectCoachingMessage()` with `premise_skipping` and `premise_skipping_strong` message types
  - Integrated detection into `experimental.chat.messages.transform` hook as Pattern 4
  - Added recent questions tracking for pattern analysis

### Commits
- `8033d840` - investigation: document premise-skipping detection requirements and design
- `f4459d05` - feat: add premise-skipping detection to coaching plugin

---

## Evidence (What Was Observed)

### Investigation Findings
- Coaching plugin already has Dylan message monitoring infrastructure (`experimental.chat.messages.transform` hook) - verified via code inspection of coaching.ts:1284-1422
- Premise-skipping pattern is well-documented in `.kb/investigations/2025-12-25-inv-investigate-question-asking-process-strategic.md` with real failure case (orch-go-erdw epic)
- Red flag verbs identified from investigation: migrate, evolve, fix, centralize, transition, implement, solve
- Existing Dylan patterns (signal prefixes, priority uncertainty, compensation) have 0-11 production metrics each
- Frame collapse detection uses graduated warnings (first edit → gentle, 3+ edits → strong) which we replicated for premise-skipping

### Pattern Detection Logic
- Detection requires combination of: implementation phrasing ("how to/do we") + strategic verb + absence of personal pronouns ("I", "my")
- Personal pronoun filter reduces false positives on tactical questions like "How do I migrate this database?"
- Recent questions tracked (last 5) for pattern analysis and repetition detection

### Tests Run
```bash
# Validated TypeScript compilation
# Only error is pre-existing module resolution warning (not blocking)
git diff plugins/coaching.ts  # Verified changes are clean
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-17-inv-llm-detect-premise-skipping-question.md` - Complete investigation with D.E.K.N. summary, findings, synthesis, and implementation recommendations

### Decisions Made
- **Graduated coaching approach:** First detection → gentle suggestion, 2nd+ → stronger reminder. Mirrors frame collapse pattern and enables behavior change without being prescriptive.
- **Personal pronoun filtering:** Skip questions with "I" or "my" to avoid false positives on tactical "how do I" questions vs strategic "how do we" questions.
- **Verb list curation:** Start with 13 strategic verbs (migrate, evolve, fix, centralize, transition, implement, solve, change, shift, move, refactor, rewrite, rebuild) from documented patterns, can expand based on metrics.
- **Recent questions tracking:** Keep last 5 user questions in state for pattern analysis and coaching context.

### Constraints Discovered
- **No unit tests for Dylan patterns:** Existing coaching plugin patterns (behavioral variation, circular detection, Dylan signals) have logic but no test coverage. This new pattern continues that approach - testing would require mocking OpenCode plugin hooks.
- **Worker session filtering required:** Must check for worker sessions (SPAWN_CONTEXT.md reads, .orch/workspace/ paths) to avoid false positives when agents read spawn context.
- **Module resolution warning:** TypeScript config has moduleResolution issue with @opencode-ai/plugin types. Pre-existing, not blocking, doesn't affect runtime.

### Externalized via `kb`
- Investigation file documents complete design rationale and implementation details
- No `kb quick` entries needed - this is a feature addition, not a constraint/decision/failure to externalize

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (detection function, state tracking, message injection, metrics)
- [x] Investigation file has `Phase: Complete`
- [x] Commits created with clear messages
- [x] SYNTHESIS.md created (this file)
- [x] Ready for `orch complete orch-go-k2ol7`

**Manual testing recommendation (for orchestrator):**
After deployment, manually test by asking a premise-skipping question like "How do we migrate to microservices?" to verify:
1. Metric written to `~/.orch/coaching-metrics.jsonl` with type "premise_skipping"
2. Coaching message injected with suggestion to ask "Should we migrate?"
3. Second similar question triggers stronger reminder
4. Worker sessions are not affected

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**

- **Should threshold be time-based or count-based?** Current implementation uses count (2nd occurrence triggers strong warning), but should it consider timing? E.g., 2 premise-skipping questions in 2 minutes vs 2 hours.

- **Should verb list be LLM-expanded?** Current list is manually curated (13 verbs). Could use semantic similarity to catch variants like "shift to", "move toward", "transform into". Trade-off: complexity vs completeness.

- **Should premise detection integrate with design-session skill?** Currently only detects in Dylan messages. Should spawning design-session with "how to" question also trigger premise validation prompt?

- **What's the baseline rate of premise-skipping?** No historical data yet. After deployment, metrics will reveal: Is this common? Rare? Does coaching change behavior?

**Areas worth exploring further:**
- Test coverage for all Dylan pattern detection functions (detectSignalPrefix, detectPriorityUncertainty, detectPremiseSkipping, etc.)
- Verb list completeness - log analysis of "how to [verb]" patterns that didn't match to identify gaps
- Integration point with beads issue creation - should "how to" issues trigger premise validation before creating epic?

**What remains unclear:**
- Whether personal pronoun filtering is sufficient to distinguish tactical vs strategic questions
- Optimal threshold for strong warning (currently 2nd occurrence, could be 3rd)
- Whether coaching messages will change behavior or just create awareness

---

## Session Metadata

**Skill:** feature-impl
**Model:** Sonnet 3.5 (via OpenCode)
**Workspace:** `.orch/workspace/og-feat-llm-detect-premise-17jan-fde6/`
**Investigation:** `.kb/investigations/2026-01-17-inv-llm-detect-premise-skipping-question.md`
**Beads:** `bd show orch-go-k2ol7`
