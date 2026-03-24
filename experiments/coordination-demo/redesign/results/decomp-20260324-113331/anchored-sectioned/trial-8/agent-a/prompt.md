# Task: Add FormatBytes — Size Formatting Utility

## Domain Context

You are adding a **data-size formatting** function to `pkg/display/display.go`. FormatBytes is conceptually a data-size utility — it converts raw byte counts into human-readable display strings.

The display package currently has two domains:
- **String operations:** Truncate, TruncateWithPadding, ShortID, StripANSI — these manipulate strings for display
- **Duration formatting:** FormatDuration, FormatDurationShort — these convert time durations to display strings

Your function establishes a new domain: **size formatting** — converting numeric data sizes into display strings. Place it where size-related utilities would naturally belong in the package organization.

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
- Follow existing code style

### Verification

After implementing, run:
```bash
go test ./pkg/display/ -v -run TestFormatBytes
```

Commit your changes when tests pass.
