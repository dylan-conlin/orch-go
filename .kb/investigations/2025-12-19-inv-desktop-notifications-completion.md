**TLDR:** Create pkg/notify wrapper for desktop notifications on agent completion. Implementation is straightforward - current main.go already uses beeep directly (line 408); extracting to pkg/notify with session context improves testability and enables workspace names in notifications. High confidence (90%) - beeep already working in codebase.

---

# Investigation: Desktop Notifications on Completion

**Question:** How to implement macOS desktop notifications when agents complete with workspace context?

**Started:** 2025-12-19
**Updated:** 2025-12-19
**Owner:** Worker Agent
**Phase:** Implementing
**Next Step:** Create notify package with TDD
**Status:** In Progress
**Confidence:** High (90%)

---

## Findings

### Finding 1: beeep already integrated in codebase

**Evidence:** `cmd/orch/main.go:408` already uses `beeep.Notify()` for completion notifications in the monitor command.

**Source:** cmd/orch/main.go:16 (import), cmd/orch/main.go:406-410 (usage)

**Significance:** No new dependencies needed. Current implementation works but has limitations - no workspace name in notification.

---

### Finding 2: Session context available but not used

**Evidence:** `DetectCompletion()` returns sessionID but monitor doesn't look up session metadata to get workspace name.

**Source:** pkg/opencode/sse.go:88-108, cmd/orch/main.go:404-425

**Significance:** Need to either store workspace names during spawn or lookup via /session API.

---

### Finding 3: Events already logged with workspace context

**Evidence:** Spawn logging includes workspace name in event data.

**Source:** cmd/orch/main.go:224-228

**Significance:** Could correlate session.spawned events with session.completed to get workspace name.

---

## Implementation Recommendations

### Recommended Approach: Simple pkg/notify wrapper

**Approach:** Create `pkg/notify/notify.go` that wraps beeep with structured notification types.

**Why this approach:**
- Testable (can mock for tests)
- Extensible (add notification types later)
- Consistent API for different notification events

**Implementation sequence:**
1. Write failing tests for Notifier interface
2. Implement Notifier with beeep
3. Update monitor command to use pkg/notify
4. Enhance notifications with workspace names

---

## References

**Files Examined:**
- cmd/orch/main.go - Current beeep usage (lines 16, 406-410)
- pkg/opencode/sse.go - SSE event parsing and completion detection
- pkg/opencode/types.go - Session and event types

---

## Investigation History

**2025-12-19:** Investigation started
- Initial question: How to implement desktop notifications for agent completion?
- Context: Part of orch-go-ph1.6 - desktop notifications feature
