package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// KBInitResult captures the result of kb initialization.
type KBInitResult struct {
	KBDir         string
	DirsCreated   []string
	DirsExisted   []string
	ReadmeCreated bool
	ReadmeExisted bool
}

var kbInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize .kb/ directory structure for a new project",
	Long: `Scaffold the .kb/ knowledge base directory with standard subdirectories.

Creates:
  .kb/
  ├── models/           Synthesized understanding (claims, probes)
  ├── investigations/   Research artifacts
  ├── decisions/        Architectural decision records
  ├── threads/          Living threads (evolving questions)
  ├── briefs/           Issue briefs (context snapshots)
  ├── guides/           Procedural knowledge (how-to references)
  ├── quick/            Quick knowledge entries (kn)
  └── README.md         Directory structure reference

This command is idempotent - running it multiple times is safe.
Existing files and directories are never overwritten.

Examples:
  orch kb init              # Initialize in current directory
  orch kb init              # Safe to re-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		result, err := kbInitProject(projectDir)
		if err != nil {
			return err
		}

		printKBInitResult(result)
		return nil
	},
}

// kbInitProject scaffolds the .kb/ directory structure.
// It is idempotent: existing directories and files are preserved.
func kbInitProject(projectDir string) (*KBInitResult, error) {
	kbDir := filepath.Join(projectDir, ".kb")

	result := &KBInitResult{
		KBDir: kbDir,
	}

	// Subdirectories to create
	subdirs := []string{
		"models",
		"investigations",
		"decisions",
		"threads",
		"briefs",
		"guides",
		"quick",
	}

	for _, sub := range subdirs {
		dir := filepath.Join(kbDir, sub)
		info, err := os.Stat(dir)
		if err == nil && info.IsDir() {
			result.DirsExisted = append(result.DirsExisted, sub)
			continue
		}

		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create .kb/%s: %w", sub, err)
		}
		result.DirsCreated = append(result.DirsCreated, sub)
	}

	// Create README.md if it doesn't exist
	readmePath := filepath.Join(kbDir, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		if err := os.WriteFile(readmePath, []byte(kbReadmeContent), 0644); err != nil {
			return nil, fmt.Errorf("failed to write README.md: %w", err)
		}
		result.ReadmeCreated = true
	} else {
		result.ReadmeExisted = true
	}

	return result, nil
}

func printKBInitResult(result *KBInitResult) {
	if len(result.DirsCreated) > 0 {
		fmt.Println("Created:")
		for _, dir := range result.DirsCreated {
			fmt.Printf("  .kb/%s/\n", dir)
		}
	}

	if len(result.DirsExisted) > 0 {
		fmt.Println("Already existed:")
		for _, dir := range result.DirsExisted {
			fmt.Printf("  .kb/%s/\n", dir)
		}
	}

	if result.ReadmeCreated {
		fmt.Println("  .kb/README.md")
	} else if result.ReadmeExisted {
		fmt.Println("README.md already exists")
	}

	if len(result.DirsCreated) == 0 && !result.ReadmeCreated {
		fmt.Println("Knowledge base already initialized (.kb/)")
	} else {
		fmt.Println("\nKnowledge base initialized (.kb/)")
	}
}

const kbReadmeContent = `# Knowledge Base (.kb/)

Project knowledge base for synthesized understanding, research, and decisions.

## Directory Structure

- **models/** — Synthesized understanding about specific topics. Each model contains claims that can be tested via probes. Models are the authoritative reference; probes are evidence that feeds them.

- **investigations/** — Research artifacts from exploring questions. Investigations produce findings that get synthesized into models or decisions.

- **decisions/** — Architectural decision records (ADRs). Document the "why" behind significant choices so future contributors understand context.

- **threads/** — Living threads that track evolving questions across sessions. Threads grow as understanding deepens and connect related investigations.

- **briefs/** — Issue briefs providing context snapshots for spawned agents. Created automatically during spawn to give agents the background they need.

- **guides/** — Procedural knowledge and how-to references. Synthesized from investigations into authoritative step-by-step documentation.

- **quick/** — Quick knowledge entries managed via ` + "`kn`" + `. Constraints, decisions, failed attempts, and other lightweight entries stored as JSONL.

## Conventions

- Investigation files: ` + "`YYYY-MM-DD-inv-<slug>.md`" + `
- Decision files: ` + "`YYYY-MM-DD-<slug>.md`" + `
- Model directories: ` + "`models/<topic-name>/model.md`" + ` with ` + "`probes/`" + ` subdirectory
- Thread files: ` + "`YYYY-MM-DD-<slug>.md`" + `
- Brief files: ` + "`<project>-<id>.md`" + `
- Guide files: ` + "`<topic-slug>.md`" + `
- Quick entries: ` + "`quick/entries.jsonl`" + `
`
