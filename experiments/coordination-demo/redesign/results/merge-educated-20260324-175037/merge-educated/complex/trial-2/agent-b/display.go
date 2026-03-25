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
// Columns are auto-sized based on the widest content (ANSI codes ignored).
// A separator line of dashes appears between the header and data rows.
//
// Rows with fewer columns than headers are right-padded with empty strings.
// Rows with more columns than headers have extra columns truncated.
// Empty rows (nil or zero-length) are skipped.
func FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	// Calculate column widths based on visual width (strip ANSI codes)
	colWidths := make([]int, len(headers))
	for i, h := range headers {
		colWidths[i] = len([]rune(StripANSI(h)))
	}

	// Update column widths based on row data
	for _, row := range rows {
		for i := 0; i < len(headers) && i < len(row); i++ {
			cellWidth := len([]rune(StripANSI(row[i])))
			if cellWidth > colWidths[i] {
				colWidths[i] = cellWidth
			}
		}
	}

	// Debug output
	fmt.Printf("DEBUG colWidths: %v\n", colWidths)

	var b strings.Builder

	// Format header row
	for i, h := range headers {
		if i > 0 {
			b.WriteString(" | ")
		}
		b.WriteString(h)
		// Don't pad the last column
		if i < len(headers)-1 {
			stripped := StripANSI(h)
			visualWidth := len([]rune(stripped))
			padding := colWidths[i] - visualWidth
			b.WriteString(strings.Repeat(" ", padding))
		}
	}
	b.WriteString("\n")

	// Separator line
	for i, width := range colWidths {
		if i > 0 {
			b.WriteString("+")
		}
		// Add 1 to width for spacing (except last column)
		dashCount := width
		if i < len(headers)-1 {
			dashCount = width + 1
		}
		b.WriteString(strings.Repeat("-", dashCount))
	}
	b.WriteString("\n")

	// Format data rows
	for _, row := range rows {
		if len(row) == 0 {
			continue // Skip empty rows
		}

		// Determine the last non-empty column for this row
		lastCol := len(row) - 1
		if lastCol >= len(headers) {
			lastCol = len(headers) - 1
		}

		for i := 0; i <= lastCol; i++ {
			if i > 0 {
				b.WriteString(" | ")
			}

			cell := row[i]
			b.WriteString(cell)

			// Don't pad the last column in the row
			if i < lastCol {
				stripped := StripANSI(cell)
				visualWidth := len([]rune(stripped))
				padding := colWidths[i] - visualWidth
				b.WriteString(strings.Repeat(" ", padding))
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}
