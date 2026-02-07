// Package main provides the claim command for associating untracked sessions with beads issues.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/spf13/cobra"
)

var claimCmd = &cobra.Command{
	Use:    "claim <session-id> <beads-id>",
	Short:  "Claim an untracked OpenCode session and associate it with a beads issue",
	Hidden: true,
	Long: `Claim an untracked OpenCode session and associate it with a beads issue.

This creates a workspace for the session, enabling full orch tooling support
(tail, complete, cleanup) using the beads ID instead of session ID.

After claiming, you can use:
  orch tail <beads-id>       # Instead of orch tail --session ses_xxx
  orch complete <beads-id>   # Instead of manual completion
  orch status                # Shows session as tracked

Examples:
  orch claim ses_3f30689ebffeBb6HL1yuXUw0l9 orch-go-21029
  orch claim ses_xxx proj-456`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]
		beadsID := args[1]
		client := opencode.NewClient(serverURL)
		return runClaim(client, sessionID, beadsID)
	},
}

func init() {
	// No additional flags needed for now
}

func runClaim(client opencode.ClientInterface, sessionID, beadsID string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Step 1: Validate session exists in OpenCode
	session, err := client.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("session %s not found in OpenCode (ensure OpenCode server is running and session exists)", truncateSessionID(sessionID))
	}

	// Step 2: Validate beads ID exists and resolve short ID to full ID
	fullBeadsID, err := resolveShortBeadsID(beadsID)
	if err != nil {
		return fmt.Errorf("beads issue validation failed: %w", err)
	}

	// Step 3: Check if session is already claimed
	workspacePath := findWorkspaceBySessionID(projectDir, sessionID)
	if workspacePath != "" {
		agentName := filepath.Base(workspacePath)
		return fmt.Errorf("session %s is already claimed (workspace: %s)", truncateSessionID(sessionID), agentName)
	}

	// Step 4: Check if beads issue is already claimed
	workspacePath, agentName := findWorkspaceByBeadsID(projectDir, fullBeadsID)
	if workspacePath != "" {
		return fmt.Errorf("beads issue %s is already claimed (workspace: %s)", fullBeadsID, agentName)
	}

	// Step 5: Create workspace directory
	workspaceName, err := generateClaimWorkspaceName(projectDir, session.Title, fullBeadsID)
	if err != nil {
		return fmt.Errorf("failed to generate workspace name: %w", err)
	}

	workspaceDir := filepath.Join(projectDir, ".orch", "workspace", workspaceName)
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// Step 6: Write workspace files
	if err := writeClaimWorkspaceFiles(workspaceDir, workspaceName, sessionID, fullBeadsID, projectDir, session.Title); err != nil {
		// Clean up on error
		os.RemoveAll(workspaceDir)
		return fmt.Errorf("failed to write workspace files: %w", err)
	}

	// Step 7: Update session title to include beads ID for visibility
	newTitle := formatSessionTitleWithBeadsID(session.Title, fullBeadsID)
	if err := client.UpdateSessionTitle(sessionID, newTitle); err != nil {
		// Don't fail the entire operation if title update fails - the workspace is the source of truth
		fmt.Fprintf(os.Stderr, "Warning: failed to update session title: %v\n", err)
		fmt.Fprintf(os.Stderr, "  Session is still tracked via workspace files\n")
	}

	// Success
	fmt.Printf("✓ Claimed session %s for beads issue %s\n", truncateSessionID(sessionID), fullBeadsID)
	fmt.Printf("  Workspace: %s\n", workspaceName)
	fmt.Printf("  Session title: %s\n", newTitle)
	fmt.Printf("\nYou can now use:\n")
	fmt.Printf("  orch tail %s\n", fullBeadsID)
	fmt.Printf("  orch complete %s\n", fullBeadsID)

	return nil
}

// formatSessionTitleWithBeadsID formats a session title to include the beads ID.
// Format: "original title [beads-id]" (e.g., "Add feature [orch-go-21029]")
// This makes the session visible as tracked in orch status.
func formatSessionTitleWithBeadsID(originalTitle, beadsID string) string {
	// If title already has a beads ID in brackets, replace it
	if start := strings.LastIndex(originalTitle, "["); start != -1 {
		if end := strings.LastIndex(originalTitle, "]"); end != -1 && end > start {
			// Remove existing beads ID
			originalTitle = strings.TrimSpace(originalTitle[:start])
		}
	}
	return fmt.Sprintf("%s [%s]", originalTitle, beadsID)
}

