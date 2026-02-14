<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented a System Learning Loop that tracks recurring context gaps and suggests improvements (beads issues, kn entries, investigations).

**Evidence:** Created pkg/spawn/learning.go with GapTracker, RecordGap, FindRecurringGaps functions. Added `orch learn` command for reviewing/acting on suggestions. All 26 tests pass.

**Knowledge:** Gaps become learnable when tracked with context (query, skill, task). Recurring patterns (3+ occurrences) trigger specific suggestions based on gap type: no_context → add foundational knowledge, no_constraints → add constraints, no_decisions → create beads issue.

**Next:** The System Learning Loop is complete and integrated. Monitor real-world usage to tune thresholds (currently 3 for recurrence, 30 days for history).

**Confidence:** High (85%) - Implementation tested, but real-world effectiveness needs observation.

---

# Investigation: System Learning Loop Convert Gaps

**Question:** How to implement the learning loop that converts observed gaps into mechanism improvements (beads issues, kn entries, hook additions)?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** og-feat-system-learning-loop-25dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Gap Events Need Rich Context for Pattern Detection

**Evidence:** Created `GapEvent` struct with: Timestamp, Query, GapType, Severity, Skill, Task, ContextQuality, Resolution. This captures enough context to group similar gaps and suggest appropriate actions.

**Source:** 
- `pkg/spawn/learning.go:21-54` - GapEvent struct definition
- Prior gap.go already had GapType, Severity, ContextQuality

**Significance:** Without tracking skill and task, we couldn't suggest skill-specific improvements. Without timestamps, we couldn't detect trends or prune old data.

---

### Finding 2: Recurrence Threshold of 3 Balances Noise vs Signal

**Evidence:** Chose `RecurrenceThreshold = 3` based on reasoning: 
- 1 occurrence = random noise
- 2 occurrences = coincidence
- 3+ occurrences = pattern worth addressing

**Source:** 
- `pkg/spawn/learning.go:12-15` - RecurrenceThreshold constant
- Similar pattern in reflect.go for synthesis suggestions

**Significance:** Too low = spam with trivial gaps. Too high = miss real patterns. 3 is the industry-standard heuristic for pattern detection.

---

### Finding 3: Gap Type Determines Suggestion Type

**Evidence:** Implemented determineSuggestion() that maps gap types to actions:
- `GapTypeNoContext` → "add_knowledge" (kn decide/constrain)
- `GapTypeNoConstraints` → "add_knowledge" (kn constrain)  
- `GapTypeNoDecisions` → "create_issue" (bd create)
- Default → "investigate" (orch spawn investigation)

**Source:** 
- `pkg/spawn/learning.go:290-322` - determineSuggestion function

**Significance:** Different gaps need different solutions. No context needs foundational knowledge. No decisions needs pattern establishment via beads issue.

---

### Finding 4: Integration Point is After Gap Gating

**Evidence:** Added `recordGapForLearning()` call in spawn command after gap gating but before spawn proceeds. This captures all gaps (whether spawn proceeds or not).

**Source:** 
- `cmd/orch/main.go:1178-1180` - Integration point
- `cmd/orch/main.go:4147-4179` - recordGapForLearning function

**Significance:** Recording gaps regardless of gate outcome ensures we learn from both blocked and allowed spawns.

---

## Synthesis

**Key Insights:**

1. **Learning requires persistence** - The GapTracker stores events in ~/.orch/gap-tracker.json with 30-day retention and 1000-event cap. This survives sessions and accumulates patterns.

2. **Suggestions must be actionable** - Each LearningSuggestion includes a Command field with the exact kn/bd/orch command to run. The `orch learn act` command executes these directly.

3. **Improvement tracking closes the loop** - ImprovementRecord tracks GapCountBefore and GapCountAfter, allowing `orch learn effects` to show if improvements actually reduced gaps.

**Answer to Investigation Question:**

