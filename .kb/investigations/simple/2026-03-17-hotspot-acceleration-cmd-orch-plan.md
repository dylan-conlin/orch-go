# Hotspot Acceleration: cmd/orch/plan_cmd.go

**TLDR:** False positive hotspot alert. The "+609 lines/30d" metric counts raw additions but ignores a 241-line extraction that already happened on Mar 13. File is at 368 lines — healthy, well-structured, no extraction needed.

**Status:** Complete
**Date:** 2026-03-17

## D.E.K.N. Summary

- **Delta:** plan_cmd.go's hotspot signal is a false positive driven by high churn (creation + extraction), not accumulation
- **Evidence:** Git history shows 562 lines added Mar 5, then 241 lines extracted to pkg/plan/ on Mar 13. Net: 368 lines — below 800-line warning threshold
- **Knowledge:** The hotspot metric tracks additions only, making files with create-then-extract patterns appear as hotspots. This is a measurement artifact, not a real risk.
- **Next:** No action needed on plan_cmd.go. Consider improving hotspot metric to use net growth (additions minus deletions) instead of raw additions.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| N/A — novel investigation | - | - | - |

## Question

Is cmd/orch/plan_cmd.go a genuine hotspot requiring extraction, or a false positive?

## Findings

### Finding 1: Git history reveals create-then-extract pattern

The file's 30-day commit history tells the full story:

| Date | Commit | Additions | Deletions | Net |
|---|---|---|---|---|
| Mar 5 | `036680d` — add orch plan CLI commands | +562 | -0 | +562 |
| Mar 13 | `02bb680` — plan lifecycle enforcement | +47 | -241 | -194 |
| **Total** | | **+609** | **-241** | **+368** |

The hotspot metric reports +609 lines (raw additions only), but the file actually **shrunk** in its most recent commit. The extraction of parsing/types to `pkg/plan/` already happened.

### Finding 2: Current structure is well-factored

Total plan-related code across 6 files:

| File | Lines | Responsibility |
|---|---|---|
| `cmd/orch/plan_cmd.go` | 368 | CLI commands + display formatting |
| `cmd/orch/plan_hydrate.go` | 250 | Hydrate command + file update |
| `cmd/orch/plan_cmd_test.go` | 387 | Tests for plan_cmd |
| `cmd/orch/plan_hydrate_test.go` | 189 | Tests for hydrate |
| `pkg/plan/plan.go` | 275 | Shared types + parsing |
| `pkg/plan/plan_test.go` | 206 | Tests for pkg/plan |

The extraction is already done:
- **Types** (`File`, `Phase`) → `pkg/plan/`
- **Parsing** (`ParseContent`, `ParseBeadsLine`, etc.) → `pkg/plan/`
- **Staleness detection** → `pkg/daemon/plan_staleness.go`
- **Hydration** → separated into `plan_hydrate.go`

What remains in `plan_cmd.go` is CLI wiring (Cobra commands) and display formatting — both tightly coupled to the CLI layer and appropriate for `cmd/orch/`.

### Finding 3: No further extraction opportunities

The remaining 368 lines in `plan_cmd.go` break down as:
- ~100 lines: Cobra command definitions (show, status, create)
- ~20 lines: beads query helper
- ~170 lines: display formatting (formatPlanShow, formatPlanStatus, icon helpers)
- ~50 lines: type aliases, function aliases, init()

The display formatting functions _could_ theoretically move to `pkg/plan/display.go`, but at 170 lines they don't justify a separate package. They depend on both `plan.File` types and `beads` status lookups — moving them would create circular dependencies or require interface abstractions that aren't warranted at this scale.

## Test Performed

Verified the git history and current file contents directly. The "+609 lines/30d" metric is arithmetically correct (summing additions across commits) but misleading — it doesn't account for the -241 lines deleted in the extraction commit.

## Conclusion

**False positive.** The file had high churn due to creation (Mar 5) followed by extraction (Mar 13), which is exactly the healthy pattern the hotspot system is trying to encourage. The file is at 368 lines, well below the 800-line advisory threshold. No extraction needed.

**Recommendation:** Improve the hotspot metric to use net growth (additions - deletions) instead of raw additions, to avoid flagging files that already underwent extraction.
