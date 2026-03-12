// Package artifactsync provides change-scope classification and drift event
// logging for the artifact sync mechanism. At completion time, it analyzes
// git diffs to classify what kind of changes were made (new-command, new-flag,
// new-event, etc.) and emits DriftEvents to ~/.orch/artifact-drift.jsonl.
package artifactsync

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Change-scope categories that trigger artifact updates.
const (
	ScopeNewCommand  = "new-command"
	ScopeNewFlag     = "new-flag"
	ScopeNewEvent    = "new-event"
	ScopeNewSkill    = "new-skill"
	ScopeNewPackage  = "new-package"
	ScopeAPIChange   = "api-change"
	ScopeConfigChange = "config-change"
)

// DriftEvent records a change-scope classification at completion time.
type DriftEvent struct {
	BeadsID      string   `json:"beads_id,omitempty"`
	Skill        string   `json:"skill,omitempty"`
	ChangeScopes []string `json:"change_scopes"`
	FilesChanged []string `json:"files_changed"`
	CommitRange  string   `json:"commit_range,omitempty"`
}

// driftEventEnvelope wraps a DriftEvent for JSONL serialization.
type driftEventEnvelope struct {
	Type      string     `json:"type"`
	Timestamp int64      `json:"timestamp"`
	Data      DriftEvent `json:"data"`
}

// DefaultDriftLogPath returns the default path to artifact-drift.jsonl.
func DefaultDriftLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".orch/artifact-drift.jsonl"
	}
	return filepath.Join(home, ".orch", "artifact-drift.jsonl")
}

// LogDriftEvent appends a DriftEvent to the given JSONL file.
func LogDriftEvent(path string, event DriftEvent) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create drift log directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open drift log: %w", err)
	}
	defer f.Close()

	envelope := driftEventEnvelope{
		Type:      "artifact.drift",
		Timestamp: time.Now().Unix(),
		Data:      event,
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to marshal drift event: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write drift event: %w", err)
	}

	return nil
}

// ClassifyChanges analyzes file paths and their diffs to determine change-scope categories.
// Returns deduplicated list of scope strings.
func ClassifyChanges(files []string, diffs map[string]string) []string {
	seen := make(map[string]bool)
	var scopes []string

	add := func(scope string) {
		if !seen[scope] {
			seen[scope] = true
			scopes = append(scopes, scope)
		}
	}

	for _, file := range files {
		diff := diffs[file]

		// new-command: new AddCommand() call in cmd/orch/
		if strings.HasPrefix(file, "cmd/orch/") && strings.HasSuffix(file, ".go") {
			if strings.Contains(diff, "AddCommand(") {
				add(ScopeNewCommand)
			}
			// new-flag: new Flags().String/Bool/Int call
			if strings.Contains(diff, ".Flags().") {
				add(ScopeNewFlag)
			}
		}

		// new-event: new event type constant in pkg/events/
		if strings.HasPrefix(file, "pkg/events/") {
			if strings.Contains(diff, "EventType") {
				add(ScopeNewEvent)
			}
		}

		// new-skill: new file in skills/src/
		if strings.HasPrefix(file, "skills/src/") {
			add(ScopeNewSkill)
		}

		// new-package: new file in pkg/ (heuristic: file contains "package " declaration in diff)
		if strings.HasPrefix(file, "pkg/") && strings.Contains(diff, "+package ") {
			// Check it's a new package, not just a modified file in existing package
			parts := strings.Split(file, "/")
			if len(parts) >= 3 {
				add(ScopeNewPackage)
			}
		}

		// api-change: modified handler in serve*.go
		if strings.HasPrefix(file, "cmd/orch/serve") && strings.HasSuffix(file, ".go") {
			if strings.Contains(diff, "+func ") {
				add(ScopeAPIChange)
			}
		}

		// config-change: modified config struct with yaml tags
		if strings.Contains(diff, "`yaml:") || strings.Contains(diff, "`json:") {
			if strings.Contains(file, "config") || strings.HasPrefix(file, "pkg/spawn/") {
				add(ScopeConfigChange)
			}
		}
	}

	return scopes
}

// CaptureChangeScopes runs git diff analysis for an agent's changes and returns
// classified change scopes. Uses the agent's git baseline for precise scoping.
// Returns nil if no notable changes detected.
func CaptureChangeScopes(projectDir, baseline string) ([]string, []string) {
	if projectDir == "" || baseline == "" {
		return nil, nil
	}

	// Get list of changed files
	cmd := exec.Command("git", "diff", "--name-only", baseline+"..HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return nil, nil
	}

	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}

	if len(files) == 0 {
		return nil, nil
	}

	// Get per-file diffs for classification
	diffs := make(map[string]string)
	for _, file := range files {
		cmd := exec.Command("git", "diff", baseline+"..HEAD", "--", file)
		cmd.Dir = projectDir
		diffOutput, err := cmd.Output()
		if err != nil {
			continue
		}
		// Only keep added lines for classification (lines starting with +)
		var addedLines []string
		for _, line := range strings.Split(string(diffOutput), "\n") {
			if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				addedLines = append(addedLines, line)
			}
		}
		if len(addedLines) > 0 {
			diffs[file] = strings.Join(addedLines, "\n")
		}
	}

	scopes := ClassifyChanges(files, diffs)
	return scopes, files
}
