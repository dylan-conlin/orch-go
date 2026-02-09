// Package main provides the work command for daemon-driven spawns.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// runWork executes the work command for a specific beads issue.
// This is called by the daemon when processing triage:ready issues.
// It infers the skill and MCP requirements from the issue, then spawns an agent.
func runWork(serverURL, beadsID string, inline bool, workdir string) error {
	// Resolve workdir to absolute path if provided
	projectDir := ""
	if workdir != "" {
		absPath, err := filepath.Abs(workdir)
		if err != nil {
			return fmt.Errorf("failed to resolve workdir path: %w", err)
		}
		info, err := os.Stat(absPath)
		if os.IsNotExist(err) {
			return fmt.Errorf("workdir does not exist: %s", absPath)
		}
		if !info.IsDir() {
			return fmt.Errorf("workdir is not a directory: %s", absPath)
		}
		projectDir = absPath
	}

	// Get issue details from verify (for description)
	// If workdir is provided, get issue from that project's beads
	var issue *verify.Issue
	var err error
	if projectDir != "" {
		issue, err = verify.GetIssueWithDir(beadsID, projectDir)
	} else {
		issue, err = verify.GetIssue(beadsID)
	}
	if err != nil {
		return fmt.Errorf("failed to get beads issue: %w", err)
	}

	// Infer skill and MCP from issue (labels, title pattern, then type)
	// Use beads.Issue which has Labels for full skill/MCP inference
	var skillName string
	var mcpServer string
	_ = withBeadsClient(projectDir, func(beadsClient *beads.Client) error {
		beadsIssue, showErr := beadsClient.Show(beadsID)
		if showErr != nil {
			return showErr
		}
		skillName = inferSkillFromBeadsIssue(beadsIssue)
		mcpServer = inferMCPFromBeadsIssue(beadsIssue)
		return nil
	})
	// Fall back to type-only inference if beads fails
	if skillName == "" {
		skillName, err = InferSkillFromIssueType(issue.IssueType)
		if err != nil {
			return fmt.Errorf("cannot work on issue %s: %w", beadsID, err)
		}
	}

	// Use issue title and description as the task for full context
	task := issue.Title
	if issue.Description != "" {
		task = issue.Title + "\n\n" + issue.Description
	}

	// Set the spawnIssue flag so runSpawnWithSkillInternal uses the existing issue
	spawnIssue = beadsID

	// Set the spawnWorkdir flag for cross-project spawns
	if projectDir != "" {
		spawnWorkdir = projectDir
	}

	// Set the spawnMCP flag if the issue has a needs:* label (e.g., needs:playwright)
	// This allows daemon-spawned agents to automatically get browser access for UI work
	if mcpServer != "" {
		spawnMCP = mcpServer
	}

	fmt.Printf("Starting work on: %s\n", beadsID)
	fmt.Printf("  Title:  %s\n", issue.Title)
	fmt.Printf("  Type:   %s\n", issue.IssueType)
	fmt.Printf("  Skill:  %s\n", skillName)
	if mcpServer != "" {
		fmt.Printf("  MCP:    %s\n", mcpServer)
	}
	if projectDir != "" {
		fmt.Printf("  Project: %s\n", projectDir)
	}

	// Work command is daemon-driven (issue already created and triaged)
	// Pass daemonDriven=true to skip triage bypass check
	return runSpawnWithSkillInternal(serverURL, skillName, task, inline, true, false, false, true)
}
