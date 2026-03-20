// stats_cmd.go - Aggregate events.jsonl metrics for orchestration observability
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var (
	statsDays       int
	statsJSONOutput bool
	statsVerbose    bool
	statsSnapshot   bool
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show aggregated agent statistics from events.jsonl",
	Long: `Aggregate events.jsonl to surface orchestration metrics.

Shows:
  - Spawn and completion counts
  - Completion and abandonment rates
  - Average session duration
  - Skill effectiveness breakdown
  - Daemon health metrics

Examples:
  orch stats                    # Show last 7 days
  orch stats --days 1           # Show last 24 hours
  orch stats --days 30          # Show last 30 days
  orch stats --json             # Output as JSON for scripting
  orch stats --verbose          # Show additional metrics
  orch stats --snapshot         # Record gate accuracy baseline`,
	RunE: func(cmd *cobra.Command, args []string) error {
		emitCommandInvoked("stats", flagsFromCmd(cmd)...)
		return runStats()
	},
}

func init() {
	statsCmd.Flags().IntVar(&statsDays, "days", 7, "Number of days to analyze")
	statsCmd.Flags().BoolVar(&statsJSONOutput, "json", false, "Output as JSON")
	statsCmd.Flags().BoolVar(&statsVerbose, "verbose", false, "Show additional metrics")
	statsCmd.Flags().BoolVar(&statsSnapshot, "snapshot", false, "Record gate accuracy baseline snapshot")
	rootCmd.AddCommand(statsCmd)
}

func runStats() error {
	// Get events file path
	eventsPath := getEventsPath()

	// Parse events within time window (+ correlation buffer for spawn lookups)
	events, err := parseEvents(eventsPath, eventsSince(statsDays))
	if err != nil {
		return fmt.Errorf("failed to parse events: %w", err)
	}

	// Aggregate statistics
	report := aggregateStats(events, statsDays)

	// Collect N-value metrics (point-in-time system scale)
	report.NValueMetrics = collectNValueMetrics(eventsPath)

	// Snapshot mode: record gate accuracy baseline
	if statsSnapshot {
		return recordGateBaseline(report)
	}

	// Output
	if statsJSONOutput {
		return outputStatsJSON(report)
	}
	return outputStatsText(report)
}

func getEventsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".orch/events.jsonl"
	}
	return filepath.Join(home, ".orch", "events.jsonl")
}

// parseEvents reads events from events.jsonl, skipping events before `since`.
// When since > 0, it estimates a file offset based on first/last timestamps
// to avoid reading the entire file. This keeps read time proportional to the
// requested time window instead of total file size.
//
// Callers should pass since = eventsSince(days) to include a correlation buffer
// for spawn→completion lookups. Use since = 0 to read all events.
func parseEvents(path string, since ...int64) ([]StatsEvent, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("events.jsonl not found at %s - no events recorded yet", path)
		}
		return nil, fmt.Errorf("failed to open events file: %w", err)
	}
	defer file.Close()

	var reader io.Reader = file

	// When since is provided and > 0, try to seek past old events
	if len(since) > 0 && since[0] > 0 {
		if seekReader, ok := seekToTimestamp(file, since[0]); ok {
			reader = seekReader
		}
	}

	var events []StatsEvent
	scanner := bufio.NewScanner(reader)
	// Increase buffer size for potentially long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event StatsEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Skip malformed lines
			continue
		}

		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading events: %w", err)
	}

	return events, nil
}

