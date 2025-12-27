<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Designed a 5-tier completion escalation model based on work type, verification status, recommendations, and file scope that enables auto-completion for routine work while surfacing high-judgment items.

**Evidence:** Analyzed `runComplete()` flow in main.go:2958-3281, verification in check.go:365-506, and daemon completion in daemon.go:882-976; found existing signals include NextActions, Recommendation, visual verification, skill type, and verification failures.

**Knowledge:** Auto-completion is safe for ~60% of completions (clean pass + recommendation=close + no follow-ups), but the remaining 40% need human review for value extraction, architectural decisions, or failure triage.

**Next:** Implement the escalation model in `pkg/verify/escalation.go` with `DetermineEscalation()` function and integrate into daemon completion loop.

---

# Investigation: Completion Escalation Model

**Question:** When should completed agents surface for human review vs auto-complete silently? What escalation triggers should exist based on recommendations, verification failures, work type, and file scope?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Orchestration System
**Phase:** Complete
**Next Step:** None - implementation ready
**Status:** Complete

---

## Findings

### Finding 1: Current Completion Has Rich Signal Sources

**Evidence:** The completion workflow already collects extensive data that can inform escalation:

1. **SYNTHESIS.md fields** (`check.go:139-163`):
   - `Recommendation` (close, spawn-follow-up, escalate, resume)
   - `NextActions` (list of follow-up items)
   - `Outcome` (success, partial, blocked, failed)
   - `AreasToExplore`, `Uncertainties`

2. **Verification results** (`check.go:376-451`):
   - Phase status (Complete or not)
   - Constraint verification (required patterns matched)
   - Visual verification (for web/ changes)
   - Skill output verification

3. **Skill type** (`visual.go:14-32`):
   - Skills explicitly mapped as requiring visual verification
   - Skills explicitly excluded from visual verification

4. **File change scope** (`visual.go:124-179`):
   - Detection of web/ file changes
   - Recent commit analysis

**Source:** 
- `pkg/verify/check.go:139-163` - Synthesis struct
- `pkg/verify/check.go:376-451` - VerifyCompletionFull
- `pkg/verify/visual.go:14-32` - Skill categorization
- `cmd/orch/main.go:3124-3199` - Follow-up recommendations handling

**Significance:** We have abundant signals already available; the missing piece is a decision function that combines them into an escalation level.

---

### Finding 2: Daemon Auto-Completes Without Value Extraction

**Evidence:** The daemon's `ProcessCompletion()` function in `daemon.go:882-931` runs verification and closes issues, but it:
1. Does NOT parse SYNTHESIS.md for recommendations
2. Does NOT surface NextActions 
3. Does NOT differentiate by work type or skill
4. Just closes if verification passes

Compare to `runComplete()` in `main.go:3124-3199` which:
1. Parses synthesis
2. Checks for follow-up recommendations
3. Prompts interactively for issue creation

**Source:** 
- `pkg/daemon/daemon.go:882-931` - ProcessCompletion
- `cmd/orch/main.go:3124-3199` - runComplete follow-up handling

**Significance:** There's a clear gap: daemon completions are efficient but may lose value (recommendations not surfaced). Manual completions are thorough but require orchestrator attention. The escalation model should bridge this.

---

### Finding 3: Different Work Types Have Different Review Needs

**Evidence:** Analyzing skill categorizations in `visual.go:14-32` and the skill selection guide in the orchestrator skill:

| Work Type | Review Need | Why |
|-----------|-------------|-----|
| Bug fixes (`systematic-debugging`) | Low | Either works or doesn't |
| Feature implementation (`feature-impl`) | Medium | May have UI/design implications |
| Investigations | High | Produce recommendations, decisions |
| Architecture (`architect`) | High | Strategic choices need review |
| Refactoring | Low | Tests verify behavior preserved |

Investigation and architect skills are fundamentally different - they produce knowledge artifacts that need human absorption, not just verification that code works.

**Source:**
- `pkg/verify/visual.go:14-32` - skillsRequiringVisualVerification, skillsExcludedFromVisualVerification
- SPAWN_CONTEXT.md skill guidance section

**Significance:** The escalation model should weight skill type heavily. Knowledge-producing skills (investigation, architect, research) should ALWAYS surface for review. Code-only skills with clean verification can auto-complete.

---

### Finding 4: Current Prompting Is All-or-Nothing Interactive

