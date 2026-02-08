package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCreateProbeFile(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, ".kb", "models"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, ".orch", "templates"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(projectDir, ".kb", "models", "spawn-architecture.md"),
		[]byte("# Spawn Architecture\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	template := "# Probe: {{question}}\n\n**Model:** {{model-path}}\n**Date:** {{date}}\n"
	if err := os.WriteFile(
		filepath.Join(projectDir, ".orch", "templates", "PROBE.md"),
		[]byte(template),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, time.February, 8, 10, 30, 0, 0, time.UTC)
	path, err := createProbeFile(projectDir, "spawn-architecture", "check-session-lifecycle", now)
	if err != nil {
		t.Fatalf("createProbeFile failed: %v", err)
	}

	expectedPath := filepath.Join(projectDir, ".kb", "models", "spawn-architecture", "probes", "2026-02-08-check-session-lifecycle.md")
	if path != expectedPath {
		t.Errorf("createProbeFile path = %q, want %q", path, expectedPath)
	}

	contentBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read created probe file: %v", err)
	}
	content := string(contentBytes)

	if !strings.Contains(content, "**Model:** spawn-architecture") {
		t.Errorf("expected model name in probe content, got:\n%s", content)
	}
	if !strings.Contains(content, "**Date:** 2026-02-08") {
		t.Errorf("expected date in probe content, got:\n%s", content)
	}
}

func TestCreateProbeFile_ModelMustExist(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, ".kb", "models"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, ".orch", "templates"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".orch", "templates", "PROBE.md"), []byte("{{model-path}} {{date}}"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := createProbeFile(projectDir, "missing-model", "probe-slug", time.Now())
	if err == nil {
		t.Fatal("expected error for missing model, got nil")
	}

	if !strings.Contains(err.Error(), "model not found") {
		t.Fatalf("expected model-not-found error, got: %v", err)
	}
}

func TestCreateProbeFile_NormalizesSlug(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, ".kb", "models"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, ".orch", "templates"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".kb", "models", "test-model.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".orch", "templates", "PROBE.md"), []byte("{{model-path}} {{date}}"), 0644); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, time.February, 8, 0, 0, 0, 0, time.UTC)
	path, err := createProbeFile(projectDir, "test-model", " Check Session Lifecycle! ", now)
	if err != nil {
		t.Fatalf("createProbeFile failed: %v", err)
	}

	if filepath.Base(path) != "2026-02-08-check-session-lifecycle.md" {
		t.Errorf("unexpected filename: %s", filepath.Base(path))
	}
}

func TestCreateProbeFile_RejectsExistingProbe(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, ".kb", "models"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, ".orch", "templates"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".kb", "models", "test-model.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".orch", "templates", "PROBE.md"), []byte("{{model-path}} {{date}}"), 0644); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, time.February, 8, 0, 0, 0, 0, time.UTC)
	_, err := createProbeFile(projectDir, "test-model", "duplicate-slug", now)
	if err != nil {
		t.Fatalf("initial createProbeFile failed: %v", err)
	}

	_, err = createProbeFile(projectDir, "test-model", "duplicate-slug", now)
	if err == nil {
		t.Fatal("expected duplicate probe error, got nil")
	}

	if !strings.Contains(err.Error(), "probe already exists") {
		t.Fatalf("expected duplicate file error, got: %v", err)
	}
}

func TestResolveProbeModelPath_AcceptsMdSuffix(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, ".kb", "models"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".kb", "models", "test-model.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	path, modelName, err := resolveProbeModelPath(projectDir, "test-model.md")
	if err != nil {
		t.Fatalf("resolveProbeModelPath failed: %v", err)
	}

	expectedPath := filepath.Join(projectDir, ".kb", "models", "test-model.md")
	if path != expectedPath {
		t.Errorf("resolveProbeModelPath path = %q, want %q", path, expectedPath)
	}
	if modelName != "test-model" {
		t.Errorf("resolveProbeModelPath modelName = %q, want %q", modelName, "test-model")
	}
}
