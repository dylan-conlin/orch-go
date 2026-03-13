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
// auto-sized columns. Each column is sized based on the widest content,
// including headers. ANSI-colored content is handled correctly using
// StripANSI to calculate true visual widths. The header is separated
// from data rows with a line of dashes.
//
// Edge cases:
// - Empty rows (headers only) produces header + separator only
// - Rows with fewer columns than headers are padded with empty cells
// - Rows with more columns than headers include the extra columns
// - Nil or empty header slice returns empty string
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

	// Calculate column widths based on all content (headers + rows)
	colWidths := make([]int, numCols)

	// Initialize with header widths
	for i, header := range headers {
		colWidths[i] = visualWidth(header)
	}

	// Update with row content widths
	for _, row := range rows {
		for i := 0; i < numCols; i++ {
			var content string
			if i < len(row) {
				content = row[i]
			}
			width := visualWidth(content)
			if width > colWidths[i] {
				colWidths[i] = width
			}
		}
	}

	// Pad headers if table has more columns than headers
	paddedHeaders := make([]string, numCols)
	for i, header := range headers {
		paddedHeaders[i] = header
	}

	// Build table lines
	var result strings.Builder

	// Header row
	result.WriteString(buildTableRow(paddedHeaders, colWidths))
	result.WriteString("\n")

	// Separator row
	result.WriteString(buildSeparatorRow(colWidths))
	result.WriteString("\n")

	// Data rows
	for _, row := range rows {
		// Pad row to match actual column count
		paddedRow := make([]string, numCols)
		for i := 0; i < numCols; i++ {
			if i < len(row) {
				paddedRow[i] = row[i]
			} else {
				paddedRow[i] = ""
			}
		}
		result.WriteString(buildTableRow(paddedRow, colWidths))
		result.WriteString("\n")
	}

	// Remove trailing newline
	output := result.String()
	if len(output) > 0 {
		output = output[:len(output)-1]
	}
	return output
}

// visualWidth returns the visual display width of a string, ignoring ANSI codes.
func visualWidth(s string) int {
	return len([]rune(StripANSI(s)))
}

// buildTableRow builds a single table row with proper padding and alignment.
func buildTableRow(cells []string, colWidths []int) string {
	var row strings.Builder
	row.WriteString("| ")
	for i, cell := range cells {
		padded := padToWidth(cell, colWidths[i])
		row.WriteString(padded)
		row.WriteString(" | ")
	}
	return row.String()
}

// buildSeparatorRow builds the header/data separator row with dashes.
func buildSeparatorRow(colWidths []int) string {
	var sep strings.Builder
	sep.WriteString("|")
	for _, width := range colWidths {
		sep.WriteString(strings.Repeat("-", width+2))
		sep.WriteString("|")
	}
	return sep.String()
}

// padToWidth right-pads a string with spaces to reach the target visual width.
// ANSI codes are preserved but don't count toward width. If the string is
// already wider than width, it is returned unchanged.
func padToWidth(s string, width int) string {
	currentWidth := visualWidth(s)
	if currentWidth >= width {
		return s
	}
	padding := width - currentWidth
	return s + strings.Repeat(" ", padding)
}
