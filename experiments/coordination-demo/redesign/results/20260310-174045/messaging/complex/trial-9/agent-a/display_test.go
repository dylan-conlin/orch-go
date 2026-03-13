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

func TestVisualWidth(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		// Plain ASCII strings
		{"hello", 5},
		{"hello world", 11},
		{"", 0},
		{"a", 1},
		// ANSI color codes
		{"\x1b[31mred text\x1b[0m", 8},      // "red text" = 8 chars
		{"\x1b[1;32mbold green\x1b[0m", 10}, // "bold green" = 10 chars
		{"\x1b[0m", 0},                      // Just codes
		// Unicode - CJK characters
		{"你好", 2},           // 2 CJK chars
		{"Hello世界", 7},     // 5 ASCII + 2 CJK
		{"一二三四五", 5},     // 5 CJK chars
		// Unicode - Emoji
		{"😀", 1},              // Single emoji = 1 rune
		{"Hello😀World", 11},  // 5 + 1 + 5 = 11 chars
		{"🔴🔵🟢", 3},         // 3 emoji
		// Mixed with ANSI
		{"\x1b[32m你好\x1b[0m", 2}, // ANSI + CJK
		{"\x1b[31m😀\x1b[0m", 1},   // ANSI + emoji
	}
	for _, tt := range tests {
		got := VisualWidth(tt.input)
		if got != tt.want {
			t.Errorf("VisualWidth(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestPadToWidth(t *testing.T) {
	tests := []struct {
		input  string
		width  int
		want   string
	}{
		// Plain ASCII strings
		{"hello", 5, "hello"},              // Already at width
		{"hello", 8, "hello   "},           // Pad to 8 (3 spaces)
		{"hi", 5, "hi   "},                 // Pad to 5 (3 spaces)
		{"", 3, "   "},                     // Empty string padded
		{"x", 1, "x"},                      // Single char at width
		// Wider than target (no padding)
		{"hello world", 5, "hello world"},  // Wider, return unchanged
		{"abcdefgh", 3, "abcdefgh"},        // Much wider, return unchanged
		// ANSI color codes - codes don't count toward width
		{"\x1b[31mhello\x1b[0m", 5, "\x1b[31mhello\x1b[0m"},     // Already 5 visual
		{"\x1b[31mhi\x1b[0m", 5, "\x1b[31mhi\x1b[0m   "},       // 2 visual + codes, pad to 5 (3 spaces)
		{"\x1b[1;32mtest\x1b[0m", 4, "\x1b[1;32mtest\x1b[0m"},  // 4 visual, at target
		{"\x1b[1;32mtest\x1b[0m", 6, "\x1b[1;32mtest\x1b[0m  "}, // 4 visual, pad to 6 (2 spaces)
		// Unicode - CJK
		{"你好", 2, "你好"},              // Already 2 visual
		{"你好", 5, "你好   "},           // 2 visual, pad to 5 (3 spaces)
		{"ab你好cd", 6, "ab你好cd"},      // 6 visual, at target
		{"ab你好cd", 8, "ab你好cd  "},    // 6 visual, pad to 8 (2 spaces)
		// Unicode - Emoji
		{"😀", 1, "😀"},                   // Already 1 visual
		{"😀", 3, "😀  "},                 // 1 visual, pad to 3 (2 spaces)
		// Mixed - ANSI + Unicode
		{"\x1b[31m你好\x1b[0m", 2, "\x1b[31m你好\x1b[0m"},      // ANSI + 2 CJK, at width
		{"\x1b[31m你好\x1b[0m", 5, "\x1b[31m你好\x1b[0m   "},   // ANSI + 2 CJK, pad to 5
		{"\x1b[32m😀\x1b[0m", 1, "\x1b[32m😀\x1b[0m"},         // ANSI + 1 emoji, at width
		{"\x1b[32m😀\x1b[0m", 4, "\x1b[32m😀\x1b[0m   "},      // ANSI + 1 emoji, pad to 4
	}
	for _, tt := range tests {
		got := PadToWidth(tt.input, tt.width)
		if got != tt.want {
			t.Errorf("PadToWidth(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
		}
		// Verify visual width after padding
		visualWidth := VisualWidth(got)
		if visualWidth < tt.width {
			t.Errorf("PadToWidth(%q, %d) resulted in visual width %d, want at least %d", tt.input, tt.width, visualWidth, tt.width)
		}
	}
}
