<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Session start hook loaded parent/global handoffs because `discoverSessionHandoff()` walked to filesystem root without respecting project boundaries.

**Evidence:** Tests show child project with `.orch/` now stops at project boundary (exit code 1), while child without `.orch/` correctly finds parent handoff (backward compatibility maintained).

**Knowledge:** Project boundaries are indicated by `.orch/` directory existence; discovery logic must check for project root and stop walking up after exhausting all session handoff locations within that project.

**Next:** Close issue - fix implemented, tested, and verified to solve reproduction case without breaking backward compatibility.

**Promote to Decision:** recommend-no - This is a bug fix with clear isolated scope, not an architectural pattern worth preserving as decision.

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

# Investigation: Session Start Hook Loads Orch

**Question:** Why does the session start hook load `.orch/session/latest` (global) instead of project-specific `.orch/session/{project}/latest`?

**Started:** 2026-01-14
**Updated:** 2026-01-14
**Owner:** Agent orch-go-wqzp8
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Discovery Logic Walks to Filesystem Root

**Evidence:** The `discoverSessionHandoff()` function walks up from the current directory to the filesystem root, checking for `.orch/session/` at every level. Lines 777-870 show the loop continues until `parent == dir` (filesystem root).

**Source:**
- `cmd/orch/session.go:763-876` - `discoverSessionHandoff()` function
- Lines 863-869 show walk continues to parent directory without stopping at project boundary

**Significance:** This causes the hook to find `.orch/session/latest` in parent directories (like `~/.orch/session/latest`), not just the current project's session directory. When working in a nested project, it may find a global or parent project's session handoff instead of the current project's.

---

### Finding 2: No Project Root Detection

**Evidence:** The discovery logic does not detect when it has reached a project root (indicated by `.orch/` directory). It continues walking up regardless of whether `.orch/` exists in the current directory.

**Source:**
- `cmd/orch/session.go:777-870` - No check for `.orch/` directory existence before continuing walk
- Lines 779-806 check for session files but don't stop the walk if `.orch/` exists without session handoffs

**Significance:** Once we find a `.orch/` directory, that's the project root - we should check for session handoffs there and stop, not continue walking up to find other projects' handoffs.

---

### Finding 3: Hooks Call orch session resume Without Working Directory Control

**Evidence:**
- Claude Code hook (session-start.sh:10) calls `orch session resume` without setting working directory
- OpenCode plugin (session-resume.js:37) passes `cwd` parameter, which is better
- Both rely on the discovery logic to find the right handoff

**Source:**
- `~/.claude/hooks/session-start.sh:10` - `HANDOFF=$(orch session resume --for-injection 2>/dev/null)`
- `~/.config/opencode/plugin/session-resume.js:37` - `const result = await execAsync('orch session resume --for-injection', { cwd });`

**Significance:** The bash hook uses the shell's current working directory, which may not be the project root. The OpenCode plugin is more reliable since it passes the session's working directory explicitly.

---

## Synthesis

**Key Insights:**

1. **Project Boundary Not Respected** - The discovery logic treats all `.orch/` directories equally, walking through parent directories without recognizing project boundaries. This violates the principle that session handoffs are project-specific.

2. **Root Cause is Over-Aggressive Discovery** - The bug occurs because `discoverSessionHandoff()` walks from current directory all the way to filesystem root, checking every parent for `.orch/session/`. Once it finds a `.orch/` directory, it should stop there (project boundary).

3. **Fix is Simple: Stop at First .orch/** - When walking up the tree, once we find a directory with `.orch/` subdirectory, that's the project root. Check for session handoffs there and stop - don't continue to parent directories.

**Answer to Investigation Question:**

The session start hook loads global `.orch/session/latest` instead of project-specific paths because `discoverSessionHandoff()` walks all the way to the filesystem root without respecting project boundaries. When it finds `~/.orch/session/latest` (or any parent project's session directory), it returns that instead of the current project's handoff. The fix is to stop the directory walk once we find a `.orch/` directory - that's the project root, and we should only look for session handoffs within that project's `.orch/session/` directory.

---

## Structured Uncertainty

**What's tested:**

- ✅ Child project WITH `.orch/` directory stops at project boundary (verified: created parent handoff, child with .orch/ returns exit code 1)
- ✅ Child directory WITHOUT `.orch/` continues walking up to find parent (verified: child without .orch/ finds parent handoff)
- ✅ Fix compiles and installs successfully (verified: make build && make install)
- ✅ Current project handoff still loads correctly (verified: orch session resume shows "Fix price-watch dashboard" handoff)

**What's untested:**

- ⚠️ Behavior with deeply nested projects (3+ levels)
- ⚠️ Edge case: symlinked `.orch/` directories
- ⚠️ Performance impact of additional stat() call per directory

**What would change this:**

- Finding would be wrong if child project with `.orch/` directory found parent handoff (would indicate project boundary check failed)
- Finding would be wrong if child directory without `.orch/` couldn't find parent handoff (would indicate backward compatibility broken)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Stop Directory Walk at Project Root** - Modify `discoverSessionHandoff()` to stop walking up when it finds a directory containing `.orch/` subdirectory (the project root).

**Why this approach:**
- Respects project boundaries - each project's `.orch/session/` is isolated from parent directories
- Simple to implement - add one check before continuing the walk
- Fixes the exact bug - prevents finding `~/.orch/session/latest` or parent project handoffs

**Trade-offs accepted:**
- Requires `.orch/` directory to exist in project root (already a requirement for orch-managed projects)
- Won't fall back to parent project handoffs (this is the desired behavior - we want isolation)

**Implementation sequence:**
1. Check if `.orch/` directory exists in current directory during walk
2. If it exists, perform all session handoff checks in that project root
3. Stop the walk after checking that project root - don't continue to parent directories

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
