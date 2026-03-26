## Summary (D.E.K.N.)

**Delta:** Implemented `orch bench` CLI surface with model alias resolution, suite config loading, `validate`/`list` subcommands, and enhanced dry-run output.

**Evidence:** 39 tests pass across pkg/bench (config, engine, report); CLI commands `bench run --dry-run`, `bench validate`, `bench list` all produce correct output with resolved model aliases.

**Knowledge:** The existing `pkg/model.Resolve()` cleanly integrates into benchmark config parsing — no new alias infrastructure needed, just wiring.

**Next:** Close. The execution engine already works; this surfaces it properly for CLI use.

**Authority:** implementation - Adding CLI surface and validation within existing patterns, no cross-boundary impact.

---

# Investigation: Implement Orch Benchmark CLI Surface

**Question:** How should the benchmark CLI expose suite config loading, model resolution, and validation?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel implementation | - | - | - |

---

## Findings

### Finding 1: Existing bench infrastructure was feature-complete but missing model resolution

**Evidence:** `pkg/bench/config.go` parsed YAML and validated fields but passed raw model strings (e.g., "opus") through to spawn without resolving them. `pkg/model/model.go` had full alias resolution but wasn't wired into bench.

**Source:** `pkg/bench/config.go:30-113`, `pkg/model/model.go:104-216`

**Significance:** The fix was pure wiring — add `ResolveModels()` that calls `model.Resolve()` per scenario, plus a `DefaultModel` field for suite-level fallback.

### Finding 2: Validation gaps — no duplicate detection or timeout format checking

**Evidence:** Original `validate()` checked for required fields but allowed duplicate scenario names and invalid timeout strings (e.g., "not-a-duration").

**Source:** `pkg/bench/config.go:85-113` (before changes)

**Significance:** Added duplicate name detection via `seen` map and `time.ParseDuration` validation for timeout fields. Both caught by new test cases.

### Finding 3: Dry-run output didn't show resolution chain

**Evidence:** Original dry-run printed raw model strings without showing what they resolve to, making it impossible to verify alias resolution before a live run.

**Source:** `cmd/orch/bench_cmd.go:77-84` (before changes)

**Significance:** New dry-run shows `model: opus (suite default) → anthropic/claude-opus-4-5-20251101`, giving full visibility into resolution chain.

---

## Structured Uncertainty

**What's tested:**

- ✅ Model alias resolution (opus, sonnet, haiku, gpt-5 → full model IDs) — verified via TestResolveModels_Aliases, TestResolveModels_GPTAliases
- ✅ DefaultModel fallback (suite-level → scenario-level) — verified via TestResolveModels_DefaultModel
- ✅ CLI --model override (replaces all scenario models) — verified via TestApplyModelOverride
- ✅ Duplicate scenario name detection — verified via TestParseConfig_ValidationErrors/duplicate_scenario_names
- ✅ Invalid timeout rejection — verified via TestParseConfig_ValidationErrors/invalid_timeout
- ✅ ListSuites discovery (valid + invalid configs, non-YAML ignored) — verified via TestListSuites

**What's untested:**

- ⚠️ Live benchmark execution with resolved models (requires running agents)
- ⚠️ Config-based model aliases (ResolveWithConfig) — only built-in aliases tested

**What would change this:**

- If spawn's `--model` flag doesn't accept full `provider/model` format, the resolved model format would need adjustment
- If new model aliases are added to pkg/model, bench resolution picks them up automatically

---

## References

**Files Examined:**
- `pkg/bench/config.go` — Config parsing, validation, model resolution
- `pkg/bench/engine.go` — Benchmark execution engine
- `pkg/bench/report.go` — Report generation and formatting
- `pkg/model/model.go` — Model alias resolution
- `cmd/orch/bench_cmd.go` — CLI command surface

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch/

# Test suite
go test ./pkg/bench/ -v -count=1

# CLI verification
go run ./cmd/orch bench validate benchmarks/worker-reliability.yaml
go run ./cmd/orch bench list benchmarks/
go run ./cmd/orch bench run benchmarks/worker-reliability.yaml --dry-run
go run ./cmd/orch bench run benchmarks/worker-reliability.yaml --dry-run --model gpt-5
```
