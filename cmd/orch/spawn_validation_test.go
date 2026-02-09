package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveTriageBypass(t *testing.T) {
	old := spawnBypassTriage
	defer func() {
		spawnBypassTriage = old
	}()

	tests := []struct {
		name   string
		flag   bool
		env    string
		wantOK bool
		wantBy string
	}{
		{name: "flag bypass", flag: true, env: "", wantOK: true, wantBy: "flag"},
		{name: "env bypass numeric", flag: false, env: "1", wantOK: true, wantBy: "env"},
		{name: "env bypass true", flag: false, env: "true", wantOK: true, wantBy: "env"},
		{name: "env bypass uppercase true", flag: false, env: "TRUE", wantOK: true, wantBy: "env"},
		{name: "env bypass yes", flag: false, env: "yes", wantOK: true, wantBy: "env"},
		{name: "invalid env value", flag: false, env: "0", wantOK: false, wantBy: ""},
		{name: "none", flag: false, env: "", wantOK: false, wantBy: ""},
		{name: "flag wins over env", flag: true, env: "0", wantOK: true, wantBy: "flag"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(triageBypassEnvVar, tt.env)
			spawnBypassTriage = tt.flag

			ok, by := resolveTriageBypass()
			if ok != tt.wantOK {
				t.Fatalf("resolveTriageBypass() ok = %v, want %v", ok, tt.wantOK)
			}
			if by != tt.wantBy {
				t.Fatalf("resolveTriageBypass() source = %q, want %q", by, tt.wantBy)
			}
		})
	}
}

func TestResolveHotspotSuppression(t *testing.T) {
	old := spawnAcknowledgeHotspot
	defer func() {
		spawnAcknowledgeHotspot = old
	}()

	tests := []struct {
		name   string
		flag   bool
		env    string
		wantOK bool
		wantBy string
	}{
		{name: "flag suppression", flag: true, env: "", wantOK: true, wantBy: "flag"},
		{name: "env suppression numeric", flag: false, env: "1", wantOK: true, wantBy: "env"},
		{name: "env suppression true", flag: false, env: "true", wantOK: true, wantBy: "env"},
		{name: "env suppression uppercase true", flag: false, env: "TRUE", wantOK: true, wantBy: "env"},
		{name: "env suppression yes", flag: false, env: "yes", wantOK: true, wantBy: "env"},
		{name: "invalid env value", flag: false, env: "0", wantOK: false, wantBy: ""},
		{name: "none", flag: false, env: "", wantOK: false, wantBy: ""},
		{name: "flag wins over env", flag: true, env: "0", wantOK: true, wantBy: "flag"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(hotspotSuppressEnvVar, tt.env)
			spawnAcknowledgeHotspot = tt.flag

			ok, by := resolveHotspotSuppression()
			if ok != tt.wantOK {
				t.Fatalf("resolveHotspotSuppression() ok = %v, want %v", ok, tt.wantOK)
			}
			if by != tt.wantBy {
				t.Fatalf("resolveHotspotSuppression() source = %q, want %q", by, tt.wantBy)
			}
		})
	}
}

func TestParseDecisionFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    *DecisionFrontmatter
		wantErr bool
	}{
		{
			name: "valid frontmatter with blocks",
			content: `---
blocks:
  - keywords: ["test", "sample"]
    patterns: ["**/test/**"]
---

# Decision Title

Content here.`,
			want: &DecisionFrontmatter{
				Blocks: []DecisionBlock{
					{
						Keywords: []string{"test", "sample"},
						Patterns: []string{"**/test/**"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no frontmatter",
			content: `# Decision Title

Content here.`,
			want:    nil,
			wantErr: false,
		},
		{
			name: "empty frontmatter",
			content: `---
---

# Decision Title

Content here.`,
			want:    &DecisionFrontmatter{},
			wantErr: false,
		},
		{
			name: "multiple blocks",
			content: `---
blocks:
  - keywords: ["test"]
  - keywords: ["sample"]
    patterns: ["**/*.test.ts"]
---

# Decision Title`,
			want: &DecisionFrontmatter{
				Blocks: []DecisionBlock{
					{Keywords: []string{"test"}},
					{
						Keywords: []string{"sample"},
						Patterns: []string{"**/*.test.ts"},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDecisionFrontmatter(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDecisionFrontmatter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want == nil && got != nil {
				t.Errorf("parseDecisionFrontmatter() = %v, want nil", got)
				return
			}
			if tt.want != nil && got == nil {
				t.Errorf("parseDecisionFrontmatter() = nil, want %v", tt.want)
				return
			}
			if tt.want != nil && got != nil {
				if len(got.Blocks) != len(tt.want.Blocks) {
					t.Errorf("parseDecisionFrontmatter() blocks count = %d, want %d", len(got.Blocks), len(tt.want.Blocks))
					return
				}
				for i := range tt.want.Blocks {
					if len(got.Blocks[i].Keywords) != len(tt.want.Blocks[i].Keywords) {
						t.Errorf("parseDecisionFrontmatter() block %d keywords count = %d, want %d", i, len(got.Blocks[i].Keywords), len(tt.want.Blocks[i].Keywords))
					}
					if len(got.Blocks[i].Patterns) != len(tt.want.Blocks[i].Patterns) {
						t.Errorf("parseDecisionFrontmatter() block %d patterns count = %d, want %d", i, len(got.Blocks[i].Patterns), len(tt.want.Blocks[i].Patterns))
					}
				}
			}
		})
	}
}

func TestExtractDecisionInfo(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantTitle   string
		wantSummary string
	}{
		{
			name: "basic decision",
			content: `# Decision: Test Decision

This is the first paragraph.
It continues here.

## Context

More content.`,
			wantTitle:   "Decision: Test Decision",
			wantSummary: "This is the first paragraph. It continues here.",
		},
		{
			name: "decision with frontmatter",
			content: `---
blocks:
  - keywords: ["test"]
---

# Decision: With Frontmatter

Summary paragraph here.

## Context`,
			wantTitle:   "Decision: With Frontmatter",
			wantSummary: "Summary paragraph here.",
		},
		{
			name: "decision with empty lines",
			content: `# Decision Title


First paragraph after empty lines.

Second paragraph.`,
			wantTitle:   "Decision Title",
			wantSummary: "First paragraph after empty lines.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTitle, gotSummary := extractDecisionInfo(tt.content)
			if gotTitle != tt.wantTitle {
				t.Errorf("extractDecisionInfo() title = %q, want %q", gotTitle, tt.wantTitle)
			}
			if gotSummary != tt.wantSummary {
				t.Errorf("extractDecisionInfo() summary = %q, want %q", gotSummary, tt.wantSummary)
			}
		})
	}
}

func TestFindBlockingDecisions(t *testing.T) {
	// Create temp directory with test decisions
	tmpDir := t.TempDir()
	kbDir := filepath.Join(tmpDir, ".kb", "decisions")
	if err := os.MkdirAll(kbDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test decision files
	decision1 := `---
blocks:
  - keywords: ["coaching plugin", "worker detection"]
---

# Decision: Test Decision 1

This blocks coaching plugin work.
`
	if err := os.WriteFile(filepath.Join(kbDir, "2026-01-28-test-decision-1.md"), []byte(decision1), 0644); err != nil {
		t.Fatal(err)
	}

	decision2 := `# Decision: No Blocks

This decision has no blocks frontmatter.
`
	if err := os.WriteFile(filepath.Join(kbDir, "2026-01-28-test-decision-2.md"), []byte(decision2), 0644); err != nil {
		t.Fatal(err)
	}

	decision3 := `---
blocks:
  - patterns: ["**/api/**"]
---

# Decision: Pattern Block

This blocks API changes.
`
	if err := os.WriteFile(filepath.Join(kbDir, "2026-01-28-test-decision-3.md"), []byte(decision3), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		task          string
		wantConflicts int
		wantDecisions []string
	}{
		{
			name:          "matches keyword",
			task:          "fix coaching plugin worker detection",
			wantConflicts: 1,
			wantDecisions: []string{"2026-01-28-test-decision-1"},
		},
		{
			name:          "no match",
			task:          "implement new feature",
			wantConflicts: 0,
			wantDecisions: nil,
		},
		{
			name:          "matches pattern",
			task:          "update api endpoint",
			wantConflicts: 1,
			wantDecisions: []string{"2026-01-28-test-decision-3"},
		},
		{
			name:          "partial keyword match",
			task:          "coaching improvements",
			wantConflicts: 1,
			wantDecisions: []string{"2026-01-28-test-decision-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conflicts, err := findBlockingDecisions(tt.task, tmpDir)
			if err != nil {
				t.Errorf("findBlockingDecisions() error = %v", err)
				return
			}
			if len(conflicts) != tt.wantConflicts {
				t.Errorf("findBlockingDecisions() conflicts = %d, want %d", len(conflicts), tt.wantConflicts)
				return
			}
			if tt.wantDecisions != nil {
				for i, wantID := range tt.wantDecisions {
					if conflicts[i].DecisionID != wantID {
						t.Errorf("findBlockingDecisions() conflict[%d] = %s, want %s", i, conflicts[i].DecisionID, wantID)
					}
				}
			}
		})
	}
}

func TestCheckDecisionConflicts(t *testing.T) {
	// Create temp directory with test decision
	tmpDir := t.TempDir()
	kbDir := filepath.Join(tmpDir, ".kb", "decisions")
	if err := os.MkdirAll(kbDir, 0755); err != nil {
		t.Fatal(err)
	}

	decision := `---
blocks:
  - keywords: ["blocked feature"]
---

# Decision: Block Feature

This blocks the feature.
`
	if err := os.WriteFile(filepath.Join(kbDir, "2026-01-28-block-test.md"), []byte(decision), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name                 string
		task                 string
		acknowledgedDecision string
		wantErr              bool
		wantConflictFound    bool
		wantAcknowledged     bool
	}{
		{
			name:                 "no conflict",
			task:                 "implement user authentication",
			acknowledgedDecision: "",
			wantErr:              false,
			wantConflictFound:    false,
			wantAcknowledged:     false,
		},
		{
			name:                 "conflict not acknowledged",
			task:                 "implement blocked feature",
			acknowledgedDecision: "",
			wantErr:              true,
			wantConflictFound:    true,
			wantAcknowledged:     false,
		},
		{
			name:                 "conflict acknowledged",
			task:                 "implement blocked feature",
			acknowledgedDecision: "2026-01-28-block-test",
			wantErr:              false,
			wantConflictFound:    true,
			wantAcknowledged:     true,
		},
		{
			name:                 "conflict acknowledged with wrong ID",
			task:                 "implement blocked feature",
			acknowledgedDecision: "wrong-decision-id",
			wantErr:              true,
			wantConflictFound:    true,
			wantAcknowledged:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := checkDecisionConflicts(tt.task, tmpDir, tt.acknowledgedDecision)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkDecisionConflicts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result.ConflictFound != tt.wantConflictFound {
				t.Errorf("checkDecisionConflicts() ConflictFound = %v, want %v", result.ConflictFound, tt.wantConflictFound)
			}
			if result.Acknowledged != tt.wantAcknowledged {
				t.Errorf("checkDecisionConflicts() Acknowledged = %v, want %v", result.Acknowledged, tt.wantAcknowledged)
			}
		})
	}
}

// TestCheckDecisionConflictsFailsClosed verifies that the decision gate
// blocks spawns (fails closed) when the decision checking itself fails.
// This is a security/safety-critical behavior - if we can't verify no
// conflicts exist, we must assume they might.
func TestCheckDecisionConflictsFailsClosed(t *testing.T) {
	// Create temp directory with .kb/decisions but make it unreadable
	tmpDir := t.TempDir()
	kbDir := filepath.Join(tmpDir, ".kb", "decisions")
	if err := os.MkdirAll(kbDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a valid decision file first
	decision := `---
blocks:
  - keywords: ["test keyword"]
---

# Decision: Test

This is a test decision.
`
	if err := os.WriteFile(filepath.Join(kbDir, "2026-01-28-test.md"), []byte(decision), 0644); err != nil {
		t.Fatal(err)
	}

	// Make the decisions directory unreadable to trigger an error
	if err := os.Chmod(kbDir, 0000); err != nil {
		t.Skip("Cannot make directory unreadable on this filesystem")
	}
	// Restore permissions after test
	defer os.Chmod(kbDir, 0755)

	result, err := checkDecisionConflicts("any task", tmpDir, "")

	// The gate should BLOCK spawn (fail closed) when decision checking fails
	if err == nil {
		t.Errorf("Expected error when decision check fails (fail-closed behavior), but got nil")
	}

	// The result should still be valid (not nil)
	if result == nil {
		t.Fatalf("Expected result to be non-nil even on error")
	}

	// ConflictFound should be false since we couldn't check
	if result.ConflictFound {
		t.Errorf("Expected ConflictFound to be false when check fails, got true")
	}

	t.Logf("Decision gate correctly blocked spawn when check failed: %v", err)
}

func TestCheckActiveAgentForBeadsID(t *testing.T) {
	tests := []struct {
		name    string
		beadsID string
		wantNil bool
		wantErr bool
	}{
		{
			name:    "empty beads ID returns nil",
			beadsID: "",
			wantNil: true,
			wantErr: false,
		},
		{
			name:    "server not running returns nil (graceful failure)",
			beadsID: "orch-go-fake-id",
			wantNil: true, // Should return nil when server is not reachable
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := checkActiveAgentForBeadsID(tt.beadsID)

			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantNil && agent != nil {
				t.Errorf("expected nil agent, got %+v", agent)
			}
		})
	}
}

func TestFormatActiveAgentError(t *testing.T) {
	tests := []struct {
		name     string
		beadsID  string
		agent    *ActiveAgentInfo
		contains []string
	}{
		{
			name:    "active agent",
			beadsID: "orch-go-123",
			agent: &ActiveAgentInfo{
				ID:        "og-feat-task-123",
				SessionID: "session-abc",
				Status:    "active",
				Phase:     "Implementing",
			},
			contains: []string{"orch-go-123", "og-feat-task-123", "actively running", "Phase: Implementing", "--force"},
		},
		{
			name:    "dead agent",
			beadsID: "orch-go-456",
			agent: &ActiveAgentInfo{
				ID:        "og-debug-task-456",
				SessionID: "session-def",
				Status:    "dead",
			},
			contains: []string{"orch-go-456", "og-debug-task-456", "dead (needs attention"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := formatActiveAgentError(tt.beadsID, tt.agent)
			errMsg := err.Error()

			for _, substr := range tt.contains {
				if !strings.Contains(errMsg, substr) {
					t.Errorf("error message should contain %q, got: %s", substr, errMsg)
				}
			}
		})
	}
}

func TestShouldAutoBypassTriageForOrchestrator(t *testing.T) {
	tests := []struct {
		name       string
		skillName  string
		wantBypass bool
		wantReason string
	}{
		{
			name:       "orchestrator skill auto-bypasses",
			skillName:  "orchestrator",
			wantBypass: true,
			wantReason: "orchestrator",
		},
		{
			name:       "meta-orchestrator skill auto-bypasses",
			skillName:  "meta-orchestrator",
			wantBypass: true,
			wantReason: "meta-orchestrator",
		},
		{
			name:       "worker skill does not auto-bypass",
			skillName:  "feature-impl",
			wantBypass: false,
			wantReason: "",
		},
		{
			name:       "investigation skill does not auto-bypass",
			skillName:  "investigation",
			wantBypass: false,
			wantReason: "",
		},
		{
			name:       "systematic-debugging skill does not auto-bypass",
			skillName:  "systematic-debugging",
			wantBypass: false,
			wantReason: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bypass, reason := shouldAutoBypassTriage(tt.skillName)
			if bypass != tt.wantBypass {
				t.Errorf("shouldAutoBypassTriage() bypass = %v, want %v", bypass, tt.wantBypass)
			}
			if reason != tt.wantReason {
				t.Errorf("shouldAutoBypassTriage() reason = %q, want %q", reason, tt.wantReason)
			}
		})
	}
}

func TestDetermineValidationLevel(t *testing.T) {
	tests := []struct {
		name       string
		skill      string
		requested  string
		task       string
		wantLevel  string
		wantRaised bool
	}{
		{
			name:      "behavioral feature impl escalates tests to integration",
			skill:     "feature-impl",
			requested: "tests",
			task: `TASK: ETL behavior

## Acceptance Criteria
- System skips admin-locked fields
`,
			wantLevel:  "integration",
			wantRaised: true,
		},
		{
			name:      "behavioral feature impl escalates none to integration",
			skill:     "feature-impl",
			requested: "none",
			task: `## Acceptance Criteria
- User can rerun workflow and data survives update
`,
			wantLevel:  "integration",
			wantRaised: true,
		},
		{
			name:      "behavioral feature impl preserves stronger validation",
			skill:     "feature-impl",
			requested: "smoke-test",
			task:      "When user submits, system triggers sync",
			wantLevel: "smoke-test",
		},
		{
			name:      "structural feature impl keeps tests",
			skill:     "feature-impl",
			requested: "tests",
			task: `## Acceptance Criteria
- Add parameter to merge_sources()
- Update call sites
`,
			wantLevel: "tests",
		},
		{
			name:      "non feature impl never escalates",
			skill:     "investigation",
			requested: "tests",
			task:      "System should skip admin fields",
			wantLevel: "tests",
		},
		{
			name:      "empty validation defaults to tests",
			skill:     "feature-impl",
			requested: "",
			task: `## Acceptance Criteria
- Add helper function
`,
			wantLevel: "tests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLevel, gotRaised, _ := determineValidationLevel(tt.skill, tt.requested, tt.task)
			if gotLevel != tt.wantLevel {
				t.Errorf("determineValidationLevel() level = %q, want %q", gotLevel, tt.wantLevel)
			}
			if gotRaised != tt.wantRaised {
				t.Errorf("determineValidationLevel() escalated = %v, want %v", gotRaised, tt.wantRaised)
			}
		})
	}
}
