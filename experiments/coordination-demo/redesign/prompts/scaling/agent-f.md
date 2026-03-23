# Task: Add Lerp to pkg/scaling

## Instructions

Add a `Lerp` function to `pkg/scaling/scaling.go` and write comprehensive tests in `pkg/scaling/scaling_test.go`.

### Requirements

1. **Function signature:** `func Lerp(a, b, t float64) float64`
2. **Behavior:**
   - Perform linear interpolation between a and b using parameter t
   - Formula: a + t*(b-a)
   - t=0 returns a, t=1 returns b, t=0.5 returns midpoint
   - t is NOT clamped: values outside [0,1] extrapolate
3. **Examples:**
   - `Lerp(0, 10, 0.5)` -> `5.0`
   - `Lerp(0, 10, 0)` -> `0.0`
   - `Lerp(0, 10, 1)` -> `10.0`
   - `Lerp(0, 10, 0.25)` -> `2.5`
   - `Lerp(10, 20, 2.0)` -> `30.0` (extrapolation)
   - `Lerp(-5, 5, 0.5)` -> `0.0`
4. **Tests:** Add `TestLerp` to `pkg/scaling/scaling_test.go` covering:
   - t=0, t=1, t=0.5
   - Extrapolation (t>1, t<0)
   - Negative ranges
   - Same a and b values

### Constraints

- Do NOT modify any existing functions
- Do NOT add new dependencies beyond what's already imported
- Follow existing code style (see existing functions for patterns)

### Verification

After implementing, run:
```bash
go test ./pkg/scaling/ -v -run TestLerp
```

Commit your changes when tests pass.
