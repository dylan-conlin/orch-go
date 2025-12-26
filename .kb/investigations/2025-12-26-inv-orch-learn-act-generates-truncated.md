<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Fixed `generateReasonFromGaps` to always produce at least 20 characters by prepending "Recurring gap for topic: X" when the base reason is too short.

**Evidence:** All 49 tests pass including new tests for minimum length validation; previously "Occurred 3 times" (16 chars) would fail kn's 20-char minimum.

**Knowledge:** When gap events lack skill/task metadata, the generated reason was just "Occurred N times" which is 16-18 chars. kn requires 20+ chars for --reason flag.

**Next:** None - implementation complete with tests.

**Confidence:** High (90%) - Comprehensive test coverage for all command patterns.

---

# Investigation: Orch Learn Act Generates Truncated kn Command

**Question:** Why does `orch learn act` generate kn commands that fail validation?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** og-feat-orch-learn-act-26dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: generateReasonFromGaps Produces Short Strings

**Evidence:** When gap events have no skill or task metadata, the function only outputs "Occurred N times". For 3 events (the minimum recurrence threshold), this is "Occurred 3 times" = 16 characters.

**Source:** `pkg/spawn/learning.go:424-482` (generateReasonFromGaps function)

**Significance:** This is below kn's required 20-character minimum for the --reason flag, causing the command to fail validation when executed.

---

### Finding 2: kn Requires 20-Character Minimum for Reason

**Evidence:** The `kn decide` and `kn constrain` commands require `--reason` to be at least 20 characters:
```
kn decide <content>         Record a decision (requires --reason, min 20 chars)
```

**Source:** `kn --help` output; `~/bin/kn` binary

**Significance:** This is a hard validation constraint that cannot be bypassed.

---

### Finding 3: No Prior Validation for Minimum Length

**Evidence:** The `ValidateCommand` function checked for flag presence and argument counts but did not validate the length of the reason string.

**Source:** `pkg/spawn/learning.go:755-780` (validateKnCommand before fix)

**Significance:** Invalid commands were not caught until execution time, leading to confusing runtime errors.

---

## Synthesis

**Key Insights:**

1. **Short reasons occur with sparse metadata** - Gap events collected during spawns may lack skill/task context if the spawn configuration doesn't include them.

2. **Validation should happen at generation time** - By ensuring generateReasonFromGaps always produces valid output, we prevent invalid commands from being generated.

3. **Defense in depth** - Adding validation in ValidateCommand catches any future cases where short reasons might slip through.

**Answer to Investigation Question:**

The kn commands failed because `generateReasonFromGaps` produced reasons shorter than 20 characters when gap events lacked skill/task metadata. The fix ensures minimum length by prepending "Recurring gap for topic: X" when needed, and adds validation in `ValidateCommand` as a safety net.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Comprehensive test coverage for all scenarios, and the fix addresses the root cause directly.

**What's certain:**

- ✅ generateReasonFromGaps now always produces 20+ character reasons
- ✅ ValidateCommand catches short reasons as a safety net
- ✅ All 49 tests pass including new edge case tests

**What's uncertain:**

- ⚠️ Edge cases with unusual Unicode characters not tested
- ⚠️ Interaction with other kn validation requirements not fully explored

**What would increase confidence to Very High (95%+):**

- End-to-end test running actual kn command
- Test with real gap tracker data from production

---

## Implementation Recommendations

**Purpose:** Document what was implemented.

### Recommended Approach ⭐

**Minimum length enforcement with query context padding** - When the base reason is short, prepend "Recurring gap for topic: {query}" to ensure 20+ characters.

**Why this approach:**
- Adds meaningful context rather than arbitrary padding
- The query is always available in the function
- Produces human-readable reasons

**Trade-offs accepted:**
- Slightly longer reason strings when padding is needed
- Format change for short reasons (acceptable as it provides more context)

**Implementation sequence:**
1. Added `MinReasonLength = 20` constant
2. Modified generateReasonFromGaps to check length and pad if needed
3. Added validation in validateKnCommand for defense in depth
4. Added tests for minimum length scenarios

---

## References

**Files Examined:**
- `pkg/spawn/learning.go` - generateReasonFromGaps and validateKnCommand functions
- `pkg/spawn/learning_test.go` - Existing tests
- `cmd/orch/learn.go` - CLI command that calls ValidateCommand

**Commands Run:**
```bash
# Check kn requirements
kn --help

# Run tests
go test ./pkg/spawn/... -v -run "GenerateReason|ValidateCommand"

# Full test suite
go test ./...
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-26-inv-orch-learn-act-commands-should.md` - Prior fix for shell parsing

---

## Investigation History

**2025-12-26:** Investigation started
- Initial question: Why does orch learn act generate invalid kn commands?
- Context: User reported kn constrain command failing with truncated reason

**2025-12-26:** Root cause identified
- generateReasonFromGaps produces "Occurred N times" (16 chars) when no skill/task
- kn requires 20+ chars for --reason

**2025-12-26:** Implementation complete
- Added MinReasonLength constant and padding logic
- Added validation in validateKnCommand
- Added 3 new test cases
- Final confidence: High (90%)
- Status: Complete
- Key outcome: orch learn act now generates valid kn commands with 20+ char reasons
