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

## IMPORTANT: File Region Boundaries

You may ONLY modify code in these regions of display.go:
- The `FormatDuration` function (currently lines 49-82)
- The import block if needed for your changes

You may ONLY modify code in these regions of display_test.go:
- The `TestFormatDuration` function (currently lines 87-114)
- The import block if needed

Do NOT touch any other lines in these files.
