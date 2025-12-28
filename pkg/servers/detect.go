// Package servers provides detection logic for inferring server configurations from project files.

package servers

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DetectedServer represents a server inferred from project file analysis.
type DetectedServer struct {
	Name        string            // Server name (e.g., "web", "api")
	Type        ServerType        // Server type (command, docker)
	Command     string            // Command to run (for command type)
	Image       string            // Docker image (for docker type)
	Port        int               // Port to expose
	Source      string            // Where this was detected from (e.g., "package.json", "docker-compose.yml")
	Env         map[string]string // Environment variables
	Workdir     string            // Working directory relative to project root
	HealthCheck *HealthCheck      // Optional health check
}

// DetectionResult holds the results of scanning a project directory.
type DetectionResult struct {
	Servers    []DetectedServer // Detected servers
	Warnings   []string         // Non-fatal warnings
	ProjectDir string           // The directory that was scanned
}

// Detect scans a project directory and returns detected server configurations.
// Detection priority:
// 1. docker-compose.yml -> docker type
// 2. package.json with "dev" script -> command type (npm/bun run dev)
// 3. go.mod with main.go -> command type (go run .)
func Detect(projectDir string) (*DetectionResult, error) {
	result := &DetectionResult{
		Servers:    []DetectedServer{},
		Warnings:   []string{},
		ProjectDir: projectDir,
	}

	// Convert to absolute path
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, err
	}

	// Check for docker-compose.yml
	dockerServers, warnings := detectDockerCompose(absDir)
	result.Servers = append(result.Servers, dockerServers...)
	result.Warnings = append(result.Warnings, warnings...)

	// Check for package.json with dev script
	if nodeServer, found := detectPackageJSON(absDir); found {
		result.Servers = append(result.Servers, nodeServer)
	}

	// Check for go.mod with main.go
	if goServer, found := detectGoProject(absDir); found {
		result.Servers = append(result.Servers, goServer)
	}

	// Check subdirectories for additional projects
	subDirs, err := detectSubdirectoryProjects(absDir)
	if err == nil {
		result.Servers = append(result.Servers, subDirs...)
	}

	return result, nil
}

// detectDockerCompose looks for docker-compose.yml and extracts service definitions.
func detectDockerCompose(projectDir string) ([]DetectedServer, []string) {
	var servers []DetectedServer
	var warnings []string

	// Check for docker-compose.yml or docker-compose.yaml
	var composePath string
	for _, name := range []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"} {
		path := filepath.Join(projectDir, name)
		if _, err := os.Stat(path); err == nil {
			composePath = path
			break
		}
	}

	if composePath == "" {
		return servers, warnings
	}

	// Read docker-compose file
	data, err := os.ReadFile(composePath)
	if err != nil {
		warnings = append(warnings, "found docker-compose.yml but could not read: "+err.Error())
		return servers, warnings
	}

	// Parse YAML to extract services
	// We use a simple approach: look for patterns rather than full YAML parsing
	// This avoids adding a YAML dependency just for detection
	content := string(data)
	lines := strings.Split(content, "\n")

	// Simple state machine to extract services
	inServices := false
	currentService := ""
	currentIndent := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Calculate indent (2-space or 4-space)
		indent := len(line) - len(strings.TrimLeft(line, " "))

		if trimmed == "services:" {
			inServices = true
			currentIndent = indent
			continue
		}

		if !inServices {
			continue
		}

		// If we've returned to the services level or less, we've left the services block
		if indent <= currentIndent && trimmed != "" && !strings.HasPrefix(trimmed, "-") {
			inServices = false
			continue
		}

		// Service name line (one indent level in)
		if indent == currentIndent+2 && strings.HasSuffix(trimmed, ":") {
			serviceName := strings.TrimSuffix(trimmed, ":")
			currentService = serviceName

			// Create a docker-type server entry
			// We'll let docker-compose handle the details
			server := DetectedServer{
				Name:   serviceName,
				Type:   TypeDocker,
				Source: filepath.Base(composePath),
			}
			servers = append(servers, server)
		}
	}

	if len(servers) > 0 {
		// If we found docker-compose services, recommend using docker-compose up
		// rather than individual container management
		warnings = append(warnings, "docker-compose services detected; consider running via 'docker compose up' wrapper")
	}

	_ = currentService // silence unused variable

	return servers, warnings
}

