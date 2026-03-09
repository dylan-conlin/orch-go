# Coordination Failure Controlled Demo — Pilot Results

**Date:** 2026-03-09
**Baseline commit:** bde5d58982
**Task:** Add `FormatBytes(bytes int64) string` to `pkg/display/display.go` with tests

## Scoring Summary

| Metric | Haiku | Opus |
|--------|-------|------|
| F0: Completion | 1/1 | 1/1 |
| F1: Compilation | 1/1 | 1/1 |
| F2: New Tests Pass | 1/1 | 1/1 |
| F3: No Regression | 1/1 | 1/1 |
| F4: File Discipline | 1/1 | 1/1 |
| F5: Spec Match | 1/1 | 1/1 |
| **Total** | **6/6** | **6/6** |
| **Duration** | **49s** | **63s** |

## Coordination Failure

**Merge result: CONFLICT (2 files)**

Both agents modified the same insertion points:
- `pkg/display/display.go` line 95 — both appended FormatBytes
- `pkg/display/display_test.go` line 135 — both appended TestFormatBytes

Post-merge: compilation fails (merge conflict markers)

## Implementation Comparison

### Code Structure

| Aspect | Haiku | Opus |
|--------|-------|------|
| Approach | Loop-based (unit array iteration) | Switch-based (const block + switch/case) |
| Lines added | 41 | 40 |
| Style | Verbose names (isNegative, unitIndex) | Idiomatic Go (const, switch, concise naming) |
| Comment style | More verbose docstring | Concise 2-line docstring |

### Test Quality

| Aspect | Haiku | Opus |
|--------|-------|------|
| Test cases | 34 | 24 |
| Negative value coverage | 10 cases | 4 cases |
| Boundary coverage | All 4 boundaries + midpoints | All 4 boundaries + midpoints |
| Fractional values | Standard halves (1.5x) | Includes non-standard (1126 → 1.1 KiB) |
| Duplicate cases | Yes (1536 repeated) | None |

### Commit Messages

Both produced IDENTICAL commit messages:
```
feat: add FormatBytes function for human-readable byte formatting
```

This is independent convergence — neither agent saw the other's work.

## Key Finding

**Both models achieved 6/6 on individual performance, but 100% coordination failure on merge.**

The coordination failure is NOT a model capability issue — it's a structural issue. When two agents independently implement the same feature:
1. They both append to the same file locations
2. Git cannot auto-merge identical insertions at the same position
3. The merge produces conflict markers that break compilation

**Implication:** Coordination failures on shared codebases are dominated by structural factors (where code is inserted), not model quality. Even perfect agents will conflict when they modify the same insertion points without coordination protocol.
