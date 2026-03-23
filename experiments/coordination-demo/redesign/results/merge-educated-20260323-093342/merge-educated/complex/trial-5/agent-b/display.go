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

// padToWidth right-pads s with spaces to the target visual width, ignoring ANSI
// escape codes. If s is already wider than or equal to width, it is returned unchanged.
func padToWidth(s string, width int) string {
	visualWidth := len(StripANSI(s))
	if visualWidth >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visualWidth)
}

// FormatTable formats headers and rows as an aligned text table with auto-sized columns.
// Column widths are determined by the widest content in each column, including headers.
// ANSI escape codes are preserved in the output but do not count toward column width.
// Rows with fewer columns than headers are padded with empty cells; rows with more
// columns are truncated to match the header count.
func FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	// Calculate column widths based on visual width (ignoring ANSI codes)
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(StripANSI(h))
	}

	for _, row := range rows {
		for i := 0; i < len(headers); i++ {
			var cellContent string
			if i < len(row) {
				cellContent = row[i]
			}
			w := len(StripANSI(cellContent))
			if w > widths[i] {
				widths[i] = w
			}
		}
	}

	// Build the table
	var buf strings.Builder

	// Header row
	for i, h := range headers {
		if i > 0 {
			buf.WriteString(" | ")
		}
		buf.WriteString(" ")
		buf.WriteString(padToWidth(h, widths[i]))
		buf.WriteString(" ")
	}

	// Only add separator and data rows if there are rows
	if len(rows) > 0 {
		buf.WriteString("\n")

		// Separator line
		for i, w := range widths {
			if i > 0 {
				buf.WriteString("-+-")
			}
			buf.WriteString(strings.Repeat("-", w+2)) // +2 for padding spaces
		}

		// Data rows
		buf.WriteString("\n")
		for _, row := range rows {
			for i := 0; i < len(headers); i++ {
				if i > 0 {
					buf.WriteString(" | ")
				}
				var cellContent string
				if i < len(row) {
					cellContent = row[i]
				}
				buf.WriteString(" ")
				buf.WriteString(padToWidth(cellContent, widths[i]))
				buf.WriteString(" ")
			}
			buf.WriteString("\n")
		}
	}

	return strings.TrimSuffix(buf.String(), "\n")
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
