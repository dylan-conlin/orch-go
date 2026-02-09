package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

const resourceCeilingMultiplier int64 = 2

// Capture baseline from warm steady-state, not cold start.
const resourceBaselineWarmupDuration = 10 * time.Minute

type resourceMetrics struct {
	Goroutines          int64 `json:"goroutines"`
	HeapBytes           int64 `json:"heap_bytes"`
	ChildProcesses      int64 `json:"child_processes"`
	OpenFileDescriptors int64 `json:"open_file_descriptors"`
}

type resourceBreach struct {
	Metric    string  `json:"metric"`
	Baseline  int64   `json:"baseline"`
	Current   int64   `json:"current"`
	Threshold int64   `json:"threshold"`
	Ratio     float64 `json:"ratio"`
}

type resourceHealthReport struct {
	Baseline          resourceMetrics   `json:"baseline"`
	Current           resourceMetrics   `json:"current"`
	CeilingMultiplier int64             `json:"ceiling_multiplier"`
	Breached          bool              `json:"breached"`
	Breaches          []resourceBreach  `json:"breaches,omitempty"`
	BaselineErrors    map[string]string `json:"baseline_errors,omitempty"`
	CurrentErrors     map[string]string `json:"current_errors,omitempty"`
}

type resourceSample struct {
	metrics resourceMetrics
	errors  map[string]string
}

type resourceMonitor struct {
	logger         *events.Logger
	sampleFn       func() resourceSample
	nowFn          func() time.Time
	startedAt      time.Time
	warmupDuration time.Duration
	calibrated     bool
	baseline       resourceMetrics
	baselineErrors map[string]string
	activeBreaches map[string]bool
	mu             sync.Mutex
}

func newResourceMonitor(logger *events.Logger) *resourceMonitor {
	return newResourceMonitorWithConfig(logger, collectResourceSample, time.Now, resourceBaselineWarmupDuration)
}

func newResourceMonitorWithSampler(logger *events.Logger, sampleFn func() resourceSample) *resourceMonitor {
	return newResourceMonitorWithConfig(logger, sampleFn, time.Now, 0)
}

func newResourceMonitorWithConfig(logger *events.Logger, sampleFn func() resourceSample, nowFn func() time.Time, warmupDuration time.Duration) *resourceMonitor {
	if sampleFn == nil {
		sampleFn = collectResourceSample
	}
	if nowFn == nil {
		nowFn = time.Now
	}
	baseline := sampleFn()
	if warmupDuration < 0 {
		warmupDuration = 0
	}
	return &resourceMonitor{
		logger:         logger,
		sampleFn:       sampleFn,
		nowFn:          nowFn,
		startedAt:      nowFn(),
		warmupDuration: warmupDuration,
		calibrated:     warmupDuration == 0,
		baseline:       baseline.metrics,
		baselineErrors: copyStringMap(baseline.errors),
		activeBreaches: make(map[string]bool),
	}
}

func (m *resourceMonitor) sampleAndCheck() resourceHealthReport {
	current := m.sampleFn()

	m.mu.Lock()
	if !m.calibrated {
		m.recalibrateBaselineDuringWarmupLocked(current)
	}

	baseline := m.baseline
	baselineErrors := copyStringMap(m.baselineErrors)

	breaches := make([]resourceBreach, 0)
	if m.calibrated {
		breaches = detectResourceBreaches(baseline, current.metrics)
	}

	breachByMetric := make(map[string]resourceBreach, len(breaches))
	for _, breach := range breaches {
		breachByMetric[breach.Metric] = breach
	}

	var newBreaches []resourceBreach
	for _, breach := range breaches {
		if !m.activeBreaches[breach.Metric] {
			newBreaches = append(newBreaches, breach)
		}
		m.activeBreaches[breach.Metric] = true
	}
	for metric := range m.activeBreaches {
		if _, stillBreached := breachByMetric[metric]; !stillBreached {
			delete(m.activeBreaches, metric)
		}
	}
	m.mu.Unlock()

	for _, breach := range newBreaches {
		m.logResourceCeilingBreach(breach, current.metrics, current.errors)
	}

	report := resourceHealthReport{
		Baseline:          baseline,
		Current:           current.metrics,
		CeilingMultiplier: resourceCeilingMultiplier,
		Breached:          len(breaches) > 0,
		Breaches:          breaches,
		BaselineErrors:    baselineErrors,
		CurrentErrors:     copyStringMap(current.errors),
	}
	if len(report.BaselineErrors) == 0 {
		report.BaselineErrors = nil
	}
	if len(report.CurrentErrors) == 0 {
		report.CurrentErrors = nil
	}

	return report
}

func (m *resourceMonitor) recalibrateBaselineDuringWarmupLocked(current resourceSample) {
	m.baseline = maxResourceMetrics(m.baseline, current.metrics)
	m.baselineErrors = copyStringMap(current.errors)

	if m.nowFn().Sub(m.startedAt) < m.warmupDuration {
		return
	}

	m.calibrated = true
	m.activeBreaches = make(map[string]bool)
}

