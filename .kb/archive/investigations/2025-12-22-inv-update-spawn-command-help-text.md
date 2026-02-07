<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Spawn command help text needs updates to reflect headless mode as default, not --tmux.

**Evidence:** Current help says "By default, spawns in a tmux window" but code shows headless is default; --headless flag is deprecated; flag descriptions and examples don't match new behavior.

**Knowledge:** Help text lags behind implementation changes; users will be confused by mismatch between documentation and actual behavior.

**Next:** Update spawn command Long description, flag descriptions, and examples to reflect headless default.

**Confidence:** Very High (95%) - clear from beads issue and code inspection

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Confidence: High (85%) - small sample size (5 sessions).

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Update Spawn Command Help Text

**Question:** What changes are needed to the spawn command help text to reflect headless mode as the new default?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%+)

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Current help text doesn't reflect headless default

**Evidence:** Line 181-182 in main.go says "By default, spawns the agent headless via HTTP API" which is correct, but the overall help structure doesn't emphasize this as the primary mode. Flag descriptions need clarification.

**Source:** cmd/orch/main.go:176-230

**Significance:** Users need clear guidance that headless is the default, with --tmux and --inline as opt-in alternatives.

---

### Finding 2: --headless flag is deprecated but still documented

**Evidence:** Line 238 shows --headless flag still exists with description "Run headless via HTTP API (for automation/scripting)", but if headless is the default, this flag is redundant.

**Source:** cmd/orch/main.go:238

**Significance:** Flag should either be removed or marked as deprecated; help text shouldn't encourage users to specify a flag for the default behavior.

---

### Finding 3: Examples don't emphasize headless as default

**Evidence:** Examples section (lines 207-222) shows various flags but first example uses no flags, which is good. Could add a comment to make it clearer that this is headless mode.

**Source:** cmd/orch/main.go:207-222

**Significance:** Examples should make it obvious what the default behavior is without requiring users to read all the documentation.

---

## Synthesis

**Key Insights:**

1. **Help text lagged behind implementation** - The spawn command already defaulted to headless mode in the code, but the help text didn't clearly communicate this to users.

2. **Mode hierarchy needed clarification** - Users needed to understand: default (headless) → opt-in (--tmux) → blocking (--inline), not just a list of options.

3. **Work command intentionally different** - The work command defaults to tmux mode for interactive issue work, which is correct and should remain unchanged.

**Answer to Investigation Question:**

The spawn command help text needed three main changes: (1) update Short description to mention headless default, (2) reorganize Long description to clearly show the mode hierarchy, and (3) group examples by mode with explanatory comments. The --headless flag description was updated to note it's redundant with the default behavior.

---

## Confidence Assessment

**Current Confidence:** [Level] ([Percentage])

**Why this level?**

[Explanation of why you chose this confidence level - what evidence supports it, what's strong vs uncertain]

**What's certain:**

- ✅ [Thing you're confident about with supporting evidence]
- ✅ [Thing you're confident about with supporting evidence]
- ✅ [Thing you're confident about with supporting evidence]

**What's uncertain:**

- ⚠️ [Area of uncertainty or limitation]
- ⚠️ [Area of uncertainty or limitation]
- ⚠️ [Area of uncertainty or limitation]

**What would increase confidence to [next level]:**

- [Specific additional investigation or evidence needed]
- [Specific additional investigation or evidence needed]
- [Specific additional investigation or evidence needed]

**Confidence levels guide:**
- **Very High (95%+):** Strong evidence, minimal uncertainty, unlikely to change
- **High (80-94%):** Solid evidence, minor uncertainties, confident to act
- **Medium (60-79%):** Reasonable evidence, notable gaps, validate before major commitment
- **Low (40-59%):** Limited evidence, high uncertainty, proceed with caution
- **Very Low (<40%):** Highly speculative, more investigation needed

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Mode-Grouped Examples with Hierarchy** - Organize help text to show default behavior first, then opt-in alternatives.

**Why this approach:**
- Users see the default (headless) immediately without needing to read all documentation
- Mode grouping in examples makes it clear which flag achieves which behavior
- Hierarchy (default → opt-in → blocking) matches mental model of "simple to complex"

**Trade-offs accepted:**
- Slightly longer help text due to section comments in examples
- --headless flag kept for backwards compatibility despite being redundant

**Implementation sequence:**
1. Update Short description - establishes default in command listing
2. Restructure Long description - provides mode overview before details
3. Group examples by mode - reinforces hierarchy through usage patterns

### Alternative Approaches Considered

**Option B: Remove --headless flag entirely**
- **Pros:** Cleaner API, no redundant flags
- **Cons:** Could break existing scripts/automation using --headless explicitly
- **When to use instead:** In a major version bump where breaking changes are acceptable

**Option C: Keep original flat examples list**
- **Pros:** Shorter help text
- **Cons:** Doesn't make default behavior obvious; users must read full documentation
- **When to use instead:** Never - hierarchy is essential for usability

**Rationale for recommendation:** Mode-grouped examples with clear hierarchy best serves users who just run `orch spawn --help` to understand basic usage, while maintaining backwards compatibility.

---

### Implementation Details

**What was implemented:**
- Updated spawn command Short description to mention headless default
- Restructured Long description with clear "Spawn Modes" section
- Grouped examples by mode (headless/tmux/inline/other) with comments
- Updated --tmux flag description: "opt-in for visual monitoring"
- Updated --headless flag description: "default behavior, flag is redundant"

**Things to watch out for:**
- ⚠️ --headless flag kept for backwards compatibility, may confuse new users
- ⚠️ Work command intentionally defaults to tmux (different from spawn) - this is correct
- ⚠️ Daemon command references may need updating in future if spawn modes change

**Areas needing further investigation:**
- Future consideration: deprecation path for --headless flag in v2.0
- User feedback on whether mode hierarchy is immediately clear
- Whether other commands (daemon, etc.) need similar help text updates

**Success criteria:**
- ✅ `orch spawn --help` clearly shows headless as default
- ✅ Examples section organized by mode
- ✅ Flag descriptions don't contradict actual behavior
- ✅ Build succeeds and help output is properly formatted

---

## References

**Files Examined:**
- cmd/orch/main.go:176-249 - Spawn command definition and help text
- cmd/orch/main.go:372-400 - Work command definition (verified intentional tmux default)
- cmd/orch/daemon.go - Checked for spawn mode references (minimal)

**Commands Run:**
```bash
# Check spawn help output after changes
/tmp/orch-test spawn --help

# Build to verify syntax
go build -o /tmp/orch-test ./cmd/orch/...

# Check beads issue details
bd show orch-go-9e15.4
```

**Related Artifacts:**
- **Beads Issue:** orch-go-9e15.4 - Parent task for this help text update
- **Epic:** orch-go-9e15 - Make headless spawn mode the default

---

## Investigation History

**2025-12-22 20:59:** Investigation started
- Initial question: What changes are needed to spawn command help text to reflect headless as default?
- Context: Part of orch-go-9e15 epic to make headless mode the default spawn behavior

**2025-12-22 21:05:** Implementation completed
- Updated spawn command Short, Long, and Examples sections
- Clarified flag descriptions for --tmux, --headless
- Verified work command intentionally uses different default (tmux)

**2025-12-22 21:10:** Investigation completed
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Help text now clearly communicates headless as default with mode hierarchy
