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
- Place functions after the existing `FormatDurationShort` function
- Follow existing code style

### Verification

After implementing, run:
```bash
go test ./pkg/display/ -v
```

All tests (existing and new) must pass. Commit your changes when tests pass.

## IMPORTANT: Coordination Context

Another agent is SIMULTANEOUSLY working on this same codebase. They are implementing a different feature that also modifies display.go and display_test.go.

Their full task description:
---
# Task: Add VisualWidth to pkg/display

## Instructions

Add a `VisualWidth` function and a `PadToWidth` function to `pkg/display/display.go`, and write comprehensive tests in `pkg/display/display_test.go`.

### Part 1: VisualWidth

1. **Function signature:** `func VisualWidth(s string) int`
2. **Behavior:** Returns the visual display width of a string, ignoring ANSI escape codes.
3. **Constraint:** You MUST use the existing `StripANSI` function — do NOT reimplement ANSI stripping.
4. **Handle Unicode correctly** — count runes, not bytes.

### Part 2: PadToWidth

1. **Function signature:** `func PadToWidth(s string, width int) string`
2. **Behavior:** Right-pads a string with spaces to reach the target visual width. ANSI codes are preserved but don't count toward width. If the string is already wider than `width`, return it unchanged.

### Tests

Add `TestVisualWidth` and `TestPadToWidth` to `pkg/display/display_test.go` covering:
- Plain ASCII strings
- Strings with ANSI color codes
- Unicode strings (CJK characters, emoji)
- Empty strings
- Edge cases (already at width, wider than target)

### Constraints

- Do NOT modify any existing functions
- Do NOT add new dependencies — standard library only
- All public functions MUST have doc comments
- Place functions after the existing `FormatDurationShort` function
- Follow existing code style

### Verification

After implementing, run:
```bash
go test ./pkg/display/ -v
```

All tests (existing and new) must pass. Commit your changes when tests pass.
---

You must coordinate to avoid merge conflicts:
- Be aware of where the other agent will insert code
- Choose insertion points that won't overlap with theirs
- Ensure your changes can be merged cleanly with theirs
- Do NOT implement their task — only implement yours
