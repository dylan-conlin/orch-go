<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** System shows strong temporal discipline (6/7 patterns well-aligned); investigation promotion gate is sole misalignment, firing at session end when context has decayed.

**Evidence:** Templates explicitly specify WHEN to fill sections; UpdateHandoffAfterComplete hook cites principle by name; phase gates verify progressive capture; one gate (investigation promotion) accumulates until session end.

**Knowledge:** Templates embody the principle, infrastructure enforces it; verification gates (check artifacts exist) differ from capture gates (force creation); session boundaries should surface context, not capture it.

**Next:** Implement completion-time promotion prompt to replace session-end gate; document pattern in orchestrator skill.

**Promote to Decision:** recommend-no - Operational finding (one misaligned gate), not architectural pattern worth preserving

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Audit Forcing Functions Temporal Alignment

**Question:** Do existing forcing functions (gates, hooks, skill triggers, documentation patterns) fire when context exists, or do they rely on end-of-session recall?

**Started:** 2026-01-18
**Updated:** 2026-01-18
**Owner:** og-feat-audit-forcing-functions-18jan-f944
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Investigation Promotion Gate Fires at Session End (Misaligned)

**Evidence:** The `gateInvestigationPromotions()` function checks for accumulated promotion candidates and blocks session end if threshold exceeded:

```go
// session.go:728-729
// This is a gate at session end to prevent accumulation of promotion candidates.
func gateInvestigationPromotions() error {
    count := checkInvestigationPromotions()
    if count <= InvestigationPromotionThreshold {
        return nil
    }
    // Prompt user to triage or abort session end
}
```

Called from `orch session end` (line 996-1001).

**Source:** `cmd/orch/session.go:624-752`

**Significance:** **MISALIGNED with Capture at Context.** The gate fires when context about why investigations should be promoted is RECONSTRUCTED (end of session), not when context EXISTS (during/after investigation completion). Orchestrator must recall which investigations produced reusable patterns vs point-in-time findings. This violates the decay curve - by session end, the "why promote" reasoning has decayed from observed to reconstructed.

---

### Finding 2: SESSION_HANDOFF.md Template Has Explicit Progressive Capture Guidance (Well-Aligned)

**Evidence:** Template includes detailed timing guidance:

```markdown
<!-- SESSION_HANDOFF.md:11-40 -->
**Fill this file AS YOU WORK, not at the end.**

Progressive documentation pattern:
1. SESSION START: Fill metadata
2. DURING: Add to Spawns, Evidence, Friction as you go
3. BEFORE HANDOFF: Synthesize Knowledge, fill Next
4. FINAL: Write TLDR, update outcome

Section timing:
| Section | When to Fill |
|---------|--------------|
| Friction | Anytime (capture frustrations immediately) |
| Spawns | During work (as you spawn/complete agents) |
| Evidence | During work (as you observe patterns) |
```

Explicit anti-pattern warning: "The anti-pattern: 'I'll write the handoff when I'm done' → leads to lost context"

**Source:** `.orch/templates/SESSION_HANDOFF.md:11-40`

**Significance:** **WELL-ALIGNED with Capture at Context.** Template explicitly specifies WHEN each section should be filled, aligning capture timing with context availability. Friction section has specific reminder about immediate capture to prevent rationalization. This is a MODEL implementation of the principle.

---

### Finding 3: SYNTHESIS.md Template Has Same Progressive Capture Pattern (Well-Aligned)

**Evidence:** Worker artifact template mirrors orchestrator handoff pattern:

```markdown
<!-- SYNTHESIS.md:11-39 -->
**Fill this file AS YOU WORK, not at the end.**

Section timing:
| Section | When to Fill |
|---------|--------------|
| TLDR | Last (after you know what happened) |
| Delta | During work (as you create/modify files) |
| Evidence | During work (as you observe things) |
| Knowledge | After implementation (patterns noticed) |
| Unexplored | Anytime (capture questions as they emerge) |
```

**Source:** `.orch/templates/SYNTHESIS.md:11-39`

