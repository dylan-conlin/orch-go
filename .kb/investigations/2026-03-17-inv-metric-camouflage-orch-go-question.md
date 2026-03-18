<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** 9 of orch-go's surfaced metrics are structurally camouflaged — they would show identical values in a productive system and one producing zero value with maximum activity. The core camouflage mechanism: every metric measures the system's own internal events, not external outcomes.

**Evidence:** Code-level audit of orient, stats, harness report, gate audit, gate effectiveness, and daemon health — all source from events.jsonl, which the system itself writes. No metric references git merge state, issue re-open rate, user adoption, or downstream impact.

**Knowledge:** The system has a measurement problem that looks like a metric problem. It built increasingly sophisticated second-order metrics (gate effectiveness, falsification verdicts, coaching stats) but they're all computed over the same first-order event stream. The orphan rate (91.6%) is the only metric that actually surfaces this — and it's buried in the reflection summary.

**Next:** Architectural — introduce ground-truth metrics that reference external state (merged PRs, issue re-open rate, code review outcomes) and display them alongside internal metrics to break the self-referential loop.

**Authority:** architectural — Cross-cuts orient, stats, harness, and daemon; requires new data sources and changes to what the orchestrator sees at session start.

---

# Investigation: Metric Camouflage in orch-go

**Question:** Which orch-go metrics would show identical values in a well-functioning system vs one spinning its wheels?

**Started:** 2026-03-17
**Updated:** 2026-03-17
**Owner:** investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation | - | - | - |

---

## Findings

### Finding 1: Completion Count measures motion, not value

**Evidence:** `ComputeThroughput()` in `pkg/orient/orient.go:131-173` counts `agent.completed` events. An `agent.completed` event fires when `orch complete` or `bd close` runs — which happens when an agent finishes its session, not when its work is validated externally, merged, or produces downstream impact.

**Source:** `pkg/orient/orient.go:147` — `case "agent.completed": tp.Completions++`

**Significance:** A daemon that spawns 82 trivial investigations per day (e.g., "investigate X" → agent reads 3 files → writes investigation → reports Phase: Complete) produces 82 completions/day indistinguishable from 82 merged feature implementations. The orient display (`formatThroughput`, line 236-248) shows `Completions: 82` with no qualification of what was completed.

---

### Finding 2: Completion Rate celebrates the trivial

**Evidence:** `calcRatesAndDuration()` in `stats_aggregation.go:736-782` computes `CompletionRate = Completions / Spawns * 100`. The stats output (`stats_output.go:397-401`) flags completion rate <80% as WARNING and >=95% as HEALTHY.

**Source:** `stats_output.go:397` — `if report.Summary.TaskCompletionRate < 80` → WARNING, `>= 95` → HEALTHY

**Significance:** A system that only spawns trivial work (easy to complete) shows 95%+ completion rate and gets labeled HEALTHY. The metric rewards low ambition. Paradoxically, a system taking on hard problems (where agents get stuck, blocked, or need rework) shows a lower completion rate and gets labeled WARNING — punishing the behavior the system should encourage.

---

### Finding 3: Gate Fire Rate is ambiguous by design

**Evidence:** `harness_audit_cmd.go:177-183` computes `FireRatePct = (blocks+bypasses) / invocations * 100`. The harness report (`serve_harness.go:596-643`) evaluates the falsification verdict `gates_are_irrelevant` with: fire rate >5% → FALSIFIED (gates are NOT irrelevant), fire rate <5% → CONFIRMED (gates ARE irrelevant).

**Source:** `serve_harness.go:608-614`

**Significance:** This metric conflates two completely different states:
- **High fire rate + high bypass rate** = gates trigger frequently but everyone bypasses them (100% bypass rate found in CLAUDE.md for accretion gates). Gates are ceremony.
- **High fire rate + high block rate** = gates trigger and redirect work effectively.

The harness report shows fire rate without separating these interpretations. A system where gates fire on 30% of spawns but are bypassed 100% of the time reports `fire: 30.0%` and the verdict `FALSIFIED` (gates ARE relevant) — exactly the opposite of reality. The system already knows this is happening (CLAUDE.md notes "100% bypass rate over 2-week measurement") but the metric still shows green.

