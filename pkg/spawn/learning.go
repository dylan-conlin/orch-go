// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Learning system constants.
const (
	// RecurrenceThreshold is the number of times a gap must occur before suggesting action.
	RecurrenceThreshold = 3

	// GapHistoryMaxAge is how long to keep gap events (30 days).
	GapHistoryMaxAge = 30 * 24 * time.Hour

	// MaxGapEvents is the maximum number of gap events to store.
	MaxGapEvents = 1000
)

// GapEvent represents a single occurrence of a context gap.
type GapEvent struct {
	// Timestamp when the gap was detected.
	Timestamp time.Time `json:"timestamp"`

	// Query that was searched when the gap occurred.
	Query string `json:"query"`

	// GapType is the type of gap detected (from gap.go).
	GapType string `json:"gap_type"`

	// Severity of the gap.
	Severity string `json:"severity"`

	// Skill being spawned (if applicable).
	Skill string `json:"skill,omitempty"`

	// Task description from spawn.
	Task string `json:"task,omitempty"`

	// ContextQuality score at time of gap.
	ContextQuality int `json:"context_quality"`

	// Resolution indicates how the gap was handled.
	// Values: "proceeded", "added_knowledge", "created_issue", "aborted"
	Resolution string `json:"resolution,omitempty"`

	// ResolutionDetails provides additional context about resolution.
	ResolutionDetails string `json:"resolution_details,omitempty"`
}

// GapTracker manages the history of context gaps and learning suggestions.
type GapTracker struct {
	// Events is the list of recorded gap events.
	Events []GapEvent `json:"events"`

	// Improvements tracks actions taken and their effectiveness.
	Improvements []ImprovementRecord `json:"improvements,omitempty"`

	// LastAnalysis tracks when pattern analysis was last run.
	LastAnalysis time.Time `json:"last_analysis,omitempty"`
}

// ImprovementRecord tracks an improvement made in response to gaps.
type ImprovementRecord struct {
	// Timestamp when the improvement was made.
	Timestamp time.Time `json:"timestamp"`

	// Type of improvement: "issue", "kn_entry", "investigation", "decision"
	Type string `json:"type"`

	// Query or topic this improvement addresses.
	Query string `json:"query"`

	// Reference to the created artifact (issue ID, kn entry ID, file path).
	Reference string `json:"reference"`

	// GapCountBefore is how many gaps existed for this topic before improvement.
	GapCountBefore int `json:"gap_count_before"`

	// GapCountAfter tracks gaps after improvement (updated over time).
	GapCountAfter int `json:"gap_count_after,omitempty"`
}

// LearningSuggestion represents a suggested action based on gap patterns.
type LearningSuggestion struct {
	// Type of suggestion: "create_issue", "add_knowledge", "investigate"
	Type string `json:"type"`

	// Priority: "high", "medium", "low"
	Priority string `json:"priority"`

	// Query or topic affected.
	Query string `json:"query"`

	// Count of times this gap has occurred.
	Count int `json:"count"`

	// Suggestion text explaining what to do.
	Suggestion string `json:"suggestion"`

	// Command to run (if applicable).
	Command string `json:"command,omitempty"`

	// GapEvents contributing to this suggestion.
	Events []GapEvent `json:"events,omitempty"`
}

// TopicAnalysis represents analysis of gaps for a specific topic.
type TopicAnalysis struct {
	// Topic or query pattern.
	Topic string `json:"topic"`

	// TotalGaps is the total number of gap events.
	TotalGaps int `json:"total_gaps"`

	// RecentGaps is gaps in the last 7 days.
	RecentGaps int `json:"recent_gaps"`

	// CriticalGaps is the count of critical-severity gaps.
	CriticalGaps int `json:"critical_gaps"`

	// AverageQuality is the average context quality score.
	AverageQuality float64 `json:"average_quality"`

	// Skills lists skills that encountered this gap.
	Skills []string `json:"skills,omitempty"`

	// Trend indicates if gaps are increasing, decreasing, or stable.
	Trend string `json:"trend"`
}

