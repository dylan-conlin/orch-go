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
			name:    "basic table with multiple rows",
			headers: []string{"Name", "Status", "Time"},
			rows: [][]string{
				{"agent-1", "running", "5m"},
				{"agent-2", "idle", "2h"},
				{"agent-3", "error", "30s"},
			},
			want: "Name    | Status  | Time\n--------+---------+-----\nagent-1 | running | 5m  \nagent-2 | idle    | 2h  \nagent-3 | error   | 30s \n",
		},
		{
			name:    "headers only (no rows)",
			headers: []string{"ID", "Value"},
			rows:    [][]string{},
			want:    "ID | Value\n---+------\n",
		},
		{
			name:    "nil rows treated as empty",
			headers: []string{"A", "B"},
			rows:    nil,
			want:    "A | B\n--+--\n",
		},
		{
			name:    "rows with fewer columns than headers",
			headers: []string{"Col1", "Col2", "Col3"},
			rows: [][]string{
				{"a", "b"},
				{"x", "y", "z"},
			},
			want: "Col1 | Col2 | Col3\n-----+------+-----\na    | b    |     \nx    | y    | z   \n",
		},
		{
			name:    "single column table",
			headers: []string{"Item"},
			rows: [][]string{
				{"apple"},
				{"banana"},
				{"cherry"},
			},
			want: "Item  \n------\napple \nbanana\ncherry\n",
		},
		{
			name:    "ANSI colored content alignment",
			headers: []string{"Color", "Text"},
			rows: [][]string{
				{"\x1b[31mred\x1b[0m", "test"},
				{"\x1b[1;32mbold green\x1b[0m", "data"},
			},
			want: "Color      | Text\n-----------+-----\n\x1b[31mred\x1b[0m        | test\n\x1b[1;32mbold green\x1b[0m | data\n",
		},
		{
			name:    "wide content forces column expansion",
			headers: []string{"Short", "Long"},
			rows: [][]string{
				{"a", "short"},
				{"bb", "this is a very long string"},
			},
			want: "Short | Long                      \n------+---------------------------\na     | short                     \nbb    | this is a very long string\n",
		},
		{
			name:    "empty table (no headers, no rows)",
			headers: []string{},
			rows:    [][]string{},
			want:    "",
		},
		{
			name:    "header with spaces requires proper width",
			headers: []string{"First Name", "Last Name"},
			rows: [][]string{
				{"John", "Doe"},
				{"Jane", "Smith"},
			},
			want: "First Name | Last Name\n-----------+----------\nJohn       | Doe      \nJane       | Smith    \n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTable(tt.headers, tt.rows)
			if got != tt.want {
				t.Errorf("FormatTable(%v, %v)\ngot:\n%q\nwant:\n%q", tt.headers, tt.rows, got, tt.want)
			}
		})
	}
}
