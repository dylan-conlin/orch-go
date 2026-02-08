<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Three parsing bugs in `orch handoff` causing stale/incorrect output: active agents not filtered by beads status, work completed parsing wrong format, pending issues parsing wrong format.

**Evidence:** Before fix: 15 active agents shown (including completed). After fix: 3 active agents (only in_progress). All tests pass.

**Knowledge:** beads output format is `{beads-id} [priority] [type] {status} - {title}` with ` - ` separator, not `:` as assumed.

**Next:** Close issue - implementation complete with tests.

**Confidence:** High (90%) - All three problems fixed and tested, but edge cases in beads output format could exist.

---

# Investigation: orch handoff generates stale/incorrect data

**Question:** Why does orch handoff show completed agents and garbled data?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** Agent og-feat-orch-handoff-generates-22dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Active agents not filtered by beads status

**Evidence:** `gatherActiveAgents()` collected agents from tmux windows and OpenCode sessions without checking if the corresponding beads issue was still `in_progress`. Tmux windows persist after work is completed.

**Source:** `cmd/orch/handoff.go:220-290` (before fix)

**Significance:** Main cause of "completed agents showing as active" - 15 agents shown when only 3 were truly in_progress.

---

### Finding 2: Work Completed parsing assumed wrong format

**Evidence:** Code assumed format `issueID: title` and used `strings.SplitN(line, ":", 2)`. Actual format is:
```
orch-go-66n [P0] [task] closed [triage:ready] - Implement Synthesis Protocol...
```
The `:` split captured `[triage` as the title prefix.

**Source:** `cmd/orch/handoff.go:331-367` (before fix), `bd list --status closed` output

**Significance:** Caused "garbled data" symptom like `ready] - Implement Synthesis...`

---

### Finding 3: Pending issues parsing missed [type] field

**Evidence:** Code assumed format `1. [priority] beads-id: title` but actual format includes type:
```
1. [P2] [feature] orch-go-xwh: Iterate on Swarm Dashboard UI/UX
```
Parsing treated `[feature]` as the beads ID.

**Source:** `cmd/orch/handoff.go:293-328` (before fix), `bd ready` output

**Significance:** Pending issues were parsed incorrectly, wrong IDs extracted.

---

## Synthesis

**Key Insights:**

1. **beads output format consistency** - All beads outputs use a consistent format: `{beads-id} [{priority}] [{type}] {status} ... - {title}` with ` - ` as the title separator.

2. **Tmux state != work state** - Tmux windows persist after agent work completes. Must cross-reference with beads status.

3. **Test coverage gap** - Original code had tests for structure but not for actual parsing of beads output formats.

**Answer to Investigation Question:**

orch handoff showed stale/incorrect data because:
1. Active agents were collected from tmux windows without verifying beads status
2. Work completed parsing split on `:` instead of ` - `, capturing wrong content
3. Pending issues parsing missed the `[type]` field in beads output

All three issues have been fixed by:
1. Adding `getInProgressBeadsIDs()` to filter active agents
2. Using `strings.Index(line, " - ")` for work completed parsing
3. Scanning for field ending with `:` for pending issues parsing

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Fixes directly address all three documented symptoms. All existing tests pass, and new tests verify parsing logic.

**What's certain:**

- ✅ Active agents now filtered by beads in_progress status
- ✅ Work completed correctly parses beads-id and title
- ✅ Pending issues correctly parses beads-id, priority, and title
- ✅ All tests pass including new parsing tests

**What's uncertain:**

- ⚠️ Other beads output format variations (different locales, edge cases)
- ⚠️ Performance impact of additional `bd list --status in_progress` call

**What would increase confidence to Very High (95%+):**

- Integration testing with real beads data across multiple sessions
- Performance benchmarking of the additional beads call

---

## Implementation Recommendations

**Purpose:** Not applicable - implementation complete.

### Completed Implementation

**Changes made:**
1. Added `getInProgressBeadsIDs()` function to query beads for in_progress issues
2. Modified `gatherActiveAgents()` to filter by in_progress beads IDs
3. Fixed `gatherPendingIssues()` parsing to handle `[type]` field
4. Fixed `gatherRecentWork()` parsing to use ` - ` separator
5. Added comprehensive parsing tests

---

## References

**Files Examined:**
- `cmd/orch/handoff.go` - Main handoff implementation
- `cmd/orch/handoff_test.go` - Test file

**Commands Run:**
```bash
# Check current handoff output
orch handoff --json

# Check beads output formats
bd list --status in_progress
bd list --status closed
bd ready
```

**Related Artifacts:**
- **Issue:** orch-go-hey6

---

## Investigation History

**2025-12-22 16:23:** Investigation started
- Initial question: Why does orch handoff show completed agents and garbled data?
- Context: Discovered during session handoff that output was stale

**2025-12-22 16:35:** Root cause identified
- Three parsing bugs in handoff.go
- beads output format different than assumed

**2025-12-22 16:50:** Implementation completed
- Fixed all three parsing functions
- Added comprehensive tests
- All tests pass
