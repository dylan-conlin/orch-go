## Summary (D.E.K.N.)

**Delta:** stats_cmd.go (1256 lines) is dominated by a single 1010-line `aggregateStats()` function. Prior extraction (types, output) already done. The remaining problem requires structural decomposition — introduce a `statsAggregator` struct to break the function into per-domain methods, then extract to `stats_aggregation.go`.

**Evidence:** Function-level analysis shows aggregateStats has 3 phases: map initialization (80 lines), event loop with 13 event types in a switch (488 lines), and post-aggregation calculations across 8 domains (440 lines). All share ~20 local maps.

**Knowledge:** Simple file splitting is insufficient for single-function bloat. The established extraction pattern (Phase 1: shared utils, Phase 2: domain code) must be adapted: introduce an intermediate struct to make the function decomposable, then extract.

**Next:** Create 1 implementation issue for the struct-based decomposition.

**Authority:** implementation — Follows extraction patterns, no new architectural decisions.

---

# Investigation: stats_cmd.go Extraction Design

**Question:** How should stats_cmd.go (1256 lines, fastest growth) be extracted before hitting the 1200-line proactive trigger?

**Started:** 2026-03-12
**Updated:** 2026-03-12
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None — implementation issue to be created
**Status:** Complete
**Model:** code-extraction-patterns

**Patches-Decision:** N/A
**Extracted-From:** .kb/investigations/2026-03-09-design-extraction-plan-three-near-critical.md (extends)

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-03-09-design-extraction-plan-three-near-critical.md | extends | Yes — Issues 1 & 2 completed (stats_types.go, stats_output.go exist) | Finding 3 underestimated aggregateStats complexity: was 790 lines, now 1010 |
| .kb/guides/code-extraction-patterns.md | applies | Yes — patterns confirmed | Single-function bloat not covered by existing patterns |

---

## Findings

### Finding 1: aggregateStats() is 1010 lines — the entire extraction problem

**Evidence:** Line 130-1140. Single function with 3 phases:

| Phase | Lines | Content |
|-------|-------|---------|
| Map initialization | 130-210 (~80 lines) | 20+ `make(map[...])` declarations for intermediate tracking state |
| Event loop | 211-698 (~488 lines) | `switch event.Type` with 13 event type handlers |
| Post-aggregation | 700-1139 (~440 lines) | 8 calculation domains building report sections |

**Source:** `cmd/orch/stats_cmd.go:130-1140`

**Significance:** File splitting (moving this function to another file) just relocates the problem. The function itself must be decomposed.

---

### Finding 2: 20+ shared maps create tight internal coupling

**Evidence:** The event loop and post-aggregation phases share these maps:

| Map | Used In Loop | Used In Post-Agg |
|-----|-------------|-------------------|
| `spawnTimes` | session.spawned | gate effectiveness (duration calc) |
| `spawnSkills` | session.spawned | agent.completed, agent.abandoned |
| `spawnBeadsIDs` | session.spawned | agent.completed, gate effectiveness |
| `completedBeadsIDs` | session.completed, agent.completed | gate effectiveness |
| `abandonedBeadsIDs` | agent.abandoned | gate effectiveness |
| `beadsCompletions` | agent.completed | (unused post-agg?) |
| `workspaceToSession` | session.spawned, agent.completed | (none) |
| `gateFailures` | verification.failed | verification stats |
| `gatesBypassed` | agent.completed, verification.bypassed | verification stats |
| `gatesAutoSkipped` | verification.auto_skipped | verification stats |
| `bypassReasons` | verification.bypassed | verification stats |
| `skillVerification` | agent.completed, verification.failed | verification stats |
| `spawnGateBypasses` | spawn.*.bypassed | spawn gate stats |
| `spawnGateReasons` | spawn.*.bypassed | spawn gate stats |
| `overrideReasons` | session.spawned, spawn.*.bypassed, agent.completed | override stats |
| `gateDecisionCounts` | spawn.gate_decision | gate decision stats |
| `gateBlockedSkills` | spawn.gate_decision | gate decision stats |
| `gatedBeadsIDs` | spawn.gate_decision | gate effectiveness |
| `blockedBeadsIDs` | spawn.gate_decision | gate effectiveness |
| `architectEscalatedIDs` | daemon.architect_escalation | gate effectiveness |
| `skillCounts` | session.spawned, session.completed, agent.completed, agent.abandoned | skill stats |
| `escapeHatchSpawns` | session.spawned | escape hatch stats |

