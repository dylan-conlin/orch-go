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
		// ASCII cases
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello..."},
		{"abcdefghij", 6, "abc..."},
		{"", 5, ""},
		// Unicode cases
		{"café", 10, "café"},                        // ASCII + combining diacritic: 4 runes, fits
		{"café", 4, "café"},                         // Exactly maxLen runes
		{"café", 3, "..."},                          // Only room for "..."
		{"😀😀😀😀😀", 8, "😀😀😀😀😀"},         // Emoji: 5 runes <= 8, fits without truncation
		{"😀😀😀😀😀😀😀", 8, "😀😀😀😀😀😀😀"}, // Emoji: 7 runes <= 8, fits without truncation
		{"hello😀world", 12, "hello😀world"},       // Mixed: 11 runes <= 12, no truncation
		{"hello😀world", 10, "hello😀w..."}, // Mixed: 11 runes > 10, truncate to 7 runes + "..."
		{"你好世界", 6, "你好世界"},                 // CJK: 4 runes <= 6, no truncation
		{"你好世界", 4, "你好世界"},                 // CJK: 4 runes == 4, exactly fits
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
		// ASCII cases
		{"hello", 10, "hello     "},
		{"hello world", 8, "hello..."},
		{"exact", 5, "exact"},
		// Unicode cases
		{"café", 10, "café      "},           // Padded to 10 runes (4 content + 6 spaces)
		{"café", 4, "café"},                  // Exactly 4 runes
		{"😀", 5, "😀    "},                   // Emoji padded: 1 emoji + 4 spaces = 5 runes
		{"hello😀", 10, "hello😀    "},        // Mixed: 6 runes, padded to 10 (add 4 spaces)
		{"hello😀world", 12, "hello😀world "}, // Mixed: 11 runes, padded to 12 (add 1 space)
		{"你好世界", 6, "你好世界  "},           // CJK padded: 4 runes + 2 spaces = 6 runes
		{"你好世界", 4, "你好世界"},             // CJK exactly: 4 runes
		{"😀😀😀", 5, "😀😀😀  "},             // Emoji padded: 3 runes, padded to 5 (add 2 spaces)
		{"😀😀😀😀😀😀", 5, "😀😀..."},        // Emoji truncated: 6 runes > 5, truncate to 2 + "..."
	}
	for _, tt := range tests {
		got := TruncateWithPadding(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("TruncateWithPadding(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
		// Check that output is exactly maxLen runes
		if len([]rune(got)) != tt.maxLen {
			t.Errorf("TruncateWithPadding(%q, %d) rune length = %d, want %d", tt.input, tt.maxLen, len([]rune(got)), tt.maxLen)
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
