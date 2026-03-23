# Task: Add RoundTo to pkg/scaling

## Instructions

Add a `RoundTo` function to `pkg/scaling/scaling.go` and write comprehensive tests in `pkg/scaling/scaling_test.go`.

### Requirements

1. **Function signature:** `func RoundTo(v float64, places int) float64`
2. **Behavior:**
   - Round a float64 to the specified number of decimal places
   - Use standard rounding (0.5 rounds up)
   - Handle zero and negative places (places=0 rounds to integer)
   - Handle negative values
3. **Examples:**
   - `RoundTo(3.14159, 2)` -> `3.14`
   - `RoundTo(3.145, 2)` -> `3.15`
   - `RoundTo(3.14159, 0)` -> `3.0`
   - `RoundTo(-2.567, 1)` -> `-2.6`
   - `RoundTo(100.0, 3)` -> `100.0`
4. **Tests:** Add `TestRoundTo` to `pkg/scaling/scaling_test.go` covering:
   - Various decimal places (0, 1, 2, 3)
   - Rounding up and down
   - Negative values
   - Already-rounded values

### Constraints

- Do NOT modify any existing functions
- Do NOT add new dependencies beyond what's already imported
- You may add `math` to imports if needed
- Follow existing code style (see existing functions for patterns)

### Verification

After implementing, run:
```bash
go test ./pkg/scaling/ -v -run TestRoundTo
```

Commit your changes when tests pass.