// trackerPathFunc is a variable to allow testing with custom paths.
var trackerPathFunc = defaultTrackerPath

// defaultTrackerPath returns the default path for the gap tracker file.
func defaultTrackerPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".orch", "gap-tracker.json")
}

// TrackerPath returns the path for the gap tracker file.
func TrackerPath() string {
	return trackerPathFunc()
}

// TrackerPathFunc returns the current tracker path function.
// Used for testing to save and restore the original function.
func TrackerPathFunc() string {
	return trackerPathFunc()
}

// SetTrackerPathFunc sets a custom tracker path function.
// Used for testing to inject a custom path.
func SetTrackerPathFunc(fn func() string) {
	trackerPathFunc = fn
}

// LoadTracker loads the gap tracker from disk.
// Returns an empty tracker if file doesn't exist.
func LoadTracker() (*GapTracker, error) {
	path := TrackerPath()
	if path == "" {
		return &GapTracker{Events: []GapEvent{}}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &GapTracker{Events: []GapEvent{}}, nil
		}
		return nil, fmt.Errorf("failed to read gap tracker: %w", err)
	}

	var tracker GapTracker
	if err := json.Unmarshal(data, &tracker); err != nil {
		return nil, fmt.Errorf("failed to parse gap tracker: %w", err)
	}

	return &tracker, nil
}

// Save saves the gap tracker to disk.
func (t *GapTracker) Save() error {
	path := TrackerPath()
	if path == "" {
		return fmt.Errorf("could not determine tracker path")
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Prune old events before saving
	t.pruneOldEvents()

	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tracker: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write tracker: %w", err)
	}

	return nil
}

// pruneOldEvents removes events older than GapHistoryMaxAge and caps at MaxGapEvents.
func (t *GapTracker) pruneOldEvents() {
	cutoff := time.Now().Add(-GapHistoryMaxAge)

	// Filter out old events
	kept := []GapEvent{}
	for _, e := range t.Events {
		if e.Timestamp.After(cutoff) {
			kept = append(kept, e)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(kept, func(i, j int) bool {
		return kept[i].Timestamp.After(kept[j].Timestamp)
	})

	// Cap at max events
	if len(kept) > MaxGapEvents {
		kept = kept[:MaxGapEvents]
	}

	t.Events = kept
}

// RecordGap adds a new gap event to the tracker.
func (t *GapTracker) RecordGap(analysis *GapAnalysis, skill, task string) {
	if analysis == nil || !analysis.HasGaps {
		return
	}

	for _, gap := range analysis.Gaps {
		event := GapEvent{
			Timestamp:      time.Now().UTC(),
			Query:          analysis.Query,
			GapType:        string(gap.Type),
			Severity:       string(gap.Severity),
			Skill:          skill,
			Task:           task,
			ContextQuality: analysis.ContextQuality,
		}
		t.Events = append(t.Events, event)
	}
}

// RecordResolution updates ALL unresolved events matching this query with resolution info.
// When a gap is resolved, all occurrences of that gap should be marked as resolved,
// otherwise they continue to appear in suggestions.
func (t *GapTracker) RecordResolution(query string, resolution string, details string) {
	normalizedQuery := normalizeQuery(query)
	for i := range t.Events {
		eventNormalized := normalizeQuery(t.Events[i].Query)
		if eventNormalized == normalizedQuery && t.Events[i].Resolution == "" {
			t.Events[i].Resolution = resolution
			t.Events[i].ResolutionDetails = details
		}
	}
}

// RecordImprovement records that an improvement was made for a gap pattern.
func (t *GapTracker) RecordImprovement(improvementType, query, reference string) {
	// Count existing gaps for this query
	gapCount := t.countGapsForQuery(query)

	record := ImprovementRecord{
		Timestamp:      time.Now().UTC(),
		Type:           improvementType,
		Query:          query,
		Reference:      reference,
		GapCountBefore: gapCount,
	}
	t.Improvements = append(t.Improvements, record)
}

// countGapsForQuery counts gap events matching a query.
func (t *GapTracker) countGapsForQuery(query string) int {
	count := 0
	queryLower := strings.ToLower(query)
	for _, e := range t.Events {
		if strings.Contains(strings.ToLower(e.Query), queryLower) {
			count++
		}
	}
	return count
}

// FindRecurringGaps identifies gaps that have occurred RecurrenceThreshold+ times.
// Only counts UNRESOLVED events - resolved gaps are excluded from suggestions.
// Also filters task-specific noise (issue IDs, phase names) from suggestions.
func (t *GapTracker) FindRecurringGaps() []LearningSuggestion {
	// Group UNRESOLVED events by normalized query
	queryGroups := make(map[string][]GapEvent)
	for _, e := range t.Events {
		// Skip resolved events - they shouldn't count toward recurrence
		if e.Resolution != "" {
			continue
		}
		// Skip task-specific noise (issue IDs, phase announcements)
		if isTaskNoise(e.Query) {
			continue
		}
		normalized := normalizeQuery(e.Query)
		queryGroups[normalized] = append(queryGroups[normalized], e)
	}

	suggestions := []LearningSuggestion{}

	for query, events := range queryGroups {
		if len(events) < RecurrenceThreshold {
			continue
		}

		// Determine priority based on severity distribution
		criticalCount := 0
		for _, e := range events {
			if e.Severity == string(GapSeverityCritical) {
				criticalCount++
			}
		}

		priority := "medium"
		if criticalCount > 0 {
			priority = "high"
		}

		// Determine suggestion type based on gap patterns
		suggestionType, suggestion, command := determineSuggestion(query, events)

		suggestions = append(suggestions, LearningSuggestion{
			Type:       suggestionType,
			Priority:   priority,
			Query:      query,
			Count:      len(events),
			Suggestion: suggestion,
			Command:    command,
			Events:     events,
		})
	}

	// Sort by count (highest first), then priority
	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].Count != suggestions[j].Count {
			return suggestions[i].Count > suggestions[j].Count
		}
		return priorityOrder(suggestions[i].Priority) < priorityOrder(suggestions[j].Priority)
	})

	return suggestions
}

