package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHandoffCommandFlags verifies the handoff command accepts expected flags.
func TestHandoffCommandFlags(t *testing.T) {
	if handoffCmd == nil {
		t.Fatal("handoffCmd is nil")
	}

	// Check -o flag exists
	oFlag := handoffCmd.Flags().Lookup("output")
	if oFlag == nil {
		t.Error("Expected -o/--output flag")
	}

	// Check --json flag exists
	jsonFlag := handoffCmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Error("Expected --json flag")
	}
}

// TestHandoffDataStructure verifies the HandoffData structure is correctly initialized.
func TestHandoffDataStructure(t *testing.T) {
	data := &HandoffData{
		Date:          "21 Dec 2025",
		TLDR:          "Test session.",
		ActiveAgents:  []ActiveAgent{},
		PendingIssues: []PendingIssue{},
		RecentWork:    []RecentWorkItem{},
		NextPriority:  []string{},
	}

	if data.Date != "21 Dec 2025" {
		t.Errorf("Expected date '21 Dec 2025', got %q", data.Date)
	}

	if len(data.ActiveAgents) != 0 {
		t.Errorf("Expected empty ActiveAgents, got %d", len(data.ActiveAgents))
	}
}

// TestGenerateTLDR verifies TLDR generation from handoff data.
func TestGenerateTLDR(t *testing.T) {
	tests := []struct {
		name     string
		data     *HandoffData
		project  string
		contains []string
	}{
		{
			name: "with active agents and focus",
			data: &HandoffData{
				ActiveAgents: []ActiveAgent{
					{BeadsID: "test-123"},
					{BeadsID: "test-456"},
				},
				Focus: &FocusInfo{
					Goal:      "Ship MVP",
					IsDrifted: false,
				},
			},
			project:  "test-project",
			contains: []string{"2 active agent(s)", "focused on: Ship MVP"},
		},
		{
			name: "with uncommitted changes",
			data: &HandoffData{
				LocalState: &LocalStateInfo{
					HasUncommitted: true,
					Summary:        "5 uncommitted changes",
				},
			},
			project:  "test-project",
			contains: []string{"5 uncommitted changes"},
		},
		{
			name: "with drift",
			data: &HandoffData{
				Focus: &FocusInfo{
					Goal:      "Other work",
					IsDrifted: true,
				},
			},
			project:  "test-project",
			contains: []string{"drifted from focus"},
		},
		{
			name:     "empty data",
			data:     &HandoffData{},
			project:  "empty-project",
			contains: []string{"Session handoff for empty-project", "No active work"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateTLDR(tt.data, tt.project)
			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("generateTLDR() = %q, want to contain %q", result, want)
				}
			}
		})
	}
}

