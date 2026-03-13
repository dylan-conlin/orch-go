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
		name  string
		input string
		want  int
	}{
		// Plain ASCII
		{"empty string", "", 0},
		{"simple ascii", "hello", 5},
		{"ascii with spaces", "hello world", 11},
		{"single character", "a", 1},

		// ANSI codes
		{"ascii with ansi red", "\x1b[31mred\x1b[0m", 3},
		{"ascii with bold green", "\x1b[1;32mbold\x1b[0m", 4},
		{"multiple ansi codes", "\x1b[31mhel\x1b[32mlo\x1b[0m", 5},
		{"ansi with spaces", "\x1b[31mhel lo\x1b[0m", 6},
		{"only ansi codes", "\x1b[31m\x1b[0m", 0},

		// Unicode
		{"single emoji", "😀", 1},
		{"emoji with ascii", "hello😀world", 11},
		{"multiple emoji", "😀😀😀", 3},
		{"cjk characters", "你好", 2},
		{"mixed unicode", "hi你好世界", 6},
		{"cjk with spaces", "你 好", 3},

		// Edge cases
		{"ansi with emoji", "\x1b[31m😀\x1b[0m", 1},
		{"ansi with cjk", "\x1b[31m你\x1b[0m", 1},
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
		name  string
		input string
		width int
		want  string
	}{
		// Basic padding
		{"empty to width", "", 5, "     "},
		{"short ascii", "hi", 5, "hi   "},
		{"short ascii to 3", "a", 3, "a  "},

		// Already at width
		{"exact width", "hello", 5, "hello"},
		{"already at width", "hello", 5, "hello"},

		// Exceeds width (no padding)
		{"exceeds width", "hello world", 5, "hello world"},
		{"exceeds width by 1", "hello!", 5, "hello!"},

		// With ANSI codes
		{"ascii with ansi to width", "\x1b[31mhi\x1b[0m", 5, "\x1b[31mhi\x1b[0m   "},
		{"ansi exact width", "\x1b[31mhello\x1b[0m", 5, "\x1b[31mhello\x1b[0m"},
		{"ansi exceeds width", "\x1b[31mhello world\x1b[0m", 5, "\x1b[31mhello world\x1b[0m"},

		// Unicode
		{"emoji padding", "😀", 5, "😀    "},
		{"emoji exact", "😀😀", 2, "😀😀"},
		{"cjk padding", "你", 5, "你    "},
		{"mixed unicode", "hi你好世界", 6, "hi你好世界"},

		// Zero width (no padding needed)
		{"to zero width", "", 0, ""},
		{"exceeds zero width", "a", 0, "a"},
	}

	for _, tt := range tests {
		got := PadToWidth(tt.input, tt.width)
		if got != tt.want {
			t.Errorf("PadToWidth(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
		}

		// Verify correctness:
		// If input visual width < target, result visual width should equal target
		// If input visual width >= target, result should be unchanged
		inputWidth := VisualWidth(tt.input)
		resultWidth := VisualWidth(got)

		if inputWidth < tt.width && resultWidth != tt.width {
			t.Errorf("PadToWidth(%q, %d): input width %d < target %d, result width %d, want %d",
				tt.input, tt.width, inputWidth, tt.width, resultWidth, tt.width)
		}

		if inputWidth >= tt.width && got != tt.input {
			t.Errorf("PadToWidth(%q, %d): input width %d >= target %d, expected input unchanged",
				tt.input, tt.width, inputWidth, tt.width)
		}
	}
}
