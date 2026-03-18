package digest

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- Product type tests ---

func TestProduct_JSON_RoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	p := Product{
		ID:           "20260314T1030-thread-governance",
		Type:         TypeThreadProgression,
		Title:        "Thread: Governance — new entry",
		Summary:      "Explores Ostrom's 8 principles.",
		Significance: SignificanceHigh,
		Source: Source{
			ArtifactType: "thread",
			Path:         ".kb/threads/governance.md",
			ChangeType:   "content_added",
			DeltaWords:   300,
		},
		State:     StateNew,
		CreatedAt: now,
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got Product
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
	if got.State != StateNew {
		t.Errorf("State = %q, want %q", got.State, StateNew)
	}
	if got.Source.DeltaWords != 300 {
		t.Errorf("DeltaWords = %d, want 300", got.Source.DeltaWords)
	}
}

// --- State tests ---

func TestState_LoadSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "digest-state.json")

	state := &State{
		LastScan:   time.Now().UTC().Truncate(time.Second),
		FileHashes: map[string]string{".kb/threads/foo.md": "sha256:abc123"},
		Stats: Stats{
			TotalProduced: 10,
			TotalRead:     7,
			TotalStarred:  2,
		},
	}

	if err := SaveState(statePath, state); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	loaded, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}

	if loaded.Stats.TotalProduced != 10 {
		t.Errorf("TotalProduced = %d, want 10", loaded.Stats.TotalProduced)
	}
	if loaded.FileHashes[".kb/threads/foo.md"] != "sha256:abc123" {
		t.Errorf("FileHash mismatch")
	}
}

func TestState_LoadMissing_ReturnsEmpty(t *testing.T) {
	state, err := LoadState("/nonexistent/path.json")
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.FileHashes == nil {
		t.Error("FileHashes should be initialized (not nil)")
	}
	if len(state.FileHashes) != 0 {
		t.Error("FileHashes should be empty")
	}
}

// --- Store tests ---

