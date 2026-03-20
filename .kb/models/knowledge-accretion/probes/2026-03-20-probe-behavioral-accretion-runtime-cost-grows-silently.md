# Probe: Behavioral Accretion — Runtime Cost Grows Silently with Scale

**Model:** knowledge-accretion
**Date:** 2026-03-20
**Status:** Complete
**claim:** extends (new substrate: runtime behavior)
**verdict:** extends

---

## Question

Does the knowledge-accretion model's core claim — "shared artifacts degrade from correct contributions when agents lack coordination" — apply to a previously unexamined substrate: runtime behavior? Specifically, do O(n) operations accrete silently when the data they operate on grows, even though the code itself doesn't change?

---

## What I Tested

Measured actual runtime costs and growth trajectories for functions that iterate over unbounded collections in orch-go.

### Test 1: events.jsonl full-file parse

```bash
$ wc -l ~/.orch/events.jsonl
153163 /Users/dylanconlin/.orch/events.jsonl

$ ls -lh ~/.orch/events.jsonl
53 MB /Users/dylanconlin/.orch/events.jsonl

$ time orch stats --days 1
# real 0.56s (parses all 153K lines to get last 24h of events)

# Date range: 2026-03-08 to 2026-03-20 (12 days)
# Growth rate: ~12,871 events/day
```

Code: `parseEvents()` at `cmd/orch/stats_cmd.go:89` reads ALL events into `[]StatsEvent` before `aggregateStats()` filters by time window. Same pattern in `events.ComputeLearning()` at `pkg/events/learning.go:47`.

**Consumers per OODA cycle:** `RunPeriodicLearningRefresh()`, `SkillPerformanceDriftDetector.Detect()`, `parseUtilizationEvents()`. Each re-parses the full file.

### Test 2: Cross-project workspace scan

```bash
# Counted workspaces across all 26 registered projects:
$ find ~/Documents/personal -maxdepth 3 -name ".orch" -type d | while read d; do
    ls -d "$d/workspace"/* 2>/dev/null | wc -l
  done | paste -sd+ | bc
# Result: ~895 active workspaces across 26 projects
```

Functions that scan all workspaces:
- `getCompletionsForReview()` at `cmd/orch/review.go:109` — the triggering case. Reads manifest + SPAWN_CONTEXT.md + SYNTHESIS.md per workspace across all KB projects.
- `LookupManifestsAcrossProjects()` at `pkg/discovery/discovery.go:209` — called on every `orch status`. Reads AGENT_MANIFEST.json per workspace per project.
- `FindByBeadsID()` at `pkg/workspace/workspace.go:21` — linear scan per project, reads manifest + SPAWN_CONTEXT.md on mismatch.

### Test 3: KB tree walks

```bash
$ find .kb -type f -name "*.md" | wc -l
# Result: 1,700+ KB files
```

Functions that walk entire KB:
- `orphan_classify.go:103` — `filepath.Walk(.kb/)` to classify all orphans
- `orphans.go:57` — `filepath.Walk(.kb/)` to find unconnected investigations
- `autolink.go:275-323` — reads all models, threads, and decisions directories
- `decision_audit.go:73` + `456` + `501` + `547` — multiple walks per audit

### Test 4: Decision that was locally reasonable (file-based detection)

Read `2026-01-17-file-based-workspace-state-detection.md`: switching from beads API to file-based detection reduced scan time from >2min to <1s for 295 workspaces. But this decision didn't address the underlying scaling issue — it made O(n) faster per iteration, not bounded.

---

## What I Observed

### 5 Distinct Instances of Behavioral Accretion

**Instance 1: events.jsonl unbounded parse**
- Current: 153K lines, 53MB, 0.56s per parse
- Growth: ~12,871 events/day (daemon alone emits 30+ event types per cycle)
- Projected 90-day: ~1.2M lines, ~405MB, ~4.2s per parse
- Projected 365-day: ~4.7M lines, ~1.6GB, ~17.2s per parse
- Multiple consumers re-parse the same file per OODA cycle
- **No rotation, truncation, or time-bounded reads**

**Instance 2: Cross-project workspace scan (getCompletionsForReview)**
- Current: ~895 workspaces across 26 projects, 3 file reads per workspace (manifest, spawn context, synthesis check)
- Growth: ~3-5 new workspaces/day (spawn rate minus archive rate)
- Called by: `orch review triage`, embedded in status dashboard
- No pagination, no caching, no index

**Instance 3: Cross-project manifest lookup (orch status)**
- `LookupManifestsAcrossProjects()` iterates all workspace directories across all registered projects
- Called on every `orch status` invocation
- N = projects × workspaces_per_project (26 × ~35 = ~900)
- Each workspace reads and parses AGENT_MANIFEST.json

