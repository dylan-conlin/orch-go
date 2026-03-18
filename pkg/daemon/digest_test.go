package daemon

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/digest"
)

// --- Daemon RunPeriodicDigest integration tests ---

func TestDaemon_RunPeriodicDigest_NotDue(t *testing.T) {
	cfg := Config{
		DigestEnabled:  true,
		DigestInterval: time.Hour,
	}
	d := &Daemon{
		Scheduler: NewSchedulerFromConfig(cfg),
	}
	d.Scheduler.SetLastRun(TaskDigest, time.Now())

	result := d.RunPeriodicDigest()
	if result != nil {
		t.Error("expected nil when not due")
	}
}

func TestDaemon_RunPeriodicDigest_NilService(t *testing.T) {
	cfg := Config{
		DigestEnabled:  true,
		DigestInterval: time.Hour,
	}
	d := &Daemon{
		Scheduler: NewSchedulerFromConfig(cfg),
	}

	result := d.RunPeriodicDigest()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Error == nil {
		t.Error("expected error when service is nil")
	}
}

type mockDigestService struct {
	threads        []digest.ArtifactChange
	models         []digest.ArtifactChange
	investigations []digest.ArtifactChange
}

func (m *mockDigestService) ScanThreads(hashes map[string]string) ([]digest.ArtifactChange, map[string]string, error) {
	newHashes := make(map[string]string)
	for k, v := range hashes {
		newHashes[k] = v
	}
	for _, c := range m.threads {
		newHashes[c.Path] = "sha256:new"
	}
	return m.threads, newHashes, nil
}

func (m *mockDigestService) ScanModels(hashes map[string]string) ([]digest.ArtifactChange, map[string]string, error) {
	newHashes := make(map[string]string)
	for k, v := range hashes {
		newHashes[k] = v
	}
	for _, c := range m.models {
		newHashes[c.Path] = "sha256:new"
	}
	return m.models, newHashes, nil
}

func (m *mockDigestService) ScanInvestigations(hashes map[string]string) ([]digest.ArtifactChange, map[string]string, error) {
	newHashes := make(map[string]string)
	for k, v := range hashes {
		newHashes[k] = v
	}
	for _, c := range m.investigations {
		newHashes[c.Path] = "sha256:new"
	}
	return m.investigations, newHashes, nil
}

func TestDaemon_RunPeriodicDigest_ProducesProducts(t *testing.T) {
	digestDir := t.TempDir()
	stateDir := t.TempDir()

	cfg := Config{
		DigestEnabled:  true,
		DigestInterval: time.Hour,
	}
	d := &Daemon{
		Scheduler: NewSchedulerFromConfig(cfg),
		Digest: &mockDigestService{
			threads: []digest.ArtifactChange{
				{
					Path:       ".kb/threads/governance.md",
					ChangeType: "content_added",
					DeltaWords: 300,
					Summary:    "Explores Ostrom's 8 principles for commons management.",
				},
			},
		},
		DigestDir:       digestDir,
		DigestStatePath: filepath.Join(stateDir, "digest-state.json"),
	}

	result := d.RunPeriodicDigest()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.Produced != 1 {
		t.Errorf("Produced = %d, want 1", result.Produced)
	}

	store := digest.NewStore(digestDir)
	products, err := store.List(digest.ListOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("expected 1 product file, got %d", len(products))
	}
	if products[0].Type != digest.TypeThreadProgression {
		t.Errorf("Type = %q, want thread_progression", products[0].Type)
	}
}

func TestDaemon_RunPeriodicDigest_SkipsBelowThreshold(t *testing.T) {
	digestDir := t.TempDir()
	stateDir := t.TempDir()

	cfg := Config{
		DigestEnabled:  true,
		DigestInterval: time.Hour,
	}
	d := &Daemon{
		Scheduler: NewSchedulerFromConfig(cfg),
		Digest: &mockDigestService{
			threads: []digest.ArtifactChange{
				{
					Path:       ".kb/threads/small.md",
					ChangeType: "content_added",
					DeltaWords: 50,
				},
			},
		},
		DigestDir:       digestDir,
		DigestStatePath: filepath.Join(stateDir, "digest-state.json"),
	}

	result := d.RunPeriodicDigest()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Produced != 0 {
		t.Errorf("Produced = %d, want 0 (below threshold)", result.Produced)
	}
	if result.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", result.Skipped)
	}
}
