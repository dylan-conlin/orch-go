package workspace

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFindByBeadsID(t *testing.T) {
	tests := []struct {
		name      string
		beadsID   string
		setup     func(t *testing.T, dir string)
		wantPath  bool
		wantAgent string
	}{
		{
			name:    "matches directory name containing beads ID",
			beadsID: "orch-go-3anf",
			setup: func(t *testing.T, dir string) {
				wsDir := filepath.Join(dir, ".orch", "workspace", "og-feat-something-orch-go-3anf-05jan-ab12")
				os.MkdirAll(wsDir, 0755)
			},
			wantPath:  true,
			wantAgent: "og-feat-something-orch-go-3anf-05jan-ab12",
		},
		{
			name:    "matches AGENT_MANIFEST.json beads_id",
			beadsID: "orch-go-9xyz",
			setup: func(t *testing.T, dir string) {
				wsDir := filepath.Join(dir, ".orch", "workspace", "og-feat-thing-05jan-ab12")
				os.MkdirAll(wsDir, 0755)
				os.WriteFile(filepath.Join(wsDir, "AGENT_MANIFEST.json"),
					[]byte(`{"beads_id":"orch-go-9xyz","skill":"feature-impl"}`), 0644)
			},
			wantPath:  true,
			wantAgent: "og-feat-thing-05jan-ab12",
		},
		{
			name:    "matches SPAWN_CONTEXT.md beads issue line",
			beadsID: "orch-go-7abc",
			setup: func(t *testing.T, dir string) {
				wsDir := filepath.Join(dir, ".orch", "workspace", "og-feat-other-05jan-cd34")
				os.MkdirAll(wsDir, 0755)
				os.WriteFile(filepath.Join(wsDir, "SPAWN_CONTEXT.md"),
					[]byte("## Context\nSpawned from beads issue: **orch-go-7abc**\n"), 0644)
			},
			wantPath:  true,
			wantAgent: "og-feat-other-05jan-cd34",
		},
		{
			name:    "no match returns empty",
			beadsID: "orch-go-none",
			setup: func(t *testing.T, dir string) {
				wsDir := filepath.Join(dir, ".orch", "workspace", "og-feat-unrelated-05jan-ef56")
				os.MkdirAll(wsDir, 0755)
			},
			wantPath:  false,
			wantAgent: "",
		},
		{
			name:    "skips archived directory",
			beadsID: "orch-go-skip",
			setup: func(t *testing.T, dir string) {
				os.MkdirAll(filepath.Join(dir, ".orch", "workspace", "archived"), 0755)
			},
			wantPath:  false,
			wantAgent: "",
		},
		{
			name:    "prefers workspace with SYNTHESIS.md on duplicate",
			beadsID: "orch-go-dup1",
			setup: func(t *testing.T, dir string) {
				ws1 := filepath.Join(dir, ".orch", "workspace", "og-feat-a-orch-go-dup1-05jan-aa11")
				ws2 := filepath.Join(dir, ".orch", "workspace", "og-feat-b-orch-go-dup1-05jan-bb22")
				os.MkdirAll(ws1, 0755)
				os.MkdirAll(ws2, 0755)
				os.WriteFile(filepath.Join(ws2, "SYNTHESIS.md"), []byte("# Synthesis"), 0644)
			},
			wantPath:  true,
			wantAgent: "og-feat-b-orch-go-dup1-05jan-bb22",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			os.MkdirAll(filepath.Join(tempDir, ".orch", "workspace"), 0755)
			tt.setup(t, tempDir)

			gotPath, gotAgent := FindByBeadsID(tempDir, tt.beadsID)
			if tt.wantPath && gotPath == "" {
				t.Errorf("FindByBeadsID() path = empty, want non-empty")
			}
			if !tt.wantPath && gotPath != "" {
				t.Errorf("FindByBeadsID() path = %q, want empty", gotPath)
			}
			if tt.wantAgent != "" && gotAgent != tt.wantAgent {
				t.Errorf("FindByBeadsID() agent = %q, want %q", gotAgent, tt.wantAgent)
			}
		})
	}
}

