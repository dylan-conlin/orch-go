## Summary (D.E.K.N.)

**Delta:** `orch abandon` wasn't deleting OpenCode sessions, causing abandoned agents to persist in `orch status` display.

**Evidence:** After abandon, sessions remained in OpenCode API (tested: ses_46a34b162 still returned from `/session` endpoint). Adding `client.DeleteSession()` call fixed the issue.

**Knowledge:** `orch status` shows agents by querying OpenCode sessions API with 30-minute idle filter. Abandon must delete sessions, not just kill tmux windows.

**Next:** Fix deployed. Monitor for edge cases where delete fails.

---

# Investigation: orch abandon doesn't remove agents from status

**Question:** Why do abandoned agents still appear in `orch status` as idle/AT-RISK after running `orch abandon`?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** systematic-debugging agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: orch abandon only kills tmux window, not OpenCode session

**Evidence:** In `runAbandon()` at `cmd/orch/abandon_cmd.go:157-163`, the function kills the tmux window but has no code to handle the OpenCode session:
```go
if windowInfo != nil {
    fmt.Printf("Killing tmux window: %s\n", windowInfo.Target)
    if err := tmux.KillWindow(windowInfo.Target); err != nil {
        ...
    }
}
```

**Source:** `cmd/orch/abandon_cmd.go:157-163`

**Significance:** OpenCode sessions persist independently from tmux windows. Killing the tmux window doesn't affect the session.

---

### Finding 2: orch status uses OpenCode sessions API as source of truth

**Evidence:** In `runStatus()` at `cmd/orch/status_cmd.go:121-144`, the function fetches all sessions and filters by 30-minute idle time:
```go
sessions, err := client.ListSessions("")
const maxIdleTime = 30 * time.Minute
for i := range sessions {
    updatedAt := time.Unix(s.Time.Updated/1000, 0)
    if now.Sub(updatedAt) <= maxIdleTime {
        beadsID := extractBeadsIDFromTitle(s.Title)
        if beadsID != "" {
            beadsToSession[beadsID] = s
        }
    }
}
```

**Source:** `cmd/orch/status_cmd.go:121-144`

**Significance:** Any session updated within 30 minutes appears in status, regardless of whether `orch abandon` was called.

---

### Finding 3: OpenCode client already has DeleteSession method

**Evidence:** `pkg/opencode/client.go:654-673` implements `DeleteSession`:
```go
func (c *Client) DeleteSession(sessionID string) error {
    req, err := http.NewRequest("DELETE", c.ServerURL+"/session/"+sessionID, nil)
    ...
    if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
        return fmt.Errorf("failed to delete session: status %d: %s", resp.StatusCode, string(body))
    }
    return nil
}
```

**Source:** `pkg/opencode/client.go:654-673`

**Significance:** The infrastructure to delete sessions already existed; it just wasn't being used in abandon.

---

## Synthesis

**Key Insights:**

1. **Layered agent state** - Agents have multiple layers of state: beads issue, OpenCode session, and tmux window. Abandon was only cleaning up one layer (tmux + beads status) but missing the session.

2. **Status as reflection of API** - `orch status` is not a registry; it's a real-time view of OpenCode sessions. Any cleanup must happen at the source (OpenCode API) to be reflected in status.

3. **Simple fix** - The OpenCode client already had `DeleteSession`. The fix was a 10-line addition to call it during abandon.

**Answer to Investigation Question:**

Abandoned agents appeared in `orch status` because `orch abandon` only killed the tmux window and reset beads status, but didn't delete the underlying OpenCode session. The session remained in OpenCode's storage and was included when `orch status` queried the sessions API.

---

## Structured Uncertainty

**What's tested:**

- ✅ DeleteSession removes agent from status (verified: abandoned orch-go-8zgi5, confirmed not in status)
- ✅ Multiple abandons work correctly (verified: cleaned up 4 agents sequentially)
- ✅ Build passes with fix (verified: `go build ./cmd/orch` succeeded)
- ✅ All tests pass (verified: `go test ./... -short` - 30 packages, all passed)

**What's untested:**

- ⚠️ Behavior when OpenCode server is unreachable during abandon
- ⚠️ Behavior when session was already deleted (e.g., abandoned twice)

**What would change this:**

- If OpenCode changes DELETE endpoint behavior, fix may need adjustment
- If sessions can be "archived" instead of deleted, we'd need to update the approach

---

## Implementation Recommendations

### Recommended Approach ⭐

**Delete OpenCode session during abandon** - Add `client.DeleteSession(sessionID)` call after finding the session.

**Why this approach:**
- Simple, single-point fix
- Uses existing infrastructure (DeleteSession method)
- Matches user expectation: abandon = gone

**Trade-offs accepted:**
- Session data is lost (can't recover conversation history)
- Acceptable because abandon implies user wants to discard the agent

**Implementation sequence:**
1. Add DeleteSession call after tmux window kill ✅
2. Add user feedback for successful deletion ✅
3. Handle errors gracefully (warn but don't fail) ✅

### Alternative Approaches Considered

**Option B: Track abandoned session IDs in a file**
- **Pros:** Preserves session data for forensics
- **Cons:** Adds complexity, requires maintaining another state file, status must read and filter
- **When to use instead:** If session history is valuable for debugging

**Option C: Rename session title to mark as abandoned**
- **Pros:** Session data preserved, easy to filter
- **Cons:** OpenCode API may not support title updates, still requires status code changes
- **When to use instead:** Never - API limitation makes this impractical

---

### Implementation Details

**What was implemented:**

Added 9 lines to `cmd/orch/abandon_cmd.go` after tmux window kill:
```go
// Delete the OpenCode session if it exists
// This prevents abandoned agents from appearing in `orch status`
if sessionID != "" {
    fmt.Printf("Deleting OpenCode session: %s\n", sessionID[:12])
    if err := client.DeleteSession(sessionID); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: failed to delete OpenCode session: %v\n", err)
    } else {
        fmt.Printf("Deleted OpenCode session\n")
    }
}
```

**Success criteria:**
- ✅ After `orch abandon <id>`, agent no longer appears in `orch status`
- ✅ Build succeeds
- ✅ Tests pass

---

## References

**Files Examined:**
- `cmd/orch/abandon_cmd.go` - abandon command implementation
- `cmd/orch/status_cmd.go` - status command, how agents are displayed
- `pkg/opencode/client.go` - OpenCode API client, DeleteSession method

**Commands Run:**
```bash
# Verify sessions still exist after abandon
orch status --json | jq '.agents[] | {beads_id, session_id}'

# Test the fix
./orch abandon orch-go-8zgi5 --reason "Testing session deletion fix"

# Verify removal
./orch status --json | jq '.agents[] | select(.beads_id == "orch-go-8zgi5")'
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-26-inv-stale-progress-status-blocks-respawn.md` - Prior related work on beads status reset (different bug, same area)

---

## Investigation History

**2026-01-06 09:54:** Investigation started
- Initial question: Why do abandoned agents appear in status?
- Context: After rate limit incident, 4 agents were abandoned but still showing in status

**2026-01-06 10:00:** Root cause identified
- Abandon doesn't delete OpenCode sessions
- Status reads from sessions API
- DeleteSession method exists but unused

**2026-01-06 10:05:** Fix implemented and tested
- Added DeleteSession call to abandon
- Verified fix by abandoning 4 stale agents
- All tests pass

**2026-01-06 10:10:** Investigation completed
- Status: Complete
- Key outcome: Fix deployed, abandoned agents now properly removed from status
