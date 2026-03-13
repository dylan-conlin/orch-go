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
		{"\x1b[31mred\x1b[0m", 3},                    // "red" is 3 chars
		{"\x1b[1;32mbold green\x1b[0m", 10},         // "bold green" is 10 chars
		{"\x1b[0mno color\x1b[0m", 8},               // "no color" is 8 chars
		{"\x1b[38;5;208morange\x1b[0m", 6},          // "orange" is 6 chars

		// Unicode strings
		{"你好", 2},                 // 2 Chinese characters
		{"🎉", 1},                  // 1 emoji (counted as 1 rune)
		{"hello你好", 7},            // 5 ASCII + 2 Chinese = 7 runes
		{"مرحبا", 5},               // 5 Arabic characters

		// Mixed: Unicode with ANSI codes
		{"\x1b[31m你好\x1b[0m", 2},  // "你好" is 2 chars after ANSI stripping
		{"\x1b[1;35m🎉\x1b[0m", 1},  // 1 emoji with formatting

		// Edge cases
		{"   ", 3},                 // spaces
		{"\t", 1},                  // tab character
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
		{"hello", 5, "hello"},
		{"", 5, "     "},
		{"a", 3, "a  "},

		// Already at exact width
		{"exact", 5, "exact"},

		// Already wider than target (should return unchanged)
		{"hello world", 5, "hello world"},
		{"toolong", 3, "toolong"},

		// Strings with ANSI color codes
		{"\x1b[31mred\x1b[0m", 5, "\x1b[31mred\x1b[0m  "},        // "red" → pad to 5
		{"\x1b[31mred\x1b[0m", 3, "\x1b[31mred\x1b[0m"},          // "red" exact
		{"\x1b[1;32mbold green\x1b[0m", 15, "\x1b[1;32mbold green\x1b[0m     "}, // 10→15

		// Unicode strings
		{"你好", 4, "你好  "},     // 2→4
		{"🎉", 3, "🎉  "},       // 1→3
		{"hello你好", 10, "hello你好   "},  // 7→10

		// Mixed: Unicode with ANSI codes
		{"\x1b[31m你好\x1b[0m", 4, "\x1b[31m你好\x1b[0m  "},
		{"\x1b[1;35m🎉\x1b[0m", 3, "\x1b[1;35m🎉\x1b[0m  "},
	}
	for _, tt := range tests {
		got := PadToWidth(tt.input, tt.width)
		if got != tt.want {
			t.Errorf("PadToWidth(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
		}
		// Verify visual width of result
		gotWidth := VisualWidth(got)
		if gotWidth < tt.width && VisualWidth(tt.input) < tt.width {
			t.Errorf("PadToWidth(%q, %d) result visual width = %d, want at least %d", tt.input, tt.width, gotWidth, tt.width)
		}
	}
}
