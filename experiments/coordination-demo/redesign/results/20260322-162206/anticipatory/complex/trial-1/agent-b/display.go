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

// FormatTable renders headers and rows as an aligned text table with column
// auto-sizing based on content width. ANSI-colored content is correctly
// handled using StripANSI for visual width calculation. The table includes
// a header separator row. Handles edge cases: empty rows (headers only), rows
// with fewer columns than headers (padded with empty cells), and rows with
// more columns than headers (extra columns are included).
func FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	// Determine number of columns: max of headers and any row
	numCols := len(headers)
	for _, row := range rows {
		if len(row) > numCols {
			numCols = len(row)
		}
	}

	// Calculate column widths based on header and all row content
	colWidths := make([]int, numCols)

	// Initialize with header widths
	for i := 0; i < len(headers); i++ {
		colWidths[i] = len(StripANSI(headers[i]))
	}

	// Update widths based on row content
	for _, row := range rows {
		for i, cell := range row {
			if i < numCols {
				w := len(StripANSI(cell))
				if w > colWidths[i] {
					colWidths[i] = w
				}
			}
		}
	}

	var result strings.Builder

	// Format header row
	for i := 0; i < numCols; i++ {
		result.WriteString("| ")
		cell := ""
		if i < len(headers) {
			cell = headers[i]
		}
		strippedWidth := len(StripANSI(cell))
		padding := colWidths[i] - strippedWidth
		result.WriteString(cell)
		result.WriteString(strings.Repeat(" ", padding))
		result.WriteString(" ")
	}
	result.WriteString("|")

	// If no data rows, just return header and separator
	if len(rows) == 0 {
		result.WriteString("\n")
		for _, w := range colWidths {
			result.WriteString("+")
			result.WriteString(strings.Repeat("-", w+2))
		}
		result.WriteString("+")
		return result.String()
	}

	// Write separator and data rows
	result.WriteString("\n")
	for _, w := range colWidths {
		result.WriteString("+")
		result.WriteString(strings.Repeat("-", w+2))
	}
	result.WriteString("+\n")

	// Format data rows
	for i, row := range rows {
		for j := 0; j < numCols; j++ {
			result.WriteString("| ")
			cell := ""
			if j < len(row) {
				cell = row[j]
			}
			strippedWidth := len(StripANSI(cell))
			padding := colWidths[j] - strippedWidth
			result.WriteString(cell)
			result.WriteString(strings.Repeat(" ", padding))
			result.WriteString(" ")
		}
		result.WriteString("|")
		if i < len(rows)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
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
