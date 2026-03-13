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
			headers: []string{"Name", "Status"},
			rows: [][]string{
				{"alice", "active"},
				{"bob", "inactive"},
			},
			want: "Name  | Status  \n------+---------\nalice | active  \nbob   | inactive",
		},
		{
			name:    "headers only (no rows)",
			headers: []string{"Col1", "Col2", "Col3"},
			rows:    nil,
			want:    "Col1 | Col2 | Col3",
		},
		{
			name:    "single column",
			headers: []string{"Items"},
			rows: [][]string{
				{"apple"},
				{"banana"},
			},
			want: "Items \n------\napple \nbanana",
		},
		{
			name:    "wide content",
			headers: []string{"Short", "VeryLongHeaderName"},
			rows: [][]string{
				{"x", "y"},
				{"abc", "this is a very long value"},
			},
			want: "Short | VeryLongHeaderName       \n------+--------------------------\nx     | y                        \nabc   | this is a very long value",
		},
		{
			name:    "ANSI colored content",
			headers: []string{"Name", "Status"},
			rows: [][]string{
				{"\x1b[32malice\x1b[0m", "\x1b[31mactive\x1b[0m"},
				{"bob", "inactive"},
			},
			want: "Name  | Status  \n------+---------\n\x1b[32malice\x1b[0m | \x1b[31mactive\x1b[0m  \nbob   | inactive",
		},
		{
			name:    "row with fewer columns than headers",
			headers: []string{"A", "B", "C"},
			rows: [][]string{
				{"1", "2"},
				{"x", "y", "z"},
			},
			want: "A | B | C\n--+---+--\n1 | 2 |  \nx | y | z",
		},
		{
			name:    "row with more columns than headers (truncated)",
			headers: []string{"A", "B"},
			rows: [][]string{
				{"1", "2", "extra1", "extra2"},
				{"x", "y"},
			},
			want: "A | B\n--+--\n1 | 2\nx | y",
		},
		{
			name:    "empty strings as content",
			headers: []string{"X", "Y"},
			rows: [][]string{
				{"", "data"},
				{"data", ""},
			},
			want: "X    | Y   \n-----+-----\n     | data\ndata |     ",
		},
		{
			name:    "nil rows slice",
			headers: []string{"H1", "H2"},
			rows:    [][]string{},
			want:    "H1 | H2",
		},
		{
			name:    "column alignment with different widths",
			headers: []string{"ID", "Description", "Count"},
			rows: [][]string{
				{"1", "small", "10"},
				{"999", "much longer description here", "5"},
				{"42", "x", "1000000"},
			},
			want: "ID  | Description                  | Count  \n----+------------------------------+--------\n1   | small                        | 10     \n999 | much longer description here | 5      \n42  | x                            | 1000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTable(tt.headers, tt.rows)
			if got != tt.want {
				t.Errorf("FormatTable() output mismatch\nGot:\n%s\n\nWant:\n%s", got, tt.want)
			}
		})
	}
}

func TestFormatTableEdgeCases(t *testing.T) {
	// Empty headers
	got := FormatTable([]string{}, [][]string{})
	if got != "" {
		t.Errorf("FormatTable with empty headers should return empty string, got %q", got)
	}

	// Single row
	got = FormatTable([]string{"A", "B"}, [][]string{{"1", "2"}})
	want := "A | B\n--+--\n1 | 2"
	if got != want {
		t.Errorf("FormatTable single row mismatch\nGot:\n%s\n\nWant:\n%s", got, want)
	}

	// All empty strings
	got = FormatTable([]string{"", ""}, [][]string{{"", ""}})
	if strings.Count(got, "\n") != 2 {
		t.Errorf("FormatTable with empty strings should have 3 lines (header, separator, data), got %d lines", strings.Count(got, "\n")+1)
	}
}
