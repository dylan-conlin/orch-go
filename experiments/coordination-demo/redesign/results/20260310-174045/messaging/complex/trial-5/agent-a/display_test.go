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
		name string
		input string
		want int
	}{
		// Plain ASCII
		{"plain ASCII", "hello", 5},
		{"empty string", "", 0},
		{"single char", "a", 1},
		{"spaces", "   ", 3},

		// ANSI color codes
		{"red text with ANSI", "\x1b[31mred\x1b[0m", 3},
		{"bold green with ANSI", "\x1b[1;32mbold\x1b[0m", 4},
		{"multiple ANSI codes", "\x1b[31mr\x1b[32me\x1b[34md\x1b[0m", 3},

		// Unicode
		{"emoji", "👋", 1},
		{"CJK character", "中", 1},
		{"mixed ASCII and emoji", "hello👋world", 11},

		// Edge cases
		{"only ANSI codes", "\x1b[31m\x1b[0m", 0},
		{"longer text", "This is a longer string", 23},
		{"ANSI with spaces", "\x1b[32mgreen  text\x1b[0m", 11},
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
		name string
		input string
		width int
		want string
	}{
		// Plain ASCII - needs padding
		{"pad plain text", "hello", 10, "hello     "},
		{"pad to exact", "hello", 5, "hello"},
		{"pad single char", "a", 5, "a    "},

		// Already at width
		{"already at width", "hello", 5, "hello"},
		{"empty string, pad to width", "", 5, "     "},

		// Already wider - no padding
		{"wider than target", "hello world", 5, "hello world"},
		{"wider than target 2", "toolong", 3, "toolong"},

		// ANSI codes preserved
		{"ANSI code with padding", "\x1b[31mred\x1b[0m", 7, "\x1b[31mred\x1b[0m    "},
		{"ANSI code no padding needed", "\x1b[31mred\x1b[0m", 3, "\x1b[31mred\x1b[0m"},
		{"ANSI code wider than target", "\x1b[31mlong text\x1b[0m", 5, "\x1b[31mlong text\x1b[0m"},

		// Unicode
		{"unicode padding", "中文", 5, "中文   "},
		{"emoji with padding", "👋", 5, "👋    "},
		{"mixed ASCII and emoji padding", "a中b", 8, "a中b     "},

		// Edge cases
		{"empty string no padding", "", 0, ""},
		{"width zero", "hello", 0, "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PadToWidth(tt.input, tt.width)
			if got != tt.want {
				t.Errorf("PadToWidth(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
			}
			// Verify visual width of result
			gotVisualWidth := VisualWidth(got)
			expectedVisualWidth := tt.width
			if gotVisualWidth < expectedVisualWidth && gotVisualWidth != VisualWidth(tt.input) {
				// If original was narrower than target, we should have padded to target
				if VisualWidth(tt.input) < tt.width {
					if gotVisualWidth != tt.width {
						t.Errorf("PadToWidth(%q, %d) visual width = %d, want %d", tt.input, tt.width, gotVisualWidth, tt.width)
					}
				}
			}
		})
	}
}
