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
		{"", 0},
		{"hello", 5},
		{"a", 1},
		{"hello world", 11},

		// Strings with ANSI color codes (codes should not count)
		{"\x1b[31mred\x1b[0m", 3},                    // 3 visible chars + ANSI codes
		{"\x1b[1;32mbold green\x1b[0m", 10},          // "bold green" = 10 chars
		{"\x1b[31mh\x1b[0m\x1b[32me\x1b[0m\x1b[34ml\x1b[0m\x1b[33ml\x1b[0m\x1b[35mo\x1b[0m", 5}, // "hello" with individual char coloring
		{"no ansi here", 12},

		// Unicode strings (counted in runes, not bytes)
		{"こんにちは", 5},                   // 5 CJK characters
		{"café", 4},                        // 4 runes (é is 1 rune)
		{"hello 世界", 8},                   // 5 + space + 2 = 8 runes
		{"🎉", 1},                          // emoji = 1 rune
		{"hello🎉world", 11},               // 5 + 1 + 5 = 11 runes
		{"Hello, 🌍!", 9},                  // H-e-l-l-o-,-space-🌍-! = 9 runes

		// ANSI codes with Unicode
		{"\x1b[31m日本語\x1b[0m", 3},           // 3 Japanese characters with color
		{"\x1b[1;33mこんにちは\x1b[0m", 5},     // 5 CJK chars with bold yellow
		{"\x1b[32m🎉\x1b[0m", 1},              // 1 emoji with color
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
		check  func(string) bool // optional validation function
	}{
		// Plain strings that need padding
		{"", 5, "     ", nil},
		{"a", 5, "a    ", nil},
		{"hi", 5, "hi   ", nil},
		{"hello", 10, "hello     ", nil},

		// Strings already at target width
		{"hello", 5, "hello", nil},
		{"abc", 3, "abc", nil},

		// Strings wider than target (unchanged)
		{"hello world", 5, "hello world", nil},
		{"abcdefghij", 3, "abcdefghij", nil},

		// ANSI codes with padding (codes preserved, don't count toward width)
		{"\x1b[31mred\x1b[0m", 6, "\x1b[31mred\x1b[0m   ", nil}, // "red" is 3 chars, pad to 6 with 3 spaces
		{"\x1b[32mok\x1b[0m", 5, "\x1b[32mok\x1b[0m   ", nil},    // "ok" is 2 chars, pad to 5 with 3 spaces

		// Unicode strings with padding
		{"café", 8, "café    ", nil},           // 4 runes, pad to 8
		{"こんにちは", 8, "こんにちは   ", nil},       // 5 runes, pad to 8 with 3 spaces
		{"🎉", 3, "🎉  ", nil},                // 1 rune (emoji), pad to 3

		// ANSI codes with Unicode
		{"\x1b[33mこんにちは\x1b[0m", 8, "\x1b[33mこんにちは\x1b[0m   ", nil}, // 5 CJK chars + ANSI, pad to 8
		{"\x1b[31m🎉\x1b[0m", 5, "\x1b[31m🎉\x1b[0m    ", nil},          // 1 emoji + ANSI, pad to 5

		// Complex ANSI with multiple color codes
		{"\x1b[1;31mtest\x1b[0m", 10, "\x1b[1;31mtest\x1b[0m      ", nil}, // "test" is 4, pad to 10 with 6 spaces
	}

	for _, tt := range tests {
		got := PadToWidth(tt.input, tt.width)
		if got != tt.want {
			t.Errorf("PadToWidth(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
		}
		// Verify visual width matches or exceeds target
		visualWidth := VisualWidth(got)
		if visualWidth < tt.width {
			t.Errorf("PadToWidth(%q, %d) result has visual width %d, expected at least %d", tt.input, tt.width, visualWidth, tt.width)
		}
	}
}

