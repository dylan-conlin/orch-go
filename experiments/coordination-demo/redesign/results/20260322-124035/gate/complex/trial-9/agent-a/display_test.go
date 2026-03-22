package display

import (
	"strings"
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
		{"\x1b[31mred\x1b[0m", 3},
		{"\x1b[1;32mbold green\x1b[0m", 10},
		{"plain\x1b[33m yellow\x1b[0m end", 16},

		// Unicode strings (CJK characters)
		{"你好", 2},
		{"世界", 2},
		{"hello世界", 7},

		// Emoji
		{"😀", 1},
		{"hello😀world", 11},

		// Combined: Unicode with ANSI
		{"\x1b[31m你好\x1b[0m", 2},
		{"\x1b[1m世界\x1b[0m test", 7},
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
		input     string
		width     int
		wantWidth int
	}{
		// Plain ASCII strings - pad to reach width
		{"hello", 10, 10},
		{"hello", 5, 5},
		{"test", 8, 8},
		{"", 5, 5},
		{"", 0, 0},

		// Strings already at target width
		{"exact", 5, 5},

		// Strings wider than target width (return unchanged)
		{"too long", 5, 8},
		{"verylongstring", 5, 14},

		// Unicode strings
		{"你好", 4, 4},
		{"世界", 5, 5},

		// Strings with ANSI color codes
		{"\x1b[31mred\x1b[0m", 5, 5},
		{"\x1b[1;32mbold\x1b[0m", 8, 8},

		// Combined: Unicode with ANSI
		{"\x1b[31m你好\x1b[0m", 4, 4},
		{"\x1b[1m世界\x1b[0m test", 8, 8},
	}
	for _, tt := range tests {
		got := PadToWidth(tt.input, tt.width)
		// Verify visual width of result
		gotWidth := VisualWidth(got)
		if gotWidth != tt.wantWidth {
			t.Errorf("PadToWidth(%q, %d) visual width = %d, want %d", tt.input, tt.width, gotWidth, tt.wantWidth)
		}
		// Verify content matches input (padding only adds spaces)
		strippedInput := StripANSI(tt.input)
		strippedGot := StripANSI(got)
		if !strings.HasPrefix(strippedGot, strippedInput) {
			t.Errorf("PadToWidth(%q, %d) = %q, but content doesn't match input", tt.input, tt.width, got)
		}
	}
}
