<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented escape hatch spawn tracking in orch stats with multi-window metrics (total, 7d, 30d) and per-account breakdown.

**Evidence:** Tests pass (TestAggregateStatsEscapeHatch), manual verification with real events.jsonl shows correct counts and rates.

**Knowledge:** spawn_mode="claude" events are tracked but often lack usage_account since claude backend doesn't use OpenCode API capacity tracking.

**Next:** Close - implementation complete and tested.

**Promote to Decision:** recommend-no (tactical feature addition, not architectural)

---

# Investigation: Track Escape Hatch Stats Impl

**Question:** How to track escape hatch spawn usage (--backend claude) and surface metrics in orch stats?

**Started:** 2026-01-14
**Updated:** 2026-01-14
**Owner:** og-feat-track-escape-hatch-14jan-f5c6
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: spawn_mode tracking already in place

**Evidence:** All spawn modes emit session.spawned events with spawn_mode field: "inline", "headless", "tmux", "claude"

**Source:** cmd/orch/spawn_cmd.go lines 1433, 1523, 1719, 1797

**Significance:** No event emission changes needed - just aggregation and display code.

---

### Finding 2: Multi-window aggregation requires different parseEvents signature

**Evidence:** Original parseEvents filtered by --days window at parse time, but escape hatch needs total, 7d, 30d windows.

**Source:** cmd/orch/stats_cmd.go (modified parseEvents to return all events)

**Significance:** Time filtering moved to aggregateStats to support both --days main stats and fixed-window escape hatch stats.

---

### Finding 3: Account info absent for claude backend spawns

**Evidence:** 65 escape hatch spawns in events.jsonl show no usage_account - all display as "unknown" in breakdown.

**Source:** grep '"spawn_mode":"claude"' ~/.orch/events.jsonl | grep usage_account returns no matches

**Significance:** Claude backend spawns don't use OpenCode API capacity tracking, so no account info is captured. Would require code change to add.

---

## Structured Uncertainty

**What's tested:**

- ✅ Total escape hatch counts (verified: TestAggregateStatsEscapeHatch test)
- ✅ 7d/30d window calculations (verified: test with events at different timestamps)
- ✅ Account breakdown aggregation (verified: test with multiple accounts)
- ✅ Escape hatch rate calculation (verified: rate changes with --days window)
- ✅ Text and JSON output formats (verified: manual testing with orch stats)

**What's untested:**

- ⚠️ Performance with very large events.jsonl (not benchmarked, but now parsing all events)

**What would change this:**

- Finding would need revision if claude backend starts emitting usage_account

---

## References

**Files Examined:**
- cmd/orch/stats_cmd.go - Main implementation file
- cmd/orch/stats_test.go - Test file
- cmd/orch/spawn_cmd.go - Event emission for spawn_mode
- ~/.orch/events.jsonl - Real event data

**Commands Run:**
```bash
# Count escape hatch spawns
grep '"spawn_mode":"claude"' ~/.orch/events.jsonl | wc -l
# Output: 65

# Test escape hatch stats
./build/orch stats
./build/orch stats --verbose
./build/orch stats --json | jq '.escape_hatch_stats'
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2026-01-14-inv-track-escape-hatch-spawn-usage.md - Prior investigation with findings
- **Workspace:** .orch/workspace/og-feat-track-escape-hatch-14jan-f5c6/SYNTHESIS.md - Implementation synthesis
