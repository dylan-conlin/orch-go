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

// TestServersInit_PackageJSON tests init detection with package.json.
func TestServersInit_PackageJSON(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "testproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create package.json with dev script
	pkgJSON := `{"name": "test-app", "scripts": {"dev": "vite"}}`
	if err := os.WriteFile(filepath.Join(projectDir, "package.json"), []byte(pkgJSON), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	// Run init (dry-run mode)
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runServersInit("testproject", projectDir, true, false, false)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Should detect the web server
	if !bytes.Contains([]byte(output), []byte("Detected")) {
		t.Error("expected output to contain 'Detected'")
	}
	if !bytes.Contains([]byte(output), []byte("web")) {
		t.Error("expected output to contain 'web' server")
	}
	if !bytes.Contains([]byte(output), []byte("package.json")) {
		t.Error("expected output to contain 'package.json' as source")
	}
}

// TestServersInit_GoMod tests init detection with go.mod.
func TestServersInit_GoMod(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "testproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create go.mod
	goMod := `module example.com/test`
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create main.go
	mainGo := `package main
func main() {}`
	if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	// Run init (dry-run mode)
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runServersInit("testproject", projectDir, true, false, false)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Should detect the api server
	if !bytes.Contains([]byte(output), []byte("api")) {
		t.Error("expected output to contain 'api' server")
	}
	if !bytes.Contains([]byte(output), []byte("go.mod")) {
		t.Error("expected output to contain 'go.mod' as source")
	}
}

// TestServersInit_ExistingServersYaml tests error when servers.yaml exists.
func TestServersInit_ExistingServersYaml(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "testproject")
	orchDir := filepath.Join(projectDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("failed to create .orch dir: %v", err)
	}

	// Create existing servers.yaml
	serversYaml := `servers: []`
	if err := os.WriteFile(filepath.Join(orchDir, "servers.yaml"), []byte(serversYaml), 0644); err != nil {
		t.Fatalf("failed to write servers.yaml: %v", err)
	}

	err := runServersInit("testproject", projectDir, false, false, false)

	if err == nil {
		t.Error("expected error for existing servers.yaml")
	}
}

// TestServersInit_Force tests --force flag overwriting existing servers.yaml.
func TestServersInit_Force(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "testproject")
	orchDir := filepath.Join(projectDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("failed to create .orch dir: %v", err)
	}

	// Create existing servers.yaml
	serversYaml := `servers: []`
	if err := os.WriteFile(filepath.Join(orchDir, "servers.yaml"), []byte(serversYaml), 0644); err != nil {
		t.Fatalf("failed to write servers.yaml: %v", err)
	}

	// Create package.json to detect
	pkgJSON := `{"scripts": {"dev": "vite"}}`
	if err := os.WriteFile(filepath.Join(projectDir, "package.json"), []byte(pkgJSON), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runServersInit("testproject", projectDir, false, false, true) // force=true

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("expected no error with --force, got: %v", err)
	}
}

// TestServersInit_NoServersDetected tests when no servers are detected.
func TestServersInit_NoServersDetected(t *testing.T) {
	tmpDir := t.TempDir()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runServersInit("testproject", tmpDir, true, false, false)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !bytes.Contains([]byte(output), []byte("No servers detected")) {
		t.Error("expected output to indicate no servers detected")
	}
}
