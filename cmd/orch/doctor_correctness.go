package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func checkBeadsIntegrity() ServiceStatus {
	status := ServiceStatus{
		Name:      "Beads DB Integrity",
		CanFix:    false,
		FixAction: "Run: sqlite3 .beads/beads.db \".recover\" > recovered.sql && mv .beads/beads.db .beads/beads.db.corrupted && sqlite3 .beads/beads.db < recovered.sql",
	}

	projectDir, err := os.Getwd()
	if err != nil {
		status.Running = true // Skip check if can't get pwd
		status.Details = "Could not determine working directory"
		return status
	}

	dbPath := filepath.Join(projectDir, ".beads", "beads.db")

	// Check if database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		status.Running = true // No beads DB is OK (not all projects use beads)
		status.Details = "No beads database (OK)"
		return status
	}

	// Run PRAGMA integrity_check
	cmd := exec.Command("sqlite3", dbPath, "PRAGMA integrity_check;")
	output, err := cmd.Output()
	if err != nil {
		status.Running = false
		status.Details = fmt.Sprintf("⚠️ integrity_check failed: %v", err)
		if doctorVerbose {
			if exitErr, ok := err.(*exec.ExitError); ok {
				status.Details = fmt.Sprintf("⚠️ integrity_check failed: %v (stderr: %s)", err, string(exitErr.Stderr))
			}
		}
		return status
	}

	result := strings.TrimSpace(string(output))
	if result == "ok" {
		status.Running = true
		status.Details = "Database integrity verified"
		return status
	}

	// Corruption detected!
	status.Running = false
	status.Details = fmt.Sprintf("⚠️ CORRUPTION DETECTED: %s", result)
	return status
}

// checkDockerBackend tests the Docker backend by spawning a trivial container.
// Only runs if Docker is available. Failure indicates Docker issues that would
// affect --backend docker spawns.
func checkDockerBackend() ServiceStatus {
	status := ServiceStatus{
		Name:      "Docker Backend",
		CanFix:    false,
		FixAction: "Start Docker Desktop or run: sudo systemctl start docker",
	}

	// Check if docker command exists
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		status.Running = true // Docker not installed is OK
		status.Details = "Docker not installed (optional)"
		return status
	}

	// Check if Docker daemon is running (quick info check)
	infoCmd := exec.Command(dockerPath, "info", "--format", "{{.ServerVersion}}")
	infoOutput, err := infoCmd.Output()
	if err != nil {
		status.Running = false
		status.Details = "⚠️ Docker daemon not responding"
		if doctorVerbose {
			if exitErr, ok := err.(*exec.ExitError); ok {
				status.Details = fmt.Sprintf("⚠️ Docker daemon not responding: %s", string(exitErr.Stderr))
			}
		}
		return status
	}

	dockerVersion := strings.TrimSpace(string(infoOutput))

	// Run a trivial container test (hello-world is small and fast)
	testCmd := exec.Command(dockerPath, "run", "--rm", "hello-world")
	testOutput, err := testCmd.CombinedOutput()
	if err != nil {
		status.Running = false
		status.Details = fmt.Sprintf("⚠️ Container test failed (Docker %s)", dockerVersion)
		if doctorVerbose {
			// Look for common failure patterns
			outputStr := string(testOutput)
			if strings.Contains(outputStr, "Cannot connect") {
				status.Details = fmt.Sprintf("⚠️ Cannot connect to Docker daemon (Docker %s)", dockerVersion)
			} else if strings.Contains(outputStr, "pull access denied") {
				status.Details = fmt.Sprintf("⚠️ Pull access denied for hello-world (Docker %s)", dockerVersion)
			} else {
				status.Details = fmt.Sprintf("⚠️ Container test failed: %v (Docker %s)", err, dockerVersion)
			}
		}
		return status
	}

	// Verify the output contains expected hello-world message
	if !strings.Contains(string(testOutput), "Hello from Docker") {
		status.Running = false
		status.Details = fmt.Sprintf("⚠️ Unexpected container output (Docker %s)", dockerVersion)
		return status
	}

	status.Running = true
	status.Details = fmt.Sprintf("Container test passed (Docker %s)", dockerVersion)
	return status
}
