package daemon

import (
	"fmt"
	"testing"
	"time"
)

// mockTriggerExpiryService implements TriggerExpiryService for tests.
type mockTriggerExpiryService struct {
	ListExpiredFunc func(maxAge time.Duration) ([]ExpiredTriggerIssue, error)
	ExpireIssueFunc func(id, reason string) error
}

func (m *mockTriggerExpiryService) ListExpiredTriggerIssues(maxAge time.Duration) ([]ExpiredTriggerIssue, error) {
	if m.ListExpiredFunc != nil {
		return m.ListExpiredFunc(maxAge)
	}
	return nil, nil
}

func (m *mockTriggerExpiryService) ExpireTriggerIssue(id, reason string) error {
	if m.ExpireIssueFunc != nil {
		return m.ExpireIssueFunc(id, reason)
	}
	return nil
}

func TestDaemon_RunPeriodicTriggerExpiry_NotDue(t *testing.T) {
	cfg := Config{
		TriggerExpiryEnabled:  true,
		TriggerExpiryInterval: 24 * time.Hour,
		TriggerExpiryMaxAge:   14 * 24 * time.Hour,
	}
	d := &Daemon{
		Config:        cfg,
		Scheduler:     NewSchedulerFromConfig(cfg),
		TriggerExpiry: &mockTriggerExpiryService{},
	}
	d.Scheduler.SetLastRun(TaskTriggerExpiry, time.Now())

	result := d.RunPeriodicTriggerExpiry()
	if result != nil {
		t.Error("RunPeriodicTriggerExpiry() should return nil when not due")
	}
}

func TestDaemon_RunPeriodicTriggerExpiry_ServiceNotConfigured(t *testing.T) {
	cfg := Config{
		TriggerExpiryEnabled:  true,
		TriggerExpiryInterval: 24 * time.Hour,
		TriggerExpiryMaxAge:   14 * 24 * time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
	}

	result := d.RunPeriodicTriggerExpiry()
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Error == nil {
		t.Error("expected error for unconfigured service")
	}
}

func TestDaemon_RunPeriodicTriggerExpiry_NoExpiredIssues(t *testing.T) {
	cfg := Config{
		TriggerExpiryEnabled:  true,
		TriggerExpiryInterval: 24 * time.Hour,
		TriggerExpiryMaxAge:   14 * 24 * time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		TriggerExpiry: &mockTriggerExpiryService{
			ListExpiredFunc: func(maxAge time.Duration) ([]ExpiredTriggerIssue, error) {
				return nil, nil
			},
		},
	}

	result := d.RunPeriodicTriggerExpiry()
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Error != nil {
		t.Errorf("unexpected error: %v", result.Error)
	}
	if result.Expired != 0 {
		t.Errorf("Expired = %d, want 0", result.Expired)
	}
	if result.Message != "Trigger expiry: no expired issues found" {
		t.Errorf("Message = %q", result.Message)
	}
}

func TestDaemon_RunPeriodicTriggerExpiry_ExpiresStaleIssues(t *testing.T) {
	expiredIDs := []string{}
	cfg := Config{
		TriggerExpiryEnabled:  true,
		TriggerExpiryInterval: 24 * time.Hour,
		TriggerExpiryMaxAge:   14 * 24 * time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		TriggerExpiry: &mockTriggerExpiryService{
			ListExpiredFunc: func(maxAge time.Duration) ([]ExpiredTriggerIssue, error) {
				if maxAge != 14*24*time.Hour {
					t.Errorf("maxAge = %v, want 14 days", maxAge)
				}
				return []ExpiredTriggerIssue{
					{ID: "orch-go-001", Title: "Old trigger 1", Age: 15 * 24 * time.Hour},
					{ID: "orch-go-002", Title: "Old trigger 2", Age: 20 * 24 * time.Hour},
				}, nil
			},
			ExpireIssueFunc: func(id, reason string) error {
				expiredIDs = append(expiredIDs, id)
				return nil
			},
		},
	}

	result := d.RunPeriodicTriggerExpiry()
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Error != nil {
		t.Errorf("unexpected error: %v", result.Error)
	}
	if result.Expired != 2 {
		t.Errorf("Expired = %d, want 2", result.Expired)
	}
	if len(expiredIDs) != 2 {
		t.Errorf("ExpireIssue called %d times, want 2", len(expiredIDs))
	}
	if len(result.ExpiredIssues) != 2 {
		t.Errorf("ExpiredIssues = %v, want 2 items", result.ExpiredIssues)
	}
}

