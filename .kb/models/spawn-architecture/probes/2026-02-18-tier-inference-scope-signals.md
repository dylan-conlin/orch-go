# Probe: Tier Inference Scope Signals

**Model:** spawn-architecture
**Date:** 2026-02-18
**Status:** Complete

---

## Question

Does tier inference upgrade light-tier defaults when the task text signals medium scope (session scope, new package/module, or test requirements)?

---

## What I Tested

- `go run /tmp/tier_repro.go` (before changes: DetermineSpawnTier called without task)
- `go run /tmp/tier_repro.go` (after changes: DetermineSpawnTier with task scope signal)
- `go test ./pkg/orch -run TestDetermineSpawnTier_TaskScopeSignals`

---

## What I Observed

- Before changes, tier inference returned `light` for a task describing a new package with tests.
- After changes, the same task string returns `full` when scope signals are present.
- New unit tests passed for session scope, new package+tests, and default light behavior.

---

## Model Impact

- [x] **Extends**: Tier inference now considers task scope signals (session scope and new package/test indicators) before falling back to skill defaults.
