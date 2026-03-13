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

		// Strings with ANSI color codes (codes don't count)
		{"\x1b[31mred text\x1b[0m", 8},          // "red text" = 8 chars
		{"\x1b[1;32mbold green\x1b[0m", 10},     // "bold green" = 10 chars
		{"\x1b[33myellow\x1b[0m", 6},            // "yellow" = 6 chars
		{"\x1b[34mblue\x1b[0m hello", 10},       // "blue hello" = 10 chars

		// Unicode strings (CJK and emoji)
		{"你好", 2},                              // Chinese: 2 runes = 2 visual chars
		{"こんにちは", 5},                         // Japanese: 5 runes
		{"😀", 1},                                // Emoji: 1 rune
		{"hello😀world", 11},                     // Mixed ASCII and emoji (5 + 1 + 5)
		{"日本語", 3},                             // Japanese: 3 runes

		// Complex combinations
		{"\x1b[31m你好\x1b[0m", 2},               // ANSI + CJK
		{"\x1b[1;32m😀\x1b[0m", 1},               // ANSI + emoji
		{"\x1b[33mhello世界\x1b[0m", 7},          // ANSI + mixed
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
		wantLen int // visual width of result
	}{
		// Plain ASCII: already exact width
		{"hello", 5, "hello", 5},

		// Plain ASCII: needs padding
		{"hi", 5, "hi   ", 5},
		{"hello", 10, "hello     ", 10},
		{"", 5, "     ", 5},

		// Plain ASCII: already wider than target (no padding)
		{"hello world", 5, "hello world", 11},
		{"long string", 3, "long string", 11},

		// ANSI codes: padding calculation ignores ANSI
		{"\x1b[31mred\x1b[0m", 5, "\x1b[31mred\x1b[0m  ", 5},   // "red" (3) + 2 spaces
		{"\x1b[1;32mbold\x1b[0m", 10, "\x1b[1;32mbold\x1b[0m      ", 10}, // "bold" (4) + 6 spaces

		// ANSI codes: already wider than target
		{"\x1b[33mlongstring\x1b[0m", 5, "\x1b[33mlongstring\x1b[0m", 10},

		// Unicode: padding
		{"你好", 5, "你好   ", 5},    // 2 chars + 3 spaces
		{"😀", 3, "😀  ", 3},        // 1 emoji + 2 spaces

		// Unicode: already wider
		{"你好世界", 2, "你好世界", 4},  // 4 chars, don't truncate

		// Complex: ANSI + Unicode
		{"\x1b[31m你好\x1b[0m", 5, "\x1b[31m你好\x1b[0m   ", 5},
		{"\x1b[1;32m😀\x1b[0m", 4, "\x1b[1;32m😀\x1b[0m   ", 4},

		// Edge cases
		{"a", 1, "a", 1},
		{"a", 10, "a         ", 10},
	}
	for _, tt := range tests {
		got := PadToWidth(tt.input, tt.width)
		if got != tt.want {
			t.Errorf("PadToWidth(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
		}
		// Verify visual width of result
		gotWidth := VisualWidth(got)
		if gotWidth != tt.wantLen {
			t.Errorf("PadToWidth(%q, %d) visual width = %d, want %d", tt.input, tt.width, gotWidth, tt.wantLen)
		}
	}
}
