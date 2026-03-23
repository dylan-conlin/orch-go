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
		{"\x1b[31mred\x1b[0m", 3},                           // "red" is 3 chars
		{"\x1b[1;32mbold green\x1b[0m", 10},                 // "bold green" is 10 chars
		{"\x1b[33myellow\x1b[0m text", 11},                  // "yellow text" is 11 chars
		{"\x1b[48;5;196mbackground\x1b[0m", 10},             // "background" is 10 chars
		// Unicode: CJK characters (each is 1 rune, but may display as 2 columns)
		{"你好", 2},                                           // 2 Chinese characters = 2 runes
		{"こんにちは", 5},                                      // 5 Japanese hiragana = 5 runes
		// Unicode: emoji (each is 1 rune)
		{"👋", 1},
		{"👋👋👋", 3},
		// Mixed ASCII and Unicode
		{"hello👋", 6},
		{"👋world", 6},
		// Unicode with ANSI codes
		{"\x1b[32m你好\x1b[0m", 2},
		{"\x1b[1m👋👋\x1b[0m", 2},
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
		{"hello", 4, "hello"},         // Already wider, return unchanged
		{"", 5, "     "},
		{"", 0, ""},
		// Strings with ANSI codes (codes are preserved)
		{"\x1b[31mred\x1b[0m", 5, "\x1b[31mred\x1b[0m  "},     // "red" needs 2 spaces to reach 5
		{"\x1b[32mok\x1b[0m", 2, "\x1b[32mok\x1b[0m"},          // "ok" already at 2, no padding
		{"\x1b[33myes\x1b[0m", 2, "\x1b[33myes\x1b[0m"},        // "yes" is wider than 2, no padding
		// Unicode strings
		{"👋", 5, "👋    "},           // 1 rune + 4 spaces = width 5
		{"你好", 4, "你好  "},          // 2 runes + 2 spaces = width 4
		// Unicode with ANSI codes
		{"\x1b[32m👋\x1b[0m", 3, "\x1b[32m👋\x1b[0m  "},
		// Edge case: exactly at width
		{"test", 4, "test"},
		// Edge case: mixed content
		{"hi\x1b[31m world\x1b[0m", 10, "hi\x1b[31m world\x1b[0m  "},  // "hi world" is 8 chars, needs 2 spaces
	}
	for _, tt := range tests {
		got := PadToWidth(tt.input, tt.width)
		if got != tt.want {
			t.Errorf("PadToWidth(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
		}
	}
}
