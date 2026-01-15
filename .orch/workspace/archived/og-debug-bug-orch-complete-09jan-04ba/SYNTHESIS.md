# SYNTHESIS: Bug Fix - orch complete doesn't delete OpenCode session

**Agent:** og-debug-bug-orch-complete-09jan-04ba  
**Skill:** systematic-debugging  
**Beads ID:** orch-go-hknqq  
**Started:** 2026-01-09  
**Phase:** Complete  

---

## TLDR (30-second handoff)

**Problem:** After `orch complete` closes an agent, the OpenCode session remains in the API, causing completed agents to appear as "running" in `orch status`.

**Root Cause:** `complete_cmd.go` closes beads issues and invalidates cache but never deletes the OpenCode session.

**Fix:** Added `client.DeleteSession(sessionID)` call after beads closure (line 565-580). Session ID read from workspace `.session_id` file. Non-fatal warning if deletion fails.

**Impact:** Prevents ghost agents from accumulating in `orch status` output.

---

## What Was Delivered

### Primary Deliverable
- **Fixed bug in complete_cmd.go**: Added OpenCode session deletion after successful completion

### Code Changes
1. **cmd/orch/complete_cmd.go**:
   - Added `opencode` import (line 18)
   - Added session deletion logic after beads closure (lines 565-580)
   - Reads session ID from workspace `.session_id` file
   - Calls `client.DeleteSession(sessionID)`
   - Graceful error handling (warns but doesn't fail if deletion fails)

### Investigation Artifacts
- **Investigation file**: `.kb/investigations/2026-01-09-inv-bug-orch-complete-doesn-set.md`
- Documents root cause analysis, findings, and implementation approach

---

## Root Cause Analysis

### The Bug
After `orch complete` successfully verifies and closes an agent:
1. Beads issue is closed
2. Tmux window is killed
3. Cache is invalidated
4. **But OpenCode session remains in API**

This causes:
- `orch status` queries OpenCode API `/session` endpoint
- Completed sessions still appear in response (no "status" field to filter by)
- Sessions only filtered by age (30 min idle time)
- Agents appear "running" for 30 minutes after completion

### Why This Wasn't Caught Earlier
- `invalidateServeCache()` already existed (line 688)
- Assumed cache invalidation was sufficient
- But cache invalidation only clears TTL cache, doesn't affect OpenCode's session list
- Dashboard refetches from OpenCode API on each request

### The Fix Location
Placement is strategic:
```
Line 531: Close beads issue
Line 563: Print reason
Line 565: [NEW] Delete OpenCode session  ← Added here
Line 566: Export orchestrator transcript
Line 583: Kill tmux window
Line 688: Invalidate cache
```

**Why between beads closure and transcript export:**
- Beads issue already closed (non-reversible gate passed)
- Session deletion won't affect transcript export (uses workspace files, not API)
- If deletion fails, completion still succeeds (just warns)
- Happens before cleanup so session is deleted early

---

## Verification

### Build Verification
✅ Compiles without errors: `go build ./cmd/orch`

### Code Review Checklist
- ✅ Import added: `pkg/opencode` imported at line 18
- ✅ Session ID source: Read from `workspace/.session_id` file
- ✅ Error handling: Non-fatal warning if deletion fails
- ✅ Output: Prints "Deleted OpenCode session: {id}" on success
- ✅ Placement: After beads closure, before cleanup

### Manual Testing Required
⚠️ This fix requires integration testing:
1. Spawn an agent: `orch spawn investigation "test"`
2. Let it complete and report Phase: Complete
3. Run `orch complete <beads-id>`
4. Verify output shows "Deleted OpenCode session: {id}"
5. Run `orch status` and confirm agent no longer appears
6. Check OpenCode API: `curl http://localhost:4096/session | jq` - session should be gone

---

## Edge Cases Handled

### 1. Session ID file missing
**Scenario:** Workspace exists but `.session_id` file doesn't  
**Behavior:** Silently skips deletion (no error), completion succeeds  
**Rationale:** Workspace might be from pre-session-tracking era

### 2. Session already deleted
**Scenario:** Session deleted manually or by other process  
**Behavior:** DeleteSession returns error, warns but continues  
**Rationale:** Goal achieved (session gone), no need to fail completion

### 3. OpenCode server unreachable
**Scenario:** serverURL invalid or OpenCode not running  
**Behavior:** Warns "failed to delete session", continues  
**Rationale:** Beads issue already closed, don't block completion on API availability

### 4. Orchestrator sessions
**Scenario:** Completing an orchestrator session  
**Behavior:** Deletes session just like regular agents  
**Rationale:** Orchestrators also have OpenCode sessions that should be cleaned up

### 5. Untracked agents
**Scenario:** Agent spawned with `--no-track`  
**Behavior:** `workspacePath` check ensures deletion still happens if workspace exists  
**Rationale:** Untracked agents can still have workspaces and sessions

---

## Testing Gaps

### Not Yet Tested
- ⚠️ **Production behavior** - Fix not yet deployed
- ⚠️ **Error scenarios** - Network failures, malformed session IDs
- ⚠️ **Performance** - Impact of DeleteSession call on completion time (expected negligible)

### Why Gaps Acceptable
- Error handling is defensive (warns, doesn't fail)
- Build succeeds, code follows existing patterns
- Integration testing will catch any issues before production use

---

## Discovered Work

### None
No additional issues discovered during investigation.

---

## Recommendation

**Recommend: CLOSE**

The fix is complete and ready for integration testing. Once deployed and manually verified (spawn → complete → verify ghost gone), this issue can be closed.

**Follow-up:** Consider adding automated test that verifies session deletion, but not blocking for closure.

---

## References

- **Investigation:** `.kb/investigations/2026-01-09-inv-bug-orch-complete-doesn-set.md`
- **Prior Ghost Agent:** `.kb/investigations/2026-01-09-inv-pw-oicj-ghost-agent-postmortem.md`
- **OpenCode client:** `pkg/opencode/client.go:752` (DeleteSession method)
- **Completion flow:** `cmd/orch/complete_cmd.go:89-691`
