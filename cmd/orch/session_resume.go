// Package main provides the CLI entry point for orch-go.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/tmux"
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

		// Look for Duration line: **Duration:** YYYY-MM-DD HH:MM → YYYY-MM-DD HH:MM or YYYY-MM-DD HH:MM → HH:MM (NNm)
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

			// Parse start time
			layout := "2006-01-02 15:04"
			startTime, err := time.Parse(layout, startStr)
			if err != nil {
				return -1
			}

			// Try parsing end time with full format first
			endTime, err := time.Parse(layout, endStr)
			if err != nil {
				// Try time-only format (same-day sessions): "HH:MM" or "HH:MM (NNm)"
				// Strip any duration annotation like "(38m)"
				endTimeOnly := strings.Split(endStr, " ")[0]
				endTime, err = time.Parse("15:04", endTimeOnly)
				if err != nil {
					// End time may contain placeholder like "{end-time}" if session is incomplete
					return -1
				}
				// Apply the same date as start time for same-day sessions
				endTime = time.Date(startTime.Year(), startTime.Month(), startTime.Day(),
					endTime.Hour(), endTime.Minute(), 0, 0, startTime.Location())
			}

			// Calculate duration in minutes
			duration := endTime.Sub(startTime)
			return int(duration.Minutes())
		}
	}

	return -1 // Duration line not found
}

// scanAllWindowsForMostRecent scans all window-scoped session directories in .orch/session/
// and returns the most recent SESSION_HANDOFF.md by comparing timestamps.
// Prefers substantive sessions (≥5 minutes) over brief test sessions.
// Returns the full path to the most recent handoff, or error if none found.
func scanAllWindowsForMostRecent(sessionBaseDir string) (string, error) {
	// Read all entries in .orch/session/
	entries, err := os.ReadDir(sessionBaseDir)
	if err != nil {
		return "", err
	}

	// Track two candidates:
	// - mostRecentSubstantive: sessions ≥5 minutes (real work sessions)
	// - mostRecentAny: all sessions regardless of duration (fallback)
	const minSubstantiveMinutes = 5

	var mostRecentSubstantivePath string
	var mostRecentSubstantiveTimestamp string
	var mostRecentAnyPath string
	var mostRecentAnyTimestamp string

	// Scan each window directory
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		windowName := entry.Name()

		// Skip legacy timestamp directories (start with digit) and special directories
		if len(windowName) > 0 && windowName[0] >= '0' && windowName[0] <= '9' {
			continue
		}
		if windowName == "latest" || windowName == "active" {
			continue
		}

		// Check for latest symlink in this window's directory
		latestPath := filepath.Join(sessionBaseDir, windowName, "latest")
		stat, err := os.Lstat(latestPath)
		if err != nil {
			continue // No latest symlink for this window
		}

		// Resolve the symlink to get the timestamp directory
		var sessionDir string
		if stat.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(latestPath)
			if err != nil {
				continue
			}
			if !filepath.IsAbs(target) {
				sessionDir = filepath.Join(sessionBaseDir, windowName, target)
			} else {
				sessionDir = target
			}
		} else {
			sessionDir = latestPath
		}

		// Check if SESSION_HANDOFF.md exists
		handoffPath := filepath.Join(sessionDir, "SESSION_HANDOFF.md")
		if _, err := os.Stat(handoffPath); err != nil {
			continue // No handoff in this session directory
		}

		// Extract timestamp from directory name for comparison
		// Format: YYYY-MM-DD-HHMM (e.g., "2026-01-13-0830")
		timestamp := filepath.Base(sessionDir)

		// Always track mostRecentAny (fallback candidate)
		if timestamp > mostRecentAnyTimestamp {
			mostRecentAnyTimestamp = timestamp
			mostRecentAnyPath = handoffPath
		}

		// Parse duration to determine if this is a substantive session
		durationMinutes := parseDurationFromHandoff(handoffPath)
		if durationMinutes >= minSubstantiveMinutes {
			// This is a substantive session (≥5 minutes)
			if timestamp > mostRecentSubstantiveTimestamp {
				mostRecentSubstantiveTimestamp = timestamp
				mostRecentSubstantivePath = handoffPath
			}
		}
	}

	// Prefer substantive sessions over brief test sessions
	if mostRecentSubstantivePath != "" {
		return mostRecentSubstantivePath, nil
	}
	return mostRecentAnyPath, nil
}

// discoverSessionHandoff discovers SESSION_HANDOFF.md by walking up the directory tree.
// Discovery priority:
//  1. Current window's active/ (mid-session resume)
//  2. Cross-window scan for most recent handoff (prefers window-scoped)
//  3. Legacy non-window-scoped structure (backward compatibility)
//
// Returns the full path to the handoff, or error if not found.
func discoverSessionHandoff() (string, error) {
	// Get current tmux window name (or "default" if not in tmux)
	windowName, err := tmux.GetCurrentWindowName()
	if err != nil {
		windowName = "default"
	}

	// Start from current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Walk up directory tree
	dir := currentDir
	for {
		// PRIORITY 1: Current window's active/ (mid-session resume)
		activePath := filepath.Join(dir, ".orch", "session", windowName, "active", "SESSION_HANDOFF.md")
		if _, err := os.Stat(activePath); err == nil {
			return activePath, nil
		}

		// PRIORITY 2: Cross-window scan for most recent handoff
		sessionBaseDir := filepath.Join(dir, ".orch", "session")
		if _, err := os.Stat(sessionBaseDir); err == nil {
			mostRecentPath, err := scanAllWindowsForMostRecent(sessionBaseDir)
			if err == nil && mostRecentPath != "" {
				return mostRecentPath, nil
			}
		}

		// PRIORITY 3: Legacy non-window-scoped structure (backward compatibility)
		legacyLatestPath := filepath.Join(dir, ".orch", "session", "latest")
		if stat, err := os.Lstat(legacyLatestPath); err == nil {
			var sessionDir string
			if stat.Mode()&os.ModeSymlink != 0 {
				target, err := os.Readlink(legacyLatestPath)
				if err == nil {
					if !filepath.IsAbs(target) {
						sessionDir = filepath.Join(dir, ".orch", "session", target)
					} else {
						sessionDir = target
					}
				}
			} else {
				sessionDir = legacyLatestPath
			}

			if sessionDir != "" {
				handoffPath := filepath.Join(sessionDir, "SESSION_HANDOFF.md")
				if _, err := os.Stat(handoffPath); err == nil {
					return handoffPath, nil
				}
			}
		}

		// PROJECT BOUNDARY CHECK: Don't cross project boundaries
		orchDir := filepath.Join(dir, ".orch")
		if _, err := os.Stat(orchDir); err == nil {
			break
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no SESSION_HANDOFF.md found (walked up from %s)", currentDir)
}
