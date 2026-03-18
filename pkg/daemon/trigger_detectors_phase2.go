// Package daemon provides autonomous overnight processing capabilities.
// This file contains Phase 2 pattern detectors for the trigger scan system:
//   - model_contradictions: detects unresolved probe contradictions in kb models
//   - hotspot_acceleration: detects files growing rapidly (>200 lines/30d)
//   - knowledge_decay: detects models with no recent probes (30d+)
//   - skill_performance_drift: detects skills whose success rate dropped significantly
package daemon

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

// --- Model Contradictions Detector ---

// ModelContradictionsSource provides I/O for the model contradictions detector.
type ModelContradictionsSource interface {
	// ListUnresolvedContradictions scans .kb/models/*/probes/*.md for probes
	// containing "contradict" where the parent model wasn't updated after the probe.
	ListUnresolvedContradictions() ([]UnresolvedContradiction, error)
}

// UnresolvedContradiction represents a probe that contradicts its parent model.
type UnresolvedContradiction struct {
	ModelSlug     string
	ProbeFilename string
	ProbeDate     time.Time
	ModelUpdated  time.Time
}

// ModelContradictionsDetector detects unresolved contradictions between probes and models.
type ModelContradictionsDetector struct {
	Source ModelContradictionsSource
}

func (d *ModelContradictionsDetector) Name() string { return "model_contradictions" }

func (d *ModelContradictionsDetector) Detect() ([]TriggerSuggestion, error) {
	if d.Source == nil {
		return nil, fmt.Errorf("model contradictions source not configured")
	}

	contradictions, err := d.Source.ListUnresolvedContradictions()
	if err != nil {
		return nil, err
	}

	var suggestions []TriggerSuggestion
	for _, c := range contradictions {
		suggestions = append(suggestions, TriggerSuggestion{
			Detector:    "model_contradictions",
			Key:         c.ModelSlug + ":" + c.ProbeFilename,
			Title:       fmt.Sprintf("Unresolved model contradiction: %s (probe: %s)", c.ModelSlug, c.ProbeFilename),
			Description: fmt.Sprintf("Probe %s contradicts model %s but the model hasn't been updated since the probe was created. The model's last update was %s, probe date was %s.", c.ProbeFilename, c.ModelSlug, c.ModelUpdated.Format("2006-01-02"), c.ProbeDate.Format("2006-01-02")),
			IssueType:   "task",
			Priority:    2,
			Labels:      []string{"skill:capture-knowledge"},
		})
	}
	return suggestions, nil
}

// --- Hotspot Acceleration Detector ---

// HotspotAccelerationSource provides I/O for the hotspot acceleration detector.
type HotspotAccelerationSource interface {
	// ListFastGrowingFiles returns files that gained more than threshold lines in the window.
	ListFastGrowingFiles(threshold int) ([]FastGrowingFile, error)
}

// FastGrowingFile is a file that's growing rapidly.
type FastGrowingFile struct {
	Path           string
	NetGrowth      int
	CurrentSize    int
	HistoricalSize int
}

// HotspotAccelerationDetector detects files growing >200 lines/30 days.
type HotspotAccelerationDetector struct {
	Source    HotspotAccelerationSource
	Threshold int // Lines added threshold (default 200)
}

func (d *HotspotAccelerationDetector) Name() string { return "hotspot_acceleration" }

func (d *HotspotAccelerationDetector) Detect() ([]TriggerSuggestion, error) {
	if d.Source == nil {
		return nil, fmt.Errorf("hotspot acceleration source not configured")
	}

	threshold := d.Threshold
	if threshold <= 0 {
		threshold = 200
	}

	files, err := d.Source.ListFastGrowingFiles(threshold)
	if err != nil {
		return nil, err
	}

	var suggestions []TriggerSuggestion
	for _, f := range files {
		suggestions = append(suggestions, TriggerSuggestion{
			Detector:    "hotspot_acceleration",
			Key:         f.Path,
			Title:       fmt.Sprintf("Hotspot acceleration: %s (net +%d lines/30d, now %d lines)", f.Path, f.NetGrowth, f.CurrentSize),
			Description: fmt.Sprintf("File %s has grown by %d net lines in the last 30 days (%d → %d lines). Consider extraction before it becomes a critical hotspot.", f.Path, f.NetGrowth, f.HistoricalSize, f.CurrentSize),
			IssueType:   "investigation",
			Priority:    3,
			Labels:      []string{"skill:architect"},
		})
	}
	return suggestions, nil
}

