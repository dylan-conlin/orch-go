<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Enhanced pre-spawn stale bug check to BLOCK spawns by default when git commits suggest bug may be fixed.

**Evidence:** Tests pass for FormatStaleBugGateError, existing CheckStaleBug functionality preserved, build succeeds.

**Knowledge:** The Gate Over Remind pattern (block by default, explicit bypass) is more effective than warnings for preventing wasted agent time.

**Next:** Close - implementation complete with tests.

---

# Investigation: Add Pre-Spawn Check to Verify Bug Still Exists

**Question:** How to implement a pre-spawn check that verifies bug/issue still exists before spawning?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Agent og-feat-add-pre-spawn-30dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Existing stale bug check only warns

**Evidence:** In `cmd/orch/main.go` lines 1357-1366, the existing stale bug check calls `CheckStaleBugForIssue` and `FormatStaleBugWarning` but only prints to stderr - doesn't block the spawn.

**Source:** `cmd/orch/main.go:1357-1366`

**Significance:** The infrastructure for detecting potentially fixed bugs already exists; we just need to change from warning to blocking behavior.

---

### Finding 2: Gate Over Remind pattern already established

**Evidence:** The failure report gate (lines 1372-1383) uses `return errors.New(...)` to block spawns when conditions aren't met. The `--skip-failure-review` flag provides explicit bypass.

**Source:** `cmd/orch/main.go:1372-1390`

**Significance:** Follow the same pattern for consistency - block by default, require explicit `--skip-stale-check` to bypass.

---

### Finding 3: StaleBugResult already captures needed data

**Evidence:** `pkg/verify/stale_bug.go` defines `StaleBugResult` with `RelatedCommits`, `IssueID`, `IssueTitle`, and `IsPotentiallyStale()` method.

**Source:** `pkg/verify/stale_bug.go:14-34`

**Significance:** The data structures are ready - just need new `FormatStaleBugGateError` function for blocking error message.

---

## Synthesis

**Key Insights:**

1. **Warning patterns are easily ignored** - The previous warn-only behavior didn't prevent wasted agent time on already-fixed bugs.

2. **Gate Over Remind is effective** - Blocking by default with explicit bypass creates pressure to verify before proceeding.

3. **Bypass is documented** - When `--skip-stale-check` is used with a potentially stale bug, it logs what was bypassed for auditability.

**Answer to Investigation Question:**

Implemented by:
1. Adding `FormatStaleBugGateError()` in `pkg/verify/stale_bug.go` - formats blocking error with evidence, resolution steps, and bypass instructions
2. Changing spawn behavior to `return errors.New(...)` instead of just printing warning
3. Enhancing `--skip-stale-check` bypass to show what commits were detected

---

## Structured Uncertainty

**What's tested:**

- ✅ `FormatStaleBugGateError` returns empty for nil/non-stale results (unit test)
- ✅ `FormatStaleBugGateError` shows beads ID, issue title, commits, bypass instructions (unit test)
- ✅ Truncation works correctly for >5 commits (unit test)
- ✅ Build compiles successfully with changes
- ✅ All verify package tests pass

**What's untested:**

- ⚠️ End-to-end spawn blocking behavior (would require spawning agent - violates worker constraint)
- ⚠️ Beads RPC communication works correctly (integration test)

**What would change this:**

- If git log parsing fails silently, stale bugs wouldn't be detected
- If beads RPC is unreliable, issue type check may fail

---

## Implementation Details

**Files Modified:**
1. `cmd/orch/main.go` - Changed stale bug check from warning to blocking
2. `pkg/verify/stale_bug.go` - Added `FormatStaleBugGateError()` function
3. `pkg/verify/stale_bug_test.go` - Added tests for new function

**Changes Made:**

1. **Blocking behavior**: Changed lines 1357-1370 to return error instead of printing warning
2. **Enhanced bypass logging**: When `--skip-stale-check` is used, shows count of potentially related commits
3. **New error function**: `FormatStaleBugGateError()` provides actionable error message with:
   - What was found (related commits)
   - Issue context (ID, title)
   - Resolution options (verify bug, close issue, or bypass)
   - Example command for bypass

---

## References

**Files Examined:**
- `cmd/orch/main.go` - Spawn command implementation
- `pkg/verify/stale_bug.go` - Stale bug detection logic
- `pkg/verify/stale_bug_test.go` - Existing tests

**Commands Run:**
```bash
# Ran tests
go test ./pkg/verify/... -run "StaleBug" -v
# All 15 stale bug tests pass

# Verified build
go build ./cmd/orch/...
# Build succeeds
```

---

## Investigation History

**2025-12-30 15:45:** Investigation started
- Initial question: How to add pre-spawn check that blocks when bug may be fixed?
- Context: Spawned from task to prevent wasted agent time on already-fixed bugs

**2025-12-30 15:50:** Found existing infrastructure
- Stale bug check exists but only warns
- Gate Over Remind pattern established in failure report gate

**2025-12-30 16:00:** Implementation complete
- Added FormatStaleBugGateError function
- Changed spawn behavior to block by default
- Tests pass, build succeeds
