<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Fixed kb-cli reflect dedup functions to fail-safe (return true on error) instead of fail-open (return false on error).

**Evidence:** Changed `synthesisIssueExists` and `openIssueExists` in reflect.go to return `true` on bd command or JSON parse errors. Updated tests to expect fail-safe behavior. Build and tests pass.

**Knowledge:** For idempotency/dedup checks, "fail-closed" (assume exists) is correct - the cost of a false positive (skipping creation) is low, while the cost of a false negative (creating duplicate) is high.

**Next:** Install updated kb binary and restart daemon to apply fix.

**Promote to Decision:** Actioned - bug fixed in kb reflect

---

# Investigation: Kb Cli Fix Reflect Dedup

**Question:** Fix the dedup check in kb reflect to return true (assume exists) on error instead of false (assume doesn't exist).

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** Architect Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Two dedup functions needed fixing

**Evidence:** Both `synthesisIssueExists` (lines 498-512) and `openIssueExists` (lines 1278-1291) had the same bug pattern - returning `false` when bd command fails or JSON parsing fails.

**Source:** `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/reflect.go:498-512, 1278-1291`

**Significance:** Both functions are used during `kb reflect --create-issue` to prevent duplicate beads issues. When they fail-open, duplicates are created.

---

### Finding 2: Error handling strategy was "fail-open"

**Evidence:** Original code:
```go
if err != nil {
    // If bd fails (e.g., not installed or no .beads), assume no duplicate
    return false, nil
}
```

Changed to:
```go
if err != nil {
    // Fail-safe: assume duplicate exists on error to prevent duplicate creation.
    fmt.Fprintf(os.Stderr, "Warning: ... - assuming issue exists\n", err)
    return true, nil
}
```

**Source:** reflect.go lines 499-504 and 509-512 (both locations)

**Significance:** The original comment even acknowledged the behavior ("assume no duplicate") but made the wrong choice for idempotency checks.

---

### Finding 3: Tests existed but expected wrong behavior

**Evidence:** `TestSynthesisIssueExists` and `TestOpenIssueExists` both expected `exists=false` when bd is not available. Updated to expect `exists=true` with clear documentation of why.

**Source:** `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/reflect_test.go:1498-1517, 1819-1838`

**Significance:** Tests now document the correct fail-safe behavior and will catch any regression.

---

## Synthesis

**Key Insights:**

1. **Fail-safe pattern for idempotency** - When checking "does X exist?" before creating X, errors should assume "yes" to prevent duplicates. The asymmetric cost (low for false positive, high for false negative) makes this the correct default.

2. **Logging enables diagnosis** - Added stderr warnings when dedup fails so administrators can diagnose issues without the system creating duplicates.

3. **Test documentation** - Updated tests now serve as documentation of the expected behavior and reasoning.

**Answer to Investigation Question:**

Fixed both `synthesisIssueExists` and `openIssueExists` functions to return `true` on any error (bd command failure or JSON parse error). Added warning logs to stderr for diagnosis. Updated corresponding tests to expect and document the new fail-safe behavior.

---

## Structured Uncertainty

**What's tested:**

- ✅ Build succeeds (verified: `go build -o build/kb ./cmd/kb`)
- ✅ Tests pass (verified: `go test -v -run "TestSynthesisIssueExists|TestOpenIssueExists"`)
- ✅ Warning logs appear on error (verified: test output shows warning messages)

**What's untested:**

- ⚠️ Production behavior with daemon (needs daemon restart to apply)
- ⚠️ Performance impact (should be negligible - just changes return value)

**What would change this:**

- Finding would be wrong if there's a legitimate need to create issues when bd fails (there isn't - bd failure should block creation)

---

## Implementation Recommendations

**Purpose:** Fix was implemented during this investigation.

### Recommended Approach ⭐

**Fail-Closed Dedup with Logging** - Already implemented. Return `true` (assume duplicate exists) on any error, with warning logging.

**Why this approach:**
- Prevents duplicates even when errors occur
- Logging enables diagnosis without blocking creation
- Aligns with "Gate Over Remind" principle - block is safer than allow on uncertainty

**Trade-offs accepted:**
- May occasionally skip creation when issue doesn't exist (false positive)
- User can manually create if needed - low cost

**Implementation sequence:**
1. ✅ Changed error handling to return `true` on any error
2. ✅ Added `fmt.Fprintf(os.Stderr, ...)` warnings for diagnosis
3. ✅ Updated tests to expect new behavior

### Post-Implementation Steps

**To deploy:**
```bash
# Install updated binary
cd ~/Documents/personal/kb-cli && make install

# Restart daemon to pick up new binary
launchctl kickstart -k gui/$(id -u)/com.orch.daemon
```

---

## References

**Files Modified:**
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/reflect.go` - Changed error handling in synthesisIssueExists and openIssueExists
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/reflect_test.go` - Updated tests for new expected behavior

**Commands Run:**
```bash
# Build verification
cd ~/Documents/personal/kb-cli && go build -o build/kb ./cmd/kb

# Test verification
go test -v -run "TestSynthesisIssueExists|TestOpenIssueExists" ./cmd/kb/...
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-07-design-recurring-problem-duplicate-synthesis-issues.md` - Root cause analysis that led to this fix

---

## Investigation History

**2026-01-08:** Investigation started
- Initial question: Fix kb-cli reflect dedup to fail-safe
- Context: Prior investigation identified root cause of 48 duplicate synthesis issues

**2026-01-08:** Fix implemented
- Changed both dedup functions to return true on error
- Added warning logs for diagnosis
- Updated tests to expect new behavior

**2026-01-08:** Investigation completed
- Status: Complete
- Key outcome: Dedup now fail-safe with logging
