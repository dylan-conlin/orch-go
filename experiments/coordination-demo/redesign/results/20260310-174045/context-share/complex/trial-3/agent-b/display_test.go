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
			name:    "basic table with 2 columns and 2 rows",
			headers: []string{"Name", "Status"},
			rows: [][]string{
				{"Alice", "active"},
				{"Bob", "inactive"},
			},
			want: "| Name  | Status   |\n| ----- | -------- |\n| Alice | active   |\n| Bob   | inactive |",
		},
		{
			name:    "headers only (empty rows)",
			headers: []string{"Header1", "Header2"},
			rows:    [][]string{},
			want:    "| Header1 | Header2 |\n| ------- | ------- |",
		},
		{
			name:    "single column table",
			headers: []string{"Item"},
			rows: [][]string{
				{"Apple"},
				{"Banana"},
			},
			want: "| Item   |\n| ------ |\n| Apple  |\n| Banana |",
		},
		{
			name:    "mismatched column count - row shorter than headers",
			headers: []string{"A", "B", "C"},
			rows: [][]string{
				{"1", "2"},
				{"3", "4", "5"},
			},
			want: "| A | B | C |\n| - | - | - |\n| 1 | 2 |   |\n| 3 | 4 | 5 |",
		},
		{
			name:    "ANSI colored content alignment",
			headers: []string{"Status", "Message"},
			rows: [][]string{
				{"\x1b[32mok\x1b[0m", "green text"},
				{"\x1b[31merror\x1b[0m", "red error"},
			},
			want: "| Status | Message    |\n| ------ | ---------- |\n| \x1b[32mok\x1b[0m     | green text |\n| \x1b[31merror\x1b[0m  | red error  |",
		},
		{
			name:    "wide content",
			headers: []string{"Short", "VeryLongHeaderName"},
			rows: [][]string{
				{"a", "this is a very long cell content"},
			},
			want: "| Short | VeryLongHeaderName               |\n| ----- | -------------------------------- |\n| a     | this is a very long cell content |",
		},
		{
			name:    "nil rows",
			headers: []string{"Header"},
			rows:    nil,
			want:    "| Header |\n| ------ |",
		},
		{
			name:    "empty headers",
			headers: []string{},
			rows:    [][]string{{"data"}},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTable(tt.headers, tt.rows)
			if got != tt.want {
				t.Errorf("FormatTable() got:\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}

func TestFormatTableEdgeCases(t *testing.T) {
	// Test with rows containing more columns than headers
	headers := []string{"A", "B"}
	rows := [][]string{
		{"1", "2", "3"},
	}
	result := FormatTable(headers, rows)
	// Should only use first 2 columns, ignoring the extra "3"
	if !contains(result, "| 1 | 2") {
		t.Errorf("FormatTable should handle extra columns gracefully, got: %q", result)
	}

	// Test with empty cells
	headers = []string{"Name", "Age"}
	rows = [][]string{
		{"Alice", ""},
		{"", "30"},
	}
	result = FormatTable(headers, rows)
	if len(result) == 0 {
		t.Errorf("FormatTable should handle empty cells, got empty result")
	}

	// Test with multi-character unicode
	headers = []string{"Name"}
	rows = [][]string{
		{"Alice"},
		{"李明"},
	}
	result = FormatTable(headers, rows)
	// Visual width should account for runes, not bytes
	lines := split(result, "\n")
	if len(lines) < 2 {
		t.Errorf("FormatTable should handle unicode, got: %q", result)
	}
}

// Helper functions for test assertions
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func split(s, sep string) []string {
	return strings.Split(s, sep)
}
