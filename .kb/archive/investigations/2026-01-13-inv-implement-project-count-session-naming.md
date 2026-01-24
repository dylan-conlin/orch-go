<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented {project}-{count} session naming with automatic tmux window renaming for consistent, discoverable session directories.

**Evidence:** Tests pass for counting logic (handles mixed projects, non-sequential numbers, empty directories), manual testing shows window renamed to "orch-go-1" and directory created at `.orch/session/orch-go-1/active/`.

**Knowledge:** Counting highest existing number (vs daily reset) avoids naming collisions while maintaining pattern; window renaming enables seamless integration with existing discovery logic.

**Next:** Commit changes, close issue.

**Promote to Decision:** recommend-no (implementation detail, not architectural choice)

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

# Investigation: Implement Project Count Session Naming

**Question:** How to implement {project}-{count} session naming with automatic tmux window renaming?

**Started:** 2026-01-13
**Updated:** 2026-01-13
**Owner:** orch-go-j054q
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Current Session Window Naming is Ad-Hoc

**Evidence:** Current session directories use tmux window names directly:
```
.orch/session/zsh/
.orch/session/test-session/
.orch/session/session/
.orch/session/pw/
.orch/session/og-debug-session-end-blocks-13jan-1f8a-orch-go-3q4y3/
```

**Source:**
- `ls -la .orch/session/` output
- `cmd/orch/session.go:161` - `windowName, err := tmux.GetCurrentWindowName()`
- `cmd/orch/session.go:167` - `activeDir := filepath.Join(projectDir, ".orch", "session", windowName, "active")`

**Significance:** Window names are unpredictable and not discoverable. "session", "zsh", "test-session" provide no information about which project or when the session occurred.

---

### Finding 2: Session Archiving Uses Timestamp Format

**Evidence:** When sessions end, active/ is archived to timestamped directories:
```
.orch/session/{window}/2026-01-13-0827/
.orch/session/{window}/2026-01-13-1000/
.orch/session/{window}/latest -> 2026-01-13-1000
```

**Source:**
- `cmd/orch/session.go:218` - `timestamp := time.Now().Format("2006-01-02-1504")`
- `cmd/orch/session.go:219` - `timestampedDir := filepath.Join(projectDir, ".orch", "session", windowName, timestamp)`

**Significance:** Archiving uses timestamp, not window name. This is separate from the window naming issue.

---

### Finding 3: No tmux Rename Function Exists

**Evidence:** No existing function in pkg/tmux/tmux.go to rename windows. The package has functions for:
- Creating windows (`CreateWindow`)
- Getting current window name (`GetCurrentWindowName`)
- Killing windows (`KillWindow`, `KillWindowByID`)

But no `RenameWindow` function.

**Source:**
- `rg "rename.*window" --type go` returns no results
- Manual review of `pkg/tmux/tmux.go`

**Significance:** Need to implement a new `RenameWindow` function that executes `tmux rename-window`.

---

## Synthesis

**Key Insights:**

1. **Session naming is decoupled from tmux windows** - The current implementation uses ad-hoc tmux window names as session directory names, creating unpredictable paths. Auto-generating names as {project}-{count} provides consistent, discoverable naming across projects and sessions.

2. **Simple counting approach is sufficient** - Instead of daily resets (which would cause naming collisions), counting all existing {project}-{number} directories and using the next available number provides uniqueness while maintaining the desired pattern.

3. **Window renaming enables seamless integration** - By renaming the tmux window to match the generated session name, all existing code that queries the current window name automatically works with the new naming scheme.

**Answer to Investigation Question:**

Implemented {project}-{count} session naming with auto tmux rename. Solution adds:
- `GenerateSessionName()` function to extract project name and count existing sessions
- `RenameCurrentWindow()` function to execute `tmux rename-window`
- Modified `runSessionStart()` to generate name, rename window, and use name for directory creation

Count is based on highest existing number for the project (not daily reset), ensuring unique names. Tests verify correct counting across various scenarios including mixed projects and non-sequential numbers.

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
