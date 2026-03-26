# Brief: orch-go-71mnt

## Frame

The benchmark runner could tell you whether a trial passed or failed, but it couldn't tell you *which model* was unreliable, or whether the overall run met a quality bar. You'd get a results.jsonl and a summary.json, and then you'd have to do the math yourself to figure out if opus was outperforming sonnet, or whether the error rate meant the infrastructure was flaky versus the model being bad.

## Resolution

The run directory now writes four artifacts instead of two: the existing `results.jsonl` and `summary.json`, plus a `report.json` (the full report with per-model breakdowns, compliance signals, and a PASS/FAIL/WARN verdict) and a `config.yaml` snapshot (so you can reproduce or compare later). The compliance signals turned out to be entirely derivable from existing trial data — first-pass rate, rework recovery, stall rate, error rate, rework rate — without needing to shell out to beads or check workspace files. Thresholds are configurable in the benchmark YAML (`thresholds.pass_rate`, `thresholds.max_error_rate`, `thresholds.max_rework_rate`) with sensible defaults that will flag non-Anthropic models as FAIL, which is what you'd want given the 67-87% stall rates documented in CLAUDE.md.

## Tension

Rework rate crossing the threshold produces a WARN, not a FAIL. I made that call because high rework rate means "slow but working" rather than "broken" — but there's a reasonable argument that if half your trials need rework, that's a FAIL-worthy signal for protocol compliance. Worth deciding whether the distinction matters for how you plan to use benchmark verdicts.
