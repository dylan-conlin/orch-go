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

func TestVisualWidth(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"plain ASCII", "hello", 5},
		{"empty string", "", 0},
		{"with ANSI color", "\x1b[31mred\x1b[0m", 3},
		{"with multiple ANSI codes", "\x1b[1;32mbold green\x1b[0m text", 15},
		{"Unicode - emoji", "hello 👋 world", 13},
		{"Unicode - CJK", "你好世界", 4},
		{"mixed Unicode and ANSI", "\x1b[31m你好\x1b[0m world", 8},
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
		verify func(t *testing.T, result string, width int)
	}{
		{
			name:  "pad plain ASCII",
			input: "hello",
			width: 10,
			want:  "hello     ",
			verify: func(t *testing.T, result string, width int) {
				if VisualWidth(result) != width {
					t.Errorf("visual width = %d, want %d", VisualWidth(result), width)
				}
			},
		},
		{
			name:  "already at width",
			input: "exact",
			width: 5,
			want:  "exact",
		},
		{
			name:  "already wider than width",
			input: "too long",
			width: 5,
			want:  "too long",
		},
		{
			name:  "empty string",
			input: "",
			width: 5,
			want:  "     ",
		},
		{
			name:  "with ANSI color codes",
			input: "\x1b[31mred\x1b[0m",
			width: 10,
			verify: func(t *testing.T, result string, width int) {
				if VisualWidth(result) != width {
					t.Errorf("visual width = %d, want %d", VisualWidth(result), width)
				}
				// Verify ANSI codes are preserved
				if !strings.Contains(result, "\x1b[31m") {
					t.Error("ANSI codes not preserved")
				}
			},
		},
		{
			name:  "Unicode emoji",
			input: "hi 👋",
			width: 10,
			verify: func(t *testing.T, result string, width int) {
				if VisualWidth(result) != width {
					t.Errorf("visual width = %d, want %d", VisualWidth(result), width)
				}
			},
		},
		{
			name:  "zero width",
			input: "test",
			width: 0,
			want:  "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PadToWidth(tt.input, tt.width)
			if tt.want != "" && got != tt.want {
				t.Errorf("PadToWidth(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
			}
			if tt.verify != nil {
				tt.verify(t, got, tt.width)
			}
		})
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
