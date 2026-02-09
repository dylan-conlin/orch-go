package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsSkillRequiringBuildVerification(t *testing.T) {
	tests := []struct {
		name      string
		skillName string
		want      bool
	}{
		// Implementation skills require build verification
		{"feature-impl requires", "feature-impl", true},
		{"systematic-debugging requires", "systematic-debugging", true},
		{"reliability-testing requires", "reliability-testing", true},

		// Documentation/research skills excluded from build verification
		{"investigation excluded", "investigation", false},
		{"architect excluded", "architect", false},
		{"research excluded", "research", false},
		{"design-session excluded", "design-session", false},
		{"codebase-audit excluded", "codebase-audit", false},
		{"issue-creation excluded", "issue-creation", false},
		{"writing-skills excluded", "writing-skills", false},

		// Edge cases: restrictive default - unknown/empty skills REQUIRE build verification
		// This prevents agents from leaving broken builds (2026-02-06 incident: 23 files with incomplete refactoring)
		{"empty skill requires (restrictive default)", "", true},
		{"unknown skill requires (restrictive default)", "unknown-skill", true},
		{"case insensitive", "Feature-Impl", true},
		{"case insensitive lower", "FEATURE-IMPL", true},
		{"case insensitive excluded", "INVESTIGATION", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSkillRequiringBuildVerification(tt.skillName)
			if got != tt.want {
				t.Errorf("IsSkillRequiringBuildVerification(%q) = %v, want %v", tt.skillName, got, tt.want)
			}
		})
	}
}

func TestIsSkillExcludedFromBuildVerification(t *testing.T) {
	tests := []struct {
		name      string
		skillName string
		want      bool
	}{
		{"investigation excluded", "investigation", true},
		{"architect excluded", "architect", true},
		{"research excluded", "research", true},
		{"feature-impl not excluded", "feature-impl", false},
		{"unknown not excluded", "unknown-skill", false},
		{"empty not excluded", "", false},
		{"case insensitive", "INVESTIGATION", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSkillExcludedFromBuildVerification(tt.skillName)
			if got != tt.want {
				t.Errorf("IsSkillExcludedFromBuildVerification(%q) = %v, want %v", tt.skillName, got, tt.want)
			}
		})
	}
}

