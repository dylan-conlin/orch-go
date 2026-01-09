<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dead agents with SYNTHESIS.md or Phase: Complete should show as "awaiting-cleanup" not "dead" - distinguishing completed-but-orphaned from crashed agents.

**Evidence:** Code analysis shows status determination only uses 4 priorities (issue closed, phase complete, synthesis exists, session status). Dead status comes from session activity timeout (3 min), ignoring completion signals.

**Knowledge:** The Priority Cascade model (determineAgentStatus) already checks Phase: Complete and SYNTHESIS.md for "completed" but this happens AFTER dead detection from session activity. Need to reorder or add new status.

**Next:** Implement "awaiting-cleanup" status for dead agents that have completion signals (Phase: Complete or SYNTHESIS.md).

**Promote to Decision:** recommend-no (tactical UI improvement, not architectural)

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

# Investigation: Handle Multiple Agents Same Beads

**Question:** How should the dashboard distinguish between crashed agents (truly dead) and completed-but-not-cleaned-up agents when multiple agents work on the same beads ID?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** Agent
**Phase:** Complete
**Next Step:** None - implement solution
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** 
**Supersedes:** 
**Superseded-By:**

---

## Findings

### Finding 1: Dead status is determined from session activity timeout

**Evidence:** In `serve_agents.go:446-447`, if `timeSinceUpdate > deadThreshold (3 minutes)`, status is set to "dead". This happens BEFORE the Priority Cascade model runs.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve_agents.go:409-447`

**Significance:** The dead detection is based purely on session activity, ignoring completion signals. A completed agent that hasn't been cleaned up will show as "dead" after 3 minutes of inactivity.

---

### Finding 2: Priority Cascade model exists but runs after dead detection

**Evidence:** `determineAgentStatus()` at line 1038 uses Priority Cascade: issueClosed > phaseComplete > SYNTHESIS.md > sessionStatus. But the initial "dead" status from line 447 becomes the `sessionStatus` input.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve_agents.go:1038-1056`

**Significance:** If Phase: Complete is reported or SYNTHESIS.md exists, the status WILL be overridden to "completed". The issue is when neither completion signal is present BUT the agent actually finished - this creates the false positive.

---

### Finding 3: Respawning on same beads ID warns but allows spawn

**Evidence:** In `spawn_cmd.go:873-904`, when spawning on an issue that's `in_progress`, the code checks for active sessions. If the session is stale (30+ min inactive), it logs a note and continues. No auto-cleanup of old session.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd.go:873-904`

**Significance:** Multiple agents can exist on the same beads ID. Old sessions are not automatically closed when respawning. This explains why dashboard shows both old and new agents.

---

### Finding 4: Workspace cache maps beads ID to workspace

