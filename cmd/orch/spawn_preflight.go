package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// dockerSocketPath is the default Docker daemon socket path.
// Can be overridden in tests.
var dockerSocketPath = "/var/run/docker.sock"

// checkDockerAvailable verifies the Docker daemon is running.
// Returns an error with actionable instructions if Docker is not available.
func checkDockerAvailable() error {
	// Check if Docker socket exists and is accessible
	if _, err := os.Stat(dockerSocketPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf(`
┌─────────────────────────────────────────────────────────────────────────────┐
│  🚫 Docker daemon not running                                               │
├─────────────────────────────────────────────────────────────────────────────┤
│  The Docker socket was not found at: %s                                     │
│                                                                             │
│  Start Docker daemon:                                                       │
│    macOS: colima start                                                      │
│    Linux: sudo systemctl start docker                                       │
│                                                                             │
│  Verify Docker is running:                                                  │
│    docker ps                                                                │
└─────────────────────────────────────────────────────────────────────────────┘
`, dockerSocketPath)
		}
		return fmt.Errorf("Docker socket exists but cannot be accessed: %w", err)
	}

	return nil
}

// checkClaudeAvailable verifies the claude CLI binary is available in PATH.
// Returns an error with actionable instructions if Claude CLI is not found.
func checkClaudeAvailable() error {
	// Check if claude binary exists in PATH
	_, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf(`
┌─────────────────────────────────────────────────────────────────────────────┐
│  🚫 claude CLI not found                                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│  The claude command-line tool is not in your PATH.                         │
│                                                                             │
│  Install Claude Code:                                                       │
│    https://github.com/anthropics/claude-code                                │
│                                                                             │
│  Verify installation:                                                       │
│    claude --version                                                         │
│                                                                             │
│  Add to PATH (if installed but not in PATH):                                │
│    export PATH="$HOME/.local/bin:$PATH"                                     │
└─────────────────────────────────────────────────────────────────────────────┘
`)
	}

	return nil
}

// checkOpencodeAvailable verifies the OpenCode API is responding at the given URL.
// Returns an error with actionable instructions if the API is not accessible.
func checkOpencodeAvailable(serverURL string) error {
	// Try to connect to the OpenCode API with a short timeout
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	// Try a simple GET request to the root endpoint
	resp, err := client.Get(serverURL)
	if err != nil {
		return fmt.Errorf(`
┌─────────────────────────────────────────────────────────────────────────────┐
│  🚫 OpenCode API not responding                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│  Could not connect to OpenCode at: %s                                       │
│                                                                             │
│  Start OpenCode server:                                                     │
│    orch-dashboard start                                                     │
│                                                                             │
│  Verify server is running:                                                  │
│    orch-dashboard status                                                    │
│    curl %s                                                                  │
│                                                                             │
│  Error: %v                                                                  │
└─────────────────────────────────────────────────────────────────────────────┘
`, serverURL, serverURL, err)
	}
	defer resp.Body.Close()

	// Accept any 2xx or 404 response (404 is fine for root endpoint)
	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		return nil
	}

	return fmt.Errorf("OpenCode API returned unexpected status: %d", resp.StatusCode)
}

// runPreSpawnRuntimeChecks performs preflight runtime checks based on the spawn backend.
// Returns an error if the required runtime is not available.
func runPreSpawnRuntimeChecks(backend, serverURL string) error {
	switch backend {
	case "docker":
		return checkDockerAvailable()
	case "claude":
		return checkClaudeAvailable()
	case "opencode", "": // Empty string means default (opencode)
		return checkOpencodeAvailable(serverURL)
	default:
		// Unknown backend - skip check
		return nil
	}
}

// getOSHint returns a helpful hint based on the current operating system.
func getOSHint(macOSCmd, linuxCmd string) string {
	if runtime.GOOS == "darwin" {
		return macOSCmd
	}
	return linuxCmd
}
