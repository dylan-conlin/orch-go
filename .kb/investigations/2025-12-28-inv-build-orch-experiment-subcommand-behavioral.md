## Summary (D.E.K.N.)

**Delta:** Implemented `orch experiment` subcommand for running behavioral experiments on agent context/attention patterns.

**Evidence:** All 5 subcommands implemented (create, list, run, status, analyze), tests pass, code compiles.

**Knowledge:** YAML schema is extensible - users can add custom fields beyond the template defaults (e.g., task, known_answer, success_criteria as seen in existing context-attention experiment).

**Next:** Close - implementation complete. Use `orch experiment` to run first experiment on context-attention hypothesis.

---

# Investigation: Build Orch Experiment Subcommand Behavioral

**Question:** How to structure the `orch experiment` CLI for running systematic behavioral experiments on agent context/attention?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** Worker agent (og-feat-build-orch-experiment-28dec)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Implementation Summary

Created CLI commands and supporting package:

### Files Created

1. **cmd/orch/experiment.go** (509 lines) - CLI commands using cobra
   - `orch experiment create <name>` - scaffolds experiment directory
   - `orch experiment list` - shows all experiments with status
   - `orch experiment run <exp> --condition A` - records a run
   - `orch experiment status <exp>` - shows progress per condition
   - `orch experiment analyze <exp>` - generates analysis.md

2. **pkg/experiment/experiment.go** (433 lines) - types and utilities
   - `Experiment`, `Condition`, `Run`, `RunMetrics` types
   - File I/O: `LoadExperiment`, `Save`, `LoadRuns`, `SaveRun`
   - Analysis: `GenerateAnalysis`, `RunsByCondition`
   - Helpers: `ListExperiments`, `ExperimentDir`, `GetNextRunNumber`

3. **pkg/experiment/experiment_test.go** (330 lines) - comprehensive tests
   - All 7 tests pass

### Directory Structure

```
.orch/experiments/<date>-<name>/
  experiment.yaml   - Configuration (conditions, metrics, hypothesis)
  runs/             - Run data files
    <condition>-<n>.json  - Individual run results
  analysis.md       - Generated analysis document
```

### Key Design Decisions

1. **Manual metric collection for v1** - Per SPAWN_CONTEXT scope, metrics are recorded manually after observing agents. Future versions may integrate with session introspection.

2. **Extensible YAML schema** - Template provides defaults but users can add custom fields (e.g., `task`, `known_answer`, `expected`, `success_criteria`).

3. **Partial name matching** - Commands accept partial experiment names and resolve to full directory names.

---

## References

**Commit:** e152cc81 - "feat(experiment): add orch experiment subcommand for behavioral experiments"

**Test Output:**
```
=== RUN   TestCreateExperiment
--- PASS: TestCreateExperiment (0.00s)
=== RUN   TestSaveAndLoadRun
--- PASS: TestSaveAndLoadRun (0.00s)
=== RUN   TestGetNextRunNumber
--- PASS: TestGetNextRunNumber (0.00s)
=== RUN   TestRunsByCondition
--- PASS: TestRunsByCondition (0.00s)
=== RUN   TestListExperiments
--- PASS: TestListExperiments (0.00s)
=== RUN   TestGenerateAnalysis
--- PASS: TestGenerateAnalysis (0.00s)
=== RUN   TestRunMetricsSerialization
--- PASS: TestRunMetricsSerialization (0.00s)
PASS
```

**Related:**
- Existing experiment: `.orch/experiments/2025-12-28-context-attention/experiment.yaml`
- Beads issue: orch-go-9ec7
