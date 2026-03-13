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
			headers: []string{"Name", "Age", "City"},
			rows: [][]string{
				{"Alice", "30", "NYC"},
				{"Bob", "25", "LA"},
			},
			want: "| Name  | Age | City |\n+-------+-----+------+\n| Alice | 30  | NYC  |\n| Bob   | 25  | LA   |\n",
		},
		{
			name:    "headers only",
			headers: []string{"ID", "Status"},
			rows:    [][]string{},
			want:    "| ID | Status |\n+----+--------+\n",
		},
		{
			name:    "single column",
			headers: []string{"Item"},
			rows: [][]string{
				{"apple"},
				{"banana"},
			},
			want: "| Item   |\n+--------+\n| apple  |\n| banana |\n",
		},
		{
			name:    "ANSI colored content",
			headers: []string{"Color", "Value"},
			rows: [][]string{
				{"\x1b[31mRed\x1b[0m", "100"},
				{"\x1b[32mGreen\x1b[0m", "200"},
			},
			want: "| Color | Value |\n+-------+-------+\n| \x1b[31mRed\x1b[0m   | 100   |\n| \x1b[32mGreen\x1b[0m | 200   |\n",
		},
		{
			name:    "mismatched columns (fewer columns in row)",
			headers: []string{"A", "B", "C"},
			rows: [][]string{
				{"x", "y"},
				{"1", "2", "3"},
			},
			want: "| A | B | C |\n+---+---+---+\n| x | y |   |\n| 1 | 2 | 3 |\n",
		},
		{
			name:    "wide content",
			headers: []string{"Short", "VeryLongHeader"},
			rows: [][]string{
				{"a", "b"},
				{"exactly", "this is a very long string that should determine column width"},
			},
			want: "| Short   | VeryLongHeader                                                |\n+---------+---------------------------------------------------------------+\n| a       | b                                                             |\n| exactly | this is a very long string that should determine column width |\n",
		},
		{
			name:    "empty strings in cells",
			headers: []string{"A", "B"},
			rows: [][]string{
				{"", "val"},
				{"val", ""},
			},
			want: "| A   | B   |\n+-----+-----+\n|     | val |\n| val |     |\n",
		},
		{
			name:    "all empty headers",
			headers: []string{"", "", ""},
			rows: [][]string{
				{"x", "y", "z"},
			},
			want: "|   |   |   |\n+---+---+---+\n| x | y | z |\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTable(tt.headers, tt.rows)
			if got != tt.want {
				t.Errorf("FormatTable(%v, %v) =\n%q\nwant\n%q", tt.headers, tt.rows, got, tt.want)
			}
		})
	}
}

func TestFormatTableEmptyHeaders(t *testing.T) {
	got := FormatTable([]string{}, [][]string{})
	if got != "" {
		t.Errorf("FormatTable([], []) = %q, want empty string", got)
	}
}
