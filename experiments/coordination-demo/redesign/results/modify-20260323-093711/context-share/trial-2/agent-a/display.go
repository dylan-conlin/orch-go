// Package display provides shared output formatting utilities used across
// orch commands and packages: string truncation, ID abbreviation, ANSI
// stripping, and human-readable duration formatting.
package display

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Truncate truncates s to maxLen characters, appending "..." if truncated.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// TruncateWithPadding truncates s to maxLen (with "...") or right-pads with
// spaces to ensure the returned string is exactly maxLen characters.
func TruncateWithPadding(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s + strings.Repeat(" ", maxLen-len(s))
}

// ShortID returns the first 12 characters of an ID string for display.
// If the string is 12 characters or shorter, it is returned unchanged.
func ShortID(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:12]
}

// ansiRegex matches ANSI escape sequences (colors, formatting, etc.)
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// StripANSI removes ANSI escape codes from a string.
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// FormatDuration formats a duration as a human-readable string.
// Output style: "0s", "45s", "3m 12s", "2h 15m", "3d 5h", "1w 2d".
func FormatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	type unit struct {
		duration time.Duration
		label    string
	}

	units := []unit{
		{7 * 24 * time.Hour, "w"},
		{24 * time.Hour, "d"},
		{time.Hour, "h"},
		{time.Minute, "m"},
		{time.Second, "s"},
	}

	// Find the appropriate unit and format with one or two units
	for i, u := range units {
		if d >= u.duration {
			count := int(d / u.duration)
			result := fmt.Sprintf("%d%s", count, u.label)
			remainder := d % u.duration

			// Add the next unit if there's a remainder
			if remainder > 0 && i+1 < len(units) {
				nextUnit := units[i+1]
				nextCount := int(remainder / nextUnit.duration)
				if nextCount > 0 {
					result += fmt.Sprintf(" %d%s", nextCount, nextUnit.label)
				}
			}
			return result
		}
	}

	// Default to seconds (should not reach here in normal usage)
	return fmt.Sprintf("%ds", int(d.Seconds()))
}

// FormatDurationShort formats a duration using short labels suitable for
// dashboard/status output: "just now", "3m", "2h".
func FormatDurationShort(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}
