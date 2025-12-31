<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added stale bug check before spawning - checks git history for commits mentioning issue ID or keywords since issue creation, warns if potentially stale.

**Evidence:** All tests pass (9 test cases covering CheckStaleBug, ExtractKeywordsFromTitle, FormatStaleBugWarning), --skip-stale-check flag visible in orch spawn --help.

**Knowledge:** Stale bug detection uses: (1) issue ID matching in commits, (2) keyword extraction from issue title with stop word filtering, (3) since-time filter based on issue creation.

**Next:** Close - implementation complete and tested.

---

# Investigation: Add Stale Bug Check Before Spawning

**Question:** How to implement a stale bug check before spawning to prevent wasted agent time investigating bugs that were already fixed?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Feature Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** .kb/investigations/2025-12-30-inv-investigate-went-wrong-session-dec.md Finding 2
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Git Log Search Pattern Works for Issue Detection

**Evidence:** Created test with temp git repo, added commit with "[issue-abc]" in message. CheckStaleBug correctly identified the commit as potentially related.

**Source:** pkg/verify/stale_bug_test.go - TestCheckStaleBug/matching_issue_ID_in_commit_message

**Significance:** Using `git log --oneline --since=<time> --format=%h|%s|%an|%aI` provides sufficient data for matching while being fast.

---

### Finding 2: Keyword Matching Requires Careful Filtering

**Evidence:** Initial test expected "bug" to be extracted from "Fix authentication bug" but implementation correctly filters it as a common issue word. Stop words (the, is, a, for, etc.) and short words (<4 chars) are filtered.

**Source:** pkg/verify/stale_bug.go:149-180 - ExtractKeywordsFromTitle function

**Significance:** Filtering common words reduces false positives. For multi-keyword searches, require at least 2 matches to reduce noise.

---

### Finding 3: Integration Point Follows Existing Pattern

**Evidence:** Stale bug check is placed right after retry pattern check in runSpawnWithSkill(), following the same pattern of warning but not blocking.

**Source:** cmd/orch/main.go:1357-1370

**Significance:** Consistent with existing pre-spawn checks (retry patterns, failure report gates). Non-blocking warning with --skip-stale-check bypass option.

---

## Synthesis

**Key Insights:**

1. **Stale detection is a warning, not a gate** - Following the "Gate Over Remind" principle selectively. Since false positives are possible (commits may not actually fix the bug), warning is appropriate while blocking is not.

2. **Multi-signal approach reduces noise** - Checking both issue ID AND keywords from title provides coverage. Requiring 2+ keyword matches (for multi-keyword searches) reduces false positives.

3. **Since-time filter is critical** - Only checking commits after issue creation prevents false positives from historical commits that mention similar terms.

**Answer to Investigation Question:**

The stale bug check is implemented as a pre-spawn warning that:
1. Only applies to bug-type issues (issue_type == "bug")
2. Searches git commits since issue creation for issue ID or keyword matches
3. Displays a warning with related commits if potentially stale
4. Can be bypassed with --skip-stale-check flag

---

## Structured Uncertainty

**What's tested:**

- ✅ Issue ID matching in commit messages (verified: TestCheckStaleBug/matching_issue_ID_in_commit_message)
- ✅ Keyword extraction with stop word filtering (verified: TestExtractKeywordsFromTitle)
- ✅ Since-time filter respects issue creation (verified: TestCheckStaleBug/respects_since_time_filter)
- ✅ Warning format includes commit details (verified: TestFormatStaleBugWarning)
- ✅ --skip-stale-check flag visible in help (verified: orch spawn --help)

**What's untested:**

- ⚠️ Real-world false positive rate (not measured)
- ⚠️ Performance on large git histories (not benchmarked)
- ⚠️ beads.Show() returning CreatedAt in all cases (depends on beads RPC)

**What would change this:**

- Finding would be wrong if CreatedAt is often empty (would use 7-day fallback too often)
- Finding would be wrong if keyword matching produces too many false positives in practice

---

## Implementation Recommendations

**Purpose:** Implementation is complete. This section documents what was built.

### What Was Implemented

**pkg/verify/stale_bug.go** - Core stale bug detection logic:
- `StaleBugResult` and `RelatedCommit` types
- `CheckStaleBug()` - main git search function
- `CheckStaleBugForIssue()` - convenience wrapper with beads integration
- `ExtractKeywordsFromTitle()` - keyword extraction with stop word filtering
- `FormatStaleBugWarning()` - user-friendly warning format

**pkg/verify/stale_bug_test.go** - Test coverage:
- TestStaleBugResult_IsPotentiallyStale
- TestCheckStaleBug (git integration tests)
- TestExtractKeywordsFromTitle
- TestFormatStaleBugWarning

**cmd/orch/main.go** - Integration:
- Added `spawnSkipStaleCheck` flag variable
- Added `--skip-stale-check` flag in init()
- Added stale bug check in runSpawnWithSkill() after retry pattern check

---

## References

**Files Created/Modified:**
- pkg/verify/stale_bug.go - Created (new file)
- pkg/verify/stale_bug_test.go - Created (new file)
- cmd/orch/main.go - Modified (added flag and check)

**Commands Run:**
```bash
# Build verification
go build ./pkg/verify/...
go build ./cmd/orch/...

# Test execution
go test ./pkg/verify/... -run "TestStaleBug|TestExtractKeywords|TestFormatStaleBugWarning|TestCheckStaleBug" -v

# Flag verification
go run ./cmd/orch spawn --help | grep skip-stale
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2025-12-30-inv-investigate-went-wrong-session-dec.md - Original finding that motivated this implementation
- **Finding 2:** "Both Issues Were 'Already Fixed' But Kept Being Respawned"

---

## Investigation History

**2025-12-30 ~17:00:** Investigation started
- Initial question: How to implement stale bug check before spawning?
- Context: Finding 2 from session investigation identified respawning already-fixed bugs as a root cause of session chaos

**2025-12-30 ~17:10:** TDD approach - tests written first
- Created stale_bug_test.go with failing tests
- Tests covered all main functionality before implementation

**2025-12-30 ~17:30:** Implementation complete
- stale_bug.go created with all functions
- All tests passing

**2025-12-30 ~17:40:** Integration complete
- Added --skip-stale-check flag
- Integrated into spawn flow after retry pattern check
- Build successful, flag visible in --help

**2025-12-30 ~17:45:** Investigation completed
- Status: Complete
- Key outcome: Stale bug detection implemented with warning-based approach and bypass option
