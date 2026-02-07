## Summary (D.E.K.N.)

**Delta:** Created `pkg/daemon/projects.go` with `ListProjects()` function to discover kb-registered projects.

**Evidence:** All 8 unit tests pass: sorting, empty output, kb unavailable, invalid JSON, single project handling.

**Knowledge:** The function follows established patterns (Build*Command for testability, graceful error handling for unavailable tools).

**Next:** Integration ready - daemon can now poll multiple projects via `ListProjects()`.

**Promote to Decision:** recommend-no (tactical implementation, follows existing patterns)

---

# Investigation: Add Project Discovery Function Pkg

**Question:** How should we implement project discovery for the cross-project daemon?

**Started:** 2026-01-21
**Updated:** 2026-01-21
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: kb projects list --json output format

**Evidence:**
```json
[{"name":"project-name","path":"/path/to/project"}, ...]
```

**Source:** `kb projects list --json` command output

**Significance:** Simple JSON array format makes parsing straightforward.

---

### Finding 2: Existing testability patterns in codebase

**Evidence:** The codebase uses `Build*Command` functions to separate command construction from execution (pkg/tmux/tmux.go:264-339). This allows unit testing without side effects.

**Source:** pkg/tmux/tmux.go, prior decision on shell-out testability

**Significance:** Following this pattern ensures consistency and testability.

---

## Structured Uncertainty

**What's tested:**

- ✅ JSON parsing works correctly (verified: TestListProjects_Success)
- ✅ Sorting by name produces alphabetical order (verified: TestListProjects_Success)
- ✅ Empty output returns empty slice (verified: TestListProjects_EmptyOutput)
- ✅ kb unavailable returns empty slice, not error (verified: TestListProjects_KbUnavailable)
- ✅ Invalid JSON returns error (verified: TestListProjects_InvalidJSON)
- ✅ Command construction is correct (verified: TestBuildListProjectsCommand)

**What's untested:**

- ⚠️ Integration with actual kb CLI in production (out of scope for unit tests)

---

## References

**Files Created:**
- `pkg/daemon/projects.go` - Project struct and ListProjects function
- `pkg/daemon/projects_test.go` - Unit tests

**Files Examined:**
- `pkg/daemon/daemon.go` - Existing daemon patterns
- `pkg/beads/cli_client.go` - Shell-out patterns
- `pkg/tmux/tmux.go` - Build*Command patterns

**Commands Run:**
```bash
# Verify kb output format
kb projects list --json

# Run unit tests
go test -v ./pkg/daemon/ -run "TestListProjects|TestBuildListProjects"
# Result: 8 tests pass

# Run all daemon tests
go test ./pkg/daemon/...
# Result: All tests pass (28.8s)
```

---

## Investigation History

**2026-01-21:** Investigation started
- Initial question: Implement project discovery function for cross-project daemon
- Context: Part of cross-project daemon implementation

**2026-01-21:** Investigation completed
- Status: Complete
- Key outcome: Created pkg/daemon/projects.go with tested ListProjects() function
