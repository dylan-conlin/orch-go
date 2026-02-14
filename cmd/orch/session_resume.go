// Package main provides the CLI entry point for orch-go.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ============================================================================
// Session Resume - Discover and display prior session handoff
// ============================================================================

// runSessionResume implements the `orch session resume` command.
// It discovers and displays the most recent SESSION_HANDOFF.md for context injection.
func runSessionResume() error {
	// Discover handoff by walking up directory tree
	handoffPath, err := discoverSessionHandoff()
	if err != nil {
		if resumeCheck {
			// Exit code 1 for --check mode when handoff not found
			os.Exit(1)
		}
		return err
	}

	if resumeCheck {
		// Exit code 0 for --check mode when handoff exists
		os.Exit(0)
	}

	// Read the handoff content
	content, err := os.ReadFile(handoffPath)
	if err != nil {
		return fmt.Errorf("failed to read handoff: %w", err)
	}

	// Output based on mode
	if resumeForInjection {
		// Condensed format for hooks (just the content, no decorations)
		fmt.Print(string(content))
	} else {
		// Interactive format with metadata
		fmt.Printf("📋 Session Handoff\n")
		fmt.Printf("   Source: %s\n", handoffPath)
		fmt.Println()
		fmt.Print(string(content))
	}

	return nil
}

// parseDurationFromHandoff reads a SESSION_HANDOFF.md file and extracts the session duration.
// Parses the Duration line in format: "**Duration:** YYYY-MM-DD HH:MM → YYYY-MM-DD HH:MM"
// Returns duration in minutes, or -1 if duration cannot be parsed (unparseable format or incomplete session).
func parseDurationFromHandoff(handoffPath string) int {
	file, err := os.Open(handoffPath)
	if err != nil {
		return -1
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	maxLines := 20 // Duration line is always in the header

	for scanner.Scan() && lineCount < maxLines {
		line := scanner.Text()
		lineCount++

		// Look for Duration line: **Duration:** YYYY-MM-DD HH:MM → YYYY-MM-DD HH:MM
		if strings.HasPrefix(line, "**Duration:**") {
			// Extract the content after "**Duration:** "
			content := strings.TrimPrefix(line, "**Duration:**")
			content = strings.TrimSpace(content)

			// Split by arrow (→) to get start and end timestamps
			parts := strings.Split(content, "→")
			if len(parts) != 2 {
				return -1 // Not in expected format
			}

			startStr := strings.TrimSpace(parts[0])
			endStr := strings.TrimSpace(parts[1])

			// Parse timestamps
			layout := "2006-01-02 15:04"
			startTime, err := time.Parse(layout, startStr)
			if err != nil {
				return -1
			}

			endTime, err := time.Parse(layout, endStr)
			if err != nil {
				// End time may contain placeholder like "{end-time}" if session is incomplete
				return -1
			}

			// Calculate duration in minutes
			duration := endTime.Sub(startTime)
			return int(duration.Minutes())
		}
	}

	return -1 // Duration line not found
}

// scanAllWindowsForMostRecent scans all window directories in .orch/session/
// and finds the most recent SESSION_HANDOFF.md by modification time.
// Returns the full path to the most recent handoff, or error if none found.
func scanAllWindowsForMostRecent(sessionBaseDir string) (string, error) {
	// List all window directories
	entries, err := os.ReadDir(sessionBaseDir)
	if err != nil {
		return "", fmt.Errorf("failed to read session directory: %w", err)
	}

	var mostRecentPath string
	var mostRecentTime time.Time

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		windowName := entry.Name()

		// Check for latest symlink first
		latestPath := filepath.Join(sessionBaseDir, windowName, "latest", "SESSION_HANDOFF.md")
		if info, err := os.Stat(latestPath); err == nil {
			// Found via latest symlink
			if mostRecentPath == "" || info.ModTime().After(mostRecentTime) {
				mostRecentPath = latestPath
				mostRecentTime = info.ModTime()
			}
			continue
		}

		// Fall back to scanning timestamped directories
		windowDir := filepath.Join(sessionBaseDir, windowName)
		timestampedEntries, err := os.ReadDir(windowDir)
		if err != nil {
			continue
		}

		for _, tsEntry := range timestampedEntries {
			if !tsEntry.IsDir() {
				continue
			}

			// Skip "active" directory (not archived yet)
			if tsEntry.Name() == "active" {
				continue
			}

			handoffPath := filepath.Join(windowDir, tsEntry.Name(), "SESSION_HANDOFF.md")
			if info, err := os.Stat(handoffPath); err == nil {
				if mostRecentPath == "" || info.ModTime().After(mostRecentTime) {
					mostRecentPath = handoffPath
					mostRecentTime = info.ModTime()
				}
			}
		}
	}

	if mostRecentPath == "" {
		return "", fmt.Errorf("no SESSION_HANDOFF.md files found in %s", sessionBaseDir)
	}

	return mostRecentPath, nil
}

// discoverSessionHandoff discovers SESSION_HANDOFF.md by walking up the directory tree.
// Discovery order:
//  1. Current directory: .orch/session/latest/SESSION_HANDOFF.md (symlink target)
//  2. Walk up tree looking for .orch/session/latest/SESSION_HANDOFF.md
//  3. If multiple windows exist in .orch/session/, find most recent by modification time
//
// Returns the full path to the handoff, or error if not found.
func discoverSessionHandoff() (string, error) {
	// Start from current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Err("failed to get current directory: %w", err)
	}

	// Walk up directory tree
	searchDir := currentDir
	for {
		// Try .orch/session/latest/SESSION_HANDOFF.md
		latestPath := filepath.Join(searchDir, ".orch", "session", "latest", "SESSION_HANDOFF.md")
		if _, err := os.Stat(latestPath); err == nil {
			return latestPath, nil
		}

		// Try scanning all windows in .orch/session/
		sessionBaseDir := filepath.Join(searchDir, ".orch", "session")
		if info, err := os.Stat(sessionBaseDir); err == nil && info.IsDir() {
			// Session directory exists - scan all windows for most recent
			handoffPath, err := scanAllWindowsForMostRecent(sessionBaseDir)
			if err == nil {
				return handoffPath, nil
			}
			// If scan failed, continue walking up (might find .orch/session at parent level)
		}

		// Move up one directory
		parent := filepath.Dir(searchDir)
		if parent == searchDir {
			// Reached root directory
			break
		}
		searchDir = parent
	}

	return "", fmt.Errorf("no SESSION_HANDOFF.md found (walked up from %s)", currentDir)
}