func TestDaemon_RunPeriodicTriggerExpiry_ContinuesOnExpireError(t *testing.T) {
	expireCount := 0
	cfg := Config{
		TriggerExpiryEnabled:  true,
		TriggerExpiryInterval: 24 * time.Hour,
		TriggerExpiryMaxAge:   14 * 24 * time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		TriggerExpiry: &mockTriggerExpiryService{
			ListExpiredFunc: func(maxAge time.Duration) ([]ExpiredTriggerIssue, error) {
				return []ExpiredTriggerIssue{
					{ID: "orch-go-001", Title: "Fail", Age: 15 * 24 * time.Hour},
					{ID: "orch-go-002", Title: "Succeed", Age: 16 * 24 * time.Hour},
				}, nil
			},
			ExpireIssueFunc: func(id, reason string) error {
				expireCount++
				if id == "orch-go-001" {
					return fmt.Errorf("bd close failed")
				}
				return nil
			},
		},
	}

	result := d.RunPeriodicTriggerExpiry()
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Expired != 1 {
		t.Errorf("Expired = %d, want 1 (one succeeded)", result.Expired)
	}
	if result.Errors != 1 {
		t.Errorf("Errors = %d, want 1", result.Errors)
	}
	if expireCount != 2 {
		t.Errorf("ExpireIssue called %d times, want 2 (should try both)", expireCount)
	}
}

func TestDaemon_RunPeriodicTriggerExpiry_ListError(t *testing.T) {
	cfg := Config{
		TriggerExpiryEnabled:  true,
		TriggerExpiryInterval: 24 * time.Hour,
		TriggerExpiryMaxAge:   14 * 24 * time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		TriggerExpiry: &mockTriggerExpiryService{
			ListExpiredFunc: func(maxAge time.Duration) ([]ExpiredTriggerIssue, error) {
				return nil, fmt.Errorf("beads unavailable")
			},
		},
	}

	result := d.RunPeriodicTriggerExpiry()
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Error == nil {
		t.Error("expected error")
	}
}

func TestDaemon_RunPeriodicTriggerExpiry_UpdatesScheduler(t *testing.T) {
	cfg := Config{
		TriggerExpiryEnabled:  true,
		TriggerExpiryInterval: 24 * time.Hour,
		TriggerExpiryMaxAge:   14 * 24 * time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		TriggerExpiry: &mockTriggerExpiryService{
			ListExpiredFunc: func(maxAge time.Duration) ([]ExpiredTriggerIssue, error) {
				return nil, nil
			},
		},
	}

	before := d.Scheduler.LastRunTime(TaskTriggerExpiry)
	if !before.IsZero() {
		t.Fatal("expected zero LastRunTime before first run")
	}

	d.RunPeriodicTriggerExpiry()

	after := d.Scheduler.LastRunTime(TaskTriggerExpiry)
	if after.IsZero() {
		t.Error("expected non-zero LastRunTime after run")
	}
}

