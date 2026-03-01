package orch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func LoadDesignArtifacts(designWorkspace, projectDir string) (mockupPath, promptPath, notes string) {
	if designWorkspace == "" {
		return "", "", ""
	}
	mockupPath, promptPath, notes = readDesignArtifacts(projectDir, designWorkspace)
	if mockupPath != "" {
		fmt.Printf("📐 Design handoff from workspace: %s\n", designWorkspace)
		fmt.Printf("   Mockup: %s\n", mockupPath)
		if promptPath != "" {
			fmt.Printf("   Prompt: %s\n", promptPath)
		}
	}
	return mockupPath, promptPath, notes
}

func readDesignArtifacts(projectDir, designWorkspace string) (mockupPath, promptPath, designNotes string) {
	workspacePath := filepath.Join(projectDir, ".orch", "workspace", designWorkspace)
	if _, err := os.Stat(workspacePath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: design workspace not found: %s\n", workspacePath)
		return "", "", ""
	}
	screenshotsPath := filepath.Join(workspacePath, "screenshots")
	if entries, err := os.ReadDir(screenshotsPath); err == nil {
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".png") {
				mockupPath = filepath.Join(screenshotsPath, entry.Name())
				promptName := strings.TrimSuffix(entry.Name(), ".png") + ".prompt.md"
				promptPath = filepath.Join(screenshotsPath, promptName)
				if _, err := os.Stat(promptPath); err != nil {
					promptPath = ""
				}
				break
			}
		}
	}
	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	if content, err := os.ReadFile(synthesisPath); err == nil {
		designNotes = extractDesignNotes(string(content))
	}
	return mockupPath, promptPath, designNotes
}

func extractDesignNotes(content string) string {
	var notes strings.Builder
	if tldr := extractSection(content, "## TLDR"); tldr != "" {
		notes.WriteString("**Design TLDR:**\n")
		notes.WriteString(tldr)
		notes.WriteString("\n\n")
	}
	if knowledge := extractSection(content, "## Knowledge"); knowledge != "" {
		notes.WriteString("**Design Knowledge:**\n")
		notes.WriteString(knowledge)
	}
	return notes.String()
}

func extractSection(content, sectionHeader string) string {
	lines := strings.Split(content, "\n")
	var sectionLines []string
	inSection := false
	for _, line := range lines {
		if strings.HasPrefix(line, sectionHeader) {
			inSection = true
			continue
		}
		if inSection && strings.HasPrefix(line, "##") {
			break
		}
		if inSection {
			sectionLines = append(sectionLines, line)
		}
	}
	if len(sectionLines) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.Join(sectionLines, "\n"))
}
