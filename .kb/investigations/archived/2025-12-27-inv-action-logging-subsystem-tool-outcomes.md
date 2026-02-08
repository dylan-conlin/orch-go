<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented action logging subsystem in pkg/action/ that tracks tool invocations and outcomes (success/empty/error/fallback), enabling behavioral pattern detection.

**Evidence:** Created ActionEvent struct, Logger, and Tracker with comprehensive tests. Integrated with existing `orch patterns` command to surface futile actions alongside retry patterns and knowledge gaps.

**Knowledge:** Action outcome tracking fills the gap identified in the prior investigation - tool failures are no longer ephemeral. Patterns like "Read on *.md returns empty 5 times" can now be detected and surfaced to orchestrators.

**Next:** Deploy with `make install`, monitor for patterns, consider PostToolUse hook integration for automatic logging.

---

# Investigation: Action Logging Subsystem Tool Outcomes

**Question:** Implement action outcome logging to enable behavioral pattern detection for orchestrator self-correction.

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Agent (orch-go-4oh7.1)
**Phase:** Complete
**Next Step:** None - implementation complete
**Status:** Complete

---

## Findings

### Finding 1: Prior Investigation Identified the Gap

**Evidence:** The investigation at `.kb/investigations/2025-12-27-inv-orchestrator-self-correction-mechanisms-problem.md` identified that:
- Current mechanisms track knowledge state (gaps, decisions, constraints) but not action outcomes
- Tool failures are ephemeral and untracked
- Self-correction requires observing action outcomes, not just knowledge state

**Source:** Prior investigation findings, lines 37-50

**Significance:** This implementation directly addresses the gap by creating a persistent action log that tracks tool invocations and their outcomes.

---

### Finding 2: Existing Infrastructure Provides Pattern

**Evidence:** 
- `pkg/events/logger.go` provides the JSONL logging pattern for events.jsonl
- `pkg/spawn/learning.go` provides the gap tracking and pattern detection pattern
- `pkg/verify/attempts.go` provides the retry pattern detection pattern

**Source:**
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/events/logger.go:26-85`
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/learning.go:56-112`

**Significance:** The implementation follows established patterns in the codebase, making it consistent and maintainable.

---

### Finding 3: Integration with Existing Patterns Command

**Evidence:**
- `cmd/orch/patterns.go` already exists and collects retry patterns and gap patterns
- Added `collectActionPatterns()` to include action outcome patterns in the unified patterns view
- PatternTypeFutileAction added as a new pattern type

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/patterns.go:111-121`

**Significance:** Orchestrators can see all behavioral patterns (retries, gaps, futile actions) in a single `orch patterns` command.

---

## Implementation

### Files Created

1. **pkg/action/action.go** - Core action logging subsystem
   - `ActionEvent` struct with Tool, Target, Outcome, SessionID, Workspace, Context
   - `Outcome` type with Success, Empty, Error, Fallback values
   - `Logger` for appending to `~/.orch/action-log.jsonl`
   - `Tracker` for loading and analyzing patterns
   - `ActionPattern` for grouped pattern detection
   - `FormatPatterns()` for human-readable output
   - `SuggestKnEntry()` for generating kn commands

2. **pkg/action/action_test.go** - Comprehensive tests
   - 14 test functions covering all functionality
   - Tests for logging, pattern detection, pruning, summary

### Files Modified

1. **cmd/orch/patterns.go** - Integration with existing patterns command
   - Added import for `github.com/dylan-conlin/orch-go/pkg/action`
   - Added `PatternTypeFutileAction` pattern type
   - Added `collectActionPatterns()` function
   - Integrated with `runPatterns()` to include action patterns

---

## Synthesis

**Key Insights:**

1. **Ephemeral to Persistent** - Tool outcomes that were previously lost after each tool call are now persisted to `~/.orch/action-log.jsonl`, enabling cross-session pattern detection.

2. **Pattern Threshold** - Actions need to occur 3+ times with non-success outcomes to be considered a pattern. This prevents noise from one-off failures.

3. **Target Normalization** - File paths are normalized to extension patterns (e.g., `/path/to/SYNTHESIS.md` -> `*.md`) for better pattern grouping.

4. **Integration, Not Isolation** - By integrating with the existing `orch patterns` command, orchestrators get a unified view of all behavioral patterns.

**Answer to Investigation Question:**

Action outcome logging is now implemented. The orchestrator can detect repeated futile actions by running `orch patterns` which shows:
- Retry patterns (from events.jsonl)
- Knowledge gaps (from gap-tracker.json)
- Futile actions (from action-log.jsonl) <- NEW

---

## Structured Uncertainty

**What's tested:**

- [x] ActionEvent struct serialization (verified: JSON marshaling in tests)
- [x] Logger appends to JSONL file (verified: TestLogger_Log)
- [x] Pattern detection with threshold (verified: TestTracker_FindPatterns)
- [x] Session-specific pattern detection (verified: TestTracker_FindPatternsForSession)
- [x] Pruning old events (verified: TestPrune)
- [x] Integration with patterns command (verified: go build succeeds)

**What's untested:**

- [ ] PostToolUse hook integration (not implemented - would require hook modification)
- [ ] Automatic logging during agent execution (requires orchestrator or hook integration)
- [ ] Performance at scale (no benchmarks, but design is O(n) file scan)

**What would change this:**

- Finding would be incomplete if the action log file format doesn't match hook output format
- Finding would need revision if patterns command output format changes

---

## Implementation Recommendations

**For Orchestrators:**

1. Run `orch patterns` regularly to see detected behavioral patterns
2. Use suggested `kn tried` commands to externalize learnings
3. Consider using `orch patterns --prune` to clean old events

**For Future Enhancement:**

1. **PostToolUse Hook Integration** - Add automatic logging to post-tool-use.sh
2. **SessionStart Surfacing** - Surface patterns in session start context
3. **Automatic kn Entries** - Optionally auto-create kn entries for severe patterns

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/events/logger.go` - Event logging pattern
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/learning.go` - Gap tracking pattern
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/attempts.go` - Retry pattern detection
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/patterns.go` - Patterns command

**Commands Run:**
```bash
# Build and test
go build ./...
go test ./pkg/action/... -v
go test ./... 
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-27-inv-orchestrator-self-correction-mechanisms-problem.md` - Problem identification
- **Implementation:** `pkg/action/` - This implementation

---

## Investigation History

**2025-12-27 ~18:30:** Investigation started
- Read prior investigation on orchestrator self-correction mechanisms
- Analyzed existing codebase patterns (events, learning, attempts)

**2025-12-27 ~19:00:** Implementation started
- Created pkg/action/action.go with ActionEvent, Logger, Tracker
- Created pkg/action/action_test.go with comprehensive tests
- All 14 tests passing

**2025-12-27 ~19:15:** Integration completed
- Added collectActionPatterns() to patterns.go
- Added PatternTypeFutileAction pattern type
- Full build and test suite passing

**2025-12-27 ~19:30:** Investigation completed
- Status: Complete
- Key outcome: Action logging subsystem implemented and integrated with orch patterns

---

## Self-Review

- [x] Real implementation completed (not just analysis)
- [x] Tests written and passing
- [x] Integration with existing patterns command
- [x] Question answered (how to track action outcomes for self-correction)
- [x] File complete (all sections filled)

**Self-Review Status:** PASSED
