package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/port"
)

// TestServersListEmpty tests listing when no projects have port allocations.
func TestServersListEmpty(t *testing.T) {
	// Create temporary port registry with no allocations
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "ports.yaml")

	reg, err := port.New(registryPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Save empty registry
	if err := reg.Save(); err != nil {
		t.Fatalf("failed to save registry: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run servers list with empty registry
	err = runServersList(registryPath)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should succeed (no error)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	// Should indicate no servers
	if output == "" {
		t.Error("expected output indicating no servers")
	}
}

// TestServersListWithAllocations tests listing when projects have port allocations.
func TestServersListWithAllocations(t *testing.T) {
	// Create temporary port registry with allocations
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "ports.yaml")

	reg, err := port.New(registryPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Add some test allocations
	_, err = reg.Allocate("test-project", "web", port.PurposeVite)
	if err != nil {
		t.Fatalf("failed to allocate port: %v", err)
	}

	_, err = reg.Allocate("test-project", "api", port.PurposeAPI)
	if err != nil {
		t.Fatalf("failed to allocate port: %v", err)
	}

	_, err = reg.Allocate("another-project", "web", port.PurposeVite)
	if err != nil {
		t.Fatalf("failed to allocate port: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run servers list
	err = runServersList(registryPath)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should succeed
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	// Should show both projects
	if !bytes.Contains([]byte(output), []byte("test-project")) {
		t.Error("expected output to contain 'test-project'")
	}
	if !bytes.Contains([]byte(output), []byte("another-project")) {
		t.Error("expected output to contain 'another-project'")
	}

	// Should show header with PROJECT, PORTS, STATUS
	if !bytes.Contains([]byte(output), []byte("PROJECT")) {
		t.Error("expected output to contain header 'PROJECT'")
	}
}

// TestServersStart tests starting servers via tmuxinator.
func TestServersStart(t *testing.T) {
	// This is a basic test that verifies the function exists and handles errors
	// We don't actually start tmux sessions in tests
	err := runServersStart("nonexistent-project")

	// Should return an error for a project without tmuxinator config
	if err == nil {
		t.Error("expected error for nonexistent project, got nil")
	}
}

// TestServersStop tests stopping servers.
func TestServersStop(t *testing.T) {
	// Test stopping a nonexistent session should handle gracefully
	err := runServersStop("nonexistent-project")

	// Should return an error or handle gracefully
	if err == nil {
		t.Error("expected error for nonexistent session, got nil")
	}
}

// TestServersAttach tests attaching to servers window.
func TestServersAttach(t *testing.T) {
	// Test attaching to nonexistent session should error
	err := runServersAttach("nonexistent-project")

	if err == nil {
		t.Error("expected error for nonexistent session, got nil")
	}
}

// TestServersOpen tests opening browser.
func TestServersOpen(t *testing.T) {
	// Create temporary port registry
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "ports.yaml")

	reg, err := port.New(registryPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Add web port allocation
	webPort, err := reg.Allocate("test-project", "web", port.PurposeVite)
	if err != nil {
		t.Fatalf("failed to allocate port: %v", err)
	}

	// Test opening browser (won't actually open in test)
	// Just verify the function handles the registry lookup
	err = runServersOpen("test-project", registryPath, true) // dry-run mode

	if err != nil {
		t.Errorf("expected no error with valid project, got: %v", err)
	}

	// Test with project that has no web port
	reg.Allocate("no-web-project", "api", port.PurposeAPI)
	err = runServersOpen("no-web-project", registryPath, true)

	if err == nil {
		t.Error("expected error for project without web port")
	}

	_ = webPort // use the variable
}

// TestServersStatus tests the status summary view.
func TestServersStatus(t *testing.T) {
	// Create temporary port registry
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "ports.yaml")

	reg, err := port.New(registryPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Add some allocations
	reg.Allocate("project-a", "web", port.PurposeVite)
	reg.Allocate("project-b", "web", port.PurposeVite)
	reg.Allocate("project-c", "web", port.PurposeVite)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run servers status
	err = runServersStatus(registryPath)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should succeed
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	// Should show summary counts
	if output == "" {
		t.Error("expected status output")
	}
}

func TestServersListReadsFromProjectConfig(t *testing.T) {
	// Create temp directory for a project with config
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "myproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create .orch/config.yaml
	orchDir := filepath.Join(projectDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("failed to create .orch dir: %v", err)
	}

	configContent := `servers:
  web: 5173
  api: 3000
`
	configPath := filepath.Join(orchDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Read servers from project config
	servers, err := port.ListProjectServers(projectDir)
	if err != nil {
		t.Fatalf("ListProjectServers failed: %v", err)
	}

	if len(servers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(servers))
	}

	// Verify ports match config
	foundWeb := false
	foundAPI := false
	for _, srv := range servers {
		if srv.Service == "web" && srv.Port == 5173 {
			foundWeb = true
		}
		if srv.Service == "api" && srv.Port == 3000 {
			foundAPI = true
		}
	}

	if !foundWeb {
		t.Error("web:5173 not found")
	}
	if !foundAPI {
		t.Error("api:3000 not found")
	}
}

// TestServersGenPlist tests plist generation from servers.yaml.
func TestServersGenPlist(t *testing.T) {
	// Create temporary project directory with servers.yaml
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "testproject")
	orchDir := filepath.Join(projectDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("failed to create .orch dir: %v", err)
	}

	// Create servers.yaml with command-type servers
	serversYaml := `servers:
  - name: web
    type: command
    command: npm run dev
    port: 5173
  - name: api
    type: command
    command: go run ./cmd/server
    port: 3000
  - name: db
    type: docker
    image: postgres:15
    port: 5432
`
	serversPath := filepath.Join(orchDir, "servers.yaml")
	if err := os.WriteFile(serversPath, []byte(serversYaml), 0644); err != nil {
		t.Fatalf("failed to write servers.yaml: %v", err)
	}

	// Run gen-plist in dry-run mode
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runServersGenPlist("testproject", projectDir, "", true, false, true) // dry-run

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should succeed
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Should generate plists for command-type servers only (web, api - not db which is docker)
	if !bytes.Contains([]byte(output), []byte("com.testproject.web.plist")) {
		t.Error("expected output to contain web plist path")
	}
	if !bytes.Contains([]byte(output), []byte("com.testproject.api.plist")) {
		t.Error("expected output to contain api plist path")
	}
	// Should not contain docker server
	if bytes.Contains([]byte(output), []byte("com.testproject.db.plist")) {
		t.Error("should not generate plist for docker-type server")
	}

	// Should contain plist XML content
	if !bytes.Contains([]byte(output), []byte("<plist version=\"1.0\">")) {
		t.Error("expected output to contain plist XML")
	}
	if !bytes.Contains([]byte(output), []byte("<key>Label</key>")) {
		t.Error("expected output to contain Label key")
	}
	if !bytes.Contains([]byte(output), []byte("com.testproject.web")) {
		t.Error("expected output to contain web label")
	}
}

// TestServersGenPlist_NoServersYaml tests error handling when no servers.yaml exists.
func TestServersGenPlist_NoServersYaml(t *testing.T) {
	tmpDir := t.TempDir()

	err := runServersGenPlist("testproject", tmpDir, "", true, false, true)

	// Should return an error indicating no servers found
	if err == nil {
		t.Error("expected error for missing servers.yaml")
	}
}

// TestServersGenPlist_NoCommandServers tests error handling when no command-type servers exist.
func TestServersGenPlist_NoCommandServers(t *testing.T) {
	// Create temporary project directory with only docker servers
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "testproject")
	orchDir := filepath.Join(projectDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("failed to create .orch dir: %v", err)
	}

	// Create servers.yaml with only docker-type servers
	serversYaml := `servers:
  - name: db
    type: docker
    image: postgres:15
    port: 5432
`
	serversPath := filepath.Join(orchDir, "servers.yaml")
	if err := os.WriteFile(serversPath, []byte(serversYaml), 0644); err != nil {
		t.Fatalf("failed to write servers.yaml: %v", err)
	}

	err := runServersGenPlist("testproject", projectDir, "", true, false, true)

	// Should return an error indicating no command-type servers
	if err == nil {
		t.Error("expected error for no command-type servers")
	}
}
