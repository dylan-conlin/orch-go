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

// FormatTable renders headers and rows as an aligned text table with fixed-width columns.
// Columns are sized based on the widest content in each column (including headers).
// The header row is separated from data rows with a line of dashes.
// Rows with fewer columns than headers are padded with empty strings.
// Rows with more columns than headers are truncated.
// ANSI escape codes in content are preserved but don't count toward column width.
func FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	// If no rows, return headers only (simple space-separated)
	if len(rows) == 0 {
		return strings.Join(headers, " | ")
	}

	// Calculate column widths based on headers and all row content
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = visualWidth(h)
	}

	for _, row := range rows {
		for i := 0; i < len(headers); i++ {
			var content string
			if i < len(row) {
				content = row[i]
			}
			w := visualWidth(content)
			if w > widths[i] {
				widths[i] = w
			}
		}
	}

	var result strings.Builder

	// Header row
	headerCells := make([]string, len(headers))
	for i, h := range headers {
		headerCells[i] = padToWidth(h, widths[i])
	}
	result.WriteString(strings.Join(headerCells, " | "))
	result.WriteString("\n")

	// Separator line
	separators := make([]string, len(widths))
	for i, w := range widths {
		separators[i] = strings.Repeat("-", w)
	}
	result.WriteString(strings.Join(separators, "-+-"))
	result.WriteString("\n")

	// Data rows
	for _, row := range rows {
		cells := make([]string, len(headers))
		for i := 0; i < len(headers); i++ {
			var content string
			if i < len(row) {
				content = row[i]
			}
			cells[i] = padToWidth(content, widths[i])
		}
		result.WriteString(strings.Join(cells, " | "))
		result.WriteString("\n")
	}

	return strings.TrimSuffix(result.String(), "\n")
}

// visualWidth returns the display width of a string, accounting for ANSI escape codes.
// It uses the existing StripANSI function to remove codes before counting runes.
func visualWidth(s string) int {
	return len([]rune(StripANSI(s)))
}

// padToWidth right-pads a string with spaces to reach the target visual width.
// ANSI escape codes are preserved but don't count toward the width.
// If the string is already at or wider than the target width, it is returned unchanged.
func padToWidth(s string, width int) string {
	vw := visualWidth(s)
	if vw >= width {
		return s
	}
	return s + strings.Repeat(" ", width-vw)
}
