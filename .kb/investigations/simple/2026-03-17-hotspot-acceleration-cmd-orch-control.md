---
title: "Hotspot acceleration: cmd/orch/control_cmd.go"
status: Complete
date: 2026-03-17
beads: orch-go-ru9ux
---

## TLDR

`control_cmd.go` hotspot alert (+292 lines/30d) is a **false positive** caused by delete/recreate churn (circuit breaker experimentation), not dangerous growth. File is 211 lines with logic properly extracted to `pkg/control/`. No action needed.

## D.E.K.N. Summary

- **Delta:** Investigated hotspot acceleration flag for `cmd/orch/control_cmd.go`
- **Evidence:** Git history shows file deleted (Feb 17, -302 lines) and recreated (Mar 1, +157 lines) — churn inflates the insertions metric
- **Knowledge:** Hotspot metric counts total insertions, not net growth; delete/recreate cycles produce false positives
- **Next:** Close. Consider adding churn-type classification to hotspot metric (distinguishing net growth from delete/recreate cycles)

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| N/A - novel investigation | - | - | - |

## Question

Is `cmd/orch/control_cmd.go` (+292 lines/30d, now 211 lines) at risk of becoming a critical hotspot requiring extraction?

## Findings

### Finding 1: Git history reveals delete/recreate cycle

Full commit history for `cmd/orch/control_cmd.go`:

| Date | Commit | Action | +lines | -lines | File size after |
|------|--------|--------|--------|--------|-----------------|
| Feb 14 | `44c0636` | Created (circuit breaker + init/status/resume) | 222 | 0 | 222 |
| Feb 14 | `764bb73` | Architect redesign (3-layer heuristics) | 105 | 25 | ~302 |
| Feb 17 | `617d7fc` | **Deleted entirely** (superseded by verification-first) | 0 | 302 | 0 |
| Mar 1 | `9cfa38f` | **Recreated** (lock/unlock/status only) | 157 | 0 | 157 |
| Mar 1 | `d8efb69` | Re-enabled circuit breaker | 79 | 2 | ~234 |
| Mar 1 | `dcc8585` | Removed circuit breaker again | 3 | 79 | ~158 |
| Mar 8 | `25cca27` | Added deny rules audit | 53 | 0 | 211 |

**Total insertions:** 619 lines. **Total deletions:** 408 lines. **Net:** 211 lines (current).

The "+292 lines/30d" metric likely counts insertions within a rolling window. The high number comes from the file being deleted and recreated during circuit breaker experimentation, not from monotonic growth.

### Finding 2: Architecture is already properly layered

- `cmd/orch/control_cmd.go` (211 lines) — Thin CLI shell: 4 Cobra command definitions + output formatting
- `pkg/control/control.go` (266 lines) — All business logic: `DiscoverControlPlaneFiles()`, `Lock()`, `Unlock()`, `FileStatus()`, `DenyRules()`
- `pkg/control/control_test.go` (673 lines) — Comprehensive test coverage

This follows the project's established pattern: `cmd/orch/*.go` files are thin shells that call into `pkg/` packages. The command file contains only what belongs in a command file — argument parsing, calling pkg functions, formatting output.

### Finding 3: No growth trajectory concern

At 211 lines, this file is:
- **14%** of the 1,500-line accretion boundary
- **26%** of the 800-line "warn on additions" threshold
- Stable since Mar 8 (no commits in 9 days)
- Circuit breaker experimentation is complete (removed twice, unlikely to return)

## Test Performed

```bash
# Verified current file size
wc -l cmd/orch/control_cmd.go
# 211 cmd/orch/control_cmd.go

# Verified pkg/control separation
wc -l pkg/control/*.go
# 673 pkg/control/control_test.go
# 266 pkg/control/control.go
# 939 total

# Verified churn pattern via git log
git log --stat --follow -- cmd/orch/control_cmd.go
# Shows delete at 617d7fc25 (-302) and recreate at 9cfa38f71 (+157)
```

## Conclusion

**False positive.** The hotspot acceleration metric was triggered by design churn (circuit breaker added → removed → file deleted → file recreated → circuit breaker re-added → removed again), not by actual file bloat. Current file is 211 lines with a clean separation of concerns. No extraction needed.

## Recommendation

The hotspot acceleration metric should ideally distinguish between:
1. **Monotonic growth** (dangerous — file keeps getting bigger) → action needed
2. **Design churn** (benign — file was rewritten/restructured) → informational only

A simple heuristic: if current file size is <400 lines and the file was deleted/recreated within the measurement window, classify as churn rather than acceleration.
