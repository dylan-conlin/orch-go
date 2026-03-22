# Task: Add FormatBytes to pkg/display

## Instructions

Add a `FormatBytes` function to `pkg/display/display.go` and write comprehensive tests in `pkg/display/display_test.go`.

### Requirements

1. **Function signature:** `func FormatBytes(bytes int64) string`
2. **Behavior:**
   - Format byte counts into human-readable strings
   - Use binary units: B, KiB, MiB, GiB, TiB
   - Show 1 decimal place for non-byte units (e.g., "1.5 MiB")
   - Handle negative values by prefixing with "-"
   - Handle zero: return "0 B"
3. **Examples:**
   - `FormatBytes(0)` → `"0 B"`
   - `FormatBytes(512)` → `"512 B"`
   - `FormatBytes(1024)` → `"1.0 KiB"`
   - `FormatBytes(1536)` → `"1.5 KiB"`
   - `FormatBytes(1048576)` → `"1.0 MiB"`
   - `FormatBytes(-1024)` → `"-1.0 KiB"`
4. **Tests:** Add `TestFormatBytes` to `pkg/display/display_test.go` covering:
   - Zero, small bytes, exact boundaries (1024, 1048576)
   - Negative values
   - Large values (GiB, TiB)
   - Values between boundaries

### Constraints

- Do NOT modify any existing functions
- Do NOT add new dependencies
- Place the function after the existing `FormatDurationShort` function
- Follow existing code style (see existing functions for patterns)

### Verification

After implementing, run:
```bash
go test ./pkg/display/ -v -run TestFormatBytes
```

Commit your changes when tests pass.

## IMPORTANT: Coordination Context

Another agent is SIMULTANEOUSLY working on this same codebase. They are implementing a different feature that also modifies display.go and display_test.go.

Their full task description:
---
# Task: Add FormatRate to pkg/display

## Instructions

Add a `FormatRate` function to `pkg/display/display.go` and write comprehensive tests in `pkg/display/display_test.go`.

### Requirements

1. **Function signature:** `func FormatRate(bytesPerSec float64) string`
2. **Behavior:**
   - Format transfer rates into human-readable strings
   - Use binary units with "/s" suffix: B/s, KiB/s, MiB/s, GiB/s
   - Show 1 decimal place for non-byte-per-sec units (e.g., "1.5 MiB/s")
   - Handle zero: return "0 B/s"
   - Handle negative values by prefixing with "-"
3. **Examples:**
   - `FormatRate(0)` → `"0 B/s"`
   - `FormatRate(512)` → `"512 B/s"`
   - `FormatRate(1024)` → `"1.0 KiB/s"`
   - `FormatRate(1536)` → `"1.5 KiB/s"`
   - `FormatRate(1048576)` → `"1.0 MiB/s"`
   - `FormatRate(-1024)` → `"-1.0 KiB/s"`
4. **Tests:** Add `TestFormatRate` to `pkg/display/display_test.go` covering:
   - Zero, small values, exact boundaries (1024, 1048576)
   - Negative values
   - Large values (GiB/s)
   - Fractional values

### Constraints

- Do NOT modify any existing functions
- Do NOT add new dependencies
- Place the function after the existing `FormatDurationShort` function
- Follow existing code style (see existing functions for patterns)

### Verification

After implementing, run:
```bash
go test ./pkg/display/ -v -run TestFormatRate
```

Commit your changes when tests pass.
---

You must coordinate to avoid merge conflicts:
- Be aware of where the other agent will insert code
- Choose insertion points that won't overlap with theirs
- Ensure your changes can be merged cleanly with theirs
- Do NOT implement their task — only implement yours

## MANDATORY: Post-Implementation Conflict Check

After you finish writing your code, you MUST perform this verification step before committing:

1. **Review your insertion points:** Look at exactly where in display.go and display_test.go you placed your new code.
2. **Predict the other agent's insertion points:** Based on their task description above, determine where they will most likely insert their code.
3. **Check for overlap:** If your insertion point is the same as or adjacent to where the other agent will likely insert code, you MUST revise your code to use a DIFFERENT insertion point that won't conflict.
4. **Document your check:** Add a comment at the top of your commit message: 'Conflict check: [your insertion point] vs predicted [their insertion point] — [conflict/no conflict]'

This verification step is REQUIRED. Do NOT skip it. If you detect a potential conflict, you MUST move your code to a different location in the file before committing.
