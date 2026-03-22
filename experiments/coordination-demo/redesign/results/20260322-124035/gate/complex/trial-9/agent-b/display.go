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

// FormatTable renders headers and rows as an aligned text table with auto-sized columns.
// Column widths are determined by the widest content in each column (including headers).
// The header row is separated from data rows with a line of dashes (only if data rows exist).
// ANSI escape codes are handled correctly using StripANSI for accurate width calculation.
// Rows with fewer columns than headers are padded with empty strings.
// If rows is empty or nil, only the header row is shown.
// If headers is empty, an empty string is returned.
func FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	// Determine the number of columns (at minimum, use header count)
	numCols := len(headers)
	for _, row := range rows {
		if len(row) > numCols {
			numCols = len(row)
		}
	}

	// Calculate column widths
	colWidths := make([]int, numCols)
	for i, h := range headers {
		colWidths[i] = len(StripANSI(h))
	}
	for _, row := range rows {
		for i, cell := range row {
			width := len(StripANSI(cell))
			if width > colWidths[i] {
				colWidths[i] = width
			}
		}
	}

	var result strings.Builder

	// Format header row
	headerLine := formatRowLine(headers, colWidths, numCols)
	result.WriteString(headerLine)

	// Only add separator and data rows if there are data rows
	if len(rows) > 0 {
		result.WriteString("\n")

		// Format separator
		for i := 0; i < numCols; i++ {
			result.WriteString(strings.Repeat("-", colWidths[i]))
			if i < numCols-1 {
				result.WriteString(" ")
			}
		}

		// Format data rows
		for _, row := range rows {
			result.WriteString("\n")
			dataLine := formatRowLine(row, colWidths, numCols)
			result.WriteString(dataLine)
		}
	}

	return result.String()
}

// formatRowLine formats a single row with proper padding and spacing, trimming trailing whitespace.
func formatRowLine(cells []string, colWidths []int, numCols int) string {
	var line strings.Builder
	for i := 0; i < numCols; i++ {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		if i < numCols-1 {
			line.WriteString(padCell(cell, colWidths[i]))
			line.WriteString(" ")
		} else {
			line.WriteString(cell)
		}
	}
	return strings.TrimRight(line.String(), " ")
}

// padCell right-pads a cell with spaces to match the desired visual width,
// accounting for ANSI escape codes using StripANSI.
func padCell(cell string, width int) string {
	stripped := StripANSI(cell)
	padding := width - len(stripped)
	if padding <= 0 {
		return cell
	}
	return cell + strings.Repeat(" ", padding)
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
