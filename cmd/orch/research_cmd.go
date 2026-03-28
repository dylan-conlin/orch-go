package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/identity"
	"github.com/dylan-conlin/orch-go/pkg/research"
	"github.com/spf13/cobra"
)

var researchWorkdir string

var researchCmd = &cobra.Command{
	Use:   "research [model] [claim-id]",
	Short: "View claim status and spawn research probes",
	Long: `Show research status for model claims and spawn probe agents.

Three modes:
  orch research                  Show all models with claim counts and test status
  orch research <model>          Show all claims for a model with their test status
  orch research <model> <id>     Create a triage:ready issue to probe a specific claim

Claims are read from claims.yaml (structured) and model.md (markdown tables).
Probe results are read from .kb/models/<model>/probes/ directories.

Examples:
  orch research                                # Summary of all models
  orch research named-incompleteness           # Claims for NI model
  orch research named-incomp                   # Prefix match works
  orch research named-incompleteness NI-01     # Spawn probe for NI-01`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir, _, err := identity.ResolveProjectDirectory(researchWorkdir)
		if err != nil {
			return fmt.Errorf("resolving project directory: %w", err)
		}
		kbDir := filepath.Join(projectDir, ".kb")

		switch len(args) {
		case 0:
			return runResearchStatus(kbDir)
		case 1:
			return runResearchModel(kbDir, args[0])
		case 2:
			return runResearchSpawn(kbDir, args[0], args[1])
		default:
			return fmt.Errorf("expected 0-2 arguments, got %d", len(args))
		}
	},
}

func init() {
	researchCmd.Flags().StringVar(&researchWorkdir, "workdir", "", "Project directory (default: current)")
}

// runResearchStatus shows a summary of all models with claim counts.
func runResearchStatus(kbDir string) error {
	models, err := research.LoadAllModels(kbDir)
	if err != nil {
		return fmt.Errorf("loading models: %w", err)
	}

	if len(models) == 0 {
		fmt.Println("No models with claims found in .kb/models/")
		return nil
	}

	fmt.Println("Research Status")
	fmt.Println(strings.Repeat("-", 70))

	totalClaims := 0
	totalTested := 0

	for _, ms := range models {
		totalClaims += ms.TotalClaims
		totalTested += ms.TestedClaims

		testedPct := 0
		if ms.TotalClaims > 0 {
			testedPct = ms.TestedClaims * 100 / ms.TotalClaims
		}

		// Count by status
		statusCounts := make(map[research.TestStatus]int)
		for _, cs := range ms.Claims {
			statusCounts[cs.TestStatus]++
		}

		statusParts := formatStatusCounts(statusCounts)

		fmt.Printf("  %-40s %d/%d tested (%d%%)  %s\n",
			ms.ModelName, ms.TestedClaims, ms.TotalClaims, testedPct, statusParts)
	}

	fmt.Println(strings.Repeat("-", 70))
	overallPct := 0
	if totalClaims > 0 {
		overallPct = totalTested * 100 / totalClaims
	}
	fmt.Printf("  %-40s %d/%d tested (%d%%)\n", "TOTAL", totalTested, totalClaims, overallPct)
	fmt.Println()
	fmt.Println("Run: orch research <model> for claim details")

	return nil
}

// runResearchModel shows detailed claim status for a single model.
func runResearchModel(kbDir, modelName string) error {
	modelDir, err := research.FindModel(kbDir, modelName)
	if err != nil {
		return err
	}

	ms, err := research.LoadModelStatus(modelDir)
	if err != nil {
		return fmt.Errorf("loading model status: %w", err)
	}
	if ms == nil {
		return fmt.Errorf("no claims found for model %s", filepath.Base(modelDir))
	}

	fmt.Printf("Research Status: %s\n", ms.ModelName)
	fmt.Printf("Source: %s\n", ms.Source)
	fmt.Println(strings.Repeat("-", 80))

	for _, cs := range ms.Claims {
		statusIcon := statusToIcon(cs.TestStatus)
		probeCount := ""
		if len(cs.Probes) > 0 {
			probeCount = fmt.Sprintf(" (%d probes)", len(cs.Probes))
		}

		priority := ""
		if cs.Priority != "" {
			priority = fmt.Sprintf(" [%s]", cs.Priority)
		}

		fmt.Printf("\n  %s %s%s  %s%s\n", statusIcon, cs.ID, priority, cs.TestStatus, probeCount)
		fmt.Printf("    %s\n", cs.Text)

		// Show probes
		for _, p := range cs.Probes {
			fmt.Printf("    - %s %s: %s\n", p.Date, p.Verdict, truncateForDisplay(p.Title, 60))
		}

		// Show how-to-verify for untested claims
		if cs.TestStatus == research.StatusUntested && cs.HowToVerify != "" {
			fmt.Printf("    Verify: %s\n", truncateForDisplay(cs.HowToVerify, 80))
		}
	}

	fmt.Println()
	fmt.Printf("Total: %d claims, %d tested\n", ms.TotalClaims, ms.TestedClaims)

	// Show spawn hint for untested claims
	untested := ms.TotalClaims - ms.TestedClaims
	if untested > 0 {
		fmt.Printf("\n%d untested claims. Run: orch research %s <claim-id> to spawn a probe\n", untested, ms.ModelName)
	}

	return nil
}

