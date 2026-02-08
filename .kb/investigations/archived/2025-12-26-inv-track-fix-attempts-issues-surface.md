## Summary (D.E.K.N.)

**Delta:** Added fix attempt tracking to surface retry patterns on beads issues - spawning now warns about repeat failures, new `orch retries` command shows all issues with retry history.

**Evidence:** Implemented pkg/verify/attempts.go with FixAttemptStats struct, integrated into spawn command, added tests - all passing.

**Knowledge:** Events.jsonl already tracks spawn/abandon/complete events with beads_id - scanning this provides retry history without new storage. Persistent failures (2+ spawns, 2+ abandons, 0 completions) suggest reliability-testing skill.

**Next:** close - Feature complete. Consider adding retry count to `orch status` output in future.

---

# Investigation: Track Fix Attempts Issues Surface

**Question:** How should we track fix attempts on issues to surface retry patterns and identify flaky issues?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Events.jsonl Already Tracks Spawn/Abandon/Complete Events

**Evidence:** The events logger in pkg/events/logger.go already logs:
- `session.spawned` - when agents spawn (includes beads_id in data)
- `agent.abandoned` - when agents are abandoned (includes beads_id in data)
- `agent.completed` - when agents complete (includes beads_id in data)

**Source:** pkg/events/logger.go:16-25, cmd/orch/main.go:812 (abandoned), cmd/orch/main.go:1460 (spawned)

**Significance:** No new storage mechanism needed - we can scan events.jsonl to count spawns, completions, and abandonments per beads issue.

---

### Finding 2: Retry Pattern Definition

**Evidence:** A retry pattern is when:
1. Multiple spawn attempts exist for the same beads ID (SpawnCount > 1)
2. At least one abandon occurred (AbandonedCount > 0)

A persistent failure is more severe:
- 2+ spawns with 2+ abandons and 0 completions

**Source:** Analysis of agent lifecycle patterns

**Significance:** These patterns indicate the issue may need a different approach (investigation or reliability-testing) rather than more direct fix attempts.

---

### Finding 3: Integration Points

**Evidence:** Two natural integration points:
1. **Spawn command** - Check retry history before spawning an issue, warn if pattern detected
2. **New `orch retries` command** - Show all issues with retry history for orchestrator awareness

**Source:** cmd/orch/main.go spawn flow analysis

**Significance:** Surfacing at spawn time creates friction that makes retry patterns "hard to ignore", while the retries command provides a summary view.

---

## Synthesis

**Key Insights:**

1. **Leverage existing events** - No new data collection needed; events.jsonl has full spawn/abandon/complete history with beads IDs

2. **Clear threshold definitions** - IsRetryPattern (>1 spawn, any abandon) and IsPersistentFailure (≥2 spawns, ≥2 abandons, 0 completions) provide actionable categorization

3. **Friction over reminder** - Displaying warnings at spawn time makes it harder to blindly respawn failing issues

**Answer to Investigation Question:**

Track fix attempts by scanning events.jsonl for session.spawned, agent.abandoned, and agent.completed events associated with each beads ID. Surface retry patterns by:
1. Warning at spawn time when an issue has retry history
2. Providing `orch retries` command to list all issues with retry patterns
3. Suggesting appropriate skills (reliability-testing for persistent failures, investigate-root-cause for retry patterns)

---

## Structured Uncertainty

**What's tested:**

- ✅ FixAttemptStats correctly counts spawns/abandons/completions (verified: unit tests passing)
- ✅ IsRetryPattern detects multiple spawns with abandons (verified: TestFixAttemptStats_IsRetryPattern)
- ✅ IsPersistentFailure detects severe pattern (verified: TestFixAttemptStats_IsPersistentFailure)
- ✅ Events file scanning handles missing files, malformed lines (verified: TestGetFixAttemptStatsFromPath)

**What's untested:**

- ⚠️ UI appearance of warnings in real terminal (not tested in CI)
- ⚠️ Performance with very large events files (not benchmarked)

**What would change this:**

- If events.jsonl format changes, parsing would break
- If beads_id is removed from event data, tracking would fail

---

## Implementation Recommendations

### Recommended Approach ⭐

**Scan events.jsonl for retry pattern detection** - Implemented in pkg/verify/attempts.go

**Why this approach:**
- Leverages existing event logging (no new storage)
- Simple file scanning (no database queries)
- Provides both point lookups and aggregate views

**Trade-offs accepted:**
- File scanning on every lookup (could cache in future)
- No real-time tracking (scans full file)

**Implementation sequence:**
1. ✅ Create FixAttemptStats struct with pattern detection methods
2. ✅ Implement GetFixAttemptStats to scan events.jsonl
3. ✅ Integrate warning into spawn command for --issue flag
4. ✅ Add `orch retries` command for aggregate view

---

## References

**Files Examined:**
- pkg/events/logger.go - Event types and logging
- cmd/orch/main.go - Spawn and abandon flows
- pkg/verify/check.go - Existing verification patterns

**Commands Run:**
```bash
# Run attempt tracking tests
go test ./pkg/verify/... -run Attempt -v

# Build to verify compilation
go build ./cmd/orch/...

# Run all tests
go test ./...
```

**Related Artifacts:**
- **Files created:** pkg/verify/attempts.go, pkg/verify/attempts_test.go
- **Files modified:** cmd/orch/main.go (spawn warning, retries command)

---

## Investigation History

**2025-12-26 18:XX:** Investigation started
- Initial question: How to track fix attempts to surface retry patterns
- Context: Issues being respawned after failures without visibility into history

**2025-12-26 18:XX:** Found events.jsonl already has needed data
- Discovered session.spawned, agent.abandoned, agent.completed events include beads_id

**2025-12-26 18:XX:** Implemented FixAttemptStats and tests
- Created pkg/verify/attempts.go with full pattern detection
- All unit tests passing

**2025-12-26 18:XX:** Integrated into spawn command
- Added warning when spawning issues with retry history
- Created `orch retries` command for aggregate view

**2025-12-26 18:XX:** Investigation completed
- Status: Complete
- Key outcome: Fix attempt tracking implemented via events.jsonl scanning, surfaced at spawn time and via retries command
