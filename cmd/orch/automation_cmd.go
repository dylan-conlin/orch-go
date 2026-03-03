package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// orchLaunchdJobs lists the launchd jobs managed by orch.
// Add new entries here when creating new launchd plists.
var orchLaunchdJobs = []struct {
	Label       string
	Description string
}{
	{"com.orch.token-keepalive", "Daily OAuth token refresh (keeps both accounts alive)"},
	{"com.orch.audit-select", "Weekly randomized completion audit selection"},
}

var automationCmd = &cobra.Command{
	Use:   "automation",
	Short: "Manage orch automation jobs",
	Long:  "View and manage launchd jobs used by orch for background automation.",
}

var automationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List orch launchd jobs and their status",
	RunE:  runAutomationList,
}

func init() {
	automationCmd.AddCommand(automationListCmd)
}

func runAutomationList(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	launchAgentsDir := filepath.Join(home, "Library", "LaunchAgents")

	fmt.Println("Orch Automation Jobs")
	fmt.Println(strings.Repeat("-", 60))

	for _, job := range orchLaunchdJobs {
		plistPath := filepath.Join(launchAgentsDir, job.Label+".plist")

		// Check if plist file exists
		_, statErr := os.Stat(plistPath)
		installed := statErr == nil

		// Check if loaded via launchctl
		loaded := false
		if installed {
			out, listErr := exec.Command("launchctl", "list", job.Label).CombinedOutput()
			loaded = listErr == nil && len(out) > 0
		}

		// Status indicator
		status := "not installed"
		if installed && loaded {
			status = "active"
		} else if installed {
			status = "installed (not loaded)"
		}

		fmt.Printf("\n  %s\n", job.Label)
		fmt.Printf("    %s\n", job.Description)
		fmt.Printf("    Status: %s\n", status)
		fmt.Printf("    Plist:  ~/%s\n", strings.TrimPrefix(plistPath, home+"/"))
	}

	fmt.Println()
	return nil
}