// seekToTimestamp estimates the byte offset in events.jsonl where events
// around the target timestamp begin. It reads the first and last timestamps,
// interpolates a file position, seeks there, and skips to the next complete line.
// Returns a reader positioned after the seek, or (nil, false) if seeking isn't beneficial.
func seekToTimestamp(file *os.File, since int64) (io.Reader, bool) {
	stat, err := file.Stat()
	if err != nil || stat.Size() < 4096 {
		return nil, false // too small to bother seeking
	}
	fileSize := stat.Size()

	// Read first timestamp
	firstTS := readFirstTimestamp(file)
	if firstTS == 0 {
		file.Seek(0, io.SeekStart)
		return nil, false
	}

	// Read last timestamp
	lastTS := readLastTimestamp(file, fileSize)
	if lastTS == 0 || lastTS <= firstTS {
		file.Seek(0, io.SeekStart)
		return nil, false
	}

	// If since is before the first event, read everything
	if since <= firstTS {
		file.Seek(0, io.SeekStart)
		return nil, false
	}

	// Interpolate: what fraction of the file should we skip?
	totalDuration := float64(lastTS - firstTS)
	skipDuration := float64(since - firstTS)
	skipFraction := skipDuration / totalDuration

	// Apply a safety margin — seek to 20% earlier than estimated
	seekFraction := skipFraction * 0.8
	if seekFraction <= 0 {
		file.Seek(0, io.SeekStart)
		return nil, false
	}

	seekPos := int64(float64(fileSize) * seekFraction)
	file.Seek(seekPos, io.SeekStart)

	// Skip the partial line at seek position
	br := bufio.NewReader(file)
	_, err = br.ReadBytes('\n')
	if err != nil {
		// If we can't find a newline, fall back to start
		file.Seek(0, io.SeekStart)
		return nil, false
	}

	return br, true
}

// readFirstTimestamp reads the first valid timestamp from the beginning of the file.
func readFirstTimestamp(file *os.File) int64 {
	file.Seek(0, io.SeekStart)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var event struct {
			Timestamp int64 `json:"timestamp"`
		}
		if err := json.Unmarshal([]byte(line), &event); err == nil && event.Timestamp > 0 {
			return event.Timestamp
		}
	}
	return 0
}

// readLastTimestamp reads the last valid timestamp from the end of the file.
func readLastTimestamp(file *os.File, fileSize int64) int64 {
	// Read the last 4KB to find the last line
	readSize := int64(4096)
	if readSize > fileSize {
		readSize = fileSize
	}
	file.Seek(fileSize-readSize, io.SeekStart)

	data := make([]byte, readSize)
	n, err := io.ReadFull(file, data)
	if err != nil && err != io.ErrUnexpectedEOF {
		return 0
	}
	data = data[:n]

	// Scan backwards for the last complete line
	lastNewline := -1
	for i := len(data) - 1; i >= 0; i-- {
		if data[i] == '\n' {
			if lastNewline == -1 {
				lastNewline = i
				continue
			}
			// Found the start of the last line
			line := data[i+1 : lastNewline]
			var event struct {
				Timestamp int64 `json:"timestamp"`
			}
			if err := json.Unmarshal(line, &event); err == nil && event.Timestamp > 0 {
				return event.Timestamp
			}
			break
		}
	}

	// Edge case: only one line in the tail chunk
	if lastNewline >= 0 {
		line := data[:lastNewline]
		var event struct {
			Timestamp int64 `json:"timestamp"`
		}
		if err := json.Unmarshal(line, &event); err == nil && event.Timestamp > 0 {
			return event.Timestamp
		}
	}

	return 0
}

// eventsSince calculates the since timestamp for parseEvents.
// It uses the requested days plus a correlation buffer to ensure
// spawn events that precede completions within the window are included.
const eventCorrelationBufferDays = 7

func eventsSince(days int) int64 {
	return time.Now().Unix() - int64(days+eventCorrelationBufferDays)*86400
}

