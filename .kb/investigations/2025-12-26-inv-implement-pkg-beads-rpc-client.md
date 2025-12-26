## Summary (D.E.K.N.)

**Delta:** pkg/beads RPC client now has full feature parity with beads internal/rpc/client.go including reconnect logic and all missing operations.

**Evidence:** All 36 tests pass; implemented WithAutoReconnect option, 13 new RPC operations, and connection error handling with exponential backoff.

**Knowledge:** The beads daemon uses Unix socket at .beads/bd.sock with JSON-over-newline protocol; reconnect logic needs backoff to avoid overwhelming daemon during restart.

**Next:** Close this issue - implementation is complete and tested.

**Confidence:** High (90%) - tested with mock daemon and integration tests against live beads.

---

# Investigation: Implement Pkg Beads RPC Client

**Question:** How to implement a complete RPC client for beads daemon in pkg/beads, patterning after beads internal/rpc/client.go?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Existing pkg/beads had basic RPC operations but lacked reconnect logic

**Evidence:** Original client.go had Connect, Reconnect (basic Close+Connect), Health, Ready, Show, List, Stats, Comments, AddComment, CloseIssue, Create. Missing: Update, Delete, Stale, Count, Status, Ping, Shutdown, AddLabel, RemoveLabel, AddDependency, RemoveDependency, ResolveID.

**Source:** pkg/beads/client.go (original), beads internal/rpc/client.go

**Significance:** The client needed to be extended with both reconnect logic and missing operations to achieve feature parity with the beads daemon's capabilities.

---

### Finding 2: Beads uses newline-delimited JSON over Unix socket

**Evidence:** Request/Response types use json.RawMessage for Args/Data fields. Protocol is: serialize Request to JSON, write with newline, read Response JSON line.

**Source:** beads internal/rpc/protocol.go:42-58, beads internal/rpc/client.go:176-214

**Significance:** The existing executeLocked function already implements this protocol correctly - no changes needed to the wire format.

---

### Finding 3: Reconnect logic needs exponential backoff and connection error detection

**Evidence:** Reference implementation in beads uses health checks after connect, but doesn't have automatic reconnect. Added isConnectionError() to detect transient errors (broken pipe, connection reset, timeout) and retry with exponential backoff capped at 2 seconds.

**Source:** Implemented in pkg/beads/client.go:180-243

**Significance:** Auto-reconnect enables long-running orch-go processes to survive daemon restarts without manual intervention.

---

## Synthesis

**Key Insights:**

1. **Functional Options Pattern** - WithAutoReconnect(maxRetries) follows existing WithTimeout/WithCwd pattern for consistent API.

2. **Lazy Connection with AutoReconnect** - When autoReconnect is enabled, execute() will auto-connect on first call, enabling simpler client usage without explicit Connect().

3. **Operation Coverage** - Added 13 new operations covering issue CRUD (Update, Delete), queries (Stale, Count, Status), lifecycle (Ping, Shutdown), and metadata (AddLabel, RemoveLabel, AddDependency, RemoveDependency, ResolveID).

**Answer to Investigation Question:**

Implemented full RPC client by:
1. Adding autoReconnect/maxRetries fields to Client struct
2. Implementing connectLocked() helper for use during reconnection
3. Adding isConnectionError() detection for transient errors
4. Implementing exponential backoff retry loop in execute()
5. Adding all missing RPC operations with corresponding types

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Implementation follows beads reference closely and all tests pass including integration tests against live daemon.

**What's certain:**

- All 36 tests pass including unit and integration tests
- Reconnect logic works with mock daemon in tests
- New operations have proper type definitions matching beads protocol

**What's uncertain:**

- Real-world behavior under network partitions not tested
- Very high load scenarios not tested
- Edge cases with daemon version mismatches

**What would increase confidence to Very High:**

- Production usage with monitoring
- Chaos testing with daemon restarts during operations

---

## Implementation Recommendations

### Recommended Approach: Merge as-is

**Why this approach:**
- All tests passing
- Follows existing patterns in codebase
- Compatible with beads daemon protocol

**Implementation is complete:**
1. Added WithAutoReconnect option
2. Implemented reconnect logic with exponential backoff
3. Added 13 new RPC operations
4. All tests pass

---

## References

**Files Examined:**
- beads internal/rpc/client.go - Reference implementation for RPC client
- beads internal/rpc/protocol.go - Operation constants and type definitions
- pkg/beads/client.go - Existing orch-go beads client
- pkg/beads/types.go - Existing type definitions

**Commands Run:**
```bash
go test ./pkg/beads/... -v
```

---

## Investigation History

**2025-12-26:** Investigation started
- Initial question: Implement pkg/beads RPC client with reconnect logic
- Context: orch-go needs robust beads client for issue tracking

**2025-12-26:** Implementation completed
- Added reconnect logic with WithAutoReconnect option
- Added 13 missing RPC operations
- All 36 tests passing

**2025-12-26:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Full-featured beads RPC client with reconnect capability
