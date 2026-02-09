package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/spf13/cobra"
)

var outcomesJSONOutput bool

var outcomesCmd = &cobra.Command{
	Use:   "outcomes",
	Short: "Show normalized outcome metrics",
	Long: `Show normalized system outcome metrics consolidated from beads issues,
workspace metadata, and events.

Includes:
  - completion/open/abandoned counts by skill
  - spawn-to-close duration distribution (p50/p90/p95)
  - abandonment reason breakdown
  - investigation-to-model throughput`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runOutcomes()
	},
}

func init() {
	outcomesCmd.Flags().BoolVar(&outcomesJSONOutput, "json", false, "Output as JSON")
	rootCmd.AddCommand(outcomesCmd)
}

type OutcomeReport struct {
	GeneratedAt          string                         `json:"generated_at"`
	ProjectDir           string                         `json:"project_dir"`
	CountsBySkill        []OutcomeSkillCounts           `json:"counts_by_skill"`
	DurationDistribution OutcomeDurationDistribution    `json:"duration_distribution"`
	AbandonmentReasons   []OutcomeReasonCount           `json:"abandonment_reasons"`
	InvestigationToModel OutcomeInvestigationModelStats `json:"investigation_to_model"`
	DataQuality          OutcomeDataQuality             `json:"data_quality"`
}

type OutcomeSkillCounts struct {
	Skill      string `json:"skill"`
	Completion int    `json:"completion"`
	Open       int    `json:"open"`
	Abandoned  int    `json:"abandoned"`
	Total      int    `json:"total"`
}

type OutcomeDurationDistribution struct {
	Samples         int     `json:"samples"`
	P50Minutes      float64 `json:"p50_minutes"`
	P90Minutes      float64 `json:"p90_minutes"`
	P95Minutes      float64 `json:"p95_minutes"`
	UnmatchedClosed int     `json:"unmatched_closed"`
}

type OutcomeReasonCount struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

type OutcomeInvestigationModelStats struct {
	TotalInvestigations             int     `json:"total_investigations"`
	InvestigationsWithModelCitation int     `json:"investigations_with_model_citation"`
	ThroughputRate                  float64 `json:"throughput_rate"`
}

type OutcomeDataQuality struct {
	IssuesTotal int `json:"issues_total"`
}

func runOutcomes() error {
	projectDir, err := currentProjectDir()
	if err != nil {
		return fmt.Errorf("failed to resolve project directory: %w", err)
	}

	report, err := buildOutcomeReport(projectDir)
	if err != nil {
		return err
	}

	if outcomesJSONOutput {
		return outputOutcomesJSON(report)
	}
	return outputOutcomesText(report)
}

func buildOutcomeReport(projectDir string) (*OutcomeReport, error) {
	issuesPath := filepath.Join(projectDir, ".beads", "issues.jsonl")
	issues, err := parseBeadsIssues(issuesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse beads issues: %w", err)
	}

	spawnTimes, err := collectSpawnTimes(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to collect workspace metadata: %w", err)
	}

	abandonmentReasons, err := collectAbandonmentReasons(getEventsPath())
	if err != nil {
		return nil, fmt.Errorf("failed to collect abandonment reasons: %w", err)
	}

	throughput, err := collectInvestigationToModelThroughput(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to collect investigation throughput: %w", err)
	}

	report := &OutcomeReport{
		GeneratedAt:          time.Now().Format(time.RFC3339),
		ProjectDir:           projectDir,
		CountsBySkill:        collectCountsBySkill(issues),
		DurationDistribution: collectDurationDistribution(issues, spawnTimes),
		AbandonmentReasons:   abandonmentReasons,
		InvestigationToModel: throughput,
		DataQuality: OutcomeDataQuality{
			IssuesTotal: len(issues),
		},
	}

	return report, nil
}

func parseBeadsIssues(path string) ([]beads.Issue, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return []beads.Issue{}, nil
	}

	if strings.HasPrefix(trimmed, "[") {
		var issues []beads.Issue
		if err := json.Unmarshal([]byte(trimmed), &issues); err != nil {
			return nil, err
		}
		return issues, nil
	}

	issues := []beads.Issue{}
	scanner := bufio.NewScanner(strings.NewReader(trimmed))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var issue beads.Issue
		if err := json.Unmarshal([]byte(line), &issue); err != nil {
			continue
		}
		if issue.ID == "" {
			continue
		}
		issues = append(issues, issue)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return issues, nil
}

