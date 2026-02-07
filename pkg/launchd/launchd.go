// Package launchd provides utilities for querying macOS launchd agents.
package launchd

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"howett.net/plist"
)

// Agent represents a launchd agent with its configuration and status.
type Agent struct {
	Label        string
	PlistPath    string
	Loaded       bool
	Running      bool
	PID          int
	LastExitCode int
	Schedule     string // Human-readable schedule description
	RunAtLoad    bool
	KeepAlive    bool
}

// Status returns the agent's status as a human-readable string.
func (a *Agent) Status() string {
	if !a.Loaded {
		return "not loaded"
	}
	if a.Running {
		return fmt.Sprintf("running (PID %d)", a.PID)
	}
	return "idle"
}

// HasFailure returns true if the agent has a non-zero exit code.
func (a *Agent) HasFailure() bool {
	return a.Loaded && a.LastExitCode != 0
}

// plistData represents the relevant fields from a launchd plist file.
type plistData struct {
	Label                 string                 `plist:"Label"`
	RunAtLoad             bool                   `plist:"RunAtLoad"`
	KeepAlive             interface{}            `plist:"KeepAlive"` // Can be bool or dict
	StartCalendarInterval interface{}            `plist:"StartCalendarInterval"`
	StartInterval         int                    `plist:"StartInterval"`
	WatchPaths            []string               `plist:"WatchPaths"`
	QueueDirectories      []string               `plist:"QueueDirectories"`
}

// calendarInterval represents a StartCalendarInterval entry.
type calendarInterval struct {
	Minute  *int `plist:"Minute"`
	Hour    *int `plist:"Hour"`
	Day     *int `plist:"Day"`
	Weekday *int `plist:"Weekday"`
	Month   *int `plist:"Month"`
}

// ParsePlist reads a launchd plist file and extracts relevant configuration.
func ParsePlist(path string) (*Agent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read plist: %w", err)
	}

	var pd plistData
	if _, err := plist.Unmarshal(data, &pd); err != nil {
		return nil, fmt.Errorf("parse plist: %w", err)
	}

	agent := &Agent{
		Label:     pd.Label,
		PlistPath: path,
		RunAtLoad: pd.RunAtLoad,
	}

	// Parse KeepAlive - can be bool or dict
	switch v := pd.KeepAlive.(type) {
	case bool:
		agent.KeepAlive = v
	case map[string]interface{}:
		// If KeepAlive is a dict with SuccessfulExit: false, it's essentially KeepAlive
		agent.KeepAlive = true
	}

	// Parse schedule
	agent.Schedule = parseSchedule(pd)

	return agent, nil
}

// parseSchedule converts schedule configuration to human-readable string.
func parseSchedule(pd plistData) string {
	// Check StartCalendarInterval (cron-like)
	if pd.StartCalendarInterval != nil {
		return parseCalendarInterval(pd.StartCalendarInterval)
	}

	// Check StartInterval (every N seconds)
	if pd.StartInterval > 0 {
		return formatInterval(pd.StartInterval)
	}

	// Check WatchPaths/QueueDirectories (file-triggered)
	if len(pd.WatchPaths) > 0 || len(pd.QueueDirectories) > 0 {
		return "file-triggered"
	}

	// RunAtLoad only
	if pd.RunAtLoad {
		return "on load"
	}

	return "manual"
}

// parseCalendarInterval handles StartCalendarInterval which can be a dict or array of dicts.
func parseCalendarInterval(v interface{}) string {
	// Try single interval (dict)
	if m, ok := v.(map[string]interface{}); ok {
		return formatCalendarInterval(m)
	}

	// Try array of intervals
	if arr, ok := v.([]interface{}); ok && len(arr) > 0 {
		if m, ok := arr[0].(map[string]interface{}); ok {
			desc := formatCalendarInterval(m)
			if len(arr) > 1 {
				return fmt.Sprintf("%s (+%d more)", desc, len(arr)-1)
			}
			return desc
		}
	}

	return "scheduled"
}

