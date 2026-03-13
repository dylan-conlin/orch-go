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
		{"hello world", 11},
		{"a", 1},

		// Strings with ANSI color codes
		{"\x1b[31mred\x1b[0m", 3},                    // ANSI stripped, "red" = 3
		{"\x1b[1;32mbold green\x1b[0m", 10},           // ANSI stripped, "bold green" = 10
		{"\x1b[33mhello\x1b[0m", 5},                  // "hello" = 5
		{"\x1b[31m\x1b[1mstyle\x1b[0m", 5},           // multiple codes, "style" = 5

		// Unicode strings (CJK characters)
		{"你好", 2},                  // 2 CJK characters
		{"日本", 2},                  // 2 CJK characters
		{"测试hello", 7},              // 2 CJK + 5 ASCII
		{"\x1b[35m你好\x1b[0m", 2},    // ANSI + CJK, visual width = 2

		// Emoji
		{"😀", 1},                    // 1 emoji (1 rune)
		{"🎉hello", 6},               // emoji + ASCII
		{"\x1b[36m😀\x1b[0m", 1},     // ANSI + emoji

		// Edge cases
		{" ", 1},
		{"   ", 3},
		{"text    ", 8},
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
		wantVW int // visual width of result
	}{
		// Plain ASCII padding
		{"", 5, "     ", 5},
		{"hello", 5, "hello", 5},
		{"hello", 10, "hello     ", 10},
		{"hi", 8, "hi      ", 8},

		// ANSI codes preserved, padding added
		{"\x1b[31mred\x1b[0m", 5, "\x1b[31mred\x1b[0m  ", 5},
		{"\x1b[31mred\x1b[0m", 3, "\x1b[31mred\x1b[0m", 3},
		{"\x1b[1;32mbold\x1b[0m", 10, "\x1b[1;32mbold\x1b[0m      ", 10},

		// Unicode (CJK) padding
		{"你好", 4, "你好  ", 4},
		{"你好", 5, "你好   ", 5},
		{"测试", 2, "测试", 2},
		{"你好", 2, "你好", 2},

		// Unicode with ANSI
		{"\x1b[33m你好\x1b[0m", 4, "\x1b[33m你好\x1b[0m  ", 4},
		{"\x1b[35m😀\x1b[0m", 5, "\x1b[35m😀\x1b[0m    ", 5},

		// Already wider or exact width
		{"hello world", 5, "hello world", 11},
		{"hello world", 11, "hello world", 11},
		{"text", 3, "text", 4},

		// Empty string
		{"", 0, "", 0},
	}
	for _, tt := range tests {
		got := PadToWidth(tt.input, tt.width)
		if got != tt.want {
			t.Errorf("PadToWidth(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
		}
		gotVW := VisualWidth(got)
		if gotVW != tt.wantVW {
			t.Errorf("PadToWidth(%q, %d) visual width = %d, want %d", tt.input, tt.width, gotVW, tt.wantVW)
		}
	}
}
