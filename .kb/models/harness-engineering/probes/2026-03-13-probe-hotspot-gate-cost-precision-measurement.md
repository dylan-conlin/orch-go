# Probe: Hotspot Gate Cost and Precision Measurement

**Model:** harness-engineering
**Date:** 2026-03-13
**Status:** Complete

---

## Question

The harness-engineering model claims "enforcement without measurement is theological — you believe the gate works but can't prove it." The hotspot spawn gate (Layer 1) fires on feature-impl and systematic-debugging skills, blocking when CRITICAL files (>1500 lines) are targeted. What are the actual cost and precision properties of this gate?

Specific claims under test:
1. Model invariant #7: "Every enforcement surface must be paired with measurement (cost, coverage, precision, relevance)"
2. The three-layer hotspot enforcement (spawn gates, daemon escalation, spawn context advisory) prevents agent work on bloated files
3. Prior decision: "Enforcement audit framework splits into harness (mechanical: cost, coverage) and skill (judgment: precision, relevance)"

---

## What I Tested

### Cost Measurement
```bash
# Pipeline timing from agent.completed events (144 completions with hotspot timing)
cat ~/.orch/events.jsonl | grep 'pipeline_timing' | python3 -c "
import sys, json
durations = []
for line in sys.stdin:
    e = json.loads(line.strip())
    for step in e.get('data',{}).get('pipeline_timing',[]):
        if step.get('name') == 'hotspot':
            durations.append(step['duration_ms'])
durations.sort()
print(f'Count: {len(durations)}, Min: {min(durations)}ms, Max: {max(durations)}ms')
print(f'Mean: {sum(durations)/len(durations):.0f}ms, Median: {durations[len(durations)//2]}ms')
print(f'P95: {durations[int(len(durations)*0.95)]}ms, P99: {durations[int(len(durations)*0.99)]}ms')
print(f'Total time: {sum(durations)/1000:.1f}s')
"
```

### Precision Measurement
```bash
# Gate decision distribution (205 events)
cat ~/.orch/events.jsonl | grep 'spawn.gate_decision' | grep '"hotspot"' | \
  python3 -c "..."  # → allow: 201, unknown: 4

# Block events
cat ~/.orch/events.jsonl | grep 'spawn.gate_decision' | grep '"hotspot"' | grep '"block"' | wc -l
# → 0

# Hotspot bypass events (132 total — ALL from test leaks)
cat ~/.orch/events.jsonl | grep 'spawn.hotspot_bypassed' | python3 -c "..."
# → 102 empty reason (tests), 30 with reason (all "fix cmd/orch/main.go" from test suite)

# Current CRITICAL file count
find . -name '*.go' -not -path '*/node_modules/*' | xargs wc -l | sort -rn | head -5
# → largest non-test Go file: 1040 lines (pkg/opencode/client.go)

# Daemon escalation history
cat ~/.orch/events.jsonl | grep 'daemon.architect_escalation' | python3 -c "..."
# → 30 evaluations, 0 actual escalations

# Task description path inclusion rate
cat ~/.orch/events.jsonl | grep 'session.spawned' | python3 -c "..."
# → 137/310 feature-impl/debugging tasks include file paths (44%)
# → 173/310 do NOT include file paths (56%)

# Harness audit summary
go run ./cmd/orch harness audit --days 30
```

---

## What I Observed

### Cost Properties

| Metric | Value |
|--------|-------|
| Mean cost per invocation | 233ms |
| Median cost | 319ms |
| P95 cost | 358ms |
| P99 cost | 363ms |
| Total accumulated cost | 33.6s across 144 completions |
| Invocations | 201 in 30 days |
| Coverage | 49.5% (201/406 spawns) |

**Cost composition:** The gate runs 4 sub-analyses every invocation:
1. `analyzeFixCommits` — git log scan
2. `analyzeInvestigationClusters` — .kb directory scan
3. `analyzeCouplingClusters` — co-change analysis
4. `analyzeBloatFiles` — file size scan

Only #4 (bloat) matters for the blocking decision. The other three contribute to the advisory warning but never influence block/allow.

### Precision Properties

| Metric | Value |
|--------|-------|
| Block rate (spawn gate, Layer 1) | 0/201 = **0.0%** |
| Escalation rate (daemon, Layer 2) | 0/30 = **0.0%** |
| True positive rate | **Unmeasurable** (0 blocks, 0 positives) |
| False positive rate | 0% (but vacuously — 0 blocks means 0 false positives) |
| Current CRITICAL files in codebase | **0** |
| Largest non-test Go file | 1040 lines (pkg/opencode/client.go) |
| Tasks with file paths in description | 44% (137/310) |
| Tasks WITHOUT file paths | 56% — **would be missed even with CRITICAL files** |

