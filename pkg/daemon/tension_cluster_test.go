package daemon

import (
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemonconfig"
)

type mockTensionClusterService struct {
	openClusters map[string]bool
	created      []string
	createErr    error
}

func (m *mockTensionClusterService) HasOpenClusterIssue(clusterID string) (bool, error) {
	return m.openClusters[clusterID], nil
}

func (m *mockTensionClusterService) CreateClusterIssue(cluster TensionClusterIssue) (string, error) {
	if m.createErr != nil {
		return "", m.createErr
	}
	id := "orch-go-tc-" + cluster.ClusterID
	m.created = append(m.created, id)
	return id, nil
}

func TestRunPeriodicTensionClusterScan_NotDue(t *testing.T) {
	d := NewWithConfig(daemonconfig.Config{
		TensionClusterScanEnabled:  true,
		TensionClusterScanInterval: time.Hour,
	})
	// Mark as just run
	d.Scheduler.MarkRun(TaskTensionClusterScan)

	result := d.RunPeriodicTensionClusterScan()
	if result != nil {
		t.Error("expected nil result when not due")
	}
}

func TestRunPeriodicTensionClusterScan_NoService(t *testing.T) {
	d := NewWithConfig(daemonconfig.Config{
		TensionClusterScanEnabled:  true,
		TensionClusterScanInterval: time.Hour,
	})

	result := d.RunPeriodicTensionClusterScan()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Message != "tension cluster service not configured" {
		t.Errorf("unexpected message: %s", result.Message)
	}
}

func TestRunPeriodicTensionClusterScan_Disabled(t *testing.T) {
	d := NewWithConfig(daemonconfig.Config{
		TensionClusterScanEnabled:  false,
		TensionClusterScanInterval: time.Hour,
	})

	result := d.RunPeriodicTensionClusterScan()
	if result != nil {
		t.Error("expected nil result when disabled")
	}
}

func TestTensionClusterScanDefaultConfig(t *testing.T) {
	config := daemonconfig.DefaultConfig()
	if !config.TensionClusterScanEnabled {
		t.Error("TensionClusterScanEnabled should be true by default")
	}
	if config.TensionClusterScanInterval != 24*time.Hour {
		t.Errorf("TensionClusterScanInterval = %v, want 24h", config.TensionClusterScanInterval)
	}
	if config.TensionClusterThreshold != 3 {
		t.Errorf("TensionClusterThreshold = %d, want 3", config.TensionClusterThreshold)
	}
}

func TestTensionClusterResultSnapshot(t *testing.T) {
	r := &TensionClusterResult{
		ClusterCount: 2,
		IssueCreated: "orch-go-tc-01",
		Message:      "created issue for cluster tc-01",
	}
	snap := r.Snapshot()
	if snap.ClusterCount != 2 {
		t.Errorf("snapshot ClusterCount = %d, want 2", snap.ClusterCount)
	}
	if snap.IssueCreated != "orch-go-tc-01" {
		t.Errorf("snapshot IssueCreated = %s, want orch-go-tc-01", snap.IssueCreated)
	}
}
