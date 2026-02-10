# Decision: System-Level Success Metrics for orch

**Date:** 2026-02-09
**Status:** Accepted
**Context:** Define a durable metric set for system success using existing telemetry, grounded in `.kb/investigations/2026-02-08-inv-outcome-data-inventory-data-exists.md` and validated by probe `.kb/models/dashboard-agent-status/probes/2026-02-09-system-success-metrics-baseline-snapshot.md`.
**Resolves:** orch-go-5tvh6

## Decision

Define system success using a **balanced outcome scorecard** of four primary metrics and one quality guardrail.

We explicitly do **not** collapse these into a single score. orch is healthy only when all dimensions are healthy enough.

### Primary Metrics (what defines success)

1. **Completion Reliability**
   - **Definition:** `closed_success / total_issues` (entity-level, deduplicated by issue ID)
   - **Source:** `.beads/issues.jsonl`
   - **Thresholds:**
     - Green: `>= 97%`
     - Watch: `95% to <97%`
     - Red: `< 95%`

2. **Time-to-Value**
   - **Definition:** Spawn-to-close duration percentiles from workspace spawn metadata and issue close times
   - **Source:** `.orch/workspace/*/.spawn_time` + `.beads/issues.jsonl.closed_at`
   - **Thresholds:**
     - Green: `p50 <= 15m` and `p90 <= 90m`
     - Watch: `p50 <= 25m` and `p90 <= 120m`
     - Red: Above watch thresholds

3. **Abandonment + Retry Pressure**
   - **Definition A (volume):** `agent.abandoned` events per period
   - **Definition B (repeat):** Share of abandoned issue IDs with `>=2` abandonments in same period
   - **Source:** `~/.orch/events.jsonl`
   - **Thresholds (initial, trend-based):**
     - Green: 4-week trend non-increasing for both volume and repeat share
     - Watch: one of volume/repeat trend increasing
     - Red: both increasing for 2+ consecutive weekly windows

4. **Investigation Promotion Throughput**
   - **Definition:** Share of investigations referenced by at least one downstream model/decision/guide
   - **Source:** Citation scan over `.kb/investigations/`, `.kb/models/`, `.kb/decisions/`, `.kb/guides/`
   - **Thresholds:**
     - Green: `>= 25%` referenced within rolling 30 days
     - Watch: `15% to <25%`
     - Red: `< 15%`

### Quality Guardrail (required for trust)

5. **Abandonment Reason Completeness**
   - **Definition:** `% of agent.abandoned events with non-empty reason`
   - **Source:** `~/.orch/events.jsonl`
   - **Thresholds:**
     - Green: `>= 95%`
     - Watch: `80% to <95%`
     - Red: `< 80%`

This is a guardrail, not a success metric, because it measures confidence in abandonment analytics rather than system outcomes directly.

---

## Rationale

### Substrate Trace

- **Principle:** `Observation Infrastructure` - If we cannot observe a state transition, we cannot manage it; metrics must be entity-level and trustworthy.
- **Decision:** `2026-01-14-separate-observation-from-intervention` - Metrics are passive observation; they must not be coupled to intervention logic.
- **Decision:** `2026-01-08-observation-infrastructure-principle` - Observation gaps are P1 because they produce false negatives and trust erosion.

### Why this metric set

- It captures both **outcome** (completion, time-to-value) and **failure pressure** (abandon/retry), then closes the learning loop with **knowledge promotion**.
- It uses telemetry that already exists, so we can start measuring immediately without blocking on schema redesign.
- It avoids single-number masking, where good completion can hide poor abandonment quality or weak knowledge conversion.

### Why these thresholds

- Thresholds are set from current and recent observed baselines, with deliberate headroom for drift.
- Time-to-value thresholds use percentile bands to avoid tail outliers dominating system health interpretation.
- Abandonment/retry starts trend-based because absolute denominators are still maturing across attempts/workspaces.

---

## Consequences

### Positive

- Defines success as a multi-dimensional, operationally measurable contract.
- Enables weekly health review without additional instrumentation work.
- Creates a clear trigger surface for when architectural instrumentation upgrades become necessary.

### Trade-offs

- Trend-based abandonment thresholding is less crisp than denominator-based rates.
- Citation-based promotion is a proxy and may undercount implicit lineage.
- Current historical comparability can drift when archives are pruned or restructured.

### Non-goals

- Defining intervention automation policies (alerts, auto-remediation).
- Designing a single composite score.
- Solving attempt-ID lineage gaps in this decision.

---

## Operationalization

Review cadence: weekly rolling window, plus 30-day view for promotion throughput.

Minimum reporting payload each cycle:
- Completion reliability (% and denominator)
- Time-to-value (p50/p90, sample size)
- Abandonment volume + repeat share + reason completeness
- Investigation promotion throughput (with denominator)

Escalation trigger:
- Any Red status for two consecutive weekly windows, or
- Two Watch statuses persisting for four consecutive weekly windows.

---

## References

- `.kb/investigations/2026-02-08-inv-outcome-data-inventory-data-exists.md`
- `.kb/models/dashboard-agent-status/probes/2026-02-09-system-success-metrics-baseline-snapshot.md`
- `.kb/decisions/2026-01-14-separate-observation-from-intervention.md`
- `.kb/decisions/2026-01-08-observation-infrastructure-principle.md`
