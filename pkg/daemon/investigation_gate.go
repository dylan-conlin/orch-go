package daemon

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// InvestigationCircuitBreakerThreshold is the maximum investigations allowed
	// in any rolling window before feature work is blocked.
	InvestigationCircuitBreakerThreshold = 50

	// InvestigationCircuitBreakerWindowDays is the rolling window size used
	// for investigation volume checks.
	InvestigationCircuitBreakerWindowDays = 30
)

type investigationGateState struct {
	active      bool
	maxInWindow int
}

func applyInvestigationCircuitBreaker(issues []Issue, projectPath string, allowFeatureWork bool, now time.Time) ([]Issue, error) {
	if allowFeatureWork {
		return issues, nil
	}

	state, err := evaluateInvestigationCircuitBreaker(projectPath, now)
	if err != nil {
		return issues, err
	}

	if !state.active {
		return issues, nil
	}

	filtered := make([]Issue, 0, len(issues))
	for _, issue := range issues {
		if issue.IssueType == "feature" {
			continue
		}
		filtered = append(filtered, issue)
	}

	return filtered, nil
}

func evaluateInvestigationCircuitBreaker(projectPath string, now time.Time) (investigationGateState, error) {
	state := investigationGateState{}

	if projectPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return state, err
		}
		projectPath = cwd
	}

	investigationsDir := filepath.Join(projectPath, ".kb", "investigations")
	dates, err := collectInvestigationDates(investigationsDir)
	if err != nil {
		return state, err
	}

	maxInWindow := maxInvestigationsInWindow(dates, InvestigationCircuitBreakerWindowDays)
	state.maxInWindow = maxInWindow
	state.active = maxInWindow > InvestigationCircuitBreakerThreshold

	_ = now // Reserved for future time-relative gating refinements.

	return state, nil
}

func collectInvestigationDates(investigationsDir string) ([]time.Time, error) {
	if _, err := os.Stat(investigationsDir); os.IsNotExist(err) {
		return nil, nil
	}

	var dates []time.Time
	err := filepath.WalkDir(investigationsDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			if d.Name() == "archived" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		date, ok := extractInvestigationDate(d.Name())
		if !ok {
			return nil
		}

		dates = append(dates, date)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return dates, nil
}

func extractInvestigationDate(filename string) (time.Time, bool) {
	if len(filename) < len("2006-01-02") {
		return time.Time{}, false
	}

	prefix := filename[:len("2006-01-02")]
	parsed, err := time.Parse("2006-01-02", prefix)
	if err != nil {
		return time.Time{}, false
	}

	return parsed, true
}

func maxInvestigationsInWindow(dates []time.Time, windowDays int) int {
	if len(dates) == 0 {
		return 0
	}
	if windowDays <= 0 {
		windowDays = 1
	}

	sorted := append([]time.Time(nil), dates...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Before(sorted[j])
	})

	maxCount := 0
	left := 0
	windowDuration := time.Duration(windowDays-1) * 24 * time.Hour

	for right := range sorted {
		for sorted[right].Sub(sorted[left]) > windowDuration {
			left++
		}

		count := right - left + 1
		if count > maxCount {
			maxCount = count
		}
	}

	return maxCount
}
