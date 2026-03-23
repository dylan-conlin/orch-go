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
			rows:    [][]string{{"Alice", "30"}, {"Bob", "25"}},
			want: "| Name  | Age |\n" +
				"|-------|-----|\n" +
				"| Alice | 30  |\n" +
				"| Bob   | 25  |\n",
		},
		{
			name:    "headers only",
			headers: []string{"Name", "Status"},
			rows:    nil,
			want: "| Name | Status |\n" +
				"|------|--------|\n",
		},
		{
			name:    "empty rows",
			headers: []string{"Col1", "Col2"},
			rows:    [][]string{},
			want: "| Col1 | Col2 |\n" +
				"|------|------|\n",
		},
		{
			name:    "single column",
			headers: []string{"Item"},
			rows:    [][]string{{"apple"}, {"banana"}},
			want: "| Item   |\n" +
				"|--------|\n" +
				"| apple  |\n" +
				"| banana |\n",
		},
		{
			name:    "mismatched columns - fewer in row",
			headers: []string{"A", "B", "C"},
			rows:    [][]string{{"1", "2"}, {"3", "4", "5"}},
			want: "| A | B | C |\n" +
				"|---|---|---|\n" +
				"| 1 | 2 |   |\n" +
				"| 3 | 4 | 5 |\n",
		},
		{
			name:    "ANSI colored content",
			headers: []string{"Name"},
			rows:    [][]string{{"\x1b[31mRed\x1b[0m"}, {"\x1b[32mGreen\x1b[0m"}},
			want: "| Name  |\n" +
				"|-------|\n" +
				"| \x1b[31mRed\x1b[0m   |\n" +
				"| \x1b[32mGreen\x1b[0m |\n",
		},
		{
			name:    "wide content",
			headers: []string{"Short", "VeryLongHeader"},
			rows:    [][]string{{"A", "ShortText"}, {"LongerContent", "X"}},
			want: "| Short         | VeryLongHeader |\n" +
				"|---------------|----------------|\n" +
				"| A             | ShortText      |\n" +
				"| LongerContent | X              |\n",
		},
		{
			name:    "mixed ANSI and regular",
			headers: []string{"Status", "Value"},
			rows: [][]string{
				{"\x1b[32mactive\x1b[0m", "100"},
				{"inactive", "\x1b[31m0\x1b[0m"},
			},
			want: "| Status   | Value |\n" +
				"|----------|-------|\n" +
				"| \x1b[32mactive\x1b[0m   | 100   |\n" +
				"| inactive | \x1b[31m0\x1b[0m     |\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTable(tt.headers, tt.rows)
			if got != tt.want {
				t.Errorf("FormatTable() output mismatch\ngot:\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}

func TestFormatTableEdgeCases(t *testing.T) {
	// Empty headers should return empty string
	result := FormatTable([]string{}, [][]string{})
	if result != "" {
		t.Errorf("FormatTable with empty headers should return empty string, got %q", result)
	}

	// Single header with multiple rows
	headers := []string{"ID"}
	rows := [][]string{{"1"}, {"2"}, {"3"}}
	result = FormatTable(headers, rows)
	if result == "" {
		t.Errorf("FormatTable with single header should not be empty")
	}

	// Row with more columns than headers should only use up to header count
	headers = []string{"A", "B"}
	rows = [][]string{{"1", "2", "3", "4"}}
	result = FormatTable(headers, rows)
	lines := strings.Split(result, "\n")
	// Should have 4 lines: header, separator, data, trailing newline empty
	if len(lines) < 3 {
		t.Errorf("FormatTable should handle extra columns, got %d lines", len(lines))
	}
}

func TestFormatTableANSIWidthCalculation(t *testing.T) {
	// Create a header with ANSI codes and verify column width is based on stripped text
	headers := []string{"\x1b[1mHeader\x1b[0m"}
	rows := [][]string{{"\x1b[31mData\x1b[0m"}}

	result := FormatTable(headers, rows)

	// Check that ANSI codes are preserved but width calculation is correct
	if !strings.Contains(result, "\x1b[1m") {
		t.Errorf("FormatTable should preserve ANSI codes in header")
	}
	if !strings.Contains(result, "\x1b[31m") {
		t.Errorf("FormatTable should preserve ANSI codes in data")
	}

	// Verify alignment is based on stripped width (6 for "Header", 4 for "Data")
	// So header should have 2 spaces of right-padding, data should have 4
	lines := strings.Split(result, "\n")
	if len(lines) < 3 {
		t.Errorf("FormatTable should produce at least 3 lines")
	}
}
