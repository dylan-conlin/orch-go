// Package process provides utilities for managing OS processes.
package process

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// BunProcess represents a bun process discovered on the system.
type BunProcess struct {
	PID     int
	Command string
	CWD     string
}

// FindBunProcessesInDirectory finds bun processes whose working directory is
// the provided directory (or a subdirectory).
func FindBunProcessesInDirectory(directory string) ([]BunProcess, error) {
	targetDir, err := filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory %q: %w", directory, err)
	}
	targetDir = filepath.Clean(targetDir)

	cmd := exec.Command("ps", "-eo", "pid,args")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run ps: %w", err)
	}

	var processes []BunProcess
	for _, line := range strings.Split(string(output), "\n") {
		pid, command, ok := parsePIDArgsLine(line)
		if !ok || !isBunCommandLine(command) {
			continue
		}

		cwd, err := processWorkingDirectory(pid)
		if err != nil {
			continue
		}

		if !isWithinDirectory(cwd, targetDir) {
			continue
		}

		processes = append(processes, BunProcess{
			PID:     pid,
			Command: command,
			CWD:     cwd,
		})
	}

	return processes, nil
}

func parsePIDArgsLine(line string) (int, string, bool) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) < 2 {
		return 0, "", false
	}

	pid, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, "", false
	}

	return pid, strings.Join(fields[1:], " "), true
}

func isBunCommandLine(command string) bool {
	fields := strings.Fields(strings.TrimSpace(command))
	if len(fields) == 0 {
		return false
	}

	for _, token := range fields {
		if token == "env" {
			continue
		}
		if strings.Contains(token, "=") && !strings.HasPrefix(token, "/") {
			continue
		}
		return filepath.Base(token) == "bun"
	}

	return false
}

func processWorkingDirectory(pid int) (string, error) {
	if runtime.GOOS == "linux" {
		cwd, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid))
		if err == nil {
			return filepath.Clean(cwd), nil
		}
	}

	cmd := exec.Command("lsof", "-a", "-p", strconv.Itoa(pid), "-d", "cwd", "-Fn")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to inspect cwd for pid %d: %w", pid, err)
	}

	cwd, err := parseLsofCWD(string(output))
	if err != nil {
		return "", err
	}

	return filepath.Clean(cwd), nil
}

func parseLsofCWD(output string) (string, error) {
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "n") {
			cwd := strings.TrimSpace(strings.TrimPrefix(line, "n"))
			if cwd != "" {
				return cwd, nil
			}
		}
	}

	return "", fmt.Errorf("cwd not found in lsof output")
}

func isWithinDirectory(path, directory string) bool {
	cleanPath := filepath.Clean(path)
	cleanDir := filepath.Clean(directory)

	if cleanPath == cleanDir {
		return true
	}

	rel, err := filepath.Rel(cleanDir, cleanPath)
	if err != nil {
		return false
	}

	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
