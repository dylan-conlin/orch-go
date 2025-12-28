package servers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect_Empty(t *testing.T) {
	tmpDir := t.TempDir()

	result, err := Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if len(result.Servers) != 0 {
		t.Errorf("expected 0 servers, got %d", len(result.Servers))
	}
}

func TestDetect_PackageJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create package.json with dev script
	pkgJSON := `{
		"name": "test-app",
		"scripts": {
			"dev": "vite"
		}
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(pkgJSON), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	result, err := Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if len(result.Servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(result.Servers))
	}

	server := result.Servers[0]
	if server.Type != TypeCommand {
		t.Errorf("expected type %s, got %s", TypeCommand, server.Type)
	}
	if server.Command != "npm run dev" {
		t.Errorf("expected 'npm run dev', got '%s'", server.Command)
	}
	if server.Port != 5173 {
		t.Errorf("expected port 5173 (vite default), got %d", server.Port)
	}
	if server.Source != "package.json" {
		t.Errorf("expected source 'package.json', got '%s'", server.Source)
	}
}

func TestDetect_PackageJSON_Bun(t *testing.T) {
	tmpDir := t.TempDir()

	// Create package.json
	pkgJSON := `{"scripts": {"dev": "node server.js"}}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(pkgJSON), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	// Create bun.lockb
	if err := os.WriteFile(filepath.Join(tmpDir, "bun.lockb"), []byte{}, 0644); err != nil {
		t.Fatalf("failed to write bun.lockb: %v", err)
	}

	result, err := Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if len(result.Servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(result.Servers))
	}

	if result.Servers[0].Command != "bun run dev" {
		t.Errorf("expected 'bun run dev', got '%s'", result.Servers[0].Command)
	}
}

func TestDetect_GoMod(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module example.com/test`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create main.go
	mainGo := `package main
func main() {}`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	result, err := Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if len(result.Servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(result.Servers))
	}

	server := result.Servers[0]
	if server.Type != TypeCommand {
		t.Errorf("expected type %s, got %s", TypeCommand, server.Type)
	}
	if server.Command != "go run ." {
		t.Errorf("expected 'go run .', got '%s'", server.Command)
	}
	if server.Port != 8080 {
		t.Errorf("expected port 8080, got %d", server.Port)
	}
}

func TestDetect_GoMod_CmdSubdir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module example.com/test`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create cmd/server/main.go
	cmdDir := filepath.Join(tmpDir, "cmd", "server")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatalf("failed to create cmd dir: %v", err)
	}

	mainGo := `package main
func main() {}`
	if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	result, err := Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if len(result.Servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(result.Servers))
	}

	if result.Servers[0].Command != "go run ./cmd/server" {
		t.Errorf("expected 'go run ./cmd/server', got '%s'", result.Servers[0].Command)
	}
}

func TestDetect_DockerCompose(t *testing.T) {
	tmpDir := t.TempDir()

	// Create docker-compose.yml
	compose := `version: '3.8'
services:
  web:
    build: .
    ports:
      - "3000:3000"
  db:
    image: postgres:15
    ports:
      - "5432:5432"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "docker-compose.yml"), []byte(compose), 0644); err != nil {
		t.Fatalf("failed to write docker-compose.yml: %v", err)
	}

	result, err := Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	// Should detect 2 services
	if len(result.Servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(result.Servers))
	}

	// Check first service
	foundWeb := false
	foundDB := false
	for _, s := range result.Servers {
		if s.Name == "web" && s.Type == TypeDocker {
			foundWeb = true
		}
		if s.Name == "db" && s.Type == TypeDocker {
			foundDB = true
		}
	}

	if !foundWeb {
		t.Error("expected to find 'web' docker service")
	}
	if !foundDB {
		t.Error("expected to find 'db' docker service")
	}
}

func TestDetect_Mixed(t *testing.T) {
	tmpDir := t.TempDir()

	// Create package.json
	pkgJSON := `{"scripts": {"dev": "vite"}}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(pkgJSON), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	// Create docker-compose.yml for db
	compose := `services:
  db:
    image: postgres:15
`
	if err := os.WriteFile(filepath.Join(tmpDir, "docker-compose.yml"), []byte(compose), 0644); err != nil {
		t.Fatalf("failed to write docker-compose.yml: %v", err)
	}

	result, err := Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	// Should detect both
	if len(result.Servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(result.Servers))
	}

	// Check types
	hasCommand := false
	hasDocker := false
	for _, s := range result.Servers {
		if s.Type == TypeCommand {
			hasCommand = true
		}
		if s.Type == TypeDocker {
			hasDocker = true
		}
	}

	if !hasCommand {
		t.Error("expected command-type server from package.json")
	}
	if !hasDocker {
		t.Error("expected docker-type server from docker-compose.yml")
	}
}

