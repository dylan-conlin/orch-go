# Investigation: bd CLI 'show --json' Exit Code for Not-Found IDs

**Date:** 2026-01-29
**Context:** Follow-up from orch-go-21012 investigation
**Status:** Complete
**Upstream Issue:** bd-85487069 (beads repository)

## Problem

Running `bd show <id> --json` for a non-existent ID returns exit code 0 with empty stdout (error on stderr). This makes callers think JSON parse failed rather than the ID not existing.

**Original observation:** `bd show specs-platform-28 --json` returns exit code 0 with empty stdout.

## Goal

Decide whether to:
1. Fix beads CLI to exit non-zero for not-found
2. Emit valid JSON error payload instead of empty stdout
3. Document/standardize empty-stdout behavior

## Findings

### Finding 1: Current behavior confirmed
**Evidence:**
```bash
$ bd show specs-platform-28 --json; echo "Exit code: $?"
Error fetching specs-platform-28: no issue found matching "specs-platform-28"
Exit code: 0
```

**Source:** Direct test execution
**Significance:** Error message goes to stderr, no stdout, exit code 0

### Finding 2: Stdout is truly empty with --json
**Evidence:**
```bash
$ bd show specs-platform-28 --json 2>/dev/null | wc -c
0
```

**Source:** Direct test execution
**Significance:** No bytes written to stdout - complete empty output, not even `null` or `{}`

### Finding 3: Non-JSON mode has identical exit behavior
**Evidence:**
```bash
$ bd show specs-platform-28 2>&1; echo "Exit: $?"
Error fetching specs-platform-28: no issue found matching "specs-platform-28"
Exit: 0
```

**Source:** Direct test execution
**Significance:** Exit code 0 regardless of JSON flag - not specific to JSON mode

### Finding 4: Multi-ID mode continues on errors
**Evidence:**
```bash
$ bd show specs-platform-28 orch-go-21027 --json 2>&1 | head -5
Error fetching specs-platform-28: no issue found matching "specs-platform-28"
[
  {
    ...
```

**Source:** Direct test execution
**Significance:** When given multiple IDs, `bd show` prints errors to stderr but continues processing remaining IDs. Valid IDs still produce JSON output. Exit code remains 0 even with partial failures.

### Finding 5: FatalErrorRespectJSON exists but isn't used
**Evidence:**
- File: `/Users/dylanconlin/Documents/personal/beads/cmd/bd/errors.go:39`
- Function `FatalErrorRespectJSON` outputs structured JSON errors and exits with code 1
- In JSON mode: `{"error": "message"}` to stdout
- In non-JSON mode: "Error: message" to stderr
- Always exits with code 1

**Source:** beads source code
**Significance:** Proper error handling mechanism exists but show.go doesn't use it for not-found errors

### Finding 6: show.go uses continue instead of fatal error
**Evidence:**
- File: `/Users/dylanconlin/Documents/personal/beads/cmd/bd/show.go`
- Line 89, 96: Routed IDs not found → `fmt.Fprintf(os.Stderr, ...)` + `continue`
- Line 353-354, 360: Direct mode not found → `fmt.Fprintf(os.Stderr, ...)` + `continue`
- Line 338-340: If all IDs fail in JSON mode, `allDetails` is empty → returns with exit code 0

**Source:** beads source code analysis
**Significance:** Errors use `continue` instead of `FatalErrorRespectJSON`, enabling multi-ID partial success but breaking exit code contract

## Synthesis

The issue is not specific to `--json` mode - **all** not-found errors in `bd show` exit with code 0. The root cause is a design conflict:

1. **Multi-ID support**: `bd show` accepts multiple IDs and uses `continue` on errors to process remaining IDs
2. **Error signaling**: Unix convention is exit code 1 for errors, but `continue` bypasses this
3. **JSON mode impact**: In JSON mode with all IDs invalid, empty `allDetails` array causes:
   - No stdout output (not even `[]` or `{"error": "..."}`)
   - No error exit code
   - Caller sees empty stdout + exit 0 = looks like JSON parse failure

**Trade-off**: The multi-ID continue-on-error behavior is valuable for batch operations (`bd show id1 id2 id3`) but breaks error signaling for single-ID callers.

**FatalErrorRespectJSON exists** for proper JSON error handling but isn't used for not-found errors.

## Recommendations

### Option 1: Exit non-zero if ANY ID fails (breaking change)
- **Pros**: Fixes exit code contract, simple to understand
- **Cons**: Breaks batch use cases - one bad ID fails entire command
- **Implementation**: Track `hadErrors` boolean, exit 1 at end if true
- **Migration**: Users expecting continue-on-error behavior would break

### Option 2: Emit valid JSON error for all-fail case
- **Pros**: Maintains multi-ID continue-on-error, fixes JSON mode empty stdout
- **Cons**: Exit code still 0 for errors (violates Unix convention)
- **Implementation**: 
  - If JSON mode && `len(allDetails) == 0` && `len(args) > 0`: emit `{"error": "no issues found"}`
  - Still doesn't fix exit code

### Option 3: Hybrid approach (RECOMMENDED)
- **Single ID behavior**: Use `FatalErrorRespectJSON` for single-ID not-found (exit 1)
- **Multi-ID behavior**: Keep continue-on-error but exit 1 if ANY ID failed
- **JSON mode**: Empty `allDetails` + errors → emit `{"errors": [...]}` and exit 1
- **Pros**: Fixes exit code, maintains batch utility, proper JSON errors
- **Cons**: More complex logic
- **Implementation**:
  ```go
  if len(args) == 1 {
      // Single ID - fail fast
      FatalErrorRespectJSON("issue not found: %s", id)
  } else {
      // Multi ID - collect errors, continue
      errors = append(errors, ...)
      hadErrors = true
  }
  // At end:
  if hadErrors {
      if jsonOutput && len(allDetails) == 0 {
          outputJSON(map[string]interface{}{"errors": errors})
      }
      os.Exit(1)
  }
  ```

### Option 4: Document current behavior (no fix)
- **Pros**: No code changes, no breaking changes
- **Cons**: Violates Unix conventions, confusing for callers
- **Not recommended**: Silent failures are error-prone

## Proposed Action

**File a bug issue in beads** with Option 3 (hybrid approach) as recommendation. This is an upstream beads CLI issue, not an orch-go issue. The fix belongs in the beads repository.

**Issue content**:
- **Title**: "bd show exits 0 for not-found IDs (violates Unix error convention)"
- **Problem**: Exit code 0 + empty stdout in JSON mode makes errors indistinguishable from success
- **Root cause**: Multi-ID continue-on-error pattern bypasses exit code signaling
- **Recommendation**: Hybrid fix (Option 3 above)
- **Affected commands**: `bd show` (possibly other multi-ID commands)

## Resolution

Created upstream bug issue **bd-85487069** in beads repository with:
- Complete reproduction steps
- Root cause analysis (multi-ID continue-on-error pattern)
- Recommended fix (Option 3: Hybrid approach)
- Reference to this investigation

**No orch-go code changes needed** - this is an upstream beads CLI issue. Fix should be implemented in the beads repository.
