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

func TestFormatTable_Basic(t *testing.T) {
	headers := []string{"Name", "Age", "City"}
	rows := [][]string{
		{"Alice", "30", "NYC"},
		{"Bob", "25", "LA"},
	}

	got := FormatTable(headers, rows)

	// Verify table structure
	lines := strings.Split(got, "\n")
	if len(lines) != 4 {
		t.Errorf("FormatTable basic: expected 4 lines (header, sep, 2 rows), got %d", len(lines))
	}

	// Verify header row exists
	if !strings.Contains(got, "Name") || !strings.Contains(got, "Age") {
		t.Errorf("FormatTable basic: headers not found in output")
	}

	// Verify data rows exist
	if !strings.Contains(got, "Alice") || !strings.Contains(got, "Bob") {
		t.Errorf("FormatTable basic: data rows not found in output")
	}

	// Verify separator row exists (line of dashes)
	if !strings.Contains(lines[1], "-") {
		t.Errorf("FormatTable basic: separator row not found")
	}
}

func TestFormatTable_HeadersOnly(t *testing.T) {
	headers := []string{"ID", "Status", "Message"}
	rows := [][]string{}

	got := FormatTable(headers, rows)

	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Errorf("FormatTable headers only: expected 2 lines (header, sep), got %d", len(lines))
	}

	if !strings.Contains(got, "ID") || !strings.Contains(got, "Status") {
		t.Errorf("FormatTable headers only: headers not found")
	}
}

func TestFormatTable_ANSIContent(t *testing.T) {
	// Test with ANSI colored content
	red := "\x1b[31m"
	reset := "\x1b[0m"

	headers := []string{"Status", "Value"}
	rows := [][]string{
		{red + "ERROR" + reset, "123"},
		{"OK", red + "456" + reset},
	}

	got := FormatTable(headers, rows)

	// Verify ANSI codes are preserved in output
	if !strings.Contains(got, "\x1b[31m") {
		t.Errorf("FormatTable ANSI: ANSI codes were stripped from output")
	}

	// Verify content is readable (ANSI codes stripped for width calc)
	if !strings.Contains(got, "ERROR") || !strings.Contains(got, "OK") {
		t.Errorf("FormatTable ANSI: content not found in output")
	}

	// Verify alignment is correct (colors shouldn't break alignment)
	lines := strings.Split(got, "\n")
	if len(lines) < 4 {
		t.Errorf("FormatTable ANSI: expected at least 4 lines, got %d", len(lines))
	}
}

func TestFormatTable_MismatchedColumns(t *testing.T) {
	headers := []string{"A", "B", "C"}
	rows := [][]string{
		{"1", "2"},        // Fewer columns than headers
		{"3", "4", "5"},   // Exact match
		{"6", "7", "8", "9"}, // More columns than headers (will be included)
	}

	got := FormatTable(headers, rows)

	// Verify all rows are present
	if !strings.Contains(got, "1") || !strings.Contains(got, "6") || !strings.Contains(got, "9") {
		t.Errorf("FormatTable mismatched: not all data present in output")
	}

	lines := strings.Split(got, "\n")
	// Should have header, separator, and 3 data rows
	if len(lines) < 5 {
		t.Errorf("FormatTable mismatched: expected at least 5 lines, got %d", len(lines))
	}
}

func TestFormatTable_SingleColumn(t *testing.T) {
	headers := []string{"Items"}
	rows := [][]string{
		{"apple"},
		{"banana"},
		{"cherry"},
	}

	got := FormatTable(headers, rows)

	if !strings.Contains(got, "Items") {
		t.Errorf("FormatTable single column: header not found")
	}

	if !strings.Contains(got, "apple") || !strings.Contains(got, "banana") || !strings.Contains(got, "cherry") {
		t.Errorf("FormatTable single column: data not found")
	}

	// Verify format with pipes
	if !strings.Contains(got, "|") {
		t.Errorf("FormatTable single column: pipe separators not found")
	}
}

func TestFormatTable_WideContent(t *testing.T) {
	headers := []string{"Short", "VeryLongHeaderName"}
	rows := [][]string{
		{"x", "this is a much longer piece of text"},
		{"longer cell here", "y"},
	}

	got := FormatTable(headers, rows)

	// Verify all content is present
	if !strings.Contains(got, "VeryLongHeaderName") || !strings.Contains(got, "this is a much longer piece of text") {
		t.Errorf("FormatTable wide content: content not found")
	}

	// Verify table is properly formatted with separators
	lines := strings.Split(got, "\n")
	if len(lines) < 4 {
		t.Errorf("FormatTable wide content: expected at least 4 lines, got %d", len(lines))
	}

	// Check that separator line has appropriate dashes for column widths
	if !strings.Contains(lines[1], "--") {
		t.Errorf("FormatTable wide content: separator line not properly formatted")
	}
}

func TestFormatTable_EmptyHeaders(t *testing.T) {
	headers := []string{}
	rows := [][]string{
		{"a", "b"},
	}

	got := FormatTable(headers, rows)

	if got != "" {
		t.Errorf("FormatTable empty headers: expected empty string, got %q", got)
	}
}
