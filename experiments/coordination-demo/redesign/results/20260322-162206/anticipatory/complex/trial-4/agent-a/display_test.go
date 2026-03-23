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
		// Plain ASCII
		{"hello", 5},
		{"hello world", 11},
		{"", 0},
		{"a", 1},

		// ANSI color codes (should be stripped)
		{"\x1b[31mred\x1b[0m", 3},
		{"\x1b[1;32mbold green\x1b[0m", 10},
		{"\x1b[33myellow\x1b[0m", 6},

		// Unicode: CJK characters (each counts as 1 rune)
		{"你好", 2},
		{"こんにちは", 5},
		{"안녕하세요", 5},

		// Emoji (each counts as 1 rune)
		{"😀", 1},
		{"👋🌍", 2},

		// Mixed: ASCII + Unicode
		{"hello世界", 7},
		{"hi👋", 3},

		// Combined: ANSI + Unicode
		{"\x1b[32m世界\x1b[0m", 2},
		{"\x1b[1;31m你好\x1b[0m", 2},

		// Edge case: multiple ANSI codes
		{"\x1b[1m\x1b[31mbold red\x1b[0m", 8},
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
	}{
		// Plain ASCII padding
		{"hi", 5},
		{"hello", 10},
		{"", 3},

		// No padding needed (already at width)
		{"hello", 5},

		// Already wider than target (should return unchanged)
		{"hello world", 5},

		// ANSI codes preserved
		{"\x1b[31mred\x1b[0m", 5},
		{"\x1b[1;32mbold green\x1b[0m", 12},

		// Unicode padding
		{"你好", 5},
		{"世", 3},

		// Combined: ANSI + Unicode with padding
		{"\x1b[32m世\x1b[0m", 3},
		{"\x1b[33m你好\x1b[0m", 5},
	}
	for _, tt := range tests {
		got := PadToWidth(tt.input, tt.width)

		// Check that original content is preserved (starts with input)
		if !strings.HasPrefix(got, tt.input) {
			t.Errorf("PadToWidth(%q, %d) doesn't preserve original: got %q", tt.input, tt.width, got)
		}

		inputWidth := VisualWidth(tt.input)
		gotWidth := VisualWidth(got)

		if inputWidth < tt.width {
			// Input was narrower: should be padded to target width
			if gotWidth != tt.width {
				t.Errorf("PadToWidth(%q, %d) padded width = %d, want %d", tt.input, tt.width, gotWidth, tt.width)
			}
		} else {
			// Input was at or wider: should return unchanged
			if got != tt.input {
				t.Errorf("PadToWidth(%q, %d) should return unchanged for input at/wider than target, got %q", tt.input, tt.width, got)
			}
			if gotWidth != inputWidth {
				t.Errorf("PadToWidth(%q, %d) width = %d, want %d", tt.input, tt.width, gotWidth, inputWidth)
			}
		}
	}
}
