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

func TestDetect_GoMod_WithServerPatterns(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module example.com/test`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create main.go WITH server patterns
	mainGo := `package main

import (
	"net/http"
)

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}`
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

func TestDetect_GoMod_NoServerPatterns(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module example.com/test`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create main.go WITHOUT server patterns (like gendoc)
	mainGo := `package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Hello, World!")
	os.Exit(0)
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	result, err := Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	// Should NOT detect a server since no server patterns
	if len(result.Servers) != 0 {
		t.Fatalf("expected 0 servers for CLI tool without server patterns, got %d", len(result.Servers))
	}
}

func TestDetect_GoMod_CmdSubdir_WithServer(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module example.com/test`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create cmd/server/main.go WITH server patterns
	cmdDir := filepath.Join(tmpDir, "cmd", "server")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatalf("failed to create cmd dir: %v", err)
	}

	mainGo := `package main

import "net/http"

func main() {
	http.ListenAndServe(":3000", nil)
}`
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
	if result.Servers[0].Port != 3000 {
		t.Errorf("expected detected port 3000, got %d", result.Servers[0].Port)
	}
}

// TestDetect_GoMod_SkipsCLITools verifies that CLI tools like gendoc are not detected as servers
func TestDetect_GoMod_SkipsCLITools(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module example.com/test`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create cmd/gendoc/main.go (CLI tool, not a server)
	gendocDir := filepath.Join(tmpDir, "cmd", "gendoc")
	if err := os.MkdirAll(gendocDir, 0755); err != nil {
		t.Fatalf("failed to create gendoc dir: %v", err)
	}

	gendocMain := `package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Generating documentation...")
	os.Exit(0)
}`
	if err := os.WriteFile(filepath.Join(gendocDir, "main.go"), []byte(gendocMain), 0644); err != nil {
		t.Fatalf("failed to write gendoc main.go: %v", err)
	}

	// Create cmd/server/main.go (actual server)
	serverDir := filepath.Join(tmpDir, "cmd", "server")
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		t.Fatalf("failed to create server dir: %v", err)
	}

	serverMain := `package main

import "net/http"

func main() {
	http.ListenAndServe(":8080", nil)
}`
	if err := os.WriteFile(filepath.Join(serverDir, "main.go"), []byte(serverMain), 0644); err != nil {
		t.Fatalf("failed to write server main.go: %v", err)
	}

	result, err := Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	// Should detect only the server, not gendoc
	if len(result.Servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(result.Servers))
	}

	if result.Servers[0].Command != "go run ./cmd/server" {
		t.Errorf("expected server to be detected, not gendoc. Got command: %s", result.Servers[0].Command)
	}
}

// TestDetect_GoMod_OnlyCLITools verifies that projects with only CLI tools detect no servers
func TestDetect_GoMod_OnlyCLITools(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module example.com/test`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create cmd/gendoc/main.go (CLI tool)
	gendocDir := filepath.Join(tmpDir, "cmd", "gendoc")
	if err := os.MkdirAll(gendocDir, 0755); err != nil {
		t.Fatalf("failed to create gendoc dir: %v", err)
	}

	gendocMain := `package main

import "fmt"

func main() {
	fmt.Println("Generating documentation...")
}`
	if err := os.WriteFile(filepath.Join(gendocDir, "main.go"), []byte(gendocMain), 0644); err != nil {
		t.Fatalf("failed to write gendoc main.go: %v", err)
	}

	// Create cmd/migrate/main.go (another CLI tool)
	migrateDir := filepath.Join(tmpDir, "cmd", "migrate")
	if err := os.MkdirAll(migrateDir, 0755); err != nil {
		t.Fatalf("failed to create migrate dir: %v", err)
	}

	migrateMain := `package main

import "os"

func main() {
	os.Exit(0)
}`
	if err := os.WriteFile(filepath.Join(migrateDir, "main.go"), []byte(migrateMain), 0644); err != nil {
		t.Fatalf("failed to write migrate main.go: %v", err)
	}

	result, err := Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	// Should detect NO servers since all cmd/ entries are CLI tools
	if len(result.Servers) != 0 {
		t.Fatalf("expected 0 servers for project with only CLI tools, got %d: %v", len(result.Servers), result.Servers)
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

	// Create api/ subdirectory with go.mod + main.go WITH server patterns
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}

	goMod := `module example.com/test`
	if err := os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	mainGo := `package main

import "net/http"

func main() {
	http.ListenAndServe(":8080", nil)
}`
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

