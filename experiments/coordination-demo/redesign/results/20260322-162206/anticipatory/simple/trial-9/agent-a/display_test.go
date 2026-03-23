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

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		// Zero
		{0, "0 B"},

		// Small bytes (less than 1 KiB)
		{1, "1 B"},
		{512, "512 B"},
		{1023, "1023 B"},

		// Exact KiB boundary and multiples
		{1024, "1.0 KiB"},
		{2 * 1024, "2.0 KiB"},
		{10 * 1024, "10.0 KiB"},

		// Values between KiB boundaries
		{1536, "1.5 KiB"},
		{1024 + 256, "1.2 KiB"},
		{512 * 1024, "512.0 KiB"},

		// Exact MiB boundary and multiples
		{1048576, "1.0 MiB"},
		{2 * 1048576, "2.0 MiB"},
		{100 * 1048576, "100.0 MiB"},

		// Values between MiB boundaries
		{1048576 + 524288, "1.5 MiB"},
		{1048576 + 209715, "1.2 MiB"},
		{512 * 1048576, "512.0 MiB"},

		// Exact GiB boundary and multiples
		{1073741824, "1.0 GiB"},
		{2 * 1073741824, "2.0 GiB"},

		// Values between GiB boundaries
		{1073741824 + 536870912, "1.5 GiB"},
		{512 * 1073741824, "512.0 GiB"},

		// Exact TiB boundary and multiples
		{1099511627776, "1.0 TiB"},
		{2 * 1099511627776, "2.0 TiB"},

		// Large TiB values
		{1099511627776 + 549755813888, "1.5 TiB"},
		{10 * 1099511627776, "10.0 TiB"},

		// Negative values - small bytes
		{-1, "-1 B"},
		{-512, "-512 B"},

		// Negative values - KiB
		{-1024, "-1.0 KiB"},
		{-1536, "-1.5 KiB"},

		// Negative values - MiB
		{-1048576, "-1.0 MiB"},
		{-1048576 - 524288, "-1.5 MiB"},

		// Negative values - GiB
		{-1073741824, "-1.0 GiB"},

		// Negative values - TiB
		{-1099511627776, "-1.0 TiB"},
	}
	for _, tt := range tests {
		got := FormatBytes(tt.input)
		if got != tt.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