func TestFindByBeadsID_CrossRepoWorkspace(t *testing.T) {
	// Simulate cross-repo scenario: beads issue is orch-go-XXXX but workspace is in kb-cli
	// This verifies the building block that findWorkspaceByBeadsIDAcrossProjects uses.
	sourceProject := t.TempDir() // e.g. orch-go
	targetProject := t.TempDir() // e.g. kb-cli

	// Create workspace dirs
	os.MkdirAll(filepath.Join(sourceProject, ".orch", "workspace"), 0755)
	os.MkdirAll(filepath.Join(targetProject, ".orch", "workspace"), 0755)

	// Workspace is in TARGET project (created by --workdir spawn)
	wsDir := filepath.Join(targetProject, ".orch", "workspace", "kc-feat-add-model-09mar-ab12")
	os.MkdirAll(wsDir, 0755)
	os.WriteFile(filepath.Join(wsDir, "AGENT_MANIFEST.json"),
		[]byte(`{"beads_id":"orch-go-x1y2","skill":"feature-impl","project_dir":"`+targetProject+`"}`), 0644)
	os.WriteFile(filepath.Join(wsDir, "SYNTHESIS.md"), []byte("# Synthesis\nDone."), 0644)

	// Search SOURCE project → should NOT find it
	path, agent := FindByBeadsID(sourceProject, "orch-go-x1y2")
	if path != "" {
		t.Errorf("FindByBeadsID(sourceProject) should return empty, got %q", path)
	}
	if agent != "" {
		t.Errorf("FindByBeadsID(sourceProject) agent should be empty, got %q", agent)
	}

	// Search TARGET project → should find it
	path, agent = FindByBeadsID(targetProject, "orch-go-x1y2")
	if path == "" {
		t.Error("FindByBeadsID(targetProject) should find workspace")
	}
	if agent != "kc-feat-add-model-09mar-ab12" {
		t.Errorf("FindByBeadsID(targetProject) agent = %q, want kc-feat-add-model-09mar-ab12", agent)
	}
}

func TestFindByName(t *testing.T) {
	tempDir := t.TempDir()
	wsDir := filepath.Join(tempDir, ".orch", "workspace", "og-feat-test-05jan")
	os.MkdirAll(wsDir, 0755)

	found := FindByName(tempDir, "og-feat-test-05jan")
	if found == "" {
		t.Error("FindByName() should find existing workspace")
	}

	notFound := FindByName(tempDir, "og-does-not-exist")
	if notFound != "" {
		t.Errorf("FindByName() should return empty for nonexistent, got %q", notFound)
	}
}

func TestIsOrchestrator(t *testing.T) {
	tempDir := t.TempDir()

	wsOrch := filepath.Join(tempDir, "ws-orch")
	os.MkdirAll(wsOrch, 0755)
	os.WriteFile(filepath.Join(wsOrch, ".orchestrator"), []byte(""), 0644)

	wsMetaOrch := filepath.Join(tempDir, "ws-meta-orch")
	os.MkdirAll(wsMetaOrch, 0755)
	os.WriteFile(filepath.Join(wsMetaOrch, ".meta-orchestrator"), []byte(""), 0644)

	wsWorker := filepath.Join(tempDir, "ws-worker")
	os.MkdirAll(wsWorker, 0755)

	if !IsOrchestrator(wsOrch) {
		t.Error("IsOrchestrator should return true for .orchestrator marker")
	}
	if !IsOrchestrator(wsMetaOrch) {
		t.Error("IsOrchestrator should return true for .meta-orchestrator marker")
	}
	if IsOrchestrator(wsWorker) {
		t.Error("IsOrchestrator should return false for worker workspace")
	}
}