func aggregateStats(events []StatsEvent, days int) *StatsReport {
	a := newStatsAggregator(days)

	for _, event := range events {
		if event.Timestamp >= a.cutoffDays {
			a.eventsInWindow++
		}

		switch event.Type {
		case "session.spawned":
			a.processSessionSpawned(event)
		case "session.completed":
			a.processSessionCompleted(event)
		case "agent.completed":
			a.processAgentCompleted(event)
		case "agent.abandoned":
			a.processAgentAbandoned(event)
		case "daemon.spawn":
			a.processDaemonSpawn(event)
		case "session.auto_completed":
			a.processAutoCompleted(event)
		case "spawn.triage_bypassed":
			a.processTriageBypassed(event)
		case "spawn.hotspot_bypassed":
			a.processHotspotBypassed(event)
		case "spawn.verification_bypassed":
			a.processVerificationBypassed(event)
		case "agent.wait.complete":
			a.processWaitComplete(event)
		case "agent.wait.timeout":
			a.processWaitTimeout(event)
		case "session.orchestrator.started", "session.started":
			a.processSessionStarted(event)
		case "session.orchestrator.ended", "session.ended":
			a.processSessionEnded(event)
		case "verification.failed":
			a.processVerificationFailed(event)
		case "verification.bypassed":
			a.processVerificationBypassedEvent(event)
		case "verification.auto_skipped":
			a.processVerificationAutoSkipped(event)
		case "spawn.gate_decision":
			a.processGateDecision(event)
		case "daemon.architect_escalation":
			a.processArchitectEscalation(event)
		case "spawn.skill_inferred":
			a.processSkillInferred(event)
		}
	}

	a.report.EventsAnalyzed = a.eventsInWindow

	a.calcEscapeHatchStats()
	a.calcVerificationStats()
	a.calcSpawnGateStats()
	a.calcOverrideStats()
	a.calcRatesAndDuration()
	a.calcGateDecisionStats()
	a.calcGateEffectiveness(events)
	a.calcSkillInferenceStats()
	a.calcCoachingStats()

	return a.report
}

// extractGateAccuracyBaseline extracts a point-in-time snapshot of gate accuracy metrics
// from a stats report. Used for prospective measurement: compare baselines over time
// to answer "do gates improve agent quality?"
func extractGateAccuracyBaseline(report *StatsReport) GateAccuracyBaseline {
	ge := report.GateEffectivenessStats
	return GateAccuracyBaseline{
		SnapshotTime:             report.GeneratedAt,
		DaysAnalyzed:             report.DaysAnalyzed,
		TotalSpawns:              report.Summary.TotalSpawns,
		TotalCompletions:         report.Summary.TotalCompletions,
		GatedSpawns:              ge.GatedCompletion.TotalSpawns,
		UngatedSpawns:            ge.UngatedCompletion.TotalSpawns,
		GateDecisions:            report.GateDecisionStats.TotalDecisions,
		TotalBlocks:              ge.TotalBlocks,
		TotalBypasses:            ge.TotalBypasses,
		GatedCompletionRate:      ge.GatedCompletion.CompletionRate,
		UngatedCompletionRate:    ge.UngatedCompletion.CompletionRate,
		GatedVerificationRate:    ge.GatedCompletion.VerificationRate,
		UngatedVerificationRate:  ge.UngatedCompletion.VerificationRate,
		GatedAvgDuration:         ge.GatedCompletion.AvgDurationMinutes,
		UngatedAvgDuration:       ge.UngatedCompletion.AvgDurationMinutes,
		SpawnGateBypassRate:      report.SpawnGateStats.BypassRate,
		VerificationFirstTryRate: report.VerificationStats.PassRate,
	}
}

