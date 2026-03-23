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
		// ASCII test cases (existing behavior)
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello..."},
		{"abcdefghij", 6, "abc..."},
		{"", 5, ""},
		// Unicode test cases
		{"café", 10, "café"},                    // Multi-byte UTF-8 (fits within limit)
		{"hello café", 8, "hello..."},          // Mixed ASCII and multi-byte truncated to 8 runes
		{"你好世界", 10, "你好世界"},             // CJK characters (4 runes, fits)
		{"你好世界你好", 4, "你..."},            // CJK truncated (6 runes → 4 runes)
		{"😀😁😂", 10, "😀😁😂"},                // Emoji (3 runes, fits)
		{"😀😁😂😃😅", 4, "😀..."},             // Emoji truncated (5 runes → 4 runes)
		{"abc", 3, "abc"},                      // Exactly at limit
		{"café", 5, "café"},                    // Exactly fits (4 runes, limit is 5)
		{"hello", 6, "hello"},                  // Fits with room to spare
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
		// ASCII test cases (existing behavior)
		{"hello", 10, "hello     "},
		{"hello world", 8, "hello..."},
		{"exact", 5, "exact"},
		// Unicode test cases
		{"café", 10, "café      "},           // Multi-byte UTF-8 padded (4 runes + 6 spaces = 10 runes)
		{"世", 5, "世    "},                  // CJK padded (1 rune + 4 spaces = 5 runes)
		{"😀", 3, "😀  "},                   // Emoji padded (1 rune + 2 spaces = 3 runes)
		{"你好世界你好", 4, "你..."},        // CJK truncated (6 runes → 4 max)
		{"hello 世界你好", 8, "hello..."},   // Mixed truncated (10 runes → 8 max)
		{"a", 1, "a"},                       // Exactly at limit
	}
	for _, tt := range tests {
		got := TruncateWithPadding(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("TruncateWithPadding(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
		// Check rune length, not byte length, since Unicode chars may be multi-byte
		if len([]rune(got)) != tt.maxLen {
			t.Errorf("TruncateWithPadding(%q, %d) rune length = %d, want %d", tt.input, tt.maxLen, len([]rune(got)), tt.maxLen)
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
