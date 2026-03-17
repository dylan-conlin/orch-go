---
Status: Complete
Question: Is cmd/orch/architecture_lint_test.go a hotspot requiring extraction?
Date: 2026-03-17
---

**TLDR:** architecture_lint_test.go is a new file (created Feb 19), not a growing one. The "+318 lines/30d" metric reflects file creation, not accretion. At 317 lines with 6 distinct lint tests, it's healthy and well below any threshold.

## D.E.K.N. Summary

- **Delta:** architecture_lint_test.go was created Feb 19 (182 lines), expanded Mar 8 (+135 lines for function size lint and package boundary tests), then a 1-line fix Mar 11. Total: 317 lines. The "+318 lines/30d" hotspot metric is the entire file — it's measuring creation, not growth.
- **Evidence:** `git log --numstat` shows 3 commits: 182+0 (creation), 135+0 (feature add), 1+1 (fix). Zero deletions except the 1-line fix. Net additions after creation: +136 lines in 26 days.
- **Knowledge:** Hotspot metrics that measure gross additions over a time window produce false positives for newly created files. The metric cannot distinguish "file created 318 lines" from "file grew by 318 lines." This is the same pattern as spawn_cmd.go (investigated same day) — gross-addition metrics mislead on recently-created/extracted files.
- **Next:** No action needed. Re-evaluate if file reaches 600 lines. If it does grow, the natural extraction boundary is by lint domain (lifecycle state, function size, package boundaries).

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
| --- | --- | --- | --- |
| .kb/investigations/2026-03-17-hotspot-acceleration-cmd-orch-spawn.md | Confirms | yes | - |

The spawn_cmd.go investigation identified the same false-positive pattern: gross-addition metrics mislead on recently-created/extracted files. This investigation confirms the pattern on a second file.

## Findings

### Finding 1: File is newly created, not accreting

The file was created from scratch on Feb 19, 2026 (commit dc6d350e8) with 182 lines implementing lifecycle state lint tests for the two-lane agent discovery architecture. The entire "+318 lines/30d" is file creation plus one feature addition.

Commit history:
- `dc6d350e8` (Feb 19): 182 lines — initial creation, lifecycle state lint
- `f7c01426c` (Mar 8): +135 lines — function size lint + package boundary tests
- `adc4360f5` (Mar 11): +1/-1 — known violation allowlist fix

Post-creation growth rate: 136 lines in 17 days = ~8 lines/day (this is "adding a new test function" pace, not "accreting complexity" pace).

### Finding 2: File structure is clean and decomposed by domain

The 317 lines contain 6 test functions plus 2 data variables and 1 helper:

| Function | Lines | Domain |
| --- | --- | --- |
| `TestArchitectureLint_NoNewLifecycleStatePackages` | 24 | Lifecycle state |
| `TestArchitectureLint_NoNewLifecycleStateInDiff` | 29 | Lifecycle state |
| `TestArchitectureLint_NoPersistentLifecycleFiles` | 32 | Lifecycle state |
| `TestArchitectureLint_ForbiddenPackageImports` | 28 | Lifecycle state |
| `TestArchitectureLint_FunctionSize` | 81 | Function size |
| `TestArchitectureLint_PackageBoundaries` | 28 | Package boundaries |
| `findProjectRoot` | 19 | Helper |

The largest function (`TestArchitectureLint_FunctionSize`) is 81 lines — well within normal test function size. No single function dominates.

### Finding 3: File is small relative to peers

At 317 lines, this file is in the 50th percentile of cmd/orch test files. For context:
- main_test.go: 1,961 lines
- stats_test.go: 1,891 lines
- hotspot_test.go: 1,692 lines
- complete_test.go: 852 lines
- architecture_lint_test.go: 317 lines (this file)

The file would need to 5x before reaching the 1,500-line critical threshold.

### Finding 4: Natural extraction boundaries exist if needed later

If the file does grow, it has clear extraction boundaries by lint domain:
- Lifecycle state tests (lines 48-165, ~118 lines) → `lifecycle_lint_test.go`
- Function size tests (lines 167-267, ~100 lines) → `function_size_lint_test.go`
- Package boundary tests (lines 269-296, ~28 lines) → could stay or merge

But at 317 lines, extraction would be premature — the cohesion benefit of having all architecture lints in one file outweighs the small size.

## Test Performed

```bash
# Verified current file size
wc -l cmd/orch/architecture_lint_test.go
# 317 lines

# Verified commit history and growth pattern
git log --format="%h %ad %s" --date=short --numstat --follow -- cmd/orch/architecture_lint_test.go
# 3 commits: creation (182), feature add (135), fix (1)

# Verified file was created within 30-day window (not pre-existing)
git log --diff-filter=A --format="%h %ad" --date=short -- cmd/orch/architecture_lint_test.go
# dc6d350e8 2026-02-19 (26 days ago)

# Verified relative size among peer test files
wc -l cmd/orch/*_test.go | sort -n | tail -10
# architecture_lint_test.go at 317 is 50th percentile
```

## Conclusion

**No extraction needed — false positive.** The "+318 lines/30d" hotspot metric flagged this file because it was *created* within the measurement window, not because it's accreting. At 317 lines with clean structure and 6 focused test functions, it's healthy. The file would need to 5x its current size to reach the critical threshold.

**Recommendation:** Re-evaluate if file reaches 600 lines. If growth accelerates, extract by lint domain (lifecycle state → its own file, function size → its own file).