// TestTruncatePriority verifies priority truncation.
func TestTruncatePriority(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Short text", "Short text"},
		{"This is a very long priority description that exceeds the limit", "This is a very long priority description that e..."},
		{"Exactly fifty characters long string goes here!", "Exactly fifty characters long string goes here!"},
	}

	for _, tt := range tests {
		t.Run(tt.input[:min(20, len(tt.input))], func(t *testing.T) {
			result := truncatePriority(tt.input)
			if result != tt.expected {
				t.Errorf("truncatePriority(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestDeriveNextPriorities verifies priority derivation.
func TestDeriveNextPriorities(t *testing.T) {
	tests := []struct {
		name        string
		data        *HandoffData
		minExpected int
	}{
		{
			name: "with active agents",
			data: &HandoffData{
				ActiveAgents: []ActiveAgent{
					{BeadsID: "agent-1", Task: "Task 1"},
					{BeadsID: "agent-2", Task: "Task 2"},
				},
			},
			minExpected: 2,
		},
		{
			name: "with P0 pending issues",
			data: &HandoffData{
				PendingIssues: []PendingIssue{
					{ID: "issue-1", Priority: "P0", Title: "Critical bug"},
					{ID: "issue-2", Priority: "P1", Title: "Important feature"},
				},
			},
			minExpected: 2,
		},
		{
			name:        "empty data",
			data:        &HandoffData{},
			minExpected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveNextPriorities(tt.data)
			if len(result) < tt.minExpected {
				t.Errorf("deriveNextPriorities() returned %d priorities, want at least %d", len(result), tt.minExpected)
			}
		})
	}
}

// TestGenerateHandoffMarkdown verifies markdown generation.
func TestGenerateHandoffMarkdown(t *testing.T) {
	data := &HandoffData{
		Date: "21 Dec 2025",
		TLDR: "Test session with active agents.",
		ActiveAgents: []ActiveAgent{
			{BeadsID: "test-123", Repo: "test-repo", Task: "Test task"},
		},
		PendingIssues: []PendingIssue{
			{ID: "issue-1", Title: "Test issue", Priority: "P0"},
		},
		NextPriority: []string{"Check test-123"},
		DEKN:         &DEKNSummary{}, // Empty DEKN for prompts
	}

	markdown, err := generateHandoffMarkdown(data)
	if err != nil {
		t.Fatalf("generateHandoffMarkdown() error = %v", err)
	}

	// Verify key sections are present
	sections := []string{
		"# Session Handoff - 21 Dec 2025",
		"## TLDR",
		"Test session with active agents.",
		"## D.E.K.N. Summary",
		"**Delta:**",
		"**Evidence:**",
		"**Knowledge:**",
		"**Next:**",
		"## Agents Still Running",
		"**test-123**",
		"## Next Session Priorities",
		"Check test-123",
		"## Quick Commands",
	}

	for _, section := range sections {
		if !strings.Contains(markdown, section) {
			t.Errorf("generateHandoffMarkdown() missing section %q", section)
		}
	}
}

// TestGenerateHandoffMarkdownWithDEKN verifies markdown generation with filled D.E.K.N.
func TestGenerateHandoffMarkdownWithDEKN(t *testing.T) {
	data := &HandoffData{
		Date: "24 Dec 2025",
		TLDR: "Session with D.E.K.N. summary.",
		DEKN: &DEKNSummary{
			Delta:     "Built authentication system",
			Evidence:  "15 commits, all tests passing",
			Knowledge: "OAuth requires refresh tokens",
			Next:      "Integrate with user service",
		},
		ActiveAgents:  []ActiveAgent{},
		PendingIssues: []PendingIssue{},
		NextPriority:  []string{},
	}

	markdown, err := generateHandoffMarkdown(data)
	if err != nil {
		t.Fatalf("generateHandoffMarkdown() error = %v", err)
	}

	// Verify D.E.K.N. content is rendered
	deknContent := []string{
		"**Delta:** Built authentication system",
		"**Evidence:** 15 commits, all tests passing",
		"**Knowledge:** OAuth requires refresh tokens",
		"**Next:** Integrate with user service",
	}

	for _, content := range deknContent {
		if !strings.Contains(markdown, content) {
			t.Errorf("generateHandoffMarkdown() missing D.E.K.N. content %q", content)
		}
	}

	// Verify placeholder text is NOT present when content is filled
	placeholders := []string{
		"describe the transformation",
		"Proof of work",
		"What was learned",
		"Recommended next",
	}

	for _, placeholder := range placeholders {
		if strings.Contains(markdown, placeholder) {
			t.Errorf("generateHandoffMarkdown() should not contain placeholder %q when D.E.K.N. is filled", placeholder)
		}
	}
}

// TestGatherLocalState verifies local state gathering.
func TestGatherLocalState(t *testing.T) {
	// Use temp dir as a non-git directory
	tmpDir := t.TempDir()

	state := gatherLocalState(tmpDir)

	// Non-git directory should have empty branch
	// (git commands will fail gracefully)
	if state == nil {
		t.Fatal("gatherLocalState() returned nil")
	}
}

// TestHandoffOutputPath verifies output path handling.
func TestHandoffOutputPath(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		output     string
		expectFile string
		createDir  bool
	}{
		{
			name:       "directory path",
			output:     tmpDir,
			expectFile: filepath.Join(tmpDir, "SESSION_HANDOFF.md"),
			createDir:  false, // Already exists
		},
		{
			name:       "file path",
			output:     filepath.Join(tmpDir, "custom.md"),
			expectFile: filepath.Join(tmpDir, "custom.md"),
			createDir:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputPath := tt.output
			// If output is a directory, append filename
			if info, err := os.Stat(tt.output); err == nil && info.IsDir() {
				outputPath = filepath.Join(tt.output, "SESSION_HANDOFF.md")
			}

			if outputPath != tt.expectFile {
				t.Errorf("output path = %q, want %q", outputPath, tt.expectFile)
			}
		})
	}
}

// TestActiveAgentExtraction verifies beads ID extraction from window names.
func TestActiveAgentExtraction(t *testing.T) {
	tests := []struct {
		windowName string
		wantID     string
	}{
		{"🏗️ og-feat-test [test-123]", "test-123"},
		{"🔬 og-inv-research [proj-abc]", "proj-abc"},
		{"🐛 og-debug-fix [snap-xyz]", "snap-xyz"},
		{"servers", ""},
		{"zsh", ""},
		{"no-brackets", ""},
	}

	for _, tt := range tests {
		t.Run(tt.windowName, func(t *testing.T) {
			result := extractBeadsIDFromWindowName(tt.windowName)
			if result != tt.wantID {
				t.Errorf("extractBeadsIDFromWindowName(%q) = %q, want %q", tt.windowName, result, tt.wantID)
			}
		})
	}
}

// TestFocusInfoStructure verifies FocusInfo initialization.
func TestFocusInfoStructure(t *testing.T) {
	info := &FocusInfo{
		Goal:      "Test goal",
		BeadsID:   "focus-123",
		IsDrifted: true,
	}

	if info.Goal != "Test goal" {
		t.Errorf("Expected goal 'Test goal', got %q", info.Goal)
	}

	if !info.IsDrifted {
		t.Error("Expected IsDrifted to be true")
	}
}

// TestPendingIssueStructure verifies PendingIssue initialization.
func TestPendingIssueStructure(t *testing.T) {
	issue := &PendingIssue{
		ID:       "issue-123",
		Title:    "Test issue",
		Priority: "P0",
	}

	if issue.ID != "issue-123" {
		t.Errorf("Expected ID 'issue-123', got %q", issue.ID)
	}

	if issue.Priority != "P0" {
		t.Errorf("Expected priority 'P0', got %q", issue.Priority)
	}
}

// TestRecentWorkItemStructure verifies RecentWorkItem initialization.
func TestRecentWorkItemStructure(t *testing.T) {
	item := &RecentWorkItem{
		Type:        "completed",
		Description: "Test work item",
		Repo:        "test-repo",
	}

	if item.Type != "completed" {
		t.Errorf("Expected type 'completed', got %q", item.Type)
	}
}

// TestLocalStateInfoStructure verifies LocalStateInfo initialization.
func TestLocalStateInfoStructure(t *testing.T) {
	state := &LocalStateInfo{
		HasUncommitted: true,
		Branch:         "feature/test",
		Summary:        "5 uncommitted changes",
	}

	if !state.HasUncommitted {
		t.Error("Expected HasUncommitted to be true")
	}

	if state.Branch != "feature/test" {
		t.Errorf("Expected branch 'feature/test', got %q", state.Branch)
	}
}

// TestDEKNSummaryStructure verifies DEKNSummary initialization.
func TestDEKNSummaryStructure(t *testing.T) {
	dekn := &DEKNSummary{
		Delta:     "Built new authentication system",
		Evidence:  "12 commits, +500 lines, all tests passing",
		Knowledge: "OAuth flow requires refresh token handling",
		Next:      "Integrate with user service",
	}

	if dekn.Delta != "Built new authentication system" {
		t.Errorf("Expected Delta 'Built new authentication system', got %q", dekn.Delta)
	}
	if dekn.Evidence != "12 commits, +500 lines, all tests passing" {
		t.Errorf("Expected Evidence '12 commits, +500 lines, all tests passing', got %q", dekn.Evidence)
	}
}

// TestIsDEKNPlaceholder verifies placeholder detection.
func TestIsDEKNPlaceholder(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{"empty string", "", true},
		{"whitespace only", "   ", true},
		{"bracketed placeholder", "[What changed this session]", true},
		{"contains TODO", "TODO: fill this in", true},
		{"contains FILL IN", "Please FILL IN this section", true},
		{"default Delta prompt", "describe the transformation", true},
		{"default Evidence prompt", "Proof of work - X commits", true},
		{"default Knowledge prompt", "What was learned - new patterns", true},
		{"default Next prompt", "Recommended next actions", true},
		{"actual content", "Built authentication middleware with JWT support", false},
		{"actual evidence", "15 commits, +2000/-500 lines, CI green", false},
		{"actual knowledge", "Rate limiting requires Redis for distributed state", false},
		{"actual next", "Deploy to staging and run load tests", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isDEKNPlaceholder(tt.text)
			if result != tt.expected {
				t.Errorf("isDEKNPlaceholder(%q) = %v, want %v", tt.text, result, tt.expected)
			}
		})
	}
}

