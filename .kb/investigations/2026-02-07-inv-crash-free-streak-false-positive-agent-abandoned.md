# Investigation: Crash-Free Streak False Positive from Agent Abandonment

**Question:** Why does `orch abandon` reset the crash-free streak when it's routine hygiene, not infrastructure failure?

**Started:** 2026-02-07
**Updated:** 2026-02-07
**Owner:** Worker agent (orch-go-21474)
**Phase:** Investigation
**Status:** Active

## Prior Work

| Investigation/Decision | Relationship | Verified | Conflicts |
|----------------------|--------------|----------|-----------|
| `.kb/decisions/2026-01-14-separate-observation-from-intervention.md` | context (observation/intervention distinction) | Yes | None |
| `.kb/investigations/2026-02-07-design-automatic-stability-measurement.md` | context (Phase 3 design) | Yes | None |

---

## Findings

### Finding 1: Agent Abandonment Currently Treated as Streak-Breaking Intervention

**Evidence:** In `cmd/orch/abandon_cmd.go` lines 475-481, the `logAbandonmentEvent` function records a stability intervention:

```go
// Record stability intervention (agent abandonment breaks the clean-session streak)
recorder := stability.NewRecorder(stability.DefaultPath())
detail := fmt.Sprintf("%s abandoned", ctx.BeadsID)
if ctx.Reason != "" {
    detail = fmt.Sprintf("%s abandoned (%s)", ctx.BeadsID, ctx.Reason)
}
recorder.RecordIntervention(stability.SourceAgentAbandoned, detail, nil, ctx.BeadsID)
```

**Source:** `cmd/orch/abandon_cmd.go:475-481`

**Significance:** Every `orch abandon` call records an intervention with source `SourceAgentAbandoned`, which breaks the crash-free streak. The comment explicitly states "agent abandonment breaks the clean-session streak."

### Finding 2: All Interventions Currently Reset the Streak

**Evidence:** In `pkg/stability/stability.go` lines 197-201, the streak calculation finds the most recent intervention regardless of source:

```go
// Compute streak: time since last intervention
if lastInterventionTs > 0 {
    t := time.Unix(lastInterventionTs, 0)
    report.LastIntervention = &t
    report.CurrentStreak = now.Sub(t)
}
```

The `ComputeReport` function scans all entries with `Type == TypeIntervention` (line 182) and finds the latest timestamp (lines 164-167), regardless of the `Source` field.

**Source:** `pkg/stability/stability.go:164-167, 197-201`

**Significance:** The streak calculation doesn't filter interventions by source. All intervention types (manual_recovery, agent_abandoned, doctor_fix) equally reset the streak.

### Finding 3: Intervention Sources Are Well-Defined Constants

**Evidence:** In `pkg/stability/stability.go` lines 22-27:

```go
// Intervention sources — what triggered the streak-breaking event.
const (
    SourceManualRecovery = "manual_recovery" // Service recovered without daemon action
    SourceAgentAbandoned = "agent_abandoned" // Agent abandoned via orch abandon
    SourceDoctorFix      = "doctor_fix"      // Manual orch doctor --fix invocation
)
```

**Source:** `pkg/stability/stability.go:22-27`

**Significance:** The intervention sources are already categorized. We can use these constants to distinguish infrastructure failures from hygiene operations.

### Finding 4: Crash-Free Streak Metric Used in Operator Health API

**Evidence:** In `cmd/orch/serve_operator_health.go` lines 193-237, the `buildCrashFreeStreakMetric` function:
- Calls `stability.ComputeReport()` (line 199)
- Returns the current streak (lines 209-212)
- Reads latest intervention for display (lines 214-225)
- Sets health status based on streak duration (lines 227-234)

**Source:** `cmd/orch/serve_operator_health.go:193-237`

**Significance:** The crash-free streak is a public API metric visible in the operator health dashboard. Fixing this affects operator health reporting.

---

## Synthesis

**Key Insights:**

1. **The problem is in two places:**
   - `abandon_cmd.go` records all abandonments as interventions
   - `stability.go` treats all interventions equally when computing streak

2. **The distinction already exists in decisions:**
   - `.kb/decisions/2026-01-14-separate-observation-from-intervention.md` defines observation (passive) vs intervention (active)
   - Agent abandonment is hygiene (observation that something is stuck), not recovery intervention

3. **Infrastructure failures that SHOULD reset streak:**
   - OOM kill (would appear as service crash → manual recovery)
   - Service crash without daemon auto-restart (manual_recovery)
   - Zombie accumulation requiring `orch doctor --fix` (doctor_fix)
   - DB corruption requiring manual rebuild (would need new intervention source)

