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
			want: "| Name  | Age |\n|-------|-----|\n| Alice | 30  |\n| Bob   | 25  |\n",
		},
		{
			name:    "headers only",
			headers: []string{"Col1", "Col2"},
			rows:    [][]string{},
			want:    "| Col1 | Col2 |\n|------|------|\n",
		},
		{
			name:    "single column",
			headers: []string{"Item"},
			rows: [][]string{
				{"apple"},
				{"banana"},
			},
			want: "| Item   |\n|--------|\n| apple  |\n| banana |\n",
		},
		{
			name:    "ANSI colored content",
			headers: []string{"Color"},
			rows: [][]string{
				{"\x1b[31mred\x1b[0m"},
				{"\x1b[32mgreen\x1b[0m"},
			},
			want: "| Color |\n|-------|\n| \x1b[31mred\x1b[0m   |\n| \x1b[32mgreen\x1b[0m |\n",
		},
		{
			name:    "mixed ANSI and plain",
			headers: []string{"Status", "Count"},
			rows: [][]string{
				{"\x1b[1;32mactive\x1b[0m", "5"},
				{"inactive", "3"},
			},
			want: "| Status   | Count |\n|----------|-------|\n| \x1b[1;32mactive\x1b[0m   | 5     |\n| inactive | 3     |\n",
		},
		{
			name:    "row with fewer columns than headers",
			headers: []string{"A", "B", "C"},
			rows: [][]string{
				{"1", "2", "3"},
				{"x", "y"},
			},
			want: "| A | B | C |\n|---|---|---|\n| 1 | 2 | 3 |\n| x | y |   |\n",
		},
		{
			name:    "row with more columns than headers",
			headers: []string{"A", "B"},
			rows: [][]string{
				{"1", "2"},
				{"x", "y", "z"},
			},
			want: "| A | B |\n|---|---|\n| 1 | 2 |\n| x | y |\n",
		},
		{
			name:    "wide content",
			headers: []string{"Short", "LongerHeader"},
			rows: [][]string{
				{"a", "value1"},
				{"bb", "x"},
			},
			want: "| Short | LongerHeader |\n|-------|" + "----" + "----" + "------" + "|\n| a     | value1       |\n| bb    | x            |\n",
		},
		{
			name:    "empty strings",
			headers: []string{"Col1", "Col2"},
			rows: [][]string{
				{"", "data"},
				{"value", ""},
			},
			want: "| Col1  | Col2 |\n|-------|------|\n|       | data |\n| value |      |\n",
		},
		{
			name:    "numbers and text",
			headers: []string{"ID", "Name", "Value"},
			rows: [][]string{
				{"1", "item", "100"},
				{"22", "x", "5"},
			},
			want: "| ID | Name | Value |\n|----|------|-------|\n| 1  | item | 100   |\n| 22 | x    | 5     |\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTable(tt.headers, tt.rows)
			if got != tt.want {
				t.Errorf("FormatTable() output mismatch\nGot:\n%s\nWant:\n%s", got, tt.want)
			}
		})
	}
}

func TestFormatTableEdgeCases(t *testing.T) {
	t.Run("empty headers", func(t *testing.T) {
		got := FormatTable([]string{}, nil)
		if got != "" {
			t.Errorf("FormatTable(empty headers) = %q, want empty string", got)
		}
	})

	t.Run("nil rows", func(t *testing.T) {
		headers := []string{"Col1", "Col2"}
		got := FormatTable(headers, nil)
		want := "| Col1 | Col2 |\n|------|------|\n"
		if got != want {
			t.Errorf("FormatTable(nil rows) = %q, want %q", got, want)
		}
	})

	t.Run("multiple ANSI codes in single cell", func(t *testing.T) {
		headers := []string{"Text"}
		rows := [][]string{
			{"\x1b[1m\x1b[31mbold red\x1b[0m"},
		}
		got := FormatTable(headers, rows)
		// Visual width is 8 (bold red), so column should be 8 wide
		want := "| Text     |\n|----------|\n| \x1b[1m\x1b[31mbold red\x1b[0m |\n"
		if got != want {
			t.Errorf("FormatTable(multiple ANSI) = %q, want %q", got, want)
		}
	})

	t.Run("single row", func(t *testing.T) {
		headers := []string{"A", "B"}
		rows := [][]string{{"1", "2"}}
		got := FormatTable(headers, rows)
		want := "| A | B |\n|---|---|\n| 1 | 2 |\n"
		if got != want {
			t.Errorf("FormatTable(single row) output mismatch\nGot:\n%s\nWant:\n%s", got, want)
		}
	})

	t.Run("all empty rows", func(t *testing.T) {
		headers := []string{"Col1", "Col2"}
		rows := [][]string{
			{"", ""},
			{"", ""},
		}
		got := FormatTable(headers, rows)
		want := "| Col1 | Col2 |\n|------|------|\n|      |      |\n|      |      |\n"
		if got != want {
			t.Errorf("FormatTable(all empty) output mismatch\nGot:\n%s\nWant:\n%s", got, want)
		}
	})
}
