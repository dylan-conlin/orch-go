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
// Each column width is determined by the widest content in that column (including header).
// ANSI-colored content is handled correctly by using StripANSI for width calculation.
// Columns are separated by " | " and the header is separated from data rows by a dash line.
// If rows have fewer columns than headers, missing cells are treated as empty strings.
// If rows have more columns than headers, extra columns are included as right-aligned data.
func FormatTable(headers []string, rows [][]string) string {
	// Handle edge case: no headers
	if len(headers) == 0 {
		return ""
	}

	// Determine the number of columns
	maxCols := len(headers)
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	// Calculate column widths based on content (using StripANSI for visual width)
	colWidths := make([]int, maxCols)
	for i, header := range headers {
		if i < maxCols {
			colWidths[i] = len(StripANSI(header))
		}
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < maxCols {
				width := len(StripANSI(cell))
				if width > colWidths[i] {
					colWidths[i] = width
				}
			}
		}
	}

	// Build the table
	var result strings.Builder

	// Write header row
	for i := 0; i < maxCols; i++ {
		if i > 0 {
			result.WriteString(" | ")
		}
		header := ""
		if i < len(headers) {
			header = headers[i]
		}
		result.WriteString(header)
		// Pad all columns except the last
		if i < maxCols-1 {
			padding := colWidths[i] - len(StripANSI(header))
			result.WriteString(strings.Repeat(" ", padding))
		}
	}
	result.WriteString("\n")

	// Write separator line (dashes and + for column dividers)
	for i := 0; i < maxCols; i++ {
		if i > 0 {
			result.WriteString("+")
		}
		result.WriteString(strings.Repeat("-", colWidths[i]))
	}
	result.WriteString("\n")

	// Write data rows
	for _, row := range rows {
		for i := 0; i < maxCols; i++ {
			if i > 0 {
				result.WriteString(" | ")
			}
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			result.WriteString(cell)
			// Pad all columns except the last
			if i < maxCols-1 {
				padding := colWidths[i] - len(StripANSI(cell))
				result.WriteString(strings.Repeat(" ", padding))
			}
		}
		result.WriteString("\n")
	}

	return result.String()
}
