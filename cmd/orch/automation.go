package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/dylan-conlin/orch-go/pkg/launchd"
	"github.com/spf13/cobra"
)

var automationCmd = &cobra.Command{
	Use:   "automation",
	Short: "Audit and manage launchd automation jobs",
	Long: `Live audit of custom launchd agents.

Commands:
  list     Show all custom launchd agents with status
  check    Health check flagging failures and issues

Scans ~/Library/LaunchAgents/ for agents matching:
  com.dylan.*, com.user.*, com.orch.*, com.cdd.*

Examples:
  orch automation list              # List all custom agents
  orch automation list --json       # JSON output
  orch automation check             # Health check with issue flagging`,
}

var automationListJSON bool

var automationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all custom launchd agents with status",
	Long: `Show all custom launchd agents with their status.

Displays:
  - Agent label (name)
  - Loaded/running status
  - Last exit code
  - Schedule (cron-like, interval, or on-load)

Examples:
  orch automation list
  orch automation list --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAutomationList(automationListJSON)
	},
}

var automationCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Health check for launchd agents",
	Long: `Run health checks on custom launchd agents.

Flags issues:
  - Failures: agents with non-zero exit codes
  - Not loaded: agents with plist files but not loaded into launchd
  - Never run: agents that should run at load but haven't

Returns exit code 1 if any issues found (useful for scripting).

Examples:
  orch automation check
  orch automation check --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAutomationCheck(automationListJSON)
	},
}

func init() {
	rootCmd.AddCommand(automationCmd)
	automationCmd.AddCommand(automationListCmd)
	automationCmd.AddCommand(automationCheckCmd)

	automationListCmd.Flags().BoolVar(&automationListJSON, "json", false, "Output in JSON format")
	automationCheckCmd.Flags().BoolVar(&automationListJSON, "json", false, "Output in JSON format")
}

func runAutomationList(jsonOutput bool) error {
	agents, err := launchd.Scan(launchd.DefaultScanOptions())
	if err != nil {
		return fmt.Errorf("scan agents: %w", err)
	}

	if len(agents) == 0 {
		if jsonOutput {
			fmt.Println("[]")
		} else {
			fmt.Println("No custom launchd agents found")
		}
		return nil
	}

	if jsonOutput {
		return outputAutomationJSON(agents)
	}

	return outputTable(agents)
}

func outputTable(agents []launchd.Agent) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tEXIT\tSCHEDULE")
	fmt.Fprintln(w, "----\t------\t----\t--------")

	for _, agent := range agents {
		exitStr := "-"
		if agent.Loaded {
			if agent.LastExitCode == 0 {
				exitStr = "0"
			} else {
				exitStr = fmt.Sprintf("%d", agent.LastExitCode)
			}
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			agent.Label,
			agent.Status(),
			exitStr,
			agent.Schedule,
		)
	}

	w.Flush()
	fmt.Printf("\nTotal: %d agents\n", len(agents))
	return nil
}

type jsonAgent struct {
	Label        string `json:"label"`
	PlistPath    string `json:"plist_path"`
	Loaded       bool   `json:"loaded"`
	Running      bool   `json:"running"`
	PID          int    `json:"pid,omitempty"`
	LastExitCode int    `json:"last_exit_code"`
	Schedule     string `json:"schedule"`
	RunAtLoad    bool   `json:"run_at_load"`
	KeepAlive    bool   `json:"keep_alive"`
	Status       string `json:"status"`
}

