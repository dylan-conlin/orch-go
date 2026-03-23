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
		// ASCII test cases (existing)
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello..."},
		{"abcdefghij", 6, "abc..."},
		{"", 5, ""},
		// Unicode test cases
		{"Hello 👋", 10, "Hello 👋"},           // Emoji fits within limit (7 runes)
		{"Hello 👋", 7, "Hello 👋"},           // Exactly at limit
		{"Hello 👋extra", 7, "Hell..."},       // Truncate with emoji (4 runes + "...")
		{"你好世界", 4, "你好世界"},            // CJK characters fit (4 runes)
		{"你好世界extra", 4, "你..."},         // CJK truncated (1 rune + "...")
		{"café", 4, "café"},                  // Combining characters (4 runes)
		{"cafélongword", 4, "c..."},           // Truncate combining characters (1 rune + "...")
		{"Hi café 👋", 10, "Hi café 👋"},     // Mixed ASCII, combining chars, emoji (9 runes)
		{"Hi café 👋extra", 7, "Hi c..."}, // Mixed truncated (4 runes + "...")
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
		// ASCII test cases (existing)
		{"hello", 10, "hello     "},
		{"hello world", 8, "hello..."},
		{"exact", 5, "exact"},
		// Unicode test cases
		{"Hi", 4, "Hi  "},                        // ASCII padding (2 runes + 2 spaces)
		{"café", 6, "café  "},                    // Unicode with padding (4 runes + 2 spaces)
		{"你好", 4, "你好  "},                    // CJK with padding (2 runes + 2 spaces)
		{"你好世界test", 4, "你..."},            // CJK truncated (1 rune + "...")
		{"Hello 👋", 10, "Hello 👋   "},         // Emoji with padding (7 runes + 3 spaces)
		{"Hello 👋world", 8, "Hello..."},        // Emoji truncated (5 runes + "...")
	}
	for _, tt := range tests {
		got := TruncateWithPadding(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("TruncateWithPadding(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
		// Verify rune length equals maxLen
		runes := []rune(got)
		if len(runes) != tt.maxLen {
			t.Errorf("TruncateWithPadding(%q, %d) rune length = %d, want %d", tt.input, tt.maxLen, len(runes), tt.maxLen)
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