// TestDetect_GinFramework tests detection of Gin-based servers
func TestDetect_GinFramework(t *testing.T) {
	tmpDir := t.TempDir()

	goMod := `module example.com/test`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	mainGo := `package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	r.GET("/", handler)
	r.Run(":9000")
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	result, err := Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if len(result.Servers) != 1 {
		t.Fatalf("expected 1 server for Gin project, got %d", len(result.Servers))
	}

	// Port should be detected from r.Run(":9000")
	if result.Servers[0].Port != 9000 {
		t.Errorf("expected port 9000, got %d", result.Servers[0].Port)
	}
}

// TestAnalyzeGoFileForServer directly tests the server pattern detection
func TestAnalyzeGoFileForServer(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectScore   bool // true if score > 0 expected
		expectPort    int  // 0 means no port expected
	}{
		{
			name: "net/http server",
			content: `package main
import "net/http"
func main() { http.ListenAndServe(":8080", nil) }`,
			expectScore: true,
			expectPort:  8080,
		},
		{
			name: "CLI tool",
			content: `package main
import "fmt"
func main() { fmt.Println("hello") }`,
			expectScore: false,
			expectPort:  0,
		},
		{
			name: "gin server",
			content: `package main
import "github.com/gin-gonic/gin"
func main() { r := gin.Default(); r.Run(":3000") }`,
			expectScore: true,
			expectPort:  3000,
		},
		{
			name: "echo server",
			content: `package main
import "github.com/labstack/echo"
func main() { e := echo.New(); e.Start(":4000") }`,
			expectScore: true,
			expectPort:  4000,
		},
		{
			name: "cobra CLI",
			content: `package main
import "github.com/spf13/cobra"
func main() { cmd := &cobra.Command{}; cmd.Execute() }`,
			expectScore: false,
			expectPort:  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "main.go")
			if err := os.WriteFile(filePath, []byte(tc.content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			score, port := analyzeGoFileForServer(filePath)

			if tc.expectScore && score <= 0 {
				t.Errorf("expected positive score for %s, got %d", tc.name, score)
			}
			if !tc.expectScore && score > 0 {
				t.Errorf("expected zero/negative score for %s, got %d", tc.name, score)
			}
			if tc.expectPort > 0 && port != tc.expectPort {
				t.Errorf("expected port %d for %s, got %d", tc.expectPort, tc.name, port)
			}
		})
	}
}

// TestIsLikelyCLITool tests the CLI tool name detection
func TestIsLikelyCLITool(t *testing.T) {
	tests := []struct {
		name     string
		dirName  string
		expected bool
	}{
		{"gendoc", "gendoc", true},
		{"gen-docs", "gen-docs", true},
		{"migrate", "migrate", true},
		{"cli", "cli", true},
		{"server", "server", false},
		{"api", "api", false},
		{"orch", "orch", false},
		{"myapp", "myapp", false},
		{"toolbox", "toolbox", true},  // ends with "tool"
		{"setup", "setup", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isLikelyCLITool(tc.dirName)
			if result != tc.expected {
				t.Errorf("isLikelyCLITool(%q) = %v, expected %v", tc.dirName, result, tc.expected)
			}
		})
	}
}

// TestDetect_OrchGoProjectStructure simulates the orch-go project structure
// where cmd/gendoc is a doc generator (CLI tool) and cmd/orch is the main CLI (not a server).
// Neither should be detected as a server since the server code is in cmd/orch/serve.go
// which is a subcommand, not the main entrypoint.
func TestDetect_OrchGoProjectStructure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/dylan-conlin/orch-go`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create cmd/orch/main.go (CLI main entry - no direct server patterns)
	orchDir := filepath.Join(tmpDir, "cmd", "orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("failed to create cmd/orch dir: %v", err)
	}

	orchMain := `package main

import (
	"fmt"
	"os"
	"github.com/spf13/cobra"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "orch",
	Short: "OpenCode orchestration CLI",
}`
	if err := os.WriteFile(filepath.Join(orchDir, "main.go"), []byte(orchMain), 0644); err != nil {
		t.Fatalf("failed to write orch main.go: %v", err)
	}

	// Note: serve.go would contain http.ListenAndServe but it's a subcommand, not main.go
	// The server patterns are in serve.go, but main.go doesn't have them directly

	// Create cmd/gendoc/main.go (doc generator - no server patterns)
	gendocDir := filepath.Join(tmpDir, "cmd", "gendoc")
	if err := os.MkdirAll(gendocDir, 0755); err != nil {
		t.Fatalf("failed to create cmd/gendoc dir: %v", err)
	}

	gendocMain := `package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func main() {
	rootCmd := buildCommandTree()
	doc.GenMarkdownTree(rootCmd, "docs/")
	fmt.Println("Documentation generated")
}`
	if err := os.WriteFile(filepath.Join(gendocDir, "main.go"), []byte(gendocMain), 0644); err != nil {
		t.Fatalf("failed to write gendoc main.go: %v", err)
	}

	result, err := Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	// Neither cmd/orch nor cmd/gendoc should be detected as a server
	// cmd/orch is a CLI (server is a subcommand, not the main entry)
	// cmd/gendoc is a doc generator (CLI tool)
	if len(result.Servers) != 0 {
		var names []string
		for _, s := range result.Servers {
			names = append(names, s.Command)
		}
		t.Errorf("expected 0 servers (orch is CLI, gendoc is doc generator), got %d: %v", len(result.Servers), names)
	}
}

