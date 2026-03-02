// Circuit breaker functions for the control plane post-commit hook.
//
// The post-commit hook (control-plane-post-commit.sh) implements a 3-layer
// circuit breaker that writes ~/.orch/halt when thresholds are exceeded.
// These functions provide the Go API for managing halt/resume state and
// human acknowledgment via the heartbeat file.
//
// See: .kb/investigations/2026-02-14-design-control-plane-heuristics.md
package control

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HaltInfo represents the current halt state of the circuit breaker.
type HaltInfo struct {
	Halted      bool
	Reason      string
	TriggeredBy string
	TriggeredAt string
}

// CircuitBreakerInfo is a composite status including halt state and heartbeat age.
type CircuitBreakerInfo struct {
	Halted       bool
	HaltReason   string
	HaltTrigger  string
	HeartbeatAge time.Duration
}

// DefaultHaltPath returns the default path to ~/.orch/halt.
// Declared as var to allow test overrides.
var DefaultHaltPath = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".orch", "halt")
}

// DefaultHeartbeatPath returns the default path to ~/.orch/heartbeat.
// Declared as var to allow test overrides.
var DefaultHeartbeatPath = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".orch", "heartbeat")
}

// Ack touches the heartbeat file to signal human presence.
// Creates the file if it doesn't exist, updates mtime if it does.
func Ack(heartbeatPath string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(heartbeatPath), 0755); err != nil {
		return fmt.Errorf("creating heartbeat directory: %w", err)
	}

	now := time.Now()
	// Try to update mtime on existing file first
	if err := os.Chtimes(heartbeatPath, now, now); err == nil {
		return nil
	}

	// File doesn't exist, create it
	f, err := os.Create(heartbeatPath)
	if err != nil {
		return fmt.Errorf("creating heartbeat: %w", err)
	}
	return f.Close()
}

// HeartbeatAge returns how long ago the heartbeat was last touched.
// If the heartbeat file doesn't exist, returns a large duration (999 days)
// to indicate "never acknowledged."
func HeartbeatAge(heartbeatPath string) (time.Duration, error) {
	info, err := os.Stat(heartbeatPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 999 * 24 * time.Hour, nil // Never acknowledged
		}
		return 0, fmt.Errorf("checking heartbeat: %w", err)
	}
	return time.Since(info.ModTime()), nil
}

// HaltStatus checks whether the circuit breaker is halted.
// Reads the halt file written by the post-commit hook.
func HaltStatus(haltPath string) (HaltInfo, error) {
	data, err := os.ReadFile(haltPath)
	if err != nil {
		if os.IsNotExist(err) {
			return HaltInfo{}, nil
		}
		return HaltInfo{}, fmt.Errorf("reading halt file: %w", err)
	}

	info := HaltInfo{Halted: true}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if key, val, ok := strings.Cut(line, ": "); ok {
			switch key {
			case "reason":
				info.Reason = val
			case "triggered_by":
				info.TriggeredBy = val
			case "triggered_at":
				info.TriggeredAt = val
			}
		}
	}
	return info, nil
}

// Resume clears the halt state and touches the heartbeat.
// Safe to call when not halted (just touches heartbeat).
func Resume(haltPath, heartbeatPath string) error {
	// Remove halt file (ignore if doesn't exist)
	if err := os.Remove(haltPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing halt file: %w", err)
	}

	// Touch heartbeat
	return Ack(heartbeatPath)
}

// CircuitBreakerStatus returns a composite view of circuit breaker state.
func CircuitBreakerStatus(haltPath, heartbeatPath string) (CircuitBreakerInfo, error) {
	halt, err := HaltStatus(haltPath)
	if err != nil {
		return CircuitBreakerInfo{}, err
	}

	age, err := HeartbeatAge(heartbeatPath)
	if err != nil {
		return CircuitBreakerInfo{}, err
	}

	return CircuitBreakerInfo{
		Halted:       halt.Halted,
		HaltReason:   halt.Reason,
		HaltTrigger:  halt.TriggeredBy,
		HeartbeatAge: age,
	}, nil
}
