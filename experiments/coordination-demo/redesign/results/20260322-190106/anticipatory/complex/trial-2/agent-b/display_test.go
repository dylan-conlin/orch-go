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
			want: `| Name  | Age |
|-------|-----|
| Alice | 30  |
| Bob   | 25  |
`,
		},
		{
			name:    "headers only (no rows)",
			headers: []string{"Col1", "Col2"},
			rows:    nil,
			want: `| Col1 | Col2 |
|------|------|
`,
		},
		{
			name:    "empty rows slice",
			headers: []string{"Name", "Value"},
			rows:    [][]string{},
			want: `| Name | Value |
|------|-------|
`,
		},
		{
			name:    "single column",
			headers: []string{"Item"},
			rows:    [][]string{{"apple"}, {"banana"}},
			want: `| Item   |
|--------|
| apple  |
| banana |
`,
		},
		{
			name:    "ANSI colored content",
			headers: []string{"Status"},
			rows: [][]string{
				{"\x1b[32mactive\x1b[0m"},
				{"\x1b[31minactive\x1b[0m"},
			},
			want: "| Status   |\n|----------|\n| \x1b[32mactive\x1b[0m   |\n| \x1b[31minactive\x1b[0m |\n",
		},
		{
			name:    "mismatched columns - fewer data columns",
			headers: []string{"A", "B", "C"},
			rows:    [][]string{{"1", "2"}, {"3"}},
			want: `| A | B | C |
|---|---|---|
| 1 | 2 |   |
| 3 |   |   |
`,
		},
		{
			name:    "wide content exceeds header",
			headers: []string{"Short", "LongHeader"},
			rows:    [][]string{{"a", "This is a very long value"}},
			want: `| Short | LongHeader                |
|-------|---------------------------|
| a     | This is a very long value |
`,
		},
		{
			name:    "empty string values",
			headers: []string{"A", "B"},
			rows:    [][]string{{"", "x"}, {"y", ""}},
			want: `| A | B |
|---|---|
|   | x |
| y |   |
`,
		},
		{
			name:    "header wider than data",
			headers: []string{"LongHeader", "Short"},
			rows:    [][]string{{"a", "b"}},
			want: `| LongHeader | Short |
|------------|-------|
| a          | b     |
`,
		},
		{
			name:    "all fields same width",
			headers: []string{"X", "Y", "Z"},
			rows:    [][]string{{"a", "b", "c"}},
			want: `| X | Y | Z |
|---|---|---|
| a | b | c |
`,
		},
		{
			name:    "ANSI with padding calculation",
			headers: []string{"Status"},
			rows: [][]string{
				{"\x1b[1;32mbold green\x1b[0m"},
				{"\x1b[1;31mbold\x1b[0m"},
			},
			want: "| Status     |\n|------------|\n| \x1b[1;32mbold green\x1b[0m |\n| \x1b[1;31mbold\x1b[0m       |\n",
		},
		{
			name:    "more columns in row than headers",
			headers: []string{"A", "B"},
			rows:    [][]string{{"1", "2", "3"}},
			want: `| A | B |
|---|---|
| 1 | 2 |
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTable(tt.headers, tt.rows)
			if got != tt.want {
				t.Errorf("FormatTable() output mismatch:\nGot:\n%s\n\nWant:\n%s\n\nGot (repr):\n%q\n\nWant (repr):\n%q",
					got, tt.want, got, tt.want)
			}
		})
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