func (m *resourceMonitor) logResourceCeilingBreach(breach resourceBreach, current resourceMetrics, currentErrors map[string]string) {
	eventData := map[string]interface{}{
		"metric":             breach.Metric,
		"baseline":           breach.Baseline,
		"current":            breach.Current,
		"threshold":          breach.Threshold,
		"ratio":              breach.Ratio,
		"baseline_metrics":   m.baseline.toMap(),
		"current_metrics":    current.toMap(),
		"ceiling_multiplier": resourceCeilingMultiplier,
	}
	if len(currentErrors) > 0 {
		eventData["sampling_errors"] = copyStringMap(currentErrors)
	}

	structured := map[string]interface{}{
		"event":     events.EventTypeResourceCeilingBreach,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"data":      eventData,
	}
	if payload, err := json.Marshal(structured); err == nil {
		fmt.Println(string(payload))
	} else {
		fmt.Fprintf(os.Stderr, "Warning: failed to marshal resource ceiling breach event: %v\n", err)
	}

	if m.logger != nil {
		event := events.Event{
			Type:      events.EventTypeResourceCeilingBreach,
			Timestamp: time.Now().Unix(),
			Data:      eventData,
		}
		if err := m.logger.Log(event); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log resource ceiling breach event: %v\n", err)
		}
	}
}

func (m resourceMetrics) toMap() map[string]interface{} {
	return map[string]interface{}{
		"goroutines":            m.Goroutines,
		"heap_bytes":            m.HeapBytes,
		"child_processes":       m.ChildProcesses,
		"open_file_descriptors": m.OpenFileDescriptors,
	}
}

func detectResourceBreaches(baseline, current resourceMetrics) []resourceBreach {
	checks := []struct {
		metric   string
		baseline int64
		current  int64
	}{
		{metric: "goroutines", baseline: baseline.Goroutines, current: current.Goroutines},
		{metric: "heap_bytes", baseline: baseline.HeapBytes, current: current.HeapBytes},
		{metric: "child_processes", baseline: baseline.ChildProcesses, current: current.ChildProcesses},
		{metric: "open_file_descriptors", baseline: baseline.OpenFileDescriptors, current: current.OpenFileDescriptors},
	}

	breaches := make([]resourceBreach, 0, len(checks))
	for _, check := range checks {
		if check.baseline <= 0 || check.current < 0 {
			continue
		}
		threshold := check.baseline * resourceCeilingMultiplier
		if check.current <= threshold {
			continue
		}
		breaches = append(breaches, resourceBreach{
			Metric:    check.metric,
			Baseline:  check.baseline,
			Current:   check.current,
			Threshold: threshold,
			Ratio:     float64(check.current) / float64(check.baseline),
		})
	}

	return breaches
}

func collectResourceSample() resourceSample {
	metrics := resourceMetrics{
		Goroutines: int64(runtime.NumGoroutine()),
		HeapBytes:  currentHeapBytes(),
	}

	errors := make(map[string]string)

	childCount, err := countChildProcesses()
	if err != nil {
		metrics.ChildProcesses = -1
		errors["child_processes"] = err.Error()
	} else {
		metrics.ChildProcesses = childCount
	}

	fdCount, err := countOpenFileDescriptors()
	if err != nil {
		metrics.OpenFileDescriptors = -1
		errors["open_file_descriptors"] = err.Error()
	} else {
		metrics.OpenFileDescriptors = fdCount
	}

	if len(errors) == 0 {
		errors = nil
	}

	return resourceSample{metrics: metrics, errors: errors}
}

func currentHeapBytes() int64 {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return int64(memStats.HeapAlloc)
}

func countChildProcesses() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ps", "-eo", "pid=,ppid=,comm=")
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return -1, fmt.Errorf("ps query timed out: %w", ctx.Err())
		}
		return -1, fmt.Errorf("ps query failed: %w", err)
	}

	selfPID := os.Getpid()
	var count int64

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
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
		if ppid != selfPID {
			continue
		}

		command := fields[2]
		if command == "ps" || strings.HasSuffix(command, "/ps") {
			continue
		}
		if pid == selfPID {
			continue
		}

		count++
	}

	if err := scanner.Err(); err != nil {
		return -1, fmt.Errorf("failed parsing ps output: %w", err)
	}

	return count, nil
}

func countOpenFileDescriptors() (int64, error) {
	paths := []string{"/proc/self/fd", "/dev/fd"}
	var lastErr error

	for _, path := range paths {
		count, err := countNumericDirectoryEntries(path)
		if err == nil {
			return count, nil
		}
		lastErr = err
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no descriptor directory available")
	}

	return -1, lastErr
}

func countNumericDirectoryEntries(path string) (int64, error) {
	dir, err := os.Open(path)
	if err != nil {
		return -1, err
	}
	defer dir.Close()

	names, err := dir.Readdirnames(-1)
	if err != nil {
		return -1, err
	}

	var count int64
	for _, name := range names {
		if _, err := strconv.Atoi(name); err == nil {
			count++
		}
	}

	return count, nil
}

func maxResourceMetrics(a, b resourceMetrics) resourceMetrics {
	return resourceMetrics{
		Goroutines:          maxNonNegativeMetric(a.Goroutines, b.Goroutines),
		HeapBytes:           maxNonNegativeMetric(a.HeapBytes, b.HeapBytes),
		ChildProcesses:      maxNonNegativeMetric(a.ChildProcesses, b.ChildProcesses),
		OpenFileDescriptors: maxNonNegativeMetric(a.OpenFileDescriptors, b.OpenFileDescriptors),
	}
}

func maxNonNegativeMetric(a, b int64) int64 {
	if a < 0 {
		return b
	}
	if b < 0 {
		return a
	}
	if b > a {
		return b
	}
	return a
}

func copyStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	clone := make(map[string]string, len(input))
	for key, value := range input {
		clone[key] = value
	}
	return clone
}
