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

// FormatBytes formats a byte count as a human-readable string with binary units.
// Output uses binary units (B, KiB, MiB, GiB, TiB) with 1 decimal place for
// non-byte units. Examples: "512 B", "1.5 MiB", "2.0 GiB".
func FormatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}

	// Handle negative values
	negative := false
	if bytes < 0 {
		negative = true
		bytes = -bytes
	}

	// Define binary units
	const (
		unit   = 1024
		kibibyte = unit
		mebibyte = unit * unit
		gibibyte = unit * unit * unit
		tebibyte = unit * unit * unit * unit
	)

	var size float64
	var suffix string

	switch {
	case bytes < kibibyte:
		suffix = "B"
		size = float64(bytes)
	case bytes < mebibyte:
		suffix = "KiB"
		size = float64(bytes) / kibibyte
	case bytes < gibibyte:
		suffix = "MiB"
		size = float64(bytes) / mebibyte
	case bytes < tebibyte:
		suffix = "GiB"
		size = float64(bytes) / gibibyte
	default:
		suffix = "TiB"
		size = float64(bytes) / tebibyte
	}

	// Format: bytes have no decimal, others have 1 decimal place
	var result string
	if suffix == "B" {
		result = fmt.Sprintf("%d %s", int64(size), suffix)
	} else {
		result = fmt.Sprintf("%.1f %s", size, suffix)
	}

	if negative {
		result = "-" + result
	}

	return result
}
