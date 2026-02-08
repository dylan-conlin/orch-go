**TLDR:** Question: Are strategic alignment commands (focus/drift/next) exposed in CLI? Answer: Yes, all commands were already fully implemented in cmd/orch/focus.go and wired into main.go. Fixed a bug in getReadyIssues parsing that caused incorrect output. Very High confidence (98%) - verified via build, tests, and manual CLI testing.

---

# Investigation: Expose Strategic Alignment Commands

**Question:** Are the strategic alignment commands (focus, drift, next) properly exposed via the CLI?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Dylan
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (98%)

---

## Findings

### Finding 1: Commands Already Fully Implemented

**Evidence:** All three commands exist and work:

- `orch-go focus [goal]` - Set or view north star priority
- `orch-go drift` - Check if active work aligns with focus
- `orch-go next` - Suggest next action based on focus and state

**Source:**

- `cmd/orch/focus.go:22-180` - All command implementations
- `cmd/orch/main.go:69-71` - Commands registered with rootCmd

**Significance:** No new implementation needed - the task was already complete. The spawn context was based on stale information.

---

### Finding 2: JSON Output Support Exists

**Evidence:** All three commands support `--json` flag for programmatic output:

- `orch-go focus --json` outputs focus object
- `orch-go drift --json` outputs drift result with is_drifting, active_issues
- `orch-go next --json` outputs suggestion with action, description

**Source:**

- `cmd/orch/focus.go:52` - focusJSON flag
- `cmd/orch/focus.go:202-205` - driftJSON flag
- `cmd/orch/focus.go:301-304` - nextJSON flag

**Significance:** JSON output requirement was already satisfied.

---

### Finding 3: Bug in getReadyIssues Parsing

**Evidence:** The `next` command displayed incorrect ready issues:

```
Ready issues:
  - 📋
  - 1.
  - 2.
  - 3.
```

The parsing was too simple - it grabbed the first word of every non-empty line, including headers and numbered list markers.

**Source:** `cmd/orch/focus.go:367-397` (original implementation)

**Significance:** Fixed parsing to properly extract issue IDs from numbered list format (`1. [P0] orch-go-o7x: title...`). Now correctly displays `orch-go-o7x`, `orch-go-e0u`, etc.

---

## Synthesis

**Key Insights:**

1. **Task was already complete** - The spawn context indicated commands needed to be implemented, but they were already fully wired up and working.

2. **Minor bug discovered and fixed** - The `getReadyIssues` function had parsing issues that caused garbage output in the `next` command.

3. **Full test coverage exists** - `pkg/focus/focus_test.go` has comprehensive tests for the underlying focus package (335 lines).

**Answer to Investigation Question:**

Yes, all strategic alignment commands were already properly exposed. The only issue was a parsing bug in `getReadyIssues()` which has been fixed.

---

## Confidence Assessment

**Current Confidence:** Very High (98%)

**Why this level?**

All three commands work end-to-end. Build succeeds, tests pass, manual CLI verification shows correct output for both human-readable and JSON formats.

**What's certain:**

- ✅ `focus`, `drift`, and `next` commands are registered and functional
- ✅ JSON output works correctly for all commands
- ✅ pkg/focus has comprehensive test coverage
- ✅ getReadyIssues parsing fix works correctly

**What's uncertain:**

- ⚠️ Haven't tested edge cases like very long issue titles

**What would increase confidence to 100%:**

- Integration tests for CLI commands (currently only pkg tests exist)

---

## Implementation Recommendations

**Purpose:** No implementation needed - this was a verification task.

### What Was Done ⭐

**Fixed getReadyIssues parsing** - Updated parsing to correctly extract issue IDs from `bd ready` output.

**Why this approach:**

- The output format from `bd ready` includes headers, blank lines, and numbered lists
- Previous parsing grabbed first word of every line
- New parsing only extracts from numbered list items

**Trade-offs accepted:**

- Parsing is tightly coupled to `bd ready` output format
- If format changes, parsing will need updating

---

## References

**Files Examined:**

- `cmd/orch/focus.go` - All strategic alignment command implementations
- `cmd/orch/main.go:52-73` - Command registration
- `pkg/focus/focus.go` - Underlying focus store implementation
- `pkg/focus/focus_test.go` - Test coverage

**Commands Run:**

```bash
# Build and test
go build -o /tmp/orch-go-test ./cmd/orch
go test ./...

# Verify commands work
/tmp/orch-go-test focus --help
/tmp/orch-go-test focus --json
/tmp/orch-go-test drift --json
/tmp/orch-go-test next
```

---

## Investigation History

**2025-12-20 16:45:** Investigation started

- Initial question: Are focus/drift/next commands exposed in CLI?
- Context: Spawned to implement commands, found they already exist

**2025-12-20 16:47:** Found commands already implemented

- All three commands registered in main.go
- Full implementation in focus.go with JSON support

**2025-12-20 16:48:** Discovered parsing bug in getReadyIssues

- next command showed garbage output for ready issues
- Fixed parsing to extract issue IDs correctly

**2025-12-20 16:50:** Investigation completed

- Final confidence: Very High (98%)
- Status: Complete
- Key outcome: Commands already implemented, fixed minor parsing bug
