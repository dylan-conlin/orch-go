## Summary (D.E.K.N.)

**Delta:** Cross-project beads comment fetching now uses PROJECT_DIR from workspace SPAWN_CONTEXT.md to query the correct project's beads database.

**Evidence:** Tests pass (10 new tests); code compiles; serve.go, main.go status command, and verify package all updated.

**Knowledge:** Beads issues are per-project; when displaying agents from different projects, must query each project's .beads/ directory using PROJECT_DIR from workspace.

**Next:** Close - implementation complete and tested.

**Confidence:** High (90%) - Unit tests pass; needs smoke test with actual cross-project agent.

---

# Investigation: Cross Project Agent Visibility Fetch

**Question:** How to fetch beads comments from an agent's project when viewing from a different project's dashboard?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** Feature Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Beads commands run in current working directory

**Evidence:** `GetComments` and `GetCommentsBatch` in `pkg/verify/check.go` call `beads.FindSocketPath("")` which uses the current working directory to find the beads socket.

**Source:** `pkg/verify/check.go:36-55`, `pkg/verify/check.go:697-734`

**Significance:** When `orch serve` runs from orch-go and an agent was spawned for skillc, comments for `skillc-xxxx` won't be found because we're looking in orch-go's `.beads/` directory.

---

### Finding 2: PROJECT_DIR is stored in SPAWN_CONTEXT.md

**Evidence:** The spawn context template includes `PROJECT_DIR: {{.ProjectDir}}` which stores the absolute path of the project where the agent was spawned.

**Source:** `pkg/spawn/context.go:59`

**Significance:** We can extract PROJECT_DIR from workspace SPAWN_CONTEXT.md to determine which project's beads database to query.

---

### Finding 3: Beads client supports custom directory via FindSocketPath

**Evidence:** `beads.FindSocketPath(dir string)` accepts a directory parameter and walks up from that directory to find `.beads/bd.sock`.

**Source:** `pkg/beads/client.go:64-94`

**Significance:** We can pass the agent's PROJECT_DIR to FindSocketPath to connect to the correct beads daemon.

---

## Synthesis

**Key Insights:**

1. **Cross-project visibility requires per-agent directory awareness** - Each agent may be from a different project, so we need to track beadsID -> projectDir mappings.

2. **Batch fetching can be optimized by grouping by project** - Instead of connecting to different beads daemons per-request, we can group beads IDs by project and reuse connections.

3. **Fallback to CLI must also respect project directory** - When RPC fails, `bd comments` must be run with the correct `cmd.Dir`.

**Answer to Investigation Question:**

The solution is to:
1. Add `extractProjectDirFromWorkspace` to parse PROJECT_DIR from SPAWN_CONTEXT.md
2. Add `GetCommentsWithDir` and `GetCommentsBatchWithProjectDirs` that take project directories
3. Update serve.go and main.go status to extract project dirs and use project-aware fetching

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Implementation is complete with passing unit tests. The code changes are straightforward and follow existing patterns.

**What's certain:**

- ✅ PROJECT_DIR is reliably stored in SPAWN_CONTEXT.md (part of spawn template)
- ✅ beads.FindSocketPath accepts and uses the directory parameter correctly
- ✅ All existing tests continue to pass

**What's uncertain:**

- ⚠️ Not yet smoke-tested with actual cross-project agent scenario
- ⚠️ Edge case: what if PROJECT_DIR points to non-existent directory?

**What would increase confidence to Very High (95%+):**

- Manual smoke test with skillc agent viewed from orch-go dashboard
- Edge case testing for missing/invalid PROJECT_DIR values

---

## Implementation (Completed)

**Files Modified:**

1. `cmd/orch/review.go` - Added `extractProjectDirFromWorkspace` function
2. `cmd/orch/review_test.go` - Added tests for `extractProjectDirFromWorkspace`
3. `pkg/verify/check.go` - Added `GetCommentsWithDir`, `FallbackCommentsWithDir`, `GetCommentsBatchWithProjectDirs`
4. `cmd/orch/serve.go` - Updated `handleAgents` to collect project dirs and use `GetCommentsBatchWithProjectDirs`
5. `cmd/orch/main.go` - Updated `runStatus` to use project-aware comment fetching

**Implementation sequence:**
1. Added `extractProjectDirFromWorkspace` to parse PROJECT_DIR from SPAWN_CONTEXT.md
2. Added project-aware functions to verify package
3. Updated callers to collect project directories and use new functions

---

## References

**Files Examined:**
- `pkg/beads/client.go` - Beads client implementation, FindSocketPath behavior
- `pkg/verify/check.go` - GetComments, GetCommentsBatch existing implementation
- `cmd/orch/serve.go` - handleAgents endpoint
- `cmd/orch/main.go` - runStatus command
- `pkg/spawn/context.go` - SPAWN_CONTEXT.md template with PROJECT_DIR

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch/...

# Run tests
make test
go test ./cmd/orch/... -run "TestExtractProjectDir" -v
```

---

## Investigation History

**2025-12-25:** Investigation started
- Initial question: How to fetch beads comments for cross-project agents?
- Context: Dashboard showing "no phase info" for agents from other projects

**2025-12-25:** Implementation completed
- Added extractProjectDirFromWorkspace function
- Added project-aware comment fetching functions
- Updated serve.go and main.go to use new functions
- All tests passing

**2025-12-25:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Cross-project beads comments now fetched from correct project's database
