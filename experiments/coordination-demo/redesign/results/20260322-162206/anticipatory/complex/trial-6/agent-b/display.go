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

// FormatTable renders headers and rows as an aligned text table.
// Columns are auto-sized based on the widest content (header or data).
// ANSI-colored content is handled correctly using StripANSI for width calculation.
// Handles edge cases: empty rows (headers only), rows with fewer or more columns than headers.
// Returns an empty string if headers is empty.
func FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	// Calculate column widths based on headers and rows
	colWidths := make([]int, len(headers))

	// Initialize with header widths
	for i, h := range headers {
		colWidths[i] = len(StripANSI(h))
	}

	// Expand for row content
	for _, row := range rows {
		for i := 0; i < len(headers) && i < len(row); i++ {
			w := len(StripANSI(row[i]))
			if w > colWidths[i] {
				colWidths[i] = w
			}
		}
	}

	var buf strings.Builder

	// Write header
	for i, h := range headers {
		if i > 0 {
			buf.WriteString(" | ")
		}
		buf.WriteString(h)
		// Pad to column width
		padding := colWidths[i] - len(StripANSI(h))
		buf.WriteString(strings.Repeat(" ", padding))
	}
	buf.WriteString("\n")

	// Write separator
	for i, w := range colWidths {
		if i > 0 {
			buf.WriteString("-+-")
		}
		buf.WriteString(strings.Repeat("-", w))
	}
	buf.WriteString("\n")

	// Write rows
	for _, row := range rows {
		for i := 0; i < len(headers); i++ {
			if i > 0 {
				buf.WriteString(" | ")
			}

			var col string
			if i < len(row) {
				col = row[i]
			}

			buf.WriteString(col)
			// Pad to column width
			padding := colWidths[i] - len(StripANSI(col))
			buf.WriteString(strings.Repeat(" ", padding))
		}
		buf.WriteString("\n")
	}

	return strings.TrimSuffix(buf.String(), "\n")
}