// queryPattern defines a semantic pattern for grouping related queries.
// Patterns use glob-style wildcards where "*" matches 1-3 words.
type queryPattern struct {
	// Pattern is the glob-style pattern to match (e.g., "synthesize * investigations").
	Pattern string
	// Canonical is the normalized form for grouping (e.g., "synthesize investigations").
	Canonical string
}

// semanticPatterns are common query patterns for grouping related gaps.
// Order matters - more specific patterns should come first.
var semanticPatterns = []queryPattern{
	// Synthesis patterns - common orchestrator tasks
	{"synthesize * investigations", "synthesize investigations"},
	{"synthesize * findings", "synthesize findings"},

	// Audit patterns
	{"audit * patterns", "audit patterns"},
	{"audit * gaps", "audit gaps"},

	// Implementation patterns
	{"implement * feature", "implement feature"},
	{"add * feature", "add feature"},
	{"create * component", "create component"},

	// Debug/investigation patterns
	{"debug * issue", "debug issue"},
	{"investigate * behavior", "investigate behavior"},
	{"analyze * flow", "analyze flow"},

	// Configuration patterns
	{"configure * settings", "configure settings"},
	{"update * config", "update config"},
}

// matchPattern checks if a query matches a glob-style pattern.
// The wildcard "*" matches 1-3 words (not entire queries).
// Returns the canonical form if matched, empty string otherwise.
func matchPattern(query string, pattern queryPattern) string {
	queryWords := strings.Fields(query)
	patternWords := strings.Fields(pattern.Pattern)

	// Find wildcard position in pattern
	wildcardIdx := -1
	for i, w := range patternWords {
		if w == "*" {
			wildcardIdx = i
			break
		}
	}

	if wildcardIdx == -1 {
		// No wildcard - exact match
		if query == pattern.Pattern {
			return pattern.Canonical
		}
		return ""
	}

	// Split pattern into prefix and suffix
	prefix := patternWords[:wildcardIdx]
	suffix := patternWords[wildcardIdx+1:]

	// Query must have at least len(prefix) + 1 (wildcard) + len(suffix) words
	minWords := len(prefix) + 1 + len(suffix)
	maxWords := len(prefix) + 3 + len(suffix) // Wildcard matches 1-3 words

	if len(queryWords) < minWords || len(queryWords) > maxWords {
		return ""
	}

	// Check prefix matches
	for i, w := range prefix {
		if i >= len(queryWords) || queryWords[i] != w {
			return ""
		}
	}

	// Check suffix matches (from end)
	suffixStart := len(queryWords) - len(suffix)
	for i, w := range suffix {
		if queryWords[suffixStart+i] != w {
			return ""
		}
	}

	// Prefix and suffix match - wildcard covers the middle
	return pattern.Canonical
}

