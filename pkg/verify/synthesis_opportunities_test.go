package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectSynthesisOpportunities_NoKBDir(t *testing.T) {
	// Create a temp directory without .kb
	tempDir := t.TempDir()

	opportunities, err := DetectSynthesisOpportunities(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opportunities.HasOpportunities() {
		t.Error("expected no opportunities for directory without .kb")
	}
}

func TestDetectSynthesisOpportunities_EmptyInvestigations(t *testing.T) {
	// Create a temp directory with empty .kb structure
	tempDir := t.TempDir()
	kbDir := filepath.Join(tempDir, ".kb")
	os.MkdirAll(filepath.Join(kbDir, "investigations"), 0755)
	os.MkdirAll(filepath.Join(kbDir, "guides"), 0755)
	os.MkdirAll(filepath.Join(kbDir, "decisions"), 0755)

	opportunities, err := DetectSynthesisOpportunities(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opportunities.HasOpportunities() {
		t.Error("expected no opportunities for empty investigations")
	}
}

func TestDetectSynthesisOpportunities_BelowThreshold(t *testing.T) {
	// Create a temp directory with 2 daemon investigations (below threshold)
	tempDir := t.TempDir()
	kbDir := filepath.Join(tempDir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	os.MkdirAll(invDir, 0755)
	os.MkdirAll(filepath.Join(kbDir, "guides"), 0755)
	os.MkdirAll(filepath.Join(kbDir, "decisions"), 0755)

	// Create 2 daemon investigations
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-daemon-first.md"), []byte("# First"), 0644)
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-daemon-second.md"), []byte("# Second"), 0644)

	opportunities, err := DetectSynthesisOpportunities(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opportunities.HasOpportunities() {
		t.Error("expected no opportunities when below threshold (2 < 3)")
	}
}

func TestDetectSynthesisOpportunities_MeetsThreshold(t *testing.T) {
	// Create a temp directory with 3 daemon investigations (meets threshold)
	tempDir := t.TempDir()
	kbDir := filepath.Join(tempDir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	os.MkdirAll(invDir, 0755)
	os.MkdirAll(filepath.Join(kbDir, "guides"), 0755)
	os.MkdirAll(filepath.Join(kbDir, "decisions"), 0755)

	// Create 3 daemon investigations
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-daemon-first.md"), []byte("# First"), 0644)
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-daemon-second.md"), []byte("# Second"), 0644)
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-daemon-third.md"), []byte("# Third"), 0644)

	opportunities, err := DetectSynthesisOpportunities(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !opportunities.HasOpportunities() {
		t.Fatal("expected opportunities when threshold met")
	}

	if len(opportunities.Opportunities) != 1 {
		t.Fatalf("expected 1 opportunity, got %d", len(opportunities.Opportunities))
	}

	opp := opportunities.Opportunities[0]
	if opp.Topic != "daemon" {
		t.Errorf("expected topic 'daemon', got '%s'", opp.Topic)
	}
	if opp.InvestigationCount != 3 {
		t.Errorf("expected 3 investigations, got %d", opp.InvestigationCount)
	}
}

func TestDetectSynthesisOpportunities_WithExistingGuide(t *testing.T) {
	// Create a temp directory with 3 daemon investigations but a daemon guide exists
	tempDir := t.TempDir()
	kbDir := filepath.Join(tempDir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	guidesDir := filepath.Join(kbDir, "guides")
	os.MkdirAll(invDir, 0755)
	os.MkdirAll(guidesDir, 0755)
	os.MkdirAll(filepath.Join(kbDir, "decisions"), 0755)

	// Create 3 daemon investigations
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-daemon-first.md"), []byte("# First"), 0644)
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-daemon-second.md"), []byte("# Second"), 0644)
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-daemon-third.md"), []byte("# Third"), 0644)

	// Create a daemon guide (should suppress the opportunity)
	os.WriteFile(filepath.Join(guidesDir, "daemon.md"), []byte("# Daemon Guide"), 0644)

	opportunities, err := DetectSynthesisOpportunities(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opportunities.HasOpportunities() {
		t.Error("expected no opportunities when guide exists for topic")
	}
}

func TestDetectSynthesisOpportunities_MultipleTopics(t *testing.T) {
	// Create a temp directory with multiple topics
	tempDir := t.TempDir()
	kbDir := filepath.Join(tempDir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	os.MkdirAll(invDir, 0755)
	os.MkdirAll(filepath.Join(kbDir, "guides"), 0755)
	os.MkdirAll(filepath.Join(kbDir, "decisions"), 0755)

	// Create 4 dashboard investigations
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-dashboard-first.md"), []byte("# First"), 0644)
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-dashboard-second.md"), []byte("# Second"), 0644)
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-dashboard-third.md"), []byte("# Third"), 0644)
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-dashboard-fourth.md"), []byte("# Fourth"), 0644)

	// Create 3 spawn investigations
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-spawn-first.md"), []byte("# First"), 0644)
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-spawn-second.md"), []byte("# Second"), 0644)
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-spawn-third.md"), []byte("# Third"), 0644)

	opportunities, err := DetectSynthesisOpportunities(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !opportunities.HasOpportunities() {
		t.Fatal("expected opportunities")
	}

	if len(opportunities.Opportunities) != 2 {
		t.Fatalf("expected 2 opportunities (dashboard, spawn), got %d", len(opportunities.Opportunities))
	}

	// Should be sorted by count (dashboard=4 first, spawn=3 second)
	if opportunities.Opportunities[0].Topic != "dashboard" {
		t.Errorf("expected first topic 'dashboard', got '%s'", opportunities.Opportunities[0].Topic)
	}
	if opportunities.Opportunities[0].InvestigationCount != 4 {
		t.Errorf("expected 4 dashboard investigations, got %d", opportunities.Opportunities[0].InvestigationCount)
	}
}

func TestDetectSynthesisOpportunities_VariousTypes(t *testing.T) {
	// Test that various investigation types are recognized
	tempDir := t.TempDir()
	kbDir := filepath.Join(tempDir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	os.MkdirAll(invDir, 0755)
	os.MkdirAll(filepath.Join(kbDir, "guides"), 0755)
	os.MkdirAll(filepath.Join(kbDir, "decisions"), 0755)

	// Create investigations with different type prefixes
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-status-command.md"), []byte("# inv"), 0644)
	os.WriteFile(filepath.Join(invDir, "2025-12-20-design-status-display.md"), []byte("# design"), 0644)
	os.WriteFile(filepath.Join(invDir, "2025-12-20-debug-status-bug.md"), []byte("# debug"), 0644)

	opportunities, err := DetectSynthesisOpportunities(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !opportunities.HasOpportunities() {
		t.Fatal("expected opportunities for status topic")
	}

	found := false
	for _, opp := range opportunities.Opportunities {
		if opp.Topic == "status" {
			found = true
			if opp.InvestigationCount != 3 {
				t.Errorf("expected 3 status investigations, got %d", opp.InvestigationCount)
			}
		}
	}
	if !found {
		t.Error("expected status topic in opportunities")
	}
}

func TestDetectSynthesisOpportunities_SimpleSubdirectory(t *testing.T) {
	// Test that investigations in simple/ subdirectory are counted
	tempDir := t.TempDir()
	kbDir := filepath.Join(tempDir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	simpleDir := filepath.Join(invDir, "simple")
	os.MkdirAll(invDir, 0755)
	os.MkdirAll(simpleDir, 0755)
	os.MkdirAll(filepath.Join(kbDir, "guides"), 0755)
	os.MkdirAll(filepath.Join(kbDir, "decisions"), 0755)

	// Create 2 in main, 1 in simple
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-opencode-first.md"), []byte("# First"), 0644)
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-opencode-second.md"), []byte("# Second"), 0644)
	os.WriteFile(filepath.Join(simpleDir, "2025-12-20-inv-opencode-third.md"), []byte("# Third"), 0644)

	opportunities, err := DetectSynthesisOpportunities(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !opportunities.HasOpportunities() {
		t.Fatal("expected opportunities when combining main and simple investigations")
	}

	found := false
	for _, opp := range opportunities.Opportunities {
		if opp.Topic == "opencode" {
			found = true
			if opp.InvestigationCount != 3 {
				t.Errorf("expected 3 opencode investigations (2 main + 1 simple), got %d", opp.InvestigationCount)
			}
		}
	}
	if !found {
		t.Error("expected opencode topic in opportunities")
	}
}

func TestHasOpportunities(t *testing.T) {
	tests := []struct {
		name     string
		input    *SynthesisOpportunities
		expected bool
	}{
		{"nil", nil, false},
		{"empty", &SynthesisOpportunities{}, false},
		{"empty opportunities", &SynthesisOpportunities{Opportunities: []SynthesisOpportunity{}}, false},
		{"has opportunities", &SynthesisOpportunities{
			Opportunities: []SynthesisOpportunity{{Topic: "test", InvestigationCount: 3}},
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.HasOpportunities()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
