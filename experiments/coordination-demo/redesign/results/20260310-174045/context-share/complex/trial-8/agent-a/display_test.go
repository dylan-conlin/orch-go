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
		{"abc123", 6},
		// ANSI color codes (should be ignored)
		{"\x1b[31mred\x1b[0m", 3},
		{"\x1b[1;32mbold green\x1b[0m", 10},
		{"\x1b[31mh\x1b[0m\x1b[32me\x1b[0m\x1b[34ml\x1b[0m\x1b[35ml\x1b[0m\x1b[36mo\x1b[0m", 5},
		// Unicode (CJK characters)
		{"你好", 2},
		{"中文test", 6},
		// Emoji
		{"👋", 1},
		{"hello 👋 world", 13},
		// Mixed: ANSI + Unicode
		{"\x1b[31m你好\x1b[0m", 2},
		{"\x1b[1;32m你好world\x1b[0m", 7},
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
		input          string
		targetWidth    int
		want           string
		wantVisualSize int
	}{
		// Plain strings needing padding
		{"hi", 5, "hi   ", 5},
		{"hello", 5, "hello", 5},
		{"", 5, "     ", 5},
		{"a", 10, "a         ", 10},
		// Already at width
		{"test", 4, "test", 4},
		// Wider than target (unchanged)
		{"hello world", 5, "hello world", 11},
		{"longstring", 3, "longstring", 10},
		// With ANSI codes
		{"\x1b[31mhi\x1b[0m", 5, "\x1b[31mhi\x1b[0m   ", 5},
		{"\x1b[1;32mtest\x1b[0m", 4, "\x1b[1;32mtest\x1b[0m", 4},
		{"\x1b[31mred\x1b[0m", 10, "\x1b[31mred\x1b[0m       ", 10},
		// Unicode with padding
		{"你", 5, "你    ", 5},
		{"\x1b[31m你\x1b[0m", 5, "\x1b[31m你\x1b[0m    ", 5},
		// Single character padding
		{"x", 1, "x", 1},
		{"x", 2, "x ", 2},
		// Edge case: width 0
		{"test", 0, "test", 4},
	}
	for _, tt := range tests {
		got := PadToWidth(tt.input, tt.targetWidth)
		if got != tt.want {
			t.Errorf("PadToWidth(%q, %d) = %q, want %q", tt.input, tt.targetWidth, got, tt.want)
		}
		// Verify the visual width of the result is correct
		gotVisualWidth := VisualWidth(got)
		if gotVisualWidth != tt.wantVisualSize {
			t.Errorf("PadToWidth(%q, %d) visual width = %d, want %d", tt.input, tt.targetWidth, gotVisualWidth, tt.wantVisualSize)
		}
	}
}
