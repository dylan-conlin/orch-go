# Task: Add FormatBytes to pkg/display

## Context

The `pkg/display` package provides shared output formatting utilities used across orch commands: string truncation (`Truncate`, `TruncateWithPadding`), ID abbreviation (`ShortID`), ANSI stripping (`StripANSI`), and human-readable duration formatting (`FormatDuration`, `FormatDurationShort`). You are adding a byte-size formatting function to this package.

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
go test ./pkg/display/ -v -run TestFormatBytes
```

Then run ALL tests to confirm no regressions:
```bash
go test ./pkg/display/ -v
```

Commit your changes when tests pass.
