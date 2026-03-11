# Probe: Measurement Surface Design — Can We Build the Numbers That Confirm or Kill the Model?

**Model:** harness-engineering
**Date:** 2026-03-11
**Status:** Complete

---

## Question

The harness engineering model has 8 hard gates and 5 soft harness components but no integrated measurement surface that can falsify its claims. Invariant #7 says "enforcement without measurement is theological." Can we design measurement for each falsification criterion — gate deflection, accretion velocity, soft harness compliance, and cross-system portability — using existing event infrastructure?

Testing invariant #7 (enforcement + measurement pairing) and the overall model claim that gates improve agent quality.

---

## What I Tested

Audited the existing measurement infrastructure against the 4 falsification criteria:

### 1. Gate Deflection Analysis

**Data source:** `spawn.gate_decision` events in events.jsonl (shipped Mar 11)

**Current state:** Only 3 gate_decision events in 7 days (1 block, 2 bypasses). This is because:
- gate_decision events only log block/bypass, not "allow" (volume management decision)
- Events were just shipped — historical data doesn't exist
- Most spawns go through without hitting gates (no hotspot match, no verification block)

**What's needed:**
- **Fire rate per gate:** `count(gate_decision WHERE gate_name=X) / count(session.spawned)` — but without "allow" events, we can't distinguish "gate evaluated and allowed" from "gate not applicable"
- **Block vs bypass ratio:** Already computable from `spawn.gate_decision` data — `blocks / (blocks + bypasses)` per gate
- **Override frequency:** Already tracked via `spawn.hotspot_bypassed`, `spawn.triage_bypassed`, `spawn.verification_bypassed` events + reasons

**Design decision:** Should we add "allow" events to get true fire rate? Trade-off: volume (every spawn would emit N allow events for N gates) vs completeness. Recommendation: add "allow" events for hotspot and verification gates only (these are the ones with falsification value), not triage (triage fires on every manual spawn, known behavior).

### 2. Accretion Velocity Tracking

**Data source:** `accretion.delta` events + git history

**Current state:**
- accretion.delta events now at ~100% coverage (fixed from 4.7% via git baseline bug fix)
- Pre-gate baseline exists: 6,131 lines/week in cmd/orch/ (week of Mar 3-10)
- Weekly trajectory exists in git: 370 → 1,473 → 6,264 → 6,131 lines/week (Feb 10 → Mar 10)
- 347 commits in cmd/orch/ from Feb 10-Mar 10, 22 in last 2 days