func TestIsGoProject(t *testing.T) {
	// Create temp directories for testing
	tempDir := t.TempDir()

	// Create a directory with go.mod
	goModDir := filepath.Join(tempDir, "with-gomod")
	if err := os.MkdirAll(goModDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goModDir, "go.mod"), []byte("module test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a directory with .go files but no go.mod
	goFilesDir := filepath.Join(tempDir, "with-gofiles")
	if err := os.MkdirAll(goFilesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goFilesDir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a non-Go directory
	nonGoDir := filepath.Join(tempDir, "non-go")
	if err := os.MkdirAll(nonGoDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nonGoDir, "package.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		projectDir string
		want       bool
	}{
		{"with go.mod", goModDir, true},
		{"with .go files", goFilesDir, true},
		{"non-go project", nonGoDir, false},
		{"nonexistent", filepath.Join(tempDir, "nonexistent"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsGoProject(tt.projectDir)
			if got != tt.want {
				t.Errorf("IsGoProject(%q) = %v, want %v", tt.projectDir, got, tt.want)
			}
		})
	}
}

func TestHasGoChangesInFiles(t *testing.T) {
	tests := []struct {
		name      string
		gitOutput string
		want      bool
	}{
		{
			name:      "has go code changes",
			gitOutput: "pkg/verify/check.go\npkg/verify/build.go\n",
			want:      true,
		},
		{
			name:      "only config changes",
			gitOutput: "config.yaml\npackage.json\n",
			want:      false,
		},
		{
			name:      "only test changes",
			gitOutput: "pkg/verify/check_test.go\npkg/verify/build_test.go\n",
			want:      true, // Test files are still Go files
		},
		{
			name:      "mixed go and other",
			gitOutput: "pkg/verify/check.go\nREADME.md\n",
			want:      true,
		},
		{
			name:      "empty output",
			gitOutput: "",
			want:      false,
		},
		{
			name:      "whitespace only",
			gitOutput: "   \n\n  \n",
			want:      false,
		},
		{
			name:      "no go files",
			gitOutput: "main.py\napp.js\nconfig.yaml\n",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasGoChangesInFiles(tt.gitOutput)
			if got != tt.want {
				t.Errorf("hasGoChangesInFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTruncateOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
		maxLen int
		want   string
	}{
		{
			name:   "short output",
			output: "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exact length",
			output: "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "truncated",
			output: "hello world this is a long string",
			maxLen: 10,
			want:   "hello worl... (truncated)",
		},
		{
			name:   "empty",
			output: "",
			maxLen: 10,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateOutput(tt.output, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateOutput(%q, %d) = %q, want %q", tt.output, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestVerifyBuild(t *testing.T) {
	// Test with a temp Go project that we know will build
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "go-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a minimal Go project
	goMod := `module test
go 1.21
`
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	mainGo := `package main
func main() {}
`
	if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a mock workspace with feature-impl skill
	workspacePath := filepath.Join(tempDir, "workspace")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a SPAWN_CONTEXT.md with feature-impl skill
	spawnContext := `TASK: Test build verification

## SKILL GUIDANCE (feature-impl)
`
	if err := os.WriteFile(filepath.Join(workspacePath, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
		t.Fatal(err)
	}

	// First verify IsGoProject works
	if !IsGoProject(projectDir) {
		t.Error("IsGoProject() did not detect Go files")
	}

	result := VerifyBuild(workspacePath, projectDir)

	// The skill requires build verification
	if result.SkillName != "feature-impl" {
		t.Errorf("Expected skill 'feature-impl', got %q", result.SkillName)
	}

	if !result.HasGoFiles {
		t.Error("VerifyBuild() did not detect Go files")
	}

	if !result.Passed {
		t.Fatalf("VerifyBuild() should pass for a valid Go project, errors: %v", result.Errors)
	}
}

func TestVerifyBuildForCompletion_NonGoProject(t *testing.T) {
	// Create a non-Go project directory
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "non-go")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "package.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a workspace
	workspacePath := filepath.Join(tempDir, "workspace")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatal(err)
	}
	spawnContext := `TASK: Test

## SKILL GUIDANCE (feature-impl)
`
	if err := os.WriteFile(filepath.Join(workspacePath, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
		t.Fatal(err)
	}

	result := VerifyBuildForCompletion(workspacePath, projectDir)

	// Should return nil for non-Go project
	if result != nil {
		t.Errorf("VerifyBuildForCompletion() should return nil for non-Go project, got %+v", result)
	}
}

func TestVerifyBuildForCompletion_ExcludedSkill(t *testing.T) {
	// Create a temp directory structure
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "go-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Add go.mod to make it a Go project
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a workspace with investigation skill (excluded)
	workspacePath := filepath.Join(tempDir, "workspace")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatal(err)
	}
	spawnContext := `TASK: Test

## SKILL GUIDANCE (investigation)
`
	if err := os.WriteFile(filepath.Join(workspacePath, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
		t.Fatal(err)
	}

	result := VerifyBuildForCompletion(workspacePath, projectDir)

	// Should return nil for excluded skill
	if result != nil {
		t.Errorf("VerifyBuildForCompletion() should return nil for excluded skill, got %+v", result)
	}
}

func TestVerifyBuildForCompletion_RunsEvenWithoutRecentGoChanges(t *testing.T) {
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "go-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	goMod := `module test
go 1.21
`
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Intentionally broken Go source: compile should fail.
	brokenGo := `package main

func main() {
	undefinedSymbol()
}
`
	if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(brokenGo), 0644); err != nil {
		t.Fatal(err)
	}

	workspacePath := filepath.Join(tempDir, "workspace")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatal(err)
	}
	spawnContext := `TASK: Test build gate

## SKILL GUIDANCE (feature-impl)
`
	if err := os.WriteFile(filepath.Join(workspacePath, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
		t.Fatal(err)
	}

	result := VerifyBuildForCompletion(workspacePath, projectDir)
	if result == nil {
		t.Fatal("VerifyBuildForCompletion() should run build gate for Go projects even without git history")
	}
	if result.Passed {
		t.Fatalf("expected build gate to fail on broken Go source, got passed with warnings: %v", result.Warnings)
	}
}