**Significance:** **WELL-ALIGNED with Capture at Context.** Workers receive same progressive capture guidance. "Unexplored Questions" section explicitly requires real-time capture. The templates teach the principle through structured timing guidance.

---

### Finding 4: UpdateHandoffAfterComplete Hook Explicitly Cites Principle (Well-Aligned)

**Evidence:** Code comment explicitly references Capture at Context:

```go
// session.go:1827-1830
// Part of "Capture at Context" principle: update handoff when agent completes
// rather than waiting until session end.
// UpdateHandoffAfterComplete prompts for handoff updates after an agent completes.
func UpdateHandoffAfterComplete(workspaceName, beadsID, outcome, keyFinding string) error {
```

Called from `orch complete` immediately after agent verification passes.

**Source:** `cmd/orch/session.go:1827-2104`

**Significance:** **WELL-ALIGNED with Capture at Context.** This hook fires at the optimal moment - when orchestrator has just read SYNTHESIS.md and context about the agent's work is fresh. Prompts for handoff updates while observations are available, not reconstructed. This is infrastructure implementing the principle (not just guidance).

---

### Finding 5: Phase Gates Verify Progressive Capture, Don't Rely on End Recall (Well-Aligned)

**Evidence:** Phase gate system has two parts:

1. **Progressive capture mechanism**: Agents report phases via `bd comment` DURING work
   ```bash
   bd comment <id> "Phase: Planning - ..."
   bd comment <id> "Phase: Implementation - ..."
   ```

2. **Completion verification**: Gate at `orch complete` verifies phases were reported
   ```go
   // phase_gates.go:120-162
   func VerifyPhaseGates(requiredPhases []Phase, comments []Comment) PhaseGateResult {
       // Extract reported phases from beads comments
       // Check if all required phases were reported
   }
   ```

**Source:** `pkg/verify/phase_gates.go:1-194`, feature-impl skill guidance

**Significance:** **WELL-ALIGNED with Capture at Context.** The system enforces progressive capture (phases reported during work) then verifies at completion. Gate checks THAT capture happened at right moments, not trying to capture AT the gate. This is the correct pattern: gate verifies temporal discipline, doesn't compensate for missing it.

---

### Finding 6: SessionStart Hooks Surface Context at Session Start (Well-Aligned)

**Evidence:** Multiple hooks inject context when orchestrator session begins:

```go
// session.go:627-645
// surfaceFocusGuidance loads ready issues and displays them grouped into thematic threads.
// Part of Capture at Context principle.
func surfaceFocusGuidance() { ... }

// session.go:650-692
// surfaceReflectSuggestions loads and displays synthesis warnings from reflect-suggestions.json.
// This proactively surfaces consolidation needs at session start
func surfaceReflectSuggestions() { ... }
```

**Source:** `cmd/orch/session.go:627-692`, hook investigation `.kb/investigations/2026-01-16-inv-audit-sessionstart-hooks-claude-code.md`

**Significance:** **WELL-ALIGNED with Capture at Context.** Hooks fire when orchestrator needs context (session start), not when convenient (end). Focus guidance and reflection suggestions surface WHEN decision-making happens, at the boundary where context is actionable. Timing matches need.

---

### Finding 7: Completion Gates Fire After Work, Checking Artifacts (Verification Pattern)

**Evidence:** `orch complete` runs verification gates at completion time:

```go
// complete_cmd.go:65-94
VERIFICATION GATES:
The following gates are checked before completion:
  - test_evidence:       Tests run and results captured
  - visual_verify:       UI changes verified via browser/screenshot  
  - git_diff:            Changes reviewed and appropriate
  - synthesis:           SYNTHESIS.md exists and complete
  - build:               Project builds successfully
  - constraint:          No constraint violations
  - phase_gate:          Required skill phases completed
  - skill_output:        Required skill outputs exist
```

Each gate can be bypassed with `--skip-{gate} --skip-reason "rationale"`.

**Source:** `cmd/orch/complete_cmd.go:48-294`

**Significance:** **MIXED ALIGNMENT.** These are *verification* gates (checking work quality) not *capture* gates (forcing documentation). Most verify artifacts that should exist from work (SYNTHESIS.md, test output, git commits). However, some gates like `test_evidence` might be checking for something that should have been captured during work. Need to distinguish: verification of completed work (OK at end) vs capture of context (should be during).

