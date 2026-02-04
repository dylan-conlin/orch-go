package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectOrphanInvestigations_NoKBDir(t *testing.T) {
	// Create a temp directory without .kb
	tempDir := t.TempDir()

	orphans, err := DetectOrphanInvestigations(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if orphans.HasOrphans() {
		t.Error("expected no orphans for directory without .kb")
	}
}

func TestDetectOrphanInvestigations_EmptyInvestigations(t *testing.T) {
	// Create a temp directory with empty .kb structure
	tempDir := t.TempDir()
	kbDir := filepath.Join(tempDir, ".kb")
	os.MkdirAll(filepath.Join(kbDir, "investigations"), 0755)

	orphans, err := DetectOrphanInvestigations(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if orphans.HasOrphans() {
		t.Error("expected no orphans for empty investigations")
	}
}

func TestDetectOrphanInvestigations_SingleInvestigation(t *testing.T) {
	// Create a temp directory with single investigation (no peers = not orphan)
	tempDir := t.TempDir()
	kbDir := filepath.Join(tempDir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	os.MkdirAll(invDir, 0755)

	// Create single daemon investigation
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-daemon-first.md"), []byte("# First\n\n**Supersedes:** N/A"), 0644)

	orphans, err := DetectOrphanInvestigations(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if orphans.HasOrphans() {
		t.Error("expected no orphans for single investigation (no peers)")
	}
}

func TestDetectOrphanInvestigations_MultipleInvestigationsNoCitation(t *testing.T) {
	// Create multiple investigations on same topic, none citing each other
	tempDir := t.TempDir()
	kbDir := filepath.Join(tempDir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	os.MkdirAll(invDir, 0755)

	// Create 3 daemon investigations with no citations
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-daemon-first.md"), []byte("# First\n\n**Supersedes:** N/A"), 0644)
	os.WriteFile(filepath.Join(invDir, "2025-12-21-inv-daemon-second.md"), []byte("# Second\n\n**Supersedes:** N/A"), 0644)
	os.WriteFile(filepath.Join(invDir, "2025-12-22-inv-daemon-third.md"), []byte("# Third\n\n**Supersedes:** N/A"), 0644)

	orphans, err := DetectOrphanInvestigations(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !orphans.HasOrphans() {
		t.Fatal("expected orphans for multiple investigations without citations")
	}

	// All 3 should be orphans (each has 2 peers it doesn't cite)
	if len(orphans.Orphans) != 3 {
		t.Errorf("expected 3 orphans, got %d", len(orphans.Orphans))
	}

	// Verify topic is detected correctly
	for _, orphan := range orphans.Orphans {
		if orphan.Topic != "daemon" {
			t.Errorf("expected topic 'daemon', got %q", orphan.Topic)
		}
	}
}

func TestDetectOrphanInvestigations_WithInTextCitation(t *testing.T) {
	// Create investigations where one cites another via in-text reference
	tempDir := t.TempDir()
	kbDir := filepath.Join(tempDir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	os.MkdirAll(invDir, 0755)

	// First investigation has no citation
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-daemon-first.md"), []byte("# First\n\n**Supersedes:** N/A"), 0644)
	
	// Second investigation cites first via path
	citedContent := "# Second\n\nAs found in .kb/investigations/2025-12-20-inv-daemon-first.md, the issue is clear."
	os.WriteFile(filepath.Join(invDir, "2025-12-21-inv-daemon-second.md"), []byte(citedContent), 0644)

	orphans, err := DetectOrphanInvestigations(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only first should be orphan (has peer but doesn't cite)
	// Second has in-text citation so NOT an orphan
	if len(orphans.Orphans) != 1 {
		t.Errorf("expected 1 orphan, got %d", len(orphans.Orphans))
	}

	if orphans.Orphans[0].Path != filepath.Join(invDir, "2025-12-20-inv-daemon-first.md") {
		t.Errorf("expected first investigation to be orphan, got %s", orphans.Orphans[0].Path)
	}
}

func TestDetectOrphanInvestigations_WithFormalSupersedes(t *testing.T) {
	// Create investigations where one formally supersedes another
	tempDir := t.TempDir()
	kbDir := filepath.Join(tempDir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	os.MkdirAll(invDir, 0755)

	// First investigation
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-daemon-first.md"), []byte("# First\n\n**Supersedes:** N/A"), 0644)
	
	// Second investigation formally supersedes first
	supersedingContent := "# Second\n\n**Supersedes:** .kb/investigations/2025-12-20-inv-daemon-first.md"
	os.WriteFile(filepath.Join(invDir, "2025-12-21-inv-daemon-second.md"), []byte(supersedingContent), 0644)

	orphans, err := DetectOrphanInvestigations(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only first should be orphan
	if len(orphans.Orphans) != 1 {
		t.Errorf("expected 1 orphan, got %d", len(orphans.Orphans))
	}
}

func TestDetectOrphanInvestigations_WithDateCitation(t *testing.T) {
	// Create investigations where one cites another via date reference
	tempDir := t.TempDir()
	kbDir := filepath.Join(tempDir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	os.MkdirAll(invDir, 0755)

	// First investigation
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-daemon-first.md"), []byte("# First"), 0644)
	
	// Second investigation cites first via date reference
	citedContent := "# Second\n\nThe Dec 20 investigation found the root cause."
	os.WriteFile(filepath.Join(invDir, "2025-12-21-inv-daemon-second.md"), []byte(citedContent), 0644)

	orphans, err := DetectOrphanInvestigations(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only first should be orphan (second has date citation)
	if len(orphans.Orphans) != 1 {
		t.Errorf("expected 1 orphan, got %d", len(orphans.Orphans))
	}
}

func TestDetectOrphanInvestigations_DifferentTopics(t *testing.T) {
	// Create investigations on different topics (not orphans of each other)
	tempDir := t.TempDir()
	kbDir := filepath.Join(tempDir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	os.MkdirAll(invDir, 0755)

	// Daemon investigation
	os.WriteFile(filepath.Join(invDir, "2025-12-20-inv-daemon-first.md"), []byte("# Daemon"), 0644)
	
	// Dashboard investigation
	os.WriteFile(filepath.Join(invDir, "2025-12-21-inv-dashboard-first.md"), []byte("# Dashboard"), 0644)

	orphans, err := DetectOrphanInvestigations(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Neither should be orphan (no same-topic peers)
	if orphans.HasOrphans() {
		t.Error("expected no orphans when investigations are on different topics")
	}
}

func TestHasInTextCitations(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "path reference",
			content:  "Found in .kb/investigations/2025-12-20-inv-test.md",
			expected: true,
		},
		{
			name:     "date reference",
			content:  "The Jan 26 investigation showed",
			expected: true,
		},
		{
			name:     "prior investigation language",
			content:  "From prior investigation, we learned",
			expected: true,
		},
		{
			name:     "no citation",
			content:  "This is a new investigation with no references",
			expected: false,
		},
		{
			name:     "earlier investigation",
			content:  "Earlier investigation confirmed this",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasInTextCitations(tt.content)
			if result != tt.expected {
				t.Errorf("hasInTextCitations(%q) = %v, want %v", tt.content, result, tt.expected)
			}
		})
	}
}

func TestIsEmptyLineageValue(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"", true},
		{"N/A", true},
		{"n/a", true},
		{"None", true},
		{"none", true},
		{"TBD", true},
		{"[Path to artifact this replaces, if applicable]", true},
		{"[some placeholder]", true},
		{".kb/investigations/2025-12-20-inv-test.md", false},
		{"extends prior findings", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := isEmptyLineageValue(tt.value)
			if result != tt.expected {
				t.Errorf("isEmptyLineageValue(%q) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestExtractPrimaryTopic(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{".kb/investigations/2025-12-20-inv-daemon-first.md", "daemon"},
		{".kb/investigations/2025-12-20-inv-dashboard-test.md", "dashboard"},
		{".kb/investigations/2025-12-20-inv-spawn-headless.md", "spawn"},
		{".kb/investigations/2025-12-20-inv-unknown-topic.md", "unknown"},
		{".kb/investigations/2025-12-20-research-opencode-api.md", "opencode"},
		{".kb/investigations/invalid-format.md", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := extractPrimaryTopic(tt.path)
			if result != tt.expected {
				t.Errorf("extractPrimaryTopic(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}
