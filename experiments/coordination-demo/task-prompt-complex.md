# Experiment Task: Add Table Renderer to pkg/display

## Instructions

Add a text table rendering capability to the `pkg/display` package. This involves modifying one existing file and creating two new files.

### Part 1: Add VisualWidth to display.go

Add a `VisualWidth` function to `pkg/display/display.go`:

1. **Function signature:** `func VisualWidth(s string) int`
2. **Behavior:** Returns the visual display width of a string, ignoring ANSI escape codes. This allows accurate column alignment when cells contain colored text.
3. **Constraint:** You MUST use the existing `StripANSI` function from this package — do NOT reimplement ANSI stripping.
4. **Tests:** Add test cases for `VisualWidth` to `pkg/display/display_test.go`.

### Part 2: Create table.go

Create a new file `pkg/display/table.go` with a table rendering function:

1. **Function signature:** `func RenderTable(headers []string, rows [][]string) string`
2. **Behavior:**
   - Render headers and rows as an aligned text table
   - Auto-size each column based on the widest content in that column (including header)
   - Separate the header from data rows visually
   - Handle ANSI-colored content correctly — columns should align even when cells contain color codes
   - Handle edge cases gracefully: empty rows slice, rows with fewer columns than headers, rows with more columns than headers
3. **Design choices left to you:**
   - Border and separator style (choose something clean and readable)
   - Column padding
   - How to handle rows with mismatched column counts
4. **Tests:** Create `pkg/display/table_test.go` with tests covering:
   - Basic table rendering (headers + rows)
   - Empty table (headers only, no rows)
   - ANSI-colored content alignment
   - Mismatched column counts
   - Single-column table

### Constraints (MUST follow)

- Do NOT modify any existing functions in `display.go` — only ADD the new `VisualWidth` function
- Do NOT add any external dependencies — standard library only
- All public functions MUST have doc comments following Go conventions
- Follow the existing code style (see existing functions in `display.go` for patterns)
- Place `VisualWidth` after the existing `FormatDurationShort` function in `display.go`

### Verification

After implementing, run:
```bash
go test ./pkg/display/ -v
```

All tests (existing and new) must pass.

### Success Criteria

- [ ] `VisualWidth` function compiles and uses `StripANSI`
- [ ] `RenderTable` function compiles
- [ ] All new test cases pass
- [ ] No existing tests broken
- [ ] Table columns align correctly with ANSI-colored content
- [ ] No external dependencies added
- [ ] Doc comments on all public functions
