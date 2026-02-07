// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// ReflectSuggestions holds the output of kb reflect analysis.
type ReflectSuggestions struct {
	// Timestamp when the analysis was performed.
	Timestamp time.Time `json:"timestamp"`

	// Synthesis suggestions for investigation clusters.
	Synthesis []SynthesisSuggestion `json:"synthesis,omitempty"`

	// Promote suggestions for kn entries worth promoting.
	Promote []PromoteSuggestion `json:"promote,omitempty"`

	// Stale decisions with low citations.
	Stale []StaleSuggestion `json:"stale,omitempty"`

	// Drift detected constraints that may conflict with code.
	Drift []DriftSuggestion `json:"drift,omitempty"`

	// Refine suggestions for kn entries that refine existing principles.
	Refine []RefineSuggestion `json:"refine,omitempty"`
}

// SynthesisSuggestion represents a topic with multiple investigations.
type SynthesisSuggestion struct {
	Topic          string   `json:"topic"`
	Count          int      `json:"count"`
	Investigations []string `json:"investigations"`
	Suggestion     string   `json:"suggestion"`
}

// PromoteSuggestion represents a kn entry worth promoting.
type PromoteSuggestion struct {
	ID         string `json:"id"`
	Content    string `json:"content"`
	Suggestion string `json:"suggestion"`
}

// StaleSuggestion represents a decision with no citations.
type StaleSuggestion struct {
	Path       string `json:"path"`
	Age        int    `json:"age_days"`
	Suggestion string `json:"suggestion"`
}

// DriftSuggestion represents a constraint that may be outdated.
type DriftSuggestion struct {
	ID         string `json:"id"`
	Content    string `json:"content"`
	Suggestion string `json:"suggestion"`
}

// RefineSuggestion represents a kn entry that refines an existing principle.
type RefineSuggestion struct {
	ID         string   `json:"id"`
	Content    string   `json:"content"`
	Principle  string   `json:"principle"`
	MatchTerms []string `json:"match_terms"`
	Suggestion string   `json:"suggestion"`
}

// kbReflectOutput represents the raw output from kb reflect --format json.
type kbReflectOutput struct {
	Synthesis []SynthesisSuggestion `json:"synthesis,omitempty"`
	Promote   []PromoteSuggestion   `json:"promote,omitempty"`
	Stale     []StaleSuggestion     `json:"stale,omitempty"`
	Drift     []DriftSuggestion     `json:"drift,omitempty"`
	Refine    []kbRefineOutput      `json:"refine,omitempty"`
}

// kbRefineOutput represents the raw refine entry from kb reflect.
type kbRefineOutput struct {
	Entry struct {
		ID      string `json:"id"`
		Content string `json:"content"`
	} `json:"entry"`
	Principle  string   `json:"principle"`
	MatchTerms []string `json:"match_terms"`
	Suggestion string   `json:"suggestion"`
}

// SuggestionsPath returns the default path for storing reflection suggestions.
func SuggestionsPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".orch", "reflect-suggestions.json")
}

// RunReflection executes kb reflect and returns the parsed suggestions.
// This is the default implementation that shells out to kb.
func RunReflection() (*ReflectSuggestions, error) {
	return RunReflectionWithOptions(false)
}

// RunReflectionWithOptions executes kb reflect with configurable options.
// If createIssues is true, it passes --type synthesis --create-issue to kb reflect
// which will automatically create beads issues for topics with 10+ investigations.
func RunReflectionWithOptions(createIssues bool) (*ReflectSuggestions, error) {
	args := []string{"reflect", "--format", "json"}
	if createIssues {
		args = append(args, "--type", "synthesis", "--create-issue")
	}

	cmd := exec.Command("kb", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run kb reflect: %w", err)
	}

	var rawOutput kbReflectOutput
	if err := json.Unmarshal(output, &rawOutput); err != nil {
		return nil, fmt.Errorf("failed to parse kb reflect output: %w", err)
	}

	// Convert refine output to suggestions
	var refine []RefineSuggestion
	for _, r := range rawOutput.Refine {
		refine = append(refine, RefineSuggestion{
			ID:         r.Entry.ID,
			Content:    r.Entry.Content,
			Principle:  r.Principle,
			MatchTerms: r.MatchTerms,
			Suggestion: r.Suggestion,
		})
	}

	suggestions := &ReflectSuggestions{
		Timestamp: time.Now().UTC(),
		Synthesis: rawOutput.Synthesis,
		Promote:   rawOutput.Promote,
		Stale:     rawOutput.Stale,
		Drift:     rawOutput.Drift,
		Refine:    refine,
	}

	return suggestions, nil
}

