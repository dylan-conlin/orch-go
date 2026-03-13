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

// FormatTable renders headers and rows as an aligned text table with
// auto-sized columns based on the widest content in each column.
// Content widths are calculated using StripANSI to handle ANSI-colored
// text correctly. The header is separated from data rows by a line of dashes.
// Handles edge cases: empty rows (headers only), rows with fewer columns
// than headers (padded with empty cells), and rows with more columns than
// headers (extra columns added to the table).
func FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	// Determine the actual number of columns (max of headers and any row)
	numCols := len(headers)
	for _, row := range rows {
		if len(row) > numCols {
			numCols = len(row)
		}
	}

	// Pad headers to match numCols
	paddedHeaders := make([]string, numCols)
	copy(paddedHeaders, headers)

	// Calculate column widths based on visual width (stripping ANSI codes)
	colWidths := make([]int, numCols)
	for i, header := range paddedHeaders {
		colWidths[i] = len(StripANSI(header))
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

	// Build the table
	var result strings.Builder

	// Helper to create a separator line
	separator := func() string {
		var sep strings.Builder
		for _, width := range colWidths {
			sep.WriteString("+")
			sep.WriteString(strings.Repeat("-", width+2))
		}
		sep.WriteString("+\n")
		return sep.String()
	}

	// Helper to format a row (header or data)
	formatRow := func(row []string) string {
		var line strings.Builder
		for i, width := range colWidths {
			var cell string
			if i < len(row) {
				cell = row[i]
			}
			// Calculate padding needed based on visual width (ANSI-stripped)
			visualWidth := len(StripANSI(cell))
			padding := width - visualWidth
			line.WriteString("| ")
			line.WriteString(cell)
			line.WriteString(strings.Repeat(" ", padding))
			line.WriteString(" ")
		}
		line.WriteString("|\n")
		return line.String()
	}

	// Top separator
	result.WriteString(separator())

	// Headers
	result.WriteString(formatRow(paddedHeaders))

	// Header separator (only if there are rows to display)
	result.WriteString(separator())

	// Data rows
	for _, row := range rows {
		result.WriteString(formatRow(row))
	}

	// Bottom separator (only if there are rows to display)
	if len(rows) > 0 {
		result.WriteString(separator())
	}

	return result.String()
}
