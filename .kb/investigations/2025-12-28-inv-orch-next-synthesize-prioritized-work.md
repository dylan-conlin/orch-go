<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Enhanced `orch next` command now synthesizes bd ready, orch patterns, and orch focus into prioritized work recommendations.

**Evidence:** Command runs successfully, outputs categorized recommendations (BLOCKER > FOCUS > MAINTENANCE > BACKLOG), supports --json and --verbose flags.

**Knowledge:** The existing codebase already had most infrastructure (beads client, patterns analyzer, focus store, retry patterns) - synthesis was just a matter of combining them.

**Next:** Close - implementation complete, tests passing.

---

# Investigation: Orch Next Synthesize Prioritized Work

**Question:** How can we synthesize multiple orchestrator signals (bd ready, orch patterns, orch focus) into actionable prioritized recommendations?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** Worker Agent (orch-go-578d)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Existing Infrastructure Available

**Evidence:** The codebase already contains:
- `pkg/beads/`: RPC and CLI clients for beads issue tracking
- `pkg/verify/attempts.go`: Retry pattern detection with `GetAllRetryPatterns()`
- `pkg/focus/focus.go`: North star focus tracking with drift detection
- `pkg/spawn/gap.go`: Gap tracker with `FindRecurringGaps()`
- `cmd/orch/patterns.go`: Pattern detection types and collection logic

**Source:** 
- `cmd/orch/focus.go:376-410` - getReadyIssues() function
- `pkg/verify/attempts.go:156-254` - GetAllRetryPatternsFromPath()
- `pkg/spawn/gap.go` - LoadTracker() and FindRecurringGaps()

**Significance:** No new packages needed - just a new command file that orchestrates existing functionality.

---

### Finding 2: Old Next Command Was Basic

**Evidence:** The previous `orch next` command in focus.go only provided simple suggestions:
- set-focus / start-work / continue / refocus
- Did not incorporate patterns, retries, or ready work details
- Did not provide actionable spawn commands

**Source:** `cmd/orch/focus.go:287-374`

**Significance:** The new implementation replaces this with a more comprehensive synthesis approach.

---

### Finding 3: Beads Ready API Provides Full Issue Details

**Evidence:** The beads RPC `Ready()` method returns full `Issue` structs with:
- ID, Title, Description
- Priority, IssueType, Status
- Labels, Dependencies

**Source:** `pkg/beads/types.go:119-137`, `pkg/beads/cli_client.go:77-94`

**Significance:** Enables rich recommendation generation with skill inference and focus matching.

---

## Synthesis

**Key Insights:**

1. **Signal Layering** - Different signals have natural priority: blockers (active failures) > focus-aligned work > maintenance > backlog.

2. **Focus Keyword Matching** - Simple keyword extraction from focus goal enables approximate matching to issue titles/descriptions without requiring explicit tagging.

3. **Actionable Commands** - Each recommendation includes a ready-to-run spawn command, reducing orchestrator decision fatigue.

**Answer to Investigation Question:**

Synthesis is achieved by:
1. Collecting blockers from retry/failure patterns (verify.GetAllRetryPatterns)
2. Collecting ready work from beads (beads.Ready)
3. Filtering out closed issues (batch status check)
4. Matching against current focus for prioritization
5. Collecting maintenance recommendations from gap patterns
6. Sorting by type priority then issue priority
7. Generating actionable spawn commands for each

---

## Structured Uncertainty

**What's tested:**

- ✅ Command builds and runs without errors (verified: `go build`, `orch next`)
- ✅ JSON output is valid (verified: `orch next --json`)
- ✅ Unit tests pass for helper functions (verified: `go test ./cmd/orch/... -run TestNext`)
- ✅ Focus alignment detection works (verified: ran with active focus)

**What's untested:**

- ⚠️ Performance with large numbers of patterns/issues (not benchmarked)
- ⚠️ Behavior when beads daemon is unavailable (fallback to CLI tested indirectly)
- ⚠️ Focus matching accuracy with complex goal descriptions

**What would change this:**

- Finding would be wrong if patterns APIs change incompatibly
- Finding would be wrong if beads Ready() output format changes

---

## Implementation Recommendations

### Recommended Approach ⭐

**New cmd/orch/next.go** - Create dedicated file with enhanced synthesis logic.

**Why this approach:**
- Clean separation from basic focus command
- Easy to extend with new signal sources
- Replaces old nextCmd via init() RemoveCommand pattern

**Trade-offs accepted:**
- Some code duplication with getReadyIssues (acceptable for clarity)
- Old basic next behavior is replaced (feature upgrade, not loss)

**Implementation sequence:**
1. Define recommendation types (BLOCKER, FOCUS, MAINTENANCE, BACKLOG)
2. Implement collectors for each signal source
3. Implement sorting and output formatting
4. Add tests for helper functions

---

## References

**Files Examined:**
- `cmd/orch/focus.go` - Old next command, focus/drift commands
- `cmd/orch/patterns.go` - Pattern collection logic
- `pkg/beads/types.go` - Issue and ReadyArgs types
- `pkg/verify/attempts.go` - Retry pattern detection
- `pkg/spawn/gap.go` - Gap tracker

**Files Created:**
- `cmd/orch/next.go` - Enhanced next command implementation
- `cmd/orch/next_test.go` - Unit tests

---

## Investigation History

**[2025-12-28 09:20]:** Investigation started
- Initial question: How to synthesize multiple signals into prioritized recommendations?
- Context: SPAWN_CONTEXT.md requested synthesized "orch next" command

**[2025-12-28 09:25]:** Analyzed existing infrastructure
- Found beads client, patterns analyzer, focus store, retry patterns already available

**[2025-12-28 09:35]:** Implementation complete
- Created cmd/orch/next.go with enhanced synthesis
- Created cmd/orch/next_test.go with unit tests
- Verified command runs successfully

**[2025-12-28 09:40]:** Investigation completed
- Status: Complete
- Key outcome: Enhanced orch next command synthesizes blockers, focus-aligned work, maintenance, and backlog into prioritized recommendations