// isTaskNoise checks if a query appears to be task-specific noise.
// Task noise includes:
// - Issue IDs (e.g., "orch-go-0vscq", "og-feat-implement-xyz")
// - Phase announcements (e.g., "Phase: Planning", "Phase: Complete")
//
// These patterns appear in gap queries due to task descriptions but
// don't represent genuine knowledge gaps worth acting on.
func isTaskNoise(query string) bool {
	normalized := strings.ToLower(strings.TrimSpace(query))

	// Check for phase announcements (e.g., "Phase: Planning")
	if strings.HasPrefix(normalized, "phase:") {
		return true
	}

	// Check for issue ID patterns (e.g., "orch-go-0vscq", "og-feat-xyz")
	// Pattern matches: project-identifier format common in beads issue IDs
	// Examples: orch-go-0vscq.5, og-feat-implement-task-noise-17jan-8344
	matched, _ := regexp.MatchString(`^[a-z]+-[a-z]+-\w+`, normalized)
	return matched
}

// normalizeQuery normalizes a query for grouping similar queries.
// Uses semantic pattern matching to group related queries, falling back
// to basic string normalization for unmatched queries.
func normalizeQuery(query string) string {
	// Convert to lowercase and normalize whitespace
	normalized := strings.ToLower(query)
	normalized = strings.Join(strings.Fields(normalized), " ")

	// Try semantic pattern matching
	for _, pattern := range semanticPatterns {
		if canonical := matchPattern(normalized, pattern); canonical != "" {
			return canonical
		}
	}

	// No pattern matched - return basic normalization
	return normalized
}

// priorityOrder returns numeric ordering for priorities.
func priorityOrder(priority string) int {
	switch priority {
	case "high":
		return 0
	case "medium":
		return 1
	case "low":
		return 2
	default:
		return 3
	}
}

// determineSuggestion determines what action to suggest based on gap patterns.
// The remediation type must match the gap type:
// - no_context → investigate (we don't know what's missing, need discovery)
// - no_constraints → kn constrain (we know constraints are missing)
// - no_decisions → bd create (we know decisions are missing, need to establish)
// - sparse_context/other → investigate (need more understanding)
func determineSuggestion(query string, events []GapEvent) (suggestionType, suggestion, command string) {
	// Check gap types to determine best suggestion
	hasNoContext := false
	hasNoConstraints := false
	hasNoDecisions := false

	for _, e := range events {
		switch e.GapType {
		case string(GapTypeNoContext):
			hasNoContext = true
		case string(GapTypeNoConstraints):
			hasNoConstraints = true
		case string(GapTypeNoDecisions):
			hasNoDecisions = true
		}
	}

	// Generate a meaningful reason from gap context
	reason := generateReasonFromGaps(query, events)

	// Priority order matters: no_constraints and no_decisions are more specific
	// than no_context, so check them first when they co-occur

	if hasNoConstraints {
		// Context exists but no constraints - suggest adding constraints
		return "add_knowledge",
			fmt.Sprintf("Gap %q has occurred %d times without constraints. Add constraints if they exist.", query, len(events)),
			fmt.Sprintf(`kn constrain "%s" --reason "%s"`, query, reason)
	}

	if hasNoDecisions {
		// Context exists but no decisions - suggest creating issue to investigate
		return "create_issue",
			fmt.Sprintf("Gap %q has occurred %d times without decisions. Create issue to establish patterns.", query, len(events)),
			fmt.Sprintf(`bd create "Establish patterns for %s" -d "%s"`, query, reason)
	}

	if hasNoContext {
		// No context at all - we don't know what type of knowledge is missing
		// Suggest investigation to discover what's needed (not kn decide which assumes decision is needed)
		return "investigate",
			fmt.Sprintf("Gap %q has occurred %d times with no context. Investigate to discover what knowledge is missing.", query, len(events)),
			fmt.Sprintf(`orch spawn investigation "what context is needed for %s"`, query)
	}

	// Default: suggest investigation (covers sparse_context and other cases)
	return "investigate",
		fmt.Sprintf("Gap %q has occurred %d times. Investigate and document findings.", query, len(events)),
		fmt.Sprintf(`orch spawn investigation "why does %s lack context"`, query)
}