---

## Synthesis

**Key Insights:**

1. **Templates embody the principle; infrastructure enforces it** - SESSION_HANDOFF.md and SYNTHESIS.md templates (Findings 2-3) explicitly teach progressive capture with timing tables. UpdateHandoffAfterComplete (Finding 4) provides infrastructure that prompts at the right moment. The principle moves from guidance → structure.

2. **Verification gates vs capture gates are distinct patterns** - Phase gates (Finding 5) verify that capture happened during work; they don't try to capture at completion. Completion gates (Finding 7) verify work quality (tests pass, builds work). This is the correct pattern: gates should check artifacts exist, not try to create them under cognitive load. Investigation promotion gate (Finding 1) violates this by asking orchestrator to triage at session end when context has decayed.

3. **Session boundaries are where hooks should surface, not capture** - SessionStart hooks (Finding 6) surface context when orchestrator begins work (focus guidance, reflection suggestions). This is aligned: provide input at moment of need. Session end should verify capture happened throughout, not force last-minute recall.

4. **Codebase already recognizes "remember to" as anti-pattern** - The codebase-audit skill searches for "remember to|don't forget" patterns as organizational debt (grep results). System knows reminders fail, but one gate (investigation promotion) still relies on end-of-session recall.

**Answer to Investigation Question:**

**Do existing forcing functions fire when context exists, or rely on end-of-session recall?**

**MOSTLY ALIGNED (6 of 7 patterns).** The system demonstrates strong temporal discipline:

- **Templates** (SESSION_HANDOFF.md, SYNTHESIS.md) explicitly specify WHEN each section should be filled
- **Hooks** (SessionStart, UpdateHandoffAfterComplete) fire at boundaries when context is actionable
- **Phase gates** verify progressive capture happened, don't try to force it at completion
- **Completion gates** verify work artifacts, not trying to create documentation under time pressure

**ONE MISALIGNMENT FOUND:** Investigation promotion gate (`gateInvestigationPromotions`) fires at session end, asking orchestrator to triage which investigations should be promoted when context about their strategic value has decayed. This violates Capture at Context - the promotion decision should happen during/after investigation completion when context is fresh, not accumulated until session end.

**LIMITATION:** This audit focused on explicit forcing functions (gates, hooks, templates). Did not audit implicit skill triggers (e.g., when skills suggest creating artifacts) or check if guidance is actually followed in practice (behavioral audit would require analyzing actual sessions).

---

## Structured Uncertainty

**What's tested:**

- ✅ Investigation promotion gate code examined (cmd/orch/session.go:726-752)
- ✅ Template progressive capture guidance verified (SESSION_HANDOFF.md, SYNTHESIS.md)
- ✅ UpdateHandoffAfterComplete code examined (session.go:1827-2104)
- ✅ Phase gates verification logic examined (pkg/verify/phase_gates.go)
- ✅ SessionStart hooks timing verified via hook investigation
- ✅ Completion gates list enumerated (complete_cmd.go:65-94)

**What's untested:**

- ⚠️ Behavioral audit - do orchestrators actually follow progressive capture guidance?
- ⚠️ Skill implicit triggers - when do skills suggest creating artifacts (not just gates)?
- ⚠️ Completion-time promotion prompt UX - would orchestrators find it helpful or annoying?
- ⚠️ Frequency of investigation promotion - how often does the session-end gate actually fire?

**What would change this:**

