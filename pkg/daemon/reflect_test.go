// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestReflectSuggestions_HasSuggestions(t *testing.T) {
	tests := []struct {
		name string
		s    *ReflectSuggestions
		want bool
	}{
		{"nil", nil, false},
		{"empty", &ReflectSuggestions{}, false},
		{"synthesis only", &ReflectSuggestions{
			Synthesis: []SynthesisSuggestion{{Topic: "test", Count: 3}},
		}, true},
		{"promote only", &ReflectSuggestions{
			Promote: []PromoteSuggestion{{ID: "1", Content: "test"}},
		}, true},
		{"stale only", &ReflectSuggestions{
			Stale: []StaleSuggestion{{Path: "test.md", Age: 10}},
		}, true},
		{"drift only", &ReflectSuggestions{
			Drift: []DriftSuggestion{{ID: "1", Content: "test"}},
		}, true},
		{"all types", &ReflectSuggestions{
			Synthesis: []SynthesisSuggestion{{Topic: "test", Count: 3}},
			Promote:   []PromoteSuggestion{{ID: "1", Content: "test"}},
			Stale:     []StaleSuggestion{{Path: "test.md", Age: 10}},
			Drift:     []DriftSuggestion{{ID: "1", Content: "test"}},
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.HasSuggestions()
			if got != tt.want {
				t.Errorf("HasSuggestions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReflectSuggestions_TotalCount(t *testing.T) {
	tests := []struct {
		name string
		s    *ReflectSuggestions
		want int
	}{
		{"nil", nil, 0},
		{"empty", &ReflectSuggestions{}, 0},
		{"synthesis only", &ReflectSuggestions{
			Synthesis: []SynthesisSuggestion{{Topic: "a"}, {Topic: "b"}},
		}, 2},
		{"mixed", &ReflectSuggestions{
			Synthesis: []SynthesisSuggestion{{Topic: "a"}},
			Promote:   []PromoteSuggestion{{ID: "1"}},
			Stale:     []StaleSuggestion{{Path: "x"}},
			Drift:     []DriftSuggestion{{ID: "2"}},
		}, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.TotalCount()
			if got != tt.want {
				t.Errorf("TotalCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReflectSuggestions_Summary(t *testing.T) {
	tests := []struct {
		name     string
		s        *ReflectSuggestions
		contains []string
	}{
		{"nil", nil, []string{"No reflection suggestions"}},
		{"empty", &ReflectSuggestions{}, []string{"No reflection suggestions"}},
		{"synthesis only", &ReflectSuggestions{
			Synthesis: []SynthesisSuggestion{{Topic: "a"}, {Topic: "b"}},
		}, []string{"2 synthesis opportunities"}},
		{"multiple types", &ReflectSuggestions{
			Synthesis: []SynthesisSuggestion{{Topic: "a"}},
			Stale:     []StaleSuggestion{{Path: "x"}, {Path: "y"}},
		}, []string{"1 synthesis", "2 stale"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.Summary()
			for _, want := range tt.contains {
				if !containsSubstring(got, want) {
					t.Errorf("Summary() = %q, want to contain %q", got, want)
				}
			}
		})
	}
}

func TestSaveSuggestions_CreatesDirAndFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	suggestions := &ReflectSuggestions{
		Timestamp: time.Now().UTC(),
		Synthesis: []SynthesisSuggestion{
			{Topic: "test", Count: 5, Suggestion: "Consider synthesis"},
		},
	}

	err := SaveSuggestions(suggestions)
	if err != nil {
		t.Fatalf("SaveSuggestions() error = %v", err)
	}

	// Verify file exists
	path := filepath.Join(tmpDir, ".orch", "reflect-suggestions.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("Expected file to exist at %s", path)
	}

	// Verify content
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var loaded ReflectSuggestions
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(loaded.Synthesis) != 1 {
		t.Errorf("Expected 1 synthesis suggestion, got %d", len(loaded.Synthesis))
	}
	if loaded.Synthesis[0].Topic != "test" {
		t.Errorf("Expected topic 'test', got %q", loaded.Synthesis[0].Topic)
	}
}

func TestLoadSuggestions_FileNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	suggestions, err := LoadSuggestions()
	if err != nil {
		t.Fatalf("LoadSuggestions() error = %v", err)
	}
	if suggestions != nil {
		t.Errorf("LoadSuggestions() expected nil for non-existent file, got %v", suggestions)
	}
}

func TestLoadSuggestions_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create the directory and file
	dir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}

	suggestions := &ReflectSuggestions{
		Timestamp: time.Now().UTC(),
		Synthesis: []SynthesisSuggestion{
			{Topic: "test", Count: 3},
		},
	}
	data, _ := json.Marshal(suggestions)
	if err := os.WriteFile(filepath.Join(dir, "reflect-suggestions.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	loaded, err := LoadSuggestions()
	if err != nil {
		t.Fatalf("LoadSuggestions() error = %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadSuggestions() expected suggestions, got nil")
	}
	if len(loaded.Synthesis) != 1 {
		t.Errorf("Expected 1 synthesis suggestion, got %d", len(loaded.Synthesis))
	}
}

func TestSuggestionsPath(t *testing.T) {
	path := SuggestionsPath()
	if path == "" {
		t.Skip("Could not determine home directory")
	}

	// Path should end with expected filename
	if filepath.Base(path) != "reflect-suggestions.json" {
		t.Errorf("SuggestionsPath() base = %q, want 'reflect-suggestions.json'", filepath.Base(path))
	}

	// Path should be in .orch directory
	if filepath.Base(filepath.Dir(path)) != ".orch" {
		t.Errorf("SuggestionsPath() parent dir = %q, want '.orch'", filepath.Base(filepath.Dir(path)))
	}
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestFilterSynthesisSuggestions_RemovesArchivedOnlyInvestigations(t *testing.T) {
	projectDir := t.TempDir()

	activeDir := filepath.Join(projectDir, ".kb", "investigations")
	if err := os.MkdirAll(filepath.Join(activeDir, "simple"), 0755); err != nil {
		t.Fatalf("mkdir investigations: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(activeDir, "archived"), 0755); err != nil {
		t.Fatalf("mkdir archived: %v", err)
	}

	mustWrite := func(path string) {
		t.Helper()
		if err := os.WriteFile(path, []byte("# test\n"), 0644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	mustWrite(filepath.Join(activeDir, "active.md"))
	mustWrite(filepath.Join(activeDir, "simple", "simple-active.md"))
	mustWrite(filepath.Join(activeDir, "archived", "archived-only.md"))
	mustWrite(filepath.Join(activeDir, "archived", "archived-cluster.md"))

	input := []SynthesisSuggestion{
		{
			Topic:          "mixed",
			Count:          2,
			Investigations: []string{"active.md", "archived-only.md"},
		},
		{
			Topic:          "archived",
			Count:          1,
			Investigations: []string{"archived-cluster.md"},
		},
		{
			Topic:          "simple",
			Count:          1,
			Investigations: []string{"simple/simple-active.md"},
		},
	}

	filtered := filterSynthesisSuggestions(input, projectDir)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 topics after filtering, got %d", len(filtered))
	}

	if filtered[0].Topic != "mixed" {
		t.Fatalf("first topic = %q, want mixed", filtered[0].Topic)
	}
	if filtered[0].Count != 1 {
		t.Fatalf("mixed count = %d, want 1", filtered[0].Count)
	}
	if !reflect.DeepEqual(filtered[0].Investigations, []string{"active.md"}) {
		t.Fatalf("mixed investigations = %v, want [active.md]", filtered[0].Investigations)
	}

	if filtered[1].Topic != "simple" {
		t.Fatalf("second topic = %q, want simple", filtered[1].Topic)
	}
	if filtered[1].Count != 1 {
		t.Fatalf("simple count = %d, want 1", filtered[1].Count)
	}
}

func TestFilterSynthesisSuggestions_NoKBDirReturnsOriginal(t *testing.T) {
	projectDir := t.TempDir()

	input := []SynthesisSuggestion{
		{
			Topic:          "topic",
			Count:          2,
			Investigations: []string{"a.md", "b.md"},
		},
	}

	filtered := filterSynthesisSuggestions(input, projectDir)
	if !reflect.DeepEqual(filtered, input) {
		t.Fatalf("expected unchanged suggestions, got %#v", filtered)
	}
}
