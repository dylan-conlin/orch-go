package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
)

// --- Test mock types implementing daemon interfaces for external tests ---

type testReflector struct {
	reflectFunc     func(createIssues bool) (*daemon.ReflectResult, error)
	reflectOpenFunc func() error
}

func (r *testReflector) Reflect(createIssues bool) (*daemon.ReflectResult, error) {
	if r.reflectFunc != nil {
		return r.reflectFunc(createIssues)
	}
	return &daemon.ReflectResult{}, nil
}

func (r *testReflector) ReflectOpen() error {
	if r.reflectOpenFunc != nil {
		return r.reflectOpenFunc()
	}
	return nil
}

type testSessionCleaner struct {
	cleanupFunc func(config daemon.Config) (int, string, error)
}

func (c *testSessionCleaner) Cleanup(config daemon.Config) (int, string, error) {
	if c.cleanupFunc != nil {
		return c.cleanupFunc(config)
	}
	return 0, "", nil
}

type testKnowledgeHealthService struct {
	checkFunc       func() (*daemon.KnowledgeHealthResult, error)
	createIssueFunc func(result *daemon.KnowledgeHealthResult) error
}

func (s *testKnowledgeHealthService) Check() (*daemon.KnowledgeHealthResult, error) {
	if s.checkFunc != nil {
		return s.checkFunc()
	}
	return &daemon.KnowledgeHealthResult{}, nil
}

func (s *testKnowledgeHealthService) CreateIssue(result *daemon.KnowledgeHealthResult) error {
	if s.createIssueFunc != nil {
		return s.createIssueFunc(result)
	}
	return nil
}

type testAgentDiscoverer struct {
	getActiveAgentsFunc    func() ([]daemon.ActiveAgent, error)
	hasExistingSessionFunc func(beadsID string) bool
}

func (d *testAgentDiscoverer) GetActiveAgents() ([]daemon.ActiveAgent, error) {
	if d.getActiveAgentsFunc != nil {
		return d.getActiveAgentsFunc()
	}
	return nil, nil
}

func (d *testAgentDiscoverer) HasExistingSession(beadsID string) bool {
	if d.hasExistingSessionFunc != nil {
		return d.hasExistingSessionFunc(beadsID)
	}
	return false
}

func (d *testAgentDiscoverer) HasExistingSessionOrError(beadsID string) (bool, error) {
	return d.HasExistingSession(beadsID), nil
}

// disableAllPeriodicTasks returns a config with all periodic tasks disabled.
// Tests that exercise one specific task enable just that task.
func disableAllPeriodicTasks() daemon.Config {
	config := daemon.DefaultConfig()
	config.ReflectEnabled = false
	config.ReflectModelDriftEnabled = false
	config.KnowledgeHealthEnabled = false
	config.CleanupEnabled = false
	config.RecoveryEnabled = false
	config.OrphanDetectionEnabled = false
	config.PhaseTimeoutEnabled = false
	config.AgreementCheckEnabled = false
	config.BeadsHealthEnabled = false
	config.FrictionAccumulationEnabled = false
	config.ArtifactSyncEnabled = false
	config.RegistryRefreshEnabled = false
	config.SynthesisAutoCreateEnabled = false
	config.LearningRefreshEnabled = false
	config.PlanStalenessEnabled = false
	config.ProactiveExtractionEnabled = false
	config.AccretionResponseEnabled = false
	config.TriggerScanEnabled = false
	config.TriggerExpiryEnabled = false
	config.DigestEnabled = false
	config.InvestigationOrphanEnabled = false
	config.VerificationFailedEscalationEnabled = false
	config.LightweightCleanupEnabled = false
	return config
}

func TestRunPeriodicTasks_NothingDue(t *testing.T) {
	// All periodic tasks disabled — should return empty result, no events
	config := disableAllPeriodicTasks()

	d := daemon.NewWithConfig(config)
	tmpDir := t.TempDir()
	logger := events.NewLogger(filepath.Join(tmpDir, "events.jsonl"))

	result := runPeriodicTasks(d, "12:00:00", false, logger)

	if result.KnowledgeHealthSnapshot != nil {
		t.Error("expected nil KnowledgeHealthSnapshot when task is disabled")
	}
}

func TestRunPeriodicTasks_ReflectionError(t *testing.T) {
	config := disableAllPeriodicTasks()
	config.ReflectEnabled = true
	config.ReflectInterval = 1 * time.Millisecond

	d := daemon.NewWithConfig(config)
	d.Reflector = &testReflector{
		reflectFunc: func(createIssues bool) (*daemon.ReflectResult, error) {
			return nil, fmt.Errorf("reflect failed")
		},
	}

	tmpDir := t.TempDir()
	logger := events.NewLogger(filepath.Join(tmpDir, "events.jsonl"))

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	runPeriodicTasks(d, "12:00:00", false, logger)

	w.Close()
	os.Stderr = oldStderr
	var buf [4096]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	if !strings.Contains(output, "Reflection error") {
		t.Errorf("expected stderr to contain 'Reflection error', got: %s", output)
	}
}

