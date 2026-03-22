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
		{"hello world", 11},
		{"", 0},
		{"a", 1},

		// ANSI color codes
		{"\x1b[31mred\x1b[0m", 3},           // red text
		{"\x1b[1;32mbold green\x1b[0m", 10}, // bold green
		{"\x1b[31m\x1b[1m\x1b[4mstyle\x1b[0m", 5}, // multiple codes

		// Unicode (CJK characters)
		{"你好", 2},
		{"こんにちは", 5},

		// Emoji
		{"👋", 1},
		{"hello👋world", 11},

		// Mixed: ANSI codes with Unicode
		{"\x1b[31m你好\x1b[0m", 2},
		{"\x1b[1;32m👋\x1b[0m", 1},

		// Only ANSI codes, no visible content
		{"\x1b[31m\x1b[0m", 0},
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
		input    string
		width    int
		want     string
		wantLen  int // expected byte length (with ANSI codes)
	}{
		// Plain ASCII - padding needed
		{"hello", 10, "hello     ", 10},
		{"hi", 5, "hi   ", 5},
		{"a", 3, "a  ", 3},

		// Plain ASCII - no padding needed
		{"hello", 5, "hello", 5},
		{"hello", 3, "hello", 5}, // already wider

		// Empty string
		{"", 5, "     ", 5},
		{"", 0, "", 0},

		// ANSI color codes (codes don't count toward width)
		{"\x1b[31mred\x1b[0m", 5, "\x1b[31mred\x1b[0m  ", 11}, // visual width 3, pad to 5
		{"\x1b[31mred\x1b[0m", 3, "\x1b[31mred\x1b[0m", 11}, // visual width 3, already at target
		{"\x1b[1;32mbold\x1b[0m", 6, "\x1b[1;32mbold\x1b[0m  ", 16}, // visual width 4, pad to 6

		// Unicode
		{"你好", 5, "你好   ", 8}, // CJK: 2 chars + 3 spaces
		{"👋", 3, "👋  ", 4},      // emoji: 1 char + 2 spaces (visual width 1, pad to 3)

		// Mixed: ANSI codes with Unicode
		{"\x1b[31m你好\x1b[0m", 5, "\x1b[31m你好\x1b[0m   ", 13}, // visual width 2, pad to 5
		{"\x1b[1;32m👋\x1b[0m", 3, "\x1b[1;32m👋\x1b[0m  ", 12}, // visual width 1, pad to 3
	}
	for _, tt := range tests {
		got := PadToWidth(tt.input, tt.width)
		if got != tt.want {
			t.Errorf("PadToWidth(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
		}
		// Verify visual width is correct
		if VisualWidth(got) < tt.width && VisualWidth(tt.input) < tt.width {
			t.Errorf("PadToWidth(%q, %d) visual width = %d, want >= %d", tt.input, tt.width, VisualWidth(got), tt.width)
		}
	}
}
