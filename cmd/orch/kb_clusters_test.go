package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunKBClusters_NoModelsDir(t *testing.T) {
	dir := t.TempDir()
	err := runKBClusters(dir, 3, false)
	if err != nil {
		t.Fatalf("expected no error for missing models dir, got: %v", err)
	}
}

func TestRunKBClusters_NoClusters(t *testing.T) {
	dir := t.TempDir()
	modelsDir := filepath.Join(dir, ".kb", "models", "model-a")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		t.Fatal(err)
	}
	// claims.yaml with no tensions
	if err := os.WriteFile(filepath.Join(modelsDir, "claims.yaml"), []byte(`
model: model-a
version: 1
claims:
  - id: A-01
    text: "a claim"
    type: observation
    scope: local
    confidence: confirmed
    priority: core
`), 0644); err != nil {
		t.Fatal(err)
	}

	err := runKBClusters(dir, 3, false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestRunKBClusters_WithClusters(t *testing.T) {
	dir := t.TempDir()
	kbModels := filepath.Join(dir, ".kb", "models")

	// Create model-a with tensions pointing to T-01 in target
	modelA := filepath.Join(kbModels, "model-a")
	if err := os.MkdirAll(modelA, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modelA, "claims.yaml"), []byte(`
model: model-a
version: 1
claims:
  - id: A-01
    text: "first claim"
    type: observation
    scope: local
    confidence: confirmed
    priority: core
    domain_tags: [gates, enforcement]
    tensions:
      - claim: T-01
        model: target-model
        type: extends
        note: "deepens"
  - id: A-02
    text: "second claim"
    type: observation
    scope: local
    confidence: confirmed
    priority: core
    tensions:
      - claim: T-01
        model: target-model
        type: contradicts
        note: "conflicts"
`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create model-b with tension pointing to same T-01
	modelB := filepath.Join(kbModels, "model-b")
	if err := os.MkdirAll(modelB, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modelB, "claims.yaml"), []byte(`
model: model-b
version: 1
claims:
  - id: B-01
    text: "third claim"
    type: observation
    scope: local
    confidence: confirmed
    priority: core
    tensions:
      - claim: T-01
        model: target-model
        type: confirms
        note: "agrees"
`), 0644); err != nil {
		t.Fatal(err)
	}

	// Should find 1 cluster with threshold=3
	err := runKBClusters(dir, 3, false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// JSON mode should also work
	err = runKBClusters(dir, 3, true)
	if err != nil {
		t.Fatalf("expected no error for JSON mode, got: %v", err)
	}
}
