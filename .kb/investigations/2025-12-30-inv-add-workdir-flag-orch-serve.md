<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added --workdir flag to orch serve and ?project= query param to beads endpoints for multi-project dashboard support.

**Evidence:** Tests pass (pool_test.go); go build succeeds; all changes in commit 5b1a4676.

**Knowledge:** BeadsClientPool enables lazy per-project daemon connections; serveEffectiveDir overrides compile-time sourceDir.

**Next:** Close - implementation complete.

---

# Investigation: Add Workdir Flag Orch Serve

**Question:** How to enable multi-project beads visibility in orch serve dashboard?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Agent og-feat-add-workdir-flag-30dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: sourceDir is compile-time constant

**Evidence:** `sourceDir = "unknown"` in main.go:44, set via ldflags at build time. runServe() uses sourceDir for beads.DefaultDir.

**Source:** cmd/orch/main.go:44, cmd/orch/serve.go:161-162

**Significance:** Need runtime override mechanism to support different project directories.

---

### Finding 2: beadsClient is global singleton

**Evidence:** `beadsClient *beads.Client` initialized once at startup in runServe(). All handlers use this single client.

**Source:** cmd/orch/serve.go:42, line 167-174

**Significance:** Multi-project support requires a client pool, not a single client.

---

### Finding 3: beads.DefaultDir controls CLI fallback behavior

**Evidence:** beads.FallbackReady(), FallbackBlocked(), FallbackStats() all use DefaultDir for working directory.

**Source:** pkg/beads/client.go:647-660, 819-841

**Significance:** Handlers must set DefaultDir temporarily when querying different projects via CLI fallback.

---

## Synthesis

**Key Insights:**

1. **Runtime Override Pattern** - Added serveEffectiveDir that prefers --workdir flag over compile-time sourceDir.

2. **Lazy Pool Architecture** - BeadsClientPool creates connections on-demand per project directory, avoiding connection overhead for projects that aren't queried.

3. **CLI Fallback Support** - Temporarily swapping beads.DefaultDir ensures CLI fallbacks work for cross-project queries.

**Answer to Investigation Question:**

Multi-project beads visibility was enabled by:
1. Adding `--workdir` flag to override default project directory
2. Adding `?project=<path>` query param to /api/beads, /api/beads/ready, /api/beads/blocked
3. Creating BeadsClientPool for lazy per-project daemon connections
4. Including `project_dir` in API responses for visibility

---

## Structured Uncertainty

**What's tested:**

- ✅ BeadsClientPool correctly manages per-directory clients (pool_test.go passes)
- ✅ Code compiles and runs (go build ./cmd/orch/... succeeds)
- ✅ Existing beads tests pass (go test ./pkg/beads/... passes)

**What's untested:**

- ⚠️ End-to-end multi-project query with live beads daemon (requires running daemons in multiple projects)
- ⚠️ Dashboard UI integration with ?project= param (frontend work)

**What would change this:**

- Finding would need revision if beads daemon socket discovery fails for relative paths
- Finding would need revision if CLI fallback race conditions occur with DefaultDir swapping

---

## Implementation Recommendations

**Purpose:** N/A - Implementation already complete.

### Implemented Approach

**BeadsClientPool with query param** - Lazy-initialized client pool + ?project= query parameter

**Implementation sequence:**
1. Created pkg/beads/pool.go with Pool struct and GetOrCreate method
2. Added pool_test.go with comprehensive tests
3. Added --workdir flag to serve command
4. Added getBeadsClientForProject helper
5. Updated handleBeads, handleBeadsReady, handleBeadsBlocked to use project param

---

## References

**Files Modified:**
- cmd/orch/serve.go - Added --workdir flag, pool initialization, project param support
- pkg/beads/pool.go - New file: BeadsClientPool implementation
- pkg/beads/pool_test.go - New file: Pool tests

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch/...

# Test execution
go test ./pkg/beads/... -v

# Full test suite
go test ./...
```

---

## Investigation History

**2025-12-30 12:00:** Investigation started
- Initial question: Add --workdir flag to orch serve for multi-project dashboard
- Context: Spawned from beads issue orch-go-hzbq

**2025-12-30 12:30:** Implementation complete
- Created BeadsClientPool in pkg/beads/pool.go
- Added --workdir flag and project query param support
- All tests passing

**2025-12-30 12:45:** Investigation completed
- Status: Complete
- Key outcome: Multi-project beads support via --workdir flag and ?project= query param
