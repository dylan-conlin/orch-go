<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Stale in_progress status blocks respawn because: (1) orch abandon doesn't reset beads status, (2) spawn's session check doesn't verify liveness.

**Evidence:** spawn check at main.go:1114 uses `strings.Contains(s.Title, beadsID)` without calling `IsSessionActive()` - matches historical sessions.

**Knowledge:** OpenCode persists sessions to disk, so ListSessions returns all sessions including historical ones. Must use IsSessionActive for liveness.

**Next:** Implement two fixes: (1) abandon resets status to open, (2) spawn uses IsSessionActive check.

---

# Investigation: Stale Progress Status Blocks Respawn

**Question:** Why does stale in_progress status block respawn after agent abandonment?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** Implement fix
**Status:** Complete

---

## Findings

### Finding 1: orch abandon doesn't reset beads status

**Evidence:** In runAbandon() at main.go:673-791, the function kills tmux window, logs event, generates failure report - but never calls verify.UpdateIssueStatus to reset status back to "open".

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go:673-791`

**Significance:** After abandonment, beads issue remains in_progress, triggering the spawn's duplicate agent check.

---

### Finding 2: spawn check doesn't verify session liveness

**Evidence:** At main.go:1109-1117, spawn checks `if issue.Status == "in_progress"` then calls `client.ListSessions("")` and loops through matching sessions by title. It returns an error if ANY session contains the beads ID in title - without verifying if the session is actually active.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go:1109-1117`

**Significance:** OpenCode persists sessions to disk. ListSessions returns historical sessions, causing false positive matches for abandoned agents.

---

### Finding 3: IsSessionActive method already exists

**Evidence:** pkg/opencode/client.go:304-313 provides `IsSessionActive(sessionID, maxIdleTime)` which checks if session was updated within maxIdleTime. This is documented as "more reliable than SessionExists() for liveness detection".

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/opencode/client.go:300-313`

**Significance:** The fix is straightforward - use IsSessionActive instead of just checking session existence.

---

## Synthesis

**Key Insights:**

1. **Two-pronged problem** - The bug requires two conditions: stale beads status AND stale session detection. Fixing either would help, but fixing both provides robust solution.

2. **Defense in depth** - Even if spawn correctly detects stale sessions, having abandon reset beads status provides cleaner state and avoids confusing warnings.

3. **Existing infrastructure** - IsSessionActive already exists with proper liveness detection, just not used in spawn's duplicate check.

**Answer to Investigation Question:**

Stale in_progress status blocks respawn because of two bugs working together:
1. `orch abandon` kills the agent but never resets beads status from in_progress to open
2. `orch spawn` checks for duplicate agents by looking for ANY session matching the beads ID, without verifying if that session is actually active (recently updated)

The fix is to:
1. Add `verify.UpdateIssueStatus(beadsID, "open")` to runAbandon
2. Add `IsSessionActive` check before returning the "already in_progress with active agent" error

---

## Implementation Recommendations

### Recommended Approach ⭐

**Two-pronged fix** - Reset beads status in abandon AND use IsSessionActive in spawn

**Why this approach:**
- Defense in depth - each fix handles edge cases the other might miss
- Clean state after abandonment - no confusing warnings
- Proper liveness detection prevents false positives from historical sessions

**Implementation sequence:**
1. Fix runAbandon to call verify.UpdateIssueStatus(beadsID, "open")
2. Fix spawn check to use client.IsSessionActive(s.ID, 30*time.Minute)
3. Add tests for both fixes

### Alternative Approaches Considered

**Option B: Only fix abandon (reset status)**
- **Pros:** Simpler, single change
- **Cons:** Still vulnerable to edge cases where spawn happens before status propagates
- **When to use instead:** If IsSessionActive has performance concerns

**Option C: Only fix spawn (liveness check)**
- **Pros:** Handles the symptom directly
- **Cons:** Leaves beads in inconsistent state, confusing warnings remain
- **When to use instead:** Never - always prefer clean state

---

## References

**Files Examined:**
- cmd/orch/main.go:673-791 - runAbandon function
- cmd/orch/main.go:1103-1127 - spawn duplicate agent check
- pkg/opencode/client.go:235-263 - ListSessions
- pkg/opencode/client.go:300-313 - IsSessionActive

---

## Investigation History

**2025-12-26 16:55:** Investigation started
- Initial question: Why does stale in_progress status block respawn?
- Context: Issue reported that orch spawn refuses with "already in_progress with active agent" after abandonment

**2025-12-26 17:00:** Root cause identified
- Two bugs: abandon doesn't reset status, spawn doesn't check liveness
- Clear fix path identified using existing IsSessionActive method

**2025-12-26 17:05:** Investigation completed
- Status: Complete
- Key outcome: Ready to implement two-pronged fix