func TestStore_WriteAndList(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	p := Product{
		ID:           "20260314T1030-thread-test",
		Type:         TypeThreadProgression,
		Title:        "Test product",
		Summary:      "Test summary",
		Significance: SignificanceMedium,
		State:        StateNew,
		CreatedAt:    time.Now().UTC(),
	}

	if err := store.Write(p); err != nil {
		t.Fatalf("Write: %v", err)
	}

	products, err := store.List(ListOpts{})
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

func TestStore_ListFilterByState(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	for i, state := range []ProductState{StateNew, StateRead} {
		p := Product{
			ID:        fmt.Sprintf("prod-%d", i),
			Type:      TypeThreadProgression,
			State:     state,
			CreatedAt: time.Now().UTC(),
		}
		if err := store.Write(p); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	products, err := store.List(ListOpts{State: StateNew})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(products) != 1 {
		t.Fatalf("List(state=new) returned %d, want 1", len(products))
	}
	if products[0].State != StateNew {
		t.Errorf("State = %q, want new", products[0].State)
	}
}

func TestStore_ListFilterByType(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	for i, typ := range []ProductType{TypeThreadProgression, TypeModelUpdate} {
		p := Product{
			ID:        fmt.Sprintf("prod-%d", i),
			Type:      typ,
			State:     StateNew,
			CreatedAt: time.Now().UTC(),
		}
		if err := store.Write(p); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	products, err := store.List(ListOpts{Type: TypeModelUpdate})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(products) != 1 {
		t.Fatalf("List(type=model_update) returned %d, want 1", len(products))
	}
}

func TestStore_UpdateState(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	p := Product{
		ID:        "prod-update",
		Type:      TypeThreadProgression,
		State:     StateNew,
		CreatedAt: time.Now().UTC(),
	}
	if err := store.Write(p); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if err := store.UpdateState("prod-update", StateRead); err != nil {
		t.Fatalf("UpdateState: %v", err)
	}

	products, err := store.List(ListOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(products) != 1 {
		t.Fatal("expected 1 product")
	}
	if products[0].State != StateRead {
		t.Errorf("State = %q, want read", products[0].State)
	}
	if products[0].ReadAt.IsZero() {
		t.Error("ReadAt should be set")
	}
}

func TestStore_UpdateState_Starred(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	p := Product{
		ID:        "prod-star",
		Type:      TypeThreadProgression,
		State:     StateNew,
		CreatedAt: time.Now().UTC(),
	}
	if err := store.Write(p); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if err := store.UpdateState("prod-star", StateStarred); err != nil {
		t.Fatalf("UpdateState: %v", err)
	}

	products, err := store.List(ListOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if products[0].State != StateStarred {
		t.Errorf("State = %q, want starred", products[0].State)
	}
	if products[0].StarredAt.IsZero() {
		t.Error("StarredAt should be set")
	}
}

func TestStore_UpdateState_NotFound(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	err := store.UpdateState("nonexistent", StateRead)
	if err == nil {
		t.Error("expected error for nonexistent product")
	}
}

func TestStore_Stats(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	states := []ProductState{StateNew, StateNew, StateRead, StateStarred}
	for i, state := range states {
		p := Product{
			ID:        fmt.Sprintf("prod-%d", i),
			Type:      TypeThreadProgression,
			State:     state,
			CreatedAt: time.Now().UTC(),
		}
		if err := store.Write(p); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	stats, err := store.StoreStats()
	if err != nil {
		t.Fatalf("StoreStats: %v", err)
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

func TestStore_ArchiveRead(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	oldTime := time.Now().UTC().Add(-10 * 24 * time.Hour)
	newTime := time.Now().UTC()

	old := Product{
		ID:        "old-read",
		Type:      TypeThreadProgression,
		State:     StateRead,
		CreatedAt: oldTime,
	}
	newer := Product{
		ID:        "new-read",
		Type:      TypeThreadProgression,
		State:     StateRead,
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

	products, err := store.List(ListOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	for _, p := range products {
		if p.ID == "old-read" && p.State != StateArchived {
			t.Errorf("old-read State = %q, want archived", p.State)
		}
		if p.ID == "new-read" && p.State != StateRead {
			t.Errorf("new-read should still be read, got %q", p.State)
		}
	}
}

// --- RunDigest tests ---

type mockService struct {
	threads        []ArtifactChange
	models         []ArtifactChange
	investigations []ArtifactChange
}

func (m *mockService) ScanThreads(hashes map[string]string) ([]ArtifactChange, map[string]string, error) {
	newHashes := make(map[string]string)
	for k, v := range hashes {
		newHashes[k] = v
	}
	for _, c := range m.threads {
		newHashes[c.Path] = "sha256:new"
	}
	return m.threads, newHashes, nil
}

func (m *mockService) ScanModels(hashes map[string]string) ([]ArtifactChange, map[string]string, error) {
	newHashes := make(map[string]string)
	for k, v := range hashes {
		newHashes[k] = v
	}
	for _, c := range m.models {
		newHashes[c.Path] = "sha256:new"
	}
	return m.models, newHashes, nil
}

func (m *mockService) ScanInvestigations(hashes map[string]string) ([]ArtifactChange, map[string]string, error) {
	newHashes := make(map[string]string)
	for k, v := range hashes {
		newHashes[k] = v
	}
	for _, c := range m.investigations {
		newHashes[c.Path] = "sha256:new"
	}
	return m.investigations, newHashes, nil
}

func TestRunDigest_ProducesProducts(t *testing.T) {
	digestDir := t.TempDir()
	stateDir := t.TempDir()

	svc := &mockService{
		threads: []ArtifactChange{
			{
				Path:       ".kb/threads/governance.md",
				ChangeType: "content_added",
				DeltaWords: 300,
				Summary:    "Explores Ostrom's 8 principles for commons management.",
			},
		},
	}

	result := RunDigest(svc, digestDir, filepath.Join(stateDir, "digest-state.json"))
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.Produced != 1 {
		t.Errorf("Produced = %d, want 1", result.Produced)
	}

	store := NewStore(digestDir)
	products, err := store.List(ListOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("expected 1 product file, got %d", len(products))
	}
	if products[0].Type != TypeThreadProgression {
		t.Errorf("Type = %q, want thread_progression", products[0].Type)
	}
}

func TestRunDigest_SkipsBelowThreshold(t *testing.T) {
	digestDir := t.TempDir()
	stateDir := t.TempDir()

	svc := &mockService{
		threads: []ArtifactChange{
			{
				Path:       ".kb/threads/small.md",
				ChangeType: "content_added",
				DeltaWords: 50,
			},
		},
	}

	result := RunDigest(svc, digestDir, filepath.Join(stateDir, "digest-state.json"))
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

	hash, err := FileHash(path)
	if err != nil {
		t.Fatalf("FileHash: %v", err)
	}

	expected := fmt.Sprintf("sha256:%x", sha256.Sum256(content))
	if hash != expected {
		t.Errorf("hash = %q, want %q", hash, expected)
	}
}

// --- Significance classification tests ---

func TestClassifySignificance_ThreadBelowThreshold(t *testing.T) {
	sig := ClassifySignificance(TypeThreadProgression, ArtifactChange{DeltaWords: 100})
	if sig != "" {
		t.Errorf("expected empty (below threshold), got %q", sig)
	}
}

func TestClassifySignificance_ThreadAboveThreshold(t *testing.T) {
	sig := ClassifySignificance(TypeThreadProgression, ArtifactChange{DeltaWords: 250})
	if sig != SignificanceMedium {
		t.Errorf("expected medium, got %q", sig)
	}
}

func TestClassifySignificance_ModelContradicts(t *testing.T) {
	sig := ClassifySignificance(TypeModelUpdate, ArtifactChange{Summary: "Probe contradicts claim about spawn dedup"})
	if sig != SignificanceHigh {
		t.Errorf("expected high (contradicts), got %q", sig)
	}
}

func TestClassifySignificance_DecisionBrief(t *testing.T) {
	sig := ClassifySignificance(TypeDecisionBrief, ArtifactChange{})
	if sig != SignificanceHigh {
		t.Errorf("expected high (decision brief), got %q", sig)
	}
}
