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
// It auto-sizes each column based on the widest content in that column (including header).
// The header is separated from data rows with a line of dashes.
// ANSI color codes are stripped when calculating column widths but preserved in output.
// Handles edge cases: empty rows (headers only), rows with fewer columns than headers,
// and rows with more columns than headers.
func FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	// Handle nil rows
	if rows == nil {
		rows = [][]string{}
	}

	// Calculate column widths
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = len(StripANSI(header))
	}

	// Update column widths based on row content
	for _, row := range rows {
		for i := 0; i < len(headers); i++ {
			var cellContent string
			if i < len(row) {
				cellContent = row[i]
			}
			width := len(StripANSI(cellContent))
			if width > colWidths[i] {
				colWidths[i] = width
			}
		}
	}

	// Build the table
	var result strings.Builder

	// Helper function to format a single row
	formatRow := func(cells []string) string {
		var line strings.Builder
		for i, width := range colWidths {
			var cellContent string
			if i < len(cells) {
				cellContent = cells[i]
			}

			// Pad the cell to the column width
			stripped := StripANSI(cellContent)
			padding := width - len(stripped)
			line.WriteString(" ")
			line.WriteString(cellContent)
			if padding > 0 {
				line.WriteString(strings.Repeat(" ", padding))
			}
			line.WriteString(" ")
			if i < len(colWidths)-1 {
				line.WriteString("|")
			}
		}
		return line.String()
	}

	// Write header row
	result.WriteString(formatRow(headers))
	result.WriteString("\n")

	// Write separator
	for i, width := range colWidths {
		result.WriteString("-")
		result.WriteString(strings.Repeat("-", width))
		result.WriteString("-")
		if i < len(colWidths)-1 {
			result.WriteString("+")
		}
	}
	result.WriteString("\n")

	// Write data rows
	for _, row := range rows {
		result.WriteString(formatRow(row))
		result.WriteString("\n")
	}

	return strings.TrimSuffix(result.String(), "\n")
}
