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
		// ASCII tests (original)
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello..."},
		{"abcdefghij", 6, "abc..."},
		{"", 5, ""},
		// Unicode tests
		{"hello 🌍", 7, "hello 🌍"},  // 6 ASCII runes + 1 emoji rune = 7 runes total, fits within 7
		{"hello 🌍", 6, "hel..."},    // 7 runes total > 6, truncate to 3 runes + "..."
		{"你好世界test", 8, "你好世界test"},  // 4 CJK + 4 ASCII = 8 runes, exactly 8, no truncation
		{"你好世界test", 7, "你好世界..."},  // 8 runes total > 7, truncate to 4 runes + "..."
		{"café", 5, "café"},          // accented character is 1 rune (4 runes total)
		{"café", 4, "café"},          // exactly 4 runes, no truncation
		{"café", 3, "..."},           // maxLen=3, can only fit "..."
		{"😀😁😂", 6, "😀😁😂"},           // 3 emojis = 3 runes, fits within 6
		{"😀😁😂", 3, "😀😁😂"},           // 3 emojis = 3 runes, exactly 3, no truncation
		{"😀😁😂😀", 3, "..."},            // 4 emojis > 3, truncate to 0 runes + "..."
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
		input        string
		maxLen       int
		want         string
		wantRuneLen  int // expected rune length
	}{
		// ASCII tests (original)
		{"hello", 10, "hello     ", 10},
		{"hello world", 8, "hello...", 8},
		{"exact", 5, "exact", 5},
		// Unicode tests - maxLen is in runes
		{"hi", 8, "hi      ", 8},                      // 2 runes + 6 spaces = 8 runes
		{"hello 🌍", 10, "hello 🌍   ", 10},          // 7 runes + 3 spaces = 10 runes
		{"hello 🌍", 7, "hello 🌍", 7},               // exactly 7 runes, no padding needed
		{"你好", 8, "你好      ", 8},                 // 2 runes + 6 spaces = 8 runes
		{"café", 6, "café  ", 6},                     // 4 runes + 2 spaces = 6 runes
		{"😀😁", 5, "😀😁   ", 5},                   // 2 runes + 3 spaces = 5 runes
	}
	for _, tt := range tests {
		got := TruncateWithPadding(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("TruncateWithPadding(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
		gotRuneLen := len([]rune(got))
		if gotRuneLen != tt.wantRuneLen {
			t.Errorf("TruncateWithPadding(%q, %d) rune length = %d, want %d", tt.input, tt.maxLen, gotRuneLen, tt.wantRuneLen)
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
