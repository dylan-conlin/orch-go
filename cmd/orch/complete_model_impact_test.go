package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractModelKeywords(t *testing.T) {
	tests := []struct {
		dirName  string
		domain   string
		wantMin  int // minimum number of keywords expected
		wantKeys []string
	}{
		{
			dirName:  "completion-verification",
			domain:   "Completion / Verification / Quality Gates",
			wantMin:  2,
			wantKeys: []string{"completion", "verification"},
		},
		{
			dirName:  "agent-lifecycle-state-model",
			domain:   "Agent Lifecycle / State Management",
			wantMin:  2,
			wantKeys: []string{"agent", "lifecycle", "state"},
		},
		{
			dirName:  "spawn-architecture",
			domain:   "Agent Spawning / Workspace Creation",
			wantMin:  2,
			wantKeys: []string{"spawn", "spawning"},
		},
		{
			dirName:  "daemon-autonomous-operation",
			domain:   "Daemon / Autonomous Spawning / Batch Processing",
			wantMin:  2,
			wantKeys: []string{"daemon", "autonomous"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.dirName, func(t *testing.T) {
			keywords := extractModelKeywords(tt.dirName, tt.domain)
			if len(keywords) < tt.wantMin {
				t.Errorf("got %d keywords, want at least %d: %v", len(keywords), tt.wantMin, keywords)
			}
			for _, want := range tt.wantKeys {
				found := false
				for _, got := range keywords {
					if got == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected keyword %q in %v", want, keywords)
				}
			}
		})
	}
}

func TestMatchSynthesisToModels(t *testing.T) {
	models := []ModelInfo{
		{
			DirName:  "completion-verification",
			Name:     "Completion Verification Architecture",
			Domain:   "Completion / Verification / Quality Gates",
			Keywords: []string{"completion", "verification", "quality", "gates"},
		},
		{
			DirName:  "spawn-architecture",
			Name:     "Spawn Architecture",
			Domain:   "Agent Spawning / Workspace Creation",
			Keywords: []string{"spawn", "spawning", "workspace", "creation"},
		},
		{
			DirName:  "macos-click-freeze",
			Name:     "macOS Click Freeze",
			Domain:   "macOS input subsystem",
			Keywords: []string{"macos", "click", "freeze", "trackpad"},
		},
	}

	tests := []struct {
		name          string
		synthesisText string
		wantMatches   []string // expected model dirNames
		wantNoMatch   []string // models that should NOT match
	}{
		{
			name:          "completion work matches completion model",
			synthesisText: "Added new verification gate to completion pipeline. Enhanced quality checks for agent completion workflow.",
			wantMatches:   []string{"completion-verification"},
			wantNoMatch:   []string{"macos-click-freeze"},
		},
		{
			name:          "spawn work matches spawn model",
			synthesisText: "Refactored spawn workspace creation flow. Fixed spawn context generation bug.",
			wantMatches:   []string{"spawn-architecture"},
			wantNoMatch:   []string{"macos-click-freeze"},
		},
		{
			name:          "unrelated work matches nothing",
			synthesisText: "Fixed typo in README. Updated documentation formatting.",
			wantNoMatch:   []string{"completion-verification", "spawn-architecture", "macos-click-freeze"},
		},
		{
			name:          "multiple matches",
			synthesisText: "Modified spawn workspace creation and added completion verification gate.",
			wantMatches:   []string{"completion-verification", "spawn-architecture"},
			wantNoMatch:   []string{"macos-click-freeze"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := matchSynthesisToModels(tt.synthesisText, models)
			matchedDirs := make(map[string]bool)
			for _, m := range matches {
				matchedDirs[m.Model.DirName] = true
			}

			for _, want := range tt.wantMatches {
				if !matchedDirs[want] {
					t.Errorf("expected match for %q, got matches: %v", want, matchedDirs)
				}
			}
			for _, notWant := range tt.wantNoMatch {
				if matchedDirs[notWant] {
					t.Errorf("unexpected match for %q", notWant)
				}
			}
		})
	}
}

func TestFormatModelImpactAdvisory(t *testing.T) {
	t.Run("no matches returns empty", func(t *testing.T) {
		result := formatModelImpactAdvisory(nil)
		if result != "" {
			t.Errorf("expected empty string for nil matches, got: %s", result)
		}
	})

	t.Run("with matches formats advisory", func(t *testing.T) {
		matches := []ModelImpactMatch{
			{
				Model: ModelInfo{
					DirName: "completion-verification",
					Name:    "Completion Verification Architecture",
				},
				MatchedKeywords: []string{"completion", "verification"},
			},
		}
		result := formatModelImpactAdvisory(matches)
		if result == "" {
			t.Error("expected non-empty advisory")
		}
		if !strings.Contains(result, "MODEL IMPACT") {
			t.Error("expected MODEL IMPACT header")
		}
		if !strings.Contains(result, "completion-verification") {
			t.Error("expected model name in advisory")
		}
	})
}

func TestDiscoverModels(t *testing.T) {
	// Create temp directory with fake model corpus
	tmpDir := t.TempDir()
	modelsDir := filepath.Join(tmpDir, ".kb", "models")

	// Create model directories with model.md files
	model1Dir := filepath.Join(modelsDir, "test-model-one")
	os.MkdirAll(model1Dir, 0755)
	os.WriteFile(filepath.Join(model1Dir, "model.md"), []byte(`# Model: Test Model One

**Domain:** Testing / Unit Tests
**Last Updated:** 2026-02-28

## Summary
This is a test model.
`), 0644)

	model2Dir := filepath.Join(modelsDir, "another-example")
	os.MkdirAll(model2Dir, 0755)
	os.WriteFile(filepath.Join(model2Dir, "model.md"), []byte(`# Model: Another Example

**Domain:** Examples / Demos
**Last Updated:** 2026-02-28

## Summary
Another model.
`), 0644)

	// Also create archived dir (should be skipped)
	archivedDir := filepath.Join(modelsDir, "archived")
	os.MkdirAll(archivedDir, 0755)

	models := discoverModels(modelsDir)
	if len(models) != 2 {
		t.Errorf("expected 2 models, got %d: %+v", len(models), models)
	}

	// Check that models have keywords
	for _, m := range models {
		if len(m.Keywords) == 0 {
			t.Errorf("model %q has no keywords", m.DirName)
		}
	}
}

func TestRunModelImpactAdvisory(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	projectDir := tmpDir

	// Create model corpus
	modelsDir := filepath.Join(projectDir, ".kb", "models")
	modelDir := filepath.Join(modelsDir, "spawn-architecture")
	os.MkdirAll(modelDir, 0755)
	os.WriteFile(filepath.Join(modelDir, "model.md"), []byte(`# Model: Spawn Architecture

**Domain:** Agent Spawning / Workspace Creation

## Summary
Spawn creates workspaces.
`), 0644)

	// Create workspace with SYNTHESIS.md
	workspacePath := filepath.Join(tmpDir, "workspace")
	os.MkdirAll(workspacePath, 0755)
	os.WriteFile(filepath.Join(workspacePath, "SYNTHESIS.md"), []byte(`## TLDR

Refactored the spawn workspace creation flow.

## Delta (What Changed)

Modified spawn context generation for workspace creation.
`), 0644)

	result := RunModelImpactAdvisory(projectDir, workspacePath)
	if result == "" {
		t.Error("expected non-empty advisory for spawn-related synthesis")
	}
	if !strings.Contains(result, "spawn-architecture") {
		t.Errorf("expected spawn-architecture match in: %s", result)
	}
}
