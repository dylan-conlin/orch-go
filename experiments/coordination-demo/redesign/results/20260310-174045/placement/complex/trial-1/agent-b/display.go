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

// FormatTable formats headers and rows as an aligned text table.
// It auto-sizes each column based on the widest content (including header),
// handles ANSI-colored content correctly, and separates headers from data
// rows with a line of dashes. Rows with fewer columns than headers are padded
// with empty strings; extra columns in rows are ignored. Each column is
// separated by " | " in data rows and "+" in the separator row.
func FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	// Calculate column widths based on max width in each column
	colWidths := make([]int, len(headers))

	// Initialize with header widths
	for i, h := range headers {
		colWidths[i] = len(StripANSI(h))
	}

	// Update with max row content widths
	for _, row := range rows {
		for i := 0; i < len(headers); i++ {
			var content string
			if i < len(row) {
				content = row[i]
			}
			width := len(StripANSI(content))
			if width > colWidths[i] {
				colWidths[i] = width
			}
		}
	}

	var result strings.Builder

	// Format header row
	headerCells := make([]string, len(headers))
	for i, h := range headers {
		headerCells[i] = padRight(h, colWidths[i])
	}
	result.WriteString(" " + strings.Join(headerCells, " | ") + " \n")

	// Write separator line
	sepCells := make([]string, len(colWidths))
	for i, width := range colWidths {
		sepCells[i] = strings.Repeat("-", width+2)
	}
	result.WriteString(strings.Join(sepCells, "+") + "\n")

	// Format data rows
	for _, row := range rows {
		dataCells := make([]string, len(headers))
		for i := 0; i < len(headers); i++ {
			var content string
			if i < len(row) {
				content = row[i]
			}
			dataCells[i] = padRight(content, colWidths[i])
		}
		result.WriteString(" " + strings.Join(dataCells, " | ") + " \n")
	}

	return strings.TrimSuffix(result.String(), "\n")
}

// padRight pads a string on the right with spaces to match the specified width.
// It uses StripANSI to calculate the true visual width of colored strings.
func padRight(s string, width int) string {
	visualWidth := len(StripANSI(s))
	if visualWidth >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visualWidth)
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
