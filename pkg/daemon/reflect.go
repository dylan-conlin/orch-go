// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/verify"
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

	// InvestigationPromotion suggestions for investigations marked recommend-yes.
	InvestigationPromotion []InvestigationPromotionSuggestion `json:"investigation_promotion,omitempty"`

	// InvestigationAuthority suggestions for investigations grouped by authority level.
	InvestigationAuthority []InvestigationAuthoritySuggestion `json:"investigation_authority,omitempty"`

	// DefectClass suggestions for recurring defect mechanisms in recent investigations.
	DefectClass []DefectClassSuggestion `json:"defect_class,omitempty"`

	// OrphanInvestigations are investigations with potential lineage gaps.
	// These have similar-topic peers but no prior-work citations.
	OrphanInvestigations []OrphanInvestigationSuggestion `json:"orphan_investigations,omitempty"`
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

// InvestigationPromotionSuggestion represents an investigation with recommend-yes awaiting decision creation.
type InvestigationPromotionSuggestion struct {
	File       string `json:"file"`
	Title      string `json:"title"`
	AgeDays    int    `json:"age_days"`
	Suggestion string `json:"suggestion"`
}

// InvestigationAuthoritySuggestion represents an investigation with unactioned recommendations grouped by authority level.
type InvestigationAuthoritySuggestion struct {
	File       string `json:"file"`
	Title      string `json:"title"`
	Authority  string `json:"authority"`
	NextAction string `json:"next_action"`
	AgeDays    int    `json:"age_days"`
	Suggestion string `json:"suggestion"`
}

// DefectClassSuggestion represents recurring investigations sharing a defect mechanism.
type DefectClassSuggestion struct {
	DefectClass    string   `json:"defect_class"`
	Count          int      `json:"count"`
	WindowDays     int      `json:"window_days"`
	Investigations []string `json:"investigations"`
	Suggestion     string   `json:"suggestion"`
	IssueCreated   bool     `json:"issue_created,omitempty"`
}

// OrphanInvestigationSuggestion represents an investigation with potential lineage gaps.
// These are investigations that have similar-topic peers but no prior-work citations.
type OrphanInvestigationSuggestion struct {
	Path                  string   `json:"path"`
	Topic                 string   `json:"topic"`
	SimilarInvestigations []string `json:"similar_investigations"`
	Suggestion            string   `json:"suggestion"`
}

// kbReflectOutput represents the raw output from kb reflect --format json.
type kbReflectOutput struct {
	Synthesis              []SynthesisSuggestion              `json:"synthesis,omitempty"`
	Promote                []PromoteSuggestion                `json:"promote,omitempty"`
	Stale                  []StaleSuggestion                  `json:"stale,omitempty"`
	Drift                  []DriftSuggestion                  `json:"drift,omitempty"`
	Refine                 []kbRefineOutput                   `json:"refine,omitempty"`
	InvestigationPromotion []InvestigationPromotionSuggestion `json:"investigation_promotion,omitempty"`
	InvestigationAuthority []InvestigationAuthoritySuggestion `json:"investigation_authority,omitempty"`
	DefectClass            []DefectClassSuggestion            `json:"defect_class,omitempty"`
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
// If createIssues is true, it additionally triggers side-effect issue creation
// for supported reflection types (currently synthesis and defect-class).
func RunReflectionWithOptions(createIssues bool) (*ReflectSuggestions, error) {
	args := []string{"reflect", "--format", "json"}

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
		Timestamp:              time.Now().UTC(),
		Synthesis:              rawOutput.Synthesis,
		Promote:                rawOutput.Promote,
		Stale:                  rawOutput.Stale,
		Drift:                  rawOutput.Drift,
		Refine:                 refine,
		InvestigationPromotion: rawOutput.InvestigationPromotion,
		InvestigationAuthority: rawOutput.InvestigationAuthority,
		DefectClass:            rawOutput.DefectClass,
	}

	if createIssues {
		for _, reflectType := range []string{"synthesis", "defect-class"} {
			if err := triggerReflectIssueCreation(reflectType); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to auto-create %s issues via kb reflect: %v\n", reflectType, err)
			}
		}
	}

	return suggestions, nil
}

func triggerReflectIssueCreation(reflectType string) error {
	cmd := exec.Command("kb", "reflect", "--type", reflectType, "--create-issue", "--format", "json")
	if _, err := cmd.Output(); err != nil {
		return fmt.Errorf("kb reflect --type %s --create-issue failed: %w", reflectType, err)
	}
	return nil
}

// RunReflectionWithOrphans executes kb reflect and also detects orphan investigations.
// projectDir is used for orphan detection; if empty, uses current working directory.
func RunReflectionWithOrphans(createIssues bool, projectDir string) (*ReflectSuggestions, error) {
	suggestions, err := RunReflectionWithOptions(createIssues)
	if err != nil {
		return nil, err
	}

	// Detect orphan investigations
	if projectDir == "" {
		projectDir, _ = os.Getwd()
	}

	orphans, err := verify.DetectOrphanInvestigations(projectDir)
	if err != nil {
		// Log warning but don't fail - orphan detection is supplementary
		fmt.Fprintf(os.Stderr, "Warning: orphan investigation detection failed: %v\n", err)
	} else if orphans.HasOrphans() {
		// Convert verify.OrphanInvestigation to daemon.OrphanInvestigationSuggestion
		for _, o := range orphans.Orphans {
			suggestions.OrphanInvestigations = append(suggestions.OrphanInvestigations, OrphanInvestigationSuggestion{
				Path:                  o.Path,
				Topic:                 o.Topic,
				SimilarInvestigations: o.SimilarInvestigations,
				Suggestion:            o.Suggestion,
			})
		}
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
	return len(s.Synthesis) > 0 || len(s.Promote) > 0 || len(s.Stale) > 0 || len(s.Drift) > 0 || len(s.Refine) > 0 || len(s.InvestigationPromotion) > 0 || len(s.InvestigationAuthority) > 0 || len(s.DefectClass) > 0 || len(s.OrphanInvestigations) > 0
}

// TotalCount returns the total number of suggestions across all categories.
func (s *ReflectSuggestions) TotalCount() int {
	if s == nil {
		return 0
	}
	return len(s.Synthesis) + len(s.Promote) + len(s.Stale) + len(s.Drift) + len(s.Refine) + len(s.InvestigationPromotion) + len(s.InvestigationAuthority) + len(s.DefectClass) + len(s.OrphanInvestigations)
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
	if len(s.InvestigationPromotion) > 0 {
		parts = append(parts, fmt.Sprintf("%d investigation promotions", len(s.InvestigationPromotion)))
	}
	if len(s.InvestigationAuthority) > 0 {
		parts = append(parts, fmt.Sprintf("%d recommendations by authority", len(s.InvestigationAuthority)))
	}
	if len(s.DefectClass) > 0 {
		parts = append(parts, fmt.Sprintf("%d recurring defect classes", len(s.DefectClass)))
	}
	if len(s.OrphanInvestigations) > 0 {
		parts = append(parts, fmt.Sprintf("%d potential lineage gaps", len(s.OrphanInvestigations)))
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

// RunAndSaveReflectionWithOrphans runs kb reflect with orphan detection and saves the results.
func RunAndSaveReflectionWithOrphans(createIssues bool, projectDir string) *ReflectResult {
	suggestions, err := RunReflectionWithOrphans(createIssues, projectDir)
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
