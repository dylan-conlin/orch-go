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

// padRight pads a string (including ANSI-colored content) to width using spaces.
// It calculates the true visual width using StripANSI and pads accordingly.
func padRight(s string, width int) string {
	trueWidth := len(StripANSI(s))
	if trueWidth >= width {
		return s
	}
	return s + strings.Repeat(" ", width-trueWidth)
}

// FormatTable renders headers and rows as an aligned text table.
// It calculates column widths based on the widest content in each column
// (including headers), using StripANSI to correctly measure ANSI-colored content.
// The header row is separated from data rows with a line of dashes.
// Handles edge cases: nil rows, rows with fewer columns than headers, and
// rows with more columns than headers (extends table width).
func FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(StripANSI(h))
	}

	// Update widths based on row content
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				cellWidth := len(StripANSI(cell))
				if cellWidth > widths[i] {
					widths[i] = cellWidth
				}
			} else {
				// Row has more columns than headers
				widths = append(widths, len(StripANSI(cell)))
			}
		}
	}

	// Build table
	var result strings.Builder

	// Write header
	result.WriteString("| ")
	for i, h := range headers {
		result.WriteString(padRight(h, widths[i]))
		if i < len(headers)-1 {
			result.WriteString(" | ")
		}
	}
	result.WriteString(" |\n")

	// Write separator
	result.WriteString("|")
	for i := range headers {
		result.WriteString(strings.Repeat("-", widths[i]+2))
		result.WriteString("|")
	}
	result.WriteString("\n")

	// Write rows
	for _, row := range rows {
		result.WriteString("| ")
		for i := 0; i < len(headers); i++ {
			var cell string
			if i < len(row) {
				cell = row[i]
			}
			result.WriteString(padRight(cell, widths[i]))
			if i < len(headers)-1 {
				result.WriteString(" | ")
			}
		}
		result.WriteString(" |\n")
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
