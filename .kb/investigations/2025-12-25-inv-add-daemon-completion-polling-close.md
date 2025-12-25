<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added completion polling to the daemon package that finds Phase: Complete agents and automatically closes their beads issues.

**Evidence:** All 72 daemon tests pass including 10 new completion processing tests; go build ./... succeeds.

**Knowledge:** The completion loop uses beads polling (not SSE) because SSE busy->idle detection has false positives; Phase: Complete in beads comments is the only reliable signal.

**Next:** Close issue - implementation complete with tests passing.

**Confidence:** High (90%) - implementation complete, tested, but not yet validated in production.

---

# Investigation: Add Daemon Completion Polling Close

**Question:** How to add daemon completion polling to close agent lifecycle loop?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** og-feat-add-daemon-completion-25dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Existing SSE-based completion service not suitable

**Evidence:** The existing `CompletionService` in `pkg/daemon/completion.go` uses SSE events for completion detection, but SSE busy->idle detection was intentionally disabled due to false positives (agents go idle during tool loading).

**Source:** `pkg/opencode/service.go:100-105`, `pkg/daemon/completion.go`

**Significance:** Cannot reuse SSE-based approach; need polling-based approach using Phase: Complete in beads comments.

---

### Finding 2: verify package provides all needed primitives

**Evidence:** The `verify` package already has:
- `ListOpenIssues()` - gets all open/in_progress issues
- `GetCommentsBatch()` - fetches comments for multiple issues
- `ParsePhaseFromComments()` - extracts Phase status from comments
- `VerifyCompletionFull()` - runs full verification
- `CloseIssue()` - closes beads issue

**Source:** `pkg/verify/check.go:91-103`, `pkg/verify/check.go:598-656`

**Significance:** Implementation can leverage existing primitives without duplicating logic.

---

### Finding 3: Daemon pattern supports injectable functions for testing

**Evidence:** Daemon struct uses function fields like `listIssuesFunc`, `spawnFunc` that can be overridden for testing. Same pattern can be used for completion processing.

**Source:** `pkg/daemon/daemon.go:87-103`, `pkg/daemon/daemon_test.go`

**Significance:** Tests can mock beads behavior without hitting real APIs.

---

## Synthesis

**Key Insights:**

1. **Polling beats SSE for completion** - The Phase: Complete signal in beads comments is the only reliable completion indicator; SSE busy->idle has fundamental reliability issues.

2. **Verification reuse** - By using `verify.VerifyCompletionFull()`, the daemon applies the same checks as manual `orch complete`.

3. **Workspace discovery** - Finding the workspace for an issue requires scanning `.orch/workspace/` directories for SPAWN_CONTEXT.md files that reference the beads ID.

**Answer to Investigation Question:**

Implementation adds three key functions to `pkg/daemon/daemon.go`:
1. `ListCompletedAgents()` - polls beads for issues with Phase: Complete but still open
2. `ProcessCompletion()` - verifies and closes a single agent  
3. `CompletionLoop()` - runs continuous polling at configurable interval (default 60s)

The `events` package was extended with `EventTypeAutoCompleted` for logging daemon-driven completions.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Implementation is complete with passing tests, but hasn't been validated in production with real beads issues.

**What's certain:**

- Implementation compiles and all tests pass
- Logic correctly identifies Phase: Complete agents
- Verification uses same primitives as `orch complete`
- Event logging captures auto-completions

**What's uncertain:**

- Real-world performance with large backlogs (216+ issues)
- Edge cases in workspace discovery (missing SPAWN_CONTEXT.md)
- Rate limiting behavior with many completions

**What would increase confidence to Very High (95%+):**

- Run against real backlog of 216 pending completions
- Observe successful issue closures with correct close reasons
- Verify event logging appears correctly in events.jsonl

---

## Implementation Recommendations

**Purpose:** Document the implementation approach taken.

### Recommended Approach (IMPLEMENTED)

**Beads Polling with Phase: Complete Detection**

**Why this approach:**
- Uses reliable Phase: Complete signal (not flaky SSE)
- Reuses existing verify package primitives
- Follows established daemon pattern with testable functions

**Trade-offs accepted:**
- 60-second polling delay before completion (acceptable for overnight processing)
- More beads API calls than SSE (acceptable with batch fetching)

**Implementation sequence:**
1. Added `CompletionConfig` and `CompletedAgent` types
2. Added `ListCompletedAgents()` with workspace discovery
3. Added `ProcessCompletion()` with full verification
4. Added `CompletionLoop()` for continuous polling
5. Added `EventTypeAutoCompleted` for logging
6. Added 10 tests covering all functionality

---

## References

**Files Modified:**
- `pkg/daemon/daemon.go` - Added completion processing functions (lines 574-808)
- `pkg/daemon/daemon_test.go` - Added 10 new tests for completion processing
- `pkg/events/logger.go` - Added EventTypeAutoCompleted and LogAutoCompleted()

**Tests Run:**
```bash
# All daemon tests
go test ./pkg/daemon/... -v
# Result: PASS (72 tests)

# Full project build
go build ./...
# Result: Success
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-25-inv-investigate-orchestration-lifecycle-end-end.md` - Prior investigation identifying the completion loop gap

---

## Investigation History

**2025-12-25 11:45:** Investigation started
- Initial question: How to add daemon completion polling?
- Context: Prior investigation identified missing completion loop as primary gap

**2025-12-25 12:00:** Implementation complete
- Added CompletionConfig, CompletedAgent, CompletionResult types
- Added ListCompletedAgents(), ProcessCompletion(), CompletionLoop() functions
- Added EventTypeAutoCompleted event type
- Added 10 tests, all passing

**2025-12-25 12:15:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Daemon can now auto-close Phase: Complete agents via polling
