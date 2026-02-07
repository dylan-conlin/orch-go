// Package main provides the CLI entry point for orch-go.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:   "attach <workspace>",
	Short: "Attach to an existing OpenCode session via workspace name",
	Long: `Attach to an existing OpenCode session by looking up the workspace.

Reads the session ID from the workspace's .session_id file and opens the
OpenCode TUI attached to that session.

Examples:
  orch attach feat-auth-impl-06jan
  orch attach og-inv-test-workspace`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspaceName := args[0]
		return runAttach(workspaceName)
	},
}

func init() {
	rootCmd.AddCommand(attachCmd)
}

// runAttach attaches to an OpenCode session via workspace name.
func runAttach(workspaceName string) error {
	// Get current directory to determine project
	projectDir, err := currentProjectDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Try exact match first, then fall back to partial match
	workspacePath := filepath.Join(projectDir, ".orch", "workspace", workspaceName)

	// Check if workspace exists with exact name
	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		// Try partial match
		fullName, matchErr := FindWorkspaceByPartialName(projectDir, workspaceName)
		if matchErr != nil {
			return fmt.Errorf("workspace not found: %s\n  %v", workspaceName, matchErr)
		}
		workspaceName = fullName
		workspacePath = filepath.Join(projectDir, ".orch", "workspace", workspaceName)
	}

	// Read session ID from workspace
	sessionID := spawn.ReadSessionID(workspacePath)
	if sessionID == "" {
		return fmt.Errorf("no session ID found in workspace %s\n  File: %s",
			workspaceName, filepath.Join(workspacePath, ".session_id"))
	}

	// Build and execute the opencode attach command
	// Uses: opencode attach <server> --session <id>
	opencodeBin := "opencode"
	if bin := os.Getenv("OPENCODE_BIN"); bin != "" {
		opencodeBin = bin
	}

	args := []string{
		"attach",
		serverURL,
		"--session", sessionID,
	}

	cmd := exec.Command(opencodeBin, args...)
	cmd.Dir = projectDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Print info before exec (won't show if exec succeeds)
	fmt.Printf("Attaching to session:\n")
	fmt.Printf("  Workspace:  %s\n", workspaceName)
	fmt.Printf("  Session ID: %s\n", sessionID)
	fmt.Printf("  Server:     %s\n", serverURL)

	if err := cmd.Run(); err != nil {
		// Check if this is the known "Session not found" issue
		// (orch-go-1qgwg: opencode attach --session returns "Session not found" even when session exists)
		return fmt.Errorf("failed to attach to session: %w\n\n"+
			"Note: If you see 'Session not found', this may be a known OpenCode issue.\n"+
			"The session may still exist - try 'orch resume %s' to send a message instead.", err, workspaceName)
	}

	return nil
}

// FindWorkspaceByPartialName finds a workspace by partial name match.
// Returns the full workspace name if exactly one match is found.
// Returns error if zero or multiple matches found.
func FindWorkspaceByPartialName(projectDir, partialName string) (string, error) {
	workspaceBase := filepath.Join(projectDir, ".orch", "workspace")

	entries, err := os.ReadDir(workspaceBase)
	if err != nil {
		return "", fmt.Errorf("failed to read workspace directory: %w", err)
	}

	var matches []string
	for _, entry := range entries {
		if entry.IsDir() && containsPartialMatch(entry.Name(), partialName) {
			matches = append(matches, entry.Name())
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no workspace found matching: %s", partialName)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("multiple workspaces match '%s': %v", partialName, matches)
	}
}

// containsPartialMatch checks if name contains the partial match.
func containsPartialMatch(name, partial string) bool {
	if len(partial) == 0 {
		return false
	}
	return name == partial || filepath.Base(name) == partial || strings.Contains(name, partial)
}
