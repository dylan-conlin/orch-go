// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

const (
	// SynthesisAutoCreateLabel is the beads label used for auto-created synthesis issues (for dedup).
	SynthesisAutoCreateLabel = "daemon:synthesis"
)

// SynthesisAutoCreateResult contains the result of running synthesis auto-creation.
type SynthesisAutoCreateResult struct {
	// Created is the number of beads issues created.
	Created int
	// Skipped is the number of clusters skipped (model exists or open issue exists).
	Skipped int
	// SkippedModelExists is how many were skipped because a model directory exists.
	SkippedModelExists int
	// SkippedDedup is how many were skipped due to existing open issue.
	SkippedDedup int
	// Evaluated is the number of clusters evaluated (count >= threshold).
	Evaluated int
	// CreatedIssues contains the beads IDs of created issues.
	CreatedIssues []string
	// Message is a human-readable summary.
	Message string
	// Error is set if the operation failed.
	Error error
}

// SynthesisAutoCreateService provides the I/O operations for synthesis auto-creation.
type SynthesisAutoCreateService interface {
	// HasOpenSynthesisIssue checks if an open beads issue already exists for the given topic.
	HasOpenSynthesisIssue(topic string) (bool, error)
	// CreateSynthesisIssue creates a beads issue for investigation synthesis of the given topic.
	// Returns the created issue ID.
	CreateSynthesisIssue(topic string, count int, investigations []string) (string, error)
	// ModelDirExists checks if .kb/models/{topic}/ directory exists.
	ModelDirExists(topic string) (bool, error)
	// LoadSynthesisSuggestions loads the current reflection suggestions.
	LoadSynthesisSuggestions() ([]SynthesisSuggestion, error)
}

// ShouldRunSynthesisAutoCreate returns true if periodic synthesis auto-creation should run.
func (d *Daemon) ShouldRunSynthesisAutoCreate() bool {
	return d.Scheduler.IsDue(TaskSynthesisAutoCreate)
}

// RunPeriodicSynthesisAutoCreate checks synthesis clusters from the most recent
// reflection results and creates triage:ready beads issues for clusters that:
//   - Have 5+ investigations (configurable threshold)
//   - Don't have a corresponding .kb/models/{topic}/ directory
//   - Don't already have an open synthesis issue
//
// Returns the result if the task was run, or nil if it wasn't due.
func (d *Daemon) RunPeriodicSynthesisAutoCreate() *SynthesisAutoCreateResult {
	if !d.ShouldRunSynthesisAutoCreate() {
		return nil
	}

	svc := d.SynthesisAutoCreate
	if svc == nil {
		return &SynthesisAutoCreateResult{
			Error:   fmt.Errorf("synthesis auto-create service not configured"),
			Message: "Synthesis auto-create: service not configured",
		}
	}

	threshold := d.Config.SynthesisAutoCreateThreshold
	if threshold <= 0 {
		threshold = 5
	}

	// Load synthesis suggestions from the most recent reflection
	suggestions, err := svc.LoadSynthesisSuggestions()
	if err != nil {
		return &SynthesisAutoCreateResult{
			Error:   err,
			Message: fmt.Sprintf("Synthesis auto-create: failed to load suggestions: %v", err),
		}
	}

	result := &SynthesisAutoCreateResult{}

	for _, s := range suggestions {
		if s.Count < threshold {
			continue
		}
		result.Evaluated++

		// Guard 2: Skip if model directory already exists
		modelExists, err := svc.ModelDirExists(s.Topic)
		if err != nil {
			// Non-fatal: log and skip this cluster
			result.Skipped++
			continue
		}
		if modelExists {
			result.Skipped++
			result.SkippedModelExists++
			continue
		}

		// Guard 1: Skip if open synthesis issue already exists
		hasOpen, err := svc.HasOpenSynthesisIssue(s.Topic)
		if err != nil {
			// Non-fatal: skip on error (fail-safe)
			result.Skipped++
			continue
		}
		if hasOpen {
			result.Skipped++
			result.SkippedDedup++
			continue
		}

		// Create the issue
		issueID, err := svc.CreateSynthesisIssue(s.Topic, s.Count, s.Investigations)
		if err != nil {
			result.Error = err
			result.Message = fmt.Sprintf("Synthesis auto-create: failed to create issue for %q: %v", s.Topic, err)
			// Continue processing other clusters
			continue
		}

		result.Created++
		result.CreatedIssues = append(result.CreatedIssues, issueID)
	}

	// Build summary message
	if result.Created > 0 {
		result.Message = fmt.Sprintf("Synthesis auto-create: created %d issue(s) for investigation clusters", result.Created)
		if result.Skipped > 0 {
			result.Message += fmt.Sprintf(", skipped %d (model: %d, dedup: %d)",
				result.Skipped, result.SkippedModelExists, result.SkippedDedup)
		}
	} else if result.Evaluated > 0 {
		result.Message = fmt.Sprintf("Synthesis auto-create: %d cluster(s) evaluated, all skipped (model: %d, dedup: %d)",
			result.Evaluated, result.SkippedModelExists, result.SkippedDedup)
	} else if result.Error == nil {
		result.Message = "Synthesis auto-create: no clusters above threshold"
	}

	d.Scheduler.MarkRun(TaskSynthesisAutoCreate)
	return result
}