// packageJSON represents the relevant fields from package.json.
type packageJSON struct {
	Name    string            `json:"name"`
	Scripts map[string]string `json:"scripts"`
}

// detectPackageJSON looks for package.json with a dev script.
func detectPackageJSON(projectDir string) (DetectedServer, bool) {
	pkgPath := filepath.Join(projectDir, "package.json")

	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return DetectedServer{}, false
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return DetectedServer{}, false
	}

	// Look for dev script
	devScript, hasDevScript := pkg.Scripts["dev"]
	if !hasDevScript {
		return DetectedServer{}, false
	}

	// Determine command runner (bun, npm, pnpm, yarn)
	runner := detectPackageManager(projectDir)

	// Infer port from the dev script
	port := inferPortFromScript(devScript)
	if port == 0 {
		port = 3000 // Default for most node servers
	}

	// Infer server name
	name := "web"
	if pkg.Name != "" {
		// If package has a name, check if it suggests API/backend
		lower := strings.ToLower(pkg.Name)
		if strings.Contains(lower, "api") || strings.Contains(lower, "server") || strings.Contains(lower, "backend") {
			name = "api"
		}
	}

	return DetectedServer{
		Name:    name,
		Type:    TypeCommand,
		Command: runner + " run dev",
		Port:    port,
		Source:  "package.json",
		HealthCheck: &HealthCheck{
			Type:     HealthHTTP,
			Path:     "/",
			Interval: Duration(5 * 1e9), // 5s
			Timeout:  Duration(2 * 1e9), // 2s
			Retries:  3,
		},
	}, true
}

// detectPackageManager determines which package manager to use.
func detectPackageManager(projectDir string) string {
	// Check for lock files in order of preference
	if _, err := os.Stat(filepath.Join(projectDir, "bun.lockb")); err == nil {
		return "bun"
	}
	if _, err := os.Stat(filepath.Join(projectDir, "pnpm-lock.yaml")); err == nil {
		return "pnpm"
	}
	if _, err := os.Stat(filepath.Join(projectDir, "yarn.lock")); err == nil {
		return "yarn"
	}
	// Default to npm
	return "npm"
}

// inferPortFromScript tries to extract a port number from a script command.
func inferPortFromScript(script string) int {
	// Common patterns:
	// --port 3000, --port=3000, -p 3000, PORT=3000
	// vite defaults to 5173

	if strings.Contains(script, "vite") {
		return 5173
	}

	// Look for --port or -p followed by number
	// This is a simplified heuristic
	words := strings.Fields(script)
	for i, word := range words {
		if word == "--port" || word == "-p" {
			if i+1 < len(words) {
				var port int
				if _, err := parsePort(words[i+1]); err == nil {
					return port
				}
			}
		}
		if strings.HasPrefix(word, "--port=") {
			portStr := strings.TrimPrefix(word, "--port=")
			if port, err := parsePort(portStr); err == nil {
				return port
			}
		}
		if strings.HasPrefix(word, "PORT=") {
			portStr := strings.TrimPrefix(word, "PORT=")
			if port, err := parsePort(portStr); err == nil {
				return port
			}
		}
	}

	return 0
}

// parsePort attempts to parse a port number from a string.
func parsePort(s string) (int, error) {
	var port int
	_, err := os.ReadFile("/dev/null") // just to use os
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		port = port*10 + int(c-'0')
	}
	if port < 1 || port > 65535 {
		return 0, err
	}
	return port, nil
}

// goServerCandidate represents a potential Go server found during detection.
type goServerCandidate struct {
	mainPath   string // Path to main.go
	dirName    string // Directory name (e.g., "orch", "gendoc")
	score      int    // Higher score = more likely to be a server
	detectedPort int  // Port detected from code, 0 if none
}

// cliToolPatterns are directory names that indicate CLI tools (not servers).
var cliToolPatterns = []string{
	"gendoc", "gen", "cli", "tool", "migrate", "seed",
	"cmd", "init", "setup", "config", "admin",
}

// serverImportPatterns are Go imports that indicate HTTP server usage.
var serverImportPatterns = []string{
	"net/http",
	"github.com/gin-gonic/gin",
	"github.com/labstack/echo",
	"github.com/go-chi/chi",
	"github.com/gofiber/fiber",
	"github.com/gorilla/mux",
}

