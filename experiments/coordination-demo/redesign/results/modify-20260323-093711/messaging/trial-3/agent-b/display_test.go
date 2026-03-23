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
		// ASCII cases (existing)
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello..."},
		{"abcdefghij", 6, "abc..."},
		{"", 5, ""},
		// Unicode cases (multi-byte UTF-8)
		{"café", 10, "café"},                                    // 4 runes, fits
		{"café", 4, "café"},                                     // 4 runes, exact fit
		{"café", 3, "..."},                                      // 4 runes, truncate to "..." (0 chars + 3 dots)
		{"cafe\u0301", 10, "cafe\u0301"},                        // combining character, fits
		{"你好世界", 6, "你好世界"},                              // 4 runes, fits
		{"你好世界", 4, "你好世界"},                              // 4 runes, exact fit
		{"你好世界", 3, "..."},                                   // 4 runes, truncate to "..."
		{"Hello 🌍", 10, "Hello 🌍"},                            // 7 runes, fits
		{"Hello 🌍", 7, "Hello 🌍"},                             // 7 runes, exact fit
		{"Hello 🌍 World", 10, "Hello 🌍..."},                   // 13 runes, truncate to 10 runes (7 runes + ...)
		{"Hello 🌍 World", 9, "Hello ..."},                      // 13 runes, truncate to 9 runes (6 runes + ...)
		{"😀😁😂", 5, "😀😁😂"},                                 // 3 runes, fits
		{"😀😁😂", 3, "😀😁😂"},                                 // 3 runes, exact fit
		{"😀😁😂", 4, "😀😁😂"},                                 // 3 runes, fits in 4
		{"😀😁😂", 2, "😀😁"},                                   // 3 runes, truncate to 2 (no room for "...")
		// Edge cases
		{"abc", 4, "abc"},                                       // 3 runes, fits
		{"abc", 3, "abc"},                                       // 3 runes, exact fit
		{"abc", 2, "ab"},                                        // 3 runes, truncate to 2
		{"abc", 1, "a"},                                         // 3 runes, truncate to 1
		{"abc", 0, ""},                                          // 3 runes, truncate to 0
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
		// ASCII cases (existing)
		{"hello", 10, "hello     "},
		{"hello world", 8, "hello..."},
		{"exact", 5, "exact"},
		// Unicode cases (multi-byte UTF-8)
		{"café", 8, "café    "},                                 // 4 runes + 4 spaces = 8 runes
		{"café", 4, "café"},                                     // 4 runes, exact fit
		{"你好", 6, "你好    "},                                  // 2 runes + 4 spaces = 6 runes
		{"你好世界", 5, "你好世界 "},                              // 4 runes (< 5), no truncate, pad 1 space
		{"你好世界", 4, "你好世界"},                              // 4 runes, exact fit
		{"你好世界", 3, "..."},                                   // 4 runes (> 3), truncate to (3-3)=0 chars + "..."
		{"Hello 🌍", 10, "Hello 🌍   "},                         // 7 runes + 3 spaces = 10 runes
		{"Hello 🌍", 7, "Hello 🌍"},                             // 7 runes, exact fit
		{"Hello 🌍 World", 13, "Hello 🌍 World"},                // 13 runes, exact fit
		{"Hello 🌍 World", 10, "Hello 🌍..."},                   // 13 runes (> 10), truncate to (10-3)=7 runes + "..." (same as Truncate)
		{"😀😁😂", 5, "😀😁😂  "},                                // 3 runes + 2 spaces = 5 runes
		{"😀😁😂", 3, "😀😁😂"},                                 // 3 runes, exact fit
		{"😀😁😂", 4, "😀😁😂 "},                                // 3 runes + 1 space = 4 runes
		{"😀😁😂", 2, "😀😁"},                                   // 3 runes (> 2), truncate to 2 (no room for "...")
		// Edge cases
		{"hi", 5, "hi   "},                                      // 2 runes + 3 spaces = 5 runes
		{"", 3, "   "},                                          // 0 runes + 3 spaces = 3 runes
	}
	for _, tt := range tests {
		got := TruncateWithPadding(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("TruncateWithPadding(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
		// Check rune count (including padding)
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