**Significance:** These maps are the reason the function is monolithic — they can't be trivially passed between separate functions without a unifying struct.

---

### Finding 3: Post-aggregation has 8 independent calculation domains

**Evidence:** After the event loop (line 700+), the code processes independent report sections:

| Domain | Lines | Depends On (maps) | Output (report field) |
|--------|-------|-------------------|----------------------|
| Escape hatch | 704-744 (~40 lines) | escapeHatchSpawns, escapeHatchInWindow | EscapeHatchStats |
| Verification | 746-810 (~65 lines) | gateFailures, gatesBypassed, gatesAutoSkipped, bypassReasons, skillVerification | VerificationStats |
| Spawn gates | 812-855 (~44 lines) | spawnGateBypasses, spawnGateReasons | SpawnGateStats |
| Overrides | 857-899 (~43 lines) | overrideReasons | OverrideStats |
| Rates + duration | 901-956 (~56 lines) | durations, skillCounts | Summary rates, SkillStats |
| Gate decisions | 958-1021 (~64 lines) | gateDecisionCounts, gateBlockedSkills | GateDecisionStats |
| Gate effectiveness | 1023-1126 (~104 lines) | blockedBeadsIDs, gatedBeadsIDs, architectEscalatedIDs, completedBeadsIDs, abandonedBeadsIDs, spawnTimes, spawnBeadsIDs, events | GateEffectivenessStats |
| Coaching | 1128-1137 (~10 lines) | (none — reads from file) | CoachingStats |

**Significance:** Each domain reads from a subset of maps and writes to a distinct report section. They are independently extractable as methods on a shared state struct.

---

### Finding 4: The remaining 246 lines (non-aggregateStats) are well-structured

**Evidence:** Outside of aggregateStats, stats_cmd.go contains:

| Content | Lines | Notes |
|---------|-------|-------|
| Imports + vars + cobra cmd + init | 1-55 | Clean command setup |
| runStats() | 57-80 | Entry point: parse events → aggregate → output |
| getEventsPath() | 82-90 | Simple path resolution |
| parseEvents() | 92-128 | Event file reader |
| extractGateAccuracyBaseline() | 1145-1166 | Pure extraction from report |
| recordGateBaseline() | 1170-1222 | Baseline I/O + delta printing |
| printDelta() | 1224-1233 | Formatting helper |
| loadGateBaselines() | 1235-1256 | Baseline file reader |

**Significance:** These are already well-sized functions. The baseline functions (1145-1256, ~112 lines) could move to the new aggregation file but don't need to.

---

## Synthesis

### Recommended Approach: statsAggregator struct

Introduce a `statsAggregator` struct that encapsulates the 20+ intermediate maps, then decompose `aggregateStats()` into methods:

**Step 1: Create `cmd/orch/stats_aggregation.go`**

