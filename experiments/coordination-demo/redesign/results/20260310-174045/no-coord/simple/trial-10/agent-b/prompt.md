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
