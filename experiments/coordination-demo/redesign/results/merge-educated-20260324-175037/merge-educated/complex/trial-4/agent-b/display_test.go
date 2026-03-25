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
			t.Errorf("FormatTable() =\n%q\nwant:\n%q", got, want)
		}
	})

	t.Run("empty table - headers only", func(t *testing.T) {
		headers := []string{"Name", "Age"}
		rows := [][]string{}
		got := FormatTable(headers, rows)
		want := "Name  Age\n" +
			"----  ---\n"
		if got != want {
			t.Errorf("FormatTable() =\n%q\nwant:\n%q", got, want)
		}
	})

	t.Run("ANSI colored content", func(t *testing.T) {
		headers := []string{"Status", "Message"}
		rows := [][]string{
			{"\x1b[32mOK\x1b[0m", "Success"},
			{"\x1b[31mERROR\x1b[0m", "Failed"},
		}
		got := FormatTable(headers, rows)
		// Status column width should be 6 (max of "Status" and "ERROR" without ANSI)
		want := "Status  Message\n" +
			"------  -------\n" +
			"\x1b[32mOK\x1b[0m      Success\n" +
			"\x1b[31mERROR\x1b[0m   Failed \n"
		if got != want {
			t.Errorf("FormatTable() =\n%q\nwant:\n%q", got, want)
		}
	})

	t.Run("mismatched column counts", func(t *testing.T) {
		headers := []string{"Name", "Age", "City"}
		rows := [][]string{
			{"Alice", "30"},     // Missing city
			{"Bob", "25", "SF"},
			{"Charlie"},         // Missing age and city
		}
		got := FormatTable(headers, rows)
		want := "Name     Age  City\n" +
			"-------  ---  ----\n" +
			"Alice    30       \n" +
			"Bob      25   SF  \n" +
			"Charlie           \n"
		if got != want {
			t.Errorf("FormatTable() =\n%q\nwant:\n%q", got, want)
		}
	})

	t.Run("single column", func(t *testing.T) {
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
			t.Errorf("FormatTable() =\n%q\nwant:\n%q", got, want)
		}
	})

	t.Run("wide content", func(t *testing.T) {
		headers := []string{"ID", "Description"}
		rows := [][]string{
			{"1", "This is a very long description that exceeds normal width"},
			{"2", "Short"},
		}
		got := FormatTable(headers, rows)
		// Description column width = 57 (length of long string)
		// "Description" = 11 chars, so 46 spaces of padding
		want := "ID  Description                                              \n" +
			"--  ---------------------------------------------------------\n" +
			"1   This is a very long description that exceeds normal width\n" +
			"2   Short                                                    \n"
		if got != want {
			t.Errorf("FormatTable() =\n%q\nwant:\n%q", got, want)
		}
	})

	t.Run("nil rows", func(t *testing.T) {
		headers := []string{"Name", "Value"}
		rows := [][]string{
			{"Alice", "100"},
			nil,
			{"Bob", "200"},
		}
		got := FormatTable(headers, rows)
		want := "Name   Value\n" +
			"-----  -----\n" +
			"Alice  100  \n" +
			"Bob    200  \n"
		if got != want {
			t.Errorf("FormatTable() =\n%q\nwant:\n%q", got, want)
		}
	})

	t.Run("empty headers", func(t *testing.T) {
		headers := []string{}
		rows := [][]string{
			{"data"},
		}
		got := FormatTable(headers, rows)
		want := ""
		if got != want {
			t.Errorf("FormatTable() = %q, want %q", got, want)
		}
	})
}
