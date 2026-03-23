# Task: Add Abs to pkg/scaling

## Instructions

Add an `Abs` function to `pkg/scaling/scaling.go` and write comprehensive tests in `pkg/scaling/scaling_test.go`.

### Requirements

1. **Function signature:** `func Abs(v float64) float64`
2. **Behavior:**
   - Return the absolute value of a float64
   - Handle positive, negative, and zero values
   - Handle special values: math.Inf, math.NaN
3. **Examples:**
   - `Abs(5.0)` -> `5.0`
   - `Abs(-5.0)` -> `5.0`
   - `Abs(0)` -> `0`
   - `Abs(-0.001)` -> `0.001`
4. **Tests:** Add `TestAbs` to `pkg/scaling/scaling_test.go` covering:
   - Positive, negative, zero
   - Small values near zero
   - Large values

### Constraints

- Do NOT modify any existing functions
- Do NOT add new dependencies beyond what's already imported
- You may add `math` to imports if needed
- Follow existing code style (see existing functions for patterns)

### Verification

After implementing, run:
```bash
go test ./pkg/scaling/ -v -run TestAbs
```

Commit your changes when tests pass.