// MinReasonLength is the minimum length required by kn for --reason argument.
const MinReasonLength = 20

// generateReasonFromGaps creates a meaningful reason string from gap event context.
// Ensures the reason is at least MinReasonLength characters to satisfy kn validation.
func generateReasonFromGaps(query string, events []GapEvent) string {
	if len(events) == 0 {
		return "No context available for this topic"
	}

	// Collect unique skills and tasks from events
	skills := make(map[string]bool)
	tasks := make([]string, 0)

	for _, e := range events {
		if e.Skill != "" {
			skills[e.Skill] = true
		}
		if e.Task != "" && len(tasks) < 3 { // Collect up to 3 unique tasks
			found := false
			for _, t := range tasks {
				if t == e.Task {
					found = true
					break
				}
			}
			if !found {
				tasks = append(tasks, e.Task)
			}
		}
	}

	// Build reason based on available context
	var parts []string

	// Add skill context
	if len(skills) > 0 {
		skillList := make([]string, 0, len(skills))
		for s := range skills {
			skillList = append(skillList, s)
		}
		sort.Strings(skillList)
		parts = append(parts, fmt.Sprintf("Used by: %s", strings.Join(skillList, ", ")))
	}

	// Add occurrence count
	parts = append(parts, fmt.Sprintf("Occurred %d times", len(events)))

	// Add task context if available
	if len(tasks) > 0 {
		// Truncate long tasks
		shortTasks := make([]string, 0, len(tasks))
		for _, t := range tasks {
			if len(t) > 40 {
				shortTasks = append(shortTasks, t[:37]+"...")
			} else {
				shortTasks = append(shortTasks, t)
			}
		}
		parts = append(parts, fmt.Sprintf("Tasks: %s", strings.Join(shortTasks, "; ")))
	}

	reason := strings.Join(parts, ". ")

	// Ensure minimum length for kn validation (requires at least 20 chars)
	if len(reason) < MinReasonLength {
		// Pad with query context to meet minimum
		reason = fmt.Sprintf("Recurring gap for topic: %s. %s", query, reason)
	}

	return reason
}

// AnalyzePatterns provides comprehensive analysis of gap patterns.
func (t *GapTracker) AnalyzePatterns() []TopicAnalysis {
	// Group by normalized query
	queryGroups := make(map[string][]GapEvent)
	for _, e := range t.Events {
		normalized := normalizeQuery(e.Query)
		queryGroups[normalized] = append(queryGroups[normalized], e)
	}

	analyses := []TopicAnalysis{}
	now := time.Now()
	weekAgo := now.Add(-7 * 24 * time.Hour)

	for topic, events := range queryGroups {
		analysis := TopicAnalysis{
			Topic:     topic,
			TotalGaps: len(events),
		}

		// Count recent gaps and critical gaps
		skillSet := make(map[string]bool)
		qualitySum := 0
		for _, e := range events {
			if e.Timestamp.After(weekAgo) {
				analysis.RecentGaps++
			}
			if e.Severity == string(GapSeverityCritical) {
				analysis.CriticalGaps++
			}
			if e.Skill != "" {
				skillSet[e.Skill] = true
			}
			qualitySum += e.ContextQuality
		}

		// Calculate average quality
		if len(events) > 0 {
			analysis.AverageQuality = float64(qualitySum) / float64(len(events))
		}

		// Extract skills
		for skill := range skillSet {
			analysis.Skills = append(analysis.Skills, skill)
		}
		sort.Strings(analysis.Skills)

		// Determine trend
		analysis.Trend = determineTrend(events)

		analyses = append(analyses, analysis)
	}

	// Sort by total gaps (highest first)
	sort.Slice(analyses, func(i, j int) bool {
		return analyses[i].TotalGaps > analyses[j].TotalGaps
	})

	return analyses
}

