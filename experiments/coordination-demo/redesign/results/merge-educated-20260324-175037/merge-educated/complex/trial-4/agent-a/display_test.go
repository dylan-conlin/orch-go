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
		{"plain ASCII", "hello", 5},
		{"empty string", "", 0},
		{"ASCII with spaces", "hello world", 11},
		{"ANSI red text", "\x1b[31mred\x1b[0m", 3},
		{"ANSI bold green", "\x1b[1;32mbold green\x1b[0m", 10},
		{"mixed ANSI and text", "before \x1b[31mred\x1b[0m after", 16},
		{"multiple ANSI codes", "\x1b[1m\x1b[31mbold red\x1b[0m\x1b[0m", 8},
		{"Unicode emoji", "hello 👋 world", 13},
		{"Unicode CJK", "你好世界", 4},
		{"mixed Unicode and ANSI", "\x1b[32m你好\x1b[0m world", 8},
		{"only ANSI codes", "\x1b[31m\x1b[0m", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VisualWidth(tt.input)
			if got != tt.want {
				t.Errorf("VisualWidth(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestPadToWidth(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		width  int
		want   string
		wantVW int // expected visual width of result
	}{
		{"plain ASCII pad needed", "hello", 10, "hello     ", 10},
		{"plain ASCII exact width", "hello", 5, "hello", 5},
		{"plain ASCII already wider", "hello world", 5, "hello world", 11},
		{"empty string", "", 5, "     ", 5},
		{"ANSI text pad needed", "\x1b[31mred\x1b[0m", 10, "\x1b[31mred\x1b[0m       ", 10},
		{"ANSI text exact width", "\x1b[31mhello\x1b[0m", 5, "\x1b[31mhello\x1b[0m", 5},
		{"ANSI text already wide", "\x1b[31mhello world\x1b[0m", 5, "\x1b[31mhello world\x1b[0m", 11},
		{"Unicode pad needed", "你好", 5, "你好   ", 5},
		{"Unicode exact width", "你好", 2, "你好", 2},
		{"mixed ANSI and Unicode", "\x1b[32m你好\x1b[0m", 5, "\x1b[32m你好\x1b[0m   ", 5},
		{"zero width", "text", 0, "text", 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PadToWidth(tt.input, tt.width)
			if got != tt.want {
				t.Errorf("PadToWidth(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
			}
			gotVW := VisualWidth(got)
			if gotVW != tt.wantVW {
				t.Errorf("PadToWidth(%q, %d) visual width = %d, want %d", tt.input, tt.width, gotVW, tt.wantVW)
			}
		})
	}
}
