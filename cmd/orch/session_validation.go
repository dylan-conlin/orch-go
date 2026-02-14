// Package main provides the CLI entry point for orch-go.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

// ============================================================================
// Session Validation - Check handoff quality without ending session
// ============================================================================

// ValidationResult holds the result of validating a session handoff.
type ValidationResult struct {
	Unfilled []HandoffSection // Sections that still have placeholders
	Content  string           // Current handoff content
}

// ValidationOutput is the JSON output format for session validate.
type ValidationOutput struct {
	Found           bool                    `json:"found"`
	HandoffPath     string                  `json:"handoff_path,omitempty"`
	WindowName      string                  `json:"window_name"`
	TotalSections   int                     `json:"total_sections"`
	UnfilledCount   int                     `json:"unfilled_count"`
	RequiredFilled  int                     `json:"required_filled"`
	RequiredTotal   int                     `json:"required_total"`
	OptionalFilled  int                     `json:"optional_filled"`
	OptionalTotal   int                     `json:"optional_total"`
	UnfilledDetails []ValidationSectionInfo `json:"unfilled_details,omitempty"`
}

// ValidationSectionInfo describes an unfilled section for JSON output.
type ValidationSectionInfo struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Placeholder string `json:"placeholder"`
	Prompt      string `json:"prompt,omitempty"`
}

// validateHandoff reads SESSION_HANDOFF.md and checks for unfilled sections.
// Returns the list of sections that still contain placeholder patterns.
func validateHandoff(activeDir string) (*ValidationResult, error) {
	handoffPath := filepath.Join(activeDir, "SESSION_HANDOFF.md")

	content, err := os.ReadFile(handoffPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read handoff: %w", err)
	}

	contentStr := string(content)
	var unfilled []HandoffSection

	for _, section := range handoffSections {
		if strings.Contains(contentStr, section.Placeholder) {
			unfilled = append(unfilled, section)
		}
	}

	return &ValidationResult{
		Unfilled: unfilled,
		Content:  contentStr,
	}, nil
}

// runSessionValidate implements the `orch session validate` command.
// It checks the active session handoff for unfilled sections without ending the session.
func runSessionValidate() error {
	// Get window name from active session or current tmux window
	windowName, err := getWindowNameForValidation()
	if err != nil {
		if validateJSON {
			return outputValidationJSON(&ValidationOutput{
				Found:      false,
				WindowName: "",
			})
		}
		return err
	}

	// Get project directory
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get project directory: %w", err)
	}

	// Look for active handoff
	activeDir := filepath.Join(projectDir, ".orch", "session", windowName, "active")
	handoffPath := filepath.Join(activeDir, "SESSION_HANDOFF.md")

	// Check if active handoff exists
	if _, err := os.Stat(handoffPath); os.IsNotExist(err) {
		if validateJSON {
			return outputValidationJSON(&ValidationOutput{
				Found:      false,
				WindowName: windowName,
			})
		}
		fmt.Printf("No active handoff found for window %q\n", windowName)
		fmt.Printf("  Expected path: %s\n", handoffPath)
		fmt.Println("\nStart a session with: orch session start \"your goal\"")
		return nil
	}

	// Validate the handoff
	validation, err := validateHandoff(activeDir)
	if err != nil {
		return fmt.Errorf("failed to validate handoff: %w", err)
	}

	// Count required vs optional sections
	requiredTotal := 0
	optionalTotal := 0
	for _, section := range handoffSections {
		if section.Required {
			requiredTotal++
		} else {
			optionalTotal++
		}
	}

	requiredFilled := requiredTotal
	optionalFilled := optionalTotal
	var unfilledDetails []ValidationSectionInfo

	for _, section := range validation.Unfilled {
		unfilledDetails = append(unfilledDetails, ValidationSectionInfo{
			Name:        section.Name,
			Required:    section.Required,
			Placeholder: section.Placeholder,
			Prompt:      section.Prompt,
		})

		if section.Required {
			requiredFilled--
		} else {
			optionalFilled--
		}
	}

	// Output based on format
	if validateJSON {
		output := &ValidationOutput{
			Found:           true,
			HandoffPath:     handoffPath,
			WindowName:      windowName,
			TotalSections:   len(handoffSections),
			UnfilledCount:   len(validation.Unfilled),
			RequiredFilled:  requiredFilled,
			RequiredTotal:   requiredTotal,
			OptionalFilled:  optionalFilled,
			OptionalTotal:   optionalTotal,
			UnfilledDetails: unfilledDetails,
		}
		return outputValidationJSON(output)
	}

	// Human-readable output
	fmt.Printf("📋 Session Handoff Validation\n")
	fmt.Printf("   Window: %s\n", windowName)
	fmt.Printf("   Path: %s\n", handoffPath)
	fmt.Println()

	if len(validation.Unfilled) == 0 {
		fmt.Println("✅ All sections filled - ready to end session")
		fmt.Printf("   Required: %d/%d\n", requiredFilled, requiredTotal)
		fmt.Printf("   Optional: %d/%d\n", optionalFilled, optionalTotal)
	} else {
		fmt.Printf("⚠️  %d section(s) still unfilled:\n", len(validation.Unfilled))
		fmt.Printf("   Required: %d/%d\n", requiredFilled, requiredTotal)
		fmt.Printf("   Optional: %d/%d\n", optionalFilled, optionalTotal)
		fmt.Println()

		for _, section := range validation.Unfilled {
			marker := " "
			if section.Required {
				marker = "!"
			}
			fmt.Printf("   [%s] %s\n", marker, section.Name)
			fmt.Printf("       Placeholder: %s\n", section.Placeholder)
		}

		fmt.Println()
		fmt.Println("Run `orch session end` to complete and archive the handoff.")
	}

	return nil
}

// getWindowNameForValidation gets the window name for validation.
// Tries active session first, falls back to current tmux window.
func getWindowNameForValidation() (string, error) {
	// Try loading active session first
	store, err := session.New("")
	if err == nil && store.IsActive() {
		// Use window name from active session
		sess := store.Get()
		if sess != nil && sess.WindowName != "" {
			return sess.WindowName, nil
		}
	}

	// Fall back to current tmux window
	windowName, err := tmux.GetCurrentWindowName()
	if err != nil {
		return "", fmt.Errorf("failed to get window name: %w", err)
	}

	return windowName, nil
}

// outputValidationJSON outputs validation results as JSON.
func outputValidationJSON(output *ValidationOutput) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}
