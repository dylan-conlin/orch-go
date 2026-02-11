package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/process"
	"github.com/dylan-conlin/orch-go/pkg/stability"
)

const (
	operatorHealthStatusHealthy  = "healthy"
	operatorHealthStatusWarning  = "warning"
	operatorHealthStatusCritical = "critical"
	operatorHealthStatusUnknown  = "unknown"

	operatorHealthAgentWindowDays = 7

	operatorHealthDefectWindowDays        = 30
	operatorHealthDefectCriticalThreshold = 10
	operatorHealthDefectWarningThreshold  = 5

	operatorHealthOrphanProcessLimit = 20
)

// OperatorHealthResponse is the JSON structure returned by /api/operator-health.
// It surfaces system-level behavioral health signals in operator language.
type OperatorHealthResponse struct {
	GeneratedAt         string                    `json:"generated_at"`
	CrashFreeStreak     crashFreeStreakMetric     `json:"crash_free_streak"`
	ResourceCeilings    resourceCeilingsMetric    `json:"resource_ceilings"`
	DefectClassClusters defectClassClustersMetric `json:"defect_class_clusters"`
	AgentHealthRatio7d  agentHealthRatioMetric    `json:"agent_health_ratio_7d"`
	ProcessCensus       processCensusMetric       `json:"process_census"`
	ZombieProcesses     zombieProcessMetric       `json:"zombie_processes"`
	Errors              []string                  `json:"errors,omitempty"`
}

type crashFreeStreakMetric struct {
	Status               string                       `json:"status"`
	CurrentStreakDays    int                          `json:"current_streak_days"`
	CurrentStreakSeconds int64                        `json:"current_streak_seconds"`
	CurrentStreak        string                       `json:"current_streak"`
	TargetDays           int                          `json:"target_days"`
	ProgressPercent      float64                      `json:"progress_percent"`
	LastIntervention     *operatorInterventionSummary `json:"last_intervention,omitempty"`
}

type operatorInterventionSummary struct {
	Timestamp string `json:"timestamp"`
	Source    string `json:"source"`
	Detail    string `json:"detail,omitempty"`
	BeadsID   string `json:"beads_id,omitempty"`
}

type resourceCeilingsMetric struct {
	Status            string            `json:"status"`
	Baseline          resourceMetrics   `json:"baseline"`
	Current           resourceMetrics   `json:"current"`
	CeilingMultiplier int64             `json:"ceiling_multiplier"`
	Breached          bool              `json:"breached"`
	Breaches          []resourceBreach  `json:"breaches,omitempty"`
	BaselineErrors    map[string]string `json:"baseline_errors,omitempty"`
	CurrentErrors     map[string]string `json:"current_errors,omitempty"`
}

type defectClassClustersMetric struct {
	Status     string                   `json:"status"`
	WindowDays int                      `json:"window_days"`
	TopClasses []defectClassClusterItem `json:"top_classes"`
	TotalTopN  int                      `json:"total_top_n"`
}

type defectClassClusterItem struct {
	DefectClass string `json:"defect_class"`
	Count       int    `json:"count"`
	WindowDays  int    `json:"window_days,omitempty"`
}

type agentHealthRatioMetric struct {
	Status                    string   `json:"status"`
	WindowDays                int      `json:"window_days"`
	Completions               int      `json:"completions"`
	Abandonments              int      `json:"abandonments"`
	CompletionShare           float64  `json:"completion_share"`
	CompletionsPerAbandonment *float64 `json:"completions_per_abandonment,omitempty"`
}

type processCensusMetric struct {
	Status            string               `json:"status"`
	ChildProcesses    int64                `json:"child_processes"`
	OrphanedCount     int                  `json:"orphaned_count"`
	OrphanedProcesses []orphanProcessEntry `json:"orphaned_processes,omitempty"`
}

type zombieProcessMetric struct {
	Status         string `json:"status"`
	BunAgentCount  int    `json:"bun_agent_count"`
	ActiveSessions int    `json:"active_sessions"`
	OrphanCount    int    `json:"orphan_count"`
	APIAvailable   bool   `json:"api_available"`
}

type orphanProcessEntry struct {
	PID     int    `json:"pid"`
	PPID    int    `json:"ppid"`
	Command string `json:"command"`
	Args    string `json:"args,omitempty"`
}

// handleOperatorHealth returns operator-readable behavioral health signals.
func (s *Server) handleOperatorHealth(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	projectDir := r.URL.Query().Get("project")
	if projectDir == "" {
		var err error
		projectDir, err = s.currentProjectDir()
		if err != nil {
			jsonErr(w, http.StatusInternalServerError, fmt.Sprintf("Failed to resolve project directory: %v", err))
			return
		}
	}

	response := buildOperatorHealthResponse(s, projectDir, time.Now())
	if err := jsonOK(w, response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode operator health: %v", err), http.StatusInternalServerError)
		return
	}
}

