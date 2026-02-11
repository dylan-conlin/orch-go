package tmux

import (
	"fmt"
	"strings"
	"time"
)

// getPaneContentFunc is the function used to get pane content.
// Can be overridden in tests.
var getPaneContentFunc = GetPaneContent

// LivenessConfig configures the post-spawn liveness probe.
type LivenessConfig struct {
	// How long to wait after sending the command before checking
	WaitDuration time.Duration

	// Error patterns to detect in the pane content
	ErrorPatterns []string
}

// DefaultLivenessConfig returns a default liveness probe configuration.
func DefaultLivenessConfig() LivenessConfig {
	return LivenessConfig{
		WaitDuration: 8 * time.Second,
		ErrorPatterns: []string{
			"Cannot connect to the Docker daemon",
			"docker: command not found",
			"docker: not found",
			"claude: command not found",
			"claude: not found",
			"No such file or directory",
			"Connection refused",
			"Cannot connect",
			"ECONNREFUSED",
			"connection refused",
			"Failed to connect",
			"Error:",
			"ERROR:",
			"command not found",
		},
	}
}

// ProbeWindowForErrors waits for the specified duration, then captures the pane content
// and checks for error patterns. Returns an error if any error pattern is detected.
// This is used to detect spawn failures that would otherwise be silently hidden in tmux panes.
func ProbeWindowForErrors(windowTarget string, cfg LivenessConfig) error {
	// Wait for the agent to start (or fail)
	time.Sleep(cfg.WaitDuration)

	// Capture the pane content
	content, err := getPaneContentFunc(windowTarget)
	if err != nil {
		return fmt.Errorf("failed to capture pane content for liveness probe: %w", err)
	}

	// Check for error patterns
	for _, pattern := range cfg.ErrorPatterns {
		if strings.Contains(content, pattern) {
			return fmt.Errorf("spawn failed - detected error pattern %q in tmux pane:\n%s", pattern, content)
		}
	}

	return nil
}