// recordGateBaseline records a gate accuracy baseline snapshot to ~/.orch/gate-baselines.jsonl.
// Each line is a JSON object with the baseline metrics at the time of recording.
func recordGateBaseline(report *StatsReport) error {
	baseline := extractGateAccuracyBaseline(report)

	data, err := json.Marshal(baseline)
	if err != nil {
		return fmt.Errorf("failed to marshal baseline: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	baselinePath := filepath.Join(home, ".orch", "gate-baselines.jsonl")
	f, err := os.OpenFile(baselinePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open baseline file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write baseline: %w", err)
	}

	// Print summary
	fmt.Printf("Gate accuracy baseline recorded to %s\n\n", baselinePath)
	fmt.Printf("  Period:           %s (%d days)\n", baseline.SnapshotTime, baseline.DaysAnalyzed)
	fmt.Printf("  Total spawns:     %d (%d gated, %d ungated)\n", baseline.TotalSpawns, baseline.GatedSpawns, baseline.UngatedSpawns)
	fmt.Printf("  Gate decisions:   %d (blocks: %d, bypasses: %d)\n", baseline.GateDecisions, baseline.TotalBlocks, baseline.TotalBypasses)
	fmt.Println()
	fmt.Printf("  Gated completion rate:      %.1f%%\n", baseline.GatedCompletionRate)
	fmt.Printf("  Ungated completion rate:    %.1f%%\n", baseline.UngatedCompletionRate)
	fmt.Printf("  Gated verification rate:    %.1f%%\n", baseline.GatedVerificationRate)
	fmt.Printf("  Ungated verification rate:  %.1f%%\n", baseline.UngatedVerificationRate)
	fmt.Printf("  Gated avg duration:         %.0fm\n", baseline.GatedAvgDuration)
	fmt.Printf("  Ungated avg duration:       %.0fm\n", baseline.UngatedAvgDuration)
	fmt.Printf("  Spawn gate bypass rate:     %.1f%%\n", baseline.SpawnGateBypassRate)
	fmt.Printf("  Verification 1st try rate:  %.1f%%\n", baseline.VerificationFirstTryRate)

	// Load and compare with previous baseline
	baselines, err := loadGateBaselines(baselinePath)
	if err == nil && len(baselines) > 1 {
		prev := baselines[len(baselines)-2]
		fmt.Println()
		fmt.Println("  Delta from previous baseline:")
		printDelta("  Gated verification rate", prev.GatedVerificationRate, baseline.GatedVerificationRate)
		printDelta("  Ungated verification rate", prev.UngatedVerificationRate, baseline.UngatedVerificationRate)
		printDelta("  Spawn gate bypass rate", prev.SpawnGateBypassRate, baseline.SpawnGateBypassRate)
		printDelta("  Verification 1st try rate", prev.VerificationFirstTryRate, baseline.VerificationFirstTryRate)
	}

	return nil
}

func printDelta(label string, prev, curr float64) {
	delta := curr - prev
	direction := "="
	if delta > 0.5 {
		direction = "+"
	} else if delta < -0.5 {
		direction = "-"
	}
	fmt.Printf("    %-30s %6.1f%% -> %6.1f%% (%s%.1f%%)\n", label, prev, curr, direction, delta)
}

// collectNValueMetrics gathers point-in-time system scale metrics.
// Scans workspace directories across all known projects (from ~/.kb/projects.json),
// counts total events, and counts KB files in the current project.
func collectNValueMetrics(eventsPath string) NValueMetrics {
	metrics := NValueMetrics{}

	// 1. Event count: total lines in events.jsonl
	metrics.EventCount, _ = countFileLines(eventsPath)

	// 2. KB file count: .md files in current project's .kb/
	cwd, _ := os.Getwd()
	kbDir := filepath.Join(cwd, ".kb")
	metrics.KBFileCount = countFilesRecursive(kbDir, ".md")

	// 3. Workspace count across all known projects
	projectPaths := getProjectPathsFromRegistry()
	if len(projectPaths) == 0 {
		// Fallback: just use current project
		projectPaths = []string{cwd}
	}
	metrics.WorkspaceProjects = len(projectPaths)
	for _, projPath := range projectPaths {
		wsDir := filepath.Join(projPath, ".orch", "workspace")
		metrics.WorkspaceCount += countDirectories(wsDir)
	}
	if statsVerbose {
		metrics.ProjectPaths = projectPaths
	}

	return metrics
}

// countFilesRecursive counts files with the given extension under a directory.
func countFilesRecursive(dir string, ext string) int {
	count := 0
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if !info.IsDir() && filepath.Ext(path) == ext {
			count++
		}
		return nil
	})
	return count
}

// countDirectories counts immediate subdirectories (non-recursive, excludes "archived").
func countDirectories(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "archived" {
			count++
		}
	}
	return count
}

// getProjectPathsFromRegistry reads ~/.kb/projects.json to get all known project paths.
func getProjectPathsFromRegistry() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	registryPath := filepath.Join(home, ".kb", "projects.json")
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return nil
	}

	// projects.json has a "projects" key containing an array of {name, path} objects
	var registry struct {
		Projects []struct {
			Path string `json:"path"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil
	}

	var paths []string
	for _, p := range registry.Projects {
		if p.Path != "" {
			paths = append(paths, p.Path)
		}
	}
	return paths
}

func loadGateBaselines(path string) ([]GateAccuracyBaseline, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var baselines []GateAccuracyBaseline
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var b GateAccuracyBaseline
		if err := json.Unmarshal([]byte(line), &b); err != nil {
			continue
		}
		baselines = append(baselines, b)
	}
	return baselines, scanner.Err()
}
