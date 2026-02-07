# Decision: Daemon Sort Strategies via pkg/daemon/sort

**Date:** 2026-02-06
**Status:** Accepted
**Deciders:** Dylan, Architect Agent

## Context

Investigation identified 5 named sort strategies for bead prioritization (Unblock Mode, Flow State Mode, Firefight Mode, Debt Reduction Mode, Decision Harvest Mode). Key design questions:
1. Where does sort logic live - bd CLI, orch daemon, or shared library?
2. Named presets vs composable sort flags?
3. How to handle sparse metadata (only 33% area label coverage)?
4. How to integrate with decidability graph leverage calculations?

## Decision

**Sort logic lives in orch daemon as named presets implemented in `pkg/daemon/sort/`.**

Key choices:
- **Named presets only** (not composable flags) - strategies have conflicting optimization targets
- **Go code with config-selected params** - pipelines call Go packages; YAML DSL would be painful
- **Graceful degradation** - skip missing dimensions, assign neutral scores
- **Cached frontier per poll cycle** - 60s staleness acceptable for batch daemon
- **Require-going-forward** - don't backfill; new issues should have area labels

## Rationale

### Why orch daemon (not bd CLI)

The 5 strategies require cross-system data:
- `dependency_leverage` - from frontier package
- `context_locality` - from active sessions + beads labels
- `authority_level` - from decidability graph

bd CLI only has beads data. orch daemon already bridges beads, frontier, and session state.

### Why named presets (not composable)

The strategies optimize for different things:
- Unblock Mode: maximize throughput by clearing bottlenecks
- Flow State Mode: minimize context switching
- Firefight Mode: triage by urgency
- Debt Reduction Mode: clean up old work
- Decision Harvest Mode: resolve blocking questions

Composing "`--sort unblock,flow`" creates conflicting priorities. Named presets give clear mental model.

### Why graceful degradation

With 33% area label coverage and no verification_cost data, sort pipelines must work with available data. Issues without area labels sort by priority + leverage + staleness.

## Consequences

### Positive
- Clear operational modes ("system is in unblock mode")
- Reuses existing frontier leverage calculation
- Testable Go functions with well-defined inputs
- Works today with sparse metadata

### Negative
- bd CLI's `--sort` flag won't gain new strategies (acceptable - different audiences)
- No composability (observe if needed)
- New strategies require code change (strategies change rarely)

## Implementation

1. Create `pkg/daemon/sort/` with `Strategy` interface
2. Implement `Priority` (current behavior) + `Unblock` (highest value)
3. Add frontier cache to daemon poll loop
4. Wire into `NextIssueExcluding()`
5. Add `--sort-mode` CLI flag
6. Implement remaining strategies (Flow State, Firefight, Debt Reduction, Decision Harvest)

## References

- Investigation: `.kb/investigations/2026-02-06-inv-design-bead-prioritization-sorting-system.md`
- Related: `.kb/decisions/2026-02-03-attention-model-single-priority.md` (daemon/attention separation)
- Model: `.kb/models/decidability-graph.md` (leverage concepts)
