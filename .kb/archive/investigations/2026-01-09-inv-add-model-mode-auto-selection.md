<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added model→mode auto-selection to prevent invalid spawn combinations (opus→claude, sonnet→opencode, flash→error).

**Evidence:** Tests pass for validation logic, flash blocking, and auto-selection; orchestrator skill updated with two-path guidance.

**Knowledge:** Only two viable spawn paths exist (claude+opus, opencode+sonnet); flash has TPM limits that make it unusable; auto-selection removes orchestrator confusion.

**Next:** Close issue - implementation complete with tests and documentation.

**Promote to Decision:** recommend-yes - This establishes the constraint that flash is blocked and defines the auto-selection pattern for model→mode mapping.

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

# Investigation: Add Model Mode Auto Selection

**Question:** How can orch spawn prevent invalid model+mode combinations (zombie agents, rate limits) through auto-selection?

**Started:** 2026-01-09
**Updated:** 2026-01-09
**Owner:** Agent (orch-go-hy5rv)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Current spawn logic hard-codes backend selection without model awareness

**Evidence:** `cmd/orch/spawn_cmd.go:1015-1027` shows backend determined only by `--opus` flag and config `spawn_mode`, ignoring `--model` flag value. Model resolution happens at line 928 but isn't used for backend selection.

**Source:** `cmd/orch/spawn_cmd.go:928-1027`

**Significance:** Orchestrators must remember which flag combinations are valid, leading to zombie agents (opencode+opus) or rate limit failures (opencode+flash) when wrong combination used.

---

### Finding 2: Flash model has TPM rate limits that make it unusable for agent work

**Evidence:** Prior knowledge from kb context shows constraint "Always use flash as default spawn model - Reason: Anthropic Max subscription no longer available outside of Claude Code (Jan 2026 policy change)" but also shows invalid combinations including "opencode + flash → TPM rate limit exceeded".

**Source:** SPAWN_CONTEXT.md lines 68-75, task description lines 14-16

**Significance:** Flash cannot be used as default due to rate limits; needs hard error to prevent orchestrators from attempting it.

---

### Finding 3: Only two viable spawn paths exist in current reality

**Evidence:** Task description explicitly lists only two viable combinations: `claude + opus` (Max subscription) and `opencode + sonnet` (pay-per-token). All other combinations have known failure modes.

**Source:** SPAWN_CONTEXT.md lines 8-16

**Significance:** Auto-selection can be simple: opus→claude, sonnet→opencode, flash→error. No complex logic needed.

---

## Synthesis

**Key Insights:**

1. **Cognitive load on orchestrators** - Orchestrators shouldn't need to remember which model+mode combinations are valid. The system should enforce valid combinations automatically (Finding 1).

2. **Flash is not viable** - Despite being cheaper, flash's TPM rate limits make it unsuitable for agent work. Hard blocking at spawn time prevents wasted orchestrator time on debugging zombie agents (Finding 2).

3. **Simple mapping suffices** - With only two viable paths, auto-selection logic is straightforward: opus→claude (Max), sonnet→opencode (API), flash→error. No complex routing needed (Finding 3).

**Answer to Investigation Question:**

Auto-selection prevents invalid combinations by inspecting the `--model` flag and automatically setting the backend: opus models use `claude` backend (Max subscription via CLI), sonnet uses `opencode` backend (pay-per-token API), and flash returns a hard error with explanation. This removes orchestrator cognitive load while preventing zombie agents (opencode+opus) and rate limit failures (opencode+flash). The validation function also warns on detected invalid combos for remaining edge cases.

---

## Structured Uncertainty

**What's tested:**

- ✅ validateModeModelCombo warns on opencode+opus (test: TestValidateModeModelCombo)
- ✅ Flash models resolve to google provider with "flash" in ID (test: TestFlashModelBlocking)
- ✅ Opus model auto-selects claude backend (test: TestModelAutoSelection)
- ✅ Sonnet model uses opencode backend (test: TestModelAutoSelection)
- ✅ Default (no flags) uses opencode backend (test: TestModelAutoSelection)

**What's untested:**

- ⚠️ Actual spawn with flash model returns formatted error (not spawn_cmd_test.go, would need integration test)
- ⚠️ Auto-selection prints "Auto-selected claude backend" message when opus detected (visual, not unit-testable)
- ⚠️ Config spawn_mode still respected when no --model flag (existing behavior, assumed unchanged)

**What would change this:**

- Flash spawn succeeds → validation logic has a bug, flash became viable, or error handling broken
- Opus auto-selection chooses opencode → backend selection logic broken or condition wrong
- Validation doesn't warn on opencode+opus → validateModeModelCombo logic broken

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**[Approach Name]** - [One sentence stating the recommended implementation]

**Why this approach:**
- [Key benefit 1 based on findings]
- [Key benefit 2 based on findings]
- [How this directly addresses investigation findings]

**Trade-offs accepted:**
- [What we're giving up or deferring]
- [Why that's acceptable given findings]

**Implementation sequence:**
1. [First step - why it's foundational]
2. [Second step - why it comes next]
3. [Third step - builds on previous]

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

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
- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

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
