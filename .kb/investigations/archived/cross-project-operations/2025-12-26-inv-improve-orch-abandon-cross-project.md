## Summary (D.E.K.N.)

**Delta:** Added `--workdir` flag to `orch abandon` and improved error messages to suggest cross-project usage when issue prefix doesn't match current project.

**Evidence:** Build passes, tests pass, `./orch abandon kb-cli-fake123` now shows helpful hint: "Hint: The issue ID suggests it belongs to project 'kb-cli', but you're in 'orch-go'."

**Knowledge:** Cross-project operations require explicit workdir specification because beads socket is project-specific; detecting project mismatch from issue ID prefix enables actionable error messages.

**Next:** Close - implementation complete.

---

# Investigation: Improve Orch Abandon Cross Project

**Question:** How can `orch abandon` provide better error messages when used cross-project?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: runAbandon used os.Getwd() without override option

**Evidence:** `runAbandon` function at cmd/orch/main.go:679 originally had `projectDir, _ := os.Getwd()` with no way to specify a different directory for cross-project operations.

**Source:** cmd/orch/main.go:688 (original), cmd/orch/main.go:679-704 (after fix)

**Significance:** This meant the beads client would look for issues in the current directory's `.beads/` which fails for cross-project issues.

---

### Finding 2: spawn command already has --workdir pattern

**Evidence:** `spawnCmd` at cmd/orch/main.go:279 has `--workdir` flag that resolves to absolute path, validates directory exists, and is used throughout the spawn workflow.

**Source:** cmd/orch/main.go:279, cmd/orch/main.go:1059-1080

**Significance:** Established pattern to follow - the workdir handling in spawn was reused for abandon.

---

### Finding 3: Issue ID prefix can indicate target project

**Evidence:** Beads issue IDs follow pattern `{project}-{hash}` (e.g., `kb-cli-abc123`, `orch-go-xyz789`). The prefix matches the project directory name.

**Source:** Issue ID format observed in beads issues

**Significance:** When the issue ID prefix doesn't match the current directory name, we can provide a specific hint about which project the issue belongs to and suggest the exact command to run.

---

## Structured Uncertainty

**What's tested:**

- ✅ Build succeeds (`go build ./cmd/orch`)
- ✅ TestAbandonNonExistentAgent passes
- ✅ All cmd/orch tests pass (0.589s)
- ✅ Help text shows --workdir flag and cross-project example
- ✅ Cross-project error message displays correctly with hint

**What's untested:**

- ⚠️ Actual cross-project abandonment (would need real issues in different projects)
- ⚠️ beads.DefaultDir setting works correctly with RPC client

---

## References

**Files Examined:**
- cmd/orch/main.go - runAbandon function, abandon command definition
- pkg/beads/client.go - FindSocketPath and DefaultDir mechanism

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch

# Test verification
go test -v -run "Abandon" ./cmd/orch/...
go test ./cmd/orch/...

# Help text verification
./orch abandon --help

# Error message verification
./orch abandon kb-cli-fake123
```
