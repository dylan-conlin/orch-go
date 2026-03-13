# Decision: Remove Health Score Spawn Gate

**Date:** 2026-03-11
**Status:** Accepted

## Context

The health score gate was introduced on Mar 10, 2026 to block feature-impl spawns when the harness health score fell below 65 (C grade). Within hours, it was downgraded from blocking to advisory after a Phase 4 probe found the score improvement (37→73) was 89% calibration artifact.

The advisory gate printed "✓ Health score: 73 (C)" on every spawn — giving the appearance of enforcement while never actually firing. The formula was calibrated to produce passing scores for the current codebase (baseline values score 69.2 under the new formula with zero extractions).

## Decision

**Chosen:** Remove the health score gate from the spawn path entirely.

**Rationale:**
1. The gate never fires — formula calibrated to pass existing codebase state
2. Advisory-only gates that never trigger provide false assurance
3. Real enforcement comes from pre-commit accretion gate and hotspot blocking
4. Health score remains useful as diagnostic metric (`orch health`, `orch doctor`)
5. Principle: honesty over ceremony — remove the false signal rather than maintaining it

## What Changed

- Deleted `pkg/spawn/gates/health.go` and `pkg/spawn/gates/health_test.go`
- Removed `skipHealthGate` and `healthScoreProvider` parameters from `RunPreFlightChecks`
- Removed `--skip-health-gate` CLI flag
- Removed `buildHealthScoreProvider()` from spawn helpers

## What Was Preserved

- `pkg/health/health.go` — score calculation (diagnostic tool for `orch doctor`)
- `pkg/health/health_test.go` — tests for the calculation
- Health snapshot collection in daemon
- `orch health` / `orch doctor --health` commands

## Consequences

- Positive: Spawn path is more honest — fewer gates, but the remaining ones actually enforce
- Positive: Reduced spawn latency (no health snapshot reads on every spawn)
- Positive: Removed `--skip-health-gate` flag surface area
- Risk: If the health score formula is ever rebuilt with honest thresholds, the gate integration points need to be re-added

## References

- Supersedes: `2026-03-10-health-score-floor-gate-downgraded-from-blocking-t.md`
- Supersedes: `2026-03-10-health-score-targets-65-floor-gate-80-target.md` (floor gate portion)
- Evidence: `.kb/models/harness-engineering/probes/2026-03-10-probe-health-score-calibration-vs-structural-improvement.md`
- Evidence: `.kb/models/knowledge-accretion/probes/2026-03-10-probe-health-score-structural-improvement-validation.md`
