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
// Each column width is auto-sized based on the widest content in that column
// (including the header). The header is separated from data rows with a line
// of dashes. ANSI-colored content is handled correctly using StripANSI to
// calculate true visual widths. Handles edge cases: empty rows (headers only),
// rows with fewer columns than headers (padded with empty cells), and rows
// with more columns than headers (extra columns ignored).
func FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	// Handle nil or empty rows
	if rows == nil {
		rows = [][]string{}
	}

	// Calculate column widths based on headers and all rows
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = len(StripANSI(header))
	}

	for _, row := range rows {
		for i := 0; i < len(headers); i++ {
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

	// Build the table
	var result strings.Builder

	// Add header row
	result.WriteString(formatTableRow(headers, colWidths))
	result.WriteString("\n")

	// Add separator
	for i, width := range colWidths {
		result.WriteString(strings.Repeat("-", width+2)) // +2 for padding
		if i < len(colWidths)-1 {
			result.WriteString("|") // pipe between columns
		}
	}
	result.WriteString("\n")

	// Add data rows
	for i, row := range rows {
		result.WriteString(formatTableRow(row, colWidths))
		if i < len(rows)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// formatTableRow formats a single row with proper column alignment and spacing.
func formatTableRow(cells []string, colWidths []int) string {
	var result strings.Builder
	for i, width := range colWidths {
		var cell string
		if i < len(cells) {
			cell = cells[i]
		}

		// Pad the cell to the column width, accounting for ANSI codes
		visualWidth := len(StripANSI(cell))
		paddingNeeded := width - visualWidth
		paddedCell := cell + strings.Repeat(" ", paddingNeeded)

		// Add cell with spacing: space | cell | space
		result.WriteString(" ")
		result.WriteString(paddedCell)
		result.WriteString(" ")

		// Add separator between columns
		if i < len(colWidths)-1 {
			result.WriteString("|")
		}
	}
	return result.String()
}
