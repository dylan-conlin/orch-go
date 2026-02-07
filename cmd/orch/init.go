// Package main provides CLI commands for orch-go.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dylan-conlin/orch-go/pkg/claudemd"
	"github.com/dylan-conlin/orch-go/pkg/port"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/spf13/cobra"
)

var (
	// Init command flags
	initForce          bool   // Force re-initialization even if directories exist
	initSkipBeads      bool   // Skip beads initialization
	initSkipKB         bool   // Skip kb initialization
	initSkipClaudeMD   bool   // Skip CLAUDE.md generation
	initSkipTmuxinator bool   // Skip tmuxinator config generation
	initBeadsPrefix    string // Custom prefix for beads issues
	initProjectType    string // Project type for CLAUDE.md template
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize orch scaffolding in the current directory",
	Long: `Initialize orch project scaffolding by creating necessary directories.

Creates:
  - .orch/workspace/     Agent workspaces
  - .orch/templates/     Shared templates (SYNTHESIS.md, etc.)
  - .kb/                 Knowledge base (via 'kb init')
  - .beads/              Issue tracking (via 'bd init')
  - CLAUDE.md            Project context for Claude agents
  - tmuxinator config    Workers session configuration (~/.tmuxinator/workers-{project}.yml)

This command is idempotent - it can be run multiple times safely.
Use --force to recreate directories even if they exist.

Project types for CLAUDE.md:
  - go-cli      Go CLI project (auto-detected via go.mod + cmd/)
  - svelte-app  SvelteKit app (auto-detected via svelte.config.js)
  - python-cli  Python CLI (auto-detected via pyproject.toml)
  - minimal     Minimal template (default fallback)

Examples:
  orch-go init                       # Initialize with defaults (auto-detect type)
  orch-go init --type go-cli         # Use go-cli template
  orch-go init --skip-beads          # Skip beads initialization
  orch-go init --skip-kb             # Skip kb initialization
  orch-go init --skip-claude         # Skip CLAUDE.md generation
  orch-go init --skip-tmuxinator     # Skip tmuxinator config generation
  orch-go init --beads-prefix snap   # Use custom beads prefix
  orch-go init --force               # Force re-initialization`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit()
	},
}

func init() {
	initCmd.Flags().BoolVar(&initForce, "force", false, "Force re-initialization even if directories exist")
	initCmd.Flags().BoolVar(&initSkipBeads, "skip-beads", false, "Skip beads initialization")
	initCmd.Flags().BoolVar(&initSkipKB, "skip-kb", false, "Skip kb initialization")
	initCmd.Flags().BoolVar(&initSkipClaudeMD, "skip-claude", false, "Skip CLAUDE.md generation")
	initCmd.Flags().BoolVar(&initSkipTmuxinator, "skip-tmuxinator", false, "Skip tmuxinator config generation")
	initCmd.Flags().StringVar(&initBeadsPrefix, "beads-prefix", "", "Custom prefix for beads issues (default: directory name)")
	initCmd.Flags().StringVar(&initProjectType, "type", "", "Project type for CLAUDE.md (go-cli, svelte-app, python-cli, minimal)")
}

// InitResult captures the result of initialization.
type InitResult struct {
	ProjectDir        string
	ProjectName       string
	DirsCreated       []string
	DirsExisted       []string
	BeadsInitiated    bool
	BeadsSkipped      bool
	BeadsError        error
	KBInitiated       bool
	KBSkipped         bool
	KBExisted         bool
	KBError           error
	ClaudeMDCreated   bool
	ClaudeMDSkipped   bool
	ClaudeMDExisted   bool
	ClaudeMDError     error
	ProjectType       claudemd.ProjectType
	PortWeb           int
	PortAPI           int
	TmuxinatorCreated bool
	TmuxinatorUpdated bool
	TmuxinatorSkipped bool
	TmuxinatorError   error
	TmuxinatorPath    string
}

// runInit initializes orch scaffolding in the current directory.
func runInit() error {
	projectDir, err := currentProjectDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	result, err := initProject(projectDir, initForce, initSkipBeads, initSkipKB, initSkipClaudeMD, initSkipTmuxinator, initBeadsPrefix, initProjectType)
	if err != nil {
		return err
	}

	// Print results
	printInitResult(result)
	return nil
}

