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
		{"café", 10, "café"},                     // Multi-byte character (4 runes), fits in 10
		{"café", 4, "café"},                      // Exactly at limit (4 runes)
		{"café", 3, "..."},                       // Truncate to just "..." (4 runes > 3)
		{"hello 🌍", 7, "hello 🌍"},             // Emoji (7 runes), exactly at limit
		{"hello 🌍", 6, "hel..."},                // Emoji (7 runes), truncate to 6 (3 + "...")
		{"你好世界", 4, "你好世界"},                 // CJK (4 runes), exactly at limit
		{"你好世界", 3, "..."},                     // CJK (4 runes), truncate to 3 ("...")
		{"a你b好c", 5, "a你b好c"},                // Mixed ASCII and CJK (5 runes), exactly at limit
		{"a你b好c", 4, "a..."},                   // Mixed (5 runes), truncate to 4 (1 rune + "...")
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
		{"café", 10, "café      "},                // Multi-byte (4 runes) + 6 spaces = 10 runes
		{"café", 4, "café"},                      // Exactly at limit (4 runes)
		{"hello 🌍", 10, "hello 🌍   "},          // Emoji (7 runes) + 3 spaces = 10 runes
		{"你好", 5, "你好   "},                     // CJK (2 runes) + 3 spaces = 5 runes
		{"你好世界", 3, "..."},                     // CJK (4 runes), truncate to 3 ("...")
		{"a你b", 5, "a你b  "},                    // Mixed (3 runes) + 2 spaces = 5 runes
	}
	for _, tt := range tests {
		got := TruncateWithPadding(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("TruncateWithPadding(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
		// Verify output is exactly maxLen runes
		gotRunes := len([]rune(got))
		if gotRunes != tt.maxLen {
			t.Errorf("TruncateWithPadding(%q, %d) rune length = %d, want %d", tt.input, tt.maxLen, gotRunes, tt.maxLen)
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