// determineTrend analyzes event timestamps to determine if gaps are increasing.
func determineTrend(events []GapEvent) string {
	if len(events) < 3 {
		return "insufficient_data"
	}

	now := time.Now()
	weekAgo := now.Add(-7 * 24 * time.Hour)
	twoWeeksAgo := now.Add(-14 * 24 * time.Hour)

	recentCount := 0
	olderCount := 0

	for _, e := range events {
		if e.Timestamp.After(weekAgo) {
			recentCount++
		} else if e.Timestamp.After(twoWeeksAgo) {
			olderCount++
		}
	}

	if recentCount > olderCount*2 {
		return "increasing"
	} else if recentCount*2 < olderCount {
		return "decreasing"
	}
	return "stable"
}

// GetSkillGapRates returns gap statistics by skill.
func (t *GapTracker) GetSkillGapRates() map[string]int {
	rates := make(map[string]int)
	for _, e := range t.Events {
		if e.Skill != "" {
			rates[e.Skill]++
		}
	}
	return rates
}

// MeasureImprovementEffectiveness checks if improvements reduced gaps.
func (t *GapTracker) MeasureImprovementEffectiveness() []ImprovementRecord {
	// Update gap counts for each improvement
	for i := range t.Improvements {
		imp := &t.Improvements[i]
		// Count gaps after the improvement was made
		gapsAfter := 0
		for _, e := range t.Events {
			if e.Timestamp.After(imp.Timestamp) &&
				strings.Contains(strings.ToLower(e.Query), strings.ToLower(imp.Query)) {
				gapsAfter++
			}
		}
		imp.GapCountAfter = gapsAfter
	}
	return t.Improvements
}

// FormatSuggestions formats learning suggestions for display.
func FormatSuggestions(suggestions []LearningSuggestion) string {
	if len(suggestions) == 0 {
		return "No recurring gaps detected - system is learning effectively!\n"
	}

	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("╔══════════════════════════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║  📚 LEARNING SUGGESTIONS - Recurring gaps detected                          ║\n")
	sb.WriteString("╠══════════════════════════════════════════════════════════════════════════════╣\n")

	for i, s := range suggestions {
		if i >= 5 {
			sb.WriteString(fmt.Sprintf("║  ... and %d more suggestions                                               ║\n", len(suggestions)-5))
			break
		}

		icon := "○"
		if s.Priority == "high" {
			icon = "●"
		} else if s.Priority == "medium" {
			icon = "◐"
		}

		sb.WriteString(fmt.Sprintf("║  %s [%s] %s (%dx)\n", icon, s.Priority, truncateWithPadding(s.Query, 50), s.Count))
		sb.WriteString(fmt.Sprintf("║      → %s\n", truncateWithPadding(s.Suggestion, 65)))
		if s.Command != "" {
			sb.WriteString(fmt.Sprintf("║      $ %s\n", truncateWithPadding(s.Command, 65)))
		}
	}

	sb.WriteString("╠══════════════════════════════════════════════════════════════════════════════╣\n")
	sb.WriteString("║  Run 'orch learn' to review and act on suggestions                          ║\n")
	sb.WriteString("╚══════════════════════════════════════════════════════════════════════════════╝\n")

	return sb.String()
}

