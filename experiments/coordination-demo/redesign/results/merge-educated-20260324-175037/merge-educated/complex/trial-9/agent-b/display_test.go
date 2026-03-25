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
	t.Run("basic table", func(t *testing.T) {
		headers := []string{"Name", "Age", "City"}
		rows := [][]string{
			{"Alice", "30", "NYC"},
			{"Bob", "25", "SF"},
			{"Charlie", "35", "LA"},
		}
		got := FormatTable(headers, rows)
		want := "Name     Age  City\n" +
			"-------  ---  ----\n" +
			"Alice    30   NYC \n" +
			"Bob      25   SF  \n" +
			"Charlie  35   LA  \n"
		if got != want {
			t.Errorf("FormatTable() =\n%q\n\nwant:\n%q", got, want)
		}
	})

	t.Run("empty table - headers only", func(t *testing.T) {
		headers := []string{"Col1", "Col2"}
		rows := [][]string{}
		got := FormatTable(headers, rows)
		want := "Col1  Col2\n" +
			"----  ----\n"
		if got != want {
			t.Errorf("FormatTable() =\n%q\n\nwant:\n%q", got, want)
		}
	})

	t.Run("nil rows", func(t *testing.T) {
		headers := []string{"A", "B"}
		got := FormatTable(headers, nil)
		want := "A  B\n" +
			"-  -\n"
		if got != want {
			t.Errorf("FormatTable() =\n%q\n\nwant:\n%q", got, want)
		}
	})

	t.Run("ANSI colored content", func(t *testing.T) {
		headers := []string{"Status", "Message"}
		rows := [][]string{
			{"\x1b[32mOK\x1b[0m", "All good"},
			{"\x1b[31mERROR\x1b[0m", "Failed"},
		}
		got := FormatTable(headers, rows)
		// Widths should be based on visible text: "Status"=6, "OK"=2, "ERROR"=5
		// So Status column should be 6 chars wide, Message column should be 8 chars ("All good")
		want := "Status  Message \n" +
			"------  --------\n" +
			"\x1b[32mOK\x1b[0m      All good\n" +
			"\x1b[31mERROR\x1b[0m   Failed  \n"
		if got != want {
			t.Errorf("FormatTable() with ANSI =\n%q\n\nwant:\n%q", got, want)
		}
	})

	t.Run("mismatched column counts - fewer columns", func(t *testing.T) {
		headers := []string{"A", "B", "C"}
		rows := [][]string{
			{"1", "2", "3"},
			{"4", "5"}, // Missing column C
			{"7"},      // Missing columns B and C
		}
		got := FormatTable(headers, rows)
		want := "A  B  C\n" +
			"-  -  -\n" +
			"1  2  3\n" +
			"4  5   \n" +
			"7      \n"
		if got != want {
			t.Errorf("FormatTable() =\n%q\n\nwant:\n%q", got, want)
		}
	})

	t.Run("mismatched column counts - more columns", func(t *testing.T) {
		headers := []string{"A", "B"}
		rows := [][]string{
			{"1", "2"},
			{"3", "4", "5", "6"}, // Extra columns ignored
		}
		got := FormatTable(headers, rows)
		want := "A  B\n" +
			"-  -\n" +
			"1  2\n" +
			"3  4\n"
		if got != want {
			t.Errorf("FormatTable() =\n%q\n\nwant:\n%q", got, want)
		}
	})

	t.Run("single column table", func(t *testing.T) {
		headers := []string{"Item"}
		rows := [][]string{
			{"Apple"},
			{"Banana"},
			{"Cherry"},
		}
		got := FormatTable(headers, rows)
		want := "Item  \n" +
			"------\n" +
			"Apple \n" +
			"Banana\n" +
			"Cherry\n"
		if got != want {
			t.Errorf("FormatTable() =\n%q\n\nwant:\n%q", got, want)
		}
	})

	t.Run("wide content", func(t *testing.T) {
		headers := []string{"Short", "Very Long Header"}
		rows := [][]string{
			{"A", "This is a very long string that is wider than the header"},
			{"B", "Short"},
		}
		got := FormatTable(headers, rows)
		want := "Short  Very Long Header                                        \n" +
			"-----  --------------------------------------------------------\n" +
			"A      This is a very long string that is wider than the header\n" +
			"B      Short                                                   \n"
		if got != want {
			t.Errorf("FormatTable() =\n%q\n\nwant:\n%q", got, want)
		}
	})

	t.Run("empty headers", func(t *testing.T) {
		headers := []string{}
		rows := [][]string{{"A", "B"}}
		got := FormatTable(headers, rows)
		if got != "" {
			t.Errorf("FormatTable() with empty headers = %q, want empty string", got)
		}
	})
}
