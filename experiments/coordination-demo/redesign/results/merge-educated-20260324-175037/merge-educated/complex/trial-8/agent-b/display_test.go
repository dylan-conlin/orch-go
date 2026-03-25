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
		headers := []string{"ID", "Status", "Duration"}
		rows := [][]string{
			{"ses_abc123", "active", "2m 30s"},
			{"ses_def456", "done", "5h 12m"},
		}
		got := FormatTable(headers, rows)
		want := "ID           Status   Duration\n" +
			"----------   ------   --------\n" +
			"ses_abc123   active   2m 30s  \n" +
			"ses_def456   done     5h 12m  \n"
		if got != want {
			t.Errorf("FormatTable() mismatch:\nGot:\n%s\nWant:\n%s", got, want)
		}
	})

	t.Run("empty table (headers only)", func(t *testing.T) {
		headers := []string{"Name", "Value"}
		rows := [][]string{}
		got := FormatTable(headers, rows)
		want := "Name   Value\n" +
			"----   -----\n"
		if got != want {
			t.Errorf("FormatTable() empty table mismatch:\nGot:\n%s\nWant:\n%s", got, want)
		}
	})

	t.Run("nil rows", func(t *testing.T) {
		headers := []string{"Col1", "Col2"}
		got := FormatTable(headers, nil)
		want := "Col1   Col2\n" +
			"----   ----\n"
		if got != want {
			t.Errorf("FormatTable() nil rows mismatch:\nGot:\n%s\nWant:\n%s", got, want)
		}
	})

	t.Run("ANSI colored content", func(t *testing.T) {
		headers := []string{"Name", "Color"}
		rows := [][]string{
			{"Alice", "\x1b[31mRed\x1b[0m"},
			{"Bob", "\x1b[32mGreen\x1b[0m"},
		}
		got := FormatTable(headers, rows)
		// Columns should align correctly despite ANSI codes
		want := "Name    Color\n" +
			"-----   -----\n" +
			"Alice   \x1b[31mRed\x1b[0m  \n" +
			"Bob     \x1b[32mGreen\x1b[0m\n"
		if got != want {
			t.Errorf("FormatTable() ANSI mismatch:\nGot:\n%s\nWant:\n%s", got, want)
		}
	})

	t.Run("mismatched column counts - fewer columns", func(t *testing.T) {
		headers := []string{"A", "B", "C"}
		rows := [][]string{
			{"1", "2", "3"},
			{"4", "5"}, // Missing third column
			{"7"},      // Missing second and third columns
		}
		got := FormatTable(headers, rows)
		want := "A   B   C\n" +
			"-   -   -\n" +
			"1   2   3\n" +
			"4   5    \n" +
			"7        \n"
		if got != want {
			t.Errorf("FormatTable() fewer columns mismatch:\nGot:\n%s\nWant:\n%s", got, want)
		}
	})

	t.Run("mismatched column counts - more columns", func(t *testing.T) {
		headers := []string{"A", "B"}
		rows := [][]string{
			{"1", "2"},
			{"3", "4", "extra", "ignored"}, // Extra columns ignored
		}
		got := FormatTable(headers, rows)
		want := "A   B\n" +
			"-   -\n" +
			"1   2\n" +
			"3   4\n"
		if got != want {
			t.Errorf("FormatTable() more columns mismatch:\nGot:\n%s\nWant:\n%s", got, want)
		}
	})

	t.Run("single column table", func(t *testing.T) {
		headers := []string{"Numbers"}
		rows := [][]string{
			{"1"},
			{"42"},
			{"999"},
		}
		got := FormatTable(headers, rows)
		want := "Numbers\n" +
			"-------\n" +
			"1      \n" +
			"42     \n" +
			"999    \n"
		if got != want {
			t.Errorf("FormatTable() single column mismatch:\nGot:\n%s\nWant:\n%s", got, want)
		}
	})

	t.Run("wide content", func(t *testing.T) {
		headers := []string{"Short", "VeryLongHeaderName"}
		rows := [][]string{
			{"a", "b"},
			{"c", "this is a very long cell value"},
		}
		got := FormatTable(headers, rows)
		want := "Short   VeryLongHeaderName            \n" +
			"-----   ------------------------------\n" +
			"a       b                             \n" +
			"c       this is a very long cell value\n"
		if got != want {
			t.Errorf("FormatTable() wide content mismatch:\nGot:\n%s\nWant:\n%s", got, want)
		}
	})

	t.Run("empty headers", func(t *testing.T) {
		headers := []string{}
		rows := [][]string{{"a", "b"}}
		got := FormatTable(headers, rows)
		want := ""
		if got != want {
			t.Errorf("FormatTable() empty headers should return empty string, got: %q", got)
		}
	})
}
