// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// RecordResolution updates the most recent matching gap event with resolution info.
func (t *GapTracker) RecordResolution(query string, resolution string, details string) {
	// Find the most recent unresolved event for this query
	for i := len(t.Events) - 1; i >= 0; i-- {
		if t.Events[i].Query == query && t.Events[i].Resolution == "" {
			t.Events[i].Resolution = resolution
			t.Events[i].ResolutionDetails = details
			break
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
func (t *GapTracker) FindRecurringGaps() []LearningSuggestion {
	// Group events by normalized query
	queryGroups := make(map[string][]GapEvent)
	for _, e := range t.Events {
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

// normalizeQuery normalizes a query for grouping similar queries.
func normalizeQuery(query string) string {
	// Convert to lowercase
	normalized := strings.ToLower(query)
	// Remove extra whitespace
	normalized = strings.Join(strings.Fields(normalized), " ")
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

	if hasNoContext {
		// No context at all - suggest creating an investigation or knowledge
		return "add_knowledge",
			fmt.Sprintf("Gap %q has occurred %d times with no context. Consider creating foundational knowledge.", query, len(events)),
			fmt.Sprintf(`kn decide "%s" --reason "TODO: document decision"`, query)
	}

	if hasNoConstraints {
		// Context exists but no constraints - suggest adding constraints
		return "add_knowledge",
			fmt.Sprintf("Gap %q has occurred %d times without constraints. Add constraints if they exist.", query, len(events)),
			fmt.Sprintf(`kn constrain "%s" --reason "TODO: document constraint"`, query)
	}

	if hasNoDecisions {
		// Context exists but no decisions - suggest creating issue to investigate
		return "create_issue",
			fmt.Sprintf("Gap %q has occurred %d times without decisions. Create issue to establish patterns.", query, len(events)),
			fmt.Sprintf(`bd create "Establish patterns for %s" -d "Recurring gap detected - needs investigation"`, query)
	}

	// Default: suggest investigation
	return "investigate",
		fmt.Sprintf("Gap %q has occurred %d times. Investigate and document findings.", query, len(events)),
		fmt.Sprintf(`orch spawn investigation "why does %s lack context"`, query)
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