func buildOperatorHealthResponse(s *Server, projectDir string, now time.Time) OperatorHealthResponse {
	response := OperatorHealthResponse{
		GeneratedAt: now.UTC().Format(time.RFC3339),
	}

	var errs []string

	crashFree, err := buildCrashFreeStreakMetric()
	if err != nil {
		errs = append(errs, fmt.Sprintf("crash_free_streak: %v", err))
	}
	response.CrashFreeStreak = crashFree

	resources := buildResourceCeilingsMetric(s)
	response.ResourceCeilings = resources

	defectClusters, err := buildDefectClassClustersMetric(projectDir)
	if err != nil {
		errs = append(errs, fmt.Sprintf("defect_class_clusters: %v", err))
	}
	response.DefectClassClusters = defectClusters

	agentRatio, err := buildAgentHealthRatioMetric()
	if err != nil {
		errs = append(errs, fmt.Sprintf("agent_health_ratio_7d: %v", err))
	}
	response.AgentHealthRatio7d = agentRatio

	processCensus, err := buildProcessCensusMetric(resources.Current.ChildProcesses)
	if err != nil {
		errs = append(errs, fmt.Sprintf("process_census: %v", err))
	}
	response.ProcessCensus = processCensus

	openCodeURL := "http://localhost:4096"
	if s != nil && s.ServerURL != "" {
		openCodeURL = s.ServerURL
	}
	zombies, err := buildZombieProcessMetric(openCodeURL)
	if err != nil {
		errs = append(errs, fmt.Sprintf("zombie_processes: %v", err))
	}
	response.ZombieProcesses = zombies

	if len(errs) > 0 {
		response.Errors = errs
	}

	return response
}

func buildCrashFreeStreakMetric() (crashFreeStreakMetric, error) {
	metric := crashFreeStreakMetric{
		Status:     operatorHealthStatusUnknown,
		TargetDays: int(stability.TargetDuration.Hours() / 24),
	}

	report, err := stability.ComputeReport(stability.DefaultPath(), 30)
	if err != nil {
		return metric, err
	}

	if !report.HasData {
		metric.CurrentStreak = "No stability history yet"
		return metric, nil
	}

	metric.CurrentStreakDays = int(report.CurrentStreak.Hours() / 24)
	metric.CurrentStreakSeconds = int64(report.CurrentStreak.Seconds())
	metric.CurrentStreak = stability.FormatDuration(report.CurrentStreak)
	metric.ProgressPercent = report.ProgressPercent

	latestIntervention, err := readLatestStabilityIntervention(stability.DefaultPath())
	if err != nil {
		return metric, err
	}
	if latestIntervention != nil {
		metric.LastIntervention = &operatorInterventionSummary{
			Timestamp: time.Unix(latestIntervention.Ts, 0).UTC().Format(time.RFC3339),
			Source:    latestIntervention.Source,
			Detail:    latestIntervention.Detail,
			BeadsID:   latestIntervention.BeadsID,
		}
	}

	switch {
	case report.CurrentStreak >= stability.TargetDuration:
		metric.Status = operatorHealthStatusHealthy
	case report.CurrentStreak >= 3*24*time.Hour:
		metric.Status = operatorHealthStatusWarning
	default:
		metric.Status = operatorHealthStatusCritical
	}

	return metric, nil
}

