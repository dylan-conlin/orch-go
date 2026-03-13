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
			want: "Name  | Age\n------+----\nAlice | 30 \nBob   | 25 ",
		},
		{
			name:    "empty table (headers only)",
			headers: []string{"Col1", "Col2"},
			rows:    [][]string{},
			want:    "Col1 | Col2\n-----+-----",
		},
		{
			name:    "single column",
			headers: []string{"Item"},
			rows: [][]string{
				{"Apple"},
				{"Banana"},
			},
			want: "Item  \n------\nApple \nBanana",
		},
		{
			name:    "wide content",
			headers: []string{"Short", "VeryLongContent"},
			rows: [][]string{
				{"A", "B"},
			},
			want: "Short | VeryLongContent\n------+----------------\nA     | B              ",
		},
		{
			name:    "mismatched columns - row shorter than headers",
			headers: []string{"A", "B", "C"},
			rows: [][]string{
				{"1", "2"},
				{"x", "y", "z", "extra"},
			},
			want: "A | B | C\n--+---+--\n1 | 2 |  \nx | y | z",
		},
		{
			name:    "ANSI-colored content alignment",
			headers: []string{"Color", "Value"},
			rows: [][]string{
				{"\x1b[31mRed\x1b[0m", "123"},
				{"Plain", "\x1b[32mGreen\x1b[0m"},
			},
			want: "Color | Value\n------+------\n\x1b[31mRed\x1b[0m   | 123  \nPlain | \x1b[32mGreen\x1b[0m",
		},
		{
			name:    "all empty content",
			headers: []string{"A", "B"},
			rows: [][]string{
				{"", ""},
			},
			want: "A | B\n--+--\n  |  ",
		},
		{
			name:    "header with spaces",
			headers: []string{"Long Header", "Short"},
			rows: [][]string{
				{"x", "y"},
			},
			want: "Long Header | Short\n------------+------\nx           | y    ",
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

func TestFormatTableEdgeCases(t *testing.T) {
	// Test nil inputs
	result := FormatTable(nil, nil)
	if result != "" {
		t.Errorf("FormatTable(nil, nil) = %q, want empty string", result)
	}

	// Test empty headers
	result = FormatTable([]string{}, [][]string{})
	if result != "" {
		t.Errorf("FormatTable(empty headers, empty rows) = %q, want empty string", result)
	}

	// Test nil rows
	result = FormatTable([]string{"A", "B"}, nil)
	expected := "A | B\n--+--"
	if result != expected {
		t.Errorf("FormatTable(headers, nil rows) = %q, want %q", result, expected)
	}

	// Test single header, multiple rows
	result = FormatTable([]string{"Name"}, [][]string{
		{"Alice"},
		{"Bob"},
		{"Charlie"},
	})
	expected = "Name   \n-------\nAlice  \nBob    \nCharlie"
	if result != expected {
		t.Errorf("FormatTable single column = %q, want %q", result, expected)
	}
}