**What's needed:**
- **Baseline snapshot mechanism:** Periodic snapshots of total lines per key directory (cmd/orch/, pkg/*) stored in events.jsonl
- **Trend detection:** Compare post-gate velocity to pre-gate velocity. Needs at least 2-3 weeks of post-gate data.
- **Mar 24 checkpoint rigor:** Need pre-gate baseline frozen now, post-gate data accumulating, clear pass/fail criteria

**Design:** `accretion.snapshot` event type emitted weekly by daemon or cron. Fields: `directory`, `total_lines`, `file_count`, `files_over_800`, `files_over_1500`. First snapshot captures current state as baseline. `orch harness report` computes velocity from snapshot deltas.

### 3. Soft Harness Compliance Measurement

**Data source:** No events exist for soft harness effectiveness. This is the hardest to measure.

**The model's own position:** "probably doesn't work until proven otherwise"

**5 soft harness components:**
1. SKILL.md content — measured (265 contrastive trials, +5 lift for knowledge)
2. CLAUDE.md — unmeasured (daemon.go grew past stated 1500-line convention)
3. .kb/ knowledge — unmeasured
4. SPAWN_CONTEXT.md — unmeasured (advisory injection, no compliance gate)
5. Coaching plugin — partially measured (only OpenCode spawns)

**What's needed:**
- **A/B removal test:** Spawn agents with and without each soft harness component, compare outcomes. This is the gold standard but requires controlled experiments.
- **Proxy metrics:** Measure compliance rates from observable behavior:
  - CLAUDE.md: Do agents follow stated conventions? Measurable via accretion gate violations (convention says 1500, agent adds anyway)
  - SPAWN_CONTEXT.md: Does hotspot advisory content change agent behavior? Measurable only if we track "agent read hotspot warning" → "agent behavior changed"
  - .kb/ knowledge: Does injected knowledge prevent re-investigation? Measurable via duplicate investigation count

**Design:** Soft harness compliance is not measurable from events alone. The controlled experiment (spawn with/without component, compare outcomes) is the only rigorous approach. Create a lightweight experiment framework: `orch experiment create soft-harness-removal --component CLAUDE.md --control-count 5 --treatment-count 5`. Track outcomes by experiment cohort.

### 4. orch harness report Command

**Data source:** All of the above, aggregated

**Current state:** `orch stats` already computes most of what's needed:
- Gate decision stats (block/bypass counts per gate)
- Gate effectiveness stats (gated vs ungated outcomes)
- Verification stats (pass rate, bypass rate, per-gate failures)
- Spawn gate stats (bypass rate per gate, miscalibration detection)
- Override stats (reasons for bypasses)

**What's missing from `orch stats` that `orch harness report` would add:**
- Accretion velocity (pre/post comparison)
- Completion field coverage percentage
- Duplication detection rate and trends
- Falsification verdict per criterion (pass/fail/insufficient-data)
- Soft harness compliance proxy metrics

---

## What I Observed

### The Infrastructure Is Richer Than Expected

The stats_cmd.go (1125 lines) and stats_types.go (274 lines) already implement:
- Gate decision aggregation from `spawn.gate_decision` events
- Gate effectiveness correlation (gated vs ungated completion rates, verification pass rates, durations)
- Per-gate failure rates, bypass reasons, miscalibration detection
- Deduplication of completion/abandonment events
- Pipeline timing aggregation

However, the effectiveness stats are **all zeros** because:
1. gate_decision events were just shipped (only 3 exist)
2. The correlation logic requires beads_id linkage between gate_decision and agent.completed events
3. Most historical spawns predate gate_decision instrumentation

### The Near-Zero Fire Rate Is Already Data

Only 3 gate_decision events in 203 spawns (7 days) = 1.5% fire rate. But this is misleading:
- The instrumentation was just shipped, so most of the 203 spawns predate it
- The legacy `spawn.hotspot_bypassed` events show 66 hotspot bypasses — those are real gate evaluations that predate the new event type
- True fire rate from legacy events: (66 hotspot + 53 triage + 22 verification) / 203 = 69.5% — the gates fire on the majority of spawns

### Falsification Criteria Status

| Criterion | Data Available | Verdict |
|-----------|---------------|---------|
| Gates fire but accretion doesn't slow → gates are ceremony | Need 2+ weeks post-gate data. Baseline: 6,131 lines/week | **Insufficient data** — checkpoint Mar 24 |
| Gate deflection rate near-zero → gates are irrelevant | Legacy events show 69.5% bypass rate, 1.5% via new events | **Partially measurable** — need "allow" events for true fire rate |
| Soft harness removal causes no behavior change → soft harness is inert | No controlled experiments exist | **Not measurable** without experiment framework |
| Second system, no benefit → framework is anecdotal | No second system instrumented | **Not measurable** — requires cross-project deployment |

---

## Model Impact

**Confirms invariant #7:** The measurement gap audit from today's session (52% field gaps, 0 gate events) demonstrates that enforcement without measurement is indeed theological. The infrastructure existed but produced no actionable data.

**Extends the model:** The distinction between "fire rate" and "bypass rate" matters. Legacy events show 69.5% bypass rate (gates fire often). But `spawn.gate_decision` events show 1.5% (new instrumentation captures almost nothing yet). The measurement surface design must handle this transition period — legacy events provide historical context, new events provide structured correlation.

**Identifies a gap:** Soft harness compliance measurement requires controlled experiments, not just event aggregation. The model says "probably doesn't work until proven otherwise" but provides no mechanism to prove otherwise. The experiment framework is the missing piece.

**Quantitative finding:** Mar 24 checkpoint viability depends on:
- At least 50+ spawns with gate_decision events (current: 3)
- Accretion velocity data for 2+ post-gate weeks
- These are achievable if current spawn rate (~29/day) continues