// formatCalendarInterval formats a single calendar interval to human-readable string.
func formatCalendarInterval(m map[string]interface{}) string {
	var parts []string

	if hour, ok := m["Hour"].(uint64); ok {
		if minute, ok := m["Minute"].(uint64); ok {
			parts = append(parts, fmt.Sprintf("%02d:%02d", hour, minute))
		} else {
			parts = append(parts, fmt.Sprintf("%02d:00", hour))
		}
	} else if minute, ok := m["Minute"].(uint64); ok {
		parts = append(parts, fmt.Sprintf("*:%02d", minute))
	}

	if weekday, ok := m["Weekday"].(uint64); ok {
		days := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
		if int(weekday) < len(days) {
			parts = append(parts, days[weekday])
		}
	}

	if day, ok := m["Day"].(uint64); ok {
		parts = append(parts, fmt.Sprintf("day %d", day))
	}

	if len(parts) == 0 {
		return "scheduled"
	}

	return strings.Join(parts, " ")
}

// formatInterval converts seconds to human-readable duration.
func formatInterval(seconds int) string {
	switch {
	case seconds < 60:
		return fmt.Sprintf("every %ds", seconds)
	case seconds < 3600:
		return fmt.Sprintf("every %dm", seconds/60)
	case seconds < 86400:
		return fmt.Sprintf("every %dh", seconds/3600)
	default:
		return fmt.Sprintf("every %dd", seconds/86400)
	}
}

// GetStatus queries launchctl for agent status information.
func GetStatus(label string) (loaded bool, running bool, pid int, exitCode int, err error) {
	uid := os.Getuid()

	// Use launchctl list to check if loaded and get basic status
	cmd := exec.Command("launchctl", "list")
	output, err := cmd.Output()
	if err != nil {
		return false, false, 0, 0, fmt.Errorf("launchctl list: %w", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		// Format: PID\tStatus\tLabel
		// PID is "-" if not running
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[2] == label {
			loaded = true

			// Parse PID
			if fields[0] != "-" {
				pid, _ = strconv.Atoi(fields[0])
				running = pid > 0
			}

			// Parse exit code
			exitCode, _ = strconv.Atoi(fields[1])

			return loaded, running, pid, exitCode, nil
		}
	}

	// Not found in list - check if we can print it (might be loaded but idle)
	printCmd := exec.Command("launchctl", "print", fmt.Sprintf("gui/%d/%s", uid, label))
	if err := printCmd.Run(); err == nil {
		// It exists in launchctl print but not in list - edge case
		loaded = true
	}

	return loaded, false, 0, 0, nil
}

// ScanOptions configures which agents to scan.
type ScanOptions struct {
	Directory string   // Default: ~/Library/LaunchAgents
	Prefixes  []string // Default: com.dylan., com.user., com.orch., com.cdd.
}

// DefaultScanOptions returns the default scan options.
func DefaultScanOptions() ScanOptions {
	homeDir, _ := os.UserHomeDir()
	return ScanOptions{
		Directory: filepath.Join(homeDir, "Library", "LaunchAgents"),
		Prefixes:  []string{"com.dylan.", "com.user.", "com.orch.", "com.cdd."},
	}
}

// Scan discovers and returns information about custom launchd agents.
func Scan(opts ScanOptions) ([]Agent, error) {
	if opts.Directory == "" {
		opts = DefaultScanOptions()
	}

	entries, err := os.ReadDir(opts.Directory)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}

	var agents []Agent
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".plist") {
			continue
		}

		// Check if name matches any prefix
		matched := false
		for _, prefix := range opts.Prefixes {
			if strings.HasPrefix(entry.Name(), prefix) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}

		// Parse plist
		plistPath := filepath.Join(opts.Directory, entry.Name())
		agent, err := ParsePlist(plistPath)
		if err != nil {
			// Log but don't fail on individual parse errors
			continue
		}

		// Get runtime status
		loaded, running, pid, exitCode, err := GetStatus(agent.Label)
		if err == nil {
			agent.Loaded = loaded
			agent.Running = running
			agent.PID = pid
			agent.LastExitCode = exitCode
		}

		agents = append(agents, *agent)
	}

	return agents, nil
}
