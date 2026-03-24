# Task: Add FormatRate to pkg/display

## Context

The `pkg/display` package provides shared output formatting utilities used across orch commands: string truncation (`Truncate`, `TruncateWithPadding`), ID abbreviation (`ShortID`), ANSI stripping (`StripANSI`), and human-readable duration formatting (`FormatDuration`, `FormatDurationShort`). You are adding a transfer-rate formatting function to this package.

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

### Code Style

Follow the existing patterns in the file:
- Table-driven tests with `[]struct` and `for _, tt := range tests`
- Error messages: `t.Errorf("FuncName(%v) = %q, want %q", ...)`
- One function per concern, clear doc comments
- Use `fmt.Sprintf` for formatting

### Constraints

- Do NOT modify any existing functions
- Do NOT add new dependencies
- Follow existing code style

### Verification

After implementing, run:
```bash
go test ./pkg/display/ -v -run TestFormatRate
```

Then run ALL tests to confirm no regressions:
```bash
go test ./pkg/display/ -v
```

Commit your changes when tests pass.