func collectCountsBySkill(issues []beads.Issue) []OutcomeSkillCounts {
	counts := map[string]*OutcomeSkillCounts{}

	for _, issue := range issues {
		skill := inferIssueSkill(issue)
		if _, ok := counts[skill]; !ok {
			counts[skill] = &OutcomeSkillCounts{Skill: skill}
		}

		row := counts[skill]
		row.Total++

		switch classifyIssueOutcome(issue) {
		case "completion":
			row.Completion++
		case "abandoned":
			row.Abandoned++
		default:
			row.Open++
		}
	}

	result := make([]OutcomeSkillCounts, 0, len(counts))
	for _, row := range counts {
		result = append(result, *row)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Total != result[j].Total {
			return result[i].Total > result[j].Total
		}
		return result[i].Skill < result[j].Skill
	})

	return result
}

func inferIssueSkill(issue beads.Issue) string {
	if skill := daemon.InferSkillFromLabels(issue.Labels); skill != "" {
		return skill
	}

	if skill := daemon.InferSkillFromTitle(issue.Title); skill != "" {
		return skill
	}

	skill, err := daemon.InferSkill(issue.IssueType)
	if err == nil && skill != "" {
		return skill
	}

	if issue.IssueType != "" {
		return issue.IssueType
	}

	return "unknown"
}

func classifyIssueOutcome(issue beads.Issue) string {
	status := strings.ToLower(strings.TrimSpace(issue.Status))

	if status == "abandoned" {
		return "abandoned"
	}

	if status == "closed" {
		if isAbandonedCloseReason(issue.CloseReason) {
			return "abandoned"
		}
		return "completion"
	}

	if status == "deferred" || status == "tombstone" {
		return "open"
	}

	return "open"
}

func isAbandonedCloseReason(reason string) bool {
	reason = strings.ToLower(reason)
	return strings.Contains(reason, "abandon") || strings.Contains(reason, "dead session")
}

func collectSpawnTimes(projectDir string) (map[string][]time.Time, error) {
	timesByIssue := map[string][]time.Time{}

	if err := scanWorkspaceSpawnTimes(filepath.Join(projectDir, ".orch", "workspace"), timesByIssue); err != nil {
		return nil, err
	}

	if err := scanWorkspaceSpawnTimes(filepath.Join(projectDir, ".orch", "workspace", "archived"), timesByIssue); err != nil {
		return nil, err
	}

	for id := range timesByIssue {
		sort.Slice(timesByIssue[id], func(i, j int) bool {
			return timesByIssue[id][i].Before(timesByIssue[id][j])
		})
	}

	return timesByIssue, nil
}

func scanWorkspaceSpawnTimes(workspaceDir string, timesByIssue map[string][]time.Time) error {
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		path := filepath.Join(workspaceDir, entry.Name())
		beadsID := spawn.ReadBeadsID(path)
		if beadsID == "" {
			continue
		}

		spawnTime := spawn.ReadSpawnTime(path)
		if spawnTime.IsZero() {
			continue
		}

		timesByIssue[beadsID] = append(timesByIssue[beadsID], spawnTime)
	}

	return nil
}

func collectDurationDistribution(issues []beads.Issue, spawnTimes map[string][]time.Time) OutcomeDurationDistribution {
	durations := []float64{}
	unmatchedClosed := 0

	for _, issue := range issues {
		if issue.ClosedAt == "" {
			continue
		}

		closedAt, err := time.Parse(time.RFC3339, issue.ClosedAt)
		if err != nil {
			continue
		}

		times := spawnTimes[issue.ID]
		if len(times) == 0 {
			unmatchedClosed++
			continue
		}

		spawnTime := matchSpawnTime(times, closedAt)
		durationMinutes := closedAt.Sub(spawnTime).Minutes()
		if durationMinutes <= 0 {
			continue
		}

		durations = append(durations, durationMinutes)
	}

	sort.Float64s(durations)

	return OutcomeDurationDistribution{
		Samples:         len(durations),
		P50Minutes:      percentile(durations, 0.50),
		P90Minutes:      percentile(durations, 0.90),
		P95Minutes:      percentile(durations, 0.95),
		UnmatchedClosed: unmatchedClosed,
	}
}

