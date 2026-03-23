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
		{"\x1b[31mred text\x1b[0m", 8},          // "red text" = 8 chars
		{"\x1b[1;32mbold green\x1b[0m", 10},     // "bold green" = 10 chars
		{"\x1b[33myellow\x1b[0m", 6},            // "yellow" = 6 chars
		{"\x1b[0;35mmagenta\x1b[0m text", 12},   // "magenta text" = 12 chars

		// Unicode strings (emoji, CJK)
		{"你好", 2},                           // Two CJK characters
		{"hello你好", 7},                      // 5 ASCII + 2 CJK
		{"🎉", 1},                              // Emoji counts as one rune
		{"hello🎉world", 11},                  // 5 + 1 + 5
		{"日本語テスト", 6},                         // 6 Japanese characters

		// ANSI codes with Unicode
		{"\x1b[32m你好\x1b[0m", 2},             // Green CJK
		{"\x1b[31m🎉\x1b[0m", 1},               // Red emoji
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
		wantLen int // Length of result (useful for ANSI-containing strings)
	}{
		// Plain ASCII strings
		{"hello", 10, "hello     ", 10},
		{"hello", 5, "hello", 5},
		{"hello", 3, "hello", 5}, // Already wider than target

		// Empty string
		{"", 5, "     ", 5},
		{"", 0, "", 0},

		// Strings with ANSI color codes (codes don't count toward width)
		{"\x1b[31mred\x1b[0m", 10, "\x1b[31mred\x1b[0m       ", 24}, // "red" = 3 chars, visual width 10 needs 7 spaces, codes add 12 chars

		// Unicode strings
		{"你好", 5, "你好   ", 7},  // 2 CJK chars + 3 spaces
		{"🎉", 3, "🎉  ", 3},       // 1 emoji + 2 spaces
		{"hello世界", 10, "hello世界   ", 11}, // 5 ASCII + 2 CJK (visual=7) + 3 spaces = 11 chars

		// Already at width
		{"hello", 5, "hello", 5},

		// Empty string at width 0
		{"", 0, "", 0},
	}
	for _, tt := range tests {
		got := PadToWidth(tt.input, tt.width)
		if got != tt.want {
			t.Errorf("PadToWidth(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
		}
		// Visual width check
		visualWidth := VisualWidth(got)
		if visualWidth < tt.width && len(got) > 0 {
			t.Errorf("PadToWidth(%q, %d) visual width = %d, want at least %d", tt.input, tt.width, visualWidth, tt.width)
		}
	}
}
