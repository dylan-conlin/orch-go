## Summary (D.E.K.N.)

**Delta:** Added benchmark report generation with per-model pass rates, compliance signals, configurable threshold verdicts, and run metadata to the bench package.

**Evidence:** 32 tests passing across config, engine, and report modules; build succeeds.

**Knowledge:** Compliance signals (first-pass rate, rework recovery, stall rate, error rate) are derivable from existing TrialResult data without additional data collection.

**Next:** Close — implementation complete, ready for integration testing with a real benchmark run.

**Authority:** implementation - Extends existing bench package within established patterns.

---

# Investigation: Add Benchmark Result Artifacts Summary

**Question:** How should benchmark result artifacts be structured to enable per-model comparison, protocol compliance assessment, and threshold-based verdicts?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** og-research-add-benchmark-result-26mar-4a88
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Findings

### Finding 1: Existing bench package has clean extension points

**Evidence:** The existing `RunResult` and `RunSummary` types aggregate across all trials without model or scenario breakdown. The `TrialResult` struct tracks scenario name but not model, making per-model analysis impossible.

**Source:** `pkg/bench/results.go:12-23`, `pkg/bench/engine.go:78-84`

**Significance:** Adding `Model` field to `TrialResult` and propagating it from `Scenario.Model` in the engine is the minimal change needed to enable per-model reporting.

### Finding 2: Protocol compliance is derivable from trial data

**Evidence:** Five useful compliance signals can be computed from existing TrialResult fields: first-pass rate (pass && reworks==0), rework recovery rate (pass after reworks>0), stall rate (timeouts/total), error rate (errors/total), rework rate (reworked/total).

**Source:** Analysis of `TrialResult` fields in `pkg/bench/results.go`

**Significance:** No additional data collection (e.g., checking SYNTHESIS.md existence or beads comments) is needed for meaningful compliance metrics.

### Finding 3: Threshold defaults align with observed rates

**Evidence:** Non-Anthropic models show 67-87% stall rates (from CLAUDE.md gotchas). Default thresholds set at: pass_rate=0.8, max_error_rate=0.1, max_rework_rate=0.5.

**Source:** Project CLAUDE.md "Non-Anthropic models" gotcha

**Significance:** Defaults are tuned for Anthropic models; non-Anthropic runs will naturally produce FAIL verdicts, which is the desired behavior for flagging unreliable models.

## Structured Uncertainty

**What's tested:**
- ✅ Per-model summaries correctly group and aggregate (TestModelSummaries)
- ✅ Per-scenario summaries correctly group and aggregate (TestScenarioSummaries)
- ✅ Compliance signals compute correct rates (TestComplianceSignals)
- ✅ Verdict evaluation produces PASS/FAIL/WARN correctly (4 verdict tests)
- ✅ Report generation assembles all components (TestGenerateReport)
- ✅ JSON round-trip works (TestWriteReport)
- ✅ Config snapshot is reproducible (TestWriteConfigSnapshot)
- ✅ Threshold YAML parsing and defaults work (2 config tests)
- ✅ Build succeeds with updated bench_cmd.go

**What's untested:**
- ⚠️ End-to-end with real agent spawn (requires live benchmark run)
- ⚠️ Report formatting with very long scenario/model names
- ⚠️ Behavior when all trials have the same model (single-model breakdown)

**What would change this:**
- If richer protocol compliance is needed (SYNTHESIS.md existence, phase reporting), would need a ComplianceFunc callback on the engine
- If report comparison across runs is needed, would need a `bench compare` subcommand

## References

**Files Created:**
- `pkg/bench/report.go` - Report types, generation, formatting, writing
- `pkg/bench/report_test.go` - 15 tests for report functionality

**Files Modified:**
- `pkg/bench/config.go` - Added Thresholds struct and defaults
- `pkg/bench/config_test.go` - Added threshold parsing tests
- `pkg/bench/results.go` - Added Model field to TrialResult
- `pkg/bench/engine.go` - Propagated model from scenario to trial result
- `cmd/orch/bench_cmd.go` - Wired report generation, git metadata, config snapshot
