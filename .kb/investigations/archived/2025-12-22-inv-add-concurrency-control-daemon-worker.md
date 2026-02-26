<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Added WorkerPool to pkg/daemon with semaphore-based slot acquisition, integrated with Daemon struct, and exposed via --concurrency flag.

**Evidence:** Tests pass for WorkerPool (11 tests) and Daemon integration (17 new tests), go build succeeds, --concurrency flag visible in help.

**Knowledge:** WorkerPool pattern is simpler than CapacityManager (no multi-account complexity), slot acquisition provides proper tracking for spawned agents.

**Next:** Close issue - implementation complete.

**Confidence:** High (90%) - Comprehensive tests, follows existing patterns.

---

# Investigation: Add Concurrency Control to Daemon

**Question:** How should we implement worker pool pattern with --concurrency flag for daemon, integrating with CapacityManager concepts?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None - implementation complete
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: CapacityManager is designed for multi-account coordination

**Evidence:** `pkg/capacity/manager.go` provides slot acquisition with per-account tracking, threshold-based selection, and queue timeout handling. It's designed for coordinating across multiple Claude Max accounts.

**Source:** pkg/capacity/manager.go:76-114

**Significance:** The daemon needs simpler concurrency control - just a maximum worker count without the multi-account complexity. A simpler WorkerPool is more appropriate.

---

### Finding 2: Daemon already has basic concurrency infrastructure

**Evidence:** Daemon struct has `MaxAgents` config and `activeCountFunc` that queries OpenCode API for session count. The `AtCapacity()` and `AvailableSlots()` methods use this.

**Source:** pkg/daemon/daemon.go:85-167

**Significance:** The infrastructure exists but uses external API calls. A local WorkerPool provides better tracking and slot management semantics.

---

### Finding 3: Slot acquisition pattern from CapacityManager is reusable

**Evidence:** CapacityManager uses `AcquireSlot(ctx)` and `ReleaseSlot(slot)` pattern with mutex/cond synchronization. This pattern can be simplified for single-pool use.

**Source:** pkg/capacity/manager.go:118-218

**Significance:** The pattern is proven; we can create a simpler version without the account selection logic.

---

## Synthesis

**Key Insights:**

1. **Simpler is better** - WorkerPool provides the same slot acquisition semantics as CapacityManager but without multi-account complexity.

2. **Backward compatible** - Integration preserves existing `activeCountFunc` fallback for cases where pool isn't configured.

3. **Proper tracking** - Slot-based tracking allows attaching BeadsID to each slot for monitoring which issues are active.

**Answer to Investigation Question:**

Created a new `WorkerPool` type in pkg/daemon/pool.go that provides semaphore-based concurrency control. The Daemon struct now optionally uses this pool for capacity tracking. Added `--concurrency` flag (with `-c` shorthand) to daemon run command.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Implementation follows established patterns, comprehensive tests pass, and the design is minimal yet complete.

**What's certain:**

- ✅ WorkerPool provides correct slot acquisition/release
- ✅ Daemon integration is backward compatible
- ✅ --concurrency flag works as expected
- ✅ Tests cover edge cases (at capacity, blocking, release on error)

**What's uncertain:**

- ⚠️ Long-running behavior under heavy load (not tested in CI)
- ⚠️ Interaction with agent completion detection (slots held until explicit release)

---

## Implementation Recommendations

### Recommended Approach ⭐

**Implemented: WorkerPool with Daemon Integration**

**Why this approach:**
- Simpler than full CapacityManager integration
- Proper slot tracking with BeadsID
- Backward compatible with existing activeCountFunc

**Trade-offs accepted:**
- Slots are held until explicitly released (not tied to agent lifecycle)
- No automatic detection of agent completion

**Implementation sequence:**
1. Created pkg/daemon/pool.go with WorkerPool type
2. Created pkg/daemon/pool_test.go with comprehensive tests
3. Updated pkg/daemon/daemon.go to integrate Pool
4. Updated cmd/orch/daemon.go with --concurrency flag

---

## References

**Files Modified:**
- pkg/daemon/pool.go - New WorkerPool implementation
- pkg/daemon/pool_test.go - WorkerPool tests
- pkg/daemon/daemon.go - Daemon integration
- pkg/daemon/daemon_test.go - Integration tests
- cmd/orch/daemon.go - CLI flag and wiring

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch/...

# Test verification  
go test ./pkg/daemon/... -v

# Help verification
go run ./cmd/orch/... daemon run --help
```

---

## Investigation History

**2025-12-22 10:00:** Investigation started
- Initial question: How to add concurrency control with worker pool pattern?
- Context: Daemon needs better concurrency control than simple session counting

**2025-12-22 10:30:** Implementation complete
- Created WorkerPool with tests
- Integrated with Daemon
- Added --concurrency flag

**2025-12-22 10:45:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: WorkerPool provides simpler alternative to full CapacityManager integration