func TestHasSessionHandoff(t *testing.T) {
	tempDir := t.TempDir()

	wsWithHandoff := filepath.Join(tempDir, "ws-with")
	os.MkdirAll(wsWithHandoff, 0755)
	os.WriteFile(filepath.Join(wsWithHandoff, "SESSION_HANDOFF.md"), []byte("# Handoff"), 0644)

	wsWithout := filepath.Join(tempDir, "ws-without")
	os.MkdirAll(wsWithout, 0755)

	if !HasSessionHandoff(wsWithHandoff) {
		t.Error("HasSessionHandoff should return true when file exists")
	}
	if HasSessionHandoff(wsWithout) {
		t.Error("HasSessionHandoff should return false when file missing")
	}
}

func TestSpawnTime(t *testing.T) {
	tempDir := t.TempDir()

	// Write a known spawn time
	os.WriteFile(filepath.Join(tempDir, ".spawn_time"), []byte("1704067200000000000"), 0644)
	got := SpawnTime(tempDir)
	if got != 1704067200000000000 {
		t.Errorf("SpawnTime() = %d, want 1704067200000000000", got)
	}

	// Missing file returns 0
	emptyDir := t.TempDir()
	if SpawnTime(emptyDir) != 0 {
		t.Error("SpawnTime() should return 0 for missing file")
	}
}

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLen  int
		wantHas  []string
		wantNot  []string
	}{
		{
			name:    "standard workspace name",
			input:   "og-inv-skillc-deploy-06jan-ed96",
			wantLen: 2,
			wantHas: []string{"skillc", "deploy"},
			wantNot: []string{"og", "inv", "06jan", "ed96"},
		},
		{
			name:    "short name returns nil",
			input:   "og-inv",
			wantLen: 0,
		},
		{
			name:    "skips common prefixes",
			input:   "og-feat-impl-something-useful-05mar-ab12",
			wantHas: []string{"something", "useful"},
			wantNot: []string{"og", "feat", "impl"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractKeywords(tt.input)
			if tt.wantLen > 0 && len(got) != tt.wantLen {
				t.Errorf("ExtractKeywords(%q) len = %d, want %d; got %v", tt.input, len(got), tt.wantLen, got)
			}
			gotSet := make(map[string]bool)
			for _, k := range got {
				gotSet[k] = true
			}
			for _, want := range tt.wantHas {
				if !gotSet[want] {
					t.Errorf("ExtractKeywords(%q) missing keyword %q; got %v", tt.input, want, got)
				}
			}
			for _, notWant := range tt.wantNot {
				if gotSet[notWant] {
					t.Errorf("ExtractKeywords(%q) should not contain %q; got %v", tt.input, notWant, got)
				}
			}
		})
	}
}

func TestExtractDate(t *testing.T) {
	currentYear := time.Now().Year()

	tests := []struct {
		name      string
		input     string
		wantZero  bool
		wantMonth time.Month
		wantDay   int
	}{
		{
			name:      "standard date suffix",
			input:     "og-feat-something-24dec",
			wantMonth: time.December,
			wantDay:   24,
		},
		{
			name:      "single digit day",
			input:     "og-feat-something-5jan",
			wantMonth: time.January,
			wantDay:   5,
		},
		{
			name:     "no date suffix (hash)",
			input:    "og-feat-something-ab12",
			wantZero: true,
		},
		{
			name:     "empty string",
			input:    "",
			wantZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractDate(tt.input)
			if tt.wantZero {
				if !got.IsZero() {
					t.Errorf("ExtractDate(%q) = %v, want zero time", tt.input, got)
				}
				return
			}
			if got.IsZero() {
				t.Errorf("ExtractDate(%q) = zero time, want non-zero", tt.input)
				return
			}
			if got.Month() != tt.wantMonth {
				t.Errorf("ExtractDate(%q) month = %v, want %v", tt.input, got.Month(), tt.wantMonth)
			}
			if got.Day() != tt.wantDay {
				t.Errorf("ExtractDate(%q) day = %d, want %d", tt.input, got.Day(), tt.wantDay)
			}
			if got.Year() != currentYear && got.Year() != currentYear-1 {
				t.Errorf("ExtractDate(%q) year = %d, want %d or %d", tt.input, got.Year(), currentYear, currentYear-1)
			}
		})
	}
}
