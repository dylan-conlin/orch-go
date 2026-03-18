<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Design for injecting ground-truth metrics into daemon orient and the learning store. Currently all 14 metric families derive from events.jsonl (the system's own output). This design introduces 5 ground-truth signals from external sources (git, beads, filesystem) that break the self-referential loop.

**Evidence:** Code audit of orient.go, learning.go, allocation.go, stats_aggregation.go, autoadjust.go, and the two prior investigations (metric-camouflage, autonomous-confidence). Every metric traces back to events.jsonl — the system measures its own reports of what happened, not what actually happened.

**Knowledge:** The fix is not replacing self-referential metrics but *pairing* them with ground-truth counterparts so divergence becomes visible. When completions go up but merge rate stays flat, the system (or operator) can detect motion-without-value.

**Next:** Implementation in 3 phases: (1) git-based ground truth in orient, (2) outcome feedback in learning store, (3) divergence alerting.

**Authority:** architectural — New data sources, changes to orient display, learning store schema, and allocation scoring.

---

# Design: Ground-Truth Metric Injection for Daemon Orient

**Question:** How should we inject external ground-truth metrics to break the self-referential measurement loop?

**Started:** 2026-03-17
**Updated:** 2026-03-17
**Owner:** architect agent
**Phase:** Complete
**Next Step:** Implementation (3 phases)
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-03-17-inv-metric-camouflage-orch-go-question | motivates | Yes — 9 camouflaged metrics cataloged | None |
| 2026-03-17-inv-autonomous-confidence-orch-go-question | motivates | Yes — 5 ungrounded decision points identified | None |
| 2026-03-13-inv-task-compliance-coordination-boundary-map | informs | Yes — learning store feeds allocation scoring | None |

---

## Problem Statement

Every metric the daemon orient surfaces is computed from events.jsonl — a file the system itself writes. This creates a closed measurement loop:

```
System acts → System logs events → System computes metrics from events → System acts based on metrics
```

The critical failure mode: a system that spawns trivial work, auto-completes it, and logs success events is *indistinguishable from a productive system* by any metric currently surfaced. The metric camouflage investigation cataloged 9 specific instances of this.

**What we need:** External signals that the system cannot fabricate — outcomes observable in git, beads, and the filesystem that confirm (or contradict) the system's self-reported metrics.

---

## Design Principle: Pair, Don't Replace

Self-referential metrics aren't useless — they track operational throughput. The problem is that they're the *only* metrics. The design adds ground-truth counterparts displayed alongside internal metrics so that **divergence becomes the signal**.

| Internal Metric | Ground-Truth Counterpart | Divergence Meaning |
|----------------|--------------------------|-------------------|
| Completions (events.jsonl) | Merged commits (git log) | Completing work that never ships |
| Success rate (outcome field) | Rework rate (beads re-opens) | Declaring success on work that gets redone |
| Completion count by skill | Net line delta by skill (git) | Motion without code impact |
| Gate fire rate | Issue re-open rate (beads) | Gates not catching quality problems |
| Abandonment rate | Orphaned workspace rate (filesystem) | Abandonments not cleaning up |

---

## Ground-Truth Signals (5 Signals)

### Signal 1: Merge Rate (git)

**Source:** `git log --format='%H %s' --since=<window>` filtered for commits containing beads IDs.

**What it measures:** How many agent-produced commits actually land in the codebase. A commit with `(orch-go-XXXXX)` in the message that appears on the main branch is confirmed delivered work.

**Computation:**
```
merge_rate = commits_with_beads_ids / agent_completions_in_window
```

**Where it lives:** New field `MergeRate float64` on `orient.Throughput`.

**Display:** Orient throughput section:
```
Last 7d:
   Completions: 42 | Merged: 38 (90%) | Abandonments: 4
```

**Why this works:** The system cannot fabricate git commits on main. Even if an agent reports Phase: Complete and fires an `agent.completed` event, if no commit with the beads ID appears on main, the work didn't land.

**Edge cases:**
- Light-tier spawns (investigations) don't produce commits → exclude investigation skill from merge rate denominator
- Squash merges lose beads ID → match by workspace name in commit message as fallback
- Multi-commit agents → count unique beads IDs, not individual commits

---

### Signal 2: Rework Rate (beads)

**Source:** `bd list --label rework` or beads issue history showing re-opens.

**What it measures:** How often "completed" work needs to be redone. This is the ground truth for success rate — if the learning store says investigation has 95% success rate but 40% of investigations get reworked, the success rate is lying.

**Computation:**
```
rework_rate = issues_with_rework_spawns / total_completed_issues_in_window
```

**Where it lives:** New field on `events.SkillLearning`:
```go
ReworkCount int     `json:"rework_count"`
ReworkRate  float64 `json:"rework_rate"`
```

**Data source:** This requires scanning beads, not events.jsonl. Use `bd list --status=closed` and check for sibling issues with `rework:` prefix or `rework` label.

**Display:** Learning store output and orient:
```
investigation: 95% self-reported success, 12% rework rate
feature-impl: 88% self-reported success, 5% rework rate
```

**Why this works:** Rework is initiated by the orchestrator (a human or human-supervised process). The daemon cannot rework its own issues — the orchestrator explicitly decides "this needs to be redone." This is external judgment, not self-assessment.

---

### Signal 3: Net Code Impact (git diff)

**Source:** `git diff --numstat <baseline>..<current>` filtered to agent-touched files.

**What it measures:** Whether agent work produces actual code changes that persist. Complements completion count — 50 completions that produce 0 net lines delta means the system is churning.

**Computation:**
```
net_impact = sum(insertions - deletions) for commits with beads IDs in window
```

Per-skill breakdown available by matching beads ID to skill from events.jsonl.

**Where it lives:** New field `NetLinesImpact int` on `orient.Throughput`.

**Display:**
```
Last 7d:
   Completions: 42 | Merged: 38 (90%) | Net lines: +1,247
   By skill: feature-impl +892, investigation +311, systematic-debugging +44
```

**Why this works:** Git numstat is ground truth. A system producing completions with zero net impact is detectable.

**Note:** Investigations and designs legitimately produce .md files, not code. The per-skill breakdown prevents penalizing knowledge work.

---

### Signal 4: Trigger Outcome Rate (beads + events.jsonl cross-reference)

**Source:** Cross-reference `daemon:trigger` labeled issues in beads with their completion outcomes.

**What it measures:** Whether detector-created issues actually lead to useful work. This is the missing feedback loop from the autonomous-confidence investigation — detectors create issues but never learn whether those issues were valuable.

**Computation:**
```
per_detector:
  issues_created = count(beads issues with label detector:<name>)
  issues_completed = count(completed issues with label detector:<name>)
  issues_abandoned = count(abandoned issues with label detector:<name>)
  useful_rate = issues_completed / issues_created
```

**Where it lives:** New type in `pkg/daemon/`:
```go
type DetectorOutcome struct {
    Detector       string  `json:"detector"`
    IssuesCreated  int     `json:"issues_created"`
    Completed      int     `json:"completed"`
    Abandoned      int     `json:"abandoned"`
    UsefulRate     float64 `json:"useful_rate"`
}
```

**How it feeds back:** Detectors with `useful_rate < 0.3` get their budget halved. Detectors with `useful_rate < 0.1` get disabled with a log message. This closes the outcome feedback loop that the autonomous-confidence investigation identified as missing.

**Display:** Daemon health section (only when non-green):
```
Daemon health:
   [!] hotspot-acceleration: 12% useful rate (47 created, 6 completed, 41 abandoned)
```

---

### Signal 5: Divergence Alert (computed)

**Source:** Comparison of internal vs ground-truth metrics.

**What it measures:** Whether the system's self-assessment matches external reality. This is the meta-signal — it doesn't measure work quality directly, but detects when the system's *belief about its quality* has diverged from observable outcomes.

**Computation:**
```
completion_divergence = |self_reported_success_rate - (1 - rework_rate)|
merge_divergence = |completion_rate - merge_rate|
```

**Threshold:** Alert when divergence exceeds 20% sustained over 7 days.

**Where it lives:** New `DivergenceAlerts []DivergenceAlert` on `OrientationData`.

**Display:**
```
Metric divergence:
   [!] feature-impl: 92% self-reported success but 71% merge rate (21% gap)
   [!] investigation: 95% self-reported success but 40% rework rate
```

**Why this works:** This is the core value of the design. Individual ground-truth metrics tell you what's actually happening; divergence alerts tell you that the system's self-model is wrong. The operator doesn't need to manually compare — the system surfaces the gap.

---

## Implementation Plan

### Phase 1: Git-Based Ground Truth in Orient (Smallest, Highest Value)

**Files changed:**
- `pkg/orient/orient.go` — Add `MergeRate`, `NetLinesImpact` to `Throughput`
- `pkg/orient/git_ground_truth.go` — New file: `ComputeMergeRate()`, `ComputeNetImpact()`
- `cmd/orch/orient_cmd.go` — Wire new collectors into orient pipeline

**Data flow:** `git log --format='%H %s' --since=7d` → extract beads IDs from messages → count unique IDs → compare against completion count from events.jsonl.

**Why first:** Zero new dependencies (git is already available). Changes only the orient display, not daemon decision-making. Immediately visible to operator. Can be implemented and validated in a single session.

**Test strategy:**
- Unit test `ComputeMergeRate()` with synthetic git log output
- Unit test beads ID extraction from commit messages
- Integration test with real git repo

### Phase 2: Outcome Feedback in Learning Store

**Files changed:**
- `pkg/events/learning.go` — Add `ReworkCount`, `ReworkRate` to `SkillLearning`
- `pkg/daemon/detector_outcomes.go` — New file: `ComputeDetectorOutcomes()`
- `pkg/daemon/trigger.go` — Add outcome-based budget adjustment
- `pkg/daemon/allocation.go` — Use ground-truth-adjusted success rate

**Data flow:** `bd list --label rework` → match to original beads IDs → compute per-skill rework rate → blend with self-reported success rate for allocation scoring.

**Key design decision:** The allocation score formula changes from:
```
score = basePriority * (1 - weight + weight * blendedSuccessRate * 2)
```
to:
```
groundTruthRate = 1 - reworkRate  // external signal
adjustedRate = 0.7 * blendedSuccessRate + 0.3 * groundTruthRate  // blend
score = basePriority * (1 - weight + weight * adjustedRate * 2)
```

The 70/30 blend ensures ground truth influences but doesn't dominate (rework data is sparser than completion data). The weight shifts toward ground truth as sample size grows.

**Why second:** Requires beads queries (slightly more complex). Changes daemon decision-making (higher risk). Benefits from Phase 1 being deployed first so operators can see the divergence before the daemon starts acting on it.

### Phase 3: Divergence Alerting

**Files changed:**
- `pkg/orient/divergence.go` — New file: `ComputeDivergence()`
- `pkg/orient/orient.go` — Add `DivergenceAlerts` to `OrientationData`
- `cmd/orch/orient_cmd.go` — Wire divergence into orient output

**Data flow:** Phase 1 and Phase 2 metrics → compare internal vs external → alert on sustained gaps.

**Why third:** Depends on Phase 1 and Phase 2 data being available. Purely additive — no changes to daemon behavior, just operator visibility.

---

## Architectural Constraints Respected

1. **No Local Agent State:** All ground-truth computations are functional (git log, bd list) — no caches or projections.
2. **Fail-open:** Missing git data or beads errors return empty results, not errors. Orient continues with internal-only metrics.
3. **No new dependencies:** Git and bd CLI are already used extensively.
4. **Backward compatible:** New fields are additive. Orient output includes them only when data is available.

## What This Does NOT Solve

- **Metric gaming by agents:** An agent could commit trivial code to inflate merge rate. This design doesn't prevent gaming — it makes the *current* gaming (activity-as-productivity) visible.
- **Investigation quality:** Investigations don't produce mergeable code. The merge rate metric explicitly excludes them. Investigation quality needs a different signal (e.g., "was the investigation referenced by subsequent work?").
- **Real-time feedback:** All signals are batch-computed at orient time. The daemon doesn't get real-time ground-truth updates during a poll cycle.

---

## VERIFICATION_SPEC

```yaml
type: design
deliverables:
  - path: .kb/investigations/2026-03-17-design-ground-truth-metric-injection-daemon-orient.md
    description: Architecture design document
verification:
  - description: Design addresses all 9 camouflaged metrics from prior investigation
    method: manual review against metric-camouflage findings
  - description: Design respects No Local Agent State constraint
    method: all computations are functional queries, no caches
  - description: Implementation plan has clear phase boundaries
    method: each phase is independently deployable
  - description: Design references prior investigations (metric-camouflage, autonomous-confidence)
    method: prior work table populated
```
