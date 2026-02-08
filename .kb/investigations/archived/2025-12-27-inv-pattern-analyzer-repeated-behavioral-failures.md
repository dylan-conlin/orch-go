<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Created pkg/patterns/ package with ActionLog and pattern detection for repeated behavioral failures, with full test coverage.

**Evidence:** All 24 tests pass. Tested: empty read detection, error detection, workspace context awareness, suppression/expiration, persistence.

**Knowledge:** Pattern detection requires: action events with outcomes, workspace context (tier, skill), and a threshold for repetition (3+ times). Light-tier SYNTHESIS.md reads are expected to be empty and marked as "info" severity.

**Next:** Integrate with action logging subsystem (issue 4oh7.1) which produces ActionEvent data, then create `orch patterns` command (issue 4oh7.3) to surface patterns to orchestrator.

---

# Investigation: Pattern Analyzer for Repeated Behavioral Failures

**Question:** How should we detect repeated actions with the same outcome (e.g., "Read SYNTHESIS.md returned empty 3 times in workspaces with .tier=light")?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Agent (orch-go-4oh7.2)
**Phase:** Complete
**Next Step:** None - implementation complete
**Status:** Complete

---

## Findings

### Finding 1: Pattern Detection Architecture

**Evidence:** Created `pkg/patterns/analyzer.go` with:
- `ActionEvent` struct for tracking tool actions with outcomes (success, empty, error, timeout)
- `ActionLog` for storing events with automatic pruning (7 days, max 500 events)
- `Pattern` struct for detected behavioral patterns with severity and suggestions
- `DetectPatterns()` method that identifies repeated empty reads and repeated errors

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/patterns/analyzer.go`

**Significance:** The architecture supports extensible pattern detection - new pattern types can be added by implementing additional detection methods similar to `detectRepeatedEmptyReads()` and `detectRepeatedErrors()`.

---

### Finding 2: Workspace Context Awareness

**Evidence:** ActionEvent includes `WorkspaceContext` map for metadata like tier, skill, phase. Pattern detection uses this context to:
- Adjust severity (SYNTHESIS.md reads in light-tier are "info", not "warning")
- Extract common context across pattern events (preserved in Pattern.Context)
- Generate context-aware suggestions

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/patterns/analyzer.go:53-65` (ActionEvent.WorkspaceContext)

**Significance:** This enables the specific use case from the epic: detecting "Read SYNTHESIS.md returned empty 3 times in workspaces with .tier=light" and appropriately marking it as expected behavior rather than a problem.

---

### Finding 3: Suppression System

**Evidence:** Implemented `SuppressedPattern` with:
- Pattern key for matching
- Optional expiration time
- Reason field for documentation
- Automatic pruning of expired suppressions

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/patterns/analyzer.go:89-105` (SuppressedPattern struct)

**Significance:** Allows orchestrator to acknowledge known patterns without them appearing repeatedly. Patterns can be permanently suppressed or set to expire after a duration.

---

## Synthesis

**Key Insights:**

1. **Decoupled Design** - The pattern analyzer consumes ActionEvent data but doesn't produce it. This allows the action logging subsystem (issue 4oh7.1) to evolve independently.

2. **Context-Aware Severity** - The same action (reading SYNTHESIS.md) can have different significance based on workspace context. Light-tier workspaces expect empty reads; standard-tier workspaces don't.

3. **Extensibility** - The `DetectPatterns()` method aggregates results from multiple detection functions. New pattern types (e.g., "repeated timeout", "cyclic navigation") can be added without modifying the core structure.

**Answer to Investigation Question:**

Pattern detection works by:
1. Grouping action events by normalized key (tool + target)
2. Filtering for events with the same outcome (empty, error, etc.)
3. Applying threshold (3+ occurrences)
4. Extracting common context across matching events
5. Generating severity and suggestions based on context

The example "Read SYNTHESIS.md returned empty 3 times in workspaces with .tier=light" is handled by:
- Detecting 3+ empty Read events for SYNTHESIS.md
- Finding common context: tier=light
- Setting severity to "info" (expected behavior in light-tier)
- Generating suggestion: "SYNTHESIS.md reads in light-tier workspaces are expected to be empty. Consider skipping this check for light-tier agents."

---

## Structured Uncertainty

**What's tested:**

- ✅ Empty read detection below, at, and above threshold (3 tests)
- ✅ Workspace context preservation and severity adjustment (2 tests)
- ✅ Repeated error detection with severity escalation (2 tests)
- ✅ Key normalization for grouping (path truncation, tool lowercase)
- ✅ Suppression and expiration logic (2 tests)
- ✅ Event pruning (age and count limits)
- ✅ Save/load persistence
- ✅ Multiple pattern type detection simultaneously

**What's untested:**

- ⚠️ Integration with actual action logging (requires 4oh7.1 completion)
- ⚠️ Performance with 500 events at scale
- ⚠️ Real-world pattern distribution and false positive rate

**What would change this:**

- Finding would need adjustment if action logging uses different outcome types
- Threshold of 3 may need tuning based on real usage patterns

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Package-based pattern analyzer** - Created `pkg/patterns/` as a standalone package that can be consumed by both the action logging subsystem and the `orch patterns` command.

**Why this approach:**
- Clean separation of concerns (logging vs detection vs surfacing)
- Testable in isolation
- Reusable across CLI and potential web dashboard

**Trade-offs accepted:**
- Requires coordination with 4oh7.1 for ActionEvent struct alignment
- JSON file storage (not a database) - acceptable for 500 event limit

**Implementation sequence:**
1. ✅ Pattern analyzer package (this issue)
2. ⏳ Action logging subsystem produces ActionEvent data (4oh7.1)
3. ⏳ `orch patterns` command consumes and displays patterns (4oh7.3)

---

### Implementation Details

**What to implement first:**
- Pattern analyzer is complete and tested

**Things to watch out for:**
- ⚠️ ActionEvent struct must match between 4oh7.1 (producer) and this package (consumer)
- ⚠️ Workspace context keys must be consistent (use "tier", "skill", "phase")

**Areas needing further investigation:**
- How to handle workspace context extraction in the action logging hook
- Whether to use JSON file or JSONL for action log persistence

**Success criteria:**
- ✅ All 24 tests pass
- ✅ Detects repeated empty reads with workspace context
- ✅ Supports suppression with optional expiration
- ✅ Provides formatted output for CLI display

---

## References

**Files Created:**
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/patterns/analyzer.go` - Core pattern analyzer implementation
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/patterns/analyzer_test.go` - Comprehensive test suite

**Commands Run:**
```bash
# Run tests
go test ./pkg/patterns/... -v
```

**Related Artifacts:**
- **Investigation:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-27-inv-orchestrator-self-correction-mechanisms-problem.md` - Root cause analysis that led to this epic
- **Epic:** orch-go-4oh7 - Parent epic for behavioral self-correction

---

## Investigation History

**2025-12-27 12:30:** Investigation started
- Initial question: How to detect repeated actions with same outcomes
- Context: Part of epic for orchestrator behavioral self-correction

**2025-12-27 13:00:** Implementation complete
- Created pkg/patterns/ package with full test coverage
- 24 tests passing

**2025-12-27 13:15:** Investigation completed
- Status: Complete
- Key outcome: Pattern analyzer package ready for integration with action logging subsystem