// initProject performs the actual initialization work.
// This is separated from runInit to make testing easier.
func initProject(projectDir string, force, skipBeads, skipKB, skipClaudeMD, skipTmuxinator bool, beadsPrefix, projectType string) (*InitResult, error) {
	projectName := filepath.Base(projectDir)

	result := &InitResult{
		ProjectDir:  projectDir,
		ProjectName: projectName,
	}

	// Directories to create (only .orch/ - .kb/ is handled by kb init)
	dirs := []string{
		filepath.Join(projectDir, ".orch", "workspace"),
		filepath.Join(projectDir, ".orch", "templates"),
	}

	// Create directories
	for _, dir := range dirs {
		created, err := ensureDir(dir, force)
		if err != nil {
			return nil, fmt.Errorf("failed to create %s: %w", dir, err)
		}

		// Track created vs existed
		relPath, _ := filepath.Rel(projectDir, dir)
		if created {
			result.DirsCreated = append(result.DirsCreated, relPath)
		} else {
			result.DirsExisted = append(result.DirsExisted, relPath)
		}
	}

	// Copy SYNTHESIS.md template if it doesn't exist
	synthTemplateSrc := filepath.Join(projectDir, ".orch", "templates", "SYNTHESIS.md")
	if _, err := os.Stat(synthTemplateSrc); os.IsNotExist(err) {
		if err := writeSynthesisTemplate(synthTemplateSrc); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write SYNTHESIS.md template: %v\n", err)
		}
	}

	// Initialize kb unless skipped
	if skipKB {
		result.KBSkipped = true
	} else {
		kbDir := filepath.Join(projectDir, ".kb")
		if _, err := os.Stat(kbDir); os.IsNotExist(err) || force {
			if err := initKB(projectDir); err != nil {
				result.KBError = err
				fmt.Fprintf(os.Stderr, "Warning: kb initialization failed: %v\n", err)
			} else {
				result.KBInitiated = true
			}
		} else {
			result.KBExisted = true
		}
	}

	// Initialize beads unless skipped
	if skipBeads {
		result.BeadsSkipped = true
	} else {
		beadsDir := filepath.Join(projectDir, ".beads")
		if _, err := os.Stat(beadsDir); os.IsNotExist(err) || force {
			if err := initBeads(projectDir, beadsPrefix); err != nil {
				result.BeadsError = err
				fmt.Fprintf(os.Stderr, "Warning: beads initialization failed: %v\n", err)
			} else {
				result.BeadsInitiated = true
			}
		}
	}

	// Allocate ports for project (needed for CLAUDE.md and tmuxinator)
	portWeb, portAPI := allocatePorts(projectName)
	result.PortWeb = portWeb
	result.PortAPI = portAPI

	// Create .orch/config.yaml with allocated ports
	if err := createProjectConfig(projectDir, portWeb, portAPI); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create project config: %v\n", err)
	}

	// Generate CLAUDE.md unless skipped
	if skipClaudeMD {
		result.ClaudeMDSkipped = true
	} else {
		claudePath := filepath.Join(projectDir, "CLAUDE.md")
		if _, err := os.Stat(claudePath); err == nil && !force {
			result.ClaudeMDExisted = true
		} else {
			// Determine project type
			var pType claudemd.ProjectType
			if projectType != "" {
				pType = claudemd.ProjectType(projectType)
			} else {
				pType = claudemd.DetectProjectType(projectDir)
			}
			result.ProjectType = pType

			// Generate CLAUDE.md
			data := claudemd.TemplateData{
				ProjectName: projectName,
				ProjectType: pType,
				PortWeb:     portWeb,
				PortAPI:     portAPI,
			}

			_, err := claudemd.WriteToProject(projectDir, data)
			if err != nil {
				result.ClaudeMDError = err
				fmt.Fprintf(os.Stderr, "Warning: failed to write CLAUDE.md: %v\n", err)
			} else {
				result.ClaudeMDCreated = true
			}
		}
	}

	// Generate tmuxinator config unless skipped
	if skipTmuxinator {
		result.TmuxinatorSkipped = true
	} else {
		// Check if config already exists
		configPath := tmux.TmuxinatorConfigPath(projectName)
		configExists := false
		if _, err := os.Stat(configPath); err == nil {
			configExists = true
		}

		// Generate/update tmuxinator config
		path, err := tmux.EnsureTmuxinatorConfig(projectName, projectDir)
		if err != nil {
			result.TmuxinatorError = err
			fmt.Fprintf(os.Stderr, "Warning: failed to create tmuxinator config: %v\n", err)
		} else {
			result.TmuxinatorPath = path
			if configExists {
				result.TmuxinatorUpdated = true
			} else {
				result.TmuxinatorCreated = true
			}
		}
	}

	return result, nil
}

// allocatePorts allocates web and API ports for a project.
// Returns 0 for ports that couldn't be allocated (best effort).
func allocatePorts(projectName string) (portWeb, portAPI int) {
	registry, err := port.New("")
	if err != nil {
		return 0, 0
	}

	// Allocate web port (vite dev server)
	portWeb, _ = registry.Allocate(projectName, "web", port.PurposeVite)

	// Allocate API port
	portAPI, _ = registry.Allocate(projectName, "api", port.PurposeAPI)

	return portWeb, portAPI
}

