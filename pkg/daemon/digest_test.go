package daemon

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- Digest product type tests ---

func TestDigestProduct_JSON_RoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	p := DigestProduct{
		ID:           "20260314T1030-thread-governance",
		Type:         DigestTypeThreadProgression,
		Title:        "Thread: Governance — new entry",
		Summary:      "Explores Ostrom's 8 principles.",
		Significance: SignificanceHigh,
		Source: DigestSource{
			ArtifactType: "thread",
			Path:         ".kb/threads/governance.md",
			ChangeType:   "content_added",
			DeltaWords:   300,
		},
		State:     DigestStateNew,
		CreatedAt: now,
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got DigestProduct
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.ID != p.ID {
		t.Errorf("ID = %q, want %q", got.ID, p.ID)
	}
	if got.Type != p.Type {
		t.Errorf("Type = %q, want %q", got.Type, p.Type)
	}
	if got.Significance != p.Significance {
		t.Errorf("Significance = %q, want %q", got.Significance, p.Significance)
	}
	if got.State != DigestStateNew {
		t.Errorf("State = %q, want %q", got.State, DigestStateNew)
	}
	if got.Source.DeltaWords != 300 {
		t.Errorf("DeltaWords = %d, want 300", got.Source.DeltaWords)
	}
}

// --- Digest state tests ---

func TestDigestState_LoadSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "digest-state.json")

	state := &DigestState{
		LastScan:   time.Now().UTC().Truncate(time.Second),
		FileHashes: map[string]string{".kb/threads/foo.md": "sha256:abc123"},
		Stats: DigestStats{
			TotalProduced: 10,
			TotalRead:     7,
			TotalStarred:  2,
		},
	}

	if err := SaveDigestState(statePath, state); err != nil {
		t.Fatalf("SaveDigestState: %v", err)
	}

	loaded, err := LoadDigestState(statePath)
	if err != nil {
		t.Fatalf("LoadDigestState: %v", err)
	}

	if loaded.Stats.TotalProduced != 10 {
		t.Errorf("TotalProduced = %d, want 10", loaded.Stats.TotalProduced)
	}
	if loaded.FileHashes[".kb/threads/foo.md"] != "sha256:abc123" {
		t.Errorf("FileHash mismatch")
	}
}

func TestDigestState_LoadMissing_ReturnsEmpty(t *testing.T) {
	state, err := LoadDigestState("/nonexistent/path.json")
	if err != nil {
		t.Fatalf("LoadDigestState: %v", err)
	}
	if state.FileHashes == nil {
		t.Error("FileHashes should be initialized (not nil)")
	}
	if len(state.FileHashes) != 0 {
		t.Error("FileHashes should be empty")
	}
}

// --- Digest store tests ---