// generateClaimWorkspaceName generates a workspace name for a claimed session.
// Format: <project>-claimed-<description>-<date>-<hash>
// Example: og-claimed-add-feature-29jan-a1b2
func generateClaimWorkspaceName(projectDir, sessionTitle, beadsID string) (string, error) {
	// Extract project prefix from beads ID (e.g., "orch-go" from "orch-go-21029")
	project := extractProjectFromBeadsID(beadsID)

	// Sanitize session title for workspace name (first 20 chars, alphanumeric + hyphens)
	description := sanitizeForWorkspaceName(sessionTitle, 20)
	if description == "" {
		description = "session"
	}

	// Generate date suffix (format: DDmon, e.g., "29jan")
	now := time.Now()
	dateSuffix := fmt.Sprintf("%02d%s", now.Day(), strings.ToLower(now.Format("Jan")))

	// Generate short hash (4 chars from beads ID)
	hash := extractHashFromBeadsID(beadsID)

	return fmt.Sprintf("%s-claimed-%s-%s-%s", project, description, dateSuffix, hash), nil
}

// sanitizeForWorkspaceName converts a string to a workspace-name-safe format.
// Keeps alphanumeric characters and hyphens, converts spaces to hyphens, lowercases.
func sanitizeForWorkspaceName(s string, maxLen int) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace spaces and underscores with hyphens
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")

	// Remove non-alphanumeric characters (except hyphens)
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	// Collapse multiple hyphens to single hyphen
	sanitized := result.String()
	for strings.Contains(sanitized, "--") {
		sanitized = strings.ReplaceAll(sanitized, "--", "-")
	}

	// Trim hyphens from start and end
	sanitized = strings.Trim(sanitized, "-")

	// Truncate to maxLen
	if len(sanitized) > maxLen {
		sanitized = sanitized[:maxLen]
		sanitized = strings.TrimRight(sanitized, "-")
	}

	return sanitized
}

// extractHashFromBeadsID extracts the hash suffix from a beads ID.
// Example: "orch-go-21029" -> "21029" (last segment)
func extractHashFromBeadsID(beadsID string) string {
	parts := strings.Split(beadsID, "-")
	if len(parts) < 2 {
		return "unkn"
	}
	return parts[len(parts)-1]
}

// writeClaimWorkspaceFiles writes the workspace files for a claimed session.
func writeClaimWorkspaceFiles(workspaceDir, workspaceName, sessionID, beadsID, projectDir, sessionTitle string) error {
	// Write .session_id
	if err := spawn.WriteSessionID(workspaceDir, sessionID); err != nil {
		return fmt.Errorf("failed to write .session_id: %w", err)
	}

	// Write .beads_id
	beadsIDPath := filepath.Join(workspaceDir, ".beads_id")
	if err := os.WriteFile(beadsIDPath, []byte(beadsID+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write .beads_id: %w", err)
	}

	// Write .tier (default to "light" for claimed sessions - no synthesis required)
	if err := spawn.WriteTier(workspaceDir, spawn.TierLight); err != nil {
		return fmt.Errorf("failed to write .tier: %w", err)
	}

	// Write .spawn_time
	if err := spawn.WriteSpawnTime(workspaceDir, time.Now()); err != nil {
		return fmt.Errorf("failed to write .spawn_time: %w", err)
	}

	// Get git baseline (current commit SHA)
	gitBaseline := getGitBaseline(projectDir)

	// Write AGENT_MANIFEST.json
	manifest := spawn.AgentManifest{
		WorkspaceName: workspaceName,
		Skill:         "claimed",
		BeadsID:       beadsID,
		ProjectDir:    projectDir,
		GitBaseline:   gitBaseline,
		SpawnTime:     time.Now().Format(time.RFC3339),
		Tier:          spawn.TierLight,
		SpawnMode:     "claimed",
	}

	if err := spawn.WriteAgentManifest(workspaceDir, manifest); err != nil {
		return fmt.Errorf("failed to write AGENT_MANIFEST.json: %w", err)
	}

	// Write SPAWN_CONTEXT.md (minimal context for claimed sessions)
	spawnContextPath := filepath.Join(workspaceDir, "SPAWN_CONTEXT.md")
	spawnContext := fmt.Sprintf(`# Claimed Session

This workspace was created by claiming an existing OpenCode session.

**Beads Issue:** %s
**Session ID:** %s
**Session Title:** %s
**Claimed At:** %s

## Original Session Context

This session was started outside of orch spawn. Use 'orch tail %s' to view session history.

## Next Steps

Continue working in this session, or use 'orch complete %s' when the work is done.
`, beadsID, sessionID, sessionTitle, time.Now().Format(time.RFC3339), beadsID, beadsID)

	if err := os.WriteFile(spawnContextPath, []byte(spawnContext), 0644); err != nil {
		return fmt.Errorf("failed to write SPAWN_CONTEXT.md: %w", err)
	}

	return nil
}

// getGitBaseline returns the current git commit SHA, or empty string if not in a git repo.
func getGitBaseline(projectDir string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// Note: findWorkspaceBySessionID is defined in serve_agents.go and reused here
