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
		// Plain ASCII
		{"hello", 5},
		{"", 0},
		{"a", 1},
		{"hello world", 11},

		// ANSI codes (should be stripped)
		{"\x1b[31mred\x1b[0m", 3},           // "red"
		{"\x1b[1;32mbold green\x1b[0m", 10}, // "bold green"
		{"\x1b[0m\x1b[0m", 0},               // only ANSI codes

		// Unicode: emoji and CJK characters (each counts as 1 rune)
		{"Hello 世界", 8},           // 5 ASCII + space + 2 CJK
		{"🎉", 1},                  // emoji is 1 rune
		{"🎉🎊", 2},                // 2 emoji
		{"café", 4},                // 4 runes (é is 1 rune)
		{"\x1b[35m🎉\x1b[0m", 1},   // emoji with color code

		// Combined: ANSI + unicode
		{"\x1b[1;33m日本語\x1b[0m", 3}, // "日本語" = 3 runes
		{"\x1b[31mHello\x1b[0m \x1b[32m世界\x1b[0m", 8}, // "Hello 世界"
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
		wantLen int // visual width of result (ignoring ANSI)
	}{
		// Plain ASCII: needs padding
		{"hi", 5, "hi   ", 5},
		{"", 3, "   ", 3},
		{"x", 1, "x", 1},

		// Already at target width
		{"hello", 5, "hello", 5},
		{"abc", 3, "abc", 3},

		// Already wider than target (no padding)
		{"hello", 3, "hello", 5},
		{"world", 2, "world", 5},

		// ANSI codes preserved, padding added to reach visual width
		{"\x1b[31mred\x1b[0m", 5, "\x1b[31mred\x1b[0m  ", 5}, // "red" + 2 spaces
		{"\x1b[1;32mbold\x1b[0m", 6, "\x1b[1;32mbold\x1b[0m  ", 6}, // "bold" + 2 spaces

		// Unicode with padding
		{"世", 3, "世  ", 3},
		{"café", 6, "café  ", 6},

		// Combined: ANSI + unicode + padding
		{"\x1b[35m日本\x1b[0m", 4, "\x1b[35m日本\x1b[0m  ", 4}, // "日本" (2 runes) + 2 spaces
	}
	for _, tt := range tests {
		got := PadToWidth(tt.input, tt.width)
		if got != tt.want {
			t.Errorf("PadToWidth(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
		}
		// Verify visual width of result
		if VisualWidth(got) != tt.wantLen {
			t.Errorf("PadToWidth(%q, %d) visual width = %d, want %d", tt.input, tt.width, VisualWidth(got), tt.wantLen)
		}
	}
}
