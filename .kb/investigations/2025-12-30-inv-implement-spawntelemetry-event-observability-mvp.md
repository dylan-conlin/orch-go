<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented SpawnTelemetry event type that logs to events.jsonl at spawn time with context size, token estimates, and kb context statistics.

**Evidence:** All tests pass (7 tests in pkg/spawn/telemetry_test.go); build succeeds; telemetry integrates with WriteContext().

**Knowledge:** GapAnalysis already contains MatchStatistics with all the count data needed for telemetry; the existing JSONL logging infrastructure in pkg/events supports arbitrary event types.

**Next:** Close - implementation complete. Future work: add `orch observe` command to query telemetry data.

---

# Investigation: Implement SpawnTelemetry Event for Observability MVP

**Question:** How to implement SpawnTelemetry event type for spawn-time observability?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Worker agent (og-feat-implement-spawntelemetry-event-30dec)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: GapAnalysis Already Contains Match Statistics

**Evidence:** The `GapAnalysis` struct in `pkg/spawn/gap.go:59-76` already contains `MatchStatistics` with `TotalMatches`, `ConstraintCount`, `DecisionCount`, and `InvestigationCount` - exactly what the telemetry schema needs.

**Source:** pkg/spawn/gap.go:78-85

**Significance:** No need to create a separate mechanism to count kb context matches - the data is already available via `Config.GapAnalysis` which is populated during spawn.

---

### Finding 2: Events Logger Supports Arbitrary Event Types

**Evidence:** The `pkg/events/logger.go` uses a generic `Event` struct with a `map[string]interface{}` for `Data`, allowing any event type to be logged to events.jsonl.

**Source:** pkg/events/logger.go:27-33

**Significance:** The existing infrastructure can handle SpawnTelemetry without modification - we just add a new event type constant and logging function.

---

### Finding 3: WriteContext Is the Natural Collection Point

**Evidence:** `WriteContext()` in `pkg/spawn/context.go:471` is called for every spawn, already generates the full context, and has access to the Config with all necessary data.

**Source:** pkg/spawn/context.go:471-508, cmd/orch/main.go:1527

**Significance:** This is the perfect place to log telemetry - it has the generated context (for size calculation) and the Config (for metadata). Telemetry is logged as a side effect, with errors not blocking the spawn.

---

## Synthesis

**Key Insights:**

1. **Piggyback on existing infrastructure** - The JSONL logging pattern from pkg/events and the match statistics from GapAnalysis provide everything needed without new infrastructure.

2. **Non-blocking telemetry** - Telemetry errors are logged to stderr but don't fail the spawn - observability should never block critical path.

3. **TDD validation** - Test-first approach confirmed the schema design before implementation, catching JSON field naming issues early.

**Answer to Investigation Question:**

SpawnTelemetry was implemented in three parts:
1. `SpawnTelemetry` and `KBContextStats` structs in `pkg/spawn/telemetry.go`
2. `CollectSpawnTelemetry()` to gather data from Config and generated context
3. `LogSpawnTelemetry()` to write events to `~/.orch/events.jsonl`
4. Integration in `WriteContext()` to log telemetry at spawn time

---

## Structured Uncertainty

**What's tested:**

- ✅ SpawnTelemetry JSON serialization (verified: TestSpawnTelemetry_Serialization)
- ✅ KBContextStats omitted when nil (verified: TestSpawnTelemetry_OmitsEmptyKBContext)
- ✅ CollectSpawnTelemetry from Config (verified: TestCollectSpawnTelemetry)
- ✅ CollectSpawnTelemetry with GapAnalysis fallback (verified: TestCollectSpawnTelemetry_WithGapAnalysis)
- ✅ LogSpawnTelemetry writes to JSONL (verified: TestLogSpawnTelemetry)

**What's untested:**

- ⚠️ Integration test with actual orch spawn (not run during implementation)
- ⚠️ Query performance on large events.jsonl (not benchmarked)
- ⚠️ Concurrent spawn telemetry logging (file locking not tested under load)

**What would change this:**

- Finding would need revision if events.jsonl becomes a bottleneck
- If Config.GapAnalysis is not always populated, we'd need alternative data source

---

## Implementation Recommendations

### Recommended Approach ⭐

**JSONL Extension** - Add SpawnTelemetry as new event type to existing events.jsonl.

**Why this approach:**
- Reuses existing infrastructure (no new log files, parsers, or analysis code)
- Aligns with investigation findings that GapAnalysis has required data
- Non-blocking integration in WriteContext()

**Trade-offs accepted:**
- KBContextFormatResult (with truncation info) not passed through Config - using GapAnalysis fallback
- No real-time querying (JSONL is batch-oriented)

**Implementation sequence:**
1. Define SpawnTelemetry and KBContextStats structs
2. Implement CollectSpawnTelemetry with GapAnalysis fallback
3. Add LogSpawnTelemetry to write events
4. Integrate into WriteContext()

### Alternative Approaches Considered

**Option B: Extend existing events.Logger with method**
- **Pros:** Cleaner API, single logger instance
- **Cons:** Requires passing Logger to spawn package, circular dependency risk
- **When to use instead:** If events package is refactored to be a proper service

**Rationale for recommendation:** Standalone function with default path is simpler and maintains package independence.

---

### Implementation Details

**What was implemented:**
- `pkg/spawn/telemetry.go` - SpawnTelemetry struct, KBContextStats, CollectSpawnTelemetry, LogSpawnTelemetry
- `pkg/spawn/telemetry_test.go` - 7 tests covering serialization, collection, and logging
- `pkg/spawn/context.go` - Integration in WriteContext() with non-blocking error handling

**Things to watch out for:**
- ⚠️ Telemetry errors logged to stderr, not returned - this is intentional
- ⚠️ KBContextStats.WasTruncated is only accurate when KBContextFormatResult passed; GapAnalysis doesn't track truncation

**Areas needing further investigation:**
- Adding `orch observe` command for telemetry querying
- CompletionTelemetry for outcome correlation
- JSONL rotation if file grows too large

**Success criteria:**
- ✅ Events logged to ~/.orch/events.jsonl with type "spawn.telemetry"
- ✅ All tests pass
- ✅ Build succeeds

---

## References

**Files Examined:**
- `pkg/spawn/context.go` - WriteContext() integration point
- `pkg/spawn/config.go` - Config struct with GapAnalysis
- `pkg/spawn/gap.go` - GapAnalysis and MatchStatistics
- `pkg/spawn/kbcontext.go` - KBContextFormatResult, CharsPerToken
- `pkg/events/logger.go` - Event struct and logging pattern

**Commands Run:**
```bash
# Run telemetry tests
go test ./pkg/spawn/... -run "Test.*Telemetry" -v

# Build all packages
go build ./...
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-30-inv-design-observability-infrastructure-validating-principle.md` - Design investigation that specified the schema

---

## Investigation History

**2025-12-30 10:00:** Investigation started
- Initial question: How to implement SpawnTelemetry for observability MVP?
- Context: Spawned from orch-go-957w to implement design from prior investigation

**2025-12-30 10:30:** TDD implementation completed
- Wrote tests first, then implemented telemetry structs and functions
- All 7 tests passing

**2025-12-30 11:00:** Integration completed
- Added telemetry logging to WriteContext()
- Non-blocking error handling implemented

**2025-12-30 11:15:** Investigation completed
- Status: Complete
- Key outcome: SpawnTelemetry event type implemented and integrated with ~160 lines of code
