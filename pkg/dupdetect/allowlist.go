package dupdetect

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

const allowlistFilename = ".dupdetectignore"

// LoadAllowlistFile reads a .dupdetectignore file from projectDir and returns
// the patterns it contains. Returns nil, nil if the file does not exist.
//
// File format:
//   - One pattern per line
//   - Lines starting with # are comments
//   - Blank lines are ignored
//   - Single patterns use filepath.Match syntax (e.g., "(Logger).Log*")
//     Both functions in a pair must match the same pattern.
//   - Pair patterns use "X <-> Y" syntax (e.g., "(Logger).Log* <-> WriteCheckpoint")
//     Each side is a glob; the pair is suppressed if either ordering matches.
func LoadAllowlistFile(projectDir string) ([]string, error) {
	path := filepath.Join(projectDir, allowlistFilename)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var patterns []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns, scanner.Err()
}

// isAllowlisted returns true if both functions in the pair match the same
// allowlist pattern. Each pattern is checked independently — the pair is
// suppressed only when both sides match a single pattern.
func isAllowlisted(a, b string, allowlist []string) bool {
	_, matched := allowlistedPattern(a, b, allowlist)
	return matched
}

// allowlistedPattern returns the pattern that suppressed a pair, and whether
// a match was found. Returns ("", false) if no pattern matches.
//
// Two pattern formats are supported:
//   - Single pattern: "Log*" — both functions must match the same pattern
//   - Pair pattern: "FuncA <-> FuncB" — each side is a glob; the pair is
//     suppressed if (a matches left AND b matches right) OR vice versa
func allowlistedPattern(a, b string, allowlist []string) (string, bool) {
	for _, pattern := range allowlist {
		if left, right, ok := parsePairPattern(pattern); ok {
			matchAL, _ := filepath.Match(left, a)
			matchBR, _ := filepath.Match(right, b)
			matchAR, _ := filepath.Match(right, a)
			matchBL, _ := filepath.Match(left, b)
			if (matchAL && matchBR) || (matchAR && matchBL) {
				return pattern, true
			}
			continue
		}
		matchA, _ := filepath.Match(pattern, a)
		matchB, _ := filepath.Match(pattern, b)
		if matchA && matchB {
			return pattern, true
		}
	}
	return "", false
}

// parsePairPattern checks if a pattern uses pair syntax ("X <-> Y").
// Returns the left and right globs and true, or zero values and false.
func parsePairPattern(pattern string) (left, right string, ok bool) {
	const sep = " <-> "
	idx := strings.Index(pattern, sep)
	if idx < 0 {
		return "", "", false
	}
	left = strings.TrimSpace(pattern[:idx])
	right = strings.TrimSpace(pattern[idx+len(sep):])
	if left == "" || right == "" {
		return "", "", false
	}
	return left, right, true
}
