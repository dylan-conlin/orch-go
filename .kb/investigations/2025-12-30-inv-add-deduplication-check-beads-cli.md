## Summary (D.E.K.N.)

**Delta:** Implemented deduplication check in beads Create methods to prevent duplicate issues with same title.

**Evidence:** All tests pass, including new deduplication tests verifying exact title match, Force flag bypass, and closed issue exclusion.

**Knowledge:** Deduplication is client-side, best-effort (fails gracefully), checks open/in_progress issues only, case-sensitive matching.

**Next:** Close - feature is complete and tested.

---

# Investigation: Add Deduplication Check Beads Cli

**Question:** How to prevent concurrent agents from creating duplicate issues via `bd create`?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: CreateArgs already had Force field added

**Evidence:** types.go already contains Force field in CreateArgs and CreateResult struct.

**Source:** pkg/beads/types.go:59-81

**Significance:** Foundation already laid - just needed to implement the actual deduplication logic.

---

### Finding 2: List method supports status filtering

**Evidence:** Both CLIClient and RPC Client have List(args *ListArgs) which accepts Status filter.

**Source:** pkg/beads/cli_client.go:117-136, pkg/beads/client.go:383-400

**Significance:** Can use List to find existing open issues by title for deduplication check.

---

### Finding 3: MockClient needed FindByTitle for testing

**Evidence:** MockClient implements BeadsClient interface and tests use it extensively.

**Source:** pkg/beads/mock_client.go, pkg/beads/dedup_test.go

**Significance:** All three clients (CLI, RPC, Mock) now implement FindByTitle consistently.

---

## Synthesis

**Key Insights:**

1. **Client-side deduplication** - Implemented at Go client level, not daemon level, making it immediately available for all orch-go operations.

2. **Best-effort approach** - If FindByTitle fails, we log warning but continue with creation. This prevents dedup check failures from blocking legitimate issue creation.

3. **Status filtering** - Only open and in_progress issues are considered duplicates. Closed issues with same title don't block new issue creation.

**Answer to Investigation Question:**

Deduplication is implemented in Create() methods of all BeadsClient implementations. Before creating, FindByTitle searches open/in_progress issues for exact title match. If found, returns existing issue. Use Force=true to bypass.

---

## Structured Uncertainty

**What's tested:**

- ✅ FindByTitle finds open issues by exact title (TestMockClient_FindByTitle)
- ✅ FindByTitle finds in_progress issues (TestMockClient_FindByTitle_InProgress)
- ✅ Create returns existing issue on duplicate title (TestMockClient_Create_Deduplication)
- ✅ Force=true bypasses deduplication (TestMockClient_Create_Force)
- ✅ Closed issues don't block new creation (TestMockClient_Create_ClosedIssueNotDuplicate)
- ✅ Title matching is case-sensitive (TestMockClient_Create_CaseSensitiveTitle)

**What's untested:**

- ⚠️ Race condition with concurrent agents (needs integration testing with multiple processes)
- ⚠️ Performance impact on large issue lists (FindByTitle scans all open issues)

**What would change this:**

- If beads adds server-side deduplication, client-side check becomes redundant
- If performance is issue, could add caching or use beads search query

---

## Implementation Recommendations

### Recommended Approach ⭐

**Client-side deduplication in Create()** - Check for existing open issue before creating.

**Why this approach:**
- Immediate availability without daemon changes
- Best-effort - doesn't break creation on check failures
- Consistent across CLI and RPC clients

**Trade-offs accepted:**
- Not atomic - race condition still theoretically possible
- Scans all open issues (acceptable for typical backlog size)

**Implementation sequence:**
1. Add FindByTitle method to BeadsClient interface
2. Implement in CLIClient using List with status filter
3. Implement in RPC Client using List with status filter
4. Modify Create to call FindByTitle unless Force=true
5. Update MockClient for testing

### Alternative Approaches Considered

**Option B: Daemon-side deduplication**
- **Pros:** Atomic, handles races properly
- **Cons:** Requires beads daemon changes, not in scope
- **When to use instead:** If race conditions become actual problem

---

## References

**Files Examined:**
- pkg/beads/cli_client.go - CLIClient implementation
- pkg/beads/client.go - RPC Client implementation  
- pkg/beads/mock_client.go - MockClient for testing
- pkg/beads/interface.go - BeadsClient interface
- pkg/beads/types.go - CreateArgs, CreateResult types

**Commands Run:**
```bash
# Run tests
/usr/local/go/bin/go test ./pkg/beads/... -v
```

---

## Investigation History

**2025-12-30:** Investigation started
- Initial question: Prevent duplicate issues from concurrent agents
- Context: 801 issues with confirmed duplicates

**2025-12-30:** Implementation complete
- Added FindByTitle to BeadsClient interface
- Implemented in CLIClient, Client (RPC), MockClient
- Updated Create methods with deduplication check
- All tests passing including new dedup tests