// --- Knowledge Decay Detector ---

// KnowledgeDecaySource provides I/O for the knowledge decay detector.
type KnowledgeDecaySource interface {
	// ListDecayedModels returns models with no recent probes.
	ListDecayedModels(maxAge time.Duration) ([]DecayedModel, error)
}

// DecayedModel is a model with no recent probes.
type DecayedModel struct {
	Slug           string
	LastProbeDate  time.Time
	DaysSinceProbe int
}

// KnowledgeDecayDetector detects models with no probes in the last 30 days.
type KnowledgeDecayDetector struct {
	Source KnowledgeDecaySource
	MaxAge time.Duration // Max age without probe before flagging (default 30 days)
}

func (d *KnowledgeDecayDetector) Name() string { return "knowledge_decay" }

func (d *KnowledgeDecayDetector) Detect() ([]TriggerSuggestion, error) {
	if d.Source == nil {
		return nil, fmt.Errorf("knowledge decay source not configured")
	}

	maxAge := d.MaxAge
	if maxAge <= 0 {
		maxAge = 30 * 24 * time.Hour
	}

	models, err := d.Source.ListDecayedModels(maxAge)
	if err != nil {
		return nil, err
	}

	var suggestions []TriggerSuggestion
	for _, m := range models {
		suggestions = append(suggestions, TriggerSuggestion{
			Detector:    "knowledge_decay",
			Key:         m.Slug,
			Title:       fmt.Sprintf("Knowledge decay: %s (%dd since last probe)", m.Slug, m.DaysSinceProbe),
			Description: fmt.Sprintf("Model %s has not been probed in %d days. Consider creating a verification probe to check if the model's claims are still accurate.", m.Slug, m.DaysSinceProbe),
			IssueType:   "task",
			Priority:    4,
			Labels:      []string{"skill:investigation"},
		})
	}
	return suggestions, nil
}

// --- Skill Performance Drift Detector ---

// SkillPerformanceDriftSource provides I/O for the skill performance drift detector.
type SkillPerformanceDriftSource interface {
	// ListDriftedSkills returns skills whose success rate is below currentThreshold
	// with enough samples to be meaningful.
	ListDriftedSkills(currentThreshold, previousMin float64) ([]DriftedSkill, error)
}

// DriftedSkill is a skill whose performance has degraded.
type DriftedSkill struct {
	Name         string
	CurrentRate  float64
	PreviousRate float64
	RecentSpawns int
}

// SkillPerformanceDriftDetector detects skills with significant success rate drops.
type SkillPerformanceDriftDetector struct {
	Source           SkillPerformanceDriftSource
	CurrentThreshold float64 // Current rate below this triggers (default 0.5)
	PreviousMin      float64 // Previous rate must have been above this (default 0.7)
}

func (d *SkillPerformanceDriftDetector) Name() string { return "skill_performance_drift" }