// createProjectConfig creates .orch/config.yaml with server port declarations.
func createProjectConfig(projectDir string, portWeb, portAPI int) error {
	configPath := filepath.Join(projectDir, ".orch", "config.yaml")

	// Build config content
	var content string
	content += "servers:\n"
	if portWeb > 0 {
		content += fmt.Sprintf("  web: %d\n", portWeb)
	}
	if portAPI > 0 {
		content += fmt.Sprintf("  api: %d\n", portAPI)
	}

	// Write config file
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// ensureDir creates a directory if it doesn't exist.
// Returns true if the directory was created, false if it already existed.
func ensureDir(path string, force bool) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() && !force {
			return false, nil // Already exists
		}
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return false, err
	}

	return true, nil
}

// initKB runs 'kb init' to initialize the knowledge base.
func initKB(projectDir string) error {
	cmd := exec.Command("kb", "init")
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// initBeads runs 'bd init' to initialize beads tracking.
func initBeads(projectDir, prefix string) error {
	args := []string{"init", "--quiet"}
	if prefix != "" {
		args = append(args, "--prefix", prefix)
	}

	cmd := exec.Command("bd", args...)
	cmd.Dir = projectDir
	cmd.Env = os.Environ() // Inherit env (including BEADS_NO_DAEMON)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// printInitResult prints a summary of what was initialized.
func printInitResult(result *InitResult) {
	fmt.Printf("Initialized orch scaffolding in %s\n\n", result.ProjectName)

	if len(result.DirsCreated) > 0 {
		fmt.Println("Created:")
		for _, dir := range result.DirsCreated {
			fmt.Printf("  %s/\n", dir)
		}
	}

	if len(result.DirsExisted) > 0 {
		fmt.Println("Already existed:")
		for _, dir := range result.DirsExisted {
			fmt.Printf("  %s/\n", dir)
		}
	}

	// KB status
	if result.KBInitiated {
		fmt.Println("\nKnowledge base initialized (.kb/)")
	} else if result.KBSkipped {
		fmt.Println("\nKB initialization skipped (--skip-kb)")
	} else if result.KBError != nil {
		fmt.Printf("\nKB initialization failed: %v\n", result.KBError)
	} else if result.KBExisted {
		fmt.Println("\nKB already initialized (.kb/)")
	}

	// Beads status
	if result.BeadsInitiated {
		fmt.Println("\nBeads initialized (.beads/)")
	} else if result.BeadsSkipped {
		fmt.Println("\nBeads initialization skipped (--skip-beads)")
	} else if result.BeadsError != nil {
		fmt.Printf("\nBeads initialization failed: %v\n", result.BeadsError)
	} else {
		fmt.Println("\nBeads already initialized (.beads/)")
	}

	// CLAUDE.md status
	if result.ClaudeMDCreated {
		fmt.Printf("\nCLAUDE.md created (type: %s)\n", result.ProjectType)
		if result.PortWeb > 0 || result.PortAPI > 0 {
			fmt.Printf("  Ports allocated: web=%d api=%d\n", result.PortWeb, result.PortAPI)
		}
	} else if result.ClaudeMDSkipped {
		fmt.Println("\nCLAUDE.md generation skipped (--skip-claude)")
	} else if result.ClaudeMDExisted {
		fmt.Println("\nCLAUDE.md already exists")
	} else if result.ClaudeMDError != nil {
		fmt.Printf("\nCLAUDE.md generation failed: %v\n", result.ClaudeMDError)
	}

	// Tmuxinator status
	if result.TmuxinatorCreated {
		fmt.Printf("\nTmuxinator config created: %s\n", result.TmuxinatorPath)
	} else if result.TmuxinatorUpdated {
		fmt.Printf("\nTmuxinator config updated: %s\n", result.TmuxinatorPath)
	} else if result.TmuxinatorSkipped {
		fmt.Println("\nTmuxinator config generation skipped (--skip-tmuxinator)")
	} else if result.TmuxinatorError != nil {
		fmt.Printf("\nTmuxinator config generation failed: %v\n", result.TmuxinatorError)
	}

	fmt.Println("\nNext steps:")
	fmt.Println("  1. Edit CLAUDE.md with project-specific details")
	fmt.Println("  2. Create a beads issue: bd create \"task description\"")
	fmt.Println("  3. Spawn an agent: orch spawn investigation \"explore codebase\"")
}

// writeSynthesisTemplate writes the default SYNTHESIS.md template.
func writeSynthesisTemplate(path string) error {
	content := `# Synthesis

**Agent:** [workspace name]
**Date:** [YYYY-MM-DD]
**Task:** [original task from SPAWN_CONTEXT]

## Summary

[1-2 sentence summary of what was accomplished]

## Key Deliverables

- [Deliverable 1]: [path or description]
- [Deliverable 2]: [path or description]

## Changes Made

[List of significant changes, commits, or artifacts created]

## Discoveries

[Any unexpected findings, issues discovered, or recommendations for follow-up]

## Status

**Phase:** Complete
**Tests:** [Passing/N/A]
**Ready for review:** Yes
`
	return os.WriteFile(path, []byte(content), 0644)
}