// runResearchSpawn creates a triage:ready issue to probe a specific claim.
func runResearchSpawn(kbDir, modelName, claimID string) error {
	modelDir, err := research.FindModel(kbDir, modelName)
	if err != nil {
		return err
	}

	ms, err := research.LoadModelStatus(modelDir)
	if err != nil {
		return fmt.Errorf("loading model status: %w", err)
	}
	if ms == nil {
		return fmt.Errorf("no claims found for model %s", filepath.Base(modelDir))
	}

	claim := research.FindClaim(ms, claimID)
	if claim == nil {
		return fmt.Errorf("claim %s not found in model %s", strings.ToUpper(claimID), ms.ModelName)
	}

	// Build issue title and description
	title := fmt.Sprintf("Probe %s: %s", claim.ID, truncateForDisplay(claim.Text, 60))

	var descParts []string
	descParts = append(descParts, fmt.Sprintf("Probe claim %s from model %s.", claim.ID, ms.ModelName))
	descParts = append(descParts, "")
	descParts = append(descParts, fmt.Sprintf("Claim: %s", claim.Text))

	if claim.HowToVerify != "" {
		descParts = append(descParts, "")
		descParts = append(descParts, fmt.Sprintf("How to verify: %s", claim.HowToVerify))
	}

	// Add prior probe context
	if len(claim.Probes) > 0 {
		descParts = append(descParts, "")
		descParts = append(descParts, fmt.Sprintf("Prior probes (%d):", len(claim.Probes)))
		for _, p := range claim.Probes {
			descParts = append(descParts, fmt.Sprintf("  - %s %s: %s", p.Date, p.Verdict, p.Title))
		}
	}

	descParts = append(descParts, "")
	descParts = append(descParts, fmt.Sprintf("Model: .kb/models/%s/model.md", ms.ModelName))
	descParts = append(descParts, "Constraint: one claim, one method, one verdict.")

	description := strings.Join(descParts, "\n")

	// Create issue via bd create
	bdPath, err := findBdCommand()
	if err != nil {
		return fmt.Errorf("bd command not found: %w", err)
	}

	args := []string{
		"create", title,
		"-d", description,
		"--type", "investigation",
		"-l", "triage:ready",
		"-l", "skill:probe",
	}

	cmd := exec.Command(bdPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd create failed: %w\nOutput: %s", err, string(output))
	}

	outputStr := strings.TrimSpace(string(output))
	fmt.Printf("Created probe issue: %s\n", outputStr)
	fmt.Printf("\nClaim: %s %s\n", claim.ID, truncateForDisplay(claim.Text, 70))
	fmt.Printf("Model: %s\n", ms.ModelName)
	fmt.Printf("Status: triage:ready (daemon will pick up or spawn manually)\n")

	return nil
}

// formatStatusCounts builds a compact status summary string.
func formatStatusCounts(counts map[research.TestStatus]int) string {
	var parts []string
	order := []research.TestStatus{
		research.StatusConfirmed,
		research.StatusExtended,
		research.StatusContradicted,
		research.StatusMixed,
		research.StatusUntested,
	}
	for _, s := range order {
		if n, ok := counts[s]; ok && n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, s))
		}
	}
	return strings.Join(parts, ", ")
}

// statusToIcon returns a display icon for a test status.
func statusToIcon(s research.TestStatus) string {
	switch s {
	case research.StatusConfirmed:
		return "[confirmed]"
	case research.StatusContradicted:
		return "[CONTRADICTED]"
	case research.StatusExtended:
		return "[extended]"
	case research.StatusMixed:
		return "[mixed]"
	case research.StatusUntested:
		return "[untested]"
	default:
		return "[?]"
	}
}

func truncateForDisplay(s string, maxLen int) string {
	// Remove newlines for display
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