func (d *SkillPerformanceDriftDetector) Detect() ([]TriggerSuggestion, error) {
	if d.Source == nil {
		return nil, fmt.Errorf("skill performance drift source not configured")
	}

	currentThreshold := d.CurrentThreshold
	if currentThreshold <= 0 {
		currentThreshold = 0.5
	}
	previousMin := d.PreviousMin
	if previousMin <= 0 {
		previousMin = 0.7
	}

	drifted, err := d.Source.ListDriftedSkills(currentThreshold, previousMin)
	if err != nil {
		return nil, err
	}

	var suggestions []TriggerSuggestion
	for _, s := range drifted {
		suggestions = append(suggestions, TriggerSuggestion{
			Detector:    "skill_performance_drift",
			Key:         s.Name,
			Title:       fmt.Sprintf("Skill performance drift: %s (%.0f%% → %.0f%%)", s.Name, s.PreviousRate*100, s.CurrentRate*100),
			Description: fmt.Sprintf("Skill %s success rate dropped from %.0f%% to %.0f%% (based on %d recent spawns). Investigate what changed.", s.Name, s.PreviousRate*100, s.CurrentRate*100, s.RecentSpawns),
			IssueType:   "investigation",
			Priority:    2,
			Labels:      []string{"skill:investigation"},
		})
	}
	return suggestions, nil
}

// --- Default Source Implementations (Phase 2) ---

// defaultModelContradictionsSource scans .kb/models/*/probes/*.md for contradictions.
type defaultModelContradictionsSource struct{}

func (s *defaultModelContradictionsSource) ListUnresolvedContradictions() ([]UnresolvedContradiction, error) {
	projectDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	modelsDir := filepath.Join(projectDir, ".kb", "models")
	modelEntries, err := os.ReadDir(modelsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []UnresolvedContradiction
	for _, modelEntry := range modelEntries {
		if !modelEntry.IsDir() {
			continue
		}

		modelSlug := modelEntry.Name()
		modelPath := filepath.Join(modelsDir, modelSlug, "model.md")
		modelInfo, err := os.Stat(modelPath)
		if err != nil {
			continue
		}
		modelUpdated := modelInfo.ModTime()

		probesDir := filepath.Join(modelsDir, modelSlug, "probes")
		probeEntries, err := os.ReadDir(probesDir)
		if err != nil {
			continue
		}

		for _, probeEntry := range probeEntries {
			if probeEntry.IsDir() || !strings.HasSuffix(probeEntry.Name(), ".md") {
				continue
			}

			probePath := filepath.Join(probesDir, probeEntry.Name())
			content, err := os.ReadFile(probePath)
			if err != nil {
				continue
			}

			contentLower := strings.ToLower(string(content))
			if !strings.Contains(contentLower, "contradict") {
				continue
			}

			// Parse probe date from filename (YYYY-MM-DD-...)
			probeName := probeEntry.Name()
			if len(probeName) >= 10 {
				probeDate, err := time.Parse("2006-01-02", probeName[:10])
				if err == nil && probeDate.After(modelUpdated) {
					result = append(result, UnresolvedContradiction{
						ModelSlug:     modelSlug,
						ProbeFilename: probeName,
						ProbeDate:     probeDate,
						ModelUpdated:  modelUpdated,
					})
				}
			}
		}
	}
	return result, nil
}

// skipAccelerationDirs are directory names excluded from hotspot acceleration detection.
// Mirrors skipBloatDirs in cmd/orch/hotspot.go plus experiments/ (static artifacts).
var skipAccelerationDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	".svelte-kit":  true,
	"dist":         true,
	"build":        true,
	"__pycache__":  true,
	".next":        true,
	".nuxt":        true,
	".output":      true,
	".opencode":    true,
	".orch":        true,
	".beads":       true,
	".claude":      true,
	"experiments":  true,
}

// isAccelerationExcluded returns true if the path should be excluded from
// hotspot acceleration detection (non-production directories, test files).
func isAccelerationExcluded(path string) bool {
	// Skip test files
	if strings.HasSuffix(path, "_test.go") {
		return true
	}
	// Skip generated files
	if strings.Contains(path, "/generated/") {
		return true
	}
	// Check if any directory component is in the skip set
	dir := filepath.Dir(path)
	for dir != "." && dir != "/" {
		if skipAccelerationDirs[filepath.Base(dir)] {
			return true
		}
		dir = filepath.Dir(dir)
	}
	// Also check if the first path component is a skipped dir (handles "experiments/...")
	if idx := strings.IndexByte(path, '/'); idx > 0 {
		if skipAccelerationDirs[path[:idx]] {
			return true
		}
	}
	return false
}

