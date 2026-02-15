<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Auto-rebuild warnings break JSON parsing when stderr is captured together with stdout; fixed by suppressing warnings when --json flag is detected.

**Evidence:** Unit tests pass; manual test shows `orch status --json 2>&1` produces clean JSON without warning contamination.

**Knowledge:** JSON output requires clean stdout; warnings must be suppressed (not just moved to stderr) when JSON parsers capture both streams together.

**Next:** Fix is implemented and tested; ready for commit and completion.

**Promote to Decision:** recommend-no (tactical bug fix, not an architectural pattern or constraint worth preserving)

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

# Investigation: Fix Bug Orch Status Json

**Question:** How to fix the bug where orch status --json outputs warning lines that break JSON parsing?

**Started:** 2026-01-09
**Updated:** 2026-01-09
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Auto-rebuild warning is already sent to stderr, not stdout

**Evidence:** The warning in autorebuild.go:155-156 uses `fmt.Fprintf(os.Stderr, ...)` which correctly sends output to stderr, not stdout.

**Source:** /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/autorebuild.go:155-156

**Significance:** The issue is that JSON parsers like jq break when users capture both stdout and stderr together (e.g., `2>&1 | jq`), mixing the warning with JSON output.

---

### Finding 2: maybeAutoRebuild() is called before flag parsing

**Evidence:** maybeAutoRebuild() is called at main.go:29, before rootCmd.Execute(), which means it runs before we know if the user specified --json.

**Source:** /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go:29

**Significance:** We cannot use cobra's flag parsing to detect --json, so we must scan os.Args directly to detect the flag.

---

### Finding 3: Warning only appears when rebuild fails

**Evidence:** The warning is only printed when autoRebuildAndReexec() returns an error (lines 153-157), which happens when the binary is stale AND rebuild fails for some reason (e.g., already in progress, build error).

**Source:** /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/autorebuild.go:153-157

**Significance:** This is an edge case that only occurs when auto-rebuild is triggered and fails, but it breaks JSON output when it does occur.

---

## Synthesis

**Key Insights:**

1. **Warning is already on stderr, but parsers capture both streams** - The auto-rebuild warning correctly goes to stderr, but JSON parsers often capture both stdout and stderr together (e.g., `2>&1 | jq`), causing the warning text to mix with JSON output.

2. **Flag detection must happen before main() executes commands** - Since maybeAutoRebuild() runs before rootCmd.Execute(), we cannot use cobra's flag parsing. We must scan os.Args directly for the --json flag.

3. **Conditional warning suppression is the cleanest fix** - Rather than moving when maybeAutoRebuild() is called or changing the entire architecture, we simply suppress the warning when --json is detected in os.Args.

**Answer to Investigation Question:**

The bug is fixed by adding a hasJSONFlag() helper that scans os.Args for "--json", and conditionally suppressing the auto-rebuild failure warning when JSON output is requested. This prevents the warning from contaminating JSON output when users capture both stdout and stderr together. The fix is minimal, surgical, and preserves all existing behavior while solving the reported issue.

---

## Structured Uncertainty

**What's tested:**

- ✅ [Claim with evidence of actual test performed - e.g., "API returns 200 (verified: ran curl command)"]
- ✅ [Claim with evidence of actual test performed]
- ✅ [Claim with evidence of actual test performed]

**What's untested:**

- ⚠️ [Hypothesis without validation - e.g., "Performance should improve (not benchmarked)"]
- ⚠️ [Hypothesis without validation]
- ⚠️ [Hypothesis without validation]

**What would change this:**

- [Falsifiability criteria - e.g., "Finding would be wrong if X produces different results"]
- [Falsifiability criteria]
- [Falsifiability criteria]

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Conditional warning suppression via os.Args scan** - Detect --json flag in os.Args before printing auto-rebuild warnings and suppress output when JSON mode is active.

**Why this approach:**
- Minimal code change - only adds a helper function and one conditional check
- Preserves all existing behavior for non-JSON commands
- Solves the problem at the source without architectural changes
- No performance impact (one-time os.Args scan at startup)

**Trade-offs accepted:**
- Warnings are completely suppressed with --json (not redirected elsewhere)
- Only checks for exact "--json" string (not "-j" or other variants, but orch doesn't have those)

**Implementation sequence:**
1. Add hasJSONFlag() helper to scan os.Args for "--json"
2. Modify maybeAutoRebuild() to check hasJSONFlag() before printing warning
3. Add unit test to verify flag detection works correctly

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
- Add hasJSONFlag() helper function (foundational utility)
- Modify maybeAutoRebuild() to use hasJSONFlag() (implements the fix)
- Add unit test for hasJSONFlag() (verifies correctness)

**Things to watch out for:**
- ⚠️ Warning is only suppressed, not logged elsewhere - user won't know rebuild failed when using --json
- ⚠️ Only exact "--json" string is detected, but this matches all current orch commands with --json flag
- ⚠️ This only affects auto-rebuild warnings, not other warnings

**Areas needing further investigation:**
- None - the fix is complete and solves the reported issue

**Success criteria:**
- ✅ Unit tests pass for hasJSONFlag() detection
- ✅ All existing autorebuild tests still pass
- ✅ Manual smoke test: running `orch status --json 2>&1` produces clean JSON without warnings

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
