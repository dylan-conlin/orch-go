// Package main provides the CLI entry point for orch-go.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/focus"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/spf13/cobra"
)

// ============================================================================
// Handoff Command - Generate session handoff document
// ============================================================================

var handoffCmd = &cobra.Command{
	Use:   "handoff",
	Short: "Generate a session handoff document",
	Long: `Generate a session handoff document capturing the current orchestration state.

The handoff document is useful for:
- Ending a work session and resuming later
- Handing off to another orchestrator
- Creating a checkpoint of multi-project work

The command aggregates:
- Current focus and drift status
- Active agents (from OpenCode and tmux)
- Pending beads issues
- Recent completions
- Local state (uncommitted changes)

Examples:
  orch-go handoff                    # Generate handoff to stdout
  orch-go handoff -o .orch/          # Write to .orch/SESSION_HANDOFF.md
  orch-go handoff --json             # Output data as JSON (for scripting)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHandoff()
	},
}

var (
	handoffOutput string // Output directory or file path
	handoffJSON   bool   // Output as JSON instead of markdown
)

func init() {
	handoffCmd.Flags().StringVarP(&handoffOutput, "output", "o", "", "Output directory (writes SESSION_HANDOFF.md) or file path")
	handoffCmd.Flags().BoolVar(&handoffJSON, "json", false, "Output as JSON for scripting")
	rootCmd.AddCommand(handoffCmd)
}

// HandoffData contains all the data needed to generate a handoff document.
type HandoffData struct {
	Date          string           `json:"date"`
	TLDR          string           `json:"tldr,omitempty"`
	Focus         *FocusInfo       `json:"focus,omitempty"`
	ActiveAgents  []ActiveAgent    `json:"active_agents"`
	PendingIssues []PendingIssue   `json:"pending_issues"`
	RecentWork    []RecentWorkItem `json:"recent_work"`
	LocalState    *LocalStateInfo  `json:"local_state,omitempty"`
	NextPriority  []string         `json:"next_priorities"`
}

// FocusInfo contains current focus information.
type FocusInfo struct {
	Goal      string `json:"goal"`
	BeadsID   string `json:"beads_id,omitempty"`
	IsDrifted bool   `json:"is_drifted"`
}

// ActiveAgent represents a currently running agent.
type ActiveAgent struct {
	BeadsID   string `json:"beads_id"`
	Repo      string `json:"repo"`
	Task      string `json:"task"`
	SessionID string `json:"session_id,omitempty"`
	Window    string `json:"window,omitempty"`
}

// PendingIssue represents a beads issue pending work.
type PendingIssue struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Priority string `json:"priority,omitempty"`
}

// RecentWorkItem represents recently completed work.
type RecentWorkItem struct {
	Type        string `json:"type"` // "completed", "pr", "decision"
	Description string `json:"description"`
	Repo        string `json:"repo,omitempty"`
}

// LocalStateInfo contains information about local uncommitted changes.
type LocalStateInfo struct {
	HasUncommitted bool   `json:"has_uncommitted"`
	Branch         string `json:"branch,omitempty"`
	Summary        string `json:"summary,omitempty"`
}

func runHandoff() error {
	// Gather handoff data
	data, err := gatherHandoffData()
	if err != nil {
		return fmt.Errorf("failed to gather handoff data: %w", err)
	}

	// Output as JSON if requested
	if handoffJSON {
		output, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
		return nil
	}

	// Generate markdown
	markdown, err := generateHandoffMarkdown(data)
	if err != nil {
		return fmt.Errorf("failed to generate markdown: %w", err)
	}

	// Write to file if output specified
	if handoffOutput != "" {
		outputPath := handoffOutput
		// If output is a directory, append filename
		if info, err := os.Stat(handoffOutput); err == nil && info.IsDir() {
			outputPath = filepath.Join(handoffOutput, "SESSION_HANDOFF.md")
		}

		if err := os.WriteFile(outputPath, []byte(markdown), 0644); err != nil {
			return fmt.Errorf("failed to write handoff file: %w", err)
		}

		fmt.Printf("Handoff document written to: %s\n", outputPath)

		// Log the handoff event
		logger := events.NewLogger(events.DefaultLogPath())
		event := events.Event{
			Type:      "handoff.created",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"output_path":   outputPath,
				"active_agents": len(data.ActiveAgents),
				"pending":       len(data.PendingIssues),
			},
		}
		if err := logger.Log(event); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
		}

		return nil
	}

	// Print to stdout
	fmt.Println(markdown)
	return nil
}

func gatherHandoffData() (*HandoffData, error) {
	now := time.Now()
	data := &HandoffData{
		Date:          now.Format("2 Jan 2006"),
		ActiveAgents:  []ActiveAgent{},
		PendingIssues: []PendingIssue{},
		RecentWork:    []RecentWorkItem{},
		NextPriority:  []string{},
	}

	// Get current project directory
	projectDir, _ := os.Getwd()
	projectName := filepath.Base(projectDir)

	// Get focus info
	if focusStore, err := focus.New(""); err == nil {
		if f := focusStore.Get(); f != nil {
			activeIssues := getActiveIssues()
			drift := focusStore.CheckDrift(activeIssues)
			data.Focus = &FocusInfo{
				Goal:      f.Goal,
				BeadsID:   f.BeadsID,
				IsDrifted: drift.IsDrifting,
			}
		}
	}

	// Get active agents from OpenCode and tmux
	data.ActiveAgents = gatherActiveAgents(projectDir)

	// Get pending issues from beads
	data.PendingIssues = gatherPendingIssues()

	// Get recent work (completed issues from beads)
	data.RecentWork = gatherRecentWork()

	// Get local state (uncommitted changes)
	data.LocalState = gatherLocalState(projectDir)

	// Derive next priorities from focus and pending issues
	data.NextPriority = deriveNextPriorities(data)

	// Generate TLDR
	data.TLDR = generateTLDR(data, projectName)

	return data, nil
}

func gatherActiveAgents(projectDir string) []ActiveAgent {
	var agents []ActiveAgent
	projectName := filepath.Base(projectDir)

	// Collect from tmux windows
	workersSessions, _ := tmux.ListWorkersSessions()
	for _, sessionName := range workersSessions {
		windows, _ := tmux.ListWindows(sessionName)
		for _, w := range windows {
			// Skip "servers" and "zsh" windows
			if w.Name == "servers" || w.Name == "zsh" {
				continue
			}

			beadsID := extractBeadsIDFromWindowName(w.Name)
			if beadsID == "" {
				continue
			}

			// Extract repo from session name (format: workers-{repo})
			repo := projectName
			if strings.HasPrefix(sessionName, "workers-") {
				repo = strings.TrimPrefix(sessionName, "workers-")
			}

			// Get task description from window name
			task := w.Name
			if idx := strings.Index(task, "["); idx > 0 {
				task = strings.TrimSpace(task[:idx])
			}

			agents = append(agents, ActiveAgent{
				BeadsID: beadsID,
				Repo:    repo,
				Task:    task,
				Window:  w.Target,
			})
		}
	}

	// Also check OpenCode sessions for headless agents
	client := opencode.NewClient(serverURL)
	sessions, _ := client.ListSessions("")
	sessionSet := make(map[string]bool)
	for _, a := range agents {
		if a.BeadsID != "" {
			sessionSet[a.BeadsID] = true
		}
	}

	for _, s := range sessions {
		beadsID := extractBeadsIDFromTitle(s.Title)
		if beadsID == "" || sessionSet[beadsID] {
			continue
		}

		// Only include recent sessions (last 30 min)
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		if time.Since(updatedAt) > 30*time.Minute {
			continue
		}

		agents = append(agents, ActiveAgent{
			BeadsID:   beadsID,
			Repo:      projectName,
			Task:      s.Title,
			SessionID: s.ID,
		})
	}

	return agents
}

func gatherPendingIssues() []PendingIssue {
	var issues []PendingIssue

	// Run bd ready to get pending issues
	cmd := exec.Command("bd", "ready")
	output, err := cmd.Output()
	if err != nil {
		return issues
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines, headers, and "No" messages
		if line == "" || strings.HasPrefix(line, "📋") || strings.HasPrefix(line, "No ") {
			continue
		}

		// Parse lines like "1. [P0] issue-id: title..."
		if len(line) >= 3 && line[0] >= '0' && line[0] <= '9' && line[1] == '.' {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				priority := strings.Trim(parts[1], "[]")
				issueID := strings.TrimSuffix(parts[2], ":")
				title := strings.Join(parts[3:], " ")

				issues = append(issues, PendingIssue{
					ID:       issueID,
					Title:    title,
					Priority: priority,
				})
			}
		}
	}

	return issues
}

func gatherRecentWork() []RecentWorkItem {
	var work []RecentWorkItem

	// Get recently closed issues from beads (today)
	cmd := exec.Command("bd", "list", "--status", "closed")
	output, err := cmd.Output()
	if err != nil {
		return work
	}

	// Parse closed issues (limited to most recent 5)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	count := 0
	for _, line := range lines {
		if count >= 5 {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "📋") || strings.HasPrefix(line, "No ") {
			continue
		}

		// Extract issue info
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				work = append(work, RecentWorkItem{
					Type:        "completed",
					Description: strings.TrimSpace(parts[1]),
				})
				count++
			}
		}
	}

	return work
}

func gatherLocalState(projectDir string) *LocalStateInfo {
	state := &LocalStateInfo{}

	// Get current branch
	branchCmd := exec.Command("git", "branch", "--show-current")
	branchCmd.Dir = projectDir
	if output, err := branchCmd.Output(); err == nil {
		state.Branch = strings.TrimSpace(string(output))
	}

	// Check for uncommitted changes
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = projectDir
	if output, err := statusCmd.Output(); err == nil {
		changes := strings.TrimSpace(string(output))
		if changes != "" {
			state.HasUncommitted = true
			// Count changes
			lines := strings.Split(changes, "\n")
			state.Summary = fmt.Sprintf("%d uncommitted changes", len(lines))
		}
	}

	return state
}

func deriveNextPriorities(data *HandoffData) []string {
	var priorities []string

	// Priority 1: Check any active agents
	if len(data.ActiveAgents) > 0 {
		for _, agent := range data.ActiveAgents[:min(2, len(data.ActiveAgents))] {
			priorities = append(priorities, fmt.Sprintf("Check %s - %s", agent.BeadsID, truncatePriority(agent.Task)))
		}
	}

	// Priority 2: Start work on pending P0/P1 issues
	for _, issue := range data.PendingIssues {
		if issue.Priority == "P0" || issue.Priority == "P1" {
			priorities = append(priorities, fmt.Sprintf("[%s] %s - %s", issue.Priority, issue.ID, truncatePriority(issue.Title)))
			if len(priorities) >= 3 {
				break
			}
		}
	}

	// If still room, add focus-related priority
	if len(priorities) < 3 && data.Focus != nil && data.Focus.BeadsID != "" {
		priorities = append(priorities, fmt.Sprintf("Focus on %s", data.Focus.Goal))
	}

	return priorities
}

func truncatePriority(s string) string {
	if len(s) > 50 {
		return s[:47] + "..."
	}
	return s
}

func generateTLDR(data *HandoffData, projectName string) string {
	var parts []string

	// Active agents
	if len(data.ActiveAgents) > 0 {
		parts = append(parts, fmt.Sprintf("%d active agent(s)", len(data.ActiveAgents)))
	}

	// Pending issues
	if len(data.PendingIssues) > 0 {
		p0Count := 0
		for _, issue := range data.PendingIssues {
			if issue.Priority == "P0" {
				p0Count++
			}
		}
		if p0Count > 0 {
			parts = append(parts, fmt.Sprintf("%d P0 issue(s) pending", p0Count))
		}
	}

	// Focus
	if data.Focus != nil {
		if data.Focus.IsDrifted {
			parts = append(parts, "drifted from focus")
		} else {
			parts = append(parts, fmt.Sprintf("focused on: %s", truncatePriority(data.Focus.Goal)))
		}
	}

	// Local state
	if data.LocalState != nil && data.LocalState.HasUncommitted {
		parts = append(parts, data.LocalState.Summary)
	}

	if len(parts) == 0 {
		return fmt.Sprintf("Session handoff for %s. No active work.", projectName)
	}

	return strings.Join(parts, ". ") + "."
}

// Handoff template for markdown generation
const handoffTemplate = `# Session Handoff - {{.Date}}

## TLDR

{{.TLDR}}

---

## What Happened This Session

### Work Completed
{{- if .RecentWork}}
{{- range .RecentWork}}
- {{.Description}}
{{- end}}
{{- else}}
*(No completed work recorded)*
{{- end}}

---

## Agents Still Running

{{- if .ActiveAgents}}
| Agent | Repo | Task |
|-------|------|------|
{{- range .ActiveAgents}}
| **{{.BeadsID}}** | {{.Repo}} | {{.Task}} |
{{- end}}

*(Use ` + "`orch status`" + ` to check current agent states)*
{{- else}}
*(No active agents)*
{{- end}}

---

{{- if .LocalState}}
{{- if .LocalState.HasUncommitted}}

## Local State

**Branch:** {{.LocalState.Branch}}
**Status:** {{.LocalState.Summary}}

Check with:
` + "```bash" + `
git status
` + "```" + `

{{- end}}
{{- end}}

---

## Next Session Priorities

{{- if .NextPriority}}
{{- range $i, $p := .NextPriority}}
{{add $i 1}}. **{{$p}}**
{{- end}}
{{- else}}
1. Review pending issues
2. Set focus for session
{{- end}}

---

## Quick Commands

` + "```bash" + `
# Check active agents
orch status

# See pending issues
bd ready

{{- if .Focus}}
# Check drift from focus
orch drift
{{- end}}

# Resume agent work
orch resume <beads-id>
` + "```" + `

---

## Session Metadata

**Generated:** {{.Date}}
{{- if .Focus}}
**Focus:** {{.Focus.Goal}}
{{- end}}
**Active agents:** {{len .ActiveAgents}}
**Pending issues:** {{len .PendingIssues}}
`

func generateHandoffMarkdown(data *HandoffData) (string, error) {
	// Template functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}

	tmpl, err := template.New("handoff").Funcs(funcMap).Parse(handoffTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
