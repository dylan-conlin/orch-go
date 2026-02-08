package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	ansiReset  = "\x1b[0m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiRed    = "\x1b[31m"
	ansiGray   = "\x1b[90m"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Show operator health summary from orch serve",
	Long: `Fetch operator health telemetry from orch serve and print a
terminal-friendly summary for the six health cards:
  - crash-free streak
  - resource ceilings
  - investigations (30d)
  - defect clusters
  - agent health ratio (7d)
  - process census

This complements:
  - orch doctor    (service liveness and correctness)
  - orch stability (Phase 3 streak tracking)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHealth()
	},
}

func init() {
	rootCmd.AddCommand(healthCmd)
}

func runHealth() error {
	projectDir, err := currentProjectDir()
	if err != nil {
		return fmt.Errorf("failed to resolve project directory: %w", err)
	}

	baseURL := fmt.Sprintf("https://localhost:%d", DefaultServePort)
	client := &http.Client{
		Timeout: 4 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfigSkipVerify(),
		},
	}

	return runHealthWithClient(client, baseURL, projectDir)
}

func runHealthWithClient(client *http.Client, baseURL, projectDir string) error {
	report, err := fetchOperatorHealthReport(client, baseURL, projectDir)
	if err != nil {
		return err
	}

	fmt.Print(formatOperatorHealthSummary(report))
	return nil
}

func fetchOperatorHealthReport(client *http.Client, baseURL, projectDir string) (*OperatorHealthResponse, error) {
	if client == nil {
		return nil, fmt.Errorf("http client is required")
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid orch serve URL %q: %w", baseURL, err)
	}

	endpoint, err := base.Parse("/api/operator-health")
	if err != nil {
		return nil, fmt.Errorf("failed to build operator health endpoint: %w", err)
	}

	query := endpoint.Query()
	if projectDir != "" {
		query.Set("project", projectDir)
	}
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequest(http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("orch serve does not appear to be running at %s; start it with 'orch-dashboard start' (or run 'orch doctor --fix'): %w", baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		bodyText := strings.TrimSpace(string(body))
		if bodyText == "" {
			return nil, fmt.Errorf("operator health endpoint returned %s", resp.Status)
		}
		return nil, fmt.Errorf("operator health endpoint returned %s: %s", resp.Status, bodyText)
	}

	var report OperatorHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		return nil, fmt.Errorf("failed to decode operator health response: %w", err)
	}

	return &report, nil
}

func formatOperatorHealthSummary(report *OperatorHealthResponse) string {
	if report == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString("orch health - Operator Health\n")
	b.WriteString("============================\n")
	b.WriteString("\n")
	if report.GeneratedAt != "" {
		b.WriteString(fmt.Sprintf("Updated: %s\n\n", formatHealthTimestamp(report.GeneratedAt)))
	}

	writeHealthCard(&b, report.CrashFreeStreak.Status, "Crash-Free Streak", []string{
		fmt.Sprintf("Current: %s", defaultIfEmpty(report.CrashFreeStreak.CurrentStreak, "No stability history yet")),
		fmt.Sprintf("Streak: %d day(s), target %d day(s), %.0f%% progress", report.CrashFreeStreak.CurrentStreakDays, report.CrashFreeStreak.TargetDays, report.CrashFreeStreak.ProgressPercent),
		formatLastIntervention(report.CrashFreeStreak.LastIntervention),
	})

	resourceLines := []string{
		fmt.Sprintf("Goroutines: %d / %d", report.ResourceCeilings.Current.Goroutines, report.ResourceCeilings.Baseline.Goroutines),
		fmt.Sprintf("Heap: %s / %s", formatHealthBytes(report.ResourceCeilings.Current.HeapBytes), formatHealthBytes(report.ResourceCeilings.Baseline.HeapBytes)),
		fmt.Sprintf("Child processes: %d / %d", report.ResourceCeilings.Current.ChildProcesses, report.ResourceCeilings.Baseline.ChildProcesses),
		fmt.Sprintf("File descriptors: %d / %d", report.ResourceCeilings.Current.OpenFileDescriptors, report.ResourceCeilings.Baseline.OpenFileDescriptors),
	}
	if report.ResourceCeilings.Breached {
		resourceLines = append(resourceLines, fmt.Sprintf("Breached: %d metric(s) over %dx baseline", len(report.ResourceCeilings.Breaches), report.ResourceCeilings.CeilingMultiplier))
	}
	if errSummary := formatResourceErrors(report.ResourceCeilings.BaselineErrors, report.ResourceCeilings.CurrentErrors); errSummary != "" {
		resourceLines = append(resourceLines, errSummary)
	}
	writeHealthCard(&b, report.ResourceCeilings.Status, "Resource Ceilings", resourceLines)

	writeHealthCard(&b, report.InvestigationRate30d.Status, "Investigations (30d)", []string{
		fmt.Sprintf("Count: %d in %d day window", report.InvestigationRate30d.Count, report.InvestigationRate30d.WindowDays),
		fmt.Sprintf("Warning at >= %d, critical at >= %d", report.InvestigationRate30d.WarningFrom, report.InvestigationRate30d.Threshold),
	})

	defectLines := []string{}
	if len(report.DefectClassClusters.TopClasses) == 0 {
		defectLines = append(defectLines, "No clustered defect class signal")
	} else {
		limit := len(report.DefectClassClusters.TopClasses)
		if limit > 4 {
			limit = 4
		}
		for i := 0; i < limit; i++ {
			item := report.DefectClassClusters.TopClasses[i]
			defectLines = append(defectLines, fmt.Sprintf("%s: %d", item.DefectClass, item.Count))
		}
	}
	writeHealthCard(&b, report.DefectClassClusters.Status, "Defect Clusters (30d)", defectLines)

	ratioText := "n/a"
	if report.AgentHealthRatio7d.CompletionsPerAbandonment != nil {
		ratioText = fmt.Sprintf("%.2f", *report.AgentHealthRatio7d.CompletionsPerAbandonment)
	}
	writeHealthCard(&b, report.AgentHealthRatio7d.Status, "Agent Health Ratio (7d)", []string{
		fmt.Sprintf("Outcomes: %d completions, %d abandonments", report.AgentHealthRatio7d.Completions, report.AgentHealthRatio7d.Abandonments),
		fmt.Sprintf("Completion share: %.0f%%", report.AgentHealthRatio7d.CompletionShare*100),
		fmt.Sprintf("Completions per abandonment: %s", ratioText),
	})

	processLines := []string{
		fmt.Sprintf("Active child processes: %d", report.ProcessCensus.ChildProcesses),
		fmt.Sprintf("Orphaned processes (PPID=1): %d", report.ProcessCensus.OrphanedCount),
	}
	if len(report.ProcessCensus.OrphanedProcesses) > 0 {
		limit := len(report.ProcessCensus.OrphanedProcesses)
		if limit > 3 {
			limit = 3
		}
		for i := 0; i < limit; i++ {
			orphan := report.ProcessCensus.OrphanedProcesses[i]
			processLines = append(processLines, fmt.Sprintf("PID %d: %s", orphan.PID, orphan.Command))
		}
	}
	writeHealthCard(&b, report.ProcessCensus.Status, "Process Census", processLines)

	if len(report.Errors) > 0 {
		b.WriteString(colorizeStatus(operatorHealthStatusWarning, "PARTIAL DATA") + " " + strings.Join(report.Errors, " | ") + "\n")
	}

	return b.String()
}

func writeHealthCard(builder *strings.Builder, status, title string, lines []string) {
	builder.WriteString(fmt.Sprintf("%s %s\n", colorizeStatus(status, strings.ToUpper(defaultIfEmpty(status, operatorHealthStatusUnknown))), title))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		builder.WriteString(fmt.Sprintf("  - %s\n", line))
	}
	builder.WriteString("\n")
}

func colorizeStatus(status, text string) string {
	return statusColor(status) + text + ansiReset
}

func statusColor(status string) string {
	switch status {
	case operatorHealthStatusHealthy:
		return ansiGreen
	case operatorHealthStatusWarning:
		return ansiYellow
	case operatorHealthStatusCritical:
		return ansiRed
	default:
		return ansiGray
	}
}

func formatHealthTimestamp(raw string) string {
	ts, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	return ts.Local().Format("2006-01-02 15:04:05 MST")
}

func formatLastIntervention(last *operatorInterventionSummary) string {
	if last == nil {
		return ""
	}
	when := formatHealthTimestamp(last.Timestamp)
	detail := ""
	if last.Detail != "" {
		detail = ": " + last.Detail
	}
	return fmt.Sprintf("Last recovery: %s at %s%s", last.Source, when, detail)
}

func formatHealthBytes(bytes int64) string {
	if bytes < 0 {
		return "n/a"
	}
	const (
		kiB = 1024
		miB = 1024 * kiB
		giB = 1024 * miB
	)
	switch {
	case bytes < kiB:
		return fmt.Sprintf("%d B", bytes)
	case bytes < miB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/kiB)
	case bytes < giB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/miB)
	default:
		return fmt.Sprintf("%.2f GB", float64(bytes)/giB)
	}
}

func formatResourceErrors(baselineErrors, currentErrors map[string]string) string {
	total := len(baselineErrors) + len(currentErrors)
	if total == 0 {
		return ""
	}

	keys := make([]string, 0, total)
	for key := range baselineErrors {
		keys = append(keys, "baseline."+key)
	}
	for key := range currentErrors {
		keys = append(keys, "current."+key)
	}
	sort.Strings(keys)

	if len(keys) > 4 {
		keys = keys[:4]
	}

	return fmt.Sprintf("Metric collection warnings: %s", strings.Join(keys, ", "))
}

func defaultIfEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