// serverCallPatterns are function call patterns that indicate server usage.
var serverCallPatterns = []*regexp.Regexp{
	regexp.MustCompile(`http\.ListenAndServe`),
	regexp.MustCompile(`http\.ListenAndServeTLS`),
	regexp.MustCompile(`net\.Listen`),
	regexp.MustCompile(`\.Run\(\s*[":0-9]`),           // gin/echo .Run() with port
	regexp.MustCompile(`\.Start\(\s*[":0-9]`),         // echo .Start()
	regexp.MustCompile(`\.Listen\(\s*[":0-9]`),        // fiber .Listen()
	regexp.MustCompile(`ListenAndServe`),
}

// portPatterns are patterns to extract port numbers from code.
var portPatterns = []*regexp.Regexp{
	regexp.MustCompile(`:(\d{4,5})`),                  // :8080, :3000, etc.
	regexp.MustCompile(`port\s*[=:]\s*(\d{4,5})`),     // port = 8080, port: 8080
	regexp.MustCompile(`Port\s*[=:]\s*(\d{4,5})`),     // Port = 8080
}

// isLikelyCLITool checks if a directory name suggests a CLI tool rather than a server.
func isLikelyCLITool(dirName string) bool {
	lower := strings.ToLower(dirName)
	for _, pattern := range cliToolPatterns {
		if strings.HasPrefix(lower, pattern) || strings.HasSuffix(lower, pattern) {
			return true
		}
	}
	return false
}

// analyzeGoPackageForServer analyzes Go files in a directory for server patterns.
// Returns a score (higher = more likely server) and detected port (0 if none).
func analyzeGoPackageForServer(dirPath string) (score int, port int) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0, 0
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		if strings.HasSuffix(entry.Name(), "_test.go") {
			continue // Skip test files
		}

		filePath := filepath.Join(dirPath, entry.Name())
		fileScore, filePort := analyzeGoFileForServer(filePath)
		score += fileScore
		if filePort > 0 && port == 0 {
			port = filePort // Use first detected port
		}
	}

	return score, port
}

// analyzeGoFileForServer analyzes a single Go file for server patterns.
// Returns a score and detected port.
func analyzeGoFileForServer(filePath string) (score int, port int) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inImportBlock := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Track import blocks
		if strings.HasPrefix(trimmed, "import (") {
			inImportBlock = true
			continue
		}
		if inImportBlock && trimmed == ")" {
			inImportBlock = false
			continue
		}

		// Check for server imports
		if inImportBlock || strings.HasPrefix(trimmed, "import ") {
			for _, importPattern := range serverImportPatterns {
				if strings.Contains(line, importPattern) {
					score += 2 // Server import is strong signal
				}
			}
		}

		// Check for server function calls
		for _, pattern := range serverCallPatterns {
			if pattern.MatchString(line) {
				score += 5 // Function call is very strong signal
			}
		}

		// Try to extract port
		if port == 0 {
			for _, pattern := range portPatterns {
				if matches := pattern.FindStringSubmatch(line); len(matches) > 1 {
					if p, err := parsePort(matches[1]); err == nil && p >= 1024 && p <= 65535 {
						port = p
					}
				}
			}
		}
	}

	return score, port
}