### The Test Contamination Problem

132 `spawn.hotspot_bypassed` events exist in events.jsonl, ALL from test suites:
- 102 with empty/none reason (from hotspot_test.go in gates package)
- 30 with reasons like "architect reviewed extraction plan" (all task="fix cmd/orch/main.go")
- All cluster at identical timestamps in groups of 3, matching test run patterns

This was identified in orch-go-9o5h2 but only the empty-reason issue was documented. The 30 "real-looking" events are also test contamination.

### Harness Audit Confirmation

```
  hotspot    201  0  0  201  0.0%  49.5%  233ms
  ⚠ hotspot  [zero_fires] 0 fires in 30d — gate may be inert
  ⚠ hotspot  [low_coverage] 50% coverage < 50% threshold
```

The harness audit correctly flags the hotspot gate as anomalous on both zero_fires and low_coverage.

### Why The Gate Cannot Fire

1. **No CRITICAL files exist**: The largest non-test Go file is 1040 lines. The blocking threshold is 1500 lines. The gate evaluates, finds no CRITICAL hotspots, and allows every spawn.
2. **Even if CRITICAL files existed**: 56% of feature-impl/debugging task descriptions don't include file paths. The gate's `extractPathsFromTask` regex can only match explicit file path patterns. A task like "fix the daemon polling logic" would not match `pkg/daemon/daemon.go` even if that file exceeded 1500 lines.
3. **All three layers are dormant**: Layer 1 (spawn gate): 0 blocks. Layer 2 (daemon escalation): 0 escalations. Layer 3 (context advisory): only fires when matched, also 0 due to no CRITICAL files.

---

## Model Impact

- [x] **Confirms** invariant #7: "Every enforcement surface must be paired with measurement" — The hotspot gate is the canonical instance of theological enforcement. It exists, evaluates every spawn at ~300ms cost, and has never blocked anything. Without measurement (now provided by `orch harness audit`), this was invisible.

- [x] **Confirms** the hard/soft distinction: The hotspot gate is categorized as a "hard gate" (binary block/allow), but its operational behavior is indistinguishable from a soft gate (always allows). The gate mechanism is correct — the environment has changed (no CRITICAL files remain after successful extractions).

- [x] **Extends** model with: **Dormancy as distinct gate failure mode.** The harness-engineering model describes failure modes for gates that fire incorrectly (false positive/negative). The hotspot gate reveals a third failure mode: **dormancy** — a gate that evaluates correctly but cannot fire because its triggering condition doesn't exist in the environment. Dormant gates still impose cost (300ms/spawn) but provide zero enforcement value. The three properties that can cause dormancy are:
  1. **Threshold exceeded by extraction success** — CRITICAL files were extracted below 1500 lines
  2. **Input format mismatch** — 56% of tasks lack the file paths the gate needs to match
  3. **Coverage gap** — only 49.5% of spawns are even evaluated

- [x] **Extends** model with: **Test contamination as measurement corruption.** 132 fake hotspot events in events.jsonl from test leaks. Any analysis consuming these events draws wrong conclusions about gate activity. This is a meta-measurement failure: the measurement infrastructure itself needs measurement.

---

## Notes

### Cost Efficiency Opportunity

The hotspot gate runs 4 sub-analyses but only bloat-size matters for blocking. If the gate were split:
- **Blocking path** (bloat only): likely ~50-100ms instead of ~300ms
- **Advisory path** (fix-density, clusters, coupling): could run async or on-demand
- Potential savings: ~200ms × 201 invocations = ~40s saved over 30 days

This is minor given total pipeline time, but aligns with the principle of not paying cost for zero enforcement value.

### Precision Design Gap

The gate's path-matching approach has a structural limitation: it can only match file paths that appear literally in the task text. In practice:
- Orchestrator task descriptions often use semantic descriptions ("fix daemon polling") not file paths
- The daemon (Layer 2) has the same limitation — it matches task text against hotspot paths
- A precision improvement would require semantic matching (task → affected files), which is a different complexity class

### What Would Make This Gate Fire Again

1. A file growing past 1500 lines (currently largest is 1040)
2. A task description that explicitly references that file path
3. The spawn being for a blocking skill (feature-impl or systematic-debugging)

All three conditions must be true simultaneously. Currently condition #1 is never met.
