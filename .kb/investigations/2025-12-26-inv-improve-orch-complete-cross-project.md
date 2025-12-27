## Summary (D.E.K.N.)

**Delta:** Added cross-project detection to `orch complete` that provides helpful error messages when running the command from the wrong project directory.

**Evidence:** Test passes: TestCompleteCrossProjectErrorMessage verifies error message contains project hint and cd suggestion.

**Knowledge:** Same pattern as `orch abandon` - extract project prefix from beads ID, compare to current directory, suggest cd command if mismatch.

**Next:** Close - implementation complete with test coverage.

---

# Investigation: Improve Orch Complete Cross Project

**Question:** How to improve `orch complete` error messages when beads ID belongs to a different project?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Feature implementation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Existing pattern in `orch abandon`

**Evidence:** Lines 708-718 of main.go show cross-project detection in `runAbandon`:
- Extracts project prefix from beads ID (e.g., "kb-cli" from "kb-cli-xyz")
- Compares with current project directory name
- Provides helpful error with `--workdir` suggestion

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go:708-718`

**Significance:** This pattern can be directly applied to `orch complete` with minor modification (cd command instead of --workdir flag).

### Finding 2: runComplete lacked cross-project awareness

**Evidence:** Original `runComplete` at line 2787 simply returned generic error on GetIssue failure:
```go
return fmt.Errorf("failed to get beads issue: %w", err)
```

**Source:** Original code at main.go:2787-2792

**Significance:** Users had no indication that the failure was due to project mismatch, leading to confusion.

---

## Implementation

Added cross-project detection logic to `runComplete` that:
1. Gets current project directory at function start (moved from later in function)
2. On `verify.GetIssue` failure, extracts project prefix from beads ID
3. Compares with current project name
4. If mismatch, provides error with:
   - Hint explaining the project mismatch
   - Suggested cd command to the correct project

**Changes:**
- `cmd/orch/main.go`: Added cross-project detection to runComplete (lines 2817-2835)
- `cmd/orch/main_test.go`: Added TestCompleteCrossProjectErrorMessage test

---

## References

**Files Modified:**
- `cmd/orch/main.go` - Added cross-project detection to runComplete
- `cmd/orch/main_test.go` - Added test for cross-project error message

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch/...

# Test cross-project detection
go test ./cmd/orch/... -run TestCompleteCrossProjectErrorMessage -v

# Full test suite
go test ./cmd/orch/... -v
```
