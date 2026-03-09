package workspace

import (
	"strconv"
	"strings"
	"time"
)

// ExtractKeywords extracts meaningful keywords from a workspace name for matching.
// Workspace names follow pattern: {project}-{skill}-{topic}-{date}-{hash}
// Example: "og-inv-skillc-deploy-06jan-ed96" -> ["skillc", "deploy"]
func ExtractKeywords(workspaceName string) []string {
	parts := strings.Split(workspaceName, "-")
	if len(parts) < 3 {
		return nil
	}

	var keywords []string

	// Skip prefix parts that are likely project or skill markers
	prefixSet := map[string]bool{
		"og": true, "inv": true, "feat": true, "fix": true,
		"debug": true, "audit": true, "impl": true, "arch": true, "research": true,
	}

	for _, part := range parts {
		// Skip short parts (likely hash or date)
		if len(part) <= 2 {
			continue
		}
		// Skip parts that look like dates (e.g., "06jan", "2026")
		if len(part) == 5 && strings.Contains(part, "jan") || strings.Contains(part, "feb") ||
			strings.Contains(part, "mar") || strings.Contains(part, "apr") ||
			strings.Contains(part, "may") || strings.Contains(part, "jun") ||
			strings.Contains(part, "jul") || strings.Contains(part, "aug") ||
			strings.Contains(part, "sep") || strings.Contains(part, "oct") ||
			strings.Contains(part, "nov") || strings.Contains(part, "dec") {
			continue
		}
		// Skip common prefixes
		if prefixSet[strings.ToLower(part)] {
			continue
		}
		// Skip parts that look like short hashes (4 hex chars at end)
		if len(part) == 4 && IsHexLike(part) {
			continue
		}
		keywords = append(keywords, part)
	}

	return keywords
}

// IsHexLike returns true if the string looks like a short hex hash (all lowercase letters/digits).
func IsHexLike(s string) bool {
	for _, c := range s {
		if !((c >= 'a' && c <= 'f') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

// ExtractDate parses the date suffix from a workspace name.
// Workspace names end with a date like "24dec" or "5jan".
// Returns zero time if no valid date suffix is found.
func ExtractDate(name string) time.Time {
	months := map[string]time.Month{
		"jan": time.January, "feb": time.February, "mar": time.March,
		"apr": time.April, "may": time.May, "jun": time.June,
		"jul": time.July, "aug": time.August, "sep": time.September,
		"oct": time.October, "nov": time.November, "dec": time.December,
	}

	parts := strings.Split(name, "-")
	if len(parts) == 0 {
		return time.Time{}
	}
	lastPart := strings.ToLower(parts[len(parts)-1])

	// Pattern: 1-2 digits followed by 3-letter month abbreviation (e.g., "24dec", "5jan")
	if len(lastPart) < 4 || len(lastPart) > 5 {
		return time.Time{}
	}

	// Extract the month abbreviation (last 3 chars)
	monthStr := lastPart[len(lastPart)-3:]
	month, ok := months[monthStr]
	if !ok {
		return time.Time{}
	}

	// Extract the day (remaining digits)
	dayStr := lastPart[:len(lastPart)-3]
	day, err := strconv.Atoi(dayStr)
	if err != nil || day < 1 || day > 31 {
		return time.Time{}
	}

	// Use current year, adjusting for year boundary
	now := time.Now()
	year := now.Year()
	parsedDate := time.Date(year, month, day, 12, 0, 0, 0, time.Local)

	// If the parsed date is more than a week in the future, assume it's from last year
	if parsedDate.After(now.AddDate(0, 0, 7)) {
		parsedDate = time.Date(year-1, month, day, 12, 0, 0, 0, time.Local)
	}

	return parsedDate
}