// SaveSuggestions saves reflection suggestions to the default path.
func SaveSuggestions(suggestions *ReflectSuggestions) error {
	path := SuggestionsPath()
	if path == "" {
		return fmt.Errorf("could not determine suggestions path")
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(suggestions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal suggestions: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write suggestions: %w", err)
	}

	return nil
}

// LoadSuggestions loads reflection suggestions from the default path.
// Returns nil if the file doesn't exist.
func LoadSuggestions() (*ReflectSuggestions, error) {
	path := SuggestionsPath()
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read suggestions: %w", err)
	}

	var suggestions ReflectSuggestions
	if err := json.Unmarshal(data, &suggestions); err != nil {
		return nil, fmt.Errorf("failed to parse suggestions: %w", err)
	}

	return &suggestions, nil
}

// HasSuggestions returns true if there are any suggestions to review.
func (s *ReflectSuggestions) HasSuggestions() bool {
	if s == nil {
		return false
	}
	return len(s.Synthesis) > 0 || len(s.Promote) > 0 || len(s.Stale) > 0 || len(s.Drift) > 0 || len(s.Refine) > 0
}

// TotalCount returns the total number of suggestions across all categories.
func (s *ReflectSuggestions) TotalCount() int {
	if s == nil {
		return 0
	}
	return len(s.Synthesis) + len(s.Promote) + len(s.Stale) + len(s.Drift) + len(s.Refine)
}

// Summary returns a human-readable summary of suggestions.
func (s *ReflectSuggestions) Summary() string {
	if s == nil || !s.HasSuggestions() {
		return "No reflection suggestions"
	}

	parts := []string{}
	if len(s.Synthesis) > 0 {
		parts = append(parts, fmt.Sprintf("%d synthesis opportunities", len(s.Synthesis)))
	}
	if len(s.Promote) > 0 {
		parts = append(parts, fmt.Sprintf("%d promotion candidates", len(s.Promote)))
	}
	if len(s.Stale) > 0 {
		parts = append(parts, fmt.Sprintf("%d stale decisions", len(s.Stale)))
	}
	if len(s.Drift) > 0 {
		parts = append(parts, fmt.Sprintf("%d potential drifts", len(s.Drift)))
	}
	if len(s.Refine) > 0 {
		parts = append(parts, fmt.Sprintf("%d principle refinements", len(s.Refine)))
	}

	result := ""
	for i, part := range parts {
		if i == 0 {
			result = part
		} else if i == len(parts)-1 {
			result += ", and " + part
		} else {
			result += ", " + part
		}
	}
	return result
}

// ReflectResult contains the result of running reflection analysis.
type ReflectResult struct {
	Suggestions *ReflectSuggestions
	Saved       bool
	Message     string
	Error       error
}

// RunAndSaveReflection runs kb reflect and saves the results.
func RunAndSaveReflection() *ReflectResult {
	return RunAndSaveReflectionWithOptions(false)
}

// RunAndSaveReflectionWithOptions runs kb reflect with options and saves the results.
// If createIssues is true, it will create beads issues for synthesis opportunities.
func RunAndSaveReflectionWithOptions(createIssues bool) *ReflectResult {
	suggestions, err := RunReflectionWithOptions(createIssues)
	if err != nil {
		return &ReflectResult{
			Error:   err,
			Message: fmt.Sprintf("Failed to run reflection: %v", err),
		}
	}

	if err := SaveSuggestions(suggestions); err != nil {
		return &ReflectResult{
			Suggestions: suggestions,
			Saved:       false,
			Error:       err,
			Message:     fmt.Sprintf("Ran reflection but failed to save: %v", err),
		}
	}

	return &ReflectResult{
		Suggestions: suggestions,
		Saved:       true,
		Message:     fmt.Sprintf("Reflection complete: %s", suggestions.Summary()),
	}
}

// DefaultRunReflection is the default implementation for running reflection.
// This is used by the Daemon and can be mocked for testing.
func DefaultRunReflection(createIssues bool) (*ReflectResult, error) {
	result := RunAndSaveReflectionWithOptions(createIssues)
	if result.Error != nil {
		return result, result.Error
	}
	return result, nil
}
