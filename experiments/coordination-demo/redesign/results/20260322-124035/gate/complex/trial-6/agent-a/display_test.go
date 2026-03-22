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

		// Strings with ANSI color codes
		{"\x1b[31mred text\x1b[0m", 8},                          // "red text" = 8 chars
		{"\x1b[1;32mbold green\x1b[0m", 10},                     // "bold green" = 10 chars
		{"\x1b[33mhello\x1b[0m \x1b[34mworld\x1b[0m", 11},       // "hello world" = 11 chars
		{"\x1b[0m", 0},                                           // Only ANSI codes
		{"\x1b[1m\x1b[2m\x1b[3mtest\x1b[0m", 4},                // "test" with multiple codes

		// Unicode strings (CJK characters)
		{"你好", 2},                                              // 2 Chinese characters
		{"こんにちは", 5},                                         // 5 Japanese characters
		{"안녕하세요", 5},                                         // 5 Korean characters

		// Emoji (each emoji is typically 1 rune in Go)
		{"😀", 1},
		{"hello 😀 world", 13},

		// Mixed: ANSI + Unicode
		{"\x1b[31m你好\x1b[0m", 2},                              // 2 Chinese chars
		{"\x1b[32m😀\x1b[0m", 1},                                // 1 emoji
		{"\x1b[33mhello\x1b[0m 世界", 8},                         // "hello 世界" = 5 + 1 + 2 = 8

		// Edge cases
		{" ", 1},
		{"   ", 3},
		{"\t", 1},
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
		input    string
		width    int
		want     string
		wantLen  int // visual width of result
	}{
		// Plain ASCII padding
		{"hello", 10, "hello     ", 10},
		{"hi", 5, "hi   ", 5},
		{"", 3, "   ", 3},
		{"a", 1, "a", 1},

		// Already at target width
		{"hello", 5, "hello", 5},
		{"abc", 3, "abc", 3},

		// Already wider than target width
		{"hello world", 5, "hello world", 11},
		{"longer string", 3, "longer string", 13},

		// With ANSI codes (codes don't count)
		{"\x1b[31mred\x1b[0m", 5, "\x1b[31mred\x1b[0m  ", 5},    // "red" is 3 chars, pad to 5
		{"\x1b[32mtest\x1b[0m", 6, "\x1b[32mtest\x1b[0m  ", 6}, // "test" is 4 chars, pad to 6

		// With Unicode (counting runes, not bytes)
		{"你", 3, "你  ", 3},     // 1 rune, pad to 3
		{"你好", 4, "你好  ", 4}, // 2 runes, pad to 4

		// Mixed: ANSI + Unicode
		{"\x1b[31m你\x1b[0m", 3, "\x1b[31m你\x1b[0m  ", 3}, // 1 Chinese char, pad to 3
		{"\x1b[33m😀\x1b[0m", 3, "\x1b[33m😀\x1b[0m  ", 3}, // 1 emoji, pad to 3
	}
	for _, tt := range tests {
		got := PadToWidth(tt.input, tt.width)
		if got != tt.want {
			t.Errorf("PadToWidth(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
		}
		gotVisualWidth := VisualWidth(got)
		if gotVisualWidth != tt.wantLen {
			t.Errorf("PadToWidth(%q, %d) visual width = %d, want %d", tt.input, tt.width, gotVisualWidth, tt.wantLen)
		}
	}
}
