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
// pipe borders and space padding. Each column is auto-sized based on the
// widest content (using visual width, ignoring ANSI codes). Headers are
// separated from data rows with a dashes line. Rows with fewer columns than
// headers are padded with empty cells. Returns a multi-line string.
func FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	// Handle nil rows
	if rows == nil {
		rows = [][]string{}
	}

	// Calculate column widths based on content
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = visualWidth(header)
	}

	for _, row := range rows {
		for i := 0; i < len(headers); i++ {
			var cellContent string
			if i < len(row) {
				cellContent = row[i]
			}
			w := visualWidth(cellContent)
			if w > colWidths[i] {
				colWidths[i] = w
			}
		}
	}

	// Add padding (1 space on each side)
	const padding = 1
	for i := range colWidths {
		colWidths[i] += 2 * padding
	}

	var buf strings.Builder

	// Helper to format a row with proper padding
	formatRow := func(cells []string) string {
		var rowStr strings.Builder
		rowStr.WriteString("|")
		for i := range headers {
			var cell string
			if i < len(cells) {
				cell = cells[i]
			}
			// Calculate padding needed
			cellVisWidth := visualWidth(cell)
			availableWidth := colWidths[i] - 2*padding
			spacePad := availableWidth - cellVisWidth
			rowStr.WriteString(" ")
			rowStr.WriteString(cell)
			rowStr.WriteString(strings.Repeat(" ", spacePad))
			rowStr.WriteString(" |")
		}
		return rowStr.String()
	}

	// Write header
	buf.WriteString(formatRow(headers))
	buf.WriteString("\n")

	// Write separator line
	buf.WriteString("|")
	for i := range headers {
		buf.WriteString(" ")
		buf.WriteString(strings.Repeat("-", colWidths[i]-2*padding))
		buf.WriteString(" |")
	}
	buf.WriteString("\n")

	// Write data rows
	for _, row := range rows {
		buf.WriteString(formatRow(row))
		buf.WriteString("\n")
	}

	return strings.TrimSuffix(buf.String(), "\n")
}

// visualWidth calculates the visual display width of a string,
// accounting for ANSI escape codes which don't take up visual space.
func visualWidth(s string) int {
	stripped := StripANSI(s)
	return len([]rune(stripped))
}
