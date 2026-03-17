# Hotspot Acceleration: cmd/orch/stats_cmd.go

**TLDR:** The +806 lines/30d churn was real but already addressed — extraction on Mar 12 split the monolith into 4 files. The biggest resulting file (stats_aggregation.go, 959 lines) is under threshold with clear internal boundaries. No immediate action needed.

**Status:** Complete

## D.E.K.N. Summary

- **Delta:** stats_cmd.go growth was feature accumulation (gate effectiveness, baselines, gate decisions, overrides, coaching) from Jan→Mar. The Mar 12 extraction (commit f910700e5) already decomposed it into 4 well-sized files. Current risk is stats_aggregation.go at 959 lines.
- **Evidence:** `wc -l cmd/orch/stats_*.go` → 302 + 304 + 959 + 465 = 2030 production lines. Git shows extraction created stats_types.go, stats_aggregation.go, stats_output.go on Mar 12.
- **Knowledge:** The 959-line aggregation file has 27 methods with natural split points (19 event processors vs 8 calculators) if it crosses the 1200-line threshold.
- **Next:** No action needed now. Monitor stats_aggregation.go — if it crosses 1200 lines, extract event processors to stats_event_processors.go.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| N/A — novel investigation | - | - | - |

## Question

Should cmd/orch/stats_cmd.go be extracted to prevent it from becoming a critical hotspot (>1500 lines)?

## Finding 1: Extraction Already Complete

The hotspot alert flagged +806 lines/30d net churn on stats_cmd.go. However, this file was already extracted on Mar 12, 2026.

**Current file family:**

| File | Lines | Content |
|---|---|---|
| stats_cmd.go | 302 | CLI setup, event parsing, baseline snapshot |
| stats_types.go | 304 | 20+ type definitions |
| stats_aggregation.go | 959 | statsAggregator struct + 27 methods |
| stats_output.go | 465 | Text and JSON output formatting |
| stats_test.go | 1891 | Tests |
| **Total** | **3921** | |

The extraction commit: `f910700e5` ("feat: decompose aggregateStats() into statsAggregator struct with per-domain methods")

## Finding 2: Growth Vector Analysis

**stats_aggregation.go (959 lines)** is the primary growth vector:
- 19 event processor methods (lines 116-540, ~425 lines)
- 8 post-aggregation calculator methods (lines 542-959, ~418 lines)
- Each new event type adds ~15-30 lines
- Each new stat section adds ~40-100 lines (calculator + types + output)

At current growth rate (~27 lines/feature), the file would cross 1500 lines after adding ~20 more event types — unlikely in the near term.

**Natural split boundary:** Event processors (process*) vs calculators (calc*) — roughly 50/50 split.

## Finding 3: Minor Duplication

`parseEvents()` in stats_cmd.go and `parseEventsForSnapshot()` in daemon_snapshot.go both parse events.jsonl but into different types (StatsEvent vs events.Event). This is intentional — the stats system uses a simplified event struct for aggregation while daemon_snapshot needs the full event type.

## Test Performed

Verified file sizes, extraction history, and internal structure via wc -l, git log, and function listing.

## Conclusion

**No extraction needed.** The hotspot was already addressed on Mar 12. Current state:
- All production files under 1000 lines (largest: stats_aggregation.go at 959)
- Well-organized with clear internal boundaries
- Future extraction path identified if stats_aggregation.go grows past 1200 lines