// minAccelerationSize is the minimum current file size (in lines) for a file to be
// flagged by the hotspot acceleration detector. Files smaller than this have enough
// headroom before hitting the 1500-line extraction threshold.
const minAccelerationSize = 500

// defaultHotspotAccelerationSource uses git diff --numstat to detect fast-growing files.
// Compares HEAD against a baseline commit ~30 days ago to measure net growth,
// eliminating false positives from churn (add+delete cycles in extractions/rewrites).
type defaultHotspotAccelerationSource struct{}

func (s *defaultHotspotAccelerationSource) ListFastGrowingFiles(threshold int) ([]FastGrowingFile, error) {
	projectDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Get the commit hash from ~30 days ago as our baseline
	baseCommit, err := runGitBaseCommit(projectDir)
	if err != nil {
		return nil, err
	}
	if baseCommit == "" {
		return nil, nil // repo younger than 30 days
	}

	// git diff --numstat gives net changes (added - deleted) per file between
	// two points in time, unlike git log --numstat which sums additions across
	// individual commits and counts churn as growth.
	output, err := runGitDiffNumstat(projectDir, baseCommit)
	if err != nil {
		return nil, err
	}
	netChanges := parseGitDiffNumstat(output)

	var result []FastGrowingFile
	for path, netGrowth := range netChanges {
		if netGrowth < threshold {
			continue
		}
		if !strings.HasSuffix(path, ".go") {
			continue
		}
		if isAccelerationExcluded(path) {
			continue
		}
		fullPath := filepath.Join(projectDir, path)
		currentSize, err := countFileLines(fullPath)
		if err != nil || currentSize < minAccelerationSize {
			continue
		}
		historicalSize := currentSize - netGrowth
		if historicalSize < 0 {
			historicalSize = 0
		}
		result = append(result, FastGrowingFile{
			Path:           path,
			NetGrowth:      netGrowth,
			CurrentSize:    currentSize,
			HistoricalSize: historicalSize,
		})
	}
	return result, nil
}

// runGitBaseCommit returns the commit hash from approximately 30 days ago.
func runGitBaseCommit(dir string) (string, error) {
	cmd := exec.Command("git", "rev-list", "-1", "--before=30 days ago", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-list failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// runGitDiffNumstat runs git diff --numstat between a base commit and HEAD.
// Unlike git log --numstat (which sums per-commit additions), this gives
// the true net change per file between two points in time.
func runGitDiffNumstat(dir, baseCommit string) (string, error) {
	cmd := exec.Command("git", "diff", "--numstat", "--no-renames", baseCommit, "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff --numstat failed: %w", err)
	}
	return string(out), nil
}

// parseGitDiffNumstat parses git diff --numstat output into per-file net line changes.
// Each line is: <added>\t<deleted>\t<path>
// Net change = added - deleted.
func parseGitDiffNumstat(output string) map[string]int {
	netChanges := make(map[string]int)
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		// Skip binary files (marked with "-")
		if parts[0] == "-" {
			continue
		}
		added, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		deleted, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		path := parts[2]
		netChanges[path] = added - deleted
	}
	return netChanges
}

// defaultKnowledgeDecaySource scans .kb/models/ for models without recent probes.
type defaultKnowledgeDecaySource struct{}

