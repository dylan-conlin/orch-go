# Task: Refactor Truncate and TruncateWithPadding in pkg/display

## Instructions

Refactor the `Truncate` and `TruncateWithPadding` functions in `pkg/display/display.go` to be Unicode-aware, and update their tests in `pkg/display/display_test.go`.

### Requirements

1. **Make Truncate Unicode-aware:**
   - Use `[]rune` instead of raw byte indexing so multi-byte characters are handled correctly
   - `maxLen` refers to number of runes, not bytes
   - Preserve the `"..."` suffix when truncated (counts as 3 runes)
   - `Truncate("hello world", 8)` → `"hello..."` (unchanged)
   - `Truncate("cafe\u0301", 10)` → `"cafe\u0301"` (unchanged — fits within limit)

2. **Make TruncateWithPadding Unicode-aware:**
   - Use `[]rune` for character counting
   - Padding should still use spaces (ASCII)
   - `TruncateWithPadding("hello", 10)` → `"hello     "` (unchanged)

3. **Preserve existing behavior:** All current test cases MUST continue to pass with identical output for ASCII inputs.

4. **Update tests:** Add test cases for:
   - Multi-byte UTF-8 characters (e.g., emoji, CJK characters)
   - Mixed ASCII and multi-byte strings
   - Edge case: string that is exactly maxLen runes

### Constraints

- Only modify the `Truncate`, `TruncateWithPadding` functions and their tests (`TestTruncate`, `TestTruncateWithPadding`)
- Do NOT modify any other functions in display.go
- Do NOT add new exported functions
- Do NOT modify any other test functions
- Do NOT add new dependencies

### Verification

After implementing, run:
```bash
go test ./pkg/display/ -v -run "TestTruncate|TestTruncateWithPadding"
```

Then run ALL tests to confirm no regressions:
```bash
go test ./pkg/display/ -v
```

Commit your changes when all tests pass.

## IMPORTANT: File Region Boundaries

You may ONLY modify code in these regions of display.go:
- The `Truncate` function (currently lines 14-19)
- The `TruncateWithPadding` function (currently lines 23-28)
- The import block if needed for your changes

You may ONLY modify code in these regions of display_test.go:
- The `TestTruncate` function (currently lines 8-26)
- The `TestTruncateWithPadding` function (currently lines 28-47)
- The import block if needed

Do NOT touch any other lines in these files.
