You are a code placement coordinator. Two parallel agents will each implement one task, modifying the same files simultaneously. Their changes must merge cleanly via git.

## Target Codebase

### pkg/display/display.go
```go
// Package display provides shared output formatting utilities used across
// orch commands and packages: string truncation, ID abbreviation, ANSI
// stripping, and human-readable duration formatting.
package display

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Truncate truncates s to maxLen characters, appending "..." if truncated.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// TruncateWithPadding truncates s to maxLen (with "...") or right-pads with
// spaces to ensure the returned string is exactly maxLen characters.
func TruncateWithPadding(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s + strings.Repeat(" ", maxLen-len(s))
}

// ShortID returns the first 12 characters of an ID string for display.
// If the string is 12 characters or shorter, it is returned unchanged.
func ShortID(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:12]
}

// ansiRegex matches ANSI escape sequences (colors, formatting, etc.)
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// StripANSI removes ANSI escape codes from a string.
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// FormatDuration formats a duration as a human-readable string.
// Output style: "0s", "45s", "3m 12s", "2h 15m", "3d 5h".
func FormatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	days := int(d.Hours()) / 24
	if days > 0 {
		hours := int(d.Hours()) % 24
		if hours > 0 {
			return fmt.Sprintf("%dd %dh", days, hours)
		}
		return fmt.Sprintf("%dd", days)
	}

	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}

	if d < time.Hour {
		minutes := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		if secs > 0 {
			return fmt.Sprintf("%dm %ds", minutes, secs)
		}
		return fmt.Sprintf("%dm", minutes)
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if minutes > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dh", hours)
}

// FormatDurationShort formats a duration using short labels suitable for
// dashboard/status output: "just now", "3m", "2h".
func FormatDurationShort(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}
```

### pkg/display/display_test.go
```go
package display

import (
	"testing"
	"time"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello..."},
		{"abcdefghij", 6, "abc..."},
		{"", 5, ""},
	}
	for _, tt := range tests {
		got := Truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestTruncateWithPadding(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello     "},
		{"hello world", 8, "hello..."},
		{"exact", 5, "exact"},
	}
	for _, tt := range tests {
		got := TruncateWithPadding(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("TruncateWithPadding(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
		if len(got) != tt.maxLen {
			t.Errorf("TruncateWithPadding(%q, %d) length = %d, want %d", tt.input, tt.maxLen, len(got), tt.maxLen)
		}
	}
}

func TestShortID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"ses_abc123def456xyz", "ses_abc123de"},
		{"short", "short"},
		{"exactly12ch", "exactly12ch"},
		{"exactly12chr", "exactly12chr"},
		{"exactly12chrs", "exactly12chr"},
		{"", ""},
	}
	for _, tt := range tests {
		got := ShortID(tt.input)
		if got != tt.want {
			t.Errorf("ShortID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"\x1b[31mred text\x1b[0m", "red text"},
		{"\x1b[1;32mbold green\x1b[0m", "bold green"},
		{"no ansi here", "no ansi here"},
		{"", ""},
	}
	for _, tt := range tests {
		got := StripANSI(tt.input)
		if got != tt.want {
			t.Errorf("StripANSI(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input time.Duration
		want  string
	}{
		{0, "0s"},
		{30 * time.Second, "30s"},
		{59 * time.Second, "59s"},
		{60 * time.Second, "1m"},
		{90 * time.Second, "1m 30s"},
		{5 * time.Minute, "5m"},
		{5*time.Minute + 15*time.Second, "5m 15s"},
		{60 * time.Minute, "1h"},
		{90 * time.Minute, "1h 30m"},
		{2 * time.Hour, "2h"},
		{2*time.Hour + 45*time.Minute, "2h 45m"},
		{24 * time.Hour, "1d"},
		{26 * time.Hour, "1d 2h"},
		{48 * time.Hour, "2d"},
		{50 * time.Hour, "2d 2h"},
	}
	for _, tt := range tests {
		got := FormatDuration(tt.input)
		if got != tt.want {
			t.Errorf("FormatDuration(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatDurationShort(t *testing.T) {
	tests := []struct {
		input time.Duration
		want  string
	}{
		{30 * time.Second, "just now"},
		{0, "just now"},
		{5 * time.Minute, "5m"},
		{90 * time.Minute, "1h"},
		{2 * time.Hour, "2h"},
		{26 * time.Hour, "26h"},
	}
	for _, tt := range tests {
		got := FormatDurationShort(tt.input)
		if got != tt.want {
			t.Errorf("FormatDurationShort(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
```

## Task for Agent A
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

## Task for Agent B
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

## Your Job

Analyze the codebase structure and both tasks. Assign specific, non-overlapping insertion points for each agent so their code changes will merge cleanly.

For each agent, specify:
1. Where in display.go to place their new function(s) — reference an existing function name
2. Where in display_test.go to place their new test(s) — reference an existing test function name

CRITICAL RULES:
- Agent A and Agent B MUST use DIFFERENT insertion points
- Reference only functions that exist in the current code
- Each placement must be "immediately after" a specific existing function

Output EXACTLY in this format (no other text):
AGENT_A_CODE_AFTER: <function name>
AGENT_A_TEST_AFTER: <test function name>
AGENT_B_CODE_AFTER: <function name>
AGENT_B_TEST_AFTER: <test function name>
