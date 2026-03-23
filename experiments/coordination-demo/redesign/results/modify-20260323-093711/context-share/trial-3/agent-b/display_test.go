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
		// ASCII tests
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello..."},
		{"abcdefghij", 6, "abc..."},
		{"", 5, ""},
		// Unicode tests: combining accents fit within limit
		{"cafe\u0301", 10, "cafe\u0301"},
		{"cafe\u0301", 5, "cafe\u0301"},
		// Unicode tests: emoji
		{"🎉hello", 8, "🎉hello"},
		{"🎉hello", 5, "🎉h..."},
		{"hello🎉world", 8, "hello..."},
		// Unicode tests: CJK characters fit and truncate
		{"你好世界", 5, "你好世界"},
		{"你好世界", 4, "你好世界"},
		{"你好世界", 3, "..."},
		// Unicode tests: mixed ASCII and multi-byte
		{"ab🎉cd", 5, "ab🎉cd"},
		{"ab🎉cdef", 5, "ab..."},
		// Edge case: exactly maxLen runes
		{"hello", 5, "hello"},
		{"🎉abc", 4, "🎉abc"},
	}
	for _, tt := range tests {
		got := Truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
		// Verify result doesn't exceed maxLen runes
		if len([]rune(got)) > tt.maxLen {
			t.Errorf("Truncate(%q, %d) rune count = %d, want <= %d", tt.input, tt.maxLen, len([]rune(got)), tt.maxLen)
		}
	}
}

func TestTruncateWithPadding(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		// ASCII tests
		{"hello", 10, "hello     "},
		{"hello world", 8, "hello..."},
		{"exact", 5, "exact"},
		// Unicode tests: emoji with padding
		{"🎉", 5, "🎉    "},
		{"hello🎉", 8, "hello🎉  "},
		// Unicode tests: CJK with padding
		{"你好", 4, "你好  "},
		{"你好世", 5, "你好世  "},
		// Unicode tests: CJK that fits exactly
		{"你好世界", 4, "你好世界"},
		// Unicode tests: CJK truncation
		{"你好世界", 3, "..."},
		// Unicode tests: combining accents with padding
		{"cafe\u0301", 8, "cafe\u0301   "},
		// Unicode tests: mixed ASCII and multi-byte with padding
		{"a🎉b", 5, "a🎉b  "},
		{"🎉🎉🎉", 5, "🎉🎉🎉  "},
		// Unicode tests: mixed ASCII and multi-byte that fits exactly
		{"🎉🎉🎉", 3, "🎉🎉🎉"},
		// Unicode tests: mixed ASCII and multi-byte truncation
		{"🎉🎉🎉🎉", 3, "..."},
		// Edge case: exactly maxLen runes
		{"hello", 5, "hello"},
		{"🎉abc", 4, "🎉abc"},
	}
	for _, tt := range tests {
		got := TruncateWithPadding(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("TruncateWithPadding(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
		// Check rune count equals maxLen
		if len([]rune(got)) != tt.maxLen {
			t.Errorf("TruncateWithPadding(%q, %d) rune count = %d, want %d", tt.input, tt.maxLen, len([]rune(got)), tt.maxLen)
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
