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

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/focus"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/spf13/cobra"
)

// ============================================================================
// Handoff Command - Generate synthesis document
// ============================================================================

var handoffCmd = &cobra.Command{
	Use:   "handoff",
	Short: "Generate a synthesis document",
	Long: `Generate a synthesis document capturing the current orchestration state.

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
  orch-go handoff -o .orch/          # Write to .orch/SYNTHESIS.md
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
	handoffCmd.Flags().StringVarP(&handoffOutput, "output", "o", "", "Output directory (writes SYNTHESIS.md) or file path")
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

	// D.E.K.N. sections - hybrid auto-generated + human-authored
	DEKN *DEKNSummary `json:"dekn,omitempty"`

	// GitStats for auto-populating Evidence
	GitStats *GitStats `json:"git_stats,omitempty"`
}

// DEKNSummary contains the Delta, Evidence, Knowledge, Next sections.
// These are prompts for the human/orchestrator to fill in before handoff.
type DEKNSummary struct {
	Delta     string `json:"delta"`     // What changed this session
	Evidence  string `json:"evidence"`  // Proof of work (commits, tests, validation)
	Knowledge string `json:"knowledge"` // What was learned
	Next      string `json:"next"`      // Recommended next actions
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

// GitStats contains git statistics for the session.
type GitStats struct {
	CommitCount  int    `json:"commit_count"`
	LinesAdded   int    `json:"lines_added"`
	LinesRemoved int    `json:"lines_removed"`
	Summary      string `json:"summary"` // Human-readable summary
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
		// Validate D.E.K.N. sections when writing to file
		if err := validateDEKN(data.DEKN); err != nil {
			return fmt.Errorf("D.E.K.N. validation failed: %w\n\nTo generate a draft without D.E.K.N. content, omit the -o flag and review the output.\nFill in the D.E.K.N. Summary section with actual session details before saving.", err)
		}

		outputPath := handoffOutput
		// If output is a directory, append filename
		if info, err := os.Stat(handoffOutput); err == nil && info.IsDir() {
			outputPath = filepath.Join(handoffOutput, "SYNTHESIS.md")
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

// validateDEKN validates that D.E.K.N. synthesis sections are filled in.
// Only gates on Knowledge and Next (synthesis sections) - Delta and Evidence can be auto-populated.
// Returns an error if synthesis sections are empty or still contain placeholder text.
func validateDEKN(dekn *DEKNSummary) error {
	if dekn == nil {
		return fmt.Errorf("D.E.K.N. summary is required when writing to file")
	}

	var missing []string

	// Only require synthesis sections (Knowledge and Next)
	// Delta and Evidence can be auto-populated from git stats and recent work
	if isDEKNPlaceholder(dekn.Knowledge) {
		missing = append(missing, "Knowledge")
	}
	if isDEKNPlaceholder(dekn.Next) {
		missing = append(missing, "Next")
	}

	if len(missing) > 0 {
		return fmt.Errorf("synthesis sections require human input: %s\n\nThese sections capture what was LEARNED and what to do NEXT - data alone can't tell this story.\nFill in these sections to complete the handoff.", strings.Join(missing, ", "))
	}

	return nil
}

// isDEKNPlaceholder returns true if the text is empty or contains placeholder markers.
func isDEKNPlaceholder(text string) bool {
	text = strings.TrimSpace(text)

	// Empty is definitely placeholder
	if text == "" {
		return true
	}

	// Check for common placeholder patterns
	placeholderPatterns := []string{
		"[",                    // Bracketed placeholder like "[What changed...]"
		"TODO",                 // TODO markers
		"FILL IN",              // Explicit fill-in markers
		"SYNTHESIS REQUIRED",   // New synthesis prompt marker
		"describe the",         // Part of default prompt text
		"Proof of work",        // Part of default prompt text
		"What was learned",     // Part of default prompt text
		"Recommended next",     // Part of default prompt text
		"what patterns",        // Part of Knowledge prompt
		"what should the next", // Part of Next prompt
	}

	textLower := strings.ToLower(text)
	for _, pattern := range placeholderPatterns {
		if strings.Contains(textLower, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

func gatherHandoffData() (*HandoffData, error) {
	now := time.Now()
	data := &HandoffData{
		Date:          now.Format("2 Jan 2006"),
		ActiveAgents:  []ActiveAgent{},
		PendingIssues: []PendingIssue{},
		RecentWork:    []RecentWorkItem{},
		NextPriority:  []string{},
		DEKN:          &DEKNSummary{}, // Initialize empty for prompts
	}

	// Get current project directory
	projectDir, _ := currentProjectDir()
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

	// Get git stats for auto-populating Evidence
	data.GitStats = gatherGitStats(projectDir)

	// Derive next priorities from focus and pending issues
	data.NextPriority = deriveNextPriorities(data)

	// Generate TLDR
	data.TLDR = generateTLDR(data, projectName)

	// Initialize DEKN (will be populated with scaffold in template)
	data.DEKN = &DEKNSummary{}

	return data, nil
}

func gatherActiveAgents(projectDir string) []ActiveAgent {
	return gatherActiveAgentsWithClient(opencode.NewClient(serverURL), projectDir)
}

func gatherActiveAgentsWithClient(client opencode.ClientInterface, projectDir string) []ActiveAgent {
	var agents []ActiveAgent
	projectName := filepath.Base(projectDir)

	// Get in-progress beads IDs for filtering
	inProgressBeads := getInProgressBeadsIDs()

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

			// Only include agents that are actually in_progress in beads
			if !inProgressBeads[beadsID] {
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

		// Only include if in_progress in beads
		if !inProgressBeads[beadsID] {
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

// getInProgressBeadsIDs returns a set of beads IDs that are currently in_progress.
// It uses the beads RPC client when available, falling back to the bd CLI.
func getInProgressBeadsIDs() map[string]bool {
	result := make(map[string]bool)

	err := withBeadsClient("", func(client *beads.Client) error {
		issues, rpcErr := client.List(&beads.ListArgs{Status: "in_progress"})
		if rpcErr != nil {
			return rpcErr
		}
		for _, issue := range issues {
			result[issue.ID] = true
		}
		return nil
	})
	if err == nil {
		return result
	}

	// Fallback to CLI
	issues, err := beads.FallbackList("in_progress")
	if err != nil {
		return result
	}

	for _, issue := range issues {
		result[issue.ID] = true
	}

	return result
}

func gatherPendingIssues() []PendingIssue {
	var issues []PendingIssue

	err := withBeadsClient("", func(client *beads.Client) error {
		readyIssues, rpcErr := client.Ready(nil)
		if rpcErr != nil {
			return rpcErr
		}
		for _, issue := range readyIssues {
			priority := fmt.Sprintf("P%d", issue.Priority)
			issues = append(issues, PendingIssue{
				ID:       issue.ID,
				Title:    issue.Title,
				Priority: priority,
			})
		}
		return nil
	})
	if err == nil {
		return issues
	}

	// Fallback to CLI
	readyIssues, err := beads.FallbackReady()
	if err != nil {
		return issues
	}

	for _, issue := range readyIssues {
		priority := fmt.Sprintf("P%d", issue.Priority)
		issues = append(issues, PendingIssue{
			ID:       issue.ID,
			Title:    issue.Title,
			Priority: priority,
		})
	}

	return issues
}

func gatherRecentWork() []RecentWorkItem {
	var work []RecentWorkItem

	err := withBeadsClient("", func(client *beads.Client) error {
		issues, rpcErr := client.List(&beads.ListArgs{Status: "closed"})
		if rpcErr != nil {
			return rpcErr
		}
		// Limit to most recent 5
		count := 0
		for _, issue := range issues {
			if count >= 5 {
				break
			}
			description := fmt.Sprintf("[%s] %s", issue.ID, issue.Title)
			work = append(work, RecentWorkItem{
				Type:        "completed",
				Description: description,
			})
			count++
		}
		return nil
	})
	if err == nil {
		return work
	}

	// Fallback to CLI
	issues, err := beads.FallbackList("closed")
	if err != nil {
		return work
	}

	// Limit to most recent 5
	count := 0
	for _, issue := range issues {
		if count >= 5 {
			break
		}
		description := fmt.Sprintf("[%s] %s", issue.ID, issue.Title)
		work = append(work, RecentWorkItem{
			Type:        "completed",
			Description: description,
		})
		count++
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

// gatherGitStats collects git statistics for recent commits (today's commits).
func gatherGitStats(projectDir string) *GitStats {
	stats := &GitStats{}

	// Get today's date in git log format
	today := time.Now().Format("2006-01-02")

	// Count commits from today
	commitCountCmd := exec.Command("git", "log", "--oneline", "--since="+today+" 00:00:00", "--format=%H")
	commitCountCmd.Dir = projectDir
	if output, err := commitCountCmd.Output(); err == nil {
		commits := strings.TrimSpace(string(output))
		if commits != "" {
			stats.CommitCount = len(strings.Split(commits, "\n"))
		}
	}

	// Get lines added/removed from today's commits
	if stats.CommitCount > 0 {
		diffStatCmd := exec.Command("git", "diff", "--stat", "--since="+today+" 00:00:00", "HEAD~"+fmt.Sprintf("%d", stats.CommitCount), "HEAD")
		diffStatCmd.Dir = projectDir
		if output, err := diffStatCmd.Output(); err == nil {
			// Parse the summary line (e.g., "10 files changed, 500 insertions(+), 200 deletions(-)")
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			if len(lines) > 0 {
				summaryLine := lines[len(lines)-1]
				stats.LinesAdded, stats.LinesRemoved = parseGitDiffSummary(summaryLine)
			}
		}

		// Alternative: use shortstat for cleaner parsing
		shortStatCmd := exec.Command("git", "diff", "--shortstat", "HEAD~"+fmt.Sprintf("%d", stats.CommitCount), "HEAD")
		shortStatCmd.Dir = projectDir
		if output, err := shortStatCmd.Output(); err == nil {
			stats.LinesAdded, stats.LinesRemoved = parseGitDiffSummary(string(output))
		}
	}

	// Build human-readable summary
	if stats.CommitCount > 0 {
		if stats.LinesAdded > 0 || stats.LinesRemoved > 0 {
			stats.Summary = fmt.Sprintf("%d commits, +%d/-%d lines", stats.CommitCount, stats.LinesAdded, stats.LinesRemoved)
		} else {
			stats.Summary = fmt.Sprintf("%d commits", stats.CommitCount)
		}
	}

	return stats
}

// parseGitDiffSummary extracts insertions and deletions from git diff summary.
func parseGitDiffSummary(summary string) (added, removed int) {
	// Match patterns like "500 insertions(+)" and "200 deletions(-)"
	summary = strings.TrimSpace(summary)

	// Look for insertions
	if idx := strings.Index(summary, "insertion"); idx > 0 {
		// Find the number before "insertion"
		numStr := ""
		for i := idx - 1; i >= 0 && (summary[i] == ' ' || (summary[i] >= '0' && summary[i] <= '9')); i-- {
			if summary[i] >= '0' && summary[i] <= '9' {
				numStr = string(summary[i]) + numStr
			}
		}
		if n, err := fmt.Sscanf(numStr, "%d", &added); n == 0 || err != nil {
			added = 0
		}
	}

	// Look for deletions
	if idx := strings.Index(summary, "deletion"); idx > 0 {
		// Find the number before "deletion"
		numStr := ""
		for i := idx - 1; i >= 0 && (summary[i] == ' ' || (summary[i] >= '0' && summary[i] <= '9')); i-- {
			if summary[i] >= '0' && summary[i] <= '9' {
				numStr = string(summary[i]) + numStr
			}
		}
		if n, err := fmt.Sscanf(numStr, "%d", &removed); n == 0 || err != nil {
			removed = 0
		}
	}

	return added, removed
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

## D.E.K.N. Summary

### Delta (What Changed)
{{- if and .DEKN .DEKN.Delta}}
{{.DEKN.Delta}}
{{- else}}
{{- if .RecentWork}}
**Completed this session:**
{{- range .RecentWork}}
- {{.Description}}
{{- end}}
{{- else}}
[Describe the transformation - what was built, fixed, or improved]
{{- end}}
{{- end}}

### Evidence (Proof of Work)
{{- if and .DEKN .DEKN.Evidence}}
{{.DEKN.Evidence}}
{{- else}}
{{- if .GitStats}}
{{- if .GitStats.Summary}}
**Git stats:** {{.GitStats.Summary}}
{{- end}}
{{- end}}
{{- if .LocalState}}
{{- if .LocalState.HasUncommitted}}
**Local state:** {{.LocalState.Summary}} on branch ` + "`{{.LocalState.Branch}}`" + `
{{- end}}
{{- end}}
**Tests:** [FILL IN: all passing / X failures / not run]
{{- end}}

### Knowledge (What Was Learned)
{{- if and .DEKN .DEKN.Knowledge}}
{{.DEKN.Knowledge}}
{{- else}}
[SYNTHESIS REQUIRED: What patterns, insights, or lessons were discovered this session?]
{{- end}}

### Next (Recommended Actions)
{{- if and .DEKN .DEKN.Next}}
{{.DEKN.Next}}
{{- else}}
{{- if .NextPriority}}
**Auto-suggested from state:**
{{- range $i, $p := .NextPriority}}
{{add $i 1}}. {{$p}}
{{- end}}

[SYNTHESIS REQUIRED: Validate/adjust priorities based on session learnings]
{{- else}}
[SYNTHESIS REQUIRED: What should the next session prioritize?]
{{- end}}
{{- end}}

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
