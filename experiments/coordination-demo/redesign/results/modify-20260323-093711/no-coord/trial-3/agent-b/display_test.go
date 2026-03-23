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
		// ASCII cases (original tests)
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello..."},
		{"abcdefghij", 6, "abc..."},
		{"", 5, ""},
		// Unicode cases
		{"café", 10, "café"},                         // fits within limit
		{"café", 4, "café"},                          // exactly fits at limit
		{"café world", 7, "café..."},                 // truncate multi-byte string (4 runes + "...")
		{"你好世界", 6, "你好世界"},                      // fits within limit
		{"你好世界朋友", 5, "你好..."},                    // truncate CJK (2 chars + "..." = 5 runes)
		{"hello😀world", 8, "hello..."},               // emoji in ASCII context
		{"😀😀😀😀😀", 5, "😀😀😀😀😀"},                 // exactly maxLen with emoji
		{"😀😀😀😀😀😀", 5, "😀😀..."},                 // truncate emoji string to 5 runes (2 emoji + "...")
	}
	for _, tt := range tests {
		got := Truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
		// Verify truncated strings don't exceed maxLen in runes
		if len([]rune(got)) > tt.maxLen {
			t.Errorf("Truncate(%q, %d) result has %d runes, exceeds maxLen %d", tt.input, tt.maxLen, len([]rune(got)), tt.maxLen)
		}
	}
}

func TestTruncateWithPadding(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		// ASCII cases (original tests)
		{"hello", 10, "hello     "},
		{"hello world", 8, "hello..."},
		{"exact", 5, "exact"},
		// Unicode cases
		{"café", 10, "café      "},             // 4 runes + 6 spaces = 10 runes
		{"café", 4, "café"},                    // exactly 4 runes
		{"café world", 7, "café..."},           // truncate multi-byte string
		{"你好", 5, "你好   "},                  // 2 CJK runes + 3 spaces = 5 runes
		{"😀😀", 5, "😀😀   "},                  // 2 emoji + 3 spaces = 5 runes
		{"😀😀😀😀😀😀", 5, "😀😀..."},         // truncate emoji string to exactly 5 runes (2 emoji + "...")
	}
	for _, tt := range tests {
		got := TruncateWithPadding(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("TruncateWithPadding(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
		// Verify result is exactly maxLen runes
		gotRunes := len([]rune(got))
		if gotRunes != tt.maxLen {
			t.Errorf("TruncateWithPadding(%q, %d) length = %d runes, want %d runes", tt.input, tt.maxLen, gotRunes, tt.maxLen)
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
