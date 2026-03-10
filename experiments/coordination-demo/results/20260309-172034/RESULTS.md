# Coordination Demo N=10 Results

**Date:** 2026-03-09
**Baseline:** 22073e5e7
**Models:** claude-haiku-4-5-20251001, claude-opus-4-5-20251101
**Task:** FormatBytes(bytes int64) string
**Trials:** 10 per model (20 total agent runs)

## Individual Performance

Both models scored 6/6 (adjusted) in all 10 trials. F4 (file discipline) scored 0 due to beads side-effect (.beads/issues.jsonl modified by environment, not agent).

| Dimension | Haiku (N=10) | Opus (N=10) |
|-----------|-------------|-------------|
| F0: Completion | 10/10 | 10/10 |
| F1: Compilation | 10/10 | 10/10 |
| F2: New tests pass | 10/10 | 10/10 |
| F3: No regression | 10/10 | 10/10 |
| F5: Spec match | 10/10 | 10/10 |

## Duration

| Model | Mean | SD | Min | Max | Durations |
|-------|------|----|-----|-----|-----------|
| Haiku | 39.1s | 13.4s | 31s | 76s | 35, 31, 32, 76, 42, 39, 31, 33, 36, 36 |
| Opus | 44.0s | 4.2s | 38s | 53s | 42, 41, 38, 47, 42, 44, 48, 42, 53, 43 |

Welch's t-test: t=1.103 (not significant at p<0.05). Duration difference is not statistically significant at N=10.

Note: Haiku trial 4 was an outlier (76s), inflating SD. Without it, haiku mean=35.0s, SD=3.6s.

## Merge Conflict Analysis

**Conflict rate: 10/10 = 100%**

| Trial | Conflict files | Notes |
|-------|---------------|-------|
| 1 | 1 | display.go only |
| 2 | 2 | both display.go and display_test.go |
| 3 | 1 | display.go only |
| 4 | 2 | both |
| 5 | 1 | display.go only |
| 6 | 2 | both |
| 7 | 2 | both |
| 8 | 1 | display.go only |
| 9 | 2 | both |
| 10 | 1 | display.go only |

5/10 trials had 1-file conflicts (display.go only — test files auto-merged when agents placed tests at different positions). 5/10 trials had 2-file conflicts (both files conflicted).

## Statistical Conclusion

- Fisher's exact test for coordination failure rate: p=1.0 (identical rates)
- Both models: 100% individual success, 100% merge conflict
- Coordination failure is independent of model capability (confirmed at N=10)
- Duration difference not significant (Haiku marginally faster but within noise)
