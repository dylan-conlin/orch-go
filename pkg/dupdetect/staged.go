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

	var modified, corpus []FuncInfo
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

		if modifiedSet[name] {
			modified = append(modified, funcs...)
		} else {
			corpus = append(corpus, funcs...)
		}
	}

	if len(modified) == 0 {
		return nil, nil
	}

	return d.FindDuplicatesAgainst(modified, corpus), nil
}

// CheckModifiedFilesProject scans the entire project tree and returns duplicate
// pairs where at least one function is from a modified file.
// modifiedFiles should be paths relative to projectDir (e.g., "pkg2/handle.go").
//
// Performance: uses scoped M×N comparison (modified vs corpus) instead of
// N² all-vs-all. For typical completions (5-10 modified files, ~30 functions)
// against a 4500-function project, this is ~50x faster.
func (d *Detector) CheckModifiedFilesProject(projectDir string, modifiedFiles []string) ([]DupPair, error) {
	if len(modifiedFiles) == 0 {
		return nil, nil
	}

	modifiedSet := make(map[string]bool)
	for _, f := range modifiedFiles {
		modifiedSet[f] = true
	}

	// Parse all functions in the project
	allFuncs, err := d.ScanProjectFuncs(projectDir)
	if err != nil {
		return nil, err
	}

	// Partition into modified and corpus
	var modified, corpus []FuncInfo
	for _, fn := range allFuncs {
		if modifiedSet[fn.File] {
			modified = append(modified, fn)
		} else {
			corpus = append(corpus, fn)
		}
	}

	if len(modified) == 0 {
		return nil, nil
	}

	// Scoped comparison: modified vs corpus + modified vs modified
	return d.FindDuplicatesAgainst(modified, corpus), nil
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
