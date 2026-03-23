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

// FormatTable renders headers and rows as an aligned text table.
// Column widths are auto-sized based on the widest content (including headers).
// ANSI color codes are preserved but don't count toward column width.
// Handles edge cases: empty rows, mismatched column counts, and nil rows.
func FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	// Calculate column widths using visual width (ignoring ANSI codes)
	colWidths := make([]int, len(headers))
	for i, h := range headers {
		colWidths[i] = len(StripANSI(h))
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(headers) {
				width := len(StripANSI(cell))
				if width > colWidths[i] {
					colWidths[i] = width
				}
			}
		}
	}

	// Helper to right-pad string to visual width, preserving ANSI codes
	padToWidth := func(s string, width int) string {
		visualWidth := len(StripANSI(s))
		if visualWidth >= width {
			return s
		}
		return s + strings.Repeat(" ", width-visualWidth)
	}

	// Build table output
	var buf strings.Builder

	// Header row
	buf.WriteString("|")
	for i, h := range headers {
		buf.WriteString(" ")
		buf.WriteString(padToWidth(h, colWidths[i]))
		buf.WriteString(" |")
	}
	buf.WriteString("\n")

	// Separator line (dashes under each column)
	buf.WriteString("|")
	for _, w := range colWidths {
		buf.WriteString(strings.Repeat("-", w+2))
		buf.WriteString("|")
	}
	buf.WriteString("\n")

	// Data rows
	for _, row := range rows {
		buf.WriteString("|")
		for i, w := range colWidths {
			buf.WriteString(" ")
			var cell string
			if i < len(row) {
				cell = row[i]
			}
			buf.WriteString(padToWidth(cell, w))
			buf.WriteString(" |")
		}
		buf.WriteString("\n")
	}

	return buf.String()
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
