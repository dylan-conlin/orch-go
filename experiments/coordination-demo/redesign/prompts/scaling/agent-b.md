# Task: Add PadLeft to pkg/scaling

## Instructions

Add a `PadLeft` function to `pkg/scaling/scaling.go` and write comprehensive tests in `pkg/scaling/scaling_test.go`.

### Requirements

1. **Function signature:** `func PadLeft(s string, width int, pad byte) string`
2. **Behavior:**
   - Left-pad the string with the pad character until it reaches the specified width
   - If the string is already at or exceeds width, return it unchanged
   - Handle empty strings: pad from nothing
3. **Examples:**
   - `PadLeft("42", 5, '0')` -> `"00042"`
   - `PadLeft("hello", 10, ' ')` -> `"     hello"`
   - `PadLeft("long string", 5, ' ')` -> `"long string"`
   - `PadLeft("", 3, 'x')` -> `"xxx"`
4. **Tests:** Add `TestPadLeft` to `pkg/scaling/scaling_test.go` covering:
   - Normal padding, no padding needed
   - Empty string padding
   - Different pad characters
   - Exact width match

### Constraints

- Do NOT modify any existing functions
- Do NOT add new dependencies beyond what's already imported
- Follow existing code style (see existing functions for patterns)

### Verification

After implementing, run:
```bash
go test ./pkg/scaling/ -v -run TestPadLeft
```

Commit your changes when tests pass.