// --- Default implementation ---

// defaultSynthesisAutoCreateService is the production implementation.
type defaultSynthesisAutoCreateService struct{}

func (s *defaultSynthesisAutoCreateService) LoadSynthesisSuggestions() ([]SynthesisSuggestion, error) {
	suggestions, err := LoadSuggestions()
	if err != nil {
		return nil, err
	}
	if suggestions == nil {
		return nil, nil
	}
	return suggestions.Synthesis, nil
}

func (s *defaultSynthesisAutoCreateService) ModelDirExists(topic string) (bool, error) {
	projectDir, err := os.Getwd()
	if err != nil {
		return false, fmt.Errorf("failed to get working directory: %w", err)
	}

	slug := TopicToSlug(topic)
	modelDir := filepath.Join(projectDir, ".kb", "models", slug)
	info, err := os.Stat(modelDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}

func (s *defaultSynthesisAutoCreateService) HasOpenSynthesisIssue(topic string) (bool, error) {
	// Use label-based dedup: list all open issues with the synthesis label,
	// then check if any match this topic in the title.
	issues, err := ListIssuesWithLabel(SynthesisAutoCreateLabel)
	if err != nil {
		return false, err
	}
	slug := TopicToSlug(topic)
	for _, issue := range issues {
		if strings.Contains(strings.ToLower(issue.Title), slug) ||
			strings.Contains(strings.ToLower(issue.Title), strings.ToLower(topic)) {
			return true, nil
		}
	}
	return false, nil
}

func (s *defaultSynthesisAutoCreateService) CreateSynthesisIssue(topic string, count int, investigations []string) (string, error) {
	slug := TopicToSlug(topic)
	title := fmt.Sprintf("Investigation synthesis: %s (%d investigations)", topic, count)

	desc := fmt.Sprintf("Auto-detected investigation cluster with %d investigations on topic %q. "+
		"No corresponding .kb/models/%s/ directory exists. "+
		"Synthesize findings into a model.", count, topic, slug)
	if len(investigations) > 0 {
		desc += "\n\nInvestigations:\n"
		for _, inv := range investigations {
			desc += "- " + inv + "\n"
		}
	}

	// Try RPC first, fallback to CLI
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			issue, err := client.Create(&beads.CreateArgs{
				Title:       title,
				Description: desc,
				IssueType:   "task",
				Priority:    3,
				Labels:      []string{SynthesisAutoCreateLabel, "triage:ready"},
			})
			if err == nil {
				return issue.ID, nil
			}
		}
	}

	// Fallback to CLI
	issue, err := beads.FallbackCreate(title, desc, "task", 3, []string{SynthesisAutoCreateLabel, "triage:ready"}, "")
	if err != nil {
		return "", err
	}
	return issue.ID, nil
}

// TopicToSlug converts a topic name to a filesystem-safe directory name.
// E.g., "Spawn Architecture" → "spawn-architecture"
var nonAlphanumDash = regexp.MustCompile(`[^a-z0-9-]+`)

func TopicToSlug(topic string) string {
	slug := strings.ToLower(strings.TrimSpace(topic))
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	slug = nonAlphanumDash.ReplaceAllString(slug, "")
	// Collapse multiple dashes
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	return slug
}
