package verify

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

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

	// Note: result.Passed may be false because there are no recent git commits
	// in the temp directory, so it skips the build check. That's expected behavior.
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

func TestRunGoVet_Clean(t *testing.T) {
	// Create a minimal Go project that passes vet
	tempDir := t.TempDir()
	goMod := "module test\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}
	mainGo := "package main\n\nfunc main() {}\n"
	if err := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	output, err := RunGoVet(tempDir)
	if err != nil {
		t.Errorf("RunGoVet() on clean project returned error: %v, output: %s", err, output)
	}
}

func TestRunGoVet_WithIssues(t *testing.T) {
	// Create a Go project with a vet issue (unreachable code)
	tempDir := t.TempDir()
	goMod := "module test\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}
	// Printf format mismatch is a classic vet catch
	mainGo := `package main

import "fmt"

func main() {
	fmt.Printf("%d", "not-a-number")
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	output, err := RunGoVet(tempDir)
	if err == nil {
		t.Error("RunGoVet() on project with vet issues should return error")
	}
	if output == "" {
		t.Error("RunGoVet() should return output describing the vet issue")
	}
}

func TestRunGoVet_NonexistentDir(t *testing.T) {
	_, err := RunGoVet("/nonexistent/path")
	if err == nil {
		t.Error("RunGoVet() on nonexistent directory should return error")
	}
}

func TestVerifyBuild_VetFailure(t *testing.T) {
	// Create a Go project that builds but fails vet
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "go-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	goMod := "module test\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}
	// Printf format mismatch: builds fine, fails vet
	mainGo := `package main

import "fmt"

func main() {
	fmt.Printf("%d", "not-a-number")
}
`
	if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize a git repo with a commit so HasGoChangesInRecentCommits works
	setupGitRepo(t, projectDir)

	// Create workspace with feature-impl skill
	workspacePath := filepath.Join(tempDir, "workspace")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatal(err)
	}
	spawnContext := "TASK: Test\n\n## SKILL GUIDANCE (feature-impl)\n"
	if err := os.WriteFile(filepath.Join(workspacePath, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
		t.Fatal(err)
	}

	result := VerifyBuild(workspacePath, projectDir)

	// Build should pass
	if !result.BuildPassed {
		t.Error("BuildPassed should be true (code compiles)")
	}

	// Vet should fail
	if result.VetPassed {
		t.Error("VetPassed should be false (Printf format mismatch)")
	}

	// Overall should fail
	if result.Passed {
		t.Error("Passed should be false when vet fails")
	}

	// Should have vet-related errors
	hasVetError := false
	for _, e := range result.Errors {
		if len(e) > 0 && (e == "'go vet ./...' failed" || len(e) > 4) {
			hasVetError = true
			break
		}
	}
	if !hasVetError {
		t.Errorf("Expected vet error messages, got: %v", result.Errors)
	}
}

func TestVerifyBuild_BuildAndVetPass(t *testing.T) {
	// Create a clean Go project
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "go-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	goMod := "module test\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}
	mainGo := "package main\n\nfunc main() {}\n"
	if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize a git repo with a commit
	setupGitRepo(t, projectDir)

	// Create workspace with feature-impl skill
	workspacePath := filepath.Join(tempDir, "workspace")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatal(err)
	}
	spawnContext := "TASK: Test\n\n## SKILL GUIDANCE (feature-impl)\n"
	if err := os.WriteFile(filepath.Join(workspacePath, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
		t.Fatal(err)
	}

	result := VerifyBuild(workspacePath, projectDir)

	if !result.BuildPassed {
		t.Errorf("BuildPassed should be true, errors: %v", result.Errors)
	}
	if !result.VetPassed {
		t.Errorf("VetPassed should be true, errors: %v", result.Errors)
	}
	if !result.Passed {
		t.Errorf("Passed should be true, errors: %v", result.Errors)
	}
}

// setupGitRepo initializes a git repo with two commits so HasGoChangesInRecentCommits
// can detect changes via HEAD~1..HEAD. The first commit is empty (with --allow-empty),
// and the second commit includes all files in the directory.
func setupGitRepo(t *testing.T, dir string) {
	t.Helper()
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "initial empty"},
		{"git", "add", "."},
		{"git", "commit", "-m", "add files"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setupGitRepo: %v failed: %v\n%s", args, err, out)
		}
	}
}

func TestVerifyBuildForCompletion_NoGitChanges(t *testing.T) {
	// Create a Go project without git — no recent Go changes means nil result
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "go-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Add go.mod to make it a Go project
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a workspace
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

	// Should return nil when no Go changes detected (no git repo)
	if result != nil {
		t.Errorf("VerifyBuildForCompletion() should return nil when no Go changes, got %+v", result)
	}
}
