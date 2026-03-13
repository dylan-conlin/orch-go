package display

import (
	"strings"
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
	tests := []struct {
		name    string
		headers []string
		rows    [][]string
		want    string
	}{
		{
			name:    "basic table",
			headers: []string{"Name", "Age"},
			rows: [][]string{
				{"Alice", "30"},
				{"Bob", "25"},
			},
			want: "Name  | Age\n------+----\nAlice | 30 \nBob   | 25 \n",
		},
		{
			name:    "headers only (no rows)",
			headers: []string{"ID", "Status"},
			rows:    [][]string{},
			want:    "ID | Status\n---+-------\n",
		},
		{
			name:    "single column table",
			headers: []string{"Status"},
			rows: [][]string{
				{"active"},
				{"pending"},
			},
			want: "Status \n-------\nactive \npending\n",
		},
		{
			name:    "wide content",
			headers: []string{"Short", "VeryLongHeader"},
			rows: [][]string{
				{"x", "short"},
				{"y", "this is a much longer string"},
			},
			want: "Short | VeryLongHeader              \n------+-----------------------------\nx     | short                       \ny     | this is a much longer string\n",
		},
		{
			name:    "mismatched columns - row fewer than headers",
			headers: []string{"A", "B", "C"},
			rows: [][]string{
				{"1", "2"},
				{"3", "4", "5"},
			},
			want: "A | B | C\n--+---+--\n1 | 2 |  \n3 | 4 | 5\n",
		},
		{
			name:    "empty headers",
			headers: []string{},
			rows: [][]string{
				{"a", "b"},
			},
			want: "",
		},
		{
			name:    "nil rows",
			headers: []string{"Header"},
			rows:    nil,
			want:    "Header\n------\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTable(tt.headers, tt.rows)
			if got != tt.want {
				t.Errorf("FormatTable() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatTableWithANSI(t *testing.T) {
	// Test that ANSI codes are preserved but alignment is calculated correctly
	headers := []string{"Color"}
	rows := [][]string{
		{"\x1b[31mred\x1b[0m"},     // red: 3 chars visually, but longer with codes
		{"\x1b[32mgreen\x1b[0m"},   // green: 5 chars visually
	}

	result := FormatTable(headers, rows)

	// ANSI codes should be preserved in output
	if !strings.Contains(result, "\x1b[31m") {
		t.Errorf("ANSI codes should be preserved in output: %q", result)
	}

	// Width calculation should be correct (5 for "Color" header)
	lines := strings.Split(strings.TrimSuffix(result, "\n"), "\n")
	if len(lines) != 4 {
		t.Errorf("Expected 4 lines, got %d: %v", len(lines), lines)
	}

	// Separator should be 5 dashes (matching "Color" width)
	if lines[1] != "-----" {
		t.Errorf("Separator should be 5 dashes, got: %q", lines[1])
	}
}

func TestFormatTableAlignment(t *testing.T) {
	// Test that columns are properly aligned with ANSI colors
	headers := []string{"Short", "Long"}
	rows := [][]string{
		{"\x1b[35mABC\x1b[0m", "123"},    // Magenta "ABC" (3 chars visible)
		{"D", "\x1b[36mEFGHIJ\x1b[0m"},   // Cyan "EFGHIJ" (6 chars visible)
	}

	result := FormatTable(headers, rows)

	// Check that each line has consistent column separators
	lines := strings.Split(strings.TrimSuffix(result, "\n"), "\n")
	if len(lines) != 4 {
		t.Errorf("Expected 4 lines (header, separator, 2 data rows), got %d", len(lines))
	}

	// Verify separator has correct structure (dashes and +)
	if !strings.Contains(lines[1], "+") {
		t.Errorf("Separator line should contain '+', got: %q", lines[1])
	}

	// All lines should have consistent column count
	for i, line := range lines {
		if i == 1 { // separator line uses different format
			continue
		}
		pipeCount := strings.Count(line, "|")
		if pipeCount != 1 {
			t.Errorf("Line %d has %d pipes, want 1: %q", i, pipeCount, line)
		}
	}
}

func TestFormatTableEdgeCases(t *testing.T) {
	// Test single cell
	result := FormatTable([]string{"H"}, [][]string{{"X"}})
	if !strings.Contains(result, "H") || !strings.Contains(result, "X") {
		t.Errorf("Single cell table missing content: %q", result)
	}

	// Test empty strings in cells
	result = FormatTable(
		[]string{"A", "B"},
		[][]string{
			{"", "text"},
			{"text", ""},
		},
	)
	lines := strings.Split(strings.TrimSuffix(result, "\n"), "\n")
	if len(lines) != 4 {
		t.Errorf("Expected 4 lines, got %d", len(lines))
	}

	// Test row with more columns than headers (should be truncated to header count)
	result = FormatTable(
		[]string{"A", "B"},
		[][]string{
			{"1", "2", "3", "4"},
		},
	)
	// Only first 2 columns should be used
	if strings.Count(result, "|") != 2 { // 1 separator in header, 1 in data
		t.Errorf("Extra columns should be ignored: %q", result)
	}

	// Test multiple ANSI codes in one cell
	result = FormatTable(
		[]string{"Mixed"},
		[][]string{
			{"\x1b[1;32mbold green\x1b[0m"},  // bold green: 10 chars visually
			{"plain text"},                   // 10 chars
		},
	)
	// Should contain ANSI codes in output
	if !strings.Contains(result, "\x1b[1;32m") {
		t.Errorf("ANSI codes should be preserved: %q", result)
	}
}
