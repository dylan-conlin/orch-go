## Summary (D.E.K.N.)

**Delta:** Designed 4-component measurement surface (gate deflection, accretion velocity, soft harness compliance, harness report) plus dashboard visualization — all using existing event infrastructure with 3 targeted additions.

**Evidence:** Audited 40+ event types, 1400 lines of stats code, 4,831 events, and 45+ API endpoints. Infrastructure is 80% ready — gaps are: no "allow" gate events (can't compute true fire rate), no accretion snapshots (can't trend velocity), no harness API endpoint (dashboard can't render pipeline view).

**Knowledge:** The legacy bypass events (69.5% fire rate) and new gate_decision events (1.5% fire rate) create a measurement transition period. Design must bridge both. Soft harness compliance is NOT measurable from events — requires controlled experiments.

**Next:** Create 5 implementation issues (data layer, gate deflection, accretion snapshots, harness report CLI, dashboard visualization). Route through implementation — no architectural decisions needed, design extends existing patterns.

**Authority:** implementation — All components extend existing event/stats/API/dashboard patterns with no new architectural primitives.

---

# Investigation: Measurement Surface for Harness Engineering Falsification

**Question:** What measurement infrastructure makes harness engineering's claims falsifiable, and what data surfaces present the numbers that confirm or kill the model?

**Started:** 2026-03-11
**Updated:** 2026-03-11
**Owner:** orch-go-d8bt5
**Phase:** Complete
**Next Step:** None — issue decomposition ready for daemon
**Status:** Complete
**Model:** harness-engineering

**Patches-Decision:** `.kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md` (extends measurement surface)

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `2026-03-11-inv-investigation-audit-orch-go-ecosystem.md` | extends — measurement gap audit found 52% field gaps, 0 gate events, survivorship bias | Yes — verified 3 gate_decision events in 7 days, zeros in gate_effectiveness_stats | None |
| `2026-03-08-probe-30-day-accretion-trajectory-gate-effectiveness.md` | extends — established pre-gate baselines and identified trajectory analysis method | Yes — confirmed 6,131 lines/week pre-gate velocity | None |
| `2026-03-10-probe-health-score-calibration-vs-structural-improvement.md` | confirms — health score removal proves soft measurement masquerades as hard | Yes — 89% was recalibration | None |

---

## Findings

### Finding 1: Infrastructure Is 80% Ready — 3 Targeted Additions Close the Gaps

**Evidence:** The existing stats pipeline (stats_cmd.go:1125 lines, stats_types.go:274 lines) already computes:
- Gate decision aggregation from `spawn.gate_decision` events (GateDecisionStats)
- Gate effectiveness correlation — gated vs ungated completion rates, verification rates, durations (GateEffectivenessStats)
- Per-gate failure rates, bypass reasons, miscalibration detection (VerificationStats, SpawnGateStats)
- Override frequency tracking across all gate types (OverrideStats)
- Pipeline timing per advisory step (from agent.completed events)

What's missing:
1. **"Allow" gate events** — without these, fire rate = blocks+bypasses/spawns, which conflates "gate evaluated and allowed" with "gate not applicable." True fire rate requires knowing when a gate evaluated.
2. **Accretion snapshots** — periodic line count snapshots for trend detection. Currently only per-completion deltas exist (accretion.delta).
3. **Harness API endpoint** — dashboard has no `/api/harness` to serve the pipeline view.

**Source:** `cmd/orch/stats_cmd.go:653-690` (gate_decision processing), `cmd/orch/stats_types.go:214-273` (gate types), `pkg/events/logger.go` (all event types), `cmd/orch/serve.go` (API endpoints)

**Significance:** The design doesn't require rebuilding — it extends. Each gap has a clear, small addition that enables a measurement surface.

---

### Finding 2: Legacy vs New Event Transition Creates Measurement Ambiguity

**Evidence:** Two parallel event systems track gate activity:
- **Legacy events** (shipped weeks ago): `spawn.hotspot_bypassed` (66), `spawn.triage_bypassed` (53), `spawn.verification_bypassed` (22) = 141 total bypass events across 203 spawns = **69.5% bypass rate**
- **New events** (shipped Mar 11): `spawn.gate_decision` = 3 events total (1 block, 2 bypasses) = **1.5% fire rate** (misleading — most spawns predate instrumentation)

`orch stats` already computes both. SpawnGateStats uses legacy events. GateDecisionStats uses new events. But they're reported separately, creating confusion about true gate activity.

**Source:** `cmd/orch/stats_cmd.go:527-571` (legacy bypass tracking), `cmd/orch/stats_cmd.go:653-690` (new gate_decision tracking), `orch stats --json` output

**Significance:** `orch harness report` must unify these into a single "gate deflection" view that handles the transition period gracefully. Until enough gate_decision events accumulate (est. 2+ weeks at 29 spawns/day), use legacy events as the primary source with gate_decision as supplement.

---

### Finding 3: Soft Harness Compliance Cannot Be Measured From Events

**Evidence:** The 5 soft harness components (SKILL.md, CLAUDE.md, .kb/ knowledge, SPAWN_CONTEXT.md, coaching plugin) influence agent behavior through context, not enforcement. Events capture what agents *did*, not what they *read and considered*.

Only SKILL.md has been measured (265 contrastive trials, +5 lift for knowledge). CLAUDE.md demonstrably failed — daemon.go grew past the stated 1500-line convention while the convention existed. But this is anecdotal, not systematic.

**Proxy metrics available:**
- Convention violation rate: `accretion.delta` events where files cross CLAUDE.md thresholds
- Re-investigation rate: duplicate investigation filenames in .kb/ (same topic investigated multiple times = knowledge not surfacing)
- Hotspot advisory compliance: agents that received SPAWN_CONTEXT.md hotspot warnings vs. agents that added to hotspot files anyway

**Source:** Model Section 1 (soft harness effectiveness table), `pkg/spawn/context.go` (SPAWN_CONTEXT.md generation), CLAUDE.md accretion boundaries section

**Significance:** The model correctly says "probably doesn't work until proven otherwise." The measurement surface can provide proxy metrics, but definitive measurement requires controlled A/B experiments (spawn with/without component, compare outcomes). That's a separate, more expensive piece of work. V1 measurement uses proxies.

---

### Finding 4: Dashboard Architecture Supports Pipeline Visualization

**Evidence:** The dashboard (Svelte, port 5188) already has:
- 45+ API endpoints serving structured JSON
- SSE push for real-time data (agents, events, services)
- REST pull on 30-60s intervals for stable metrics
- Collapsible sections with progressive disclosure
- Stats bar with compact metric display
- Mode toggle (operational/historical)

A harness pipeline view fits naturally as a new page or section:
- Data: `GET /api/harness` endpoint (new) serving gate deflection, accretion velocity, compliance proxies, falsification verdicts
- Visualization: Pipeline diagram (spawn → authoring → pre-commit → completion) with harness components mapped to stages
- Refresh: 60s REST poll (gate data changes slowly)

**Source:** `cmd/orch/serve.go` (endpoint registration), `web/src/routes/+page.svelte` (home page sections), `web/src/lib/stores/` (data flow patterns)

**Significance:** No new architectural patterns needed. The dashboard already handles health analytics (coaching, attention, verification). Harness report is the same pattern applied to a new data domain.

---

## Synthesis

**Key Insights:**

1. **Unify legacy and new gate events** — The "harness report" must present a single coherent view of gate deflection that bridges the transition from legacy bypass events to structured gate_decision events. The transition period (now → Mar 24) requires both sources.

2. **Accretion velocity is the strongest falsification signal** — Pre-gate baseline exists (6,131 lines/week), post-gate data accumulating, Mar 24 checkpoint gives 2 weeks of post-gate data. This is the one criterion that can produce a clear pass/fail by the checkpoint.

3. **Soft harness measurement requires a different approach** — Events measure enforcement outcomes, not knowledge influence. Proxy metrics are available (convention violations, re-investigation rate) but definitive measurement requires controlled experiments. V1 uses proxies; V2 considers experiment framework.

4. **Dashboard visualization and CLI command share the same data layer** — Design the API endpoint (`GET /api/harness`) to serve structured data that both `orch harness report` (CLI) and the dashboard visualization consume. Single data source, two rendering surfaces.

**Answer to Investigation Question:**

Yes, we can build the measurement surface using existing infrastructure with 3 targeted additions. The design decomposes into 5 implementation components:

1. **Data layer** — Add "allow" gate events, accretion snapshots, and harness API endpoint
2. **Gate deflection analysis** — Compute fire rate, block/bypass ratio, override frequency from unified event sources
3. **Accretion velocity tracking** — Periodic snapshots + velocity computation with pre/post gate comparison
4. **`orch harness report` CLI** — Single command producing falsification verdicts
5. **Dashboard harness page** — Pipeline visualization with measurement status overlay

---

## Structured Uncertainty

**What's tested:**

- ✅ Event infrastructure supports gate deflection analysis (verified: 3 gate_decision events, 141 legacy bypass events parseable via stats_cmd.go)
- ✅ Accretion velocity baseline exists (verified: 6,131 lines/week from git log, 347 commits in 4 weeks)
- ✅ Dashboard architecture supports new visualization pages (verified: 45+ endpoints, SSE+REST pattern, Svelte routing)
- ✅ Completion field coverage now ~100% (verified: bare event dedup shipped, all 5 emission paths enriched)
- ✅ Gate effectiveness correlation logic exists in stats_cmd.go:1007-1110 (verified: reads gate_decision + agent.completed events)

**What's untested:**

- ⚠️ Whether 2 weeks of post-gate data (Mar 10-24) is sufficient for statistically meaningful accretion velocity comparison
- ⚠️ Whether proxy metrics for soft harness (convention violations, re-investigation rate) correlate with actual soft harness effectiveness
- ⚠️ Whether the dashboard harness page provides actionable insight vs. just being another data display
- ⚠️ Whether gate_decision "allow" events create unacceptable event volume (est. ~5 events per spawn × 29 spawns/day = ~145/day — likely acceptable)

**What would change this:**

- If accretion velocity shows no change by Mar 24, the pre-commit gate is ceremony (gates are falsified for accretion)
- If gate deflection shows >90% bypass rate with no quality difference, gates are noise (gate calibration spiral confirmed)
- If soft harness proxy metrics show 0% correlation with outcomes, soft harness measurement requires only controlled experiments (proxies useless)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add "allow" gate events to spawn preflight | implementation | Extends existing LogGateDecision pattern, no new event types |
| Add accretion.snapshot periodic event | implementation | Extends existing event logger, follows accretion.delta pattern |
| Add `GET /api/harness` endpoint | implementation | Follows existing serve.go endpoint pattern |
| Add `orch harness report` command | implementation | Follows existing stats_cmd.go pattern |
| Add dashboard harness page | implementation | Follows existing Svelte page/component pattern |

### Recommended Approach: Layered Build With Mar 24 Checkpoint Focus

**Build the data layer first, then CLI, then dashboard — each layer consumes the previous.**

**Why this approach:**
- Data layer enables both CLI and dashboard (build once, render twice)
- CLI can ship before dashboard (faster feedback on falsification criteria)
- Mar 24 checkpoint needs gate deflection + accretion velocity — prioritize these

**Trade-offs accepted:**
- Soft harness compliance measured via proxies only (not controlled experiments) — acceptable because the model already says "probably doesn't work"
- Dashboard visualization ships after Mar 24 — acceptable because CLI provides the same data for the checkpoint

**Implementation sequence:**
1. **Data layer** (parallel) — "allow" gate events + accretion snapshots + harness API endpoint
2. **`orch harness report`** — CLI command consuming harness API data, producing falsification verdicts
3. **Dashboard harness page** — Svelte pipeline visualization consuming same API

---

### Implementation Details

#### Component 1: Gate "Allow" Events (spawn.gate_decision with decision=allow)

**Where:** `pkg/orch/spawn_preflight.go`

**What:** After each gate evaluates and the spawn proceeds (no block, no bypass), emit:
```go
logGateDecision("hotspot", "allow", skill, "no matching hotspot files", targetFiles)
logGateDecision("verification", "allow", skill, "no unverified work", nil)
```

**Not needed for triage gate** — triage fires on every manual spawn (known, not falsifiable).

**Schema:** Same `spawn.gate_decision` event, `decision: "allow"`. No new event types.

**Volume estimate:** ~2 "allow" events per spawn (hotspot + verification evaluated) × 29 spawns/day = ~58 events/day. Acceptable.

**Stats integration:** Update `stats_cmd.go` gate_decision processing to include "allow" in GateDecisionStats (currently only tracks block/bypass). GateEffectivenessStats already has `TotalAllows` field — currently calculated as `TotalEvaluations - TotalBlocks - TotalBypasses`, which is wrong (evaluations only count block+bypass events). Fix to count actual "allow" events.

#### Component 2: Accretion Snapshots (accretion.snapshot event)

**Where:** New function in `pkg/events/logger.go` + emit from daemon or cron

**What:** Weekly snapshot of directory-level metrics:
```json
{
  "type": "accretion.snapshot",
  "timestamp": 1773360000,
  "data": {
    "directory": "cmd/orch/",
    "total_lines": 14523,
    "file_count": 24,
    "files_over_800": 12,
    "files_over_1500": 2,
    "largest_file": {"path": "stats_cmd.go", "lines": 1125},
    "weekly_delta_lines": 340,
    "snapshot_type": "weekly"
  }
}
```

**Trigger:** Daemon emits snapshot at startup if last snapshot >6 days ago. Or: `orch harness snapshot` manual command.

**Baseline capture:** Emit first snapshot NOW to freeze pre-gate baseline for Mar 24 comparison.

**Stats integration:** `orch harness report` computes velocity = (snapshot[n].total_lines - snapshot[n-1].total_lines) / days_between.

#### Component 3: Harness API Endpoint (GET /api/harness)

**Where:** `cmd/orch/serve_harness.go` (new file, ~200 lines)

**What:** Aggregates gate deflection, accretion velocity, completion coverage, falsification verdicts into structured JSON:

```json
{
  "pipeline": [
    {
      "stage": "spawn",
      "components": [
        {
          "name": "triage_gate",
          "type": "hard",
          "measurement_status": "flowing",
          "fire_rate": 0.26,
          "block_rate": 0.02,
          "bypass_rate": 0.24,
          "last_fired": "2026-03-11T14:23:00Z",
          "recent_decisions": [...]
        },
        {
          "name": "hotspot_gate",
          "type": "hard",
          "measurement_status": "flowing",
          "fire_rate": 0.33,
          "block_rate": 0.01,
          "bypass_rate": 0.32,
          "override_reasons": [...]
        }
      ]
    },
    {
      "stage": "authoring",
      "components": [
        {
          "name": "CLAUDE.md",
          "type": "soft",
          "measurement_status": "proxy_only",
          "convention_violation_rate": 0.15,
          "proxy_metric": "accretion gate violations on convention-documented files"
        },
        {
          "name": "SPAWN_CONTEXT.md",
          "type": "soft",
          "measurement_status": "proxy_only",
          "hotspot_advisory_compliance": null
        },
        {
          "name": "kb_knowledge",
          "type": "soft",
          "measurement_status": "proxy_only",
          "re_investigation_rate": null
        }
      ]
    },
    {
      "stage": "pre_commit",
      "components": [
        {
          "name": "accretion_gate",
          "type": "hard",
          "measurement_status": "flowing",
          "fire_rate": null,
          "block_rate": null,
          "last_fired": null
        },
        {
          "name": "build_gate",
          "type": "hard",
          "measurement_status": "flowing",
          "fail_rate": 0.015
        }
      ]
    },
    {
      "stage": "completion",
      "components": [
        {
          "name": "verification_pipeline",
          "type": "hard+soft",
          "measurement_status": "flowing",
          "pass_rate": 0.061,
          "bypass_rate": 0.338,
          "gates": [...]
        },
        {
          "name": "duplication_detector",
          "type": "hard",
          "measurement_status": "flowing",
          "detection_count": 4,
          "avg_pipeline_ms": 742
        },
        {
          "name": "explain_back",
          "type": "human",
          "measurement_status": "flowing",
          "fail_rate": 0.253
        }
      ]
    }
  ],
  "accretion_velocity": {
    "current_weekly_lines": 340,
    "baseline_weekly_lines": 6131,
    "velocity_change_pct": -94.5,
    "snapshots": [...],
    "trend": "declining"
  },
  "completion_coverage": {
    "total_completions": 198,
    "with_skill": 190,
    "with_outcome": 195,
    "with_duration": 198,
    "coverage_pct": 96.0
  },
  "falsification_verdicts": {
    "gates_are_ceremony": {
      "criterion": "Gates ship, accretion doesn't slow",
      "status": "insufficient_data",
      "evidence": "Pre-gate: 6,131 lines/week. Post-gate: 2 days data. Checkpoint: Mar 24.",
      "threshold": "Post-gate velocity must be <50% of pre-gate for 2+ consecutive weeks"
    },
    "gates_are_irrelevant": {
      "criterion": "Gate deflection rate near-zero",
      "status": "falsified",
      "evidence": "69.5% spawn bypass rate (141/203 spawns). Gates fire on majority of spawns.",
      "threshold": "Fire rate <5% would indicate irrelevance"
    },
    "soft_harness_is_inert": {
      "criterion": "Soft harness removal causes no behavior change",
      "status": "not_measurable",
      "evidence": "No controlled experiments. Proxy: CLAUDE.md convention violation (daemon.go grew past 1500 stated limit).",
      "threshold": "Requires A/B test: spawn with vs without soft harness component"
    },
    "framework_is_anecdotal": {
      "criterion": "Second system, no benefit",
      "status": "not_measurable",
      "evidence": "No second system instrumented.",
      "threshold": "Deploy harness tooling to second project, compare outcomes"
    }
  },
  "measurement_coverage": {
    "total_components": 13,
    "with_measurement": 8,
    "proxy_only": 3,
    "unmeasured": 2,
    "theological_zone": ["accretion_precommit_allow", "coaching_plugin_effectiveness"]
  }
}
```

**Data source:** Reads events.jsonl (reuses parseEvents from stats_cmd.go), computes harness-specific aggregations.

#### Component 4: `orch harness report` CLI Command

**Where:** `cmd/orch/harness_report_cmd.go` (new file, ~150 lines)

**What:** Calls the same aggregation as `/api/harness`, renders to terminal:

```
═══ HARNESS REPORT (Last 7 days, 203 spawns) ═══

PIPELINE VIEW
  SPAWN          triage ■■■■░  hotspot ■■■■░  verification ■■░░░
  AUTHORING      CLAUDE.md ░░░  SPAWN_CONTEXT ░░░  .kb/ ░░░
  PRE-COMMIT     accretion ■■░  build ■■■■■
  COMPLETION     verify ■■■■░  dupdetect ■■■■■  explain_back ■■░░░

  ■ = measurement flowing  ░ = proxy/unmeasured

GATE DEFLECTION (from 203 spawns)
  triage        fire: 26.1%  block: 0%   bypass: 26.1%  [53 bypasses]
  hotspot       fire: 32.5%  block: 0.5% bypass: 32.0%  [66 bypasses, 1 block]
  verification  fire: 10.8%  block: 0%   bypass: 10.8%  [22 bypasses]
  ⚠ hotspot: 66 bypasses with no recorded reason

ACCRETION VELOCITY
  Pre-gate baseline:  6,131 lines/week (Feb 10 - Mar 10)
  Post-gate current:  [insufficient data — 2 days]
  Checkpoint:         Mar 24 (13 days remaining)
  Target:             <3,066 lines/week (50% reduction)

COMPLETION COVERAGE
  With skill field:   96.0% (190/198)
  With outcome field: 98.5% (195/198)
  With duration:      100%  (198/198)
  Overall:            96.0%

FALSIFICATION VERDICTS
  ✗ Gates are ceremony:    INSUFFICIENT DATA (need 2+ weeks post-gate)
  ✓ Gates are irrelevant:  FALSIFIED (69.5% fire rate — gates fire often)
  ? Soft harness is inert: NOT MEASURABLE (no controlled experiments)
  ? Framework is anecdotal: NOT MEASURABLE (no second system)
```

**Flags:** `--days N` (default 7), `--json` (machine-readable), `--verbose` (include recent decisions per gate)

#### Component 5: Dashboard Harness Page

**Where:** `web/src/routes/harness/+page.svelte` (new page), `web/src/lib/stores/harness.ts` (new store), supporting components

**Design:**

**Pipeline View (main visualization):**
- Horizontal pipeline: SPAWN → AUTHORING → PRE-COMMIT → COMPLETION
- Each stage is a column containing harness component cards
- Cards color-coded by type: hard (blue), soft (yellow), human (green)
- Measurement status overlay: flowing (bright), proxy (dim), unmeasured (gray/dashed border)
- Click card → expand to show fire rate, recent decisions, override reasons

**Live Data Overlay:**
- Gate deflection rates shown on each card as mini-bar
- Override frequency as small counter badge
- Last N firings shown on hover/expand
- Auto-refresh every 60s from `GET /api/harness`

**Coverage Heat Map:**
- Bottom section: matrix of files/areas vs gate coverage
- Rows: key files (cmd/orch/*.go, pkg/daemon/*.go)
- Columns: which gates cover this file (hotspot, accretion, build, dupdetect)
- Color: number of gates covering (0=red, 1=yellow, 2+=green)
- Exposes blind spots (files with 0-1 gate coverage)

**Falsification Dashboard:**
- Right panel or bottom section
- 4 falsification criteria as cards
- Status: pass (green), fail (red), insufficient_data (gray), not_measurable (striped)
- Each card shows threshold, current value, evidence snippet
- Accretion velocity trend sparkline

**Accretion Trend:**
- Small chart showing weekly lines over time
- Vertical line marking gate wiring date (Mar 10)
- Pre-gate trend vs post-gate trend clearly visible

**Data flow:**
```
GET /api/harness (60s poll)
  → harness.ts store
    → +page.svelte renders pipeline, gates, velocity, verdicts
```

**Coverage heat map data:** Extend `/api/harness` to include file-level gate coverage matrix, computed from:
- Hotspot analysis output (which files are hotspots)
- Accretion gate thresholds (which files are >800, >1500 lines)
- Duplication detector scope (which files had duplications detected)
- Build/vet gate (covers all Go files)

---

### Things to watch out for:

- ⚠️ **Defect class #2 (Multi-Backend Blindness):** Gate events from Claude CLI spawns vs OpenCode spawns must both flow through the same event infrastructure. Verify `logGateDecision` is called in both spawn paths.
- ⚠️ **Defect class #5 (Contradictory Authority Signals):** Legacy bypass events and new gate_decision events must not produce conflicting numbers. The harness report should clearly label which data source each metric comes from during the transition period.
- ⚠️ **Defect class #3 (Stale Artifact Accumulation):** Accretion snapshots accumulate in events.jsonl forever. Consider adding a `snapshot_type` field so cleanup can differentiate snapshot events from operational events.
- ⚠️ **Tab indentation in Svelte files:** Dashboard harness page will use tabs. Use Write tool for new files, Edit with extra context lines for modifications.

### Success criteria:

- ✅ `orch harness report` produces all 4 falsification verdicts with data sources cited
- ✅ Mar 24 checkpoint has pre-gate baseline + 2 weeks post-gate accretion velocity data
- ✅ Gate deflection rate is computable from unified event sources (legacy + new)
- ✅ Dashboard harness page shows pipeline view with measurement status overlay
- ✅ Coverage heat map identifies files with <2 gate coverage (blind spots)
- ✅ Completion field coverage is >95%

---

## References

**Files Examined:**
- `cmd/orch/stats_cmd.go` — Stats aggregation (1125 lines), gate_decision processing, gate effectiveness correlation
- `cmd/orch/stats_types.go` — Type definitions (274 lines), GateEffectivenessStats, QualityMetrics
- `pkg/events/logger.go` — Event schema, 40+ event types
- `pkg/orch/spawn_preflight.go` — Gate decision emission points (lines 16-57, 189-199)
- `cmd/orch/serve.go` — 45+ API endpoints, registration pattern
- `web/src/routes/+page.svelte` — Dashboard home page (900+ lines), section structure
- `.kb/models/harness-engineering/model.md` — 8 hard gates, 5 soft harness components, 8 invariants
- `.kb/plans/2026-03-11-measurement-instrumentation.md` — Phase 1-3 shipped, Phase 4 blocked
- `.kb/plans/2026-03-11-gate-signal-vs-noise.md` — Gate census and classification plan

**Commands Run:**
```bash
# Current stats state
orch stats --days 7 --json | python3 -c "import json, sys; ..."

# Commit velocity
git log --oneline --since="2026-02-10" --until="2026-03-10" -- cmd/orch/ | wc -l  # 347
git log --oneline --since="2026-03-10" -- cmd/orch/ | wc -l  # 22

# Events volume
wc -l ~/.orch/events.jsonl  # 4,831
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md` — Gate architecture this measurement validates
- **Plan:** `.kb/plans/2026-03-11-measurement-instrumentation.md` — Instrumentation plan this design builds on
- **Plan:** `.kb/plans/2026-03-11-gate-signal-vs-noise.md` — Gate signal/noise classification this design enables
- **Probe:** `.kb/models/harness-engineering/probes/2026-03-11-probe-measurement-surface-design-falsification.md` — Companion probe with raw data

---

## Investigation History

**2026-03-11 14:00:** Investigation started
- Initial question: What measurement infrastructure makes harness engineering falsifiable?
- Context: Model has 8 hard gates and 5 soft harness components but no integrated measurement. Invariant #7 says enforcement without measurement is theological. Mar 24 checkpoint is first scheduled falsification test.

**2026-03-11 14:30:** Exploration complete — 4 forks identified
- Infrastructure richer than expected (40+ event types, comprehensive stats)
- Main gap: near-zero gate_decision events (3 in 7 days), zero gate_effectiveness correlations

**2026-03-11 15:00:** Design complete — 5 implementation components
- Data layer, gate deflection, accretion snapshots, harness report CLI, dashboard visualization
- All extend existing patterns, no architectural decisions needed

**2026-03-11 15:30:** Investigation completed
- Status: Complete
- Key outcome: 5-component measurement surface designed using existing infrastructure + 3 targeted additions