// Summary returns a brief summary of the tracker state.
func (t *GapTracker) Summary() string {
	if len(t.Events) == 0 {
		return "No gaps tracked yet"
	}

	recurring := t.FindRecurringGaps()
	return fmt.Sprintf("%d gap events tracked, %d recurring patterns", len(t.Events), len(recurring))
}

// ParseShellCommand parses a shell command string into arguments, respecting quotes.
// This is similar to POSIX shell word splitting.
// Examples:
//
//	`kn decide "auth" --reason "test reason"` -> ["kn", "decide", "auth", "--reason", "test reason"]
//	`echo "hello world"` -> ["echo", "hello world"]
func ParseShellCommand(cmdStr string) ([]string, error) {
	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for i, r := range cmdStr {
		switch {
		case r == '"' || r == '\'':
			if inQuote {
				if r == quoteChar {
					// End of quote
					inQuote = false
					quoteChar = 0
				} else {
					// Different quote char inside a quote, treat as literal
					current.WriteRune(r)
				}
			} else {
				// Start of quote
				inQuote = true
				quoteChar = r
			}
		case r == ' ' || r == '\t':
			if inQuote {
				current.WriteRune(r)
			} else if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}

		// Check for unterminated quote at end of string
		if i == len(cmdStr)-1 && inQuote {
			return nil, fmt.Errorf("unterminated quote in command: %s", cmdStr)
		}
	}

	// Add final argument if any
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	if len(args) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	return args, nil
}

// ValidateCommand checks if a command string is valid and executable.
// It verifies:
// 1. The command can be parsed correctly
// 2. The executable exists (for known commands)
// 3. Required arguments are present
func ValidateCommand(cmdStr string) error {
	if cmdStr == "" {
		return fmt.Errorf("empty command")
	}

	args, err := ParseShellCommand(cmdStr)
	if err != nil {
		return fmt.Errorf("failed to parse command: %w", err)
	}

	if len(args) == 0 {
		return fmt.Errorf("no arguments in command")
	}

	// Validate known command patterns
	switch args[0] {
	case "kn":
		return validateKnCommand(args)
	case "bd":
		return validateBdCommand(args)
	case "orch":
		return validateOrchCommand(args)
	default:
		// Unknown command - just check it can be parsed
		return nil
	}
}

// validateKnCommand checks kn command syntax.
func validateKnCommand(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("kn command requires subcommand (decide, constrain, etc.)")
	}

	switch args[1] {
	case "decide", "constrain", "tried":
		// These require at least a description
		if len(args) < 3 {
			return fmt.Errorf("kn %s requires a description", args[1])
		}
		// Check for --reason flag value and validate minimum length
		for i, arg := range args {
			if arg == "--reason" {
				if i == len(args)-1 {
					return fmt.Errorf("--reason flag requires a value")
				}
				reasonValue := args[i+1]
				if len(reasonValue) < MinReasonLength {
					return fmt.Errorf("--reason must be at least %d characters (got %d)", MinReasonLength, len(reasonValue))
				}
			}
		}
	case "question":
		if len(args) < 3 {
			return fmt.Errorf("kn question requires a question text")
		}
	}

	return nil
}

// validateBdCommand checks bd command syntax.
func validateBdCommand(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("bd command requires subcommand (create, close, etc.)")
	}

	switch args[1] {
	case "create":
		// bd create requires a title
		if len(args) < 3 {
			return fmt.Errorf("bd create requires a title")
		}
		// Check for -d flag value
		for i, arg := range args {
			if arg == "-d" && i == len(args)-1 {
				return fmt.Errorf("-d flag requires a value")
			}
		}
	}

	return nil
}

// validateOrchCommand checks orch command syntax.
func validateOrchCommand(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("orch command requires subcommand")
	}

	switch args[1] {
	case "spawn":
		// orch spawn requires skill and task
		if len(args) < 4 {
			return fmt.Errorf("orch spawn requires skill and task")
		}
	}

	return nil
}