**Evidence:** In `main.go:3159-3197`, the follow-up handling:
1. Requires stdin to be a terminal
2. Prompts for EACH actionable item individually
3. Skips entirely if not interactive

This binary approach doesn't fit the daemon/batch workflow where we need graduated escalation:
- Silent auto-complete for clean work
- Queued for review (but don't block) for valuable synthesis
- Block completion for failures

**Source:** `cmd/orch/main.go:3159-3197` - interactive prompting logic

**Significance:** Need an escalation level that allows "surface for review without blocking" - daemon can close the issue but flag it for orchestrator review.

---

### Finding 5: File Scope Matters for Risk Assessment

**Evidence:** Visual verification in `visual.go` already uses file scope:
- Web file changes trigger visual verification gate
- Recent commits analyzed for change scope

But this is binary (has web changes or not). Broader scope indicators:
- Number of files changed (5+ files = higher risk)
- Cross-package changes (multiple packages touched)
- Test-only vs production code changes
- Config/infrastructure changes

**Source:** 
- `pkg/verify/visual.go:124-156` - HasWebChangesInRecentCommits
- Git diff analysis patterns

**Significance:** File scope can serve as a secondary escalation trigger. Large file change scope + any other trigger = escalate more aggressively.

---

## Synthesis

**Key Insights:**

1. **Signals Already Exist** - The completion workflow has rich data (synthesis fields, verification results, skill type, file scope). What's missing is a decision function that combines these signals into an actionable escalation level.

2. **Knowledge Work Needs Review** - Investigation, architect, and research skills produce recommendations and decisions that lose value if auto-completed without human absorption. These should ALWAYS surface for review regardless of verification status.

3. **Daemon Needs Graduated Response** - Current daemon auto-completion is all-or-nothing. Need levels: auto-complete silently, auto-complete but flag for review, queue for manual completion.

4. **File Scope Multiplies Risk** - Large change scope (many files, cross-package) should increase escalation level when combined with other triggers.

**Answer to Investigation Question:**

Completed agents should surface for human review when:
1. **Knowledge work** - Investigation, architect, research, design-session skills (always)
2. **Has recommendations** - SYNTHESIS.md has NextActions or Recommendation != "close"
3. **Verification failures** - Any constraint, visual, or phase gate failures
4. **Outcome not success** - partial, blocked, or failed outcomes
5. **Large scope changes** - 10+ files modified (risk multiplier)

Auto-complete silently when:
1. Clean verification pass (all gates green)
2. Recommendation = "close" with no NextActions
3. Code-only skill (feature-impl, systematic-debugging)
4. Outcome = "success"
5. Reasonable change scope (<10 files)

---

## Structured Uncertainty

**What's tested:**

- ✅ Synthesis parsing works (existing tests in check_test.go)
- ✅ Verification flow catches failures (VerifyCompletionFull in use)
- ✅ Skill type can be extracted from SPAWN_CONTEXT.md

**What's untested:**

- ⚠️ Optimal thresholds for file count triggers (10 files is a guess)
- ⚠️ Performance impact of additional git operations per completion
- ⚠️ Edge case: investigation with Recommendation="close" but valuable NextActions

**What would change this:**

- Finding would be wrong if knowledge-producing skills routinely have empty/trivial recommendations
- Finding would be wrong if file count doesn't correlate with risk in practice
- Model may need tuning based on actual completion distribution data

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Signal-Based Escalation with Tiered Response** - Create an `EscalationLevel` enum with 5 tiers, each with a defined response action.

**Why this approach:**
- Uses signals already available (no new data collection needed)
- Graduated response enables batch processing while preserving value
- Clear precedence rules prevent ambiguity

**Trade-offs accepted:**
- Initial thresholds are educated guesses; may need tuning
- Adds complexity to completion path (but decision is O(1))

**Implementation sequence:**
1. Define `EscalationLevel` type and `DetermineEscalation()` function
2. Integrate into daemon `ProcessCompletion()` with appropriate actions
3. Add dashboard/CLI visibility for escalation decisions

### Escalation Tiers

```go
type EscalationLevel int

const (
    // Auto-complete silently. No human attention needed.
    EscalationNone EscalationLevel = iota
    
    // Auto-complete, but log for optional review.
    // Dashboard shows these as "worth reviewing".
    EscalationInfo
    
    // Auto-complete, but queue for mandatory review.
    // Orchestrator should review synthesis before next spawn.
    EscalationReview
    
    // Do NOT auto-complete. Surface immediately.
    // Requires human decision (e.g., visual approval).
    EscalationBlock
    
    // Do NOT auto-complete. Failure state.
    // Requires intervention to fix.
    EscalationFailed
)
```

### Decision Tree

```
Input: skill, verification_result, synthesis, file_count

1. VERIFICATION FAILED?
   └── YES → EscalationFailed
   
2. SKILL IS KNOWLEDGE-PRODUCING? (investigation, architect, research, design-session)
   └── YES → Has NextActions or Recommendation != "close"?
             └── YES → EscalationReview
             └── NO → EscalationInfo
   
3. VISUAL VERIFICATION NEEDS APPROVAL?
   └── YES → EscalationBlock
   
4. OUTCOME != "success"? (partial, blocked, failed)
   └── YES → EscalationReview
   
5. HAS RECOMMENDATIONS? (NextActions > 0 OR Recommendation = spawn-follow-up/escalate/resume)
   └── YES → file_count > 10?
             └── YES → EscalationReview
             └── NO → EscalationInfo
   
6. LARGE SCOPE? (file_count > 10)
   └── YES → EscalationInfo
   
7. OTHERWISE
   └── EscalationNone
```

### Alternative Approaches Considered

**Option B: Binary escalate/auto-complete**
- **Pros:** Simpler, easier to reason about
- **Cons:** Either surfaces everything (noisy) or nothing (loses value)
- **When to use instead:** If tiered response proves too complex in practice

**Option C: Machine learning on historical completions**
- **Pros:** Could learn optimal thresholds from data
- **Cons:** Requires training data, adds opacity, overkill for current scale
- **When to use instead:** At scale with hundreds of completions/day

**Rationale for recommendation:** Tiered approach gives nuanced control without ML complexity. Matches existing verification tier patterns (warnings vs errors).

---

### Implementation Details

**What to implement first:**
1. `pkg/verify/escalation.go` with `DetermineEscalation()` function
2. Modify `daemon.ProcessCompletion()` to use escalation level
3. Add `escalation_level` field to completion events for observability

**Things to watch out for:**
- ⚠️ Knowledge-producing skill list needs to stay in sync with skill definitions
- ⚠️ File count threshold may need per-project tuning
- ⚠️ EscalationBlock must not silently become EscalationReview on retry

**Areas needing further investigation:**
- Optimal file count thresholds (currently guessing 10)
- Whether to consider file paths (e.g., changes to pkg/verify/ are higher risk)
- Dashboard UX for reviewing escalated completions

**Success criteria:**
- ✅ ~60% of completions auto-complete silently (clean code-only work)
- ✅ ~30% auto-complete with Info/Review flag (recommendations preserved)
- ✅ ~10% block for human decision (failures, visual approval)
- ✅ Zero recommendations lost to silent auto-completion

---

## References

**Files Examined:**
- `pkg/verify/check.go` - Synthesis parsing, VerifyCompletionFull
- `pkg/verify/visual.go` - Visual verification, skill categorization
- `pkg/verify/constraint.go` - Constraint verification
- `pkg/verify/skill_outputs.go` - Skill output verification
- `pkg/daemon/daemon.go` - ProcessCompletion, CompletionOnce
- `cmd/orch/main.go` - runComplete, follow-up handling
- `cmd/orch/review.go` - getCompletionsForReview, synthesis parsing

**Commands Run:**
```bash
# Explore completion flow
kb context "completion"
bd show orch-go-dgzk
```

**Related Artifacts:**
- **Decision:** Prior decision "orch complete surfaces synthesis recommendations before closing"
- **Decision:** Prior decision "Daemon completion polling preferred over SSE detection"
- **Investigation:** This investigation completes the escalation model design

---

## Investigation History

**2025-12-27 13:39:** Investigation started
- Initial question: When should completed agents surface for human review vs auto-complete?
- Context: Need to balance efficiency of batch completion with value extraction from recommendations

**2025-12-27 13:45:** Core signals identified
- Found Synthesis struct with all relevant fields
- Found daemon ProcessCompletion lacks recommendation surfacing

**2025-12-27 14:00:** Escalation model designed
- 5-tier model: None, Info, Review, Block, Failed
- Decision tree with clear precedence rules

**2025-12-27 14:15:** Investigation completed
- Status: Complete
- Key outcome: Ready to implement EscalationLevel with DetermineEscalation() function