```go
// statsAggregator holds intermediate state during event aggregation.
type statsAggregator struct {
    report          *StatsReport
    days            int
    cutoffDays      int64
    cutoff7d        int64
    cutoff30d       int64

    // Spawn correlation
    spawnTimes      map[string]int64
    spawnSkills     map[string]string
    spawnBeadsIDs   map[string]string
    workspaceToSession map[string]string

    // Deduplication
    completedBeadsIDs map[string]bool
    abandonedBeadsIDs map[string]bool

    // Duration tracking
    durations       []float64

    // Verification
    gateFailures    map[string]int
    gatesBypassed   map[string]int
    gatesAutoSkipped map[string]int
    bypassReasons   map[string]int
    skillVerification map[string]*SkillVerificationStats

    // Spawn gates
    spawnGateBypasses map[string]int
    spawnGateReasons  map[string]int
    overrideReasons   map[string]int

    // Gate decisions
    gateDecisionCounts map[string]int
    gateBlockedSkills  map[string]int
    gatedBeadsIDs      map[string]bool
    blockedBeadsIDs    map[string]bool
    architectEscalatedIDs map[string]bool

    // Skill tracking
    skillCounts     map[string]*SkillStatsSummary
    beadsCompletions map[string]int64

    // Escape hatch
    escapeHatchSpawns []escapeHatchSpawn
    escapeHatchInWindow int
}
```

**Step 2: Decompose event loop into per-type methods**

```go
func (a *statsAggregator) processSessionSpawned(event StatsEvent) { ... }
func (a *statsAggregator) processSessionCompleted(event StatsEvent) { ... }
func (a *statsAggregator) processAgentCompleted(event StatsEvent) { ... }
func (a *statsAggregator) processAgentAbandoned(event StatsEvent) { ... }
func (a *statsAggregator) processVerificationEvent(event StatsEvent) { ... }
func (a *statsAggregator) processGateDecision(event StatsEvent) { ... }
func (a *statsAggregator) processSpawnGateBypassed(event StatsEvent) { ... }
func (a *statsAggregator) processDaemonEvent(event StatsEvent) { ... }
func (a *statsAggregator) processSessionEvent(event StatsEvent) { ... }
func (a *statsAggregator) processWaitEvent(event StatsEvent) { ... }
```

**Step 3: Extract post-aggregation into domain calculators**

```go
func (a *statsAggregator) calcEscapeHatchStats() { ... }      // ~40 lines
func (a *statsAggregator) calcVerificationStats() { ... }     // ~65 lines
func (a *statsAggregator) calcSpawnGateStats() { ... }        // ~44 lines
func (a *statsAggregator) calcOverrideStats() { ... }         // ~43 lines
func (a *statsAggregator) calcRatesAndDuration() { ... }      // ~56 lines
func (a *statsAggregator) calcGateDecisionStats() { ... }     // ~64 lines
func (a *statsAggregator) calcGateEffectiveness(events []StatsEvent) { ... } // ~104 lines
func (a *statsAggregator) calcCoachingStats() { ... }         // ~10 lines
```

**Step 4: Simplify aggregateStats() to orchestrator**

```go
func aggregateStats(events []StatsEvent, days int) *StatsReport {
    a := newStatsAggregator(days)

    for _, event := range events {
        if event.Timestamp >= a.cutoffDays {
            a.report.EventsAnalyzed++  // was eventsInWindow
        }
        switch event.Type {
        case "session.spawned":
            a.processSessionSpawned(event)
        case "session.completed":
            a.processSessionCompleted(event)
        // ... etc
        }
    }

    a.calcEscapeHatchStats()
    a.calcVerificationStats()
    a.calcSpawnGateStats()
    a.calcOverrideStats()
    a.calcRatesAndDuration()
    a.calcGateDecisionStats()
    a.calcGateEffectiveness(events)
    a.calcCoachingStats()

    return a.report
}
```

### File Size Estimates After Extraction

| File | Before | After | Change |
|------|--------|-------|--------|
| stats_cmd.go | 1256 | ~200 | -1056 (command setup + entry + parse + baseline funcs) |
| stats_aggregation.go | (new) | ~1100 | (struct + constructor + event processors + calculators) |

**Wait — that's just moving the problem!**

True, but with a critical difference: stats_aggregation.go will contain ~18 focused methods averaging ~55 lines each, not one 1010-line function. Each method is independently testable and reviewable. The file is larger in total lines but dramatically more maintainable.

If 1100 lines is still concerning, a further split is possible:
- `stats_aggregation.go` — struct + constructor + event processors (~600 lines)
- `stats_calculations.go` — post-aggregation calculators (~500 lines)

