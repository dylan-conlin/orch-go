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
// It auto-sizes each column based on the widest content in that column
// (including header), and separates the header from data rows with a line of dashes.
// Uses StripANSI for width calculation to handle ANSI-colored content correctly.
// Returns an empty string if headers is empty.
// Handles rows with fewer columns than headers by padding with empty cells.
func FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	// Calculate column widths based on stripped content
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = len(StripANSI(header))
	}

	for _, row := range rows {
		for i := 0; i < len(headers); i++ {
			var width int
			if i < len(row) {
				width = len(StripANSI(row[i]))
			}
			if width > colWidths[i] {
				colWidths[i] = width
			}
		}
	}

	// Helper to pad content considering ANSI codes
	padContent := func(content string, width int) string {
		stripped := StripANSI(content)
		padding := width - len(stripped)
		return content + strings.Repeat(" ", padding)
	}

	// Build separator line
	var separatorParts []string
	for _, width := range colWidths {
		separatorParts = append(separatorParts, strings.Repeat("-", width))
	}
	separator := strings.Join(separatorParts, "-+-")

	// Build header line
	var headerParts []string
	for i, header := range headers {
		headerParts = append(headerParts, padContent(header, colWidths[i]))
	}
	headerLine := strings.Join(headerParts, " | ")

	// Build data lines
	var lines []string
	for _, row := range rows {
		var dataParts []string
		for i := 0; i < len(headers); i++ {
			var content string
			if i < len(row) {
				content = row[i]
			}
			dataParts = append(dataParts, padContent(content, colWidths[i]))
		}
		lines = append(lines, strings.Join(dataParts, " | "))
	}

	// Combine all parts
	var result strings.Builder
	result.WriteString(headerLine)
	result.WriteString("\n")
	result.WriteString(separator)
	if len(lines) > 0 {
		result.WriteString("\n")
		result.WriteString(strings.Join(lines, "\n"))
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
