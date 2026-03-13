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
		{"\x1b[31mred text\x1b[0m", 8}, // "red text" is 8 chars
		{"\x1b[1;32mbold green\x1b[0m", 10},
		{"\x1b[0m\x1b[1m\x1b[31mtext\x1b[0m", 4}, // multiple ANSI codes, "text" is 4 chars

		// Unicode strings (CJK characters)
		{"你好", 2},
		{"こんにちは", 5},
		{"한글", 2},
		{"hello你好world", 12},

		// Emoji
		{"👋", 1},
		{"👋 hello", 7},
		{"hello 👋 world", 13},

		// Mixed: Unicode + ANSI codes
		{"\x1b[32m你好\x1b[0m", 2},
		{"\x1b[1;31m日本語テキスト\x1b[0m", 7},
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
		{"hi", 0, "hi"},

		// Strings with ANSI codes (width should ignore ANSI, padding preserves it)
		{"\x1b[31mred\x1b[0m", 6, "\x1b[31mred\x1b[0m   "},      // visual width 3, pad to 6
		{"\x1b[1;32mbold green\x1b[0m", 15, "\x1b[1;32mbold green\x1b[0m     "}, // visual width 10, pad to 15

		// Unicode strings
		{"你好", 4, "你好  "},
		{"日本", 4, "日本  "},
		{"hello", 8, "hello   "},

		// Already at or wider than width
		{"hello", 5, "hello"},
		{"hello", 3, "hello"},
		{"\x1b[31mred text\x1b[0m", 8, "\x1b[31mred text\x1b[0m"}, // visual width 8, already at width

		// Mixed: Unicode + ANSI
		{"\x1b[32m你好\x1b[0m", 4, "\x1b[32m你好\x1b[0m  "}, // visual width 2, pad to 4
	}
	for _, tt := range tests {
		got := PadToWidth(tt.input, tt.width)
		if got != tt.want {
			t.Errorf("PadToWidth(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
		}
		// Verify visual width is correct
		visualWidth := VisualWidth(got)
		if visualWidth < tt.width {
			t.Errorf("PadToWidth(%q, %d) visual width = %d, want at least %d", tt.input, tt.width, visualWidth, tt.width)
		}
	}
}
