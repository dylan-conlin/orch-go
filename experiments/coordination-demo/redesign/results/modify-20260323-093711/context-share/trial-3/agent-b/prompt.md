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

## IMPORTANT: Coordination Context

Another agent is SIMULTANEOUSLY working on this same codebase. They are refactoring different functions in display.go and display_test.go.

Their full task description:
---
# Task: Refactor FormatDuration in pkg/display

## Instructions

Refactor the `FormatDuration` function in `pkg/display/display.go` and update its tests in `pkg/display/display_test.go`.

### Requirements

1. **Add week support:** Durations of 7+ days should display as weeks.
   - `7d` → `"1w"`
   - `8d 5h` → `"1w 1d"`
   - `14d` → `"2w"`
   - `15d 12h` → `"2w 1d"`
2. **Refactor the cascading if/else:** Replace the current chain with a cleaner approach using a slice of time thresholds.
3. **Preserve existing behavior:** All current test cases MUST continue to pass with identical output.
4. **Update TestFormatDuration:** Add test cases for:
   - Exact week boundaries (7 days, 14 days)
   - Weeks with remainder days
   - Large values (30+ days)

### Constraints

- Only modify the `FormatDuration` function and `TestFormatDuration` test
- Do NOT modify any other functions in display.go
- Do NOT add new exported functions
- Do NOT modify any other test functions
- Do NOT add new dependencies

### Verification

After implementing, run:
```bash
go test ./pkg/display/ -v -run TestFormatDuration
```

Then run ALL tests to confirm no regressions:
```bash
go test ./pkg/display/ -v
```

Commit your changes when all tests pass.
---

You must coordinate to avoid merge conflicts:
- Be aware of which functions the other agent will modify
- Do NOT modify any functions the other agent is working on
- Ensure your changes can be merged cleanly with theirs
- Do NOT implement their task — only implement yours
