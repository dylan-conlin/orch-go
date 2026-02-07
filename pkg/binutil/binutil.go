// Package binutil provides utilities for finding executable binaries.
package binutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CommonSearchPaths returns common installation locations for binaries.
// These paths work across macOS and Linux. Windows uses USERPROFILE instead of HOME.
func CommonSearchPaths(binaryName string) []string {
	return []string{
		"$HOME/bin/" + binaryName,
		"$HOME/go/bin/" + binaryName,
		"$HOME/.bun/bin/" + binaryName,
		"$HOME/.local/bin/" + binaryName,
		"/usr/local/bin/" + binaryName,
		"/opt/homebrew/bin/" + binaryName,
	}
}

// ResolveBinary attempts to find an executable binary using multiple strategies.
// This is essential for processes running under launchd or other minimal PATH environments.
//
// Search order:
//  1. Environment variable (if envVarName is provided and set)
//  2. Current PATH (via exec.LookPath)
//  3. Known installation locations (searchPaths)
//
// Parameters:
//   - name: the binary name (e.g., "bd", "opencode")
//   - envVarName: optional environment variable to check first (e.g., "BD_BIN", "OPENCODE_BIN")
//     Pass empty string to skip env var check
//   - searchPaths: list of paths to check, typically from CommonSearchPaths()
//
// Returns:
//   - Absolute path to the binary if found
//   - Error with list of searched locations if not found
//
// Example:
//
//	path, err := ResolveBinary("opencode", "OPENCODE_BIN", CommonSearchPaths("opencode"))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	cmd := exec.Command(path, "serve", "--port", "4096")
func ResolveBinary(name string, envVarName string, searchPaths []string) (string, error) {
	var searchedLocations []string

	// 1. Check environment variable override (explicit user preference)
	if envVarName != "" {
		if envPath := os.Getenv(envVarName); envPath != "" {
			// Expand $HOME if present
			envPath = expandHome(envPath)

			// Verify the path exists
			if _, err := os.Stat(envPath); err == nil {
				absPath, err := filepath.Abs(envPath)
				if err != nil {
					absPath = envPath // Use as-is if Abs fails
				}
				return absPath, nil
			}
			searchedLocations = append(searchedLocations, fmt.Sprintf("%s=%s", envVarName, envPath))
		}
	}

	// 2. Check PATH (fast path for normal environments)
	if path, err := exec.LookPath(name); err == nil {
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path // Use as-is if Abs fails
		}
		return absPath, nil
	}
	searchedLocations = append(searchedLocations, "PATH")

	// 3. Check known installation locations
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE") // Windows fallback
	}

	for _, searchPath := range searchPaths {
		// Expand $HOME
		expanded := strings.Replace(searchPath, "$HOME", home, 1)

		if _, err := os.Stat(expanded); err == nil {
			return expanded, nil
		}
		searchedLocations = append(searchedLocations, expanded)
	}

	// Not found anywhere - return helpful error
	return "", fmt.Errorf("%s executable not found. Searched:\n  - %s\n\nEnsure %s is installed or set %s environment variable",
		name,
		strings.Join(searchedLocations, "\n  - "),
		name,
		envVarName,
	)
}

// expandHome replaces $HOME with the actual home directory path.
func expandHome(path string) string {
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE") // Windows fallback
	}
	return strings.Replace(path, "$HOME", home, 1)
}