// TestDetect_OrchGoWithServeSubcommand shows that if serve.go is in the cmd/orch directory,
// the server patterns ARE detected because we analyze all .go files in the package.
func TestDetect_OrchGoWithServeSubcommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/dylan-conlin/orch-go`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create cmd/orch/main.go (CLI main entry)
	orchDir := filepath.Join(tmpDir, "cmd", "orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("failed to create cmd/orch dir: %v", err)
	}

	orchMain := `package main

import "github.com/spf13/cobra"

func main() {
	rootCmd.Execute()
}

var rootCmd = &cobra.Command{Use: "orch"}`
	if err := os.WriteFile(filepath.Join(orchDir, "main.go"), []byte(orchMain), 0644); err != nil {
		t.Fatalf("failed to write orch main.go: %v", err)
	}

	// Create cmd/orch/serve.go with server patterns
	// This simulates orch-go's actual structure where serve.go contains the HTTP server
	serveGo := `package main

import "net/http"

func runServe(port int) error {
	return http.ListenAndServe(":3348", nil)
}`
	if err := os.WriteFile(filepath.Join(orchDir, "serve.go"), []byte(serveGo), 0644); err != nil {
		t.Fatalf("failed to write serve.go: %v", err)
	}

	result, err := Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	// cmd/orch SHOULD be detected because serve.go has server patterns
	if len(result.Servers) != 1 {
		t.Fatalf("expected 1 server (cmd/orch has serve.go with http.ListenAndServe), got %d", len(result.Servers))
	}

	if result.Servers[0].Command != "go run ./cmd/orch" {
		t.Errorf("expected 'go run ./cmd/orch', got '%s'", result.Servers[0].Command)
	}

	// Port should be detected from serve.go
	if result.Servers[0].Port != 3348 {
		t.Errorf("expected port 3348, got %d", result.Servers[0].Port)
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
