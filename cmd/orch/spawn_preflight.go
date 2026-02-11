package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/config"
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

// checkAPIKeyBilling checks if pay-per-token API keys are in the environment
// and blocks spawn unless --allow-api-billing is explicitly set.
// This prevents silent fallback from OAuth to pay-per-token billing.
func checkAPIKeyBilling(cfg *config.Config) error {
	// Check for pay-per-token API keys in environment
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	openaiKey := os.Getenv("OPENAI_API_KEY")

	// If no API keys are set, OAuth will be used - safe to proceed
	if anthropicKey == "" && openaiKey == "" {
		return nil
	}

	// If --allow-api-billing flag is set, user has explicitly opted in
	if spawnAllowAPIBilling {
		return nil
	}

	// If config allows API billing, user has opted in via config file
	if cfg != nil && cfg.Spawn.AllowAPIBilling {
		return nil
	}

	// Build error message with detected keys
	var detectedKeys []string
	if anthropicKey != "" {
		detectedKeys = append(detectedKeys, "ANTHROPIC_API_KEY")
	}
	if openaiKey != "" {
		detectedKeys = append(detectedKeys, "OPENAI_API_KEY")
	}

	keysStr := strings.Join(detectedKeys, " and ")

	return fmt.Errorf(`
┌─────────────────────────────────────────────────────────────────────────────┐
│  🚫 Pay-per-token API key detected                                          │
├─────────────────────────────────────────────────────────────────────────────┤
│  Environment variable: %s                                                   │
│                                                                             │
│  DANGER: These API keys bypass OAuth and use pay-per-token billing.        │
│          A single overnight daemon run could cost $100+ in API charges.    │
│          (January 2026: $402 surprise bill from OPENAI_API_KEY fallback)   │
│                                                                             │
│  To use OAuth (flat $200/mo subscription):                                 │
│    unset %s                                                                 │
│    # Restart OpenCode server to pick up change                             │
│                                                                             │
│  To explicitly opt-in to pay-per-token billing:                            │
│    orch spawn --allow-api-billing ...                                      │
│                                                                             │
│  Reference: orch-go-4bo36 (Safety gate: prevent silent API key fallback)   │
└─────────────────────────────────────────────────────────────────────────────┘
`, keysStr, keysStr)
}
