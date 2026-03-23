# Task: Add Repeat to pkg/scaling

## Instructions

Add a `Repeat` function to `pkg/scaling/scaling.go` and write comprehensive tests in `pkg/scaling/scaling_test.go`.

### Requirements

1. **Function signature:** `func Repeat(s string, n int, sep string) string`
2. **Behavior:**
   - Repeat string s exactly n times, joined by sep
   - If n <= 0, return ""
   - If n == 1, return s (no separator)
   - Handle empty strings and empty separators
3. **Examples:**
   - `Repeat("ha", 3, " ")` -> `"ha ha ha"`
   - `Repeat("ab", 2, "-")` -> `"ab-ab"`
   - `Repeat("x", 1, ",")` -> `"x"`
   - `Repeat("y", 0, ",")` -> `""`
   - `Repeat("z", 3, "")` -> `"zzz"`
4. **Tests:** Add `TestRepeat` to `pkg/scaling/scaling_test.go` covering:
   - Normal repetition with separator
   - Single repetition (no separator)
   - Zero or negative count
   - Empty string, empty separator

### Constraints

- Do NOT modify any existing functions
- Do NOT add new dependencies beyond what's already imported
- Follow existing code style (see existing functions for patterns)

### Verification

After implementing, run:
```bash
go test ./pkg/scaling/ -v -run TestRepeat
```

Commit your changes when tests pass.
