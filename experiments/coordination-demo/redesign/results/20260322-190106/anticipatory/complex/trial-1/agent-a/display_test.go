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

func TestVisualWidth(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		// Plain ASCII strings
		{"", 0},
		{"hello", 5},
		{"hello world", 11},
		{"a", 1},

		// Strings with ANSI color codes (codes shouldn't count)
		{"\x1b[31mred\x1b[0m", 3},
		{"\x1b[1;32mbold green\x1b[0m", 10},
		{"\x1b[31mhi\x1b[0m\x1b[32mbye\x1b[0m", 5},

		// Unicode strings (emoji, CJK)
		{"hello🌟", 6},        // 5 ASCII + 1 emoji rune = 6 total runes
		{"你好", 2},           // 2 CJK characters
		{"🎯🎪", 2},           // 2 emoji runes
		{"café", 4},           // accented character (é is 1 rune)
		{"こんにちは", 5},     // 5 Japanese hiragana characters

		// Mixed: ANSI codes with Unicode
		{"\x1b[31m你好\x1b[0m", 2},
		{"\x1b[32m🎯\x1b[0m", 1},

		// Edge cases
		{"  ", 2},           // spaces
		{"\t", 1},           // tab counts as 1 rune
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
		input             string
		width             int
		want              string
		checkVisualWidth  bool // only check visual width when we actually padded
	}{
		// Plain ASCII
		{"hello", 10, "hello     ", true},
		{"hi", 5, "hi   ", true},
		{"test", 4, "test", true},

		// Already at target width
		{"hello", 5, "hello", true},

		// Already wider than target (should return unchanged)
		{"hello world", 5, "hello world", false},
		{"longer", 3, "longer", false},

		// Empty string
		{"", 5, "     ", true},
		{"", 0, "", true},

		// With ANSI codes (codes don't count toward width, but are preserved)
		{"\x1b[31mred\x1b[0m", 5, "\x1b[31mred\x1b[0m  ", true}, // "red" is 3 chars, pad to 5 = 2 spaces
		{"\x1b[32mhi\x1b[0m", 4, "\x1b[32mhi\x1b[0m  ", true},   // "hi" is 2 chars, pad to 4 = 2 spaces

		// Unicode strings
		{"hello", 7, "hello  ", true},
		{"你好", 4, "你好  ", true},  // 2 CJK chars, pad to 4 = 2 spaces
		{"🎯", 3, "🎯  ", true},      // 1 emoji, pad to 3 = 2 spaces

		// Mixed ANSI and Unicode
		{"\x1b[31m你好\x1b[0m", 4, "\x1b[31m你好\x1b[0m  ", true}, // visual width is 2, pad to 4
	}
	for _, tt := range tests {
		got := PadToWidth(tt.input, tt.width)
		if got != tt.want {
			t.Errorf("PadToWidth(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
		}
		// Verify visual width is correct (only when we expect padding)
		if tt.checkVisualWidth {
			visualWidth := VisualWidth(got)
			if visualWidth != tt.width {
				t.Errorf("PadToWidth(%q, %d) visual width = %d, want %d", tt.input, tt.width, visualWidth, tt.width)
			}
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
