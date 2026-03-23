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
// auto-sized columns. Column widths are determined by the widest content
// in each column (including headers). ANSI escape codes are handled correctly
// via StripANSI for accurate visual width calculation. The header row is
// separated from data rows with a line of dashes.
func FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	// Handle nil or empty rows
	if rows == nil {
		rows = [][]string{}
	}

	// Calculate column widths based on headers and content
	colWidths := make([]int, len(headers))
	for i, h := range headers {
		colWidths[i] = len(StripANSI(h))
	}

	for _, row := range rows {
		for i := 0; i < len(headers) && i < len(row); i++ {
			width := len(StripANSI(row[i]))
			if width > colWidths[i] {
				colWidths[i] = width
			}
		}
	}

	// Build header row
	headerLine := formatTableRow(headers, colWidths)

	// Build separator line
	separatorParts := make([]string, len(colWidths))
	for i, w := range colWidths {
		separatorParts[i] = strings.Repeat("-", w)
	}
	separator := "| " + strings.Join(separatorParts, " | ") + " |"

	// Build data rows
	var buf strings.Builder
	buf.WriteString(headerLine)
	buf.WriteString("\n")
	buf.WriteString(separator)

	for _, row := range rows {
		buf.WriteString("\n")
		buf.WriteString(formatTableRow(row, colWidths))
	}

	return buf.String()
}

// formatTableRow renders a single row with proper spacing to match column widths.
// Content is left-aligned and padded with spaces to reach the column width.
// ANSI codes are preserved but don't count toward width.
func formatTableRow(cols []string, widths []int) string {
	parts := make([]string, len(widths))
	for i := 0; i < len(widths); i++ {
		var content string
		if i < len(cols) {
			content = cols[i]
		}
		parts[i] = padToWidth(content, widths[i])
	}
	return "| " + strings.Join(parts, " | ") + " |"
}

// padToWidth pads content to reach the target visual width using spaces.
// ANSI codes are preserved but don't count toward the width.
func padToWidth(s string, width int) string {
	visualLen := len(StripANSI(s))
	if visualLen >= width {
		return s
	}
	padding := width - visualLen
	return s + strings.Repeat(" ", padding)
}