**Evidence:** `wsCache.lookupWorkspace(beadsID)` returns workspace path. This is populated from SPAWN_CONTEXT.md files in workspace directories.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve_agents_cache.go:473-494`

**Significance:** The system can identify which workspace belongs to which beads ID. This enables potential auto-cleanup when respawning.

---

## Synthesis

**Key Insights:**

1. **Status determination needs workspace-aware completion detection** - Dead agents that have SYNTHESIS.md or Phase: Complete in their workspace should show differently than truly crashed agents. The Priority Cascade model already has this logic but it only applies to agents we fetch beads data for.

2. **No auto-cleanup on respawn is intentional** - Allowing respawns without cleaning up old sessions provides flexibility (old session might have useful artifacts). The issue is visibility, not the multiple sessions themselves.

3. **Awaiting-cleanup is a distinct state** - An agent that completed work (SYNTHESIS.md exists) but wasn't formally completed via `orch complete` is not "dead" (crashed) - it's waiting for cleanup. This distinction helps orchestrators prioritize attention.

**Answer to Investigation Question:**

The dashboard should introduce a new status "awaiting-cleanup" for agents that:
1. Have "dead" session status (no activity for 3+ minutes), AND
2. Have SYNTHESIS.md in their workspace OR Phase: Complete in beads comments

This distinguishes completed-but-orphaned agents from truly crashed/stuck agents. The fix is in `determineAgentStatus()` - add a new return value when sessionStatus is "dead" but completion signals exist.

For multiple agents on same beads ID: Show the latest as primary, archive older ones. Dashboard already deduplicates by session title (line 520-531 in serve_agents.go). The issue is when both are visible but one shows as "dead" when it should show as "awaiting-cleanup".

---

## Structured Uncertainty

**What's tested:**

- ✅ Status determination logic reviewed in code (serve_agents.go:409-447, 1038-1056)
- ✅ Respawn behavior reviewed in code (spawn_cmd.go:873-904)
- ✅ Workspace cache structure reviewed (serve_agents_cache.go)

**What's untested:**

- ⚠️ Dashboard rendering of "awaiting-cleanup" status (need to test after implementation)
- ⚠️ Impact on dashboard filtering logic (may need adjustment for new status)
- ⚠️ Stalled detection interaction with awaiting-cleanup (timing edge cases)

**What would change this:**

- If Phase: Complete check is too slow/expensive, may need to reconsider approach
- If dashboard users want different visibility (e.g., hide awaiting-cleanup), UX needs adjustment
- If auto-cleanup on respawn is desired, need separate implementation

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Add "awaiting-cleanup" status** - Modify `determineAgentStatus()` to return "awaiting-cleanup" when sessionStatus is "dead" but completion signals (Phase: Complete OR SYNTHESIS.md) exist.

**Why this approach:**
- Minimal code change (modify one function, add one status value)
- Uses existing completion signal detection (no new data fetching)
- Clear semantic distinction for orchestrators

**Trade-offs accepted:**
- Dashboard CSS/rendering needs update for new status (minor)
- Doesn't auto-cleanup old sessions (separate concern)

**Implementation sequence:**
1. Add "awaiting-cleanup" constant and update `determineAgentStatus()` to check for dead+completed
2. Update web/src dashboard to render awaiting-cleanup status (amber color, clock icon)
3. Test with existing stale sessions

### Alternative Approaches Considered

**Option B: Auto-cleanup on respawn**
- **Pros:** Prevents multiple agents on same beads ID
- **Cons:** Complex - need to close OpenCode sessions, archive workspaces, handle edge cases
- **When to use instead:** When cleanup automation is higher priority than visibility

**Option C: Dashboard grouping by beads ID**
- **Pros:** Shows all agents for same issue together
- **Cons:** Major dashboard refactor, unclear if valuable
- **When to use instead:** If users frequently work with multiple agents per issue

**Rationale for recommendation:** Awaiting-cleanup status is the minimal viable change that addresses the core problem (distinguishing completed from crashed). Auto-cleanup and grouping are orthogonal features.

---

### Implementation Details

**What to implement first:**
1. Modify `determineAgentStatus()` in serve_agents.go to return "awaiting-cleanup" 
2. Add API response field documentation
3. Update dashboard CSS for new status

**Things to watch out for:**
- ⚠️ Dashboard filtering logic may need adjustment (include awaiting-cleanup in Needs Attention?)
- ⚠️ orch status CLI may need update to display new status
- ⚠️ Priority cascade ordering matters - awaiting-cleanup should be between completed and dead

**Areas needing further investigation:**
- Auto-cleanup mechanism (separate issue)
- Dashboard UX for managing awaiting-cleanup agents (batch complete?)

**Success criteria:**
- ✅ Dead agents with SYNTHESIS.md show as "awaiting-cleanup" instead of "dead"
- ✅ Dead agents with Phase: Complete show as "awaiting-cleanup"
- ✅ Dashboard renders awaiting-cleanup distinctly from dead and completed
- ✅ Truly crashed agents (no completion signals) still show as "dead"

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
