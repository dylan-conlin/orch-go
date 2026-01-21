package daemon

import (
	"os/exec"
	"testing"
)

// mockCmd creates a mock command that outputs the given JSON.
func mockCmd(output string, exitCode int) func() *exec.Cmd {
	return func() *exec.Cmd {
		// Use echo to simulate command output
		// For exit codes, we need a different approach
		if exitCode != 0 {
			// Use a command that will fail
			return exec.Command("false")
		}
		return exec.Command("echo", "-n", output)
	}
}

func TestListProjects_Success(t *testing.T) {
	// JSON output from kb projects list --json
	mockOutput := `[{"name":"project-b","path":"/path/to/b"},{"name":"project-a","path":"/path/to/a"}]`

	projects, err := listProjectsWithCommand(mockCmd(mockOutput, 0))
	if err != nil {
		t.Fatalf("listProjectsWithCommand() unexpected error: %v", err)
	}

	// Should return 2 projects
	if len(projects) != 2 {
		t.Fatalf("listProjectsWithCommand() got %d projects, want 2", len(projects))
	}

	// Should be sorted alphabetically by name
	if projects[0].Name != "project-a" {
		t.Errorf("listProjectsWithCommand() first project = %q, want 'project-a'", projects[0].Name)
	}
	if projects[1].Name != "project-b" {
		t.Errorf("listProjectsWithCommand() second project = %q, want 'project-b'", projects[1].Name)
	}

	// Verify paths
	if projects[0].Path != "/path/to/a" {
		t.Errorf("listProjectsWithCommand() first project path = %q, want '/path/to/a'", projects[0].Path)
	}
	if projects[1].Path != "/path/to/b" {
		t.Errorf("listProjectsWithCommand() second project path = %q, want '/path/to/b'", projects[1].Path)
	}
}

func TestListProjects_EmptyOutput(t *testing.T) {
	// Empty string output
	projects, err := listProjectsWithCommand(mockCmd("", 0))
	if err != nil {
		t.Fatalf("listProjectsWithCommand() unexpected error: %v", err)
	}

	if len(projects) != 0 {
		t.Errorf("listProjectsWithCommand() got %d projects, want 0", len(projects))
	}
}

func TestListProjects_EmptyArray(t *testing.T) {
	// Empty JSON array
	projects, err := listProjectsWithCommand(mockCmd("[]", 0))
	if err != nil {
		t.Fatalf("listProjectsWithCommand() unexpected error: %v", err)
	}

	if len(projects) != 0 {
		t.Errorf("listProjectsWithCommand() got %d projects, want 0", len(projects))
	}
}

func TestListProjects_KbUnavailable(t *testing.T) {
	// Simulate kb command failure (command exits with non-zero)
	projects, err := listProjectsWithCommand(mockCmd("", 1))
	if err != nil {
		t.Fatalf("listProjectsWithCommand() unexpected error when kb unavailable: %v", err)
	}

	// Should return empty slice, not error
	if projects == nil {
		t.Error("listProjectsWithCommand() returned nil, want empty slice")
	}
	if len(projects) != 0 {
		t.Errorf("listProjectsWithCommand() got %d projects, want 0", len(projects))
	}
}

func TestListProjects_InvalidJSON(t *testing.T) {
	// Invalid JSON should return error
	_, err := listProjectsWithCommand(mockCmd("not valid json", 0))
	if err == nil {
		t.Error("listProjectsWithCommand() expected error for invalid JSON, got nil")
	}
}

func TestListProjects_AlreadySorted(t *testing.T) {
	// Already sorted input should remain sorted
	mockOutput := `[{"name":"alpha","path":"/a"},{"name":"beta","path":"/b"},{"name":"gamma","path":"/g"}]`

	projects, err := listProjectsWithCommand(mockCmd(mockOutput, 0))
	if err != nil {
		t.Fatalf("listProjectsWithCommand() unexpected error: %v", err)
	}

	if len(projects) != 3 {
		t.Fatalf("listProjectsWithCommand() got %d projects, want 3", len(projects))
	}

	expectedOrder := []string{"alpha", "beta", "gamma"}
	for i, name := range expectedOrder {
		if projects[i].Name != name {
			t.Errorf("listProjectsWithCommand() project[%d].Name = %q, want %q", i, projects[i].Name, name)
		}
	}
}

func TestListProjects_SingleProject(t *testing.T) {
	// Single project should work correctly
	mockOutput := `[{"name":"solo","path":"/only/one"}]`

	projects, err := listProjectsWithCommand(mockCmd(mockOutput, 0))
	if err != nil {
		t.Fatalf("listProjectsWithCommand() unexpected error: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("listProjectsWithCommand() got %d projects, want 1", len(projects))
	}

	if projects[0].Name != "solo" {
		t.Errorf("listProjectsWithCommand() project.Name = %q, want 'solo'", projects[0].Name)
	}
	if projects[0].Path != "/only/one" {
		t.Errorf("listProjectsWithCommand() project.Path = %q, want '/only/one'", projects[0].Path)
	}
}

func TestBuildListProjectsCommand(t *testing.T) {
	cmd := BuildListProjectsCommand()

	// Verify command is correct
	if cmd.Path == "" {
		t.Error("BuildListProjectsCommand() cmd.Path is empty")
	}

	// Check args contain expected values
	args := cmd.Args
	if len(args) < 4 {
		t.Fatalf("BuildListProjectsCommand() args = %v, want at least 4 args", args)
	}

	// args[0] is the command itself, args[1:] are the arguments
	// We expect: kb projects list --json
	if args[1] != "projects" || args[2] != "list" || args[3] != "--json" {
		t.Errorf("BuildListProjectsCommand() args = %v, want [kb, projects, list, --json]", args)
	}
}
