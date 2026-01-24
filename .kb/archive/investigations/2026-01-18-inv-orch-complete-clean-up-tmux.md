<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Tmux windows remain open for orchestrator sessions because cleanup code uses `FindWindowByBeadsIDAllSessions` which searches for beads ID patterns that don't exist in orchestrator window names.

**Evidence:** Examined complete_cmd.go:997-1013 and tmux.go:802-818 showing orchestrator sessions use workspace names (agentName) but cleanup searches for `[beadsID]` pattern in window names.

**Knowledge:** Orchestrator windows use format "og-orch-goal-date" without beads IDs while worker windows use "og-inv-topic-date [beads-id]" - cleanup code must use different search functions for each type.

**Next:** Implement conditional window search - use `FindWindowByWorkspaceNameAllSessions` for orchestrators and `FindWindowByBeadsIDAllSessions` for workers.

**Promote to Decision:** recommend-no (tactical bug fix, not architectural pattern)

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

# Investigation: Orch Complete Clean Up Tmux

**Question:** Why do tmux windows remain open after `orch complete` and `orch abandon` for orchestrator sessions?

**Started:** 2026-01-18
**Updated:** 2026-01-18
**Owner:** architect agent
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

### Finding 1: Complete and abandon commands have tmux cleanup code

**Evidence:** Both `complete_cmd.go` (lines 997-1013) and `abandon_cmd.go` (lines 202-207) contain code to find and kill tmux windows. The complete command calls `tmux.FindWindowByBeadsIDAllSessions` followed by `tmux.KillWindow`. The abandon command calls `tmux.KillWindow` if a window is found.

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/complete_cmd.go:997-1013`
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/abandon_cmd.go:202-207`

**Significance:** The cleanup code exists but is not working correctly for orchestrator sessions, indicating the bug is in how windows are located, not in missing cleanup logic.

---

### Finding 2: Orchestrator windows use workspace names, not beads IDs

**Evidence:** In `complete_cmd.go` lines 998-1006, for orchestrator sessions the code sets `windowSearchID = agentName` (workspace name like "og-orch-goal-04jan"). However, the cleanup code then calls `FindWindowByBeadsIDAllSessions(windowSearchID)` which searches for the pattern `[beadsID]` in window names. Orchestrator windows don't have beads IDs in their names - they only contain the workspace name.

**Source:**
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/complete_cmd.go:998-1006`
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/tmux/tmux.go:802-818` (FindWindowByBeadsID searches for `[beadsID]` pattern)

**Significance:** The window search function doesn't match orchestrator window naming patterns, causing the cleanup to silently skip closing the window. This is the root cause of the bug.

---

### Finding 3: A workspace name search function already exists

