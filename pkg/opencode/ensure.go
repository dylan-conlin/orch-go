package opencode

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// EnsureRunning checks if OpenCode is reachable, and starts it if not.
// Returns nil if OpenCode is running (or was successfully started), error otherwise.
func EnsureRunning(serverURL string) error {
	client := NewClient(serverURL)
	_, err := client.ListSessions("")
	if err == nil {
		return nil // Already running
	}

	// Check if it's a connection error (not running)
	if !strings.Contains(err.Error(), "connection refused") {
		return nil // Some other error, let it proceed
	}

	fmt.Fprintf(os.Stderr, "OpenCode not running, starting it...\n")

	// Start OpenCode server in background, fully detached via shell
	// This ensures the process survives even if the parent is killed
	// Set ORCH_WORKER=1 so agents spawned by this server know they are orch-managed workers
	cmd := exec.Command("sh", "-c", "ORCH_WORKER=1 opencode serve --port 4096 </dev/null >/dev/null 2>&1 &")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start OpenCode: %w", err)
	}

	// Wait for it to be ready (poll for up to 10 seconds)
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		_, err := client.ListSessions("")
		if err == nil {
			fmt.Fprintf(os.Stderr, "OpenCode started successfully\n")
			return nil
		}
	}

	return fmt.Errorf("OpenCode started but not responding after 10s")
}
