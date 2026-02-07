// Package focus provides north star tracking for multi-project prioritization.
package focus

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// BeadsIssue represents an issue from bd ready --json output.
type BeadsIssue struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Priority    int    `json:"priority"`
	IssueType   string `json:"issue_type"`
	Labels      string `json:"labels,omitempty"`
}

// Thread represents a group of related issues.
type Thread struct {
	Name    string       // Human-readable thread name
	Issues  []BeadsIssue // Issues in this thread
	Notes   string       // Brief description or context
	Keyword string       // The keyword that grouped these issues
}

// threadKeyword defines a keyword pattern for thread grouping.
type threadKeyword struct {
	keyword string // The keyword to match (lowercase)
	name    string // Human-readable thread name
}

// threadKeywords defines the priority-ordered keywords for thread detection.
// Earlier keywords take precedence when multiple match.
var threadKeywords = []threadKeyword{
	{keyword: "session", name: "Session tooling"},
	{keyword: "model", name: "Model system"},
	{keyword: "dashboard", name: "Dashboard"},
	{keyword: "spawn", name: "Spawn system"},
	{keyword: "daemon", name: "Daemon"},
	{keyword: "kb", name: "Knowledge base"},
	{keyword: "orch", name: "Orch tooling"},
	{keyword: "beads", name: "Beads integration"},
	{keyword: "doctor", name: "Orch tooling"}, // Group with orch tooling
	{keyword: "clean", name: "Cleanup"},
	{keyword: "reflect", name: "Reflection"},
	{keyword: "template", name: "Templates"},
	{keyword: "escape", name: "Escape hatch"},
	{keyword: "search", name: "Search"},
}

// MaxThreads is the maximum number of threads to display for readability.
const MaxThreads = 7

// LoadReadyIssues calls bd ready --json and parses the result.
func LoadReadyIssues() ([]BeadsIssue, error) {
	cmd := exec.Command("bd", "ready", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run bd ready: %w", err)
	}

	// Handle empty output
	if len(output) == 0 || strings.TrimSpace(string(output)) == "" {
		return nil, nil
	}

	var issues []BeadsIssue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd ready output: %w", err)
	}

	return issues, nil
}

// detectThreadKeyword finds the best matching keyword for an issue title.
// Returns the keyword and thread name, or empty strings if no match.
func detectThreadKeyword(title string) (keyword, threadName string) {
	lowerTitle := strings.ToLower(title)

	for _, tk := range threadKeywords {
		if strings.Contains(lowerTitle, tk.keyword) {
			return tk.keyword, tk.name
		}
	}

	return "", ""
}

// GroupIntoThreads groups issues into thematic threads.
func GroupIntoThreads(issues []BeadsIssue) []Thread {
	if len(issues) == 0 {
		return nil
	}

	// Map from thread name to thread
	threadMap := make(map[string]*Thread)
	var misc []BeadsIssue

	for _, issue := range issues {
		keyword, threadName := detectThreadKeyword(issue.Title)
		if threadName == "" {
			misc = append(misc, issue)
			continue
		}

		if t, exists := threadMap[threadName]; exists {
			t.Issues = append(t.Issues, issue)
		} else {
			threadMap[threadName] = &Thread{
				Name:    threadName,
				Issues:  []BeadsIssue{issue},
				Keyword: keyword,
			}
		}
	}

	// Add misc thread if there are ungrouped issues
	if len(misc) > 0 {
		threadMap["Misc"] = &Thread{
			Name:   "Misc",
			Issues: misc,
		}
	}

	// Convert map to slice and sort by issue count (descending)
	var threads []Thread
	for _, t := range threadMap {
		// Generate notes from first issue
		if len(t.Issues) > 0 {
			t.Notes = generateThreadNotes(t.Issues)
		}
		threads = append(threads, *t)
	}

	sort.Slice(threads, func(i, j int) bool {
		// Sort by issue count descending
		if len(threads[i].Issues) != len(threads[j].Issues) {
			return len(threads[i].Issues) > len(threads[j].Issues)
		}
		// Then by name alphabetically
		return threads[i].Name < threads[j].Name
	})

	// Cap at MaxThreads
	if len(threads) > MaxThreads {
		// Merge overflow into Misc
		miscThread := findOrCreateMisc(&threads)
		for i := MaxThreads; i < len(threads); i++ {
			if threads[i].Name != "Misc" {
				miscThread.Issues = append(miscThread.Issues, threads[i].Issues...)
			}
		}
		threads = threads[:MaxThreads]
	}

	return threads
}

// findOrCreateMisc finds the Misc thread or creates one.
func findOrCreateMisc(threads *[]Thread) *Thread {
	for i := range *threads {
		if (*threads)[i].Name == "Misc" {
			return &(*threads)[i]
		}
	}
	// Create new Misc thread
	*threads = append(*threads, Thread{Name: "Misc"})
	return &(*threads)[len(*threads)-1]
}

// generateThreadNotes creates a brief description for a thread.
func generateThreadNotes(issues []BeadsIssue) string {
	if len(issues) == 0 {
		return ""
	}

	// Extract meaningful snippet from first issue title
	title := issues[0].Title
	// Truncate if too long
	if len(title) > 50 {
		title = title[:47] + "..."
	}
	return title
}

// FocusGuidance represents the complete focus guidance output.
type FocusGuidance struct {
	TotalIssues  int
	ThreadCount  int
	Threads      []Thread
	PromptText   string
}

// GenerateFocusGuidance loads ready issues and generates focus guidance.
func GenerateFocusGuidance() (*FocusGuidance, error) {
	issues, err := LoadReadyIssues()
	if err != nil {
		return nil, err
	}

	if len(issues) == 0 {
		return &FocusGuidance{
			TotalIssues: 0,
			ThreadCount: 0,
			PromptText:  "No ready issues found. Use 'bd create' to add work or 'bd ready' to check filters.",
		}, nil
	}

	threads := GroupIntoThreads(issues)

	return &FocusGuidance{
		TotalIssues: len(issues),
		ThreadCount: len(threads),
		Threads:     threads,
		PromptText:  "What's nagging you or feels most important?",
	}, nil
}

// FormatFocusGuidance formats the guidance for display.
func FormatFocusGuidance(guidance *FocusGuidance) string {
	if guidance == nil || guidance.TotalIssues == 0 {
		return "\n" + guidance.PromptText
	}

	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("\n📋 Focus Guidance (%d ready issues in %d threads):\n\n",
		guidance.TotalIssues, guidance.ThreadCount))

	// Threads
	for _, thread := range guidance.Threads {
		// Collect issue IDs
		var ids []string
		for _, issue := range thread.Issues {
			ids = append(ids, issue.ID)
		}

		// Format thread line with proper padding
		idList := strings.Join(ids, ", ")
		sb.WriteString(fmt.Sprintf("Thread: %-18s → %s\n", thread.Name, idList))

		// Notes (if available and not the same as the thread name)
		if thread.Notes != "" {
			sb.WriteString(fmt.Sprintf("  Notes: %s\n", thread.Notes))
		}
		sb.WriteString("\n")
	}

	// Prompt
	sb.WriteString(guidance.PromptText)
	sb.WriteString("\n")

	return sb.String()
}
