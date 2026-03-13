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
// Column widths are auto-sized based on the widest content in each column
// (including headers). ANSI-colored content is handled correctly using StripANSI
// for width calculation. Handles edge cases: empty rows, mismatched column counts,
// and nil rows. If rows have more columns than headers, they are included as
// additional columns with empty header cells.
func FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	// Determine the maximum number of columns
	maxCols := len(headers)
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	// Calculate column widths
	colWidths := make([]int, maxCols)

	// Initialize with header widths
	for i := 0; i < len(headers); i++ {
		colWidths[i] = len(StripANSI(headers[i]))
	}

	// Update with row content widths
	for _, row := range rows {
		for i, cell := range row {
			width := len(StripANSI(cell))
			if width > colWidths[i] {
				colWidths[i] = width
			}
		}
	}

	// Build the table
	var result strings.Builder

	// Header row
	result.WriteString("|")
	for i := 0; i < maxCols; i++ {
		result.WriteString(" ")
		var header string
		if i < len(headers) {
			header = headers[i]
		}
		width := colWidths[i]
		visualWidth := len(StripANSI(header))
		paddedHeader := header + strings.Repeat(" ", width-visualWidth)
		result.WriteString(paddedHeader)
		result.WriteString(" ")
		result.WriteString("|")
	}
	result.WriteString("\n")

	// Separator
	result.WriteString("|")
	for i := 0; i < maxCols; i++ {
		result.WriteString("-")
		result.WriteString(strings.Repeat("-", colWidths[i]))
		result.WriteString("-")
		result.WriteString("|")
	}
	result.WriteString("\n")

	// Data rows
	for _, row := range rows {
		result.WriteString("|")
		for i := 0; i < maxCols; i++ {
			result.WriteString(" ")
			var cell string
			if i < len(row) {
				cell = row[i]
			}
			width := colWidths[i]
			visualWidth := len(StripANSI(cell))
			paddedCell := cell + strings.Repeat(" ", width-visualWidth)
			result.WriteString(paddedCell)
			result.WriteString(" ")
			result.WriteString("|")
		}
		result.WriteString("\n")
	}

	return result.String()
}
