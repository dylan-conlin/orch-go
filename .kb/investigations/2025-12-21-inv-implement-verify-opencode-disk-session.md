## Summary (D.E.K.N.)

**Delta:** Implemented `--verify-opencode` flag for orch clean that queries disk sessions via OpenCode API, identifies orphans, and deletes them.

**Evidence:** All 15 unit tests passing; build successful; help output shows new flag.

**Knowledge:** OpenCode API supports directory-scoped session listing via header and DELETE /session/{id} for cleanup.

**Next:** Close issue - implementation complete and tested.

**Confidence:** High (90%) - unit tested but not live-tested against real OpenCode server yet.

---

# Investigation: Implement --verify-opencode disk session cleanup

**Question:** How to implement disk session cleanup that compares OpenCode disk sessions against the registry and deletes orphans?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: OpenCode API supports directory-scoped session listing

**Evidence:** Existing `ListSessions()` already uses `x-opencode-directory` header. Added `ListDiskSessions()` with the same pattern but requiring the directory parameter.

**Source:** `pkg/opencode/client.go:179-207` (existing ListSessions)

**Significance:** This enables querying all disk sessions for the current project directory, which is needed to compare against registry.

---

### Finding 2: OpenCode API supports session deletion

**Evidence:** Added `DeleteSession()` method that sends `DELETE /session/{id}`. API accepts both 200 OK and 204 No Content as success.

**Source:** `pkg/opencode/client.go:428-446` (new DeleteSession method)

**Significance:** This enables deleting orphaned sessions that are not tracked in the registry.

---

### Finding 3: Registry correctly excludes deleted agents

**Evidence:** `ListAgents()` excludes agents with `StateDeleted` status, so their session IDs won't be included in the "tracked" set.

**Source:** `pkg/registry/registry.go:391-402` (ListAgents implementation)

**Significance:** This ensures that sessions from deleted agents are correctly identified as orphans and cleaned up.

---

## Synthesis

**Key Insights:**

1. **API is straightforward** - OpenCode's HTTP API supports all needed operations (list by directory, delete by ID) with standard REST semantics.

2. **Registry is source of truth** - By comparing disk sessions against registry session IDs, we can accurately identify orphans.

3. **Dry-run support built-in** - The existing `--dry-run` flag pattern works seamlessly with the new disk session cleanup.

**Answer to Investigation Question:**

Implementation complete. The `--verify-opencode` flag adds a fourth layer of cleanup that:
1. Queries all disk sessions for current directory via `ListDiskSessions()`
2. Builds tracked session ID set from registry's non-deleted agents
3. Identifies orphans (disk sessions not in registry)
4. Deletes orphans via `DeleteSession()` API (or shows count in dry-run)

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Unit tests cover all API interactions and orphan detection logic. Build succeeds and help output is correct. Haven't tested against live OpenCode server yet.

**What's certain:**

- ✅ API methods work correctly (8 unit tests)
- ✅ Orphan detection logic is correct (3 unit tests)
- ✅ Integration with existing clean command works

**What's uncertain:**

- ⚠️ Live behavior with real OpenCode server
- ⚠️ Performance with large numbers of disk sessions

**What would increase confidence to Very High (95%+):**

- Run `orch clean --verify-opencode --dry-run` against real server
- Verify DELETE actually removes sessions

---

## Implementation Recommendations

**Recommended Approach ⭐** - Implemented as designed

**Implementation complete. Key files changed:**
- `pkg/opencode/client.go` - Added ListDiskSessions and DeleteSession methods
- `cmd/orch/main.go` - Added --verify-opencode flag and cleanOrphanedDiskSessions function
- Tests added for all new functionality

---

## References

**Files Examined:**
- `pkg/opencode/client.go` - Existing API patterns
- `pkg/registry/registry.go` - Agent state management
- `cmd/orch/main.go` - Clean command structure

**Commands Run:**
```bash
# Run tests
go test ./cmd/orch/... ./pkg/opencode/... -v -run "TestClean|TestListDiskSessions|TestDeleteSession"
# PASS: all tests passing

# Build
go build -o build/orch ./cmd/orch
# Success

# Verify help
./build/orch clean --help
# Shows --verify-opencode flag
```

---

## Investigation History

**2025-12-21:** Investigation started
- Initial question: How to clean up orphaned OpenCode disk sessions?
- Context: 238 disk sessions but only 2-4 tracked in registry

**2025-12-21:** Implementation complete
- Final confidence: High (90%)
- Status: Complete
- Key outcome: `--verify-opencode` flag implemented with full test coverage
