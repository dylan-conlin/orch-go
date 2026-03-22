package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	daemonconfig "github.com/dylan-conlin/orch-go/pkg/daemonconfig"
)

type mockClaimProbeService struct {
	openProbes    map[string]bool // claimID -> has open probe
	createdProbes []string        // issued IDs
	createErr     error
}

func (m *mockClaimProbeService) HasOpenProbeForClaim(claimID, modelName string) (bool, error) {
	return m.openProbes[claimID], nil
}

func (m *mockClaimProbeService) CreateProbeIssue(claimID, claimText, falsifiesIf, modelName string) (string, error) {
	if m.createErr != nil {
		return "", m.createErr
	}
	id := "probe-" + claimID
	m.createdProbes = append(m.createdProbes, id)
	return id, nil
}

func TestRunPeriodicClaimProbeGeneration_NotDue(t *testing.T) {
	d := NewWithConfig(daemonconfig.Config{
		ClaimProbeGenerationEnabled:  true,
		ClaimProbeGenerationInterval: time.Hour,
	})
	// Mark as just run
	d.Scheduler.MarkRun(TaskClaimProbeGeneration)

	result := d.RunPeriodicClaimProbeGeneration()
	if result != nil {
		t.Error("expected nil when not due")
	}
}

func TestRunPeriodicClaimProbeGeneration_NoService(t *testing.T) {
	d := NewWithConfig(daemonconfig.Config{
		ClaimProbeGenerationEnabled:  true,
		ClaimProbeGenerationInterval: time.Hour,
	})

	result := d.RunPeriodicClaimProbeGeneration()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Message != "claim probe service not configured" {
		t.Errorf("Message = %q", result.Message)
	}
}

func TestRunClaimProbeGeneration_WithEligibleClaim(t *testing.T) {
	// Create temp models dir with claims.yaml
	dir := t.TempDir()
	modelDir := filepath.Join(dir, ".kb", "models", "test-model")
	os.MkdirAll(modelDir, 0755)

	// Create model.md (needs to exist and be recent for activity check)
	os.WriteFile(filepath.Join(modelDir, "model.md"), []byte("# Test Model\n**Last Updated:** 2026-03-19\n"), 0644)

	// Create claims.yaml with one probe-eligible claim
	claimsYAML := `model: test-model
version: 1
claims:
  - id: TM-01
    text: "Test claim"
    type: mechanism
    scope: local
    confidence: unconfirmed
    priority: core
    falsifies_if: "This never happens"
    domain_tags: ["testing"]
`
	os.WriteFile(filepath.Join(modelDir, "claims.yaml"), []byte(claimsYAML), 0644)

	// Change to temp dir so findModelsDir works
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	mock := &mockClaimProbeService{
		openProbes: make(map[string]bool),
	}

	d := NewWithConfig(daemonconfig.Config{
		ClaimProbeGenerationEnabled:  true,
		ClaimProbeGenerationInterval: time.Hour,
	})
	d.ClaimProbeService = mock

	result := d.RunPeriodicClaimProbeGeneration()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.ProbeCount != 1 {
		t.Errorf("ProbeCount = %d, want 1", result.ProbeCount)
	}
	if len(mock.createdProbes) != 1 {
		t.Errorf("created probes = %d, want 1", len(mock.createdProbes))
	}
}

func TestRunClaimProbeGeneration_DeduplicatesExisting(t *testing.T) {
	dir := t.TempDir()
	modelDir := filepath.Join(dir, ".kb", "models", "test-model")
	os.MkdirAll(modelDir, 0755)

	os.WriteFile(filepath.Join(modelDir, "model.md"), []byte("# Test\n"), 0644)

	claimsYAML := `model: test-model
version: 1
claims:
  - id: TM-01
    text: "Test claim"
    type: mechanism
    scope: local
    confidence: unconfirmed
    priority: core
    falsifies_if: "This never happens"
`
	os.WriteFile(filepath.Join(modelDir, "claims.yaml"), []byte(claimsYAML), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	mock := &mockClaimProbeService{
		openProbes: map[string]bool{"TM-01": true}, // Already has open probe
	}

	d := NewWithConfig(daemonconfig.Config{
		ClaimProbeGenerationEnabled:  true,
		ClaimProbeGenerationInterval: time.Hour,
	})
	d.ClaimProbeService = mock

	result := d.RunPeriodicClaimProbeGeneration()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ProbeCount != 0 {
		t.Errorf("ProbeCount = %d, want 0 (should be deduped)", result.ProbeCount)
	}
}

func TestRunClaimProbeGeneration_SkipsClaimWithRecentEvidence(t *testing.T) {
	dir := t.TempDir()
	modelDir := filepath.Join(dir, ".kb", "models", "test-model")
	os.MkdirAll(modelDir, 0755)

	os.WriteFile(filepath.Join(modelDir, "model.md"), []byte("# Test\n"), 0644)

	// Claim is unconfirmed but has recent evidence — should NOT be re-probed
	claimsYAML := `model: test-model
version: 1
claims:
  - id: TM-01
    text: "Test claim with recent evidence"
    type: mechanism
    scope: local
    confidence: unconfirmed
    priority: core
    falsifies_if: "This never happens"
    evidence:
      - source: "prior probe found indirect support"
        date: "` + time.Now().AddDate(0, 0, -5).Format("2006-01-02") + `"
        verdict: extends
`
	os.WriteFile(filepath.Join(modelDir, "claims.yaml"), []byte(claimsYAML), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	mock := &mockClaimProbeService{
		openProbes: make(map[string]bool),
	}

	d := NewWithConfig(daemonconfig.Config{
		ClaimProbeGenerationEnabled:  true,
		ClaimProbeGenerationInterval: time.Hour,
	})
	d.ClaimProbeService = mock

	result := d.RunPeriodicClaimProbeGeneration()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ProbeCount != 0 {
		t.Errorf("ProbeCount = %d, want 0 (should skip claim with recent evidence)", result.ProbeCount)
	}
	if len(mock.createdProbes) != 0 {
		t.Errorf("created probes = %d, want 0", len(mock.createdProbes))
	}
}

func TestClaimProbeGenerationDefaultConfig(t *testing.T) {
	config := daemonconfig.DefaultConfig()
	if !config.ClaimProbeGenerationEnabled {
		t.Error("ClaimProbeGenerationEnabled should be true by default")
	}
	if config.ClaimProbeGenerationInterval != 2*time.Hour {
		t.Errorf("ClaimProbeGenerationInterval = %v, want 2h", config.ClaimProbeGenerationInterval)
	}
}