4. **Hygiene operations that should NOT reset streak:**
   - Routine `orch abandon` of stuck agents (agent_abandoned)
   - This is expected operational maintenance, not system failure

**Answer to Investigation Question:**

Agent abandonment is currently treated as a streak-breaking intervention because:
1. It records an intervention with `SourceAgentAbandoned`
2. The streak calculation doesn't filter by intervention source

The fix requires distinguishing "infrastructure health interventions" from "agent health interventions":
- Infrastructure interventions (manual_recovery, doctor_fix) indicate system stability problems → reset streak
- Agent interventions (agent_abandoned) indicate individual agent issues → don't reset streak

---

## Structured Uncertainty

**What's tested:**
- ✅ `abandon_cmd.go` line 481 calls `RecordIntervention` with `SourceAgentAbandoned`
- ✅ `stability.go` lines 164-167 track last intervention timestamp regardless of source
- ✅ `stability.go` lines 197-201 compute streak from last intervention without filtering

**What's untested:**
- ⚠️ Whether there are OTHER places that record `SourceAgentAbandoned` interventions besides abandon command
- ⚠️ Whether DB corruption or OOM kill are currently captured as interventions at all
- ⚠️ Impact on existing stability.jsonl data (will old agent_abandoned entries still reset streak?)

**What would change this:**
- If agent abandonment rate is itself a reliability signal (high abandonment = poor agent quality), then it should reset streak
- If future intervention sources blur the hygiene/infrastructure distinction

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Filter agent_abandoned from streak calculation | implementation | Changes internal logic, preserves API contract, aligns with existing decision (observation/intervention separation) |

### Recommended Approach ⭐

**Filter Infrastructure Interventions in Streak Calculation**

Modify `pkg/stability/stability.go` `ComputeReport` to only consider infrastructure interventions when computing streak:

```go
// Track last INFRASTRUCTURE intervention (not agent interventions)
if entry.Type == TypeIntervention && isInfrastructureIntervention(entry.Source) {
    if entry.Ts > lastInterventionTs {
        lastInterventionTs = entry.Ts
    }
}

// Helper function
func isInfrastructureIntervention(source string) bool {
    switch source {
    case SourceManualRecovery, SourceDoctorFix:
        return true
    case SourceAgentAbandoned:
        return false
    default:
        // Unknown sources default to infrastructure (fail-safe)
        return true
    }
}
```

**Why this approach:**
- Minimal code change (one filter function + condition change)
- Preserves all existing data (agent_abandoned events stay in log)
- Aligns with observation/intervention decision (agent abandonment is observation, not recovery)
- Backward compatible (old data files still work)
- Agent abandonment events still visible in intervention list, just don't reset streak

**Trade-offs accepted:**
- Existing streak values will jump when fix deploys (if last intervention was agent_abandoned)
- API consumers relying on "any intervention resets streak" behavior will see change
- Requires clear documentation of what counts as infrastructure vs agent intervention

### Alternative Approaches Considered

**Option B: Stop recording agent_abandoned as intervention**
- **Pros:** Cleaner data model, simpler streak calculation
- **Cons:** Loses audit trail of abandonments, harder to track abandonment rate trends
- **When to use instead:** If abandonment events aren't useful for any analysis

**Option C: Add intervention_type field (infrastructure vs agent)**
- **Pros:** More flexible, allows multiple intervention categories
- **Cons:** Requires data migration, more complex API, over-engineering for current need
- **When to use instead:** If we anticipate many more intervention categories

---

## Next Steps

1. ✅ Investigation complete
2. ⏭ Transition to Implementation Phase (TDD mode)
3. ⏭ Write failing tests for:
   - Streak calculation ignores agent_abandoned interventions
   - Streak calculation respects manual_recovery and doctor_fix interventions
   - isInfrastructureIntervention categorizes sources correctly
4. ⏭ Implement the filter logic
5. ⏭ Validate with existing stability.jsonl data
6. ⏭ Update documentation/comments

---

## References

**Files Examined:**
- `cmd/orch/serve_operator_health.go` - Crash-free streak metric API
- `cmd/orch/abandon_cmd.go` - Where agent_abandoned interventions are recorded
- `pkg/stability/stability.go` - Streak calculation logic
- `.kb/decisions/2026-01-14-separate-observation-from-intervention.md` - Observation/intervention distinction
- `.kb/investigations/2026-02-07-design-automatic-stability-measurement.md` - Phase 3 design

**Related Artifacts:**
- **Beads Issue:** orch-go-21474
- **Decision:** Observation vs Intervention separation principle applies here
