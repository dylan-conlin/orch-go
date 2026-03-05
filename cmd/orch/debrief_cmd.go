// Package main provides the debrief command for session-end knowledge capture.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/debrief"
	"github.com/dylan-conlin/orch-go/pkg/focus"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/spf13/cobra"
)

var (
	debriefChanged string
	debriefNext    string
	debriefJSON    bool
	debriefDryRun  bool
)

var debriefCmd = &cobra.Command{
	Use:   "debrief [focus]",
	Short: "Generate session debrief with auto-populated sections",
	Long: `Generate a durable session debrief at .kb/sessions/YYYY-MM-DD-debrief.md.

Auto-populates from:
  - events.jsonl: completions, spawns, abandonments
  - bd list --status=in_progress: in-flight work
  - bd ready: ready issues for what's next
  - orch session: session duration and goal

Override or supplement sections with flags:
  --changed "we decided X because Y"
  --next "integrate debrief into orient;ship snap MVP"

Semicolons separate multiple items in --changed and --next.

If a debrief already exists for today, it will be overwritten.

Examples:
  orch debrief                              # Auto-populate everything
  orch debrief "Ship snap MVP"              # Set focus explicitly
  orch debrief --changed "decided to use JWT"
  orch debrief --next "fix auth;ship snap"
  orch debrief --dry-run                    # Preview without writing
  orch debrief --json                       # Output data as JSON`,
	RunE: func(cmd *cobra.Command, args []string) error {
		focusOverride := ""
		if len(args) > 0 {
			focusOverride = strings.Join(args, " ")
		}
		return runDebrief(focusOverride)
	},
}

func init() {
	debriefCmd.Flags().StringVar(&debriefChanged, "changed", "", "What changed this session (semicolon-separated)")
	debriefCmd.Flags().StringVar(&debriefNext, "next", "", "What's next (semicolon-separated)")
	debriefCmd.Flags().BoolVar(&debriefJSON, "json", false, "Output as JSON")
	debriefCmd.Flags().BoolVar(&debriefDryRun, "dry-run", false, "Preview output without writing file")
}

func runDebrief(focusOverride string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	now := time.Now()
	dateStr := now.Format("2006-01-02")

	// Ensure .kb/sessions/ exists
	sessionsDir := filepath.Join(projectDir, ".kb", "sessions")
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		return fmt.Errorf(".kb/sessions/ directory not found — run: mkdir -p .kb/sessions")
	}

	data := &debrief.DebriefData{
		Date: dateStr,
	}

	// 1. Focus: flag > session goal > fallback
	data.Focus = collectDebriefFocus(focusOverride)

	// 2. Duration from session
	data.Duration = collectDebriefDuration()

	// 3. What Happened from events.jsonl
	events := loadDebriefEvents(now)
	data.WhatHappened = debrief.CollectWhatHappened(events)

	// 4. What We Learned: --changed flag + completion reasons from events
	data.WhatWeLearned = collectDebriefChanged(events)

	// 5. What's In Flight from bd list --status=in_progress
	data.InFlight = collectDebriefInFlight()

	// 6. What's Next: --next flag + auto-detected from bd ready
	data.WhatsNext = collectDebriefNext()

	// JSON output mode
	if debriefJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}

	// Render markdown
	output := debrief.RenderDebrief(data)

	// Dry-run: print and exit
	if debriefDryRun {
		fmt.Print(output)
		return nil
	}

	// Write to file
	debriefPath := debrief.DebriefFilePath(projectDir, dateStr)
	if err := os.WriteFile(debriefPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write debrief: %w", err)
	}

	fmt.Printf("Debrief written: %s\n", debriefPath)
	fmt.Printf("  Date:     %s\n", data.Date)
	fmt.Printf("  Focus:    %s\n", data.Focus)
	fmt.Printf("  Learned:  %d item(s)\n", len(data.WhatWeLearned))
	fmt.Printf("  Happened: %d item(s)\n", len(data.WhatHappened))
	fmt.Printf("  In flight: %d item(s)\n", len(data.InFlight))
	fmt.Printf("  Next:     %d item(s)\n", len(data.WhatsNext))

	return nil
}

// collectDebriefFocus resolves focus from override, session, or focus store.
func collectDebriefFocus(override string) string {
	if override != "" {
		return override
	}

	// Try session goal
	store, err := session.New("")
	if err == nil {
		if sess := store.Get(); sess != nil {
			return sess.Goal
		}
	}

	// Try focus store
	focusStore, err := focus.New("")
	if err == nil {
		if f := focusStore.Get(); f != nil {
			return f.Goal
		}
	}

	return "(not set)"
}

// collectDebriefDuration gets session duration if a session is active.
func collectDebriefDuration() string {
	store, err := session.New("")
	if err != nil {
		return ""
	}
	if !store.IsActive() {
		return ""
	}
	return debrief.FormatDuration(store.Duration())
}

// loadDebriefEvents reads events.jsonl and filters to today.
func loadDebriefEvents(now time.Time) []debrief.SessionEvent {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	eventsPath := filepath.Join(home, ".orch", "events.jsonl")
	file, err := os.Open(eventsPath)
	if err != nil {
		return nil
	}
	defer file.Close()

	var events []debrief.SessionEvent
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var event debrief.SessionEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		events = append(events, event)
	}

	return debrief.FilterEventsToday(events, now)
}

// collectDebriefChanged merges --changed flag with completion reasons from events.
func collectDebriefChanged(events []debrief.SessionEvent) []string {
	var items []string

	// User-provided via --changed flag
	if debriefChanged != "" {
		items = append(items, debrief.ParseMultiValue(debriefChanged)...)
	}

	// Auto-detect: completion reasons from agent.completed events
	items = append(items, debrief.CollectWhatWeLearned(events)...)

	return items
}

// collectDebriefInFlight gets in-progress issues from beads.
func collectDebriefInFlight() []string {
	cmd := exec.Command("bd", "list", "--status=in_progress")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var issues []debrief.InFlightIssue
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.Contains(line, " in_progress ") {
			continue
		}
		// Parse: "orch-go-abc1 [P2] [feature] in_progress Title here"
		parts := strings.Fields(line)
		if len(parts) < 5 {
			continue
		}
		id := parts[0]
		// Find title after "in_progress"
		idx := strings.Index(line, "in_progress ")
		title := ""
		if idx >= 0 {
			title = strings.TrimSpace(line[idx+len("in_progress "):])
		}
		issues = append(issues, debrief.InFlightIssue{
			ID:     id,
			Title:  title,
			Status: "in_progress",
		})
	}

	return debrief.CollectInFlight(issues)
}

// collectDebriefNext merges --next flag with auto-detected ready issues.
func collectDebriefNext() []string {
	var items []string

	// User-provided via --next flag
	if debriefNext != "" {
		items = append(items, debrief.ParseMultiValue(debriefNext)...)
	}

	// Auto-detect: top 3 ready issues from beads
	readyItems := collectReadyForNext()
	items = append(items, readyItems...)

	return items
}

// collectReadyForNext gets top ready issues from bd ready.
func collectReadyForNext() []string {
	cmd := exec.Command("bd", "ready")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	issues := parseBdReadyForOrient(string(output), 3)
	var items []string
	for _, issue := range issues {
		items = append(items, fmt.Sprintf("[%s] %s (%s)", issue.Priority, issue.Title, issue.ID))
	}
	return items
}