**Evidence:** The tmux package provides `FindWindowByWorkspaceNameAllSessions` (lines 839-869) which searches for windows containing a workspace name. This function searches all workers sessions, the orchestrator session, and the meta-orchestrator session.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/tmux/tmux.go:839-869`

**Significance:** The fix is straightforward - use the correct search function for orchestrator sessions instead of trying to search by beads ID.

---

## Synthesis

**Key Insights:**

1. **Cleanup code exists but uses wrong search function** - Both complete and abandon commands have tmux cleanup logic, but for orchestrator sessions they use `FindWindowByBeadsIDAllSessions` which searches for `[beadsID]` patterns. Orchestrator windows don't have beads IDs in their names.

2. **Window naming patterns differ by session type** - Worker windows use format `🔬 og-inv-topic-date [beads-id]` while orchestrator windows use format `⚙️ og-orch-goal-date` without beads IDs. The cleanup code doesn't account for this difference.

3. **The correct function exists but isn't used** - The tmux package already provides `FindWindowByWorkspaceNameAllSessions` which searches by workspace name instead of beads ID. This would work for both orchestrator and worker windows.

**Answer to Investigation Question:**

Tmux windows remain open for orchestrator sessions because `orch complete` and `orch abandon` use `FindWindowByBeadsIDAllSessions` to locate windows, which searches for beads IDs in window names. Orchestrator windows only contain workspace names (like "og-orch-goal-04jan"), not beads IDs, so the search fails and cleanup is skipped. The fix is to use `FindWindowByWorkspaceNameAllSessions` for orchestrator sessions.

---

## Structured Uncertainty

**What's tested:**

- ✅ Code compiles without errors (verified: ran `go build ./cmd/orch`)
- ✅ Window search functions exist with correct signatures (verified: read tmux.go source)
- ✅ Orchestrator detection logic exists (verified: found `isOrchestratorWorkspace` usage)

**What's untested:**

- ⚠️ Fix works for live orchestrator sessions (not manually tested with running orchestrator)
- ⚠️ Fix works for workers spawned with --tmux flag (assumed pattern matches)
- ⚠️ Edge cases like orchestrator windows with non-standard names

**What would change this:**

- Finding would be wrong if orchestrator windows actually do contain beads IDs in their names
- Finding would be wrong if `FindWindowByWorkspaceNameAllSessions` doesn't search orchestrator/meta-orchestrator sessions
- Fix would fail if window naming pattern changes in the future

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Use workspace name search for orchestrator sessions** - For orchestrator sessions, use `FindWindowByWorkspaceNameAllSessions` instead of `FindWindowByBeadsIDAllSessions` to locate tmux windows before cleanup.

**Why this approach:**
- Matches actual orchestrator window naming patterns (workspace name only, no beads ID)
- Reuses existing, tested function in the tmux package
- Minimal code change - only affects window lookup logic
- Works for both orchestrator and meta-orchestrator sessions

**Trade-offs accepted:**
- None - this is a straightforward bug fix with no downsides

**Implementation sequence:**
1. In `complete_cmd.go` lines 997-1013, replace the single `FindWindowByBeadsIDAllSessions` call with conditional logic based on `isOrchestratorSession`
2. For orchestrator sessions, call `FindWindowByWorkspaceNameAllSessions(agentName)`
3. For worker sessions, keep existing `FindWindowByBeadsIDAllSessions(beadsID)` call
4. Verify both code paths still properly kill windows when found

### Alternative Approaches Considered

**Option B: Modify FindWindowByBeadsIDAllSessions to also search by workspace name**
- **Pros:** Single function call, no conditional logic
- **Cons:** Conflates two different search strategies; reduces clarity; would need to search twice (once for beads ID pattern, once for workspace name)
- **When to use instead:** Never - the current design with separate functions is clearer

**Rationale for recommendation:** The recommended approach directly addresses the root cause (wrong search function for orchestrator windows) with minimal code change and no architectural complications.

---

### Implementation Details

**What to implement first:**
- Modify complete_cmd.go to use conditional window search (workspace name for orchestrators)
- Add orchestrator workspace name search fallback to abandon_cmd.go
- Test compilation to ensure no syntax errors

**Things to watch out for:**
- ⚠️ Variable shadowing with `err` variable (use separate name like `findErr` or `tmuxSessionName`)
- ⚠️ Orchestrator detection must happen before window search in abandon command
- ⚠️ Both complete and abandon need the fix (different code structures)

**Areas needing further investigation:**
- Should we add automated tests for tmux cleanup?
- Are there other commands with similar orchestrator vs worker handling issues?
- Should window naming patterns be documented more explicitly?

**Success criteria:**
- ✅ Code compiles without errors (DONE)
- ✅ Complete command uses workspace name search for orchestrators (DONE)
- ✅ Abandon command has workspace name search fallback (DONE)
- ✅ Manual testing with live orchestrator session would confirm full fix

---

## References

**Files Examined:**
- `cmd/orch/complete_cmd.go:997-1013` - Existing tmux cleanup code that was failing for orchestrators
- `cmd/orch/abandon_cmd.go:148-207` - Window discovery and cleanup in abandon command
- `pkg/tmux/tmux.go:802-869` - Window search functions (by beads ID and workspace name)

**Commands Run:**
```bash
# Test compilation after changes
go build ./cmd/orch
# PASS: code compiles without errors

# Check git status
git status
```

**Related Artifacts:**
- **Workspace:** `.orch/workspace/og-arch-orch-complete-clean-18jan-e191/` - This session's workspace
- **Issue:** `orch-go-gdcp9` - Bug report for tmux cleanup failure

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