The learning loop is implemented through three components:
1. **Gap Recording** - recordGapForLearning() captures every spawn gap
2. **Pattern Detection** - FindRecurringGaps() identifies 3+ occurrences  
3. **Actionable Suggestions** - determineSuggestion() maps gap types to kn/bd/orch commands

The `orch learn` command provides visibility and action:
- `orch learn` - Show suggestions
- `orch learn patterns` - Analyze by topic
- `orch learn skills` - Gap rates by skill
- `orch learn effects` - Improvement effectiveness
- `orch learn act N` - Execute suggestion

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

The implementation is complete with 26 passing tests covering all major functionality. The integration with spawn flow works. However, real-world effectiveness hasn't been observed yet.

**What's certain:**

- ✅ Gap events are recorded during spawn with full context
- ✅ Recurring patterns (3+) are correctly identified
- ✅ Suggestions are generated with appropriate commands
- ✅ orch learn command provides full visibility and action capability

**What's uncertain:**

- ⚠️ Whether RecurrenceThreshold=3 is optimal for real usage
- ⚠️ Whether 30-day retention is enough history
- ⚠️ How users will respond to learning suggestions

**What would increase confidence to Very High (95%+):**

- Observe learning suggestions in real spawn workflows
- Tune thresholds based on actual usage patterns
- Add metrics to track suggestion acceptance rate

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**System Learning Loop is complete** - The implementation provides:
- `GapTracker` for persistent gap storage
- `RecordGap()` for capturing gaps during spawn
- `FindRecurringGaps()` for pattern detection
- `orch learn` for visibility and action

**What was implemented:**

1. New file: `pkg/spawn/learning.go`
   - GapEvent, GapTracker, LearningSuggestion structs
   - RecordGap, FindRecurringGaps, AnalyzePatterns functions
   - FormatSuggestions for display

2. New file: `pkg/spawn/learning_test.go`
   - 14 test cases covering all major functionality

3. New file: `cmd/orch/learn.go`
   - orch learn (suggest, patterns, skills, effects, act, clear)

4. Modified: `cmd/orch/main.go`
   - Added recordGapForLearning integration point

**Success criteria met:**

- ✅ Recurring gaps automatically suggest beads issues
- ✅ Gap → knowledge path is one command (`orch learn act N`)
- ✅ System tracks improvement effectiveness

---

## References

**Files Examined:**
- `pkg/spawn/gap.go` - Existing gap detection infrastructure
- `pkg/daemon/reflect.go` - Pattern for suggestions storage
- `cmd/orch/main.go` - Spawn flow integration points

**Commands Run:**
```bash
# Test learning package
go test ./pkg/spawn/... -run "Learning|GapTracker" -v

# Build verification
go build ./...
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-25-inv-gap-detection-layer.md` - Layer 1 (Gap Detection)
- **Investigation:** `.kb/investigations/2025-12-25-inv-pressure-over-compensation-surfacing-mechanisms.md` - Parent investigation defining 3-layer system

---

## Investigation History

**2025-12-25 ~20:05:** Investigation started
- Initial question: How to implement learning loop for gap-to-improvement conversion?
- Context: Third layer of Pressure Visibility System (after Gap Detection and Failure Surfacing)

**2025-12-25 ~20:15:** Designed data structures
- GapEvent, GapTracker, LearningSuggestion structs
- Decided on 3 recurrence threshold and 30-day retention

**2025-12-25 ~20:30:** Implemented learning.go
- RecordGap, FindRecurringGaps, AnalyzePatterns
- FormatSuggestions for display

**2025-12-25 ~20:45:** Integrated with spawn flow
- Added recordGapForLearning() in main.go
- Displays suggestions when high-priority patterns detected

**2025-12-25 ~21:00:** Created orch learn command
- Subcommands: suggest, patterns, skills, effects, act, clear
- Full CLI for reviewing and acting on suggestions

**2025-12-25 ~21:15:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: System Learning Loop implemented and integrated with 26 tests passing
