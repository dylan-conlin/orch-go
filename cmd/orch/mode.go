package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// DevModeInfo represents the .dev-mode file contents
type DevModeInfo struct {
	Reason  string    `json:"reason"`
	Started time.Time `json:"started"`
	By      string    `json:"by,omitempty"`
}

// ModeHistoryEntry represents an entry in mode-history.jsonl
type ModeHistoryEntry struct {
	Time   time.Time `json:"time"`
	Action string    `json:"action"` // "dev", "ops", "bypass"
	Reason string    `json:"reason,omitempty"`
}

var modeCmd = &cobra.Command{
	Use:   "mode [dev|ops] [reason]",
	Short: "Switch between dev and ops mode",
	Long: `Switch between development and operations mode.

Dev mode:  Agents can modify infrastructure (dashboard, status, spawn).
           Use when actively building/fixing the system itself.

Ops mode:  Infrastructure is protected from changes.
           Use when running agents to do actual work.

The system defaults to ops mode (safe). Pre-commit hooks block
infrastructure changes unless dev mode is active.

Infrastructure paths protected:
  - cmd/orch/serve.go, main.go, status.go
  - pkg/state/, pkg/opencode/
  - web/src/lib/stores/agents.ts, daemon.ts
  - web/src/lib/components/agent-card/

Examples:
  orch mode              # Show current mode
  orch mode dev "fixing dashboard agent count"
  orch mode ops          # Return to operations mode
`,
	Args: cobra.MaximumNArgs(2),
	RunE: runMode,
}

func init() {
	rootCmd.AddCommand(modeCmd)
}

func runMode(cmd *cobra.Command, args []string) error {
	devModeFile := ".dev-mode"
	historyFile := ".orch/mode-history.jsonl"

	// No args - show current mode
	if len(args) == 0 {
		return showCurrentMode(devModeFile)
	}

	mode := args[0]

	switch mode {
	case "dev":
		if len(args) < 2 {
			return fmt.Errorf("dev mode requires a reason: orch mode dev \"reason for changes\"")
		}
		reason := args[1]
		return enableDevMode(devModeFile, historyFile, reason)

	case "ops":
		return enableOpsMode(devModeFile, historyFile)

	default:
		return fmt.Errorf("unknown mode: %s (use 'dev' or 'ops')", mode)
	}
}

func showCurrentMode(devModeFile string) error {
	info, err := readDevModeFile(devModeFile)
	if err != nil {
		// No dev mode file = ops mode
		fmt.Println("Current mode: ops")
		fmt.Println("")
		fmt.Println("Infrastructure is protected. Commits touching agent infrastructure")
		fmt.Println("will be blocked by pre-commit hooks.")
		fmt.Println("")
		fmt.Println("To enable infrastructure changes:")
		fmt.Println("  orch mode dev \"reason for changes\"")
		return nil
	}

	duration := time.Since(info.Started).Round(time.Minute)
	fmt.Println("Current mode: dev")
	fmt.Printf("  Reason:  %s\n", info.Reason)
	fmt.Printf("  Started: %s ago\n", duration)
	fmt.Println("")
	fmt.Println("⚠️  Infrastructure is UNPROTECTED. Commits can modify agent infrastructure.")
	fmt.Println("")
	fmt.Println("When done, return to ops mode:")
	fmt.Println("  orch mode ops")
	return nil
}

func enableDevMode(devModeFile, historyFile, reason string) error {
	// Check if already in dev mode
	if _, err := os.Stat(devModeFile); err == nil {
		existing, _ := readDevModeFile(devModeFile)
		return fmt.Errorf("already in dev mode since %s ago (%s)\nUse 'orch mode ops' first to switch modes",
			time.Since(existing.Started).Round(time.Minute), existing.Reason)
	}

	info := DevModeInfo{
		Reason:  reason,
		Started: time.Now(),
		By:      os.Getenv("USER"),
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dev mode info: %w", err)
	}

	if err := os.WriteFile(devModeFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write dev mode file: %w", err)
	}

	// Log to history
	logModeChange(historyFile, "dev", reason)

	fmt.Printf("✓ Switched to dev mode: %s\n", reason)
	fmt.Println("")
	fmt.Println("Infrastructure is now UNPROTECTED. You can modify:")
	fmt.Println("  - cmd/orch/serve.go, main.go, status.go")
	fmt.Println("  - pkg/state/, pkg/opencode/")
	fmt.Println("  - web/src/lib/stores/agents.ts, daemon.ts")
	fmt.Println("  - web/src/lib/components/agent-card/")
	fmt.Println("")
	fmt.Println("When done, return to ops mode:")
	fmt.Println("  orch mode ops")
	return nil
}

func enableOpsMode(devModeFile, historyFile string) error {
	// Check if in dev mode
	info, err := readDevModeFile(devModeFile)
	if err != nil {
		fmt.Println("Already in ops mode.")
		return nil
	}

	if err := os.Remove(devModeFile); err != nil {
		return fmt.Errorf("failed to remove dev mode file: %w", err)
	}

	duration := time.Since(info.Started).Round(time.Minute)

	// Log to history
	logModeChange(historyFile, "ops", fmt.Sprintf("was in dev mode for %s: %s", duration, info.Reason))

	fmt.Printf("✓ Switched to ops mode (was in dev mode for %s)\n", duration)
	fmt.Println("")
	fmt.Println("Infrastructure is now PROTECTED. Pre-commit hooks will block")
	fmt.Println("changes to agent infrastructure.")
	return nil
}

func readDevModeFile(path string) (*DevModeInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var info DevModeInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

func logModeChange(historyFile, action, reason string) {
	// Ensure .orch directory exists
	dir := filepath.Dir(historyFile)
	os.MkdirAll(dir, 0755)

	entry := ModeHistoryEntry{
		Time:   time.Now(),
		Action: action,
		Reason: reason,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return // Silently fail - logging is best effort
	}

	f, err := os.OpenFile(historyFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	f.WriteString(string(data) + "\n")
}