// TestValidateDEKN verifies D.E.K.N. validation logic.
func TestValidateDEKN(t *testing.T) {
	tests := []struct {
		name        string
		dekn        *DEKNSummary
		expectError bool
		errContains string
	}{
		{
			name:        "nil DEKN",
			dekn:        nil,
			expectError: true,
			errContains: "D.E.K.N. summary is required",
		},
		{
			name: "all empty",
			dekn: &DEKNSummary{
				Delta:     "",
				Evidence:  "",
				Knowledge: "",
				Next:      "",
			},
			expectError: true,
			errContains: "Delta, Evidence, Knowledge, Next",
		},
		{
			name: "all placeholders",
			dekn: &DEKNSummary{
				Delta:     "[What changed]",
				Evidence:  "[Proof of work]",
				Knowledge: "[What was learned]",
				Next:      "[Recommended next]",
			},
			expectError: true,
			errContains: "Delta, Evidence, Knowledge, Next",
		},
		{
			name: "partial - missing Delta",
			dekn: &DEKNSummary{
				Delta:     "",
				Evidence:  "5 commits, tests passing",
				Knowledge: "Learned about caching",
				Next:      "Deploy to production",
			},
			expectError: true,
			errContains: "Delta",
		},
		{
			name: "partial - missing Evidence",
			dekn: &DEKNSummary{
				Delta:     "Built feature X",
				Evidence:  "[TODO]",
				Knowledge: "Learned about caching",
				Next:      "Deploy to production",
			},
			expectError: true,
			errContains: "Evidence",
		},
		{
			name: "all filled with actual content",
			dekn: &DEKNSummary{
				Delta:     "Built authentication middleware with JWT support",
				Evidence:  "15 commits, +2000/-500 lines, CI green",
				Knowledge: "Rate limiting requires Redis for distributed state",
				Next:      "Deploy to staging and run load tests",
			},
			expectError: false,
		},
		{
			name: "minimal but valid content",
			dekn: &DEKNSummary{
				Delta:     "Fixed bug",
				Evidence:  "1 commit",
				Knowledge: "Edge case in parser",
				Next:      "Monitor errors",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDEKN(tt.dekn)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestParseInProgressBeadsOutput verifies parsing of bd list --status in_progress output.
func TestParseInProgressBeadsOutput(t *testing.T) {
	// Sample output from bd list --status in_progress
	sampleOutput := `orch-go-3dem [P1] [feature] in_progress - [orch-go] Redesign orch status output to be actionable
kb-cli-e9z [P2] [feature] in_progress - [kb-cli] kb context should detect stale investigations with closed linked issues
orch-go-hey6 [P2] [task] in_progress - orch handoff generates stale/incorrect data
orch-go-ipq9 [P2] [task] in_progress - orch spawn: Auto-init if .orch directories missing`

	// Parse the output (simulating what getInProgressBeadsIDs does)
	result := make(map[string]bool)
	lines := strings.Split(strings.TrimSpace(sampleOutput), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 1 {
			beadsID := parts[0]
			if strings.Contains(beadsID, "-") && len(beadsID) > 3 {
				result[beadsID] = true
			}
		}
	}

	// Verify expected IDs are present
	expectedIDs := []string{"orch-go-3dem", "kb-cli-e9z", "orch-go-hey6", "orch-go-ipq9"}
	for _, id := range expectedIDs {
		if !result[id] {
			t.Errorf("Expected beads ID %q to be in result", id)
		}
	}

	// Verify no false positives
	unexpectedIDs := []string{"orch-go-66n", "P1", "feature", "in_progress"}
	for _, id := range unexpectedIDs {
		if result[id] {
			t.Errorf("Did not expect %q to be in result", id)
		}
	}
}

// TestParseBdReadyOutput verifies parsing of bd ready output format.
func TestParseBdReadyOutput(t *testing.T) {
	// Sample output from bd ready
	sampleOutput := `📋 Ready work (10 issues with no blockers):

1. [P2] [feature] orch-go-xwh: Iterate on Swarm Dashboard UI/UX
2. [P2] [task] orch-go-36b: [orch-go] design-session: Dashboard needs better agent activity visibilit...
3. [P2] [task] orch-go-vut1: [feature] Model flexibility - phase 2`

	var issues []PendingIssue
	lines := strings.Split(strings.TrimSpace(sampleOutput), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "📋") || strings.HasPrefix(line, "No ") {
			continue
		}
		if len(line) >= 3 && line[0] >= '0' && line[0] <= '9' && line[1] == '.' {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				priority := strings.Trim(parts[1], "[]")
				var beadsID, title string
				for i := 2; i < len(parts); i++ {
					if strings.HasSuffix(parts[i], ":") {
						beadsID = strings.TrimSuffix(parts[i], ":")
						if i+1 < len(parts) {
							title = strings.Join(parts[i+1:], " ")
						}
						break
					}
				}
				if beadsID != "" {
					issues = append(issues, PendingIssue{
						ID:       beadsID,
						Title:    title,
						Priority: priority,
					})
				}
			}
		}
	}

	// Verify parsed issues
	if len(issues) != 3 {
		t.Fatalf("Expected 3 issues, got %d", len(issues))
	}

	// Check first issue
	if issues[0].ID != "orch-go-xwh" {
		t.Errorf("Expected first issue ID 'orch-go-xwh', got %q", issues[0].ID)
	}
	if issues[0].Priority != "P2" {
		t.Errorf("Expected first issue priority 'P2', got %q", issues[0].Priority)
	}
	if issues[0].Title != "Iterate on Swarm Dashboard UI/UX" {
		t.Errorf("Expected first issue title 'Iterate on Swarm Dashboard UI/UX', got %q", issues[0].Title)
	}

	// Check second issue (has colons in title)
	if issues[1].ID != "orch-go-36b" {
		t.Errorf("Expected second issue ID 'orch-go-36b', got %q", issues[1].ID)
	}
}

// TestParseBdClosedOutput verifies parsing of bd list --status closed output format.
func TestParseBdClosedOutput(t *testing.T) {
	// Sample output from bd list --status closed
	sampleOutput := `orch-go-66n [P0] [task] closed [triage:ready] - Implement Synthesis Protocol (D.E.K.N. Schema & Verification Gate)
orch-go-o7x [P0] [task] closed [triage:ready] - Full HTTP API integration for orch send (Native Q&A)
orch-go-5b9 [P0] [bug] closed - Fix: tmux spawn should not use --format json`

	var work []RecentWorkItem
	lines := strings.Split(strings.TrimSpace(sampleOutput), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if idx := strings.Index(line, " - "); idx > 0 {
			parts := strings.Fields(line[:idx])
			beadsID := ""
			if len(parts) >= 1 {
				beadsID = parts[0]
			}
			title := line[idx+3:]
			description := title
			if beadsID != "" {
				description = "[" + beadsID + "] " + title
			}
			work = append(work, RecentWorkItem{
				Type:        "completed",
				Description: description,
			})
		}
	}

	// Verify parsed work items
	if len(work) != 3 {
		t.Fatalf("Expected 3 work items, got %d", len(work))
	}

	// Check first item
	expectedDesc := "[orch-go-66n] Implement Synthesis Protocol (D.E.K.N. Schema & Verification Gate)"
	if work[0].Description != expectedDesc {
		t.Errorf("Expected description %q, got %q", expectedDesc, work[0].Description)
	}

	// Check that we're not including the brackets or status in the description
	if strings.Contains(work[0].Description, "[P0]") {
		t.Error("Description should not contain priority")
	}
	if strings.Contains(work[0].Description, "closed") {
		t.Error("Description should not contain status")
	}
}
