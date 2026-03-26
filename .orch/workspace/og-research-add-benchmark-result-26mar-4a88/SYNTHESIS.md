# Session Synthesis

**Agent:** og-research-add-benchmark-result-26mar-4a88
**Issue:** orch-go-71mnt
**Outcome:** success

---

## Plain-Language Summary

Added benchmark report generation to the `pkg/bench` package. After a benchmark run completes, the system now writes a `report.json` alongside existing `results.jsonl` and `summary.json`. The report breaks down pass rates by model (e.g., opus vs sonnet) and by scenario, computes protocol-compliance signals (first-pass rate, rework recovery, stall rate, error rate), and evaluates configurable thresholds to produce a PASS/FAIL/WARN verdict. A config snapshot (`config.yaml`) is also saved so runs can be compared later without re-reading raw workspaces. The human-readable output now prints all of this to the terminal.

## Verification Contract

See `VERIFICATION_SPEC.yaml` — 32 tests pass, build succeeds.

---

## TLDR

Added per-model pass rates, compliance signals, and threshold-based verdicts to the benchmark runner. Each run now writes `report.json` + `config.yaml` alongside existing artifacts, with enough metadata (git SHA, branch, config snapshot) to compare runs later.

---

## Delta (What Changed)

### Files Created
- `pkg/bench/report.go` - Report types, generation, compliance computation, verdict evaluation, JSON/text output
- `pkg/bench/report_test.go` - 15 tests covering model summaries, scenario summaries, compliance, verdicts, I/O, config snapshots

### Files Modified
- `pkg/bench/config.go` - Added `Thresholds` struct (pass_rate, max_error_rate, max_rework_rate) with defaults (0.8, 0.1, 0.5)
- `pkg/bench/config_test.go` - Added threshold parsing and default tests
- `pkg/bench/results.go` - Added `Model` field to `TrialResult`
- `pkg/bench/engine.go` - Propagated model from scenario to trial result
- `cmd/orch/bench_cmd.go` - Wired report generation, git metadata collection, config snapshot writing, artifact listing

---

## Evidence (What Was Observed)

- Existing bench package had clean extension points — `TrialResult` just needed a `Model` field
- Compliance signals are fully derivable from existing trial data without new collection mechanisms
- Threshold defaults (0.8 pass rate, 0.1 error rate, 0.5 rework rate) align with observed Anthropic model reliability

### Tests Run
```bash
go test ./pkg/bench/... -v -count=1
# 32 tests, all PASS (0.49s)

go build ./cmd/orch/...
# exit 0
```

---

## Architectural Choices

### Derive compliance signals from trial data vs. collect from beads/workspace
- **What I chose:** Derive from existing TrialResult fields
- **What I rejected:** Adding ComplianceFunc callback to check SYNTHESIS.md existence, beads comments
- **Why:** Keeps the engine simple and fast; the five derived signals (first-pass, rework recovery, stall, error, rework rates) capture the most actionable information
- **Risk accepted:** Can't detect protocol violations that don't affect eval outcome (e.g., missing SYNTHESIS.md that still passes tests)

### Threshold as warn vs fail for rework rate
- **What I chose:** Rework rate exceeding threshold produces WARN, not FAIL
- **What I rejected:** Making all thresholds binary PASS/FAIL
- **Why:** High rework rate indicates inefficiency but not broken behavior; FAIL should be reserved for unreliable output (low pass rate) or infrastructure issues (high error rate)
- **Risk accepted:** Users might want stricter rework enforcement; can be adjusted per-config

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete
- [x] Tests passing (32/32)
- [x] Investigation file has Status: Complete
- [x] Ready for `orch complete orch-go-71mnt`

---

## Unexplored Questions

- **Cross-run comparison:** A `bench compare` subcommand could diff two report.json files to show regression/improvement. Not in scope but natural next step.
- **Richer compliance:** Checking SYNTHESIS.md existence, phase reporting, or beads comment patterns would give deeper protocol insight but requires workspace path resolution.

---

## Friction

No friction — smooth session.

---

## Session Metadata

**Skill:** research
**Workspace:** `.orch/workspace/og-research-add-benchmark-result-26mar-4a88/`
**Investigation:** `.kb/investigations/2026-03-26-inv-add-benchmark-result-artifacts-summary.md`
**Beads:** `bd show orch-go-71mnt`