func (s *defaultKnowledgeDecaySource) ListDecayedModels(maxAge time.Duration) ([]DecayedModel, error) {
	projectDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	modelsDir := filepath.Join(projectDir, ".kb", "models")
	modelEntries, err := os.ReadDir(modelsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	now := time.Now()
	var result []DecayedModel
	for _, modelEntry := range modelEntries {
		if !modelEntry.IsDir() {
			continue
		}

		modelSlug := modelEntry.Name()
		modelPath := filepath.Join(modelsDir, modelSlug, "model.md")
		if _, err := os.Stat(modelPath); err != nil {
			continue
		}

		probesDir := filepath.Join(modelsDir, modelSlug, "probes")
		probeEntries, err := os.ReadDir(probesDir)
		if err != nil {
			// No probes directory = no recent probes
			result = append(result, DecayedModel{
				Slug:           modelSlug,
				DaysSinceProbe: 999, // sentinel for "never probed"
			})
			continue
		}

		var latestProbeDate time.Time
		for _, pe := range probeEntries {
			if pe.IsDir() || !strings.HasSuffix(pe.Name(), ".md") {
				continue
			}
			if len(pe.Name()) >= 10 {
				if d, err := time.Parse("2006-01-02", pe.Name()[:10]); err == nil {
					if d.After(latestProbeDate) {
						latestProbeDate = d
					}
				}
			}
		}

		if latestProbeDate.IsZero() {
			result = append(result, DecayedModel{
				Slug:           modelSlug,
				DaysSinceProbe: 999,
			})
			continue
		}

		daysSince := int(now.Sub(latestProbeDate).Hours() / 24)
		if now.Sub(latestProbeDate) > maxAge {
			result = append(result, DecayedModel{
				Slug:           modelSlug,
				LastProbeDate:  latestProbeDate,
				DaysSinceProbe: daysSince,
			})
		}
	}
	return result, nil
}

// defaultSkillPerformanceDriftSource uses events.ComputeLearning for skill metrics.
// Compares a recent time window against a previous window to detect real drift.
type defaultSkillPerformanceDriftSource struct {
	// RecentWindow is the duration of the "recent" window (default 30 days).
	RecentWindow time.Duration
	// Now allows injecting time for testing (defaults to time.Now).
	Now func() time.Time
	// EventsPath overrides the default events.jsonl path (for testing).
	EventsPath string
}

const (
	defaultRecentWindow  = 30 * 24 * time.Hour
	minOutcomesPerWindow = 5
)

func (s *defaultSkillPerformanceDriftSource) ListDriftedSkills(currentThreshold, previousMin float64) ([]DriftedSkill, error) {
	eventsPath := s.EventsPath
	if eventsPath == "" {
		var err error
		eventsPath, err = defaultEventsPath()
		if err != nil {
			return nil, err
		}
	}

	now := time.Now()
	if s.Now != nil {
		now = s.Now()
	}

	recentWindow := s.RecentWindow
	if recentWindow <= 0 {
		recentWindow = defaultRecentWindow
	}

	recentStart := now.Add(-recentWindow)
	previousEnd := recentStart

	// Compute recent window (last 30 days)
	recentStore, err := events.ComputeLearningInWindow(eventsPath, recentStart, time.Time{})
	if err != nil {
		return nil, err
	}

	// Compute previous window (everything before the recent window)
	previousStore, err := events.ComputeLearningInWindow(eventsPath, time.Time{}, previousEnd)
	if err != nil {
		return nil, err
	}

	var result []DriftedSkill
	for name, recent := range recentStore.Skills {
		recentOutcomes := recent.TotalCompletions + recent.AbandonedCount
		if recentOutcomes < minOutcomesPerWindow {
			continue
		}
		if recent.SuccessRate >= currentThreshold {
			continue
		}

		// Only report drift if we have a measured previous rate to compare against
		prev, hasPrevious := previousStore.Skills[name]
		if !hasPrevious {
			continue
		}
		prevOutcomes := prev.TotalCompletions + prev.AbandonedCount
		if prevOutcomes < minOutcomesPerWindow {
			continue
		}
		if prev.SuccessRate < previousMin {
			continue // previous rate was already low — not a drift
		}

		result = append(result, DriftedSkill{
			Name:         name,
			CurrentRate:  recent.SuccessRate,
			PreviousRate: prev.SuccessRate,
			RecentSpawns: recent.SpawnCount,
		})
	}
	return result, nil
}

// defaultEventsPath returns the path to events.jsonl.
func defaultEventsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".orch", "events.jsonl"), nil
}
