package display

import (
	"testing"
	"time"
)

func TestTruncate(t *testing.T) {
	// Existing ASCII test cases
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

	// Unicode test cases
	unicodeTests := []struct {
		input  string
		maxLen int
		want   string
	}{
		// Multi-byte UTF-8: combining diacritical marks
		{"café\u0301", 10, "café\u0301"},    // 5 runes, fits in 10
		{"café\u0301", 4, "c..."},           // 5 runes, truncate to 4 (1 rune + "...")
		// Emoji (multi-byte)
		{"😀", 10, "😀"},                    // Single emoji fits in 10
		{"😀😀😀😀😀", 4, "😀..."},         // 5 emojis, truncate to 4 runes
		// CJK characters
		{"你好世界", 10, "你好世界"},         // 4 runes, fits in 10
		{"你好世界北京", 5, "你好..."},      // 6 runes, truncate to 5 (2 runes + "...")
		// Mixed ASCII and multi-byte
		{"hello😀world", 10, "hello😀w..."}, // 11 runes, truncate to 10
		{"hi😀", 10, "hi😀"},                // 3 runes, fits in 10
		// Edge case: exactly maxLen runes
		{"exactly", 7, "exactly"},
		{"12345", 5, "12345"},
		{"你好", 2, "你好"},
	}
	for _, tt := range unicodeTests {
		got := Truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
		// Verify rune count matches or is truncated correctly
		gotRunes := len([]rune(got))
		if gotRunes > tt.maxLen {
			t.Errorf("Truncate(%q, %d) rune count = %d, exceeds maxLen %d", tt.input, tt.maxLen, gotRunes, tt.maxLen)
		}
	}
}

func TestTruncateWithPadding(t *testing.T) {
	// Existing ASCII test cases (check byte length for backward compatibility)
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

	// Unicode test cases (check rune count)
	unicodeTests := []struct {
		input  string
		maxLen int
		want   string
	}{
		// Multi-byte UTF-8: combining diacritical marks
		{"café\u0301", 10, "café\u0301     "}, // 5 runes + 5 spaces = 10 runes
		{"café\u0301", 5, "café\u0301"},       // Exactly 5 runes, no padding
		// Emoji (multi-byte)
		{"😀", 5, "😀    "},           // 1 emoji + 4 spaces = 5 runes
		{"😀😀😀😀😀", 4, "😀..."}, // 5 emojis, truncate to 4 runes
		// CJK characters
		{"你好", 5, "你好   "},              // 2 CJK + 3 spaces = 5 runes
		{"你好世界北京", 5, "你好..."},      // 6 CJK, truncate to 5 runes
		// Mixed ASCII and multi-byte
		{"hi😀", 5, "hi😀  "},         // 3 runes + 2 spaces = 5 runes
		{"hello😀world", 10, "hello😀w..."}, // 11 runes, truncate to 10
		// Edge case: exactly maxLen runes
		{"exact", 5, "exact"},          // Exactly 5 ASCII runes, no padding
		{"你好", 2, "你好"},           // Exactly 2 CJK runes, no padding
	}
	for _, tt := range unicodeTests {
		got := TruncateWithPadding(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("TruncateWithPadding(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
		// Verify rune count is exactly maxLen
		gotRunes := len([]rune(got))
		if gotRunes != tt.maxLen {
			t.Errorf("TruncateWithPadding(%q, %d) rune count = %d, want %d", tt.input, tt.maxLen, gotRunes, tt.maxLen)
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
