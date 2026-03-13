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
		{"\x1b[31mred\x1b[0m", 3},                          // red (3 chars)
		{"\x1b[1;32mbold green\x1b[0m", 10},                // bold green (10 chars)
		{"\x1b[38;5;196merror\x1b[0m", 5},                  // 256-color red error (5 chars)
		{"prefix\x1b[35mmagenta\x1b[0m", 13},              // prefix (6) + magenta (7) = 13
		{"\x1b[0m\x1b[1m\x1b[31mx\x1b[0m", 1},              // x with multiple ANSI codes (1 char)

		// Unicode strings
		{"你好", 2},            // 2 Chinese characters
		{"hello你好", 7},       // 5 ASCII + 2 Chinese
		{"🎉", 1},             // 1 emoji
		{"hello🎉world", 11},  // 5 + 1 + 5
		{"café", 4},           // 4 chars with combining accent
		{"こんにちは", 5},      // 5 Japanese hiragana

		// Mixed ANSI and Unicode
		{"\x1b[32m你好\x1b[0m", 2},          // 2 Chinese with ANSI
		{"\x1b[35m🎉\x1b[0m", 1},           // 1 emoji with ANSI
		{"\x1b[1m你好world\x1b[0m", 7},     // Mixed Unicode and ASCII with ANSI
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
		testDesc string
	}{
		// Plain ASCII strings
		{"hello", 10, "hello     ", "pad ASCII to 10"},
		{"hello", 5, "hello", "ASCII already at width"},
		{"hello", 3, "hello", "ASCII already wider than target"},
		{"", 5, "     ", "empty string pad to 5"},
		{"a", 1, "a", "single char at width"},
		{"ab", 5, "ab   ", "two chars pad to 5"},

		// ANSI colored strings (ANSI codes preserved but don't count toward width)
		{"\x1b[31mred\x1b[0m", 5, "\x1b[31mred\x1b[0m  ", "ANSI red pad to 5"},
		{"\x1b[32mgreen\x1b[0m", 10, "\x1b[32mgreen\x1b[0m     ", "ANSI green pad to 10"},
		{"\x1b[31mhi\x1b[0m", 2, "\x1b[31mhi\x1b[0m", "ANSI hi at width 2"},
		{"\x1b[1;32mbold\x1b[0m", 10, "\x1b[1;32mbold\x1b[0m      ", "ANSI bold pad to 10"},

		// Unicode strings
		{"你好", 4, "你好  ", "Chinese 2 chars pad to 4"},
		{"你好", 2, "你好", "Chinese already at width"},
		{"你好", 1, "你好", "Chinese already wider than target"},
		{"🎉", 3, "🎉  ", "emoji pad to 3"},
		{"café", 6, "café  ", "accented chars pad to 6"},

		// Mixed ANSI and Unicode
		{"\x1b[32m你好\x1b[0m", 5, "\x1b[32m你好\x1b[0m   ", "ANSI Chinese pad to 5"},
		{"\x1b[35m🎉\x1b[0m", 3, "\x1b[35m🎉\x1b[0m  ", "ANSI emoji pad to 3"},
	}
	for _, tt := range tests {
		got := PadToWidth(tt.input, tt.width)
		if got != tt.want {
			t.Errorf("PadToWidth(%q, %d) [%s] = %q, want %q", tt.input, tt.width, tt.testDesc, got, tt.want)
		}
		gotVisualWidth := VisualWidth(got)
		if gotVisualWidth < tt.width {
			t.Errorf("PadToWidth(%q, %d) [%s] visual width = %d, want >= %d", tt.input, tt.width, tt.testDesc, gotVisualWidth, tt.width)
		}
	}
}
