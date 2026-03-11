package main

import (
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/dupdetect"
)

// RunDuplicationAdvisoryForCompletion checks if the agent introduced functions
// that are near-clones of existing functions. Scans the project for all Go
// functions and reports pairs above threshold where at least one function is
// in a file the agent modified.
//
// This is informational only — it does not block completion.
// Reads GitBaseline from AGENT_MANIFEST.json to scope diff to agent's actual changes.
func RunDuplicationAdvisoryForCompletion(projectDir, workspacePath string) string {
	pairs := findDuplicationInModifiedFiles(projectDir, workspacePath)
	if len(pairs) == 0 {
		return ""
	}
	return dupdetect.FormatDuplicationAdvisory(pairs)
}

// countDuplicationAdvisoryMatches returns the count of duplication matches for
// the agent's modified files. Used by review tier escalation.
func countDuplicationAdvisoryMatches(projectDir, workspacePath string) int {
	return len(findDuplicationInModifiedFiles(projectDir, workspacePath))
}

// findDuplicationInModifiedFiles returns duplicate pairs involving the agent's
// modified Go files. Shared logic for both advisory formatting and counting.
func findDuplicationInModifiedFiles(projectDir, workspacePath string) []dupdetect.DupPair {
	if projectDir == "" {
		return nil
	}

	// Reuse readHotspotBaseline — reads the same AGENT_MANIFEST.json field
	baseline := readHotspotBaseline(workspacePath)
	modifiedFiles, err := getModifiedFilesSinceBaseline(projectDir, baseline, 5)
	if err != nil || len(modifiedFiles) == 0 {
		return nil
	}

	var goFiles []string
	for _, f := range modifiedFiles {
		if strings.HasSuffix(f, ".go") && !strings.HasSuffix(f, "_test.go") {
			goFiles = append(goFiles, f)
		}
	}
	if len(goFiles) == 0 {
		return nil
	}

	d := dupdetect.NewDetector()
	d.MinBodyLines = 10
	d.Threshold = 0.85 // higher threshold for completion advisory — only flag clear clones

	pairs, err := d.CheckModifiedFilesProject(projectDir, goFiles)
	if err != nil {
		return nil
	}
	return pairs
}
