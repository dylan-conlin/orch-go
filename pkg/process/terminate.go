// Package process provides utilities for managing OS processes.
package process

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"
)

// Terminate safely terminates a process by PID with graceful shutdown.
// Returns true if process was terminated, false if it didn't exist or was already dead.
// Uses SIGTERM first for graceful shutdown, then SIGKILL if needed.
func Terminate(pid int, processName string) bool {
	if pid <= 0 {
		return false
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		// Process doesn't exist (Unix always succeeds, so this is unlikely)
		return false
	}

	// Try SIGTERM first for graceful shutdown
	if err := process.Signal(syscall.SIGTERM); err != nil {
		if err == os.ErrProcessDone || strings.Contains(err.Error(), "process already finished") {
			// Process already dead
			return false
		}
		// ESRCH means no such process (already terminated or PID reused)
		if strings.Contains(err.Error(), "no such process") {
			return false
		}
		// If SIGTERM failed for other reasons, try SIGKILL
		if err := process.Kill(); err != nil {
			return false
		}
		fmt.Printf("Terminated process %d (%s) with SIGKILL\n", pid, processName)
		return true
	}

	// Wait briefly for graceful shutdown
	time.Sleep(500 * time.Millisecond)

	// Verify process is actually dead, if not send SIGKILL
	if err := process.Signal(syscall.Signal(0)); err == nil {
		// Process still alive, force kill
		process.Kill()
		fmt.Printf("Terminated process %d (%s) with SIGKILL\n", pid, processName)
	} else {
		fmt.Printf("Terminated process %d (%s) with SIGTERM\n", pid, processName)
	}

	return true
}
