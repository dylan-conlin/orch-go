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
			headers: []string{"Name", "Age", "City"},
			rows: [][]string{
				{"Alice", "30", "NYC"},
				{"Bob", "25", "LA"},
			},
			want: (` Name   |  Age  |  City ` +
				"\n" + `--------+-------+-------` +
				"\n" + ` Alice  |  30   |  NYC  ` +
				"\n" + ` Bob    |  25   |  LA   `),
		},
		{
			name:    "headers only (empty rows)",
			headers: []string{"ID", "Status"},
			rows:    [][]string{},
			want: (` ID  |  Status ` +
				"\n" + `-----+---------`),
		},
		{
			name:    "single column table",
			headers: []string{"Item"},
			rows: [][]string{
				{"Apple"},
				{"Banana"},
				{"Cherry"},
			},
			want: (` Item   ` +
				"\n" + `--------` +
				"\n" + ` Apple  ` +
				"\n" + ` Banana ` +
				"\n" + ` Cherry `),
		},
		{
			name:    "wide content",
			headers: []string{"Short", "Long"},
			rows: [][]string{
				{"A", "This is a very long string"},
				{"B", "Short"},
			},
			want: (` Short  |  Long                       ` +
				"\n" + `--------+-----------------------------` +
				"\n" + ` A      |  This is a very long string ` +
				"\n" + ` B      |  Short                      `),
		},
		{
			name:    "rows with fewer columns than headers",
			headers: []string{"A", "B", "C"},
			rows: [][]string{
				{"1", "2"},
				{"3"},
				{"4", "5", "6"},
			},
			want: (` A  |  B  |  C ` +
				"\n" + `----+-----+----` +
				"\n" + ` 1  |  2  |    ` +
				"\n" + ` 3  |     |    ` +
				"\n" + ` 4  |  5  |  6 `),
		},
		{
			name:    "rows with more columns than headers",
			headers: []string{"A", "B"},
			rows: [][]string{
				{"1", "2", "3"},
				{"4", "5"},
			},
			want: (` A  |  B  |    ` +
				"\n" + `----+-----+----` +
				"\n" + ` 1  |  2  |  3 ` +
				"\n" + ` 4  |  5  |    `),
		},
		{
			name:    "ANSI colored content",
			headers: []string{"Name", "Status"},
			rows: [][]string{
				{"\x1b[31mAlice\x1b[0m", "\x1b[32mActive\x1b[0m"},
				{"Bob", "\x1b[31mInactive\x1b[0m"},
			},
			want: " Name   |  Status   \n--------+-----------\n \x1b[31mAlice\x1b[0m  |  \x1b[32mActive\x1b[0m   \n Bob    |  \x1b[31mInactive\x1b[0m ",
		},
		{
			name:    "all empty strings",
			headers: []string{"", ""},
			rows: [][]string{
				{"", ""},
				{"", ""},
			},
			want: (`   |   ` +
				"\n" + `---+---` +
				"\n" + `   |   ` +
				"\n" + `   |   `),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTable(tt.headers, tt.rows)
			if got != tt.want {
				t.Errorf("FormatTable() mismatch\ngot:\n%q\n\nwant:\n%q", got, tt.want)
				// Print readable output for debugging
				t.Logf("Got output:\n%s\n", got)
				t.Logf("Want output:\n%s\n", tt.want)
			}
		})
	}
}

func TestFormatTableEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		headers []string
		rows    [][]string
		check   func(string) bool
	}{
		{
			name:    "nil rows",
			headers: []string{"Col"},
			rows:    nil,
			check: func(got string) bool {
				return strings.Contains(got, "Col") && strings.Count(got, "\n") == 1
			},
		},
		{
			name:    "empty headers returns empty string",
			headers: []string{},
			rows:    [][]string{{"data"}},
			check: func(got string) bool {
				return got == ""
			},
		},
		{
			name:    "column alignment is correct",
			headers: []string{"X", "YY", "ZZZ"},
			rows: [][]string{
				{"a", "b", "c"},
			},
			check: func(got string) bool {
				// Each column should be aligned
				lines := strings.Split(got, "\n")
				if len(lines) < 3 {
					return false
				}
				// Separator line should have matching column positions
				sep := lines[1]
				return strings.Count(sep, "+") == 2
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTable(tt.headers, tt.rows)
			if !tt.check(got) {
				t.Errorf("FormatTable() check failed\ngot:\n%s", got)
			}
		})
	}
}
