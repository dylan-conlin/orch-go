<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Fixed `orch learn act` to generate meaningful reasons from gap context (skills, tasks, occurrence count) instead of placeholder TODOs.

**Evidence:** New `generateReasonFromGaps` function extracts skill names, task descriptions, and occurrence counts from gap events to create contextual reason strings.

**Knowledge:** Gap events contain rich context (skill, task, timestamp) that can be synthesized into human-readable reasons for kn entries.

**Next:** None - implementation complete with tests passing.

**Confidence:** High (90%) - All 27 tests pass, new functionality is straightforward.

---

# Investigation: Fix Orch Learn Act Generate

**Question:** How to fix `orch learn act` to generate real reasons from gap context instead of placeholder TODOs?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** og-feat-fix-orch-learn-25dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: determineSuggestion Generated Placeholder Reasons

**Evidence:** The `determineSuggestion` function at `pkg/spawn/learning.go:368` generated commands with hardcoded placeholder strings like `--reason "TODO: document decision"`.

**Source:** `pkg/spawn/learning.go:390, 397` - The kn decide/constrain commands had literal "TODO" strings.

**Significance:** This made `orch learn act` useless for automatically generating kn entries, requiring manual editing after running the command.

---

### Finding 2: Gap Events Contain Rich Context

**Evidence:** The `GapEvent` struct contains `Skill`, `Task`, `Query`, `Timestamp`, and `ContextQuality` fields that provide meaningful context about why the gap occurred.

**Source:** `pkg/spawn/learning.go:27-55` - GapEvent struct definition with all context fields.

**Significance:** This context can be synthesized into a useful reason string like "Used by: investigation, feature-impl. Occurred 5 times. Tasks: analyze auth flow; add db migration".

---

### Finding 3: Resolve Subcommand Missing

**Evidence:** The `orch learn` command had `act` but no way to mark gaps as resolved without running the suggested command.

**Source:** `cmd/orch/learn.go` - No resolve subcommand existed.

**Significance:** Users needed a way to mark gaps as resolved when they'd already addressed them through other means.

---

## Synthesis

**Key Insights:**

1. **Context extraction is straightforward** - Gap events contain skill, task, and occurrence data that can be directly formatted into reason strings.

2. **Reasons should be actionable** - Instead of "TODO", the reason should explain what was observed and why it matters.

3. **Resolution flexibility needed** - Users may resolve gaps through means other than the suggested command (e.g., manual kn entry, existing documentation).

**Answer to Investigation Question:**

Fixed by adding a `generateReasonFromGaps` function that:
1. Collects unique skills from gap events
2. Extracts up to 3 task descriptions (truncated to 40 chars)
3. Formats them as "Used by: <skills>. Occurred N times. Tasks: <tasks>"

Added `orch learn resolve` subcommand to mark gaps as resolved without running commands.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Implementation is straightforward, all tests pass, and the functionality works as expected.

**What's certain:**

- ✅ Reasons now include skill names and occurrence counts
- ✅ Tasks are included when available (truncated for length)
- ✅ Resolve subcommand allows manual resolution tracking
- ✅ All 27 tests pass

**What's uncertain:**

- ⚠️ Optimal reason format (may need tuning based on usage)
- ⚠️ Whether 40-char task truncation is ideal

---

## Implementation Recommendations

**Purpose:** Document what was implemented.

### Recommended Approach ⭐

**Generate contextual reasons from gap events** - Extract skills, tasks, and occurrence counts to create meaningful kn entry reasons.

**Implementation sequence:**
1. Added `generateReasonFromGaps(query, events)` function to `pkg/spawn/learning.go`
2. Modified `determineSuggestion` to call it instead of using TODOs
3. Added `orch learn resolve` subcommand with validation types

### What was implemented:

**pkg/spawn/learning.go:**
- Added `generateReasonFromGaps` function that:
  - Collects unique skills from events
  - Extracts up to 3 unique tasks (truncated to 40 chars)
  - Formats as "Used by: <skills>. Occurred N times. Tasks: <tasks>"
- Modified `determineSuggestion` to use real reasons

**cmd/orch/learn.go:**
- Added `learnResolveCmd` with resolution types: added_knowledge, created_issue, investigated, wont_fix, custom
- Added `runLearnResolve` function to record resolutions

**pkg/spawn/learning_test.go:**
- Added `TestGenerateReasonFromGaps` with 5 test cases

---

## References

**Files Examined:**
- `pkg/spawn/learning.go` - Core learning system implementation
- `cmd/orch/learn.go` - CLI command implementation
- `pkg/spawn/learning_test.go` - Existing tests

**Commands Run:**
```bash
# Build verification
go build ./...

# Run tests
go test ./pkg/spawn/... -v

# Full test suite
go test ./...
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-25-inv-system-learning-loop-convert-gaps.md` - Original learning loop implementation

---

## Investigation History

**2025-12-25:** Investigation started
- Initial question: How to fix orch learn act to generate real reasons?
- Context: Placeholder TODOs made the command less useful

**2025-12-25:** Implementation complete
- Added generateReasonFromGaps function
- Added orch learn resolve subcommand
- All 27 tests passing
- Final confidence: High (90%)
