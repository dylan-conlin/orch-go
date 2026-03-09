// Package beadsutil provides shared beads ID parsing, extraction, and resolution utilities.
package beadsutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// ExtractIDFromTitle extracts a beads ID from an OpenCode session title.
// Looks for patterns like "[beads-id]" at the end of the title.
func ExtractIDFromTitle(title string) string {
	if start := strings.LastIndex(title, "["); start != -1 {
		if end := strings.LastIndex(title, "]"); end != -1 && end > start {
			return strings.TrimSpace(title[start+1 : end])
		}
	}
	return ""
}

// ExtractIDFromWindowName extracts a beads ID from a tmux window name.
// Window names follow format: "emoji workspace-name [beads-id]"
func ExtractIDFromWindowName(name string) string {
	if start := strings.LastIndex(name, "["); start != -1 {
		if end := strings.LastIndex(name, "]"); end != -1 && end > start {
			return strings.TrimSpace(name[start+1 : end])
		}
	}
	return ""
}

// ExtractProjectFromID extracts the project name from a beads ID.
// Beads IDs follow the format: project-xxxx (e.g., orch-go-3anf).
// The last segment after the final hyphen is the hash; everything before is the project.
func ExtractProjectFromID(beadsID string) string {
	if beadsID == "" {
		return ""
	}
	parts := strings.Split(beadsID, "-")
	if len(parts) < 2 {
		return beadsID
	}
	return strings.Join(parts[:len(parts)-1], "-")
}

// ResolveShortID resolves a potentially short beads ID to a full ID.
// Short IDs like "57dn" are resolved to full IDs like "orch-go-57dn".
// Tries RPC client first, then falls back to CLI.
// Returns an error if the issue doesn't exist.
func ResolveShortID(id string) (string, error) {
	// Try RPC client first for ID resolution
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath)
		if err := client.Connect(); err == nil {
			defer client.Close()

			resolvedID, err := client.ResolveID(id)
			if err == nil && resolvedID != "" {
				return resolvedID, nil
			}
		}
	}

	// Fallback: Use bd show to resolve the ID
	issue, err := beads.FallbackShow(id, "")
	if err != nil {
		hint := ""
		if parts := strings.Split(id, "-"); len(parts) >= 2 {
			possibleProject := parts[0]
			if len(parts) >= 3 {
				possibleProject = parts[0] + "-" + parts[1]
			}
			hint = fmt.Sprintf("\n\nHint: Issue '%s' may belong to a different project.\n"+
				"If the issue is in '%s', try:\n"+
				"  cd ~/Documents/personal/%s && orch complete %s\n"+
				"Or use --workdir:\n"+
				"  orch complete %s --workdir ~/Documents/personal/%s",
				id, possibleProject, possibleProject, id, id, possibleProject)
		}
		return "", fmt.Errorf("beads issue '%s' not found%s", id, hint)
	}

	return issue.ID, nil
}

// ResolveShortIDSimple resolves a short beads ID by prefixing with the current project name.
// Unlike ResolveShortID, this does not call beads RPC/CLI — it uses CWD-based inference.
// If the ID already contains a hyphen, it's returned as-is.
func ResolveShortIDSimple(id string) (string, error) {
	if strings.Contains(id, "-") {
		return id, nil
	}
	projectDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	projectName := filepath.Base(projectDir)
	return fmt.Sprintf("%s-%s", projectName, id), nil
}
