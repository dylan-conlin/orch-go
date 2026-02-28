# Session Synthesis

**Agent:** og-debug-investigate-architect-skill-28feb-4101
**Issue:** orch-go-e3oi
**Outcome:** success

---

## Plain-Language Summary

The architect skill's reported 20% abandonment rate (12/60 spawns) was **not a skill quality problem** — it was a metrics bug. Two issues in the stats pipeline inflated abandonment counts by ~3x:

1. **Duplicate event emission**: `orch abandon` emits TWO `agent.abandoned` events per abandonment — one from the LifecycleManager and one from a separate telemetry function. Every other event type has a single emission point.
2. **No deduplication**: `orch stats` deduplicates completion events by beads_id but did NOT deduplicate abandonment events. So each duplicate event was counted separately.

After fixing both issues, the architect skill's real abandonment rate is **6.3% (4/66 spawns)** — comparable to feature-impl (5.9%) and systematic-debugging (4.8%). Of those 4 unique abandonments, only 1 was architect-specific (agent stuck in Exploration phase); the other 3 were system-level issues (infrastructure death, skill misrouting).

## Verification Contract

See `VERIFICATION_SPEC.yaml` for test commands and expected outcomes.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/stats_cmd.go` — Added `abandonedBeadsIDs` deduplication map and dedup logic for `agent.abandoned` events (mirrors existing completion dedup pattern)
- `cmd/orch/stats_test.go` — Added `TestAggregateStatsAbandonmentDeduplication` and `TestAggregateStatsAbandonmentRetries` regression tests
- `pkg/events/logger.go` — Changed `LogAgentAbandoned` to emit `agent.abandoned.telemetry` instead of `agent.abandoned`, preventing duplicate counting. Added `EventTypeAgentAbandonedTelemetry` constant.

---

## Evidence (What Was Observed)

- **95% of abandon events were duplicated** — 35 of 37 unique events in last 7d appeared twice (identical timestamp + beads_id + reason)
- **Architect inflation was 3.5x** — 14 raw events for 4 unique beads IDs (worst among skills due to nn43 being retried 3x)
- **Total abandonment inflation was 2.7x** — 72 raw events represented only 27 unique abandonments
- The duplicate emission traced to `abandon_cmd.go`: `lm.Abandon()` emits event #1, then `logAbandonmentTelemetry()` emits event #2
- `orch stats` had deduplication for completions (line 617-627: `completedBeadsIDs`) but NOT for abandonments

### Architect Abandonment Root Causes (4 unique)
| Beads ID | Date | Cause | Architect-specific? |
|----------|------|-------|---------------------|
| orch-go-1228 | Feb 25 | Agent died mid-planning | No (infra death) |
| orch-go-1235 | Feb 25 | Wrong approach, re-routed | No (misrouting) |
| orch-go-iwb3 | Feb 27 | Phantom daemon spawn as wrong skill | No (daemon routing) |
| orch-go-nn43 | Feb 28 | Stuck at Exploration phase (3 retries) | **Yes** |

### Tests Run
```bash
go test ./cmd/orch/ -run "TestAggregateStatsAbandonment" -v
# PASS: TestAggregateStatsAbandonmentDeduplication (0.00s)
# PASS: TestAggregateStatsAbandonmentRetries (0.00s)

go test ./cmd/orch/ -v
# PASS: all 66 tests passing (3.2s)

go test ./pkg/events/ -v
# PASS: all 17 tests passing (0.007s)
```

### Smoke Test
```bash
# Before fix:
# architect: 63 spawns, 40 complete, 12 abandon (19%)
# Total abandonments: 72 (15.1%)

# After fix (orch stats):
# architect: 63 spawns, 42 complete, 4 abandon (6.3%)
# Total abandonments: 27 (5.6%)
```

---

## Architectural Choices

### Dedup in stats vs single emission fix
- **What I chose:** Both — deduplicate in stats AND change telemetry event type
- **What I rejected:** Only fixing emission (wouldn't help existing events in events.jsonl)
- **Why:** Defense in depth. Stats dedup handles historical data. Event type change prevents future duplicates. Either fix alone is sufficient for new data.
- **Risk accepted:** Historical telemetry events remain with `agent.abandoned` type in events.jsonl, but stats dedup handles them correctly

### Telemetry event type change vs removing telemetry
- **What I chose:** Changed `LogAgentAbandoned` to emit `agent.abandoned.telemetry` type
- **What I rejected:** Removing the telemetry function entirely
- **Why:** Telemetry data (skill, tokens, duration) is valuable for model performance tracking. Just needs a distinct event type so it's not double-counted.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- `orch stats` completion dedup existed but abandonment dedup was missing — parity gap
- `orch abandon` has two-phase event emission by design (lifecycle + telemetry) but both used the same event type

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete
- [x] Tests passing (2 new + all existing)
- [x] Smoke test confirms fix (19% → 6.3%)
- [x] Ready for `orch complete orch-go-e3oi`

---

## Unexplored Questions

- **nn43 (agent stuck in Exploration)**: The one architect-specific abandonment — worth investigating why architect agents get stuck in Exploration phase. May relate to skill guidance about when to transition from exploration to design.
- **Daemon misrouting**: orch-go-iwb3 was spawned as feature-impl but was actually an architect task. The daemon's skill inference may need improvement for design tasks.

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-debug-investigate-architect-skill-28feb-4101/`
**Beads:** `bd show orch-go-e3oi`
