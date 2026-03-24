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

// FormatBytes formats a byte count into a human-readable string using binary units.
// Output style: "0 B", "512 B", "1.5 KiB", "1.0 MiB", "2.3 GiB".
func FormatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}

	neg := bytes < 0
	b := bytes
	if neg {
		b = -b
	}

	const (
		unitB  = 1
		unitKi = 1024
		unitMi = 1024 * 1024
		unitGi = 1024 * 1024 * 1024
		unitTi = 1024 * 1024 * 1024 * 1024
	)

	var value float64
	var unit string

	switch {
	case b >= unitTi:
		value = float64(b) / float64(unitTi)
		unit = "TiB"
	case b >= unitGi:
		value = float64(b) / float64(unitGi)
		unit = "GiB"
	case b >= unitMi:
		value = float64(b) / float64(unitMi)
		unit = "MiB"
	case b >= unitKi:
		value = float64(b) / float64(unitKi)
		unit = "KiB"
	default:
		if neg {
			return fmt.Sprintf("-%d B", b)
		}
		return fmt.Sprintf("%d B", b)
	}

	result := fmt.Sprintf("%.1f %s", value, unit)
	if neg {
		return "-" + result
	}
	return result
}
