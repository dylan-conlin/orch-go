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
		{"", 0},
		{"hello", 5},
		{"hello world", 11},
		{"abc123", 6},
		// ANSI color codes (should be stripped before counting)
		{"\x1b[31mred\x1b[0m", 3},
		{"\x1b[1;32mbold green\x1b[0m", 10},
		{"\x1b[33myellow text\x1b[0m", 11},
		// Unicode: emoji (single rune each)
		{"😀", 1},
		{"hello 😀", 7}, // 5 letters + 1 space + 1 emoji rune
		// Unicode: CJK characters (single rune each)
		{"你好", 2},
		{"こんにちは", 5},
		// Mixed: ANSI + Unicode (ANSI stripped, runes counted)
		{"\x1b[31m你好\x1b[0m", 2},
		{"\x1b[1;32m😀 emoji\x1b[0m", 7}, // 1 emoji + 1 space + 5 letters
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
		input   string
		width   int
		want    string
		wantLen int // visual width (rune count) of result
	}{
		// Empty string
		{"", 5, "     ", 5},
		// Plain ASCII - below target
		{"hi", 5, "hi   ", 5},
		// Plain ASCII - at target
		{"hello", 5, "hello", 5},
		// Plain ASCII - above target (no padding)
		{"hello world", 5, "hello world", 11},
		// ANSI color code - below target
		{"\x1b[31mred\x1b[0m", 5, "\x1b[31mred\x1b[0m  ", 5},
		// ANSI color code - at target
		{"\x1b[1;32mbold green\x1b[0m", 10, "\x1b[1;32mbold green\x1b[0m", 10},
		// ANSI color code - above target (no padding)
		{"\x1b[31mhello world\x1b[0m", 5, "\x1b[31mhello world\x1b[0m", 11},
		// Unicode emoji (4 runes: 1 emoji + 1 space + 2 letters "hi")
		{"😀 hi", 8, "😀 hi    ", 8},
		// Unicode emoji (7 runes: 1 emoji + 1 space + 5 letters "hello")
		{"😀 hello", 8, "😀 hello ", 8},
		// Unicode CJK (2 runes)
		{"你好", 4, "你好  ", 4},
		// Unicode CJK - at target
		{"你好", 2, "你好", 2},
	}
	for _, tt := range tests {
		got := PadToWidth(tt.input, tt.width)
		if got != tt.want {
			t.Errorf("PadToWidth(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
		}
		// Verify that padding was applied to reach target width
		gotVisualWidth := VisualWidth(got)
		if gotVisualWidth != tt.wantLen {
			t.Errorf("PadToWidth(%q, %d) result visual width = %d, want %d", tt.input, tt.width, gotVisualWidth, tt.wantLen)
		}
	}
}
