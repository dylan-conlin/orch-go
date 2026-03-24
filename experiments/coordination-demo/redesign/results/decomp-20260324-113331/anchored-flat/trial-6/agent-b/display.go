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
// Output style: "0s", "45s", "3m 12s", "2h 15m", "3d 5h".
func FormatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	days := int(d.Hours()) / 24
	if days > 0 {
		hours := int(d.Hours()) % 24
		if hours > 0 {
			return fmt.Sprintf("%dd %dh", days, hours)
		}
		return fmt.Sprintf("%dd", days)
	}

	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}

	if d < time.Hour {
		minutes := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		if secs > 0 {
			return fmt.Sprintf("%dm %ds", minutes, secs)
		}
		return fmt.Sprintf("%dm", minutes)
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if minutes > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dh", hours)
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

// FormatRate formats a transfer rate in bytes per second as a human-readable
// string using binary units with "/s" suffix.
// Output style: "0 B/s", "512 B/s", "1.5 KiB/s", "10.2 MiB/s", "2.1 GiB/s".
func FormatRate(bytesPerSec float64) string {
	// Handle zero case
	if bytesPerSec == 0 {
		return "0 B/s"
	}

	// Handle negative values
	isNegative := bytesPerSec < 0
	value := bytesPerSec
	if isNegative {
		value = -value
	}

	// Define binary units
	const (
		kiB = 1024
		miB = 1024 * 1024
		giB = 1024 * 1024 * 1024
	)

	var result string

	// Choose appropriate unit
	if value >= giB {
		result = fmt.Sprintf("%.1f GiB/s", value/giB)
	} else if value >= miB {
		result = fmt.Sprintf("%.1f MiB/s", value/miB)
	} else if value >= kiB {
		result = fmt.Sprintf("%.1f KiB/s", value/kiB)
	} else {
		result = fmt.Sprintf("%.0f B/s", value)
	}

	if isNegative {
		result = "-" + result
	}

	return result
}