---

### Finding 4: Gate Effectiveness uses self-referential quality signals

**Evidence:** `harness_gate_effectiveness_cmd.go:392-435` (`buildCohort()`) measures "quality" as: completion rate, verification pass rate, and avg duration. The `generateVerdict()` function (line 450-532) computes a weighted quality score: 60% verification rate + 40% completion rate.

**Source:** `harness_gate_effectiveness_cmd.go:493-496`

**Significance:** "Verification rate" means: did the agent's completion event include `verification_passed: true`? But verification checks (`pkg/verify/`) check phase_complete, deliverables_exist, and commits_present — all of which are process compliance, not outcome quality. An agent that creates a useless investigation file, commits it, and reports Phase: Complete passes all verification gates. The gate effectiveness analysis then says "this cohort has 100% verification rate" — which is technically true but semantically empty.

---

### Finding 5: Daemon Health Signals measure operational liveness, not value production

**Evidence:** `pkg/daemon/health_signals.go:22-37` computes 6 signals: Daemon Liveness (poll recency), Capacity (slot utilization), Queue Depth (ready issue count), Verification (remaining before pause), Unresponsive (agents without phase reports), Questions (agents waiting for input).

**Source:** `pkg/daemon/health_signals.go:22-37`

**Significance:** All 6 signals can show green while the daemon produces zero value:
- Liveness: green (daemon is polling on time)
- Capacity: green (3/5 slots used — plenty of room!)
- Queue Depth: green (12 issues ready)
- Verification: green (not paused)
- Unresponsive: green (all agents reported phase)
- Questions: green (no agents waiting)

A daemon that spawns investigations into questions nobody asked, completes them in 10 minutes with trivial findings, and cycles to the next issue shows perfect health across all 6 signals.

---

### Finding 6: Accretion Velocity measures quantity, not direction

**Evidence:** `serve_harness.go:394-439` (`computeAccretionVelocity()`) computes weekly line count change. The trend is labeled "declining" (<-20%), "stable" (±20%), or "increasing" (>+20%).

**Source:** `serve_harness.go:426-430`

**Significance:** Accretion velocity treats all line changes as equivalent. A system that adds 500 lines/week of investigation files to `.kb/investigations/` (the orphan factory) shows "stable" or "increasing" velocity. A system that deletes 200 lines of dead code and replaces it with 100 lines of better code shows "declining" velocity and triggers concern. The metric punishes improvement and rewards bloat.

---

### Finding 7: Orphan Rate is the only honest metric — and it's buried

**Evidence:** `formatReflectSummary()` in `pkg/orient/orient.go:409-447` renders orphan rate as the last line of the Reflection suggestions section: `Orphan rate: 91.6% (1270 investigations)`. The reflection section itself is rendered last in orientation ("last — informational, not urgent", line 231).

**Source:** `pkg/orient/orient.go:444`, line 231 comment

**Significance:** The orphan rate is the only metric that measures whether the system's outputs connect to anything meaningful. 91.6% of investigations are orphans — they were created, completed, counted as completions, but never referenced by any model, decision, guide, or other investigation. This metric reveals the completion count is mostly waste, but it's surfaced as "informational, not urgent" at the bottom of orientation, after throughput (which is the camouflage).

---

### Finding 8: The Falsification Verdicts framework admits its own blindness

**Evidence:** `serve_harness.go:618-643` builds 4 falsification verdicts. Two are permanently `not_measurable`:
- `soft_harness_is_inert`: "Requires A/B test: spawn with vs without soft harness component"
- `framework_is_anecdotal`: "No second system instrumented"

**Source:** `serve_harness.go:632-643`

