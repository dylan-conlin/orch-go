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
// Each column is auto-sized based on the widest content in that column
// (including headers). ANSI color codes in content are preserved but don't
// count toward column width. The header row is separated from data rows by
// a line of dashes. Handles edge cases: empty rows (headers only), rows with
// fewer columns than headers (padded with empty cells), and nil rows.
func FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	// Calculate column widths based on headers and all rows
	colWidths := make([]int, len(headers))

	// Start with header widths (using StripANSI for accurate width)
	for i, h := range headers {
		colWidths[i] = len(StripANSI(h))
	}

	// Update with row widths
	for _, row := range rows {
		for i, cell := range row {
			if i < len(headers) {
				cellWidth := len(StripANSI(cell))
				if cellWidth > colWidths[i] {
					colWidths[i] = cellWidth
				}
			}
		}
	}

	var result strings.Builder

	// Render header row
	renderRow(&result, headers, colWidths)

	// Render separator line
	renderSeparator(&result, colWidths)

	// Render data rows
	for _, row := range rows {
		// Pad row to match number of headers
		paddedRow := make([]string, len(headers))
		for i := 0; i < len(headers); i++ {
			if i < len(row) {
				paddedRow[i] = row[i]
			}
		}
		renderRow(&result, paddedRow, colWidths)
	}

	return result.String()
}

// renderRow writes a single row to the result buffer, padding cells to their
// column widths. Padding accounts for ANSI codes when calculating visual width.
func renderRow(sb *strings.Builder, cells []string, colWidths []int) {
	sb.WriteString("| ")
	for i, cell := range cells {
		// Calculate padding needed based on visual width
		visualWidth := len(StripANSI(cell))
		padding := colWidths[i] - visualWidth
		paddedCell := cell + strings.Repeat(" ", padding)
		sb.WriteString(paddedCell)

		if i < len(cells)-1 {
			sb.WriteString(" | ")
		} else {
			sb.WriteString(" |")
		}
	}
	sb.WriteString("\n")
}

// renderSeparator writes a separator line with dashes and plus signs.
func renderSeparator(sb *strings.Builder, colWidths []int) {
	sb.WriteString("+")
	for _, width := range colWidths {
		sb.WriteString(strings.Repeat("-", width+2))
		sb.WriteString("+")
	}
	sb.WriteString("\n")
}