func matchSpawnTime(spawnTimes []time.Time, closedAt time.Time) time.Time {
	matched := spawnTimes[0]
	for _, spawnTime := range spawnTimes {
		if spawnTime.After(closedAt) {
			break
		}
		matched = spawnTime
	}
	return matched
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}

	if len(values) == 1 {
		return values[0]
	}

	position := p * float64(len(values)-1)
	low := int(math.Floor(position))
	high := int(math.Ceil(position))

	if low == high {
		return values[low]
	}

	weight := position - float64(low)
	return values[low] + (values[high]-values[low])*weight
}

func collectAbandonmentReasons(eventsPath string) ([]OutcomeReasonCount, error) {
	events, err := parseEvents(eventsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []OutcomeReasonCount{}, nil
		}
		return nil, err
	}

	counts := map[string]int{}
	for _, event := range events {
		if event.Type != "agent.abandoned" {
			continue
		}

		reason := "unknown"
		if event.Data != nil {
			if value, ok := event.Data["reason"].(string); ok {
				cleaned := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
				if cleaned != "" {
					reason = cleaned
				}
			}
		}

		counts[reason]++
	}

	result := make([]OutcomeReasonCount, 0, len(counts))
	for reason, count := range counts {
		result = append(result, OutcomeReasonCount{Reason: reason, Count: count})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		return result[i].Reason < result[j].Reason
	})

	return result, nil
}

func collectInvestigationToModelThroughput(projectDir string) (OutcomeInvestigationModelStats, error) {
	investigations, err := listMarkdownFiles(filepath.Join(projectDir, ".kb", "investigations"), true)
	if err != nil {
		return OutcomeInvestigationModelStats{}, err
	}

	models, err := listMarkdownFiles(filepath.Join(projectDir, ".kb", "models"), false)
	if err != nil {
		return OutcomeInvestigationModelStats{}, err
	}

	modelContent := make([]string, 0, len(models))
	for _, modelPath := range models {
		data, err := os.ReadFile(modelPath)
		if err != nil {
			continue
		}
		modelContent = append(modelContent, string(data))
	}

	promoted := 0
	for _, invPath := range investigations {
		name := filepath.Base(invPath)
		for _, content := range modelContent {
			if strings.Contains(content, name) {
				promoted++
				break
			}
		}
	}

	rate := 0.0
	if len(investigations) > 0 {
		rate = float64(promoted) / float64(len(investigations)) * 100
	}

	return OutcomeInvestigationModelStats{
		TotalInvestigations:             len(investigations),
		InvestigationsWithModelCitation: promoted,
		ThroughputRate:                  rate,
	}, nil
}

func listMarkdownFiles(root string, skipArchived bool) ([]string, error) {
	files := []string{}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if skipArchived && d.Name() == "archived" {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(d.Name(), ".md") {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	return files, nil
}

func outputOutcomesJSON(report *OutcomeReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal outcomes JSON: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

func outputOutcomesText(report *OutcomeReport) error {
	fmt.Println()
	fmt.Println("Outcome Metrics")
	fmt.Println("===============")
	fmt.Printf("Generated: %s\n", report.GeneratedAt)
	fmt.Printf("Project:   %s\n", report.ProjectDir)

	fmt.Println()
	fmt.Println("Counts by Skill")
	for _, row := range report.CountsBySkill {
		fmt.Printf("  %-24s completion:%4d  open:%4d  abandoned:%4d  total:%4d\n", row.Skill, row.Completion, row.Open, row.Abandoned, row.Total)
	}

	fmt.Println()
	fmt.Println("Spawn-to-Close Duration (minutes)")
	fmt.Printf("  samples:%d  p50:%.2f  p90:%.2f  p95:%.2f  unmatched_closed:%d\n",
		report.DurationDistribution.Samples,
		report.DurationDistribution.P50Minutes,
		report.DurationDistribution.P90Minutes,
		report.DurationDistribution.P95Minutes,
		report.DurationDistribution.UnmatchedClosed,
	)

	fmt.Println()
	fmt.Println("Abandonment Reasons")
	if len(report.AbandonmentReasons) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, reason := range report.AbandonmentReasons {
			fmt.Printf("  %-40s %4d\n", reason.Reason, reason.Count)
		}
	}

	fmt.Println()
	fmt.Println("Investigation -> Model Throughput")
	fmt.Printf("  cited:%d/%d (%.1f%%)\n",
		report.InvestigationToModel.InvestigationsWithModelCitation,
		report.InvestigationToModel.TotalInvestigations,
		report.InvestigationToModel.ThroughputRate,
	)

	return nil
}