// detectGoProject looks for go.mod with a main package that appears to be a server.
// It analyzes the content of main packages to identify actual HTTP servers,
// skipping CLI tools like doc generators, migration scripts, etc.
func detectGoProject(projectDir string) (DetectedServer, bool) {
	// Check for go.mod
	modPath := filepath.Join(projectDir, "go.mod")
	if _, err := os.Stat(modPath); err != nil {
		return DetectedServer{}, false
	}

	var candidates []goServerCandidate

	// Check root main.go
	rootMain := filepath.Join(projectDir, "main.go")
	if _, err := os.Stat(rootMain); err == nil {
		score, port := analyzeGoPackageForServer(projectDir)
		candidates = append(candidates, goServerCandidate{
			mainPath:     rootMain,
			dirName:      filepath.Base(projectDir),
			score:        score,
			detectedPort: port,
		})
	}

	// Check cmd/ subdirectories
	cmdDir := filepath.Join(projectDir, "cmd")
	if entries, err := os.ReadDir(cmdDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			dirName := entry.Name()
			mainPath := filepath.Join(cmdDir, dirName, "main.go")

			if _, err := os.Stat(mainPath); err != nil {
				continue // No main.go in this cmd/ subdir
			}

			// Penalize directories that look like CLI tools
			penalty := 0
			if isLikelyCLITool(dirName) {
				penalty = 10 // Heavy penalty for likely CLI tools
			}

			score, port := analyzeGoPackageForServer(filepath.Join(cmdDir, dirName))
			score -= penalty

			candidates = append(candidates, goServerCandidate{
				mainPath:     mainPath,
				dirName:      dirName,
				score:        score,
				detectedPort: port,
			})
		}
	}

	if len(candidates) == 0 {
		return DetectedServer{}, false
	}

	// Find the best candidate (highest score, must have positive score to be considered a server)
	var best *goServerCandidate
	for i := range candidates {
		if candidates[i].score > 0 {
			if best == nil || candidates[i].score > best.score {
				best = &candidates[i]
			}
		}
	}

	// If no candidate has server patterns, don't detect a server
	if best == nil {
		return DetectedServer{}, false
	}

	// Determine command based on main.go location
	var command string
	if filepath.Dir(best.mainPath) == projectDir {
		command = "go run ."
	} else {
		// cmd/xxx/main.go -> go run ./cmd/xxx
		relDir, _ := filepath.Rel(projectDir, filepath.Dir(best.mainPath))
		command = "go run ./" + relDir
	}

	// Use detected port or default to 8080
	port := best.detectedPort
	if port == 0 {
		port = 8080
	}

	return DetectedServer{
		Name:    "api",
		Type:    TypeCommand,
		Command: command,
		Port:    port,
		Source:  "go.mod + " + filepath.Base(best.mainPath),
		HealthCheck: &HealthCheck{
			Type:     HealthTCP,
			Interval: Duration(5 * 1e9), // 5s
			Timeout:  Duration(2 * 1e9), // 2s
			Retries:  3,
		},
	}, true
}

// detectSubdirectoryProjects looks for projects in common subdirectory patterns.
func detectSubdirectoryProjects(projectDir string) ([]DetectedServer, error) {
	var servers []DetectedServer

	// Common patterns: web/, frontend/, client/, api/, backend/, server/
	subDirs := []struct {
		dir  string
		name string
	}{
		{"web", "web"},
		{"frontend", "web"},
		{"client", "web"},
		{"api", "api"},
		{"backend", "api"},
		{"server", "api"},
	}

	for _, subDir := range subDirs {
		subPath := filepath.Join(projectDir, subDir.dir)
		if _, err := os.Stat(subPath); os.IsNotExist(err) {
			continue
		}

		// Check for package.json in subdirectory
		if server, found := detectPackageJSON(subPath); found {
			server.Name = subDir.name
			server.Workdir = subDir.dir
			servers = append(servers, server)
		}

		// Check for go.mod in subdirectory
		if server, found := detectGoProject(subPath); found {
			server.Name = subDir.name
			server.Workdir = subDir.dir
			servers = append(servers, server)
		}
	}

	return servers, nil
}

// ToConfig converts a DetectionResult to a servers Config.
func (r *DetectionResult) ToConfig() *Config {
	cfg := &Config{
		Servers: make([]Server, 0, len(r.Servers)),
	}

	for _, detected := range r.Servers {
		server := Server{
			Name:    detected.Name,
			Type:    detected.Type,
			Command: detected.Command,
			Image:   detected.Image,
			Port:    detected.Port,
			Env:     detected.Env,
			Workdir: detected.Workdir,
			Health:  detected.HealthCheck,
		}
		cfg.Servers = append(cfg.Servers, server)
	}

	return cfg
}

// DeduplicateByName removes duplicate servers by name, keeping the first occurrence.
func (r *DetectionResult) DeduplicateByName() {
	seen := make(map[string]bool)
	unique := make([]DetectedServer, 0, len(r.Servers))

	for _, server := range r.Servers {
		if !seen[server.Name] {
			seen[server.Name] = true
			unique = append(unique, server)
		}
	}

	r.Servers = unique
}
