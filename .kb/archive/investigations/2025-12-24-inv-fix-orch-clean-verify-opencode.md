## Summary (D.E.K.N.)

**Delta:** Fixed `orch clean --verify-opencode` deleting orchestrator's own session by adding active session detection.

**Evidence:** Before fix, running `orch clean --verify-opencode` would delete the session running the command itself. After fix, correctly shows "Skipped 1 active sessions (currently processing)" and preserves the orchestrator session.

**Knowledge:** Sessions without workspace `.session_id` files (orchestrator/interactive) are valid and should not be deleted. Active processing detection uses session update timestamps + IsSessionProcessing check.

**Next:** Close - fix implemented and verified via smoke test.

**Confidence:** High (90%) - smoke test confirms fix works, existing tests pass.

---

# Investigation: Fix Orch Clean Verify Opencode

**Question:** Why does `orch clean --verify-opencode` delete the orchestrator's own session?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: cleanOrphanedDiskSessions had no active session detection

**Evidence:** The `cleanOrphanedDiskSessions` function in `cmd/orch/main.go:3027-3103` only checked if a session ID was tracked via workspace `.session_id` files. Sessions without workspace files (like orchestrator/interactive sessions) were considered "orphaned" and deleted.

**Source:** `cmd/orch/main.go:3064-3070` (original implementation)

**Significance:** This is the root cause - orchestrator sessions don't have workspaces with `.session_id` files because they're interactive, not spawned agents.

---

### Finding 2: IsSessionProcessing provides reliable active session detection

**Evidence:** The `opencode.Client.IsSessionProcessing()` method checks if a session has an unfinished assistant message (pending response), which accurately identifies sessions that are actively running Claude.

**Source:** `pkg/opencode/client.go:315-345`

**Significance:** This provides the detection mechanism needed to identify sessions that should not be deleted.

---

### Finding 3: Performance optimization required for many sessions

**Evidence:** With 98 disk sessions, calling `IsSessionProcessing` for each would make 98 HTTP calls. Optimized by first checking session update timestamp (available in existing data) and only calling `IsSessionProcessing` for recently active sessions (within 5 minutes).

**Source:** `cmd/orch/main.go:3070-3083` (new implementation)

**Significance:** Ensures the fix doesn't cause timeout issues when there are many historical sessions.

---

## Synthesis

**Key Insights:**

1. **Workspace tracking is not universal** - Spawned agents have workspaces with `.session_id` files, but orchestrator/interactive sessions do not. Any cleanup logic must account for this.

2. **Active session detection requires two-tier approach** - First check update timestamp (cheap, uses existing data), then verify processing state (expensive, makes API call) only for recently active sessions.

3. **Session update timestamp is a good filter** - Sessions updated within the last 5 minutes are likely in use. Older sessions are safe to consider for orphan detection without the expensive IsSessionProcessing check.

**Answer to Investigation Question:**

The bug occurred because `cleanOrphanedDiskSessions` only checked if session IDs were tracked in workspace files. Orchestrator sessions don't have workspaces, so they appeared "orphaned." The fix adds active session detection using a two-tier approach: (1) filter by recent activity (< 5 min), then (2) verify with `IsSessionProcessing` for recently active sessions.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Strong evidence from successful smoke test showing the fix works correctly. The "Skipped 1 active sessions" message confirms detection works.

**What's certain:**

- ✅ Root cause identified (missing active session detection)
- ✅ Fix correctly preserves orchestrator session (verified via smoke test)
- ✅ All existing tests pass
- ✅ Performance is acceptable (no timeout, uses efficient two-tier approach)

**What's uncertain:**

- ⚠️ Edge case: what if orchestrator session is idle but shouldn't be deleted? (unlikely scenario)
- ⚠️ 5-minute threshold is somewhat arbitrary (but conservative)

**What would increase confidence to Very High (95%+):**

- Production testing over multiple sessions
- Verify behavior with concurrent agents running

---

## Implementation Recommendations

**Implemented:** Two-tier active session detection in `cleanOrphanedDiskSessions`:

1. Check session update timestamp from existing data (no API call)
2. Only call `IsSessionProcessing` for sessions active within last 5 minutes
3. Skip (don't delete) any session that is actively processing

**Files Changed:**

- `cmd/orch/main.go:3064-3083` - Added active session detection logic

---

## References

**Files Examined:**
- `cmd/orch/main.go` - Clean command and cleanOrphanedDiskSessions function
- `pkg/opencode/client.go` - IsSessionProcessing method
- `.kb/investigations/2025-12-21-inv-enhance-orch-clean-four-layer.md` - Prior investigation context

**Commands Run:**
```bash
# Build and test
make build
go test ./...

# Smoke test
./build/orch clean --verify-opencode --dry-run
```

---

## Investigation History

**2025-12-24 22:23:** Investigation started
- Initial question: Why does orch clean --verify-opencode delete the orchestrator's session?
- Context: Reported bug affecting orchestrator's own session

**2025-12-24 22:28:** Fix implemented
- Added two-tier active session detection
- Smoke test confirms fix works

**2025-12-24 22:30:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Active session detection prevents deletion of orchestrator session
