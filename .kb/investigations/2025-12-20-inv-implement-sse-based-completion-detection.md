**TLDR:** Implemented SSE-based completion detection and notifications for orch-go. Created a Monitor that tracks session status transitions (busy->idle), a CompletionService that integrates notifications, registry updates, and beads phase updates. High confidence (90%) - all tests pass, clean build, matches spawn context requirements.

---

# Investigation: SSE-Based Completion Detection and Notifications

**Question:** How to implement background SSE monitoring to detect session completion, trigger notifications, and auto-update beads status?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Feature implementation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Existing SSE infrastructure is well-designed

**Evidence:** The existing `pkg/opencode/sse.go` already has:

- `SSEClient` for connecting to OpenCode SSE stream
- `ParseSSEEvent` for parsing SSE format
- `ParseSessionStatus` that handles both old and new JSON formats
- `DetectCompletion` for basic completion detection

**Source:** `pkg/opencode/sse.go:1-160`

**Significance:** Could build on existing parsing logic rather than reinventing. The completion detection needed enhancement to track per-session state for busy->idle transitions.

---

### Finding 2: Registry provides agent tracking by beads ID

**Evidence:** The `pkg/registry/registry.go` tracks agents with:

- ID (workspace name)
- BeadsID (foreign key to beads issue)
- WindowID (tmux window)
- Status (active, completed, abandoned, deleted)

**Source:** `pkg/registry/registry.go:36-60`

**Significance:** The registry enables mapping session completions back to beads issues for phase updates.

---

### Finding 3: Notify package provides desktop notifications

**Evidence:** `pkg/notify/notify.go` has:

- `Notifier` with Backend abstraction (mockable)
- `SessionComplete(sessionID, workspace)` method
- Uses beeep library for cross-platform notifications

**Source:** `pkg/notify/notify.go:1-55`

**Significance:** Ready-to-use notification infrastructure, just needed to wire into completion detection.

---

## Synthesis

**Key Insights:**

1. **Layered architecture** - Created Monitor (SSE handling + completion detection) and CompletionService (orchestrates notifications, registry, beads) as separate concerns.

2. **Stateful session tracking** - Monitor tracks `WasBusy` per session to correctly detect completion (busy->idle transition, not just idle status).

3. **Graceful reconnection** - Monitor handles SSE connection drops with automatic reconnection.

**Answer to Investigation Question:**

Implemented via:

1. `pkg/opencode/monitor.go` - Background SSE monitor with completion detection
2. `pkg/opencode/service.go` - CompletionService integrating all completion actions
3. Updated `cmd/orch/main.go` - `runMonitor` now uses CompletionService

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

All tests pass, clean build, implementation matches spawn context requirements. Minor uncertainty around edge cases in production.

**What's certain:**

- ✅ Monitor correctly detects busy->idle transitions (verified by tests)
- ✅ Desktop notifications are triggered on completion
- ✅ Registry is updated when agent completes
- ✅ Beads phase update is attempted (bd comment)

**What's uncertain:**

- ⚠️ Behavior when OpenCode server restarts mid-monitoring
- ⚠️ Handling of very rapid session completions
- ⚠️ Whether bd command is always available in PATH

**What would increase confidence to Very High:**

- End-to-end testing with real OpenCode sessions
- Testing the SSE reconnection logic under network failures
- Verifying beads command works in all environments

---

## Implementation Summary

### Files Created

1. **`pkg/opencode/monitor.go`** - SSE monitor with:
   - Background goroutine for SSE streaming
   - Per-session state tracking (`SessionState` with `WasBusy` flag)
   - Completion handler registration (`OnCompletion`)
   - Automatic reconnection on connection drop

2. **`pkg/opencode/monitor_test.go`** - Tests covering:
   - Basic completion detection
   - New SSE format handling
   - Multiple concurrent sessions
   - Edge cases (direct idle, ignored events)

3. **`pkg/opencode/service.go`** - CompletionService with:
   - Notification sending via pkg/notify
   - Registry updates
   - Beads phase updates via bd command
   - Event logging

### Files Modified

1. **`cmd/orch/main.go`** - Updated `runMonitor` to use CompletionService

### Key Design Decisions

- Monitor handles SSE parsing and completion detection
- CompletionService handles side effects (notifications, registry, beads)
- Session info can be registered explicitly or looked up via API
- Phase updates only add comment if "Phase: Complete" not already present

---

## References

**Files Examined:**

- `pkg/opencode/sse.go` - Existing SSE parsing logic
- `pkg/opencode/types.go` - Type definitions
- `pkg/opencode/client.go` - OpenCode API client
- `pkg/registry/registry.go` - Agent tracking
- `pkg/notify/notify.go` - Desktop notifications
- `pkg/verify/check.go` - Beads interaction patterns
- `cmd/orch/main.go` - CLI entry points

**Commands Run:**

```bash
# Build verification
go build ./...

# Run tests
go test ./pkg/opencode/... -v -run 'TestMonitor'
go test ./...
```

---

## Investigation History

**2025-12-20:** Investigation started

- Initial question: How to implement SSE-based completion detection
- Context: SPAWN_CONTEXT.md task requirements

**2025-12-20:** Implementation complete

- Created Monitor and CompletionService
- All tests passing
- Clean build
- Final confidence: High (90%)
