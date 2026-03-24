# Task: Add FormatTable to pkg/display

## Instructions

Add a `FormatTable` function to `pkg/display/display.go` and write comprehensive tests in `pkg/display/display_test.go`.

### Requirements

1. **Function signature:** `func FormatTable(headers []string, rows [][]string) string`
2. **Behavior:**
   - Render headers and rows as an aligned text table
   - Auto-size each column based on the widest content in that column (including header)
   - Separate the header from data rows with a line of dashes
   - Handle ANSI-colored content correctly — use the existing `StripANSI` function to calculate true visual widths
   - Handle edge cases: empty rows (headers only), rows with fewer columns than headers, nil rows
3. **Design choices (up to you):**
   - Border and separator style (pipes, spaces, etc.)
   - Column padding amount
   - How to handle rows with more columns than headers
4. **Constraint:** You MUST use the existing `StripANSI` function for width calculation — do NOT reimplement ANSI stripping.

### Tests

Add `TestFormatTable` to `pkg/display/display_test.go` covering:
- Basic table (headers + rows)
- Empty table (headers only, no rows)
- ANSI-colored content alignment
- Mismatched column counts
- Single-column table
- Wide content (long strings)

### Constraints

- Do NOT modify any existing functions
- Do NOT add new dependencies — standard library only
- All public functions MUST have doc comments
- Follow existing code style

### Verification

After implementing, run:
```bash
go test ./pkg/display/ -v
```

All tests (existing and new) must pass. Commit your changes when tests pass.