**Instance 4: KB filepath.Walk operations**
- `orphan_classify`, `orphans`, `autolink`, `decision_audit` all walk the full .kb/ tree
- 1,700+ files currently; grows with every investigation/probe/thread/decision
- `decision_audit.go` performs 4 separate `filepath.Walk` calls per audit invocation
- daemon periodic triggers call these

**Instance 5: Daemon compound effect**
- 23+ periodic tasks per OODA cycle
- Multiple tasks independently scan: workspaces, events.jsonl, KB tree, beads issues
- Cycle frequency: every 2 minutes (configurable)
- Compound cost = Σ(O(n_i)) where each n_i grows independently
- No shared scanning or caching between periodic tasks in the same cycle

### The Pattern

All 5 instances share the same structure:
1. **Decision was locally reasonable** — "scan all workspaces" is fine at N=10
2. **N grew silently** — workspaces, events, KB files accumulate without bounds
3. **No degradation signal** — no metric, alert, or log watches these N values
4. **Cost is invisible until pain** — only discovered when command latency becomes noticeable
5. **The code didn't change** — the function is identical to when it was written; only the world it operates on grew

### Structural Similarity to Knowledge Accretion

The knowledge-accretion model describes: "individually correct contributions compose into structural degradation when shared infrastructure is missing."

Behavioral accretion is the same mechanism applied to runtime:
- **Correct contribution:** Each workspace/event/KB file is individually correct
- **Composition degrades:** The aggregate cost of scanning all of them degrades performance
- **Missing infrastructure:** No metric tracks the growth of N; no mechanism bounds N; no alert fires when O(n) crosses a latency threshold

### Historical Instance (from KB)

**File-based workspace detection** (decision 2026-01-17): The system hit >2min latency for 295 workspaces when using beads API calls. The fix (file-based detection) reduced per-item cost but didn't address unbounded N. The same workspaces that caused the original bottleneck continue to grow. This is behavioral accretion that was patched (reduce constant factor) but not resolved (bound N or index it).

**Two-lane agent discovery** (decision 2026-02-18): "5 local state layers each built to solve slow queries, each drifted from reality." Each cache was a response to behavioral accretion in the query layer — scanning got slow, so someone added a cache, which added its own accretion dynamics.

---

## Model Impact

- [x] **Extends** model with: **Runtime behavioral accretion as a third substrate** alongside code accretion and knowledge accretion

### Extension Details

The knowledge-accretion model currently covers two substrates:
1. **Code** — files grow from correct commits (daemon.go +892 lines)
2. **Knowledge** — orphan investigations accumulate (85.5% orphan rate)

This investigation adds a third:
3. **Runtime behavior** — O(n) operations degrade silently as N grows from correct usage

The mechanism is identical: individually correct contributions (spawning agents, emitting events, creating KB files) compose into degradation (slow commands, compound daemon cost) when no coordination mechanism bounds or observes the growth.

### Testable Claims for New Model

| ID | Claim | Testable? | Evidence |
|----|-------|-----------|----------|
| BA-01 | events.jsonl parse time grows linearly with file size | Yes | 0.56s at 153K lines; measure at 300K and 500K |
| BA-02 | orch status latency correlates with total workspace count across projects | Yes | Time orch status before and after archiving stale workspaces |
| BA-03 | daemon OODA cycle wall-clock time grows with workspace/event/KB counts | Yes | Log cycle times, correlate with Σ(N_i) |
| BA-04 | No metric currently tracks any of the N values (workspace count, event count, KB file count) | Yes | Grep for monitoring/alerting on these values |
| BA-05 | Interventions that reduce per-item cost (like file-based detection) without bounding N will require re-intervention as N continues to grow | Yes | Track whether file-based workspace scan becomes slow again at N=2000 |

---

## Notes

### Sufficient for Model Extension?

Yes — 5 distinct instances with the same structural pattern, testable claims, and a clear connection to the existing knowledge-accretion model. The extension is: accretion dynamics are not substrate-specific (the model already claims this); runtime behavior is a third substrate (newly demonstrated).

### Potential Interventions

1. **events.jsonl rotation** — Rotate or truncate to last 30 days; archive older events
2. **Workspace count as tracked metric** — Add to `orch stats` output
3. **Time-bounded file reads** — `parseEvents()` should seek to approximate timestamp instead of reading from beginning
4. **Shared scanning in daemon** — One workspace scan per OODA cycle, results shared across periodic tasks
5. **Decision-time cost annotations** — When adding a scan-all-workspaces function, annotate expected N and growth rate
