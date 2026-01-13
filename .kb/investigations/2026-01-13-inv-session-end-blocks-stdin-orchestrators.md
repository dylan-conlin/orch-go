<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added terminal detection to `orch session end` - non-interactive contexts now skip prompts and create minimal handoffs instead of blocking on stdin.

**Evidence:** Test with `echo "" | orch session end` completed in 1s (previously would block indefinitely), created valid handoff with placeholders, all existing tests pass.

**Knowledge:** Terminal detection via `term.IsTerminal()` is the Go idiom for distinguishing interactive from automated contexts; existing handoff creation already handles empty reflections gracefully.

**Next:** Commit fix, mark complete - this resolves the orchestrator blocking issue.

**Promote to Decision:** recommend-no - This is a straightforward bug fix using standard Go patterns, not an architectural choice.

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

# Investigation: Session End Blocks Stdin Orchestrators

**Question:** Why does `orch session end` block on stdin for orchestrators in non-interactive contexts?

**Started:** 2026-01-13
**Updated:** 2026-01-13
**Owner:** systematic-debugging agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: promptSessionReflection() has no terminal check

**Evidence:**
- `promptSessionReflection()` at cmd/orch/session.go:562 prompts for user input via `readMultiline()`
- `readMultiline()` at line 630 uses `fmt.Scanln(&line)` which blocks waiting for stdin
- No check for whether stdin is a terminal before prompting
- Called from `runSessionEnd()` at line 503

**Source:** cmd/orch/session.go:562-628, 630-644

**Significance:** When orchestrators run `orch session end` in non-interactive contexts (background, automated, no TTY), the command blocks indefinitely waiting for stdin input that will never arrive.

---

### Finding 2: Go provides term.IsTerminal() for TTY detection

**Evidence:**
- Package `golang.org/x/term` provides `IsTerminal(fd)` function
- Can check `term.IsTerminal(int(os.Stdin.Fd()))` to detect if stdin is a terminal
- Standard approach in Go for detecting interactive vs non-interactive contexts

**Source:** golang.org/x/term package documentation

**Significance:** We have a standard, reliable way to detect when prompts should be skipped.

---

### Finding 3: Minimal handoff creation already exists

**Evidence:**
- `createSessionHandoffDirectory()` at line 814 already handles empty reflection fields
- Uses placeholders like `[No summary provided]` when fields are empty (line 838)
- This logic can be reused for non-interactive mode

**Source:** cmd/orch/session.go:814-942, specifically lines 836-864

**Significance:** We can create a valid handoff with empty/placeholder content for non-interactive contexts without duplicating code.

---

## Synthesis

**Key Insights:**

1. **Terminal detection is the correct abstraction** - Using `term.IsTerminal()` to detect TTY provides a standard, reliable way to distinguish interactive from non-interactive contexts. This is the Go idiom for handling this scenario.

2. **Minimal handoff is already supported** - The `createSessionHandoffDirectory()` function already handles empty reflection fields gracefully with placeholders, so non-interactive mode just needs to return an empty reflection.

3. **Active work auto-population works in both modes** - Both interactive and non-interactive modes can auto-populate active agents, preserving useful context even when user input isn't available.

**Answer to Investigation Question:**

`orch session end` blocks on stdin for orchestrators because `promptSessionReflection()` unconditionally calls `readMultiline()` which uses `fmt.Scanln()`, without checking if stdin is a terminal. The fix is to add `term.IsTerminal(int(os.Stdin.Fd()))` check at the start of `promptSessionReflection()` - if stdin is not a terminal (non-interactive context), return an empty reflection immediately. The existing handoff creation logic already handles empty reflections correctly with placeholder text.

---

## Structured Uncertainty

**What's tested:**

- ✅ Non-interactive mode creates handoff without blocking (verified: `echo "" | orch session end` completed in 1s)
- ✅ Minimal handoff has placeholder content (verified: read SESSION_HANDOFF.md, all fields show `[No X provided]`)
- ✅ Build succeeds with term package import (verified: `make build` completed successfully)
- ✅ Existing tests still pass (verified: `go test ./cmd/orch` and `go test ./pkg/session` - all PASS)

**What's untested:**

- ⚠️ Interactive mode still shows prompts correctly (would need manual testing with TTY)
- ⚠️ Behavior when multiple orchestrators end sessions concurrently (edge case)
- ⚠️ Handoff resume after non-interactive session end (assumes minimal handoff is sufficient)

**What would change this:**

- Finding would be wrong if non-interactive mode still blocks (tested - does not block)
- Finding would be wrong if handoff creation fails in non-interactive mode (tested - succeeds)
- Finding would be wrong if term.IsTerminal() returns incorrect values on macOS (standard package, trusted)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Terminal Detection with Fallback** - Check `term.IsTerminal()` at start of `promptSessionReflection()`, return empty reflection if false.

**Why this approach:**
- Uses standard Go idiom for TTY detection (`golang.org/x/term` package)
- Minimal code change - single check at function entry
- Reuses existing placeholder logic in `createSessionHandoffDirectory()`
- Preserves interactive mode exactly as-is for human users

**Trade-offs accepted:**
- Non-interactive handoffs have minimal content (placeholder text)
- No ability to provide custom reflection in automated contexts
- This is acceptable because orchestrators can populate reflection programmatically if needed in future

**Implementation sequence:**
1. Add `golang.org/x/term` import (already in go.mod)
2. Add `term.IsTerminal()` check at start of `promptSessionReflection()`
3. Return empty reflection in non-interactive path (with active agents if any)
4. Interactive path continues unchanged

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
