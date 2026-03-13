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
// Each column width is determined by the widest content in that column (including headers).
// The header row is separated from data rows by a dashed line.
// ANSI color codes in cell content are handled correctly using StripANSI for width calculation.
// Edge cases: empty rows returns header only; rows with fewer columns than headers are padded
// with empty strings; rows with more columns than headers are included; nil or empty headers
// return empty string.
func FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	// Determine final number of columns (max of headers and all rows)
	numCols := len(headers)
	for _, row := range rows {
		if len(row) > numCols {
			numCols = len(row)
		}
	}

	// Pad headers to numCols with empty strings
	paddedHeaders := make([]string, numCols)
	copy(paddedHeaders, headers)

	// Calculate column widths
	widths := make([]int, numCols)
	for i, h := range paddedHeaders {
		widths[i] = len(StripANSI(h))
	}
	for _, row := range rows {
		for i := 0; i < numCols; i++ {
			var cell string
			if i < len(row) {
				cell = row[i]
			}
			width := len(StripANSI(cell))
			if width > widths[i] {
				widths[i] = width
			}
		}
	}

	// Build the table
	var result strings.Builder

	// Header row
	for i, h := range paddedHeaders {
		if i > 0 {
			result.WriteString(" | ")
		}
		result.WriteString(padRight(h, widths[i]))
	}
	result.WriteString("\n")

	// Separator line
	for i := 0; i < numCols; i++ {
		if i > 0 {
			result.WriteString("-+-")
		}
		result.WriteString(strings.Repeat("-", widths[i]))
	}
	result.WriteString("\n")

	// Data rows
	for _, row := range rows {
		for i := 0; i < numCols; i++ {
			if i > 0 {
				result.WriteString(" | ")
			}
			var cell string
			if i < len(row) {
				cell = row[i]
			}
			result.WriteString(padRight(cell, widths[i]))
		}
		result.WriteString("\n")
	}

	// Trim trailing spaces from each line (but preserve ANSI codes)
	lines := strings.Split(strings.TrimSuffix(result.String(), "\n"), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n") + "\n"
}

// padRight pads s to width with spaces, accounting for ANSI codes in the string.
func padRight(s string, width int) string {
	stripped := StripANSI(s)
	padding := width - len(stripped)
	if padding < 0 {
		padding = 0
	}
	return s + strings.Repeat(" ", padding)
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
