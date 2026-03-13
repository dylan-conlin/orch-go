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
// Output uses B, KiB, MiB, GiB, TiB with 1 decimal place for units above bytes.
// Examples: "512 B", "1.5 KiB", "2.3 MiB", "-1.0 KiB" (for negative values).
func FormatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}

	isNegative := bytes < 0
	absBytes := bytes
	if isNegative {
		absBytes = -bytes
	}

	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	fBytes := float64(absBytes)

	for i := 0; i < len(units)-1; i++ {
		if fBytes < 1024.0 {
			if i == 0 {
				// For bytes, show as integer
				if isNegative {
					return fmt.Sprintf("-%d B", -bytes)
				}
				return fmt.Sprintf("%d B", bytes)
			}
			// For larger units, show with 1 decimal place
			if isNegative {
				return fmt.Sprintf("-%.1f %s", fBytes, units[i])
			}
			return fmt.Sprintf("%.1f %s", fBytes, units[i])
		}
		fBytes /= 1024.0
	}

	// Largest unit (TiB)
	if isNegative {
		return fmt.Sprintf("-%.1f %s", fBytes, units[len(units)-1])
	}
	return fmt.Sprintf("%.1f %s", fBytes, units[len(units)-1])
}