func TestDetect_Subdirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create web/ subdirectory with package.json
	webDir := filepath.Join(tmpDir, "web")
	if err := os.MkdirAll(webDir, 0755); err != nil {
		t.Fatalf("failed to create web dir: %v", err)
	}

	pkgJSON := `{"scripts": {"dev": "vite"}}`
	if err := os.WriteFile(filepath.Join(webDir, "package.json"), []byte(pkgJSON), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	// Create api/ subdirectory with go.mod + main.go
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}

	goMod := `module example.com/test`
	if err := os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	mainGo := `package main
func main() {}`
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	result, err := Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	// Should detect both subdirectory projects
	if len(result.Servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(result.Servers))
	}

	// Check workdir is set
	for _, s := range result.Servers {
		if s.Name == "web" && s.Workdir != "web" {
			t.Errorf("expected web workdir 'web', got '%s'", s.Workdir)
		}
		if s.Name == "api" && s.Workdir != "api" {
			t.Errorf("expected api workdir 'api', got '%s'", s.Workdir)
		}
	}
}

func TestDetectionResult_ToConfig(t *testing.T) {
	result := &DetectionResult{
		Servers: []DetectedServer{
			{
				Name:    "web",
				Type:    TypeCommand,
				Command: "npm run dev",
				Port:    3000,
			},
			{
				Name:  "db",
				Type:  TypeDocker,
				Image: "postgres:15",
				Port:  5432,
			},
		},
	}

	cfg := result.ToConfig()

	if len(cfg.Servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(cfg.Servers))
	}

	if cfg.Servers[0].Name != "web" || cfg.Servers[0].Type != TypeCommand {
		t.Error("first server should be web command")
	}

	if cfg.Servers[1].Name != "db" || cfg.Servers[1].Type != TypeDocker {
		t.Error("second server should be db docker")
	}
}

func TestDetectionResult_DeduplicateByName(t *testing.T) {
	result := &DetectionResult{
		Servers: []DetectedServer{
			{Name: "web", Port: 3000},
			{Name: "api", Port: 8080},
			{Name: "web", Port: 5000}, // duplicate
		},
	}

	result.DeduplicateByName()

	if len(result.Servers) != 2 {
		t.Fatalf("expected 2 servers after dedup, got %d", len(result.Servers))
	}

	// First web should be kept
	if result.Servers[0].Port != 3000 {
		t.Errorf("expected first web (port 3000) to be kept, got port %d", result.Servers[0].Port)
	}
}

func TestDetectPackageManager(t *testing.T) {
	tests := []struct {
		name     string
		lockFile string
		expected string
	}{
		{"bun", "bun.lockb", "bun"},
		{"pnpm", "pnpm-lock.yaml", "pnpm"},
		{"yarn", "yarn.lock", "yarn"},
		{"npm default", "", "npm"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			if tc.lockFile != "" {
				if err := os.WriteFile(filepath.Join(tmpDir, tc.lockFile), []byte{}, 0644); err != nil {
					t.Fatalf("failed to create lock file: %v", err)
				}
			}

			result := detectPackageManager(tmpDir)
			if result != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestInferPortFromScript(t *testing.T) {
	tests := []struct {
		script   string
		expected int
	}{
		{"vite", 5173},
		{"vite --port 3000", 5173}, // vite always returns 5173
		{"node server.js", 0},
		{"next dev", 0},
	}

	for _, tc := range tests {
		t.Run(tc.script, func(t *testing.T) {
			result := inferPortFromScript(tc.script)
			if result != tc.expected {
				t.Errorf("script '%s': expected port %d, got %d", tc.script, tc.expected, result)
			}
		})
	}
}