func readLatestStabilityIntervention(path string) (*stability.Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var latest *stability.Entry

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry stability.Entry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry.Type != stability.TypeIntervention {
			continue
		}
		if !stability.IsInfrastructureIntervention(entry.Source) {
			continue
		}

		if latest == nil || entry.Ts > latest.Ts {
			entryCopy := entry
			latest = &entryCopy
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return latest, nil
}
func buildResourceCeilingsMetric(s *Server) resourceCeilingsMetric {
	var report resourceHealthReport
	if s != nil && s.ResourceMonitor != nil {
		report = s.ResourceMonitor.sampleAndCheck()
	} else {
		sample := collectResourceSample()
		report = resourceHealthReport{
			Baseline:          sample.metrics,
			Current:           sample.metrics,
			CeilingMultiplier: resourceCeilingMultiplier,
			Breached:          false,
			BaselineErrors:    copyStringMap(sample.errors),
			CurrentErrors:     copyStringMap(sample.errors),
		}
	}

	status := operatorHealthStatusHealthy
	if report.Breached {
		status = operatorHealthStatusCritical
	} else if len(report.CurrentErrors) > 0 || len(report.BaselineErrors) > 0 {
		status = operatorHealthStatusWarning
	}

	return resourceCeilingsMetric{
		Status:            status,
		Baseline:          report.Baseline,
		Current:           report.Current,
		CeilingMultiplier: report.CeilingMultiplier,
		Breached:          report.Breached,
		Breaches:          report.Breaches,
		BaselineErrors:    report.BaselineErrors,
		CurrentErrors:     report.CurrentErrors,
	}
}

func buildDefectClassClustersMetric(projectDir string) (defectClassClustersMetric, error) {
	items, err := fetchKBReflect(projectDir, "defect-class")
	if err != nil {
		return defectClassClustersMetric{
			Status:     operatorHealthStatusUnknown,
			WindowDays: operatorHealthDefectWindowDays,
			TopClasses: []defectClassClusterItem{},
		}, err
	}

	clusters := make([]defectClassClusterItem, 0, len(items))
	for _, item := range items {
		defectClass, _ := item["defect_class"].(string)
		if defectClass == "" {
			continue
		}
		count := intFromInterface(item["count"])
		if count <= 0 {
			continue
		}

		cluster := defectClassClusterItem{
			DefectClass: defectClass,
			Count:       count,
			WindowDays:  intFromInterface(item["window_days"]),
		}
		clusters = append(clusters, cluster)
	}

	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Count > clusters[j].Count
	})

	status := operatorHealthStatusHealthy
	if len(clusters) > 0 {
		topCount := clusters[0].Count
		if topCount >= operatorHealthDefectCriticalThreshold {
			status = operatorHealthStatusCritical
		} else if topCount >= operatorHealthDefectWarningThreshold {
			status = operatorHealthStatusWarning
		}
	}

	return defectClassClustersMetric{
		Status:     status,
		WindowDays: operatorHealthDefectWindowDays,
		TopClasses: clusters,
		TotalTopN:  len(clusters),
	}, nil
}

func buildAgentHealthRatioMetric() (agentHealthRatioMetric, error) {
	metric := agentHealthRatioMetric{
		Status:     operatorHealthStatusUnknown,
		WindowDays: operatorHealthAgentWindowDays,
	}

	events, err := parseEvents(getEventsPath())
	if err != nil {
		if strings.Contains(err.Error(), "events.jsonl not found") {
			return metric, nil
		}
		return metric, err
	}

	stats := aggregateStats(events, operatorHealthAgentWindowDays, false)
	metric.Completions = stats.Summary.TotalCompletions
	metric.Abandonments = stats.Summary.TotalAbandonments

	totalOutcomes := metric.Completions + metric.Abandonments
	if totalOutcomes == 0 {
		return metric, nil
	}

	metric.CompletionShare = float64(metric.Completions) / float64(totalOutcomes)
	if metric.Abandonments > 0 {
		ratio := float64(metric.Completions) / float64(metric.Abandonments)
		metric.CompletionsPerAbandonment = &ratio
	}

	switch {
	case metric.CompletionShare >= 0.80:
		metric.Status = operatorHealthStatusHealthy
	case metric.CompletionShare >= 0.60:
		metric.Status = operatorHealthStatusWarning
	default:
		metric.Status = operatorHealthStatusCritical
	}

	return metric, nil
}

func buildProcessCensusMetric(childProcesses int64) (processCensusMetric, error) {
	metric := processCensusMetric{
		Status:         operatorHealthStatusHealthy,
		ChildProcesses: childProcesses,
	}

	orphans, err := listOrphanedOrchProcesses(operatorHealthOrphanProcessLimit)
	if err != nil {
		metric.Status = operatorHealthStatusUnknown
		return metric, err
	}

	metric.OrphanedCount = len(orphans)
	metric.OrphanedProcesses = orphans

	if metric.OrphanedCount > 0 {
		metric.Status = operatorHealthStatusCritical
	} else if childProcesses < 0 {
		metric.Status = operatorHealthStatusWarning
	}

	return metric, nil
}