func TestRunPeriodicTasks_CleanupLogsEvent(t *testing.T) {
	config := disableAllPeriodicTasks()
	config.CleanupEnabled = true
	config.CleanupInterval = 1 * time.Millisecond

	d := daemon.NewWithConfig(config)
	d.Cleaner = &testSessionCleaner{
		cleanupFunc: func(cfg daemon.Config) (int, string, error) {
			return 3, "Deleted 3 stale sessions", nil
		},
	}

	tmpDir := t.TempDir()
	eventsPath := filepath.Join(tmpDir, "events.jsonl")
	logger := events.NewLogger(eventsPath)

	runPeriodicTasks(d, "12:00:00", false, logger)

	// Verify event was logged
	data, err := os.ReadFile(events.RotatedLogPath(eventsPath))
	if err != nil {
		t.Fatalf("failed to read events file: %v", err)
	}

	var event events.Event
	if err := json.Unmarshal(data[:len(data)-1], &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if event.Type != "daemon.cleanup" {
		t.Errorf("expected event type daemon.cleanup, got %s", event.Type)
	}

	deleted, ok := event.Data["deleted"].(float64)
	if !ok || int(deleted) != 3 {
		t.Errorf("expected deleted=3, got %v", event.Data["deleted"])
	}
}

func TestRunPeriodicTasks_KnowledgeHealthSnapshot(t *testing.T) {
	config := disableAllPeriodicTasks()
	config.KnowledgeHealthEnabled = true
	config.KnowledgeHealthInterval = 1 * time.Millisecond
	config.KnowledgeHealthThreshold = 50

	d := daemon.NewWithConfig(config)
	d.KnowledgeHealth = &testKnowledgeHealthService{
		checkFunc: func() (*daemon.KnowledgeHealthResult, error) {
			return &daemon.KnowledgeHealthResult{
				TotalActive: 25,
				ByType:      map[string]int{"decision": 10, "constraint": 15},
				Message:     "Knowledge health: 25 active entries",
			}, nil
		},
	}

	tmpDir := t.TempDir()
	logger := events.NewLogger(filepath.Join(tmpDir, "events.jsonl"))

	result := runPeriodicTasks(d, "12:00:00", false, logger)

	if result.KnowledgeHealthSnapshot == nil {
		t.Fatal("expected KnowledgeHealthSnapshot to be set")
	}

	if result.KnowledgeHealthSnapshot.TotalActive != 25 {
		t.Errorf("expected TotalActive=25, got %d", result.KnowledgeHealthSnapshot.TotalActive)
	}
}

func TestRunPeriodicTasks_RecoveryErrorLogsEvent(t *testing.T) {
	// Recovery calls GetActiveAgents() directly (not mockable via setter).
	// In test environment without beads, it returns an error which should
	// produce an error event in the log.
	config := disableAllPeriodicTasks()
	config.RecoveryEnabled = true
	config.RecoveryInterval = 1 * time.Millisecond
	config.RecoveryIdleThreshold = 1 * time.Hour
	config.RecoveryRateLimit = 1 * time.Hour

	d := daemon.NewWithConfig(config)

	tmpDir := t.TempDir()
	eventsPath := filepath.Join(tmpDir, "events.jsonl")
	logger := events.NewLogger(eventsPath)

	// Capture stderr to suppress noise
	oldStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	runPeriodicTasks(d, "12:00:00", false, logger)

	w.Close()
	os.Stderr = oldStderr

	// Recovery errored — should log an error event
	data, _ := os.ReadFile(events.RotatedLogPath(eventsPath))
	if len(data) == 0 {
		t.Error("expected error event when recovery fails, got nothing")
		return
	}

	var event events.Event
	if err := json.Unmarshal(data[:len(data)-1], &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if event.Type != "daemon.recovery" {
		t.Errorf("expected event type daemon.recovery, got %s", event.Type)
	}

	if _, hasError := event.Data["error"]; !hasError {
		t.Error("expected event to have error field")
	}
}

func TestFilterNewStalePlans_FirstNotification(t *testing.T) {
	planStalenessNotifiedMu.Lock()
	planStalenessNotified = make(map[string]time.Time)
	planStalenessNotifiedMu.Unlock()

	stalePlans := []daemon.StalePlan{
		{Slug: "plan-a", Title: "Plan A", Reason: "stale", StalenessType: daemon.StalenessNoProgress},
		{Slug: "plan-b", Title: "Plan B", Reason: "stale", StalenessType: daemon.StalenessUnhydrated},
		{Slug: "plan-c", Title: "Plan C", Reason: "stale", StalenessType: daemon.StalenessAdvancementStall},
	}

	result := filterNewStalePlans(stalePlans)
	if len(result) != 3 {
		t.Fatalf("expected 3 new stale plans, got %d: %v", len(result), result)
	}
}

func TestFilterNewStalePlans_CooldownSuppresses(t *testing.T) {
	planStalenessNotifiedMu.Lock()
	planStalenessNotified = map[string]time.Time{
		"plan-a": time.Now().Add(-1 * time.Hour),
		"plan-b": time.Now().Add(-1 * time.Hour),
	}
	planStalenessNotifiedMu.Unlock()

	stalePlans := []daemon.StalePlan{
		{Slug: "plan-a", Title: "Plan A", Reason: "stale"},
		{Slug: "plan-b", Title: "Plan B", Reason: "stale"},
		{Slug: "plan-c", Title: "Plan C", Reason: "stale"},
	}

	result := filterNewStalePlans(stalePlans)
	if len(result) != 1 {
		t.Fatalf("expected 1 new stale plan (plan-c), got %d: %v", len(result), result)
	}
	if result[0] != "plan-c" {
		t.Fatalf("expected plan-c, got %s", result[0])
	}
}

func TestFilterNewStalePlans_ExpiredCooldownRenotifies(t *testing.T) {
	planStalenessNotifiedMu.Lock()
	planStalenessNotified = map[string]time.Time{
		"plan-a": time.Now().Add(-25 * time.Hour),
	}
	planStalenessNotifiedMu.Unlock()

	stalePlans := []daemon.StalePlan{
		{Slug: "plan-a", Title: "Plan A", Reason: "stale"},
	}

	result := filterNewStalePlans(stalePlans)
	if len(result) != 1 {
		t.Fatalf("expected 1 plan (cooldown expired), got %d", len(result))
	}
}

func TestFilterNewStalePlans_DedupsWithinResult(t *testing.T) {
	planStalenessNotifiedMu.Lock()
	planStalenessNotified = make(map[string]time.Time)
	planStalenessNotifiedMu.Unlock()

	stalePlans := []daemon.StalePlan{
		{Slug: "plan-a", Title: "Plan A", Reason: "reason1", StalenessType: daemon.StalenessNoProgress},
		{Slug: "plan-a", Title: "Plan A", Reason: "reason2", StalenessType: daemon.StalenessAdvancementStall},
	}

	result := filterNewStalePlans(stalePlans)
	if len(result) != 1 {
		t.Fatalf("expected 1 deduplicated slug, got %d: %v", len(result), result)
	}
}

func TestFilterNewStalePlans_AllSuppressed(t *testing.T) {
	planStalenessNotifiedMu.Lock()
	planStalenessNotified = map[string]time.Time{
		"plan-a": time.Now(),
		"plan-b": time.Now(),
	}
	planStalenessNotifiedMu.Unlock()

	stalePlans := []daemon.StalePlan{
		{Slug: "plan-a", Title: "Plan A", Reason: "stale"},
		{Slug: "plan-b", Title: "Plan B", Reason: "stale"},
	}

	result := filterNewStalePlans(stalePlans)
	if len(result) != 0 {
		t.Fatalf("expected 0 new stale plans (all suppressed), got %d: %v", len(result), result)
	}
}

func TestMarkPlansNotified(t *testing.T) {
	planStalenessNotifiedMu.Lock()
	planStalenessNotified = make(map[string]time.Time)
	planStalenessNotifiedMu.Unlock()

	before := time.Now()
	markPlansNotified([]string{"plan-x", "plan-y"})
	after := time.Now()

	planStalenessNotifiedMu.Lock()
	defer planStalenessNotifiedMu.Unlock()

	for _, slug := range []string{"plan-x", "plan-y"} {
		ts, ok := planStalenessNotified[slug]
		if !ok {
			t.Fatalf("expected %s to be marked as notified", slug)
		}
		if ts.Before(before) || ts.After(after) {
			t.Fatalf("expected timestamp for %s between %v and %v, got %v", slug, before, after, ts)
		}
	}
}

func TestRunPeriodicTasks_OrphanDetectionLogsEvent(t *testing.T) {
	config := disableAllPeriodicTasks()
	config.OrphanDetectionEnabled = true
	config.OrphanDetectionInterval = 1 * time.Millisecond
	config.OrphanAgeThreshold = 1 * time.Hour

	d := daemon.NewWithConfig(config)
	// Mock GetActiveAgents to return no agents
	d.Agents = &testAgentDiscoverer{
		getActiveAgentsFunc: func() ([]daemon.ActiveAgent, error) {
			return nil, nil
		},
	}

	tmpDir := t.TempDir()
	eventsPath := filepath.Join(tmpDir, "events.jsonl")
	logger := events.NewLogger(eventsPath)

	runPeriodicTasks(d, "12:00:00", false, logger)

	// Orphan detection ran but found nothing — no event should be logged
	data, _ := os.ReadFile(events.RotatedLogPath(eventsPath))
	if len(data) > 0 {
		t.Errorf("expected no events when orphan detection finds nothing, got: %s", string(data))
	}
}
