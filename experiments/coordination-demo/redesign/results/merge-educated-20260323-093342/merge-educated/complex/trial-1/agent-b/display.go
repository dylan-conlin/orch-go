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
// Each column width is determined by the widest content in that column (including the header).
// ANSI color codes in cell content are preserved but do not count toward visual width.
// Rows with fewer columns than headers are padded with empty cells.
// Nil rows slice is treated as having no data rows (headers only).
func FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	// Calculate visual width of a string, stripping ANSI codes and counting runes
	visualWidth := func(s string) int {
		return len([]rune(StripANSI(s)))
	}

	// Determine column widths based on headers and all rows
	colWidths := make([]int, len(headers))
	for i, h := range headers {
		colWidths[i] = visualWidth(h)
	}
	for _, row := range rows {
		for i := 0; i < len(headers); i++ {
			var cell string
			if i < len(row) {
				cell = row[i]
			}
			w := visualWidth(cell)
			if w > colWidths[i] {
				colWidths[i] = w
			}
		}
	}

	// Helper to pad a cell to a target visual width, preserving ANSI codes
	padCell := func(s string, width int) string {
		currentWidth := visualWidth(s)
		if currentWidth >= width {
			return s
		}
		padding := width - currentWidth
		return s + strings.Repeat(" ", padding)
	}

	// Helper to format a row with padding
	formatRow := func(cells []string) string {
		var parts []string
		for i := 0; i < len(headers); i++ {
			var cell string
			if i < len(cells) {
				cell = cells[i]
			}
			parts = append(parts, padCell(cell, colWidths[i]))
		}
		return "| " + strings.Join(parts, " | ") + " |"
	}

	// Helper to create separator line
	createSeparator := func() string {
		var parts []string
		for _, w := range colWidths {
			parts = append(parts, strings.Repeat("-", w))
		}
		return "+-" + strings.Join(parts, "-+-") + "-+"
	}

	// Build output
	var result strings.Builder
	result.WriteString(formatRow(headers))
	result.WriteString("\n")
	result.WriteString(createSeparator())
	result.WriteString("\n")

	for _, row := range rows {
		result.WriteString(formatRow(row))
		result.WriteString("\n")
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
