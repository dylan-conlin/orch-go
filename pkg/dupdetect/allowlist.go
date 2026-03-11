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
//   - Patterns use filepath.Match syntax (e.g., "(Logger).Log*")
//
// Semantics: A duplicate pair is suppressed when BOTH functions match the
// SAME pattern line. This means intentionally parallel code (like Logger.Log*
// methods) can be allowlisted without hiding real duplication involving those
// functions.
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
	for _, pattern := range allowlist {
		matchA, _ := filepath.Match(pattern, a)
		matchB, _ := filepath.Match(pattern, b)
		if matchA && matchB {
			return true
		}
	}
	return false
}
