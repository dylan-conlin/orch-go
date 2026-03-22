// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
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
			Synthesis:  []SynthesisSuggestion{{Topic: "test", Count: 3}},
			Promote:    []PromoteSuggestion{{ID: "1", Content: "test"}},
			Stale:      []StaleSuggestion{{Path: "test.md", Age: 10}},
			Drift:      []DriftSuggestion{{ID: "1", Content: "test"}},
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
			Synthesis:  []SynthesisSuggestion{{Topic: "a"}},
			Promote:    []PromoteSuggestion{{ID: "1"}},
			Stale:      []StaleSuggestion{{Path: "x"}},
			Drift:      []DriftSuggestion{{ID: "2"}},
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
		{"drift", &ReflectSuggestions{
			Drift: []DriftSuggestion{{ID: "1", Content: "test"}},
		}, []string{"drift"}},
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

func TestReflectSuggestions_HasSuggestions_DefectClass(t *testing.T) {
	s := &ReflectSuggestions{
		DefectClass: []DefectClassSuggestion{
			{DefectClass: "configuration-drift", Count: 5, WindowDays: 30},
		},
	}
	if !s.HasSuggestions() {
		t.Error("HasSuggestions() = false, want true when DefectClass present")
	}
}

func TestReflectSuggestions_TotalCount_IncludesDefectClass(t *testing.T) {
	s := &ReflectSuggestions{
		Synthesis: []SynthesisSuggestion{{Topic: "a"}},
		DefectClass: []DefectClassSuggestion{
			{DefectClass: "configuration-drift", Count: 5},
			{DefectClass: "unbounded-growth", Count: 3},
		},
	}
	got := s.TotalCount()
	if got != 3 {
		t.Errorf("TotalCount() = %d, want 3 (1 synthesis + 2 defect-class)", got)
	}
}

func TestReflectSuggestions_Summary_DefectClass(t *testing.T) {
	s := &ReflectSuggestions{
		DefectClass: []DefectClassSuggestion{
			{DefectClass: "configuration-drift", Count: 5},
			{DefectClass: "unbounded-growth", Count: 3},
		},
	}
	got := s.Summary()
	want := "2 defect-class patterns"
	if !containsSubstring(got, want) {
		t.Errorf("Summary() = %q, want to contain %q", got, want)
	}
}

func TestReflectSuggestions_Summary_DefectClassWithOtherTypes(t *testing.T) {
	s := &ReflectSuggestions{
		Synthesis: []SynthesisSuggestion{{Topic: "a"}},
		DefectClass: []DefectClassSuggestion{
			{DefectClass: "configuration-drift", Count: 5},
		},
	}
	got := s.Summary()
	if !containsSubstring(got, "1 synthesis") {
		t.Errorf("Summary() = %q, want to contain '1 synthesis'", got)
	}
	if !containsSubstring(got, "1 defect-class pattern") {
		t.Errorf("Summary() = %q, want to contain '1 defect-class pattern'", got)
	}
}

func TestDefectClassSuggestion_JSONRoundTrip(t *testing.T) {
	input := `{"defect_class":[{"defect_class":"configuration-drift","count":5,"window_days":30,"investigations":["inv-a","inv-b"],"suggestion":"Consider synthesis"}]}`

	var raw kbReflectOutput
	if err := json.Unmarshal([]byte(input), &raw); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if len(raw.DefectClass) != 1 {
		t.Fatalf("Expected 1 defect-class entry, got %d", len(raw.DefectClass))
	}
	dc := raw.DefectClass[0]
	if dc.DefectClass != "configuration-drift" {
		t.Errorf("DefectClass = %q, want 'configuration-drift'", dc.DefectClass)
	}
	if dc.Count != 5 {
		t.Errorf("Count = %d, want 5", dc.Count)
	}
	if dc.WindowDays != 30 {
		t.Errorf("WindowDays = %d, want 30", dc.WindowDays)
	}
	if len(dc.Investigations) != 2 {
		t.Errorf("Investigations len = %d, want 2", len(dc.Investigations))
	}
}

func TestDefectClassSuggestion_IssueCreatedField(t *testing.T) {
	input := `{"defect_class":[{"defect_class":"unbounded-growth","count":3,"window_days":30,"investigations":["a"],"suggestion":"fix it","issue_created":true}]}`

	var raw kbReflectOutput
	if err := json.Unmarshal([]byte(input), &raw); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if !raw.DefectClass[0].IssueCreated {
		t.Error("IssueCreated = false, want true")
	}
}

func TestSaveSuggestions_IncludesDefectClass(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	suggestions := &ReflectSuggestions{
		Timestamp: time.Now().UTC(),
		DefectClass: []DefectClassSuggestion{
			{DefectClass: "configuration-drift", Count: 5, WindowDays: 30, Investigations: []string{"inv-a"}, Suggestion: "Review pattern"},
		},
	}

	if err := SaveSuggestions(suggestions); err != nil {
		t.Fatalf("SaveSuggestions() error = %v", err)
	}

	loaded, err := LoadSuggestions()
	if err != nil {
		t.Fatalf("LoadSuggestions() error = %v", err)
	}
	if len(loaded.DefectClass) != 1 {
		t.Fatalf("Expected 1 defect-class, got %d", len(loaded.DefectClass))
	}
	dc := loaded.DefectClass[0]
	if dc.DefectClass != "configuration-drift" {
		t.Errorf("DefectClass = %q, want 'configuration-drift'", dc.DefectClass)
	}
	if dc.Count != 5 {
		t.Errorf("Count = %d, want 5", dc.Count)
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