- Finding would be wrong if investigation promotion gate doesn't actually fire at session end (code inspection confirms it does)
- Finding would be wrong if templates don't specify timing (they explicitly do in lines 11-40)
- Recommendation would be wrong if context exists equally well at session end (Capture at Context decision argues it doesn't)
- Claim of "mostly aligned" would be wrong if more misaligned patterns exist in skill guidance (would require deeper audit of all skill files)

---

## Implementation Recommendations

**Purpose:** Address the one misalignment found (investigation promotion gate) and strengthen existing well-aligned patterns.

### Recommended Approach ⭐

**Move investigation promotion trigger to completion time** - Prompt for promotion decision during `orch complete` when investigation context is fresh, not accumulated until session end.

**Why this approach:**
- Aligns with Capture at Context: promotion decision happens when context exists (just read SYNTHESIS.md)
- Follows existing pattern: UpdateHandoffAfterComplete already prompts during completion for related decisions
- Reduces cognitive load: one decision at a time (this investigation) vs batch triage (5+ investigations)
- Prevents accumulation: backlog never forms if decisions happen progressively

**Trade-offs accepted:**
- Adds prompt to `orch complete` flow (acceptable - orchestrator is already engaged with artifact)
- Per-investigation decision vs batch processing (acceptable - batch processing was the problem)

**Implementation sequence:**
1. **Add promotion prompt to orch complete** (~1-2 hours)
   - After SYNTHESIS.md verification passes, check investigation file metadata
   - If investigation has recommendation section, prompt: "Promote to decision/guide? [y/N/later]"
   - Record choice in investigation metadata or kb quick
   
2. **Remove session-end investigation gate** (~30 min)
   - Remove `gateInvestigationPromotions()` call from session end
   - Gate no longer needed - decisions happen progressively
   
3. **Update orchestrator skill guidance** (~30 min)
   - Document new completion flow: verification → promotion decision → handoff update
   - Clarify WHEN promotion decisions should be made (during completion, not session end)

### Alternative Approaches Considered

**Option B: Keep session-end gate, add completion prompt too**
- **Pros:** Defense in depth, catches any missed during completion
- **Cons:** Redundant; if completion prompt works, session-end gate is unnecessary noise
- **When to use instead:** If completion prompt adoption is low after 2-4 weeks

**Option C: Make kb reflect show promotion candidates in SessionStart hook**
- **Pros:** Surfaces at session start when orchestrator is planning
- **Cons:** Context still decayed (not immediately after reading investigation); adds to session-start cognitive load
- **When to use instead:** As supplement to completion prompt, not replacement

**Option D: Automated promotion based on investigation metadata**
- **Pros:** Zero orchestrator cognitive load
- **Cons:** Loses strategic judgment (not all investigations with recommendations should promote); automation would need ML or complex heuristics
- **When to use instead:** After 6+ months if promotion decisions become formulaic

**Rationale for recommendation:** Option A (completion-time prompt) is the minimal change that aligns temporal placement with context availability. Follows existing UpdateHandoffAfterComplete pattern. Options B-D are either redundant (B), partially helpful (C), or over-engineered (D).

---

### Implementation Details

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- `cmd/orch/session.go:624-752` - Investigation promotion gate implementation
- `cmd/orch/session.go:1827-2104` - UpdateHandoffAfterComplete hook
- `cmd/orch/complete_cmd.go:48-294` - Completion gate definitions
- `pkg/verify/phase_gates.go:1-194` - Phase gate verification logic
- `.orch/templates/SESSION_HANDOFF.md:11-40` - Progressive capture guidance
- `.orch/templates/SYNTHESIS.md:11-39` - Worker progressive capture guidance
- `.claude/skills/worker/codebase-audit/SKILL.md:737-738` - "Remember to" anti-pattern detection

**Commands Run:**
```bash
# Search for gate patterns
rg "(gate|Gate|GATE)" --type go

# Search for hook patterns
rg "(hook|Hook|HOOK)" --type go

# Search for progressive documentation patterns
rg "(progressive|as you work|fill.*throughout)" --type md

# Search for reminder anti-patterns
rg "(remember to|don't forget|make sure to)" --type md .claude/skills/
```

**Related Artifacts:**
- **Decision:** `~/.kb/principles.md` (Capture at Context principle) - Foundation for this audit
- **Decision:** `.kb/decisions/2026-01-14-capture-at-context.md` - Principle definition and derivability reasoning
- **Investigation:** `.kb/investigations/2026-01-16-inv-audit-sessionstart-hooks-claude-code.md` - SessionStart hook timing analysis
- **Decision:** `.kb/decisions/2026-01-17-role-aware-hook-filtering.md` - Hook temporal alignment example

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
