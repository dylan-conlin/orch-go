<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode DeleteSession API works and can clean up stale sessions - 461 sessions deleted in test run (627 → 166).

**Evidence:** Tested `orch clean --sessions` - successfully deleted 461 sessions older than 7 days; sessions with active processing correctly skipped.

**Knowledge:** OpenCode session cleanup must skip sessions that are actively processing (IsSessionProcessing check); dry-run mode essential for previewing impact.

**Next:** Feature implemented - close issue. Consider adding automatic cleanup to daemon in future.

---

# Investigation: Implement OpenCode Session Cleanup Mechanism

**Question:** How can we clean up accumulated OpenCode sessions to prevent dashboard slowness and memory growth?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Worker agent og-feat-implement-opencode-session-06jan-7e6e
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: OpenCode DeleteSession API exists and works

**Evidence:** Tested DELETE /session/{id} - returns HTTP 200 with body "true"

**Source:** 
- `pkg/opencode/client.go:653-674` - DeleteSession method already implemented
- `curl -X DELETE http://localhost:4096/session/ses_xxx` - returns 200

**Significance:** No need to implement the API client - just need to add the cleanup command

---

### Finding 2: Session age can be determined from Time.Updated field

**Evidence:** OpenCode sessions have `Time.Updated` (milliseconds since epoch) that tracks last activity

**Source:**
- `pkg/opencode/types.go` - Session struct with Time.Updated field
- Tested filtering: 588 of 624 sessions (94%) were older than 24 hours

**Significance:** Can reliably identify stale sessions based on age

---

### Finding 3: Must skip actively processing sessions

**Evidence:** IsSessionProcessing checks last message finish status to detect active work

**Source:** 
- `pkg/opencode/client.go:376-402` - IsSessionProcessing method
- Test run skipped 5 active sessions (currently processing)

**Significance:** Prevents accidentally deleting sessions that are in use (like the current orchestrator)

---

## Synthesis

**Key Insights:**

1. **Existing infrastructure sufficient** - DeleteSession API and session listing already implemented in opencode client

2. **Age-based cleanup is safe** - 7 day default retention period is conservative; active sessions protected by IsSessionProcessing check

3. **Pattern matches existing cleanup flags** - Added `--sessions` and `--sessions-days` flags following the same pattern as `--stale` and `--stale-days`

**Answer to Investigation Question:**

OpenCode sessions can be cleaned up by adding a `--sessions` flag to `orch clean` that deletes sessions older than a configurable number of days (default 7). The implementation uses the existing DeleteSession API and protects active sessions by checking IsSessionProcessing before deletion.

---

## Structured Uncertainty

**What's tested:**

- ✅ DeleteSession API works (verified: deleted 461 sessions)
- ✅ Session age filtering works (verified: only sessions older than 7 days targeted)
- ✅ Active sessions protected (verified: 5 active sessions skipped)
- ✅ Dry-run mode works (verified: shows "Would delete" without deleting)

**What's untested:**

- ⚠️ Performance with very large session counts (1000+)
- ⚠️ Automatic cleanup via daemon (not implemented)
- ⚠️ Session cleanup across multiple projects (only tested current directory)

**What would change this:**

- If DeleteSession fails for certain session types/states
- If OpenCode changes API behavior

---

## Implementation Recommendations

### Recommended Approach ⭐

**Manual cleanup via `orch clean --sessions`** - Users can run this command to clean up stale sessions when needed.

**Why this approach:**
- Simple, predictable behavior
- User has control over when cleanup happens
- Matches existing cleanup patterns in orch clean

**Trade-offs accepted:**
- Not automatic - users must remember to run periodically
- Could add daemon-based cleanup in future

**Implementation sequence:**
1. ✅ Add `--sessions` and `--sessions-days` flags to clean command
2. ✅ Implement cleanStaleSessions function
3. ✅ Integrate with existing cleanup reporting and logging

### Alternative Approaches Considered

**Option B: Automatic cleanup in daemon**
- **Pros:** No manual intervention needed
- **Cons:** More complex, needs careful throttling to avoid impacting running agents
- **When to use instead:** If manual cleanup proves insufficient

---

### Implementation Details

**What was implemented:**
- Added `--sessions` flag to `orch clean` command
- Added `--sessions-days` flag (default: 7) for configurable retention
- Implemented cleanStaleSessions function that:
  - Lists all OpenCode sessions
  - Filters to sessions older than cutoff
  - Skips sessions that are actively processing
  - Deletes remaining stale sessions
- Integrated with existing cleanup reporting and event logging

**Test results:**
- 627 total sessions → 166 after cleanup (461 deleted)
- 5 active sessions correctly skipped
- Dry-run mode works as expected

---

## References

**Files Examined:**
- `cmd/orch/clean_cmd.go` - Clean command implementation
- `pkg/opencode/client.go` - OpenCode client with DeleteSession
- `pkg/daemon/daemon.go` - Daemon structure (for future automatic cleanup)

**Commands Run:**
```bash
# Test session count before cleanup
curl -s http://localhost:4096/session | jq 'length'  # 627

# Test dry-run
go run ./cmd/orch clean --sessions --dry-run

# Test actual cleanup
go run ./cmd/orch clean --sessions

# Verify session count after
curl -s http://localhost:4096/session | jq 'length'  # 166
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-06-inv-workspace-session-architecture.md`
- **Beads Issue:** orch-go-vbpci (Dashboard API slow - P1)
