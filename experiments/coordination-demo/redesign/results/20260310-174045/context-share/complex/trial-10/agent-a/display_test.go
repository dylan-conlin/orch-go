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
		{"", 0},
		{"a", 1},
		{"hello world", 11},
		// ANSI color codes (should be stripped)
		{"\x1b[31mred\x1b[0m", 3},                     // "red" without colors = 3
		{"\x1b[1;32mbold green\x1b[0m", 10},           // "bold green" without colors = 10
		{"\x1b[38;5;196mcomplex\x1b[0m", 7},          // "complex" = 7
		// Unicode strings (CJK and emoji)
		{"中文", 2},                           // 2 Chinese characters
		{"你好世界", 4},                      // 4 Chinese characters
		{"こんにちは", 5},                    // 5 Hiragana characters
		{"Hello🌟", 6},                      // 5 ASCII + 1 emoji = 6 runes
		// Combinations: ANSI codes with Unicode
		{"\x1b[31m中文\x1b[0m", 2},          // Chinese with colors
		{"\x1b[1;35mこんにちは\x1b[0m", 5}, // Hiragana with colors
		// Edge cases
		{"\x1b[0m", 0}, // Just reset code
		{"\x1b[31m\x1b[0m", 0}, // Colors with no text
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
		input string
		width int
		want  string
	}{
		// Plain ASCII strings
		{"hello", 10, "hello     "},
		{"hi", 5, "hi   "},
		{"", 5, "     "},
		{"exact", 5, "exact"},
		// Already at width
		{"hello", 5, "hello"},
		// Already wider than width
		{"hello world", 5, "hello world"},
		{"abcdefgh", 4, "abcdefgh"},
		// ANSI-colored strings (codes don't count toward width)
		{"\x1b[31mred\x1b[0m", 5, "\x1b[31mred\x1b[0m  "},     // "red" = 3, pad to 5
		{"\x1b[1;32mbold\x1b[0m", 6, "\x1b[1;32mbold\x1b[0m  "}, // "bold" = 4, pad to 6
		// Unicode strings
		{"中文", 5, "中文   "},      // 2 Chinese chars, pad to 5
		{"你好", 4, "你好  "},      // 2 Chinese chars, pad to 4
		{"こんにちは", 8, "こんにちは   "}, // 5 Hiragana, pad to 8
		// ANSI codes with Unicode
		{"\x1b[31m中\x1b[0m", 3, "\x1b[31m中\x1b[0m  "}, // 1 Chinese char = 1 width, pad to 3
		// Edge cases
		{"", 0, ""},
		{"x", 1, "x"},
		{"\x1b[0m", 0, "\x1b[0m"}, // Reset code only, width 0
	}
	for _, tt := range tests {
		got := PadToWidth(tt.input, tt.width)
		if got != tt.want {
			t.Errorf("PadToWidth(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
		}
		// Verify visual width is correct
		gotVisualWidth := VisualWidth(got)
		if gotVisualWidth < tt.width {
			t.Errorf("PadToWidth(%q, %d) visual width = %d, want >= %d", tt.input, tt.width, gotVisualWidth, tt.width)
		}
	}
}
