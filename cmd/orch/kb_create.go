package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/dylan-conlin/orch-go/pkg/spawn/gates"
)

// validModelName matches lowercase kebab-case names.
var validModelName = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// runKBCreateModel creates a new model directory with template and probes subdirectory.
func runKBCreateModel(name, projectDir string) error {
	if name == "" {
		return fmt.Errorf("model name is required")
	}
	if !validModelName.MatchString(name) {
		return fmt.Errorf("invalid model name %q: must be lowercase kebab-case (e.g., \"spawn-architecture\")", name)
	}

	modelsDir := filepath.Join(projectDir, ".kb", "models")
	modelDir := filepath.Join(modelsDir, name)

	// Check if model already exists
	if _, err := os.Stat(modelDir); err == nil {
		return fmt.Errorf("model %q already exists at %s", name, modelDir)
	}

	// Read template
	templatePath := filepath.Join(modelsDir, "TEMPLATE.md")
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read TEMPLATE.md: %w (expected at %s)", err, templatePath)
	}

	// Create directory structure
	probesDir := filepath.Join(modelDir, "probes")
	if err := os.MkdirAll(probesDir, 0755); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}

	// Write model.md from template
	modelFile := filepath.Join(modelDir, "model.md")
	if err := os.WriteFile(modelFile, templateContent, 0644); err != nil {
		return fmt.Errorf("failed to write model.md: %w", err)
	}

	fmt.Printf("Created model: %s\n", modelDir)
	fmt.Printf("  %s/model.md\n", name)
	fmt.Printf("  %s/probes/\n", name)

	return nil
}

// kbAgreementCheckResult mirrors the JSON output of kb agreements check --json.
type kbAgreementCheckResult struct {
	AgreementID string `json:"agreement_id"`
	Title       string `json:"title"`
	Severity    string `json:"severity"`
	Pass        bool   `json:"pass"`
	Message     string `json:"message"`
	AutoFix     string `json:"auto_fix,omitempty"`
}

// buildAgreementsChecker creates a function that runs kb agreements check --json
// in a given project directory and returns parsed results.
func buildAgreementsChecker() func(string) (*gates.AgreementsResult, error) {
	return func(projectDir string) (*gates.AgreementsResult, error) {
		cmd := exec.Command("kb", "agreements", "check", "--json")
		cmd.Dir = projectDir

		output, err := cmd.Output()
		if err != nil {
			// kb agreements check exits non-zero on failures, but still outputs JSON.
			// Only treat as error if we can't get any output.
			if exitErr, ok := err.(*exec.ExitError); ok {
				// Exit code 1 or 2 means checks failed but JSON was produced
				if len(output) == 0 {
					return nil, fmt.Errorf("kb agreements check failed: %s", string(exitErr.Stderr))
				}
				// Fall through to parse the JSON output
			} else {
				return nil, fmt.Errorf("kb agreements check: %w", err)
			}
		}

		var checks []kbAgreementCheckResult
		if err := json.Unmarshal(output, &checks); err != nil {
			return nil, fmt.Errorf("failed to parse kb agreements check output: %w", err)
		}

		result := &gates.AgreementsResult{
			Total: len(checks),
		}

		for _, check := range checks {
			if check.Pass {
				result.Passed++
			} else {
				result.Failed++
				result.Failures = append(result.Failures, gates.AgreementFailure{
					AgreementID: check.AgreementID,
					Title:       check.Title,
					Severity:    check.Severity,
					Message:     check.Message,
					AutoFix:     check.AutoFix,
				})
			}
		}

		return result, nil
	}
}
