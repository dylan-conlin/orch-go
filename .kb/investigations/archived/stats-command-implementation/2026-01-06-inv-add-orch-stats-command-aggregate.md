<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Created `orch stats` command that aggregates events.jsonl to surface orchestration metrics including completion rates, skill effectiveness, and daemon health.

**Evidence:** Command tested with real events.jsonl (1092 events in 7 days), shows 66.3% completion rate, 36.1% daemon spawn rate, skill breakdown with per-skill completion rates.

**Knowledge:** events.jsonl contains rich telemetry (27 event types) but was previously not aggregated for decision support. Now surfaced via `orch stats`.

**Next:** Complete - command is implemented with tests, ready for daily use.

---

# Investigation: Add Orch Stats Command Aggregate

**Question:** How should we implement an `orch stats` command to aggregate events.jsonl metrics for orchestration observability?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** feature-impl spawn
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Rich Event Data Structure

**Evidence:** events.jsonl contains 27 event types with structured data:
- `session.spawned` - skill, beads_id, model, spawn_mode, gap analysis data
- `agent.completed` - beads_id, reason, forced flag
- `agent.abandoned` - beads_id, reason
- `daemon.spawn` - beads_id, skill, count
- Wait operations, session lifecycle, account switches

**Source:** `~/.orch/events.jsonl`, `pkg/events/logger.go:14-25`

**Significance:** The event structure already supports the analysis needed. Implementation is mainly aggregation logic.

---

### Finding 2: Existing Commands Provide Pattern

**Evidence:** `orch history` and `orch hotspot` follow consistent patterns:
- Flag structure: `--days`, `--json`, `--verbose`
- Output structure: Header, metrics sections, breakdown tables
- Time filtering via cutoff calculation

**Source:** `cmd/orch/history.go`, `cmd/orch/hotspot.go`

**Significance:** Following existing patterns ensures consistency and reduces learning curve.

---

### Finding 3: Session Duration Calculation Complexity

**Evidence:** Duration calculation requires correlating spawn and completion events by beads_id since session_id may differ between session.spawned and agent.completed events.

**Source:** Testing with real events showed session_id doesn't always match.

**Significance:** Implemented correlation via beads_id mapping. Sanity check filter (< 8 hours) prevents outliers from skewing averages.

---

## Synthesis

**Key Insights:**

1. **Data-Rich, Insight-Poor Gap Filled** - events.jsonl captured extensive telemetry but it wasn't surfaced for decision support. Now orchestrators can see completion rates, skill effectiveness, and daemon health at a glance.

2. **Consistent CLI Pattern** - Following existing `orch history` and `orch hotspot` patterns (--days, --json, --verbose flags) makes the command immediately familiar.

3. **Daemon Visibility** - 36.1% of spawns come from daemon, which wasn't visible before. This enables monitoring daemon effectiveness.

**Answer to Investigation Question:**

Implementation complete. Created `cmd/orch/stats_cmd.go` with:
- Core metrics: spawn count, completion rate (66.3%), abandonment rate (8.4%)
- Time windowing via `--days` flag (default 7)
- Skill breakdown with per-skill completion rates
- Daemon health metrics (spawn rate, auto-completions, triage bypassed)
- Wait operation stats (timeout rate)
- Orchestrator session metrics
- JSON output for scripting
- Health assessment warning at <80% completion rate

---

## Structured Uncertainty

**What's tested:**

- ✅ Event parsing works with real events.jsonl (verified: `orch stats` with 1092 events)
- ✅ Time filtering works (verified: `--days 1` shows 233 events vs 1092 for 7 days)
- ✅ Skill breakdown is accurate (verified: matches manual jq grouping)
- ✅ JSON output is valid (verified: `orch stats --json | jq .` parses successfully)
- ✅ Unit tests pass (verified: 6 tests covering parsing, aggregation, edge cases)

**What's untested:**

- ⚠️ Performance with very large events.jsonl (> 100k events)
- ⚠️ Duration calculation accuracy across all edge cases

**What would change this:**

- If events.jsonl is rotated/pruned, historical analysis would be limited
- If spawn timestamps don't correlate with session starts, duration metrics could be wrong

---

## Implementation Recommendations

N/A - Implementation is complete. See cmd/orch/stats_cmd.go and stats_test.go.

---

## References

**Files Examined:**
- `pkg/events/logger.go` - Event type definitions and logging API
- `cmd/orch/history.go` - Pattern for --days flag and skill analysis
- `cmd/orch/hotspot.go` - Pattern for JSON output and formatted display
- `~/.orch/events.jsonl` - Real event data for testing

**Commands Run:**
```bash
# Count events by type
cat ~/.orch/events.jsonl | jq -s 'group_by(.type) | map({type: .[0].type, count: length})'

# Test the command
orch stats
orch stats --days 1
orch stats --json

# Run tests
go test ./cmd/orch/... -run TestParseEvents -v
go test ./cmd/orch/... -run TestAggregate -v
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-06-inv-orch-stats-command-if-so.md` - Prior exploration that recommended this implementation

---

## Investigation History

**2026-01-06 15:45:** Investigation started
- Initial question: How to implement orch stats command?
- Context: Prior investigation recommended creating this command to aggregate events.jsonl

**2026-01-06 15:55:** Implementation complete
- Created stats_cmd.go with full functionality
- Created stats_test.go with 6 unit tests
- All tests passing, command working with real data

**2026-01-06 16:00:** Investigation completed
- Status: Complete
- Key outcome: `orch stats` command implemented and tested
