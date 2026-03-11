package dupdetect

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CheckModifiedFiles scans all Go files in dir and returns duplicate pairs
// where at least one function is from a modified file.
// modifiedFiles should be filenames relative to dir (e.g., "new_file.go").
func (d *Detector) CheckModifiedFiles(dir string, modifiedFiles []string) ([]DupPair, error) {
	if len(modifiedFiles) == 0 {
		return nil, nil
	}

	modifiedSet := make(map[string]bool)
	for _, f := range modifiedFiles {
		modifiedSet[f] = true
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	var allFuncs []FuncInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		filePath := filepath.Join(dir, name)
		src, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		funcs, err := d.ParseSource(name, string(src))
		if err != nil {
			continue
		}
		allFuncs = append(allFuncs, funcs...)
	}

	// Find all pairs, then filter to only those involving modified files
	allPairs := d.FindDuplicates(allFuncs)
	return filterModifiedPairs(allPairs, modifiedSet), nil
}

// CheckModifiedFilesProject scans the entire project tree and returns duplicate
// pairs where at least one function is from a modified file.
// modifiedFiles should be paths relative to projectDir (e.g., "pkg2/handle.go").
func (d *Detector) CheckModifiedFilesProject(projectDir string, modifiedFiles []string) ([]DupPair, error) {
	if len(modifiedFiles) == 0 {
		return nil, nil
	}

	modifiedSet := make(map[string]bool)
	for _, f := range modifiedFiles {
		modifiedSet[f] = true
	}

	// Reuse ScanProject to get all functions with relative paths
	allPairs, err := d.ScanProject(projectDir)
	if err != nil {
		return nil, err
	}

	return filterModifiedPairs(allPairs, modifiedSet), nil
}

// filterModifiedPairs returns only pairs where at least one function is in
// a modified file.
func filterModifiedPairs(pairs []DupPair, modifiedSet map[string]bool) []DupPair {
	var filtered []DupPair
	for _, pair := range pairs {
		if modifiedSet[pair.FuncA.File] || modifiedSet[pair.FuncB.File] {
			filtered = append(filtered, pair)
		}
	}
	return filtered
}

// FormatDuplicationAdvisory formats duplicate pairs into a readable advisory
// block for completion output. Returns empty string if no pairs.
func FormatDuplicationAdvisory(pairs []DupPair) string {
	if len(pairs) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("┌─────────────────────────────────────────────────────────────┐\n")
	sb.WriteString("│  DUPLICATION ADVISORY: Agent introduced similar functions   │\n")
	sb.WriteString("├─────────────────────────────────────────────────────────────┤\n")

	for _, pair := range pairs {
		line := fmt.Sprintf("│  %.0f%% %s ↔ %s", pair.Similarity*100, pair.FuncA.Name, pair.FuncB.Name)
		for len(line) < 62 {
			line += " "
		}
		sb.WriteString(line + "│\n")

		loc := fmt.Sprintf("│      %s:%d ↔ %s:%d", pair.FuncA.File, pair.FuncA.StartLine, pair.FuncB.File, pair.FuncB.StartLine)
		for len(loc) < 62 {
			loc += " "
		}
		sb.WriteString(loc + "│\n")
	}

	sb.WriteString("├─────────────────────────────────────────────────────────────┤\n")
	sb.WriteString("│  Run `orch dupdetect` for full analysis                     │\n")
	sb.WriteString("└─────────────────────────────────────────────────────────────┘\n")

	return sb.String()
}
