package servers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUp_NoServersYaml(t *testing.T) {
	tmpDir := t.TempDir()

	results, err := Up("test-project", tmpDir)
	if err == nil {
		t.Error("Expected error when no servers defined")
	}
	if len(results) != 0 {
		t.Errorf("Expected no results, got %d", len(results))
	}
}

func TestDown_NoServersYaml(t *testing.T) {
	tmpDir := t.TempDir()

	results, err := Down("test-project", tmpDir)
	if err == nil {
		t.Error("Expected error when no servers defined")
	}
	if len(results) != 0 {
		t.Errorf("Expected no results, got %d", len(results))
	}
}

func TestStatus_NoServersYaml(t *testing.T) {
	tmpDir := t.TempDir()

	states, err := Status("test-project", tmpDir)
	if err != nil {
		t.Errorf("Status should not error on empty config: %v", err)
	}
	if len(states) != 0 {
		t.Errorf("Expected no states, got %d", len(states))
	}
}

func TestStatus_WithServersYaml(t *testing.T) {
	tmpDir := t.TempDir()
	orchDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a servers.yaml with a command server
	configContent := `servers:
  - name: web
    type: command
    command: echo hello
    port: 3000
  - name: api
    type: command
    command: echo world
    port: 3001
`
	if err := os.WriteFile(filepath.Join(orchDir, "servers.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	states, err := Status("test-project", tmpDir)
	if err != nil {
		t.Errorf("Status should not error: %v", err)
	}
	if len(states) != 2 {
		t.Errorf("Expected 2 states, got %d", len(states))
	}

	// Both should be stopped since we haven't started them
	for _, s := range states {
		if s.Status != StatusStopped {
			t.Errorf("Expected status stopped for %s, got %s", s.Name, s.Status)
		}
	}
}

func TestGetLaunchdLabel(t *testing.T) {
	label := getLaunchdLabel("myproject", "web")
	expected := "com.myproject.web"
	if label != expected {
		t.Errorf("getLaunchdLabel() = %q, want %q", label, expected)
	}
}

func TestGetDockerContainerName(t *testing.T) {
	name := getDockerContainerName("myproject", "db")
	expected := "myproject-db"
	if name != expected {
		t.Errorf("getDockerContainerName() = %q, want %q", name, expected)
	}
}

func TestEnsureLogDir(t *testing.T) {
	tmpDir := t.TempDir()

	if err := EnsureLogDir(tmpDir); err != nil {
		t.Errorf("EnsureLogDir failed: %v", err)
	}

	// Check directory was created
	logDir := filepath.Join(tmpDir, ".orch", "logs")
	info, err := os.Stat(logDir)
	if err != nil {
		t.Errorf("Log directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("Expected directory, got file")
	}
}

func TestUp_InvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	orchDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create an invalid servers.yaml (missing required fields)
	configContent := `servers:
  - name: web
    port: 3000
`
	if err := os.WriteFile(filepath.Join(orchDir, "servers.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Up("test-project", tmpDir)
	if err == nil {
		t.Error("Expected validation error for missing command")
	}
}

func TestServerState(t *testing.T) {
	state := ServerState{
		Name:    "test",
		Type:    TypeCommand,
		Port:    3000,
		Status:  StatusRunning,
		Message: "test message",
	}

	if state.Name != "test" {
		t.Errorf("Name = %q, want %q", state.Name, "test")
	}
	if state.Type != TypeCommand {
		t.Errorf("Type = %q, want %q", state.Type, TypeCommand)
	}
	if state.Status != StatusRunning {
		t.Errorf("Status = %q, want %q", state.Status, StatusRunning)
	}
}

func TestLifecycleResult(t *testing.T) {
	result := LifecycleResult{
		Server:  "web",
		Success: true,
		Message: "started",
	}

	if result.Server != "web" {
		t.Errorf("Server = %q, want %q", result.Server, "web")
	}
	if !result.Success {
		t.Error("Expected success = true")
	}
}