func outputAutomationJSON(agents []launchd.Agent) error {
	jsonAgents := make([]jsonAgent, len(agents))
	for i, a := range agents {
		jsonAgents[i] = jsonAgent{
			Label:        a.Label,
			PlistPath:    a.PlistPath,
			Loaded:       a.Loaded,
			Running:      a.Running,
			PID:          a.PID,
			LastExitCode: a.LastExitCode,
			Schedule:     a.Schedule,
			RunAtLoad:    a.RunAtLoad,
			KeepAlive:    a.KeepAlive,
			Status:       a.Status(),
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(jsonAgents)
}

// Issue represents a detected problem with a launchd agent.
type Issue struct {
	Agent   string `json:"agent"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

func runAutomationCheck(jsonOutput bool) error {
	agents, err := launchd.Scan(launchd.DefaultScanOptions())
	if err != nil {
		return fmt.Errorf("scan agents: %w", err)
	}

	var issues []Issue

	for _, agent := range agents {
		// Check: Not loaded
		if !agent.Loaded {
			issues = append(issues, Issue{
				Agent:   agent.Label,
				Type:    "not_loaded",
				Message: "Agent plist exists but not loaded into launchd",
			})
			continue
		}

		// Check: Non-zero exit code (failure)
		if agent.HasFailure() {
			issues = append(issues, Issue{
				Agent:   agent.Label,
				Type:    "failure",
				Message: fmt.Sprintf("Last exit code: %d", agent.LastExitCode),
			})
		}
	}

	if jsonOutput {
		return outputCheckJSON(agents, issues)
	}

	return outputCheckText(agents, issues)
}

func outputCheckText(agents []launchd.Agent, issues []Issue) error {
	fmt.Printf("Launchd Agent Health Check\n")
	fmt.Printf("%s\n", strings.Repeat("-", 40))
	fmt.Printf("Total agents:    %d\n", len(agents))

	// Count by status
	loaded := 0
	running := 0
	failed := 0
	for _, a := range agents {
		if a.Loaded {
			loaded++
		}
		if a.Running {
			running++
		}
		if a.HasFailure() {
			failed++
		}
	}

	fmt.Printf("Loaded:          %d\n", loaded)
	fmt.Printf("Running:         %d\n", running)
	fmt.Printf("Failed (exit!=0):%d\n", failed)
	fmt.Println()

	if len(issues) == 0 {
		fmt.Println("No issues found")
		return nil
	}

	fmt.Printf("Issues Found: %d\n", len(issues))
	fmt.Printf("%s\n", strings.Repeat("-", 40))

	for _, issue := range issues {
		icon := getIssueIcon(issue.Type)
		fmt.Printf("%s %s: %s\n", icon, issue.Agent, issue.Message)
	}

	// Return exit code 1 if issues found (for scripting)
	os.Exit(1)
	return nil
}

func getIssueIcon(issueType string) string {
	switch issueType {
	case "failure":
		return "[FAIL]"
	case "not_loaded":
		return "[WARN]"
	default:
		return "[INFO]"
	}
}

type checkResult struct {
	Summary struct {
		Total   int `json:"total"`
		Loaded  int `json:"loaded"`
		Running int `json:"running"`
		Failed  int `json:"failed"`
	} `json:"summary"`
	Issues []Issue `json:"issues"`
	Agents []jsonAgent `json:"agents"`
}

func outputCheckJSON(agents []launchd.Agent, issues []Issue) error {
	result := checkResult{
		Issues: issues,
	}

	result.Summary.Total = len(agents)
	for _, a := range agents {
		if a.Loaded {
			result.Summary.Loaded++
		}
		if a.Running {
			result.Summary.Running++
		}
		if a.HasFailure() {
			result.Summary.Failed++
		}
	}

	result.Agents = make([]jsonAgent, len(agents))
	for i, a := range agents {
		result.Agents[i] = jsonAgent{
			Label:        a.Label,
			PlistPath:    a.PlistPath,
			Loaded:       a.Loaded,
			Running:      a.Running,
			PID:          a.PID,
			LastExitCode: a.LastExitCode,
			Schedule:     a.Schedule,
			RunAtLoad:    a.RunAtLoad,
			KeepAlive:    a.KeepAlive,
			Status:       a.Status(),
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		return err
	}

	if len(issues) > 0 {
		os.Exit(1)
	}
	return nil
}
