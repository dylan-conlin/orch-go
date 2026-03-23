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

## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the `FormatDuration` function in display.go.
Place your new test function(s) IMMEDIATELY after `TestFormatDuration` in display_test.go.

Do NOT place code anywhere else in these files.
