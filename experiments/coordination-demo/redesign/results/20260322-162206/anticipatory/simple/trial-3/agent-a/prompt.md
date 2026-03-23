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

## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the `FormatDuration` function in display.go.
Place your new test function(s) IMMEDIATELY after `TestFormatDuration` in display_test.go.

Do NOT place code anywhere else in these files.
