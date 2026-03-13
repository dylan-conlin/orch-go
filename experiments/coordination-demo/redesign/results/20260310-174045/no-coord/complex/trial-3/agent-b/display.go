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
// Column widths are auto-sized based on the widest content in each column,
// including headers. ANSI-colored content is handled correctly by using
// StripANSI for width calculation. The header is separated from data rows
// by a line of dashes.
//
// Edge cases handled:
// - Empty rows (headers only)
// - Rows with fewer columns than headers (padded with empty strings)
// - Rows with more columns than headers (extra columns are included)
// - ANSI color codes (stripped for width calculation only)
func FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	// Determine the number of columns
	numCols := len(headers)
	for _, row := range rows {
		if len(row) > numCols {
			numCols = len(row)
		}
	}

	// Pad headers with empty strings if needed
	paddedHeaders := make([]string, numCols)
	copy(paddedHeaders, headers)

	// Calculate column widths based on stripped ANSI width
	colWidths := make([]int, numCols)
	for i, h := range paddedHeaders {
		colWidths[i] = len(StripANSI(h))
	}
	for _, row := range rows {
		for i := 0; i < numCols; i++ {
			var cell string
			if i < len(row) {
				cell = row[i]
			}
			width := len(StripANSI(cell))
			if width > colWidths[i] {
				colWidths[i] = width
			}
		}
	}

	// Add padding around content (1 space on each side)
	const padding = 1
	for i := range colWidths {
		colWidths[i] += 2 * padding
	}

	// Build the table
	var buf strings.Builder

	// Header row
	for i, h := range paddedHeaders {
		if i > 0 {
			buf.WriteString(" | ")
		}
		stripH := StripANSI(h)
		pad := colWidths[i] - len(stripH)
		buf.WriteString(strings.Repeat(" ", padding))
		buf.WriteString(h)
		buf.WriteString(strings.Repeat(" ", pad-padding))
	}
	buf.WriteString("\n")

	// Separator line
	for i, width := range colWidths {
		if i > 0 {
			buf.WriteString("-+-")
		}
		buf.WriteString(strings.Repeat("-", width))
	}
	buf.WriteString("\n")

	// Data rows
	for _, row := range rows {
		for i := 0; i < numCols; i++ {
			if i > 0 {
				buf.WriteString(" | ")
			}
			var cell string
			if i < len(row) {
				cell = row[i]
			}
			stripCell := StripANSI(cell)
			pad := colWidths[i] - len(stripCell)
			buf.WriteString(strings.Repeat(" ", padding))
			buf.WriteString(cell)
			buf.WriteString(strings.Repeat(" ", pad-padding))
		}
		buf.WriteString("\n")
	}

	return strings.TrimSuffix(buf.String(), "\n")
}
