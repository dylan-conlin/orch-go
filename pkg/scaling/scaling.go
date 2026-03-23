// Package scaling provides numeric and string scaling utilities.
package scaling

import "strings"

// Normalize lowercases a string and trims whitespace.
func Normalize(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

// Clamp restricts v to the range [min, max].
func Clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// Wrap breaks s into lines of at most width characters, splitting on spaces.
func Wrap(s string, width int) string {
	if width <= 0 || len(s) <= width {
		return s
	}
	var lines []string
	for len(s) > width {
		idx := strings.LastIndex(s[:width+1], " ")
		if idx <= 0 {
			idx = width
		}
		lines = append(lines, s[:idx])
		s = strings.TrimSpace(s[idx:])
	}
	if len(s) > 0 {
		lines = append(lines, s)
	}
	return strings.Join(lines, "\n")
}
