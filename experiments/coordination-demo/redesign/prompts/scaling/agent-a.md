# Task: Add Capitalize to pkg/scaling

## Instructions

Add a `Capitalize` function to `pkg/scaling/scaling.go` and write comprehensive tests in `pkg/scaling/scaling_test.go`.

### Requirements

1. **Function signature:** `func Capitalize(s string) string`
2. **Behavior:**
   - Capitalize the first letter of each word in the string
   - A word is defined as a sequence of non-space characters separated by spaces
   - Preserve existing spacing between words
   - Handle empty strings: return ""
3. **Examples:**
   - `Capitalize("hello world")` -> `"Hello World"`
   - `Capitalize("already Capital")` -> `"Already Capital"`
   - `Capitalize("")` -> `""`
   - `Capitalize("a b c")` -> `"A B C"`
   - `Capitalize("  spaces  between  ")` -> `"  Spaces  Between  "`
4. **Tests:** Add `TestCapitalize` to `pkg/scaling/scaling_test.go` covering:
   - Normal strings, empty string
   - Already capitalized words
   - Multiple spaces between words
   - Single character words

### Constraints

- Do NOT modify any existing functions
- Do NOT add new dependencies beyond what's already imported
- You may add `unicode` to imports if needed
- Follow existing code style (see existing functions for patterns)

### Verification

After implementing, run:
```bash
go test ./pkg/scaling/ -v -run TestCapitalize
```

Commit your changes when tests pass.
