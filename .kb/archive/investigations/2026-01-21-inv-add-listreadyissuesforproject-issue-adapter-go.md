## Summary (D.E.K.N.)

**Delta:** Added `ListReadyIssuesForProject(projectPath)` function to pkg/daemon/issue_adapter.go for cross-project daemon support.

**Evidence:** Unit tests pass - 4 tests covering empty path error, nonexistent path resilience, no-beads-dir handling, and path targeting.

**Knowledge:** Function follows existing pattern: RPC daemon first via `beads.FindSocketPath(projectPath)`, CLI fallback with `cmd.Dir = projectPath`. Returns empty list on error (no crash).

**Next:** Close - implementation complete with tests.

**Promote to Decision:** recommend-no (tactical implementation, follows existing patterns)

---

# Investigation: Add ListReadyIssuesForProject to Issue Adapter

**Question:** How to add project-aware issue listing for cross-project daemon?

**Started:** 2026-01-21
**Updated:** 2026-01-21
**Owner:** orch-go-00zsx
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Existing pattern uses FindSocketPath with empty string

**Evidence:** `ListReadyIssues()` calls `beads.FindSocketPath("")` which uses current directory or DefaultDir.

**Source:** `pkg/daemon/issue_adapter.go:19`, `pkg/beads/client.go:145`

**Significance:** For project targeting, simply pass the project path to `FindSocketPath(projectPath)` instead.

---

### Finding 2: CLI fallback uses cmd.Dir for directory targeting

**Evidence:** `FallbackShowWithDir` in beads package sets `cmd.Dir = dir` to run bd CLI in specific project.

**Source:** `pkg/beads/client.go:784-810`

**Significance:** Same pattern works for `bd ready --json --limit 0` - set `cmd.Dir = projectPath`.

---

### Finding 3: Error handling should return empty list per acceptance criteria

**Evidence:** Acceptance criteria states "Error in one project doesn't crash - returns empty list with logged warning."

**Source:** Task description

**Significance:** Unlike `listReadyIssuesCLI()` which returns errors, the project version logs warnings and returns `[]Issue{}`.

---

## Structured Uncertainty

**What's tested:**

- ✅ Empty projectPath returns error (verified: TestListReadyIssuesForProject_EmptyPath)
- ✅ Nonexistent path returns empty list, no crash (verified: TestListReadyIssuesForProject_NonExistentPath)
- ✅ Path without .beads returns empty list (verified: TestListReadyIssuesForProject_NoBeadsDir)
- ✅ Full package tests pass (verified: go test ./pkg/daemon/... - ok)

**What's untested:**

- ⚠️ RPC path with running beads daemon (requires live daemon, tested via CLI fallback)
- ⚠️ Integration with actual cross-project daemon (depends on parent feature)

---

## References

**Files Modified:**
- `pkg/daemon/issue_adapter.go` - Added ListReadyIssuesForProject and listReadyIssuesForProjectCLI
- `pkg/daemon/issue_adapter_test.go` - Created with 4 test cases

**Commands Run:**
```bash
# Run unit tests
/usr/local/go/bin/go test ./pkg/daemon/... -v -run TestListReadyIssuesForProject

# Run full package tests
/usr/local/go/bin/go test ./pkg/daemon/...
```

---

## Investigation History

**2026-01-21:** Implementation complete
- Added `ListReadyIssuesForProject(projectPath string)` function
- Added `listReadyIssuesForProjectCLI(projectPath string)` helper
- Created unit tests in `issue_adapter_test.go`
- All tests pass (4 new tests + existing package tests)