func TestExpiredTriggerIssue_DetectorName(t *testing.T) {
	tests := []struct {
		name   string
		labels []string
		want   string
	}{
		{"extracts detector from labels", []string{"daemon:trigger", "daemon:trigger:hotspot_acceleration", "triage:ready"}, "hotspot_acceleration"},
		{"returns unknown when no detector label", []string{"daemon:trigger", "triage:ready"}, "unknown"},
		{"returns unknown when labels empty", nil, "unknown"},
		{"ignores base trigger label", []string{"daemon:trigger"}, "unknown"},
		{"picks first detector label", []string{"daemon:trigger:model_contradictions", "daemon:trigger:knowledge_decay"}, "model_contradictions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := ExpiredTriggerIssue{Labels: tt.labels}
			got := issue.DetectorName()
			if got != tt.want {
				t.Errorf("DetectorName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDaemon_RunPeriodicTriggerExpiry_TracksDetectorOutcomes(t *testing.T) {
	cfg := Config{
		TriggerExpiryEnabled:  true,
		TriggerExpiryInterval: 24 * time.Hour,
		TriggerExpiryMaxAge:   14 * 24 * time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		TriggerExpiry: &mockTriggerExpiryService{
			ListExpiredFunc: func(maxAge time.Duration) ([]ExpiredTriggerIssue, error) {
				return []ExpiredTriggerIssue{
					{ID: "orch-go-001", Title: "Hotspot: file.go", Age: 15 * 24 * time.Hour, Labels: []string{"daemon:trigger", "daemon:trigger:hotspot_acceleration"}},
					{ID: "orch-go-002", Title: "Hotspot: other.go", Age: 16 * 24 * time.Hour, Labels: []string{"daemon:trigger", "daemon:trigger:hotspot_acceleration"}},
					{ID: "orch-go-003", Title: "Decay: some-model", Age: 20 * 24 * time.Hour, Labels: []string{"daemon:trigger", "daemon:trigger:knowledge_decay"}},
				}, nil
			},
			ExpireIssueFunc: func(id, reason string) error { return nil },
		},
	}

	result := d.RunPeriodicTriggerExpiry()
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Expired != 3 {
		t.Errorf("Expired = %d, want 3", result.Expired)
	}
	if result.DetectorOutcomes == nil {
		t.Fatal("DetectorOutcomes is nil")
	}
	if got := result.DetectorOutcomes["hotspot_acceleration"]; got != 2 {
		t.Errorf("DetectorOutcomes[hotspot_acceleration] = %d, want 2", got)
	}
	if got := result.DetectorOutcomes["knowledge_decay"]; got != 1 {
		t.Errorf("DetectorOutcomes[knowledge_decay] = %d, want 1", got)
	}
}

func TestDaemon_RunPeriodicTriggerExpiry_DetectorOutcomesExcludeErrors(t *testing.T) {
	cfg := Config{
		TriggerExpiryEnabled:  true,
		TriggerExpiryInterval: 24 * time.Hour,
		TriggerExpiryMaxAge:   14 * 24 * time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		TriggerExpiry: &mockTriggerExpiryService{
			ListExpiredFunc: func(maxAge time.Duration) ([]ExpiredTriggerIssue, error) {
				return []ExpiredTriggerIssue{
					{ID: "orch-go-001", Title: "Fails", Age: 15 * 24 * time.Hour, Labels: []string{"daemon:trigger", "daemon:trigger:hotspot_acceleration"}},
					{ID: "orch-go-002", Title: "Succeeds", Age: 16 * 24 * time.Hour, Labels: []string{"daemon:trigger", "daemon:trigger:hotspot_acceleration"}},
				}, nil
			},
			ExpireIssueFunc: func(id, reason string) error {
				if id == "orch-go-001" {
					return fmt.Errorf("failed")
				}
				return nil
			},
		},
	}

	result := d.RunPeriodicTriggerExpiry()
	if result == nil {
		t.Fatal("expected result")
	}
	// Only the successfully expired issue should be tracked
	if got := result.DetectorOutcomes["hotspot_acceleration"]; got != 1 {
		t.Errorf("DetectorOutcomes[hotspot_acceleration] = %d, want 1 (failed expiry excluded)", got)
	}
}

func TestDefaultConfig_IncludesTriggerExpiry(t *testing.T) {
	config := DefaultConfig()

	if !config.TriggerExpiryEnabled {
		t.Error("DefaultConfig().TriggerExpiryEnabled should be true")
	}
	if config.TriggerExpiryInterval != 24*time.Hour {
		t.Errorf("DefaultConfig().TriggerExpiryInterval = %v, want 24h", config.TriggerExpiryInterval)
	}
	if config.TriggerExpiryMaxAge != 14*24*time.Hour {
		t.Errorf("DefaultConfig().TriggerExpiryMaxAge = %v, want 14d", config.TriggerExpiryMaxAge)
	}
}