func TestDigestStore_WriteAndList(t *testing.T) {
	dir := t.TempDir()
	store := NewDigestStore(dir)

	p := DigestProduct{
		ID:           "20260314T1030-thread-test",
		Type:         DigestTypeThreadProgression,
		Title:        "Test product",
		Summary:      "Test summary",
		Significance: SignificanceMedium,
		State:        DigestStateNew,
		CreatedAt:    time.Now().UTC(),
	}

	if err := store.Write(p); err != nil {
		t.Fatalf("Write: %v", err)
	}

	products, err := store.List(DigestListOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(products) != 1 {
		t.Fatalf("List returned %d products, want 1", len(products))
	}
	if products[0].ID != p.ID {
		t.Errorf("ID = %q, want %q", products[0].ID, p.ID)
	}
}

func TestDigestStore_ListFilterByState(t *testing.T) {
	dir := t.TempDir()
	store := NewDigestStore(dir)

	for i, state := range []DigestProductState{DigestStateNew, DigestStateRead} {
		p := DigestProduct{
			ID:        fmt.Sprintf("prod-%d", i),
			Type:      DigestTypeThreadProgression,
			State:     state,
			CreatedAt: time.Now().UTC(),
		}
		if err := store.Write(p); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	products, err := store.List(DigestListOpts{State: DigestStateNew})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(products) != 1 {
		t.Fatalf("List(state=new) returned %d, want 1", len(products))
	}
	if products[0].State != DigestStateNew {
		t.Errorf("State = %q, want new", products[0].State)
	}
}

func TestDigestStore_ListFilterByType(t *testing.T) {
	dir := t.TempDir()
	store := NewDigestStore(dir)

	for i, typ := range []DigestProductType{DigestTypeThreadProgression, DigestTypeModelUpdate} {
		p := DigestProduct{
			ID:        fmt.Sprintf("prod-%d", i),
			Type:      typ,
			State:     DigestStateNew,
			CreatedAt: time.Now().UTC(),
		}
		if err := store.Write(p); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	products, err := store.List(DigestListOpts{Type: DigestTypeModelUpdate})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(products) != 1 {
		t.Fatalf("List(type=model_update) returned %d, want 1", len(products))
	}
}

func TestDigestStore_UpdateState(t *testing.T) {
	dir := t.TempDir()
	store := NewDigestStore(dir)

	p := DigestProduct{
		ID:        "prod-update",
		Type:      DigestTypeThreadProgression,
		State:     DigestStateNew,
		CreatedAt: time.Now().UTC(),
	}
	if err := store.Write(p); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if err := store.UpdateState("prod-update", DigestStateRead); err != nil {
		t.Fatalf("UpdateState: %v", err)
	}

	products, err := store.List(DigestListOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(products) != 1 {
		t.Fatal("expected 1 product")
	}
	if products[0].State != DigestStateRead {
		t.Errorf("State = %q, want read", products[0].State)
	}
	if products[0].ReadAt.IsZero() {
		t.Error("ReadAt should be set")
	}
}

func TestDigestStore_UpdateState_Starred(t *testing.T) {
	dir := t.TempDir()
	store := NewDigestStore(dir)

	p := DigestProduct{
		ID:        "prod-star",
		Type:      DigestTypeThreadProgression,
		State:     DigestStateNew,
		CreatedAt: time.Now().UTC(),
	}
	if err := store.Write(p); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if err := store.UpdateState("prod-star", DigestStateStarred); err != nil {
		t.Fatalf("UpdateState: %v", err)
	}

	products, err := store.List(DigestListOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if products[0].State != DigestStateStarred {
		t.Errorf("State = %q, want starred", products[0].State)
	}
	if products[0].StarredAt.IsZero() {
		t.Error("StarredAt should be set")
	}
}

func TestDigestStore_UpdateState_NotFound(t *testing.T) {
	dir := t.TempDir()
	store := NewDigestStore(dir)

	err := store.UpdateState("nonexistent", DigestStateRead)
	if err == nil {
		t.Error("expected error for nonexistent product")
	}
}

func TestDigestStore_Stats(t *testing.T) {
	dir := t.TempDir()
	store := NewDigestStore(dir)

	states := []DigestProductState{DigestStateNew, DigestStateNew, DigestStateRead, DigestStateStarred}
	for i, state := range states {
		p := DigestProduct{
			ID:        fmt.Sprintf("prod-%d", i),
			Type:      DigestTypeThreadProgression,
			State:     state,
			CreatedAt: time.Now().UTC(),
		}
		if err := store.Write(p); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}

	if stats.Unread != 2 {
		t.Errorf("Unread = %d, want 2", stats.Unread)
	}
	if stats.Read != 1 {
		t.Errorf("Read = %d, want 1", stats.Read)
	}
	if stats.Starred != 1 {
		t.Errorf("Starred = %d, want 1", stats.Starred)
	}
	if stats.Total != 4 {
		t.Errorf("Total = %d, want 4", stats.Total)
	}
}

func TestDigestStore_ArchiveRead(t *testing.T) {
	dir := t.TempDir()
	store := NewDigestStore(dir)

	oldTime := time.Now().UTC().Add(-10 * 24 * time.Hour)
	newTime := time.Now().UTC()

	old := DigestProduct{
		ID:        "old-read",
		Type:      DigestTypeThreadProgression,
		State:     DigestStateRead,
		CreatedAt: oldTime,
	}
	newer := DigestProduct{
		ID:        "new-read",
		Type:      DigestTypeThreadProgression,
		State:     DigestStateRead,
		CreatedAt: newTime,
	}

	if err := store.Write(old); err != nil {
		t.Fatalf("Write old: %v", err)
	}
	if err := store.Write(newer); err != nil {
		t.Fatalf("Write new: %v", err)
	}

	archived, err := store.ArchiveRead(7 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("ArchiveRead: %v", err)
	}

	if archived != 1 {
		t.Errorf("Archived = %d, want 1", archived)
	}

	products, err := store.List(DigestListOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	for _, p := range products {
		if p.ID == "old-read" && p.State != DigestStateArchived {
			t.Errorf("old-read State = %q, want archived", p.State)
		}
		if p.ID == "new-read" && p.State != DigestStateRead {
			t.Errorf("new-read should still be read, got %q", p.State)
		}
	}
}

// --- Daemon RunPeriodicDigest tests ---

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
	threads        []DigestArtifactChange
	models         []DigestArtifactChange
	investigations []DigestArtifactChange
}

func (m *mockDigestService) ScanThreads(hashes map[string]string) ([]DigestArtifactChange, map[string]string, error) {
	newHashes := make(map[string]string)
	for k, v := range hashes {
		newHashes[k] = v
	}
	for _, c := range m.threads {
		newHashes[c.Path] = "sha256:new"
	}
	return m.threads, newHashes, nil
}

func (m *mockDigestService) ScanModels(hashes map[string]string) ([]DigestArtifactChange, map[string]string, error) {
	newHashes := make(map[string]string)
	for k, v := range hashes {
		newHashes[k] = v
	}
	for _, c := range m.models {
		newHashes[c.Path] = "sha256:new"
	}
	return m.models, newHashes, nil
}

func (m *mockDigestService) ScanInvestigations(hashes map[string]string) ([]DigestArtifactChange, map[string]string, error) {
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
			threads: []DigestArtifactChange{
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

	store := NewDigestStore(digestDir)
	products, err := store.List(DigestListOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("expected 1 product file, got %d", len(products))
	}
	if products[0].Type != DigestTypeThreadProgression {
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
			threads: []DigestArtifactChange{
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

// --- File hash utility tests ---

func TestFileHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	content := []byte("hello world")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	hash, err := fileHash(path)
	if err != nil {
		t.Fatalf("fileHash: %v", err)
	}

	expected := fmt.Sprintf("sha256:%x", sha256.Sum256(content))
	if hash != expected {
		t.Errorf("hash = %q, want %q", hash, expected)
	}
}

// --- Significance classification tests ---

func TestClassifySignificance_ThreadBelowThreshold(t *testing.T) {
	sig := classifySignificance(DigestTypeThreadProgression, DigestArtifactChange{DeltaWords: 100})
	if sig != "" {
		t.Errorf("expected empty (below threshold), got %q", sig)
	}
}

func TestClassifySignificance_ThreadAboveThreshold(t *testing.T) {
	sig := classifySignificance(DigestTypeThreadProgression, DigestArtifactChange{DeltaWords: 250})
	if sig != SignificanceMedium {
		t.Errorf("expected medium, got %q", sig)
	}
}

func TestClassifySignificance_ModelContradicts(t *testing.T) {
	sig := classifySignificance(DigestTypeModelUpdate, DigestArtifactChange{Summary: "Probe contradicts claim about spawn dedup"})
	if sig != SignificanceHigh {
		t.Errorf("expected high (contradicts), got %q", sig)
	}
}

func TestClassifySignificance_DecisionBrief(t *testing.T) {
	sig := classifySignificance(DigestTypeDecisionBrief, DigestArtifactChange{})
	if sig != SignificanceHigh {
		t.Errorf("expected high (decision brief), got %q", sig)
	}
}
