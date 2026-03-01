package main

import (
	"fmt"
	"os"

	"github.com/dylan-conlin/orch-go/pkg/daemonconfig"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
)

// ConfigDrift represents a single configuration drift between expected and actual values.
type ConfigDrift struct {
	Field    string `json:"field"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
}

// ConfigDriftReport contains the results of config drift detection.
type ConfigDriftReport struct {
	Healthy    bool          `json:"healthy"`
	PlistFound bool          `json:"plist_found"`
	Drifts     []ConfigDrift `json:"drifts"`
}

// runConfigDriftCheck compares the expected config (from config.yaml) with the actual plist.
func runConfigDriftCheck() error {
	fmt.Println("orch doctor --config")
	fmt.Println("Checking daemon plist drift against ~/.orch/config.yaml...")
	fmt.Println()

	report, err := checkPlistDrift()
	if err != nil {
		return fmt.Errorf("drift check error: %w", err)
	}

	if !report.PlistFound {
		fmt.Println("✗ Plist not found: ~/Library/LaunchAgents/com.orch.daemon.plist")
		fmt.Println()
		fmt.Println("To generate the plist from config:")
		fmt.Println("  orch config generate plist")
		return nil
	}

	if report.Healthy {
		fmt.Println("✓ No drift detected - plist matches config.yaml")
		return nil
	}

	fmt.Printf("✗ Found %d drift(s):\n", len(report.Drifts))
	fmt.Println()
	for _, drift := range report.Drifts {
		fmt.Printf("  %s:\n", drift.Field)
		fmt.Printf("    config:  %s\n", drift.Expected)
		fmt.Printf("    plist:   %s\n", drift.Actual)
		fmt.Println()
	}

	fmt.Println("To fix, regenerate the plist from config:")
	fmt.Println("  orch config generate plist")

	return nil
}

// checkPlistDrift compares expected plist values from config.yaml with actual plist file.
func checkPlistDrift() (*ConfigDriftReport, error) {
	report := &ConfigDriftReport{
		Healthy: true,
		Drifts:  make([]ConfigDrift, 0),
	}

	// Get expected values from config
	cfg, err := userconfig.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Read actual plist
	plistPath := daemonconfig.GetPlistPath()
	plistContent, err := os.ReadFile(plistPath)
	if err != nil {
		if os.IsNotExist(err) {
			report.PlistFound = false
			report.Healthy = false
			return report, nil
		}
		return nil, fmt.Errorf("failed to read plist: %w", err)
	}
	report.PlistFound = true

	// Parse plist to extract values
	actualValues, err := daemonconfig.ParsePlistValues(string(plistContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse plist: %w", err)
	}

	// Compare expected vs actual
	comparisons := []struct {
		Field    string
		Expected string
		Actual   string
	}{
		{
			Field:    "poll_interval",
			Expected: fmt.Sprintf("%d", cfg.DaemonPollInterval()),
			Actual:   actualValues["poll_interval"],
		},
		{
			Field:    "max_agents",
			Expected: fmt.Sprintf("%d", cfg.DaemonMaxAgents()),
			Actual:   actualValues["max_agents"],
		},
		{
			Field:    "label",
			Expected: cfg.DaemonLabel(),
			Actual:   actualValues["label"],
		},
		{
			Field:    "verbose",
			Expected: fmt.Sprintf("%v", cfg.DaemonVerbose()),
			Actual:   actualValues["verbose"],
		},
		{
			Field:    "reflect_issues",
			Expected: fmt.Sprintf("%v", cfg.DaemonReflectIssues()),
			Actual:   actualValues["reflect_issues"],
		},
		{
			Field:    "reflect_open",
			Expected: fmt.Sprintf("%v", cfg.DaemonReflectOpen()),
			Actual:   actualValues["reflect_open"],
		},
		{
			Field:    "working_directory",
			Expected: cfg.DaemonWorkingDirectory(),
			Actual:   actualValues["working_directory"],
		},
	}

	for _, c := range comparisons {
		if c.Expected != c.Actual {
			report.Drifts = append(report.Drifts, ConfigDrift{
				Field:    c.Field,
				Expected: c.Expected,
				Actual:   c.Actual,
			})
			report.Healthy = false
		}
	}

	return report, nil
}

// DocDebtReport contains the results of doc debt detection.
type DocDebtReport struct {
	Healthy             bool                      `json:"healthy"`
	TotalCommands       int                       `json:"total_commands"`
	UndocumentedCount   int                       `json:"undocumented_count"`
	DocumentedCount     int                       `json:"documented_count"`
	UndocumentedEntries []userconfig.DocDebtEntry `json:"undocumented_entries"`
}

// runDocDebtCheck surfaces undocumented CLI commands from the doc debt tracker.
func runDocDebtCheck() error {
	fmt.Println("orch doctor --docs")
	fmt.Println("Checking for undocumented CLI commands...")
	fmt.Println()

	debt, err := userconfig.LoadDocDebt()
	if err != nil {
		return fmt.Errorf("failed to load doc debt: %w", err)
	}

	report := &DocDebtReport{
		TotalCommands:       len(debt.Commands),
		UndocumentedEntries: debt.UndocumentedCommands(),
	}
	report.UndocumentedCount = len(report.UndocumentedEntries)
	report.DocumentedCount = report.TotalCommands - report.UndocumentedCount
	report.Healthy = report.UndocumentedCount == 0

	if report.TotalCommands == 0 {
		fmt.Println("No CLI commands tracked yet.")
		fmt.Println("Doc debt tracking starts automatically when new commands are detected during 'orch complete'.")
		return nil
	}

	// Print summary
	fmt.Printf("Total tracked commands: %d\n", report.TotalCommands)
	fmt.Printf("Documented: %d\n", report.DocumentedCount)
	fmt.Printf("Undocumented: %d\n", report.UndocumentedCount)
	fmt.Println()

	if report.Healthy {
		fmt.Println("✓ All tracked CLI commands are documented")
		return nil
	}

	// Print undocumented commands
	fmt.Println("✗ Undocumented commands:")
	fmt.Println()
	for _, entry := range report.UndocumentedEntries {
		fmt.Printf("  • %s (added %s)\n", entry.CommandFile, entry.DateAdded)
		if doctorVerbose && len(entry.DocLocations) > 0 {
			for _, loc := range entry.DocLocations {
				fmt.Printf("      → %s\n", loc)
			}
		}
	}

	fmt.Println()
	fmt.Println("Documentation locations to update:")
	fmt.Println("  - ~/.claude/skills/meta/orchestrator/SKILL.md")
	fmt.Println("  - docs/orch-commands-reference.md")
	fmt.Println()
	fmt.Println("After documenting, mark as complete:")
	fmt.Println("  orch docs mark <command-file>")

	return nil
}
