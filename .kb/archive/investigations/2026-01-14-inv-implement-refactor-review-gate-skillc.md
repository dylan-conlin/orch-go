<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented refactor review gate that blocks deploy when token count decreases >10% without acknowledgment.

**Evidence:** All tests pass (4 test functions, 14 test cases) validating threshold detection, error/warning classification, and decrease calculation.

**Knowledge:** Stats.json provides reliable token history; integrating with existing load-bearing pattern registry enables comprehensive refactor safety.

**Next:** Consider adding --force-refactor flag to check/deploy commands to acknowledge reviews.

**Promote to Decision:** recommend-no (tactical implementation, follows existing pattern system)

---

# Investigation: Implement Refactor Review Gate Skillc

**Question:** How to implement a gate that warns/blocks accidental removal of load-bearing guidance during skill refactors?

**Started:** 2026-01-14
**Updated:** 2026-01-14
**Owner:** orch-go-lv3yx.6 worker
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Existing Load-Bearing Pattern System

**Evidence:** skillc already has a load-bearing pattern validation system in checker.go with `LoadBearingEntry`, `LoadBearingPattern`, `LoadBearingResult`, and `ValidateLoadBearing()` function. Patterns are defined in skill.yaml with pattern, provenance, evidence, and severity fields.

**Source:** `/Users/dylanconlin/Documents/personal/skillc/pkg/checker/checker.go:38-51`, `/Users/dylanconlin/Documents/personal/skillc/pkg/compiler/manifest.go:21-27`

**Significance:** The refactor review gate can leverage this existing system rather than creating a new one. Patterns registered as load-bearing should be verified after any significant content removal.

---

### Finding 2: Stats.json Provides Build History

**Evidence:** stats.json contains an array of build records with timestamps, total tokens, and per-source token counts. The `Stats` type has methods like `LastBuild()`, `GetGrowthRate()`, and `GetMonthlyChange()` for analyzing trends.

**Source:** `/Users/dylanconlin/Documents/personal/skillc/pkg/compiler/stats.go:31-161`

**Significance:** Token history comparison is straightforward - compare current build to `LastBuild()` to detect significant decreases.

---

### Finding 3: Checker Integration Pattern

**Evidence:** The `Check()` function aggregates multiple validation results (budget, checksum, links, load-bearing patterns) and returns a unified `CheckResult`. `HasErrors()` and `HasWarnings()` methods classify validation outcomes.

**Source:** `/Users/dylanconlin/Documents/personal/skillc/pkg/checker/checker.go:341-397`

**Significance:** Adding refactor review follows the established pattern - add result type, validation function, and integrate into Check(). Classify unacknowledged review as error (blocks deploy) and triggered review as warning (informational).

---

## Synthesis

**Key Insights:**

1. **Token decrease is a proxy for content removal** - A >10% decrease in tokens strongly indicates significant content was removed, warranting review of whether load-bearing patterns were affected.

2. **Blocking by default is safer** - Requiring acknowledgment (--force-refactor) ensures intentional review rather than accidental removal.

3. **Leveraging existing registry** - The load-bearing patterns already registered for a skill represent hard-won knowledge about what guidance is critical. Listing these during refactor review helps human reviewers know what to check.

**Answer to Investigation Question:**

The refactor review gate is implemented by:
1. Adding `RefactorReviewResult` type to track trigger state, token change, and patterns to review
2. Adding `ValidateRefactorReview()` that compares current tokens to last build from stats.json
3. Integrating into `Check()` and classifying unacknowledged reviews as blocking errors
4. Updating CLI output to show review requirements and guidance

---

## Structured Uncertainty

**What's tested:**

- ✅ Threshold detection works correctly (verified: 14 test cases covering edge cases)
- ✅ HasErrors() blocks on unacknowledged review (verified: TestCheckResult_HasErrors_WithRefactorReview)
- ✅ Decrease calculation is accurate (verified: TestRefactorReviewDecreaseCalculation with 20%, 5%, and -10%)

**What's untested:**

- ⚠️ End-to-end flow in actual skill compilation (not run against real skill)
- ⚠️ --force-refactor flag implementation (designed but not implemented)
- ⚠️ JSON output handling (code written, not manually verified)

**What would change this:**

- Finding would be wrong if stats.json format changes and LastBuild() returns incorrect data
- Design would need revision if 10% threshold is too sensitive or too lenient in practice

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach: Current Implementation (Done)

**Add refactor review gate to checker** - Validate token decrease against threshold, block deploy if unacknowledged.

**Why this approach:**
- Integrates with existing checker pattern
- Uses reliable stats.json history
- Lists known load-bearing patterns to guide human review

**Trade-offs accepted:**
- 10% threshold is somewhat arbitrary (can be adjusted via constant)
- Requires --force-refactor flag for acknowledgment (not yet implemented in CLI args)

**Implementation sequence:**
1. Add types to checker.go (done)
2. Implement validation function (done)
3. Integrate into Check() (done)
4. Update CLI output (done)
5. Add tests (done)
6. Add --force-refactor flag to CLI (future work)

---

### Implementation Details

**What was implemented:**
- `RefactorReviewResult` type with all tracking fields
- `ValidateRefactorReview()` function
- Integration into `Check()` loading stats.json
- CLI human-readable output in `printCheckResult()`
- JSON output in `checkJSON()`
- Comprehensive test coverage

**Things to watch out for:**
- ⚠️ Stats.json may not exist for new skills (handled: graceful skip)
- ⚠️ Token count of 0 in previous build would cause division by zero (handled: guard clause)

**Areas needing further investigation:**
- Optimal threshold value (10% is current default)
- UX for --force-refactor acknowledgment pattern

**Success criteria:**
- ✅ `skillc check` reports token decrease when significant
- ✅ Deploy is blocked when unacknowledged
- ✅ Load-bearing patterns are listed for review guidance

---

## References

**Files Modified:**
- `/Users/dylanconlin/Documents/personal/skillc/pkg/checker/checker.go` - Added types and validation
- `/Users/dylanconlin/Documents/personal/skillc/pkg/checker/checker_test.go` - Added tests
- `/Users/dylanconlin/Documents/personal/skillc/cmd/skillc/main.go` - Added CLI output

**Commands Run:**
```bash
# Build verification
cd ~/Documents/personal/skillc && go build -o /dev/null ./...

# Test run
cd ~/Documents/personal/skillc && go test ./pkg/checker/... -v -run "Refactor"

# Full test suite
cd ~/Documents/personal/skillc && go test ./...
```

---

## Investigation History

**2026-01-14 14:30:** Investigation started
- Initial question: How to implement refactor review gate for skillc?
- Context: Spawned from orch-go-lv3yx.6 to prevent accidental removal of load-bearing guidance

**2026-01-14 14:35:** Found existing systems
- stats.json provides token history
- Load-bearing pattern system already exists in checker

**2026-01-14 14:50:** Implementation complete
- All code written and tested
- 4 test functions with 14 test cases pass

**2026-01-14 14:55:** Investigation completed
- Status: Complete
- Key outcome: Refactor review gate implemented and tested
