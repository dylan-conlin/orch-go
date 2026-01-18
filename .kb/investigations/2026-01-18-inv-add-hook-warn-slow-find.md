<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Created a PreToolUse hook plugin that warns agents about slow `find` commands before execution.

**Evidence:** Plugin implements Context Injection pattern from guarded-files.ts, tested with 11 test cases covering slow patterns and edge cases (all passing).

**Knowledge:** OpenCode plugins use `tool.execute.before` hook with `noReply: true` pattern to inject non-blocking warnings; regex word boundaries don't work with `/` character so patterns need careful testing.

**Next:** Plugin is implemented and tested; ready for production use; orchestrator will commit and verify in practice.

**Promote to Decision:** recommend-no (tactical implementation following existing patterns, not architectural)

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

# Investigation: Add Hook Warn Slow Find

**Question:** How do we implement a PreToolUse hook to warn agents about slow `find ~/Documents` commands?

**Started:** 2026-01-18
**Updated:** 2026-01-18
**Owner:** Worker agent og-feat-add-hook-warn-18jan-4a19
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

### Finding 1: OpenCode Plugin System Uses Context Injection Pattern for Warnings

**Evidence:** The guarded-files.ts plugin demonstrates the pattern:
- Hooks `tool.execute.before` for the target tool
- Checks conditions (in that case, file path)
- Uses `client.session.prompt()` with `noReply: true` to inject warnings
- Maintains a `Set` to track warned items and avoid spam

**Source:** `/Users/dylanconlin/.config/opencode/plugin/guarded-files.ts` lines 42-94

**Significance:** This is the exact pattern needed for warning about slow find commands - non-blocking, informational, with deduplication.

---

### Finding 2: Plugin Should Be Project-Level, Not Global

**Evidence:** The coaching.ts plugin exists in `.opencode/plugin/` (project-level) and contains project-specific logic.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.opencode/plugin/coaching.ts`

**Significance:** Since slow find warnings are specific to the orch-go project structure and performance characteristics, the plugin should be placed in `.opencode/plugin/` rather than `~/.config/opencode/plugin/`.

---

### Finding 3: Detection Must Check for Absence of -maxdepth

**Evidence:** The problem occurs when commands like `find ~/Documents -name "*.md"` run without depth limits, taking ~32 seconds.

**Source:** SPAWN_CONTEXT.md evidence - timed commands showing 32.4s and 32.9s

**Significance:** The warning should only trigger when the command is both broad (~/Documents, ~/, or ~) AND lacks the `-maxdepth` flag. If an agent already uses `-maxdepth`, they're being careful and shouldn't be warned.

---

## Synthesis

**Key Insights:**

1. **Context Injection Pattern is the Right Tool** - The plugin system provides a non-blocking warning mechanism via `client.session.prompt` with `noReply: true`, which fits perfectly for "Gate Over Remind" without blocking execution.

2. **Project-Level Plugins for Project-Specific Concerns** - Since this warning is specific to orch-go's directory structure and performance characteristics, placing it in `.opencode/plugin/` (project-level) rather than global makes it discoverable and maintainable.

3. **Regex Testing is Essential for Edge Cases** - The initial pattern `/\bfind\s+~\/\b/` failed for `find ~/ -type f` because word boundaries don't work with `/`; testing revealed this and led to the fix using non-capturing groups.

**Answer to Investigation Question:**

Implement a PreToolUse hook by:
1. Creating a TypeScript plugin in `.opencode/plugin/`
2. Hooking `tool.execute.before` for the `bash` tool
3. Extracting command from `output.args.command`
4. Checking patterns: `find ~/Documents`, `find ~/`, `find ~` without `-maxdepth`
5. Using `client.session.prompt` with `noReply: true` to inject warning
6. Maintaining a `Set` for deduplication to avoid spam

The implementation follows the exact pattern from `guarded-files.ts` and has been tested with 11 test cases covering both positive (should warn) and negative (should not warn) scenarios.

---

## Structured Uncertainty

**What's tested:**

- ✅ Plugin syntax is valid TypeScript (verified: bun run loads without errors)
- ✅ Detection logic correctly identifies slow patterns (verified: 11 test cases all passing)
- ✅ Detection logic excludes commands with maxdepth (verified: test cases confirm)
- ✅ Detection logic excludes non-broad paths (verified: test cases for `.` and specific paths)

**What's untested:**

- ⚠️ Plugin actually loads in OpenCode server context (not tested with actual server restart)
- ⚠️ Warning injection works in practice with real agent sessions (not tested end-to-end)
- ⚠️ Deduplication prevents spam across multiple commands (unit tested, not integration tested)
- ⚠️ Memory management of Set clearing works as expected (threshold of 100 not tested)

**What would change this:**

- Finding would be wrong if OpenCode plugin loader fails to load due to import issues
- Finding would be wrong if `client.session.prompt` API has changed and doesn't work as documented
- Finding would be wrong if real `find` commands use variations not covered by regex patterns

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
