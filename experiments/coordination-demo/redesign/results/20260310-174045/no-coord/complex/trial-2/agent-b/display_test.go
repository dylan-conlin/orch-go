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
			name:    "basic table with headers and rows",
			headers: []string{"Name", "Age"},
			rows: [][]string{
				{"Alice", "30"},
				{"Bob", "25"},
			},
			want: "| Name  | Age |\n|-------|-----|\n| Alice | 30  |\n| Bob   | 25  |\n",
		},
		{
			name:    "empty table (headers only, no rows)",
			headers: []string{"Col1", "Col2"},
			rows:    [][]string{},
			want:    "| Col1 | Col2 |\n|------|------|\n",
		},
		{
			name:    "single-column table",
			headers: []string{"ID"},
			rows: [][]string{
				{"123"},
				{"456"},
			},
			want: "| ID  |\n|-----|\n| 123 |\n| 456 |\n",
		},
		{
			name:    "wide content alignment",
			headers: []string{"Short", "VeryLongHeaderName"},
			rows: [][]string{
				{"x", "y"},
				{"verylongcontent", "z"},
			},
			want: "| Short           | VeryLongHeaderName |\n|-----------------|--------------------|\n| x               | y                  |\n| verylongcontent | z                  |\n",
		},
		{
			name:    "rows with fewer columns than headers",
			headers: []string{"A", "B", "C"},
			rows: [][]string{
				{"1", "2"},
				{"3", "4", "5"},
			},
			want: "| A | B | C |\n|---|---|---|\n| 1 | 2 |   |\n| 3 | 4 | 5 |\n",
		},
		{
			name:    "rows with more columns than headers",
			headers: []string{"A", "B"},
			rows: [][]string{
				{"1", "2", "3"},
				{"4", "5"},
			},
			want: "| A | B |   |\n|---|---|---|\n| 1 | 2 | 3 |\n| 4 | 5 |   |\n",
		},
		{
			name:    "ANSI colored content",
			headers: []string{"Name", "Status"},
			rows: [][]string{
				{"\x1b[31mAlice\x1b[0m", "\x1b[32mActive\x1b[0m"},
				{"Bob", "Inactive"},
			},
			want: "| Name  | Status   |\n|-------|----------|\n| \x1b[31mAlice\x1b[0m | \x1b[32mActive\x1b[0m   |\n| Bob   | Inactive |\n",
		},
		{
			name:    "mixed ANSI and plain text alignment",
			headers: []string{"Col"},
			rows: [][]string{
				{"\x1b[1;32mbold green text\x1b[0m"},
				{"plain"},
			},
			want: "| Col             |\n|-----------------|\n| \x1b[1;32mbold green text\x1b[0m |\n| plain           |\n",
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

func TestFormatTableEdgeCases(t *testing.T) {
	// Empty headers - should return empty string
	got := FormatTable([]string{}, [][]string{})
	if got != "" {
		t.Errorf("FormatTable with empty headers should return empty string, got %q", got)
	}

	// Nil rows - should handle gracefully
	got = FormatTable([]string{"A", "B"}, nil)
	want := "| A | B |\n|---|---|\n"
	if got != want {
		t.Errorf("FormatTable with nil rows =\n%q\nwant\n%q", got, want)
	}

	// Single header, no rows
	got = FormatTable([]string{"Only"}, [][]string{})
	want = "| Only |\n|------|\n"
	if got != want {
		t.Errorf("FormatTable with single header and no rows =\n%q\nwant\n%q", got, want)
	}

	// Row with all empty strings
	got = FormatTable([]string{"A"}, [][]string{{""}})
	want = "| A |\n|---|\n|   |\n"
	if got != want {
		t.Errorf("FormatTable with empty row content =\n%q\nwant\n%q", got, want)
	}
}
