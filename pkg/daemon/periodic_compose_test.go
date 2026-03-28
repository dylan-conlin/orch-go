package daemon

import (
	"errors"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/compose"
	"github.com/dylan-conlin/orch-go/pkg/daemonconfig"
)

type mockComposeService struct {
	briefCount int
	briefErr   error
	digest     *compose.Digest
	composeErr error
	writePath  string
	writeErr   error
}

func (m *mockComposeService) CountUndigestedBriefs() (int, error) {
	return m.briefCount, m.briefErr
}

func (m *mockComposeService) Compose() (*compose.Digest, error) {
	return m.digest, m.composeErr
}

func (m *mockComposeService) WriteDigest(d *compose.Digest) (string, error) {
	return m.writePath, m.writeErr
}

func TestRunPeriodicCompose_NotDue(t *testing.T) {
	d := &Daemon{
		Scheduler: NewPeriodicScheduler(),
	}
	d.Scheduler.Register(TaskCompose, true, 2*time.Hour)
	d.Scheduler.SetLastRun(TaskCompose, time.Now()) // just ran

	result := d.RunPeriodicCompose()
	if result != nil {
		t.Error("expected nil when not due")
	}
}

func TestRunPeriodicCompose_BelowThreshold(t *testing.T) {
	d := NewWithConfig(daemonconfig.Config{
		ComposeEnabled:   true,
		ComposeInterval:  2 * time.Hour,
		ComposeThreshold: 8,
	})
	d.ComposeService = &mockComposeService{briefCount: 5}

	result := d.RunPeriodicCompose()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Error != nil {
		t.Errorf("unexpected error: %v", result.Error)
	}
	if result.Composed {
		t.Error("should not compose when below threshold")
	}
	if result.BriefCount != 5 {
		t.Errorf("BriefCount = %d, want 5", result.BriefCount)
	}
}

func TestRunPeriodicCompose_AboveThreshold(t *testing.T) {
	svc := &mockComposeService{
		briefCount: 12,
		digest: &compose.Digest{
			BriefsComposed: 12,
			ClustersFound:  3,
		},
		writePath: "/tmp/digests/2026-03-28-digest.md",
	}
	d := NewWithConfig(daemonconfig.Config{
		ComposeEnabled:   true,
		ComposeInterval:  2 * time.Hour,
		ComposeThreshold: 8,
	})
	d.ComposeService = svc

	result := d.RunPeriodicCompose()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Error != nil {
		t.Errorf("unexpected error: %v", result.Error)
	}
	if !result.Composed {
		t.Error("should compose when above threshold")
	}
	if result.BriefCount != 12 {
		t.Errorf("BriefCount = %d, want 12", result.BriefCount)
	}
	if result.ClustersFound != 3 {
		t.Errorf("ClustersFound = %d, want 3", result.ClustersFound)
	}
	if result.DigestPath != "/tmp/digests/2026-03-28-digest.md" {
		t.Errorf("DigestPath = %q", result.DigestPath)
	}
}

func TestRunPeriodicCompose_CountError(t *testing.T) {
	d := NewWithConfig(daemonconfig.Config{
		ComposeEnabled:   true,
		ComposeInterval:  2 * time.Hour,
		ComposeThreshold: 8,
	})
	d.ComposeService = &mockComposeService{
		briefErr: errors.New("briefs dir unreadable"),
	}

	result := d.RunPeriodicCompose()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Error == nil {
		t.Error("expected error")
	}
}

func TestRunPeriodicCompose_ComposeError(t *testing.T) {
	d := NewWithConfig(daemonconfig.Config{
		ComposeEnabled:   true,
		ComposeInterval:  2 * time.Hour,
		ComposeThreshold: 8,
	})
	d.ComposeService = &mockComposeService{
		briefCount: 10,
		composeErr: errors.New("clustering failed"),
	}

	result := d.RunPeriodicCompose()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Error == nil {
		t.Error("expected error from compose")
	}
}

func TestRunPeriodicCompose_MarksRun(t *testing.T) {
	d := NewWithConfig(daemonconfig.Config{
		ComposeEnabled:   true,
		ComposeInterval:  2 * time.Hour,
		ComposeThreshold: 8,
	})
	d.ComposeService = &mockComposeService{briefCount: 3}

	d.RunPeriodicCompose()

	if d.Scheduler.LastRunTime(TaskCompose).IsZero() {
		t.Error("expected LastRunTime to be set after running")
	}
}

func TestRunPeriodicCompose_ExactThreshold(t *testing.T) {
	svc := &mockComposeService{
		briefCount: 8,
		digest: &compose.Digest{
			BriefsComposed: 8,
			ClustersFound:  2,
		},
		writePath: "/tmp/digests/2026-03-28-digest.md",
	}
	d := NewWithConfig(daemonconfig.Config{
		ComposeEnabled:   true,
		ComposeInterval:  2 * time.Hour,
		ComposeThreshold: 8,
	})
	d.ComposeService = svc

	result := d.RunPeriodicCompose()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Composed {
		t.Error("should compose when at exact threshold")
	}
}
