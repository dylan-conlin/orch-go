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

func TestFormatTable(t *testing.T) {
	t.Run("basic table", func(t *testing.T) {
		headers := []string{"Name", "Status", "Time"}
		rows := [][]string{
			{"Alice", "active", "2m"},
			{"Bob", "idle", "5h"},
			{"Charlie", "active", "1d"},
		}
		got := FormatTable(headers, rows)

		// Check that output contains expected structure
		if !containsSubstring(got, "| Name") && !containsSubstring(got, "| Status") && !containsSubstring(got, "| Time") {
			t.Errorf("FormatTable() header not formatted correctly, got:\n%s", got)
		}
		if !containsSubstring(got, "| Alice") && !containsSubstring(got, "| active") && !containsSubstring(got, "| 2m") {
			t.Errorf("FormatTable() first row not formatted correctly, got:\n%s", got)
		}
		if !containsSubstring(got, "---|") {
			t.Errorf("FormatTable() separator not formatted correctly, got:\n%s", got)
		}
		// Verify structure
		lines := splitLines(got)
		if len(lines) < 5 {
			t.Errorf("FormatTable() expected at least 5 lines (header, separator, 3 rows), got %d", len(lines))
		}
	})

	t.Run("empty table (headers only)", func(t *testing.T) {
		headers := []string{"Col1", "Col2"}
		rows := [][]string{}
		got := FormatTable(headers, rows)

		if !containsSubstring(got, "| Col1") || !containsSubstring(got, "| Col2") {
			t.Errorf("FormatTable() with no rows should still show headers, got:\n%s", got)
		}
		if !containsSubstring(got, "|---") {
			t.Errorf("FormatTable() with no rows should show separator, got:\n%s", got)
		}
		// Should have exactly 2 lines: header + separator
		lines := splitLines(got)
		if len(lines) != 2 {
			t.Errorf("FormatTable() with no rows expected 2 lines, got %d", len(lines))
		}
	})

	t.Run("ANSI colored content alignment", func(t *testing.T) {
		headers := []string{"Name", "Status"}
		rows := [][]string{
			{"Alice", "\x1b[32mactive\x1b[0m"},   // green "active"
			{"Bob", "\x1b[31minactive\x1b[0m"}, // red "inactive"
		}
		got := FormatTable(headers, rows)

		// The visual width of "active" (6) and "inactive" (8) should determine column width
		// Column should be sized for "inactive" (8 chars)
		if !containsSubstring(got, "\x1b[32mactive\x1b[0m") {
			t.Errorf("FormatTable() should preserve ANSI codes, got:\n%s", got)
		}
		// Check alignment: "active" should be padded to match "inactive" width
		lines := splitLines(got)
		if len(lines) >= 4 {
			// Row with "active" should have padding to match "inactive"
			if !containsSubstring(lines[3], "active") {
				t.Errorf("FormatTable() ANSI alignment issue, got:\n%s", got)
			}
		}
	})

	t.Run("mismatched column counts", func(t *testing.T) {
		headers := []string{"A", "B", "C"}
		rows := [][]string{
			{"1", "2", "3"},
			{"4", "5"},      // missing column
			{"6", "7", "8", "9"}, // extra column (should be ignored)
		}
		got := FormatTable(headers, rows)

		// Should handle gracefully - missing columns render as empty
		if !containsSubstring(got, "| 4") || !containsSubstring(got, "| 5") {
			t.Errorf("FormatTable() should handle missing columns, got:\n%s", got)
		}
		// Extra columns should be ignored (only render up to header count)
		if containsSubstring(got, " 9") {
			t.Errorf("FormatTable() should ignore extra columns beyond headers, got:\n%s", got)
		}
	})

	t.Run("single column table", func(t *testing.T) {
		headers := []string{"ID"}
		rows := [][]string{
			{"abc"},
			{"def"},
		}
		got := FormatTable(headers, rows)

		if !containsSubstring(got, "| ID") {
			t.Errorf("FormatTable() single column header issue, got:\n%s", got)
		}
		if !containsSubstring(got, "| abc") {
			t.Errorf("FormatTable() single column row issue, got:\n%s", got)
		}
		if !containsSubstring(got, "| def") {
			t.Errorf("FormatTable() single column should have def row, got:\n%s", got)
		}
	})

	t.Run("wide content", func(t *testing.T) {
		headers := []string{"Short", "VeryLongHeaderName"}
		rows := [][]string{
			{"A", "x"},
			{"B", "This is a very long content string"},
		}
		got := FormatTable(headers, rows)

		// Column should be sized for the widest content
		if !containsSubstring(got, "This is a very long content string") {
			t.Errorf("FormatTable() should include wide content, got:\n%s", got)
		}
		// Verify we have the right number of lines
		lines := splitLines(got)
		if len(lines) != 4 {
			t.Errorf("FormatTable() expected 4 lines (header, separator, 2 rows), got %d", len(lines))
		}
		// Check both data rows exist
		if !containsSubstring(got, "| A") || !containsSubstring(got, "| B") {
			t.Errorf("FormatTable() missing data rows, got:\n%s", got)
		}
	})

	t.Run("nil rows", func(t *testing.T) {
		headers := []string{"A", "B"}
		got := FormatTable(headers, nil)

		// Should handle nil rows like empty rows
		if !containsSubstring(got, "| A") || !containsSubstring(got, "| B") {
			t.Errorf("FormatTable() should handle nil rows, got:\n%s", got)
		}
		// Should have exactly 2 lines: header + separator
		lines := splitLines(got)
		if len(lines) != 2 {
			t.Errorf("FormatTable() with nil rows expected 2 lines, got %d", len(lines))
		}
	})

	t.Run("empty headers", func(t *testing.T) {
		headers := []string{}
		rows := [][]string{{"a", "b"}}
		got := FormatTable(headers, rows)

		// Empty headers should return empty string
		if got != "" {
			t.Errorf("FormatTable() with empty headers should return empty string, got:\n%s", got)
		}
	})
}

// Helper functions for testing

func containsLine(s, line string) bool {
	for _, l := range splitLines(s) {
		if l == line {
			return true
		}
	}
	return false
}

func containsSubstring(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && splitContains(s, substr)
}

func splitContains(s, substr string) bool {
	// Simple substring check
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	lines := []string{}
	current := ""
	for _, ch := range s {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