### Alternative Considered: Extract Post-Aggregation Only

Move only the 8 calc functions (440 lines) to a separate file, keep event loop in stats_cmd.go. This avoids the struct but requires passing 20+ maps as function parameters:

```go
func calcEscapeHatchStats(report *StatsReport, spawns []escapeHatchSpawn, inWindow int, cutoff7d, cutoff30d int64) { ... }
```

**Rejected because:** 20+ map parameters is a code smell worse than the monolithic function. The struct approach is cleaner and the Go idiom for this pattern.

### Things to Watch Out For

- **Test coverage:** `stats_test.go` (1891 lines) tests `aggregateStats` via `runStats`. The refactored function should produce identical output — run existing tests as regression.
- **The events parameter on calcGateEffectiveness:** This calculator re-iterates `events[]` to find completion events per beads_id. Consider optimizing during decomposition (pre-build a map in the event loop).
- **escapeHatchSpawn type:** Currently declared inside `aggregateStats()` as a local type. Move to `statsAggregator` or `stats_types.go`.

---

## Structured Uncertainty

**What's tested:**
- ✅ Full file read and line-by-line analysis
- ✅ Map dependency tracking between event loop and post-aggregation
- ✅ Prior extraction results verified (stats_types.go 298, stats_output.go 465)
- ✅ Git log confirms 11 commits in ~30 days (fast growth confirmed)

**What's untested:**
- ⚠️ Whether all 20+ maps are truly needed (some may be dead code after recent changes)
- ⚠️ Exact line counts after refactoring (method overhead adds ~2 lines per method)
- ⚠️ Whether stats_test.go needs updates for struct-based API

**What would change this:**
- If most growth comes from new event types (likely), the struct approach scales well — just add a new process method
- If coaching.ReadMetricsSince moves to a different architecture, calcCoachingStats can be removed

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Introduce statsAggregator struct | implementation | Standard Go pattern for decomposing state-heavy functions |
| Split into 2 files if >1100 lines | implementation | Follows project's 800-line target |
| Keep baseline functions in stats_cmd.go | implementation | They're small (112 lines) and logically part of the command |

### Implementation Issue

**Title:** Decompose aggregateStats() into statsAggregator struct with per-domain methods

**Scope:**
1. Create `cmd/orch/stats_aggregation.go` with `statsAggregator` struct
2. Move `escapeHatchSpawn` type to stats_types.go
3. Extract event processors as methods (10 methods, ~488 lines total)
4. Extract post-aggregation calculators as methods (8 methods, ~440 lines total)
5. Reduce `aggregateStats()` in stats_cmd.go to ~30-line orchestrator
6. Run `go test ./cmd/orch/...` — all existing tests must pass unchanged
7. If stats_aggregation.go > 1000 lines, split into `stats_aggregation.go` + `stats_calculations.go`

**Risk:** Low — pure refactoring, all behavior preserved, existing tests as regression safety net.

---

## References

**Files Examined:**
- `cmd/orch/stats_cmd.go` (1256 lines) — full read, function-level analysis
- `cmd/orch/stats_types.go` (298 lines) — size check
- `cmd/orch/stats_output.go` (465 lines) — size check
- `cmd/orch/stats_test.go` (1891 lines) — size check
- `.kb/investigations/2026-03-09-design-extraction-plan-three-near-critical.md` — prior work
- `.kb/guides/code-extraction-patterns.md` — established patterns

---

## Investigation History

**2026-03-12:** Investigation started
- Context: stats_cmd.go at 1256 lines with fastest growth rate (11 commits/30d)
- Prior Issues 1 & 2 completed (stats_types.go, stats_output.go exist)
- Remaining problem: single 1010-line aggregateStats() function

**2026-03-12:** Design complete
- Recommended: statsAggregator struct-based decomposition
- Expected result: stats_cmd.go → ~200 lines, stats_aggregation.go → ~1100 lines (18 methods)
