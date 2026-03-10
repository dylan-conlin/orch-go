package spawn

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

// extractProjectPrefix returns the project prefix from a beads ID.
// For "pw-8972" returns "pw", for "orch-go-123" returns "orch-go".
func extractProjectPrefix(beadsID string) string {
	// Find the last occurrence of -<digits> and take everything before it
	for i := len(beadsID) - 1; i >= 0; i-- {
		if beadsID[i] == '-' {
			// Check if everything after the hyphen is digits
			suffix := beadsID[i+1:]
			allDigits := true
			for _, c := range suffix {
				if c < '0' || c > '9' {
					allDigits = false
					break
				}
			}
			if allDigits && len(suffix) > 0 {
				return beadsID[:i]
			}
		}
	}
	return beadsID
}

// ValidateBeadsIDConsistency checks if the task text references a beads ID
// from the same project that differs from the tracking beads ID.
// Returns a warning message if a mismatch is detected, empty string otherwise.
//
// This catches a class of spawn bugs where the task description references
// one issue (e.g., "fix pw-8972") but the --issue flag tracks a different
// issue (e.g., pw-8975), leading to confusing SPAWN_CONTEXT where the TASK
// line says one thing but bd comment instructions reference another.
func ValidateBeadsIDConsistency(task string, beadsID string) string {
	if beadsID == "" {
		return ""
	}

	trackingPrefix := extractProjectPrefix(beadsID)

	// Find all beads-like IDs in the task text
	matches := regexBeadsIDInText.FindAllString(strings.ToLower(task), -1)
	for _, match := range matches {
		matchPrefix := extractProjectPrefix(match)

		// Only check IDs from the same project (same prefix)
		if matchPrefix != trackingPrefix {
			continue
		}

		// Same project, check if it's the same ID
		if match != strings.ToLower(beadsID) {
			return fmt.Sprintf(
				"Warning: task text references %s but tracking issue is %s (same project prefix %q). "+
					"This may cause agent confusion — TASK line will say %s but bd comment instructions will use %s.",
				match, beadsID, trackingPrefix, match, beadsID,
			)
		}
	}

	return ""
}

// GenerateServerContext creates the server context section for SPAWN_CONTEXT.md.
// Returns empty string if no servers are configured for the project.
func GenerateServerContext(projectDir string) string {
	cfg, err := config.Load(projectDir)
	if err != nil {
		return "" // No config or can't load - skip silently
	}

	if len(cfg.Servers) == 0 {
		return "" // No servers configured
	}

	// Get project name from directory
	projectName := filepath.Base(projectDir)
	sessionName := tmux.GetWorkersSessionName(projectName)

	// Check if servers are running
	running := tmux.SessionExists(sessionName)
	status := "stopped"
	if running {
		status = "running"
	}

	// Build server list
	var serverLines []string
	for service, port := range cfg.Servers {
		serverLines = append(serverLines, fmt.Sprintf("- **%s:** http://localhost:%d", service, port))
	}

	// Format the context section
	var sb strings.Builder
	sb.WriteString("## LOCAL SERVERS\n\n")
	sb.WriteString(fmt.Sprintf("**Project:** %s\n", projectName))
	sb.WriteString(fmt.Sprintf("**Status:** %s\n\n", status))
	sb.WriteString("**Ports:**\n")
	for _, line := range serverLines {
		sb.WriteString(line + "\n")
	}
	sb.WriteString("\n**Quick commands:**\n")
	sb.WriteString(fmt.Sprintf("- Start servers: `orch servers start %s`\n", projectName))
	sb.WriteString(fmt.Sprintf("- Stop servers: `orch servers stop %s`\n", projectName))
	sb.WriteString(fmt.Sprintf("- Open in browser: `orch servers open %s`\n", projectName))
	sb.WriteString("\n")

	return sb.String()
}

// RegisteredProject represents a project registered with kb.
type RegisteredProject struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// DetectAreaFromTask attempts to detect a knowledge area/cluster from the task description or beads issue.
// Returns empty string if no clear area is detected.
// Checks against known clusters in .kb/investigations/synthesized/ and model directories.
func DetectAreaFromTask(task string, beadsID string, projectDir string) string {
	// Get list of known clusters from filesystem
	kbDir := filepath.Join(projectDir, ".kb")

	// Check investigations/synthesized/ for clusters
	synthesizedDir := filepath.Join(kbDir, "investigations", "synthesized")
	var knownClusters []string
	if entries, err := os.ReadDir(synthesizedDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				knownClusters = append(knownClusters, entry.Name())
			}
		}
	}

	// Add model directories as potential clusters
	modelsDir := filepath.Join(kbDir, "models")
	if entries, err := os.ReadDir(modelsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				knownClusters = append(knownClusters, entry.Name())
			}
		}
	}

	// Always include "models" and "decisions" as default clusters
	knownClusters = append(knownClusters, "models", "decisions")

	// Check task description for cluster keywords
	taskLower := strings.ToLower(task)
	for _, cluster := range knownClusters {
		// Check if cluster name appears in task (word boundary match)
		// Use regex to match whole words only
		pattern := `\b` + regexp.QuoteMeta(strings.ToLower(cluster)) + `\b`
		if matched, _ := regexp.MatchString(pattern, taskLower); matched {
			return cluster
		}
	}

	// If beads issue is provided, check labels for area:* pattern
	if beadsID != "" {
		// Try to get beads issue and check labels
		socketPath, err := beads.FindSocketPath("")
		if err == nil {
			client := beads.NewClient(socketPath)
			if err := client.Connect(); err == nil {
				defer client.Close()
				if issue, err := client.Show(beadsID); err == nil {
					for _, label := range issue.Labels {
						if strings.HasPrefix(label, "area:") {
							area := strings.TrimPrefix(label, "area:")
							// Verify area exists as a known cluster
							for _, cluster := range knownClusters {
								if cluster == area {
									return area
								}
							}
						}
					}
				}
			}
		}
	}

	return ""
}

// GetClusterSummary fetches a summary for a specific cluster using orch tree --format summary.
// Returns empty string if cluster not found or command fails.
func GetClusterSummary(clusterName string, projectDir string) string {
	if clusterName == "" {
		return ""
	}

	// Run orch tree --cluster <name> --format summary
	cmd := exec.Command("orch", "tree", "--cluster", clusterName, "--format", "summary")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// GenerateRegisteredProjectsContext creates the registered projects section for orchestrator contexts.
// Returns empty string if kb projects list fails or returns no projects.
func GenerateRegisteredProjectsContext() string {
	projects, err := GetRegisteredProjects()
	if err != nil || len(projects) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Registered Projects\n\n")
	sb.WriteString("These projects are registered with `kb` for cross-project orchestration:\n\n")
	sb.WriteString("| Project | Path |\n")
	sb.WriteString("|---------|------|\n")
	for _, p := range projects {
		sb.WriteString(fmt.Sprintf("| %s | `%s` |\n", p.Name, p.Path))
	}
	sb.WriteString("\n**Usage:** `orch spawn --workdir <path> SKILL \"task\"`\n\n")

	return sb.String()
}

// GetRegisteredProjects fetches the list of registered projects from kb.
func GetRegisteredProjects() ([]RegisteredProject, error) {
	cmd := exec.Command("kb", "projects", "list", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("kb projects list failed: %w", err)
	}

	var projects []RegisteredProject
	if err := json.Unmarshal(output, &projects); err != nil {
		return nil, fmt.Errorf("failed to parse kb projects output: %w", err)
	}

	return projects, nil
}