// buildZombieProcessMetric compares bun agent processes against active OpenCode
// sessions to detect zombies — processes that are still running but no longer
// associated with any session. These are the processes that accumulate and
// eventually exhaust RAM, crashing WindowServer (mouse stops working).
func buildZombieProcessMetric(openCodeURL string) (zombieProcessMetric, error) {
	metric := zombieProcessMetric{
		Status: operatorHealthStatusUnknown,
	}

	agents, err := process.FindAgentProcesses()
	if err != nil {
		return metric, fmt.Errorf("find agent processes: %w", err)
	}
	metric.BunAgentCount = len(agents)

	if len(agents) == 0 {
		metric.Status = operatorHealthStatusHealthy
		metric.APIAvailable = true
		return metric, nil
	}

	client := opencode.NewClient(openCodeURL)
	sessions, err := client.ListSessions("")
	if err != nil {
		// API unavailable — can't determine zombies, report what we know
		metric.APIAvailable = false
		if len(agents) > 0 {
			metric.Status = operatorHealthStatusWarning
		}
		return metric, nil
	}

	metric.APIAvailable = true
	metric.ActiveSessions = len(sessions)

	activeIDs := make(map[string]bool, len(sessions))
	activeTitles := make(map[string]bool, len(sessions))
	for _, s := range sessions {
		if s.ID != "" {
			activeIDs[s.ID] = true
		}
		if s.Title != "" {
			activeTitles[s.Title] = true
		}
	}

	orphans, err := process.FindOrphanProcesses(activeTitles, activeIDs)
	if err != nil {
		return metric, fmt.Errorf("find orphan processes: %w", err)
	}
	metric.OrphanCount = len(orphans)

	switch {
	case metric.OrphanCount == 0:
		metric.Status = operatorHealthStatusHealthy
	case metric.OrphanCount <= 2:
		metric.Status = operatorHealthStatusWarning
	default:
		metric.Status = operatorHealthStatusCritical
	}

	return metric, nil
}

func listOrphanedOrchProcesses(limit int) ([]orphanProcessEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ps", "-eo", "pid=,ppid=,comm=,args=")
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("ps query timed out: %w", ctx.Err())
		}
		return nil, fmt.Errorf("ps query failed: %w", err)
	}

	return parseOrphanedOrchProcesses(string(output), limit, os.Getpid())
}

func parseOrphanedOrchProcesses(output string, limit, selfPID int) ([]orphanProcessEntry, error) {
	orphans := make([]orphanProcessEntry, 0)
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}

		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		if ppid != 1 {
			continue
		}
		if pid == selfPID {
			continue
		}

		command := fields[2]
		args := strings.Join(fields[3:], " ")
		if !isOrchRelatedProcess(command, args) {
			continue
		}

		entry := orphanProcessEntry{
			PID:     pid,
			PPID:    ppid,
			Command: command,
			Args:    truncateOrphanArgs(args, 160),
		}
		orphans = append(orphans, entry)

		if limit > 0 && len(orphans) >= limit {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed parsing ps output: %w", err)
	}

	return orphans, nil
}

func isOrchRelatedProcess(command, args string) bool {
	commandLower := strings.ToLower(command)
	argsLower := strings.ToLower(args)
	text := commandLower + " " + argsLower

	// Whitelist: legitimate PPID=1 processes that should NOT be flagged as orphans
	legitimateProcesses := []string{
		"overmind", // Process manager launched by orch-dashboard
		"tmux",     // Terminal multiplexer managed by launchd
		"launchd",  // System init process
	}

	for _, legitimate := range legitimateProcesses {
		if commandLower == legitimate || strings.HasPrefix(commandLower, legitimate) {
			return false
		}
	}

	// Launchd-managed opencode server is intentionally PPID=1.
	isLaunchdOpencodeServe := (strings.Contains(commandLower, "/.bun/bin/opencode") || strings.Contains(argsLower, "/.bun/bin/opencode")) && strings.Contains(" "+argsLower, " serve")
	if isLaunchdOpencodeServe {
		return false
	}

	// Sketchybar helper scripts poll orch status and may be PPID=1.
	if strings.Contains(commandLower, "/.config/sketchybar/helpers/") || strings.Contains(argsLower, "/.config/sketchybar/helpers/") {
		return false
	}

	// macOS system processes
	if strings.HasPrefix(command, "/System/Library/") {
		return false
	}

	// Development servers (vite, etc.) should not be flagged as orphans
	// They are intentionally long-running processes
	if strings.Contains(argsLower, "vite") && strings.Contains(argsLower, "dev") {
		return false
	}

	// Orch-related keywords that indicate potential orphans
	orchKeywords := []string{
		"opencode",
		"orch",
		".orch",
		"run --attach",
		"beads",
	}

	for _, keyword := range orchKeywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}

	return false
}

func truncateOrphanArgs(args string, maxLen int) string {
	if maxLen <= 0 || len(args) <= maxLen {
		return args
	}
	return args[:maxLen-3] + "..."
}

func intFromInterface(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0
		}
		return int(i)
	case string:
		i, err := strconv.Atoi(v)
		if err != nil {
			return 0
		}
		return i
	default:
		return 0
	}
}