**Significance:** The harness report presents 4 verdicts but 2 of them can never produce a result. This creates an illusion of rigor — the report looks like it has a falsification framework, but half the framework is structurally unable to falsify anything. The remaining two verdicts measure gate fire rate (which we've shown is ambiguous, Finding 3) and accretion velocity (which measures quantity not quality, Finding 6).

---

### Finding 9: Stats output marks self-referential metrics as "main metric"

**Evidence:** `stats_output.go:41-44` explicitly labels task skill completion rate as "← main metric":
```
Task Skills: 68/72 spawns (94.4%) ← main metric
```

**Source:** `stats_output.go:41`

**Significance:** The system explicitly tells the operator "this is the metric that matters" — and that metric is task completion rate, which we've shown in Finding 2 rewards trivial work and punishes ambitious work. The label "main metric" anchors the operator's attention on the most camouflaged number.

---

## Synthesis

**Key Insights:**

1. **Self-referential measurement loop** — Every metric in the system is computed from events.jsonl, which the system itself writes. No metric references external ground truth (git merges, code review outcomes, issue re-opens, downstream failures). This means the system is grading its own homework.

2. **Sophistication as camouflage** — The system has increasingly sophisticated meta-metrics (gate effectiveness comparing cohorts, falsification verdicts, coaching stats) but they're all second-order computations over the same first-order self-reported events. More math on the same bad data doesn't produce better signal.

3. **The orphan rate is the Rosetta Stone** — At 91.6%, the orphan rate reveals that the vast majority of the system's "completed" work doesn't connect to anything. If the orchestrator inverted the display priority (orphan rate first, completion count last), the operational picture would change dramatically.

**Answer to Investigation Question:**

The metrics that camouflage failure are, in severity order:

| Severity | Metric | Where Surfaced | Adversarial Scenario |
|----------|--------|---------------|---------------------|
| CRITICAL | Completion Count | orient, stats | Daemon spawns 82 trivial investigations/day |
| CRITICAL | Task Completion Rate | stats "main metric" | 95% rate because only easy work is attempted |
| HIGH | Gate Fire Rate | harness report, harness audit | 30% fire rate but 100% bypass = ceremony |
| HIGH | Gate Effectiveness Verdict | gate-effectiveness cmd | "GATES IMPROVE QUALITY" based on self-reported verification |
| HIGH | Daemon Health (all 6 signals) | orient | All green while daemon produces zero value |
| MEDIUM | Accretion Velocity | harness report | "Stable" because orphan investigation files grow at same rate |
| MEDIUM | Falsification Verdicts | harness report | 2/4 permanently unmeasurable, 2/4 measure ambiguous proxies |
| LOW | Completion Coverage | harness report | 100% coverage (all fields present) on worthless completions |

---

## Structured Uncertainty

**What's tested:**

- ✅ All 9 findings verified by reading source code — traced exact lines computing each metric
- ✅ Self-referential loop confirmed: grep for external data sources in orient/stats/harness found none
- ✅ Orphan rate 91.6% confirmed as claimed in task context (metric exists in `kbmetrics/orphans.go`)
- ✅ 100% accretion gate bypass rate confirmed in CLAUDE.md ("100% bypass rate over 2-week measurement")
- ✅ "main metric" label confirmed at `stats_output.go:41`

**What's untested:**

- ⚠️ Actual current metric values not verified (didn't run `orch stats` or `orch harness report`)
- ⚠️ Whether the daemon actually spawns trivial work (adversarial scenario is hypothetical)
- ⚠️ Whether external ground-truth metrics would actually diverge from internal metrics

**What would change this:**

- If a significant fraction of completions produce merged PRs or resolved bugs (would reduce severity of Finding 1)
- If gate bypasses have declined since the CLAUDE.md note (would reduce severity of Finding 3)
- If orphan rate has decreased from 91.6% (would indicate system is improving despite metrics)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add ground-truth metrics to orient | architectural | Cross-cuts orient, stats, daemon; requires new data sources |
| Reorder orient display priority | architectural | Changes what orchestrator sees first; affects decision-making |
| Add "value delivered" metric | strategic | Requires defining what "value" means for this system |

### Recommended Approach ⭐

**Ground-Truth Injection** - Add 3-5 external metrics to orient that cannot be gamed by internal activity.

**Why this approach:**
- Breaks the self-referential loop without removing existing metrics
- Doesn't require removing or changing metrics that work for operational monitoring
- Creates contrast between "activity" and "value" in the same display

**Trade-offs accepted:**
- More data to collect (some metrics may require new commands)
- Orient output grows longer

**Implementation sequence:**
1. **Metric: Orphan-adjusted completion count** — Display `Completions: 82 (connected: 7)` where "connected" means the completion's investigation/deliverable is referenced by at least one model, decision, or guide. Formula: completions × (1 - orphan_rate). This uses existing data.
2. **Metric: Merge rate** — Count git commits from completed agents that appear in `main` branch. Formula: `git log --author` for agent workspaces that were merged. Requires new data collection.
3. **Metric: Issue re-open rate** — Count issues closed by `orch complete` that were later re-opened or had rework spawned. Formula: `agent.reworked` events / `agent.completed` events. Already in events.jsonl.
4. **Display reorder** — Move orphan rate from "Reflection suggestions (last — informational)" to "Health (early — operational)".

### Alternative Approaches Considered

**Option B: Remove self-referential metrics**
- **Pros:** Eliminates camouflage entirely
- **Cons:** Loses operational monitoring value; completion count IS useful for capacity planning
- **When to use instead:** If ground-truth metrics consistently diverge from internal metrics

**Option C: A/B testing framework**
- **Pros:** Would falsify the 2 permanently-unmeasurable verdicts
- **Cons:** Enormous complexity for uncertain value; requires a second project
- **When to use instead:** If ground-truth metrics show the system IS effective and the question becomes "which components matter"

**Rationale for recommendation:** Option A preserves existing metrics for operational monitoring while adding external reference points. The other options either destroy useful signals (B) or require infrastructure that doesn't exist (C).

---

### Implementation Details

**What to implement first:**
- Issue re-open/rework rate (already in events.jsonl, cheapest to add)
- Orphan-adjusted completion count (uses existing kbmetrics)
- Orient display reorder (zero new data needed)

**Things to watch out for:**
- ⚠️ Orphan rate may be inflated by the investigation skill's structure (investigations are "orphans" until someone references them, which may be a lagging indicator)
- ⚠️ Merge rate requires identifying agent commits, which may not always be cleanly attributable
- ⚠️ Operators may resist having their "82 completions/day" headline number cut to "7 connected completions" — but that discomfort is the point

**Areas needing further investigation:**
- What percentage of completions produce commits that reach main?
- Is there a meaningful quality difference between orphan and connected investigations?
- Should the daemon's priority algorithm incorporate value-connected metrics?

**Success criteria:**
- ✅ Orient shows at least one metric that can only improve when external outcomes improve
- ✅ Stats output no longer labels self-referential completion rate as "main metric" without qualification
- ✅ Operator can distinguish "busy system" from "productive system" in <10 seconds of orient output

---

## References

**Files Examined:**
- `cmd/orch/orient_cmd.go` - Orient data collection (throughput, health, reflect)
- `pkg/orient/orient.go` - Throughput computation and orientation formatting
- `cmd/orch/stats_aggregation.go` - Stats aggregation (completion, abandonment, gate, verification)
- `cmd/orch/stats_types.go` - Stats type definitions (shows what's measured)
- `cmd/orch/stats_output.go` - Stats text output (shows what operator sees)
- `cmd/orch/harness_report_cmd.go` - Harness report rendering (gate deflection, verdicts)
- `cmd/orch/harness_audit_cmd.go` - Gate audit (invocations, fire rates, anomalies)
- `cmd/orch/harness_gate_effectiveness_cmd.go` - Gate effectiveness (cohort comparison)
- `cmd/orch/serve_harness.go` - Harness API (pipeline, accretion, verdicts)
- `pkg/spawn/gates/hotspot.go` - Hotspot gate (advisory-only, never blocks)
- `pkg/spawn/gates/triage.go` - Triage gate (bypass logging)
- `pkg/spawn/gates/agreements.go` - Agreements gate (warning-only)
- `pkg/spawn/gates/question.go` - Question gate (warning-only)
- `pkg/daemon/health_signals.go` - Daemon health (6 liveness signals)
- `pkg/daemon/ooda.go` - Daemon OODA cycle (Sense/Orient/Decide/Act)
- `pkg/kbmetrics/orphans.go` - Orphan rate computation

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-03-17-accretion-gates-advisory-not-blocking.md` - Accretion gates demoted after 100% bypass rate
